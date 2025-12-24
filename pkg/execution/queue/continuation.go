package queue

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"golang.org/x/sync/errgroup"
)

// addContinue adds a continuation for the given partition.  This hints that the queue should
// peek and process this partition on the next loop, allowing us to hint that a partition
// should be processed when a step finishes (to decrease inter-step latency on non-connect
// workloads).
func (q *queueProcessor) addContinue(ctx context.Context, p *QueuePartition, ctr uint) {
	if !q.runMode.Continuations {
		// continuations are not enabled.
		return
	}

	if ctr >= q.continuationLimit {
		q.removeContinue(ctx, p, true)
		return
	}

	q.continuesLock.Lock()
	defer q.continuesLock.Unlock()

	// If this is the first continuation, check if we're on a cooldown, or if we're
	// beyond capacity.
	if ctr == 1 {
		if len(q.continues) > consts.QueueContinuationMaxPartitions {
			metrics.IncrQueueContinuationMaxCapcityCounter(ctx, metrics.CounterOpt{PkgName: pkgName})
			return
		}
		if t, ok := q.continueCooldown[p.Queue()]; ok && t.After(time.Now()) {
			metrics.IncrQueueContinuationCooldownCounter(ctx, metrics.CounterOpt{PkgName: pkgName})
			return
		}

		// Remove the continuation cooldown.
		delete(q.continueCooldown, p.Queue())
	}

	c, ok := q.continues[p.Queue()]
	if !ok || c.count < ctr {
		// Update the continue count if it doesn't exist, or the current counter
		// is higher.  This ensures that we always have the highest continuation
		// count stored for queue processing.
		q.continues[p.Queue()] = continuation{partition: p, count: ctr}
		metrics.IncrQueueContinuationAddedCounter(ctx, metrics.CounterOpt{PkgName: pkgName})
	}
}

func (q *queueProcessor) removeContinue(ctx context.Context, p *QueuePartition, cooldown bool) {
	if !q.runMode.Continuations {
		// continuations are not enabled.
		return
	}

	// This is over the limit for conntinuing the partition, so force it to be
	// removed in every case.
	q.continuesLock.Lock()
	defer q.continuesLock.Unlock()

	metrics.IncrQueueContinuationRemovedCounter(ctx, metrics.CounterOpt{PkgName: pkgName})

	delete(q.continues, p.Queue())

	if cooldown {
		// Add a cooldown, preventing this partition from being added as a continuation
		// for a given period of time.
		//
		// Note that this isn't shared across replicas;  cooldowns
		// only exist in the current replica.
		q.continueCooldown[p.Queue()] = time.Now().Add(
			consts.QueueContinuationCooldownPeriod,
		)
	}
}

func (q *queueProcessor) scanContinuations(ctx context.Context) error {
	if !q.runMode.Continuations {
		// continuations are not enabled.
		return nil
	}

	// Have some chance of skipping continuations in this iteration.
	if rand.Float64() <= consts.QueueContinuationSkipProbability {
		return nil
	}

	eg := errgroup.Group{}
	// If we have continued partitions, process those immediately.
	q.continuesLock.Lock()
	for _, c := range q.continues {
		cont := c
		eg.Go(func() error {
			p := cont.partition
			if q.capacity() == 0 {
				// no longer any available workers for partition, so we can skip
				// work
				metrics.IncrQueueScanNoCapacityCounter(ctx, metrics.CounterOpt{PkgName: pkgName})
				return nil
			}

			q.log.Trace("continue partition processing", "partition_id", p.ID, "count", c.count)

			if err := q.processPartition(ctx, p, cont.count, false); err != nil {
				if err == ErrPartitionNotFound || err == ErrPartitionGarbageCollected {
					q.removeContinue(ctx, p, false)
					return nil
				}
				if errors.Unwrap(err) != context.Canceled {
					q.log.Error("error processing partition", "error", err)
				}
				return err
			}

			metrics.IncrQueuePartitionProcessedCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
			})
			return nil
		})
	}
	q.continuesLock.Unlock()
	return eg.Wait()
}
