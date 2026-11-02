package executor

import (
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/stretchr/testify/require"
)

// A crash after the event's data is saved and before the next step is queued
// must be repaired by the event being delivered again: the wait's result is
// saved once, and the run gets exactly one discovery job.
func TestWaitForEventCrashAfterSave(t *testing.T) {
	h := newSubscribeHarness(t, waitOp("w", "test/w"))

	pause := h.pause("test/w", "w")
	require.NotNil(t, pause, "executing the wait op must leave a pause")
	require.NotContains(t, h.steps(), pause.DataKey)

	discovery := h.discoveryJobID(pause)

	h.crash.arm("after:ConsumePause")
	_ = h.deliver(event.Event{Name: "test/w", Data: map[string]any{"n": 1}})

	saved := h.steps()
	require.Contains(t, saved, pause.DataKey, "the event's data was saved before the crash")
	require.Zero(t, timesIn(h.jobIDs(), discovery), "the crash came before the next step was queued")

	require.NoError(t, h.deliver(event.Event{Name: "test/w", Data: map[string]any{"n": 1}}))

	require.Equal(t, saved, h.steps(), "the redelivery saved the same result")
	require.Equal(t, 1, timesIn(h.jobIDs(), discovery), "the redelivery queued the next step once")
}
