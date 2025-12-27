package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

const (
	pkgName = "queue.processor"
)

// Enqueue adds an item to the queue to be processed at the given time.
// TODO: Lift this function and the queue interface to a higher level, so that it's disconnected from the
// concrete Redis implementation.
func (q *queueProcessor) Enqueue(ctx context.Context, item Item, at time.Time, opts EnqueueOpts) error {
	l := logger.StdlibLogger(ctx)

	// propagate
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}

	id := ""
	if item.JobID != nil {
		id = *item.JobID
	}

	if item.QueueName == nil {
		// Check if we have a kind mapping.
		if name, ok := q.queueKindMapping[item.Kind]; ok {
			item.QueueName = &name
		}
	}

	qi := QueueItem{
		ID:          id,
		AtMS:        at.UnixMilli(),
		WorkspaceID: item.WorkspaceID,
		FunctionID:  item.Identifier.WorkflowID,
		Data:        item,
		QueueName:   item.QueueName,
		WallTimeMS:  at.UnixMilli(),
	}

	l = l.With(
		"item", qi,
		"account_id", item.Identifier.AccountID,
		"env_id", item.WorkspaceID,
		"app_id", item.Identifier.AppID,
		"fn_id", item.Identifier.WorkflowID,
		"queue_shard", q.primaryQueueShard.Name(),
	)

	if item.QueueName == nil && qi.FunctionID == uuid.Nil {
		err := fmt.Errorf("queue name or function ID must be set")
		l.ReportError(err, "attempted to enqueue QueueItem without function ID or queueName override")
		return err
	}

	// Pass optional idempotency period to queue item
	if opts.IdempotencyPeriod != nil {
		qi.IdempotencyPeriod = opts.IdempotencyPeriod
	}

	// Use the queue item's score, ensuring we process older function runs first
	// (eg. before at)
	next := time.UnixMilli(qi.Score(q.Clock().Now()))

	if factor := qi.Data.GetPriorityFactor(); factor != 0 {
		// Ensure we mutate the AtMS time by the given priority factor.
		qi.AtMS -= factor
	}

	shard, err := q.selectShard(ctx, opts.ForceQueueShardName, qi)
	if err != nil {
		return err
	}

	metrics.IncrQueueItemStatusCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"status":      "enqueued",
			"kind":        item.Kind,
			"queue_shard": shard.Name,
		},
	})

	switch shard.Kind() {
	case enums.QueueShardKindRedis:
		_, err := shard.EnqueueItem(ctx, qi, next, opts)
		if err != nil {
			return err
		}

		// XXX: If we've enqueued a user queue item (sleep, retry, step, etc.) and it's in the future,
		// we want to ensure that we schedule a rebalance job which takes the queue item and places it
		// at the correct score based off of the item's run ID when it becomes available.
		//
		// Without this, step.sleep or retries for a very old workflow may still lag behind steps from
		// later workflows when scheduled in the future.  This can, worst case, cause never-ending runs.
		if !q.enableJobPromotion || !qi.RequiresPromotionJob(q.Clock().Now()) {
			// scheule a rebalance job automatically.
			return nil
		}

		// This is to prevent infinite recursion in case RequiresPromotion is accidentally refactored
		// to include the below job kind.
		if qi.Data.Kind == KindJobPromote {
			return nil
		}

		// This is the fudge job.  What a name!
		//
		// If we're processing a user function and the sleep duration is in the future,
		// enqueue a sleep scavenge system queue item that will Requeue the original sleep queue item.
		// We do this to fudge the original queue item at the exact time, the run was scheduled for to ensure
		// sleeps for existing function runs are picked up earlier than items for later function runs.
		promoteAt := time.UnixMilli(qi.AtMS).Add(consts.FutureAtLimit * -1)
		promoteJobID := fmt.Sprintf("promote-%s", qi.ID)
		promoteQueueName := fmt.Sprintf("job-promote:%s", qi.FunctionID)
		err = q.Enqueue(ctx, Item{
			JobID:          &promoteJobID,
			WorkspaceID:    qi.Data.WorkspaceID,
			QueueName:      &promoteQueueName,
			Kind:           KindJobPromote,
			Identifier:     qi.Data.Identifier,
			PriorityFactor: qi.Data.PriorityFactor,
			Attempt:        0,
			Payload: PayloadJobPromote{
				PromoteJobID: qi.ID,
				ScheduledAt:  qi.AtMS,
			},
		}, promoteAt, EnqueueOpts{})
		if err != nil && err != ErrQueueItemExists {
			// This is best effort, and shouldn't fail the OG enqueue.
			l.ReportError(err, "error scheduling promotion job")
		}
		return nil
	default:
		return fmt.Errorf("unknown shard kind: %s", shard.Kind)
	}
}
