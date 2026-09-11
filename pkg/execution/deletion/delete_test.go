package deletion

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestDeleteManager(t *testing.T) {
	// Set up in-memory Redis instance
	redisCluster := miniredis.RunT(t)

	redisClient, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{redisCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	// Create unsharded client and managers
	unshardedClient := redis_state.NewUnshardedClient(redisClient, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindEdge:          queue.KindEdge,
			queue.KindPause:         queue.KindPause,
			queue.KindScheduleBatch: queue.KindScheduleBatch,
			queue.KindDebounce:      queue.KindDebounce,
		}),
	}

	// Set up queue shard
	shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)
	shardRegistry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	// Create queue manager
	queueManager, err := queue.New(
		context.Background(),
		"delete-test",
		shardRegistry,
		opts...,
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Create sharded client for batch access and pause store
	shardedClient := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: redisClient,
		BatchClient:            redisClient,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.NeverShardOnRun,
	})

	// Create pause manager
	pauseMgr := pauses.NewPauseStoreManager(unshardedClient)
	batchClient := shardedClient.Batch()
	batchManager := batch.NewRedisBatchManager(batchClient, queueManager)

	// Create debounce manager
	debouncer, err := debounce.NewDebouncer(shardRegistry, shard.Name(), queueManager)
	require.NoError(t, err)

	// Create DeleteManager with all dependencies
	deleteManager, err := NewDeleteManager(
		WithPauseManager(pauseMgr),
		WithBatchManager(batchManager),
		WithDebouncer(debouncer),
	)
	require.NoError(t, err)

	accountID, workspaceID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	t.Run("KindEdge", func(t *testing.T) {
		// Test deletion of KindEdge items (no additional cleanup required)
		// Create a KindEdge queue item
		queueItem := &queue.QueueItem{
			ID:          ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String(),
			AtMS:        time.Now().UnixMilli(),
			WallTimeMS:  time.Now().UnixMilli(),
			FunctionID:  functionID,
			WorkspaceID: workspaceID,
			QueueName:   nil,
			Data: queue.Item{
				WorkspaceID: workspaceID,
				Kind:        queue.KindEdge,
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: workspaceID,
					AppID:       appID,
					WorkflowID:  functionID,
					Key:         "test-key",
				},
			},
		}

		// Enqueue the item
		err := queueManager.Enqueue(ctx, queueItem.Data, time.Now(), queue.EnqueueOpts{})
		require.NoError(t, err)

		// Delete the queue item (this is what we're actually testing)
		err = deleteManager.DeleteQueueItem(ctx, shard, queueItem)
		require.NoError(t, err, "DeleteQueueItem should succeed for KindEdge")
	})

	t.Run("KindPause", func(t *testing.T) {
		// Test deletion of KindPause items (should delete associated pause)
		// Create a pause first
		pauseID := uuid.New()
		runID := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)
		pauseIndex := pauses.Index{
			WorkspaceID: workspaceID,
			EventName:   "test.event",
		}

		pause := &state.Pause{
			ID:          pauseID,
			WorkspaceID: workspaceID,
			Identifier: state.PauseIdentifier{
				RunID:      runID,
				FunctionID: functionID,
				AccountID:  accountID,
			},
			Event:   &pauseIndex.EventName,
			Expires: state.Time(time.Now().Add(10 * time.Hour)),
		}

		// Write the pause to the pause manager
		_, err := pauseMgr.Write(ctx, pauseIndex, pause)
		require.NoError(t, err)

		require.True(t, redisCluster.Exists(unshardedClient.Pauses().KeyGenerator().Pause(ctx, pauseID)), redisCluster.Dump())

		// Verify pause was created
		retrievedPause, err := pauseMgr.PauseByID(ctx, pauseIndex, pauseID)
		require.NoError(t, err, redisCluster.Dump())
		require.NotNil(t, retrievedPause)
		require.Equal(t, pauseID, retrievedPause.ID)

		// Create a KindPause queue item with PayloadPauseTimeout
		queueItem := &queue.QueueItem{
			ID:          ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String(),
			AtMS:        time.Now().UnixMilli(),
			WallTimeMS:  time.Now().UnixMilli(),
			FunctionID:  functionID,
			WorkspaceID: workspaceID,
			QueueName:   nil,
			Data: queue.Item{
				WorkspaceID: workspaceID,
				Kind:        queue.KindPause,
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: workspaceID,
					AppID:       appID,
					WorkflowID:  functionID,
					Key:         "test-pause",
				},
				Payload: queue.PayloadPauseTimeout{
					PauseID: pauseID,
					Pause:   *pause,
				},
			},
		}

		// Enqueue the item
		err = queueManager.Enqueue(ctx, queueItem.Data, time.Now(), queue.EnqueueOpts{})
		require.NoError(t, err)

		// Delete the queue item (should also delete the pause)
		err = deleteManager.DeleteQueueItem(ctx, shard, queueItem)
		require.NoError(t, err, "DeleteQueueItem should succeed for KindPause")

		// Verify pause was also deleted
		_, err = pauseMgr.PauseByID(ctx, pauseIndex, pauseID)
		require.Error(t, err)
		require.ErrorIs(t, err, state.ErrPauseNotFound)
	})

	t.Run("KindScheduleBatch", func(t *testing.T) {
		// Test deletion of KindScheduleBatch items (should delete associated batch)
		// Create a batch item first using AppendAndScheduleBatch
		eventID := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)

		batchItem := batch.BatchItem{
			AccountID:       accountID,
			WorkspaceID:     workspaceID,
			AppID:           appID,
			FunctionID:      functionID,
			FunctionVersion: 1,
			EventID:         eventID,
			Event: event.Event{
				ID:        eventID.String(),
				Name:      "test.batch.event",
				Data:      map[string]interface{}{"key": "value"},
				Timestamp: time.Now().UnixMilli(),
			},
			Version: 1,
		}

		fn := inngest.Function{
			ID:   functionID,
			Name: "test-batch-function",
			Triggers: []inngest.Trigger{
				{
					EventTrigger: &inngest.EventTrigger{
						Event: "test.batch.event",
					},
				},
			},
			EventBatch: &inngest.EventBatchConfig{
				MaxSize: 10,
				Timeout: "30s",
			},
		}

		// Create the batch using Append (simpler method)
		result, err := batchManager.Append(ctx, batchItem, fn)
		require.NoError(t, err)
		require.NotNil(t, result)

		batchID := ulid.MustParse(result.BatchID)

		// Verify batch was created by checking if items can be retrieved
		items, err := batchManager.RetrieveItems(ctx, functionID, batchID)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, eventID, items[0].EventID)

		// Create a KindScheduleBatch queue item with ScheduleBatchPayload
		queueItem := &queue.QueueItem{
			ID:          ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String(),
			AtMS:        time.Now().UnixMilli(),
			WallTimeMS:  time.Now().UnixMilli(),
			FunctionID:  functionID,
			WorkspaceID: workspaceID,
			QueueName:   nil,
			Data: queue.Item{
				WorkspaceID: workspaceID,
				Kind:        queue.KindScheduleBatch,
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: workspaceID,
					AppID:       appID,
					WorkflowID:  functionID,
					Key:         "test-batch",
				},
				Payload: batch.ScheduleBatchPayload{
					BatchID:         batchID,
					BatchPointer:    batchID.String(),
					AccountID:       accountID,
					WorkspaceID:     workspaceID,
					AppID:           appID,
					FunctionID:      functionID,
					FunctionVersion: 1,
				},
			},
		}

		// Enqueue the item
		err = queueManager.Enqueue(ctx, queueItem.Data, time.Now(), queue.EnqueueOpts{})
		require.NoError(t, err)

		// Delete the queue item (should also delete the batch)
		err = deleteManager.DeleteQueueItem(ctx, shard, queueItem)
		require.NoError(t, err, "DeleteQueueItem should succeed for KindScheduleBatch")

		// Verify batch was also deleted by trying to retrieve items
		items, err = batchManager.RetrieveItems(ctx, functionID, batchID)
		require.NoError(t, err)
		require.Len(t, items, 0, "Batch should be deleted")
	})

	t.Run("KindDebounce", func(t *testing.T) {
		// Test deletion of KindDebounce items (should delete associated debounce)
		// Create a debounce item first using Debounce()
		eventID := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)

		debounceItem := debounce.DebounceItem{
			AccountID:   accountID,
			WorkspaceID: workspaceID,
			AppID:       appID,
			FunctionID:  functionID,
			EventID:     eventID,
			Event: event.Event{
				ID:        eventID.String(),
				Name:      "test.debounce.event",
				Data:      map[string]interface{}{"key": "value"},
				Timestamp: time.Now().UnixMilli(),
			},
		}

		fn := inngest.Function{
			ID:   functionID,
			Name: "test-debounce-function",
			Triggers: []inngest.Trigger{
				{
					EventTrigger: &inngest.EventTrigger{
						Event: "test.debounce.event",
					},
				},
			},
			Debounce: &inngest.Debounce{
				Key:    nil,
				Period: "10s",
			},
		}

		// Create the debounce using Debounce()
		debounceID, err := debouncer.Debounce(ctx, debounceItem, fn)
		require.NoError(t, err)
		require.NotNil(t, debounceID)

		// Create a KindDebounce queue item with DebouncePayload
		queueItem := &queue.QueueItem{
			ID:          ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String(),
			AtMS:        time.Now().UnixMilli(),
			WallTimeMS:  time.Now().UnixMilli(),
			FunctionID:  functionID,
			WorkspaceID: workspaceID,
			QueueName:   nil,
			Data: queue.Item{
				WorkspaceID: workspaceID,
				Kind:        queue.KindDebounce,
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: workspaceID,
					AppID:       appID,
					WorkflowID:  functionID,
					Key:         "test-debounce",
				},
				Payload: debounce.DebouncePayload{
					DebounceID:      *debounceID,
					AccountID:       accountID,
					WorkspaceID:     workspaceID,
					AppID:           appID,
					FunctionID:      functionID,
					FunctionVersion: 1,
				},
			},
		}

		// Enqueue the item
		err = queueManager.Enqueue(ctx, queueItem.Data, time.Now(), queue.EnqueueOpts{})
		require.NoError(t, err)

		// Delete the queue item (should also delete the debounce)
		err = deleteManager.DeleteQueueItem(ctx, shard, queueItem)
		require.NoError(t, err)

		_, err = debouncer.GetDebounceItem(ctx, queue.Scope{
			AccountID:  accountID,
			EnvID:      workspaceID,
			FunctionID: functionID,
		}, *debounceID)
		require.ErrorIs(t, err, debounce.ErrDebounceNotFound)
	})

	t.Run("UnknownKind", func(t *testing.T) {
		// Test deletion of unknown queue item kind
		// This should test the handler mechanism for unknown kinds
		var handlerCallCount int

		// Create a DeleteManager with a custom handler for unknown kinds
		deleteManagerWithHandler, err := NewDeleteManager(
			WithPauseManager(pauseMgr),
			WithBatchManager(batchManager),
			WithDebouncer(debouncer),
			WithUnknownHandler(func(ctx context.Context, shard queue.QueueShard, item *queue.QueueItem) error {
				handlerCallCount++
				return nil
			}),
		)
		require.NoError(t, err)

		// Create a queue item with unknown kind "unknownItem"
		queueItem := &queue.QueueItem{
			ID:          ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String(),
			AtMS:        time.Now().UnixMilli(),
			WallTimeMS:  time.Now().UnixMilli(),
			FunctionID:  functionID,
			WorkspaceID: workspaceID,
			QueueName:   nil,
			Data: queue.Item{
				WorkspaceID: workspaceID,
				Kind:        "unknownItem",
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: workspaceID,
					AppID:       appID,
					WorkflowID:  functionID,
					Key:         "test-unknown",
				},
			},
		}

		// Enqueue the item
		err = queueManager.Enqueue(ctx, queueItem.Data, time.Now(), queue.EnqueueOpts{})
		require.NoError(t, err)

		// Delete the queue item (should call our custom handler)
		err = deleteManagerWithHandler.DeleteQueueItem(ctx, shard, queueItem)
		require.NoError(t, err, "DeleteQueueItem should succeed for unknown kind")

		// Validate that the handler was called at least once
		require.GreaterOrEqual(t, handlerCallCount, 1, "Handler should be called at least once for unknown kind")
	})

	t.Run("KindPause Edge Cases", func(t *testing.T) {
		t.Run("NilPauseManager", func(t *testing.T) {
			// DeleteManager without pause manager should skip pause deletion
			deleteManagerNoPause, err := NewDeleteManager(
				WithBatchManager(batchManager),
				WithDebouncer(debouncer),
			)
			require.NoError(t, err)

			queueItem := &queue.QueueItem{
				ID:          ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String(),
				AtMS:        time.Now().UnixMilli(),
				WallTimeMS:  time.Now().UnixMilli(),
				FunctionID:  functionID,
				WorkspaceID: workspaceID,
				QueueName:   nil,
				Data: queue.Item{
					WorkspaceID: workspaceID,
					Kind:        queue.KindPause,
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: workspaceID,
						AppID:       appID,
						WorkflowID:  functionID,
						Key:         "test-pause-nil-manager",
					},
					Payload: queue.PayloadPauseTimeout{
						PauseID: uuid.New(),
						Pause:   state.Pause{},
					},
				},
			}

			err = queueManager.Enqueue(ctx, queueItem.Data, time.Now(), queue.EnqueueOpts{})
			require.NoError(t, err)

			err = deleteManagerNoPause.DeleteQueueItem(ctx, shard, queueItem)
			require.NoError(t, err, "Should succeed even with nil pause manager")
		})

		t.Run("InvalidPayloadType", func(t *testing.T) {
			// KindPause with wrong payload type should skip pause deletion
			queueItem := &queue.QueueItem{
				ID:          ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String(),
				AtMS:        time.Now().UnixMilli(),
				WallTimeMS:  time.Now().UnixMilli(),
				FunctionID:  functionID,
				WorkspaceID: workspaceID,
				QueueName:   nil,
				Data: queue.Item{
					WorkspaceID: workspaceID,
					Kind:        queue.KindPause,
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: workspaceID,
						AppID:       appID,
						WorkflowID:  functionID,
						Key:         "test-pause-invalid-payload",
					},
					Payload: "invalid-payload-type",
				},
			}

			err := queueManager.Enqueue(ctx, queueItem.Data, time.Now(), queue.EnqueueOpts{})
			require.NoError(t, err)

			err = deleteManager.DeleteQueueItem(ctx, shard, queueItem)
			require.NoError(t, err, "Should succeed even with invalid payload type")
		})

		t.Run("PauseNotFound", func(t *testing.T) {
			// KindPause with non-existent pause ID should skip pause deletion
			nonExistentPauseID := uuid.New()
			queueItem := &queue.QueueItem{
				ID:          ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String(),
				AtMS:        time.Now().UnixMilli(),
				WallTimeMS:  time.Now().UnixMilli(),
				FunctionID:  functionID,
				WorkspaceID: workspaceID,
				QueueName:   nil,
				Data: queue.Item{
					WorkspaceID: workspaceID,
					Kind:        queue.KindPause,
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: workspaceID,
						AppID:       appID,
						WorkflowID:  functionID,
						Key:         "test-pause-not-found",
					},
					Payload: queue.PayloadPauseTimeout{
						PauseID: nonExistentPauseID,
						Pause: state.Pause{
							ID:          nonExistentPauseID,
							WorkspaceID: workspaceID,
							Event:       nil,
						},
					},
				},
			}

			err := queueManager.Enqueue(ctx, queueItem.Data, time.Now(), queue.EnqueueOpts{})
			require.NoError(t, err)

			err = deleteManager.DeleteQueueItem(ctx, shard, queueItem)
			require.NoError(t, err, "Should succeed even when pause is not found")
		})
	})

	t.Run("KindDebounce Edge Cases", func(t *testing.T) {
		t.Run("NilDebouncer", func(t *testing.T) {
			// DeleteManager without debouncer should skip debounce deletion
			deleteManagerNoDebounce, err := NewDeleteManager(
				WithPauseManager(pauseMgr),
				WithBatchManager(batchManager),
			)
			require.NoError(t, err)

			queueItem := &queue.QueueItem{
				ID:          ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String(),
				AtMS:        time.Now().UnixMilli(),
				WallTimeMS:  time.Now().UnixMilli(),
				FunctionID:  functionID,
				WorkspaceID: workspaceID,
				QueueName:   nil,
				Data: queue.Item{
					WorkspaceID: workspaceID,
					Kind:        queue.KindDebounce,
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: workspaceID,
						AppID:       appID,
						WorkflowID:  functionID,
						Key:         "test-debounce-nil-manager",
					},
					Payload: debounce.DebouncePayload{
						DebounceID:  ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader),
						AccountID:   accountID,
						WorkspaceID: workspaceID,
						AppID:       appID,
						FunctionID:  functionID,
					},
				},
			}

			err = queueManager.Enqueue(ctx, queueItem.Data, time.Now(), queue.EnqueueOpts{})
			require.NoError(t, err)

			err = deleteManagerNoDebounce.DeleteQueueItem(ctx, shard, queueItem)
			require.NoError(t, err, "Should succeed even with nil debouncer")
		})

		t.Run("InvalidPayloadType", func(t *testing.T) {
			// KindDebounce with wrong payload type should skip debounce deletion
			queueItem := &queue.QueueItem{
				ID:          ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String(),
				AtMS:        time.Now().UnixMilli(),
				WallTimeMS:  time.Now().UnixMilli(),
				FunctionID:  functionID,
				WorkspaceID: workspaceID,
				QueueName:   nil,
				Data: queue.Item{
					WorkspaceID: workspaceID,
					Kind:        queue.KindDebounce,
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: workspaceID,
						AppID:       appID,
						WorkflowID:  functionID,
						Key:         "test-debounce-invalid-payload",
					},
					Payload: "invalid-payload-type",
				},
			}

			err := queueManager.Enqueue(ctx, queueItem.Data, time.Now(), queue.EnqueueOpts{})
			require.NoError(t, err)

			err = deleteManager.DeleteQueueItem(ctx, shard, queueItem)
			require.NoError(t, err, "Should succeed even with invalid payload type")
		})
	})

	t.Run("KindScheduleBatch Edge Cases", func(t *testing.T) {
		t.Run("NilBatchManager", func(t *testing.T) {
			// DeleteManager without batch manager should skip batch deletion
			deleteManagerNoBatch, err := NewDeleteManager(
				WithPauseManager(pauseMgr),
				WithDebouncer(debouncer),
			)
			require.NoError(t, err)

			queueItem := &queue.QueueItem{
				ID:          ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String(),
				AtMS:        time.Now().UnixMilli(),
				WallTimeMS:  time.Now().UnixMilli(),
				FunctionID:  functionID,
				WorkspaceID: workspaceID,
				QueueName:   nil,
				Data: queue.Item{
					WorkspaceID: workspaceID,
					Kind:        queue.KindScheduleBatch,
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: workspaceID,
						AppID:       appID,
						WorkflowID:  functionID,
						Key:         "test-batch-nil-manager",
					},
					Payload: batch.ScheduleBatchPayload{
						BatchID:     ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader),
						AccountID:   accountID,
						WorkspaceID: workspaceID,
						AppID:       appID,
						FunctionID:  functionID,
					},
				},
			}

			err = queueManager.Enqueue(ctx, queueItem.Data, time.Now(), queue.EnqueueOpts{})
			require.NoError(t, err)

			err = deleteManagerNoBatch.DeleteQueueItem(ctx, shard, queueItem)
			require.NoError(t, err, "Should succeed even with nil batch manager")
		})

		t.Run("InvalidPayloadType", func(t *testing.T) {
			// KindScheduleBatch with wrong payload type should skip batch deletion
			queueItem := &queue.QueueItem{
				ID:          ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String(),
				AtMS:        time.Now().UnixMilli(),
				WallTimeMS:  time.Now().UnixMilli(),
				FunctionID:  functionID,
				WorkspaceID: workspaceID,
				QueueName:   nil,
				Data: queue.Item{
					WorkspaceID: workspaceID,
					Kind:        queue.KindScheduleBatch,
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: workspaceID,
						AppID:       appID,
						WorkflowID:  functionID,
						Key:         "test-batch-invalid-payload",
					},
					Payload: "invalid-payload-type",
				},
			}

			err := queueManager.Enqueue(ctx, queueItem.Data, time.Now(), queue.EnqueueOpts{})
			require.NoError(t, err)

			err = deleteManager.DeleteQueueItem(ctx, shard, queueItem)
			require.NoError(t, err, "Should succeed even with invalid payload type")
		})
	})
}

// stubShard satisfies queue.QueueShard for the final RemoveQueueItem step.
type stubShard struct {
	queue.QueueShard
}

func (stubShard) RemoveQueueItem(context.Context, queue.Scope, string, string) error { return nil }

// recordingBatchManager records the arguments and context pins that reach
// DeleteKeys so we can assert the batch location survives payload decoding.
type recordingBatchManager struct {
	batch.BatchManager

	called     bool
	functionID uuid.UUID
	batchID    ulid.ULID
	cluster    string
	generation string
}

func (m *recordingBatchManager) DeleteKeys(ctx context.Context, functionID uuid.UUID, batchID ulid.ULID) error {
	m.called = true
	m.functionID = functionID
	m.batchID = batchID
	m.cluster = batch.BatchCluster(ctx)
	m.generation = redis_state.BatchGeneration(ctx)
	return nil
}

// TestDeleteQueueItemDecodesPersistedPayloads covers the shape a queue item
// actually has once it has been read back from the queue: queue.Item.Payload is
// a json.RawMessage for every kind that decodePayloadForKind does not decode,
// which includes KindScheduleBatch. A plain type assertion silently skipped
// cleanup for all of those items.
func TestDeleteQueueItemDecodesPersistedPayloads(t *testing.T) {
	ctx := context.Background()
	functionID := uuid.New()
	batchID := ulid.MustNew(ulid.Now(), rand.Reader)

	payload := batch.ScheduleBatchPayload{
		BatchID:         batchID,
		BatchPointer:    "pointer",
		BatchCluster:    "valkey-batching-a",
		BatchGeneration: "01K0T21HZW9DHDZ5P5TQKBN1E6",
		FunctionID:      functionID,
	}

	// Round-trip through the real queue.Item codec rather than hand-building a
	// json.RawMessage, so this breaks if decodePayloadForKind ever starts
	// decoding KindScheduleBatch and the runtime payload shape changes.
	t.Run("queue round-tripped payload still deletes batch keys", func(t *testing.T) {
		encoded, err := json.Marshal(queue.Item{
			Kind:       queue.KindScheduleBatch,
			Identifier: state.Identifier{WorkflowID: functionID},
			Payload:    payload,
		})
		require.NoError(t, err)

		var decoded queue.Item
		require.NoError(t, json.Unmarshal(encoded, &decoded))
		_, isRaw := decoded.Payload.(json.RawMessage)
		require.True(t, isRaw, "queue decoding must leave KindScheduleBatch as json.RawMessage")

		bm := &recordingBatchManager{}
		dm, err := NewDeleteManager(WithBatchManager(bm))
		require.NoError(t, err)

		err = dm.DeleteQueueItem(ctx, stubShard{}, &queue.QueueItem{
			FunctionID: functionID,
			Data:       decoded,
		})
		require.NoError(t, err)

		require.True(t, bm.called, "DeleteKeys must run for a persisted payload")
		require.Equal(t, functionID, bm.functionID)
		require.Equal(t, batchID, bm.batchID)
		require.Equal(t, "valkey-batching-a", bm.cluster, "cluster pin must reach DeleteKeys")
		require.Equal(t, "01K0T21HZW9DHDZ5P5TQKBN1E6", bm.generation, "generation pin must reach DeleteKeys")
	})

	t.Run("struct payload keeps working", func(t *testing.T) {
		bm := &recordingBatchManager{}
		dm, err := NewDeleteManager(WithBatchManager(bm))
		require.NoError(t, err)

		err = dm.DeleteQueueItem(ctx, stubShard{}, &queue.QueueItem{
			FunctionID: functionID,
			Data: queue.Item{
				Kind:       queue.KindScheduleBatch,
				Identifier: state.Identifier{WorkflowID: functionID},
				Payload:    payload,
			},
		})
		require.NoError(t, err)
		require.True(t, bm.called)
		require.Equal(t, "valkey-batching-a", bm.cluster)
	})

	t.Run("legacy payload without pins selects the default namespace", func(t *testing.T) {
		legacy := payload
		legacy.BatchCluster = ""
		legacy.BatchGeneration = ""
		raw, err := json.Marshal(legacy)
		require.NoError(t, err)

		bm := &recordingBatchManager{}
		dm, err := NewDeleteManager(WithBatchManager(bm))
		require.NoError(t, err)

		err = dm.DeleteQueueItem(ctx, stubShard{}, &queue.QueueItem{
			FunctionID: functionID,
			Data: queue.Item{
				Kind:       queue.KindScheduleBatch,
				Identifier: state.Identifier{WorkflowID: functionID},
				Payload:    json.RawMessage(raw),
			},
		})
		require.NoError(t, err)
		require.True(t, bm.called)
		require.Empty(t, bm.cluster)
		require.Empty(t, bm.generation)
	})
}

// notFoundDebouncer reports an absent debounce the way the real debouncer does:
// an ErrDebounceNotFound error rather than a nil item.
type notFoundDebouncer struct {
	debounce.Debouncer
	deleted bool
}

func (d *notFoundDebouncer) GetDebounceItem(context.Context, queue.Scope, ulid.ULID) (*debounce.DebounceItem, error) {
	return nil, debounce.ErrDebounceNotFound
}

func (d *notFoundDebouncer) DeleteDebounceItem(context.Context, queue.Scope, ulid.ULID, debounce.DebounceItem) error {
	d.deleted = true
	return nil
}

// TestDeleteQueueItemAbsentDebounce guards against the queue item becoming
// undeletable. GetDebounceItem reports absence as ErrDebounceNotFound, which is
// expected for a stale timeout job, so cleanup must treat it as already done
// and still remove the queue item instead of retrying forever.
func TestDeleteQueueItemAbsentDebounce(t *testing.T) {
	functionID := uuid.New()
	raw, err := json.Marshal(debounce.DebouncePayload{
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		FunctionID:  functionID,
		DebounceID:  ulid.MustNew(ulid.Now(), rand.Reader),
	})
	require.NoError(t, err)

	deb := &notFoundDebouncer{}
	dm, err := NewDeleteManager(WithDebouncer(deb))
	require.NoError(t, err)

	err = dm.DeleteQueueItem(context.Background(), stubShard{}, &queue.QueueItem{
		FunctionID: functionID,
		Data: queue.Item{
			Kind:       queue.KindDebounce,
			Identifier: state.Identifier{WorkflowID: functionID},
			Payload:    json.RawMessage(raw),
		},
	})
	require.NoError(t, err, "an absent debounce must not make the queue item undeletable")
	require.False(t, deb.deleted, "nothing to delete when the debounce is already gone")
}
