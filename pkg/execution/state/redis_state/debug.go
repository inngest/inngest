package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
)

type PartitionInspectionResult struct {
	QueuePartition       *QueuePartition
	QueueShadowPartition *QueueShadowPartition

	Paused            bool `json:"paused"`
	AccountActive     int  `json:"acct_active"`
	AccountInProgress int  `json:"acct_in_progress"`
	Ready             int  `json:"ready"`
	InProgress        int  `json:"in_progress"`
	Active            int  `json:"active"`
	Future            int  `json:"future"`
	Backlogs          int  `json:"backlogs"`
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

	var result PartitionInspectionResult
	{
		keys := []string{
			sqp.accountActiveKey(kg),
			sqp.accountInProgressKey(kg),
			sqp.readyQueueKey(kg),
			sqp.inProgressKey(kg),
			sqp.activeKey(kg),
			kg.ShadowPartitionSet(sqp.PartitionID),
		}
		args, err := StrSlice([]any{
			q.clock.Now().UnixMilli(),
		})
		if err != nil {
			return nil, fmt.Errorf("error preparing args for redis: %w", err)
		}

		byt, err := scripts["queue/countCheck"].Exec(
			ctx,
			rc,
			keys,
			args,
		).AsBytes()
		if err != nil {
			return nil, fmt.Errorf("error retriving counters: %w", err)
		}

		if err := json.Unmarshal(byt, &result); err != nil {
			return nil, fmt.Errorf("error unmarhalling counter check: %w", err)
		}
	}

	result.QueuePartition = &qp
	result.QueueShadowPartition = &sqp

	return &result, nil
}
