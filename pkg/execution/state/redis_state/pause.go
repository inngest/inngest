package redis_state

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

func (q *queue) UnpauseFunction(ctx context.Context, shardName string, acctID, fnID uuid.UUID) error {
	shard, ok := q.queueShardClients[shardName]
	if !ok {
		return fmt.Errorf("invalid shard %q", shardName)
	}

	part := &QueuePartition{
		ID:         fnID.String(),
		FunctionID: &fnID,
		AccountID:  acctID,
	}

	err := q.PartitionRequeue(ctx, shard, part, q.clock.Now(), false)
	if err != nil && !errors.Is(err, ErrPartitionNotFound) && !errors.Is(err, ErrPartitionGarbageCollected) {
		q.log.Error("failed to requeue unpaused partition", "error", err, "partition", part)
		return fmt.Errorf("could not unpause partition: %w", err)
	}

	// Also unpause shadow partition if key queues enabled
	if q.allowKeyQueues(ctx, acctID) {
		shadowPart := &QueueShadowPartition{
			PartitionID: fnID.String(),
			FunctionID:  &fnID,
			AccountID:   &acctID,
		}
		// requeue to earliest item
		err = q.ShadowPartitionRequeue(ctx, shadowPart, nil)
		if err != nil && !errors.Is(err, ErrShadowPartitionNotFound) {
			return fmt.Errorf("could not unpause shadow partition: %w", err)
		}
	}

	q.log.Trace("requeued unpaused partition", "partition", part.Queue())
	return nil
}
