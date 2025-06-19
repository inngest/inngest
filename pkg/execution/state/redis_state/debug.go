package redis_state

import (
	"context"
	"fmt"
)

func (q *queue) PartitionByID(ctx context.Context, shard QueueShard, partitionID string) (*QueuePartition, *QueueShadowPartition, error) {
	return nil, nil, fmt.Errorf("not implemented")
}
