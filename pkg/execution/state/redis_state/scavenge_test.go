package redis_state

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueueScavenge(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	shard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	kg := shard.RedisClient.KeyGenerator()

	t.Run("in-progress items must be added to scavenger index", func(t *testing.T) {
		r.FlushAll()

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountID := uuid.New()
		fnID := uuid.New()

		qi := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:  accountID,
					WorkflowID: fnID,
				},
			},
		}

		start := time.Now().Truncate(time.Second)

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		leaseExpiry := clock.Now().Add(5 * time.Second)

		// Lease item in legacy/fallback mode (do not disable lease checks)
		leaseID, err := q.Lease(ctx, item, 5*time.Second, clock.Now(), nil, LeaseOptionDisableConstraintChecks(false))
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		// Check that partition scavenger index + concurrency index are populated
		require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item.ID)))
		require.True(t, r.Exists(kg.ConcurrencyIndex()))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

		// Legacy: Since we did not disable lease checks,
		require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item.ID)))

		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		// Expire lease and expect scores to represent new expiry
		leaseExpiry = clock.Now().Add(5 * time.Second)
		leaseID, err = q.ExtendLease(ctx, item, *leaseID, 5*time.Second)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item.ID)))
		require.True(t, r.Exists(kg.ConcurrencyIndex()))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

		// Legacy: Since we did not disable lease checks,
		require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item.ID)))

		// Dequeue item and check scavenger index was cleaned up
		err = q.Dequeue(ctx, shard, item, DequeueOptionDisableConstraintUpdates(false))
		require.NoError(t, err)

		require.False(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.False(t, r.Exists(kg.ConcurrencyIndex()))
		// Legacy: Since we did not disable lease checks,
		require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
	})

	t.Run("enqueueing multiple items should lead to earliest lease to expire to be pointer score", func(t *testing.T) {
		r.FlushAll()

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountID := uuid.New()
		fnID := uuid.New()

		qi := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:  accountID,
					WorkflowID: fnID,
				},
			},
		}

		start := time.Now().Truncate(time.Second)

		item1, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item2, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		leaseExpiry := clock.Now().Add(5 * time.Second)
		leaseExpiry2 := clock.Now().Add(3 * time.Second)
		require.NotEqual(t, leaseExpiry, leaseExpiry2)

		// Lease item in legacy/fallback mode (do not disable lease checks)
		leaseID1, err := q.Lease(ctx, item1, 5*time.Second, clock.Now(), nil, LeaseOptionDisableConstraintChecks(false))
		require.NoError(t, err)
		require.NotNil(t, leaseID1)

		leaseID2, err := q.Lease(ctx, item2, 3*time.Second, clock.Now(), nil, LeaseOptionDisableConstraintChecks(false))
		require.NoError(t, err)
		require.NotNil(t, leaseID2)

		// Ensure both items are in scavenger index
		require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item1.ID)))
		require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

		// The earliest expiring lease should become the pointer score
		require.True(t, r.Exists(kg.ConcurrencyIndex()))
		require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))
	})

	t.Run("existing items in in-progress sets must be covered by scavenger", func(t *testing.T) {
		r.FlushAll()

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountID := uuid.New()
		fnID := uuid.New()

		qi := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:  accountID,
					WorkflowID: fnID,
				},
			},
		}

		start := time.Now().Truncate(time.Second)

		item1, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Simulate existing lease valid for another second
		leaseExpiry := clock.Now().Add(time.Second)
		_, err = r.ZAdd(kg.Concurrency("p", fnID.String()), float64(leaseExpiry.UnixMilli()), item1.ID)
		require.NoError(t, err)

		_, err = r.ZAdd(kg.ConcurrencyIndex(), float64(leaseExpiry.UnixMilli()), fnID.String())
		require.NoError(t, err)

		// First run should not find any items since lease is still valid

		scavenged, err := q.Scavenge(ctx, 100)
		require.NoError(t, err)
		require.Equal(t, 0, scavenged)

		require.True(t, r.Exists(kg.ConcurrencyIndex()))
		require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))

		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		require.True(t, clock.Now().After(leaseExpiry))

		scavenged, err = q.Scavenge(ctx, 100)
		require.NoError(t, err)
		require.Equal(t, 1, scavenged)

		require.False(t, r.Exists(kg.ConcurrencyIndex()))
		require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
	})

	t.Run("scavenger must clean up expired leases", func(t *testing.T) {
		r.FlushAll()

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountID := uuid.New()
		fnID := uuid.New()

		qi := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:  accountID,
					WorkflowID: fnID,
				},
			},
		}

		start := time.Now().Truncate(time.Second)

		item1, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Simulate existing lease valid for another second
		leaseExpiry := clock.Now().Add(5 * time.Second)
		leaseID, err := q.Lease(ctx, item1, 5*time.Second, clock.Now(), nil, LeaseOptionDisableConstraintChecks(false))
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		// First run should not find any items since lease is still valid

		scavenged, err := q.Scavenge(ctx, 100)
		require.NoError(t, err)
		require.Equal(t, 0, scavenged)

		require.True(t, r.Exists(kg.ConcurrencyIndex()))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item1.ID)))
		require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))

		clock.Advance(6 * time.Second)
		r.FastForward(6 * time.Second)
		r.SetTime(clock.Now())

		require.True(t, clock.Now().After(leaseExpiry))

		scavenged, err = q.Scavenge(ctx, 100)
		require.NoError(t, err)
		require.Equal(t, 1, scavenged)

		require.False(t, r.Exists(kg.ConcurrencyIndex()))
		require.False(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
	})

	// NOTE: This test covers backward compatibility with in-progress items tracked by the queue
	t.Run("scavenger must clean up expired leases from in-progress sets", func(t *testing.T) {
	})

	t.Run("scavenging removes leftover traces of key queues", func(t *testing.T) {
		r.FlushAll()

		q := NewQueue(
			shard,
			WithClock(clock),
		)
		ctx := context.Background()

		id := uuid.New()

		qi := osqueue.QueueItem{
			FunctionID: id,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
			},
		}

		start := clock.Now().Truncate(time.Second)

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.Zero)
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		qp := getDefaultPartition(t, r, id)

		leaseStart := clock.Now()
		leaseExpires := q.clock.Now().Add(time.Second)

		itemCountMatches := func(num int) {
			zsetKey := qp.zsetKey(q.primaryQueueShard.RedisClient.kg)
			items, err := rc.Do(ctx, rc.B().
				Zrangebyscore().
				Key(zsetKey).
				Min("-inf").
				Max("+inf").
				Build()).AsStrSlice()
			require.NoError(t, err)
			assert.Equal(t, num, len(items), "expected %d items in the queue %q", num, zsetKey, r.Dump())
		}

		concurrencyItemCountMatches := func(num int) {
			items, err := rc.Do(ctx, rc.B().
				Zrangebyscore().
				Key(qp.concurrencyKey(q.primaryQueueShard.RedisClient.kg)).
				Min("-inf").
				Max("+inf").
				Build()).AsStrSlice()
			require.NoError(t, err)
			assert.Equal(t, num, len(items), "expected %d items in the concurrency queue", num, r.Dump())
		}

		itemCountMatches(1)
		concurrencyItemCountMatches(0)

		leaseId, err := q.Lease(ctx, item, time.Second, leaseStart, nil)
		require.NoError(t, err)
		require.NotNil(t, leaseId)

		itemCountMatches(0)
		concurrencyItemCountMatches(1)

		// wait til leases are expired
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())
		require.True(t, clock.Now().After(leaseExpires))

		incompatibleConcurrencyIndexItem := q.primaryQueueShard.RedisClient.kg.Concurrency("p", id.String())
		compatibleConcurrencyIndexItem := id.String()

		indexMembers, err := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(indexMembers))
		require.Contains(t, indexMembers, compatibleConcurrencyIndexItem)

		leftoverData := []string{
			q.primaryQueueShard.RedisClient.kg.Concurrency("p", id.String()),
			"{queue}:concurrency:p:0ffd4629-317c-4f65-8b8f-b30fccfde46f",
			"{queue}:concurrency:custom:f:0ffd4629-317c-4f65-8b8f-b30fccfde46f:1nt4mu0skse4a",
		}
		score := float64(leaseStart.Add(time.Second).UnixMilli())
		for _, leftover := range leftoverData {
			_, err = r.ZAdd(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex(), score, leftover)
			require.NoError(t, err)
		}
		indexMembers, err = r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
		require.NoError(t, err)
		require.Equal(t, 4, len(indexMembers))
		for _, datum := range leftoverData {
			require.Contains(t, indexMembers, datum)
		}

		requeued, err := q.Scavenge(ctx, ScavengePeekSize)
		require.NoError(t, err)
		assert.Equal(t, 1, requeued, "expected one item with expired leases to be requeued by scavenge", r.Dump())

		itemCountMatches(1)
		concurrencyItemCountMatches(0)

		_, err = r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
		require.Error(t, err, r.Dump())
		require.ErrorIs(t, err, miniredis.ErrKeyNotFound)

		newConcurrencyQueueItems, err := rc.Do(ctx, rc.B().Zcard().Key(incompatibleConcurrencyIndexItem).Build()).AsInt64()
		require.NoError(t, err)
		assert.Equal(t, 0, int(newConcurrencyQueueItems), "expected no items in the new concurrency queue", r.Dump())

		oldConcurrencyQueueItems, err := rc.Do(ctx, rc.B().Zcard().Key(compatibleConcurrencyIndexItem).Build()).AsInt64()
		require.NoError(t, err)
		assert.Equal(t, 0, int(oldConcurrencyQueueItems), "expected no items in the old concurrency queue", r.Dump())
	})
}
