package redis_state

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"

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

	t.Run("basic edge item", func(t *testing.T) {
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
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))
	})

	t.Run("basic start item", func(t *testing.T) {
		expected := QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:start", fnID),
			Start:             true,
			ShadowPartitionID: fnID.String(),
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
				Throttle:              nil,
				CustomConcurrencyKeys: nil,
				QueueName:             nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, backlog)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-level concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))
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
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))

	})

	t.Run("throttle", func(t *testing.T) {
		throttleKeyExpressionHash := util.XXHash("event.data.customerID")
		rawThrottleKey := fnID.String()
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:start:t<%s:%s>", fnID, throttleKeyExpressionHash, hashedThrottleKey),
			Start:             true,
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
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))
	})

	t.Run("throttle with key", func(t *testing.T) {
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)
		exprHash := util.XXHash(rawThrottleKey)

		expected := QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:start:t<%s:%s>", fnID, exprHash, hashedThrottleKey),
			Start:             true,
			ShadowPartitionID: fnID.String(),

			Throttle: &BacklogThrottle{
				ThrottleKey:               hashedThrottleKey,
				ThrottleKeyRawValue:       rawThrottleKey,
				ThrottleKeyExpressionHash: exprHash,
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
					KeyExpressionHash:   exprHash,
				},
				CustomConcurrencyKeys: nil,
				QueueName:             nil,
			},
			QueueName: nil,
		})

		require.Equal(t, expected, backlog)

		// default backlog is not for a concurrency key, so the concurrency key should be empty (function-llevel concurrency accounting is handled for the shadow partition)
		kg := queueKeyGenerator{queueDefaultKey: QueueDefaultKey}
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))
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
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))
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
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))
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
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))
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
			BacklogID:         fmt.Sprintf("fn:%s:c1<%s:%s>", fnID, hashedConcurrencyKeyExpr, util.XXHash(fullKey)),
			ShadowPartitionID: fnID.String(),

			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					CanonicalKeyID:      fullKey,
					Scope:               scope,
					EntityID:            entity,
					HashedKeyExpression: hashedConcurrencyKeyExpr,
					HashedValue:         checksum,
					UnhashedValue:       unhashedValue,
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
		require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope, entity, unhashedValue)), backlog.customKeyInProgress(kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlog.customKeyInProgress(kg, 2))
	})

	t.Run("concurrency + throttle start item", func(t *testing.T) {
		rawThrottleKey := "customer1"
		hashedThrottleExpr := util.XXHash(rawThrottleKey)
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
			BacklogID:         fmt.Sprintf("fn:%s:start:t<%s:%s>:c1<%s:%s>", fnID, hashedThrottleExpr, hashedThrottleExpr, hashedConcurrencyKeyExpr, util.XXHash(fullKey)),
			Start:             true,
			ShadowPartitionID: fnID.String(),

			Throttle: &BacklogThrottle{
				ThrottleKey:               hashedThrottleKey,
				ThrottleKeyRawValue:       rawThrottleKey,
				ThrottleKeyExpressionHash: hashedThrottleExpr,
			},

			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					CanonicalKeyID:      fullKey,
					Scope:               scope,
					EntityID:            entity,
					HashedKeyExpression: hashedConcurrencyKeyExpr,
					HashedValue:         checksum,
					UnhashedValue:       unhashedValue,
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
					KeyExpressionHash:   hashedThrottleExpr,
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
			BacklogID:         fmt.Sprintf("fn:%s:c1<%s:%s>", fnID, hashedConcurrencyKeyExpr, util.XXHash(fullKey)),
			ShadowPartitionID: fnID.String(),

			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					CanonicalKeyID:      fullKey,
					Scope:               scope,
					HashedKeyExpression: hashedConcurrencyKeyExpr,
					EntityID:            entity,
					HashedValue:         checksum,
					UnhashedValue:       unhashedValue,
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
			BacklogID:         fmt.Sprintf("fn:%s:c1<%s:%s>:c2<%s:%s>", fnID, hashedConcurrencyKeyExpr, util.XXHash(fullKey), hashedConcurrencyKeyExpr2, util.XXHash(fullKey2)),
			ShadowPartitionID: fnID.String(),

			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					CanonicalKeyID:      fullKey,
					Scope:               scope,
					EntityID:            entity,
					HashedKeyExpression: hashedConcurrencyKeyExpr,
					HashedValue:         checksum,
					UnhashedValue:       unhashedValue,
				},
				{
					CanonicalKeyID:      fullKey2,
					Scope:               scope2,
					EntityID:            entity2,
					HashedKeyExpression: hashedConcurrencyKeyExpr2,
					HashedValue:         checksum2,
					UnhashedValue:       unhashedValue2,
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
		require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope, entity, unhashedValue)), backlog.customKeyInProgress(kg, 1))
		require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope2, entity2, unhashedValue2)), backlog.customKeyInProgress(kg, 2))
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
			PartitionID:     fnID.String(),
			FunctionID:      &fnID,
			EnvID:           &wsID,
			AccountID:       &accID,
			SystemQueueName: nil,
			Concurrency: ShadowPartitionConcurrency{
				SystemConcurrency:     0,
				AccountConcurrency:    100,
				FunctionConcurrency:   25,
				CustomConcurrencyKeys: nil,
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
			PartitionID:     sysQueueName,
			FunctionID:      nil,
			EnvID:           nil,
			AccountID:       nil,
			SystemQueueName: &sysQueueName,
			Concurrency: ShadowPartitionConcurrency{
				SystemConcurrency:     250,
				AccountConcurrency:    0,
				FunctionConcurrency:   0,
				CustomConcurrencyKeys: nil,
			},
			Throttle:     nil,
			PauseRefill:  false,
			PauseEnqueue: false,
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
		require.Equal(t, kg.Concurrency("account", ""), shadowPart.accountInProgressKey(kg))
	})

	t.Run("throttle", func(t *testing.T) {
		hashedThrottleKeyExpr := util.XXHash("event.data.customerID")
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := QueueShadowPartition{
			PartitionID:     fnID.String(),
			FunctionID:      &fnID,
			EnvID:           &wsID,
			AccountID:       &accID,
			SystemQueueName: nil,
			Concurrency: ShadowPartitionConcurrency{
				SystemConcurrency:     0,
				AccountConcurrency:    100,
				FunctionConcurrency:   25,
				CustomConcurrencyKeys: nil,
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
			PartitionID:     fnID.String(),
			FunctionID:      &fnID,
			EnvID:           &wsID,
			AccountID:       &accID,
			SystemQueueName: nil,
			Concurrency: ShadowPartitionConcurrency{
				SystemConcurrency:   0,
				AccountConcurrency:  100,
				FunctionConcurrency: 25,
				CustomConcurrencyKeys: []CustomConcurrencyLimit{
					{
						Scope:               enums.ConcurrencyScopeFn,
						HashedKeyExpression: hashedConcurrencyKeyExpr,
						Limit:               23,
					},
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
			PartitionID:     fnID.String(),
			FunctionID:      &fnID,
			EnvID:           &wsID,
			AccountID:       &accID,
			SystemQueueName: nil,
			Concurrency: ShadowPartitionConcurrency{
				SystemConcurrency:   0,
				AccountConcurrency:  100,
				FunctionConcurrency: 25,
				CustomConcurrencyKeys: []CustomConcurrencyLimit{
					{
						Scope:               enums.ConcurrencyScopeFn,
						HashedKeyExpression: hashedConcurrencyKeyExpr,
						Limit:               23,
					},
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

		concurrency := ShadowPartitionConcurrency{
			CustomConcurrencyKeys: []CustomConcurrencyLimit{
				{
					Scope:               enums.ConcurrencyScopeFn,
					HashedKeyExpression: keyHash,
					Limit:               10,
				},
			},
		}

		constraints := PartitionConstraintConfig{
			Concurrency: concurrency,
		}

		backlog := &QueueBacklog{
			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					CanonicalKeyID:      fmt.Sprintf("f:%s:%s", uuid.Nil, util.XXHash("xyz")),
					Scope:               enums.ConcurrencyScopeFn,
					EntityID:            uuid.UUID{},
					HashedKeyExpression: keyHash,
					HashedValue:         util.XXHash("xyz"),
					UnhashedValue:       "xyz",
				},
			},
			Throttle: nil,
		}

		require.Equal(t, enums.QueueNormalizeReasonUnchanged, backlog.isOutdated(&constraints))
	})

	t.Run("adding concurrency keys should not mark default partition as outdated", func(t *testing.T) {
		keyHash := util.XXHash("event.data.customerID")

		constraints := &PartitionConstraintConfig{
			Concurrency: ShadowPartitionConcurrency{
				CustomConcurrencyKeys: []CustomConcurrencyLimit{
					{
						Scope:               enums.ConcurrencyScopeFn,
						HashedKeyExpression: keyHash,
						Limit:               10,
					},
				},
			},
		}
		backlog := &QueueBacklog{}

		require.Equal(t, enums.QueueNormalizeReasonUnchanged, backlog.isOutdated(constraints))
	})

	t.Run("changing concurrency key should mark as outdated", func(t *testing.T) {
		keyHashOld := util.XXHash("event.data.customerID")
		keyHashNew := util.XXHash("event.data.orgID")

		constraints := &PartitionConstraintConfig{
			Concurrency: ShadowPartitionConcurrency{
				CustomConcurrencyKeys: []CustomConcurrencyLimit{
					{
						Scope:               enums.ConcurrencyScopeFn,
						HashedKeyExpression: keyHashNew,
						Limit:               10,
					},
				},
			},
		}
		backlog := &QueueBacklog{
			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					Scope:               enums.ConcurrencyScopeFn,
					EntityID:            uuid.UUID{},
					HashedKeyExpression: keyHashOld,
					HashedValue:         util.XXHash("xyz"),
					UnhashedValue:       "xyz",
				},
			},
		}

		require.Equal(t, enums.QueueNormalizeReasonCustomConcurrencyKeyNotFoundOnShadowPartition, backlog.isOutdated(constraints))
	})

	t.Run("removing concurrency key should mark as outdated", func(t *testing.T) {
		keyHashOld := util.XXHash("event.data.customerID")

		constraints := &PartitionConstraintConfig{
			Concurrency: ShadowPartitionConcurrency{
				CustomConcurrencyKeys: nil,
			},
		}
		backlog := &QueueBacklog{
			ConcurrencyKeys: []BacklogConcurrencyKey{
				{
					Scope:               enums.ConcurrencyScopeFn,
					EntityID:            uuid.UUID{},
					HashedKeyExpression: keyHashOld,
					HashedValue:         util.XXHash("xyz"),
					UnhashedValue:       "xyz",
				},
			},
		}

		require.Equal(t, enums.QueueNormalizeReasonCustomConcurrencyKeyCountMismatch, backlog.isOutdated(constraints))
	})

	t.Run("changing throttle key should mark as outdated", func(t *testing.T) {
		keyHashOld := util.XXHash("event.data.customerID")
		keyHashNew := util.XXHash("event.data.orgID")

		constraints := &PartitionConstraintConfig{
			Throttle: &ShadowPartitionThrottle{
				ThrottleKeyExpressionHash: keyHashNew,
			},
		}
		backlog := &QueueBacklog{
			Throttle: &BacklogThrottle{
				ThrottleKeyExpressionHash: keyHashOld,
			},
		}

		require.Equal(t, enums.QueueNormalizeReasonThrottleKeyChanged, backlog.isOutdated(constraints))
	})

	t.Run("same throttle key should not mark as outdated", func(t *testing.T) {
		keyHash := util.XXHash("event.data.orgID")

		constraints := &PartitionConstraintConfig{
			Throttle: &ShadowPartitionThrottle{
				ThrottleKeyExpressionHash: keyHash,
			},
		}
		backlog := &QueueBacklog{
			Throttle: &BacklogThrottle{
				ThrottleKeyExpressionHash: keyHash,
			},
		}

		require.Equal(t, enums.QueueNormalizeReasonUnchanged, backlog.isOutdated(constraints))
	})

	t.Run("removing throttle key should mark as outdated", func(t *testing.T) {
		keyHashOld := util.XXHash("event.data.customerID")

		constraints := &PartitionConstraintConfig{
			Throttle: nil,
		}
		backlog := &QueueBacklog{
			Throttle: &BacklogThrottle{
				ThrottleKeyExpressionHash: keyHashOld,
			},
		}

		require.Equal(t, enums.QueueNormalizeReasonThrottleRemoved, backlog.isOutdated(constraints))
	})
}

func TestShuffleBacklogs(t *testing.T) {
	iterations := 1000

	matches := 0

	for i := 0; i < iterations; i++ {
		b1Start := &QueueBacklog{
			BacklogID: "b-1:start",
			Start:     true,
		}

		b1 := &QueueBacklog{
			BacklogID: "b-1",
		}

		b2Start := &QueueBacklog{
			BacklogID: "b-2:start",
			Start:     true,
		}

		b2 := &QueueBacklog{
			BacklogID: "b-2",
		}

		b3Start := &QueueBacklog{
			BacklogID: "b-3:start",
			Start:     true,
		}

		b3 := &QueueBacklog{
			BacklogID: "b-3",
		}

		shuffled := shuffleBacklogs([]*QueueBacklog{
			b1,
			b1Start,
			b2,
			b2Start,
			b3,
			b3Start,
		})

		findIndex := func(b *QueueBacklog) int {
			for i, backlog := range shuffled {
				if backlog.BacklogID == b.BacklogID {
					return i
				}
			}

			return -1
		}

		if findIndex(b1) > findIndex(b1Start) {
			continue
		}
		if findIndex(b2) > findIndex(b2Start) {
			continue
		}
		if findIndex(b3) > findIndex(b3Start) {
			continue
		}

		matches++
	}

	require.Greater(t, matches, int(math.Ceil(float64(iterations)/2)))
}
