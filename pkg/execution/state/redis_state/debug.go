package redis_state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/redis/rueidis"
)

type PartitionInspectionResult struct {
	QueuePartition       *QueuePartition
	QueueShadowPartition *QueueShadowPartition

	Paused            bool `json:"paused"`
	Migrate           bool `json:"migrate"`
	AccountActive     int  `json:"acct_active"`
	AccountInProgress int  `json:"acct_in_progress"`
	Ready             int  `json:"ready"`
	InProgress        int  `json:"in_progress"`
	Active            int  `json:"active"`
	Future            int  `json:"future"`
	Backlogs          int  `json:"backlogs"`
}

func (q *queue) InspectPartition(ctx context.Context, shard QueueShard, partitionID string) (*PartitionInspectionResult, error) {
	var (
		result PartitionInspectionResult
		sqp    QueueShadowPartition
	)

	rc := shard.RedisClient.Client()
	kg := shard.RedisClient.kg

	// load queue partition
	qp, err := q.PartitionByID(ctx, shard, partitionID)
	if err != nil {
		return nil, fmt.Errorf("could not load partition by ID: %w", err)
	}

	// load shadow partition
	{
		cmd := rc.B().Hget().Key(kg.ShadowPartitionMeta()).Field(partitionID).Build()
		byt, err := rc.Do(ctx, cmd).AsBytes()
		switch err {
		case rueidis.Nil:
			// no-op
			// there are cases shadow partition won't exists even when key queues are on.
			// e.g. everything is processed, and nothing new is being scheduled

		case nil:
			if err := json.Unmarshal(byt, &sqp); err != nil {
				return nil, fmt.Errorf("error unmarshalling shadow partition: %w", err)
			}
			result.QueueShadowPartition = &sqp

		default:
			return nil, fmt.Errorf("error retrieving shadow partition: %w", err)
		}
	}

	{
		keys := []string{
			kg.ActiveSet("account", qp.AccountID.String()),
			kg.Concurrency("account", qp.AccountID.String()),
			kg.PartitionQueueSet(enums.PartitionTypeDefault, qp.ID, ""),
			kg.Concurrency("p", qp.ID),
			kg.ActiveSet("p", qp.ID),
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

	// Fetch paused + migrating state
	if qp.FunctionID != nil {
		paused := q.partitionPausedGetter(ctx, *qp.FunctionID)
		result.Paused = paused.Paused

		locked, err := q.isMigrationLocked(ctx, shard, *qp.FunctionID)
		if err != nil {
			return nil, fmt.Errorf("could not get locked state: %w", err)
		}
		result.Migrate = locked != nil
	}

	return &result, nil
}
