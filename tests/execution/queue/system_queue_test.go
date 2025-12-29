package queue

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/debounce"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestSystemQueueConfigs(t *testing.T) {
	mapping := map[string]string{
		osqueue.KindScheduleBatch: osqueue.KindScheduleBatch,
		"pause-event":             "pause-event",
		osqueue.KindDebounce:      osqueue.KindDebounce,
		osqueue.KindQueueMigrate:  osqueue.KindQueueMigrate,
	}

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	batchClient := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)

	defaultShard := redis_state.QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey), Name: consts.DefaultQueueShardName}
	kg := defaultShard.RedisClient.KeyGenerator()

	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
	now := clock.Now()

	q := redis_state.NewQueue(
		defaultShard,
		redis_state.WithClock(clock),
		redis_state.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
			return false
		}),
		redis_state.WithKindToQueueMapping(mapping),
		redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.QueueShard, error) {
			return defaultShard, nil
		}),
	)
	ctx := context.Background()

	accountId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	t.Run("batch items should be added to system queue", func(t *testing.T) {
		r.FlushAll()

		batcher := batch.NewRedisBatchManager(batchClient, q)

		batchID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)
		bp := "test-pointer"

		err := batcher.ScheduleExecution(ctx, batch.ScheduleBatchOpts{
			ScheduleBatchPayload: batch.ScheduleBatchPayload{
				BatchID:         batchID,
				BatchPointer:    bp,
				AccountID:       accountId,
				WorkspaceID:     wsID,
				FunctionID:      fnID,
				FunctionVersion: 0,
			},
			At: now,
		})
		require.NoError(t, err)

		require.True(t, r.Exists(kg.PartitionQueueSet(enums.PartitionTypeDefault, osqueue.KindScheduleBatch, "")))
		require.True(t, r.Exists(kg.AccountPartitionIndex(accountId)))
		require.True(t, r.Exists(kg.GlobalAccountIndex()))
		require.True(t, hasMember(t, r, kg.GlobalAccountIndex(), accountId.String()))
	})

	t.Run("debounce timeouts should not be added to function queue", func(t *testing.T) {
		r.FlushAll()

		debouncer, err := debounce.NewRedisDebouncerWithMigration(debounce.DebouncerOpts{
			PrimaryDebounceClient: redis_state.NewDebounceClient(rc, redis_state.QueueDefaultKey),
			PrimaryQueue:          q,
			PrimaryQueueShard:     defaultShard,
			ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
				return false
			},
			Clock: clock,
		})
		require.NoError(t, err)

		debounceID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

		tester := debouncer.(debounce.DebounceTest)

		qi := tester.TestQueueItem(ctx, debounce.DebounceItem{
			AccountID:   accountId,
			WorkspaceID: wsID,
			FunctionID:  fnID,
		}, debounceID)
		require.NoError(t, err)

		err = q.Enqueue(ctx, qi, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		require.True(t, r.Exists(kg.PartitionQueueSet(enums.PartitionTypeDefault, osqueue.KindDebounce, "")))
		require.True(t, r.Exists(kg.AccountPartitionIndex(accountId)))
		require.True(t, r.Exists(kg.GlobalAccountIndex()))
		require.True(t, hasMember(t, r, kg.GlobalAccountIndex(), accountId.String()))
	})

	t.Run("pause timeouts should belong to fn queue", func(t *testing.T) {
		r.FlushAll()

		err := q.Enqueue(ctx, osqueue.Item{
			WorkspaceID: wsID,
			Kind:        osqueue.KindPause,
			Identifier: state.Identifier{
				WorkflowID:  fnID,
				AccountID:   accountId,
				WorkspaceID: wsID,
			},
			Payload: nil,
		}, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		require.True(t, r.Exists(kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), "")), r.Dump())
		require.True(t, r.Exists(kg.AccountPartitionIndex(accountId)))
		require.True(t, r.Exists(kg.GlobalAccountIndex()))
		require.True(t, hasMember(t, r, kg.GlobalAccountIndex(), accountId.String()))
	})

	t.Run("pause events should not belong to fn queue", func(t *testing.T) {
		r.FlushAll()

		queueName := fmt.Sprintf("pause:%s", wsID.String())
		err := q.Enqueue(ctx, osqueue.Item{
			QueueName: &queueName, WorkspaceID: wsID,
			Kind:    "pause-event",
			Payload: nil,
		}, now, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		require.True(t, r.Exists(kg.PartitionQueueSet(enums.PartitionTypeDefault, queueName, "")))
		require.False(t, r.Exists(kg.AccountPartitionIndex(accountId)))
		require.False(t, r.Exists(kg.GlobalAccountIndex()))
		require.False(t, hasMember(t, r, kg.GlobalAccountIndex(), accountId.String()))
	})
}

func hasMember(t *testing.T, r *miniredis.Miniredis, key string, member string) bool {
	if !r.Exists(key) {
		return false
	}

	members, err := r.ZMembers(key)
	require.NoError(t, err)

	for _, s := range members {
		if s == member {
			return true
		}
	}
	return false
}
