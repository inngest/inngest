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

			if err := q.ProcessPartition(ctx, p, cont.count, false); err != nil {
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

// AddShadowContinue is the equivalent of addContinue for shadow partitions
func (q *queueProcessor) AddShadowContinue(ctx context.Context, p *QueueShadowPartition, ctr uint) {
	if !q.runMode.ShadowContinuations {
		// shadow continuations are not enabled.
		return
	}

	if ctr >= q.shadowContinuationLimit {
		q.removeShadowContinue(ctx, p, true)
		return
	}

	q.shadowContinuesLock.Lock()
	defer q.shadowContinuesLock.Unlock()

	// If this is the first shadow continuation, check if we're on a cooldown, or if we're
	// beyond capacity.
	if ctr == 1 {
		if len(q.shadowContinues) > consts.QueueShadowContinuationMaxPartitions {
			metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name(), "op": "max_capacity"}})
			return
		}
		if t, ok := q.shadowContinueCooldown[p.PartitionID]; ok && t.After(time.Now()) {
			metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name(), "op": "cooldown"}})
			return
		}

		// Remove the shadow continuation cooldown.
		delete(q.shadowContinueCooldown, p.PartitionID)
	}

	c, ok := q.shadowContinues[p.PartitionID]
	if !ok || c.Count < ctr {
		// Update the continue count if it doesn't exist, or the current counter
		// is higher.  This ensures that we always have the highest continuation
		// count stored for queue processing.
		q.shadowContinues[p.PartitionID] = ShadowContinuation{ShadowPart: p, Count: ctr}
		metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name(), "op": "added"}})
	}
}

func (q *queueProcessor) removeShadowContinue(ctx context.Context, p *QueueShadowPartition, cooldown bool) {
	if !q.runMode.ShadowContinuations {
		// shadow continuations are not enabled.
		return
	}

	// This is over the limit for continuing the shadow partition, so force it to be
	// removed in every case.
	q.shadowContinuesLock.Lock()
	defer q.shadowContinuesLock.Unlock()

	metrics.IncrQueueShadowContinuationOpCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name(), "op": "removed"}})

	delete(q.shadowContinues, p.PartitionID)

	if cooldown {
		// Add a cooldown, preventing this partition from being added as a continuation
		// for a given period of time.
		//
		// Note that this isn't shared across replicas;  cooldowns
		// only exist in the current replica.
		q.shadowContinueCooldown[p.PartitionID] = time.Now().Add(
			consts.QueueShadowContinuationCooldownPeriod,
		)
	}
}

func (q *queueProcessor) scanShadowContinuations(ctx context.Context) error {
	if !q.runMode.ShadowContinuations {
		return nil
	}

	if rand.Float64() <= q.runMode.ShadowContinuationSkipProbability {
		return nil
	}

	eg := errgroup.Group{}
	q.shadowContinuesLock.Lock()
	for _, c := range q.shadowContinues {
		cont := c
		eg.Go(func() error {
			sp := cont.ShadowPart

			_, err := DurationWithTags(
				ctx,
				q.primaryQueueShard.Name(),
				"shadow_partition_process_duration",
				q.Clock().Now(),
				func(ctx context.Context) (any, error) {
					err := q.ProcessShadowPartition(ctx, sp, cont.Count)
					if errors.Is(err, context.Canceled) {
						return nil, nil
					}
					return nil, err
				},
				map[string]any{
					// "partition_id": sp.PartitionID,
				},
			)
			if err != nil {
				if err == ErrShadowPartitionLeaseNotFound {
					return nil
				}
				if !errors.Is(err, context.Canceled) {
					q.log.Error("error processing shadow partition", "error", err, "continuation", true, "continuation_count", cont.Count)
				}
				return err
			}

			return nil
		})
	}
	q.shadowContinuesLock.Unlock()
	return eg.Wait()
}
