package redis_state

import (
	"context"
	"errors"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

// ValidateItemForHint applies discovery eligibility without fetching the item.
// The atomic lease checks stored existence, lease, due time and generation.
func (q *queue) ValidateItemForHint(ctx context.Context, item osqueue.QueueItem) error {
	if item.Data.Kind != osqueue.KindStart || item.Data.Attempt != 0 || item.AtMS > q.Clock.Now().UnixMilli() {
		return osqueue.ErrQueueItemNotReady
	}
	p := osqueue.ItemPartition(ctx, item)
	if p.IsSystem() || (len(q.AllowQueues) > 0 && !checkList(p.Queue(), q.AllowQueueMap, q.AllowQueuePrefixes)) ||
		(len(q.DenyQueues) > 0 && checkList(p.Queue(), q.DenyQueueMap, q.DenyQueuePrefixes)) {
		return osqueue.ErrQueueItemNotReady
	}
	if q.PartitionPausedGetter(ctx, item.FunctionID).Paused || ctx.Err() != nil {
		return osqueue.ErrQueueItemNotReady
	}
	kg, rc := q.RedisClient.kg, q.RedisClient.unshardedRc
	lock, err := rc.Do(ctx, rc.B().Get().Key(kg.QueueMigrationLock(item.FunctionID)).Build()).ToString()
	if err != nil && !errors.Is(err, rueidis.Nil) {
		return err
	}
	if lock != "" {
		lease, err := ulid.Parse(lock)
		if err != nil || ulid.Time(lease.Time()).After(q.Clock.Now()) {
			return osqueue.ErrQueueItemNotReady
		}
	}
	return nil
}
