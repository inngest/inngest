package redis_state

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (q *queue) PartitionByID(ctx context.Context, shard QueueShard, partitionID uuid.UUID) (*QueuePartition, *QueueShadowPartition, error) {
	return nil, nil, fmt.Errorf("not implemented")
}
