package redis_state

import (
	"context"
	"fmt"
	"time"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/oklog/ulid/v2"
)

// Implements ShardLease() in ShardOperations interface
//
// ShardLease allows a worker to lease config keys to process the shard
// Leasing this key works similar to leasing partitions or queue items:
//
//   - If the key has fewer than maxLeases granted, a new lease is accepted.
//   - If some of the leases in the key are expired, a new lease is granted to replenish those expired leases
//   - If an existing lease ID is provided, it is renewed if it is currently unexpired.
//
// This returns the new lease ID on success.
func (q *queue) ShardLease(ctx context.Context, key string, duration time.Duration, maxLeases int, existingLeaseID ...*ulid.ULID) (*ulid.ULID, error) {
	if duration > osqueue.ShardLeaseMax {
		return nil, osqueue.ErrShardLeaseExceedsLimits
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ShardLease"), redis_telemetry.ScopeQueue)

	now := q.Clock.Now()
	newLeaseID, err := ulid.New(ulid.Timestamp(now.Add(duration)), rnd)
	if err != nil {
		return nil, err
	}

	var existing string
	if len(existingLeaseID) > 0 && existingLeaseID[0] != nil {
		existing = existingLeaseID[0].String()
	}

	args, err := StrSlice([]any{
		now.UnixMilli(),
		newLeaseID.String(),
		existing,
		maxLeases,
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["queue/shardLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "shardLease"),
		q.RedisClient.unshardedRc,
		[]string{
			q.RedisClient.kg.ShardLeaseKey(key),
		},
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error claiming shard lease: %w", err)
	}
	switch status {
	case 0:
		return &newLeaseID, nil
	case 1:
		return nil, osqueue.ErrShardLeaseNotFound
	case 2:
		return nil, osqueue.ErrShardLeaseExpired
	case 3:
		return nil, osqueue.ErrAllShardsAlreadyLeased
	default:
		return nil, fmt.Errorf("unknown response claiming shard lease: %d", status)
	}
}
