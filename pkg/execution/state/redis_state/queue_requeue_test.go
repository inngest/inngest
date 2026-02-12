package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
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

func TestQueueRequeue(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	_, shard := newQueue(t, rc,
		osqueue.WithClock(clock),
	)
	kg := shard.Client().kg
	ctx := context.Background()

	t.Run("Re-enqueuing a leased item should succeed", func(t *testing.T) {
		now := time.Now()

		fnID := uuid.New()
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					RunID:      runID,
					WorkflowID: fnID,
				},
			},
		}, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = shard.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		// Assert partition index is original
		pi := osqueue.QueuePartition{FunctionID: &item.FunctionID}
		requirePartitionScoreEquals(t, r, pi.FunctionID, now.Truncate(time.Second))

		requirePartitionInProgress(t, shard, item.FunctionID, 1)

		next := now.Add(time.Hour)
		err = shard.Requeue(ctx, item, next)
		require.NoError(t, err)

		t.Run("It should re-enqueue the item with the future time", func(t *testing.T) {
			requireItemScoreEquals(t, r, item, next)
		})

		t.Run("It should always remove the lease from the re-enqueued item", func(t *testing.T) {
			fetched := getQueueItem(t, r, item.ID)
			require.Nil(t, fetched.LeaseID)
		})

		t.Run("It should decrease the in-progress count", func(t *testing.T) {
			requirePartitionInProgress(t, shard, item.FunctionID, 0)
		})

		t.Run("It should update the partition's earliest time, if earliest", func(t *testing.T) {
			// Assert partition index is updated, as there's only one item here.
			requirePartitionScoreEquals(t, r, pi.FunctionID, next)
		})

		t.Run("run indexes are updated on requeue to partition", func(t *testing.T) {
			require.False(t, r.Exists(kg.ActiveRunsSet("p", item.FunctionID.String())))
			require.False(t, r.Exists(kg.ActiveSet("run", runID.String())))
		})

		t.Run("It should not update the partition's earliest time, if later", func(t *testing.T) {
			_, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					Identifier: state.Identifier{
						RunID:      runID,
						WorkflowID: fnID,
					},
				},
			}, now, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			requirePartitionScoreEquals(t, r, pi.FunctionID, now)

			next := now.Add(2 * time.Hour)
			err = shard.Requeue(ctx, item, next)
			require.NoError(t, err)

			requirePartitionScoreEquals(t, r, pi.FunctionID, now)
		})

		t.Run("Updates default indexes", func(t *testing.T) {
			at := time.Now().Truncate(time.Second)
			rid := ulid.MustNew(ulid.Now(), rand.Reader)
			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: uuid.New(),
				Data: osqueue.Item{
					Kind: osqueue.KindEdge,
					Identifier: state.Identifier{
						RunID: rid,
					},
				},
			}, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			key := fmt.Sprintf("{queue}:idx:run:%s", rid)

			keys, err := r.ZMembers(key)
			require.NoError(t, err)
			require.Equal(t, 1, len(keys))

			// Score for entry should be the first enqueue time.
			scores, err := r.ZMScore(key, keys[0])
			require.NoError(t, err)
			require.EqualValues(t, at.UnixMilli(), scores[0])

			next := now.Add(2 * time.Hour)
			err = shard.Requeue(ctx, item, next)
			require.NoError(t, err)

			// Score should be the requeue time.
			scores, err = r.ZMScore(key, keys[0])
			require.NoError(t, err)
			require.EqualValues(t, next.UnixMilli(), scores[0])

			// Still only one member.
			keys, err = r.ZMembers(key)
			require.NoError(t, err)
			require.Equal(t, 1, len(keys))
		})
	})

	t.Run("For a queue item with concurrency keys it requeues all partitions", func(t *testing.T) {
		r.FlushAll()

		fnID, acctID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("fn")),
			uuid.NewSHA1(uuid.NameSpaceDNS, []byte("acct"))

		now := time.Now()
		item := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					AccountID: acctID,
				},
				CustomConcurrencyKeys: []state.CustomConcurrency{
					{
						Key: util.ConcurrencyKey(
							enums.ConcurrencyScopeAccount,
							acctID,
							"test-plz",
						),
						Limit: 5,
					},
					{
						Key: util.ConcurrencyKey(
							enums.ConcurrencyScopeFn,
							fnID,
							"another-id",
						),
						Limit: 2,
					},
				},
			},
		}
		item, err := shard.EnqueueItem(ctx, item, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		fnPart := osqueue.ItemPartition(ctx, item)
		backlog := osqueue.ItemBacklog(ctx, item)

		// Get all scores
		require.False(t, r.Exists(backlogCustomKeyInProgress(backlog, kg, 1)))
		require.False(t, r.Exists(backlogCustomKeyInProgress(backlog, kg, 2)))
		itemScoreDefault, _ := r.ZMScore(partitionZsetKey(fnPart, kg), item.ID)
		partScoreDefault, _ := r.ZMScore(kg.GlobalPartitionIndex(), fnPart.ID)
		accountPartScore, _ := r.ZMScore(kg.AccountPartitionIndex(acctID), fnPart.ID)
		accountScore, _ := r.ZMScore(kg.GlobalAccountIndex(), acctID.String())

		require.NotEmpty(t, itemScoreDefault)
		require.NotEmpty(t, partScoreDefault)
		require.Equal(t, partScoreDefault, accountPartScore, "expected account partitions to match global partitions")
		require.Equal(t, accountPartScore[0], accountScore[0], "expected account score to match earliest account partition")

		_, err = shard.Lease(ctx, item, time.Second, clock.Now(), nil)
		require.NoError(t, err)

		// Requeue
		next := now.Add(time.Hour)
		err = shard.Requeue(ctx, item, next)
		require.NoError(t, err)

		t.Run("It requeues all partitions", func(t *testing.T) {
			newItemScore, _ := r.ZMScore(partitionZsetKey(fnPart, kg), item.ID)
			newPartScore, _ := r.ZMScore(kg.GlobalPartitionIndex(), fnPart.ID)
			newAccountPartScore, _ := r.ZMScore(kg.AccountPartitionIndex(acctID), fnPart.ID)
			newAccountScore, _ := r.ZMScore(kg.GlobalAccountIndex(), acctID.String())

			require.NotEqual(t, itemScoreDefault, newItemScore)
			require.NotEqual(t, partScoreDefault, newPartScore)
			require.Equal(t, newPartScore, newAccountPartScore)
			require.Equal(t, newPartScore, newAccountPartScore)
			require.Equal(t, next.Truncate(time.Second).Unix(), int64(newPartScore[0]))
			require.Equal(t, newAccountPartScore[0], newAccountScore[0], "expected account score to match earliest account partition", r.Dump())
			require.EqualValues(t, next.UnixMilli(), int(newItemScore[0]))
			require.EqualValues(t, next.Unix(), int(newPartScore[0]))
		})
	})
}

func TestQueueRequeueToBacklog(t *testing.T) {
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
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should requeue item to backlog", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

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
						RunID:       runID,
					},
					QueueName:             nil,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: nil,
			}

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// put item in progress, this is tested separately
			now := clock.Now()
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := shard.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := osqueue.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := osqueue.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			fnPart := osqueue.ItemPartition(ctx, item)
			require.NotEmpty(t, fnPart.ID)

			require.False(t, hasMember(t, r, partitionZsetKey(fnPart, kg), qi.ID))

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartitionAccountInProgressKey(shadowPartition, kg), qi.ID)))

			// no active set for default partition since this uses the in progress key
			require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 1))
			require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 2))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, partitionConcurrencyKey(fnPart, kg), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			itemIsMember, err := r.SIsMember(kg.ActiveSet("run", runID.String()), qi.ID)
			require.NoError(t, err)
			require.True(t, itemIsMember)

			isMember, err := r.SIsMember(kg.ActiveRunsSet("p", fnID.String()), runID.String())
			require.NoError(t, err)
			require.True(t, isMember)

			requeueFor := at.Add(30 * time.Minute).Truncate(time.Minute)

			require.False(t, r.Exists(kg.GlobalAccountShadowPartitions()))
			require.False(t, r.Exists(kg.AccountShadowPartitions(accountId)))

			err = shard.Requeue(ctx, qi, requeueFor)
			require.NoError(t, err)

			// expect item to be requeued to backlog
			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))
			require.True(t, r.Exists(kg.GlobalAccountShadowPartitions()))
			require.True(t, r.Exists(kg.AccountShadowPartitions(accountId)))

			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountId.String())))
			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountId), shadowPartition.PartitionID)))

			// expect key queue accounting to be updated
			remainingMembers, _ := r.ZMembers(shadowPartitionInProgressKey(shadowPartition, kg))

			require.False(t, hasMember(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID), remainingMembers)
			require.False(t, hasMember(t, r, shadowPartitionAccountInProgressKey(shadowPartition, kg), qi.ID))

			require.False(t, r.Exists(kg.ActiveSet("run", runID.String())))
			require.False(t, r.Exists(kg.ActiveRunsSet("p", fnID.String())))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, partitionConcurrencyKey(fnPart, kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, partitionZsetKey(fnPart, kg), qi.ID), r.Keys())
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
		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should requeue item to backlog", func(t *testing.T) {
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
				osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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
				QueueName: nil,
			}

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// sanity check: empty key should never be stored
			require.False(t, r.Exists(kg.Concurrency("", "")))

			// put item in progress, this is tested separately
			now := clock.Now().Truncate(time.Minute)
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := shard.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)
			require.Equal(t, leaseExpires, ulid.Time(leaseID.Time()), now)

			backlog := osqueue.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)
			require.Equal(t, enums.ConcurrencyScopeFn, backlog.ConcurrencyKeys[0].Scope)
			require.NotEmpty(t, backlog.ConcurrencyKeys[0].HashedKeyExpression)
			require.NotEmpty(t, backlog.ConcurrencyKeys[0].EntityID)

			shadowPartition := osqueue.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			fnPart := osqueue.ItemPartition(ctx, item)

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartitionAccountInProgressKey(shadowPartition, kg), qi.ID)))

			// 1 active set for custom concurrency key
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope, fnID, unhashedValue)), backlogCustomKeyInProgress(backlog, kg, 1))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlogCustomKeyInProgress(backlog, kg, 1), qi.ID)))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("account", accountId.String()), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), qi.ID)), r.Keys())
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("custom", fullKey), qi.ID)))

			// sanity check: empty key should never be stored
			require.False(t, r.Exists(kg.Concurrency("", "")))

			requeueFor := at.Add(30 * time.Minute).Truncate(time.Minute)

			err = shard.Requeue(ctx, qi, requeueFor)
			require.NoError(t, err)

			// sanity check: empty key should never be stored
			require.False(t, r.Exists(kg.Concurrency("", "")))

			// expect item to be requeued to backlog
			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))

			// expect key queue accounting to be updated
			remainingMembers, _ := r.ZMembers(shadowPartitionInProgressKey(shadowPartition, kg))

			require.False(t, hasMember(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID), remainingMembers)
			require.False(t, hasMember(t, r, shadowPartitionAccountInProgressKey(shadowPartition, kg), qi.ID))
			require.False(t, hasMember(t, r, backlogCustomKeyInProgress(backlog, kg, 1), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, partitionZsetKey(fnPart, kg), qi.ID))
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
		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should requeue item to backlog", func(t *testing.T) {
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
				osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// sanity check: empty key should never be stored
			require.False(t, r.Exists(kg.Concurrency("", "")))

			// put item in progress, this is tested separately
			now := clock.Now().Truncate(time.Minute)
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := shard.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)
			require.Equal(t, leaseExpires, ulid.Time(leaseID.Time()), now)

			backlog := osqueue.ItemBacklog(ctx, item)
			require.Len(t, backlog.ConcurrencyKeys, 2)
			require.NotEmpty(t, backlog.ConcurrencyKeys[0].HashedKeyExpression)
			require.Equal(t, enums.ConcurrencyScopeFn, backlog.ConcurrencyKeys[0].Scope)
			require.NotEmpty(t, backlog.ConcurrencyKeys[0].EntityID)

			require.NotEmpty(t, backlog.ConcurrencyKeys[1].HashedKeyExpression)
			require.NotEmpty(t, backlog.ConcurrencyKeys[1].Scope)
			require.NotEmpty(t, backlog.ConcurrencyKeys[1].EntityID)

			shadowPartition := osqueue.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			fnPart := osqueue.ItemPartition(ctx, item)
			require.NotEmpty(t, fnPart.ID)

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartitionAccountInProgressKey(shadowPartition, kg), qi.ID)))

			// 2 active set for custom concurrency keys

			// first key
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope1, fnID, unhashedValue1)), backlogCustomKeyInProgress(backlog, kg, 1))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlogCustomKeyInProgress(backlog, kg, 1), qi.ID)))

			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope2, wsID, unhashedValue2)), backlogCustomKeyInProgress(backlog, kg, 2))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlogCustomKeyInProgress(backlog, kg, 2), qi.ID)))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("account", accountId.String()), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), qi.ID)), r.Keys())

			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("custom", fullKey1), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, kg.Concurrency("custom", fullKey2), qi.ID)))

			// sanity check: empty key should never be stored
			require.False(t, r.Exists(kg.Concurrency("", "")))

			requeueFor := at.Add(30 * time.Minute).Truncate(time.Minute)

			err = shard.Requeue(ctx, qi, requeueFor)
			require.NoError(t, err)

			// sanity check: empty key should never be stored
			require.False(t, r.Exists(kg.Concurrency("", "")))

			// expect item to be requeued to backlog
			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))

			// expect key queue accounting to be updated
			remainingMembers, _ := r.ZMembers(shadowPartitionInProgressKey(shadowPartition, kg))

			require.False(t, hasMember(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID), remainingMembers)
			require.False(t, hasMember(t, r, shadowPartitionAccountInProgressKey(shadowPartition, kg), qi.ID))

			require.False(t, hasMember(t, r, backlogCustomKeyInProgress(backlog, kg, 1), qi.ID))
			require.False(t, hasMember(t, r, backlogCustomKeyInProgress(backlog, kg, 2), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey1), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey2), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, partitionZsetKey(fnPart, kg), qi.ID))
		})
	})

	t.Run("system queues", func(t *testing.T) {
		t.Skip("system queues are not enqueued to backlogs")
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
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			// WithEnqueueSystemPartitionsToBacklog(true),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		sysQueueName := osqueue.KindQueueMigrate

		t.Run("should requeue item to backlog", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			item := osqueue.QueueItem{
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

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// put item in progress, this is tested separately
			now := clock.Now()
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := shard.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := osqueue.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := osqueue.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			fnPart := osqueue.ItemPartition(ctx, item)
			require.NotEmpty(t, fnPart.ID)
			require.True(t, fnPart.IsSystem())

			require.False(t, hasMember(t, r, partitionZsetKey(fnPart, kg), qi.ID))

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID)))

			require.Equal(t, kg.Concurrency("account", ""), shadowPartitionAccountInProgressKey(shadowPartition, kg))
			require.False(t, r.Exists(shadowPartitionAccountInProgressKey(shadowPartition, kg)))

			// no active set for default partition since this uses the in progress key
			require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 1))
			require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 2))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, partitionConcurrencyKey(fnPart, kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", fnPart.Queue()), qi.ID)) // pseudo-limit for system qeueus
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			requeueFor := at.Add(30 * time.Minute).Truncate(time.Minute)

			err = shard.Requeue(ctx, qi, requeueFor)
			require.NoError(t, err)

			// expect item to be requeued to backlog
			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))

			// expect key queue accounting to be updated
			remainingMembers, _ := r.ZMembers(shadowPartitionInProgressKey(shadowPartition, kg))

			require.False(t, hasMember(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID), remainingMembers)
			require.False(t, hasMember(t, r, shadowPartitionAccountInProgressKey(shadowPartition, kg), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, partitionConcurrencyKey(fnPart, kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", fnPart.Queue()), qi.ID)) // pseudo-limit for system queues
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, partitionZsetKey(fnPart, kg), qi.ID))
		})
	})

	t.Run("don't update run indexes if another queue item is active", func(t *testing.T) {
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
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		require.Len(t, r.Keys(), 0)

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
					RunID:       runID,
				},
				QueueName:             nil,
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
			},
			QueueName: nil,
		}

		//
		// Add two queue items to in progress
		//

		// directly enqueue to partition
		enqueueToBacklog = false
		qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item.ID = ""
		qi2, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		enqueueToBacklog = true

		// put item in progress, this is tested separately
		leaseDur := 5 * time.Second
		leaseID, err := shard.Lease(ctx, qi, leaseDur, now, nil)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		leaseID, err = shard.Lease(ctx, qi2, leaseDur, now, nil)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		backlog := osqueue.ItemBacklog(ctx, item)
		require.NotEmpty(t, backlog.BacklogID)

		shadowPartition := osqueue.ItemShadowPartition(ctx, item)
		require.NotEmpty(t, shadowPartition.PartitionID)

		itemIsMember, err := r.SIsMember(kg.ActiveSet("run", runID.String()), qi.ID)
		require.NoError(t, err)
		require.True(t, itemIsMember)

		itemIsMember, err = r.SIsMember(kg.ActiveSet("run", runID.String()), qi2.ID)
		require.NoError(t, err)
		require.True(t, itemIsMember)

		isMember, err := r.SIsMember(kg.ActiveRunsSet("p", fnID.String()), runID.String())
		require.NoError(t, err)
		require.True(t, isMember)

		//
		// Requeue first active item, expect active run items to be updated (decreased by 1)
		// but run still has another active item
		//

		requeueFor := at.Add(30 * time.Minute).Truncate(time.Minute)

		err = shard.Requeue(ctx, qi, requeueFor)
		require.NoError(t, err)

		itemIsMember, err = r.SIsMember(kg.ActiveSet("run", runID.String()), qi.ID)
		require.NoError(t, err)
		require.False(t, itemIsMember)

		itemIsMember, err = r.SIsMember(kg.ActiveSet("run", runID.String()), qi2.ID)
		require.NoError(t, err)
		require.True(t, itemIsMember)

		isMember, err = r.SIsMember(kg.ActiveRunsSet("p", fnID.String()), runID.String())
		require.NoError(t, err)
		require.True(t, isMember)

		//
		// Requeue final active item, expect indexes to be cleared out
		//

		err = shard.Requeue(ctx, qi2, requeueFor.Add(time.Hour))
		require.NoError(t, err)

		runSetExists := r.Exists(kg.ActiveSet("run", runID.String()))
		require.False(t, runSetExists)
		require.False(t, r.Exists(kg.ActiveRunsSet("p", fnID.String())))
	})

	t.Run("item without throttle key expression should be backfilled", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
		now := clock.Now()

		oldThrottle := &osqueue.Throttle{
			Key:                 util.XXHash("old"),
			Limit:               10,
			Period:              60,
			UnhashedThrottleKey: "old",
			// Test: Do not store expression hash yet!
			// KeyExpressionHash:   util.XXHash("old-hash"),
		}
		newThrottle := &osqueue.Throttle{
			Key:                 util.XXHash("new"),
			Limit:               10,
			Period:              60,
			UnhashedThrottleKey: "new",
			KeyExpressionHash:   util.XXHash("new-hash"),
		}

		enqueueToBacklog := false
		var refreshCalled bool
		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			osqueue.WithRefreshItemThrottle(func(ctx context.Context, item *osqueue.QueueItem) (*osqueue.Throttle, error) {
				refreshCalled = true
				return newThrottle, nil
			}),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should requeue item to backlog", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			item := osqueue.QueueItem{
				ID:          "test",
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindStart,
					Identifier: state.Identifier{
						WorkflowID:  fnID,
						AccountID:   accountId,
						WorkspaceID: wsID,
						RunID:       runID,
					},
					QueueName:             nil,
					Throttle:              oldThrottle,
					CustomConcurrencyKeys: nil,
				},
				QueueName: nil,
			}

			oldBacklog := osqueue.ItemBacklog(ctx, item)

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			require.False(t, refreshCalled)

			require.False(t, hasMember(t, r, kg.BacklogSet(oldBacklog.BacklogID), qi.ID))

			shadowPartition := osqueue.ItemShadowPartition(ctx, item)

			fnPart := osqueue.ItemPartition(ctx, item)

			require.True(t, hasMember(t, r, partitionZsetKey(fnPart, kg), qi.ID), r.Keys())

			requeueFor := at.Add(30 * time.Minute).Truncate(time.Minute)

			err = shard.Requeue(ctx, qi, requeueFor)
			require.NoError(t, err)

			item.Data.Throttle = newThrottle
			newBacklog := osqueue.ItemBacklog(ctx, item)

			require.True(t, refreshCalled)

			require.False(t, hasMember(t, r, partitionZsetKey(fnPart, kg), qi.ID), r.Keys())
			require.False(t, hasMember(t, r, kg.BacklogSet(oldBacklog.BacklogID), qi.ID))
			require.True(t, hasMember(t, r, kg.BacklogSet(newBacklog.BacklogID), qi.ID))

			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.BacklogSet(newBacklog.BacklogID), qi.ID)))
			require.True(t, r.Exists(kg.GlobalAccountShadowPartitions()))
			require.True(t, r.Exists(kg.AccountShadowPartitions(accountId)))

			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountId.String())))
			require.Equal(t, requeueFor.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountId), shadowPartition.PartitionID)))

			var requeuedItem osqueue.QueueItem
			queueItemStr := r.HGet(kg.QueueItem(), qi.ID)
			require.NotEmpty(t, queueItemStr)
			require.NoError(t, json.Unmarshal([]byte(queueItemStr), &requeuedItem))

			require.NotNil(t, requeuedItem.Data.Throttle, queueItemStr)
			require.Equal(t, newThrottle.KeyExpressionHash, requeuedItem.Data.Throttle.KeyExpressionHash)
		})
	})

	t.Run("requeue should remove item from ready queue", func(t *testing.T) {
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
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		// use future timestamp because scores will be bounded to the present
		at := now.Add(10 * time.Minute)

		t.Run("should requeue item to backlog", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

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
						RunID:       runID,
					},
					QueueName:             nil,
					Throttle:              nil,
					CustomConcurrencyKeys: nil,
				},
				QueueName: nil,
			}

			// directly enqueue to partition
			enqueueToBacklog = false
			qi, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			backlog := osqueue.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := osqueue.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			fnPart := osqueue.ItemPartition(ctx, item)
			require.NotEmpty(t, fnPart.ID)

			require.True(t, hasMember(t, r, partitionZsetKey(fnPart, kg), qi.ID))

			requeueAt := clock.Now()

			err = shard.Requeue(ctx, qi, requeueAt)
			require.NoError(t, err)

			// expect item to be requeued to backlog
			require.Equal(t, requeueAt.UnixMilli(), int64(score(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID)))
			require.True(t, r.Exists(kg.GlobalAccountShadowPartitions()))
			require.True(t, r.Exists(kg.AccountShadowPartitions(accountId)))

			require.Equal(t, requeueAt.UnixMilli(), int64(score(t, r, kg.GlobalAccountShadowPartitions(), accountId.String())))
			require.Equal(t, requeueAt.UnixMilli(), int64(score(t, r, kg.AccountShadowPartitions(accountId), shadowPartition.PartitionID)))

			require.False(t, hasMember(t, r, partitionZsetKey(fnPart, kg), qi.ID), r.Keys())
		})
	})
}

func TestQueueRequeueWithDisabledConstraintUpdates(t *testing.T) {
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
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return true
		}),
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
		},
	}

	start := time.Now().Truncate(time.Second)

	item, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	// Lease item in new mode (skip checks)
	leaseID, err := shard.Lease(ctx, item, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(true))
	require.NoError(t, err)
	require.NotNil(t, leaseID)

	require.Equal(t, clock.Now().Add(5*time.Second).UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item.ID)))
	require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
	require.False(t, r.Exists(kg.Concurrency("account", accountID.String())))

	err = shard.Requeue(ctx, item, clock.Now().Add(time.Minute), osqueue.RequeueOptionDisableConstraintUpdates(true))
	require.NoError(t, err)

	require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
	require.False(t, r.Exists(kg.Concurrency("account", accountID.String())))
	require.False(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
}
