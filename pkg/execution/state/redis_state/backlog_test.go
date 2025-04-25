package redis_state

import (
	"context"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestQueueItemBacklogs(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(
		QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName},
		WithConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
			return PartitionConcurrencyLimits{
				AccountLimit:   100,
				FunctionLimit:  25,
				CustomKeyLimit: 0, // this is just used for PartitionLease on key queues v1
			}
		}),
		WithCustomConcurrencyKeyLimitRefresher(func(ctx context.Context, i osqueue.QueueItem) []state.CustomConcurrency {
			// Pretend current keys are latest version
			return i.Data.GetConcurrencyKeys()
		}),
		WithSystemConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) SystemPartitionConcurrencyLimits {
			return SystemPartitionConcurrencyLimits{
				// this is used by the old system as "account concurrency" for system queues -- bounding the entirety of system queue concurrency
				GlobalLimit: 0,

				// this is used to enforce concurrency limits on individual system queues
				PartitionLimit: 250,
			}
		}),
	)
	ctx := context.Background()

	fnID, wsID, accID := uuid.New(), uuid.New(), uuid.New()

	t.Run("basic item", func(t *testing.T) {
		expected := []QueueBacklog{
			// expect default backlog to be used
			{
				BacklogID:         fmt.Sprintf("default:%s", fnID),
				ShadowPartitionID: fnID.String(),
			},
		}

		backlogs := q.ItemBacklogs(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
				QueueName:             nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, backlogs)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-level concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlogs[0].activeKey(kg))
	})

	t.Run("system queue", func(t *testing.T) {
		sysQueueName := osqueue.KindQueueMigrate

		expected := []QueueBacklog{
			// expect default backlog to be used
			{
				BacklogID:         fmt.Sprintf("system:%s", sysQueueName),
				ShadowPartitionID: sysQueueName,
			},
		}

		backlogs := q.ItemBacklogs(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
				QueueName:             &sysQueueName,
			},
			QueueName: &sysQueueName,
		})

		require.Equal(t, expected, backlogs)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-level concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlogs[0].activeKey(kg))
	})

	t.Run("throttle", func(t *testing.T) {
		rawThrottleKey := fnID.String()
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := []QueueBacklog{
			{
				BacklogID:         fmt.Sprintf("throttle:%s:%s", fnID, hashedThrottleKey),
				ShadowPartitionID: fnID.String(),

				ThrottleKey:         &hashedThrottleKey,
				ThrottleKeyRawValue: &rawThrottleKey,
			},
		}

		backlogs := q.ItemBacklogs(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindStart,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle: &osqueue.Throttle{
					Key:                 hashedThrottleKey,
					Limit:               120,
					Burst:               30,
					Period:              700,
					UnhashedThrottleKey: rawThrottleKey,
				},
				CustomConcurrencyKeys: nil,
				QueueName:             nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, backlogs)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-level concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlogs[0].activeKey(kg))
	})

	t.Run("throttle with key", func(t *testing.T) {
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := []QueueBacklog{
			{
				BacklogID:         fmt.Sprintf("throttle:%s:%s", fnID, hashedThrottleKey),
				ShadowPartitionID: fnID.String(),

				ThrottleKey:         &hashedThrottleKey,
				ThrottleKeyRawValue: &rawThrottleKey,
			},
		}

		backlogs := q.ItemBacklogs(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindStart,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle: &osqueue.Throttle{
					Key:                 hashedThrottleKey,
					Limit:               120,
					Burst:               30,
					Period:              700,
					UnhashedThrottleKey: rawThrottleKey,
				},
				CustomConcurrencyKeys: nil,
				QueueName:             nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, backlogs)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-llevel concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlogs[0].activeKey(kg))
	})

	t.Run("throttle on edge item", func(t *testing.T) {
		rawThrottleKey := fnID.String()
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := []QueueBacklog{
			// edge should go to default backlog if no concurrency keys are specified
			{
				BacklogID:         fmt.Sprintf("default:%s", fnID),
				ShadowPartitionID: fnID.String(),
			},
		}

		backlogs := q.ItemBacklogs(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle: &osqueue.Throttle{
					Key:                 hashedThrottleKey,
					Limit:               120,
					Burst:               30,
					Period:              700,
					UnhashedThrottleKey: rawThrottleKey,
				},
				CustomConcurrencyKeys: nil,
				QueueName:             nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, backlogs)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-llevel concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlogs[0].activeKey(kg))
	})

	t.Run("function concurrency", func(t *testing.T) {
		expected := []QueueBacklog{
			// expect default backlog to be used
			{
				BacklogID:         fmt.Sprintf("default:%s", fnID),
				ShadowPartitionID: fnID.String(),
			},
		}

		backlogs := q.ItemBacklogs(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
				QueueName:             nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, backlogs)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-llevel concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlogs[0].activeKey(kg))
	})

	t.Run("account concurrency", func(t *testing.T) {
		expected := []QueueBacklog{
			// expect default backlog to be used
			{
				BacklogID:         fmt.Sprintf("default:%s", fnID),
				ShadowPartitionID: fnID.String(),
			},
		}

		backlogs := q.ItemBacklogs(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
				QueueName:             nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, backlogs)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-llevel concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlogs[0].activeKey(kg))
	})

	t.Run("custom concurrency", func(t *testing.T) {
		hashedConcurrencyKeyExpr := hashConcurrencyKey("event.data.customerId")
		unhashedValue := "customer1"
		scope := enums.ConcurrencyScopeFn
		entity := fnID
		fullKey := util.ConcurrencyKey(scope, fnID, unhashedValue)
		_, _, checksum, _ := state.CustomConcurrency{
			Key:   fullKey,
			Hash:  hashedConcurrencyKeyExpr,
			Limit: 123,
		}.ParseKey()

		expected := []QueueBacklog{
			{
				BacklogID:         fmt.Sprintf("conc:%s", fullKey),
				ShadowPartitionID: fnID.String(),

				ConcurrencyScope:            &scope,
				ConcurrencyScopeEntity:      &entity,
				ConcurrencyKey:              &hashedConcurrencyKeyExpr,
				ConcurrencyKeyValue:         &checksum,
				ConcurrencyKeyUnhashedValue: &unhashedValue,
			},
		}

		backlogs := q.ItemBacklogs(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle: nil,
				CustomConcurrencyKeys: []state.CustomConcurrency{
					{
						Key:                       fullKey,
						Hash:                      hashedConcurrencyKeyExpr,
						Limit:                     123,
						UnhashedEvaluatedKeyValue: unhashedValue,
					},
				},
				QueueName: nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, backlogs)

		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope, entity, checksum)), backlogs[0].activeKey(kg))
	})

	t.Run("concurrency + throttle start item", func(t *testing.T) {
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		hashedConcurrencyKeyExpr := hashConcurrencyKey("event.data.customerId")
		unhashedValue := "customer1"
		entity := fnID
		scope := enums.ConcurrencyScopeFn
		fullKey := util.ConcurrencyKey(scope, fnID, unhashedValue)
		_, _, checksum, _ := state.CustomConcurrency{
			Key:   fullKey,
			Hash:  hashedConcurrencyKeyExpr,
			Limit: 123,
		}.ParseKey()

		expected := []QueueBacklog{
			{
				BacklogID:         fmt.Sprintf("throttle:%s:%s", fnID, hashedThrottleKey),
				ShadowPartitionID: fnID.String(),

				ThrottleKey:         &hashedThrottleKey,
				ThrottleKeyRawValue: &rawThrottleKey,
			},
			{
				BacklogID:         fmt.Sprintf("conc:%s", fullKey),
				ShadowPartitionID: fnID.String(),

				ConcurrencyScope:            &scope,
				ConcurrencyScopeEntity:      &entity,
				ConcurrencyKey:              &hashedConcurrencyKeyExpr,
				ConcurrencyKeyValue:         &checksum,
				ConcurrencyKeyUnhashedValue: &unhashedValue,
			},
		}

		backlogs := q.ItemBacklogs(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindStart,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle: &osqueue.Throttle{
					Key:                 hashedThrottleKey,
					Limit:               120,
					Burst:               30,
					Period:              700,
					UnhashedThrottleKey: rawThrottleKey,
				},
				CustomConcurrencyKeys: []state.CustomConcurrency{
					{
						Key:                       fullKey,
						Hash:                      hashedConcurrencyKeyExpr,
						Limit:                     123,
						UnhashedEvaluatedKeyValue: unhashedValue,
					},
				},
				QueueName: nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, backlogs)
	})

	t.Run("concurrency + throttle edge item", func(t *testing.T) {
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		hashedConcurrencyKeyExpr := hashConcurrencyKey("event.data.customerId")
		unhashedValue := "customer1"
		scope := enums.ConcurrencyScopeFn
		entity := fnID
		fullKey := util.ConcurrencyKey(scope, fnID, unhashedValue)
		_, _, checksum, _ := state.CustomConcurrency{
			Key:   fullKey,
			Hash:  hashedConcurrencyKeyExpr,
			Limit: 123,
		}.ParseKey()

		expected := []QueueBacklog{
			{
				BacklogID:         fmt.Sprintf("conc:%s", fullKey),
				ShadowPartitionID: fnID.String(),

				ConcurrencyScope:            &scope,
				ConcurrencyKey:              &hashedConcurrencyKeyExpr,
				ConcurrencyScopeEntity:      &entity,
				ConcurrencyKeyValue:         &checksum,
				ConcurrencyKeyUnhashedValue: &unhashedValue,
			},
		}

		backlogs := q.ItemBacklogs(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle: &osqueue.Throttle{
					Key:                 hashedThrottleKey,
					Limit:               120,
					Burst:               30,
					Period:              700,
					UnhashedThrottleKey: rawThrottleKey,
				},
				CustomConcurrencyKeys: []state.CustomConcurrency{
					{
						Key:                       fullKey,
						Hash:                      hashedConcurrencyKeyExpr,
						Limit:                     123,
						UnhashedEvaluatedKeyValue: unhashedValue,
					},
				},
				QueueName: nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, backlogs)
	})
}

func TestQueueItemShadowPartition(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	q := NewQueue(
		QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName},
		WithConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
			return PartitionConcurrencyLimits{
				AccountLimit:   100,
				FunctionLimit:  25,
				CustomKeyLimit: 0, // this is just used for PartitionLease on key queues v1
			}
		}),
		WithCustomConcurrencyKeyLimitRefresher(func(ctx context.Context, i osqueue.QueueItem) []state.CustomConcurrency {
			// Pretend current keys are latest version
			return i.Data.GetConcurrencyKeys()
		}),
		WithSystemConcurrencyLimitGetter(func(ctx context.Context, p QueuePartition) SystemPartitionConcurrencyLimits {
			return SystemPartitionConcurrencyLimits{
				// this is used by the old system as "account concurrency" for system queues -- bounding the entirety of system queue concurrency
				GlobalLimit: 0,

				// this is used to enforce concurrency limits on individual system queues
				PartitionLimit: 250,
			}
		}),
	)
	ctx := context.Background()

	fnID, wsID, accID := uuid.New(), uuid.New(), uuid.New()

	t.Run("basic item", func(t *testing.T) {
		expected := QueueShadowPartition{
			ShadowPartitionID:     fnID.String(),
			FunctionID:            &fnID,
			EnvID:                 &wsID,
			AccountID:             &accID,
			SystemQueueName:       nil,
			SystemConcurrency:     0,
			AccountConcurrency:    100,
			FunctionConcurrency:   25,
			CustomConcurrencyKeys: nil,
			Throttle:              nil,
			PauseRefill:           false,
			PauseEnqueue:          false,
		}

		shadowPart := q.ItemShadowPartition(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
				QueueName:             nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, shadowPart)

		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("p", fnID.String()), shadowPart.inProgressKey(kg))
		require.Equal(t, kg.Concurrency("account", accID.String()), shadowPart.accountInProgressKey(kg))
	})

	t.Run("system queue", func(t *testing.T) {
		sysQueueName := osqueue.KindQueueMigrate

		expected := QueueShadowPartition{
			ShadowPartitionID:     sysQueueName,
			FunctionID:            nil,
			EnvID:                 nil,
			AccountID:             nil,
			SystemQueueName:       &sysQueueName,
			SystemConcurrency:     250,
			AccountConcurrency:    0,
			FunctionConcurrency:   0,
			CustomConcurrencyKeys: nil,
			Throttle:              nil,
			PauseRefill:           false,
			PauseEnqueue:          false,
		}

		shadowPart := q.ItemShadowPartition(ctx, osqueue.QueueItem{
			ID: "test",
			Data: osqueue.Item{
				Kind:                  osqueue.KindQueueMigrate,
				Identifier:            state.Identifier{},
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
				QueueName:             &sysQueueName,
			},
			QueueName: &sysQueueName,
		})

		require.Equal(t, expected, shadowPart)

		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("p", sysQueueName), shadowPart.inProgressKey(kg))

		// expect empty key: system queues should not track account concurrency
		require.Equal(t, kg.Concurrency("account", sysQueueName), shadowPart.accountInProgressKey(kg))
	})

	t.Run("throttle", func(t *testing.T) {
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := QueueShadowPartition{
			ShadowPartitionID:     fnID.String(),
			FunctionID:            &fnID,
			EnvID:                 &wsID,
			AccountID:             &accID,
			SystemQueueName:       nil,
			SystemConcurrency:     0,
			AccountConcurrency:    100,
			FunctionConcurrency:   25,
			CustomConcurrencyKeys: nil,
			Throttle: &osqueue.Throttle{
				Key:                 hashedThrottleKey,
				Limit:               70,
				Burst:               20,
				Period:              600,
				UnhashedThrottleKey: rawThrottleKey,
			},
			PauseRefill:  false,
			PauseEnqueue: false,
		}

		shadowPart := q.ItemShadowPartition(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle: &osqueue.Throttle{
					Key:                 hashedThrottleKey,
					Limit:               70,
					Burst:               20,
					Period:              600,
					UnhashedThrottleKey: rawThrottleKey,
				},
				CustomConcurrencyKeys: nil,
				QueueName:             nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, shadowPart)

		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("p", fnID.String()), shadowPart.inProgressKey(kg))
		require.Equal(t, kg.Concurrency("account", accID.String()), shadowPart.accountInProgressKey(kg))
	})

	t.Run("custom concurrency", func(t *testing.T) {
		hashedConcurrencyKeyExpr := hashConcurrencyKey("event.data.customerId")
		unhashedValue := "customer1"
		fullKey := util.ConcurrencyKey(enums.ConcurrencyScopeFn, fnID, unhashedValue)

		expected := QueueShadowPartition{
			ShadowPartitionID:   fnID.String(),
			FunctionID:          &fnID,
			EnvID:               &wsID,
			AccountID:           &accID,
			SystemQueueName:     nil,
			SystemConcurrency:   0,
			AccountConcurrency:  100,
			FunctionConcurrency: 25,
			CustomConcurrencyKeys: []CustomConcurrencyLimit{
				{
					Scope: enums.ConcurrencyScopeFn,
					Key:   hashedConcurrencyKeyExpr,
					Limit: 23,
				},
			},
			Throttle:     nil,
			PauseRefill:  false,
			PauseEnqueue: false,
		}

		shadowPart := q.ItemShadowPartition(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle: nil,
				CustomConcurrencyKeys: []state.CustomConcurrency{
					{
						Key:   fullKey,
						Hash:  hashedConcurrencyKeyExpr,
						Limit: 23,

						// This isn't stored in the shadow partition
						UnhashedEvaluatedKeyValue: unhashedValue,
					},
				},
				QueueName: nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, shadowPart)

		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("p", fnID.String()), shadowPart.inProgressKey(kg))
		require.Equal(t, kg.Concurrency("account", accID.String()), shadowPart.accountInProgressKey(kg))
	})

	t.Run("concurrency + throttle", func(t *testing.T) {
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		hashedConcurrencyKeyExpr := hashConcurrencyKey("event.data.customerId")
		unhashedValue := "customer1"
		fullKey := util.ConcurrencyKey(enums.ConcurrencyScopeFn, fnID, unhashedValue)

		expected := QueueShadowPartition{
			ShadowPartitionID:   fnID.String(),
			FunctionID:          &fnID,
			EnvID:               &wsID,
			AccountID:           &accID,
			SystemQueueName:     nil,
			SystemConcurrency:   0,
			AccountConcurrency:  100,
			FunctionConcurrency: 25,
			CustomConcurrencyKeys: []CustomConcurrencyLimit{
				{
					Scope: enums.ConcurrencyScopeFn,
					Key:   hashedConcurrencyKeyExpr,
					Limit: 23,
				},
			},
			Throttle: &osqueue.Throttle{
				Key:                 hashedThrottleKey,
				Limit:               70,
				Burst:               20,
				Period:              600,
				UnhashedThrottleKey: rawThrottleKey,
			},
			PauseRefill:  false,
			PauseEnqueue: false,
		}

		shadowPart := q.ItemShadowPartition(ctx, osqueue.QueueItem{
			ID:          "test",
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					WorkflowID:  fnID,
					AccountID:   accID,
					WorkspaceID: wsID,
				},
				Throttle: &osqueue.Throttle{
					Key:                 hashedThrottleKey,
					Limit:               70,
					Burst:               20,
					Period:              600,
					UnhashedThrottleKey: rawThrottleKey,
				},
				CustomConcurrencyKeys: []state.CustomConcurrency{
					{
						Key:   fullKey,
						Hash:  hashedConcurrencyKeyExpr,
						Limit: 23,

						// This isn't stored in the shadow partition
						UnhashedEvaluatedKeyValue: unhashedValue,
					},
				},
				QueueName: nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, shadowPart)

		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("p", fnID.String()), shadowPart.inProgressKey(kg))
		require.Equal(t, kg.Concurrency("account", accID.String()), shadowPart.accountInProgressKey(kg))
	})
}

func hashConcurrencyKey(key string) string {
	return strconv.FormatUint(xxhash.Sum64String(key), 36)
}
