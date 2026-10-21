package redis_state

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// blockingPauseDeleter holds DeletePausesForRun open until released.
type blockingPauseDeleter struct {
	called  chan ulid.ULID
	release chan struct{}
}

func (b *blockingPauseDeleter) DeletePausesForRun(_ context.Context, runID ulid.ULID, _ uuid.UUID) error {
	b.called <- runID
	<-b.release
	return nil
}

func newPauseDeleteTestMgr(t *testing.T, d state.PauseDeleter, opts ...Opt) state.Manager {
	t.Helper()
	mr := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{mr.Addr()}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	unsharded := NewUnshardedClient(rc, StateDefaultKey, QueueDefaultKey)
	sharded := NewShardedClient(ShardedClientOpts{
		UnshardedClient:        unsharded,
		FunctionRunStateClient: rc,
		BatchClient:            rc,
		StateDefaultKey:        StateDefaultKey,
		QueueDefaultKey:        QueueDefaultKey,
		FnRunIsSharded:         AlwaysShardOnRun,
	})
	m, err := New(context.Background(), append([]Opt{WithShardedClient(sharded), WithPauseDeleter(d)}, opts...)...)
	require.NoError(t, err)
	return m
}

// With async pause deletion, Delete returns once the run's state is removed,
// and the run's pauses are still deleted in the background.
func TestDeleteWithAsyncPauseDeletion(t *testing.T) {
	ctx := context.Background()
	id := state.Identifier{RunID: ulid.Make(), WorkflowID: uuid.New(), AccountID: uuid.New(), WorkspaceID: uuid.New()}

	t.Run("sync by default", func(t *testing.T) {
		d := &blockingPauseDeleter{called: make(chan ulid.ULID, 1), release: make(chan struct{})}
		m := newPauseDeleteTestMgr(t, d)
		done := make(chan error, 1)
		go func() { done <- m.Delete(ctx, id) }()
		require.Equal(t, id.RunID, <-d.called)
		select {
		case <-done:
			t.Fatal("Delete returned before pause deletion finished")
		case <-time.After(50 * time.Millisecond):
		}
		close(d.release)
		require.NoError(t, <-done)
	})

	t.Run("async", func(t *testing.T) {
		d := &blockingPauseDeleter{called: make(chan ulid.ULID, 1), release: make(chan struct{})}
		m := newPauseDeleteTestMgr(t, d, WithAsyncPauseDeletion())
		require.NoError(t, m.Delete(ctx, id))
		select {
		case runID := <-d.called:
			require.Equal(t, id.RunID, runID)
		case <-time.After(5 * time.Second):
			t.Fatal("pauses were not deleted")
		}
		close(d.release)
	})
}
