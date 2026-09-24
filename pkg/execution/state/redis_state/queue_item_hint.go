package redis_state

import (
	"context"
	"errors"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

// LoadItemForHint reloads authoritative state and applies the eligibility checks
// normally made during discovery. Backlog items use the ordinary constrained
// item lease directly; hints do not acquire refill ownership or move queue sets.
func (q *queue) LoadItemForHint(ctx context.Context, id string) (*osqueue.QueueItem, error) {
	item, err := q.LoadQueueItem(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.Data.Kind != osqueue.KindStart || item.Data.Attempt != 0 || item.AtMS > q.Clock.Now().UnixMilli() {
		return nil, osqueue.ErrQueueItemNotReady
	}
	p := osqueue.ItemPartition(ctx, *item)
	if p.IsSystem() || (len(q.AllowQueues) > 0 && !checkList(p.Queue(), q.AllowQueueMap, q.AllowQueuePrefixes)) ||
		(len(q.DenyQueues) > 0 && checkList(p.Queue(), q.DenyQueueMap, q.DenyQueuePrefixes)) {
		return nil, osqueue.ErrQueueItemNotReady
	}
	if q.PartitionPausedGetter(ctx, item.FunctionID).Paused || ctx.Err() != nil {
		return nil, osqueue.ErrQueueItemNotReady
	}
	kg, rc := q.RedisClient.kg, q.RedisClient.unshardedRc
	lock, err := rc.Do(ctx, rc.B().Get().Key(kg.QueueMigrationLock(item.FunctionID)).Build()).ToString()
	if err != nil && !errors.Is(err, rueidis.Nil) {
		return nil, err
	}
	if lock != "" {
		lease, err := ulid.Parse(lock)
		if err != nil || ulid.Time(lease.Time()).After(q.Clock.Now()) {
			return nil, osqueue.ErrQueueItemNotReady
		}
	}
	// Observe backlog residency without making it an admission requirement.
	backlog := osqueue.ItemBacklog(ctx, *item)
	if _, err := rc.Do(ctx, rc.B().Zscore().Key(kg.BacklogSet(backlog.BacklogID)).Member(id).Build()).ToFloat64(); err == nil {
		metrics.RecordCounterMetric(ctx, 1, metrics.CounterOpt{PkgName: pkgName, MetricName: "queue_item_hint_backlog_resident_total", Tags: map[string]any{"queue_shard": q.Name()}})
	}
	return item, nil
}
