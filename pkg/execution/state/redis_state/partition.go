package redis_state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/rueidis"
)

func (q *queue) PartitionByID(ctx context.Context, shard QueueShard, partitionID string) (*QueuePartition, error) {
	rc := shard.RedisClient.Client()
	kg := shard.RedisClient.kg

	// load queue partition
	cmd := rc.B().Hget().Key(kg.PartitionItem()).Field(partitionID).Build()
	byt, err := rc.Do(ctx, cmd).AsBytes()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, ErrPartitionNotFound
		}

		return nil, fmt.Errorf("error retrieving queue partition: %w", err)
	}

	var qp QueuePartition
	if err := json.Unmarshal(byt, &qp); err != nil {
		return nil, fmt.Errorf("error unmarshalling queue partition: %w", err)
	}

	return &qp, nil
}
