package queue

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

var (
	LatencyAccountID  = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	LatencyEnvID      = uuid.MustParse("ffffffff-ffff-ffff-ffff-fffffffffffe")
	LatencyFunctionID = uuid.MustParse("ffffffff-ffff-ffff-ffff-fffffffffffd")
)

// IsLatencyPartition reports whether the given partition ID belongs to
// the latency tracking function.
func IsLatencyPartition(id string) bool {
	return id == LatencyFunctionID.String()
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
				_ = q.enqueueLatencyJob(ctx, i)
			}
		}
	}
}

// enqueueLatencyJob enqueues a single latency tracking canary item into the
// given partition number.
func (q *queueProcessor) enqueueLatencyJob(ctx context.Context, partition int) error {
	jobID := fmt.Sprintf("ltrack-%d-%d", partition, q.Clock().Now().UnixMilli())
	idempotency := q.latencyPartition.Interval

	return q.Enqueue(ctx, Item{
		JobID: &jobID,
		Kind:  KindLatencyTrack,
	}, q.Clock().Now(), EnqueueOpts{
		IdempotencyPeriod:   &idempotency,
		ForceQueueShardName: q.primaryQueueShard.Name(),
	})
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
