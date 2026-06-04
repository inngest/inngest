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

// singleShardEnv is all components needed for a non-migration debounce test.
type singleShardEnv struct {
	cluster *miniredis.Miniredis
	client  *redis_state.UnshardedClient // for direct Redis inspection
	shard   queue.QueueShard
	deb     Debouncer
}

// newSingleShardEnv spins up an in-memory Redis shard and wires a Debouncer to
// it. Uses the real clock; for fake-clock tests build the environment manually.
func newSingleShardEnv(t *testing.T) singleShardEnv {
	t.Helper()
	cluster := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{cluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	client := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{queue.KindDebounce: queue.KindDebounce}),
	}
	shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, client.Queue(), opts...)
	reg, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := queue.New(context.Background(), "debounce-test", reg, opts...)
	require.NoError(t, err)
	deb, err := NewDebouncer(reg, shard.Name(), q)
	require.NoError(t, err)
	return singleShardEnv{cluster: cluster, client: client, shard: shard, deb: deb}
}

func testScope(accountID, workspaceID, functionID uuid.UUID) queue.Scope {
	return queue.Scope{AccountID: accountID, EnvID: workspaceID, FunctionID: functionID}
}

func migrationShardMap(defaultShard, newSystemShard queue.QueueShard) map[string]queue.QueueShard {
	return map[string]queue.QueueShard{
		consts.DefaultQueueShardName: defaultShard,
		newSystemShard.Name():        newSystemShard,
	}
}

// migrationShardSelector routes system queue items (queueName != nil) to the
// new system shard and everything else to the default shard.
func migrationShardSelector(defaultShard, newSystemShard queue.QueueShard) func(ctx context.Context, accountID uuid.UUID, queueName *string) (queue.QueueShard, error) {
	return func(_ context.Context, _ uuid.UUID, queueName *string) (queue.QueueShard, error) {
		if queueName != nil {
			return newSystemShard, nil
		}
		return defaultShard, nil
	}
}

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

	q, err := queue.New(context.Background(), "debounce-test", shardRegistry, opts...)
	require.NoError(t, err)
	kg := shard.Client().KeyGenerator()

	fakeClock := clockwork.NewFakeClock()

	deb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:           shardRegistry,
		PrimaryShardName: shard.Name(),
		Queue:            q,
		Clock:            fakeClock,
	})
	require.NoError(t, err)
	redisDebouncer := deb.(debouncer)

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

		di, err := redisDebouncer.GetDebounceItem(ctx, testScope(accountId, workspaceId, functionId), debounceId)
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

		err = redisDebouncer.DeleteDebounceItem(ctx, testScope(accountId, workspaceId, functionId), debounceId, di)
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

	fakeClock := clockwork.NewFakeClock()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
		queue.WithClock(fakeClock),
	}

	defaultQueueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.NewQueueShard("new-system", newSystemClusterClient.Queue(), opts...)
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	// TODO What happens if both old and new services are running? Does this break debounces?
	//  Do we need to keep the old behavior and flip using a feature flag (LaunchDarkly)
	//  once all services running `Schedule` (new-runs, executor) are rolled out?
	oldShardRegistry, err := queue.NewSingleShardRegistry(defaultQueueShard)
	require.NoError(t, err)
	oldQueue, err := queue.New(context.Background(), "old-queue", oldShardRegistry, opts...)
	require.NoError(t, err)

	newShardRegistry, err := queue.NewShardRegistry(
		migrationShardMap(defaultQueueShard, newSystemShard),
		queue.WithShardSelector(migrationShardSelector(defaultQueueShard, newSystemShard)),
		queue.WithPrimary(newSystemShard),
	)

	require.NoError(t, err)

	newQueue, err := queue.New(context.Background(), "new-queue", newShardRegistry, opts...)
	require.NoError(t, err)

	kg := defaultQueueShard.Client().KeyGenerator()

	oldDeb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:           oldShardRegistry,
		PrimaryShardName: defaultQueueShard.Name(),
		Queue:            oldQueue,
		Clock:            fakeClock,
	})
	require.NoError(t, err)
	oldRedisDebouncer := oldDeb.(debouncer)

	deb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:             newShardRegistry,
		PrimaryShardName:   newSystemShard.Name(),
		SecondaryShardName: defaultQueueShard.Name(),
		Queue:              newQueue,
		ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
		Clock: fakeClock,
	})
	require.NoError(t, err)
	newRedisDebouncer := deb.(debouncer)

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
		di.Event.ClearSize()
		require.Equal(t, expectedDi, di)

		// Queue state should match
		{
			queueItemIds, err := newSystemCluster.HKeys(kg.QueueItem())
			require.NoError(t, err)
			require.Len(t, queueItemIds, 1)

			var qi queue.QueueItem
			err = json.Unmarshal([]byte(newSystemCluster.HGet(kg.QueueItem(), queueItemIds[0])), &qi)
			require.NoError(t, err)

			require.Equal(t, queue.KindDebounce, qi.Data.Kind)

			expectedPayload := di.QueuePayload()
			expectedPayload.DebounceID = debounceId

			rawPayload := qi.Data.Payload.(json.RawMessage)

			var payload DebouncePayload
			err = json.Unmarshal(rawPayload, &payload)
			require.NoError(t, err)

			require.Equal(t, expectedPayload, payload)

			itemScore, err := newSystemCluster.ZScore(kg.PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
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

	fakeClock := clockwork.NewFakeClock()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
		queue.WithClock(fakeClock),
	}

	defaultQueueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.NewQueueShard("new-system", newSystemClusterClient.Queue(), opts...)
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	// TODO What happens if both old and new services are running? Does this break debounces?
	//  Do we need to keep the old behavior and flip using a feature flag (LaunchDarkly)
	//  once all services running `Schedule` (new-runs, executor) are rolled out?
	oldShardRegistry, err := queue.NewSingleShardRegistry(defaultQueueShard)
	require.NoError(t, err)
	oldQueue, err := queue.New(context.Background(), "old-queue", oldShardRegistry, opts...)
	require.NoError(t, err)

	newShardRegistry, err := queue.NewShardRegistry(
		migrationShardMap(defaultQueueShard, newSystemShard),
		queue.WithShardSelector(migrationShardSelector(defaultQueueShard, newSystemShard)),
		queue.WithPrimary(newSystemShard),
	)

	require.NoError(t, err)

	newQueue, err := queue.New(context.Background(), "new-queue", newShardRegistry, opts...)
	require.NoError(t, err)

	kg := defaultQueueShard.Client().KeyGenerator()
	oldDeb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:           oldShardRegistry,
		PrimaryShardName: defaultQueueShard.Name(),
		Queue:            oldQueue,
		Clock:            fakeClock,
	})
	require.NoError(t, err)
	oldRedisDebouncer := oldDeb.(debouncer)

	deb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:             newShardRegistry,
		PrimaryShardName:   newSystemShard.Name(),
		SecondaryShardName: defaultQueueShard.Name(),
		Queue:              newQueue,
		ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
		Clock: fakeClock,
	})
	require.NoError(t, err)
	newRedisDebouncer := deb.(debouncer)

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
		di.Event.ClearSize()
		require.Equal(t, expectedDi, di)

		// Queue state should match
		{
			queueItemIds, err := newSystemCluster.HKeys(kg.QueueItem())
			require.NoError(t, err)
			require.Len(t, queueItemIds, 1)

			var qi queue.QueueItem
			err = json.Unmarshal([]byte(newSystemCluster.HGet(kg.QueueItem(), queueItemIds[0])), &qi)
			require.NoError(t, err)

			require.Equal(t, queue.KindDebounce, qi.Data.Kind)

			expectedPayload := di.QueuePayload()
			expectedPayload.DebounceID = debounceId

			rawPayload := qi.Data.Payload.(json.RawMessage)

			var payload DebouncePayload
			err = json.Unmarshal(rawPayload, &payload)
			require.NoError(t, err)

			require.Equal(t, expectedPayload, payload)

			itemScore, err := newSystemCluster.ZScore(kg.PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
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

	fakeClock := clockwork.NewFakeClock()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
		queue.WithClock(fakeClock),
	}

	defaultQueueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.NewQueueShard("new-system", newSystemClusterClient.Queue(), opts...)
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	// TODO What happens if both old and new services are running? Does this break debounces?
	//  Do we need to keep the old behavior and flip using a feature flag (LaunchDarkly)
	//  once all services running `Schedule` (new-runs, executor) are rolled out?
	oldShardRegistry, err := queue.NewSingleShardRegistry(defaultQueueShard)
	require.NoError(t, err)
	oldQueue, err := queue.New(context.Background(), "old-queue", oldShardRegistry, opts...)
	require.NoError(t, err)

	newShardRegistry, err := queue.NewShardRegistry(
		migrationShardMap(defaultQueueShard, newSystemShard),
		queue.WithShardSelector(migrationShardSelector(defaultQueueShard, newSystemShard)),
		queue.WithPrimary(newSystemShard),
	)

	require.NoError(t, err)

	newQueue, err := queue.New(context.Background(), "new-queue", newShardRegistry, opts...)
	require.NoError(t, err)

	kg := defaultQueueShard.Client().KeyGenerator()

	oldDeb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:           oldShardRegistry,
		PrimaryShardName: defaultQueueShard.Name(),
		Queue:            oldQueue,
		Clock:            fakeClock,
	})
	require.NoError(t, err)
	oldRedisDebouncer := oldDeb.(debouncer)

	deb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:             newShardRegistry,
		PrimaryShardName:   newSystemShard.Name(),
		SecondaryShardName: defaultQueueShard.Name(),
		Queue:              newQueue,
		ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
		Clock: fakeClock,
	})
	require.NoError(t, err)
	newRedisDebouncer := deb.(debouncer)

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
		di.Event.ClearSize()
		require.Equal(t, expectedDi, di)

		// Queue state should match
		{
			queueItemIds, err := newSystemCluster.HKeys(kg.QueueItem())
			require.NoError(t, err)
			require.Len(t, queueItemIds, 1)

			var qi queue.QueueItem
			err = json.Unmarshal([]byte(newSystemCluster.HGet(kg.QueueItem(), queueItemIds[0])), &qi)
			require.NoError(t, err)

			require.Equal(t, queue.KindDebounce, qi.Data.Kind)

			expectedPayload := di.QueuePayload()
			expectedPayload.DebounceID = debounceId

			rawPayload := qi.Data.Payload.(json.RawMessage)

			var payload DebouncePayload
			err = json.Unmarshal(rawPayload, &payload)
			require.NoError(t, err)

			require.Equal(t, expectedPayload, payload)

			itemScore, err := newSystemCluster.ZScore(kg.PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
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

	fakeClock := clockwork.NewFakeClock()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
		queue.WithClock(fakeClock),
	}

	defaultQueueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.NewQueueShard("new-system", newSystemClusterClient.Queue(), opts...)
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	// TODO What happens if both old and new services are running? Does this break debounces?
	//  Do we need to keep the old behavior and flip using a feature flag (LaunchDarkly)
	//  once all services running `Schedule` (new-runs, executor) are rolled out?
	oldShardRegistry, err := queue.NewSingleShardRegistry(defaultQueueShard)
	require.NoError(t, err)
	oldQueue, err := queue.New(context.Background(), "old-queue", oldShardRegistry, opts...)
	require.NoError(t, err)

	newShardRegistry, err := queue.NewShardRegistry(
		migrationShardMap(defaultQueueShard, newSystemShard),
		queue.WithShardSelector(migrationShardSelector(defaultQueueShard, newSystemShard)),
		queue.WithPrimary(newSystemShard),
	)

	require.NoError(t, err)

	newQueue, err := queue.New(context.Background(), "new-queue", newShardRegistry, opts...)
	require.NoError(t, err)

	kg := defaultQueueShard.Client().KeyGenerator()

	oldDeb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:           oldShardRegistry,
		PrimaryShardName: defaultQueueShard.Name(),
		Queue:            oldQueue,
		Clock:            fakeClock,
	})
	require.NoError(t, err)
	oldRedisDebouncer := oldDeb.(debouncer)

	deb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:             newShardRegistry,
		PrimaryShardName:   newSystemShard.Name(),
		SecondaryShardName: defaultQueueShard.Name(),
		Queue:              newQueue,
		ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
		Clock: fakeClock,
	})
	require.NoError(t, err)
	newRedisDebouncer := deb.(debouncer)

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
			queueItemIds, err := newSystemCluster.HKeys(kg.QueueItem())
			require.NoError(t, err)
			require.Len(t, queueItemIds, 1)

			var qi queue.QueueItem
			err = json.Unmarshal([]byte(newSystemCluster.HGet(kg.QueueItem(), queueItemIds[0])), &qi)
			require.NoError(t, err)

			require.Equal(t, queue.KindDebounce, qi.Data.Kind)

			expectedPayload := di.QueuePayload()
			expectedPayload.DebounceID = debounceId

			rawPayload := qi.Data.Payload.(json.RawMessage)

			var payload DebouncePayload
			err = json.Unmarshal(rawPayload, &payload)
			require.NoError(t, err)

			require.Equal(t, expectedPayload, payload)

			itemScore, err := newSystemCluster.ZScore(kg.PartitionQueueSet(enums.PartitionTypeDefault, queue.KindDebounce, ""), qi.ID)
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

	fakeClock := clockwork.NewFakeClock()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
		queue.WithClock(fakeClock),
	}

	defaultQueueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.NewQueueShard("new-system", newSystemClusterClient.Queue(), opts...)

	// TODO What happens if both old and new services are running? Does this break debounces?
	//  Do we need to keep the old behavior and flip using a feature flag (LaunchDarkly)
	//  once all services running `Schedule` (new-runs, executor) are rolled out?
	oldShardRegistry, err := queue.NewSingleShardRegistry(defaultQueueShard)
	require.NoError(t, err)
	oldQueue, err := queue.New(context.Background(), "old-queue", oldShardRegistry, opts...)
	require.NoError(t, err)

	newShardRegistry, err := queue.NewShardRegistry(
		migrationShardMap(defaultQueueShard, newSystemShard),
		queue.WithShardSelector(migrationShardSelector(defaultQueueShard, newSystemShard)),
		queue.WithPrimary(newSystemShard),
	)

	require.NoError(t, err)

	newQueue, err := queue.New(context.Background(), "new-queue", newShardRegistry, opts...)
	require.NoError(t, err)

	_, err = NewDebouncerWithMigration(DebouncerOpts{
		Shards:           oldShardRegistry,
		PrimaryShardName: defaultQueueShard.Name(),
		Queue:            oldQueue,
		Clock:            fakeClock,
	})
	require.NoError(t, err)

	// Initial state: Only one primary configured, feature flag off.
	t.Run("before two clusters are configured for migration, use primary", func(t *testing.T) {
		deb, err := NewDebouncerWithMigration(DebouncerOpts{
			Shards:           newShardRegistry,
			PrimaryShardName: newSystemShard.Name(),
			Queue:            newQueue,
			ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
				return true
			},
			Clock: fakeClock,
		})
		require.NoError(t, err)
		newRedisDebouncer := deb.(debouncer)

		// If only a single cluster is configured as primary, use that. This is the target state.
		require.True(t, newRedisDebouncer.usePrimary(false))
	})

	// Preparation for migration: Switch primary -> secondary and add new primary.
	t.Run("when two clusters are configured, use secondary", func(t *testing.T) {
		deb, err := NewDebouncerWithMigration(DebouncerOpts{
			Shards:             newShardRegistry,
			PrimaryShardName:   newSystemShard.Name(),
			SecondaryShardName: defaultQueueShard.Name(),
			Queue:              newQueue,
			ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
				return true
			},
			Clock: fakeClock,
		})
		require.NoError(t, err)
		newRedisDebouncer := deb.(debouncer)

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
		deb, err := NewDebouncerWithMigration(DebouncerOpts{
			Shards:             newShardRegistry,
			PrimaryShardName:   newSystemShard.Name(),
			SecondaryShardName: defaultQueueShard.Name(),
			Queue:              newQueue,
			ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
				return true
			},
			Clock: fakeClock,
		})
		require.NoError(t, err)
		newRedisDebouncer := deb.(debouncer)

		// When the feature flag is flipped, start using the primary. Also migrate existing entries.
		require.True(t, newRedisDebouncer.usePrimary(true))
	})

	// In a real migration: Wait until all debounces are migrated. Manually move leftover debounces.

	// Once all debounces are moved from the old shard, we can remove the reference (by dropping the secondary).
	// To prevent old deployments from using the old cluster again, we must keep the feature flag enabled during this time.
	t.Run("after removing secondary once migration is completed, use primary", func(t *testing.T) {
		deb, err := NewDebouncerWithMigration(DebouncerOpts{
			Shards:           newShardRegistry,
			PrimaryShardName: newSystemShard.Name(),
			Queue:            newQueue,
			ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
				return true
			},
			Clock: fakeClock,
		})
		require.NoError(t, err)
		newRedisDebouncer := deb.(debouncer)

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

	fakeClock := clockwork.NewFakeClock()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
		queue.WithClock(fakeClock),
	}

	defaultQueueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.NewQueueShard("new-system", newSystemClusterClient.Queue(), opts...)
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	// TODO What happens if both old and new services are running? Does this break debounces?
	//  Do we need to keep the old behavior and flip using a feature flag (LaunchDarkly)
	//  once all services running `Schedule` (new-runs, executor) are rolled out?
	oldShardRegistry, err := queue.NewSingleShardRegistry(defaultQueueShard)
	require.NoError(t, err)
	oldQueue, err := queue.New(context.Background(), "old-queue", oldShardRegistry, opts...)
	require.NoError(t, err)

	newShardRegistry, err := queue.NewShardRegistry(
		migrationShardMap(defaultQueueShard, newSystemShard),
		queue.WithShardSelector(migrationShardSelector(defaultQueueShard, newSystemShard)),
		queue.WithPrimary(newSystemShard),
	)

	require.NoError(t, err)

	newQueue, err := queue.New(context.Background(), "new-queue", newShardRegistry, opts...)
	require.NoError(t, err)

	kg := defaultQueueShard.Client().KeyGenerator()

	oldDeb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:           oldShardRegistry,
		PrimaryShardName: defaultQueueShard.Name(),
		Queue:            oldQueue,
		Clock:            fakeClock,
	})
	require.NoError(t, err)
	oldRedisDebouncer := oldDeb.(debouncer)

	newDeb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:             newShardRegistry,
		PrimaryShardName:   newSystemShard.Name(),
		SecondaryShardName: defaultQueueShard.Name(),
		Queue:              newQueue,
		// Always migrate
		ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
		Clock: fakeClock,
	})
	require.NoError(t, err)
	newRedisDebouncer := newDeb.(debouncer)

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

	t.Run("retrieve and execute debounce from secondary cluster should work", func(t *testing.T) {
		debounceIds, err := unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		debounceId := ulid.MustParse(debounceIds[0])

		di, err := newRedisDebouncer.GetDebounceItem(ctx, testScope(accountId, workspaceId, functionId), debounceId)
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
		fakeID := ulid.MustNew(ulid.Now(), rand.Reader)
		existingID, _, err := defaultQueueShard.DebouncePrepareMigration(ctx, testScope(accountId, workspaceId, fn.ID), fn.ID.String(), fakeID)
		require.NoError(t, err)
		require.Nil(t, existingID)

		err = newRedisDebouncer.DeleteDebounceItem(ctx, testScope(accountId, workspaceId, functionId), debounceId, *di)
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

	fakeClock := clockwork.NewFakeClock()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
		queue.WithClock(fakeClock),
	}

	defaultQueueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)

	// Create new single-shard (but multi-replica) Valkey cluster for system queues + colocated debounce state
	newSystemCluster := miniredis.RunT(t)
	newSystemClusterRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	newSystemClusterClient := redis_state.NewUnshardedClient(newSystemClusterRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	require.NoError(t, err)
	newSystemShard := redis_state.NewQueueShard("new-system", newSystemClusterClient.Queue(), opts...)
	newSystemDebounceClient := newSystemClusterClient.Debounce()

	// TODO What happens if both old and new services are running? Does this break debounces?
	//  Do we need to keep the old behavior and flip using a feature flag (LaunchDarkly)
	//  once all services running `Schedule` (new-runs, executor) are rolled out?
	oldShardRegistry, err := queue.NewSingleShardRegistry(defaultQueueShard)
	require.NoError(t, err)
	oldQueue, err := queue.New(context.Background(), "old-queue", oldShardRegistry, opts...)
	require.NoError(t, err)

	newShardRegistry, err := queue.NewShardRegistry(
		migrationShardMap(defaultQueueShard, newSystemShard),
		queue.WithShardSelector(migrationShardSelector(defaultQueueShard, newSystemShard)),
		queue.WithPrimary(newSystemShard),
	)

	require.NoError(t, err)

	newQueue, err := queue.New(context.Background(), "new-queue", newShardRegistry, opts...)
	require.NoError(t, err)

	kg := defaultQueueShard.Client().KeyGenerator()

	oldDeb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:           oldShardRegistry,
		PrimaryShardName: defaultQueueShard.Name(),
		Queue:            oldQueue,
		Clock:            fakeClock,
	})
	require.NoError(t, err)
	oldRedisDebouncer := oldDeb.(debouncer)

	newDeb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:             newShardRegistry,
		PrimaryShardName:   newSystemShard.Name(),
		SecondaryShardName: defaultQueueShard.Name(),
		Queue:              newQueue,
		// Always migrate
		ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
		Clock: fakeClock,
	})
	require.NoError(t, err)
	newRedisDebouncer := newDeb.(debouncer)

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

	t.Run("execution should not race migration", func(t *testing.T) {
		debounceIds, err := unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().Debounce(ctx))
		require.NoError(t, err)
		require.Len(t, debounceIds, 1)

		debounceId := ulid.MustParse(debounceIds[0])

		di, err := newRedisDebouncer.GetDebounceItem(ctx, testScope(accountId, workspaceId, functionId), debounceId)
		require.NoError(t, err)

		// Must retrieve from secondary cluster
		require.True(t, di.isSecondary)

		// If prepareMigration is called first, it must lock the execution from running the debounce item
		fakeID := ulid.MustNew(ulid.Now(), rand.Reader)
		existingID, _, err := defaultQueueShard.DebouncePrepareMigration(ctx, testScope(accountId, workspaceId, fn.ID), fn.ID.String(), fakeID)
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

		err = defaultQueueShard.DebounceDeleteMigratingFlag(ctx, testScope(accountId, workspaceId, functionId), debounceId)
		require.NoError(t, err)

		// Lock must be gone
		require.False(t, unshardedCluster.Exists(unshardedDebounceClient.KeyGenerator().DebounceMigrating(ctx)))
	})
}

func TestGetDebounceInfo(t *testing.T) {
	env := newSingleShardEnv(t)
	redisDebouncer := env.deb
	ctx := context.Background()
	accountId, workspaceId, appId, functionId := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	t.Run("no debounce exists returns empty info", func(t *testing.T) {
		info, err := redisDebouncer.GetDebounceInfo(ctx, testScope(accountId, workspaceId, functionId), functionId.String())
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
		info, err := redisDebouncer.GetDebounceInfo(ctx, testScope(accountId, workspaceId, functionId), functionId.String())
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
		info, err := redisDebouncer.GetDebounceInfo(ctx, testScope(accountId, workspaceId, customFnId), customKey)
		require.NoError(t, err)
		require.NotEmpty(t, info.DebounceID)
		require.NotNil(t, info.Item)
		require.Equal(t, eventId, info.Item.EventID)

		// Query with wrong key should return empty
		info2, err := redisDebouncer.GetDebounceInfo(ctx, testScope(accountId, workspaceId, customFnId), "wrong-key")
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
		info, err := redisDebouncer.GetDebounceInfo(ctx, testScope(accountId, workspaceId, updateFnId), updateFnId.String())
		require.NoError(t, err)
		require.NotEmpty(t, info.DebounceID)
		require.NotNil(t, info.Item)
		require.Equal(t, eventId2, info.Item.EventID)
	})

	t.Run("non-existent function returns empty", func(t *testing.T) {
		nonExistentFnId := uuid.New()
		info, err := redisDebouncer.GetDebounceInfo(ctx, testScope(accountId, workspaceId, nonExistentFnId), nonExistentFnId.String())
		require.NoError(t, err)
		require.Equal(t, "", info.DebounceID)
		require.Nil(t, info.Item)
	})
}

func TestDeleteDebounce(t *testing.T) {
	env := newSingleShardEnv(t)
	redisDebouncer := env.deb
	ctx := context.Background()
	accountId, workspaceId, appId := uuid.New(), uuid.New(), uuid.New()

	t.Run("delete non-existent debounce returns deleted=false", func(t *testing.T) {
		nonExistentFnId := uuid.New()
		result, err := redisDebouncer.DeleteDebounce(ctx, testScope(accountId, workspaceId, nonExistentFnId), nonExistentFnId.String())
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
		info, err := redisDebouncer.GetDebounceInfo(ctx, testScope(accountId, workspaceId, functionId), functionId.String())
		require.NoError(t, err)
		require.NotEmpty(t, info.DebounceID)
		require.NotNil(t, info.Item)
		debounceID := info.DebounceID

		// Delete the debounce
		result, err := redisDebouncer.DeleteDebounce(ctx, testScope(accountId, workspaceId, functionId), functionId.String())
		require.NoError(t, err)
		require.True(t, result.Deleted)
		require.Equal(t, debounceID, result.DebounceID)
		require.Equal(t, eventId.String(), result.EventID)

		// Verify debounce no longer exists
		infoAfter, err := redisDebouncer.GetDebounceInfo(ctx, testScope(accountId, workspaceId, functionId), functionId.String())
		require.NoError(t, err)
		require.Equal(t, "", infoAfter.DebounceID)
		require.Nil(t, infoAfter.Item)
	})
}

func TestRunDebounce(t *testing.T) {
	env := newSingleShardEnv(t)
	redisDebouncer := env.deb
	ctx := context.Background()
	accountId, workspaceId, appId := uuid.New(), uuid.New(), uuid.New()

	t.Run("run non-existent debounce returns scheduled=false", func(t *testing.T) {
		nonExistentFnId := uuid.New()
		result, err := redisDebouncer.RunDebounce(ctx, RunDebounceOpts{
			FunctionID:  nonExistentFnId,
			DebounceKey: nonExistentFnId.String(),
			AccountID:   accountId,
			EnvID:       workspaceId,
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
		info, err := redisDebouncer.GetDebounceInfo(ctx, testScope(accountId, workspaceId, functionId), functionId.String())
		require.NoError(t, err)
		require.NotEmpty(t, info.DebounceID)
		debounceID := info.DebounceID

		// Run the debounce
		result, err := redisDebouncer.RunDebounce(ctx, RunDebounceOpts{
			FunctionID:  functionId,
			DebounceKey: functionId.String(),
			AccountID:   accountId,
			EnvID:       workspaceId,
		})
		require.NoError(t, err)
		require.True(t, result.Scheduled)
		require.Equal(t, debounceID, result.DebounceID)
		require.Equal(t, eventId.String(), result.EventID)
	})
}

func TestDeleteDebounceByID(t *testing.T) {
	env := newSingleShardEnv(t)
	redisDebouncer := env.deb
	debounceClient := env.client.Debounce()
	ctx := context.Background()
	accountId, workspaceId, appId := uuid.New(), uuid.New(), uuid.New()

	// helper to create a debounce and return its scope and ULID
	createDebounce := func(t *testing.T, functionId uuid.UUID) (queue.Scope, ulid.ULID) {
		t.Helper()
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

		scope := testScope(accountId, workspaceId, functionId)
		info, err := redisDebouncer.GetDebounceInfo(ctx, scope, functionId.String())
		require.NoError(t, err)
		require.NotEmpty(t, info.DebounceID)

		debounceID := ulid.MustParse(info.DebounceID)
		return scope, debounceID
	}

	// hashFieldExists checks if a field exists in a Redis hash using miniredis.
	hashFieldExists := func(key, field string) bool {
		val := env.cluster.HGet(key, field)
		return val != ""
	}

	t.Run("no debounce exists should succeed", func(t *testing.T) {
		fakeID := ulid.MustNew(ulid.Now(), rand.Reader)
		err := redisDebouncer.DeleteDebounceByID(ctx, testScope(accountId, workspaceId, uuid.New()), fakeID)
		require.NoError(t, err)
	})

	t.Run("delete current debounce by ID", func(t *testing.T) {
		functionId := uuid.New()
		scope, debounceID := createDebounce(t, functionId)

		// Verify the debounce item exists in the hash
		debounceKey := debounceClient.KeyGenerator().Debounce(ctx)
		require.True(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should exist in hash")

		// Verify the timeout queue item exists
		queueItemId := queue.HashID(ctx, debounceID.String())
		_, err := env.shard.LoadQueueItem(ctx, queueItemId)
		require.NoError(t, err, "timeout queue item should exist")

		// Delete by ID
		err = redisDebouncer.DeleteDebounceByID(ctx, scope, debounceID)
		require.NoError(t, err)

		// Verify the debounce item is gone from the hash
		require.False(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should be deleted from hash")

		// Verify the timeout queue item is gone
		_, err = env.shard.LoadQueueItem(ctx, queueItemId)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound, "timeout queue item should be deleted")

		// The pointer key may still exist (DeleteDebounceByID does not clean it up),
		// but GetDebounceInfo should handle this gracefully.
		info, err := redisDebouncer.GetDebounceInfo(ctx, scope, functionId.String())
		require.NoError(t, err)
		require.Nil(t, info.Item, "debounce item should not be found via pointer")
	})

	t.Run("delete debounce after pointer is dropped", func(t *testing.T) {
		scope, debounceID := createDebounce(t, uuid.New())

		debounceKey := debounceClient.KeyGenerator().Debounce(ctx)
		require.True(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should exist in hash before deletion")

		// Verify the timeout queue item exists
		queueItemId := queue.HashID(ctx, debounceID.String())
		_, err := env.shard.LoadQueueItem(ctx, queueItemId)
		require.NoError(t, err, "timeout queue item should exist")

		// Delete by ID (pointer is gone, but item + timeout still exist)
		err = redisDebouncer.DeleteDebounceByID(ctx, scope, debounceID)
		require.NoError(t, err)

		// Verify item is gone from hash
		require.False(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should be deleted from hash")

		// Verify timeout queue item is gone
		_, err = env.shard.LoadQueueItem(ctx, queueItemId)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound, "timeout queue item should be deleted")
	})

	t.Run("delete debounce when timeout already removed", func(t *testing.T) {
		scope, debounceID := createDebounce(t, uuid.New())

		debounceKey := debounceClient.KeyGenerator().Debounce(ctx)
		require.True(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should exist in hash")

		// Manually remove the timeout queue item first
		queueItemId := queue.HashID(ctx, debounceID.String())
		err := env.shard.RemoveQueueItem(ctx, scope, queue.KindDebounce, queueItemId)
		require.NoError(t, err)

		// Verify timeout is gone
		_, err = env.shard.LoadQueueItem(ctx, queueItemId)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound, "timeout should already be gone")

		// Delete by ID should still succeed (timeout removal is best-effort)
		err = redisDebouncer.DeleteDebounceByID(ctx, scope, debounceID)
		require.NoError(t, err)

		// Verify item is gone from hash
		require.False(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should be deleted from hash")
	})

	t.Run("missing item but existing timeout job should clean up timeout", func(t *testing.T) {
		scope, debounceID := createDebounce(t, uuid.New())

		debounceKey := debounceClient.KeyGenerator().Debounce(ctx)

		// Manually remove the debounce item from the hash, leaving only the timeout
		env.cluster.HDel(debounceKey, debounceID.String())
		require.False(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should be gone from hash")

		// Verify the timeout queue item still exists
		queueItemId := queue.HashID(ctx, debounceID.String())
		_, err := env.shard.LoadQueueItem(ctx, queueItemId)
		require.NoError(t, err, "timeout queue item should still exist")

		// Delete by ID — HDEL on missing item is a no-op, but timeout should be cleaned up
		err = redisDebouncer.DeleteDebounceByID(ctx, scope, debounceID)
		require.NoError(t, err)

		// Verify timeout queue item is now gone
		_, err = env.shard.LoadQueueItem(ctx, queueItemId)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound, "timeout queue item should be deleted")
	})

	t.Run("batch delete multiple debounce IDs", func(t *testing.T) {
		scope, debounceID1 := createDebounce(t, uuid.New())
		_, debounceID2 := createDebounce(t, uuid.New())

		debounceKey := debounceClient.KeyGenerator().Debounce(ctx)

		// Both items should exist
		require.True(t, hashFieldExists(debounceKey, debounceID1.String()))
		require.True(t, hashFieldExists(debounceKey, debounceID2.String()))

		queueItemId1 := queue.HashID(ctx, debounceID1.String())
		queueItemId2 := queue.HashID(ctx, debounceID2.String())

		_, err := env.shard.LoadQueueItem(ctx, queueItemId1)
		require.NoError(t, err)
		_, err = env.shard.LoadQueueItem(ctx, queueItemId2)
		require.NoError(t, err)

		// Batch delete both
		err = redisDebouncer.DeleteDebounceByID(ctx, scope, debounceID1, debounceID2)
		require.NoError(t, err)

		// Both items should be gone from hash
		require.False(t, hashFieldExists(debounceKey, debounceID1.String()))
		require.False(t, hashFieldExists(debounceKey, debounceID2.String()))

		// Both timeout queue items should be gone
		_, err = env.shard.LoadQueueItem(ctx, queueItemId1)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
		_, err = env.shard.LoadQueueItem(ctx, queueItemId2)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
	})

	t.Run("empty ID list is a no-op", func(t *testing.T) {
		err := redisDebouncer.DeleteDebounceByID(ctx, testScope(accountId, workspaceId, uuid.New()))
		require.NoError(t, err)
	})
}

// TestDebounceRetryBackoff verifies the debounceRetryDelay schedule for issue #4150.
//
// The previous implementation slept a flat 750ms per retry (5 retries max = 3,750ms worst
// case). This test locks in the new exponential schedule and its worst-case total so any
// future drift is caught immediately.
func TestDebounceRetryBackoff(t *testing.T) {
	// debounceBaseDelay is pure/deterministic — test with exact values.
	t.Run("base schedule grows exponentially and is capped", func(t *testing.T) {
		cases := []struct {
			attempt int
			want    time.Duration
		}{
			{0, 50 * time.Millisecond},
			{1, 100 * time.Millisecond},
			{2, 200 * time.Millisecond}, // capped at debounceMaxBackoff
			{3, 200 * time.Millisecond}, // still capped
			{5, 200 * time.Millisecond}, // still capped
		}
		for _, tc := range cases {
			got := debounceBaseDelay(tc.attempt)
			require.Equal(t, tc.want, got, "attempt %d: unexpected base delay", tc.attempt)
		}
	})

	t.Run("overflow guard: large attempt returns max backoff", func(t *testing.T) {
		require.Equal(t, debounceMaxBackoff, debounceBaseDelay(100))
		require.Equal(t, debounceMaxBackoff, debounceBaseDelay(63)) // 1<<63 overflows int64
	})

	t.Run("base worst-case total across all retries is exactly 350ms", func(t *testing.T) {
		var total time.Duration
		for i := 0; i < debounceMaxRetries; i++ {
			total += debounceBaseDelay(i)
		}
		// 50 + 100 + 200 = 350ms — down from 1,250ms (v1) and 3,750ms (original)
		require.Equal(t, 350*time.Millisecond, total)
	})

	// debounceRetryDelay adds ±25% jitter — test with range assertions.
	t.Run("jittered delay stays within ±25% of base", func(t *testing.T) {
		for attempt := 0; attempt < debounceMaxRetries; attempt++ {
			base := debounceBaseDelay(attempt)
			quarter := base / debounceJitterFraction
			lo := base - quarter
			hi := base + quarter

			// Sample several times; jitter is based on wall-clock nanoseconds so
			// each call may differ. All samples must land in [base-25%, base+25%).
			for sample := 0; sample < 20; sample++ {
				got := debounceRetryDelay(attempt)
				require.GreaterOrEqual(t, got, lo,
					"attempt %d sample %d: jittered delay %v below floor %v", attempt, sample, got, lo)
				require.Less(t, got, hi,
					"attempt %d sample %d: jittered delay %v at or above ceiling %v", attempt, sample, got, hi)
			}
		}
	})

	t.Run("no jittered delay exceeds debounceMaxBackoff+25%%", func(t *testing.T) {
		ceiling := debounceMaxBackoff + debounceMaxBackoff/debounceJitterFraction
		for i := 0; i <= debounceMaxRetries+10; i++ {
			d := debounceRetryDelay(i)
			require.Less(t, d, ceiling,
				"attempt %d returned %v which exceeds jitter ceiling %v", i, d, ceiling)
		}
	})
}

// TestDebounceContextCancellationDuringRetry verifies that a cancelled context is
// respected between retry attempts and does not block for a full backoff window.
func TestDebounceContextCancellationDuringRetry(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)
	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
	}

	shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)
	shardRegistry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := queue.New(context.Background(), "debounce-cancel-test", shardRegistry, opts...)
	require.NoError(t, err)

	fakeClock := clockwork.NewFakeClock()

	deb, err := NewDebouncerWithMigration(DebouncerOpts{
		Shards:           shardRegistry,
		PrimaryShardName: shard.Name(),
		Queue:            q,
		Clock:            fakeClock,
	})
	require.NoError(t, err)
	d := deb.(debouncer)

	accountID, workspaceID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ID: functionID,
		Debounce: &inngest.Debounce{
			Period: "10s",
		},
	}

	// Create an initial debounce so any subsequent call hits the update path.
	baseCtx := context.Background()
	initialEvent := ulid.MustNew(ulid.Now(), rand.Reader)
	err = d.Debounce(baseCtx, DebounceItem{
		AccountID:   accountID,
		WorkspaceID: workspaceID,
		AppID:       appID,
		FunctionID:  functionID,
		EventID:     initialEvent,
		Event: event.Event{
			Name:      "test/cancel",
			ID:        initialEvent.String(),
			Timestamp: fakeClock.Now().UnixMilli(),
		},
	}, fn)
	require.NoError(t, err)

	// Cancel the context before the second Debounce call so that any retry sleep
	// should be preempted and return ctx.Err() rather than sleeping debounceBaseBackoff.
	cancelCtx, cancel := context.WithCancel(baseCtx)
	cancel() // cancelled immediately

	start := time.Now()
	nextEvent := ulid.MustNew(ulid.Now(), rand.Reader)
	err = d.Debounce(cancelCtx, DebounceItem{
		AccountID:   accountID,
		WorkspaceID: workspaceID,
		AppID:       appID,
		FunctionID:  functionID,
		EventID:     nextEvent,
		Event: event.Event{
			Name:      "test/cancel",
			ID:        nextEvent.String(),
			Timestamp: fakeClock.Now().UnixMilli(),
		},
	}, fn)
	elapsed := time.Since(start)

	// The call may succeed (no conflict) or return a context error — both are valid.
	// What must NOT happen is blocking for 750ms × retries.
	if err != nil {
		require.ErrorIs(t, err, context.Canceled, "expected context cancellation, got: %v", err)
	}
	require.Less(t, elapsed, debounceBaseBackoff*3,
		"debounce with cancelled context blocked for %v — retry sleep was not preempted", elapsed)
}

// Backoff comparison helpers for issue #4150.
// retryStrategy lets tests compare delay schedules without touching real time or the debouncer.

// retryStrategy is a function that returns the wait duration for the nth retry attempt.
// Using a type alias keeps the signature explicit and self-documenting at call sites.
type retryStrategy func(attempt int) time.Duration

// oldDebounceRetryDelay replicates the pre-fix behavior (issue #4150): a flat 750ms
// sleep regardless of which retry attempt this is.
func oldDebounceRetryDelay(_ int) time.Duration {
	return 750 * time.Millisecond
}

// simulateWorstCase accumulates the total wait across maxAttempts using the given strategy.
// It is pure, allocation-free, and safe to call from both tests and benchmarks.
func simulateWorstCase(maxAttempts int, strategy retryStrategy) time.Duration {
	var total time.Duration
	for i := 0; i < maxAttempts; i++ {
		total += strategy(i)
	}
	return total
}

// v1DebounceRetryDelay replicates the first-pass fix (5 retries, exp backoff capped at 500ms).
// Kept here purely for the three-way comparison in TestDebounceRetryStrategyComparison.
func v1DebounceRetryDelay(attempt int) time.Duration {
	const v1MaxRetries = 5
	const v1MaxBackoff = 500 * time.Millisecond
	d := debounceBaseBackoff << attempt
	if d > v1MaxBackoff || d <= 0 {
		return v1MaxBackoff
	}
	return d
}

// TestDebounceRetryStrategyComparison is a three-way table-driven comparison:
//   - old: flat 750ms × 5 retries          = 3,750ms worst case
//   - v1:  exp backoff 50→500ms × 5 retries = 1,250ms worst case  (-67%)
//   - opt: exp backoff 50→200ms × 3 retries =   350ms worst case  (-91% vs old, -72% vs v1)
//
// Uses the retryStrategy protocol so any future strategy can be plugged in for comparison
// without touching the debouncer, the clock, or Redis.
func TestDebounceRetryStrategyComparison(t *testing.T) {
	const oldMaxRetries = 5
	const v1MaxRetries = 5

	type strategy struct {
		name       string
		maxRetries int
		fn         retryStrategy
	}

	strategies := []strategy{
		{"old (flat 750ms)", oldMaxRetries, oldDebounceRetryDelay},
		{"v1  (exp, cap 500ms, 5 retries)", v1MaxRetries, v1DebounceRetryDelay},
		{"opt (exp+jitter, cap 200ms, 3 retries)", debounceMaxRetries, debounceBaseDelay},
	}

	totals := make([]time.Duration, len(strategies))
	for i, s := range strategies {
		totals[i] = simulateWorstCase(s.maxRetries, s.fn)
	}

	oldTotal := totals[0]

	// Per-attempt side-by-side table (columns = strategies)
	maxAttempts := oldMaxRetries
	t.Log("Per-attempt backoff (base delays, no jitter):")
	t.Log("┌─────────┬──────────────┬──────────────┬──────────────┐")
	t.Logf("│ attempt │ %-12s │ %-12s │ %-12s │", "old", "v1", "opt")
	t.Log("├─────────┼──────────────┼──────────────┼──────────────┤")
	for attempt := range maxAttempts {
		cols := make([]string, len(strategies))
		for si, s := range strategies {
			if attempt < s.maxRetries {
				cols[si] = s.fn(attempt).String()
			} else {
				cols[si] = "—"
			}
		}
		t.Logf("│    %d    │ %12s │ %12s │ %12s │", attempt, cols[0], cols[1], cols[2])
	}
	t.Log("├─────────┼──────────────┼──────────────┼──────────────┤")
	t.Logf("│  TOTAL  │ %-12s │ %-12s │ %-12s │", totals[0], totals[1], totals[2])
	for i := 1; i < len(strategies); i++ {
		pct := float64(oldTotal-totals[i]) / float64(oldTotal) * 100
		t.Logf("│         │              │              │  −%.0f%% vs old (#%d)", pct, i)
	}
	t.Logf("│  summary: old=%-7s v1=%-7s opt=%-7s", totals[0], totals[1], totals[2])
	t.Log("└─────────┴──────────────┴──────────────┴──────────────┘")

	savedVsOldV1 := oldTotal - totals[1]
	savedVsOldOpt := oldTotal - totals[2]
	savedV1VsOpt := totals[1] - totals[2]
	pctVsOldV1 := float64(savedVsOldV1) / float64(oldTotal) * 100
	pctVsOldOpt := float64(savedVsOldOpt) / float64(oldTotal) * 100
	pctV1VsOpt := float64(savedV1VsOpt) / float64(totals[1]) * 100

	t.Logf("v1  vs old: saved %s (%.0f%% reduction)", savedVsOldV1, pctVsOldV1)
	t.Logf("opt vs old: saved %s (%.0f%% reduction)", savedVsOldOpt, pctVsOldOpt)
	t.Logf("opt vs v1:  saved %s (%.0f%% additional reduction)", savedV1VsOpt, pctV1VsOpt)

	// Hard assertions — these lock in the improvement guarantees.
	require.Equal(t, 3750*time.Millisecond, totals[0], "old: 750ms × 5 = 3,750ms")
	require.Equal(t, 1250*time.Millisecond, totals[1], "v1: 50+100+200+400+500 = 1,250ms")
	require.Equal(t, 350*time.Millisecond, totals[2], "opt: 50+100+200 = 350ms")

	require.InDelta(t, 67.0, pctVsOldV1, 1.0, "v1 must reduce worst-case by ~67%% vs old")
	require.InDelta(t, 91.0, pctVsOldOpt, 1.0, "opt must reduce worst-case by ~91%% vs old")
	require.InDelta(t, 72.0, pctV1VsOpt, 1.0, "opt must reduce worst-case by ~72%% vs v1")

	// Every optimized delay must be strictly shorter than the corresponding old delay.
	for attempt := 0; attempt < debounceMaxRetries; attempt++ {
		require.Less(t, debounceBaseDelay(attempt), oldDebounceRetryDelay(attempt),
			"attempt %d: optimized base delay must beat old flat delay", attempt)
	}
}

// BenchmarkRetryDelay_Old benchmarks the old flat-750ms delay computation.
// Since the old strategy ignores the attempt arg entirely, this measures pure
// function call overhead — the baseline.
func BenchmarkRetryDelay_Old(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = oldDebounceRetryDelay(i % debounceMaxRetries)
	}
}

// BenchmarkRetryDelay_New benchmarks the new exponential backoff computation.
// Should be equally cheap — the bit-shift + cap check adds no allocations.
func BenchmarkRetryDelay_New(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = debounceRetryDelay(i % debounceMaxRetries)
	}
}

// BenchmarkSimulateWorstCase_Old measures the full worst-case simulation for the old strategy.
func BenchmarkSimulateWorstCase_Old(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = simulateWorstCase(debounceMaxRetries, oldDebounceRetryDelay)
	}
}

// BenchmarkSimulateWorstCase_New measures the full worst-case simulation for the new strategy.
func BenchmarkSimulateWorstCase_New(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = simulateWorstCase(debounceMaxRetries, debounceRetryDelay)
	}
}
