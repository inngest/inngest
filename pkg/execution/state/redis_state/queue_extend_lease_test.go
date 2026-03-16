package redis_state

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
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

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	_, shard := newQueue(t, rc,
		osqueue.WithClock(clock),
	)
	kg := shard.Client().kg

	start := clock.Now().Truncate(time.Second)
	t.Run("It leases an item", func(t *testing.T) {
		item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := osqueue.ItemPartition(ctx, item)

		now := clock.Now()
		id, err := shard.Lease(ctx, item, time.Second, clock.Now(), nil)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		now = clock.Now()
		nextID, err := shard.ExtendLease(ctx, item, *id, 10*time.Second)
		require.NoError(t, err)

		require.False(t, r.Exists(partitionConcurrencyKey(osqueue.QueuePartition{}, kg)))

		// Ensure the leased item has the next ID.
		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, nextID, item.LeaseID)
		require.WithinDuration(t, now.Add(10*time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It extends the score of the partition concurrency queue", func(t *testing.T) {
			at := ulid.Time(nextID.Time())
			scores := concurrencyQueueScores(t, r, partitionConcurrencyKey(p, kg), clock.Now())
			require.Len(t, scores, 1)
			// Ensure that the score matches the lease.
			require.Equal(t, at, scores[item.ID], "%s not extended\n%s", partitionConcurrencyKey(p, kg), r.Dump())
		})

		t.Run("It fails with an invalid lease ID", func(t *testing.T) {
			invalid := ulid.MustNew(ulid.Now(), rnd)
			nextID, err := shard.ExtendLease(ctx, item, invalid, 10*time.Second)
			require.EqualValues(t, osqueue.ErrQueueItemLeaseMismatch, err)
			require.Nil(t, nextID)
		})
	})

	t.Run("It does not extend an unleased item", func(t *testing.T) {
		item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		nextID, err := shard.ExtendLease(ctx, item, ulid.Zero, 10*time.Second)
		require.EqualValues(t, osqueue.ErrQueueItemNotLeased, err)
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

		_, shard := newQueue(t, rc,
			osqueue.WithClock(clock),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  123,
						FunctionConcurrency: 45,
						CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
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
			}),
		)

		item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
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
		fnPart := osqueue.ItemPartition(ctx, item)
		require.NotEmpty(t, fnPart.ID)

		// Lease the item.
		id, err := shard.Lease(ctx, item, time.Second, clock.Now(), nil)
		require.NoError(t, err)
		require.NotNil(t, id)

		backlog := osqueue.ItemBacklog(ctx, item)

		score0, err := r.ZMScore(partitionConcurrencyKey(fnPart, kg), item.ID)
		require.NoError(t, err)
		score1, err := r.ZMScore(backlogCustomKeyInProgress(backlog, kg, 1), item.ID)
		require.NoError(t, err)
		require.Equal(t, score0[0], score1[0], "Partition scores should match after leasing")

		t.Run("extending the lease should extend both items in all partition's concurrency queues", func(t *testing.T) {
			id, err = shard.ExtendLease(ctx, item, *id, 98712*time.Millisecond)
			require.NoError(t, err)
			require.NotNil(t, id)

			newScore0, err := r.ZMScore(partitionConcurrencyKey(fnPart, kg), item.ID)
			require.NoError(t, err)
			newScore1, err := r.ZMScore(backlogCustomKeyInProgress(backlog, kg, 1), item.ID)
			require.NoError(t, err)

			require.Equal(t, newScore0, newScore1, "Partition scores should match after leasing")
			require.NotEqual(t, int(score0[0]), int(newScore0[0]), "Partition scores should not have been updated: %v", newScore0)
			require.NotEqual(t, score1, newScore1, "Partition scores should have been updated")

			// And, the account-level concurrency queue is updated
			acctScore, err := r.ZMScore(kg.Concurrency("account", item.Data.Identifier.AccountID.String()), item.ID)
			require.NoError(t, err)
			require.EqualValues(t, acctScore[0], newScore0[0])
		})

		t.Run("Scavenge queue is updated", func(t *testing.T) {
			mem, err := r.ZMembers(kg.ConcurrencyIndex())
			require.NoError(t, err)
			require.Equal(t, 1, len(mem), "scavenge queue should have 1 item", mem)
			require.Contains(t, mem, fnPart.ID)
			require.NotContains(t, mem, backlogCustomKeyInProgress(backlog, kg, 1))
			require.NotContains(t, mem, backlogCustomKeyInProgress(backlog, kg, 2))

			score, err := r.ZMScore(kg.ConcurrencyIndex(), fnPart.ID)
			require.NoError(t, err)
			require.NotZero(t, score[0])

			id, err = shard.ExtendLease(ctx, item, *id, 1238712*time.Millisecond)
			require.NoError(t, err)
			require.NotNil(t, id)

			nextScore, err := r.ZMScore(kg.ConcurrencyIndex(), fnPart.ID)
			require.NoError(t, err)

			require.NotEqual(t, score[0], nextScore[0])
		})
	})
}

