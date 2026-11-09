package executor

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

// A single-message read reported beside a step is answered by a message already
// saved when it is recorded. If the step finished before that, the read's wake
// is the run's only continuation. When the response of the record is lost the
// adapter sends the call again: the executor must still queue that
// continuation, or the run waits for a wake nobody queues.
func TestSubscribeReadRetryAfterALostResponseWakesTheRun(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))

	// The message is in state before the read is recorded, so recording it
	// finds the message and takes the satisfied path.
	require.NoError(t, h.deliver(event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))

	before := h.jobIDs()

	h.crash.hold("before:RecordInboxWait", "work-done")
	h.crash.arm("retry:RecordInboxWait")

	read := subscribeWaitOp("read-1", "inbox", "")
	h.driver.response = &state.DriverResponse{StatusCode: 206, RequestVersion: 2, Generator: []*state.GeneratorOpcode{plannedOp("work"), read}}

	reported := make(chan error, 1)
	go func() {
		reported <- h.tryExecute()
	}()

	time.Sleep(200 * time.Millisecond)
	h.report(stepRunOp("work", `"done"`))
	require.NoError(t, h.crash.fire("work-done"))
	require.NoError(t, <-reported)

	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("read-1", "")), "the read woke the run once: %v", h.newJobs(before))
}

// The mark that a read was answered shares the hash the waits are in, so
// loading a run's waits has to pass over it. A crash after the read is answered
// and before its wake is queued leaves the mark in place, which is when a load
// would otherwise try to read it as a wait.
func TestSubscribeAnsweredMarkIsNotReadAsAWait(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))

	// Answering the read writes the mark; crashing straight after leaves it
	// there, with no wake queued and nothing to delete it.
	h.crash.arm("after:RecordInboxWait")
	h.driver.response = &state.DriverResponse{StatusCode: 206, RequestVersion: 2, Generator: []*state.GeneratorOpcode{subscribeWaitOp("next-1", "inbox", "inbox")}}
	_ = h.tryExecute()

	waits, err := h.infra.smv2.LoadInboxWaits(h.infra.ctx, h.run.ID)
	require.NoError(t, err, "the mark is not decoded as a wait")
	for _, wait := range waits.Waits {
		require.NotEmpty(t, wait.ID, "every wait loaded is a real one: %+v", waits.Waits)
	}
}
