package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

// ProcessPartition processes a given partition, peeking jobs from the partition to run.
//
// It accepts a uint continuationCount which represents the number of times that the partition
// has been continued;  this occurs when a job enqueues another job to the same partition and
// hints that we have more work to do, which lowers inter-step latency on job-per-step execution
// models.
//
// randomOffset allows us to peek jobs out-of-order, and occurs when we hit concurrency key issues
// such that we can attempt to work on other jobs not blocked by heading concurrency key issues.
func (q *queueProcessor) ProcessPartition(ctx context.Context, p *QueuePartition, continuationCount uint, randomOffset bool) error {
	l := logger.StdlibLogger(ctx)

	// When Constraint API is enabled, disable capacity checks on PartitionLease.
	// This is necessary as capacity was already granted to individual items, and
	// constraints like concurrency were consumed.
	var disableLeaseChecks bool
	if p.AccountID != uuid.Nil && p.EnvID != nil && p.FunctionID != nil && q.CapacityManager != nil && q.UseConstraintAPI != nil {
		enableConstraintAPI, _ := q.UseConstraintAPI(ctx, p.AccountID, *p.EnvID, *p.FunctionID)
		disableLeaseChecks = enableConstraintAPI
	}

	// Attempt to lease items.  This checks partition-level concurrency limits
	//
	// For optimization, because this is the only thread that can be leasing
	// jobs for this partition, we store the partition limit and current count
	// as a variable and iterate in the loop without loading keys from the state
	// store.
	//
	// There's no way to know when queue items finish processing;  we don't
	// store average runtimes for queue items (and we don't know because
	// items are dynamic generators).  This means that we have to delay
	// processing the partition by N seconds, meaning the latency is increased by
	// up to this period for scheduled items behind the concurrency limits.
	_, err := Duration(ctx, q.primaryQueueShard.Name(), "partition_lease", q.Clock().Now(), func(ctx context.Context) (int, error) {
		l, capacity, err := q.primaryQueueShard.PartitionLease(ctx, p, PartitionLeaseDuration, PartitionLeaseOptionDisableLeaseChecks(disableLeaseChecks))
		p.LeaseID = l
		return capacity, err
	})
	if errors.Is(err, ErrPartitionConcurrencyLimit) {
		if p.FunctionID != nil {
			q.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *p.FunctionID)
		}
		metrics.IncrQueuePartitionConcurrencyLimitCounter(ctx,
			metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"kind": "function", "queue_shard": q.primaryQueueShard.Name()},
			},
		)
		return q.primaryQueueShard.PartitionRequeue(ctx, p, q.Clock().Now().Truncate(time.Second).Add(PartitionConcurrencyLimitRequeueExtension), true)
	}
	if errors.Is(err, ErrAccountConcurrencyLimit) {
		// For backwards compatibility, we report on the function level as well
		if p.FunctionID != nil {
			q.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *p.FunctionID)
		}
		q.lifecycles.OnAccountConcurrencyLimitReached(
			context.WithoutCancel(ctx),
			p.AccountID,
			p.EnvID,
		)
		metrics.IncrQueuePartitionConcurrencyLimitCounter(ctx,
			metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"kind": "account", "queue_shard": q.primaryQueueShard.Name()},
			},
		)
		return q.primaryQueueShard.PartitionRequeue(ctx, p, q.Clock().Now().Truncate(time.Second).Add(PartitionConcurrencyLimitRequeueExtension), true)
	}
	if errors.Is(err, ErrPartitionAlreadyLeased) {
		metrics.IncrQueuePartitionLeaseContentionCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name()}})
		// If this is a continuation, remove it from the continuation counter.
		// This prevents us from keeping partitions as continuations forever until
		// we hit the max limit.
		q.removeContinue(ctx, p, false)
		return nil
	}
	if errors.Is(err, ErrPartitionNotFound) || errors.Is(err, ErrPartitionGarbageCollected) {
		// Another worker must have processed this partition between
		// this worker's peek and process.  Increase partition
		// contention metric and continue.  This is unsolvable.

		// If this is a continuation, remove it from the continuation counter.
		// This prevents us from keeping partitions as continuations forever until
		// we hit the max limit.
		q.removeContinue(ctx, p, false)

		metrics.IncrPartitionGoneCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name()}})
		return nil
	}
	if errors.Is(err, ErrPartitionPaused) {
		// Don't return an error and remove continuations;  this isn't workable.
		q.removeContinue(ctx, p, false)
		return nil
	}

	if err != nil {
		return fmt.Errorf("error leasing partition: %w", err)
	}

	begin := q.Clock().Now()
	defer func() {
		metrics.HistogramProcessPartitionDuration(ctx, q.Clock().Since(begin).Milliseconds(), metrics.HistogramOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"queue_shard":     q.primaryQueueShard.Name(),
				"is_continuation": continuationCount > 0,
			},
		})
	}()

	// Ensure that peek doesn't take longer than the partition lease, to
	// reduce contention.
	peekCtx, cancel := context.WithTimeout(ctx, PartitionLeaseDuration)
	defer cancel()

	// We need to round ourselves up to the nearest second, then add another second
	// to peek for jobs in the next <= 1999 milliseconds.
	//
	// There's a really subtle issue:  if two jobs contend for a pause and are scheduled
	// within 5ms of each other, we fetch them in order but we may process them out of
	// order, depending on how long it takes for the item to pass through the channel
	// to the worker, how long Redis takes to lease the item, etc.
	fetch := q.Clock().Now().Truncate(time.Second).Add(PartitionLookahead)

	queue, err := Duration(peekCtx, q.primaryQueueShard.Name(), "peek", q.Clock().Now(), func(ctx context.Context) ([]*QueueItem, error) {
		peek := q.peekSize(ctx, p)
		// NOTE: would love to instrument this value to see it over time per function but
		// it's likely too high of a cardinality
		go metrics.HistogramQueuePeekEWMA(ctx, peek, metrics.HistogramOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name()}})

		if randomOffset {
			return q.primaryQueueShard.PeekRandom(peekCtx, p, fetch, peek)
		}

		return q.primaryQueueShard.Peek(peekCtx, p, fetch, peek)
	})
	if err != nil {
		return err
	}

	metrics.HistogramQueuePeekSize(ctx, int64(len(queue)), metrics.HistogramOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard":     q.primaryQueueShard.Name(),
			"is_continuation": continuationCount > 0,
		},
	})

	// Record the number of partitions we're leasing.
	metrics.IncrQueuePartitionLeasedCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard":     q.primaryQueueShard.Name(),
			"is_continuation": continuationCount > 0,
		},
	})

	// parallel all queue names with internal mappings for now.
	// XXX: Allow parallel partitions for all functions except for fns opting into FIFO
	_, isSystemFn := q.queueKindMapping[p.Queue()]
	_, parallelFn := q.disableFifoForFunctions[p.Queue()]
	_, parallelAccount := q.disableFifoForAccounts[p.AccountID.String()]

	parallel := parallelFn || parallelAccount || isSystemFn

	iter := ProcessorIterator{
		Partition:            p,
		Items:                queue,
		PartitionContinueCtr: continuationCount,
		Queue:                q,
		Denies:               NewLeaseDenyList(),
		StaticTime:           q.Clock().Now(),
		Parallel:             parallel,
	}

	if processErr := iter.Iterate(ctx); processErr != nil {
		// Report the eerror.
		l.Error("error iterating queue items", "error", processErr, "partition", p)
		return processErr

	}

	if q.usePeekEWMA {
		if err := q.primaryQueueShard.SetPeekEWMA(ctx, p.FunctionID, int64(iter.CtrConcurrency+iter.CtrRateLimit)); err != nil {
			l.Warn("error recording concurrency limit for EWMA", "error", err)
		}
	}

	if iter.IsRequeuable() && iter.IsCustomKeyLimitOnly && !randomOffset && parallel {
		// We hit custom concurrency key issues.  Re-process this partition at a random offset, as long
		// as random offset is currently false (so we don't loop forever)

		// Note: we must requeue the partition to remove the lease.
		err := q.primaryQueueShard.PartitionRequeue(ctx, p, q.Clock().Now().Truncate(time.Second).Add(PartitionConcurrencyLimitRequeueExtension), true)
		if err != nil {
			l.Warn("error requeuieng partition for random peek", "error", err)
		}

		return q.ProcessPartition(ctx, p, continuationCount, true)
	}

	// If we've hit concurrency issues OR we've only hit rate limit issues, re-enqueue the partition
	// with a force:  ensure that we won't re-scan it until 2 seconds in the future.
	if iter.IsRequeuable() {
		requeue := PartitionConcurrencyLimitRequeueExtension
		if iter.CtrConcurrency == 0 {
			// This has been throttled only.  Don't requeue so far ahead, otherwise we'll be waiting longer
			// than the minimum throttle.
			//
			// TODO: When we create throttle queues, requeue this appropriately depending on the throttle
			//       period.
			requeue = PartitionThrottleLimitRequeueExtension
		}

		// Requeue this partition as we hit concurrency limits.
		metrics.IncrQueuePartitionConcurrencyLimitCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name()}})
		err = q.primaryQueueShard.PartitionRequeue(ctx, p, q.Clock().Now().Truncate(time.Second).Add(requeue), true)
		if errors.Is(err, ErrPartitionGarbageCollected) {
			q.removeContinue(ctx, p, false)
		}
		return err
	}

	// XXX: If we haven't been able to lease a single item, ensure we enqueue this
	// for a minimum of 5 seconds.

	// Requeue the partition, which reads the next unleased job or sets a time of
	// 30 seconds.  This is why we have to lease items above, else this may return an item that is
	// about to be leased and processed by the worker.
	_, err = Duration(ctx, q.primaryQueueShard.Name(), "partition_requeue", q.Clock().Now(), func(ctx context.Context) (any, error) {
		err = q.primaryQueueShard.PartitionRequeue(ctx, p, q.Clock().Now().Add(PartitionRequeueExtension), false)
		return nil, err
	})
	if err == ErrPartitionGarbageCollected {
		q.removeContinue(ctx, p, false)
		// Safe;  we're preventing this from wasting cycles in the future.
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}
