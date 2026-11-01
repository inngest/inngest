package executor

import (
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

// sleepOp is the op an SDK reports for step.sleep.
func sleepOp(id string) *state.GeneratorOpcode {
	return &state.GeneratorOpcode{
		Op:   enums.OpcodeSleep,
		ID:   id,
		Name: "1h",
	}
}

// newJobs is the queue items that exist now and did not before.
func (h *subscribeHarness) newJobs(before []string) []string {
	h.t.Helper()

	known := map[string]bool{}
	for _, id := range before {
		known[id] = true
	}

	out := []string{}
	for _, id := range h.jobIDs() {
		if !known[id] {
			out = append(out, id)
		}
	}

	return out
}

// A read that was first reported beside another op and is reported again with
// the close that ends it wakes the run once. The executor handles the two ops
// at the same time, so the test fixes which of them goes first.
func TestSubscribeCloseOfAReadReportedAgainWakesOnce(t *testing.T) {
	orders := map[string]func(h *subscribeHarness){
		"the close first": func(h *subscribeHarness) {
			h.crash.hold("before:RecordInboxWait", "after:CloseInbox")
		},
		"the read first": func(h *subscribeHarness) {
			h.crash.hold("before:CloseInbox", "after:RecordInboxWait")
		},
	}

	for name, order := range orders {
		t.Run(name, func(t *testing.T) {
			h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
			h.report(subscribeWaitOp("next-1", "inbox", "inbox"), sleepOp("nap"))

			// The sleep is done, so the read is the only thing pending.
			_, err := h.infra.smv2.SaveStep(h.infra.ctx, h.run.ID, "nap", []byte(`{"data":null}`))
			require.NoError(t, err)
			before := h.jobIDs()

			order(h)
			h.report(subscribeWaitOp("next-1", "inbox", "inbox"), closeOp("inbox:close", "inbox"))

			require.Len(t, h.newJobs(before), 1, "one job continues the run after the close")
		})
	}
}

// A message wakes a read once when the SDK reports the read again after the
// message arrived, alone, so under no coalesce key.
func TestSubscribeMessageWakeOfAReadReportedAgainIsOneJob(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"), sleepOp("nap"))

	_, err := h.infra.smv2.SaveStep(h.infra.ctx, h.run.ID, "nap", []byte(`{"data":null}`))
	require.NoError(t, err)
	before := h.jobIDs()

	require.NoError(t, h.deliver(event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))

	require.Len(t, h.newJobs(before), 1, "one job continues the run after the message")
}

// discoveries is the queue items made since before, other than the timer of a
// read: each is a job that runs every step after the read.
func (h *subscribeHarness) discoveries(before []string) int {
	h.t.Helper()

	n := 0
	for _, id := range h.newJobs(before) {
		if id != h.timerJobID("next-1") {
			n++
		}
	}

	return n
}

// A message takes a read out of the pending set but leaves its record until its
// wake is queued and the record deleted. A close that runs in that window finds
// the same record. The read is woken once.
func TestSubscribeCloseDuringAMessageWakeWakesOnce(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))
	before := h.jobIDs()

	h.crash.hold("before:CloseInbox", "after:SaveInboxMessage")
	h.crash.hold("before:DeleteInboxWait", "after:CloseInbox")

	delivered := make(chan error, 1)
	go func() {
		delivered <- h.deliver(event.Event{Name: "test/msg", Data: map[string]any{"n": 1}})
	}()

	h.report(closeOp("inbox:close", "inbox"))
	require.NoError(t, <-delivered)

	require.Equal(t, 1, h.discoveries(before), "one job continues the run after the read")
}

// The same for the timer of the read: it runs between the message's save and
// the deletion of the record.
func TestSubscribeTimeoutDuringAMessageWakeWakesOnce(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitTimeoutOp("next-1", "inbox", "inbox", "1m"))
	tm := h.timerOf("next-1")
	before := h.jobIDs()

	h.crash.hold("before:TimeoutInboxWait", "after:SaveInboxMessage")
	h.crash.hold("before:DeleteInboxWait", "after:TimeoutInboxWait")

	delivered := make(chan error, 1)
	go func() {
		delivered <- h.deliver(event.Event{Name: "test/msg", Data: map[string]any{"n": 1}})
	}()

	require.NoError(t, h.runTimer(tm))
	require.NoError(t, <-delivered)

	require.Equal(t, 1, h.discoveries(before), "one job continues the run after the read")
}
