package redis_state

import (
	"context"
	"fmt"
)

type PartitionInspectionResult struct {
	QueuePartition       *QueuePartition
	QueueShadowPartition *QueueShadowPartition
}

func (q *queue) PartitionByID(ctx context.Context, shard QueueShard, partitionID string) (*PartitionInspectionResult, error) {
	return nil, fmt.Errorf("not implemented")
}
