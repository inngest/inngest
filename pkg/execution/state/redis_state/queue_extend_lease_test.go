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
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestQueueExtendLease(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	queueClient := RedisQueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	q := NewQueue(queueClient)
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)
	t.Run("It leases an item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := q.ItemPartition(ctx, q.primaryQueueShard, item)

		now := time.Now()
		id, err := q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		now = time.Now()
		nextID, err := q.ExtendLease(ctx, item, *id, 10*time.Second)
		require.NoError(t, err)

		require.False(t, r.Exists(QueuePartition{}.concurrencyKey(q.primaryQueueShard.RedisClient.kg)))

		// Ensure the leased item has the next ID.
		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, nextID, item.LeaseID)
		require.WithinDuration(t, now.Add(10*time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It extends the score of the partition concurrency queue", func(t *testing.T) {
			at := ulid.Time(nextID.Time())
			scores := concurrencyQueueScores(t, r, p.concurrencyKey(q.primaryQueueShard.RedisClient.kg), time.Now())
			require.Len(t, scores, 1)
			// Ensure that the score matches the lease.
			require.Equal(t, at, scores[item.ID], "%s not extended\n%s", p.concurrencyKey(q.primaryQueueShard.RedisClient.kg), r.Dump())
		})

		t.Run("It fails with an invalid lease ID", func(t *testing.T) {
			invalid := ulid.MustNew(ulid.Now(), rnd)
			nextID, err := q.ExtendLease(ctx, item, invalid, 10*time.Second)
			require.EqualValues(t, ErrQueueItemLeaseMismatch, err)
			require.Nil(t, nextID)
		})
	})

	t.Run("It does not extend an unleased item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		nextID, err := q.ExtendLease(ctx, item, ulid.Zero, 10*time.Second)
		require.EqualValues(t, ErrQueueItemNotLeased, err)
		require.Nil(t, nextID)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)
	})

	t.Run("With custom keys in multiple partitions", func(t *testing.T) {
		r.FlushAll()

		ckA := state.CustomConcurrency{
			Key: util.ConcurrencyKey(
				enums.ConcurrencyScopeAccount,
				uuid.Nil,
				"acct-id",
			),
			Limit: 10,
		}
		ckB := state.CustomConcurrency{
			Key: util.ConcurrencyKey(
				enums.ConcurrencyScopeFn,
				uuid.Nil,
				"fn-id",
			),
			Limit: 5,
		}

		q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
			return PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					AccountConcurrency:  123,
					FunctionConcurrency: 45,
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Scope:               enums.ConcurrencyScopeAccount,
							HashedKeyExpression: ckA.Hash,
							Limit:               ckA.Limit,
						},
						{
							Scope:               enums.ConcurrencyScopeFn,
							HashedKeyExpression: ckB.Hash,
							Limit:               ckB.Limit,
						},
					},
				},
			}
		}

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: uuid.New(),
			Data: osqueue.Item{
				CustomConcurrencyKeys: []state.CustomConcurrency{
					ckA,
					ckB,
				},
			},
		}, start, osqueue.EnqueueOpts{})
		require.Nil(t, err)

		// First 2 partitions will be custom.
		fnPart, custom1, custom2 := q.ItemPartitions(ctx, q.primaryQueueShard, item)
		require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
		require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
		require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom2.PartitionType)

		// Lease the item.
		id, err := q.Lease(ctx, item, time.Second, q.clock.Now(), nil)
		require.NoError(t, err)
		require.NotNil(t, id)

		score0, err := r.ZMScore(fnPart.concurrencyKey(q.primaryQueueShard.RedisClient.kg), item.ID)
		require.NoError(t, err)
		score1, err := r.ZMScore(custom1.concurrencyKey(q.primaryQueueShard.RedisClient.kg), item.ID)
		require.NoError(t, err)
		require.Equal(t, score0[0], score1[0], "Partition scores should match after leasing")

		t.Run("extending the lease should extend both items in all partition's concurrency queues", func(t *testing.T) {
			id, err = q.ExtendLease(ctx, item, *id, 98712*time.Millisecond)
			require.NoError(t, err)
			require.NotNil(t, id)

			newScore0, err := r.ZMScore(fnPart.concurrencyKey(q.primaryQueueShard.RedisClient.kg), item.ID)
			require.NoError(t, err)
			newScore1, err := r.ZMScore(custom1.concurrencyKey(q.primaryQueueShard.RedisClient.kg), item.ID)
			require.NoError(t, err)

			require.Equal(t, newScore0, newScore1, "Partition scores should match after leasing")
			require.NotEqual(t, int(score0[0]), int(newScore0[0]), "Partition scores should not have been updated: %v", newScore0)
			require.NotEqual(t, score1, newScore1, "Partition scores should have been updated")

			// And, the account-level concurrency queue is updated
			acctScore, err := r.ZMScore(q.primaryQueueShard.RedisClient.kg.Concurrency("account", item.Data.Identifier.AccountID.String()), item.ID)
			require.NoError(t, err)
			require.EqualValues(t, acctScore[0], newScore0[0])
		})

		t.Run("Scavenge queue is updated", func(t *testing.T) {
			mem, err := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
			require.NoError(t, err)
			require.Equal(t, 1, len(mem), "scavenge queue should have 1 item", mem)
			require.Contains(t, mem, fnPart.ID)
			require.NotContains(t, mem, custom1.concurrencyKey(q.primaryQueueShard.RedisClient.kg))
			require.NotContains(t, mem, custom2.concurrencyKey(q.primaryQueueShard.RedisClient.kg))

			score, err := r.ZMScore(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex(), fnPart.ID)
			require.NoError(t, err)
			require.NotZero(t, score[0])

			id, err = q.ExtendLease(ctx, item, *id, 1238712*time.Millisecond)
			require.NoError(t, err)
			require.NotNil(t, id)

			nextScore, err := r.ZMScore(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex(), fnPart.ID)
			require.NoError(t, err)

			require.NotEqual(t, score[0], nextScore[0])
		})
	})
}

func TestQueueExtendLeaseWithDisabledConstraintUpdates(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	shard := RedisQueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	kg := shard.RedisClient.KeyGenerator()

	q := NewQueue(
		shard,
		WithClock(clock),
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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

	// Lease item in new mode (skip checks)
	leaseID, err := q.Lease(ctx, item, 5*time.Second, clock.Now(), nil, LeaseOptionDisableConstraintChecks(true))
	require.NoError(t, err)
	require.NotNil(t, leaseID)

	require.Equal(t, clock.Now().Add(5*time.Second).UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item.ID)))
	require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
	require.False(t, r.Exists(kg.Concurrency("account", accountID.String())))

	newLeaseID, err := q.ExtendLease(ctx, item, *leaseID, 10*time.Second, ExtendLeaseOptionDisableConstraintUpdates(true))
	require.NoError(t, err)
	require.NotNil(t, newLeaseID)
	require.NotEqual(t, *leaseID, *newLeaseID)

	require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
	require.False(t, r.Exists(kg.Concurrency("account", accountID.String())))
	require.Equal(t, clock.Now().Add(10*time.Second).UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item.ID)))
}
