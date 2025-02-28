package debounce

import (
	"context"
	"crypto/rand"
	"encoding/json"
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
	"testing"
	"time"
)

func TestDebounce(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)

	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	debounceClient := unshardedClient.Debounce()

	defaultQueueShard := redis_state.QueueShard{Name: consts.DefaultQueueShardName, RedisClient: unshardedClient.Queue(), Kind: string(enums.QueueShardKindRedis)}

	q := redis_state.NewQueue(
		defaultQueueShard,
		redis_state.WithQueueShardClients(
			map[string]redis_state.QueueShard{
				defaultQueueShard.Name: defaultQueueShard,
			},
		),
		redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.QueueShard, error) {
			return defaultQueueShard, nil
		}),
	)

	fakeClock := clockwork.NewFakeClock()

	redisDebouncer := NewRedisDebouncer(debounceClient, defaultQueueShard, q).(debouncer)
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
		unshardedCluster.SetTime(fakeClock.Now())

		eventId := ulid.MustNew(ulid.Timestamp(eventTime), rand.Reader)

		expectedDi := DebounceItem{
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      functionId,
			FunctionVersion: 0,
			EventID:         eventId,
			Event: event.Event{
				Name:      "test-data",
				Data:      nil,
				User:      nil,
				ID:        eventId.String(),
				Timestamp: eventTime.UnixMilli(),
				Version:   "",
			},
			Timeout:          eventTime.Add(60 * time.Second).UnixMilli(),
			FunctionPausedAt: nil,
		}

		err := redisDebouncer.Debounce(ctx, expectedDi, fn)
		require.NoError(t, err)

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
		require.Equal(t, expectedDi, di)

		// Queue state should match
		{
			queueItemIds, err := unshardedCluster.HKeys(defaultQueueShard.RedisClient.KeyGenerator().QueueItem())
			require.NoError(t, err)
			require.Len(t, queueItemIds, 1)

			var qi queue.QueueItem
			err = json.Unmarshal([]byte(unshardedCluster.HGet(defaultQueueShard.RedisClient.KeyGenerator().QueueItem(), queueItemIds[0])), &qi)
			require.NoError(t, err)

			require.Equal(t, queue.KindDebounce, qi.Data.Kind)

			expectedPayload := di.QueuePayload()
			expectedPayload.DebounceID = debounceId

			rawPayload := qi.Data.Payload.(json.RawMessage)

			var payload DebouncePayload
			err = json.Unmarshal(rawPayload, &payload)
			require.NoError(t, err)

			require.Equal(t, expectedPayload, payload)

			itemScore, err := unshardedCluster.ZScore(defaultQueueShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, functionId.String(), ""), qi.ID)
			require.NoError(t, err)
			expectedQueueScore := eventTime.
				Add(10 * time.Second). // Debounce period
				Add(50 * time.Millisecond). // Buffer
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
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      functionId,
			FunctionVersion: 0,
			EventID:         eventId,
			Event: event.Event{
				Name:      "test-data",
				Data:      nil,
				User:      nil,
				ID:        eventId.String(),
				Timestamp: eventTime.UnixMilli(),
				Version:   "",
			},
			Timeout:          evt0Time.Add(60 * time.Second).UnixMilli(), // Must match initial event, timeout may never change
			FunctionPausedAt: nil,
		}

		// Time has passed, so TTL was decreased
		ttl := unshardedCluster.TTL(debounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.Equal(t, 5*time.Second, ttl, "expected ttl to match", unshardedCluster.Keys())

		err := redisDebouncer.Debounce(ctx, expectedDi, fn)
		require.NoError(t, err)

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
		require.Equal(t, expectedDi, di)

		// Queue state should match
		{
			queueItemIds, err := unshardedCluster.HKeys(defaultQueueShard.RedisClient.KeyGenerator().QueueItem())
			require.NoError(t, err)
			require.Len(t, queueItemIds, 1)

			var qi queue.QueueItem
			err = json.Unmarshal([]byte(unshardedCluster.HGet(defaultQueueShard.RedisClient.KeyGenerator().QueueItem(), queueItemIds[0])), &qi)
			require.NoError(t, err)

			require.Equal(t, queue.KindDebounce, qi.Data.Kind)

			expectedPayload := di.QueuePayload()
			expectedPayload.DebounceID = debounceId

			rawPayload := qi.Data.Payload.(json.RawMessage)

			var payload DebouncePayload
			err = json.Unmarshal(rawPayload, &payload)
			require.NoError(t, err)

			require.Equal(t, expectedPayload, payload)

			itemScore, err := unshardedCluster.ZScore(defaultQueueShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, functionId.String(), ""), qi.ID)
			require.NoError(t, err)

			initialScore := evt0Time.
				Add(10 * time.Second). // Debounce period
				Add(50 * time.Millisecond). // Buffer
				Add(time.Second).UnixMilli() // Allow updateDebounce on TTL 0
			expectedRequeueScore := eventTime.
				Add(10 * time.Second). // Debounce period
				Add(50 * time.Millisecond). // Buffer
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

		di, err := redisDebouncer.GetDebounceItem(ctx, debounceId)
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

		err = redisDebouncer.DeleteDebounceItem(ctx, debounceId)
		require.NoError(t, err)

		debounceIds, err = unshardedCluster.HKeys(debounceClient.KeyGenerator().Debounce(ctx))
		require.Error(t, err)
		require.ErrorContains(t, err, "no such key")
	})
}

func TestDebounceWithMigration(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)
	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	unshardedDebounceClient := unshardedClient.Debounce()

	defaultQueueShard := redis_state.QueueShard{Name: consts.DefaultQueueShardName, RedisClient: unshardedClient.Queue(), Kind: string(enums.QueueShardKindRedis)}

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.QueueShard{Name: "new-system", RedisClient: newSystemClusterClient.Queue(), Kind: string(enums.QueueShardKindRedis)}
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	// TODO What happens if both old and new services are running? Does this break debounces?
	//  Do we need to keep the old behavior and flip using a feature flag (LaunchDarkly)
	//  once all services running `Schedule` (new-runs, executor) are rolled out?
	oldQueue := redis_state.NewQueue(
		defaultQueueShard,
		redis_state.WithQueueShardClients(
			map[string]redis_state.QueueShard{
				defaultQueueShard.Name: defaultQueueShard,
			},
		),
		redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.QueueShard, error) {
			return defaultQueueShard, nil
		}),
	)

	newQueue := redis_state.NewQueue(
		newSystemShard, // Primary
		redis_state.WithQueueShardClients(
			map[string]redis_state.QueueShard{
				defaultQueueShard.Name: defaultQueueShard,
				newSystemShard.Name:    newSystemShard,
			},
		),
		redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.QueueShard, error) {
			// Enqueue new system queue items to new system queue shard
			if queueName != nil {
				return newSystemShard, nil
			}

			return defaultQueueShard, nil
		}),
	)

	fakeClock := clockwork.NewFakeClock()

	oldRedisDebouncer := NewRedisDebouncer(unshardedDebounceClient, defaultQueueShard, oldQueue).(debouncer)
	oldRedisDebouncer.c = fakeClock

	newRedisDebouncer := NewRedisDebouncerWithMigration(DebouncerOpts{
		DefaultDebounceClient: unshardedDebounceClient,
		DefaultQueue:          oldQueue,
		DefaultQueueShard:     defaultQueueShard,

		SystemDebounceClient: newSystemDebounceClient,
		SystemQueue:          newQueue,
		SystemQueueShard:     newSystemShard,
		ShouldMigrate: func(ctx context.Context) bool {
			return true
		},
	}).(debouncer)
	newRedisDebouncer.c = fakeClock

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

	t.Run("create debounce on old queue should work", func(t *testing.T) {
		eventTime := evt0Time
		unshardedCluster.SetTime(fakeClock.Now())

		eventId := ulid.MustNew(ulid.Timestamp(eventTime), rand.Reader)

		expectedDi := DebounceItem{
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      functionId,
			FunctionVersion: 0,
			EventID:         eventId,
			Event: event.Event{
				Name:      "test-data",
				Data:      nil,
				User:      nil,
				ID:        eventId.String(),
				Timestamp: eventTime.UnixMilli(),
				Version:   "",
			},
			Timeout:          eventTime.Add(60 * time.Second).UnixMilli(),
			FunctionPausedAt: nil,
		}

		err := oldRedisDebouncer.Debounce(ctx, expectedDi, fn)
		require.NoError(t, err)

		ttl := unshardedCluster.TTL(unshardedDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.Equal(t, 10*time.Second, ttl, "expected ttl to match", unshardedCluster.Keys())

		debounceIds, err := unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		debounceId := ulid.MustParse(debounceIds[0])

		var di DebounceItem
		err = json.Unmarshal([]byte(unshardedCluster.HGet(unshardedDebounceClient.KeyGenerator().Debounce(ctx), debounceIds[0])), &di)
		require.NoError(t, err)

		require.NotEmpty(t, debounceIds[0])
		require.Equal(t, expectedDi, di)

		// Queue state should match
		{
			queueItemIds, err := unshardedCluster.HKeys(defaultQueueShard.RedisClient.KeyGenerator().QueueItem())
			require.NoError(t, err)
			require.Len(t, queueItemIds, 1)

			var qi queue.QueueItem
			err = json.Unmarshal([]byte(unshardedCluster.HGet(defaultQueueShard.RedisClient.KeyGenerator().QueueItem(), queueItemIds[0])), &qi)
			require.NoError(t, err)

			require.Equal(t, queue.KindDebounce, qi.Data.Kind)

			expectedPayload := di.QueuePayload()
			expectedPayload.DebounceID = debounceId

			rawPayload := qi.Data.Payload.(json.RawMessage)

			var payload DebouncePayload
			err = json.Unmarshal(rawPayload, &payload)
			require.NoError(t, err)

			require.Equal(t, expectedPayload, payload)

			itemScore, err := unshardedCluster.ZScore(defaultQueueShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, functionId.String(), ""), qi.ID)
			require.NoError(t, err)
			expectedQueueScore := eventTime.
				Add(10 * time.Second). // Debounce period
				Add(50 * time.Millisecond). // Buffer
				Add(time.Second).UnixMilli() // Allow updateDebounce on TTL 0
			require.Equal(t, expectedQueueScore, int64(itemScore))
		}
	})

	t.Run("update and migrate debounce should work", func(t *testing.T) {
		unshardedCluster.FastForward(5 * time.Second)
		fakeClock.Advance(5 * time.Second)

		eventTime := fakeClock.Now()

		eventId := ulid.MustNew(ulid.Timestamp(eventTime), rand.Reader)

		expectedDi := DebounceItem{
			AccountID:       accountId,
			WorkspaceID:     workspaceId,
			AppID:           appId,
			FunctionID:      functionId,
			FunctionVersion: 0,
			EventID:         eventId,
			Event: event.Event{
				Name:      "test-data",
				Data:      nil,
				User:      nil,
				ID:        eventId.String(),
				Timestamp: eventTime.UnixMilli(),
				Version:   "",
			},
			Timeout:          evt0Time.Add(60 * time.Second).UnixMilli(), // Must match initial event, timeout may never change
			FunctionPausedAt: nil,
		}

		// Time has passed, so TTL was decreased
		ttl := unshardedCluster.TTL(unshardedDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.Equal(t, 5*time.Second, ttl, "expected ttl to match", unshardedCluster.Keys())

		err := newRedisDebouncer.Debounce(ctx, expectedDi, fn)
		require.NoError(t, err)

		// TTL is reset
		ttl = newSystemCluster.TTL(newSystemDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.Equal(t, 10*time.Second, ttl, "expected ttl to match", newSystemCluster.Keys())

		debounceIds, err := newSystemCluster.HKeys(newSystemDebounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		debounceId := ulid.MustParse(debounceIds[0])

		var di DebounceItem
		err = json.Unmarshal([]byte(newSystemCluster.HGet(newSystemDebounceClient.KeyGenerator().Debounce(ctx), debounceIds[0])), &di)
		require.NoError(t, err)

		require.NotEmpty(t, debounceIds[0])
		require.Equal(t, expectedDi, di)

		// Queue state should match
		{
			queueItemIds, err := newSystemCluster.HKeys(newSystemShard.RedisClient.KeyGenerator().QueueItem())
			require.NoError(t, err)
			require.Len(t, queueItemIds, 1)

			var qi queue.QueueItem
			err = json.Unmarshal([]byte(newSystemCluster.HGet(newSystemShard.RedisClient.KeyGenerator().QueueItem(), queueItemIds[0])), &qi)
			require.NoError(t, err)

			require.Equal(t, queue.KindDebounce, qi.Data.Kind)

			expectedPayload := di.QueuePayload()
			expectedPayload.DebounceID = debounceId

			rawPayload := qi.Data.Payload.(json.RawMessage)

			var payload DebouncePayload
			err = json.Unmarshal(rawPayload, &payload)
			require.NoError(t, err)

			require.Equal(t, expectedPayload, payload)

			itemScore, err := newSystemCluster.ZScore(newSystemShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, functionId.String(), ""), qi.ID)
			require.NoError(t, err)

			initialScore := evt0Time.
				Add(10 * time.Second). // Debounce period
				Add(50 * time.Millisecond). // Buffer
				Add(time.Second).UnixMilli() // Allow updateDebounce on TTL 0
			expectedRequeueScore := eventTime.
				Add(10 * time.Second). // Debounce period
				Add(50 * time.Millisecond). // Buffer
				Add(time.Second).UnixMilli() // Allow updateDebounce on TTL 0

			require.NotEqual(t, initialScore, expectedRequeueScore)
			require.Equal(t, expectedRequeueScore, int64(itemScore))
		}
	})

	//t.Run("start debounce should work", func(t *testing.T) {
	//	debounceIds, err := unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().Debounce(ctx))
	//	require.NoError(t, err)
	//	require.Len(t, debounceIds, 1)
	//
	//	debounceId := ulid.MustParse(debounceIds[0])
	//
	//	val, err := unshardedCluster.Get(unshardedDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
	//	require.NoError(t, err)
	//	require.Equal(t, debounceId.String(), val)
	//
	//	di, err := oldRedisDebouncer.GetDebounceItem(ctx, debounceId)
	//	require.NoError(t, err)
	//
	//	err = oldRedisDebouncer.StartExecution(ctx, *di, fn, debounceId)
	//	require.NoError(t, err)
	//
	//	val, err = unshardedCluster.Get(unshardedDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
	//	require.NoError(t, err)
	//	require.NotEmpty(t, debounceId.String(), val)
	//})
	//
	//t.Run("delete debounce should work", func(t *testing.T) {
	//	debounceIds, err := unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().Debounce(ctx))
	//	require.NoError(t, err)
	//	require.Len(t, debounceIds, 1)
	//
	//	debounceId := ulid.MustParse(debounceIds[0])
	//
	//	err = oldRedisDebouncer.DeleteDebounceItem(ctx, debounceId)
	//	require.NoError(t, err)
	//
	//	debounceIds, err = unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().Debounce(ctx))
	//	require.Error(t, err)
	//	require.ErrorContains(t, err, "no such key")
	//})
}
