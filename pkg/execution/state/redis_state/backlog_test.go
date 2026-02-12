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
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
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

	ctx := context.Background()

	fnID, wsID, accID := uuid.New(), uuid.New(), uuid.New()

	t.Run("basic edge item", func(t *testing.T) {
		expected := osqueue.QueueBacklog{
			// expect default backlog to be used
			BacklogID:         fmt.Sprintf("fn:%s", fnID),
			ShadowPartitionID: fnID.String(),
		}

		backlog := osqueue.ItemBacklog(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 2))
	})

	t.Run("basic start item", func(t *testing.T) {
		expected := osqueue.QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:start", fnID),
			Start:             true,
			ShadowPartitionID: fnID.String(),
		}

		backlog := osqueue.ItemBacklog(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 2))
	})

	t.Run("system queue", func(t *testing.T) {
		sysQueueName := osqueue.KindQueueMigrate

		expected := osqueue.QueueBacklog{
			// expect default backlog to be used
			BacklogID:         fmt.Sprintf("system:%s", sysQueueName),
			ShadowPartitionID: sysQueueName,
		}

		backlog := osqueue.ItemBacklog(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 2))
	})

	t.Run("throttle", func(t *testing.T) {
		throttleKeyExpressionHash := util.XXHash("event.data.customerID")
		rawThrottleKey := fnID.String()
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := osqueue.QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:start:t<%s:%s>", fnID, throttleKeyExpressionHash, hashedThrottleKey),
			Start:             true,
			ShadowPartitionID: fnID.String(),

			Throttle: &osqueue.BacklogThrottle{
				ThrottleKey:               hashedThrottleKey,
				ThrottleKeyRawValue:       rawThrottleKey,
				ThrottleKeyExpressionHash: throttleKeyExpressionHash,
			},
		}

		backlog := osqueue.ItemBacklog(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 2))
	})

	t.Run("throttle with key", func(t *testing.T) {
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)
		exprHash := util.XXHash(rawThrottleKey)

		expected := osqueue.QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:start:t<%s:%s>", fnID, exprHash, hashedThrottleKey),
			Start:             true,
			ShadowPartitionID: fnID.String(),

			Throttle: &osqueue.BacklogThrottle{
				ThrottleKey:               hashedThrottleKey,
				ThrottleKeyRawValue:       rawThrottleKey,
				ThrottleKeyExpressionHash: exprHash,
			},
		}

		backlog := osqueue.ItemBacklog(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 2))
	})

	t.Run("throttle on edge item", func(t *testing.T) {
		rawThrottleKey := fnID.String()
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := osqueue.QueueBacklog{
			// edge should go to default backlog if no concurrency keys are specified
			BacklogID:         fmt.Sprintf("fn:%s", fnID),
			ShadowPartitionID: fnID.String(),
		}

		backlog := osqueue.ItemBacklog(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 2))
	})

	t.Run("function concurrency", func(t *testing.T) {
		expected := osqueue.QueueBacklog{
			// expect default backlog to be used
			BacklogID:         fmt.Sprintf("fn:%s", fnID),
			ShadowPartitionID: fnID.String(),
		}

		backlog := osqueue.ItemBacklog(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 2))
	})

	t.Run("account concurrency", func(t *testing.T) {
		expected := osqueue.QueueBacklog{
			// expect default backlog to be used
			BacklogID:         fmt.Sprintf("fn:%s", fnID),
			ShadowPartitionID: fnID.String(),
		}

		backlog := osqueue.ItemBacklog(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 2))
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

		expected := osqueue.QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:c1<%s:%s>", fnID, hashedConcurrencyKeyExpr, util.XXHash(fullKey)),
			ShadowPartitionID: fnID.String(),

			ConcurrencyKeys: []osqueue.BacklogConcurrencyKey{
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

		backlog := osqueue.ItemBacklog(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope, entity, unhashedValue)), backlogCustomKeyInProgress(backlog, kg, 1))
		require.Equal(t, kg.Concurrency("", ""), backlogCustomKeyInProgress(backlog, kg, 2))
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

		expected := osqueue.QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:start:t<%s:%s>:c1<%s:%s>", fnID, hashedThrottleExpr, hashedThrottleExpr, hashedConcurrencyKeyExpr, util.XXHash(fullKey)),
			Start:             true,
			ShadowPartitionID: fnID.String(),

			Throttle: &osqueue.BacklogThrottle{
				ThrottleKey:               hashedThrottleKey,
				ThrottleKeyRawValue:       rawThrottleKey,
				ThrottleKeyExpressionHash: hashedThrottleExpr,
			},

			ConcurrencyKeys: []osqueue.BacklogConcurrencyKey{
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

		backlog := osqueue.ItemBacklog(ctx, osqueue.QueueItem{
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

		expected := osqueue.QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:c1<%s:%s>", fnID, hashedConcurrencyKeyExpr, util.XXHash(fullKey)),
			ShadowPartitionID: fnID.String(),

			ConcurrencyKeys: []osqueue.BacklogConcurrencyKey{
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

		backlog := osqueue.ItemBacklog(ctx, osqueue.QueueItem{
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

		expected := osqueue.QueueBacklog{
			BacklogID:         fmt.Sprintf("fn:%s:c1<%s:%s>:c2<%s:%s>", fnID, hashedConcurrencyKeyExpr, util.XXHash(fullKey), hashedConcurrencyKeyExpr2, util.XXHash(fullKey2)),
			ShadowPartitionID: fnID.String(),

			ConcurrencyKeys: []osqueue.BacklogConcurrencyKey{
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

		backlog := osqueue.ItemBacklog(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope, entity, unhashedValue)), backlogCustomKeyInProgress(backlog, kg, 1))
		require.Equal(t, kg.Concurrency("custom", util.ConcurrencyKey(scope2, entity2, unhashedValue2)), backlogCustomKeyInProgress(backlog, kg, 2))
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

	ctx := context.Background()

	fnID, wsID, accID := uuid.New(), uuid.New(), uuid.New()

	t.Run("basic item", func(t *testing.T) {
		expected := osqueue.QueueShadowPartition{
			PartitionID:     fnID.String(),
			FunctionID:      &fnID,
			EnvID:           &wsID,
			AccountID:       &accID,
			SystemQueueName: nil,
		}

		shadowPart := osqueue.ItemShadowPartition(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("p", fnID.String()), shadowPartitionInProgressKey(shadowPart, kg))
		require.Equal(t, kg.Concurrency("account", accID.String()), shadowPartitionAccountInProgressKey(shadowPart, kg))
	})

	t.Run("system queue", func(t *testing.T) {
		sysQueueName := osqueue.KindQueueMigrate

		expected := osqueue.QueueShadowPartition{
			PartitionID:     sysQueueName,
			FunctionID:      nil,
			EnvID:           nil,
			AccountID:       nil,
			SystemQueueName: &sysQueueName,
		}

		shadowPart := osqueue.ItemShadowPartition(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("p", sysQueueName), shadowPartitionInProgressKey(shadowPart, kg))

		// expect empty key: system queues should not track account concurrency
		require.Equal(t, kg.Concurrency("account", ""), shadowPartitionAccountInProgressKey(shadowPart, kg))
	})

	t.Run("throttle", func(t *testing.T) {
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := osqueue.QueueShadowPartition{
			PartitionID:     fnID.String(),
			FunctionID:      &fnID,
			EnvID:           &wsID,
			AccountID:       &accID,
			SystemQueueName: nil,
		}

		shadowPart := osqueue.ItemShadowPartition(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("p", fnID.String()), shadowPartitionInProgressKey(shadowPart, kg))
		require.Equal(t, kg.Concurrency("account", accID.String()), shadowPartitionAccountInProgressKey(shadowPart, kg))
	})

	t.Run("custom concurrency", func(t *testing.T) {
		hashedConcurrencyKeyExpr := hashConcurrencyKey("event.data.customerId")
		unhashedValue := "customer1"
		fullKey := util.ConcurrencyKey(enums.ConcurrencyScopeFn, fnID, unhashedValue)

		expected := osqueue.QueueShadowPartition{
			PartitionID:     fnID.String(),
			FunctionID:      &fnID,
			EnvID:           &wsID,
			AccountID:       &accID,
			SystemQueueName: nil,
		}

		shadowPart := osqueue.ItemShadowPartition(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("p", fnID.String()), shadowPartitionInProgressKey(shadowPart, kg))
		require.Equal(t, kg.Concurrency("account", accID.String()), shadowPartitionAccountInProgressKey(shadowPart, kg))
	})

	t.Run("concurrency + throttle", func(t *testing.T) {
		hashedThrottleKeyExpr := hashConcurrencyKey("event.data.customerId")
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		hashedConcurrencyKeyExpr := hashConcurrencyKey("event.data.customerId")
		unhashedValue := "customer1"
		fullKey := util.ConcurrencyKey(enums.ConcurrencyScopeFn, fnID, unhashedValue)

		expected := osqueue.QueueShadowPartition{
			PartitionID:     fnID.String(),
			FunctionID:      &fnID,
			EnvID:           &wsID,
			AccountID:       &accID,
			SystemQueueName: nil,
		}

		shadowPart := osqueue.ItemShadowPartition(ctx, osqueue.QueueItem{
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
		require.Equal(t, kg.Concurrency("p", fnID.String()), shadowPartitionInProgressKey(shadowPart, kg))
		require.Equal(t, kg.Concurrency("account", accID.String()), shadowPartitionAccountInProgressKey(shadowPart, kg))
	})
}

func hashConcurrencyKey(key string) string {
	return strconv.FormatUint(xxhash.Sum64String(key), 36)
}

func TestBacklogIsOutdated(t *testing.T) {
	t.Run("same config should not be marked as outdated", func(t *testing.T) {
		keyHash := util.XXHash("event.data.customerID")

		concurrency := osqueue.PartitionConcurrency{
			CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
				{
					Scope:               enums.ConcurrencyScopeFn,
					HashedKeyExpression: keyHash,
					Limit:               10,
				},
			},
		}

		constraints := osqueue.PartitionConstraintConfig{
			Concurrency: concurrency,
		}

		backlog := &osqueue.QueueBacklog{
			ConcurrencyKeys: []osqueue.BacklogConcurrencyKey{
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

		require.Equal(t, enums.QueueNormalizeReasonUnchanged, backlog.IsOutdated(constraints))
	})

	t.Run("adding concurrency keys should not mark default partition as outdated", func(t *testing.T) {
		keyHash := util.XXHash("event.data.customerID")

		constraints := osqueue.PartitionConstraintConfig{
			Concurrency: osqueue.PartitionConcurrency{
				CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
					{
						Scope:               enums.ConcurrencyScopeFn,
						HashedKeyExpression: keyHash,
						Limit:               10,
					},
				},
			},
		}
		backlog := &osqueue.QueueBacklog{}

		require.Equal(t, enums.QueueNormalizeReasonUnchanged, backlog.IsOutdated(constraints))
	})

	t.Run("changing concurrency key should mark as outdated", func(t *testing.T) {
		keyHashOld := util.XXHash("event.data.customerID")
		keyHashNew := util.XXHash("event.data.orgID")

		constraints := osqueue.PartitionConstraintConfig{
			Concurrency: osqueue.PartitionConcurrency{
				CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{
					{
						Scope:               enums.ConcurrencyScopeFn,
						HashedKeyExpression: keyHashNew,
						Limit:               10,
					},
				},
			},
		}
		backlog := &osqueue.QueueBacklog{
			ConcurrencyKeys: []osqueue.BacklogConcurrencyKey{
				{
					Scope:               enums.ConcurrencyScopeFn,
					EntityID:            uuid.UUID{},
					HashedKeyExpression: keyHashOld,
					HashedValue:         util.XXHash("xyz"),
					UnhashedValue:       "xyz",
				},
			},
		}

		require.Equal(t, enums.QueueNormalizeReasonCustomConcurrencyKeyNotFoundOnShadowPartition, backlog.IsOutdated(constraints))
	})

	t.Run("removing concurrency key should mark as outdated", func(t *testing.T) {
		keyHashOld := util.XXHash("event.data.customerID")

		constraints := osqueue.PartitionConstraintConfig{
			Concurrency: osqueue.PartitionConcurrency{
				CustomConcurrencyKeys: nil,
			},
		}
		backlog := &osqueue.QueueBacklog{
			ConcurrencyKeys: []osqueue.BacklogConcurrencyKey{
				{
					Scope:               enums.ConcurrencyScopeFn,
					EntityID:            uuid.UUID{},
					HashedKeyExpression: keyHashOld,
					HashedValue:         util.XXHash("xyz"),
					UnhashedValue:       "xyz",
				},
			},
		}

		require.Equal(t, enums.QueueNormalizeReasonCustomConcurrencyKeyCountMismatch, backlog.IsOutdated(constraints))
	})

	t.Run("changing throttle key should mark as outdated", func(t *testing.T) {
		keyHashOld := util.XXHash("event.data.customerID")
		keyHashNew := util.XXHash("event.data.orgID")

		constraints := osqueue.PartitionConstraintConfig{
			Throttle: &osqueue.PartitionThrottle{
				ThrottleKeyExpressionHash: keyHashNew,
			},
		}
		backlog := &osqueue.QueueBacklog{
			Throttle: &osqueue.BacklogThrottle{
				ThrottleKeyExpressionHash: keyHashOld,
			},
		}

		require.Equal(t, enums.QueueNormalizeReasonThrottleKeyChanged, backlog.IsOutdated(constraints))
	})

	t.Run("same throttle key should not mark as outdated", func(t *testing.T) {
		keyHash := util.XXHash("event.data.orgID")

		constraints := osqueue.PartitionConstraintConfig{
			Throttle: &osqueue.PartitionThrottle{
				ThrottleKeyExpressionHash: keyHash,
			},
		}
		backlog := &osqueue.QueueBacklog{
			Throttle: &osqueue.BacklogThrottle{
				ThrottleKeyExpressionHash: keyHash,
			},
		}

		require.Equal(t, enums.QueueNormalizeReasonUnchanged, backlog.IsOutdated(constraints))
	})

	t.Run("removing throttle key should mark as outdated", func(t *testing.T) {
		keyHashOld := util.XXHash("event.data.customerID")

		constraints := osqueue.PartitionConstraintConfig{
			Throttle: nil,
		}
		backlog := &osqueue.QueueBacklog{
			Throttle: &osqueue.BacklogThrottle{
				ThrottleKeyExpressionHash: keyHashOld,
			},
		}

		require.Equal(t, enums.QueueNormalizeReasonThrottleRemoved, backlog.IsOutdated(constraints))
	})
}

func TestShuffleBacklogs(t *testing.T) {
	iterations := 1000

	matches := 0

	for i := 0; i < iterations; i++ {
		b1Start := &osqueue.QueueBacklog{
			BacklogID: "b-1:start",
			Start:     true,
		}

		b1 := &osqueue.QueueBacklog{
			BacklogID: "b-1",
		}

		b2Start := &osqueue.QueueBacklog{
			BacklogID: "b-2:start",
			Start:     true,
		}

		b2 := &osqueue.QueueBacklog{
			BacklogID: "b-2",
		}

		b3Start := &osqueue.QueueBacklog{
			BacklogID: "b-3:start",
			Start:     true,
		}

		b3 := &osqueue.QueueBacklog{
			BacklogID: "b-3",
		}

		shuffled := osqueue.ShuffleBacklogs([]*osqueue.QueueBacklog{
			b1,
			b1Start,
			b2,
			b2Start,
			b3,
			b3Start,
		})

		findIndex := func(b *osqueue.QueueBacklog) int {
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

			q, shard := newQueue(
				t, rc,
				osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
					return true
				}),
				osqueue.WithClock(clock),
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

				_, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)
			}

			items, err := q.BacklogsByPartition(ctx, shard, fnID.String(), tc.from, tc.until,
				osqueue.WithQueueItemIterBatchSize(tc.batchSize),
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

	q, shard := newQueue(
		t, rc,
		osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return osqueue.PartitionConstraintConfig{
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency:  100,
					FunctionConcurrency: 25,
				},
			}
		}),
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
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
			backlog := osqueue.ItemBacklog(ctx, item)
			backlogID = backlog.BacklogID
		}

		_, err := shard.EnqueueItem(ctx, item, time.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)
	}
	require.NotEmpty(t, backlogID)

	size, err := q.BacklogSize(ctx, shard, backlogID)
	require.NoError(t, err)

	require.EqualValues(t, count, size)
}

func TestPartitionBacklogSize(t *testing.T) {
	r1, rc1 := initRedis(t)
	defer rc1.Close()

	r2, rc2 := initRedis(t)
	defer rc2.Close()

	ctx := context.Background()
	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelInfo))
	ctx = logger.WithStdlib(ctx, l)

	clock := clockwork.NewFakeClock()

	opts := []osqueue.QueueOpt{
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithClock(clock),
	}

	shard1 := shardFromClient("one", rc1, opts...)
	shard2 := shardFromClient("two", rc2, opts...)
	queueShards := mapFromShards(shard1, shard2)

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

			q1, err := osqueue.New(
				ctx,
				"q1",
				shard1,
				queueShards,
				func(ctx context.Context, accountId uuid.UUID, queueName *string) (osqueue.QueueShard, error) {
					return shard1, nil
				},
				opts...,
			)
			require.NoError(t, err)
			q2, err := osqueue.New(
				ctx,
				"q2",
				shard2,
				queueShards,
				func(ctx context.Context, accountId uuid.UUID, queueName *string) (osqueue.QueueShard, error) {
					return shard2, nil
				},
				opts...,
			)
			require.NoError(t, err)

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
						_, err := shard1.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
						require.NoError(t, err)
					case 1:
						_, err := shard2.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
						require.NoError(t, err)
					}
				} else {
					_, err := shard1.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
					require.NoError(t, err)
				}
			}

			// NOTE: should return the same result regardless of which shard initiated the instrumentation
			size1, err := q1.PartitionBacklogSize(ctx, fnID.String())
			require.NoError(t, err)
			require.EqualValues(t, int64(tc.num), size1)

			size2, err := q2.PartitionBacklogSize(ctx, fnID.String())
			require.NoError(t, err)
			require.EqualValues(t, int64(tc.num), size2)
		})
	}
}

func TestShadowPartitionFunctionBacklog(t *testing.T) {
	fnID, envID, accountID := uuid.New(), uuid.New(), uuid.New()
	sysQueueName := "test-system-queue"

	t.Run("system queue backlog should work", func(t *testing.T) {
		sp := osqueue.QueueShadowPartition{
			SystemQueueName: &sysQueueName,
		}

		constraints := osqueue.PartitionConstraintConfig{
			Concurrency: osqueue.PartitionConcurrency{},
		}

		b := sp.DefaultBacklog(constraints, false)

		require.Equal(t, &osqueue.QueueBacklog{
			BacklogID:         fmt.Sprintf("system:%s", sysQueueName),
			ShadowPartitionID: sysQueueName,
		}, b)
	})

	t.Run("empty queue backlog should not work", func(t *testing.T) {
		sp := osqueue.QueueShadowPartition{}

		constraints := osqueue.PartitionConstraintConfig{
			Concurrency: osqueue.PartitionConcurrency{},
		}

		b := sp.DefaultBacklog(constraints, false)
		require.Nil(t, b)
	})

	t.Run("non-start backlog should work", func(t *testing.T) {
		sp := osqueue.QueueShadowPartition{
			FunctionVersion: 1,
			FunctionID:      &fnID,
			SystemQueueName: nil,
			LeaseID:         nil,
			EnvID:           &envID,
			PartitionID:     fnID.String(),
			AccountID:       &accountID,
		}

		constraints := osqueue.PartitionConstraintConfig{
			FunctionVersion: 2,
			Concurrency:     osqueue.PartitionConcurrency{},
		}

		b := sp.DefaultBacklog(constraints, false)

		require.Equal(t, &osqueue.QueueBacklog{
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
		sp := osqueue.QueueShadowPartition{
			FunctionVersion: 1,
			FunctionID:      &fnID,
			SystemQueueName: nil,
			LeaseID:         nil,
			EnvID:           &envID,
			PartitionID:     fnID.String(),
			AccountID:       &accountID,
		}

		constraints := osqueue.PartitionConstraintConfig{
			FunctionVersion: 2,
			Concurrency:     osqueue.PartitionConcurrency{},
		}

		b := sp.DefaultBacklog(constraints, true)

		require.Equal(t, &osqueue.QueueBacklog{
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
		sp := osqueue.QueueShadowPartition{
			FunctionVersion: 1,
			FunctionID:      &fnID,
			SystemQueueName: nil,
			LeaseID:         nil,
			EnvID:           &envID,
			PartitionID:     fnID.String(),
			AccountID:       &accountID,
		}

		constraints := osqueue.PartitionConstraintConfig{
			FunctionVersion: 2,
			Concurrency:     osqueue.PartitionConcurrency{},
			Throttle: &osqueue.PartitionThrottle{
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
		sp := osqueue.QueueShadowPartition{
			FunctionVersion: 1,
			FunctionID:      &fnID,
			SystemQueueName: nil,
			LeaseID:         nil,
			EnvID:           &envID,
			PartitionID:     fnID.String(),
			AccountID:       &accountID,
		}

		constraints := osqueue.PartitionConstraintConfig{
			FunctionVersion: 2,
			Concurrency:     osqueue.PartitionConcurrency{},
			Throttle: &osqueue.PartitionThrottle{
				ThrottleKeyExpressionHash: "expr-hash",
				Limit:                     1,
				Burst:                     1,
				Period:                    60,
			},
		}

		b := sp.DefaultBacklog(constraints, false)

		require.Equal(t, &osqueue.QueueBacklog{
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
