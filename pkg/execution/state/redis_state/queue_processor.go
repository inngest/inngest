package redis_state

import (
	"context"
	"time"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
)

func (q *queue) PeekAccountPartitions(
	ctx context.Context,
	accountID uuid.UUID,
	peekLimit int64,
	peekUntil time.Time,
	sequential bool,
) ([]*osqueue.QueuePartition, error) {
	partitionKey := q.RedisClient.kg.AccountPartitionIndex(accountID)

	// Peek 1s into the future to pull jobs off ahead of time, minimizing 0 latency
	partitions, err := osqueue.DurationWithTags(ctx, q.name, "partition_peek", q.Clock.Now(), func(ctx context.Context) ([]*osqueue.QueuePartition, error) {
		return q.partitionPeek(ctx, partitionKey, sequential, peekUntil, peekLimit, &accountID)
	}, map[string]any{
		"is_global_partition_peek": false,
	})
	if err != nil {
		return nil, err
	}

	return partitions, nil
}

func (q *queue) PeekGlobalPartitions(
	ctx context.Context,
	peekLimit int64,
	peekUntil time.Time,
	sequential bool,
) ([]*osqueue.QueuePartition, error) {
	partitionKey := q.RedisClient.kg.GlobalPartitionIndex()

	// Peek 1s into the future to pull jobs off ahead of time, minimizing 0 latency
	partitions, err := osqueue.DurationWithTags(ctx, q.name, "partition_peek", q.Clock.Now(), func(ctx context.Context) ([]*osqueue.QueuePartition, error) {
		return q.partitionPeek(ctx, partitionKey, sequential, peekUntil, peekLimit, nil)
	}, map[string]any{
		"is_global_partition_peek": true,
	})
	if err != nil {
		return nil, err
	}

	return partitions, nil
}
