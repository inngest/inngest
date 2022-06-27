package inmemory

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/execution/queue"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/inngest/inngest-cli/pkg/execution/state/testharness"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestStateHarness(t *testing.T) {
	testharness.CheckState(t, func() state.Manager { return NewStateManager() })
}

func TestInMemoryPause(t *testing.T) {
	ctx := context.Background()
	sm := NewStateManager()

	s, err := sm.New(ctx, inngest.Workflow{}, ulid.MustNew(ulid.Now(), rand.Reader), map[string]any{})
	require.NoError(t, err)

	pauseID := uuid.New()

	// Deleting a noneixstent pause should error.
	err = sm.ConsumePause(ctx, pauseID)
	require.ErrorIs(t, state.ErrPauseNotFound, err)

	// Deleting a pause works as expected.
	err = sm.SavePause(ctx, state.Pause{
		ID:         pauseID,
		Identifier: s.Identifier(),
		// XXX: Right now, in memory state does not validate that the outgoing and
		// incoming edges exist in the workflow, so this won't break.  Yet.
		Outgoing: "a",
		Incoming: "b",
		Expires:  time.Now().Add(time.Second),
	})
	require.NoError(t, err)

	err = sm.ConsumePause(ctx, pauseID)
	require.NoError(t, err)

	// And you can't re-consume pauses.
	err = sm.ConsumePause(ctx, pauseID)
	require.ErrorIs(t, err, state.ErrPauseNotFound)

	// Create a new pause, and wait until the expires

	pauseID = uuid.New()
	err = sm.SavePause(ctx, state.Pause{
		ID:         pauseID,
		Identifier: s.Identifier(),
		Outgoing:   "b",
		Incoming:   "c",
		Expires:    time.Now().Add(20 * time.Millisecond),
	})
	require.NoError(t, err)
	<-time.After(21 * time.Millisecond)

	// The pause should be "not found"
	err = sm.ConsumePause(ctx, pauseID)
	require.NotNil(t, err)
	require.ErrorIs(t, err, state.ErrPauseNotFound)

	// And finally, a pause that is OnTimeout should enqueue an edge.
	pauseID = uuid.New()
	pre := time.Now()
	err = sm.SavePause(ctx, state.Pause{
		ID:         pauseID,
		Identifier: s.Identifier(),
		Outgoing:   "c",
		Incoming:   "d",
		Expires:    time.Now().Add(20 * time.Millisecond),
		OnTimeout:  true,
	})
	require.NoError(t, err)

	select {
	case next := <-sm.Channel():
		require.WithinDuration(
			t,
			pre,
			time.Now().Add(-20*time.Millisecond),
			5*time.Millisecond,
		)
		require.EqualValues(
			t,
			inngest.Edge{
				Outgoing: "c",
				Incoming: "d",
			},
			next.Payload.(queue.PayloadEdge).Edge,
		)
	case <-time.After(time.Second):
		t.Fatalf("Didn't receive enqueued item on pause timeout")
	}
}
