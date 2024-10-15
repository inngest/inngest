package redis_state

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/oklog/ulid/v2"
	"time"
)

type redisEnqueuer struct {
	u *QueueClient
	q *queue
}

func (r redisEnqueuer) EnqueueItem(ctx context.Context, i osqueue.QueueItem, at time.Time) (osqueue.QueueItem, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "EnqueueItem"), redis_telemetry.ScopeQueue)

	if len(i.ID) == 0 {
		i.SetID(ctx, ulid.MustNew(ulid.Now(), rnd).String())
	} else {
		i.ID = osqueue.HashID(ctx, i.ID)
	}

	// XXX: If the length of ID >= max, error.
	if i.WallTimeMS == 0 {
		i.WallTimeMS = at.UnixMilli()
	}

	if at.Before(r.q.clock.Now()) {
		// Normalize to now to minimize latency.
		i.WallTimeMS = r.q.clock.Now().UnixMilli()
	}

	// Add the At timestamp, if not included.
	if i.AtMS == 0 {
		i.AtMS = at.UnixMilli()
	}

	if i.Data.JobID == nil {
		i.Data.JobID = &i.ID
	}

	partitionTime := at
	if at.Before(r.q.clock.Now()) {
		// We don't want to enqueue partitions (pointers to fns) before now.
		// Doing so allows users to stay at the front of the queue for
		// leases.
		partitionTime = r.q.clock.Now()
	}

	parts, _ := r.q.ItemPartitions(ctx, i)
	isSystemPartition := parts[0].IsSystem()

	if i.Data.Identifier.AccountID == uuid.Nil && !isSystemPartition {
		r.q.logger.Warn().Interface("item", i).Msg("attempting to enqueue item to non-system partition without account ID")
	}

	var (
		guaranteedCapacity *GuaranteedCapacity

		// initialize guaranteed capacity key for automatic cleanup
		guaranteedCapacityKey = guaranteedCapacityKeyForAccount(i.Data.Identifier.AccountID)
	)
	if r.q.gcf != nil && !isSystemPartition {
		// Fetch guaranteed capacity for the given account. If there is no guaranteed
		// capacity configured, this will return nil, and we will remove any leftover
		// items in the guaranteed capacity map
		// Note: This function is called _a lot_ so the calls should be memoized.
		guaranteedCapacity = r.q.gcf(ctx, r.q.queueShardName, i.Data.Identifier.AccountID)
		if guaranteedCapacity != nil {
			guaranteedCapacity.Leases = []ulid.ULID{}
			guaranteedCapacityKey = guaranteedCapacity.Key()
		}
	}

	keys := []string{
		r.u.kg.QueueItem(),            // Queue item
		r.u.kg.PartitionItem(),        // Partition item, map
		r.u.kg.GlobalPartitionIndex(), // Global partition queue
		r.u.kg.GlobalAccountIndex(),
		r.u.kg.AccountPartitionIndex(i.Data.Identifier.AccountID), // new queue items always contain the account ID
		r.u.kg.Idempotency(i.ID),
		r.u.kg.FnMetadata(i.FunctionID),
		r.u.kg.GuaranteedCapacityMap(),

		// Add all 3 partition sets
		parts[0].zsetKey(r.u.kg),
		parts[1].zsetKey(r.u.kg),
		parts[2].zsetKey(r.u.kg),
	}
	// Append indexes
	for _, idx := range r.q.itemIndexer(ctx, i, r.u.kg) {
		if idx != "" {
			keys = append(keys, idx)
		}
	}

	args, err := StrSlice([]any{
		i,
		i.ID,
		at.UnixMilli(),
		partitionTime.Unix(),
		r.q.clock.Now().UnixMilli(),
		FnMetadata{
			// enqueue.lua only writes function metadata if it doesn't already exist.
			// if it doesn't exist, and we're enqueuing something, this implies the fn is not currently paused.
			FnID:   i.FunctionID,
			Paused: false,
		},
		parts[0],
		parts[1],
		parts[2],

		parts[0].ID,
		parts[1].ID,
		parts[2].ID,

		parts[0].PartitionType,
		parts[1].PartitionType,
		parts[2].PartitionType,

		i.Data.Identifier.AccountID.String(),

		guaranteedCapacity,
		guaranteedCapacityKey,
	})
	if err != nil {
		return i, err
	}

	r.q.logger.Trace().Interface("item", i).Interface("parts", parts).Interface("keys", keys).Interface("args", args).Msg("enqueueing item")

	status, err := scripts["queue/enqueue"].Exec(
		redis_telemetry.WithScriptName(ctx, "enqueue"),
		r.u.Client(),
		keys,
		args,
	).AsInt64()
	if err != nil {
		return i, fmt.Errorf("error enqueueing item: %w", err)
	}
	switch status {
	case 0:
		return i, nil
	case 1:
		return i, ErrQueueItemExists
	default:
		return i, fmt.Errorf("unknown response enqueueing item: %v (%T)", status, status)
	}
}

func NewRedisEnqueuer(q *queue, u *QueueClient) osqueue.Enqueuer {
	return &redisEnqueuer{
		u: u,
		q: q,
	}
}
