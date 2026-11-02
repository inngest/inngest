package executor

import (
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

// plannedOp is the op an SDK reports for a step.run it did not run inline.
func plannedOp(id string) *state.GeneratorOpcode {
	return &state.GeneratorOpcode{
		Op:   enums.OpcodeStepPlanned,
		ID:   id,
		Name: id,
	}
}

// A read reported beside a step is in the run's pending set. A message that
// satisfies the read takes it out in the same call that saves the message, so
// the step's own completion later finds nothing pending, and the message does
// not wake the run while the step is still pending.
func TestSubscribeParallelReadLeavesThePendingSetOnArrival(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(plannedOp("work"), subscribeWaitOp("next-1", "inbox", "inbox"))
	// The read's timer leaves the queue when the read is satisfied or closed.
	queued := len(h.jobIDs()) - 1

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	require.Contains(t, h.steps(), messageStepID("inbox", id))
	require.Equal(t, queued, len(h.jobIDs()), "the step is still pending, so the message queued no discovery")

	// If the read were still in the pending set, the step's completion would
	// see it and report that something is pending.
	hasPending, err := h.infra.smv2.SaveStep(h.infra.ctx, h.run.ID, "work", []byte(`{"data":"done"}`))
	require.NoError(t, err)
	require.False(t, hasPending, "the satisfied read left the pending set")
}

// When the step finished first, the read is the only thing pending: the message
// that satisfies it queues the discovery that continues the run.
func TestSubscribeParallelReadIsLastPending(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(plannedOp("work"), subscribeWaitOp("next-1", "inbox", "inbox"))

	hasPending, err := h.infra.smv2.SaveStep(h.infra.ctx, h.run.ID, "work", []byte(`{"data":"done"}`))
	require.NoError(t, err)
	require.True(t, hasPending, "the read is still pending")
	// The read's timer leaves the queue when the read is satisfied or closed.
	queued := len(h.jobIDs()) - 1

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	require.Equal(t, queued+1, len(h.jobIDs()), "the last pending step is done, so the message queued the discovery")

	// The same event delivered again finds the same discovery job.
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	require.Equal(t, queued+1, len(h.jobIDs()))
}

// A read in a race wakes the run whether or not anything else is pending.
func TestSubscribeRacingReadWakesWhileStepPending(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))

	racing := subscribeWaitOp("next-1", "inbox", "inbox")
	racing.Opts.(map[string]any)["parallelMode"] = "race"
	h.report(plannedOp("work"), racing)
	// The read's timer leaves the queue when the read is satisfied or closed.
	queued := len(h.jobIDs()) - 1

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	require.Equal(t, queued+1, len(h.jobIDs()), "a racing read queues its own discovery")
}
