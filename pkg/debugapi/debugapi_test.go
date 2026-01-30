package debugapi

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/singleton"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (rueidis.Client, *miniredis.Miniredis) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	return rc, r
}

func setupBatchManager(t *testing.T, rc rueidis.Client) batch.BatchManager {
	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	shardedClient := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.NeverShardOnRun,
	})
	queueClient := unshardedClient.Queue()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindScheduleBatch: queue.KindScheduleBatch,
		}),
	}
	shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, queueClient, opts...)

	q, err := queue.New(
		context.Background(),
		"batch-test",
		shard,
		map[string]queue.QueueShard{
			consts.DefaultQueueShardName: shard,
		},
		func(ctx context.Context, accountId uuid.UUID, queueName *string) (queue.QueueShard, error) {
			return shard, nil
		},
		opts...,
	)
	require.NoError(t, err)

	return batch.NewRedisBatchManager(shardedClient.Batch(), q)
}

func setupDebouncer(t *testing.T, rc rueidis.Client) debounce.Debouncer {
	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	debounceClient := unshardedClient.Debounce()
	queueClient := unshardedClient.Queue()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	}
	shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, queueClient, opts...)

	q, err := queue.New(
		context.Background(),
		"debounce-test",
		shard,
		map[string]queue.QueueShard{
			consts.DefaultQueueShardName: shard,
		},
		func(ctx context.Context, accountId uuid.UUID, queueName *string) (queue.QueueShard, error) {
			return shard, nil
		},
		opts...,
	)
	require.NoError(t, err)

	return debounce.NewRedisDebouncer(debounceClient, shard, q)
}

func TestGetBatchInfo(t *testing.T) {
	rc, _ := setupTestRedis(t)
	ctx := context.Background()

	batchManager := setupBatchManager(t, rc)

	// Create debug API instance
	d := &debugAPI{
		batchManager: batchManager,
	}

	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	functionID := uuid.New()

	t.Run("no batch exists", func(t *testing.T) {
		resp, err := d.GetBatchInfo(ctx, &BatchInfoRequest{
			FunctionID: functionID.String(),
			BatchKey:   "test-key",
			AccountID:  accountID.String(),
		})
		require.NoError(t, err)
		require.Equal(t, "", resp.BatchID)
		require.Equal(t, int32(0), resp.ItemCount)
		require.Equal(t, "none", resp.Status)
	})

	t.Run("batch with items exists", func(t *testing.T) {
		// Create a batch using the batch manager
		fn := inngest.Function{
			ID: functionID,
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "60s",
			},
		}

		eventID := ulid.MustNew(ulid.Now(), rand.Reader)
		batchItem := batch.BatchItem{
			AccountID:       accountID,
			WorkspaceID:     workspaceID,
			AppID:           appID,
			FunctionID:      functionID,
			FunctionVersion: 1,
			EventID:         eventID,
			Event: event.Event{
				Name: "test/event",
				Data: map[string]any{"foo": "bar"},
			},
		}

		result, err := batchManager.Append(ctx, batchItem, fn)
		require.NoError(t, err)
		require.NotEmpty(t, result.BatchID)

		// Query the batch using the default key (no batch key expression)
		resp, err := d.GetBatchInfo(ctx, &BatchInfoRequest{
			FunctionID: functionID.String(),
			BatchKey:   "default",
			AccountID:  accountID.String(),
		})
		require.NoError(t, err)
		require.Equal(t, result.BatchID, resp.BatchID)
		require.Equal(t, int32(1), resp.ItemCount)
		require.Len(t, resp.Items, 1)
		require.Equal(t, eventID.String(), resp.Items[0].EventID)
		require.Equal(t, functionID.String(), resp.Items[0].FunctionID)
	})
}

func TestGetSingletonInfo(t *testing.T) {
	rc, _ := setupTestRedis(t)
	ctx := context.Background()

	// Create clients
	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	queueClient := unshardedClient.Queue()

	// Create singleton store
	shardSelector := func(ctx context.Context, accountId uuid.UUID, queueName *string) (queue.QueueShard, error) {
		return redis_state.NewQueueShard(consts.DefaultQueueShardName, queueClient), nil
	}
	singletonStore := singleton.New(ctx, map[string]*redis_state.QueueClient{
		consts.DefaultQueueShardName: queueClient,
	}, shardSelector)

	// Create debug API instance
	d := &debugAPI{
		singletonStore: singletonStore,
	}

	accountID := uuid.New()
	functionID := uuid.New()

	t.Run("no singleton lock exists", func(t *testing.T) {
		singletonKey := functionID.String()

		resp, err := d.GetSingletonInfo(ctx, &SingletonInfoRequest{
			SingletonKey: singletonKey,
			AccountID:    accountID.String(),
		})
		require.NoError(t, err)
		require.False(t, resp.HasLock)
		require.Equal(t, "", resp.CurrentRunID)
	})

	t.Run("singleton lock exists", func(t *testing.T) {
		singletonKey := functionID.String() + "-custom"
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		// Set the singleton lock manually
		redisKey := queueClient.KeyGenerator().SingletonKey(&queue.Singleton{Key: singletonKey})
		err := rc.Do(ctx, rc.B().Set().Key(redisKey).Value(runID.String()).Build()).Error()
		require.NoError(t, err)

		resp, err := d.GetSingletonInfo(ctx, &SingletonInfoRequest{
			SingletonKey: singletonKey,
			AccountID:    accountID.String(),
		})
		require.NoError(t, err)
		require.True(t, resp.HasLock)
		require.Equal(t, runID.String(), resp.CurrentRunID)
	})
}

func TestGetDebounceInfo(t *testing.T) {
	rc, _ := setupTestRedis(t)
	ctx := context.Background()

	redisDebouncer := setupDebouncer(t, rc)

	// Create debug API instance
	d := &debugAPI{
		debouncer: redisDebouncer,
	}

	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	functionID := uuid.New()

	t.Run("no debounce exists", func(t *testing.T) {
		resp, err := d.GetDebounceInfo(ctx, &DebounceInfoRequest{
			FunctionID:  functionID.String(),
			DebounceKey: functionID.String(),
			AccountID:   accountID.String(),
		})
		require.NoError(t, err)
		require.False(t, resp.HasDebounce)
	})

	t.Run("debounce exists", func(t *testing.T) {
		eventID := ulid.MustNew(ulid.Now(), rand.Reader)
		di := debounce.DebounceItem{
			AccountID:       accountID,
			WorkspaceID:     workspaceID,
			AppID:           appID,
			FunctionID:      functionID,
			FunctionVersion: 1,
			EventID:         eventID,
			Event: event.Event{
				Name:      "test/debounce-event",
				ID:        eventID.String(),
				Timestamp: time.Now().UnixMilli(),
				Data:      map[string]any{"key": "value"},
			},
		}

		fn := inngest.Function{
			ID: functionID,
			Debounce: &inngest.Debounce{
				Key:     nil, // Uses function ID as key
				Period:  "10s",
				Timeout: util.StrPtr("60s"),
			},
		}

		err := redisDebouncer.Debounce(ctx, di, fn)
		require.NoError(t, err)

		// Query the debounce info
		resp, err := d.GetDebounceInfo(ctx, &DebounceInfoRequest{
			FunctionID:  functionID.String(),
			DebounceKey: functionID.String(),
			AccountID:   accountID.String(),
		})
		require.NoError(t, err)
		require.True(t, resp.HasDebounce)
		require.NotEmpty(t, resp.DebounceID)
		require.Equal(t, eventID.String(), resp.EventID)
		require.Equal(t, accountID.String(), resp.AccountID)
		require.Equal(t, workspaceID.String(), resp.WorkspaceID)
		require.Equal(t, functionID.String(), resp.FunctionID)
	})
}

func TestGetBatchInfoNilManager(t *testing.T) {
	d := &debugAPI{
		batchManager: nil,
	}

	_, err := d.GetBatchInfo(context.Background(), &BatchInfoRequest{
		FunctionID: uuid.New().String(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "batch manager not configured")
}

func TestGetSingletonInfoNilStore(t *testing.T) {
	d := &debugAPI{
		singletonStore: nil,
	}

	_, err := d.GetSingletonInfo(context.Background(), &SingletonInfoRequest{
		SingletonKey: "test",
		AccountID:    uuid.New().String(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "singleton store not configured")
}

func TestGetDebounceInfoNilDebouncer(t *testing.T) {
	d := &debugAPI{
		debouncer: nil,
	}

	_, err := d.GetDebounceInfo(context.Background(), &DebounceInfoRequest{
		FunctionID: uuid.New().String(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "debouncer not configured")
}

func TestGetBatchInfoInvalidFunctionID(t *testing.T) {
	rc, _ := setupTestRedis(t)
	batchManager := setupBatchManager(t, rc)

	d := &debugAPI{
		batchManager: batchManager,
	}

	_, err := d.GetBatchInfo(context.Background(), &BatchInfoRequest{
		FunctionID: "invalid-uuid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid function_id")
}

func TestGetSingletonInfoInvalidAccountID(t *testing.T) {
	rc, _ := setupTestRedis(t)

	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	queueClient := unshardedClient.Queue()

	shardSelector := func(ctx context.Context, accountId uuid.UUID, queueName *string) (queue.QueueShard, error) {
		return redis_state.NewQueueShard(consts.DefaultQueueShardName, queueClient), nil
	}
	singletonStore := singleton.New(context.Background(), map[string]*redis_state.QueueClient{
		consts.DefaultQueueShardName: queueClient,
	}, shardSelector)

	d := &debugAPI{
		singletonStore: singletonStore,
	}

	_, err := d.GetSingletonInfo(context.Background(), &SingletonInfoRequest{
		SingletonKey: "test",
		AccountID:    "invalid-uuid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid account_id")
}

func TestGetDebounceInfoInvalidFunctionID(t *testing.T) {
	rc, _ := setupTestRedis(t)
	redisDebouncer := setupDebouncer(t, rc)

	d := &debugAPI{
		debouncer: redisDebouncer,
	}

	_, err := d.GetDebounceInfo(context.Background(), &DebounceInfoRequest{
		FunctionID: "invalid-uuid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid function_id")
}
