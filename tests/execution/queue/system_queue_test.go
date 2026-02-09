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
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestSystemQueueConfigs(t *testing.T) {
	mapping := map[string]string{
		queue.KindScheduleBatch: queue.KindScheduleBatch,
		"pause-event":           "pause-event",
		queue.KindDebounce:      queue.KindDebounce,
		queue.KindQueueMigrate:  queue.KindQueueMigrate,
	}

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	batchClient := redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey)

	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
	opts := []queue.QueueOpt{
		queue.WithClock(clock),
		queue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
			return false
		}),
		queue.WithKindToQueueMapping(mapping),
	}

	shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey), queue.ShardAssignmentConfig{}, opts...)

	kg := shard.Client().KeyGenerator()

	now := clock.Now()

	q, err := queue.New(
		context.Background(),
		"test-queue",
		shard,
		map[string]queue.QueueShard{
			shard.Name(): shard,
		},
		func(ctx context.Context, accountId uuid.UUID, queueName *string) (queue.QueueShard, error) {
			return shard, nil
		},
		opts...,
	)
	require.NoError(t, err)
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

		require.True(t, r.Exists(kg.PartitionQueueSet(enums.PartitionTypeDefault, queue.KindScheduleBatch, "")))
		require.True(t, r.Exists(kg.AccountPartitionIndex(accountId)))
		require.True(t, r.Exists(kg.GlobalAccountIndex()))
		require.True(t, hasMember(t, r, kg.GlobalAccountIndex(), accountId.String()))
	})

	t.Run("debounce timeouts should not be added to function queue", func(t *testing.T) {
		r.FlushAll()

		debouncer, err := debounce.NewRedisDebouncerWithMigration(debounce.DebouncerOpts{
			PrimaryDebounceClient: redis_state.NewDebounceClient(rc, redis_state.QueueDefaultKey),
			PrimaryQueue:          q,
			PrimaryQueueShard:     shard,
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

		err = q.Enqueue(ctx, qi, now, queue.EnqueueOpts{})
		require.NoError(t, err)

		require.True(t, r.Exists(kg.PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, "")))
		require.True(t, r.Exists(kg.AccountPartitionIndex(accountId)))
		require.True(t, r.Exists(kg.GlobalAccountIndex()))
		require.True(t, hasMember(t, r, kg.GlobalAccountIndex(), accountId.String()))
	})

	t.Run("pause timeouts should belong to fn queue", func(t *testing.T) {
		r.FlushAll()

		err := q.Enqueue(ctx, queue.Item{
			WorkspaceID: wsID,
			Kind:        queue.KindPause,
			Identifier: state.Identifier{
				WorkflowID:  fnID,
				AccountID:   accountId,
				WorkspaceID: wsID,
			},
			Payload: nil,
		}, now, queue.EnqueueOpts{})
		require.NoError(t, err)

		require.True(t, r.Exists(kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), "")), r.Dump())
		require.True(t, r.Exists(kg.AccountPartitionIndex(accountId)))
		require.True(t, r.Exists(kg.GlobalAccountIndex()))
		require.True(t, hasMember(t, r, kg.GlobalAccountIndex(), accountId.String()))
	})

	t.Run("pause events should not belong to fn queue", func(t *testing.T) {
		r.FlushAll()

		queueName := fmt.Sprintf("pause:%s", wsID.String())
		err := q.Enqueue(ctx, queue.Item{
			QueueName: &queueName, WorkspaceID: wsID,
			Kind:    "pause-event",
			Payload: nil,
		}, now, queue.EnqueueOpts{})
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
