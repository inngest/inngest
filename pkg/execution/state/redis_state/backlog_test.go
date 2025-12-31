package redis_state

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
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
		WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
			return PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					SystemConcurrency:   250,
					AccountConcurrency:  100,
					FunctionConcurrency: 25,
				},
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

	hashedThrottleKeyExpr := util.XXHash("event.data.customerID")

	q := NewQueue(
		QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName},
		WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
			return PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					SystemConcurrency:   250,
					AccountConcurrency:  100,
					FunctionConcurrency: 25,
				},
				Throttle: &PartitionThrottle{
					ThrottleKeyExpressionHash: hashedThrottleKeyExpr,
					Limit:                     70,
					Burst:                     20,
					Period:                    600,
				},
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
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := QueueShadowPartition{
			PartitionID:     fnID.String(),
			FunctionID:      &fnID,
			EnvID:           &wsID,
			AccountID:       &accID,
			SystemQueueName: nil,
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

		concurrency := PartitionConcurrency{
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

		require.Equal(t, enums.QueueNormalizeReasonUnchanged, backlog.isOutdated(constraints))
	})

	t.Run("adding concurrency keys should not mark default partition as outdated", func(t *testing.T) {
		keyHash := util.XXHash("event.data.customerID")

		constraints := PartitionConstraintConfig{
			Concurrency: PartitionConcurrency{
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

		constraints := PartitionConstraintConfig{
			Concurrency: PartitionConcurrency{
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

		constraints := PartitionConstraintConfig{
			Concurrency: PartitionConcurrency{
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

		constraints := PartitionConstraintConfig{
			Throttle: &PartitionThrottle{
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

		constraints := PartitionConstraintConfig{
			Throttle: &PartitionThrottle{
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

		constraints := PartitionConstraintConfig{
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

func TestBacklogsByPartition(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	testcases := []struct {
		name          string
		num           int
		interval      time.Duration
		from          time.Time
		until         time.Time
		expectedItems int
		batchSize     int64
	}{
		{
			name:          "simple",
			num:           10,
			expectedItems: 10,
			until:         clock.Now().Add(time.Minute),
		},
		{
			name:          "with interval",
			num:           100,
			until:         clock.Now().Add(time.Minute),
			interval:      -1 * time.Second,
			expectedItems: 100,
		},
		{
			name:          "with out of range interval",
			num:           10,
			from:          clock.Now(),
			until:         clock.Now().Add(7 * time.Second).Truncate(time.Second),
			interval:      time.Second,
			expectedItems: 7,
		},
		{
			name:          "with batch size",
			num:           500,
			until:         clock.Now().Add(10 * time.Second).Truncate(time.Second),
			interval:      10 * time.Millisecond,
			expectedItems: 500,
			batchSize:     150,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r.FlushAll()

			q := NewQueue(
				defaultShard,
				WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
					return true
				}),
				WithClock(clock),
			)

			for i := range tc.num {
				at := clock.Now()
				if !tc.from.IsZero() {
					at = tc.from
				}
				at = at.Add(time.Duration(i) * tc.interval)

				id := fmt.Sprintf("test%d", i)
				item := osqueue.QueueItem{
					ID:          id,
					FunctionID:  fnID,
					WorkspaceID: wsID,
					Data: osqueue.Item{
						WorkspaceID: wsID,
						Kind:        osqueue.KindEdge,
						Identifier: state.Identifier{
							AccountID:       acctId,
							WorkspaceID:     wsID,
							WorkflowID:      fnID,
							WorkflowVersion: 1,
						},
						CustomConcurrencyKeys: []state.CustomConcurrency{
							{
								Key:   id,
								Hash:  hashConcurrencyKey(id),
								Limit: 10,
							},
						},
					},
				}

				_, err := q.EnqueueItem(ctx, defaultShard, item, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)
			}

			items, err := q.BacklogsByPartition(ctx, defaultShard, fnID.String(), tc.from, tc.until,
				WithQueueItemIterBatchSize(tc.batchSize),
			)
			require.NoError(t, err)

			var count int
			for range items {
				count++
			}

			require.Equal(t, tc.expectedItems, count)
		})
	}
}

func TestBacklogSize(t *testing.T) {
	_, rc := initRedis(t)
	defer rc.Close()

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}

	q := NewQueue(
		defaultShard,
		WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
			return PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					AccountConcurrency:  100,
					FunctionConcurrency: 25,
				},
			}
		}),
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
			return true
		}),
	)
	ctx := context.Background()

	fnID, wsID, accID := uuid.New(), uuid.New(), uuid.New()

	count := 10
	var backlogID string

	for i := range count {
		item := osqueue.QueueItem{
			ID:          fmt.Sprintf("test%d", i),
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
		}

		if backlogID == "" {
			backlog := q.ItemBacklog(ctx, item)
			backlogID = backlog.BacklogID
		}

		_, err := q.EnqueueItem(ctx, defaultShard, item, time.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)
	}
	require.NotEmpty(t, backlogID)

	size, err := q.BacklogSize(ctx, defaultShard, backlogID)
	require.NoError(t, err)

	require.EqualValues(t, count, size)
}

func TestPartitionBacklogSize(t *testing.T) {
	r1, rc1 := initRedis(t)
	defer rc1.Close()

	r2, rc2 := initRedis(t)
	defer rc2.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	shard1 := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc1, QueueDefaultKey), Name: "one"}
	shard2 := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc2, QueueDefaultKey), Name: "two"}
	queueShards := map[string]QueueShard{
		"one": shard1,
		"two": shard2,
	}

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	testcases := []struct {
		name   string
		num    int
		rotate bool
	}{
		{
			name: "enqueue on one shard",
			num:  100,
		},
		{
			name:   "enqueue on both shards",
			num:    200,
			rotate: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r1.FlushAll()
			r2.FlushAll()

			q1 := NewQueue(
				shard1,
				WithQueueShardClients(queueShards),
				WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
					return true
				}),
				WithClock(clock),
			)
			q2 := NewQueue(
				shard2,
				WithQueueShardClients(queueShards),
				WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
					return true
				}),
				WithClock(clock),
			)

			for i := range tc.num {
				id := fmt.Sprintf("test%d", i)
				item := osqueue.QueueItem{
					ID:          id,
					FunctionID:  fnID,
					WorkspaceID: wsID,
					Data: osqueue.Item{
						WorkspaceID: wsID,
						Kind:        osqueue.KindEdge,
						Identifier: state.Identifier{
							AccountID:       acctId,
							WorkspaceID:     wsID,
							WorkflowID:      fnID,
							WorkflowVersion: 1,
						},
						CustomConcurrencyKeys: []state.CustomConcurrency{
							{
								Key:   id,
								Hash:  hashConcurrencyKey(id),
								Limit: 10,
							},
						},
					},
				}

				if tc.rotate {
					// enqueue to both queues, simulate queue migrations
					switch i % 2 {
					case 0:
						_, err := q1.EnqueueItem(ctx, shard1, item, clock.Now(), osqueue.EnqueueOpts{})
						require.NoError(t, err)
					case 1:
						_, err := q2.EnqueueItem(ctx, shard2, item, clock.Now(), osqueue.EnqueueOpts{})
						require.NoError(t, err)
					}
				} else {
					_, err := q1.EnqueueItem(ctx, shard1, item, clock.Now(), osqueue.EnqueueOpts{})
					require.NoError(t, err)
				}
			}

			// NOTE: should return the same result regardless of which shard initiated the instrumentation
			size1, err := q1.PartitionBacklogSize(ctx, fnID.String())
			require.NoError(t, err)
			require.EqualValues(t, tc.num, size1)

			size2, err := q2.PartitionBacklogSize(ctx, fnID.String())
			require.NoError(t, err)
			require.EqualValues(t, tc.num, size2)
		})
	}
}

func TestShadowPartitionFunctionBacklog(t *testing.T) {
	fnID, envID, accountID := uuid.New(), uuid.New(), uuid.New()
	sysQueueName := "test-system-queue"

	t.Run("system queue backlog should work", func(t *testing.T) {
		sp := QueueShadowPartition{
			SystemQueueName: &sysQueueName,
		}

		constraints := PartitionConstraintConfig{
			Concurrency: PartitionConcurrency{},
		}

		b := sp.DefaultBacklog(constraints, false)

		require.Equal(t, &QueueBacklog{
			BacklogID:         fmt.Sprintf("system:%s", sysQueueName),
			ShadowPartitionID: sysQueueName,
		}, b)
	})

	t.Run("empty queue backlog should not work", func(t *testing.T) {
		sp := QueueShadowPartition{}

		constraints := PartitionConstraintConfig{
			Concurrency: PartitionConcurrency{},
		}

		b := sp.DefaultBacklog(constraints, false)
		require.Nil(t, b)
	})

	t.Run("non-start backlog should work", func(t *testing.T) {
		sp := QueueShadowPartition{
			FunctionVersion: 1,
			FunctionID:      &fnID,
			SystemQueueName: nil,
			LeaseID:         nil,
			EnvID:           &envID,
			PartitionID:     fnID.String(),
			AccountID:       &accountID,
		}

		constraints := PartitionConstraintConfig{
			FunctionVersion: 2,
			Concurrency:     PartitionConcurrency{},
		}

		b := sp.DefaultBacklog(constraints, false)

		require.Equal(t, &QueueBacklog{
			BacklogID:                              fmt.Sprintf("fn:%s", fnID),
			ShadowPartitionID:                      fnID.String(),
			EarliestFunctionVersion:                2,
			Start:                                  false,
			Throttle:                               nil,
			ConcurrencyKeys:                        nil,
			SuccessiveThrottleConstrained:          0,
			SuccessiveCustomConcurrencyConstrained: 0,
		}, b)
	})

	t.Run("start backlog should work", func(t *testing.T) {
		sp := QueueShadowPartition{
			FunctionVersion: 1,
			FunctionID:      &fnID,
			SystemQueueName: nil,
			LeaseID:         nil,
			EnvID:           &envID,
			PartitionID:     fnID.String(),
			AccountID:       &accountID,
		}

		constraints := PartitionConstraintConfig{
			FunctionVersion: 2,
			Concurrency:     PartitionConcurrency{},
		}

		b := sp.DefaultBacklog(constraints, true)

		require.Equal(t, &QueueBacklog{
			BacklogID:                              fmt.Sprintf("fn:%s:start", fnID),
			ShadowPartitionID:                      fnID.String(),
			EarliestFunctionVersion:                2,
			Start:                                  true,
			Throttle:                               nil,
			ConcurrencyKeys:                        nil,
			SuccessiveThrottleConstrained:          0,
			SuccessiveCustomConcurrencyConstrained: 0,
		}, b)
	})

	t.Run("throttle backlog should not work", func(t *testing.T) {
		sp := QueueShadowPartition{
			FunctionVersion: 1,
			FunctionID:      &fnID,
			SystemQueueName: nil,
			LeaseID:         nil,
			EnvID:           &envID,
			PartitionID:     fnID.String(),
			AccountID:       &accountID,
		}

		constraints := PartitionConstraintConfig{
			FunctionVersion: 2,
			Concurrency:     PartitionConcurrency{},
			Throttle: &PartitionThrottle{
				ThrottleKeyExpressionHash: "expr-hash",
				Limit:                     1,
				Burst:                     1,
				Period:                    60,
			},
		}

		b := sp.DefaultBacklog(constraints, true)

		require.Nil(t, b)
	})

	t.Run("non start throttle backlog should work", func(t *testing.T) {
		sp := QueueShadowPartition{
			FunctionVersion: 1,
			FunctionID:      &fnID,
			SystemQueueName: nil,
			LeaseID:         nil,
			EnvID:           &envID,
			PartitionID:     fnID.String(),
			AccountID:       &accountID,
		}

		constraints := PartitionConstraintConfig{
			FunctionVersion: 2,
			Concurrency:     PartitionConcurrency{},
			Throttle: &PartitionThrottle{
				ThrottleKeyExpressionHash: "expr-hash",
				Limit:                     1,
				Burst:                     1,
				Period:                    60,
			},
		}

		b := sp.DefaultBacklog(constraints, false)

		require.Equal(t, &QueueBacklog{
			BacklogID:                              fmt.Sprintf("fn:%s", fnID),
			ShadowPartitionID:                      fnID.String(),
			EarliestFunctionVersion:                2,
			Start:                                  false,
			Throttle:                               nil,
			ConcurrencyKeys:                        nil,
			SuccessiveThrottleConstrained:          0,
			SuccessiveCustomConcurrencyConstrained: 0,
		}, b)
	})
}
