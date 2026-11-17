package executor

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/loops"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/stretchr/testify/require"
)

// scheduleRun schedules one input, saves the steps on the run, and returns
// the run instance that loopRequest reads.
func (h *loopHarness) scheduleRun(t *testing.T, input *event.Event, steps ...string) (sv2.ID, *runInstance) {
	t.Helper()
	req, id := h.request(input)
	_, _, err := h.e.Schedule(h.ctx, req)
	require.NoError(t, err)
	for _, step := range steps {
		_, err := h.e.smv2.SaveStep(h.ctx, id, step, []byte(`{"data":null}`))
		require.NoError(t, err)
	}
	md, err := h.e.smv2.LoadMetadata(h.ctx, id)
	require.NoError(t, err)
	evts, err := h.e.smv2.LoadEvents(h.ctx, id)
	require.NoError(t, err)
	return id, &runInstance{
		md:     md,
		f:      req.Function,
		events: evts,
	}
}

// commitExpiring schedules the trigger and commits one state from its run
// that expires after d.
func (h *loopHarness) commitExpiring(t *testing.T, d time.Duration) loops.Snapshot {
	t.Helper()
	first, firstID := h.request(nil)
	_, _, err := h.e.Schedule(h.ctx, first)
	require.NoError(t, err)
	snap, err := h.store.Commit(h.ctx, loops.Commit{
		ID:        h.loopID("user-1"),
		RunID:     firstID.RunID,
		Counter:   0,
		State:     json.RawMessage(`{"n":1}`),
		ExpiresAt: time.Now().Add(d),
	})
	require.NoError(t, err)
	return snap
}

func TestLoopRequestTouch(t *testing.T) {
	t.Run("expiry within half the ttl is reset", func(t *testing.T) {
		h := newLoopHarness(t)
		h.req.Function.Loop.IdleTTL = time.Hour
		committed := h.commitExpiring(t, 29*time.Minute)

		_, run := h.scheduleRun(t, new(loops.NewTickEvent("user-1")))
		loop, replay, err := h.e.loopRequest(h.ctx, run)
		require.NoError(t, err)
		require.Nil(t, replay)
		require.Equal(t, committed.Counter+1, loop.Version)
		require.JSONEq(t, `{"n":1}`, string(loop.State))

		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, loop.Version, snap.Counter)
		require.WithinDuration(t, time.Now().Add(time.Hour), snap.ExpiresAt, 10*time.Second)
		require.Equal(t, committed.LastRunID, snap.LastRunID)
		require.Equal(t, committed.OpenedBy, snap.OpenedBy)
	})

	t.Run("the run commits on the version after the reset", func(t *testing.T) {
		h := newLoopHarness(t)
		h.req.Function.Loop.IdleTTL = time.Hour
		committed := h.commitExpiring(t, 29*time.Minute)

		d := &loopReturnDriver{
			echo: true,
			ops: []*state.GeneratorOpcode{
				loopReturnOp(t, state.LoopReturn{
					State: json.RawMessage(`{"n":2}`),
				}),
			},
		}
		id, resp, err := h.runLoop(t, d, new(loops.NewTickEvent("user-1")))
		require.NoError(t, err)
		require.Nil(t, resp.Err)
		require.Equal(t, committed.Counter+1, d.seen.Version)

		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, committed.Counter+2, snap.Counter)
		require.JSONEq(t, `{"n":2}`, string(snap.State))
		require.Equal(t, id.RunID, *snap.LastRunID)
	})

	t.Run("expiry further out writes nothing", func(t *testing.T) {
		h := newLoopHarness(t)
		h.req.Function.Loop.IdleTTL = time.Hour
		committed := h.commitExpiring(t, 31*time.Minute)

		_, run := h.scheduleRun(t, new(loops.NewTickEvent("user-1")))
		loop, _, err := h.e.loopRequest(h.ctx, run)
		require.NoError(t, err)
		require.Equal(t, committed.Counter, loop.Version)

		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, committed, snap)
	})

	t.Run("a run with saved steps is not reset", func(t *testing.T) {
		h := newLoopHarness(t)
		h.req.Function.Loop.IdleTTL = time.Hour
		committed := h.commitExpiring(t, 29*time.Minute)

		_, run := h.scheduleRun(t, new(loops.NewTickEvent("user-1")), "step-1")
		loop, _, err := h.e.loopRequest(h.ctx, run)
		require.NoError(t, err)
		require.Equal(t, committed.Counter, loop.Version)

		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, committed, snap)
	})

	t.Run("an instance this run opened is not reset", func(t *testing.T) {
		// the default TTL is 7 days, so the one day expiry of an open is
		// within half of it.
		h := newLoopHarness(t)
		_, run := h.scheduleRun(t, nil)
		loop, _, err := h.e.loopRequest(h.ctx, run)
		require.NoError(t, err)
		require.Equal(t, int64(0), loop.Version)
		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, int64(0), snap.Counter)
		require.WithinDuration(t, time.Now().Add(loops.OpenTTL), snap.ExpiresAt, 10*time.Second)

		// the same holds when the request opens the instance itself.
		require.NoError(t, h.store.End(h.ctx, snap))
		loop, _, err = h.e.loopRequest(h.ctx, run)
		require.NoError(t, err)
		require.Equal(t, int64(0), loop.Version)
		snap, err = h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, int64(0), snap.Counter)
		require.Equal(t, run.md.ID.RunID, snap.OpenedBy)
		require.WithinDuration(t, time.Now().Add(loops.OpenTTL), snap.ExpiresAt, 10*time.Second)
	})

	t.Run("an instance another run opened is reset", func(t *testing.T) {
		h := newLoopHarness(t)
		first, _ := h.request(nil)
		_, _, err := h.e.Schedule(h.ctx, first)
		require.NoError(t, err)

		_, run := h.scheduleRun(t, new(loops.NewTickEvent("user-1")))
		loop, _, err := h.e.loopRequest(h.ctx, run)
		require.NoError(t, err)
		require.Equal(t, int64(1), loop.Version)
		require.Nil(t, loop.State)
		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, *first.RunID, snap.OpenedBy)
		require.WithinDuration(t, time.Now().Add(7*24*time.Hour), snap.ExpiresAt, 10*time.Second)
	})
}

// TestLoopRequestMissingInstance runs inputs whose instance is gone.  a
// run with saved steps fails without retry.  the first request of a
// trigger opens a fresh instance, and the first request of a tick is
// cancelled.
func TestLoopRequestMissingInstance(t *testing.T) {
	tests := []struct {
		name   string
		input  *event.Event
		steps  []string
		opens  bool
		cancel bool
	}{
		{
			name:  "trigger with saved steps fails",
			steps: []string{"step-1"},
		},
		{
			name:  "tick with saved steps fails",
			input: new(loops.NewTickEvent("user-1")),
			steps: []string{"step-1"},
		},
		{
			name:  "first request of a trigger opens",
			opens: true,
		},
		{
			name:   "first request of a tick cancels",
			input:  new(loops.NewTickEvent("user-1")),
			cancel: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newLoopHarness(t)
			h.openAndCommit(t)
			d := &loopDriverV1{}
			h.e.driverv1 = map[string]driver.DriverV1{
				d.Name(): d,
			}
			id, _ := h.scheduleRun(t, tt.input, tt.steps...)
			snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
			require.NoError(t, err)
			require.NoError(t, h.store.End(h.ctx, snap))

			resp, err := h.execute(t, id)
			switch {
			case tt.opens:
				require.ErrorContains(t, err, "loop driver stub")
				require.Equal(t, 1, d.calls)
				require.Equal(t, int64(0), d.seen.Version)
				snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
				require.NoError(t, err)
				require.Equal(t, id.RunID, snap.OpenedBy)
				return
			case tt.cancel:
				require.ErrorIs(t, err, state.ErrFunctionCancelled)
			default:
				require.ErrorContains(t, err, "expired or was removed while run")
				require.ErrorContains(t, err, `"user-1"`)
				require.False(t, queue.ShouldRetry(err, 0, 3))
			}
			require.Nil(t, resp)
			require.Equal(t, 0, d.calls)
			_, err = h.store.Load(h.ctx, h.loopID("user-1"))
			require.ErrorIs(t, err, loops.ErrNotFound)
		})
	}
}
