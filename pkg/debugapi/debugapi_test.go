package debugapi

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/singleton"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// mockCQRSManager is a minimal mock for cqrs.Manager used in tests.
// It embeds a nil cqrs.Manager to satisfy the interface but only implements
// the methods actually used by the code under test.
type mockCQRSManager struct {
	cqrs.Manager
	fn *cqrs.Function
}

func (m *mockCQRSManager) GetFunctionByInternalUUID(ctx context.Context, fnID uuid.UUID) (*cqrs.Function, error) {
	return m.fn, nil
}

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

// TestGetBatchInfoHandler tests the debug API handler for batch info.
// Edge cases for BatchManager.GetBatchInfo are tested in pkg/execution/batch/batch_test.go.
func TestGetBatchInfoHandler(t *testing.T) {
	rc, _ := setupTestRedis(t)
	ctx := context.Background()

	batchManager := setupBatchManager(t, rc)
	d := &debugAPI{batchManager: batchManager}

	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	functionID := uuid.New()

	// Create a batch
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

	// Test handler correctly converts manager response to protobuf
	resp, err := d.GetBatchInfo(ctx, &pb.BatchInfoRequest{
		FunctionId: functionID.String(),
		BatchKey:   "default",
	})
	require.NoError(t, err)
	require.Equal(t, result.BatchID, resp.BatchId)
	require.Equal(t, int32(1), resp.ItemCount)
	require.Len(t, resp.Items, 1)
	require.Equal(t, eventID.String(), resp.Items[0].EventId)
	require.Equal(t, functionID.String(), resp.Items[0].FunctionId)
}

// TestGetSingletonInfoHandler tests the debug API handler for singleton info.
func TestGetSingletonInfoHandler(t *testing.T) {
	rc, _ := setupTestRedis(t)
	ctx := context.Background()

	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	queueClient := unshardedClient.Queue()

	shardSelector := func(ctx context.Context, accountId uuid.UUID, queueName *string) (queue.QueueShard, error) {
		return redis_state.NewQueueShard(consts.DefaultQueueShardName, queueClient), nil
	}
	singletonStore := singleton.New(ctx, map[string]*redis_state.QueueClient{
		consts.DefaultQueueShardName: queueClient,
	}, shardSelector)

	d := &debugAPI{singletonStore: singletonStore}

	functionID := uuid.New()
	singletonKey := functionID.String()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)

	// Set the singleton lock
	redisKey := queueClient.KeyGenerator().SingletonKey(&queue.Singleton{Key: singletonKey})
	err := rc.Do(ctx, rc.B().Set().Key(redisKey).Value(runID.String()).Build()).Error()
	require.NoError(t, err)

	// Test handler correctly converts store response to protobuf
	resp, err := d.GetSingletonInfo(ctx, &pb.SingletonInfoRequest{
		FunctionId: functionID.String(),
	})
	require.NoError(t, err)
	require.True(t, resp.HasLock)
	require.Equal(t, runID.String(), resp.CurrentRunId)
}

// TestGetDebounceInfoHandler tests the debug API handler for debounce info.
// Edge cases for Debouncer.GetDebounceInfo are tested in pkg/execution/debounce/debounce_test.go.
func TestGetDebounceInfoHandler(t *testing.T) {
	rc, _ := setupTestRedis(t)
	ctx := context.Background()

	redisDebouncer := setupDebouncer(t, rc)
	d := &debugAPI{debouncer: redisDebouncer}

	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	functionID := uuid.New()

	// Create a debounce
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
			Key:     nil,
			Period:  "10s",
			Timeout: util.StrPtr("60s"),
		},
	}

	err := redisDebouncer.Debounce(ctx, di, fn)
	require.NoError(t, err)

	// Test handler correctly converts debouncer response to protobuf
	resp, err := d.GetDebounceInfo(ctx, &pb.DebounceInfoRequest{
		FunctionId:  functionID.String(),
		DebounceKey: functionID.String(),
	})
	require.NoError(t, err)
	require.True(t, resp.HasDebounce)
	require.NotEmpty(t, resp.DebounceId)
	require.Equal(t, eventID.String(), resp.EventId)
	require.Equal(t, accountID.String(), resp.AccountId)
	require.Equal(t, workspaceID.String(), resp.WorkspaceId)
	require.Equal(t, functionID.String(), resp.FunctionId)
}

func TestGetBatchInfoNilManager(t *testing.T) {
	d := &debugAPI{
		batchManager: nil,
	}

	_, err := d.GetBatchInfo(context.Background(), &pb.BatchInfoRequest{
		FunctionId: uuid.New().String(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "batch manager not configured")
}

func TestGetSingletonInfoNilStore(t *testing.T) {
	d := &debugAPI{
		singletonStore: nil,
	}

	_, err := d.GetSingletonInfo(context.Background(), &pb.SingletonInfoRequest{
		FunctionId: uuid.New().String(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "singleton store not configured")
}

func TestGetDebounceInfoNilDebouncer(t *testing.T) {
	d := &debugAPI{
		debouncer: nil,
	}

	_, err := d.GetDebounceInfo(context.Background(), &pb.DebounceInfoRequest{
		FunctionId: uuid.New().String(),
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

	_, err := d.GetBatchInfo(context.Background(), &pb.BatchInfoRequest{
		FunctionId: "invalid-uuid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid function_id")
}

func TestGetSingletonInfoInvalidFunctionID(t *testing.T) {
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

	_, err := d.GetSingletonInfo(context.Background(), &pb.SingletonInfoRequest{
		FunctionId: "invalid-uuid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid function_id")
}

func TestGetDebounceInfoInvalidFunctionID(t *testing.T) {
	rc, _ := setupTestRedis(t)
	redisDebouncer := setupDebouncer(t, rc)

	d := &debugAPI{
		debouncer: redisDebouncer,
	}

	_, err := d.GetDebounceInfo(context.Background(), &pb.DebounceInfoRequest{
		FunctionId: "invalid-uuid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid function_id")
}

// TestDeleteBatchHandler tests the debug API handler for deleting batches.
func TestDeleteBatchHandler(t *testing.T) {
	rc, _ := setupTestRedis(t)
	ctx := context.Background()

	batchManager := setupBatchManager(t, rc)
	d := &debugAPI{batchManager: batchManager}

	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	functionID := uuid.New()

	// Create a batch
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

	// Test handler correctly deletes the batch
	resp, err := d.DeleteBatch(ctx, &pb.DeleteBatchRequest{
		FunctionId: functionID.String(),
		BatchKey:   "default",
	})
	require.NoError(t, err)
	require.True(t, resp.Deleted)
	require.Equal(t, result.BatchID, resp.BatchId)
	require.Equal(t, int32(1), resp.ItemCount)

	// Verify batch no longer exists
	infoResp, err := d.GetBatchInfo(ctx, &pb.BatchInfoRequest{
		FunctionId: functionID.String(),
		BatchKey:   "default",
	})
	require.NoError(t, err)
	require.Equal(t, "", infoResp.BatchId)
}

// TestRunBatchHandler tests the debug API handler for running batches.
func TestRunBatchHandler(t *testing.T) {
	rc, _ := setupTestRedis(t)
	ctx := context.Background()

	batchManager := setupBatchManager(t, rc)

	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	functionID := uuid.New()

	// Create mock db that returns the function with required IDs
	mockDB := &mockCQRSManager{
		fn: &cqrs.Function{
			ID:    functionID,
			EnvID: workspaceID,
			AppID: appID,
		},
	}

	d := &debugAPI{batchManager: batchManager, db: mockDB}

	// Create a batch
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

	// Test handler correctly schedules the batch
	resp, err := d.RunBatch(ctx, &pb.RunBatchRequest{
		FunctionId: functionID.String(),
		BatchKey:   "default",
	})
	require.NoError(t, err)
	require.True(t, resp.Scheduled)
	require.Equal(t, result.BatchID, resp.BatchId)
	require.Equal(t, int32(1), resp.ItemCount)
}

// TestDeleteSingletonLockHandler tests the debug API handler for deleting singleton locks.
func TestDeleteSingletonLockHandler(t *testing.T) {
	rc, _ := setupTestRedis(t)
	ctx := context.Background()

	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	queueClient := unshardedClient.Queue()

	shardSelector := func(ctx context.Context, accountId uuid.UUID, queueName *string) (queue.QueueShard, error) {
		return redis_state.NewQueueShard(consts.DefaultQueueShardName, queueClient), nil
	}
	singletonStore := singleton.New(ctx, map[string]*redis_state.QueueClient{
		consts.DefaultQueueShardName: queueClient,
	}, shardSelector)

	d := &debugAPI{singletonStore: singletonStore}

	functionID := uuid.New()
	singletonKey := functionID.String()
	runID := ulid.MustNew(ulid.Now(), rand.Reader)

	// Set the singleton lock
	redisKey := queueClient.KeyGenerator().SingletonKey(&queue.Singleton{Key: singletonKey})
	err := rc.Do(ctx, rc.B().Set().Key(redisKey).Value(runID.String()).Build()).Error()
	require.NoError(t, err)

	// Test handler correctly deletes the lock
	resp, err := d.DeleteSingletonLock(ctx, &pb.DeleteSingletonLockRequest{
		FunctionId: functionID.String(),
	})
	require.NoError(t, err)
	require.True(t, resp.Deleted)
	require.Equal(t, runID.String(), resp.RunId)

	// Verify lock no longer exists
	infoResp, err := d.GetSingletonInfo(ctx, &pb.SingletonInfoRequest{
		FunctionId: functionID.String(),
	})
	require.NoError(t, err)
	require.False(t, infoResp.HasLock)
	require.Empty(t, infoResp.CurrentRunId)
}

func TestDeleteBatchNilManager(t *testing.T) {
	d := &debugAPI{
		batchManager: nil,
	}

	_, err := d.DeleteBatch(context.Background(), &pb.DeleteBatchRequest{
		FunctionId: uuid.New().String(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "batch manager not configured")
}

func TestRunBatchNilManager(t *testing.T) {
	d := &debugAPI{
		batchManager: nil,
	}

	_, err := d.RunBatch(context.Background(), &pb.RunBatchRequest{
		FunctionId: uuid.New().String(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "batch manager not configured")
}

func TestDeleteSingletonLockNilStore(t *testing.T) {
	d := &debugAPI{
		singletonStore: nil,
	}

	_, err := d.DeleteSingletonLock(context.Background(), &pb.DeleteSingletonLockRequest{
		FunctionId: uuid.New().String(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "singleton store not configured")
}

func TestDeleteBatchInvalidFunctionID(t *testing.T) {
	rc, _ := setupTestRedis(t)
	batchManager := setupBatchManager(t, rc)

	d := &debugAPI{
		batchManager: batchManager,
	}

	_, err := d.DeleteBatch(context.Background(), &pb.DeleteBatchRequest{
		FunctionId: "invalid-uuid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid function_id")
}

func TestRunBatchInvalidFunctionID(t *testing.T) {
	rc, _ := setupTestRedis(t)
	batchManager := setupBatchManager(t, rc)

	d := &debugAPI{
		batchManager: batchManager,
	}

	_, err := d.RunBatch(context.Background(), &pb.RunBatchRequest{
		FunctionId: "invalid-uuid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid function_id")
}

func TestDeleteSingletonLockInvalidFunctionID(t *testing.T) {
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

	_, err := d.DeleteSingletonLock(context.Background(), &pb.DeleteSingletonLockRequest{
		FunctionId: "invalid-uuid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid function_id")
}

// TestDeleteDebounceHandler tests the debug API handler for deleting debounces.
func TestDeleteDebounceHandler(t *testing.T) {
	rc, _ := setupTestRedis(t)
	ctx := context.Background()

	redisDebouncer := setupDebouncer(t, rc)
	d := &debugAPI{debouncer: redisDebouncer}

	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	functionID := uuid.New()

	// Create a debounce
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
			Key:     nil,
			Period:  "10s",
			Timeout: util.StrPtr("60s"),
		},
	}

	err := redisDebouncer.Debounce(ctx, di, fn)
	require.NoError(t, err)

	// Test handler correctly deletes the debounce
	resp, err := d.DeleteDebounce(ctx, &pb.DeleteDebounceRequest{
		FunctionId:  functionID.String(),
		DebounceKey: functionID.String(),
	})
	require.NoError(t, err)
	require.True(t, resp.Deleted)
	require.NotEmpty(t, resp.DebounceId)
	require.Equal(t, eventID.String(), resp.EventId)

	// Verify debounce no longer exists
	infoResp, err := d.GetDebounceInfo(ctx, &pb.DebounceInfoRequest{
		FunctionId:  functionID.String(),
		DebounceKey: functionID.String(),
	})
	require.NoError(t, err)
	require.False(t, infoResp.HasDebounce)
}

// TestRunDebounceHandler tests the debug API handler for running debounces.
func TestRunDebounceHandler(t *testing.T) {
	rc, _ := setupTestRedis(t)
	ctx := context.Background()

	redisDebouncer := setupDebouncer(t, rc)
	d := &debugAPI{debouncer: redisDebouncer}

	accountID := uuid.New()
	workspaceID := uuid.New()
	appID := uuid.New()
	functionID := uuid.New()

	// Create a debounce
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
			Key:     nil,
			Period:  "10s",
			Timeout: util.StrPtr("60s"),
		},
	}

	err := redisDebouncer.Debounce(ctx, di, fn)
	require.NoError(t, err)

	// Test handler correctly schedules the debounce
	resp, err := d.RunDebounce(ctx, &pb.RunDebounceRequest{
		FunctionId:  functionID.String(),
		DebounceKey: functionID.String(),
	})
	require.NoError(t, err)
	require.True(t, resp.Scheduled)
	require.NotEmpty(t, resp.DebounceId)
	require.Equal(t, eventID.String(), resp.EventId)
}

func TestDeleteDebounceNilDebouncer(t *testing.T) {
	d := &debugAPI{
		debouncer: nil,
	}

	_, err := d.DeleteDebounce(context.Background(), &pb.DeleteDebounceRequest{
		FunctionId: uuid.New().String(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "debouncer not configured")
}

func TestRunDebounceNilDebouncer(t *testing.T) {
	d := &debugAPI{
		debouncer: nil,
	}

	_, err := d.RunDebounce(context.Background(), &pb.RunDebounceRequest{
		FunctionId: uuid.New().String(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "debouncer not configured")
}

func TestDeleteDebounceInvalidFunctionID(t *testing.T) {
	rc, _ := setupTestRedis(t)
	redisDebouncer := setupDebouncer(t, rc)

	d := &debugAPI{
		debouncer: redisDebouncer,
	}

	_, err := d.DeleteDebounce(context.Background(), &pb.DeleteDebounceRequest{
		FunctionId: "invalid-uuid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid function_id")
}

func TestRunDebounceInvalidFunctionID(t *testing.T) {
	rc, _ := setupTestRedis(t)
	redisDebouncer := setupDebouncer(t, rc)

	d := &debugAPI{
		debouncer: redisDebouncer,
	}

	_, err := d.RunDebounce(context.Background(), &pb.RunDebounceRequest{
		FunctionId: "invalid-uuid",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid function_id")
}
