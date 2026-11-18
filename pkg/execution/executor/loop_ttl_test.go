package executor

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
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

// commitExpiring opens the instance for a scheduled trigger run and commits
// one state from that run that expires after d.
func (h *loopHarness) commitExpiring(t *testing.T, d time.Duration) loops.Snapshot {
	t.Helper()
	_, firstID := h.open(t)
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

// TestLoopRequestTouch covers the touch that every request makes.  the
// harness redis time moves forward to show which expiry the record has.
func TestLoopRequestTouch(t *testing.T) {
	t.Run("a request moves the expiry and not the counter", func(t *testing.T) {
		h := newLoopHarness(t)
		h.req.Function.Loop.IdleTTL = time.Hour
		committed := h.commitExpiring(t, 2*time.Minute)

		_, run := h.scheduleRun(t, new(loops.NewTickEvent("user-1")))
		loop, replay, err := h.e.loopRequest(h.ctx, run)
		require.NoError(t, err)
		require.Nil(t, replay)
		require.Equal(t, committed.Counter, loop.Version)
		require.JSONEq(t, `{"n":1}`, string(loop.State))

		// the record is past the expiry of its commit and has not changed.
		h.mr.FastForward(30 * time.Minute)
		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, committed, snap)

		// the new expiry is the idle ttl plus the longest request after the
		// request.  the record is there at the end of the longest request
		// and gone one idle ttl later.
		h.mr.FastForward(consts.MaxFunctionTimeout)
		_, err = h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		h.mr.FastForward(31 * time.Minute)
		_, err = h.store.Load(h.ctx, h.loopID("user-1"))
		require.ErrorIs(t, err, loops.ErrNotFound)
	})

	t.Run("a request of a run with saved steps moves the expiry", func(t *testing.T) {
		h := newLoopHarness(t)
		h.req.Function.Loop.IdleTTL = time.Hour
		committed := h.commitExpiring(t, 2*time.Minute)

		_, run := h.scheduleRun(t, new(loops.NewTickEvent("user-1")), "step-1")
		loop, _, err := h.e.loopRequest(h.ctx, run)
		require.NoError(t, err)
		require.Equal(t, committed.Counter, loop.Version)

		h.mr.FastForward(30 * time.Minute)
		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, committed, snap)
	})

	t.Run("an expiry further out is kept", func(t *testing.T) {
		h := newLoopHarness(t)
		h.req.Function.Loop.IdleTTL = time.Hour
		committed := h.commitExpiring(t, 6*time.Hour)

		_, run := h.scheduleRun(t, new(loops.NewTickEvent("user-1")))
		loop, _, err := h.e.loopRequest(h.ctx, run)
		require.NoError(t, err)
		require.Equal(t, committed.Counter, loop.Version)

		// the touch asks for three hours, the idle ttl plus the longest
		// request.  the record still has the six hours of its commit.
		h.mr.FastForward(5 * time.Hour)
		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, committed, snap)
	})

	t.Run("the run commits on the version it was sent", func(t *testing.T) {
		h := newLoopHarness(t)
		h.req.Function.Loop.IdleTTL = time.Hour
		committed := h.commitExpiring(t, 2*time.Minute)

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
		require.Equal(t, committed.Counter, d.seen.Version)

		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, committed.Counter+1, snap.Counter)
		require.JSONEq(t, `{"n":2}`, string(snap.State))
		require.Equal(t, id.RunID, *snap.LastRunID)
	})

	t.Run("a record the request opens gets the expiry of a request", func(t *testing.T) {
		// the default idle ttl is 7 days, which is past the one day expiry
		// of an open.
		h := newLoopHarness(t)
		_, run := h.scheduleRun(t, nil)
		_, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.ErrorIs(t, err, loops.ErrNotFound)

		loop, _, err := h.e.loopRequest(h.ctx, run)
		require.NoError(t, err)
		require.Equal(t, int64(0), loop.Version)
		require.Nil(t, loop.State)

		h.mr.FastForward(loops.OpenTTL + time.Hour)
		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, int64(0), snap.Counter)
		require.Equal(t, run.md.ID.RunID, snap.OpenedBy)
	})

	t.Run("a record another run opened stays at counter 0", func(t *testing.T) {
		h := newLoopHarness(t)
		_, first := h.open(t)

		_, run := h.scheduleRun(t, new(loops.NewTickEvent("user-1")))
		loop, _, err := h.e.loopRequest(h.ctx, run)
		require.NoError(t, err)
		require.Equal(t, int64(0), loop.Version)
		require.Nil(t, loop.State)
		snap, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.NoError(t, err)
		require.Equal(t, first.RunID, snap.OpenedBy)
		require.Equal(t, int64(0), snap.Counter)
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
