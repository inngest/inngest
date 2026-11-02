package executor

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

// A drain reported beside a step is answered when it is recorded. If the step
// finished before that, the drain ends the pending set, so its wake is the
// run's only continuation. When the response of the record is lost the adapter
// sends the call again: the executor must still queue that continuation.
func TestSubscribeDrainRetryAfterALostResponseWakesTheRun(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	before := h.jobIDs()

	h.crash.hold("before:RecordInboxWait", "work-done")
	h.crash.arm("retry:RecordInboxWait")

	drain := subscribeBatchOp("drain-1", "inbox", "inbox", map[string]any{"drain": true})
	h.driver.response = &state.DriverResponse{StatusCode: 206, RequestVersion: 2, Generator: []*state.GeneratorOpcode{plannedOp("work"), drain}}

	reported := make(chan error, 1)
	go func() {
		reported <- h.tryExecute()
	}()

	time.Sleep(200 * time.Millisecond)
	h.report(stepRunOp("work", `"done"`))
	require.NoError(t, h.crash.fire("work-done"))
	require.NoError(t, <-reported)

	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("drain-1", "")), "the drain woke the run once: %v", h.newJobs(before))
}

// A message answers a batch read while a sibling step is pending, so it does
// not wake the run. The sibling's result then continues the run, and the SDK's
// report of the read, in the same response, must not continue it a second time.
func TestSubscribeBatchReadReportedAgainAfterItsSiblingContinuesOnce(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	read := subscribeBatchOp("batch-1", "inbox", "inbox", map[string]any{})
	h.report(plannedOp("work"), read)
	before := h.jobIDs()

	require.NoError(t, h.deliver(event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	h.report(stepRunOp("work", `"done"`), read)

	continuations := 0
	for _, id := range h.newJobs(before) {
		if id != h.timerJobID("batch-1") {
			continuations++
		}
	}
	require.Equal(t, 1, continuations, "one job continues the run: %v", h.newJobs(before))
}
