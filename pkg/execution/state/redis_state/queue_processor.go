package redis_state

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/emperorearth/vitess/go/ewma"
	"github.com/inngest/inngest/pkg/backoff"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/semaphore"
)

const (
	ErrMaxConsecutiveProcessErrors = 20
	minWorkersFree                 = 5
)

var (
	latencyAvg *ewma.EWMA
)

func init() {
	latencyAvg = ewma.NewEWMA(ewma.DefaultWeightingFactor)
}

func (q *queue) Run(ctx context.Context, f osqueue.RunFunc) error {
	for i := int32(0); i < q.numWorkers; i++ {
		go q.worker(ctx, f)
	}

	// Attempt to be the worker which processes items sequentially.
	go q.claimSequentialLease(ctx)

	tick := time.NewTicker(q.pollTick)

	logger.From(ctx).Debug().Msg("starting queue worker")

LOOP:
	for {
		select {
		case <-ctx.Done():
			// Kill signal
			tick.Stop()
			break LOOP
		case err := <-q.quit:
			// An inner function received an error which was deemed irrecoverable, so
			// we're quitting the queue.
			logger.From(ctx).Error().Err(err).Msg("quitting runner internally")
			tick.Stop()
			break LOOP
		case <-tick.C:
			q.seqLeaseLock.RLock()
			if q.capacity() < minWorkersFree {
				q.seqLeaseLock.RUnlock()
				// Wait until we have more workers free.  This stops us from
				// claiming a partition to work on a single job, ensuring we
				// have capacity to run at least MinWorkersFree concurrent
				// QueueItems.  This reduces latency of enqueued items when
				// there are lots of enqueued and available jobs.
				continue
			}
			q.seqLeaseLock.RUnlock()

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
func (q *queue) claimSequentialLease(ctx context.Context) {
	// Attempt to claim the lease immediately.
	leaseID, err := q.LeaseSequential(ctx, SequentialLeaseDuration, q.sequentialLease())
	if err != ErrSequentialAlreadyLeased && err != nil {
		q.quit <- err
		return
	}

	q.seqLeaseLock.Lock()
	q.seqLeaseID = leaseID
	q.seqLeaseLock.Unlock()

	tick := time.NewTicker(SequentialLeaseDuration / 3)
	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return
		case <-tick.C:
			leaseID, err := q.LeaseSequential(ctx, SequentialLeaseDuration, q.sequentialLease())
			if err == ErrSequentialAlreadyLeased {
				// This is expected; every time there is > 1 runner listening to the
				// queue there will be contention.
				continue
			}
			if err != nil {
				logger.From(ctx).Error().Err(err).Msg("error claiming sequential lease")
				continue
			}

			q.seqLeaseLock.Lock()
			q.seqLeaseID = leaseID
			q.seqLeaseLock.Unlock()
		}
	}
}

// worker runs a blocking process that listens to items being pushed into the
// worker channel.  This allows us to process an individual item from a queue.
func (q *queue) worker(ctx context.Context, f osqueue.RunFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-q.quit:
			return
		case qi := <-q.workers:
			err := q.process(ctx, qi, f)
			q.sem.Release(1)
			if err == nil {
				continue
			}

			// We handle the error individually within process, requeueing
			// the item into the queue.  Here, the worker can continue as
			// usual to process the next item.
			logger.From(ctx).Error().Err(err).Msg("error processing queue item")
		}
	}
}

func (q *queue) scan(ctx context.Context, f osqueue.RunFunc) error {
	partitions, err := q.PartitionPeek(ctx, q.isSequential(), time.Now(), PartitionPeekMax)
	if err != nil {
		return err
	}

	for _, p := range partitions {
		if q.capacity() == 0 {
			// no available workers for partition
			return nil
		}
		if err := q.processPartition(ctx, p, f); err != nil {
			logger.From(ctx).Error().Err(err).Msg("error processing partition")
			return err
		}
	}

	return nil
}

func (q *queue) processPartition(ctx context.Context, p *QueuePartition, f osqueue.RunFunc) error {
	// Attempt to lease items
	_, err := q.PartitionLease(ctx, p.WorkflowID, PartitionLeaseDuration)
	if err == ErrPartitionAlreadyLeased {
		// TODO: Increase metric for partition contention
		return nil
	}
	if err != nil {
		return err
	}

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
	fetch := time.Now().Truncate(time.Second).Add(2 * time.Second)
	queue, err := q.Peek(peekCtx, p.WorkflowID, fetch, q.peekSize())
	if err != nil {
		return err
	}

	for _, item := range queue {
		if item.LeaseID != nil && ulid.Time(item.LeaseID.Time()).After(time.Now()) {
			// TODO: Increase metric for queue contention
			continue
		}

		// Cbeck if there's capacity in our queue atomically prior to leasing our tiems.
		if !q.sem.TryAcquire(1) {
			break
		}

		// Attempt to lease this item before passing this to a worker.  We have to do this
		// synchronously as we need to lease prior to requeueing the partition pointer. If
		// we don't do this here, the workers may not lease the items before calling Peek
		// to re-enqeueu the pointer, which then increases contention - as we requeue a
		// pointer too early.
		//
		// This is safe:  only one process runs scan(), and we guard the total number of
		// available workers with the above semaphore.
		leaseID, err := q.Lease(ctx, item.WorkflowID, item.ID, QueueLeaseDuration)
		if err == ErrQueueItemNotFound {
			// Already handled.
			q.sem.Release(1)
			continue
		}
		if err == ErrQueueItemAlreadyLeased {
			// XXX: Increase counter for lease contention
			q.sem.Release(1)
			logger.From(ctx).Warn().Interface("item", item).Msg("worker attempting to claim existing lease")
			continue
		}
		if err != nil {
			q.sem.Release(1)
			return fmt.Errorf("error leasing in process: %w", err)
		}

		// Assign the lease ID and pass this to be handled by the available worker.
		item.LeaseID = leaseID
		q.workers <- *item
	}

	// Requeue the partition, which reads the next unleased job or sets a time of
	// 30 seconds.  This is why we have to lease above, else this may return an item that is
	// about to be leased and processed by the worker.
	err = q.PartitionRequeue(ctx, p.WorkflowID, time.Now().Add(PartitionRequeueExtension))
	if err == ErrPartitionGarbageCollected {
		// Safe;  we're preventing this from wasting cycles in the future.
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

func (q *queue) process(ctx context.Context, qi QueueItem, f osqueue.RunFunc) error {
	var err error
	leaseID := qi.LeaseID

	// TODO Verify we have lease.

	// Allow the main runner to block until this work is done
	q.wg.Add(1)
	defer q.wg.Done()

	// Continually the lease while this job is being processed.
	extendLeaseTick := time.NewTicker(QueueLeaseDuration / 2)
	defer extendLeaseTick.Stop()

	// XXX: Increase counter for queue items processed
	// XXX: Increase / defer decrease gauge for items processing

	errCh := make(chan error)
	doneCh := make(chan struct{})

	// Continually extend lease in the background while we're working on this job
	go func() {
		for range extendLeaseTick.C {
			if ctx.Err() != nil {
				// Don't extend lease when the ctx is done.
				return
			}
			leaseID, err = q.ExtendLease(ctx, qi, *leaseID, QueueLeaseDuration)
			if err != nil && err != ErrQueueItemNotFound {
				// XXX: Increase counter here.
				logger.From(ctx).Error().Err(err).Msg("error extending lease")
				errCh <- fmt.Errorf("error extending lease while processing: %w", err)
				return
			}
		}
	}()

	// XXX: Add a max job time here, configurable.
	jobCtx, jobCancel := context.WithCancel(ctx)
	defer jobCancel()

	go func() {
		// This job may be up to 1999 ms in the future, as explained in processPartition.
		// Just... wait until the job is available.
		delay := time.Until(time.UnixMilli(qi.AtMS))

		if delay > 0 {
			<-time.After(delay)

			logger.From(ctx).Trace().
				Int64("at", qi.AtMS).
				Int64("ms", delay.Milliseconds()).
				Msg("delaying job in memory")
		}

		// Track the latency on average globally.
		latency := time.Now().Sub(time.UnixMilli(qi.AtMS))
		latencyAvg.AddValue(float64(latency))
		// XXX: Add indinvidual latency to metrics

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
			if err := q.Dequeue(ctx, qi); err != nil {
				return err
			}
			return nil
		}

		// TODO: Remove requeueing from the execution service;  just return a failed job here.
		qi.Attempt += 1
		qi.Data.ErrorCount += 1
		if err := q.Requeue(ctx, qi, backoff.LinearJitterBackoff(qi.Attempt)); err != nil {
			logger.From(ctx).Error().Err(err).Interface("item", qi).Msg("error requeuing job")
		}
	case <-doneCh:
		//logger.From(ctx).Debug().Interface("item", qi).Msg("queue item complete")
		if err := q.Dequeue(ctx, qi); err != nil {
			return err
		}
	}

	return nil
}

// sequentialLease is a helper method for concurrently reading the sequential
// lease ID.
func (q *queue) sequentialLease() *ulid.ULID {
	q.seqLeaseLock.RLock()
	defer q.seqLeaseLock.RUnlock()
	if q.seqLeaseID == nil {
		return nil
	}
	copied := *q.seqLeaseID
	return &copied
}

func (q *queue) capacity() int64 {
	return int64(q.numWorkers) - atomic.LoadInt64(&q.sem.counter)
}

// peekSize returns the total number of available workers which can consume individual
// queue items.
func (q *queue) peekSize() int64 {
	f := q.capacity()
	if f > QueuePeekMax {
		return QueuePeekMax
	}
	return f
}

func (q *queue) isSequential() bool {
	if q.sequentialLease() == nil {
		return false
	}
	return ulid.Time(q.sequentialLease().Time()).After(time.Now())
}

// trackingSemaphore returns a semaphore that tracks closely - but not atomically -
// the total number of items in the semaphore.  This is best effort, and is loosely
// accurate to reduce further contention.
//
// This is only used as an indicator as to whether to scan.
type trackingSemaphore struct {
	*semaphore.Weighted
	counter int64
}

func (t *trackingSemaphore) TryAcquire(n int64) bool {
	if !t.Weighted.TryAcquire(n) {
		return false
	}
	atomic.AddInt64(&t.counter, n)
	return true
}

func (t *trackingSemaphore) Acquire(ctx context.Context, n int64) error {
	if err := t.Weighted.Acquire(ctx, n); err != nil {
		return err
	}
	atomic.AddInt64(&t.counter, n)
	return nil
}

func (t *trackingSemaphore) Release(n int64) {
	t.Weighted.Release(n)
	atomic.AddInt64(&t.counter, -n)
}
