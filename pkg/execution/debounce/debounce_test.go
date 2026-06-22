package debounce

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
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

func migrationShardMap(defaultShard, newSystemShard queue.QueueShard) map[string]queue.QueueShard {
	return map[string]queue.QueueShard{
		consts.DefaultQueueShardName: defaultShard,
		newSystemShard.Name():        newSystemShard,
	}
}

func testScope(accountID, workspaceID, functionID uuid.UUID) queue.Scope {
	return queue.Scope{
		AccountID:  accountID,
		EnvID:      workspaceID,
		FunctionID: functionID,
	}
}

type setPointerFailingShard struct {
	queue.QueueShard
	err                  error
	deleteMigratingCalls int
}

func (s *setPointerFailingShard) DebounceSetPointer(ctx context.Context, scope queue.Scope, key string, debounceID ulid.ULID, ttl time.Duration) error {
	return s.err
}

func (s *setPointerFailingShard) DebounceDeleteMigratingFlag(ctx context.Context, scope queue.Scope, debounceID ulid.ULID) error {
	s.deleteMigratingCalls++
	return s.QueueShard.DebounceDeleteMigratingFlag(ctx, scope, debounceID)
}

type removeQueueItemFailingShard struct {
	queue.QueueShard
	err                  error
	deleteMigratingCalls int
}

func (s *removeQueueItemFailingShard) RemoveQueueItem(ctx context.Context, scope queue.Scope, partitionID string, itemID string) error {
	return s.err
}

func (s *removeQueueItemFailingShard) DebounceDeleteMigratingFlag(ctx context.Context, scope queue.Scope, debounceID ulid.ULID) error {
	s.deleteMigratingCalls++
	return s.QueueShard.DebounceDeleteMigratingFlag(ctx, scope, debounceID)
}

// migrationShardSelector routes system queue items (queueName != nil) to the
// new system shard and everything else to the default shard.
func migrationShardSelector(defaultShard, newSystemShard queue.QueueShard) func(ctx context.Context, accountID uuid.UUID, queueName *string) (queue.QueueShard, error) {
	return func(ctx context.Context, accountID uuid.UUID, queueName *string) (queue.QueueShard, error) {
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

func TestDebounceUpdateMissingTimeoutRespectsEventTimestamp(t *testing.T) {
	tests := []struct {
		name         string
		incomingAt   func(time.Time) time.Time
		incomingName string
		expectedName string
	}{
		{
			name:         "older event preserves stored debounce and restores timeout job",
			incomingAt:   func(storedAt time.Time) time.Time { return storedAt.Add(-2 * time.Second) },
			incomingName: "older",
			expectedName: "stored",
		},
		{
			name:         "newer event updates stored debounce and restores timeout job",
			incomingAt:   func(storedAt time.Time) time.Time { return storedAt.Add(2 * time.Second) },
			incomingName: "newer",
			expectedName: "newer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unshardedCluster := miniredis.RunT(t)

			unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
				InitAddress:  []string{unshardedCluster.Addr()},
				DisableCache: true,
			})
			require.NoError(t, err)

			unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
			debounceClient := unshardedClient.Debounce()

			fakeClock := clockwork.NewFakeClock()

			opts := []queue.QueueOpt{
				queue.WithKindToQueueMapping(map[string]string{
					queue.KindDebounce: queue.KindDebounce,
				}),
				queue.WithClock(fakeClock),
			}

			shard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)

			shardRegistry, err := queue.NewSingleShardRegistry(shard)
			require.NoError(t, err)

			q, err := queue.New(context.Background(), "debounce-missing-timeout-test", shardRegistry, opts...)
			require.NoError(t, err)
			kg := shard.Client().KeyGenerator()

			deb, err := NewDebouncerWithMigration(DebouncerOpts{
				Shards:           shardRegistry,
				PrimaryShardName: shard.Name(),
				Queue:            q,
				Clock:            fakeClock,
			})
			require.NoError(t, err)
			redisDebouncer := deb.(debouncer)

			ctx := context.Background()
			accountID, workspaceID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			scope := testScope(accountID, workspaceID, functionID)

			fn := inngest.Function{
				ID: functionID,
				Debounce: &inngest.Debounce{
					Period: "10s",
				},
			}

			storedAt := fakeClock.Now().Add(2 * time.Second)
			storedEventID := ulid.MustNew(ulid.Timestamp(storedAt), rand.Reader)
			storedDI := DebounceItem{
				AccountID:   accountID,
				WorkspaceID: workspaceID,
				AppID:       appID,
				FunctionID:  functionID,
				EventID:     storedEventID,
				Event: event.Event{
					Name:      "stored",
					ID:        storedEventID.String(),
					Timestamp: storedAt.UnixMilli(),
				},
			}

			byt, err := json.Marshal(storedDI)
			require.NoError(t, err)

			debounceID := ulid.MustNew(ulid.Timestamp(fakeClock.Now()), rand.Reader)
			existingID, err := shard.DebounceCreate(ctx, scope, functionID.String(), debounceID, byt, 10*time.Second)
			require.NoError(t, err)
			require.Nil(t, existingID)

			incomingAt := tt.incomingAt(storedAt)
			incomingEventID := ulid.MustNew(ulid.Timestamp(incomingAt), rand.Reader)
			err = redisDebouncer.Debounce(ctx, DebounceItem{
				AccountID:   accountID,
				WorkspaceID: workspaceID,
				AppID:       appID,
				FunctionID:  functionID,
				EventID:     incomingEventID,
				Event: event.Event{
					Name:      tt.incomingName,
					ID:        incomingEventID.String(),
					Timestamp: incomingAt.UnixMilli(),
				},
			}, fn)
			require.NoError(t, err)

			debounceIDs, err := unshardedCluster.HKeys(debounceClient.KeyGenerator().Debounce(ctx))
			require.NoError(t, err)
			require.Equal(t, []string{debounceID.String()}, debounceIDs)

			var stored DebounceItem
			err = json.Unmarshal([]byte(unshardedCluster.HGet(debounceClient.KeyGenerator().Debounce(ctx), debounceID.String())), &stored)
			require.NoError(t, err)
			require.Equal(t, tt.expectedName, stored.Event.Name)
			if tt.expectedName == "stored" {
				require.Equal(t, storedAt.UnixMilli(), stored.Event.Timestamp)
			} else {
				require.Equal(t, incomingAt.UnixMilli(), stored.Event.Timestamp)
			}

			queueItemIDs, err := unshardedCluster.HKeys(kg.QueueItem())
			require.NoError(t, err)
			require.Equal(t, []string{queue.HashID(ctx, debounceID.String())}, queueItemIDs)
		})
	}
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
		existingID, _, _, err := defaultQueueShard.DebouncePrepareMigration(ctx, testScope(accountId, workspaceId, fn.ID), fn.ID.String(), fakeID)
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
		existingID, _, _, err := defaultQueueShard.DebouncePrepareMigration(ctx, testScope(accountId, workspaceId, fn.ID), fn.ID.String(), fakeID)
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

func TestRollbackPreparedMigrationKeepsMigratingFlagWhenPointerRestoreFails(t *testing.T) {
	secondaryCluster := miniredis.RunT(t)
	primaryCluster := miniredis.RunT(t)

	secondaryRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{secondaryCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer secondaryRc.Close()

	primaryRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{primaryCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer primaryRc.Close()

	secondaryClient := redis_state.NewUnshardedClient(secondaryRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	primaryClient := redis_state.NewUnshardedClient(primaryRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	secondaryDebounceClient := secondaryClient.Debounce()

	secondaryShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, secondaryClient.Queue())
	primaryShard := redis_state.NewQueueShard("new-system", primaryClient.Queue())
	failingSecondary := &setPointerFailingShard{
		QueueShard: secondaryShard,
		err:        errors.New("set pointer failed"),
	}

	shards, err := queue.NewShardRegistry(
		migrationShardMap(failingSecondary, primaryShard),
		queue.WithShardSelector(migrationShardSelector(failingSecondary, primaryShard)),
		queue.WithPrimary(primaryShard),
	)
	require.NoError(t, err)

	redisDebouncer := debouncer{
		shards:             shards,
		primaryShardName:   primaryShard.Name(),
		secondaryShardName: failingSecondary.Name(),
	}

	ctx := context.Background()
	accountID, workspaceID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	scope := testScope(accountID, workspaceID, functionID)
	fn := inngest.Function{
		ID: functionID,
		Debounce: &inngest.Debounce{
			Period:  "10s",
			Timeout: util.StrPtr("60s"),
		},
	}
	key := functionID.String()
	debounceID := ulid.MustNew(ulid.Now(), rand.Reader)
	fakeDebounceID := ulid.MustNew(ulid.Now()+1, rand.Reader)
	di := DebounceItem{
		AccountID:   accountID,
		WorkspaceID: workspaceID,
		AppID:       appID,
		FunctionID:  functionID,
		EventID:     ulid.MustNew(ulid.Now()+2, rand.Reader),
		Event: event.Event{
			Name: "initial",
			ID:   "initial-event",
		},
		Timeout: time.Now().Add(time.Minute).UnixMilli(),
	}
	item, err := json.Marshal(di)
	require.NoError(t, err)

	existingID, err := secondaryShard.DebounceCreate(ctx, scope, key, debounceID, item, 10*time.Second)
	require.NoError(t, err)
	require.Nil(t, existingID)

	preparedID, _, pointerTTL, err := secondaryShard.DebouncePrepareMigration(ctx, scope, key, fakeDebounceID)
	require.NoError(t, err)
	require.NotNil(t, preparedID)
	require.Equal(t, debounceID, *preparedID)

	err = redisDebouncer.rollbackPreparedMigration(ctx, di, fn, &preparedMigration{
		debounceID:       debounceID,
		timeoutUnixMilli: di.Timeout,
		pointerTTL:       pointerTTL,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to restore secondary debounce pointer")
	require.ErrorContains(t, err, "migration flag left in place")
	require.Equal(t, 0, failingSecondary.deleteMigratingCalls)

	pointerVal, err := secondaryCluster.Get(secondaryDebounceClient.KeyGenerator().DebouncePointer(ctx, functionID, key))
	require.NoError(t, err)
	require.Equal(t, fakeDebounceID.String(), pointerVal)

	migratingIDs, err := secondaryCluster.HKeys(secondaryDebounceClient.KeyGenerator().DebounceMigrating(ctx))
	require.NoError(t, err)
	require.Contains(t, migratingIDs, debounceID.String())
	require.NotEmpty(t, secondaryCluster.HGet(secondaryDebounceClient.KeyGenerator().Debounce(ctx), debounceID.String()))
}

func TestFinalizePreparedMigrationCommitsWhenPrimaryReadyAfterError(t *testing.T) {
	secondaryCluster := miniredis.RunT(t)
	primaryCluster := miniredis.RunT(t)

	secondaryRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{secondaryCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer secondaryRc.Close()

	primaryRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{primaryCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer primaryRc.Close()

	secondaryClient := redis_state.NewUnshardedClient(secondaryRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	primaryClient := redis_state.NewUnshardedClient(primaryRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	secondaryDebounceClient := secondaryClient.Debounce()
	primaryDebounceClient := primaryClient.Debounce()

	fakeClock := clockwork.NewFakeClock()
	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
		queue.WithClock(fakeClock),
	}

	secondaryShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, secondaryClient.Queue(), opts...)
	primaryShard := redis_state.NewQueueShard("new-system", primaryClient.Queue(), opts...)

	shards, err := queue.NewShardRegistry(
		migrationShardMap(secondaryShard, primaryShard),
		queue.WithShardSelector(migrationShardSelector(secondaryShard, primaryShard)),
		queue.WithPrimary(primaryShard),
	)
	require.NoError(t, err)

	q, err := queue.New(context.Background(), "new-queue", shards, opts...)
	require.NoError(t, err)

	redisDebouncer := debouncer{
		shards:             shards,
		primaryShardName:   primaryShard.Name(),
		secondaryShardName: secondaryShard.Name(),
		queue:              q,
		c:                  fakeClock,
	}

	ctx := context.Background()
	accountID, workspaceID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	scope := testScope(accountID, workspaceID, functionID)
	fn := inngest.Function{
		ID: functionID,
		Debounce: &inngest.Debounce{
			Period:  "10s",
			Timeout: util.StrPtr("60s"),
		},
	}
	key := functionID.String()
	debounceID := ulid.MustNew(ulid.Now(), rand.Reader)
	fakeDebounceID := ulid.MustNew(ulid.Now()+1, rand.Reader)
	initialTime := fakeClock.Now()
	di := DebounceItem{
		AccountID:   accountID,
		WorkspaceID: workspaceID,
		AppID:       appID,
		FunctionID:  functionID,
		EventID:     ulid.MustNew(ulid.Timestamp(initialTime), rand.Reader),
		Event: event.Event{
			Name:      "initial",
			ID:        "initial-event",
			Timestamp: initialTime.UnixMilli(),
		},
		Timeout: initialTime.Add(time.Minute).UnixMilli(),
	}
	item, err := json.Marshal(di)
	require.NoError(t, err)

	existingID, err := secondaryShard.DebounceCreate(ctx, scope, key, debounceID, item, 10*time.Second)
	require.NoError(t, err)
	require.Nil(t, existingID)

	queueItem := redisDebouncer.queueItem(ctx, di, debounceID)
	err = q.Enqueue(ctx, queueItem, initialTime.Add(10*time.Second), queue.EnqueueOpts{ForceQueueShardName: secondaryShard.Name()})
	require.NoError(t, err)

	preparedID, timeoutUnixMillis, pointerTTL, err := secondaryShard.DebouncePrepareMigration(ctx, scope, key, fakeDebounceID)
	require.NoError(t, err)
	require.NotNil(t, preparedID)
	require.Equal(t, debounceID, *preparedID)
	require.Equal(t, di.Timeout, timeoutUnixMillis)

	// This fills the coverage gap where the primary debounce is already fully
	// ready, but a later migration operation returns an error. In that case we
	// must commit cleanup of the secondary instead of rolling back and leaving
	// duplicate runnable debounce state on both shards.
	createdID, err := redisDebouncer.newDebounce(ctx, di, fn, pointerTTL, true, debounceID)
	require.NoError(t, err)
	require.NotNil(t, createdID)
	require.Equal(t, debounceID, *createdID)

	err = redisDebouncer.finalizePreparedMigration(ctx, di, fn, &preparedMigration{
		debounceID:       debounceID,
		timeoutUnixMilli: timeoutUnixMillis,
		pointerTTL:       pointerTTL,
	}, errors.New("post-primary migration error"))
	require.NoError(t, err)

	_, err = secondaryCluster.Get(secondaryDebounceClient.KeyGenerator().DebouncePointer(ctx, functionID, key))
	require.Error(t, err)
	require.ErrorContains(t, err, "no such key")

	_, err = secondaryCluster.HKeys(secondaryDebounceClient.KeyGenerator().DebounceMigrating(ctx))
	require.Error(t, err)
	require.ErrorContains(t, err, "no such key")
	require.Empty(t, secondaryCluster.HGet(secondaryDebounceClient.KeyGenerator().Debounce(ctx), debounceID.String()))
	require.Empty(t, secondaryCluster.HGet(secondaryClient.Queue().KeyGenerator().QueueItem(), queue.HashID(ctx, debounceID.String())))

	primaryPointer, err := primaryCluster.Get(primaryDebounceClient.KeyGenerator().DebouncePointer(ctx, functionID, key))
	require.NoError(t, err)
	require.Equal(t, debounceID.String(), primaryPointer)
	require.NotEmpty(t, primaryCluster.HGet(primaryDebounceClient.KeyGenerator().Debounce(ctx), debounceID.String()))
	require.NotEmpty(t, primaryCluster.HGet(primaryClient.Queue().KeyGenerator().QueueItem(), queue.HashID(ctx, debounceID.String())))
}

func TestCompletePreparedMigrationKeepsMigratingFlagWhenSecondaryCleanupFails(t *testing.T) {
	secondaryCluster := miniredis.RunT(t)
	primaryCluster := miniredis.RunT(t)

	secondaryRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{secondaryCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer secondaryRc.Close()

	primaryRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{primaryCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer primaryRc.Close()

	secondaryClient := redis_state.NewUnshardedClient(secondaryRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	primaryClient := redis_state.NewUnshardedClient(primaryRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	secondaryDebounceClient := secondaryClient.Debounce()

	fakeClock := clockwork.NewFakeClock()
	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
		queue.WithClock(fakeClock),
	}

	secondaryShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, secondaryClient.Queue(), opts...)
	failingSecondary := &removeQueueItemFailingShard{
		QueueShard: secondaryShard,
		err:        errors.New("remove queue item failed"),
	}
	primaryShard := redis_state.NewQueueShard("new-system", primaryClient.Queue(), opts...)

	shards, err := queue.NewShardRegistry(
		migrationShardMap(failingSecondary, primaryShard),
		queue.WithShardSelector(migrationShardSelector(failingSecondary, primaryShard)),
		queue.WithPrimary(primaryShard),
	)
	require.NoError(t, err)

	q, err := queue.New(context.Background(), "new-queue", shards, opts...)
	require.NoError(t, err)

	redisDebouncer := debouncer{
		shards:             shards,
		primaryShardName:   primaryShard.Name(),
		secondaryShardName: failingSecondary.Name(),
		queue:              q,
		c:                  fakeClock,
	}

	ctx := context.Background()
	accountID, workspaceID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	scope := testScope(accountID, workspaceID, functionID)
	fn := inngest.Function{
		ID: functionID,
		Debounce: &inngest.Debounce{
			Period:  "10s",
			Timeout: util.StrPtr("60s"),
		},
	}
	key := functionID.String()
	debounceID := ulid.MustNew(ulid.Now(), rand.Reader)
	fakeDebounceID := ulid.MustNew(ulid.Now()+1, rand.Reader)
	initialTime := fakeClock.Now()
	di := DebounceItem{
		AccountID:   accountID,
		WorkspaceID: workspaceID,
		AppID:       appID,
		FunctionID:  functionID,
		EventID:     ulid.MustNew(ulid.Timestamp(initialTime), rand.Reader),
		Event: event.Event{
			Name:      "initial",
			ID:        "initial-event",
			Timestamp: initialTime.UnixMilli(),
		},
		Timeout: initialTime.Add(time.Minute).UnixMilli(),
	}
	item, err := json.Marshal(di)
	require.NoError(t, err)

	existingID, err := secondaryShard.DebounceCreate(ctx, scope, key, debounceID, item, 10*time.Second)
	require.NoError(t, err)
	require.Nil(t, existingID)

	queueItem := redisDebouncer.queueItem(ctx, di, debounceID)
	err = q.Enqueue(ctx, queueItem, initialTime.Add(10*time.Second), queue.EnqueueOpts{ForceQueueShardName: secondaryShard.Name()})
	require.NoError(t, err)

	preparedID, timeoutUnixMillis, pointerTTL, err := secondaryShard.DebouncePrepareMigration(ctx, scope, key, fakeDebounceID)
	require.NoError(t, err)
	require.NotNil(t, preparedID)
	require.Equal(t, debounceID, *preparedID)

	createdID, err := redisDebouncer.newDebounce(ctx, di, fn, pointerTTL, true, debounceID)
	require.NoError(t, err)
	require.NotNil(t, createdID)

	err = redisDebouncer.finalizePreparedMigration(ctx, di, fn, &preparedMigration{
		debounceID:       debounceID,
		timeoutUnixMilli: timeoutUnixMillis,
		pointerTTL:       pointerTTL,
	}, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "secondary debounce migration guard left in place")
	require.Equal(t, 0, failingSecondary.deleteMigratingCalls)

	migratingIDs, err := secondaryCluster.HKeys(secondaryDebounceClient.KeyGenerator().DebounceMigrating(ctx))
	require.NoError(t, err)
	require.Contains(t, migratingIDs, debounceID.String())
}

func TestDebounceMigrationFailurePreservesExistingDebounce(t *testing.T) {
	unshardedCluster := miniredis.RunT(t)
	newSystemCluster := miniredis.RunT(t)

	unshardedRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{unshardedCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer unshardedRc.Close()

	newSystemRc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{newSystemCluster.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	newSystemClient := redis_state.NewUnshardedClient(newSystemRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)

	unshardedDebounceClient := unshardedClient.Debounce()
	newSystemDebounceClient := newSystemClient.Debounce()

	fakeClock := clockwork.NewFakeClock()

	opts := []queue.QueueOpt{
		queue.WithKindToQueueMapping(map[string]string{
			queue.KindDebounce: queue.KindDebounce,
		}),
		queue.WithClock(fakeClock),
	}

	defaultQueueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), opts...)
	newSystemShard := redis_state.NewQueueShard("new-system", newSystemClient.Queue(), opts...)

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
		ShouldMigrate: func(ctx context.Context, accountID uuid.UUID) bool {
			return true
		},
		Clock: fakeClock,
	})
	require.NoError(t, err)
	newRedisDebouncer := newDeb.(debouncer)

	ctx := context.Background()
	accountID, workspaceID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ID: functionID,
		Debounce: &inngest.Debounce{
			Period:  "10s",
			Timeout: util.StrPtr("60s"),
		},
	}

	firstEventTime := fakeClock.Now()
	firstEventID := ulid.MustNew(ulid.Timestamp(firstEventTime), rand.Reader)

	err = oldRedisDebouncer.Debounce(ctx, DebounceItem{
		AccountID:   accountID,
		WorkspaceID: workspaceID,
		AppID:       appID,
		FunctionID:  functionID,
		EventID:     firstEventID,
		Event: event.Event{
			Name:      "initial",
			ID:        firstEventID.String(),
			Timestamp: firstEventTime.UnixMilli(),
		},
	}, fn)
	require.NoError(t, err)

	debounceIDs, err := unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().Debounce(ctx))
	require.NoError(t, err)
	require.Len(t, debounceIDs, 1)
	originalDebounceID := debounceIDs[0]

	queueItemIDs, err := unshardedCluster.HKeys(defaultQueueShard.Client().KeyGenerator().QueueItem())
	require.NoError(t, err)
	require.Len(t, queueItemIDs, 1)
	require.Equal(t, queue.HashID(ctx, originalDebounceID), queueItemIDs[0])

	// Force the primary path to fail after prepareMigration() has already
	// locked the old debounce for migration.
	newSystemRc.Close()

	fakeClock.Advance(5 * time.Second)
	secondEventTime := fakeClock.Now()
	secondEventID := ulid.MustNew(ulid.Timestamp(secondEventTime), rand.Reader)

	err = newRedisDebouncer.Debounce(ctx, DebounceItem{
		AccountID:   accountID,
		WorkspaceID: workspaceID,
		AppID:       appID,
		FunctionID:  functionID,
		EventID:     secondEventID,
		Event: event.Event{
			Name:      "migrate",
			ID:        secondEventID.String(),
			Timestamp: secondEventTime.UnixMilli(),
		},
	}, fn)
	require.Error(t, err)

	// If the new primary debounce is never created, the old debounce must remain
	// runnable on the secondary cluster.
	require.NotEmpty(t, unshardedCluster.HGet(unshardedDebounceClient.KeyGenerator().Debounce(ctx), originalDebounceID))

	pointerVal, err := unshardedCluster.Get(unshardedDebounceClient.KeyGenerator().DebouncePointer(ctx, functionID, functionID.String()))
	require.NoError(t, err)
	require.NotEmpty(t, pointerVal)
	require.Equal(t, originalDebounceID, pointerVal)

	_, err = unshardedCluster.HKeys(unshardedDebounceClient.KeyGenerator().DebounceMigrating(ctx))
	require.Error(t, err)
	require.ErrorContains(t, err, "no such key")

	queueItemIDs, err = unshardedCluster.HKeys(defaultQueueShard.Client().KeyGenerator().QueueItem())
	require.NoError(t, err)
	require.Len(t, queueItemIDs, 1)
	require.Equal(t, queue.HashID(ctx, originalDebounceID), queueItemIDs[0])

	require.False(t, newSystemCluster.Exists(newSystemDebounceClient.KeyGenerator().DebouncePointer(ctx, functionID, functionID.String())))
	require.False(t, newSystemCluster.Exists(newSystemDebounceClient.KeyGenerator().Debounce(ctx)))
}

func TestGetDebounceInfo(t *testing.T) {
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

	q, err := queue.New(context.Background(), "debounce-test", shardRegistry, opts...)
	require.NoError(t, err)

	redisDebouncer, err := NewDebouncer(shardRegistry, shard.Name(), q)
	require.NoError(t, err)

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

	q, err := queue.New(context.Background(), "debounce-test", shardRegistry, opts...)
	require.NoError(t, err)

	redisDebouncer, err := NewDebouncer(shardRegistry, shard.Name(), q)
	require.NoError(t, err)

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

	q, err := queue.New(context.Background(), "debounce-test", shardRegistry, opts...)
	require.NoError(t, err)

	redisDebouncer, err := NewDebouncer(shardRegistry, shard.Name(), q)
	require.NoError(t, err)

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

	redisDebouncer, err := NewDebouncer(shardRegistry, shard.Name(), q)
	require.NoError(t, err)

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
		val := unshardedCluster.HGet(key, field)
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
		_, err := shard.LoadQueueItem(ctx, queueItemId)
		require.NoError(t, err, "timeout queue item should exist")

		// Delete by ID
		err = redisDebouncer.DeleteDebounceByID(ctx, scope, debounceID)
		require.NoError(t, err)

		// Verify the debounce item is gone from the hash
		require.False(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should be deleted from hash")

		// Verify the timeout queue item is gone
		_, err = shard.LoadQueueItem(ctx, queueItemId)
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
		_, err := shard.LoadQueueItem(ctx, queueItemId)
		require.NoError(t, err, "timeout queue item should exist")

		// Delete by ID (pointer is gone, but item + timeout still exist)
		err = redisDebouncer.DeleteDebounceByID(ctx, scope, debounceID)
		require.NoError(t, err)

		// Verify item is gone from hash
		require.False(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should be deleted from hash")

		// Verify timeout queue item is gone
		_, err = shard.LoadQueueItem(ctx, queueItemId)
		require.ErrorIs(t, err, queue.ErrQueueItemNotFound, "timeout queue item should be deleted")
	})

	t.Run("delete debounce when timeout already removed", func(t *testing.T) {
		scope, debounceID := createDebounce(t, uuid.New())

		debounceKey := debounceClient.KeyGenerator().Debounce(ctx)
		require.True(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should exist in hash")

		// Manually remove the timeout queue item first
		queueItemId := queue.HashID(ctx, debounceID.String())
		err := shard.RemoveQueueItem(ctx, scope, queue.KindDebounce, queueItemId)
		require.NoError(t, err)

		// Verify timeout is gone
		_, err = shard.LoadQueueItem(ctx, queueItemId)
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
		unshardedCluster.HDel(debounceKey, debounceID.String())
		require.False(t, hashFieldExists(debounceKey, debounceID.String()), "debounce item should be gone from hash")

		// Verify the timeout queue item still exists
		queueItemId := queue.HashID(ctx, debounceID.String())
		_, err := shard.LoadQueueItem(ctx, queueItemId)
		require.NoError(t, err, "timeout queue item should still exist")

		// Delete by ID — HDEL on missing item is a no-op, but timeout should be cleaned up
		err = redisDebouncer.DeleteDebounceByID(ctx, scope, debounceID)
		require.NoError(t, err)

		// Verify timeout queue item is now gone
		_, err = shard.LoadQueueItem(ctx, queueItemId)
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

		_, err := shard.LoadQueueItem(ctx, queueItemId1)
		require.NoError(t, err)
		_, err = shard.LoadQueueItem(ctx, queueItemId2)
		require.NoError(t, err)

		// Batch delete both
		err = redisDebouncer.DeleteDebounceByID(ctx, scope, debounceID1, debounceID2)
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
		err := redisDebouncer.DeleteDebounceByID(ctx, testScope(accountId, workspaceId, uuid.New()))
		require.NoError(t, err)
	})
}
