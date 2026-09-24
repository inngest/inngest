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
	item, err := q.loadHintItem(ctx, id)
	if err != nil {
		return nil, err
	}
	rc := q.RedisClient.unshardedRc
	_, err = rc.Do(ctx, rc.B().Zscore().Key(shadowPartitionReadyQueueKey(osqueue.ItemShadowPartition(ctx, *item), q.RedisClient.kg)).Member(id).Build()).ToFloat64()
	if errors.Is(err, rueidis.Nil) {
		return nil, osqueue.ErrQueueItemNotReady
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (q *queue) loadHintItem(ctx context.Context, id string) (*osqueue.QueueItem, error) {
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

	return item, nil
}

// LoadBacklogItem is a point lookup of the stored backlog metadata and membership.
// Callers must use the ordinary shadow lease/refill path before item leasing.
func (q *queue) LoadBacklogItem(ctx context.Context, id string) (*osqueue.QueueItem, *osqueue.QueueBacklog, error) {
	item, err := q.loadHintItem(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	computed := osqueue.ItemBacklog(ctx, *item)
	backlog, err := q.BacklogByID(ctx, computed.BacklogID)
	if err != nil {
		return nil, nil, err
	}
	rc := q.RedisClient.unshardedRc
	score, err := rc.Do(ctx, rc.B().Zscore().Key(q.RedisClient.kg.BacklogSet(backlog.BacklogID)).Member(id).Build()).ToFloat64()
	if errors.Is(err, rueidis.Nil) || (err == nil && score > float64(q.Clock.Now().UnixMilli())) {
		return nil, nil, osqueue.ErrQueueItemNotReady
	}
	if err != nil {
		return nil, nil, err
	}
	return item, backlog, nil
}
