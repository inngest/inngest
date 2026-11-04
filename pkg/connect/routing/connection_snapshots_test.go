package routing

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConnectionSnapshotTTLAndCopies(t *testing.T) {
	cache := newConnectionSnapshotCache()
	clock := clockwork.NewFakeClock()
	cache.clock = clock
	key := connectionSnapshotKey{uuid.New(), uuid.New()}
	var calls int
	source := snapshotTestConnections("connection")
	manager := &snapshotTestState{load: func(_ context.Context, env, app uuid.UUID) ([]*connectpb.ConnMetadata, error) {
		require.Equal(t, key.envID, env)
		require.Equal(t, key.appID, app)
		calls++
		return source, nil
	}}
	first, hit, err := cache.get(context.Background(), manager, key)
	require.NoError(t, err)
	require.False(t, hit)
	first.connections[0].Id = "changed"
	first.connections[0].SyncedWorkerGroups["app"] = "changed"
	first.connections[0].LastHeartbeatAt.Seconds = 0
	source[0].AllWorkerGroups["group"] = "changed"
	second, hit, err := cache.get(context.Background(), manager, key)
	require.NoError(t, err)
	require.True(t, hit)
	require.Equal(t, "connection", second.connections[0].Id)
	require.Equal(t, "group", second.connections[0].SyncedWorkerGroups["app"])
	require.Equal(t, "group", second.connections[0].AllWorkerGroups["group"])
	require.NotZero(t, second.connections[0].LastHeartbeatAt.Seconds)
	require.Same(t, first.token, second.token)
	clock.Advance(connectionSnapshotTTL - time.Nanosecond)
	_, hit, err = cache.get(context.Background(), manager, key)
	require.NoError(t, err)
	require.True(t, hit)
	clock.Advance(time.Nanosecond)
	third, hit, err := cache.get(context.Background(), manager, key)
	require.NoError(t, err)
	require.False(t, hit)
	require.NotSame(t, second.token, third.token)
	require.Equal(t, 2, calls)
}

func TestConnectionSnapshotRefreshAndConditionalInvalidation(t *testing.T) {
	cache := newConnectionSnapshotCache()
	key := connectionSnapshotKey{uuid.New(), uuid.New()}
	var calls int
	manager := &snapshotTestState{load: func(context.Context, uuid.UUID, uuid.UUID) ([]*connectpb.ConnMetadata, error) {
		calls++
		return snapshotTestConnections("connection"), nil
	}}
	first, _, err := cache.get(context.Background(), manager, key)
	require.NoError(t, err)
	second, hit, err := cache.refresh(context.Background(), manager, key, first)
	require.NoError(t, err)
	require.False(t, hit)
	require.NotSame(t, first.token, second.token)
	cache.invalidate(key, first)
	third, hit, err := cache.get(context.Background(), manager, key)
	require.NoError(t, err)
	require.True(t, hit)
	require.Same(t, second.token, third.token)
	fourth, hit, err := cache.refresh(context.Background(), manager, key, first)
	require.NoError(t, err)
	require.True(t, hit)
	require.Same(t, second.token, fourth.token)
	require.Equal(t, 2, calls)
	cache.invalidate(key, nil)
	_, hit, err = cache.get(context.Background(), manager, key)
	require.NoError(t, err)
	require.False(t, hit)
	require.Equal(t, 3, calls)
}

func TestConnectionSnapshotDoesNotRetainErrorsOrEmptyResults(t *testing.T) {
	for _, result := range []string{"error", "empty"} {
		t.Run(result, func(t *testing.T) {
			cache := newConnectionSnapshotCache()
			var calls int
			lookupErr := errors.New("lookup failed")
			manager := &snapshotTestState{load: func(context.Context, uuid.UUID, uuid.UUID) ([]*connectpb.ConnMetadata, error) {
				calls++
				if result == "error" {
					return nil, lookupErr
				}
				return nil, nil
			}}
			key := connectionSnapshotKey{uuid.New(), uuid.New()}
			for range 2 {
				snapshot, hit, err := cache.get(context.Background(), manager, key)
				require.False(t, hit)
				if result == "error" {
					require.ErrorIs(t, err, lookupErr)
				} else {
					require.NoError(t, err)
					require.Empty(t, snapshot.connections)
				}
			}
			require.Equal(t, 2, calls)
			require.Empty(t, cache.states)
			require.Zero(t, cache.bytes)
		})
	}
}

func TestConnectionSnapshotStaleReaderJoinsReplacementLoad(t *testing.T) {
	cache := newConnectionSnapshotCache()
	key := connectionSnapshotKey{uuid.New(), uuid.New()}
	release := make(chan struct{})
	var calls atomic.Int32
	manager := &snapshotTestState{load: func(ctx context.Context, _, _ uuid.UUID) ([]*connectpb.ConnMetadata, error) {
		if calls.Add(1) > 1 {
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return snapshotTestConnections("connection"), nil
	}}
	first, _, err := cache.get(context.Background(), manager, key)
	require.NoError(t, err)
	cache.invalidate(key, first)
	replacement := snapshotTestGet(context.Background(), cache, manager, key)
	snapshotTestWaiters(t, cache, key, 1)
	cache.invalidate(key, first)
	joined := make(chan snapshotTestResult, 1)
	go func() {
		snapshot, hit, err := cache.refresh(context.Background(), manager, key, first)
		joined <- snapshotTestResult{snapshot, hit, err}
	}()
	snapshotTestWaiters(t, cache, key, 2)
	close(release)
	a, b := snapshotTestReceive(t, replacement), snapshotTestReceive(t, joined)
	require.NoError(t, a.err)
	require.NoError(t, b.err)
	require.Same(t, a.snapshot.token, b.snapshot.token)
	require.NotSame(t, first.token, a.snapshot.token)
	require.Equal(t, int32(2), calls.Load())
}

func TestConnectionSnapshotCoalescesLoadsAndClonesForWaiters(t *testing.T) {
	cache := newConnectionSnapshotCache()
	key := connectionSnapshotKey{uuid.New(), uuid.New()}
	release := make(chan struct{})
	var calls atomic.Int32
	manager := &snapshotTestState{load: func(context.Context, uuid.UUID, uuid.UUID) ([]*connectpb.ConnMetadata, error) {
		calls.Add(1)
		<-release
		return snapshotTestConnections("shared"), nil
	}}
	results := make([]<-chan snapshotTestResult, 20)
	for i := range results {
		results[i] = snapshotTestGet(context.Background(), cache, manager, key)
	}
	snapshotTestWaiters(t, cache, key, len(results))
	close(release)
	var previous *connectionSnapshot
	for _, result := range results {
		r := snapshotTestReceive(t, result)
		require.NoError(t, r.err)
		require.False(t, r.hit)
		require.Equal(t, "shared", r.snapshot.connections[0].Id)
		if previous != nil {
			require.NotSame(t, previous.connections[0], r.snapshot.connections[0])
			require.Same(t, previous.token, r.snapshot.token)
		}
		previous = r.snapshot
	}
	require.Equal(t, int32(1), calls.Load())
}

func TestConnectionSnapshotWaiterCancellation(t *testing.T) {
	cache := newConnectionSnapshotCache()
	key := connectionSnapshotKey{uuid.New(), uuid.New()}
	backend := make(chan context.Context, 1)
	release := make(chan struct{})
	manager := &snapshotTestState{load: func(ctx context.Context, _, _ uuid.UUID) ([]*connectpb.ConnMetadata, error) {
		backend <- ctx
		select {
		case <-release:
			return snapshotTestConnections("shared"), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}}
	ctx, cancel := context.WithCancel(context.Background())
	first := snapshotTestGet(ctx, cache, manager, key)
	second := snapshotTestGet(context.Background(), cache, manager, key)
	snapshotTestWaiters(t, cache, key, 2)
	backendCtx := <-backend
	cancel()
	require.ErrorIs(t, snapshotTestReceive(t, first).err, context.Canceled)
	require.NoError(t, backendCtx.Err())
	close(release)
	r := snapshotTestReceive(t, second)
	require.NoError(t, r.err)
	require.Equal(t, "shared", r.snapshot.connections[0].Id)
}

func TestConnectionSnapshotCancelsLoadWhenAllWaitersLeave(t *testing.T) {
	cache := newConnectionSnapshotCache()
	key := connectionSnapshotKey{uuid.New(), uuid.New()}
	backendCanceled := make(chan struct{})
	manager := &snapshotTestState{load: func(ctx context.Context, _, _ uuid.UUID) ([]*connectpb.ConnMetadata, error) {
		<-ctx.Done()
		close(backendCanceled)
		return nil, ctx.Err()
	}}
	ctx, cancel := context.WithCancel(context.Background())
	result := snapshotTestGet(ctx, cache, manager, key)
	snapshotTestWaiters(t, cache, key, 1)
	cancel()
	require.ErrorIs(t, snapshotTestReceive(t, result).err, context.Canceled)
	select {
	case <-backendCanceled:
	case <-time.After(5 * time.Second):
		t.Fatal("backend read was not canceled")
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	require.Empty(t, cache.states)
	require.Zero(t, cache.bytes)
}

func TestConnectionSnapshotInvalidationFencesLateLoads(t *testing.T) {
	cache := newConnectionSnapshotCache()
	key := connectionSnapshotKey{uuid.New(), uuid.New()}
	oldRelease := make(chan struct{})
	oldReturned := make(chan struct{})
	newRelease := make(chan struct{})
	var calls atomic.Int32
	manager := &snapshotTestState{load: func(context.Context, uuid.UUID, uuid.UUID) ([]*connectpb.ConnMetadata, error) {
		if calls.Add(1) == 1 {
			<-oldRelease
			close(oldReturned)
			return snapshotTestConnections("old"), nil
		}
		<-newRelease
		return snapshotTestConnections("new"), nil
	}}
	first := snapshotTestGet(context.Background(), cache, manager, key)
	second := snapshotTestGet(context.Background(), cache, manager, key)
	snapshotTestWaiters(t, cache, key, 2)
	cache.mu.Lock()
	oldLoad := cache.states[key].load
	cache.mu.Unlock()
	cache.invalidate(key, nil)
	require.Eventually(t, func() bool { return calls.Load() == 2 }, 5*time.Second, time.Millisecond)
	snapshotTestWaiters(t, cache, key, 2)
	close(newRelease)
	for _, result := range []<-chan snapshotTestResult{first, second} {
		r := snapshotTestReceive(t, result)
		require.NoError(t, r.err)
		require.Equal(t, "new", r.snapshot.connections[0].Id)
	}
	close(oldRelease)
	<-oldReturned
	select {
	case <-oldLoad.finished:
	case <-time.After(5 * time.Second):
		t.Fatal("old snapshot read did not complete")
	}
	snapshot, hit, err := cache.get(context.Background(), manager, key)
	require.NoError(t, err)
	require.True(t, hit)
	require.Equal(t, "new", snapshot.connections[0].Id)
	require.Equal(t, int32(2), calls.Load())
}

func TestConnectionSnapshotLRUAndByteBounds(t *testing.T) {
	for _, bound := range []string{"entries", "bytes"} {
		t.Run(bound, func(t *testing.T) {
			cache := newConnectionSnapshotCache()
			cache.clock = clockwork.NewFakeClock()
			if bound == "entries" {
				cache.maxEntries = 2
			} else {
				cache.maxBytes = 2 * connectionSnapshotBytes(snapshotTestConnections("record"))
			}
			var calls int
			manager := &snapshotTestState{load: func(context.Context, uuid.UUID, uuid.UUID) ([]*connectpb.ConnMetadata, error) {
				calls++
				return snapshotTestConnections("record"), nil
			}}
			keys := []connectionSnapshotKey{{uuid.New(), uuid.New()}, {uuid.New(), uuid.New()}, {uuid.New(), uuid.New()}}
			for _, key := range keys[:2] {
				_, hit, err := cache.get(context.Background(), manager, key)
				require.NoError(t, err)
				require.False(t, hit)
			}
			_, hit, err := cache.get(context.Background(), manager, keys[0])
			require.NoError(t, err)
			require.True(t, hit)
			_, hit, err = cache.get(context.Background(), manager, keys[2])
			require.NoError(t, err)
			require.False(t, hit)
			_, hit, err = cache.get(context.Background(), manager, keys[0])
			require.NoError(t, err)
			require.True(t, hit)
			_, hit, err = cache.get(context.Background(), manager, keys[1])
			require.NoError(t, err)
			require.False(t, hit)
			require.Equal(t, 4, calls)
			require.LessOrEqual(t, cache.lru.Len(), 2)
			require.LessOrEqual(t, cache.bytes, cache.maxBytes)
			require.Len(t, cache.states, 2)
		})
	}
}

func TestConnectionSnapshotDoesNotRetainOversizedResults(t *testing.T) {
	cache := newConnectionSnapshotCache()
	cache.maxBytes = 1024
	var calls int
	manager := &snapshotTestState{load: func(context.Context, uuid.UUID, uuid.UUID) ([]*connectpb.ConnMetadata, error) {
		calls++
		return snapshotTestConnections(strings.Repeat("x", 2048)), nil
	}}
	key := connectionSnapshotKey{uuid.New(), uuid.New()}
	for range 2 {
		snapshot, hit, err := cache.get(context.Background(), manager, key)
		require.NoError(t, err)
		require.False(t, hit)
		require.Len(t, snapshot.connections[0].Id, 2048)
	}
	require.Equal(t, 2, calls)
	require.Empty(t, cache.states)
	require.Zero(t, cache.bytes)
}

type snapshotTestState struct {
	state.StateManager
	load func(context.Context, uuid.UUID, uuid.UUID) ([]*connectpb.ConnMetadata, error)
}

func (m *snapshotTestState) GetConnectionsByAppID(ctx context.Context, env, app uuid.UUID) ([]*connectpb.ConnMetadata, error) {
	return m.load(ctx, env, app)
}

func snapshotTestConnections(id string) []*connectpb.ConnMetadata {
	return []*connectpb.ConnMetadata{{
		Id: id, AllWorkerGroups: map[string]string{"group": "group"},
		SyncedWorkerGroups: map[string]string{"app": "group"},
		LastHeartbeatAt:    timestamppb.New(time.Unix(1000, 0)),
	}}
}

type snapshotTestResult struct {
	snapshot *connectionSnapshot
	hit      bool
	err      error
}

func snapshotTestGet(ctx context.Context, cache *connectionSnapshotCache, manager state.StateManager, key connectionSnapshotKey) <-chan snapshotTestResult {
	result := make(chan snapshotTestResult, 1)
	go func() {
		snapshot, hit, err := cache.get(ctx, manager, key)
		result <- snapshotTestResult{snapshot, hit, err}
	}()
	return result
}

func snapshotTestReceive(t *testing.T, result <-chan snapshotTestResult) snapshotTestResult {
	t.Helper()
	select {
	case r := <-result:
		return r
	case <-time.After(5 * time.Second):
		t.Fatal("snapshot read did not complete")
		return snapshotTestResult{}
	}
}

func snapshotTestWaiters(t *testing.T, cache *connectionSnapshotCache, key connectionSnapshotKey, waiters int) {
	t.Helper()
	require.Eventually(t, func() bool {
		cache.mu.Lock()
		defer cache.mu.Unlock()
		current := cache.states[key]
		return current != nil && current.load != nil && current.load.waiters == waiters
	}, 5*time.Second, time.Millisecond)
}
