package queue

import (
	"context"
	"errors"
	"fmt"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

func (q *queueProcessor) dropPermanentlyUnroutableItem(ctx context.Context, item QueueItem, cause error) error {
	handler := q.Options().PermanentConstraintErrorHandler
	if handler == nil {
		return fmt.Errorf("permanent constraint error handler is not configured")
	}

	if err := handler(ctx, item, cause); err != nil {
		return fmt.Errorf("permanent constraint error handler failed: %w", err)
	}

	if err := q.Shard().Dequeue(ctx, item); err != nil && !errors.Is(err, ErrQueueItemNotFound) {
		return fmt.Errorf("dequeue permanently unroutable item: %w", err)
	}

	logger.StdlibLogger(ctx).Warn(
		"dropped queue item with missing constraint shard",
		"error", cause,
		"queue_shard", q.Shard().Name(),
		"item_id", item.ID,
		"run_id", item.Data.Identifier.RunID,
		"account_id", item.Data.Identifier.AccountID,
		"env_id", item.Data.Identifier.WorkspaceID,
		"function_id", item.Data.Identifier.WorkflowID,
	)
	metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"status":            "constraint_shard_not_found",
			"queue_shard":       q.Shard().Name(),
			"constraint_source": "constraint_api",
		},
	})

	return nil
}
