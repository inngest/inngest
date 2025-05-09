package redis_state

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
)

const (
	ShadowPartitionAccountPeekMax  = int64(30)
	ShadowPartitionPeekMax         = int64(300) // same as PartitionPeekMax for now
	ShadowPartitionPeekMinBacklogs = int64(10)
	ShadowPartitionPeekMaxBacklogs = int64(100)
)

// shadowWorker runs a blocking process that listens to item being pushed into the
// shadow queue partition channel. This allows us to process an individual shadow partition.
func (q *queue) shadowWorker(ctx context.Context, qspc chan *QueueShadowPartition) {
	l := logger.StdlibLogger(ctx)

	for {
		select {
		case <-ctx.Done():
			return

		case shadowPart := <-qspc:
			err := q.processShadowPartition(ctx, shadowPart, 0)
			if err != nil {
				l.Error("could not scan shadow partition", "error", err, "shadow_part", shadowPart)
			}
		}
	}
}

func (q *queue) processShadowPartition(ctx context.Context, shadowPart *QueueShadowPartition, continuationCount uint) error {
	// acquire lease for shadow partition
	leaseID, err := duration(ctx, q.primaryQueueShard.Name, "shadow_partition_lease", q.clock.Now(), func(ctx context.Context) (*ulid.ULID, error) {
		leaseID, err := q.ShadowPartitionLease(ctx, shadowPart, ShadowPartitionLeaseDuration)
		return leaseID, err
	})
	if err != nil {
		if errors.Is(err, ErrShadowPartitionAlreadyLeased) {
			// contention
			return nil
		}
	}

	if leaseID == nil {
		return fmt.Errorf("missing shadow partition leaseID")
	}

	metrics.ActiveShadowScannerCount(ctx, 1, metrics.CounterOpt{PkgName: pkgName})
	defer metrics.ActiveShadowScannerCount(ctx, -1, metrics.CounterOpt{PkgName: pkgName})

	extendLeaseCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// keep extending lease while we're processing
	go func() {
		for {
			select {
			case <-extendLeaseCtx.Done():
				return
			case <-time.Tick(ShadowPartitionLeaseDuration / 2):
				leaseID, err = q.ShadowPartitionExtendLease(ctx, shadowPart, *leaseID, ShadowPartitionLeaseDuration)
				if err != nil {
					if errors.Is(err, ErrShadowPartitionAlreadyLeased) || errors.Is(err, ErrShadowPartitionLeaseNotFound) {
						// contention
						return
					}
					return
				}
			}
		}
	}()

	// Check if shadow partition cannot be processed (paused/refill disabled, etc.)
	if shadowPart.PauseRefill {
		q.removeShadowContinue(ctx, shadowPart, false)

		forceRequeueAt := q.clock.Now().Add(ShadowPartitionRefillPausedRequeueExtension)
		err = q.ShadowPartitionRequeue(ctx, shadowPart, *leaseID, &forceRequeueAt)
		if err != nil {
			return fmt.Errorf("could not requeue shadow partition: %w", err)
		}

		return nil
	}

	limit := ShadowPartitionPeekMaxBacklogs
	refillUntil := q.clock.Now().Truncate(time.Second).Add(PartitionLookahead)

	// Default to sequential but pick a random backlog every once in a while
	sequential := rand.Intn(100) <= 80

	backlogs, totalCount, err := q.ShadowPartitionPeek(ctx, shadowPart, sequential, refillUntil, limit)
	if err != nil {
		return fmt.Errorf("could not peek backlogs for shadow partition: %w", err)
	}

	// Refill backlogs in random order
	for _, idx := range util.RandPerm(len(backlogs)) {
		backlog := backlogs[idx]

		// May need to normalize - this will not happen for default backlogs
		if backlog.isOutdated(shadowPart) {
			// Prepare normalization, this will just run once as the shadow scanner
			// won't pick it up again after this.
			_, shouldNormalizeAsync, err := q.BacklogPrepareNormalize(
				ctx,
				backlog,
				shadowPart,
				q.backlogNormalizeAsyncLimit(ctx),
			)
			if err != nil {
				return fmt.Errorf("could not prepare backlog for normalization: %w", err)
			}

			// If there are just a couple of items in the backlog, we can
			// normalize right away, we have the guarantee that the backlog
			// is not being normalized right now as it wouldn't be picked up
			// by the shadow scanner otherwise.
			if !shouldNormalizeAsync {
				if err := q.leaseBacklogForNormalization(ctx, backlog); err != nil {
					return err
				}

				if err := q.normalizeBacklog(ctx, backlog); err != nil {
					return fmt.Errorf("could not normalize backlog: %w", err)
				}
			}

			continue
		}

		status, _, err := q.BacklogRefill(ctx, backlog, shadowPart, refillUntil)
		if err != nil {
			return fmt.Errorf("could not refill backlog: %w", err)
		}

		// If backlog is limited by function or account-level concurrency, stop refilling
		if status == enums.QueueConstraintAccountConcurrency || status == enums.QueueConstraintFunctionConcurrency {
			q.removeShadowContinue(ctx, shadowPart, false)

			forceRequeueAt := q.clock.Now().Add(ShadowPartitionRefillCapacityReachedRequeueExtension)
			err = q.ShadowPartitionRequeue(ctx, shadowPart, *leaseID, &forceRequeueAt)
			if err != nil {
				return fmt.Errorf("could not requeue shadow partition: %w", err)
			}

			return nil
		}
	}

	hasMoreBacklogs := int(totalCount) > len(backlogs)
	if !hasMoreBacklogs {
		// No more backlogs right now, we can continue the scan loop until new items are added
		q.removeShadowContinue(ctx, shadowPart, false)

		err = q.ShadowPartitionRequeue(ctx, shadowPart, *leaseID, nil)
		if err != nil {
			return fmt.Errorf("could not requeue shadow partition: %w", err)
		}
	}

	// More backlogs, we can add a continuation
	q.addShadowContinue(ctx, shadowPart, continuationCount)

	// Clear out current lease
	err = q.ShadowPartitionRequeue(ctx, shadowPart, *leaseID, nil)
	if err != nil {
		return fmt.Errorf("could not requeue shadow partition: %w", err)
	}

	return nil
}

func (q *queue) scanShadowContinuations(ctx context.Context) error {
	if !q.runMode.ShadowContinuations {
		// continuations are not enabled.
		return nil
	}

	// Have some chance of skipping continuations in this iteration.
	if rand.Float64() <= consts.QueueContinuationSkipProbability {
		return nil
	}

	eg := errgroup.Group{}

	q.shadowContinuesLock.Lock()
	defer q.shadowContinuesLock.Unlock()

	// If we have continued partitions, process those immediately.
	for _, c := range q.shadowContinues {
		cont := c
		eg.Go(func() error {
			p := cont.shadowPart

			if err := q.processShadowPartition(ctx, p, cont.count); err != nil {
				if !errors.Is(err, context.Canceled) {
					q.log.Error("error processing shadow partition", "error", err)
				}
				return err
			}

			return nil
		})
	}
	return eg.Wait()
}

func (q *queue) scanShadowPartitions(ctx context.Context, until time.Time, qspc chan *QueueShadowPartition) error {
	// If there are shadow continuations, process those immediately.
	if err := q.scanShadowContinuations(ctx); err != nil {
		return fmt.Errorf("error scanning shadow continuations: %w", err)
	}

	// TODO introduce weight probability to blend account/global scanning
	shouldScanAccount := q.runMode.AccountShadowPartition && rand.Intn(100) <= q.runMode.AccountShadowPartitionWeight
	if shouldScanAccount {
		sequential := false
		peekedAccounts, err := q.peekGlobalShadowPartitionAccounts(ctx, sequential, until, ShadowPartitionAccountPeekMax)
		if err != nil {
			return fmt.Errorf("could not peek global shadow partition accounts: %w", err)
		}

		if len(peekedAccounts) == 0 {
			return nil
		}

		// Reduce number of peeked partitions as we're processing multiple accounts in parallel
		// Note: This is not optimal as some accounts may have fewer partitions than others and
		// we're leaving capacity on the table. We'll need to find a better way to determine the
		// optimal peek size in this case.
		accountPartitionPeekMax := int64(math.Round(float64(ShadowPartitionPeekMax / int64(len(peekedAccounts)))))

		// Scan and process account partitions in parallel
		wg := sync.WaitGroup{}
		for _, account := range peekedAccounts {
			account := account

			wg.Add(1)
			go func(account uuid.UUID) {
				defer wg.Done()
				partitionKey := q.primaryQueueShard.RedisClient.kg.AccountShadowPartitions(account)

				parts, err := q.peekShadowPartitions(ctx, partitionKey, sequential, accountPartitionPeekMax, until)
				if err != nil {
					q.logger.Error().Err(err).Msg("error processing account partitions")
					return
				}

				for _, part := range parts {
					qspc <- part
				}
			}(account)
		}

		wg.Wait()

		return nil
	}

	kg := q.primaryQueueShard.RedisClient.kg
	sequential := false
	parts, err := q.peekShadowPartitions(ctx, kg.GlobalShadowPartitionSet(), sequential, ShadowPartitionPeekMax, until)
	if err != nil {
		return fmt.Errorf("could not peek global shadow partitions: %w", err)
	}

	for _, part := range parts {
		qspc <- part
	}

	return nil
}

// shadowScan iterates through the shadow partitions and attempt to add queue items
// to the function partition for processing
func (q *queue) shadowScan(ctx context.Context) error {
	l := logger.StdlibLogger(ctx)
	qspc := make(chan *QueueShadowPartition)

	for i := int32(0); i < q.numShadowWorkers; i++ {
		go q.shadowWorker(ctx, qspc)
	}

	tick := q.clock.NewTicker(q.pollTick)
	l.Debug("starting shadow scanner", "poll", q.pollTick.String())

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return nil

		case <-tick.Chan():
			if err := q.scanShadowPartitions(ctx, q.clock.Now(), qspc); err != nil {
				return fmt.Errorf("could not scan shadow partitions: %w", err)
			}
		}
	}
}

// peekShadowPartitions returns pending shadow partitions within the global shadow partition pointer _or_ account shadow partition pointer ZSET.
func (q *queue) peekShadowPartitions(ctx context.Context, partitionIndexKey string, sequential bool, peekLimit int64, until time.Time) ([]*QueueShadowPartition, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for peekShadowPartitions: %s", q.primaryQueueShard.Kind)
	}

	p := peeker[QueueShadowPartition]{
		q:               q,
		opName:          "peekShadowPartitions",
		keyMetadataHash: q.primaryQueueShard.RedisClient.kg.ShadowPartitionMeta(),
		max:             ShadowPartitionPeekMax,
		maker: func() *QueueShadowPartition {
			return &QueueShadowPartition{}
		},
		handleMissingItems: func(pointers []string) error {
			logger.StdlibLogger(ctx).Warn("found missing shadow partitions", "missing", pointers, "partitionKey", partitionIndexKey)

			return nil
		},
	}

	res, err := p.peek(ctx, partitionIndexKey, sequential, until, peekLimit)
	if err != nil {
		if errors.Is(err, ErrPeekerPeekExceedsMaxLimits) {
			return nil, ErrShadowPartitionPeekMaxExceedsLimits
		}
		return nil, fmt.Errorf("could not peek shadow partitions: %w", err)
	}

	return res.Items, nil
}

// addShadowContinue is the equivalent of addContinue for shadow partitions
func (q *queue) addShadowContinue(ctx context.Context, p *QueueShadowPartition, ctr uint) {
	if !q.runMode.ShadowContinuations {
		// shadow continuations are not enabled.
		return
	}

	if ctr >= q.shadowContinuationLimit {
		q.removeShadowContinue(ctx, p, true)
		return
	}

	q.shadowContinuesLock.Lock()
	defer q.shadowContinuesLock.Unlock()

	// If this is the first shadow continuation, check if we're on a cooldown, or if we're
	// beyond capacity.
	if ctr == 1 {
		if len(q.shadowContinues) > consts.QueueShadowContinuationMaxPartitions {
			metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"op": "max_capacity"}})
			return
		}
		if t, ok := q.shadowContinueCooldown[p.PartitionID]; ok && t.After(time.Now()) {
			metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"op": "cooldown"}})
			return
		}

		// Remove the shadow continuation cooldown.
		delete(q.shadowContinueCooldown, p.PartitionID)
	}

	c, ok := q.shadowContinues[p.PartitionID]
	if !ok || c.count < ctr {
		// Update the continue count if it doesn't exist, or the current counter
		// is higher.  This ensures that we always have the highest continuation
		// count stored for queue processing.
		q.shadowContinues[p.PartitionID] = shadowContinuation{shadowPart: p, count: ctr}
		metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"op": "added"}})
	}
}

func (q *queue) removeShadowContinue(ctx context.Context, p *QueueShadowPartition, cooldown bool) {
	if !q.runMode.ShadowContinuations {
		// shadow continuations are not enabled.
		return
	}

	// This is over the limit for continuing the shadow partition, so force it to be
	// removed in every case.
	q.shadowContinuesLock.Lock()
	defer q.shadowContinuesLock.Unlock()

	metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"op": "removed"}})

	delete(q.shadowContinues, p.PartitionID)

	if cooldown {
		// Add a cooldown, preventing this partition from being added as a continuation
		// for a given period of time.
		//
		// Note that this isn't shared across replicas;  cooldowns
		// only exist in the current replica.
		q.shadowContinueCooldown[p.PartitionID] = time.Now().Add(
			consts.QueueShadowContinuationCooldownPeriod,
		)
	}
}
