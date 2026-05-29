package redis_state

import (
	"context"
	"errors"
	"fmt"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
)

func (q *queue) UnpauseFunction(ctx context.Context, scope osqueue.Scope) error {
	l := logger.StdlibLogger(ctx)

	part := &osqueue.QueuePartition{
		ID:         scope.FunctionID.String(),
		FunctionID: &scope.FunctionID,
		AccountID:  scope.AccountID,
		EnvID:      &scope.EnvID,
	}

	err := q.PartitionRequeue(ctx, part, q.Clock.Now(), false)
	if err != nil && !errors.Is(err, osqueue.ErrPartitionNotFound) && !errors.Is(err, osqueue.ErrPartitionGarbageCollected) {
		l.Error("failed to requeue unpaused partition", "error", err, "partition", part)
		return fmt.Errorf("could not unpause partition: %w", err)
	}

	// Also unpause shadow partition if key queues enabled
	if q.AllowKeyQueues(ctx, scope.AccountID, scope.EnvID, scope.FunctionID) {
		shadowPart := &osqueue.QueueShadowPartition{
			PartitionID: scope.FunctionID.String(),
			FunctionID:  &scope.FunctionID,
			AccountID:   &scope.AccountID,
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
