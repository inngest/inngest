package redis_state

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/backoff"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

const (
	ErrMaxConsecutiveProcessErrors = 20

	numWorkers     = 5_000
	minWorkersFree = 5
	pollTick       = 20 * time.Millisecond
)

func (q *queue) Run(ctx context.Context, f osqueue.RunFunc) error {
	q.workers = make(chan *QueueItem, numWorkers)
	for i := 0; i < numWorkers; i++ {
		go q.worker(ctx, f)
	}

	go q.claimSequentialLease(ctx)

	tick := time.Tick(pollTick)

LOOP:
	for {
		select {
		case <-ctx.Done():
			// Kill signal
			break LOOP
		case <-q.quit:
			break LOOP
		case <-tick:
			if q.capacity() < minWorkersFree {
				// Wait until we have more workers free.  This stops us from
				// claiming a partition to work on a single job, ensuring we
				// have capacity to run at least MinWorkersFree concurrent
				// QueueItems.  This reduces latency of enqueued items when
				// there are lots of enqueued and available jobs.
				continue
			}

			if err := q.scan(ctx, f); err != nil {
				// On scan errors, halt the worker entirely.
				if errors.Unwrap(err) != context.Canceled {
					logger.From(ctx).Error().Err(err).Msg("error scanning partition pointers")
				}
				break LOOP
			}
		}
	}

	// Wait for all in-progress items to complete.
	q.wg.Wait()

	return nil
}

// claimSequentialLease is a process which continually runs while listening to the queue,
// attempting to claim a lease on sequential processing.  Only one worker is allowed to
// work on partitions sequentially;  this reduces contention.
func (q *queue) claimSequentialLease(ctx context.Context) error {
	// Attempt to claim the lease immediately.
	leaseID, err := q.LeaseSequential(ctx, SequentialLeaseDuration, q.seqLeaseID)
	if err != ErrSequentialAlreadyLeased && err != nil {
		return err
	}
	q.seqLeaseID = leaseID

	tick := time.NewTicker(SequentialLeaseDuration / 2)
	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return nil
		case <-tick.C:
			leaseID, err := q.LeaseSequential(ctx, SequentialLeaseDuration, q.seqLeaseID)
			if err == ErrSequentialAlreadyLeased {
				// This is expected; every time there is > 1 runner listening to the
				// queue there will be contention.
				continue
			}
			if err != nil {
				logger.From(ctx).Error().Err(err).Msg("error claiming sequential lease")
				continue
			}
			q.seqLeaseID = leaseID
		}
	}
}

// worker runs a blocking process that listens to items being pushed into the
// worker channel.  This allows us to process an individual item from a queue.
func (q queue) worker(ctx context.Context, f osqueue.RunFunc) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case qi := <-q.workers:
			err := q.process(ctx, qi, f)
			if err == nil {
				continue
			}

			logger.From(ctx).Error().Err(err).Msg("error processing queue item")
			// We handle the error individually within process, requeueing
			// the item into the queue.  Here, the worker can continue as
			// usual to process the next item.
		}
	}
}

func (q queue) process(ctx context.Context, qi *QueueItem, f osqueue.RunFunc) error {
	l := logger.From(ctx)

	leaseID, err := q.Lease(ctx, qi.WorkflowID, qi.ID, QueueLeaseDuration)
	if err == ErrQueueItemNotFound {
		// Already handled.
		return nil
	}
	if err == ErrQueueItemAlreadyLeased {
		// XXX: Increase counter for lease contention
		l.Warn().Interface("item", qi).Msg("worker attempting to claim existing lease")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error leasing in process: %w", err)
	}

	// Allow the main runner to block until this work is done
	q.wg.Add(1)
	defer q.wg.Done()

	// Continually the lease while this job is being processed.
	extendLeaseTick := time.NewTicker(QueueLeaseDuration / 2)
	defer extendLeaseTick.Stop()

	// XXX: Increase counter for queue items processed
	// XXX: Increase / defer decrease gauge for items processing
	// XXX: Track latency as metric from qi.ItemID (enqueue at time)

	errCh := make(chan error)
	doneCh := make(chan struct{})

	// Continually extend lease in the background while we're working on this job
	go func() {
		for range extendLeaseTick.C {
			leaseID, err = q.ExtendLease(ctx, *qi, *leaseID, QueueLeaseDuration)
			if err != nil && err != ErrQueueItemNotFound {
				// XXX: Increase counter here.
				logger.From(ctx).Error().Err(err).Msg("error extending lease")
				errCh <- fmt.Errorf("error extending lease while processing: %w", err)
			}
		}
	}()

	// XXX: Add a max job time here, configurable.
	jobCtx, jobCancel := context.WithCancel(ctx)
	defer jobCancel()

	go func() {
		logger.From(ctx).Debug().Interface("item", qi).Msg("queue item starting")

		err := f(jobCtx, qi.Data)
		extendLeaseTick.Stop()
		if err != nil {
			// XXX: Increase counter for queue item error
			errCh <- err
			return
		}
		doneCh <- struct{}{}
	}()

	select {
	case <-errCh:
		// Job errored or extending lease errored.  Cancel the job ASAP.
		jobCancel()

		if qi.Attempt == qi.MaxAttempts {
			// XXX: Increase failed counter here.
			logger.From(ctx).Debug().Interface("item", qi).Msg("dequeueing failed job")

			// Dequeue entirely.
			if err := q.Dequeue(ctx, *qi); err != nil {
				return err
			}
			return nil
		}

		// TODO: Remove requeueing from the execution service;  just return a failed job here.
		qi.Attempt += 1
		qi.Data.ErrorCount += 1
		if err := q.Requeue(ctx, *qi, backoff.LinearJitterBackoff(qi.Attempt)); err != nil {
			logger.From(ctx).Error().Err(err).Interface("item", qi).Msg("error requeuing job")
		}
	case <-doneCh:
		logger.From(ctx).Debug().Interface("item", qi).Msg("queue item complete")
		if err := q.Dequeue(ctx, *qi); err != nil {
			return err
		}
	}

	return nil
}

func (q queue) scan(ctx context.Context, f osqueue.RunFunc) error {
	partitions, err := q.PartitionPeek(ctx, q.isSequential(), time.Now(), PartitionPeekMax)
	if err != nil {
		return err
	}

	for _, p := range partitions {
		if q.capacity() == 0 {
			// no available workers for partition
			return nil
		}

		// Attempt to lease items
		_, err := q.PartitionLease(ctx, p.WorkflowID, PartitionLeaseDuration)
		if err == ErrPartitionAlreadyLeased {
			// TODO: Increase metric for partition contention
			continue
		}
		if err != nil {
			return err
		}

		// Ensure that peek doesn't take longer than the partition lease, to
		// reduce contention.
		peekCtx, cancel := context.WithTimeout(ctx, PartitionLeaseDuration)
		defer cancel()

		queue, err := q.Peek(peekCtx, p.WorkflowID, time.Now(), q.peekSize())
		if err != nil {
			return err
		}

		if len(queue) == 0 {
			// XXX: Here we can dequeue, which must check if there are any items _at all_ in this
			// workflow queue.
		}

		for _, item := range queue {
			if item.LeaseID != nil && ulid.Time(item.LeaseID.Time()).After(time.Now()) {
				// TODO: Increase metric for queue contention
				continue
			}
			q.workers <- item
		}

		// Read the next queue item available.
		queue, err = q.Peek(peekCtx, p.WorkflowID, time.Now().Add(24*time.Hour), -1)
		if err != nil {
			return err
		}
		next := time.Now().Add(10 * time.Second)
		if len(queue) > 0 {
			next = time.UnixMilli(queue[0].At)
		}
		if err := q.PartitionRequeue(ctx, p.WorkflowID, next); err != nil {
			return err
		}
	}

	return nil
}

func (q queue) capacity() int64 {
	return int64(cap(q.workers) - len(q.workers))
}

// peekSize returns the total number of available workers which can consume individual
// queue items.
func (q queue) peekSize() int64 {
	f := q.capacity() + 1
	if f > QueuePeekMax {
		return QueuePeekMax
	}
	return f
}

func (q queue) isSequential() bool {
	if q.seqLeaseID == nil {
		return false
	}
	return ulid.Time(q.seqLeaseID.Time()).After(time.Now())
}
