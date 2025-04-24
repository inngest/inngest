package redis_state

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestQueueItemBacklogs(t *testing.T) {
	t.Run("basic item", func(t *testing.T) {

	})

	t.Run("system queue", func(t *testing.T) {

	})

	t.Run("throttle", func(t *testing.T) {

	})

	t.Run("function concurrency", func(t *testing.T) {

	})

	t.Run("account concurrency", func(t *testing.T) {

	})

	t.Run("custom concurrency", func(t *testing.T) {

	})

	t.Run("concurrency + throttle", func(t *testing.T) {

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
		expected := ShadowPartition{
			FunctionID:            fnID,
			EnvID:                 wsID,
			AccountID:             accID,
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
	})

	t.Run("system queue", func(t *testing.T) {
		sysQueueName := osqueue.KindQueueMigrate

		expected := ShadowPartition{
			FunctionID:            uuid.UUID{},
			EnvID:                 uuid.UUID{},
			AccountID:             uuid.UUID{},
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
	})

	t.Run("throttle", func(t *testing.T) {
		rawThrottleKey := "customer1"
		hashedThrottleKey := osqueue.HashID(ctx, rawThrottleKey)

		expected := ShadowPartition{
			FunctionID:            fnID,
			EnvID:                 wsID,
			AccountID:             accID,
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
	})

	t.Run("function concurrency", func(t *testing.T) {

	})

	t.Run("account concurrency", func(t *testing.T) {

	})

	t.Run("custom concurrency", func(t *testing.T) {

	})

	t.Run("concurrency + throttle", func(t *testing.T) {

	})
}
