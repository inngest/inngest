package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
)

type ProcessorIterator struct {
	Partition *QueuePartition
	Items     []*QueueItem
	// PartitionContinueCtr is the number of times the partition has currently been
	// continued already in the chain.  we must record this such that a partition isn't
	// forced indefinitely.
	PartitionContinueCtr uint

	// Queue is the Queue that owns this processor.
	Queue *queueProcessor

	// Denies records a denylist as keys hit concurrency and throttling limits.
	// this lets us prevent lease attempts for consecutive keys, as soon as the first
	// key is denied.
	Denies *LeaseDenies

	// error returned when processing
	Err error

	// StaticTime is used as the processing time for all items in the queue.
	// We process queue items sequentially, and time progresses linearly as each
	// queue item is processed.  We want to use a static time to prevent out-of-order
	// processing with regards to things like rate limiting;  if we use time.Now(),
	// queue items later in the array may be processed before queue items earlier in
	// the array depending on eg. a rate limit becoming available half way through
	// iteration.
	StaticTime time.Time

	// Parallel indicates whether the partition's jobs can be processed in Parallel.
	// Parallel processing breaks best effort fifo but increases throughput.
	Parallel bool

	// These flags are used to handle partition rqeueueing.
	CtrSuccess     int32
	CtrConcurrency int32
	CtrRateLimit   int32

	// IsCustomKeyLimitOnly records whether we ONLY hit custom concurrency key limits.
	// This lets us know whether to peek from a random offset if we have FIFO disabled
	// to attempt to find other possible functions outside of the key(s) with issues.
	IsCustomKeyLimitOnly bool
}

func (p *ProcessorIterator) iterate(ctx context.Context) error {
	var err error

	// set flag to true to begin with
	p.IsCustomKeyLimitOnly = true

	eg := errgroup.Group{}
	for _, i := range p.Items {
		if i == nil {
			// THIS SHOULD NEVER HAPPEN. Skip gracefully and log error
			logger.StdlibLogger(ctx).Error("nil queue item in partition", "partition", p.Partition)
			continue
		}

		if p.Parallel {
			item := *i
			eg.Go(func() error {
				err := p.process(ctx, &item)
				if err != nil {
					// NOTE: ignore if the queue item is not found
					if errors.Is(err, ErrQueueItemNotFound) {
						return nil
					}
				}
				return err
			})
			continue
		}

		// non-parallel (sequential fifo) processing.
		if err = p.process(ctx, i); err != nil {
			// NOTE: ignore if the queue item is not found
			if errors.Is(err, ErrQueueItemNotFound) {
				continue
			}
			// always break on the first error;  if processing returns an error we
			// always assume that we stop iterating.
			//
			// we return errors when:
			// * there's no capacity (so dont continue, because FIFO)
			// * we hit fn concurrency limits (so don't continue, because FIFO too)
			// * some other error, which means something went wrong.
			break
		}
	}

	if p.Parallel {
		// normalize errors from parallel
		err = eg.Wait()
	}

	if errors.Is(err, ErrProcessStopIterator) {
		// This is safe;  it's stopping safely but isn't an error.
		return nil
	}
	if errors.Is(err, ErrProcessNoCapacity) {
		// This is safe;  it's stopping safely but isn't an error.
		return nil
	}

	// someting went wrong.  report the error.
	return err
}

func (p *ProcessorIterator) process(ctx context.Context, item *QueueItem) error {
	l := p.Queue.log.With("partition", p.Partition, "item", item)

	// TODO: Create an in-memory mapping of rate limit keys that have been hit,
	//       and don't bother to process if the queue item has a limited key.  This
	//       lessens work done in the queue, as we can `continue` immediately.
	if item.IsLeased(p.Queue.Clock.Now()) {
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "lease_contention", "queue_shard": p.Queue.primaryQueueShard.Name(), "constraint_source": "lease"},
		})
		return nil
	}

	// Check if there's capacity from our local workers atomically prior to leasing our items.
	if !p.Queue.sem.TryAcquire(1) {
		metrics.IncrQueuePartitionProcessNoCapacityCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": p.Queue.primaryQueueShard.Name()}})
		// Break the entire loop to prevent out of order work.
		return ErrProcessNoCapacity
	}

	metrics.WorkerQueueCapacityCounter(ctx, 1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": p.Queue.primaryQueueShard.Name()}})

	backlog := ItemBacklog(ctx, *item)
	partition := ItemShadowPartition(ctx, *item)
	constraints := p.Queue.PartitionConstraintConfigGetter(ctx, partition.Identifier())

	leaseOptions := []LeaseOptionFn{
		LeaseBacklog(backlog),
		LeaseShadowPartition(partition),
		LeaseConstraints(constraints),
	}

	constraintRes, err := p.Queue.primaryQueueShard.ItemLeaseConstraintCheck(
		ctx,
		&partition,
		&backlog,
		constraints,
		item,
		p.StaticTime,
	)
	if err != nil {
		p.Queue.sem.Release(1)
		metrics.WorkerQueueCapacityCounter(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": p.Queue.primaryQueueShard.Name()}})

		return fmt.Errorf("could not check constraints to lease item: %w", err)
	}

	constraint_check_source := "lease"
	if constraintRes.SkipConstraintChecks {
		constraint_check_source = "constraint-api"
		leaseOptions = append(leaseOptions, LeaseOptionDisableConstraintChecks(true))
	}

	var leaseID *ulid.ULID
	switch constraintRes.LimitingConstraint {
	// If no constraints were hit (or we didn't invoke the Constraint API)
	case enums.QueueConstraintNotLimited:

		// Attempt to lease this item before passing this to a worker.  We have to do this
		// synchronously as we need to lease prior to requeueing the partition pointer. If
		// we don't do this here, the workers may not lease the items before calling Peek
		// to re-enqeueu the pointer, which then increases contention - as we requeue a
		// pointer too early.
		//
		// This is safe:  only one process runs scan(), and we guard the total number of
		// available workers with the above semaphore.
		leaseID, err = Duration(ctx, p.Queue.primaryQueueShard.Name(), "lease", p.Queue.Clock.Now(), func(ctx context.Context) (*ulid.ULID, error) {
			return p.Queue.primaryQueueShard.Lease(
				ctx,
				*item,
				QueueLeaseDuration,
				p.StaticTime,
				p.Denies,
				leaseOptions...,
			)
		})
		// NOTE: If this loop ends in an error, we must _always_ release an item from the
		// semaphore to free capacity.  This will happen automatically when the worker
		// finishes processing a queue item on success.
		if err != nil {
			// Continue on and handle the error below.
			p.Queue.sem.Release(1)
			metrics.WorkerQueueCapacityCounter(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": p.Queue.primaryQueueShard.Name()}})
		}
	// Simulate errors returned by Lease
	case enums.QueueConstraintThrottle:
		err = ErrQueueItemThrottled
	case enums.QueueConstraintAccountConcurrency:
		err = NewKeyError(ErrAccountConcurrencyLimit, partition.AccountID.String())
	case enums.QueueConstraintFunctionConcurrency:
		err = NewKeyError(ErrPartitionConcurrencyLimit, partition.FunctionID.String())
	case enums.QueueConstraintCustomConcurrencyKey1:
		err = NewKeyError(ErrConcurrencyLimitCustomKey, backlog.CustomConcurrencyKeyID(1))
	case enums.QueueConstraintCustomConcurrencyKey2:
		err = NewKeyError(ErrConcurrencyLimitCustomKey, backlog.CustomConcurrencyKeyID(2))
	default:
		l.ReportError(errors.New("unhandled queue constraint type"), fmt.Sprintf("constraint type: %s", constraintRes.LimitingConstraint))
		// Limited but the constraint is unknown?
	}

	// Check the sojourn delay for this item in the queue. Tracking system latency vs
	// sojourn latency from concurrency is important.
	//
	// Firstly, we check:  does the job store the first peek time?  If so, the
	// delta between now and that time is the sojourn latency.  If not, this is either
	// one of two cases:
	//   - This is a new job in the queue, and we're peeking it for the first time.
	//     Sojourn latency is 0.  Easy.
	//   - We've peeked the queue since adding the job.  At this point, the only
	//     conclusion is that the job wasn't peeked because of concurrency/capacity
	//     issues, so the delta between now - job added is sojourn latency.
	//
	// NOTE: You might see that we use tracking semaphores and the worker itself has
	// a maximum capacity.  We must ALWAYS peek the available capacity in our worker
	// via the above Peek() call so that worker capacity doesn't prevent us from accessing
	// all jobs in a peek.  This would break sojourn latency:  it only works if we know
	// we're quitting early because of concurrency issues in a user's function, NOT because
	// of capacity issues in our system.
	//
	// Anyway, here we set the first peek item to the item's start time if there was a
	// peek since the job was added.
	if p.Partition.Last > 0 && p.Partition.Last > item.AtMS {
		// Fudge the earliest peek time because we know this wasn't peeked and so
		// the peek time wasn't set;  but, as we were still processing jobs after
		// the job was added this item was concurrency-limited.
		item.EarliestPeekTime = item.AtMS
	}

	// We may return a keyError, which masks the actual error underneath.  If so,
	// grab the cause.
	cause := err
	var key KeyError
	if errors.As(err, &key) {
		cause = key.cause
	}

	l = l.With(
		"cause", cause,
		"item_id", item.ID,
		"account_id", item.Data.Identifier.AccountID.String(),
		"env_id", item.WorkspaceID.String(),
		"app_id", item.Data.Identifier.AppID.String(),
		"fn_id", item.FunctionID.String(),
		"queue_shard", p.Queue.primaryQueueShard.Name(),
	)

	// used for error reporting
	errTags := map[string]string{}
	if cause != nil {
		errTags["cause"] = cause.Error()
	}
	if leaseID != nil {
		errTags["lease"] = leaseID.String()
	}

	switch cause {
	case ErrQueueItemThrottled:
		p.IsCustomKeyLimitOnly = false
		// Here we denylist each throttled key that's been limited here, then ignore
		// any other jobs from being leased as we continue to iterate through the loop.
		// This maintains FIFO ordering amongst all custom concurrency keys.
		p.Denies.AddThrottled(err)

		p.CtrRateLimit++
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "throttled", "queue_shard": p.Queue.primaryQueueShard.Name(), "constraint_source": constraint_check_source},
		})

		if p.Queue.ItemEnableKeyQueues(ctx, *item) {
			err := p.Queue.primaryQueueShard.Requeue(ctx, *item, time.UnixMilli(item.AtMS))
			if err != nil && !errors.Is(err, ErrQueueItemNotFound) {
				l.ReportError(err, "could not requeue item to backlog after hitting throttle limit",
					logger.WithErrorReportTags(errTags),
				)
				return fmt.Errorf("could not requeue to backlog: %w", err)
			}

			metrics.IncrRequeueExistingToBacklogCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": p.Queue.primaryQueueShard.Name(),
					// "partition_id": item.FunctionID.String(),
					"status": "throttled",
				},
			})
		}

		return nil
	case ErrPartitionConcurrencyLimit, ErrAccountConcurrencyLimit, ErrSystemConcurrencyLimit:
		p.IsCustomKeyLimitOnly = false

		p.CtrConcurrency++
		// Since the queue is at capacity on a fn or account level, no
		// more jobs in this loop should be worked on - so break.
		//
		// Even if we have capacity for the next job in the loop we do NOT
		// want to claim the job, as this breaks ordering guarantees.  The
		// only safe thing to do when we hit a function or account level
		// concurrency key.
		var status string
		switch cause {
		case ErrSystemConcurrencyLimit:
			status = "system_concurrency_limit"
		case ErrPartitionConcurrencyLimit:
			status = "partition_concurrency_limit"
			if p.Partition.FunctionID != nil {
				p.Queue.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *p.Partition.FunctionID)
			}
		case ErrAccountConcurrencyLimit:
			status = "account_concurrency_limit"
			// For backwards compatibility, we report on the function level as well
			if p.Partition.FunctionID != nil {
				p.Queue.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *p.Partition.FunctionID)
			}

			p.Queue.lifecycles.OnAccountConcurrencyLimitReached(
				context.WithoutCancel(ctx),
				p.Partition.AccountID,
				p.Partition.EnvID,
			)
		}

		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": status, "queue_shard": p.Queue.primaryQueueShard.Name(), "constraint_source": constraint_check_source},
		})

		if p.Queue.ItemEnableKeyQueues(ctx, *item) {
			err := p.Queue.primaryQueueShard.Requeue(ctx, *item, time.UnixMilli(item.AtMS))
			if err != nil && !errors.Is(err, ErrQueueItemNotFound) {
				l.ReportError(err, "could not requeue item to backlog after hitting concurrency limit",
					logger.WithErrorReportTags(errTags),
				)
				return fmt.Errorf("could not requeue to backlog: %w", err)
			}

			metrics.IncrRequeueExistingToBacklogCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": p.Queue.primaryQueueShard.Name(),
					// "partition_id": item.FunctionID.String(),
					"status": status,
				},
			})
		}

		return fmt.Errorf("concurrency hit: %w", ErrProcessStopIterator)
	case ErrConcurrencyLimitCustomKey:
		p.CtrConcurrency++

		// Custom concurrency keys are different.  Each job may have a different key,
		// so we cannot break the loop in case the next job has a different key and
		// has capacity.
		//
		// Here we denylist each concurrency key that's been limited here, then ignore
		// any other jobs from being leased as we continue to iterate through the loop.
		p.Denies.AddConcurrency(err)

		// For backwards compatibility, we report on the function level as well
		if p.Partition.FunctionID != nil {
			p.Queue.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *p.Partition.FunctionID)
		}

		// TODO: Report on key that was hit (this must have been empty previously)
		// p.queue.lifecycles.OnCustomKeyConcurrencyLimitReached(context.WithoutCancel(ctx), p.partition.EvaluatedConcurrencyKey)

		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "custom_key_concurrency_limit", "queue_shard": p.Queue.primaryQueueShard.Name(), "constraint_source": constraint_check_source},
		})

		if p.Queue.ItemEnableKeyQueues(ctx, *item) {
			err := p.Queue.primaryQueueShard.Requeue(ctx, *item, time.UnixMilli(item.AtMS))
			if err != nil && !errors.Is(err, ErrQueueItemNotFound) {
				l.ReportError(err, "could not requeue item to backlog after hitting custom concurrency limit",
					logger.WithErrorReportTags(errTags),
				)
				return fmt.Errorf("could not requeue to backlog: %w", err)
			}

			metrics.IncrRequeueExistingToBacklogCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": p.Queue.primaryQueueShard.Name(),
					// "partition_id": item.FunctionID.String(),
					"status": "custom_key_concurrency_limit",
				},
			})
		}
		return nil
	case ErrQueueItemNotFound:
		// This is an okay error.  Move to the next job item.
		p.CtrSuccess++ // count as a success for stats purposes.
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "success", "queue_shard": p.Queue.primaryQueueShard.Name(), "constraint_source": constraint_check_source},
		})
		return nil
	case ErrQueueItemAlreadyLeased:
		// This is an okay error.  Move to the next job item.
		p.CtrSuccess++ // count as a success for stats purposes.
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "success", "queue_shard": p.Queue.primaryQueueShard.Name(), "constraint_source": constraint_check_source},
		})
		return nil
	}

	// Handle other errors.
	if err != nil || leaseID == nil {
		p.Err = fmt.Errorf("error leasing in process: %w", err)
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "error", "queue_shard": p.Queue.primaryQueueShard.Name(), "constraint_source": constraint_check_source},
		})
		return p.Err
	}

	// Assign the lease ID and pass this to be handled by the available worker.
	// There should always be capacity on this queue as we track capacity via
	// a semaphore.
	item.LeaseID = leaseID

	// increase success counter.
	p.CtrSuccess++
	metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags:    map[string]any{"status": "success", "queue_shard": p.Queue.primaryQueueShard.Name(), "constraint_source": constraint_check_source},
	})
	p.Queue.workers <- ProcessItem{
		P:    *p.Partition,
		I:    *item,
		PCtr: p.PartitionContinueCtr,

		CapacityLease: constraintRes.CapacityLease,
		// Disable constraint updates in case we skipped constraint checks.
		// This should always be linked, as we want consistent behavior while
		// processing a queue item.
		DisableConstraintUpdates: constraintRes.SkipConstraintChecks,
	}

	return nil
}

func (p *ProcessorIterator) isRequeuable() bool {
	// if we have concurrency OR we hit rate limiting/throttling.
	return p.CtrConcurrency > 0 || (p.CtrRateLimit > 0 && p.CtrConcurrency == 0 && p.CtrSuccess == 0)
}
