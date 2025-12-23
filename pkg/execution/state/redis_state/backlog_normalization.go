package redis_state

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/sourcegraph/conc/pool"
	"golang.org/x/sync/errgroup"
)

const (
	// NormalizeAccountPeekMax sets the maximum number of accounts that can be peeked from the global normalization index.
	NormalizeAccountPeekMax = int64(30)
	// NormalizePartitionPeekMax sets the maximum number of backlogs that can be peeked from the shadow partition.
	NormalizePartitionPeekMax = int64(100)
	// NormalizeBacklogPeekMax sets the maximum number of items that can be peeked from a backlog during normalization.
	NormalizeBacklogPeekMax = int64(100) // same as ShadowPartitionPeekMax

	// BacklogRefillHardLimit sets the maximum number of items that can be refilled in a single backlogRefill operation.
	BacklogRefillHardLimit = int64(1000)

	// BacklogNormalizeHardLimit sets the batch size of items to be reenqueued into the appropriate backlogs durign normalization
	BacklogNormalizeHardLimit = int64(1000)
)

var (
	errBacklogNormalizationLeaseExpired     = fmt.Errorf("backlog normalization lease expired")
	errBacklogAlreadyLeasedForNormalization = fmt.Errorf("backlog already leased for normalization")
)

type normalizeWorkerChanMsg struct {
	b           *QueueBacklog
	sp          *QueueShadowPartition
	constraints PartitionConstraintConfig
}

// backlogNormalizationWorker runs a blocking process that listens to item being pushed into the normalization partition. This allows us to process individual
// backlogs that need to be normalized
func (q *queue) backlogNormalizationWorker(ctx context.Context, nc chan normalizeWorkerChanMsg) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-nc:
			_, err := durationWithTags(ctx, q.primaryQueueShard.Name, "normalize_backlog", q.clock.Now(), func(ctx context.Context) (any, error) {
				err := q.normalizeBacklog(ctx, msg.b, msg.sp, msg.constraints)
				return nil, err
			}, map[string]any{
				"async_processing": true,
			})
			if err != nil {
				q.log.Error("could not normalize backlog", "error", err, "backlog", msg.b, "shadow", msg.sp)
			}
		}
	}
}

// backlogNormalizationScan iterates through a partition of backlogs and reenqueue
// the items to the appropriate backlogs
func (q *queue) backlogNormalizationScan(ctx context.Context) error {
	l := q.log.With("method", "backlogNormalizationScan")
	bc := make(chan normalizeWorkerChanMsg)

	for i := int32(0); i < q.numBacklogNormalizationWorkers; i++ {
		go q.backlogNormalizationWorker(ctx, bc)
	}

	tick := q.clock.NewTicker(q.backlogNormalizePollTick)
	l.Debug("starting normalization scanner", "poll", q.backlogNormalizePollTick.String())

	backoff := 200 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return nil

		case <-tick.Chan():
			until := q.clock.Now()

			if err := q.iterateNormalizationPartition(ctx, until, bc); err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					l.Warn("deadline exceeded scanning backlog normalization partition")
					<-time.After(backoff)

					// Backoff doubles up to 5 seconds
					backoff = time.Duration(math.Min(float64(backoff*2), float64(5*time.Second)))
					continue
				}

				if !errors.Is(err, context.Canceled) {
					l.Error("error scanning backlog normalization partitions", "error", err)
				}

				return fmt.Errorf("error scanning global normalization partition: %w", err)
			}

			backoff = 200 * time.Millisecond
		}
	}
}

// iterateNormalizationPartition scans and iterate through the global normalization partition to process backlogs needing to be normalized
func (q *queue) iterateNormalizationPartition(ctx context.Context, until time.Time, bc chan normalizeWorkerChanMsg) error {
	// introduce weight probability to blend account/global scanning
	peekedAccounts, err := q.peekGlobalNormalizeAccounts(ctx, until, NormalizeAccountPeekMax)
	if err != nil {
		return fmt.Errorf("could not peek global normalize accounts: %w", err)
	}

	if len(peekedAccounts) == 0 {
		return nil
	}

	// Reduce number of peeked partitions as we're processing multiple accounts in parallel
	// Note: This is not optimal as some accounts may have fewer partitions than others and
	// we're leaving capacity on the table. We'll need to find a better way to determine the
	// optimal peek size in this case.
	accountShadowPartitionPeekMax := int64(math.Round(float64(ShadowPartitionPeekMax / int64(len(peekedAccounts)))))

	// Scan and process account shadow partitions in parallel
	eg := errgroup.Group{}
	for _, account := range peekedAccounts {
		partitionKey := q.primaryQueueShard.RedisClient.kg.AccountNormalizeSet(account)

		eg.Go(func() error {
			return q.iterateNormalizationShadowPartition(ctx, partitionKey, accountShadowPartitionPeekMax, until, bc)
		})
	}

	err = eg.Wait()
	if err != nil {
		return fmt.Errorf("failed to scan and normalize backlogs for accounts: %w", err)
	}

	return nil
}

func (q *queue) iterateNormalizationShadowPartition(ctx context.Context, shadowPartitionIndexKey string, peekLimit int64, until time.Time, bc chan normalizeWorkerChanMsg) error {
	// Find partitions in account or globally with backlogs to normalize
	sequential := false
	shadowPartitions, err := q.peekShadowPartitions(ctx, shadowPartitionIndexKey, sequential, peekLimit, until)
	if err != nil {
		return fmt.Errorf("could not peek shadow partitions to normalize: %w", err)
	}

	// For each partition, attempt to normalize backlogs
	for _, partition := range shadowPartitions {
		backlogs, err := duration(ctx, q.primaryQueueShard.Name, "normalize_peek", until, func(ctx context.Context) ([]*QueueBacklog, error) {
			return q.ShadowPartitionPeekNormalizeBacklogs(ctx, partition, NormalizePartitionPeekMax)
		})
		if err != nil {
			return err
		}

		constraints := q.partitionConstraintConfigGetter(ctx, partition.Identifier())

		for _, bl := range backlogs {
			// lease the backlog
			_, err := duration(ctx, q.primaryQueueShard.Name, "normalize_lease", q.clock.Now(), func(ctx context.Context) (any, error) {
				err := q.leaseBacklogForNormalization(ctx, bl)
				return nil, err
			})
			if err != nil {
				if errors.Is(err, errBacklogAlreadyLeasedForNormalization) {
					continue
				}

				return fmt.Errorf("error leasing backlog for normalization: %w", err)
			}

			metrics.IncrBacklogNormalizationScannedCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": q.primaryQueueShard.Name,
					// "partition_id": partition.PartitionID,
				},
			})

			// dump it into the channel for the workers to do their thing
			bc <- normalizeWorkerChanMsg{
				b:           bl,
				sp:          partition,
				constraints: constraints,
			}
		}
	}

	return nil
}

func (q *queue) leaseBacklogForNormalization(ctx context.Context, bl *QueueBacklog) error {
	leaseExpiry := q.clock.Now().Add(BacklogNormalizeLeaseDuration)
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return fmt.Errorf("could not generate leaseID: %w", err)
	}

	shard := q.primaryQueueShard

	rc := shard.RedisClient.Client()
	cmd := rc.B().
		Set().
		Key(shard.RedisClient.kg.BacklogNormalizationLease(bl.BacklogID)).
		Value(leaseID.String()).
		Nx().
		Get().
		Exat(leaseExpiry).
		Build()

	_, err = rc.Do(ctx, cmd).ToString()
	if err == rueidis.Nil {
		// successfully leased since prior value was nil
		return nil
	}
	if err != nil {
		return err
	}

	return errBacklogAlreadyLeasedForNormalization
}

func (q *queue) extendBacklogNormalizationLease(ctx context.Context, now time.Time, bl *QueueBacklog) error {
	leaseExpiry := now.Add(BacklogNormalizeLeaseDuration)
	newLeaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return fmt.Errorf("could not generate newLeaseID: %w", err)
	}

	shard := q.primaryQueueShard

	rc := shard.RedisClient.Client()
	cmd := rc.B().
		Set().
		Key(shard.RedisClient.kg.BacklogNormalizationLease(bl.BacklogID)).
		Value(newLeaseID.String()).
		Xx().
		Get().
		Exat(leaseExpiry).
		Build()

	_, err = rc.Do(ctx, cmd).ToAny()
	if err == rueidis.Nil {
		return errBacklogNormalizationLeaseExpired
	}
	if err != nil {
		return err
	}

	// successfully extended lease
	return nil
}

// normalizeBacklog must be called with exclusive access to the shadow partition
// NOTE: ideally this is one transaction in a lua script but enqueue_to_backlog is way too much work to
// utilize
func (q *queue) normalizeBacklog(ctx context.Context, backlog *QueueBacklog, sp *QueueShadowPartition, latestConstraints PartitionConstraintConfig) error {
	ctx, cancelNormalization := context.WithCancel(ctx)
	defer cancelNormalization()

	_, file, line, _ := runtime.Caller(1)
	caller := fmt.Sprintf("%s:%d", file, line)

	metrics.ActiveBacklogNormalizeCount(ctx, 1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})
	defer metrics.ActiveBacklogNormalizeCount(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})

	l := q.log.With(
		"backlog", backlog,
		"sp", sp,
		"constraints", latestConstraints,
		"caller", caller,
	)

	// extend the lease
	extendLeaseCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		for {
			select {
			case <-extendLeaseCtx.Done():
				return
			case <-time.Tick(BacklogNormalizeLeaseDuration / 2):
				if err := q.extendBacklogNormalizationLease(ctx, q.clock.Now(), backlog); err != nil {
					switch err {
					// can't extend since it's already expired
					case errBacklogNormalizationLeaseExpired:
						l.Debug("normalization lease expired")
						cancelNormalization()
						return
					}
					l.Error("error extending backlog normalization lease", "error", err)
					return
				}
			}
		}
	}()

	l.Debug("starting backlog normalization")

	shard := q.primaryQueueShard
	var total int64
	for {
		// If context is canceled, stop normalizing
		if ctx.Err() == context.Canceled {
			return nil
		}

		res, err := q.BacklogNormalizePeek(ctx, backlog, NormalizeBacklogPeekMax)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("could not peek backlog items for normalization: %w", err)
		}

		if res.TotalCount == 0 {
			l.Debug("no more items in backlog", res.RemovedCount)
			break
		}

		l.Debug("peeked items to normalize", "count", len(res.Items), "total", res.TotalCount, "removed", res.RemovedCount)

		wg := pool.New().WithMaxGoroutines(int(q.backlogNormalizeConcurrency))
		for _, item := range res.Items {
			item := item // capture range variable
			wg.Go(func() {
				_, err := q.normalizeItem(logger.WithStdlib(ctx, l), shard, sp, latestConstraints, backlog, *item)
				if err != nil && !errors.Is(err, context.Canceled) {
					l.ReportError(err, "could not normalize item",
						logger.WithErrorReportTags(map[string]string{
							"item_id":     item.ID,
							"account_id":  item.Data.Identifier.AccountID.String(),
							"env_id":      item.WorkspaceID.String(),
							"app_id":      item.Data.Identifier.AppID.String(),
							"fn_id":       item.FunctionID.String(),
							"backlog_id":  backlog.BacklogID,
							"queue_shard": q.primaryQueueShard.Name,
						}),
					)
				}
			})
		}

		wg.Wait()

		processed := int64(len(res.Items))

		l.Info("processed normalization for backlog",
			"processed", processed,
			"removed", res.RemovedCount,
		)

		metrics.IncrBacklogNormalizedItemCounter(ctx, processed, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"queue_shard": q.primaryQueueShard.Name,
				// "partition_id": backlog.ShadowPartitionID,
			},
		})

		total += processed
	}

	metrics.IncrBacklogNormalizedCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard": q.primaryQueueShard.Name,
			// "partition_id": backlog.ShadowPartitionID,
		},
	})

	l.Debug("normalized backlog", "processed_total", total)

	return nil
}

func (q *queue) normalizeItem(
	ctx context.Context,
	shard RedisQueueShard,
	sp *QueueShadowPartition,
	latestConstraints PartitionConstraintConfig,
	sourceBacklog *QueueBacklog,
	item osqueue.QueueItem,
) (osqueue.QueueItem, error) {
	// We must modify the queue item to ensure q.ItemBacklog and q.ItemShadowPartition
	// return the new values properly. Otherwise, we'd enqueue to the same backlog, not
	// the desired new backlog.
	existingThrottle := item.Data.Throttle
	existingKeys := item.Data.GetConcurrencyKeys()

	log := logger.StdlibLogger(ctx).With(
		"item", item,
		"existing_concurrency", existingKeys,
		"existing_throttle", existingThrottle,
	)

	cleanupItem := func() {
		// If event for item cannot be found, remove it from the backlog
		err := q.Dequeue(ctx, shard, item)
		if err != nil {
			log.Warn("could not dequeue queue item with missing event", "err", err)
		}
	}

	refreshedCustomConcurrencyKeys, err := q.normalizeRefreshItemCustomConcurrencyKeys(ctx, &item, existingKeys, sp)
	if err != nil {
		// If event for item cannot be found, remove it from the backlog
		if errors.Is(err, state.ErrEventNotFound) {
			cleanupItem()
			return osqueue.QueueItem{}, nil
		}
		return osqueue.QueueItem{}, fmt.Errorf("could not refresh custom concurrency keys for item: %w", err)
	}

	item.Data.CustomConcurrencyKeys = refreshedCustomConcurrencyKeys
	item.Data.Identifier.CustomConcurrencyKeys = nil
	log = log.With("refreshed_concurrency", refreshedCustomConcurrencyKeys)

	refreshedThrottle, err := q.refreshItemThrottle(ctx, &item)
	if err != nil {
		// If event for item cannot be found, remove it from the backlog
		if errors.Is(err, state.ErrEventNotFound) {
			cleanupItem()
			return osqueue.QueueItem{}, nil
		}
		return osqueue.QueueItem{}, fmt.Errorf("could not refresh throttle for item: %w", err)
	}

	item.Data.Throttle = refreshedThrottle
	log = log.With("refreshed_throttle", refreshedThrottle)

	targetBacklog := q.ItemBacklog(ctx, item)
	log = log.With("target", targetBacklog)

	if reason := targetBacklog.isOutdated(latestConstraints); reason != enums.QueueNormalizeReasonUnchanged {
		log.Warn("target backlog in normalization is outdated, this likely causes infinite normalization")
	}

	log.Debug("retrieved refreshed backlog")

	if _, err := q.EnqueueItem(ctx, shard, item, time.UnixMilli(item.AtMS), osqueue.EnqueueOpts{
		PassthroughJobId:       true,
		NormalizeFromBacklogID: sourceBacklog.BacklogID,
	}); err != nil {
		return osqueue.QueueItem{}, fmt.Errorf("could not re-enqueue backlog item: %w", err)
	}

	return item, nil
}

func (q *queue) ShadowPartitionPeekNormalizeBacklogs(ctx context.Context, sp *QueueShadowPartition, limit int64) ([]*QueueBacklog, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for ShadowPartitionPeekNormalizeBacklogs: %s", q.primaryQueueShard.Kind)
	}

	rc := q.primaryQueueShard.RedisClient

	partitionNormalizeSet := rc.kg.PartitionNormalizeSet(sp.PartitionID)

	p := peeker[QueueBacklog]{
		q:               q,
		opName:          "ShadowPartitionPeekNormalizeBacklogs",
		keyMetadataHash: q.primaryQueueShard.RedisClient.kg.BacklogMeta(),
		max:             NormalizePartitionPeekMax,
		maker: func() *QueueBacklog {
			return &QueueBacklog{}
		},
		handleMissingItems: CleanupMissingPointers(ctx, partitionNormalizeSet, rc.Client(), q.log.With("sp", sp)),
		// faster option: load items regardless of zscore
		ignoreUntil:            true,
		isMillisecondPrecision: true,
	}

	res, err := p.peek(ctx, partitionNormalizeSet, false, q.clock.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("could not peek backlogs for normalization: %w", err)
	}

	return res.Items, nil
}

func (q *queue) BacklogNormalizePeek(ctx context.Context, b *QueueBacklog, limit int64) (*peekResult[osqueue.QueueItem], error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for BacklogNormalizePeek: %s", q.primaryQueueShard.Kind)
	}

	rc := q.primaryQueueShard.RedisClient

	backlogSet := rc.kg.BacklogSet(b.BacklogID)

	p := peeker[osqueue.QueueItem]{
		q:               q,
		opName:          "BacklogNormalizePeek",
		keyMetadataHash: q.primaryQueueShard.RedisClient.kg.QueueItem(),
		max:             NormalizeBacklogPeekMax,
		maker: func() *osqueue.QueueItem {
			return &osqueue.QueueItem{}
		},
		handleMissingItems: CleanupMissingPointers(ctx, backlogSet, rc.Client(), q.log.With("backlog", b)),
		// faster option: load items regardless of zscore
		ignoreUntil:            true,
		isMillisecondPrecision: true,
	}

	// this is essentially +inf as no queue items should ever be scheduled >2y out
	normalizeLookahead := q.clock.Now().Add(time.Hour * 24 * 365 * 2)

	res, err := p.peek(ctx, backlogSet, false, normalizeLookahead, limit)
	if err != nil {
		return nil, fmt.Errorf("could not peek backlog items for normalization: %w", err)
	}

	return res, nil
}
