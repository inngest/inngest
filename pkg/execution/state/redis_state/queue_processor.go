package redis_state

import (
	"context"
	"time"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

func (q *queue) PeekAccountPartitions(
	ctx context.Context,
	accountID uuid.UUID,
	peekLimit int64,
	peekUntil time.Time,
	sequential bool,
) ([]*osqueue.QueuePartition, error) {
	return q.timedPartitionPeek(ctx, &accountID, sequential, peekUntil, peekLimit, false)
}

// PeekPartitionsForScan leaves the pause check to partition workers when
// enabled. Direct peeks retain the synchronous pause check.
func (q *queue) PeekPartitionsForScan(ctx context.Context, accountID *uuid.UUID, sequential bool, until time.Time, limit int64) ([]*osqueue.QueuePartition, func(context.Context, *osqueue.QueuePartition) bool, error) {
	partitions, err := q.timedPartitionPeek(ctx, accountID, sequential, until, limit, q.pauseCheckSlots != nil)
	if err != nil || q.pauseCheckSlots == nil {
		return partitions, nil, err
	}
	global := accountID == nil
	tags := map[string]any{"is_global_partition_peek": global}
	return partitions, func(ctx context.Context, p *osqueue.QueuePartition) bool {
		if p.FunctionID == nil {
			return true
		}

		if q.CachedPartitionPauseGetter != nil {
			if paused, found := q.CachedPartitionPauseGetter(*p.FunctionID); found {
				return !paused && ctx.Err() == nil
			}
		}

		// A scan can return while its already-admitted workers are still
		// checking pauses. Skip duplicate checks before taking a lookup slot.
		if _, pending := q.pauseChecksInFlight.LoadOrStore(p.ID, struct{}{}); pending {
			return false
		}
		defer q.pauseChecksInFlight.Delete(p.ID)

		if ctx.Err() != nil {
			return false
		}
		// Keep partition workers available for ready work. The unleased
		// pointer remains eligible for a later scan when lookup slots free up.
		if !q.pauseCheckSlots.TryAcquire(1) {
			metrics.RecordCounterMetric(ctx, 1, metrics.CounterOpt{
				PkgName:     pkgName,
				MetricName:  "queue_partition_pause_check_slots_busy_total",
				Description: "Partition pause checks deferred because all lookup slots are busy",
				Tags: map[string]any{
					"queue_shard":              q.name,
					"is_global_partition_peek": global,
				},
			})
			return false
		}
		defer q.pauseCheckSlots.Release(1)

		var timing partitionPauseTimings
		paused, _ := osqueue.DurationWithTags(ctx, q.name, "partition_pause_check", time.Time{}, func(ctx context.Context) (bool, error) {
			return q.partitionPaused(ctx, p, &timing, true), nil
		}, tags, func() []any {
			return append([]any{
				"queue_shard", q.name,
				"is_global_partition_peek", global,
			}, timing.logFields()...)
		})
		return !paused && ctx.Err() == nil
	}, nil
}
