package redis_state

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
)

func (q *queue) UnpauseFunction(ctx context.Context, acctID, fnID uuid.UUID) error {
	l := logger.StdlibLogger(ctx)

	part := &osqueue.QueuePartition{
		ID:         fnID.String(),
		FunctionID: &fnID,
		AccountID:  acctID,
	}

	err := q.PartitionRequeue(ctx, part, q.Clock.Now(), false)
	if err != nil && !errors.Is(err, osqueue.ErrPartitionNotFound) && !errors.Is(err, osqueue.ErrPartitionGarbageCollected) {
		l.Error("failed to requeue unpaused partition", "error", err, "partition", part)
		return fmt.Errorf("could not unpause partition: %w", err)
	}

	// Also unpause shadow partition if key queues enabled
	if q.AllowKeyQueues(ctx, acctID, fnID) {
		shadowPart := &osqueue.QueueShadowPartition{
			PartitionID: fnID.String(),
			FunctionID:  &fnID,
			AccountID:   &acctID,
		}
		// requeue to earliest item
		err = q.ShadowPartitionRequeue(ctx, shadowPart, nil)
		if err != nil && !errors.Is(err, osqueue.ErrShadowPartitionNotFound) {
			return fmt.Errorf("could not unpause shadow partition: %w", err)
		}
	}

	l.Trace("requeued unpaused partition", "partition", part.Queue())
	return nil
}
