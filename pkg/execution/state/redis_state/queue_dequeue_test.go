package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueueDequeueUpdateAccounting(t *testing.T) {
	t.Run("simple item", func(t *testing.T) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		clock := clockwork.NewFakeClock()
		enqueueToBacklog := false
		_, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := time.Now().Add(10 * time.Minute)

		t.Run("should dequeue item and update accounting", func(t *testing.T) {
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

			err = shard.Dequeue(ctx, qi)
			require.NoError(t, err)

			// expect item not to be requeued
			require.False(t, hasMember(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID))

			// expect key queue accounting to be updated
			require.False(t, hasMember(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID))
			require.False(t, hasMember(t, r, shadowPartitionAccountInProgressKey(shadowPartition, kg), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, partitionConcurrencyKey(fnPart, kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, partitionZsetKey(fnPart, kg), qi.ID))
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

		enqueueToBacklog := false
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := time.Now().Add(10 * time.Minute)

		t.Run("should dequeue item and update accounting", func(t *testing.T) {
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

			clock := clockwork.NewFakeClock()
			_, shard := newQueue(
				t, rc,
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
				osqueue.WithClock(clock),
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

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartitionAccountInProgressKey(shadowPartition, kg), qi.ID)))

			// 1 active set for custom concurrency key
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope, fnID, unhashedValue)), backlogCustomKeyInProgress(backlog, kg, 1))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlogCustomKeyInProgress(backlog, kg, 1), qi.ID)))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), qi.ID), r.Keys())
			require.True(t, hasMember(t, r, kg.Concurrency("custom", fullKey), qi.ID))

			err = shard.Dequeue(ctx, qi)
			require.NoError(t, err)

			// expect item not to be requeued
			require.False(t, hasMember(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID))

			// expect key queue accounting to be updated
			require.False(t, hasMember(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID))
			require.False(t, hasMember(t, r, shadowPartitionAccountInProgressKey(shadowPartition, kg), qi.ID))
			require.False(t, hasMember(t, r, backlogCustomKeyInProgress(backlog, kg, 1), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), qi.ID))
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

		enqueueToBacklog := false
		ctx := context.Background()

		accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

		// use future timestamp because scores will be bounded to the present
		at := time.Now().Add(10 * time.Minute)

		t.Run("should dequeue item and update accounting", func(t *testing.T) {
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

			clock := clockwork.NewFakeClock()
			_, shard := newQueue(
				t, rc,
				osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
					return enqueueToBacklog
				}),
				osqueue.WithClock(clock),
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
			require.Len(t, backlog.ConcurrencyKeys, 2)

			shadowPartition := osqueue.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			fnPart := osqueue.ItemPartition(ctx, item)
			require.NotEmpty(t, fnPart.ID)

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartitionAccountInProgressKey(shadowPartition, kg), qi.ID)))

			// 2 active set for custom concurrency keys
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope1, fnID, unhashedValue1)), backlogCustomKeyInProgress(backlog, kg, 1))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlogCustomKeyInProgress(backlog, kg, 1), qi.ID)))

			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope2, wsID, unhashedValue2)), backlogCustomKeyInProgress(backlog, kg, 2))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlogCustomKeyInProgress(backlog, kg, 2), qi.ID)))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), qi.ID), r.Keys())
			require.True(t, hasMember(t, r, kg.Concurrency("custom", fullKey1), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("custom", fullKey2), qi.ID))

			err = shard.Dequeue(ctx, qi)
			require.NoError(t, err)

			// expect item not to be requeued
			require.False(t, hasMember(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID))

			// expect key queue accounting to be updated
			require.False(t, hasMember(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID))
			require.False(t, hasMember(t, r, shadowPartitionAccountInProgressKey(shadowPartition, kg), qi.ID))
			require.False(t, hasMember(t, r, backlogCustomKeyInProgress(backlog, kg, 1), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey1), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey2), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, partitionZsetKey(fnPart, kg), qi.ID))
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

		enqueueToBacklog := false
		clock := clockwork.NewFakeClock()
		_, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		sysQueueName := osqueue.KindQueueMigrate

		// use future timestamp because scores will be bounded to the present
		at := time.Now().Add(10 * time.Minute)

		t.Run("should dequeue item and update accounting", func(t *testing.T) {
			require.Len(t, r.Keys(), 0)

			item := osqueue.QueueItem{
				ID: "test",
				Data: osqueue.Item{
					Kind:                  osqueue.KindQueueMigrate,
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

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID)))

			require.Equal(t, kg.Concurrency("account", ""), shadowPartitionAccountInProgressKey(shadowPartition, kg))
			require.False(t, r.Exists(shadowPartitionAccountInProgressKey(shadowPartition, kg)))

			// no active set for default partition since this uses the in progress key
			require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 1))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, partitionConcurrencyKey(fnPart, kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", fnPart.Queue()), qi.ID)) // pseudo-limit for system queue
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			err = shard.Dequeue(ctx, qi)
			require.NoError(t, err)

			// expect item not to be requeued
			require.False(t, hasMember(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID))

			// expect key queue accounting to be updated
			require.False(t, hasMember(t, r, shadowPartitionInProgressKey(shadowPartition, kg), qi.ID))
			require.False(t, hasMember(t, r, shadowPartitionAccountInProgressKey(shadowPartition, kg), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, partitionConcurrencyKey(fnPart, kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", fnPart.Queue()), qi.ID)) // pseudo-limit for system queue
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, partitionZsetKey(fnPart, kg), qi.ID))
		})
	})
}

func TestQueueDequeue(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()
	_, shard := newQueue(t, rc, osqueue.WithClock(clock))
	ctx := context.Background()
	kg := shard.Client().kg

	t.Run("It always changes global partition scores", func(t *testing.T) {
		r.FlushAll()

		fnID, acctID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("fn")),
			uuid.NewSHA1(uuid.NameSpaceDNS, []byte("acct"))

		start := time.Now().Truncate(time.Second)

		// Enqueue two items to the same function
		itemA, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
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
							"acct-id",
						),
						Limit: 10,
					},
					{
						Key: util.ConcurrencyKey(
							enums.ConcurrencyScopeFn,
							fnID,
							"fn-id",
						),
						Limit: 5,
					},
				},
			},
		}, start, osqueue.EnqueueOpts{})
		require.Nil(t, err)
		_, err = shard.EnqueueItem(ctx, osqueue.QueueItem{
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
							"acct-id",
						),
						Limit: 10,
					},
					{
						Key: util.ConcurrencyKey(
							enums.ConcurrencyScopeFn,
							fnID,
							"fn-id",
						),
						Limit: 5,
					},
				},
			},
		}, start, osqueue.EnqueueOpts{})
		require.Nil(t, err)

		// First 2 partitions will be custom, third one default
		fnPart := osqueue.ItemPartition(ctx, itemA)
		require.NotEmpty(t, fnPart.ID)

		// Lease the first item, pretending it's in progress.
		_, err = shard.Lease(ctx, itemA, 10*time.Second, clock.Now(), nil)
		require.NoError(t, err)

		// Note: Originally, this test used the concurrency key queue for testing Dequeue(),
		// but this was changed to the default partition, as we do not enqueue to key queues anymore.
		partitionToDequeue := fnPart

		// Force requeue the partition such that it's pushed forward, pretending there's
		// no capacity.
		err = shard.PartitionRequeue(ctx, &partitionToDequeue, start.Add(30*time.Minute), true)
		require.NoError(t, err)

		t.Run("Requeueing partitions updates the score", func(t *testing.T) {
			partScoreA, _ := r.ZMScore(kg.GlobalPartitionIndex(), partitionToDequeue.ID)
			require.EqualValues(t, start.Add(30*time.Minute).Unix(), partScoreA[0])

			partScoreA, _ = r.ZMScore(kg.AccountPartitionIndex(acctID), partitionToDequeue.ID)
			require.NotNil(t, partScoreA, "expected partition requeue to update account partition index", r.Dump())
			require.EqualValues(t, start.Add(30*time.Minute).Unix(), partScoreA[0])
		})

		// Dequeue to pull partition back to now
		err = shard.Dequeue(ctx, itemA)
		require.Nil(t, err)

		t.Run("The outstanding partition scores should reset", func(t *testing.T) {
			partScoreA, _ := r.ZMScore(kg.GlobalPartitionIndex(), partitionToDequeue.ID)
			require.EqualValues(t, start, time.Unix(int64(partScoreA[0]), 0), r.Dump(), partitionToDequeue, start.UnixMilli())
		})
	})

	t.Run("with concurrency keys", func(t *testing.T) {
		start := time.Now()

		t.Run("with an unleased item", func(t *testing.T) {
			r.FlushAll()
			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: uuid.New(),
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
			}, start, osqueue.EnqueueOpts{})
			require.Nil(t, err)

			// First 2 partitions will be custom.
			fnPart := osqueue.ItemPartition(ctx, item)
			require.NotEmpty(t, fnPart.ID)

			err = shard.Dequeue(ctx, item)
			require.Nil(t, err)

			t.Run("The outstanding partition items should be empty", func(t *testing.T) {
				mem, _ := r.ZMembers(partitionZsetKey(fnPart, kg))
				require.Equal(t, 0, len(mem))
			})
		})

		t.Run("with a leased item", func(t *testing.T) {
			r.FlushAll()
			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
				FunctionID: uuid.New(),
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
			}, start, osqueue.EnqueueOpts{})
			require.Nil(t, err)

			id, err := shard.Lease(ctx, item, 10*time.Second, time.Now(), nil)
			require.NoError(t, err)
			require.NotEmpty(t, id)

			t.Run("The scavenger queue should not yet be empty", func(t *testing.T) {
				mems, err := r.ZMembers(kg.ConcurrencyIndex())
				require.NoError(t, err)
				require.NotEmpty(t, mems)
			})

			err = shard.Dequeue(ctx, item)
			require.Nil(t, err)

			t.Run("The scavenger queue should now be empty", func(t *testing.T) {
				mems, _ := r.ZMembers(kg.ConcurrencyIndex())
				require.Empty(t, mems)
			})
		})
	})

	t.Run("It should remove a queue item", func(t *testing.T) {
		r.FlushAll()

		start := time.Now()

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
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.QueuePartition{FunctionID: &item.FunctionID}

		id, err := shard.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		t.Run("The lease exists in the partition queue", func(t *testing.T) {
			count, err := shard.InProgress(ctx, "p", p.FunctionID.String())
			require.NoError(t, err)
			require.EqualValues(t, 1, count, r.Dump())
		})

		err = shard.Dequeue(ctx, item)
		require.NoError(t, err)

		t.Run("It should remove the item from the queue map", func(t *testing.T) {
			val := r.HGet(kg.QueueItem(), id.String())
			require.Empty(t, val)
		})

		t.Run("Extending a lease should fail after dequeue", func(t *testing.T) {
			id, err := shard.ExtendLease(ctx, item, *id, time.Minute)
			require.Equal(t, osqueue.ErrQueueItemNotFound, err)
			require.Nil(t, id)
		})

		t.Run("It should remove the item from the queue index", func(t *testing.T) {
			items, err := shard.Peek(ctx, &p, time.Now().Add(time.Hour), 10)
			require.NoError(t, err)
			require.EqualValues(t, 0, len(items))
		})

		t.Run("It should remove the item from the concurrency partition's queue", func(t *testing.T) {
			count, err := shard.InProgress(ctx, "p", p.FunctionID.String())
			require.NoError(t, err)
			require.EqualValues(t, 0, count)
		})

		t.Run("run indexes are updated", func(t *testing.T) {
			// Run indexes should be updated

			require.False(t, r.Exists(kg.ActiveSet("run", runID.String())))
			require.False(t, r.Exists(kg.ActiveRunsSet("p", fnID.String())))
		})

		t.Run("It should work if the item is not leased (eg. deletions)", func(t *testing.T) {
			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			err = shard.Dequeue(ctx, item)
			require.NoError(t, err)

			val := r.HGet(kg.QueueItem(), id.String())
			require.Empty(t, val)
		})

		t.Run("Removes default indexes", func(t *testing.T) {
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

			keys, err := r.ZMembers(fmt.Sprintf("{queue}:idx:run:%s", rid))
			require.NoError(t, err)
			require.Equal(t, 1, len(keys))

			err = shard.Dequeue(ctx, item)
			require.NoError(t, err)

			keys, err = r.ZMembers(fmt.Sprintf("{queue}:idx:run:%s", rid))
			require.NotNil(t, err)
			require.Equal(t, true, strings.Contains(err.Error(), "no such key"))
			require.Equal(t, 0, len(keys))
		})
	})

	t.Run("backcompat: it should not drop previous partition names from concurrency index", func(t *testing.T) {
		// This tests backwards compatibility with the old concurrency index member naming scheme
		r.FlushAll()
		start := time.Now().Truncate(time.Second)

		customQueueName := "custom-queue-name"
		item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
			FunctionID: uuid.New(),
			Data: osqueue.Item{
				QueueName: &customQueueName,
			},
			QueueName: &customQueueName,
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		fnPart := osqueue.ItemPartition(ctx, item)

		itemCountMatches := func(num int) {
			zsetKey := partitionZsetKey(fnPart, kg)
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
				Key(partitionConcurrencyKey(fnPart, kg)).
				Min("-inf").
				Max("+inf").
				Build()).AsStrSlice()
			require.NoError(t, err)
			assert.Equal(t, num, len(items), "expected %d items in the concurrency queue", num, r.Dump())
		}

		itemCountMatches(1)
		concurrencyItemCountMatches(0)

		_, err = shard.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		itemCountMatches(0)
		concurrencyItemCountMatches(1)

		// Ensure the concurrency index is updated.
		mem, err := r.ZMembers(kg.ConcurrencyIndex())
		require.NoError(t, err)
		assert.Equal(t, 1, len(mem))
		assert.Contains(t, mem[0], fnPart.ID)

		// Dequeue the item.
		err = shard.Dequeue(ctx, item)
		require.NoError(t, err)

		itemCountMatches(0)
		concurrencyItemCountMatches(0)

		// Ensure the concurrency index is updated.
		numMembers, err := rc.Do(ctx, rc.B().Zcard().Key(kg.ConcurrencyIndex()).Build()).AsInt64()
		require.NoError(t, err, r.Dump())
		assert.Equal(t, int64(0), numMembers, "concurrency index should be empty", mem)
	})
}

func TestQueueDequeueWithDisabledConstraintUpdates(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	q, shard := newQueue(
		t, rc,
		osqueue.WithClock(clock),
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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

	err = q.Dequeue(ctx, shard, item, osqueue.DequeueOptionDisableConstraintUpdates(true))
	require.NoError(t, err)

	require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
	require.False(t, r.Exists(kg.Concurrency("account", accountID.String())))
	require.False(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
}
