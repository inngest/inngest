package queue

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

func (q *queueProcessor) process(
	ctx context.Context,
	i processItem,
	f RunFunc,
) error {
	qi := i.I
	p := i.P
	continuationCtr := i.PCtr

	capacityLeaseID := newCapacityLease(i.capacityLease)

	disableConstraintUpdates := i.disableConstraintUpdates

	var err error
	leaseID := qi.LeaseID

	// Allow the main runner to block until this work is done
	q.wg.Add(1)
	defer q.wg.Done()

	// Continually the lease while this job is being processed.
	extendLeaseTick := q.Clock.NewTicker(QueueLeaseDuration / 2)
	defer extendLeaseTick.Stop()

	extendCapacityLeaseTick := q.Clock.NewTicker(q.CapacityLeaseExtendInterval)
	defer extendCapacityLeaseTick.Stop()

	errCh := make(chan error)
	doneCh := make(chan struct{})

	// Continually extend lease in the background while we're working on this job
	go func() {
		for {
			select {
			case <-doneCh:
				return
			case <-extendLeaseTick.Chan():
				if ctx.Err() != nil {
					// Don't extend lease when the ctx is done.
					return
				}

				if leaseID == nil {
					q.log.Error("cannot extend lease since lease ID is nil", "qi", qi, "partition", p)
					// Don't extend lease since one doesn't exist
					errCh <- fmt.Errorf("cannot extend lease since lease ID is nil")
					return
				}

				// Once a job has started, use a BG context to always renew.
				leaseID, err = q.PrimaryQueueShard.Processor().ExtendLease(
					context.Background(),
					qi,
					*leaseID,
					QueueLeaseDuration,
					// When holding a capacity lease, do not update constraint state
					ExtendLeaseOptionDisableConstraintUpdates(disableConstraintUpdates),
				)
				if err != nil {
					// log error if unexpected; the queue item may be removed by a Dequeue() operation
					// invoked by finalize() (Cancellations, Parallelism)
					if !errors.Is(ErrQueueItemNotFound, err) {
						q.log.Error("error extending lease", "error", err, "qi", qi, "partition", p)
					}

					// always stop processing the queue item if lease cannot be extended
					errCh <- fmt.Errorf("error extending lease while processing: %w", err)
					return
				}
			case <-extendCapacityLeaseTick.Chan():
				if ctx.Err() != nil {
					// Don't extend lease when the ctx is done.
					return
				}

				// If no initial capacity lease was provided for this queue item, no-op
				// This specifically happens when
				// - the item is enqueued to a system queue
				// - the Constraint API is disabled or the current account is not enrolled
				// - the Constraint API provided a lease which expired at the time of leasing the queue item
				if i.capacityLease == nil {
					q.log.Trace("item has no capacity lease, skipping lease extension")
					continue
				}

				currentCapacityLease := capacityLeaseID.get()
				if currentCapacityLease == nil {
					q.log.Error("cannot extend capacity lease since capacity lease ID is nil", "qi", qi, "partition", p)
					// Don't extend lease since one doesn't exist
					errCh <- fmt.Errorf("cannot extend lease since lease ID is nil")
					return
				}

				// This idempotency key will change with every refreshed lease, which makes sense.
				operationIdempotencyKey := currentCapacityLease.String()

				res, err := q.CapacityManager.ExtendLease(context.Background(), &constraintapi.CapacityExtendLeaseRequest{
					AccountID:      p.AccountID,
					IdempotencyKey: operationIdempotencyKey,
					LeaseID:        *currentCapacityLease,
					Migration: constraintapi.MigrationIdentifier{
						IsRateLimit: false,
						QueueShard:  q.PrimaryQueueShard.Name(),
					},
					Duration: QueueLeaseDuration,
					Source:   constraintapi.LeaseSource{Location: constraintapi.CallerLocationItemLease},
				})
				if err != nil {
					// log error if unexpected; the queue item may be removed by a Dequeue() operation
					// invoked by finalize() (Cancellations, Parallelism)
					if !errors.Is(ErrQueueItemNotFound, err) {
						q.log.ReportError(
							err,
							"error extending capacity lease",
							logger.WithErrorReportLog(true),
							logger.WithErrorReportTags(map[string]string{
								"partitionID": p.ID,
								"accountID":   p.AccountID.String(),
								"item":        qi.ID,
								"leaseID":     currentCapacityLease.String(),
							}),
						)
					}

					// always stop processing the queue item if lease cannot be extended
					errCh <- fmt.Errorf("error extending lease while processing: %w", err)
					return
				}

				if res.LeaseID == nil {
					// Lease could not be extended
					errCh <- fmt.Errorf("failed to extend capacity lease, no new lease ID received")
					return
				}

				capacityLeaseID.set(res.LeaseID)
			}
		}
	}()

	// XXX: Add a max job time here, configurable.
	jobCtx, jobCancel := context.WithCancel(context.WithoutCancel(ctx))
	defer jobCancel()

	// Add the job ID to the queue context.  This allows any logic that handles the run function
	// to inspect job IDs, eg. for tracing or logging, without having to thread this down as
	// arguments.
	//
	// NOTE: It is important that we keep this here for every job;  the exeuctor uses this to pass
	// along the job ID as metadata to the SDK.  We also need to pass in shard information.
	jobCtx = WithShardID(jobCtx, q.PrimaryQueueShard.Name())
	jobCtx = WithJobID(jobCtx, qi.ID)
	// Same with the group ID, if it exists.
	if qi.Data.GroupID != "" {
		jobCtx = state.WithGroupID(jobCtx, qi.Data.GroupID)
	}

	startedAt := q.Clock.Now()
	go func() {
		longRunningJobStatusTick := q.Clock.NewTicker(5 * time.Minute)
		defer longRunningJobStatusTick.Stop()

		for {
			select {
			case <-jobCtx.Done():
				return
			case <-longRunningJobStatusTick.Chan():
			}

			q.log.Debug("long running queue job tick", "item", qi, "dur", q.Clock.Now().Sub(startedAt).String())
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Always retry this job.
				stack := debug.Stack()
				q.log.Error("job panicked", "error", fmt.Errorf("%v", r), "stack", string(stack))
				errCh <- AlwaysRetryError(fmt.Errorf("job panicked: %v", r))
			}
		}()

		// This job may be up to 1999 ms in the future, as explained in processPartition.
		// Just... wait until the job is available.
		delay := time.UnixMilli(qi.AtMS).Sub(q.Clock.Now())

		if delay > 0 {
			<-q.Clock.After(delay)
			q.log.Trace("delaying job in memory",
				"at", qi.AtMS,
				"ms", delay.Milliseconds(),
			)
		}
		n := q.Clock.Now()

		// Track the sojourn (concurrency) latency.
		sojourn := qi.SojournLatency(n)
		doCtx := context.WithValue(jobCtx, sojournKey, sojourn)

		// Track the latency on average globally.  Do this in a goroutine so that it doesn't
		// at all delay the job during concurrenty locking contention.
		if qi.WallTimeMS == 0 {
			qi.WallTimeMS = qi.AtMS // backcompat while WallTimeMS isn't valid.
		}
		latency := qi.Latency(n)
		doCtx = context.WithValue(doCtx, latencyKey, latency)

		// store started at and latency in ctx
		doCtx = context.WithValue(doCtx, startedAtKey, n)

		go func() {
			// Update the ewma
			latencySem.Lock()
			latencyAvg.Add(float64(latency))
			metrics.GaugeQueueItemLatencyEWMA(ctx, int64(latencyAvg.Value()/1e6), metrics.GaugeOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"kind": qi.Data.Kind, "queue_shard": q.PrimaryQueueShard.Name()},
			})
			latencySem.Unlock()

			// Set the metrics historgram and gauge, which reports the ewma value.
			metrics.HistogramQueueItemLatency(ctx, latency.Milliseconds(), metrics.HistogramOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"kind": qi.Data.Kind, "queue_shard": q.PrimaryQueueShard.Name()},
			})
		}()

		metrics.IncrQueueItemStatusCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "started", "queue_shard": q.PrimaryQueueShard.Name()},
		})

		runInfo := RunInfo{
			Latency:             latency,
			SojournDelay:        sojourn,
			Priority:            q.PartitionPriorityFinder(ctx, p),
			QueueShardName:      q.PrimaryQueueShard.Name(),
			ContinueCount:       continuationCtr,
			RefilledFromBacklog: qi.RefilledFrom,
			CapacityLease:       i.capacityLease,
		}

		// Call the run func.
		res, err := f(doCtx, runInfo, qi.Data)
		extendLeaseTick.Stop()
		if err != nil {
			metrics.IncrQueueItemStatusCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"status": "errored", "queue_shard": q.PrimaryQueueShard.Name()},
			})
			errCh <- err
			return
		}
		metrics.IncrQueueItemStatusCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "completed", "queue_shard": q.PrimaryQueueShard.Name()},
		})

		if res.ScheduledImmediateJob {
			// Add the partition to be continued again.  Note that if we've already
			// continued beyond the limit this is a noop.
			q.addContinue(ctx, &p, continuationCtr+1)
		}

		// Closing this channel prevents the goroutine which extends lease from leaking,
		// and dequeues the job
		close(doneCh)
	}()

	// When capacity is leased, release it after requeueing/dequeueing the item.
	// This is optional and best-effort to free up concurrency capacity as quickly as possible
	// for the next worker to lease a queue item.
	if capacityLeaseID.has() {
		defer service.Go(func() {
			currentLeaseID := capacityLeaseID.get()
			if capacityLeaseID == nil {
				return
			}

			res, err := q.CapacityManager.Release(context.Background(), &constraintapi.CapacityReleaseRequest{
				AccountID:      p.AccountID,
				IdempotencyKey: qi.ID,
				LeaseID:        *currentLeaseID,
				Migration: constraintapi.MigrationIdentifier{
					IsRateLimit: false,
					QueueShard:  q.PrimaryQueueShard.Name(),
				},
				Source: constraintapi.LeaseSource{Location: constraintapi.CallerLocationItemLease},
			})
			if err != nil {
				q.log.ReportError(err, "failed to release capacity", logger.WithErrorReportTags(map[string]string{
					"account_id":  p.AccountID.String(),
					"lease_id":    currentLeaseID.String(),
					"function_id": p.FunctionID.String(),
				}))
			}

			q.log.Trace("released capacity", "res", res)
		})
	}

	select {
	case err := <-errCh:
		// Job errored or extending lease errored.  Cancel the job ASAP.
		jobCancel()

		if ShouldRetry(err, qi.Data.Attempt, qi.Data.GetMaxAttempts()) {
			at := q.backoffFunc(qi.Data.Attempt)

			// Attempt to find any RetryAtSpecifier in the error tree.
			if specifier := AsRetryAtError(err); specifier != nil {
				next := specifier.NextRetryAt()
				at = *next
			}

			if !IsAlwaysRetryable(err) {
				qi.Data.Attempt += 1
			}

			qi.AtMS = at.UnixMilli()
			if err := q.PrimaryQueueShard.Processor().Requeue(context.WithoutCancel(ctx), qi, at, RequeueOptionDisableConstraintUpdates(disableConstraintUpdates)); err != nil {
				if err == ErrQueueItemNotFound {
					// Safe. The executor may have dequeued.
					return nil
				}

				q.log.Error("error requeuing job", "error", err, "item", qi)
				return err
			}
			if _, ok := err.(QuitError); ok {
				q.quit <- err
				return err
			}
			return nil
		}

		// Dequeue this entirely, as this permanently failed.
		// XXX: Increase permanently failed counter here.
		if err := q.PrimaryQueueShard.Processor().Dequeue(context.WithoutCancel(ctx), qi, DequeueOptionDisableConstraintUpdates(disableConstraintUpdates)); err != nil {
			if err == ErrQueueItemNotFound {
				// Safe. The executor may have dequeued.
				return nil
			}
			return err
		}

		if _, ok := err.(osqueue.QuitError); ok {
			q.log.Warn("received queue quit error", "error", err)
			q.quit <- err
			return err
		}

	case <-doneCh:
		if err := q.Dequeue(context.WithoutCancel(ctx), q.primaryQueueShard, qi, DequeueOptionDisableConstraintUpdates(disableConstraintUpdates)); err != nil {
			if err == ErrQueueItemNotFound {
				// Safe. The executor may have dequeued.
				return nil
			}
			return err
		}
	}

	return nil
}
