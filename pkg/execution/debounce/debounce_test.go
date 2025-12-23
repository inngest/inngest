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

	defaultQueueShard := redis_state.RedisQueueShard{Name: consts.DefaultQueueShardName, RedisClient: unshardedClient.Queue(), Kind: string(enums.QueueShardKindRedis)}

	q := redis_state.NewQueue(
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
			queue.KindDebounce: queue.KindDebounce,
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

			itemScore, err := unshardedCluster.ZScore(defaultQueueShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
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

			itemScore, err := unshardedCluster.ZScore(defaultQueueShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
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

// TestJITDebounceMigration verifies the JIT migration flow for debounces works.
func TestJITDebounceMigration(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)
	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	unshardedDebounceClient := unshardedClient.Debounce()

	defaultQueueShard := redis_state.RedisQueueShard{Name: consts.DefaultQueueShardName, RedisClient: unshardedClient.Queue(), Kind: string(enums.QueueShardKindRedis)}

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.RedisQueueShard{Name: "new-system", RedisClient: newSystemClusterClient.Queue(), Kind: string(enums.QueueShardKindRedis)}
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	// TODO What happens if both old and new services are running? Does this break debounces?
	//  Do we need to keep the old behavior and flip using a feature flag (LaunchDarkly)
	//  once all services running `Schedule` (new-runs, executor) are rolled out?
	oldQueue := redis_state.NewQueue(
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
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	newQueue := redis_state.NewQueue(
		newSystemShard, // Primary
		redis_state.WithQueueShardClients(
			map[string]redis_state.RedisQueueShard{
				defaultQueueShard.Name: defaultQueueShard,
				newSystemShard.Name:    newSystemShard,
			},
		),
		redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.RedisQueueShard, error) {
			// Enqueue new system queue items to new system queue shard
			if queueName != nil {
				return newSystemShard, nil
			}

			return defaultQueueShard, nil
		}),
		redis_state.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	fakeClock := clockwork.NewFakeClock()

	oldRedisDebouncer := NewRedisDebouncer(unshardedDebounceClient, defaultQueueShard, oldQueue).(debouncer)
	oldRedisDebouncer.c = fakeClock

	deb, err := NewRedisDebouncerWithMigration(DebouncerOpts{
		PrimaryDebounceClient: newSystemDebounceClient,
		PrimaryQueue:          newQueue,
		PrimaryQueueShard:     newSystemShard,

		SecondaryDebounceClient: unshardedDebounceClient,
		SecondaryQueue:          oldQueue,
		SecondaryQueueShard:     defaultQueueShard,

		ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
	})
	require.NoError(t, err)
	newRedisDebouncer := deb.(debouncer)
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

		err := oldRedisDebouncer.Debounce(ctx, expectedDi, fn)
		require.NoError(t, err)

		expectedDi.Timeout = eventTime.Add(60 * time.Second).UnixMilli()

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

			itemScore, err := unshardedCluster.ZScore(defaultQueueShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
			require.NoError(t, err)
			expectedQueueScore := eventTime.
				Add(10 * time.Second).       // Debounce period
				Add(buffer).                 // Buffer
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
		ttl := unshardedCluster.TTL(unshardedDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.Equal(t, 5*time.Second, ttl, "expected ttl to match", unshardedCluster.Keys())

		err := newRedisDebouncer.Debounce(ctx, expectedDi, fn)
		require.NoError(t, err)

		expectedDi.Timeout = evt0Time.Add(60 * time.Second).UnixMilli() // Must match initial event, timeout may never change

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

			itemScore, err := newSystemCluster.ZScore(newSystemShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
			require.NoError(t, err)

			initialScore := evt0Time.
				Add(10 * time.Second). // Debounce period
				Add(buffer).
				Add(time.Second).UnixMilli() // Allow updateDebounce on TTL 0
			expectedRequeueScore := eventTime.
				Add(10 * time.Second).       // Debounce period
				Add(buffer).                 // Buffer
				Add(time.Second).UnixMilli() // Allow updateDebounce on TTL 0

			require.NotEqual(t, initialScore, expectedRequeueScore)
			require.Equal(t, expectedRequeueScore, int64(itemScore))

			// Item should be removed from previous cluster
			require.Empty(t, unshardedCluster.HGet(unshardedDebounceClient.KeyGenerator().Debounce(ctx), debounceIds[0]))

			// Queue should be cleaned up
			_, err = unshardedCluster.HKeys(unshardedClient.Queue().KeyGenerator().QueueItem())
			require.Error(t, err)
			require.ErrorContains(t, err, "no such key")

			// Queue should be cleaned up
			_, err = unshardedCluster.ZMembers(unshardedClient.Queue().KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""))
			require.Error(t, err, "expected no key for the debounce partition", unshardedCluster.Keys())
			require.ErrorContains(t, err, "no such key")
		}
	})
}

// TestDebounceMigrationWithoutTimeout verifies the JIT migration flow works when no timeout is provided
func TestDebounceMigrationWithoutTimeout(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)
	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	unshardedDebounceClient := unshardedClient.Debounce()

	defaultQueueShard := redis_state.RedisQueueShard{Name: consts.DefaultQueueShardName, RedisClient: unshardedClient.Queue(), Kind: string(enums.QueueShardKindRedis)}

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.RedisQueueShard{Name: "new-system", RedisClient: newSystemClusterClient.Queue(), Kind: string(enums.QueueShardKindRedis)}
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	// TODO What happens if both old and new services are running? Does this break debounces?
	//  Do we need to keep the old behavior and flip using a feature flag (LaunchDarkly)
	//  once all services running `Schedule` (new-runs, executor) are rolled out?
	oldQueue := redis_state.NewQueue(
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
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	newQueue := redis_state.NewQueue(
		newSystemShard, // Primary
		redis_state.WithQueueShardClients(
			map[string]redis_state.RedisQueueShard{
				defaultQueueShard.Name: defaultQueueShard,
				newSystemShard.Name:    newSystemShard,
			},
		),
		redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.RedisQueueShard, error) {
			// Enqueue new system queue items to new system queue shard
			if queueName != nil {
				return newSystemShard, nil
			}

			return defaultQueueShard, nil
		}),
		redis_state.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	fakeClock := clockwork.NewFakeClock()

	oldRedisDebouncer := NewRedisDebouncer(unshardedDebounceClient, defaultQueueShard, oldQueue).(debouncer)
	oldRedisDebouncer.c = fakeClock

	deb, err := NewRedisDebouncerWithMigration(DebouncerOpts{
		PrimaryDebounceClient: newSystemDebounceClient,
		PrimaryQueue:          newQueue,
		PrimaryQueueShard:     newSystemShard,

		SecondaryDebounceClient: unshardedDebounceClient,
		SecondaryQueue:          oldQueue,
		SecondaryQueueShard:     defaultQueueShard,

		ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
	})
	require.NoError(t, err)
	newRedisDebouncer := deb.(debouncer)
	newRedisDebouncer.c = fakeClock

	ctx := context.Background()
	accountId, workspaceId, appId, functionId := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ID: functionId,
		Debounce: &inngest.Debounce{
			Period: "10s",
		},
	}

	evt0Time := fakeClock.Now()

	t.Run("create debounce on old queue should work", func(t *testing.T) {
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

			itemScore, err := unshardedCluster.ZScore(defaultQueueShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
			require.NoError(t, err)
			expectedQueueScore := eventTime.
				Add(10 * time.Second).       // Debounce period
				Add(buffer).                 // Buffer
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

			itemScore, err := newSystemCluster.ZScore(newSystemShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
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

			// Item should be removed from previous cluster
			require.Empty(t, unshardedCluster.HGet(newSystemDebounceClient.KeyGenerator().Debounce(ctx), debounceIds[0]))
		}
	})
}

// TestDebounceTimeoutIsPreserved verifies the initial debounce timeout is preserved after a JIT migration.
func TestDebounceTimeoutIsPreserved(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)
	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	unshardedDebounceClient := unshardedClient.Debounce()

	defaultQueueShard := redis_state.RedisQueueShard{Name: consts.DefaultQueueShardName, RedisClient: unshardedClient.Queue(), Kind: string(enums.QueueShardKindRedis)}

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.RedisQueueShard{Name: "new-system", RedisClient: newSystemClusterClient.Queue(), Kind: string(enums.QueueShardKindRedis)}
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	oldQueue := redis_state.NewQueue(
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
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	newQueue := redis_state.NewQueue(
		newSystemShard, // Primary
		redis_state.WithQueueShardClients(
			map[string]redis_state.RedisQueueShard{
				defaultQueueShard.Name: defaultQueueShard,
				newSystemShard.Name:    newSystemShard,
			},
		),
		redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.RedisQueueShard, error) {
			// Enqueue new system queue items to new system queue shard
			if queueName != nil {
				return newSystemShard, nil
			}

			return defaultQueueShard, nil
		}),
		redis_state.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	fakeClock := clockwork.NewFakeClock()

	oldRedisDebouncer := NewRedisDebouncer(unshardedDebounceClient, defaultQueueShard, oldQueue).(debouncer)
	oldRedisDebouncer.c = fakeClock

	deb, err := NewRedisDebouncerWithMigration(DebouncerOpts{
		PrimaryDebounceClient: newSystemDebounceClient,
		PrimaryQueue:          newQueue,
		PrimaryQueueShard:     newSystemShard,

		SecondaryDebounceClient: unshardedDebounceClient,
		SecondaryQueue:          oldQueue,
		SecondaryQueueShard:     defaultQueueShard,

		ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
	})
	require.NoError(t, err)
	newRedisDebouncer := deb.(debouncer)
	newRedisDebouncer.c = fakeClock

	ctx := context.Background()
	accountId, workspaceId, appId, functionId := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ID: functionId,
		Debounce: &inngest.Debounce{
			Period:  "4s",
			Timeout: util.StrPtr("6s"),
		},
	}

	evt0Time := fakeClock.Now()

	t.Run("create debounce on old queue should work", func(t *testing.T) {
		eventTime := evt0Time

		eventId := ulid.MustNew(ulid.Timestamp(eventTime), rand.Reader)

		err := oldRedisDebouncer.Debounce(ctx, DebounceItem{
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
		}, fn)
		require.NoError(t, err)

		debounceIds, err := unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		var di DebounceItem
		err = json.Unmarshal([]byte(unshardedCluster.HGet(unshardedDebounceClient.KeyGenerator().Debounce(ctx), debounceIds[0])), &di)
		require.NoError(t, err)

		require.NotEmpty(t, debounceIds[0])
		require.Equal(t, evt0Time.Add(6*time.Second).UnixMilli(), di.Timeout)

		// Full 4s of ttl
		ttl := unshardedCluster.TTL(unshardedDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.Equal(t, 4*time.Second, ttl, "expected ttl to match", unshardedCluster.Keys())
	})

	t.Run("update and migrate debounce should work", func(t *testing.T) {
		unshardedCluster.FastForward(3 * time.Second)
		fakeClock.Advance(3 * time.Second)

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

		// Time has passed, so TTL was decreased (4s-3s = 1s)
		ttl := unshardedCluster.TTL(unshardedDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.Equal(t, 1*time.Second, ttl, "expected ttl to match", unshardedCluster.Keys())

		err := newRedisDebouncer.Debounce(ctx, expectedDi, fn)
		require.NoError(t, err)

		// TTL on new cluster must be adjusted (6s to timeout, 3s already passed, renew by 4s is greater so we set an upper bound to the 6-3=3s remaining seconds)
		ttl = newSystemCluster.TTL(newSystemDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.Equal(t, 3*time.Second, ttl, "expected ttl to match", newSystemCluster.Keys())

		debounceIds, err := newSystemCluster.HKeys(newSystemDebounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		debounceId := ulid.MustParse(debounceIds[0])

		var di DebounceItem
		err = json.Unmarshal([]byte(newSystemCluster.HGet(newSystemDebounceClient.KeyGenerator().Debounce(ctx), debounceIds[0])), &di)
		require.NoError(t, err)

		// Timeout should still be original value
		expectedDi.Timeout = evt0Time.Add(6 * time.Second).UnixMilli()

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

			itemScore, err := newSystemCluster.ZScore(newSystemShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
			require.NoError(t, err)

			expectedRequeueScore := eventTime.
				Add(3 * time.Second).        // Remaining TTL applied
				Add(buffer).                 // Buffer
				Add(time.Second).UnixMilli() // Allow updateDebounce on TTL 0

			require.Equal(t, expectedRequeueScore, int64(itemScore))

			// Item should be removed from previous cluster
			require.Empty(t, unshardedCluster.HGet(newSystemDebounceClient.KeyGenerator().Debounce(ctx), debounceIds[0]))
		}
	})
}

// TestDebounceExplicitMigration verifies the debounce migration flow with Migrate().
func TestDebounceExplicitMigration(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)
	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	unshardedDebounceClient := unshardedClient.Debounce()

	defaultQueueShard := redis_state.RedisQueueShard{Name: consts.DefaultQueueShardName, RedisClient: unshardedClient.Queue(), Kind: string(enums.QueueShardKindRedis)}

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.RedisQueueShard{Name: "new-system", RedisClient: newSystemClusterClient.Queue(), Kind: string(enums.QueueShardKindRedis)}
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	oldQueue := redis_state.NewQueue(
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
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	newQueue := redis_state.NewQueue(
		newSystemShard, // Primary
		redis_state.WithQueueShardClients(
			map[string]redis_state.RedisQueueShard{
				defaultQueueShard.Name: defaultQueueShard,
				newSystemShard.Name:    newSystemShard,
			},
		),
		redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.RedisQueueShard, error) {
			// Enqueue new system queue items to new system queue shard
			if queueName != nil {
				return newSystemShard, nil
			}

			return defaultQueueShard, nil
		}),
		redis_state.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	fakeClock := clockwork.NewFakeClock()

	oldRedisDebouncer := NewRedisDebouncer(unshardedDebounceClient, defaultQueueShard, oldQueue).(debouncer)
	oldRedisDebouncer.c = fakeClock

	deb, err := NewRedisDebouncerWithMigration(DebouncerOpts{
		PrimaryDebounceClient: newSystemDebounceClient,
		PrimaryQueue:          newQueue,
		PrimaryQueueShard:     newSystemShard,

		SecondaryDebounceClient: unshardedDebounceClient,
		SecondaryQueue:          oldQueue,
		SecondaryQueueShard:     defaultQueueShard,

		ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
	})
	require.NoError(t, err)
	newRedisDebouncer := deb.(debouncer)
	newRedisDebouncer.c = fakeClock

	ctx := context.Background()
	accountId, workspaceId, appId, functionId := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ID: functionId,
		Debounce: &inngest.Debounce{
			Period: "10s",
		},
	}

	evt0Time := fakeClock.Now()

	t.Run("create debounce on old queue should work", func(t *testing.T) {
		eventTime := evt0Time

		eventId := ulid.MustNew(ulid.Timestamp(eventTime), rand.Reader)

		err := oldRedisDebouncer.Debounce(ctx, DebounceItem{
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
		}, fn)
		require.NoError(t, err)

		debounceIds, err := unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		var di DebounceItem
		err = json.Unmarshal([]byte(unshardedCluster.HGet(unshardedDebounceClient.KeyGenerator().Debounce(ctx), debounceIds[0])), &di)
		require.NoError(t, err)
	})

	t.Run("update and migrate debounce should work", func(t *testing.T) {
		debounceIds, err := unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		debounceId := ulid.MustParse(debounceIds[0])

		var di DebounceItem
		err = json.Unmarshal([]byte(unshardedCluster.HGet(unshardedDebounceClient.KeyGenerator().Debounce(ctx), debounceIds[0])), &di)
		require.NoError(t, err)

		unshardedCluster.FastForward(5 * time.Second)
		fakeClock.Advance(5 * time.Second)

		eventTime := fakeClock.Now()

		// Time has passed, so TTL was decreased (10s-5s = 5s)
		ttl := unshardedCluster.TTL(unshardedDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.Equal(t, 5*time.Second, ttl, "expected ttl to match", unshardedCluster.Keys())

		err = newRedisDebouncer.Migrate(ctx, debounceId, di, 5*time.Second, fn)
		require.NoError(t, err)

		// TTL on new cluster must be kept (no timeout _but_ we already used up 5s of the 10s timeout)
		ttl = newSystemCluster.TTL(newSystemDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.Equal(t, 5*time.Second, ttl, "expected ttl to match", newSystemCluster.Keys())

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

			itemScore, err := newSystemCluster.ZScore(newSystemShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
			require.NoError(t, err)

			expectedRequeueScore := eventTime.
				Add(5 * time.Second).        // Remaining TTL applied
				Add(buffer).                 // Buffer
				Add(time.Second).UnixMilli() // Allow updateDebounce on TTL 0

			require.Equal(t, expectedRequeueScore, int64(itemScore))

			// Item should be removed from previous cluster
			require.Empty(t, unshardedCluster.HGet(newSystemDebounceClient.KeyGenerator().Debounce(ctx), debounceIds[0]))
		}
	})
}

func TestDebouncePrimaryChooser(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)
	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	unshardedDebounceClient := unshardedClient.Debounce()

	defaultQueueShard := redis_state.RedisQueueShard{Name: consts.DefaultQueueShardName, RedisClient: unshardedClient.Queue(), Kind: string(enums.QueueShardKindRedis)}

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.RedisQueueShard{Name: "new-system", RedisClient: newSystemClusterClient.Queue(), Kind: string(enums.QueueShardKindRedis)}
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	oldQueue := redis_state.NewQueue(
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
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	newQueue := redis_state.NewQueue(
		newSystemShard, // Primary
		redis_state.WithQueueShardClients(
			map[string]redis_state.RedisQueueShard{
				defaultQueueShard.Name: defaultQueueShard,
				newSystemShard.Name:    newSystemShard,
			},
		),
		redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.RedisQueueShard, error) {
			// Enqueue new system queue items to new system queue shard
			if queueName != nil {
				return newSystemShard, nil
			}

			return defaultQueueShard, nil
		}),
		redis_state.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	fakeClock := clockwork.NewFakeClock()

	oldRedisDebouncer := NewRedisDebouncer(unshardedDebounceClient, defaultQueueShard, oldQueue).(debouncer)
	oldRedisDebouncer.c = fakeClock

	// Initial state: Only one primary configured, feature flag off.
	t.Run("before two clusters are configured for migration, use primary", func(t *testing.T) {
		deb, err := NewRedisDebouncerWithMigration(DebouncerOpts{
			PrimaryDebounceClient: newSystemDebounceClient,
			PrimaryQueue:          newQueue,
			PrimaryQueueShard:     newSystemShard,

			ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
				return true
			},
		})
		require.NoError(t, err)
		newRedisDebouncer := deb.(debouncer)
		newRedisDebouncer.c = fakeClock

		// If only a single cluster is configured as primary, use that. This is the target state.
		require.True(t, newRedisDebouncer.usePrimary(false))
	})

	// Preparation for migration: Switch primary -> secondary and add new primary.
	t.Run("when two clusters are configured, use secondary", func(t *testing.T) {
		deb, err := NewRedisDebouncerWithMigration(DebouncerOpts{
			PrimaryDebounceClient: newSystemDebounceClient,
			PrimaryQueue:          newQueue,
			PrimaryQueueShard:     newSystemShard,

			SecondaryDebounceClient: unshardedDebounceClient,
			SecondaryQueue:          oldQueue,
			SecondaryQueueShard:     defaultQueueShard,

			ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
				return true
			},
		})
		require.NoError(t, err)
		newRedisDebouncer := deb.(debouncer)
		newRedisDebouncer.c = fakeClock

		// When the feature flag is not yet enabled, keep using the (previous/old) secondary cluster.
		// This is important because we'd otherwise lose existing debounces and create inconsistencies.
		require.False(t, newRedisDebouncer.usePrimary(false))
	})

	// In a real migration: Wait until all clusters are safely deployed before flipping feature toggle.
	// Also set feature toggle to a future date to ensure the feature flag is loaded into memory in time,
	// to prevent clock drift. Alternatively, hard code feature flag switch timestamp (as we did with the function
	// run state sharding rollout). Or use an atomic value in Redis.

	// Start migration: Enable feature flag. This test assumes the change propagation is immediate and is registered by
	// all consumers at once.
	t.Run("during migration, use primary", func(t *testing.T) {
		deb, err := NewRedisDebouncerWithMigration(DebouncerOpts{
			PrimaryDebounceClient: newSystemDebounceClient,
			PrimaryQueue:          newQueue,
			PrimaryQueueShard:     newSystemShard,

			SecondaryDebounceClient: unshardedDebounceClient,
			SecondaryQueue:          oldQueue,
			SecondaryQueueShard:     defaultQueueShard,

			ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
				return true
			},
		})
		require.NoError(t, err)
		newRedisDebouncer := deb.(debouncer)
		newRedisDebouncer.c = fakeClock

		// When the feature flag is flipped, start using the primary. Also migrate existing entries.
		require.True(t, newRedisDebouncer.usePrimary(true))
	})

	// In a real migration: Wait until all debounces are migrated. Manually move leftover debounces.

	// Once all debounces are moved from the old shard, we can remove the reference (by dropping the secondary).
	// To prevent old deployments from using the old cluster again, we must keep the feature flag enabled during this time.
	t.Run("after removing secondary once migration is completed, use primary", func(t *testing.T) {
		deb, err := NewRedisDebouncerWithMigration(DebouncerOpts{
			PrimaryDebounceClient: newSystemDebounceClient,
			PrimaryQueue:          newQueue,
			PrimaryQueueShard:     newSystemShard,

			ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
				return true
			},
		})
		require.NoError(t, err)
		newRedisDebouncer := deb.(debouncer)
		newRedisDebouncer.c = fakeClock

		// This is similar to the first test case, but the feature flag is still enabled because there
		// may be containers still running the old code which has both clusters configured and relies on the flag
		// for choosing the primary. If we reset the flag too early, we would use the secondary.
		require.True(t, newRedisDebouncer.usePrimary(true))
	})

	// In a real migration: Wait for rollout to finish so that the secondary cluster is not referenced in any
	// deployment anymore. Then toggle the feature flag off again. After that, we'll wrap around to the first test case.
}

func TestDebounceExecutionDuringMigrationWorks(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)
	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	unshardedDebounceClient := unshardedClient.Debounce()

	defaultQueueShard := redis_state.RedisQueueShard{Name: consts.DefaultQueueShardName, RedisClient: unshardedClient.Queue(), Kind: string(enums.QueueShardKindRedis)}

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.RedisQueueShard{Name: "new-system", RedisClient: newSystemClusterClient.Queue(), Kind: string(enums.QueueShardKindRedis)}
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	// TODO What happens if both old and new services are running? Does this break debounces?
	//  Do we need to keep the old behavior and flip using a feature flag (LaunchDarkly)
	//  once all services running `Schedule` (new-runs, executor) are rolled out?
	oldQueue := redis_state.NewQueue(
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
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	newQueue := redis_state.NewQueue(
		newSystemShard, // Primary
		redis_state.WithQueueShardClients(
			map[string]redis_state.RedisQueueShard{
				defaultQueueShard.Name: defaultQueueShard,
				newSystemShard.Name:    newSystemShard,
			},
		),
		redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.RedisQueueShard, error) {
			// Enqueue new system queue items to new system queue shard
			if queueName != nil {
				return newSystemShard, nil
			}

			return defaultQueueShard, nil
		}),
		redis_state.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	fakeClock := clockwork.NewFakeClock()

	oldRedisDebouncer := NewRedisDebouncer(unshardedDebounceClient, defaultQueueShard, oldQueue).(debouncer)
	oldRedisDebouncer.c = fakeClock

	newRedisDebouncer, err := NewRedisDebouncerWithMigration(DebouncerOpts{
		PrimaryDebounceClient: newSystemDebounceClient,
		PrimaryQueue:          newQueue,
		PrimaryQueueShard:     newSystemShard,

		SecondaryDebounceClient: unshardedDebounceClient,
		SecondaryQueue:          oldQueue,
		SecondaryQueueShard:     defaultQueueShard,

		// Always migrate
		ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
		Clock: fakeClock,
	})
	require.NoError(t, err)

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

		err := oldRedisDebouncer.Debounce(ctx, expectedDi, fn)
		require.NoError(t, err)

		expectedDi.Timeout = eventTime.Add(60 * time.Second).UnixMilli()

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

			itemScore, err := unshardedCluster.ZScore(defaultQueueShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
			require.NoError(t, err)
			expectedQueueScore := eventTime.
				Add(10 * time.Second).       // Debounce period
				Add(buffer).                 // Buffer
				Add(time.Second).UnixMilli() // Allow updateDebounce on TTL 0
			require.Equal(t, expectedQueueScore, int64(itemScore))
		}
	})

	t.Run("retrieve and execute debounce from secondary cluster should work", func(t *testing.T) {
		debounceIds, err := unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		debounceId := ulid.MustParse(debounceIds[0])

		di, err := newRedisDebouncer.GetDebounceItem(ctx, debounceId, accountId)
		require.NoError(t, err)

		// Must retrieve from secondary cluster
		require.True(t, di.isSecondary)

		err = newRedisDebouncer.StartExecution(ctx, *di, fn, debounceId)
		require.NoError(t, err)

		// Debounce pointer should be set to new value
		pointerVal, err := unshardedCluster.Get(unshardedDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.NoError(t, err)
		require.NotEmpty(t, pointerVal)
		require.NotEqual(t, debounceId.String(), pointerVal)

		// Debounce should still exist
		require.NotEmpty(t, unshardedCluster.HGet(unshardedDebounceClient.KeyGenerator().Debounce(ctx), debounceId.String()))

		// New cluster should not have pointer set
		require.False(t, newSystemCluster.Exists(newSystemDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String())))

		// A racing prepareMigration call must not be able to find the debounce
		deb := newRedisDebouncer.(debouncer)
		existingID, _, err := deb.prepareMigration(ctx, *di, fn)
		require.NoError(t, err)
		require.Nil(t, existingID)

		err = newRedisDebouncer.DeleteDebounceItem(ctx, debounceId, *di, accountId)
		require.NoError(t, err)

		// Debounce should be dropped
		require.Empty(t, unshardedCluster.HGet(unshardedDebounceClient.KeyGenerator().Debounce(ctx), debounceId.String()))
	})
}

func TestDebounceExecutionShouldNotRaceMigration(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)
	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	unshardedDebounceClient := unshardedClient.Debounce()

	defaultQueueShard := redis_state.RedisQueueShard{Name: consts.DefaultQueueShardName, RedisClient: unshardedClient.Queue(), Kind: string(enums.QueueShardKindRedis)}

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.RedisQueueShard{Name: "new-system", RedisClient: newSystemClusterClient.Queue(), Kind: string(enums.QueueShardKindRedis)}
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	// TODO What happens if both old and new services are running? Does this break debounces?
	//  Do we need to keep the old behavior and flip using a feature flag (LaunchDarkly)
	//  once all services running `Schedule` (new-runs, executor) are rolled out?
	oldQueue := redis_state.NewQueue(
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
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	newQueue := redis_state.NewQueue(
		newSystemShard, // Primary
		redis_state.WithQueueShardClients(
			map[string]redis_state.RedisQueueShard{
				defaultQueueShard.Name: defaultQueueShard,
				newSystemShard.Name:    newSystemShard,
			},
		),
		redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.RedisQueueShard, error) {
			// Enqueue new system queue items to new system queue shard
			if queueName != nil {
				return newSystemShard, nil
			}

			return defaultQueueShard, nil
		}),
		redis_state.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	)

	fakeClock := clockwork.NewFakeClock()

	oldRedisDebouncer := NewRedisDebouncer(unshardedDebounceClient, defaultQueueShard, oldQueue).(debouncer)
	oldRedisDebouncer.c = fakeClock

	newRedisDebouncer, err := NewRedisDebouncerWithMigration(DebouncerOpts{
		PrimaryDebounceClient: newSystemDebounceClient,
		PrimaryQueue:          newQueue,
		PrimaryQueueShard:     newSystemShard,

		SecondaryDebounceClient: unshardedDebounceClient,
		SecondaryQueue:          oldQueue,
		SecondaryQueueShard:     defaultQueueShard,

		// Always migrate
		ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
		Clock: fakeClock,
	})
	require.NoError(t, err)

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

		err := oldRedisDebouncer.Debounce(ctx, expectedDi, fn)
		require.NoError(t, err)

		expectedDi.Timeout = eventTime.Add(60 * time.Second).UnixMilli()

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

			itemScore, err := unshardedCluster.ZScore(defaultQueueShard.RedisClient.KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
			require.NoError(t, err)
			expectedQueueScore := eventTime.
				Add(10 * time.Second).       // Debounce period
				Add(buffer).                 // Buffer
				Add(time.Second).UnixMilli() // Allow updateDebounce on TTL 0
			require.Equal(t, expectedQueueScore, int64(itemScore))
		}
	})

	t.Run("execution should not race migration", func(t *testing.T) {
		debounceIds, err := unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		debounceId := ulid.MustParse(debounceIds[0])

		di, err := newRedisDebouncer.GetDebounceItem(ctx, debounceId, accountId)
		require.NoError(t, err)

		// Must retrieve from secondary cluster
		require.True(t, di.isSecondary)

		// If prepareMigration is called first, it must lock the execution from running the debounce item
		deb := newRedisDebouncer.(debouncer)
		existingID, _, err := deb.prepareMigration(ctx, *di, fn)
		require.NoError(t, err)
		require.NotNil(t, existingID)
		require.Equal(t, debounceId, *existingID)

		// Lock must be set
		hkeys, err := unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().DebounceMigrating(ctx))
		require.NoError(t, err)
		require.Contains(t, hkeys, debounceId.String())
		require.Len(t, hkeys, 1)

		// Debounce pointer should be set to new value
		pointerVal, err := unshardedCluster.Get(unshardedDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String()))
		require.NoError(t, err)
		require.NotEmpty(t, pointerVal)
		require.NotEqual(t, debounceId.String(), pointerVal)

		// Debounce should still exist
		require.NotEmpty(t, unshardedCluster.HGet(unshardedDebounceClient.KeyGenerator().Debounce(ctx), debounceId.String()))

		// New cluster should not have pointer set
		require.False(t, newSystemCluster.Exists(newSystemDebounceClient.KeyGenerator().DebouncePointer(ctx, functionId, functionId.String())))

		// Starting the debounce timeout item as it is being migrated must return ErrDebounceMigrating
		err = newRedisDebouncer.StartExecution(ctx, *di, fn, debounceId)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrDebounceMigrating)

		err = deb.deleteMigratingFlag(ctx, debounceId, unshardedDebounceClient)
		require.NoError(t, err)

		// Lock must be gone
		require.False(t, unshardedCluster.Exists(unshardedDebounceClient.KeyGenerator().DebounceMigrating(ctx)))
	})
}
