package debounce

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// TestDebounce ensures the debounce feature works in general.
func TestDebounce(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)

	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	debounceClient := unshardedClient.Debounce()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	}

	shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)

	shardRegistry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := queue.New(
		context.Background(),
		"debounce-test",
		shardRegistry,
		opts...,
	)
	require.NoError(t, err)
	kg := shard.Client().KeyGenerator()

	fakeClock := clockwork.NewFakeClock()

	redisDebouncer := NewRedisDebouncer(debounceClient, shard, q).(debouncer)
	redisDebouncer.c = fakeClock

	ctx := context.Background()
	accountId, workspaceId, appId, functionId := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ID: functionId,
		Debounce: &inngest.Debounce{
			Key:     nil,
			Period:  "10s",
			Timeout: util.StrPtr("60s"),
		},
	}

	evt0Time := fakeClock.Now()

	t.Run("create debounce should work", func(t *testing.T) {
		eventTime := evt0Time

		eventId := ulid.MustNew(ulid.Timestamp(eventTime), rand.Reader)

		expectedDi := DebounceItem{
			AccountID:   accountId,
			WorkspaceID: workspaceId,
			AppID:       appId,
			FunctionID:  functionId,
			EventID:     eventId,
			Event: event.Event{
				Name:      "test-data",
				ID:        eventId.String(),
				Timestamp: eventTime.UnixMilli(),
			},
		}

		err := redisDebouncer.Debounce(ctx, expectedDi, fn)
		require.NoError(t, err)

		expectedDi.Timeout = eventTime.Add(60 * time.Second).UnixMilli()

		ttl := unshardedCluster.TTL(debounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.Equal(t, 10*time.Second, ttl, "expected ttl to match", unshardedCluster.Keys())

		debounceIds, err := unshardedCluster.HKeys(debounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		debounceId := ulid.MustParse(debounceIds[0])

		var di DebounceItem
		err = json.Unmarshal([]byte(unshardedCluster.HGet(debounceClient.KeyGenerator().Debounce(ctx), debounceIds[0])), &di)
		require.NoError(t, err)

		require.NotEmpty(t, debounceIds[0])
		di.Event.ClearSize()
		require.Equal(t, expectedDi, di)

		// Queue state should match
		{
			queueItemIds, err := unshardedCluster.HKeys(kg.QueueItem())
			require.NoError(t, err)
			require.Len(t, queueItemIds, 1)

			var qi queue.QueueItem
			err = json.Unmarshal([]byte(unshardedCluster.HGet(kg.QueueItem(), queueItemIds[0])), &qi)
			require.NoError(t, err)

			require.Equal(t, queue.KindDebounce, qi.Data.Kind)

			expectedPayload := di.QueuePayload()
			expectedPayload.DebounceID = debounceId

			rawPayload := qi.Data.Payload.(json.RawMessage)

			var payload DebouncePayload
			err = json.Unmarshal(rawPayload, &payload)
			require.NoError(t, err)

			require.Equal(t, expectedPayload, payload)

			itemScore, err := unshardedCluster.ZScore(kg.PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
			require.NoError(t, err)
			expectedQueueScore := eventTime.
				Add(10 * time.Second).       // Debounce period
				Add(buffer).                 // Buffer
				Add(time.Second).UnixMilli() // Allow updateDebounce on TTL 0
			require.Equal(t, expectedQueueScore, int64(itemScore))
		}
	})

	t.Run("update debounce should work", func(t *testing.T) {
		unshardedCluster.FastForward(5 * time.Second)
		fakeClock.Advance(5 * time.Second)

		eventTime := fakeClock.Now()

		eventId := ulid.MustNew(ulid.Timestamp(eventTime), rand.Reader)

		expectedDi := DebounceItem{
			AccountID:   accountId,
			WorkspaceID: workspaceId,
			AppID:       appId,
			FunctionID:  functionId,
			EventID:     eventId,
			Event: event.Event{
				Name:      "test-data",
				ID:        eventId.String(),
				Timestamp: eventTime.UnixMilli(),
			},
		}

		// Time has passed, so TTL was decreased
		ttl := unshardedCluster.TTL(debounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.Equal(t, 5*time.Second, ttl, "expected ttl to match", unshardedCluster.Keys())

		err := redisDebouncer.Debounce(ctx, expectedDi, fn)
		require.NoError(t, err)

		expectedDi.Timeout = evt0Time.Add(60 * time.Second).UnixMilli() // Must match initial event, timeout may never change

		// TTL is reset
		ttl = unshardedCluster.TTL(debounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.Equal(t, 10*time.Second, ttl, "expected ttl to match", unshardedCluster.Keys())

		debounceIds, err := unshardedCluster.HKeys(debounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		debounceId := ulid.MustParse(debounceIds[0])

		var di DebounceItem
		err = json.Unmarshal([]byte(unshardedCluster.HGet(debounceClient.KeyGenerator().Debounce(ctx), debounceIds[0])), &di)
		require.NoError(t, err)

		require.NotEmpty(t, debounceIds[0])
		di.Event.ClearSize()
		require.Equal(t, expectedDi, di)

		// Queue state should match
		{
			queueItemIds, err := unshardedCluster.HKeys(kg.QueueItem())
			require.NoError(t, err)
			require.Len(t, queueItemIds, 1)

			var qi queue.QueueItem
			err = json.Unmarshal([]byte(unshardedCluster.HGet(kg.QueueItem(), queueItemIds[0])), &qi)
			require.NoError(t, err)

			require.Equal(t, queue.KindDebounce, qi.Data.Kind)

			expectedPayload := di.QueuePayload()
			expectedPayload.DebounceID = debounceId

			rawPayload := qi.Data.Payload.(json.RawMessage)

			var payload DebouncePayload
			err = json.Unmarshal(rawPayload, &payload)
			require.NoError(t, err)

			require.Equal(t, expectedPayload, payload)

			itemScore, err := unshardedCluster.ZScore(kg.PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
			require.NoError(t, err)

			initialScore := evt0Time.
				Add(10 * time.Second).       // Debounce period
				Add(buffer).                 // Buffer
				Add(time.Second).UnixMilli() // Allow updateDebounce on TTL 0
			expectedRequeueScore := eventTime.
				Add(10 * time.Second).       // Debounce period
				Add(buffer).                 // Buffer
				Add(time.Second).UnixMilli() // Allow updateDebounce on TTL 0

			require.NotEqual(t, initialScore, expectedRequeueScore)
			require.Equal(t, expectedRequeueScore, int64(itemScore))
		}
	})

	t.Run("start debounce should work", func(t *testing.T) {
		debounceIds, err := unshardedCluster.HKeys(debounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		debounceId := ulid.MustParse(debounceIds[0])

		val, err := unshardedCluster.Get(debounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.NoError(t, err)
		require.Equal(t, debounceId.String(), val)

		di, err := redisDebouncer.GetDebounceItem(ctx, debounceId, accountId)
		require.NoError(t, err)

		err = redisDebouncer.StartExecution(ctx, *di, fn, debounceId)
		require.NoError(t, err)

		val, err = unshardedCluster.Get(debounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.NoError(t, err)
		require.NotEmpty(t, debounceId.String(), val)
	})

	t.Run("delete debounce should work", func(t *testing.T) {
		debounceIds, err := unshardedCluster.HKeys(debounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		debounceId := ulid.MustParse(debounceIds[0])

		var di DebounceItem
		err = json.Unmarshal([]byte(unshardedCluster.HGet(debounceClient.KeyGenerator().Debounce(ctx), debounceIds[0])), &di)
		require.NoError(t, err)

		err = redisDebouncer.DeleteDebounceItem(ctx, debounceId, di, accountId)
		require.NoError(t, err)

		_, err = unshardedCluster.HKeys(debounceClient.KeyGenerator().Debounce(ctx))
		require.Error(t, err)
		require.ErrorContains(t, err, "no such key")
	})
}

func TestGetDebounceInfo(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)

	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	debounceClient := unshardedClient.Debounce()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	}

	shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)

	shardRegistry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := queue.New(
		context.Background(),
		"debounce-test",
		shardRegistry,
		opts...,
	)
	require.NoError(t, err)

	redisDebouncer := NewRedisDebouncer(debounceClient, shard, q)

	ctx := context.Background()
	accountId, workspaceId, appId, functionId := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	t.Run("no debounce exists returns empty info", func(t *testing.T) {
		info, err := redisDebouncer.GetDebounceInfo(ctx, functionId, functionId.String())
		require.NoError(t, err)
		require.Equal(t, "", info.DebounceID)
		require.Nil(t, info.Item)
	})

	t.Run("debounce with default key", func(t *testing.T) {
		fn := inngest.Function{
			ID: functionId,
			Debounce: &inngest.Debounce{
				Key:     nil, // Uses function ID as key
				Period:  "10s",
				Timeout: util.StrPtr("60s"),
			},
		}

		eventId := ulid.MustNew(ulid.Now(), rand.Reader)
		di := DebounceItem{
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      functionId,
			FunctionVersion: 1,
			EventID:         eventId,
			Event: event.Event{
				Name:      "test/debounce-event",
				ID:        eventId.String(),
				Timestamp: time.Now().UnixMilli(),
				Data:      map[string]any{"key": "value"},
			},
		}

		err := redisDebouncer.Debounce(ctx, di, fn)
		require.NoError(t, err)

		// Query with function ID as debounce key (default)
		info, err := redisDebouncer.GetDebounceInfo(ctx, functionId, functionId.String())
		require.NoError(t, err)
		require.NotEmpty(t, info.DebounceID)
		require.NotNil(t, info.Item)
		require.Equal(t, eventId, info.Item.EventID)
		require.Equal(t, accountId, info.Item.AccountID)
		require.Equal(t, workspaceId, info.Item.WorkspaceID)
		require.Equal(t, functionId, info.Item.FunctionID)
	})

	t.Run("debounce with custom key", func(t *testing.T) {
		customFnId := uuid.New()
		customKey := "custom-debounce-key"

		fn := inngest.Function{
			ID: customFnId,
			Debounce: &inngest.Debounce{
				Key:     util.StrPtr("event.data.debounce_key"),
				Period:  "10s",
				Timeout: util.StrPtr("60s"),
			},
		}

		eventId := ulid.MustNew(ulid.Now(), rand.Reader)
		di := DebounceItem{
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      customFnId,
			FunctionVersion: 1,
			EventID:         eventId,
			Event: event.Event{
				Name:      "test/debounce-event",
				ID:        eventId.String(),
				Timestamp: time.Now().UnixMilli(),
				Data:      map[string]any{"debounce_key": customKey},
			},
		}

		err := redisDebouncer.Debounce(ctx, di, fn)
		require.NoError(t, err)

		// Query with the custom key
		info, err := redisDebouncer.GetDebounceInfo(ctx, customFnId, customKey)
		require.NoError(t, err)
		require.NotEmpty(t, info.DebounceID)
		require.NotNil(t, info.Item)
		require.Equal(t, eventId, info.Item.EventID)

		// Query with wrong key should return empty
		info2, err := redisDebouncer.GetDebounceInfo(ctx, customFnId, "wrong-key")
		require.NoError(t, err)
		require.Equal(t, "", info2.DebounceID)
		require.Nil(t, info2.Item)
	})

	t.Run("debounce updates preserve latest event", func(t *testing.T) {
		updateFnId := uuid.New()

		fn := inngest.Function{
			ID: updateFnId,
			Debounce: &inngest.Debounce{
				Key:     nil,
				Period:  "10s",
				Timeout: util.StrPtr("60s"),
			},
		}

		// Add first event
		eventId1 := ulid.MustNew(ulid.Now(), rand.Reader)
		di1 := DebounceItem{
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      updateFnId,
			FunctionVersion: 1,
			EventID:         eventId1,
			Event: event.Event{
				Name:      "test/debounce-event",
				ID:        eventId1.String(),
				Timestamp: time.Now().UnixMilli(),
				Data:      map[string]any{"version": 1},
			},
		}
		err := redisDebouncer.Debounce(ctx, di1, fn)
		require.NoError(t, err)

		// Add second event (update)
		eventId2 := ulid.MustNew(ulid.Now(), rand.Reader)
		di2 := DebounceItem{
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      updateFnId,
			FunctionVersion: 1,
			EventID:         eventId2,
			Event: event.Event{
				Name:      "test/debounce-event",
				ID:        eventId2.String(),
				Timestamp: time.Now().UnixMilli(),
				Data:      map[string]any{"version": 2},
			},
		}
		err = redisDebouncer.Debounce(ctx, di2, fn)
		require.NoError(t, err)

		// Query should return the latest event
		info, err := redisDebouncer.GetDebounceInfo(ctx, updateFnId, updateFnId.String())
		require.NoError(t, err)
		require.NotEmpty(t, info.DebounceID)
		require.NotNil(t, info.Item)
		require.Equal(t, eventId2, info.Item.EventID)
	})

	t.Run("non-existent function returns empty", func(t *testing.T) {
		nonExistentFnId := uuid.New()
		info, err := redisDebouncer.GetDebounceInfo(ctx, nonExistentFnId, nonExistentFnId.String())
		require.NoError(t, err)
		require.Equal(t, "", info.DebounceID)
		require.Nil(t, info.Item)
	})
}

func TestDeleteDebounce(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)

	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	debounceClient := unshardedClient.Debounce()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	}

	shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)

	shardRegistry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := queue.New(
		context.Background(),
		"debounce-test",
		shardRegistry,
		opts...,
	)
	require.NoError(t, err)

	redisDebouncer := NewRedisDebouncer(debounceClient, shard, q)

	ctx := context.Background()
	accountId, workspaceId, appId := uuid.New(), uuid.New(), uuid.New()

	t.Run("delete non-existent debounce returns deleted=false", func(t *testing.T) {
		nonExistentFnId := uuid.New()
		result, err := redisDebouncer.DeleteDebounce(ctx, nonExistentFnId, nonExistentFnId.String())
		require.NoError(t, err)
		require.False(t, result.Deleted)
		require.Equal(t, "", result.DebounceID)
		require.Equal(t, "", result.EventID)
	})

	t.Run("delete existing debounce", func(t *testing.T) {
		functionId := uuid.New()
		fn := inngest.Function{
			ID: functionId,
			Debounce: &inngest.Debounce{
				Key:     nil,
				Period:  "10s",
				Timeout: util.StrPtr("60s"),
			},
		}

		eventId := ulid.MustNew(ulid.Now(), rand.Reader)
		di := DebounceItem{
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      functionId,
			FunctionVersion: 1,
			EventID:         eventId,
			Event: event.Event{
				Name:      "test/debounce-event",
				ID:        eventId.String(),
				Timestamp: time.Now().UnixMilli(),
				Data:      map[string]any{"key": "value"},
			},
		}

		err := redisDebouncer.Debounce(ctx, di, fn)
		require.NoError(t, err)

		// Verify debounce exists
		info, err := redisDebouncer.GetDebounceInfo(ctx, functionId, functionId.String())
		require.NoError(t, err)
		require.NotEmpty(t, info.DebounceID)
		require.NotNil(t, info.Item)
		debounceID := info.DebounceID

		// Delete the debounce
		result, err := redisDebouncer.DeleteDebounce(ctx, functionId, functionId.String())
		require.NoError(t, err)
		require.True(t, result.Deleted)
		require.Equal(t, debounceID, result.DebounceID)
		require.Equal(t, eventId.String(), result.EventID)

		// Verify debounce no longer exists
		infoAfter, err := redisDebouncer.GetDebounceInfo(ctx, functionId, functionId.String())
		require.NoError(t, err)
		require.Equal(t, "", infoAfter.DebounceID)
		require.Nil(t, infoAfter.Item)
	})
}

func TestRunDebounce(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)

	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	debounceClient := unshardedClient.Debounce()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	}

	shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)

	shardRegistry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := queue.New(
		context.Background(),
		"debounce-test",
		shardRegistry,
		opts...,
	)
	require.NoError(t, err)

	redisDebouncer := NewRedisDebouncer(debounceClient, shard, q)

	ctx := context.Background()
	accountId, workspaceId, appId := uuid.New(), uuid.New(), uuid.New()

	t.Run("run non-existent debounce returns scheduled=false", func(t *testing.T) {
		nonExistentFnId := uuid.New()
		result, err := redisDebouncer.RunDebounce(ctx, RunDebounceOpts{
			FunctionID:  nonExistentFnId,
			DebounceKey: nonExistentFnId.String(),
			AccountID:   accountId,
		})
		require.NoError(t, err)
		require.False(t, result.Scheduled)
		require.Equal(t, "", result.DebounceID)
		require.Equal(t, "", result.EventID)
	})

	t.Run("run existing debounce", func(t *testing.T) {
		functionId := uuid.New()
		fn := inngest.Function{
			ID: functionId,
			Debounce: &inngest.Debounce{
				Key:     nil,
				Period:  "10s",
				Timeout: util.StrPtr("60s"),
			},
		}

		eventId := ulid.MustNew(ulid.Now(), rand.Reader)
		di := DebounceItem{
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      functionId,
			FunctionVersion: 1,
			EventID:         eventId,
			Event: event.Event{
				Name:      "test/debounce-event",
				ID:        eventId.String(),
				Timestamp: time.Now().UnixMilli(),
				Data:      map[string]any{"key": "value"},
			},
		}

		err := redisDebouncer.Debounce(ctx, di, fn)
		require.NoError(t, err)

		// Verify debounce exists
		info, err := redisDebouncer.GetDebounceInfo(ctx, functionId, functionId.String())
		require.NoError(t, err)
		require.NotEmpty(t, info.DebounceID)
		debounceID := info.DebounceID

		// Run the debounce
		result, err := redisDebouncer.RunDebounce(ctx, RunDebounceOpts{
			FunctionID:  functionId,
			DebounceKey: functionId.String(),
			AccountID:   accountId,
		})
		require.NoError(t, err)
		require.True(t, result.Scheduled)
		require.Equal(t, debounceID, result.DebounceID)
		require.Equal(t, eventId.String(), result.EventID)
	})
}

func TestDeleteDebounceByID(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)

	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	debounceClient := unshardedClient.Debounce()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	}

	shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)

	shardRegistry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := queue.New(
		context.Background(),
		"debounce-test",
		shardRegistry,
		opts...,
	)
	require.NoError(t, err)

	redisDebouncer := NewRedisDebouncer(debounceClient, shard, q)

	ctx := context.Background()
	accountId, workspaceId, appId := uuid.New(), uuid.New(), uuid.New()

	// helper to create a debounce and return its ULID
	createDebounce := func(t *testing.T) (uuid.UUID, ulid.ULID) {
		t.Helper()
		functionId := uuid.New()
		fn := inngest.Function{
			ID: functionId,
			Debounce: &inngest.Debounce{
				Key:     nil,
				Period:  "10s",
				Timeout: util.StrPtr("60s"),
			},
		}

		eventId := ulid.MustNew(ulid.Now(), rand.Reader)
		di := DebounceItem{
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      functionId,
			FunctionVersion: 1,
			EventID:         eventId,
			Event: event.Event{
				Name:      "test/debounce-event",
				ID:        eventId.String(),
				Timestamp: time.Now().UnixMilli(),
				Data:      map[string]any{"key": "value"},
			},
		}

		err := redisDebouncer.Debounce(ctx, di, fn)
		require.NoError(t, err)

		info, err := redisDebouncer.GetDebounceInfo(ctx, functionId, functionId.String())
		require.NoError(t, err)
		require.NotEmpty(t, info.DebounceID)

		debounceID := ulid.MustParse(info.DebounceID)
		return functionId, debounceID
	}

	// hashFieldExists checks if a field exists in a Redis hash using miniredis.
	hashFieldExists := func(key, field string) bool {
		val := unshardedCluster.HGet(key, field)
		return val != ""
	}

	t.Run("no debounce exists should succeed", func(t *testing.T) {
		fakeID := ulid.MustNew(ulid.Now(), rand.Reader)
		err := redisDebouncer.DeleteDebounceByID(ctx, fakeID)
		require.NoError(t, err)
	})

	t.Run("delete current debounce by ID", func(t *testing.T) {
		functionId, debounceID := createDebounce(t)

		// Verify the debounce item exists in the hash
		debounceKey := debounceClient.KeyGenerator().Debounce(ctx)
		require.True(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should exist in hash")

		// Verify the timeout queue item exists
		queueItemId := queue.HashID(ctx, debounceID.String())
		_, err := shard.LoadQueueItem(ctx, queueItemId)
		require.NoError(t, err, "timeout queue item should exist")

		// Delete by ID
		err = redisDebouncer.DeleteDebounceByID(ctx, debounceID)
		require.NoError(t, err)

		// Verify the debounce item is gone from the hash
		require.False(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should be deleted from hash")

		// Verify the timeout queue item is gone
		_, err = shard.LoadQueueItem(ctx, queueItemId)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound, "timeout queue item should be deleted")

		// The pointer key may still exist (DeleteDebounceByID does not clean it up),
		// but GetDebounceInfo should handle this gracefully.
		info, err := redisDebouncer.GetDebounceInfo(ctx, functionId, functionId.String())
		require.NoError(t, err)
		require.Nil(t, info.Item, "debounce item should not be found via pointer")
	})

	t.Run("delete debounce after pointer is dropped", func(t *testing.T) {
		_, debounceID := createDebounce(t)

		debounceKey := debounceClient.KeyGenerator().Debounce(ctx)
		require.True(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should exist in hash before deletion")

		// Verify the timeout queue item exists
		queueItemId := queue.HashID(ctx, debounceID.String())
		_, err := shard.LoadQueueItem(ctx, queueItemId)
		require.NoError(t, err, "timeout queue item should exist")

		// Delete by ID (pointer is gone, but item + timeout still exist)
		err = redisDebouncer.DeleteDebounceByID(ctx, debounceID)
		require.NoError(t, err)

		// Verify item is gone from hash
		require.False(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should be deleted from hash")

		// Verify timeout queue item is gone
		_, err = shard.LoadQueueItem(ctx, queueItemId)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound, "timeout queue item should be deleted")
	})

	t.Run("delete debounce when timeout already removed", func(t *testing.T) {
		_, debounceID := createDebounce(t)

		debounceKey := debounceClient.KeyGenerator().Debounce(ctx)
		require.True(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should exist in hash")

		// Manually remove the timeout queue item first
		queueItemId := queue.HashID(ctx, debounceID.String())
		err := shard.RemoveQueueItem(ctx, queue.KindDebounce, queueItemId)
		require.NoError(t, err)

		// Verify timeout is gone
		_, err = shard.LoadQueueItem(ctx, queueItemId)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound, "timeout should already be gone")

		// Delete by ID should still succeed (timeout removal is best-effort)
		err = redisDebouncer.DeleteDebounceByID(ctx, debounceID)
		require.NoError(t, err)

		// Verify item is gone from hash
		require.False(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should be deleted from hash")
	})

	t.Run("missing item but existing timeout job should clean up timeout", func(t *testing.T) {
		_, debounceID := createDebounce(t)

		debounceKey := debounceClient.KeyGenerator().Debounce(ctx)

		// Manually remove the debounce item from the hash, leaving only the timeout
		unshardedCluster.HDel(debounceKey, debounceID.String())
		require.False(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should be gone from hash")

		// Verify the timeout queue item still exists
		queueItemId := queue.HashID(ctx, debounceID.String())
		_, err := shard.LoadQueueItem(ctx, queueItemId)
		require.NoError(t, err, "timeout queue item should still exist")

		// Delete by ID — HDEL on missing item is a no-op, but timeout should be cleaned up
		err = redisDebouncer.DeleteDebounceByID(ctx, debounceID)
		require.NoError(t, err)

		// Verify timeout queue item is now gone
		_, err = shard.LoadQueueItem(ctx, queueItemId)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound, "timeout queue item should be deleted")
	})

	t.Run("batch delete multiple debounce IDs", func(t *testing.T) {
		_, debounceID1 := createDebounce(t)
		_, debounceID2 := createDebounce(t)

		debounceKey := debounceClient.KeyGenerator().Debounce(ctx)

		// Both items should exist
		require.True(t, hashFieldExists(debounceKey, debounceID1.String()))
		require.True(t, hashFieldExists(debounceKey, debounceID2.String()))

		queueItemId1 := queue.HashID(ctx, debounceID1.String())
		queueItemId2 := queue.HashID(ctx, debounceID2.String())

		_, err := shard.LoadQueueItem(ctx, queueItemId1)
		require.NoError(t, err)
		_, err = shard.LoadQueueItem(ctx, queueItemId2)
		require.NoError(t, err)

		// Batch delete both
		err = redisDebouncer.DeleteDebounceByID(ctx, debounceID1, debounceID2)
		require.NoError(t, err)

		// Both items should be gone from hash
		require.False(t, hashFieldExists(debounceKey, debounceID1.String()))
		require.False(t, hashFieldExists(debounceKey, debounceID2.String()))

		// Both timeout queue items should be gone
		_, err = shard.LoadQueueItem(ctx, queueItemId1)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
		_, err = shard.LoadQueueItem(ctx, queueItemId2)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
	})

	t.Run("empty ID list is a no-op", func(t *testing.T) {
		err := redisDebouncer.DeleteDebounceByID(ctx)
		require.NoError(t, err)
	})
}
