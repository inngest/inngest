package redis_state

import (
	"context"
	"errors"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

// LoadReadyItem performs point reads, preserving the eligibility checks normally
// made during partition discovery without scanning or taking a partition lease.
func (q *queue) LoadReadyItem(ctx context.Context, id string) (*osqueue.QueueItem, error) {
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
	kg := q.RedisClient.kg
	rc := q.RedisClient.unshardedRc
	results := rc.DoMulti(ctx,
		rc.B().Get().Key(kg.QueueMigrationLock(item.FunctionID)).Build(),
		rc.B().Zscore().Key(shadowPartitionReadyQueueKey(osqueue.ItemShadowPartition(ctx, *item), kg)).Member(id).Build(),
	)
	lock, err := results[0].ToString()
	if err != nil && !errors.Is(err, rueidis.Nil) {
		return nil, err
	}
	if lock != "" {
		lease, err := ulid.Parse(lock)
		if err != nil || ulid.Time(lease.Time()).After(q.Clock.Now()) {
			return nil, osqueue.ErrQueueItemNotReady
		}
	}
	if _, err := results[1].ToFloat64(); err != nil {
		if errors.Is(err, rueidis.Nil) {
			return nil, osqueue.ErrQueueItemNotReady
		}
		return nil, err
	}
	return item, nil
}
