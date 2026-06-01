package queue

import (
	"context"
	"fmt"
	"time"
)

const defaultLatencyTrackerInterval = 5 * time.Second

func NewLatencyTrackerRole(opts ...QueueRoleOpt) QueueRole {
	return newLatencyTrackerRole(LatencyPartitionOptions{
		Partitions: 1,
		Interval:   defaultLatencyTrackerInterval,
	}, opts...)
}

func newLatencyTrackerRole(latency LatencyPartitionOptions, opts ...QueueRoleOpt) QueueRole {
	if latency.Partitions <= 0 {
		latency.Partitions = 1
	}
	if latency.Interval <= 0 {
		latency.Interval = defaultLatencyTrackerInterval
	}

	return newQueueRole(QueueRoleLatencyTracker, RoleLeaseDuration, latency.Interval, func(ctx context.Context, shard QueueShard) error {
		for i := 1; i <= latency.Partitions; i++ {
			if err := enqueueLatencyJob(ctx, shard, i, latency.Interval, time.Now()); err != nil {
				return err
			}
		}
		return nil
	}, opts...)
}

// runLatencyTracker is a background goroutine that periodically enqueues
// latency tracking canary jobs into the queue.
func (q *queueProcessor) runLatencyTracker(ctx context.Context) {
	if q.latencyPartition == nil {
		return
	}

	tick := q.Clock().NewTicker(q.latencyPartition.Interval)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.Chan():
			for i := 1; i <= q.latencyPartition.Partitions; i++ {
				_ = enqueueLatencyJob(ctx, q.Shard(), i, q.latencyPartition.Interval, q.Clock().Now())
			}
		}
	}
}

// enqueueLatencyJob enqueues a single latency tracking canary item into the
// queue shard.
func enqueueLatencyJob(ctx context.Context, shard QueueShard, partition int, interval time.Duration, at time.Time) error {
	jobID := fmt.Sprintf("ltrack-%d-%d", partition, at.UnixMilli())
	queueName := "ltc"
	item := Item{
		JobID:     &jobID,
		Kind:      KindLatencyTrack,
		QueueName: &queueName,
	}
	qi := QueueItem{
		ID:                jobID,
		AtMS:              at.UnixMilli(),
		WallTimeMS:        at.UnixMilli(),
		Data:              item,
		QueueName:         item.QueueName,
		IdempotencyPeriod: &interval,
	}

	_, err := shard.EnqueueItem(ctx, qi, at, EnqueueOpts{
		IdempotencyPeriod:   &interval,
		ForceQueueShardName: shard.Name(),
	})
	return err
}

// wrapRunFuncWithLatency wraps a RunFunc to intercept latency tracking items.
// When a latency tracking item is processed, it invokes the configured callback
// with the measured latency and returns immediately without calling the original RunFunc.
func (q *queueProcessor) wrapRunFuncWithLatency(f RunFunc) RunFunc {
	if q.latencyPartition == nil || q.latencyPartition.Callback == nil {
		return f
	}
	return func(ctx context.Context, info RunInfo, item Item) (RunResult, error) {
		if item.Kind == KindLatencyTrack {
			q.latencyPartition.Callback(ctx, info)
			return RunResult{}, nil
		}
		return f(ctx, info, item)
	}
}
