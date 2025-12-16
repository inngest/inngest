package redis_state

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
)

type QueueOpOpt func(o *queueOpOpt)

type queueOpOpt struct {
	shard *QueueShard
}

// newQueueOpOpt returns the default option settings for queue operations
func newQueueOpOpt() queueOpOpt {
	return queueOpOpt{}
}

func newQueueOpOptWithOpts(opts ...QueueOpOpt) queueOpOpt {
	opt := newQueueOpOpt()
	for _, apply := range opts {
		apply(&opt)
	}

	return opt
}

func WithQueueOpShard(shard QueueShard) QueueOpOpt {
	return func(o *queueOpOpt) {
		o.shard = &shard
	}
}

func (q *queue) DequeueByJobID(ctx context.Context, jobID string, opts ...QueueOpOpt) error {
	opt := newQueueOpOpt()
	for _, apply := range opts {
		apply(&opt)
	}

	item, err := q.ItemByID(ctx, jobID, opts...)
	switch err {
	case nil:
		// no-op
	case ErrQueueItemNotFound:
		return nil
	default:
		return fmt.Errorf("error retrieving item by ID: %w", err)
	}

	shard := q.primaryQueueShard
	if opt.shard != nil {
		shard = *opt.shard
	}

	return q.Dequeue(ctx, shard, *item)
}

type dequeueOptions struct {
	disableConstraintUpdates bool
}

func DequeueOptionDisableConstraintUpdates(disableUpdates bool) dequeueOptionFn {
	return func(o *dequeueOptions) {
		o.disableConstraintUpdates = disableUpdates
	}
}

type dequeueOptionFn func(o *dequeueOptions)

// Dequeue removes an item from the queue entirely.
func (q *queue) Dequeue(ctx context.Context, queueShard QueueShard, i osqueue.QueueItem, options ...dequeueOptionFn) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Dequeue"), redis_telemetry.ScopeQueue)

	o := &dequeueOptions{}
	for _, opt := range options {
		opt(o)
	}

	if queueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for Dequeue: %s", queueShard.Kind)
	}

	kg := queueShard.RedisClient.kg

	partition := q.ItemShadowPartition(ctx, i)
	backlog := q.ItemBacklog(ctx, i)

	keys := []string{
		kg.QueueItem(),
		kg.PartitionItem(),

		kg.ConcurrencyIndex(),

		partition.readyQueueKey(kg),
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
		partition.accountInProgressKey(kg),
		partition.inProgressKey(kg),
		backlog.customKeyInProgress(kg, 1),
		backlog.customKeyInProgress(kg, 2),

		// Active set keys
		partition.accountActiveKey(kg),
		partition.activeKey(kg),
		backlog.customKeyActive(kg, 1),
		backlog.customKeyActive(kg, 2),
		backlog.activeKey(kg),

		// Active run sets
		kg.RunActiveSet(i.Data.Identifier.RunID), // Set for active items in run
		partition.accountActiveRunKey(kg),        // Set for active runs in account
		partition.activeRunKey(kg),               // Set for active runs in partition
		backlog.customKeyActiveRuns(kg, 1),       // Set for active runs with custom concurrency key 1
		backlog.customKeyActiveRuns(kg, 2),       // Set for active runs with custom concurrency key 2

		kg.Idempotency(i.ID),

		// Singleton
		kg.SingletonRunKey(i.Data.Identifier.RunID.String()),

		kg.PartitionScavengerIndex(partition.PartitionID),
	}

	// Append indexes
	for _, idx := range q.itemIndexer(ctx, i, queueShard.RedisClient.kg) {
		if idx != "" {
			keys = append(keys, idx)
		}
	}

	idempotency := q.idempotencyTTL
	if q.idempotencyTTLFunc != nil {
		idempotency = q.idempotencyTTLFunc(ctx, i)
	}
	// If custom idempotency period is set on the queue item, use that
	if i.IdempotencyPeriod != nil {
		idempotency = *i.IdempotencyPeriod
	}

	// Enable concurrency state updates by default, disable under some circumstances
	// - processing system queue items
	// - holding a valid capacity lease
	updateConstraintStateVal := "1"
	if o.disableConstraintUpdates {
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
		queueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error dequeueing item: %w", err)
	}
	switch status {
	case 0:
		if rand.Float64() < 0.05 {
			q.log.Debug("dequeued item", "job_id", i.ID, "item", i)
		}

		return nil
	case 1:
		return ErrQueueItemNotFound
	default:
		return fmt.Errorf("unknown response dequeueing item: %d", status)
	}
}

type requeueOptions struct {
	disableConstraintUpdates bool
}

func RequeueOptionDisableConstraintUpdates(disableUpdates bool) requeueOptionFn {
	return func(o *requeueOptions) {
		o.disableConstraintUpdates = disableUpdates
	}
}

type requeueOptionFn func(o *requeueOptions)

// Requeue requeues an item in the future.
func (q *queue) Requeue(ctx context.Context, queueShard QueueShard, i osqueue.QueueItem, at time.Time, options ...requeueOptionFn) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Requeue"), redis_telemetry.ScopeQueue)

	o := &requeueOptions{}
	for _, opt := range options {
		opt(o)
	}

	l := q.log.With("item", i)

	if queueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for Requeue: %s", queueShard.Kind)
	}

	kg := queueShard.RedisClient.kg

	now := q.clock.Now()
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

	fnPartition := q.ItemPartition(ctx, queueShard, i)
	shadowPartition := q.ItemShadowPartition(ctx, i)

	requeueToBacklog := q.itemEnableKeyQueues(ctx, i)

	requeueToBacklogsVal := "0"
	if requeueToBacklog {
		requeueToBacklogsVal = "1"

		// To avoid requeueing item into a stale backlog, retrieve latest throttle
		if i.Data.Throttle != nil && i.Data.Throttle.KeyExpressionHash == "" {
			refreshedThrottle, err := q.refreshItemThrottle(ctx, &i)
			if err != nil {
				// If we cannot find the event for the queue item, dequeue it. The state
				// must exist for the entire duration of a function run.
				if errors.Is(err, state.ErrEventNotFound) {
					l.Warn("could not find event for refreshing throttle before requeue")

					err := q.Dequeue(ctx, queueShard, i)
					if err != nil && !errors.Is(err, ErrQueueItemNotFound) {
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

	backlog := q.ItemBacklog(ctx, i)

	keys := []string{
		kg.QueueItem(),
		kg.PartitionItem(), // Partition item, map
		kg.ConcurrencyIndex(),

		kg.GlobalPartitionIndex(),
		kg.GlobalAccountIndex(),
		kg.AccountPartitionIndex(i.Data.Identifier.AccountID),

		shadowPartition.readyQueueKey(kg),

		// In progress (concurrency) keys
		shadowPartition.accountInProgressKey(kg),
		shadowPartition.inProgressKey(kg),
		backlog.customKeyInProgress(kg, 1),
		backlog.customKeyInProgress(kg, 2),

		// Active set keys
		shadowPartition.accountActiveKey(kg),
		shadowPartition.activeKey(kg),
		backlog.customKeyActive(kg, 1),
		backlog.customKeyActive(kg, 2),
		backlog.activeKey(kg),

		// Active run sets
		kg.RunActiveSet(i.Data.Identifier.RunID), // Set for active items in run
		shadowPartition.accountActiveRunKey(kg),  // Set for active runs in account
		shadowPartition.activeRunKey(kg),         // Set for active runs in partition
		backlog.customKeyActiveRuns(kg, 1),       // Set for active runs with custom concurrency key 1
		backlog.customKeyActiveRuns(kg, 2),       // Set for active runs with custom concurrency key 2

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
	for _, idx := range q.itemIndexer(ctx, i, queueShard.RedisClient.kg) {
		if idx != "" {
			keys = append(keys, idx)
		}
	}

	// Enable concurrency state updates by default, disable under some circumstances
	// - processing system queue items
	// - holding a valid capacity lease
	updateConstraintStateVal := "1"
	if o.disableConstraintUpdates {
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

	q.log.Trace("requeueing queue item",
		"id", i.ID,
		"kind", i.Data.Kind,
		"time", at.Format(time.StampMilli),
		"partition_id", shadowPartition.PartitionID,
		"backlog", requeueToBacklogsVal,
	)

	status, err := scripts["queue/requeue"].Exec(
		redis_telemetry.WithScriptName(ctx, "requeue"),
		queueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		q.log.Error("error requeueing queue item",
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
					"queue_shard": q.primaryQueueShard.Name,
					// "partition_id": i.FunctionID.String(),
				},
			})
		}

		return nil
	case 1:
		// This should only ever happen if a run is cancelled and all queue items
		// are deleted before requeueing.
		return ErrQueueItemNotFound
	default:
		return fmt.Errorf("unknown response requeueing item: %v (%T)", status, status)
	}
}
