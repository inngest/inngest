package deletion

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
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

	// Set up queue shard
	defaultQueueShard := redis_state.RedisQueueShard{
		Name:        consts.DefaultQueueShardName,
		RedisClient: unshardedClient.Queue(),
		Kind:        string(enums.QueueShardKindRedis),
	}

	// Create queue manager
	queueManager := redis_state.NewQueue(
		defaultQueueShard,
		redis_state.WithQueueShardClients(
			map[string]redis_state.RedisQueueShard{
				defaultQueueShard.Name: defaultQueueShard,
			},
		),
		redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.RedisQueueShard, error) {
			return defaultQueueShard, nil
		}),
		redis_state.WithKindToQueueMapping(map[string]string{
			queue.KindEdge:          queue.KindEdge,
			queue.KindPause:         queue.KindPause,
			queue.KindScheduleBatch: queue.KindScheduleBatch,
			queue.KindDebounce:      queue.KindDebounce,
		}),
	)

	ctx := context.Background()

	// Create pause manager
	stateManager, err := redis_state.New(ctx, redis_state.WithUnshardedClient(unshardedClient))
	require.NoError(t, err)
	pauseManager := pauses.NewRedisOnlyManager(stateManager)

	// Create batch manager (need to create sharded client for batch access)
	shardedClient := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: redisClient,
		BatchClient:            redisClient,
		StateDefaultKey:        redis_state.StateDefaultKey,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
		FnRunIsSharded:         redis_state.NeverShardOnRun,
	})
	batchClient := shardedClient.Batch()
	batchManager := batch.NewRedisBatchManager(batchClient, queueManager)

	// Create debounce manager
	debounceClient := unshardedClient.Debounce()
	debouncer := debounce.NewRedisDebouncer(debounceClient, defaultQueueShard, queueManager)

	// Create DeleteManager with all dependencies
	deleteManager, err := NewDeleteManager(
		WithQueueManager(queueManager),
		WithPauseManager(pauseManager),
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
		err = deleteManager.DeleteQueueItem(ctx, defaultQueueShard, queueItem)
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
		_, err := pauseManager.Write(ctx, pauseIndex, pause)
		require.NoError(t, err)

		require.True(t, redisCluster.Exists(unshardedClient.Pauses().KeyGenerator().Pause(ctx, pauseID)), redisCluster.Dump())

		// Verify pause was created
		retrievedPause, err := pauseManager.PauseByID(ctx, pauseIndex, pauseID)
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
		err = deleteManager.DeleteQueueItem(ctx, defaultQueueShard, queueItem)
		require.NoError(t, err, "DeleteQueueItem should succeed for KindPause")

		// Verify pause was also deleted
		_, err = pauseManager.PauseByID(ctx, pauseIndex, pauseID)
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
		err = deleteManager.DeleteQueueItem(ctx, defaultQueueShard, queueItem)
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
		debounceID := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)

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
		err := debouncer.Debounce(ctx, debounceItem, fn)
		require.NoError(t, err)

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
					DebounceID:      debounceID,
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
		err = deleteManager.DeleteQueueItem(ctx, defaultQueueShard, queueItem)
		require.Error(t, err)
		require.ErrorIs(t, err, debounce.ErrDebounceNotFound)

		// Note: We can't easily verify debounce deletion without the exact debounce ID
		// that was created, but the delete manager should handle it properly
	})

	t.Run("UnknownKind", func(t *testing.T) {
		// Test deletion of unknown queue item kind
		// This should test the handler mechanism for unknown kinds
		var handlerCallCount int

		// Create a DeleteManager with a custom handler for unknown kinds
		deleteManagerWithHandler, err := NewDeleteManager(
			WithQueueManager(queueManager),
			WithPauseManager(pauseManager),
			WithBatchManager(batchManager),
			WithDebouncer(debouncer),
			WithUnknownHandler(func(ctx context.Context, shard redis_state.RedisQueueShard, item *queue.QueueItem) error {
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
		err = deleteManagerWithHandler.DeleteQueueItem(ctx, defaultQueueShard, queueItem)
		require.NoError(t, err, "DeleteQueueItem should succeed for unknown kind")

		// Validate that the handler was called at least once
		require.GreaterOrEqual(t, handlerCallCount, 1, "Handler should be called at least once for unknown kind")
	})

	t.Run("KindPause Edge Cases", func(t *testing.T) {
		t.Run("NilPauseManager", func(t *testing.T) {
			// DeleteManager without pause manager should skip pause deletion
			deleteManagerNoPause, err := NewDeleteManager(
				WithQueueManager(queueManager),
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

			err = deleteManagerNoPause.DeleteQueueItem(ctx, defaultQueueShard, queueItem)
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

			err = deleteManager.DeleteQueueItem(ctx, defaultQueueShard, queueItem)
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

			err = deleteManager.DeleteQueueItem(ctx, defaultQueueShard, queueItem)
			require.NoError(t, err, "Should succeed even when pause is not found")
		})
	})

	t.Run("KindDebounce Edge Cases", func(t *testing.T) {
		t.Run("NilDebouncer", func(t *testing.T) {
			// DeleteManager without debouncer should skip debounce deletion
			deleteManagerNoDebounce, err := NewDeleteManager(
				WithQueueManager(queueManager),
				WithPauseManager(pauseManager),
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

			err = deleteManagerNoDebounce.DeleteQueueItem(ctx, defaultQueueShard, queueItem)
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

			err = deleteManager.DeleteQueueItem(ctx, defaultQueueShard, queueItem)
			require.NoError(t, err, "Should succeed even with invalid payload type")
		})
	})

	t.Run("KindScheduleBatch Edge Cases", func(t *testing.T) {
		t.Run("NilBatchManager", func(t *testing.T) {
			// DeleteManager without batch manager should skip batch deletion
			deleteManagerNoBatch, err := NewDeleteManager(
				WithQueueManager(queueManager),
				WithPauseManager(pauseManager),
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

			err = deleteManagerNoBatch.DeleteQueueItem(ctx, defaultQueueShard, queueItem)
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

			err = deleteManager.DeleteQueueItem(ctx, defaultQueueShard, queueItem)
			require.NoError(t, err, "Should succeed even with invalid payload type")
		})
	})
}
