package queue

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestQueueOperations(t *testing.T) {
	ctx := context.Background()

	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelDebug))
	ctx = logger.WithStdlib(ctx, l)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader)

	options := []queue.QueueOpt{
		queue.WithClock(clock),
		queue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p queue.PartitionIdentifier) queue.PartitionConstraintConfig {
			return queue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: queue.PartitionConcurrency{
					SystemConcurrency:   consts.DefaultConcurrencyLimit,
					AccountConcurrency:  consts.DefaultConcurrencyLimit,
					FunctionConcurrency: consts.DefaultConcurrencyLimit,
				},
			}
		}),
	}

	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName("test"),
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
	)
	require.NoError(t, err)

	options = append(options, queue.WithCapacityManager(cm))
	options = append(options,
		queue.WithAcquireCapacityLeaseOnBacklogRefill(true),
	)

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard("test", queueClient, options...)

	var item *queue.QueueItem
	t.Run("EnqueueItem", func(t *testing.T) {
		qi, err := shard.EnqueueItem(ctx, queue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: queue.Item{
				WorkspaceID: envID,
				Kind:        queue.KindStart,
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: envID,
					WorkflowID:  fnID,
					RunID:       runID,
				},
			},
		}, clock.Now(), queue.EnqueueOpts{})
		require.NoError(t, err)

		loaded, err := shard.LoadQueueItem(ctx, qi.ID)
		require.NoError(t, err)

		require.Equal(t, qi, *loaded)

		item = loaded
	})

	var partition *queue.QueuePartition
	t.Run("PartitionPeek", func(t *testing.T) {
		parts, err := shard.PartitionPeek(ctx, true, clock.Now().Add(time.Minute), 10)
		require.NoError(t, err)

		require.Len(t, parts, 1)

		require.Equal(t, fnID, *parts[0].FunctionID)
		require.Equal(t, accountID, parts[0].AccountID)

		require.Nil(t, parts[0].LeaseID)

		partition = parts[0]
	})

	t.Run("PartitionLease", func(t *testing.T) {
		leaseID, err := shard.PartitionLease(ctx, partition, 5*time.Second)
		require.NoError(t, err)

		require.NotNil(t, leaseID)

		partition.LeaseID = leaseID
		partition.Last = clock.Now().UnixMilli()

		res, err := shard.PartitionByID(ctx, partition.ID)
		require.NoError(t, err)
		require.Equal(t, partition, res.QueuePartition)
	})

	t.Run("PartitionRequeue", func(t *testing.T) {
		err := shard.PartitionRequeue(ctx, partition, clock.Now().Add(10*time.Second), false)
		require.NoError(t, err)
	})

	t.Run("Peek", func(t *testing.T) {
		peeked, err := shard.Peek(ctx, partition, clock.Now().Add(10*time.Second), 10)
		require.NoError(t, err)

		require.Len(t, peeked, 1)
		require.Equal(t, item, peeked[0], "items must match", item.Data.JobID, peeked[0].Data.JobID)
	})

	var leaseID *ulid.ULID
	t.Run("Lease", func(t *testing.T) {
		lID, err := shard.Lease(ctx, *item, 10*time.Second, clock.Now())
		require.NoError(t, err)

		require.NotNil(t, lID)
		require.Equal(t, clock.Now().Add(10*time.Second).Truncate(time.Millisecond), lID.Timestamp())

		t.Run("should not be possible to lease again", func(t *testing.T) {
			_, err := shard.Lease(ctx, *item, 10*time.Second, clock.Now())
			require.Error(t, err)
			require.ErrorIs(t, err, queue.ErrQueueItemAlreadyLeased)
		})

		t.Run("should not find anything in queue when peeking", func(t *testing.T) {
			peeked, err := shard.Peek(ctx, partition, clock.Now().Add(10*time.Second), 10)
			require.NoError(t, err)
			require.Len(t, peeked, 0)
		})

		leaseID = lID
	})

	t.Run("ExtendLease", func(t *testing.T) {
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		lID, err := shard.ExtendLease(ctx, *item, *leaseID, 10*time.Second)
		require.NoError(t, err)

		require.NotNil(t, lID)
		require.NotEqual(t, *leaseID, *lID)

		leaseID = lID
	})

	t.Run("Requeue", func(t *testing.T) {
		requeueAt := clock.Now().Add(20 * time.Second)
		err := shard.Requeue(ctx, *item, requeueAt)
		require.NoError(t, err)

		item.WallTimeMS = requeueAt.UnixMilli()
		item.AtMS = requeueAt.UnixMilli()
		item.EnqueuedAt = clock.Now().UnixMilli()

		loaded, err := shard.LoadQueueItem(ctx, item.ID)
		require.NoError(t, err)

		require.Nil(t, loaded.LeaseID)
		require.Equal(t, *item, *loaded)

		t.Run("should find item in queue when peeking", func(t *testing.T) {
			peeked, err := shard.Peek(ctx, partition, clock.Now().Add(30*time.Second), 10)
			require.NoError(t, err)
			require.Len(t, peeked, 1)

			require.Equal(t, item, peeked[0], "items must match")
		})
	})

	t.Run("Dequeue", func(t *testing.T) {
		lID, err := shard.Lease(ctx, *item, 10*time.Second, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, lID)

		err = shard.Dequeue(ctx, *item)
		require.NoError(t, err)

		_, err = shard.LoadQueueItem(ctx, item.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound)

		t.Run("should not find anything in queue when peeking", func(t *testing.T) {
			peeked, err := shard.Peek(ctx, partition, clock.Now().Add(10*time.Second), 10)
			require.NoError(t, err)
			require.Len(t, peeked, 0)
		})
	})
}
