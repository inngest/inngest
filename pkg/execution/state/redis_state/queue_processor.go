package redis_state

import (
	"context"
	"fmt"
	"time"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

const (
	numWorkers     = 5_000
	minWorkersFree = 5
	pollTick       = 25 * time.Millisecond
)

func (q queue) Run(ctx context.Context, f osqueue.RunFunc) error {
	q.workers = make(chan *QueueItem, numWorkers)
	for i := 0; i < numWorkers; i++ {
		go q.worker(ctx, f)
	}

	// TODO: Claim lease on sequential

	tick := time.Tick(pollTick)
	for {
		select {
		case <-ctx.Done():
			// TODO: Clean up in-process items
			panic("not implemented")
		case <-tick:
			if q.capacity() < minWorkersFree {
				// Wait until we have more workers free.  This stops us from
				// claiming a partition to work on a single job, ensuring we
				// have capacity to run at least MinWorkersFree concurrent
				// QueueItems.
				continue
			}

			err := q.scan(ctx, f)
			if err == nil {
				continue
			}

			panic("queue error not implemented")
		}
	}
}

func (q queue) worker(ctx context.Context, f osqueue.RunFunc) error {
	for qi := range q.workers {
		if err := q.process(ctx, qi, f); err != nil {
			logger.From(ctx).Error().Err(err).Msg("error processing queue item")
		}
	}
	return nil
}

func (q queue) process(ctx context.Context, qi *QueueItem, f osqueue.RunFunc) error {
	l := logger.From(ctx)

	leaseID, err := q.Lease(ctx, qi.WorkflowID, qi.ID, QueueLeaseDuration)
	if err == ErrQueueItemAlreadyLeased {
		// TODO: Increase leased counter metric
		l.Warn().Msg("worker attempting to claim existing lease")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error leasing in process: %w", err)
	}

	extendLeaseTick := time.NewTicker(QueueLeaseDuration / 2)
	defer extendLeaseTick.Stop()

	// Continually extend lease in the background while we're working on this job
	go func() {
		for range extendLeaseTick.C {
			leaseID, err = q.ExtendLease(ctx, *qi, *leaseID, QueueLeaseDuration)
			if err != nil {
				// TODO: Get this func to quit and return this.
				_ = err
			}
		}
	}()

	// TODO: Add a max job time here

	if err := f(ctx, qi.Data); err != nil {
		// TODO: REQUEUE with backoff.  Does the runner handle requeueing?
		//       Do we really need to handle this?
	}

	if err := q.Dequeue(ctx, *qi); err != nil {
		return err
	}

	return nil
}

func (q queue) capacity() int64 {
	return int64(cap(q.workers) - len(q.workers))
}

func (q queue) peekSize() int64 {
	f := q.capacity()
	if f > QueuePeekMax {
		return QueuePeekMax
	}
	return f
}

func (q queue) scan(ctx context.Context, f osqueue.RunFunc) error {
	partitions, err := q.PartitionPeek(ctx, true, time.Now(), PartitionPeekMax)
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

		for _, item := range queue {
			if item.LeaseID != nil && ulid.Time(item.LeaseID.Time()).After(time.Now()) {
				// TODO: Increase metric for queue contention
				continue
			}
			q.workers <- item
		}

		// TODO: Re-quueue pointer, finding the next time available.
	}

	return nil
}
