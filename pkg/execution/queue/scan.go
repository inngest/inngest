package queue

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

func (q *queueProcessor) executionScan(ctx context.Context, f RunFunc) error {
	l := q.log.With(
		"queue_shard", q.PrimaryQueueShard.Name(),
	)

	for i := int32(0); i < q.numWorkers; i++ {
		go q.worker(ctx, f)
	}

	if !q.runMode.Partition && !q.runMode.Account {
		return fmt.Errorf("need to specify either partition, account, or both in queue run mode")
	}

	tick := q.clock.NewTicker(q.pollTick)
	l.Debug("starting queue worker", "poll", q.pollTick.String())

	backoff := time.Millisecond * 250

	var err error
LOOP:
	for {
		select {
		case <-ctx.Done():
			// Kill signal
			tick.Stop()
			break LOOP
		case err = <-q.quit:
			// An inner function received an error which was deemed irrecoverable, so
			// we're quitting the queue.
			q.log.ReportError(err, "quitting runner internally")
			tick.Stop()
			break LOOP

		case <-tick.Chan():
			if q.capacity() < minWorkersFree {
				// Wait until we have more workers free.  This stops us from
				// claiming a partition to work on a single job, ensuring we
				// have capacity to run at least MinWorkersFree concurrent
				// QueueItems.  This reduces latency of enqueued items when
				// there are lots of enqueued and available jobs.
				l.Trace("all workers busy, early exiting scan", "worker_capacity", q.capacity())
				continue
			}

			if err = q.scan(ctx); err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					q.log.Warn("deadline exceeded scanning partition pointers")
					<-time.After(backoff)

					// Backoff doubles up to 3 seconds.
					backoff = time.Duration(math.Min(float64(backoff*2), float64(time.Second*5)))
					continue
				}

				// On scan errors, halt the worker entirely.
				if !errors.Is(err, context.Canceled) {
					q.log.ReportError(err, "error scanning partition pointers")
				}
				break LOOP
			}

			backoff = time.Millisecond * 250
		}
	}

	// Wait for all in-progress items to complete.
	q.log.Info("queue waiting to quit")
	q.wg.Wait()

	return err
}
