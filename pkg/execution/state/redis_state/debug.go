package redis_state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/redis/rueidis"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
)

func (q *queue) PartitionByID(ctx context.Context, partitionID string) (*osqueue.PartitionInspectionResult, error) {
	var (
		result osqueue.PartitionInspectionResult
		qp     osqueue.QueuePartition
		sqp    osqueue.QueueShadowPartition
	)

	rc := q.RedisClient.Client()
	kg := q.RedisClient.kg

	// load queue partition
	{
		cmd := rc.B().Hget().Key(kg.PartitionItem()).Field(partitionID).Build()
		byt, err := rc.Do(ctx, cmd).AsBytes()
		if err != nil {
			if rueidis.IsRedisNil(err) {
				return nil, osqueue.ErrPartitionNotFound
			}

			return nil, fmt.Errorf("error retrieving queue partition: %w", err)
		}

		if err := json.Unmarshal(byt, &qp); err != nil {
			return nil, fmt.Errorf("error unmarshalling queue partition: %w", err)
		}
		result.QueuePartition = &qp
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
			q.Clock.Now().UnixMilli(),
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
		paused := q.PartitionPausedGetter(ctx, *qp.FunctionID)
		result.Paused = paused.Paused

		locked, err := q.isMigrationLocked(ctx, *qp.FunctionID)
		if err != nil {
			return nil, fmt.Errorf("could not get locked state: %w", err)
		}
		result.Migrate = locked != nil
	}

	return &result, nil
}
