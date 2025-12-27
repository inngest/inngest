package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
)

func TestQueueLease(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()
	_, shard := newQueue(
		t, rc,
		osqueue.WithClock(clock),
	)
	kg := shard.Client().kg

	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	t.Run("It leases an item", func(t *testing.T) {
		fnID, accountID := uuid.New(), uuid.New()
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					RunID:      runID,
					WorkflowID: fnID,
					AccountID:  accountID,
				},
			},
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := osqueue.QueuePartition{
			ID:         fnID.String(),
			FunctionID: &fnID,
			AccountID:  accountID,
		} // Default workflow ID etc

		t.Run("It should exist in the pending partition queue", func(t *testing.T) {
			mem, err := r.ZMembers(partitionZsetKey(p, kg))
			require.NoError(t, err)
			require.Equal(t, 1, len(mem))
		})

		now := time.Now()
		leaseExpiry := now.Add(time.Second)
		id, err := shard.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, leaseExpiry, ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It should remove from the pending partition queue", func(t *testing.T) {
			mem, _ := r.ZMembers(partitionZsetKey(p, kg))
			require.Empty(t, mem)
		})

		t.Run("It should add the item to the function's in-progress concurrency queue", func(t *testing.T) {
			count, err := shard.InProgress(ctx, "p", fnID.String())
			require.NoError(t, err)
			require.EqualValues(t, 1, count, r.Dump())
		})

		t.Run("run indexes are updated", func(t *testing.T) {
			// Run indexes should be updated
			{
				itemIsMember, err := r.SIsMember(kg.ActiveSet("run", runID.String()), item.ID)
				require.NoError(t, err)
				require.True(t, itemIsMember)

				isMember, err := r.SIsMember(kg.ActiveRunsSet("p", fnID.String()), runID.String())
				require.NoError(t, err)
				require.True(t, isMember)

				accountIsMember, err := r.SIsMember(kg.ActiveRunsSet("account", accountID.String()), runID.String())
				require.NoError(t, err)
				require.True(t, accountIsMember)
			}
		})

		t.Run("Scavenge queue is updated", func(t *testing.T) {
			mem, err := r.ZMembers(kg.ConcurrencyIndex())
			require.NoError(t, err)
			require.Equal(t, 1, len(mem), "scavenge queue should have 1 item", mem)
			require.Contains(t, mem, p.FunctionID.String())

			score, err := r.ZMScore(kg.ConcurrencyIndex(), p.FunctionID.String())
			require.NoError(t, err)

			require.WithinDuration(t, leaseExpiry, time.UnixMilli(int64(score[0])), 2*time.Millisecond)
		})

		t.Run("Leasing again should fail", func(t *testing.T) {
			for i := 0; i < 50; i++ {
				id, err := shard.Lease(ctx, item, time.Second, time.Now(), nil)
				require.Equal(t, osqueue.ErrQueueItemAlreadyLeased, err)
				require.Nil(t, id)
				<-time.After(5 * time.Millisecond)
			}
		})

		t.Run("Leasing an expired lease should succeed", func(t *testing.T) {
			<-time.After(1005 * time.Millisecond)

			// Now expired
			t.Run("After expiry, no items should be in progress", func(t *testing.T) {
				count, err := shard.InProgress(ctx, "p", p.FunctionID.String())
				require.NoError(t, err)
				require.EqualValues(t, 0, count)
			})

			now := time.Now()
			id, err := shard.Lease(ctx, item, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)
			require.NoError(t, err)

			item = getQueueItem(t, r, item.ID)
			require.NotNil(t, item.LeaseID)
			require.EqualValues(t, id, item.LeaseID)
			require.WithinDuration(t, now.Add(5*time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

			t.Run("Leasing an expired key has one in-progress", func(t *testing.T) {
				count, err := shard.InProgress(ctx, "p", p.FunctionID.String())
				require.NoError(t, err)
				require.EqualValues(t, 1, count)
			})
		})

		t.Run("It should remove the item from the function queue, as this is now in the partition's in-progress concurrency queue", func(t *testing.T) {
			start := time.Now()
			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			require.Nil(t, item.LeaseID)

			requireItemScoreEquals(t, r, item, start)

			_, err = shard.Lease(ctx, item, time.Minute, time.Now(), nil)
			require.NoError(t, err)

			_, err = r.ZScore(kg.FnQueueSet(item.FunctionID.String()), item.ID)
			require.Error(t, err, "no such key")
		})

		t.Run("it should not update the partition score to the next item", func(t *testing.T) {
			r.FlushAll()

			timeNow := time.Now().Truncate(time.Second)
			timeNowPlusFiveSeconds := timeNow.Add(time.Second * 5).Truncate(time.Second)

			acctId := uuid.New()

			// Enqueue future item (partition time will be now + 5s)
			item, err = shard.EnqueueItem(ctx, osqueue.QueueItem{
				Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}},
			}, timeNowPlusFiveSeconds, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			require.Nil(t, item.LeaseID)

			qp := getDefaultPartition(t, r, uuid.Nil)

			requireItemScoreEquals(t, r, item, timeNowPlusFiveSeconds)
			requirePartitionItemScoreEquals(t, r, kg.GlobalPartitionIndex(), qp, timeNowPlusFiveSeconds)
			requirePartitionItemScoreEquals(t, r, kg.AccountPartitionIndex(acctId), qp, timeNowPlusFiveSeconds)
			requireAccountScoreEquals(t, r, acctId, timeNowPlusFiveSeconds)

			// Enqueue current item (partition time will be moved up to now)
			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, timeNow, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			require.Nil(t, item.LeaseID)

			// We do expect the item score to change!
			requireItemScoreEquals(t, r, item, timeNow)

			requirePartitionItemScoreEquals(t, r, kg.GlobalPartitionIndex(), qp, timeNow)
			requirePartitionItemScoreEquals(t, r, kg.AccountPartitionIndex(acctId), qp, timeNow)
			requireAccountScoreEquals(t, r, acctId, timeNow)

			// Lease item (keeps partition time constant)
			_, err = shard.Lease(ctx, item, time.Minute, clock.Now(), nil)
			require.NoError(t, err)

			requirePartitionItemScoreEquals(t, r, kg.GlobalPartitionIndex(), qp, timeNow)
			requirePartitionItemScoreEquals(t, r, kg.AccountPartitionIndex(acctId), qp, timeNow)
			requireAccountScoreEquals(t, r, acctId, timeNow)
		})
	})

	// Test default partition-level concurrency limits (not custom)
	t.Run("With partition concurrency limits", func(t *testing.T) {
		r.FlushAll()

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  1,
						FunctionConcurrency: 1,
						SystemConcurrency:   1,
					},
				}
			}),
		)

		fnID := uuid.New()
		// Create a new item
		itemA, err := shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: fnID}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		itemB, err := shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: fnID}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		// Use the new item's workflow ID
		p := osqueue.QueuePartition{ID: itemA.FunctionID.String(), FunctionID: &itemA.FunctionID}

		t.Run("With denylists it does not lease.", func(t *testing.T) {
			list := osqueue.NewLeaseDenyList()
			list.AddConcurrency(osqueue.NewKeyError(osqueue.ErrPartitionConcurrencyLimit, p.Queue()))
			id, err := shard.Lease(ctx, itemA, 5*time.Second, time.Now(), list)
			require.NotNil(t, err, "Expcted error leasing denylists")
			require.Nil(t, id, "Expected nil ID with denylists")
			require.ErrorIs(t, err, osqueue.ErrPartitionConcurrencyLimit)
		})

		t.Run("Leases with capacity", func(t *testing.T) {
			_, err = shard.Lease(ctx, itemA, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)
		})

		t.Run("Errors without capacity", func(t *testing.T) {
			id, err := shard.Lease(ctx, itemB, 5*time.Second, time.Now(), nil)
			require.Nil(t, id, "Leased item when concurrency limits are reached.\n%s", r.Dump())
			require.Error(t, err)
		})
	})

	// Test default account concurrency limits (not custom)
	t.Run("With account concurrency limits", func(t *testing.T) {
		r.FlushAll()

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  1,
						FunctionConcurrency: osqueue.NoConcurrencyLimit,
					},
				}
			}),
		)

		acctId := uuid.New()

		// Create a new item
		itemA, err := shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: uuid.New(), Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		itemB, err := shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: uuid.New(), Data: osqueue.Item{Identifier: state.Identifier{AccountID: acctId}}}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		t.Run("Leases with capacity", func(t *testing.T) {
			_, err = shard.Lease(ctx, itemA, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)
		})

		t.Run("Errors without capacity", func(t *testing.T) {
			id, err := shard.Lease(ctx, itemB, 5*time.Second, time.Now(), nil)
			require.Nil(t, id)
			require.Error(t, err)
			require.ErrorIs(t, err, osqueue.ErrAccountConcurrencyLimit)
		})
	})

	t.Run("With custom concurrency limits", func(t *testing.T) {
		t.Run("with account keys", func(t *testing.T) {
			r.FlushAll()

			ck := createConcurrencyKey(enums.ConcurrencyScopeAccount, uuid.Nil, "foo", 1)

			_, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					return osqueue.PartitionConstraintConfig{
						Concurrency: osqueue.PartitionConcurrency{
							AccountConcurrency:  osqueue.NoConcurrencyLimit,
							FunctionConcurrency: osqueue.NoConcurrencyLimit,
							CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
								{
									Scope:               enums.ConcurrencyScopeAccount,
									HashedKeyExpression: ck.Hash,
									Limit:               ck.Limit,
								},
							},
						},
					}
				}),
			)

			// Create a new item
			fnA := uuid.New()
			itemA, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: fnA,
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						ck,
					},
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			itemB, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						ck,
					},
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			t.Run("With denylists it does not lease.", func(t *testing.T) {
				list := osqueue.NewLeaseDenyList()
				list.AddConcurrency(osqueue.NewKeyError(osqueue.ErrConcurrencyLimitCustomKey, ck.Key))
				_, err = shard.Lease(ctx, itemA, 5*time.Second, time.Now(), list)
				require.NotNil(t, err)
				require.ErrorIs(t, err, osqueue.ErrConcurrencyLimitCustomKey)
			})

			t.Run("Leases with capacity", func(t *testing.T) {
				now := time.Now()
				_, err = shard.Lease(ctx, itemA, 5*time.Second, now, nil)
				require.NoError(t, err)

				t.Run("Scavenge queue is updated", func(t *testing.T) {
					mem, err := r.ZMembers(kg.ConcurrencyIndex())
					require.NoError(t, err, r.Dump())
					require.Equal(t, 1, len(mem), "scavenge queue should have 1 item", mem)
					require.Contains(t, mem, fnA.String())

					score, err := r.ZMScore(kg.ConcurrencyIndex(), fnA.String())
					require.NoError(t, err)
					require.Equal(t, float64(now.Add(5*time.Second).UnixMilli()), score[0])
				})
			})

			t.Run("Errors without capacity", func(t *testing.T) {
				id, err := shard.Lease(ctx, itemB, 5*time.Second, time.Now(), nil)
				require.Nil(t, id)
				require.Error(t, err)
			})
		})

		t.Run("with function keys", func(t *testing.T) {
			r.FlushAll()

			accountId := uuid.New()
			fnId := uuid.New()

			ck := createConcurrencyKey(enums.ConcurrencyScopeFn, fnId, "foo", 1)
			_, _, keyExprChecksum, err := ck.ParseKey()
			require.NoError(t, err)

			// Only allow a single leased item via custom concurrency limits
			_, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					return osqueue.PartitionConstraintConfig{
						Concurrency: osqueue.PartitionConcurrency{
							AccountConcurrency:  osqueue.NoConcurrencyLimit,
							FunctionConcurrency: osqueue.NoConcurrencyLimit,
							CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
								{
									Scope:               enums.ConcurrencyScopeFn,
									HashedKeyExpression: ck.Hash,
									Limit:               ck.Limit,
								},
							},
						},
					}
				}),
			)

			// Create a new item
			itemA, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: fnId,
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key:   ck.Key,
							Limit: 1,
						},
					},
					Identifier: state.Identifier{
						AccountID: accountId,
					},
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			itemB, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: fnId,
				Data: osqueue.Item{
					CustomConcurrencyKeys: []state.CustomConcurrency{
						{
							Key:   ck.Key,
							Limit: 1,
						},
					},
					Identifier: state.Identifier{
						AccountID: accountId,
					},
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			zsetKeyA := kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnId.String(), keyExprChecksum)
			pA := osqueue.QueuePartition{ID: zsetKeyA, AccountID: accountId, FunctionID: &itemA.FunctionID}

			t.Run("With denylists it does not lease.", func(t *testing.T) {
				list := osqueue.NewLeaseDenyList()
				list.AddConcurrency(osqueue.NewKeyError(osqueue.ErrConcurrencyLimitCustomKey, ck.Key))
				_, err = shard.Lease(ctx, itemA, 5*time.Second, time.Now(), list)
				require.NotNil(t, err)
				require.ErrorIs(t, err, osqueue.ErrConcurrencyLimitCustomKey)
			})

			t.Run("Leases with capacity", func(t *testing.T) {
				// Use the new item's workflow ID
				require.Equal(t, partitionZsetKey(pA, kg), zsetKeyA)

				// partition key queue does not exist
				require.False(t, r.Exists(partitionZsetKey(pA, kg)), "partition shouldn't have been added by enqueue or lease")
				// require.True(t, r.Exists(zsetKeyA))
				// memPart, err := r.ZMembers(zsetKeyA)
				// require.NoError(t, err)
				// require.Equal(t, 2, len(memPart))
				// require.Contains(t, memPart, itemA.ID)
				// require.Contains(t, memPart, itemB.ID)

				// concurrency key queue does not yet exist
				require.False(t, r.Exists(partitionConcurrencyKey(pA, kg)))

				_, err = shard.Lease(ctx, itemA, 5*time.Second, time.Now(), nil)
				require.NoError(t, err)

				// memPart, err = r.ZMembers(zsetKeyA)
				// require.NoError(t, err)
				// require.Equal(t, 1, len(memPart))
				// require.Contains(t, memPart, itemB.ID)

				require.True(t, r.Exists(partitionConcurrencyKey(pA, kg)))
				memConcurrency, err := r.ZMembers(partitionConcurrencyKey(pA, kg))
				require.NoError(t, err)
				require.Equal(t, 1, len(memConcurrency))
				require.Contains(t, memConcurrency, itemA.ID)
			})

			t.Run("Errors without capacity", func(t *testing.T) {
				id, err := shard.Lease(ctx, itemB, 5*time.Second, time.Now(), nil)
				require.Nil(t, id)
				require.Error(t, err)
				require.ErrorIs(t, err, osqueue.ErrConcurrencyLimitCustomKey)
			})
		})

		// this test is the unit variant of TestConcurrency_ScopeFunction_FanOut in cloud
		t.Run("with two distinct functions it processes both", func(t *testing.T) {
			r.FlushAll()

			fnIDA := uuid.New()
			fnIDB := uuid.New()

			ckA := createConcurrencyKey(enums.ConcurrencyScopeFn, fnIDA, "foo", 1)
			_, _, evaluatedKeyChecksumA, err := ckA.ParseKey()
			require.NoError(t, err)

			ckB := createConcurrencyKey(enums.ConcurrencyScopeFn, fnIDB, "foo", 1)
			_, _, evaluatedKeyChecksumB, err := ckB.ParseKey()
			require.NoError(t, err)

			_, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					return osqueue.PartitionConstraintConfig{
						Concurrency: osqueue.PartitionConcurrency{
							AccountConcurrency:  123_456,
							FunctionConcurrency: 1,
							CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
								{
									Scope:               enums.ConcurrencyScopeFn,
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

			// Create a new item
			itemA1, err := shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: fnIDA, Data: osqueue.Item{CustomConcurrencyKeys: []state.CustomConcurrency{ckA}}}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			itemA2, err := shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: fnIDA, Data: osqueue.Item{CustomConcurrencyKeys: []state.CustomConcurrency{ckA}}}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			itemB1, err := shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: fnIDB, Data: osqueue.Item{CustomConcurrencyKeys: []state.CustomConcurrency{ckB}}}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			itemB2, err := shard.EnqueueItem(ctx, osqueue.QueueItem{FunctionID: fnIDB, Data: osqueue.Item{CustomConcurrencyKeys: []state.CustomConcurrency{ckB}}}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			// Use the new item's workflow ID
			zsetKeyA := kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnIDA.String(), evaluatedKeyChecksumA)

			partitionIsMissingInHash(t, r, enums.PartitionTypeConcurrencyKey, fnIDA, evaluatedKeyChecksumA)

			zsetKeyB := kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, fnIDB.String(), evaluatedKeyChecksumB)
			partitionIsMissingInHash(t, r, enums.PartitionTypeConcurrencyKey, fnIDB, evaluatedKeyChecksumB)

			// Both key queues do not exist
			require.False(t, r.Exists(zsetKeyA))
			require.False(t, r.Exists(zsetKeyB))

			// Lease item A1 - should work
			_, err = shard.Lease(ctx, itemA1, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)

			// Lease item B1 - should work
			_, err = shard.Lease(ctx, itemB1, 5*time.Second, time.Now(), nil)
			require.NoError(t, err)

			// Lease item A2 - should fail due to custom concurrency limit
			_, err = shard.Lease(ctx, itemA2, 5*time.Second, time.Now(), nil)
			require.ErrorIs(t, err, osqueue.ErrConcurrencyLimitCustomKey)

			// Lease item B1 - should fail due to custom concurrency limit
			_, err = shard.Lease(ctx, itemB2, 5*time.Second, time.Now(), nil)
			require.ErrorIs(t, err, osqueue.ErrConcurrencyLimitCustomKey)
		})
	})

	t.Run("It should update the global partition index", func(t *testing.T) {
		t.Run("With no concurrency keys", func(t *testing.T) {
			r.FlushAll()

			_, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					return osqueue.PartitionConstraintConfig{}
				}),
			)

			// NOTE: We need two items to ensure that this updates.  Leasing an
			// item removes it from the fn queue.
			t.Run("With a single item in the queue hwen leasing, nothing updates", func(t *testing.T) {
				at := time.Now().Truncate(time.Second).Add(time.Second)
				accountId := uuid.New()
				item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
					Data: osqueue.Item{Identifier: state.Identifier{AccountID: accountId}},
				}, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)
				p := osqueue.QueuePartition{FunctionID: &item.FunctionID}

				score, err := r.ZScore(kg.GlobalPartitionIndex(), p.Queue())
				require.NoError(t, err)
				require.EqualValues(t, at.Unix(), score, r.Dump())

				score, err = r.ZScore(kg.AccountPartitionIndex(accountId), p.Queue())
				require.NoError(t, err)
				require.EqualValues(t, at.Unix(), score, r.Dump())

				// Nothing should update here, as there's nothing left in the fn queue
				// so nothing happens.
				_, err = shard.Lease(ctx, item, 10*time.Second, time.Now(), nil)
				require.NoError(t, err)

				nextScore, err := r.ZScore(kg.GlobalPartitionIndex(), p.Queue())
				require.NoError(t, err)
				require.EqualValues(t, int(score), int(nextScore), "score should not equal previous score")

				nextScore, err = r.ZScore(kg.AccountPartitionIndex(accountId), p.Queue())
				require.NoError(t, err)
				require.EqualValues(t, int(score), int(nextScore), "account score should not equal previous score")
			})
		})

		t.Run("With custom concurrency keys", func(t *testing.T) {
			r.FlushAll()

			t.Run("It moves items from each concurrency queue", func(t *testing.T) {
				at := time.Now().Truncate(time.Second).Add(time.Second)
				itemA, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
					Data: osqueue.Item{
						CustomConcurrencyKeys: []state.CustomConcurrency{
							{
								Key: util.ConcurrencyKey(
									enums.ConcurrencyScopeAccount,
									uuid.Nil,
									"acct-id",
								),
								Limit: 10,
							},
							{
								Key: util.ConcurrencyKey(
									enums.ConcurrencyScopeFn,
									uuid.Nil,
									"fn-id",
								),
								Limit: 5,
							},
						},
					},
				}, at, osqueue.EnqueueOpts{})
				backlogA := osqueue.ItemBacklog(ctx, itemA)
				require.NoError(t, err)
				itemB, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
					Data: osqueue.Item{
						CustomConcurrencyKeys: []state.CustomConcurrency{
							{
								Key: util.ConcurrencyKey(
									enums.ConcurrencyScopeAccount,
									uuid.Nil,
									"acct-id",
								),
								Limit: 10,
							},
							{
								Key: util.ConcurrencyKey(
									enums.ConcurrencyScopeFn,
									uuid.Nil,
									"fn-id",
								),
								Limit: 5,
							},
						},
					},
				}, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)
				backlogB := osqueue.ItemBacklog(ctx, itemB)

				defaultPartition := getDefaultPartition(t, r, uuid.Nil)

				// Since we do not enqueue concurrency queues, we need to check for the default partition score
				score, err := r.ZScore(kg.GlobalPartitionIndex(), defaultPartition.ID)
				require.NoError(t, err)
				require.EqualValues(t, at.Unix(), score, r.Dump())

				// Concurrency queue should be emptyu
				t.Run("Concurrency and scavenge queues are empty", func(t *testing.T) {
					mem, _ := r.ZMembers(kg.ConcurrencyIndex())
					require.Empty(t, mem, "concurrency queue is not empty")
				})

				// Do the lease.
				_, err = shard.Lease(ctx, itemA, 10*time.Second, clock.Now(), nil)
				require.NoError(t, err)

				// The queue item is removed from each partition
				t.Run("The queue item is removed from each partition", func(t *testing.T) {
					mem, _ := r.ZMembers(partitionZsetKey(defaultPartition, kg))
					require.Equal(t, 1, len(mem), "leased item not removed from first partition", partitionZsetKey(defaultPartition, kg))
				})

				t.Run("The scavenger queue is updated with just the default partition", func(t *testing.T) {
					mem, _ := r.ZMembers(kg.ConcurrencyIndex())
					require.Equal(t, 1, len(mem), "scavenge queue not updated", mem)
					require.NotContains(t, mem, backlogCustomKeyInProgress(backlogA, kg, 1))
					require.NotContains(t, mem, backlogCustomKeyInProgress(backlogB, kg, 1))
					require.NotContains(t, mem, partitionConcurrencyKey(defaultPartition, kg))
					require.Contains(t, mem, defaultPartition.FunctionID.String())
				})

				t.Run("Pointer queues don't update with a single queue item", func(t *testing.T) {
					nextScore, err := r.ZScore(kg.GlobalPartitionIndex(), defaultPartition.Queue())
					require.NoError(t, err)
					require.EqualValues(t, int(score), int(nextScore), "score should not equal previous score")
				})
			})
		})

		t.Run("With more than one item in the fn queue, it uses the next val for the global partition index", func(t *testing.T) {
			r.FlushAll()

			atA := time.Now().Truncate(time.Second).Add(time.Second)
			atB := atA.Add(time.Minute)

			itemA, err := shard.EnqueueItem(ctx, osqueue.QueueItem{}, atA, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			_, err = shard.EnqueueItem(ctx, osqueue.QueueItem{}, atB, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			p := osqueue.ItemPartition(ctx, itemA)

			score, err := r.ZScore(kg.GlobalPartitionIndex(), p.Queue())
			require.NoError(t, err)
			require.EqualValues(t, atA.Unix(), score)

			// Leasing the item should update the score.
			_, err = shard.Lease(ctx, itemA, 10*time.Second, time.Now(), nil)
			require.NoError(t, err)

			nextScore, err := r.ZScore(kg.GlobalPartitionIndex(), p.Queue())
			require.NoError(t, err)
			// lease should match first item, as we don't update pointer scores during lease
			require.EqualValues(t, itemA.AtMS/1000, int(nextScore))
			require.EqualValues(t, int(score), int(nextScore), "score should not equal previous score")
		})
	})

	t.Run("It does nothing for a zero value partition", func(t *testing.T) {
		r.FlushAll()

		item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := osqueue.QueuePartition{} // Empty partition

		now := time.Now()
		id, err := shard.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It should NOT add the item to the function's in-progress concurrency queue", func(t *testing.T) {
			require.False(t, r.Exists(partitionConcurrencyKey(p, kg)))
		})
	})

	t.Run("system partitions should be leased properly", func(t *testing.T) {
		r.FlushAll()

		systemQueueName := "system-queue"
		item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
			QueueName: &systemQueueName,
			Data: osqueue.Item{
				QueueName: &systemQueueName,
			},
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		require.True(t, r.Exists("{queue}:queue:sorted:system-queue"))

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := getSystemPartition(t, r, systemQueueName)

		now := time.Now()
		id, err := shard.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		require.False(t, r.Exists("{queue}:queue:sorted:system-queue"))
		require.False(t, r.Exists("{queue}:concurrency:account:system-queue"), r.Dump()) // System queues should not have account concurrency set
		require.True(t, r.Exists("{queue}:concurrency:p:system-queue"))

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		require.True(t, r.Exists(partitionConcurrencyKey(p, kg)), r.Dump())
	})

	t.Run("batch system partitions should be leased properly", func(t *testing.T) {
		r.FlushAll()

		systemQueueName := osqueue.KindScheduleBatch
		qi := osqueue.QueueItem{
			QueueName: &systemQueueName,
			Data: osqueue.Item{
				QueueName: &systemQueueName,
			},
		}

		kg := queueKeyGenerator{
			queueDefaultKey: QueueDefaultKey,
			queueItemKeyGenerator: queueItemKeyGenerator{
				queueDefaultKey: QueueDefaultKey,
			},
		}

		// Sanity check: Ensure partitions are created properly and keys match old system
		fnPart := osqueue.ItemPartition(ctx, qi)
		require.Equal(t, osqueue.QueuePartition{
			ID:        systemQueueName,
			QueueName: &systemQueueName,
		}, fnPart)
		require.True(t, fnPart.IsSystem())

		require.Equal(t, "{queue}:queue:sorted:schedule-batch", partitionZsetKey(fnPart, kg))
		require.Equal(t, "{queue}:concurrency:p:schedule-batch", partitionConcurrencyKey(fnPart, kg))

		item, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		require.True(t, r.Exists("{queue}:queue:sorted:schedule-batch"))

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		p := getSystemPartition(t, r, systemQueueName)

		now := time.Now()
		id, err := shard.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		require.False(t, r.Exists("{queue}:queue:sorted:schedule-batch"))

		// batching uses different rules for concurrency keys
		require.True(t, r.Exists("{queue}:concurrency:p:schedule-batch"))
		require.False(t, r.Exists("{queue}:concurrency:account:schedule-batch"), r.Dump())

		require.False(t, r.Exists("{queue}:concurrency:account:00000000-0000-0000-0000-000000000000"), r.Dump())
		require.False(t, r.Exists("{queue}:concurrency:p:00000000-0000-0000-0000-000000000000"))

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		require.True(t, r.Exists(partitionConcurrencyKey(p, kg)), r.Dump())
	})

	t.Run("leasing key queue should clear backward-compat default partition", func(t *testing.T) {
		r.FlushAll()

		// This is required as not dropping items from all partitions during lease will cause a leftover item to be in the default partition
		// When the item has been processed, and we run Dequeue, this only happens on the key queue, and the default partition retains its pointer even though the queue item is deleted
		// This leads to Peek errors in default partitions, including system partitions (encountered missing queue items in partition queue)

		accountId := uuid.New()

		evaluatedKey := util.ConcurrencyKey(enums.ConcurrencyScopeAccount, accountId, "customer-1")
		ck := state.CustomConcurrency{
			Key:   evaluatedKey,
			Hash:  util.XXHash("event.data.customerId"),
			Limit: 10,
		}

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					Concurrency: osqueue.PartitionConcurrency{
						SystemConcurrency:   0,
						AccountConcurrency:  123,
						FunctionConcurrency: 45,
						CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
							{
								Scope:               enums.ConcurrencyScopeAccount,
								HashedKeyExpression: ck.Hash,
								Limit:               ck.Limit,
							},
						},
					},
				}
			}),
		)
		kg := shard.Client().kg

		fnId := uuid.New()
		item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
			FunctionID: fnId,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID:  accountId,
					WorkflowID: fnId,
				},
				CustomConcurrencyKeys: []state.CustomConcurrency{
					ck,
				},
			},
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		defaultPart := getDefaultPartition(t, r, fnId)
		backlog := osqueue.ItemBacklog(ctx, item)
		sp := osqueue.ItemShadowPartition(ctx, item)

		require.True(t, r.Exists(partitionZsetKey(defaultPart, kg)))

		now := time.Now()
		id, err := shard.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		require.False(t, r.Exists(partitionZsetKey(defaultPart, kg)))

		require.True(t, r.Exists(backlogCustomKeyInProgress(backlog, kg, 1)))
		require.True(t, r.Exists(shadowPartitionAccountInProgressKey(sp, kg)))
		require.True(t, r.Exists(partitionConcurrencyKey(defaultPart, kg)))

		err = shard.Dequeue(ctx, item)
		require.NoError(t, err)

		require.False(t, r.Exists(partitionZsetKey(defaultPart, kg)))
		require.False(t, r.Exists(backlogCustomKeyInProgress(backlog, kg, 1)))
		require.False(t, r.Exists(shadowPartitionAccountInProgressKey(sp, kg)))
		require.False(t, r.Exists(partitionConcurrencyKey(defaultPart, kg)))
	})

	t.Run("leasing with throttle item data or constraints", func(t *testing.T) {
		t.Run("item with throttle but no constraints", func(t *testing.T) {
			r.FlushAll()
			fnID, accountID := uuid.New(), uuid.New()
			runID := ulid.MustNew(ulid.Now(), rand.Reader)

			_, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					return osqueue.PartitionConstraintConfig{
						Throttle: nil,
					}
				}),
			)

			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					Kind: osqueue.KindStart,
					Identifier: state.Identifier{
						RunID:      runID,
						WorkflowID: fnID,
						AccountID:  accountID,
					},
					Throttle: &osqueue.Throttle{
						Key:               "throttle-key",
						Limit:             1,
						Burst:             0,
						Period:            5,
						KeyExpressionHash: util.XXHash("expr"),
					},
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			leaseID, err := shard.Lease(ctx, item, 10*time.Second, clock.Now(), nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)
		})

		t.Run("item with throttle and matching constraints", func(t *testing.T) {
			r.FlushAll()
			fnID, accountID := uuid.New(), uuid.New()
			runID := ulid.MustNew(ulid.Now(), rand.Reader)

			_, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					return osqueue.PartitionConstraintConfig{
						Throttle: &osqueue.PartitionThrottle{
							ThrottleKeyExpressionHash: util.XXHash("expr"),
							Limit:                     1,
							Burst:                     0,
							Period:                    5,
						},
					}
				}),
			)

			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					Kind: osqueue.KindStart,
					Identifier: state.Identifier{
						RunID:      runID,
						WorkflowID: fnID,
						AccountID:  accountID,
					},
					Throttle: &osqueue.Throttle{
						Key:               "throttle-key",
						Limit:             1,
						Burst:             0,
						Period:            5,
						KeyExpressionHash: util.XXHash("expr"),
					},
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			leaseID, err := shard.Lease(ctx, item, 10*time.Second, clock.Now(), nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)
		})

		t.Run("item with throttle and mismatching constraints", func(t *testing.T) {
			r.FlushAll()
			fnID, accountID := uuid.New(), uuid.New()
			runID := ulid.MustNew(ulid.Now(), rand.Reader)

			_, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					return osqueue.PartitionConstraintConfig{
						Throttle: &osqueue.PartitionThrottle{
							ThrottleKeyExpressionHash: util.XXHash("different-constraints"),
							Limit:                     5,
							Burst:                     1,
							Period:                    60,
						},
					}
				}),
			)

			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					Kind: osqueue.KindStart,
					Identifier: state.Identifier{
						RunID:      runID,
						WorkflowID: fnID,
						AccountID:  accountID,
					},
					Throttle: &osqueue.Throttle{
						Key:               "throttle-key",
						Limit:             1,
						Burst:             0,
						Period:            5,
						KeyExpressionHash: util.XXHash("expr"),
					},
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			leaseID, err := shard.Lease(ctx, item, 10*time.Second, clock.Now(), nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)
		})

		t.Run("item without throttle but constraints", func(t *testing.T) {
			r.FlushAll()
			fnID, accountID := uuid.New(), uuid.New()
			runID := ulid.MustNew(ulid.Now(), rand.Reader)

			_, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					return osqueue.PartitionConstraintConfig{
						Throttle: &osqueue.PartitionThrottle{
							ThrottleKeyExpressionHash: util.XXHash("expr"),
							Limit:                     1,
							Burst:                     0,
							Period:                    5,
						},
					}
				}),
			)

			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					Kind: osqueue.KindStart,
					Identifier: state.Identifier{
						RunID:      runID,
						WorkflowID: fnID,
						AccountID:  accountID,
					},
					Throttle: nil,
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			leaseID, err := shard.Lease(ctx, item, 10*time.Second, clock.Now(), nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)
		})

		t.Run("non-start item with throttle constraints", func(t *testing.T) {
			r.FlushAll()
			fnID, accountID := uuid.New(), uuid.New()
			runID := ulid.MustNew(ulid.Now(), rand.Reader)

			_, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					return osqueue.PartitionConstraintConfig{
						Throttle: &osqueue.PartitionThrottle{
							ThrottleKeyExpressionHash: util.XXHash("expr"),
							Limit:                     1,
							Burst:                     0,
							Period:                    5,
						},
					}
				}),
			)

			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					Kind: osqueue.KindEdge,
					Identifier: state.Identifier{
						RunID:      runID,
						WorkflowID: fnID,
						AccountID:  accountID,
					},
					Throttle: nil,
				},
			}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			leaseID, err := shard.Lease(ctx, item, 10*time.Second, clock.Now(), nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)
		})
	})
}

func TestQueueLeaseWithoutValidation(t *testing.T) {
	t.Run("simple item", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
		now := clock.Now()

		enqueueToBacklog := false
		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute).Truncate(time.Minute)

		t.Run("should lease item", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			item1 := osqueue.QueueItem{
				ID:          "test",
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						WorkflowID:  fnID,
						AccountID:   accountId,
						WorkspaceID: wsID,
					},
					QueueName:             nil,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName:    nil,
				RefilledFrom: "fake-backlog",
				RefilledAt:   at.UnixMilli(),
			}

			fnPart := osqueue.ItemPartition(ctx, item1)

			// for simplicity, this enqueue should go directly to the partition
			enqueueToBacklog = false
			qi, err := shard.EnqueueItem(ctx, item1, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			now := clock.Now()
			leaseDur := 5 * time.Second

			// simulate having hit a partition concurrency limit in a previous operation,
			// without disabling validation this should cause Lease() to fail
			denies := osqueue.NewLeaseDenyList()
			denies.AddConcurrency(osqueue.NewKeyError(osqueue.ErrPartitionConcurrencyLimit, fnPart.Queue()))

			leaseID, err := shard.Lease(ctx, qi, leaseDur, now, denies, osqueue.LeaseOptionDisableConstraintChecks(true))
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := osqueue.ItemBacklog(ctx, item1)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := osqueue.ItemShadowPartition(ctx, item1)
			require.NotEmpty(t, shadowPartition.PartitionID)

			// NOTE: With the Constraint API, disabling lease checks also removes constraint state updates. We still add the item to the new scavenger index,
			// but no longer populate the in progress sets.
			require.False(t, r.Exists(shadowPartitionAccountInProgressKey(shadowPartition, kg)))
			require.False(t, r.Exists(shadowPartitionInProgressKey(shadowPartition, kg)))
			require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 1))
			require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 2))
			require.False(t, r.Exists(backlogCustomKeyInProgress(backlog, kg, 1)))

			require.False(t, r.Exists(kg.Concurrency("account", accountId.String())))
			require.False(t, r.Exists(partitionConcurrencyKey(fnPart, kg)))
		})
	})

	t.Run("single custom concurrency key", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
		now := clock.Now()

		enqueueToBacklog := false
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute).Truncate(time.Minute)

		t.Run("should enqueue item to backlog", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			hashedConcurrencyKeyExpr := hashConcurrencyKey("event.data.customerId")
			unhashedValue := "customer1"
			scope := enums.ConcurrencyScopeFn
			fullKey := util.ConcurrencyKey(scope, fnID, unhashedValue)

			ckA := state.CustomConcurrency{
				Key:                       fullKey,
				Hash:                      hashedConcurrencyKeyExpr,
				Limit:                     123,
				UnhashedEvaluatedKeyValue: unhashedValue,
			}

			_, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
					return enqueueToBacklog
				}),
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					return osqueue.PartitionConstraintConfig{
						Concurrency: osqueue.PartitionConcurrency{
							AccountConcurrency:  123,
							FunctionConcurrency: 45,
							CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
								{
									Scope:               enums.ConcurrencyScopeFn,
									HashedKeyExpression: ckA.Hash,
									Limit:               ckA.Limit,
								},
							},
						},
					}
				}),
			)
			kg := shard.Client().kg

			item := osqueue.QueueItem{
				ID:          "test",
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						WorkflowID:  fnID,
						AccountID:   accountId,
						WorkspaceID: wsID,
					},
					QueueName: nil,
					Throttle:  nil,
					CustomConcurrencyKeys: []state.CustomConcurrency{
						ckA,
					},
				},
				QueueName:    nil,
				RefilledFrom: "fake-backlog",
				RefilledAt:   at.UnixMilli(),
			}

			fnPart := osqueue.ItemPartition(ctx, item)
			require.NotEmpty(t, fnPart.ID)

			// for simplicity, this enqueue should go directly to the partition
			enqueueToBacklog = false
			qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			now := clock.Now()
			leaseDur := 5 * time.Second

			// simulate having hit a partition concurrency limit in a previous operation,
			// without disabling validation this should cause Lease() to fail
			denies := osqueue.NewLeaseDenyList()
			denies.AddConcurrency(osqueue.NewKeyError(osqueue.ErrPartitionConcurrencyLimit, fnPart.Queue()))

			leaseID, err := shard.Lease(ctx, qi, leaseDur, now, denies, osqueue.LeaseOptionDisableConstraintChecks(true))
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := osqueue.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := osqueue.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			// key queue v2 accounting
			require.False(t, r.Exists(shadowPartitionAccountInProgressKey(shadowPartition, kg)))
			require.False(t, r.Exists(shadowPartitionInProgressKey(shadowPartition, kg)))
			require.False(t, r.Exists(backlogCustomKeyInProgress(backlog, kg, 1)))

			// expect classic partition concurrency to include item
			require.False(t, r.Exists(kg.Concurrency("account", accountId.String())))
			require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
		})
	})

	t.Run("two custom concurrency keys", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
		now := clock.Now()

		enqueueToBacklog := false
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should enqueue item to backlog", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			hashedConcurrencyKeyExpr1 := hashConcurrencyKey("event.data.userId")
			unhashedValue1 := "user1"
			scope1 := enums.ConcurrencyScopeFn
			fullKey1 := util.ConcurrencyKey(scope1, fnID, unhashedValue1)

			hashedConcurrencyKeyExpr2 := hashConcurrencyKey("event.data.orgId")
			unhashedValue2 := "org1"
			scope2 := enums.ConcurrencyScopeEnv
			fullKey2 := util.ConcurrencyKey(scope2, wsID, unhashedValue2)

			ckA := state.CustomConcurrency{
				Key:                       fullKey1,
				Hash:                      hashedConcurrencyKeyExpr1,
				Limit:                     123,
				UnhashedEvaluatedKeyValue: unhashedValue1,
			}
			ckB := state.CustomConcurrency{
				Key:                       fullKey2,
				Hash:                      hashedConcurrencyKeyExpr2,
				Limit:                     234,
				UnhashedEvaluatedKeyValue: unhashedValue2,
			}

			_, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
					return enqueueToBacklog
				}),
				osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					return osqueue.PartitionConstraintConfig{
						Concurrency: osqueue.PartitionConcurrency{
							AccountConcurrency:  123,
							FunctionConcurrency: 45,
							CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
								{
									Scope:               enums.ConcurrencyScopeFn,
									HashedKeyExpression: ckA.Hash,
									Limit:               ckA.Limit,
								},
								{
									Scope:               enums.ConcurrencyScopeEnv,
									HashedKeyExpression: ckB.Hash,
									Limit:               ckB.Limit,
								},
							},
						},
					}
				}),
			)
			kg := shard.Client().kg

			item := osqueue.QueueItem{
				ID:          "test",
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						WorkflowID:  fnID,
						AccountID:   accountId,
						WorkspaceID: wsID,
					},
					QueueName: nil,
					Throttle:  nil,
					CustomConcurrencyKeys: []state.CustomConcurrency{
						ckA,
						ckB,
					},
				},
				QueueName: nil,
			}

			fnPart := osqueue.ItemPartition(ctx, item)
			require.NotEmpty(t, fnPart.ID)

			// for simplicity, this enqueue should go directly to the partition
			enqueueToBacklog = false
			qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			backlog := osqueue.ItemBacklog(ctx, item)

			now := clock.Now()
			leaseDur := 5 * time.Second

			// simulate having hit a partition concurrency limit in a previous operation,
			// without disabling validation this should cause Lease() to fail
			denies := osqueue.NewLeaseDenyList()
			denies.AddConcurrency(osqueue.NewKeyError(osqueue.ErrPartitionConcurrencyLimit, backlog.CustomConcurrencyKeyID(2)))

			leaseID, err := shard.Lease(ctx, qi, leaseDur, now, denies, osqueue.LeaseOptionDisableConstraintChecks(true))
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			require.Len(t, backlog.ConcurrencyKeys, 2)

			shadowPartition := osqueue.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			// key queue v2 accounting
			require.False(t, r.Exists(shadowPartitionAccountInProgressKey(shadowPartition, kg)))
			require.False(t, r.Exists(shadowPartitionInProgressKey(shadowPartition, kg)))

			// first key
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope1, fnID, unhashedValue1)), backlogCustomKeyInProgress(backlog, kg, 1))
			require.False(t, r.Exists(backlogCustomKeyInProgress(backlog, kg, 1)))

			// second key
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope2, wsID, unhashedValue2)), backlogCustomKeyInProgress(backlog, kg, 2))
			require.False(t, r.Exists(backlogCustomKeyInProgress(backlog, kg, 2)))

			// expect classic partition concurrency to include item
			require.False(t, r.Exists(kg.Concurrency("account", accountId.String())))
			require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
		})
	})

	t.Run("system queues", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
		now := clock.Now()

		enqueueToBacklog := false
		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute).Truncate(time.Minute)

		t.Run("should lease item", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			sysQueueName := osqueue.KindQueueMigrate

			item1 := osqueue.QueueItem{
				ID: "test",
				Data: osqueue.Item{
					Kind:                  osqueue.KindEdge,
					Identifier:            state.Identifier{},
					QueueName:             &sysQueueName,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: &sysQueueName,
			}

			fnPart := osqueue.ItemPartition(ctx, item1)
			require.True(t, fnPart.IsSystem())

			// for simplicity, this enqueue should go directly to the partition
			enqueueToBacklog = false
			qi, err := shard.EnqueueItem(ctx, item1, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			now := clock.Now()
			leaseDur := 5 * time.Second

			// simulate having hit a partition concurrency limit in a previous operation,
			// without disabling validation this should cause Lease() to fail
			denies := osqueue.NewLeaseDenyList()
			denies.AddConcurrency(osqueue.NewKeyError(osqueue.ErrPartitionConcurrencyLimit, fnPart.Queue()))

			leaseID, err := shard.Lease(ctx, qi, leaseDur, now, denies, osqueue.LeaseOptionDisableConstraintChecks(true))
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := osqueue.ItemBacklog(ctx, item1)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := osqueue.ItemShadowPartition(ctx, item1)
			require.NotEmpty(t, shadowPartition.PartitionID)

			// key queue v2 accounting
			// should not track account concurrency for system partition
			require.False(t, r.Exists(shadowPartitionAccountInProgressKey(shadowPartition, kg)))
			require.False(t, r.Exists(shadowPartitionInProgressKey(shadowPartition, kg)))

			// expect classic partition concurrency to include item
			require.False(t, r.Exists(kg.Concurrency("p", sysQueueName)))
			require.False(t, r.Exists(partitionConcurrencyKey(fnPart, kg)))
		})
	})
}

type testRolloutManager struct{}

// Acquire implements constraintapi.RolloutManager.
func (t *testRolloutManager) Acquire(ctx context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	panic("unimplemented")
}

// Check implements constraintapi.RolloutManager.
func (t *testRolloutManager) Check(ctx context.Context, req *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	panic("unimplemented")
}

// ExtendLease implements constraintapi.RolloutManager.
func (t *testRolloutManager) ExtendLease(ctx context.Context, req *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	panic("unimplemented")
}

// Release implements constraintapi.RolloutManager.
func (t *testRolloutManager) Release(ctx context.Context, req *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	panic("unimplemented")
}

func TestQueueLeaseConstraintIdempotency(t *testing.T) {
	t.Run("should skip constraint updates when disabled", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		clock := clockwork.NewFakeClock()

		var cm constraintapi.CapacityManager = &testRolloutManager{}
		rolloutManager := constraintapi.NewRolloutManager(cm, QueueDefaultKey, "rl")

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Throttle: &osqueue.PartitionThrottle{
						Limit:                     1,
						Period:                    5,
						ThrottleKeyExpressionHash: "throttle-expr-key",
					},
				}
			}),

			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			osqueue.WithCapacityManager(rolloutManager),
		)

		kg := shard.Client().kg
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
				Throttle: &osqueue.Throttle{
					Period:            5,
					Limit:             1,
					Key:               "throttle-key",
					KeyExpressionHash: "throttle-expr-key",
				},
			},
		}

		start := time.Now().Truncate(time.Second)

		t.Run("constraint state should be set when not skipping", func(t *testing.T) {
			r.FlushAll()

			item, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			leaseID, err := shard.Lease(ctx, item, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			require.True(t, r.Exists(kg.ThrottleKey(qi.Data.Throttle)))
			require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
			require.True(t, r.Exists(kg.Concurrency("account", accountID.String())))
		})

		t.Run("constraint state should not be set when skipped", func(t *testing.T) {
			r.FlushAll()

			item, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			leaseID, err := shard.Lease(ctx, item, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(true))
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			require.False(t, r.Exists(kg.ThrottleKey(qi.Data.Throttle)))
			require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
			require.False(t, r.Exists(kg.Concurrency("account", accountID.String())))
		})
	})

	t.Run("should skip gcra when constraint check idempotency key is set", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		clock := clockwork.NewFakeClock()

		var cm constraintapi.CapacityManager = &testRolloutManager{}
		rolloutManager := constraintapi.NewRolloutManager(cm, QueueDefaultKey, "rl")

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Throttle: &osqueue.PartitionThrottle{
						Limit:                     1,
						Period:                    5,
						ThrottleKeyExpressionHash: "throttle-expr-key",
					},
				}
			}),

			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			osqueue.WithCapacityManager(rolloutManager),
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
				Throttle: &osqueue.Throttle{
					Period:            5,
					Limit:             1,
					Key:               "throttle-key",
					KeyExpressionHash: "throttle-expr-key",
				},
			},
		}

		start := time.Now().Truncate(time.Second)

		item, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// First call should succeed - Use up all capacity
		leaseID, err := shard.Lease(ctx, item, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		clock.Advance(time.Second)
		r.FastForward(time.Second)
		r.SetTime(clock.Now())

		item2, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Second call should fail - Capacity all used up
		leaseID, err = shard.Lease(ctx, item2, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
		require.Error(t, err)
		require.ErrorIs(t, err, osqueue.ErrQueueItemThrottled)
		require.Nil(t, leaseID)

		// Set idempotency key
		keyConstraintCheckIdempotency := rolloutManager.KeyConstraintCheckIdempotency(constraintapi.MigrationIdentifier{
			QueueShard: shard.Name(),
		}, accountID, item2.ID)

		err = r.Set(keyConstraintCheckIdempotency, strconv.Itoa(int(clock.Now().UnixMilli())))
		require.NoError(t, err)

		// Do not skip lease checks but handle idempotency
		leaseID, err = shard.Lease(ctx, item2, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		// Skip all checks
		item3, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		leaseID, err = shard.Lease(ctx, item3, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(true))
		require.NoError(t, err)
		require.NotNil(t, leaseID)
	})
}
