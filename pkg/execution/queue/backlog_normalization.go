package queue

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/sourcegraph/conc/pool"
	"golang.org/x/sync/errgroup"
)

type normalizeWorkerChanMsg struct {
	b           *QueueBacklog
	sp          *QueueShadowPartition
	constraints PartitionConstraintConfig
}

// backlogNormalizationWorker runs a blocking process that listens to item being pushed into the normalization partition. This allows us to process individual
// backlogs that need to be normalized
func (q *queueProcessor) backlogNormalizationWorker(ctx context.Context, nc chan normalizeWorkerChanMsg) {
	l := logger.StdlibLogger(ctx)
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-nc:
			_, err := DurationWithTags(ctx, q.name, "normalize_backlog", q.Clock.Now(), func(ctx context.Context) (any, error) {
				err := q.normalizeBacklog(ctx, msg.b, msg.sp, msg.constraints)
				return nil, err
			}, map[string]any{
				"async_processing": true,
			})
			if err != nil {
				l.Error("could not normalize backlog", "error", err, "backlog", msg.b, "shadow", msg.sp)
			}
		}
	}
}

// backlogNormalizationScan iterates through a partition of backlogs and reenqueue
// the items to the appropriate backlogs
func (q *queueProcessor) backlogNormalizationScan(ctx context.Context) error {
	l := logger.StdlibLogger(ctx).With("method", "backlogNormalizationScan")
	bc := make(chan normalizeWorkerChanMsg)

	for i := int32(0); i < q.numBacklogNormalizationWorkers; i++ {
		go q.backlogNormalizationWorker(ctx, bc)
	}

	tick := q.Clock.NewTicker(q.backlogNormalizePollTick)
	l.Debug("starting normalization scanner", "poll", q.backlogNormalizePollTick.String())

	backoff := 200 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return nil

		case <-tick.Chan():
			until := q.Clock.Now()

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
func (q *queueProcessor) iterateNormalizationPartition(ctx context.Context, until time.Time, bc chan normalizeWorkerChanMsg) error {
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

func (q *queueProcessor) iterateNormalizationShadowPartition(ctx context.Context, shadowPartitionIndexKey string, peekLimit int64, until time.Time, bc chan normalizeWorkerChanMsg) error {
	// Find partitions in account or globally with backlogs to normalize
	sequential := false
	shadowPartitions, err := q.peekShadowPartitions(ctx, shadowPartitionIndexKey, sequential, peekLimit, until)
	if err != nil {
		return fmt.Errorf("could not peek shadow partitions to normalize: %w", err)
	}

	// For each partition, attempt to normalize backlogs
	for _, partition := range shadowPartitions {
		backlogs, err := Duration(ctx, q.primaryQueueShard.Name(), "normalize_peek", until, func(ctx context.Context) ([]*QueueBacklog, error) {
			return q.ShadowPartitionPeekNormalizeBacklogs(ctx, partition, NormalizePartitionPeekMax)
		})
		if err != nil {
			return err
		}

		constraints := q.PartitionConstraintConfigGetter(ctx, partition.Identifier())

		for _, bl := range backlogs {
			// lease the backlog
			_, err := Duration(ctx, q.primaryQueueShard.Name, "normalize_lease", q.Clock.Now(), func(ctx context.Context) (any, error) {
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

// normalizeBacklog must be called with exclusive access to the shadow partition
// NOTE: ideally this is one transaction in a lua script but enqueue_to_backlog is way too much work to
// utilize
func (q *queueProcessor) normalizeBacklog(ctx context.Context, backlog *QueueBacklog, sp *QueueShadowPartition, latestConstraints PartitionConstraintConfig) error {
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
				if err := q.extendBacklogNormalizationLease(ctx, q.Clock.Now(), backlog); err != nil {
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
							"queue_shard": q.primaryQueueShard.Name(),
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

func (q *queueProcessor) normalizeItem(
	ctx context.Context,
	sp *QueueShadowPartition,
	latestConstraints PartitionConstraintConfig,
	sourceBacklog *QueueBacklog,
	item QueueItem,
) (QueueItem, error) {
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
		err := q.primaryQueueShard.Dequeue(ctx, item)
		if err != nil {
			log.Warn("could not dequeue queue item with missing event", "err", err)
		}
	}

	refreshedCustomConcurrencyKeys, err := q.NormalizeRefreshItemCustomConcurrencyKeys(ctx, &item, existingKeys, sp)
	if err != nil {
		// If event for item cannot be found, remove it from the backlog
		if errors.Is(err, state.ErrEventNotFound) {
			cleanupItem()
			return QueueItem{}, nil
		}
		return QueueItem{}, fmt.Errorf("could not refresh custom concurrency keys for item: %w", err)
	}

	item.Data.CustomConcurrencyKeys = refreshedCustomConcurrencyKeys
	item.Data.Identifier.CustomConcurrencyKeys = nil
	log = log.With("refreshed_concurrency", refreshedCustomConcurrencyKeys)

	refreshedThrottle, err := q.RefreshItemThrottle(ctx, &item)
	if err != nil {
		// If event for item cannot be found, remove it from the backlog
		if errors.Is(err, state.ErrEventNotFound) {
			cleanupItem()
			return QueueItem{}, nil
		}
		return QueueItem{}, fmt.Errorf("could not refresh throttle for item: %w", err)
	}

	item.Data.Throttle = refreshedThrottle
	log = log.With("refreshed_throttle", refreshedThrottle)

	targetBacklog := ItemBacklog(ctx, item)
	log = log.With("target", targetBacklog)

	if reason := targetBacklog.IsOutdated(latestConstraints); reason != enums.QueueNormalizeReasonUnchanged {
		log.Warn("target backlog in normalization is outdated, this likely causes infinite normalization")
	}

	log.Debug("retrieved refreshed backlog")

	if _, err := q.primaryQueueShard.EnqueueItem(ctx, item, time.UnixMilli(item.AtMS), EnqueueOpts{
		PassthroughJobId:       true,
		NormalizeFromBacklogID: sourceBacklog.BacklogID,
	}); err != nil {
		return QueueItem{}, fmt.Errorf("could not re-enqueue backlog item: %w", err)
	}

	return item, nil
}
