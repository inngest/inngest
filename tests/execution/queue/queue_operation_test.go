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

	options := append([]queue.QueueOpt{
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
	})

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
	})

	t.Run("PartitionPeek", func(t *testing.T) {
		parts, err := shard.PartitionPeek(ctx, true, clock.Now().Add(time.Minute), 10)
		require.NoError(t, err)

		require.Len(t, parts, 1)

		require.Equal(t, fnID, parts[0].FunctionID)
		require.Equal(t, accountID, parts[0].AccountID)

		require.Nil(t, parts[0].LeaseID)
	})

	t.Run("PartitionLease", func(t *testing.T) {
	})

	t.Run("Peek", func(t *testing.T) {
	})

	t.Run("Lease", func(t *testing.T) {
	})

	t.Run("ExtendLease", func(t *testing.T) {
	})

	t.Run("Requeue", func(t *testing.T) {
	})

	t.Run("Dequeue", func(t *testing.T) {
	})
}
