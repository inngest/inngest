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
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestQueuePartitionLease(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	idA, idB, idC := uuid.New(), uuid.New(), uuid.New()
	atA, atB, atC := now, now.Add(time.Second), now.Add(2*time.Second)

	pA := osqueue.QueuePartition{ID: idA.String(), FunctionID: &idA}

	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	_, shard := newQueue(t, rc)
	ctx := context.Background()

	_, err = shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: idA}, atA, osqueue.EnqueueOpts{})
	require.NoError(t, err)
	_, err = shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: idB}, atB, osqueue.EnqueueOpts{})
	require.NoError(t, err)
	_, err = shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: idC}, atC, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	t.Run("Partitions are in order after enqueueing", func(t *testing.T) {
		items, err := shard.PartitionPeek(ctx, true, time.Now().Add(time.Hour), osqueue.PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 3)
		require.EqualValues(t, []*osqueue.QueuePartition{
			{ID: idA.String(), FunctionID: &idA, AccountID: uuid.Nil},
			{ID: idB.String(), FunctionID: &idB, AccountID: uuid.Nil},
			{ID: idC.String(), FunctionID: &idC, AccountID: uuid.Nil},
		}, items)
	})

	leaseUntil := now.Add(3 * time.Second)

	t.Run("It leases a partition", func(t *testing.T) {
		// Lease the first item now.
		leasedAt := time.Now()
		leaseID, capacity, err := shard.PartitionLease(ctx, &pA, time.Until(leaseUntil))
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.NotZero(t, capacity)

		// Pause so that we can assert that the last lease time was set correctly.
		<-time.After(50 * time.Millisecond)

		t.Run("It updates the partition score", func(t *testing.T) {
			items, err := shard.PartitionPeek(ctx, true, now.Add(time.Hour), osqueue.PartitionPeekMax)

			// Require the lease ID is within 25 MS of the expected value.
			require.WithinDuration(t, leaseUntil, ulid.Time(leaseID.Time()), 25*time.Millisecond)

			require.NoError(t, err)
			require.Len(t, items, 3)
			require.EqualValues(t, []*osqueue.QueuePartition{
				{ID: idB.String(), FunctionID: &idB, AccountID: uuid.Nil},
				{ID: idC.String(), FunctionID: &idC, AccountID: uuid.Nil},
				{
					ID:         idA.String(),
					FunctionID: &idA,
					AccountID:  uuid.Nil,
					Last:       items[2].Last, // Use the leased partition time.
					LeaseID:    leaseID,
				}, // idA is now last.
			}, items)
			requirePartitionScoreEquals(t, r, &idA, leaseUntil)
			// require that the last leased time is within 5ms for tests
			require.WithinDuration(t, leasedAt, time.UnixMilli(items[2].Last), 5*time.Millisecond)
		})

		t.Run("It can't lease an existing partition lease", func(t *testing.T) {
			id, capacity, err := shard.PartitionLease(ctx, &pA, time.Second*29)
			require.Equal(t, osqueue.ErrPartitionAlreadyLeased, err)
			require.Nil(t, id)
			require.Zero(t, capacity)

			// Assert that score didn't change (we added 1 second in the previous test)
			requirePartitionScoreEquals(t, r, &idA, leaseUntil)
		})
	})

	t.Run("It allows leasing an expired partition lease", func(t *testing.T) {
		<-time.After(time.Until(leaseUntil))

		requirePartitionScoreEquals(t, r, &idA, leaseUntil)

		id, capacity, err := shard.PartitionLease(ctx, &pA, time.Second*5)
		require.Nil(t, err)
		require.NotNil(t, id)
		require.NotZero(t, capacity)

		requirePartitionScoreEquals(t, r, &idA, time.Now().Add(time.Second*5))
	})

	t.Run("With key partitions", func(t *testing.T) {
		fnID := uuid.New()

		// Enqueueing an item
		ck := createConcurrencyKey(enums.ConcurrencyScopeFn, fnID, "test", 1)

		_, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				CustomConcurrencyKeys: []state.CustomConcurrency{ck},
			},
		}, now.Add(10*time.Second), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		defaultPartition := getDefaultPartition(t, r, fnID)

		leaseUntil := now.Add(3 * time.Second)
		leaseID, capacity, err := shard.PartitionLease(ctx, &defaultPartition, time.Until(leaseUntil))
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.NotZero(t, capacity)
	})

	t.Run("concurrency is checked early", func(t *testing.T) {
		start := time.Now().Truncate(time.Second)

		t.Run("With partition concurrency limits", func(t *testing.T) {
			r.FlushAll()

			_, shard := newQueue(t, rc,
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					// Only allow a single leased item
					return osqueue.PartitionConstraintConfig{
						Concurrency: osqueue.PartitionConcurrency{
							AccountConcurrency:  1,
							FunctionConcurrency: 1,
						},
					}
				}))

			fnID := uuid.New()
			// Create a new item
			itemA, err := shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: fnID}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			_, err = shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: fnID}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			// Use the new item's workflow ID
			p := osqueue.QueuePartition{ID: itemA.FunctionID.String(), FunctionID: &itemA.FunctionID}

			t.Run("Leases with capacity", func(t *testing.T) {
				_, err = shard.Lease(ctx, itemA, 5*time.Second, time.Now(), nil)
				require.NoError(t, err)
			})

			t.Run("Partition lease errors without capacity", func(t *testing.T) {
				leaseId, _, err := shard.PartitionLease(ctx, &p, 5*time.Second)
				require.Nil(t, leaseId, "No lease id when leasing fails.\n%s", r.Dump())
				require.Error(t, err)
				require.ErrorIs(t, err, osqueue.ErrPartitionConcurrencyLimit)
			})
		})

		t.Run("With account concurrency limits", func(t *testing.T) {
			r.FlushAll()

			_, shard := newQueue(t, rc,
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					// Only allow a single leased item via account limits
					return osqueue.PartitionConstraintConfig{
						Concurrency: osqueue.PartitionConcurrency{
							AccountConcurrency:  1,
							FunctionConcurrency: 100,
						},
					}
				}))

			acctId := uuid.New()

			// Create a new item
			itemA, err := shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: uuid.New(), Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			_, err = shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: uuid.New(), Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			// Use the new item's workflow ID
			p := osqueue.QueuePartition{AccountID: acctId, FunctionID: &itemA.FunctionID}

			t.Run("Leases with capacity", func(t *testing.T) {
				_, err = shard.Lease(ctx, itemA, 5*time.Second, time.Now(), nil)
				require.NoError(t, err)
			})

			t.Run("Partition lease errors without capacity", func(t *testing.T) {
				leaseId, _, err := shard.PartitionLease(ctx, &p, 5*time.Second)
				require.Nil(t, leaseId, "No lease id when leasing fails.\n%s", r.Dump())
				require.Error(t, err)
				require.ErrorIs(t, err, osqueue.ErrAccountConcurrencyLimit)
			})
		})

		t.Run("With custom concurrency limits", func(t *testing.T) {
			r.FlushAll()
			// Only allow a single leased item via account limits
			ckHash := util.XXHash("key-expr")

			_, shard := newQueue(t, rc,
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					// Only allow a single leased item via account limits
					return osqueue.PartitionConstraintConfig{
						Concurrency: osqueue.PartitionConcurrency{
							AccountConcurrency:  100,
							FunctionConcurrency: 100,
							CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
								{
									Scope:               enums.ConcurrencyScopeAccount,
									HashedKeyExpression: ckHash,
									Limit:               1,
								},
							},
						},
					}
				}))

			accountId := uuid.New()
			ck := createConcurrencyKey(enums.ConcurrencyScopeAccount, accountId, "foo", 1)

			// Create a new item
			itemA, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					Identifier: state.Identifier{AccountID: accountId},
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key:   ck.Key,
							Hash:  ckHash,
							Limit: 1,
						},
					},
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			_, err = shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					Identifier: state.Identifier{AccountID: accountId},
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key:   ck.Key,
							Limit: 1,
						},
					},
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			t.Run("Leases with capacity", func(t *testing.T) {
				_, err = shard.Lease(ctx, itemA, 5*time.Second, time.Now(), nil)
				require.NoError(t, err)
			})

			t.Run("Partition lease on default fn does not error without capacity", func(t *testing.T) {
				p := osqueue.QueuePartition{FunctionID: &itemA.FunctionID, AccountID: accountId}

				// Since we don't peek and lease concurrency key queue partitions anymore,
				// we won't check for custom concurrency limits ahead of processing items.
				// Leasing a default partition works even though the concurrency key has no additional capacity.
				leaseId, _, err := shard.PartitionLease(ctx, &p, 5*time.Second)
				require.NotNil(t, leaseId, "Expected lease id.\n%s", r.Dump())
				require.NoError(t, err)
			})
		})
	})

	t.Run("disabling constraint checks should work", func(t *testing.T) {
		r.FlushAll()

		_, shard := newQueue(t, rc,
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				// Only allow a single leased item via account limits
				return osqueue.PartitionConstraintConfig{
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  1,
						FunctionConcurrency: 1,
					},
				}
			}))

		start := time.Now()

		fnID := uuid.New()
		// Create a new item
		itemA, err := shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: fnID}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		_, err = shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: fnID}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		// Use the new item's workflow ID
		p := osqueue.QueuePartition{ID: itemA.FunctionID.String(), FunctionID: &itemA.FunctionID}

		_, err = shard.Lease(ctx, itemA, 5*time.Second, time.Now(), nil)
		require.NoError(t, err)

		// Should fail without skipping
		leaseId, _, err := shard.PartitionLease(ctx, &p, 5*time.Second)
		require.Nil(t, leaseId, "No lease id when leasing fails.\n%s", r.Dump())
		require.Error(t, err)
		require.ErrorIs(t, err, osqueue.ErrPartitionConcurrencyLimit)

		// Should work with skip
		leaseId, _, err = shard.PartitionLease(ctx, &p, 5*time.Second, osqueue.PartitionLeaseOptionDisableLeaseChecks(true))
		require.NoError(t, err)
		require.NotNil(t, leaseId)
	})
}
