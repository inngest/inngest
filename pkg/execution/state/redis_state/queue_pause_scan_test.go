package redis_state

import (
	"bytes"
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestRedisScanDeferredPauseChecks(t *testing.T) {
	for _, tc := range []struct {
		name        string
		accountScan bool
		concurrency int
		warmFast    bool
	}{
		{name: "global default"},
		{name: "account default", accountScan: true},
		{name: "global deferred", concurrency: 2},
		{name: "account deferred", accountScan: true, concurrency: 2},
		{name: "warm job bypasses busy slot", concurrency: 1, warmFast: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			accountID, envID, slowID, fastID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			slowStarted, releaseSlow := make(chan struct{}), make(chan struct{})
			shard, scan := newRedisPauseScan(t, tc.concurrency, func(ctx context.Context, fnID uuid.UUID) osqueue.PartitionPausedInfo {
				if fnID == slowID {
					close(slowStarted)
					select {
					case <-releaseSlow:
					case <-ctx.Done():
					}
					return osqueue.PartitionPausedInfo{Paused: true}
				}
				if tc.concurrency > 0 {
					select {
					case <-slowStarted:
					case <-ctx.Done():
					}
				}
				return osqueue.PartitionPausedInfo{}
			}, osqueue.WithCachedPartitionPauseGetter(func(id uuid.UUID) (bool, bool) {
				return false, tc.warmFast && id == fastID
			}))
			slowScope := osqueue.Scope{AccountID: accountID, EnvID: envID, FunctionID: slowID}
			enqueueRedisPauseTestItem(t, shard, slowScope)
			enqueueRedisPauseTestItem(t, shard, osqueue.Scope{AccountID: accountID, EnvID: envID, FunctionID: fastID})

			dispatches := make(chan osqueue.QueueItem, 2)
			done := make(chan error, 1)
			go func() {
				var account *uuid.UUID
				if tc.accountScan {
					account = &accountID
				}
				done <- scan(ctx, account, func(_ context.Context, item osqueue.ProcessItem) (osqueue.DispatchedItem, error) {
					dispatches <- item.I
					return nil, nil
				})
			}()
			awaitRedisPauseTestValue(t, ctx, slowStarted)
			if tc.concurrency == 0 {
				require.Empty(t, dispatches)
				close(releaseSlow)
			}

			item := awaitRedisPauseTestValue(t, ctx, dispatches)
			require.Equal(t, fastID, item.FunctionID)
			require.NotNil(t, item.LeaseID)
			if tc.concurrency > 0 {
				close(releaseSlow)
			}
			require.NoError(t, awaitRedisPauseTestValue(t, ctx, done))
			require.Empty(t, dispatches, "paused partition must not dispatch")
			partition := readRedisPauseTestPartition(t, shard, slowScope)
			require.Greater(t, partition.ForceAtMS, time.Now().UnixMilli())
		})
	}
}

func TestRedisScanDeferredPauseRechecksPartition(t *testing.T) {
	for _, name := range []string{"paused with a live lease", "stale paused", "new park", "new migration lock"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			started, release := make(chan struct{}), make(chan struct{})
			shard, scan := newRedisPauseScan(t, 2, func(ctx context.Context, _ uuid.UUID) osqueue.PartitionPausedInfo {
				close(started)
				select {
				case <-release:
				case <-ctx.Done():
				}
				return osqueue.PartitionPausedInfo{Paused: name == "paused with a live lease" || name == "stale paused", Stale: name == "stale paused"}
			})
			scope := osqueue.Scope{AccountID: uuid.New(), EnvID: uuid.New(), FunctionID: uuid.New()}
			enqueueRedisPauseTestItem(t, shard, scope)
			partition := readRedisPauseTestPartition(t, shard, scope)

			dispatches := make(chan osqueue.QueueItem, 1)
			done := make(chan error, 1)
			go func() {
				done <- scan(ctx, nil, func(_ context.Context, item osqueue.ProcessItem) (osqueue.DispatchedItem, error) {
					dispatches <- item.I
					return nil, nil
				})
			}()
			awaitRedisPauseTestValue(t, ctx, started)
			until := time.Now().Add(time.Minute).Truncate(time.Millisecond)
			switch name {
			case "paused with a live lease":
				lease, err := shard.PartitionLease(ctx, partition, time.Minute)
				require.NoError(t, err)
				partition.LeaseID = lease
			case "new park":
				require.NoError(t, shard.PartitionRequeue(ctx, partition, until, true))
				partition.ForceAtMS = until.UnixMilli()
			case "new migration lock":
				require.NoError(t, shard.SetFunctionMigrate(ctx, scope, &until))
			}

			close(release)
			require.NoError(t, awaitRedisPauseTestValue(t, ctx, done))
			require.Empty(t, dispatches)
			after := readRedisPauseTestPartition(t, shard, scope)
			require.Equal(t, partition.LeaseID, after.LeaseID)
			require.Equal(t, partition.ForceAtMS, after.ForceAtMS)
		})
	}
}

func TestRedisScanDeferredPauseChecksSkipBusyLookupSlots(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	const limit = 3
	entered, release := make(chan struct{}, limit+1), make(chan struct{})
	shard, scan := newRedisPauseScan(t, limit, func(ctx context.Context, _ uuid.UUID) osqueue.PartitionPausedInfo {
		entered <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
		}
		return osqueue.PartitionPausedInfo{}
	})
	accountID, envID := uuid.New(), uuid.New()
	for range limit {
		enqueueRedisPauseTestItem(t, shard, osqueue.Scope{AccountID: accountID, EnvID: envID, FunctionID: uuid.New()})
	}
	retryScope := osqueue.Scope{AccountID: uuid.New(), EnvID: envID, FunctionID: uuid.New()}
	enqueueRedisPauseTestItem(t, shard, retryScope)

	dispatches := make(chan osqueue.QueueItem, limit+1)
	dispatch := func(_ context.Context, item osqueue.ProcessItem) (osqueue.DispatchedItem, error) {
		dispatches <- item.I
		return nil, nil
	}
	done := make(chan error, 1)
	go func() { done <- scan(ctx, &accountID, dispatch) }()
	for range limit {
		awaitRedisPauseTestValue(t, ctx, entered)
	}

	require.NoError(t, scan(ctx, &retryScope.AccountID, dispatch))
	require.NoError(t, ctx.Err(), "a busy pause slot must not block the scan")
	require.Empty(t, entered, "the extra lookup must not run while all slots are busy")
	require.Empty(t, dispatches)
	partition := readRedisPauseTestPartition(t, shard, retryScope)
	require.Nil(t, partition.LeaseID)
	require.Zero(t, partition.ForceAtMS)

	close(release)
	require.NoError(t, awaitRedisPauseTestValue(t, ctx, done))
	require.Len(t, dispatches, limit)
	for range limit {
		<-dispatches
	}
	require.NoError(t, scan(ctx, &retryScope.AccountID, dispatch))
	require.Equal(t, retryScope.FunctionID, awaitRedisPauseTestValue(t, ctx, dispatches).FunctionID)
}

func TestRedisScanDeferredPauseSkipsCachedPausedPartitionsBeforeSelection(t *testing.T) {
	accountID, envID, activeID := uuid.New(), uuid.New(), uuid.New()
	shard, _ := newRedisPauseScan(t, 2, nil, osqueue.WithCachedPartitionPauseGetter(func(id uuid.UUID) (bool, bool) {
		return id != activeID, true
	}))
	for range osqueue.PartitionSelectionMax {
		enqueueRedisPauseTestItem(t, shard, osqueue.Scope{AccountID: accountID, EnvID: envID, FunctionID: uuid.New()})
	}
	enqueueRedisPauseTestItem(t, shard, osqueue.Scope{AccountID: accountID, EnvID: envID, FunctionID: activeID})

	scanner := shard.(*queue)
	partitions, checkPause, err := scanner.PeekPartitionsForScan(t.Context(), nil, true, time.Now().Add(time.Second), osqueue.PartitionSelectionMax+1)
	require.NoError(t, err)
	require.Len(t, partitions, 1)
	require.Equal(t, &activeID, partitions[0].FunctionID)
	require.True(t, checkPause(t.Context(), partitions[0]))
}

func TestRedisScanDeferredPauseSkipsDuplicateInFlightLookups(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	started, release := make(chan struct{}), make(chan struct{})
	var lookups atomic.Int32
	shard, scan := newRedisPauseScan(t, 2, func(ctx context.Context, _ uuid.UUID) osqueue.PartitionPausedInfo {
		if lookups.Add(1) == 1 {
			close(started)
		}
		select {
		case <-release:
		case <-ctx.Done():
		}
		return osqueue.PartitionPausedInfo{Paused: true}
	})
	enqueueRedisPauseTestItem(t, shard, osqueue.Scope{AccountID: uuid.New(), EnvID: uuid.New(), FunctionID: uuid.New()})

	dispatch := func(context.Context, osqueue.ProcessItem) (osqueue.DispatchedItem, error) {
		return nil, nil
	}
	done := make(chan error, 1)
	go func() { done <- scan(ctx, nil, dispatch) }()
	awaitRedisPauseTestValue(t, ctx, started)
	require.NoError(t, scan(ctx, nil, dispatch))
	require.NoError(t, ctx.Err(), "a duplicate scan must return while the first lookup is pending")
	close(release)
	require.NoError(t, awaitRedisPauseTestValue(t, ctx, done))
	require.EqualValues(t, 1, lookups.Load())
}

type redisPauseScan func(context.Context, *uuid.UUID, osqueue.DispatchFunc) error

func newRedisPauseScan(t *testing.T, limit int, getter osqueue.PartitionPausedGetter, opts ...osqueue.QueueOpt) (RedisQueueShard, redisPauseScan) {
	t.Helper()
	opts = append(opts, osqueue.WithConcurrentPartitionPauseChecks(limit), osqueue.WithPartitionPausedGetter(getter))
	_, client := initRedis(t)
	t.Cleanup(client.Close)
	shard := shardFromClient("test", client, opts...)
	registry, err := osqueue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	processor, err := osqueue.New(t.Context(), "test", registry, osqueue.WithNumWorkers(32), osqueue.WithNumPartitionWorkers(16))
	require.NoError(t, err)
	return shard, func(ctx context.Context, account *uuid.UUID, dispatch osqueue.DispatchFunc) error {
		until := time.Now().Add(time.Second)
		if account == nil {
			return processor.ScanGlobalPartitions(ctx, 30, until, shard.Name(), nil, dispatch)
		}
		return processor.ScanAccountPartitions(ctx, *account, 30, until, shard.Name(), nil, dispatch)
	}
}

func enqueueRedisPauseTestItem(t *testing.T, shard RedisQueueShard, scope osqueue.Scope) {
	t.Helper()
	_, err := shard.EnqueueItem(t.Context(), osqueue.QueueItem{
		ID: uuid.NewString(), FunctionID: scope.FunctionID, WorkspaceID: scope.EnvID,
		Data: osqueue.Item{
			Kind: osqueue.KindStart, WorkspaceID: scope.EnvID,
			Identifier: state.Identifier{AccountID: scope.AccountID, WorkspaceID: scope.EnvID, WorkflowID: scope.FunctionID, RunID: ulid.Make()},
		},
	}, time.Now().Add(-time.Second), osqueue.EnqueueOpts{})
	require.NoError(t, err)
}

func readRedisPauseTestPartition(t *testing.T, shard RedisQueueShard, scope osqueue.Scope) *osqueue.QueuePartition {
	t.Helper()
	reader := NewQueueShard(shard.Name(), shard.Client())
	partition, err := reader.PartitionByID(t.Context(), scope, scope.FunctionID.String())
	require.NoError(t, err)
	return partition.QueuePartition
}

func awaitRedisPauseTestValue[T any](t *testing.T, ctx context.Context, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-ctx.Done():
		t.Fatal("timed out waiting for Redis scan")
		var zero T
		return zero
	}
}

func TestSlowRedisPartitionPeekLogsPauseTiming(t *testing.T) {
	var logs bytes.Buffer
	ctx := logger.WithStdlib(t.Context(), logger.StdlibLogger(t.Context(),
		logger.WithHandler(logger.JSONHandler), logger.WithLoggerWriter(&logs),
	))
	accountID, envID, functionID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now().Add(-2 * time.Second)
	_, client := initRedis(t)
	t.Cleanup(client.Close)
	shard := shardFromClient("test", client,
		osqueue.WithClock(clockwork.NewFakeClockAt(now)),
		osqueue.WithPartitionPausedGetter(func(context.Context, uuid.UUID) osqueue.PartitionPausedInfo {
			time.Sleep(5 * time.Millisecond)
			return osqueue.PartitionPausedInfo{CachePathTime: 2 * time.Millisecond, PostgresTime: 3 * time.Millisecond}
		}),
	)
	_, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
		ID: uuid.NewString(), FunctionID: functionID, WorkspaceID: envID,
		Data: osqueue.Item{
			Kind: osqueue.KindStart, WorkspaceID: envID,
			Identifier: state.Identifier{AccountID: accountID, WorkspaceID: envID, WorkflowID: functionID, RunID: ulid.Make()},
		},
	}, now, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	global, err := shard.PartitionPeek(ctx, true, time.Now(), 10)
	require.NoError(t, err)
	require.Len(t, global, 1)
	account, err := shard.PeekAccountPartitions(ctx, accountID, 10, time.Now(), true)
	require.NoError(t, err)
	require.Len(t, account, 1)

	dec := json.NewDecoder(&logs)
	for _, global := range []bool{true, false} {
		var entry map[string]any
		require.NoError(t, dec.Decode(&entry))
		require.Equal(t, "queue operation took >1s", entry["msg"])
		require.Equal(t, "partition_peek", entry["op"])
		require.Equal(t, global, entry["is_global_partition_peek"])
		require.EqualValues(t, 1, entry["pause_check_count"])
		require.GreaterOrEqual(t, entry["pause_check_ms"], float64(5))
		require.EqualValues(t, 2, entry["pause_cache_path_ms"])
		require.EqualValues(t, 3, entry["pause_postgres_ms"])
	}
	require.False(t, dec.More(), "expected one warning per slow scan")
}
