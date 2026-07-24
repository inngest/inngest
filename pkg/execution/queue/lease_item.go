package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/attribute"
)

func (q *queueProcessor) LeaseItem(ctx context.Context, req LeaseItemRequest, dispatch DispatchFunc) (LeaseItemResult, error) {
	item := req.Item
	partition := req.Partition
	l := logger.StdlibLogger(ctx).With("partition", partition, "item", item)

	ctx, span := q.Options().ConditionalTracer.NewSpan(ctx, "queue.LeaseItem", TraceScopeFromQueueItem(*item, q.Shard().Name()))
	defer span.End()
	span.SetAttributes(attribute.String("partition_id", partition.ID))
	span.SetAttributes(attribute.String("item_kind", item.Data.Kind))
	span.SetAttributes(attribute.String("run_id", item.Data.Identifier.RunID.String()))
	span.SetAttributes(attribute.String("item_id", item.ID))
	if item.Data.JobID != nil {
		span.SetAttributes(attribute.String("job_id", *item.Data.JobID))
	}
	if dispatch == nil {
		span.RecordError(ErrProcessMissingDispatch)
		return LeaseItemResult{Status: LeaseItemStatusLeaseError, Err: ErrProcessMissingDispatch}, ErrProcessMissingDispatch
	}

	// Copy queue timestamps into the inner item for easier access during processing.
	// We convert these to time.Time here so that we don't have to convert them multiple times during processing.
	item.Data.At = time.UnixMilli(item.AtMS)
	item.Data.EnqueuedAt = time.UnixMilli(item.EnqueuedAt)

	if item.IsLeased(q.Clock().Now()) {
		span.SetAttributes(attribute.String("skip_reason", "already_leased"))
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "lease_contention", "queue_shard": q.Shard().Name(), "constraint_source": "lease"},
		})
		return LeaseItemResult{Status: LeaseItemStatusAlreadyLeased}, nil
	}

	// Check if there's capacity from our local workers atomically prior to leasing our items.
	if !q.Semaphore().TryAcquire(1) {
		span.SetAttributes(attribute.String("skip_reason", "no_queue_worker_capacity"))
		metrics.IncrQueuePartitionProcessNoCapacityCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.Shard().Name()}})
		// Break the entire loop to prevent out of order work.
		return LeaseItemResult{Status: LeaseItemStatusNoWorkerCapacity}, ErrProcessNoCapacity
	}

	metrics.WorkerQueueCapacityCounter(ctx, 1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.Shard().Name()}})

	// Release semaphore capacity, will be called when this function
	// exits unless explicitly committed (see below).
	//
	// release() can be called early to make worker capacity available again
	release := sync.OnceFunc(func() {
		q.Semaphore().Release(1)
		metrics.WorkerQueueCapacityCounter(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.Shard().Name()}})
	})

	// Default to resetting the semaphore if we don't explicitly keep it.
	// This should prevent forgetting the .Release() case.
	commitSemaphoreAcquire := false
	defer func() {
		if commitSemaphoreAcquire {
			return
		}

		release()
	}()

	// In case the item has no earliest peek time set and stamping is enabled,
	// call SetEarliestPeekTime before verifying and user constraints.
	// This is intentionally called AFTER queue worker capacity; unlike user constraints,
	// worker capacity counts towards system latency, not user latency.
	if item.EarliestPeekTime == 0 && q.Options().ItemEarliestPeekTimeConfig(ctx, q.Shard().Name(), *item).Enabled {
		peekTime := req.StaticTime
		if peekTime.IsZero() {
			peekTime = q.Clock().Now()
		}

		earliestPeekTime, err := q.Shard().SetEarliestPeekTime(ctx, *item, peekTime)
		if err != nil {
			span.RecordError(err)
			l.Warn("could not set earliest peek time", "error", err)
		} else {
			item.EarliestPeekTime = earliestPeekTime.UnixMilli()
		}
	}

	//
	// Before we can do any work on the queue item, we need to check if all the constraints have capacity.
	// This was previously done in Lease and has since been moved to the Constraint API.
	//
	// The Constraint API employs in-memory caching for a subset of constraints, ensuring low-latency
	// responses on hit constraints and avoiding overloading the API.
	//

	backlog := ItemBacklog(ctx, *item)
	shadowPartition := ItemShadowPartition(ctx, *item)
	constraints := q.Options().PartitionConstraintConfigGetter(ctx, shadowPartition.Identifier())

	// The following lease options simply specify some objects that are required during lease but were already generated.
	leaseOptions := []LeaseOptionFn{
		LeaseBacklog(backlog),
		LeaseShadowPartition(shadowPartition),
		LeaseConstraints(constraints),
	}

	// Acquire capacity lease, in case the Constraint API is enabled and the current queue item should use capacity leases.
	// We only ignore capacity leases for system queues and items missing account ID / env ID / function ID combinations.
	// When the Constraint API is enabled, it will handle concurrency and throttle checks on the queue item.
	// This is for an individual lease. If a constraint is at capacity, no leases will be returned and we will handle the missing capacity accordingly.
	constraintRes, err := q.ItemLeaseConstraintCheck(
		ctx,
		&shadowPartition,
		&backlog,
		constraints,
		item,
		q.Clock().Now(),
	)
	if err != nil {
		span.RecordError(err)
		l.ReportError(err, "could not check constraints to lease item")
		// Stop iterator but don't quit the queue.
		return LeaseItemResult{}, ErrProcessStopIterator
	}

	// If we're limited by constraints, release semaphore early since we won't be leasing or processing.
	if constraintRes.LimitingConstraint != enums.QueueConstraintNotLimited {
		release()

		span.SetAttributes(attribute.String("limiting_constraint", constraintRes.LimitingConstraint.String()))
	}

	var leaseID *ulid.ULID
	switch constraintRes.LimitingConstraint {
	// If no constraints were hit (or we didn't invoke the Constraint API).
	case enums.QueueConstraintNotLimited:

		// Attempt to lease this item before passing this to a worker.  We have to do this
		// synchronously as we need to lease prior to requeueing the partition pointer. If
		// we don't do this here, the workers may not lease the items before calling Peek
		// to re-enqeueu the pointer, which then increases contention - as we requeue a
		// pointer too early.
		//
		// This is safe:  only one process runs scan(), and we guard the total number of
		// available workers with the above semaphore.
		leaseID, err = Duration(ctx, q.Shard().Name(), "lease", q.Clock().Now(), func(ctx context.Context) (*ulid.ULID, error) {
			return q.Shard().Lease(
				ctx,
				*item,
				QueueLeaseDuration,
				req.StaticTime,
				leaseOptions...,
			)
		})
		// NOTE: If this loop ends in an error, we must _always_ release an item from the
		// semaphore to free capacity.  This will happen automatically when the worker
		// finishes processing a queue item on success.
		if err != nil {
			// Continue on and handle the error below.
			release()
		}
	// Simulate errors returned by Lease.
	case enums.QueueConstraintThrottle:
		err = ErrQueueItemThrottled
	case enums.QueueConstraintAccountConcurrency:
		err = NewKeyError(ErrAccountConcurrencyLimit, shadowPartition.AccountID.String())
	case enums.QueueConstraintFunctionConcurrency:
		err = NewKeyError(ErrPartitionConcurrencyLimit, shadowPartition.FunctionID.String())
	case enums.QueueConstraintCustomConcurrencyKey1:
		err = NewKeyError(ErrConcurrencyLimitCustomKey, backlog.CustomConcurrencyKeyID(1))
	case enums.QueueConstraintCustomConcurrencyKey2:
		err = NewKeyError(ErrConcurrencyLimitCustomKey, backlog.CustomConcurrencyKeyID(2))
	case enums.QueueConstraintSemaphore:
		err = ErrSemaphoreLimit
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
	if item.EarliestPeekTime == 0 && partition.Last > 0 && partition.Last > item.AtMS {
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
		"queue_shard", q.Shard().Name(),
	)

	// Used for error reporting.
	errTags := map[string]string{}
	if cause != nil {
		errTags["cause"] = cause.Error()
	}
	if leaseID != nil {
		errTags["lease"] = leaseID.String()
	}
	limitedRetryAfter := constraintRes.RetryAfter

	switch {
	case errors.Is(cause, ErrQueueItemThrottled):
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "throttled", "queue_shard": q.Shard().Name(), "constraint_source": "constraintapi"},
		})

		if q.Options().ItemEnableKeyQueues(ctx, *item) {
			err := q.Shard().Requeue(ctx, *item, time.UnixMilli(item.AtMS))
			if err != nil && !errors.Is(err, ErrQueueItemNotFound) {
				l.ReportError(err, "could not requeue item to backlog after hitting throttle limit",
					logger.WithErrorReportTags(errTags),
				)
				return LeaseItemResult{Status: LeaseItemStatusThrottled, RetryAfter: limitedRetryAfter}, fmt.Errorf("could not requeue to backlog: %w", err)
			}

			metrics.IncrRequeueExistingToBacklogCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": q.Shard().Name(),
					// "partition_id": item.FunctionID.String(),
					"status": "throttled",
				},
			})
		}

		return LeaseItemResult{Status: LeaseItemStatusThrottled, RetryAfter: limitedRetryAfter}, nil
	case errors.Is(cause, ErrPartitionConcurrencyLimit), errors.Is(cause, ErrAccountConcurrencyLimit), errors.Is(cause, ErrSystemConcurrencyLimit):
		// Since the queue is at capacity on a fn or account level, no
		// more jobs in this loop should be worked on - so break.
		//
		// Even if we have capacity for the next job in the loop we do NOT
		// want to claim the job, as this breaks ordering guarantees.  The
		// only safe thing to do when we hit a function or account level
		// concurrency key.
		var status string
		switch {
		case errors.Is(cause, ErrSystemConcurrencyLimit):
			status = "system_concurrency_limit"
		case errors.Is(cause, ErrPartitionConcurrencyLimit):
			status = "partition_concurrency_limit"
			if partition.FunctionID != nil {
				q.Options().lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *partition.FunctionID)
			}
		case errors.Is(cause, ErrAccountConcurrencyLimit):
			status = "account_concurrency_limit"
			// For backwards compatibility, we report on the function level as well.
			if partition.FunctionID != nil {
				q.Options().lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *partition.FunctionID)
			}

			q.Options().lifecycles.OnAccountConcurrencyLimitReached(
				context.WithoutCancel(ctx),
				partition.AccountID,
				partition.EnvID,
			)
		}

		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": status, "queue_shard": q.Shard().Name(), "constraint_source": "constraintapi"},
		})

		if q.Options().ItemEnableKeyQueues(ctx, *item) {
			err := q.Shard().Requeue(ctx, *item, time.UnixMilli(item.AtMS))
			if err != nil && !errors.Is(err, ErrQueueItemNotFound) {
				l.ReportError(err, "could not requeue item to backlog after hitting concurrency limit",
					logger.WithErrorReportTags(errTags),
				)
				return LeaseItemResult{Status: LeaseItemStatusConcurrencyLimited, RetryAfter: limitedRetryAfter}, fmt.Errorf("could not requeue to backlog: %w", err)
			}

			metrics.IncrRequeueExistingToBacklogCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": q.Shard().Name(),
					// "partition_id": item.FunctionID.String(),
					"status": status,
				},
			})
		}

		return LeaseItemResult{Status: LeaseItemStatusConcurrencyLimited, RetryAfter: limitedRetryAfter}, fmt.Errorf("concurrency hit: %w", ErrProcessNoUserConstraintCapacity)
	case errors.Is(cause, ErrConcurrencyLimitCustomKey):
		// For backwards compatibility, we report on the function level as well.
		if partition.FunctionID != nil {
			q.Options().lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *partition.FunctionID)
		}

		// TODO: Report on key that was hit (this must have been empty previously)
		// p.queue.lifecycles.OnCustomKeyConcurrencyLimitReached(context.WithoutCancel(ctx), p.partition.EvaluatedConcurrencyKey)

		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "custom_key_concurrency_limit", "queue_shard": q.Shard().Name(), "constraint_source": "constraintapi"},
		})

		if q.Options().ItemEnableKeyQueues(ctx, *item) {
			err := q.Shard().Requeue(ctx, *item, time.UnixMilli(item.AtMS))
			if err != nil && !errors.Is(err, ErrQueueItemNotFound) {
				l.ReportError(err, "could not requeue item to backlog after hitting custom concurrency limit",
					logger.WithErrorReportTags(errTags),
				)
				return LeaseItemResult{Status: LeaseItemStatusCustomConcurrencyLimited, RetryAfter: limitedRetryAfter}, fmt.Errorf("could not requeue to backlog: %w", err)
			}

			metrics.IncrRequeueExistingToBacklogCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": q.Shard().Name(),
					// "partition_id": item.FunctionID.String(),
					"status": "custom_key_concurrency_limit",
				},
			})
		}
		return LeaseItemResult{Status: LeaseItemStatusCustomConcurrencyLimited, RetryAfter: limitedRetryAfter}, nil
	case errors.Is(cause, ErrSemaphoreLimit):
		// Semaphore capacity exhausted for this specific item (e.g., start job with fn concurrency).
		// Skip this item and continue scanning; other items without semaphores (step 2, etc.)
		// can still be processed.
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "semaphore_limit", "queue_shard": q.Shard().Name(), "constraint_source": "constraintapi"},
		})
		return LeaseItemResult{Status: LeaseItemStatusSemaphoreLimited, RetryAfter: limitedRetryAfter}, nil
	case errors.Is(cause, ErrQueueItemNotFound):
		// This is an okay error.  Move to the next job item.
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "success", "queue_shard": q.Shard().Name(), "constraint_source": "constraintapi"},
		})
		return LeaseItemResult{Status: LeaseItemStatusNotFound}, nil
	case errors.Is(cause, ErrQueueItemAlreadyLeased):
		// This is an okay error.  Move to the next job item.
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "success", "queue_shard": q.Shard().Name(), "constraint_source": "constraintapi"},
		})
		return LeaseItemResult{Status: LeaseItemStatusLeaseContention}, nil
	}

	// Handle other errors.
	if err != nil || leaseID == nil {
		span.RecordError(err)
		processErr := fmt.Errorf("error leasing in process: %w", err)
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "error", "queue_shard": q.Shard().Name(), "constraint_source": "constraintapi"},
		})
		return LeaseItemResult{Status: LeaseItemStatusLeaseError, Err: processErr}, processErr
	}

	// Assign the lease ID and pass this to be handled by the available worker.
	// There should always be capacity on this queue as we track capacity via
	// a semaphore. GenerationID was loaded from the queue hash on Peek and is
	// preserved across Lease (the Lua script does not touch it).
	item.LeaseID = leaseID

	metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags:    map[string]any{"status": "success", "queue_shard": q.Shard().Name(), "constraint_source": "constraintapi"},
	})
	err = dispatch(ctx, ProcessItem{
		P:    *partition,
		I:    *item,
		PCtr: req.PartitionContinueCtr,

		CapacityLease:       constraintRes.CapacityLease,
		ConditionalTraceCtx: context.WithoutCancel(ctx),
	})
	result := LeaseItemResult{Status: LeaseItemStatusDispatched}
	if err != nil {
		span.RecordError(err)
		return result, err
	}
	commitSemaphoreAcquire = true

	return result, nil
}
