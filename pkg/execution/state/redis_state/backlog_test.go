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
		expected := QueueBacklog{
			// expect default backlog to be used
			BacklogID:         fmt.Sprintf("fn:%s", fnID),
			ShadowPartitionID: fnID.String(),
		}

		backlog := q.ItemBacklog(ctx, osqueue.QueueItem{
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

		require.Equal(t, expected, backlog)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-level concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 2))
	})

	t.Run("system queue", func(t *testing.T) {
		sysQueueName := osqueue.KindQueueMigrate

		expected := QueueBacklog{
			// expect default backlog to be used
			BacklogID:         fmt.Sprintf("system:%s", sysQueueName),
			ShadowPartitionID: sysQueueName,
		}

		backlog := q.ItemBacklog(ctx, osqueue.QueueItem{
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

		require.Equal(t, expected, backlog)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-level concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 2))

	})

	t.Run("throttle", func(t *testing.T) {
		throttleKeyExpressionHash := util.XXHash("event.data.customerID")
		rawThrottleKey := fnID.String()
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:t<%s>", fnID, hashedThrottleKey),
			ShadowPartitionID: fnID.String(),

			Throttle: &BacklogThrottle{
				ThrottleKey:               hashedThrottleKey,
				ThrottleKeyRawValue:       rawThrottleKey,
				ThrottleKeyExpressionHash: throttleKeyExpressionHash,
			},
		}

		backlog := q.ItemBacklog(ctx, osqueue.QueueItem{
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
					KeyExpressionHash:   throttleKeyExpressionHash,
				},
				CustomConcurrencyKeys: nil,
				QueueName:             nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, backlog)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-level concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 2))
	})

	t.Run("throttle with key", func(t *testing.T) {
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:t<%s>", fnID, hashedThrottleKey),
			ShadowPartitionID: fnID.String(),

			Throttle: &BacklogThrottle{
				ThrottleKey:         hashedThrottleKey,
				ThrottleKeyRawValue: rawThrottleKey,
			},
		}

		backlog := q.ItemBacklog(ctx, osqueue.QueueItem{
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

		require.Equal(t, expected, backlog)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-llevel concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 2))
	})

	t.Run("throttle on edge item", func(t *testing.T) {
		rawThrottleKey := fnID.String()
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := QueueBacklog{
			// edge should go to default backlog if no concurrency keys are specified
			BacklogID:         fmt.Sprintf("fn:%s", fnID),
			ShadowPartitionID: fnID.String(),
		}

		backlog := q.ItemBacklog(ctx, osqueue.QueueItem{
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

		require.Equal(t, expected, backlog)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-llevel concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 2))
	})

	t.Run("function concurrency", func(t *testing.T) {
		expected := QueueBacklog{
			// expect default backlog to be used
			BacklogID:         fmt.Sprintf("fn:%s", fnID),
			ShadowPartitionID: fnID.String(),
		}

		backlog := q.ItemBacklog(ctx, osqueue.QueueItem{
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

		require.Equal(t, expected, backlog)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-llevel concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 2))
	})

	t.Run("account concurrency", func(t *testing.T) {
		expected := QueueBacklog{
			// expect default backlog to be used
			BacklogID:         fmt.Sprintf("fn:%s", fnID),
			ShadowPartitionID: fnID.String(),
		}

		backlog := q.ItemBacklog(ctx, osqueue.QueueItem{
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

		require.Equal(t, expected, backlog)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-llevel concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 2))
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

		expected := QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:c1<%s>", fnID, util.XXHash(fullKey)),
			ShadowPartitionID: fnID.String(),

			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					ConcurrencyScope:            scope,
					ConcurrencyScopeEntity:      entity,
					ConcurrencyKey:              hashedConcurrencyKeyExpr,
					ConcurrencyKeyValue:         checksum,
					ConcurrencyKeyUnhashedValue: unhashedValue,
				},
			},
		}

		backlog := q.ItemBacklog(ctx, osqueue.QueueItem{
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

		require.Equal(t, expected, backlog)

		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope, entity, unhashedValue)), backlog.concurrencyKey(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.concurrencyKey(kg, 2))
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

		expected := QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:t<%s>:c1<%s>", fnID, hashedThrottleKey, util.XXHash(fullKey)),
			ShadowPartitionID: fnID.String(),

			Throttle: &BacklogThrottle{
				ThrottleKey:         hashedThrottleKey,
				ThrottleKeyRawValue: rawThrottleKey,
			},

			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					ConcurrencyScope:            scope,
					ConcurrencyScopeEntity:      entity,
					ConcurrencyKey:              hashedConcurrencyKeyExpr,
					ConcurrencyKeyValue:         checksum,
					ConcurrencyKeyUnhashedValue: unhashedValue,
				},
			},
		}

		backlog := q.ItemBacklog(ctx, osqueue.QueueItem{
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

		require.Equal(t, expected, backlog)
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

		expected := QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:c1<%s>", fnID, util.XXHash(fullKey)),
			ShadowPartitionID: fnID.String(),

			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					ConcurrencyScope:            scope,
					ConcurrencyKey:              hashedConcurrencyKeyExpr,
					ConcurrencyScopeEntity:      entity,
					ConcurrencyKeyValue:         checksum,
					ConcurrencyKeyUnhashedValue: unhashedValue,
				},
			},
		}

		backlog := q.ItemBacklog(ctx, osqueue.QueueItem{
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

		require.Equal(t, expected, backlog)
	})

	t.Run("two custom concurrency keys", func(t *testing.T) {
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

		hashedConcurrencyKeyExpr2 := hashConcurrencyKey("event.data.orgID")
		unhashedValue2 := "orgID"
		scope2 := enums.ConcurrencyScopeEnv
		entity2 := fnID
		fullKey2 := util.ConcurrencyKey(scope2, fnID, unhashedValue2)
		_, _, checksum2, _ := state.CustomConcurrency{
			Key:   fullKey2,
			Hash:  hashedConcurrencyKeyExpr2,
			Limit: 123,
		}.ParseKey()

		expected := QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:c1<%s>:c2<%s>", fnID, util.XXHash(fullKey), util.XXHash(fullKey2)),
			ShadowPartitionID: fnID.String(),

			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					ConcurrencyScope:            scope,
					ConcurrencyScopeEntity:      entity,
					ConcurrencyKey:              hashedConcurrencyKeyExpr,
					ConcurrencyKeyValue:         checksum,
					ConcurrencyKeyUnhashedValue: unhashedValue,
				},
				{
					ConcurrencyScope:            scope2,
					ConcurrencyScopeEntity:      entity2,
					ConcurrencyKey:              hashedConcurrencyKeyExpr2,
					ConcurrencyKeyValue:         checksum2,
					ConcurrencyKeyUnhashedValue: unhashedValue2,
				},
			},
		}

		backlog := q.ItemBacklog(ctx, osqueue.QueueItem{
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
					{
						Key:                       fullKey2,
						Hash:                      hashedConcurrencyKeyExpr2,
						Limit:                     123,
						UnhashedEvaluatedKeyValue: unhashedValue2,
					},
				},
				QueueName: nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, backlog)

		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope, entity, unhashedValue)), backlog.concurrencyKey(kg, 1))
		require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope2, entity2, unhashedValue2)), backlog.concurrencyKey(kg, 2))
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
			PartitionID:           fnID.String(),
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
			PartitionID:           sysQueueName,
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
		hashedThrottleKeyExpr := util.XXHash("event.data.customerID")
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := QueueShadowPartition{
			PartitionID:           fnID.String(),
			FunctionID:            &fnID,
			EnvID:                 &wsID,
			AccountID:             &accID,
			SystemQueueName:       nil,
			SystemConcurrency:     0,
			AccountConcurrency:    100,
			FunctionConcurrency:   25,
			CustomConcurrencyKeys: nil,
			Throttle: &ShadowPartitionThrottle{
				ThrottleKeyExpressionHash: hashedThrottleKeyExpr,
				Limit:                     70,
				Burst:                     20,
				Period:                    600,
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
					Key:               hashedThrottleKey,
					Limit:             70,
					Burst:             20,
					Period:            600,
					KeyExpressionHash: hashedThrottleKeyExpr,
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
			PartitionID:         fnID.String(),
			FunctionID:          &fnID,
			EnvID:               &wsID,
			AccountID:           &accID,
			SystemQueueName:     nil,
			SystemConcurrency:   0,
			AccountConcurrency:  100,
			FunctionConcurrency: 25,
			CustomConcurrencyKeys: map[string]CustomConcurrencyLimit{
				concurrencyKeyID(enums.ConcurrencyScopeFn, hashedConcurrencyKeyExpr): {
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
		hashedThrottleKeyExpr := hashConcurrencyKey("event.data.customerId")
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		hashedConcurrencyKeyExpr := hashConcurrencyKey("event.data.customerId")
		unhashedValue := "customer1"
		fullKey := util.ConcurrencyKey(enums.ConcurrencyScopeFn, fnID, unhashedValue)

		expected := QueueShadowPartition{
			PartitionID:         fnID.String(),
			FunctionID:          &fnID,
			EnvID:               &wsID,
			AccountID:           &accID,
			SystemQueueName:     nil,
			SystemConcurrency:   0,
			AccountConcurrency:  100,
			FunctionConcurrency: 25,
			CustomConcurrencyKeys: map[string]CustomConcurrencyLimit{
				concurrencyKeyID(enums.ConcurrencyScopeFn, hashedConcurrencyKeyExpr): {
					Scope: enums.ConcurrencyScopeFn,
					Key:   hashedConcurrencyKeyExpr,
					Limit: 23,
				},
			},
			Throttle: &ShadowPartitionThrottle{
				ThrottleKeyExpressionHash: hashedThrottleKeyExpr,
				Limit:                     70,
				Burst:                     20,
				Period:                    600,
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
					KeyExpressionHash:   hashedThrottleKeyExpr,
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

func TestBacklogIsOutdated(t *testing.T) {
	t.Run("same config should not be marked as outdated", func(t *testing.T) {
		keyHash := util.XXHash("event.data.customerID")

		shadowPart := &QueueShadowPartition{
			CustomConcurrencyKeys: map[string]CustomConcurrencyLimit{
				concurrencyKeyID(enums.ConcurrencyScopeFn, keyHash): {
					Scope: enums.ConcurrencyScopeFn,
					Key:   keyHash,
					Limit: 10,
				},
			},
		}
		backlog := &QueueBacklog{
			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					ConcurrencyScope:            enums.ConcurrencyScopeFn,
					ConcurrencyScopeEntity:      uuid.UUID{},
					ConcurrencyKey:              keyHash,
					ConcurrencyKeyValue:         util.XXHash("xyz"),
					ConcurrencyKeyUnhashedValue: "xyz",
				},
			},
			Throttle: nil,
		}

		require.False(t, backlog.isOutdated(shadowPart))
	})

	t.Run("adding concurrency keys should not mark default partition as outdated", func(t *testing.T) {
		keyHash := util.XXHash("event.data.customerID")

		shadowPart := &QueueShadowPartition{
			CustomConcurrencyKeys: map[string]CustomConcurrencyLimit{
				concurrencyKeyID(enums.ConcurrencyScopeFn, keyHash): {
					Scope: enums.ConcurrencyScopeFn,
					Key:   keyHash,
					Limit: 10,
				},
			},
		}
		backlog := &QueueBacklog{}

		require.False(t, backlog.isOutdated(shadowPart))
	})

	t.Run("changing concurrency key should mark as outdated", func(t *testing.T) {
		keyHashOld := util.XXHash("event.data.customerID")
		keyHashNew := util.XXHash("event.data.orgID")

		shadowPart := &QueueShadowPartition{
			CustomConcurrencyKeys: map[string]CustomConcurrencyLimit{
				concurrencyKeyID(enums.ConcurrencyScopeFn, keyHashNew): {
					Scope: enums.ConcurrencyScopeFn,
					Key:   keyHashNew,
					Limit: 10,
				},
			},
		}
		backlog := &QueueBacklog{
			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					ConcurrencyScope:            enums.ConcurrencyScopeFn,
					ConcurrencyScopeEntity:      uuid.UUID{},
					ConcurrencyKey:              keyHashOld,
					ConcurrencyKeyValue:         util.XXHash("xyz"),
					ConcurrencyKeyUnhashedValue: "xyz",
				},
			},
		}

		require.True(t, backlog.isOutdated(shadowPart))
	})

	t.Run("removing concurrency key should mark as outdated", func(t *testing.T) {
		keyHashOld := util.XXHash("event.data.customerID")

		shadowPart := &QueueShadowPartition{
			CustomConcurrencyKeys: nil,
		}
		backlog := &QueueBacklog{
			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					ConcurrencyScope:            enums.ConcurrencyScopeFn,
					ConcurrencyScopeEntity:      uuid.UUID{},
					ConcurrencyKey:              keyHashOld,
					ConcurrencyKeyValue:         util.XXHash("xyz"),
					ConcurrencyKeyUnhashedValue: "xyz",
				},
			},
		}

		require.True(t, backlog.isOutdated(shadowPart))
	})

	t.Run("changing throttle key should mark as outdated", func(t *testing.T) {
		keyHashOld := util.XXHash("event.data.customerID")
		keyHashNew := util.XXHash("event.data.orgID")

		shadowPart := &QueueShadowPartition{
			Throttle: &ShadowPartitionThrottle{
				ThrottleKeyExpressionHash: keyHashNew,
			},
		}
		backlog := &QueueBacklog{
			Throttle: &BacklogThrottle{
				ThrottleKeyExpressionHash: keyHashOld,
			},
		}

		require.True(t, backlog.isOutdated(shadowPart))
	})

	t.Run("removing throttle key should mark as outdated", func(t *testing.T) {
		keyHashOld := util.XXHash("event.data.customerID")

		shadowPart := &QueueShadowPartition{
			Throttle: nil,
		}
		backlog := &QueueBacklog{
			Throttle: &BacklogThrottle{
				ThrottleKeyExpressionHash: keyHashOld,
			},
		}

		require.True(t, backlog.isOutdated(shadowPart))
	})
}
