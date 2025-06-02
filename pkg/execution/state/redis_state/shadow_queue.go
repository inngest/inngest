package redis_state

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
)

const (
	ShadowPartitionAccountPeekMax  = int64(30)
	ShadowPartitionPeekMax         = int64(300) // same as PartitionPeekMax for now
	ShadowPartitionPeekMinBacklogs = int64(10)
	ShadowPartitionPeekMaxBacklogs = int64(100)

	ShadowPartitionRequeueExtendedDuration = 5 * time.Second
)

// shadowWorker runs a blocking process that listens to item being pushed into the
// shadow queue partition channel. This allows us to process an individual shadow partition.
func (q *queue) shadowWorker(ctx context.Context, qspc chan shadowPartitionChanMsg) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-qspc:
			_, err := durationWithTags(
				ctx,
				q.primaryQueueShard.Name,
				"shadow_partition_process_duration",
				q.clock.Now(),
				func(ctx context.Context) (any, error) {
					err := q.processShadowPartition(ctx, msg.sp, msg.continuationCount)
					return nil, err
				},
				map[string]any{"partition_id": msg.sp.PartitionID},
			)
			if err != nil {
				q.log.Error("could not scan shadow partition", "error", err, "shadow_part", msg.sp, "continuation_count", msg.continuationCount)
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
		q.removeShadowContinue(ctx, shadowPart, false)
		if errors.Is(err, ErrShadowPartitionAlreadyLeased) {
			metrics.IncrQueueShadowPartitionLeaseContentionCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard":  q.primaryQueueShard.Name,
					"partition_id": shadowPart.PartitionID,
				},
			})
			return nil
		}
		if errors.Is(err, ErrShadowPartitionNotFound) {
			metrics.IncrQueueShadowPartitionGoneCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard":  q.primaryQueueShard.Name,
					"partition_id": shadowPart.PartitionID,
				},
			})
			return nil
		}
		if errors.Is(err, ErrShadowPartitionPaused) {
			return nil
		}
		return fmt.Errorf("error leasing shadow partition: %w", err)
	}

	if leaseID == nil {
		return fmt.Errorf("missing shadow partition leaseID")
	}

	metrics.ActiveShadowScannerCount(ctx, 1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})
	defer metrics.ActiveShadowScannerCount(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})

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

	// Scan a little further into the future
	refillUntil := q.clock.Now().Truncate(time.Second).Add(2 * PartitionLookahead)

	// Pick a random backlog offset every time
	sequential := false

	backlogs, totalCount, err := q.ShadowPartitionPeek(ctx, shadowPart, sequential, refillUntil, limit)
	if err != nil {
		return fmt.Errorf("could not peek backlogs for shadow partition: %w", err)
	}
	metrics.GaugeShadowPartitionSize(ctx, int64(totalCount), metrics.GaugeOpt{
		PkgName: pkgName,
		Tags:    map[string]any{"partition_id": shadowPart.PartitionID},
	})

	// q.log.Trace("processing backlogs",
	// 	"partition_id", shadowPart.PartitionID,
	// 	"until", refillUntil.Format(time.StampMilli),
	// 	"backlogs", len(backlogs),
	// )

	// Refill backlogs in random order
	fullyProcessedBacklogs := 0
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
				if _, err := duration(ctx, q.primaryQueueShard.Name, "normalize_lease", q.clock.Now(), func(ctx context.Context) (any, error) {
					err := q.leaseBacklogForNormalization(ctx, backlog)
					return nil, err
				}); err != nil {
					return err
				}

				if err := q.normalizeBacklog(ctx, backlog, shadowPart); err != nil {
					return fmt.Errorf("could not normalize backlog: %w", err)
				}
			}

			continue
		}

		res, err := durationWithTags(
			ctx,
			q.primaryQueueShard.Name,
			"backlog_process_duration",
			q.clock.Now(),
			func(ctx context.Context) (*BacklogRefillResult, error) {
				return q.BacklogRefill(ctx, backlog, shadowPart, refillUntil)
			},
			map[string]any{"partition_id": shadowPart.PartitionID},
		)
		if err != nil {
			return fmt.Errorf("could not refill backlog: %w", err)
		}

		q.log.Trace("processed backlog",
			"backlog", backlog.BacklogID,
			"total", res.TotalBacklogCount,
			"until", res.BacklogCountUntil,
			"constrained", res.Constraint,
			"capacity", res.Capacity,
			"refill", res.Refill,
			"refilled", res.Refilled,
			"throttle", shadowPart.Throttle,
		)

		// instrumentation
		{
			opts := metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard":  q.primaryQueueShard.Name,
					"partition_id": shadowPart.PartitionID,
				},
			}

			metrics.IncrBacklogProcessedCounter(ctx, opts)
			metrics.IncrQueueBacklogRefilledCounter(ctx, int64(res.Refilled), opts)

			switch res.Constraint {
			case enums.QueueConstraintNotLimited: // no-op
			default:
				metrics.IncrQueueBacklogRefillConstraintCounter(ctx, metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"queue_shard":  q.primaryQueueShard.Name,
						"partition_id": shadowPart.PartitionID,
						"constraint":   res.Constraint.String(),
					},
				})
			}

			// NOTE: custom method to instrument result - potentially handling high cardinality data
			q.instrumentBacklogResult(ctx, backlog, res)
		}

		var forceRequeueShadowPartitionAt time.Time
		var forceRequeueBacklogAt time.Time

		// If backlog is limited by function or account-level concurrency, stop refilling
		isConcurrencyLimited := res.Constraint == enums.QueueConstraintAccountConcurrency ||
			res.Constraint == enums.QueueConstraintFunctionConcurrency
		if isConcurrencyLimited {
			forceRequeueShadowPartitionAt = q.clock.Now().Add(PartitionConcurrencyLimitRequeueExtension)
		}

		// If backlog is concurrency limited by custom key, requeue just this backlog in the future
		if res.Constraint == enums.QueueConstraintCustomConcurrencyKey1 || res.Constraint == enums.QueueConstraintCustomConcurrencyKey2 {
			forceRequeueBacklogAt = backlog.requeueBackOff(q.clock.Now(), res.Constraint, shadowPart)
		}

		// If backlog is throttled, requeue just this backlog in the future
		if res.Constraint == enums.QueueConstraintThrottle {
			forceRequeueBacklogAt = backlog.requeueBackOff(q.clock.Now(), res.Constraint, shadowPart)
		}

		if !forceRequeueBacklogAt.IsZero() {
			err = q.BacklogRequeue(ctx, backlog, shadowPart, forceRequeueBacklogAt)
			if err != nil && !errors.Is(err, ErrBacklogNotFound) {
				return fmt.Errorf("could not requeue shadow partition: %w", err)
			}
		}

		if !forceRequeueShadowPartitionAt.IsZero() {
			q.removeShadowContinue(ctx, shadowPart, false)

			err = q.ShadowPartitionRequeue(ctx, shadowPart, *leaseID, &forceRequeueShadowPartitionAt)
			if err != nil {
				return fmt.Errorf("could not requeue shadow partition: %w", err)
			}

			return nil
		}

		remainingItems := res.TotalBacklogCount - res.Refilled
		if remainingItems == 0 {
			fullyProcessedBacklogs++
		}
	}

	metrics.IncrQueueShadowPartitionProcessedCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard": q.primaryQueueShard.Name,
		},
	})

	hasMoreBacklogs := totalCount > fullyProcessedBacklogs
	if !hasMoreBacklogs {
		// No more backlogs right now, we can continue the scan loop until new items are added
		q.removeShadowContinue(ctx, shadowPart, false)

		err = q.ShadowPartitionRequeue(ctx, shadowPart, *leaseID, nil)
		if err != nil {
			return fmt.Errorf("could not requeue shadow partition: %w", err)
		}

		return nil
	}

	// More backlogs, we can add a continuation
	q.addShadowContinue(ctx, shadowPart, continuationCount+1)

	// Clear out current lease
	requeueAt := q.clock.Now().Add(ShadowPartitionRequeueExtendedDuration)
	err = q.ShadowPartitionRequeue(ctx, shadowPart, *leaseID, &requeueAt)
	if err != nil {
		return fmt.Errorf("could not requeue shadow partition: %w", err)
	}

	return nil
}

type shadowPartitionChanMsg struct {
	sp                *QueueShadowPartition
	continuationCount uint
}

func (q *queue) scanShadowContinuations(ctx context.Context) error {
	if !q.runMode.ShadowContinuations {
		return nil
	}

	if mrand.Float64() <= q.runMode.ShadowContinuationSkipProbability {
		return nil
	}

	eg := errgroup.Group{}
	q.shadowContinuesLock.Lock()
	for _, c := range q.shadowContinues {
		cont := c
		eg.Go(func() error {
			sp := cont.shadowPart

			if err := q.processShadowPartition(ctx, sp, c.count+1); err != nil {
				if err == ErrShadowPartitionLeaseNotFound {
					q.removeShadowContinue(ctx, sp, false)
					return nil
				}
				if !errors.Is(err, context.Canceled) {
					q.log.Error("error processing shadow partition", "error", err, "continue", true)
				}
				return err
			}

			metrics.IncrQueueShadowPartitionProcessedCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": q.primaryQueueShard.Name,
				},
			})
			return nil
		})
	}
	q.shadowContinuesLock.Unlock()
	return eg.Wait()
}

func (q *queue) scanShadowPartitions(ctx context.Context, until time.Time, qspc chan shadowPartitionChanMsg) error {
	// check whether continuations are enabled and apply chance of skipping continuations in this iteration
	if err := q.scanShadowContinuations(ctx); err != nil {
		return fmt.Errorf("error scanning shadow continuations: %w", err)
	}

	shouldScanAccount := q.runMode.AccountShadowPartition && mrand.Intn(100) <= q.runMode.AccountShadowPartitionWeight
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
					q.log.Error("error processing account partitions", "error", err)
					return
				}

				for _, part := range parts {
					qspc <- shadowPartitionChanMsg{
						sp:                part,
						continuationCount: 0,
					}
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
		qspc <- shadowPartitionChanMsg{
			sp:                part,
			continuationCount: 0,
		}
	}

	return nil
}

// shadowScan iterates through the shadow partitions and attempt to add queue items
// to the function partition for processing
func (q *queue) shadowScan(ctx context.Context) error {
	qspc := make(chan shadowPartitionChanMsg)

	for i := int32(0); i < q.numShadowWorkers; i++ {
		go q.shadowWorker(ctx, qspc)
	}

	tick := q.clock.NewTicker(q.pollTick)
	q.log.Debug("starting shadow scanner", "poll", q.pollTick.String())

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return nil

		case <-tick.Chan():
			// Scan a little further into the future
			now := q.clock.Now()
			scanUntil := now.Truncate(time.Second).Add(2 * PartitionLookahead)
			if err := q.scanShadowPartitions(ctx, scanUntil, qspc); err != nil {
				return fmt.Errorf("could not scan shadow partitions: %w", err)
			}

			// q.log.Trace("scan loop",
			// 	"start", now.Format(time.StampMilli),
			// 	"until", scanUntil.Format(time.StampMilli),
			// 	"dur", q.clock.Now().Sub(now).String(),
			// )
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
			q.log.Warn("found missing shadow partitions", "missing", pointers, "partitionKey", partitionIndexKey)

			return nil
		},
		isMillisecondPrecision: true,
	}

	res, err := p.peek(ctx, partitionIndexKey, sequential, until, peekLimit)
	if err != nil {
		if errors.Is(err, ErrPeekerPeekExceedsMaxLimits) {
			return nil, ErrShadowPartitionPeekMaxExceedsLimits
		}
		return nil, fmt.Errorf("could not peek shadow partitions: %w", err)
	}

	if res.TotalCount > 0 {
		for _, p := range res.Items {
			q.log.Trace("peeked shadow partition", "partition_id", p.PartitionID, "until", until.Format(time.StampMilli))
		}
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
			metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name, "op": "max_capacity"}})
			return
		}
		if t, ok := q.shadowContinueCooldown[p.PartitionID]; ok && t.After(time.Now()) {
			metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name, "op": "cooldown"}})
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
		metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name, "op": "added"}})
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

	metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name, "op": "removed"}})

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

func (q *queue) ShadowPartitionPeek(ctx context.Context, sp *QueueShadowPartition, sequential bool, until time.Time, limit int64) ([]*QueueBacklog, int, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, 0, fmt.Errorf("unsupported queue shard kind for ShadowPartitionPeek: %s", q.primaryQueueShard.Kind)
	}

	rc := q.primaryQueueShard.RedisClient

	shadowPartitionSet := rc.kg.ShadowPartitionSet(sp.PartitionID)

	p := peeker[QueueBacklog]{
		q:               q,
		opName:          "ShadowPartitionPeek",
		keyMetadataHash: q.primaryQueueShard.RedisClient.kg.BacklogMeta(),
		max:             ShadowPartitionPeekMaxBacklogs,
		maker: func() *QueueBacklog {
			return &QueueBacklog{}
		},
		handleMissingItems: func(pointers []string) error {
			err := rc.Client().Do(ctx, rc.Client().B().Zrem().Key(shadowPartitionSet).Member(pointers...).Build()).Error()
			if err != nil {
				q.log.Warn("failed to clean up dangling backlogs from shard partition",
					"missing", pointers,
					"sp", sp,
				)
			}

			return nil
		},
		isMillisecondPrecision: true,
	}

	res, err := p.peek(ctx, shadowPartitionSet, sequential, until, limit)
	if err != nil {
		if errors.Is(err, ErrPeekerPeekExceedsMaxLimits) {
			return nil, 0, ErrShadowPartitionBacklogPeekMaxExceedsLimits
		}
		return nil, 0, fmt.Errorf("could not peek shadow partition backlogs: %w", err)
	}

	return res.Items, res.TotalCount, nil
}

func (q *queue) ShadowPartitionExtendLease(ctx context.Context, sp *QueueShadowPartition, leaseID ulid.ULID, duration time.Duration) (*ulid.ULID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ShadowPartitionExtendLease"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for ShadowPartitionExtendLease: %s", q.primaryQueueShard.Kind)
	}

	now := q.clock.Now()
	leaseExpiry := now.Add(duration)
	newLeaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("could not generate new leaseID: %w", err)
	}

	sp.LeaseID = &newLeaseID

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	keys := []string{
		q.primaryQueueShard.RedisClient.kg.ShadowPartitionMeta(),
		q.primaryQueueShard.RedisClient.kg.GlobalShadowPartitionSet(),
		q.primaryQueueShard.RedisClient.kg.GlobalAccountShadowPartitions(),
		q.primaryQueueShard.RedisClient.kg.AccountShadowPartitions(accountID),
	}
	args, err := StrSlice([]any{
		sp.PartitionID,
		accountID,
		leaseID,
		newLeaseID,
		now.UnixMilli(),
		leaseExpiry.UnixMilli(),
	})
	if err != nil {
		return nil, fmt.Errorf("could not serialize args: %w", err)
	}

	status, err := scripts["queue/shadowPartitionExtendLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "shadowPartitionExtendLease"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error extending shadow partition lease: %w", err)
	}
	switch status {
	case 0:
		return &newLeaseID, nil
	case -1:
		return nil, ErrShadowPartitionNotFound
	case -2:
		return nil, ErrShadowPartitionLeaseNotFound
	case -3:
		return nil, ErrShadowPartitionAlreadyLeased
	case -4:
		return nil, ErrShadowPartitionPaused
	default:
		return nil, fmt.Errorf("unknown response extending shadow partition lease: %v (%T)", status, status)
	}
}

func (q *queue) ShadowPartitionRequeue(ctx context.Context, sp *QueueShadowPartition, leaseID ulid.ULID, requeueAt *time.Time) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ShadowPartitionRequeue"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for ShadowPartitionRequeue: %s", q.primaryQueueShard.Kind)
	}

	sp.LeaseID = nil

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	var requeueAtMS int64
	var requeueAtStr string
	if requeueAt != nil {
		requeueAtMS = requeueAt.UnixMilli()
		requeueAtStr = requeueAt.Format(time.StampMilli)
	}

	keys := []string{
		q.primaryQueueShard.RedisClient.kg.ShadowPartitionMeta(),
		q.primaryQueueShard.RedisClient.kg.GlobalShadowPartitionSet(),
		q.primaryQueueShard.RedisClient.kg.GlobalAccountShadowPartitions(),
		q.primaryQueueShard.RedisClient.kg.AccountShadowPartitions(accountID),
		q.primaryQueueShard.RedisClient.kg.ShadowPartitionSet(sp.PartitionID),
	}
	args, err := StrSlice([]any{
		sp.PartitionID,
		accountID,
		leaseID,
		q.clock.Now().UnixMilli(),
		requeueAtMS,
	})
	if err != nil {
		return fmt.Errorf("could not serialize args: %w", err)
	}

	status, err := scripts["queue/shadowPartitionRequeue"].Exec(
		redis_telemetry.WithScriptName(ctx, "shadowPartitionRequeue"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error returning shadow partition lease: %w", err)
	}

	q.log.Trace("requeued shadow partition",
		"id", sp.PartitionID,
		"time", requeueAtStr,
		"status", status,
	)

	switch status {
	case 0:
		return nil
	case -1:
		return ErrShadowPartitionNotFound
	case -2:
		return ErrShadowPartitionAlreadyLeased
	case -3:
		return ErrShadowPartitionLeaseNotFound
	default:
		return fmt.Errorf("unknown response returning shadow partition lease: %v (%T)", status, status)
	}
}
