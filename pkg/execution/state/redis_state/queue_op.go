package redis_state

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
)

func (q *queue) DequeueByJobID(ctx context.Context, jobID string) error {
	item, err := q.ItemByID(ctx, jobID)
	switch err {
	case nil:
		// no-op
	case osqueue.ErrQueueItemNotFound:
		return nil
	default:
		return fmt.Errorf("error retrieving item by ID: %w", err)
	}

	return q.Dequeue(ctx, *item)
}

// Dequeue removes an item from the queue entirely.
func (q *queue) Dequeue(ctx context.Context, i osqueue.QueueItem, options ...osqueue.DequeueOptionFn) error {
	l := logger.StdlibLogger(ctx)

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Dequeue"), redis_telemetry.ScopeQueue)

	o := &osqueue.DequeueOptions{}
	for _, opt := range options {
		opt(o)
	}

	kg := q.RedisClient.kg

	partition := osqueue.ItemShadowPartition(ctx, i)
	backlog := osqueue.ItemBacklog(ctx, i)

	keys := []string{
		kg.QueueItem(),
		kg.PartitionItem(),

		kg.ConcurrencyIndex(),

		shadowPartitionReadyQueueKey(partition, kg),
		kg.GlobalPartitionIndex(),
		kg.GlobalAccountIndex(),
		kg.AccountPartitionIndex(i.Data.Identifier.AccountID),

		kg.ShadowPartitionMeta(),
		kg.BacklogMeta(),

		kg.BacklogSet(backlog.BacklogID),
		kg.ShadowPartitionSet(partition.PartitionID),
		kg.GlobalShadowPartitionSet(),
		kg.GlobalAccountShadowPartitions(),
		kg.AccountShadowPartitions(i.Data.Identifier.AccountID),
		kg.PartitionNormalizeSet(partition.PartitionID),

		// In progress keys
		shadowPartitionAccountInProgressKey(partition, kg),
		shadowPartitionInProgressKey(partition, kg),
		backlogCustomKeyInProgress(backlog, kg, 1),
		backlogCustomKeyInProgress(backlog, kg, 2),

		// Active set keys
		shadowPartitionAccountActiveKey(partition, kg),
		shadowPartitionActiveKey(partition, kg),
		backlogCustomKeyActive(backlog, kg, 1),
		backlogCustomKeyActive(backlog, kg, 2),
		backlogActiveKey(backlog, kg),

		// Active run sets
		kg.RunActiveSet(i.Data.Identifier.RunID),          // Set for active items in run
		shadowPartitionAccountActiveRunKey(partition, kg), // Set for active runs in account
		shadowPartitionActiveRunKey(partition, kg),        // Set for active runs in partition
		backlogCustomKeyActiveRuns(backlog, kg, 1),        // Set for active runs with custom concurrency key 1
		backlogCustomKeyActiveRuns(backlog, kg, 2),        // Set for active runs with custom concurrency key 2

		kg.Idempotency(i.ID),

		// Singleton
		kg.SingletonRunKey(i.Data.Identifier.RunID.String()),

		kg.PartitionScavengerIndex(partition.PartitionID),
	}

	// Append indexes
	for _, idx := range q.itemIndexer(ctx, i, q.RedisClient.kg) {
		if idx != "" {
			keys = append(keys, idx)
		}
	}

	idempotency := q.IdempotencyTTL
	if q.IdempotencyTTLFunc != nil {
		idempotency = q.IdempotencyTTLFunc(ctx, i)
	}
	// If custom idempotency period is set on the queue item, use that
	if i.IdempotencyPeriod != nil {
		idempotency = *i.IdempotencyPeriod
	}

	// Enable concurrency state updates by default, disable under some circumstances
	// - processing system queue items
	// - holding a valid capacity lease
	updateConstraintStateVal := "1"
	if o.DisableConstraintUpdates {
		updateConstraintStateVal = "0"
	}

	args, err := StrSlice([]any{
		i.ID,
		partition.PartitionID,
		backlog.BacklogID,
		i.Data.Identifier.AccountID.String(),
		i.Data.Identifier.RunID.String(),

		int(idempotency.Seconds()),

		updateConstraintStateVal,
	})
	if err != nil {
		return err
	}

	status, err := scripts["queue/dequeue"].Exec(
		redis_telemetry.WithScriptName(ctx, "dequeue"),
		q.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error dequeueing item: %w", err)
	}
	switch status {
	case 0:
		if rand.Float64() < 0.05 {
			l.Trace("dequeued item", "job_id", i.ID, "item", i)
		}

		return nil
	case 1:
		return osqueue.ErrQueueItemNotFound
	default:
		return fmt.Errorf("unknown response dequeueing item: %d", status)
	}
}

// Requeue requeues an item in the future.
func (q *queue) Requeue(ctx context.Context, i osqueue.QueueItem, at time.Time, options ...osqueue.RequeueOptionFn) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Requeue"), redis_telemetry.ScopeQueue)

	o := &osqueue.RequeueOptions{}
	for _, opt := range options {
		opt(o)
	}

	l := logger.StdlibLogger(ctx).With("item", i)

	kg := q.RedisClient.kg

	now := q.Clock.Now()
	if at.Before(now) {
		at = now
	}

	// Unset any lease ID as this is requeued.
	i.LeaseID = nil
	// Update the At timestamp.
	// NOTE: This does no priority factorization or FIFO for function ordering,
	// eg. adjusting AtMS based off of function run time.
	i.AtMS = at.UnixMilli()
	// Update the wall time that this should run at.
	i.WallTimeMS = at.UnixMilli()

	// Reset refill details
	i.RefilledFrom = ""
	i.RefilledAt = 0

	// Reset enqueuedAt (used for latency calculation)
	i.EnqueuedAt = now.UnixMilli()

	fnPartition := osqueue.ItemPartition(ctx, i)
	shadowPartition := osqueue.ItemShadowPartition(ctx, i)

	requeueToBacklog := q.ItemEnableKeyQueues(ctx, i)

	requeueToBacklogsVal := "0"
	if requeueToBacklog {
		requeueToBacklogsVal = "1"

		// To avoid requeueing item into a stale backlog, retrieve latest throttle
		if i.Data.Throttle != nil && i.Data.Throttle.KeyExpressionHash == "" {
			refreshedThrottle, err := q.RefreshItemThrottle(ctx, &i)
			if err != nil {
				// If we cannot find the event for the queue item, dequeue it. The state
				// must exist for the entire duration of a function run.
				if errors.Is(err, state.ErrEventNotFound) {
					l.Warn("could not find event for refreshing throttle before requeue")

					err := q.Dequeue(ctx, i)
					if err != nil && !errors.Is(err, osqueue.ErrQueueItemNotFound) {
						return fmt.Errorf("could not dequeue item with missing throttle state: %w", err)
					}

					return nil
				}

				return fmt.Errorf("could not refresh item throttle: %w", err)
			}

			// Update throttle to latest evaluated value + expression hash
			i.Data.Throttle = refreshedThrottle
		}
	}

	backlog := osqueue.ItemBacklog(ctx, i)

	keys := []string{
		kg.QueueItem(),
		kg.PartitionItem(), // Partition item, map
		kg.ConcurrencyIndex(),

		kg.GlobalPartitionIndex(),
		kg.GlobalAccountIndex(),
		kg.AccountPartitionIndex(i.Data.Identifier.AccountID),

		shadowPartitionReadyQueueKey(shadowPartition, kg),

		// In progress (concurrency) keys
		shadowPartitionAccountInProgressKey(shadowPartition, kg),
		shadowPartitionInProgressKey(shadowPartition, kg),
		backlogCustomKeyInProgress(backlog, kg, 1),
		backlogCustomKeyInProgress(backlog, kg, 2),

		// Active set keys
		shadowPartitionAccountActiveKey(shadowPartition, kg),
		shadowPartitionActiveKey(shadowPartition, kg),
		backlogCustomKeyActive(backlog, kg, 1),
		backlogCustomKeyActive(backlog, kg, 2),
		backlogActiveKey(backlog, kg),

		// Active run sets
		kg.RunActiveSet(i.Data.Identifier.RunID),                // Set for active items in run
		shadowPartitionAccountActiveRunKey(shadowPartition, kg), // Set for active runs in account
		shadowPartitionActiveRunKey(shadowPartition, kg),        // Set for active runs in partition
		backlogCustomKeyActiveRuns(backlog, kg, 1),              // Set for active runs with custom concurrency key 1
		backlogCustomKeyActiveRuns(backlog, kg, 2),              // Set for active runs with custom concurrency key 2

		// key queues v2
		kg.BacklogSet(backlog.BacklogID),
		kg.BacklogMeta(),
		kg.GlobalShadowPartitionSet(),
		kg.ShadowPartitionSet(shadowPartition.PartitionID),
		kg.ShadowPartitionMeta(),
		kg.GlobalAccountShadowPartitions(),
		kg.AccountShadowPartitions(i.Data.Identifier.AccountID), // empty for system partitions

		kg.PartitionScavengerIndex(shadowPartition.PartitionID),
	}
	// Append indexes
	for _, idx := range q.itemIndexer(ctx, i, q.RedisClient.kg) {
		if idx != "" {
			keys = append(keys, idx)
		}
	}

	// Enable concurrency state updates by default, disable under some circumstances
	// - processing system queue items
	// - holding a valid capacity lease
	updateConstraintStateVal := "1"
	if o.DisableConstraintUpdates {
		updateConstraintStateVal = "0"
	}

	args, err := StrSlice([]any{
		i.ID,
		i,
		at.UnixMilli(),

		i.Data.Identifier.AccountID.String(),
		i.Data.Identifier.RunID.String(),
		fnPartition.ID,
		fnPartition,

		now.UnixMilli(),

		requeueToBacklogsVal,
		shadowPartition,
		backlog.BacklogID,
		backlog,

		updateConstraintStateVal,
	})
	if err != nil {
		return err
	}

	l.Trace("requeueing queue item",
		"id", i.ID,
		"kind", i.Data.Kind,
		"time", at.Format(time.StampMilli),
		"partition_id", shadowPartition.PartitionID,
		"backlog", requeueToBacklogsVal,
	)

	status, err := scripts["queue/requeue"].Exec(
		redis_telemetry.WithScriptName(ctx, "requeue"),
		q.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		l.Error("error requeueing queue item",
			"error", err,
			"item", i,
			"partition", fnPartition,
			"shadow", shadowPartition,
		)
		return fmt.Errorf("error requeueing item: %w", err)
	}
	switch status {
	case 0:
		switch requeueToBacklogsVal {
		case "1":
			metrics.IncrBacklogRequeuedCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": q.Name,
					// "partition_id": i.FunctionID.String(),
				},
			})
		}

		return nil
	case 1:
		// This should only ever happen if a run is cancelled and all queue items
		// are deleted before requeueing.
		return osqueue.ErrQueueItemNotFound
	default:
		return fmt.Errorf("unknown response requeueing item: %v (%T)", status, status)
	}
}
