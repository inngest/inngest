package redis_state

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util"
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

		defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
		kg := defaultShard.RedisClient.kg

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
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
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// put item in progress, this is tested separately
			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := q.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 0)

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.Empty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom1.PartitionType)
			require.Empty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom2.PartitionType)

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.accountInProgressKey(kg), qi.ID)))

			// no active set for default partition since this uses the in progress key
			require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
			require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, fnPart.concurrencyKey(kg), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			err = q.Dequeue(ctx, defaultShard, qi)
			require.NoError(t, err)

			// expect item not to be requeued
			require.False(t, hasMember(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID))

			// expect key queue accounting to be updated
			require.False(t, hasMember(t, r, shadowPartition.inProgressKey(kg), qi.ID))
			require.False(t, hasMember(t, r, shadowPartition.accountInProgressKey(kg), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, fnPart.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))
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

		defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
		kg := defaultShard.RedisClient.kg

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
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

			q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{
					Concurrency: PartitionConcurrency{
						AccountConcurrency:  123,
						FunctionConcurrency: 45,
						CustomConcurrencyKeys: []CustomConcurrencyLimit{
							{
								Scope:               enums.ConcurrencyScopeFn,
								HashedKeyExpression: ckA.Hash,
								Limit:               ckA.Limit,
							},
						},
					},
				}
			}

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
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// put item in progress, this is tested separately
			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := q.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 1)

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.NotEmpty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
			require.Empty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom2.PartitionType)

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.accountInProgressKey(kg), qi.ID)))

			// 1 active set for custom concurrency key
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope, fnID, unhashedValue)), backlog.customKeyInProgress(kg, 1))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlog.customKeyInProgress(kg, 1), qi.ID)))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, custom1.concurrencyKey(kg), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), qi.ID), r.Keys())
			require.True(t, hasMember(t, r, kg.Concurrency("custom", fullKey), qi.ID))

			err = q.Dequeue(ctx, defaultShard, qi)
			require.NoError(t, err)

			// expect item not to be requeued
			require.False(t, hasMember(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID))

			// expect key queue accounting to be updated
			require.False(t, hasMember(t, r, shadowPartition.inProgressKey(kg), qi.ID))
			require.False(t, hasMember(t, r, shadowPartition.accountInProgressKey(kg), qi.ID))
			require.False(t, hasMember(t, r, backlog.customKeyInProgress(kg, 1), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, custom1.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))
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

		defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
		kg := defaultShard.RedisClient.kg

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
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

			q.partitionConstraintConfigGetter = func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return PartitionConstraintConfig{
					Concurrency: PartitionConcurrency{
						AccountConcurrency:  123,
						FunctionConcurrency: 45,
						CustomConcurrencyKeys: []CustomConcurrencyLimit{
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
			}

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
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// put item in progress, this is tested separately
			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := q.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item)
			require.Len(t, backlog.ConcurrencyKeys, 2)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 2)

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.NotEmpty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
			require.NotEmpty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom2.PartitionType)

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.accountInProgressKey(kg), qi.ID)))

			// 2 active set for custom concurrency keys
			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope1, fnID, unhashedValue1)), backlog.customKeyInProgress(kg, 1))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlog.customKeyInProgress(kg, 1), qi.ID)))

			require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope2, wsID, unhashedValue2)), backlog.customKeyInProgress(kg, 2))
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, backlog.customKeyInProgress(kg, 2), qi.ID)))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, custom1.concurrencyKey(kg), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), qi.ID), r.Keys())
			require.True(t, hasMember(t, r, kg.Concurrency("custom", fullKey1), qi.ID))
			require.True(t, hasMember(t, r, kg.Concurrency("custom", fullKey2), qi.ID))

			err = q.Dequeue(ctx, defaultShard, qi)
			require.NoError(t, err)

			// expect item not to be requeued
			require.False(t, hasMember(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID))

			// expect key queue accounting to be updated
			require.False(t, hasMember(t, r, shadowPartition.inProgressKey(kg), qi.ID))
			require.False(t, hasMember(t, r, shadowPartition.accountInProgressKey(kg), qi.ID))
			require.False(t, hasMember(t, r, backlog.customKeyInProgress(kg, 1), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, custom1.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", accountId.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey1), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("custom", fullKey2), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))
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

		defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
		kg := defaultShard.RedisClient.kg

		enqueueToBacklog := false
		q := NewQueue(
			defaultShard,
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
				return enqueueToBacklog
			}),
			WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
				return true
			}),
		)
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
			qi, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			enqueueToBacklog = true

			// put item in progress, this is tested separately
			now := q.clock.Now()
			leaseDur := 5 * time.Second
			leaseExpires := now.Add(leaseDur)
			leaseID, err := q.Lease(ctx, qi, leaseDur, now, nil)
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			backlog := q.ItemBacklog(ctx, item)
			require.NotEmpty(t, backlog.BacklogID)

			shadowPartition := q.ItemShadowPartition(ctx, item)
			require.NotEmpty(t, shadowPartition.PartitionID)

			constraints := q.partitionConstraintConfigGetter(ctx, shadowPartition.Identifier())
			require.Len(t, constraints.Concurrency.CustomConcurrencyKeys, 0)

			fnPart, custom1, custom2 := q.ItemPartitions(ctx, defaultShard, item)
			require.NotEmpty(t, fnPart.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.True(t, fnPart.IsSystem())
			require.Empty(t, custom1.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom1.PartitionType)
			require.Empty(t, custom2.ID)
			require.Equal(t, int(enums.PartitionTypeDefault), custom2.PartitionType)

			// expect key queue accounting to contain item in in-progress
			require.Equal(t, leaseExpires.UnixMilli(), int64(score(t, r, shadowPartition.inProgressKey(kg), qi.ID)))

			require.Equal(t, kg.Concurrency("account", ""), shadowPartition.accountInProgressKey(kg))
			require.False(t, r.Exists(shadowPartition.accountInProgressKey(kg)))

			// no active set for default partition since this uses the in progress key
			require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.True(t, hasMember(t, r, fnPart.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", fnPart.Queue()), qi.ID)) // pseudo-limit for system queue
			require.True(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			err = q.Dequeue(ctx, defaultShard, qi)
			require.NoError(t, err)

			// expect item not to be requeued
			require.False(t, hasMember(t, r, kg.BacklogSet(backlog.BacklogID), qi.ID))

			// expect key queue accounting to be updated
			require.False(t, hasMember(t, r, shadowPartition.inProgressKey(kg), qi.ID))
			require.False(t, hasMember(t, r, shadowPartition.accountInProgressKey(kg), qi.ID))

			// expect old accounting to be updated
			// TODO Do we actually want to update previous accounting?
			require.False(t, hasMember(t, r, fnPart.concurrencyKey(kg), qi.ID))
			require.False(t, hasMember(t, r, kg.Concurrency("account", fnPart.Queue()), qi.ID)) // pseudo-limit for system queue
			require.False(t, hasMember(t, r, kg.Concurrency("p", fnPart.Queue()), qi.ID))

			// item must not be in classic backlog
			require.False(t, hasMember(t, r, fnPart.zsetKey(kg), qi.ID))
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

	queueClient := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	q := NewQueue(queueClient)
	ctx := context.Background()

	t.Run("It always changes global partition scores", func(t *testing.T) {
		r.FlushAll()

		fnID, acctID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("fn")),
			uuid.NewSHA1(uuid.NameSpaceDNS, []byte("acct"))

		start := time.Now().Truncate(time.Second)

		// Enqueue two items to the same function
		itemA, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
		_, err = q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
		fnPart, custom1, custom2 := q.ItemPartitions(ctx, q.primaryQueueShard, itemA)
		require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
		require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
		require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom2.PartitionType)

		// Lease the first item, pretending it's in progress.
		_, err = q.Lease(ctx, itemA, 10*time.Second, q.clock.Now(), nil)
		require.NoError(t, err)

		// Note: Originally, this test used the concurrency key queue for testing Dequeue(),
		// but this was changed to the default partition, as we do not enqueue to key queues anymore.
		partitionToDequeue := fnPart

		// Force requeue the partition such that it's pushed forward, pretending there's
		// no capacity.
		err = q.PartitionRequeue(ctx, q.primaryQueueShard, &partitionToDequeue, start.Add(30*time.Minute), true)
		require.NoError(t, err)

		t.Run("Requeueing partitions updates the score", func(t *testing.T) {
			partScoreA, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), partitionToDequeue.ID)
			require.EqualValues(t, start.Add(30*time.Minute).Unix(), partScoreA[0])

			partScoreA, _ = r.ZMScore(q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(acctID), partitionToDequeue.ID)
			require.NotNil(t, partScoreA, "expected partition requeue to update account partition index", r.Dump())
			require.EqualValues(t, start.Add(30*time.Minute).Unix(), partScoreA[0])
		})

		// Dequeue to pull partition back to now
		err = q.Dequeue(ctx, q.primaryQueueShard, itemA)
		require.Nil(t, err)

		t.Run("The outstanding partition scores should reset", func(t *testing.T) {
			partScoreA, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), partitionToDequeue.ID)
			require.EqualValues(t, start, time.Unix(int64(partScoreA[0]), 0), r.Dump(), partitionToDequeue, start.UnixMilli())
		})
	})

	t.Run("with concurrency keys", func(t *testing.T) {
		start := time.Now()

		t.Run("with an unleased item", func(t *testing.T) {
			r.FlushAll()
			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
			fnPart, custom1, custom2 := q.ItemPartitions(ctx, q.primaryQueueShard, item)
			require.Equal(t, int(enums.PartitionTypeDefault), fnPart.PartitionType)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom2.PartitionType)

			err = q.Dequeue(ctx, q.primaryQueueShard, item)
			require.Nil(t, err)

			t.Run("The outstanding partition items should be empty", func(t *testing.T) {
				mem, _ := r.ZMembers(fnPart.zsetKey(q.primaryQueueShard.RedisClient.kg))
				require.Equal(t, 0, len(mem))

				mem, _ = r.ZMembers(custom1.zsetKey(q.primaryQueueShard.RedisClient.kg))
				require.NoError(t, err)
				require.Equal(t, 0, len(mem))
			})
		})

		t.Run("with a leased item", func(t *testing.T) {
			r.FlushAll()
			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
			_, custom1, custom2 := q.ItemPartitions(ctx, q.primaryQueueShard, item)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom1.PartitionType)
			require.Equal(t, int(enums.PartitionTypeConcurrencyKey), custom2.PartitionType)

			id, err := q.Lease(ctx, item, 10*time.Second, time.Now(), nil)
			require.NoError(t, err)
			require.NotEmpty(t, id)

			t.Run("The scavenger queue should not yet be empty", func(t *testing.T) {
				mems, err := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
				require.NoError(t, err)
				require.NotEmpty(t, mems)
			})

			err = q.Dequeue(ctx, q.primaryQueueShard, item)
			require.Nil(t, err)

			t.Run("The outstanding partition items should be empty", func(t *testing.T) {
				mem, _ := r.ZMembers(custom1.zsetKey(q.primaryQueueShard.RedisClient.kg))
				require.Equal(t, 0, len(mem))

				mem, _ = r.ZMembers(custom2.zsetKey(q.primaryQueueShard.RedisClient.kg))
				require.NoError(t, err)
				require.Equal(t, 0, len(mem))
			})

			t.Run("The concurrenty partition items should be empty", func(t *testing.T) {
				mem, _ := r.ZMembers(custom1.concurrencyKey(q.primaryQueueShard.RedisClient.kg))
				require.Equal(t, 0, len(mem))

				mem, _ = r.ZMembers(custom2.concurrencyKey(q.primaryQueueShard.RedisClient.kg))
				require.NoError(t, err)
				require.Equal(t, 0, len(mem))
			})

			t.Run("The scavenger queue should now be empty", func(t *testing.T) {
				mems, _ := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
				require.Empty(t, mems)
			})
		})
	})

	t.Run("It should remove a queue item", func(t *testing.T) {
		r.FlushAll()

		start := time.Now()

		fnID := uuid.New()
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					RunID:      runID,
					WorkflowID: fnID,
				},
			},
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := QueuePartition{FunctionID: &item.FunctionID}

		id, err := q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		t.Run("The lease exists in the partition queue", func(t *testing.T) {
			count, err := q.InProgress(ctx, "p", p.FunctionID.String())
			require.NoError(t, err)
			require.EqualValues(t, 1, count, r.Dump())
		})

		err = q.Dequeue(ctx, q.primaryQueueShard, item)
		require.NoError(t, err)

		t.Run("It should remove the item from the queue map", func(t *testing.T) {
			val := r.HGet(q.primaryQueueShard.RedisClient.kg.QueueItem(), id.String())
			require.Empty(t, val)
		})

		t.Run("Extending a lease should fail after dequeue", func(t *testing.T) {
			id, err := q.ExtendLease(ctx, item, *id, time.Minute)
			require.Equal(t, ErrQueueItemNotFound, err)
			require.Nil(t, id)
		})

		t.Run("It should remove the item from the queue index", func(t *testing.T) {
			items, err := q.Peek(ctx, &p, time.Now().Add(time.Hour), 10)
			require.NoError(t, err)
			require.EqualValues(t, 0, len(items))
		})

		t.Run("It should remove the item from the concurrency partition's queue", func(t *testing.T) {
			count, err := q.InProgress(ctx, "p", p.FunctionID.String())
			require.NoError(t, err)
			require.EqualValues(t, 0, count)
		})

		t.Run("run indexes are updated", func(t *testing.T) {
			kg := q.primaryQueueShard.RedisClient.kg
			// Run indexes should be updated

			require.False(t, r.Exists(kg.ActiveSet("run", runID.String())))
			require.False(t, r.Exists(kg.ActiveRunsSet("p", fnID.String())))
		})

		t.Run("It should work if the item is not leased (eg. deletions)", func(t *testing.T) {
			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{}, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			err = q.Dequeue(ctx, q.primaryQueueShard, item)
			require.NoError(t, err)

			val := r.HGet(q.primaryQueueShard.RedisClient.kg.QueueItem(), id.String())
			require.Empty(t, val)
		})

		t.Run("Removes default indexes", func(t *testing.T) {
			at := time.Now().Truncate(time.Second)
			rid := ulid.MustNew(ulid.Now(), rand.Reader)
			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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

			err = q.Dequeue(ctx, q.primaryQueueShard, item)
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
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: uuid.New(),
			Data: osqueue.Item{
				QueueName: &customQueueName,
			},
			QueueName: &customQueueName,
		}, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		fnPart := q.ItemPartition(ctx, q.primaryQueueShard, item)

		itemCountMatches := func(num int) {
			zsetKey := fnPart.zsetKey(q.primaryQueueShard.RedisClient.kg)
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
				Key(fnPart.concurrencyKey(q.primaryQueueShard.RedisClient.kg)).
				Min("-inf").
				Max("+inf").
				Build()).AsStrSlice()
			require.NoError(t, err)
			assert.Equal(t, num, len(items), "expected %d items in the concurrency queue", num, r.Dump())
		}

		itemCountMatches(1)
		concurrencyItemCountMatches(0)

		_, err = q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		itemCountMatches(0)
		concurrencyItemCountMatches(1)

		// Ensure the concurrency index is updated.
		mem, err := r.ZMembers(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex())
		require.NoError(t, err)
		assert.Equal(t, 1, len(mem))
		assert.Contains(t, mem[0], fnPart.ID)

		// Dequeue the item.
		err = q.Dequeue(ctx, q.primaryQueueShard, item)
		require.NoError(t, err)

		itemCountMatches(0)
		concurrencyItemCountMatches(0)

		// Ensure the concurrency index is updated.
		numMembers, err := rc.Do(ctx, rc.B().Zcard().Key(q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex()).Build()).AsInt64()
		require.NoError(t, err, r.Dump())
		assert.Equal(t, int64(0), numMembers, "concurrency index should be empty", mem)
	})
}

func TestQueueRequeue(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName})
	ctx := context.Background()

	t.Run("Re-enqueuing a leased item should succeed", func(t *testing.T) {
		now := time.Now()

		fnID := uuid.New()
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					RunID:      runID,
					WorkflowID: fnID,
				},
			},
		}, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = q.Lease(ctx, item, time.Second, time.Now(), nil)
		require.NoError(t, err)

		// Assert partition index is original
		pi := QueuePartition{FunctionID: &item.FunctionID}
		requirePartitionScoreEquals(t, r, pi.FunctionID, now.Truncate(time.Second))

		requirePartitionInProgress(t, q, item.FunctionID, 1)

		next := now.Add(time.Hour)
		err = q.Requeue(ctx, q.primaryQueueShard, item, next)
		require.NoError(t, err)

		t.Run("It should re-enqueue the item with the future time", func(t *testing.T) {
			requireItemScoreEquals(t, r, item, next)
		})

		t.Run("It should always remove the lease from the re-enqueued item", func(t *testing.T) {
			fetched := getQueueItem(t, r, item.ID)
			require.Nil(t, fetched.LeaseID)
		})

		t.Run("It should decrease the in-progress count", func(t *testing.T) {
			requirePartitionInProgress(t, q, item.FunctionID, 0)
		})

		t.Run("It should update the partition's earliest time, if earliest", func(t *testing.T) {
			// Assert partition index is updated, as there's only one item here.
			requirePartitionScoreEquals(t, r, pi.FunctionID, next)
		})

		t.Run("run indexes are updated on requeue to partition", func(t *testing.T) {
			kg := q.primaryQueueShard.RedisClient.kg

			require.False(t, r.Exists(kg.ActiveRunsSet("p", item.FunctionID.String())))
			require.False(t, r.Exists(kg.ActiveSet("run", runID.String())))
		})

		t.Run("It should not update the partition's earliest time, if later", func(t *testing.T) {
			_, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
			err = q.Requeue(ctx, q.primaryQueueShard, item, next)
			require.NoError(t, err)

			requirePartitionScoreEquals(t, r, pi.FunctionID, now)
		})

		t.Run("Updates default indexes", func(t *testing.T) {
			at := time.Now().Truncate(time.Second)
			rid := ulid.MustNew(ulid.Now(), rand.Reader)
			item, err := q.EnqueueItem(ctx, q.primaryQueueShard, osqueue.QueueItem{
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
			err = q.Requeue(ctx, q.primaryQueueShard, item, next)
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
		item, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		fnPart, custom1, custom2 := q.ItemPartitions(ctx, q.primaryQueueShard, item)

		// Get all scores
		require.False(t, r.Exists(custom1.zsetKey(q.primaryQueueShard.RedisClient.kg)))
		require.False(t, r.Exists(custom2.zsetKey(q.primaryQueueShard.RedisClient.kg)))
		itemScoreDefault, _ := r.ZMScore(fnPart.zsetKey(q.primaryQueueShard.RedisClient.kg), item.ID)
		partScoreDefault, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), fnPart.ID)
		accountPartScore, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(acctID), fnPart.ID)
		accountScore, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.GlobalAccountIndex(), acctID.String())

		require.NotEmpty(t, itemScoreDefault, "Couldn't find item in '%s':\n%s", custom1.zsetKey(q.primaryQueueShard.RedisClient.kg), r.Dump())
		require.NotEmpty(t, partScoreDefault)
		require.Equal(t, partScoreDefault, accountPartScore, "expected account partitions to match global partitions")
		require.Equal(t, accountPartScore[0], accountScore[0], "expected account score to match earliest account partition")

		_, err = q.Lease(ctx, item, time.Second, q.clock.Now(), nil)
		require.NoError(t, err)

		// Requeue
		next := now.Add(time.Hour)
		err = q.Requeue(ctx, q.primaryQueueShard, item, next)
		require.NoError(t, err)

		t.Run("It requeues all partitions", func(t *testing.T) {
			newItemScore, _ := r.ZMScore(fnPart.zsetKey(q.primaryQueueShard.RedisClient.kg), item.ID)
			newPartScore, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), fnPart.ID)
			newAccountPartScore, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(acctID), fnPart.ID)
			newAccountScore, _ := r.ZMScore(q.primaryQueueShard.RedisClient.kg.GlobalAccountIndex(), acctID.String())

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
