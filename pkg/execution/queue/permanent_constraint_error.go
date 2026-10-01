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
		q.logPermanentConstraintRoutingOutcome(ctx, item, cause, "ignored", "handler_not_configured")
		return fmt.Errorf("permanent constraint error handler is not configured")
	}

	if err := handler(ctx, item, cause); err != nil {
		q.logPermanentConstraintRoutingOutcome(ctx, item, cause, "ignored", "cleanup_failed")
		return fmt.Errorf("permanent constraint error handler failed: %w", err)
	}

	if err := q.Shard().Dequeue(ctx, item); err != nil && !errors.Is(err, ErrQueueItemNotFound) {
		q.logPermanentConstraintRoutingOutcome(ctx, item, cause, "ignored", "dequeue_failed")
		return fmt.Errorf("dequeue permanently unroutable item: %w", err)
	}

	q.logPermanentConstraintRoutingOutcome(ctx, item, cause, "dropped", "handled")
	metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"status":            "constraint_shard_not_found",
			"outcome":           "dropped",
			"queue_shard":       q.Shard().Name(),
			"constraint_source": "constraint_api",
		},
	})

	return nil
}

func (q *queueProcessor) logPermanentConstraintRoutingOutcome(ctx context.Context, item QueueItem, cause error, outcome, reason string) {
	logger.StdlibLogger(ctx).Warn(
		"queue item has missing constraint shard",
		"error", cause,
		"outcome", outcome,
		"reason", reason,
		"queue_shard", q.Shard().Name(),
		"item_id", item.ID,
		"run_id", item.Data.Identifier.RunID,
		"account_id", item.Data.Identifier.AccountID,
		"env_id", item.Data.Identifier.WorkspaceID,
		"function_id", item.Data.Identifier.WorkflowID,
	)
}
