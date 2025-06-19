package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
)

type PartitionInspectionResult struct {
	QueuePartition       *QueuePartition
	QueueShadowPartition *QueueShadowPartition
}

func (q *queue) PartitionByID(ctx context.Context, shard QueueShard, partitionID string) (*PartitionInspectionResult, error) {
	var (
		qp  QueuePartition
		sqp QueueShadowPartition
	)

	rc := shard.RedisClient.Client()
	kg := shard.RedisClient.kg
	// load queue partition
	{
		cmd := rc.B().Hget().Key(kg.PartitionItem()).Field(partitionID).Build()
		byt, err := rc.Do(ctx, cmd).AsBytes()
		if err != nil {
			return nil, fmt.Errorf("error retrieving queue partition: %w", err)
		}

		if err := json.Unmarshal(byt, &qp); err != nil {
			return nil, fmt.Errorf("error unmarshalling queue partition: %w", err)
		}
	}

	// load shadow partition
	{
		cmd := rc.B().Hget().Key(kg.ShadowPartitionMeta()).Field(partitionID).Build()
		byt, err := rc.Do(ctx, cmd).AsBytes()
		if err != nil {
			return nil, fmt.Errorf("error retrieving shadow partition: %w", err)
		}

		if err := json.Unmarshal(byt, &sqp); err != nil {
			return nil, fmt.Errorf("error unmarshalling shadow partition: %w", err)
		}
	}

	return &PartitionInspectionResult{
		QueuePartition:       &qp,
		QueueShadowPartition: &sqp,
	}, nil
}
