package executor

import (
	"fmt"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

// subscribeWaitTimeoutOp is the op an SDK reports for a read that has nothing
// unread and a timeout.
func subscribeWaitTimeoutOp(wait string, inbox string, seen string, timeout string) *state.GeneratorOpcode {
	op := subscribeWaitOp(wait, inbox, seen)
	op.Opts.(map[string]any)["timeout"] = timeout

	return op
}

// timerJobID is the id of the queue item that times a read out, as the queue
// stores it (hashed).
func (h *subscribeHarness) timerJobID(wait string) string {
	return queue.HashID(h.infra.ctx, fmt.Sprintf("%s-%s-timer", h.run.IdempotencyKey(), wait))
}

// timeoutJobID is the id of the job that runs the step after a read timed out
// (hashed).
func (h *subscribeHarness) timeoutJobID(wait string) string {
	return queue.HashID(h.infra.ctx, fmt.Sprintf("%s-%s-timeout", h.run.IdempotencyKey(), wait))
}

// timer is the queue item that times a read out, as it is queued.
type timer struct {
	jobID   string
	payload queue.PayloadPauseTimeout
}

// timerOf loads the timer queued for a read.
func (h *subscribeHarness) timerOf(wait string) timer {
	h.t.Helper()

	jobs, err := h.infra.rq.RunJobs(h.infra.ctx, h.infra.queueShard.Name(), queue.Scope{
		AccountID:  h.run.ID.Tenant.AccountID,
		EnvID:      h.run.ID.Tenant.EnvID,
		FunctionID: h.run.ID.FunctionID,
	}, h.run.ID.RunID, 1000, 0)
	require.NoError(h.t, err)

	for _, job := range jobs {
		if job.JobID != h.timerJobID(wait) {
			continue
		}

		raw, ok := job.Raw.(*queue.QueueItem)
		require.True(h.t, ok)

		payload, ok := raw.Data.Payload.(queue.PayloadPauseTimeout)
		require.True(h.t, ok, "the timer carries a pause timeout payload, got %T", raw.Data.Payload)

		return timer{jobID: job.JobID, payload: payload}
	}

	h.t.Fatalf("no timer queued for read %q", wait)

	return timer{}
}

// run runs the timer the way the queue does once it is due.
func (h *subscribeHarness) runTimer(tm timer) error {
	return h.exec.ResumePauseTimeout(h.infra.ctx, tm.payload.Pause, execution.ResumeRequest{
		IsTimeout:      true,
		IdempotencyKey: tm.jobID,
	})
}

// A timed read arms a timer. When it fires with nothing received, the read
// gets a null and the run is woken once; a message after that wakes nothing.
func TestSubscribeTimeoutWithNoMessage(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitTimeoutOp("next-1", "inbox", "inbox", "1m"))

	require.Equal(t, 1, timesIn(h.jobIDs(), h.timerJobID("next-1")), "the read armed a timer")
	require.NotContains(t, h.steps(), "next-1")

	tm := h.timerOf("next-1")
	require.NoError(t, h.runTimer(tm))

	require.Equal(t, "null", h.steps()["next-1"], "the read got a null")
	require.Equal(t, 1, timesIn(h.jobIDs(), h.timeoutJobID("next-1")), "the run was woken once")

	jobs := h.jobIDs()
	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	require.Contains(t, h.steps(), messageStepID("inbox", id), "the message is saved")
	require.Equal(t, len(jobs), len(h.jobIDs()), "the timed out read is not woken by it")

	// The queue runs a timer again if it was not acknowledged; nothing changes.
	require.NoError(t, h.runTimer(tm))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.timeoutJobID("next-1")))
}

// A message that arrives first wakes the read and takes its timer away. A timer
// that runs anyway finds no read and saves nothing.
func TestSubscribeMessageBeforeTimeout(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitTimeoutOp("next-1", "inbox", "inbox", "1m"))
	tm := h.timerOf("next-1")

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	msg := messageStepID("inbox", id)

	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-1", msg)), "the message woke the read")
	require.Zero(t, timesIn(h.jobIDs(), tm.jobID), "the message took the timer away")

	// A timer that was already running when the message came still runs.
	require.NoError(t, h.runTimer(tm))
	require.NotContains(t, h.steps(), "next-1", "no null was saved for a read that was satisfied")
	require.Zero(t, timesIn(h.jobIDs(), h.timeoutJobID("next-1")))
}

// A message found by the wait's own scan also takes the timer away, so a read
// recorded after its message was saved does not leave a timer behind.
func TestSubscribeWaitFindingAMessageLeavesNoTimer(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))

	h.report(subscribeWaitTimeoutOp("next-1", "inbox", "inbox", "1m"))

	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-1", messageStepID("inbox", id))))
	require.Zero(t, timesIn(h.jobIDs(), h.timerJobID("next-1")))
}

// The timer fires between an arrival's wake and its delete of the read: state
// holds the message and a null for the read, and the message is first.
func TestSubscribeTimerBetweenWakeAndDelete(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitTimeoutOp("next-1", "inbox", "inbox", "1m"))
	tm := h.timerOf("next-1")

	id := newEventID()
	evt := event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}
	msg := messageStepID("inbox", id)

	// The crash lands after the wake was queued and before the read was deleted.
	h.crash.arm("after:Enqueue")
	_ = h.deliverID(id, evt)
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-1", msg)), "the wake was queued")

	require.NoError(t, h.runTimer(tm))

	steps := h.steps()
	require.Contains(t, steps, msg)
	require.Equal(t, "null", steps["next-1"])

	stack, err := h.infra.smv2.LoadStack(h.infra.ctx, h.run.ID)
	require.NoError(t, err)
	require.Less(t, indexIn(stack, msg), indexIn(stack, "next-1"), "the message is first in the stack")
}

// A crash after the timeout is saved and before the run is woken is repaired by
// the timer running again.
func TestSubscribeCrashAfterTimeoutSave(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitTimeoutOp("next-1", "inbox", "inbox", "1m"))

	tm := h.timerOf("next-1")

	h.crash.arm("after:TimeoutInboxWait")
	require.Error(t, h.runTimer(tm))

	require.Equal(t, "null", h.steps()["next-1"], "the null was saved before the crash")
	require.Zero(t, timesIn(h.jobIDs(), h.timeoutJobID("next-1")), "the crash came before the wake")

	require.NoError(t, h.runTimer(tm))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.timeoutJobID("next-1")), "running the timer again woke the run once")
}

// A crash after the message is saved and before the wake is repaired by the
// event being delivered again, and the repair takes the timer away.
func TestSubscribeCrashAfterMessageSaveWithTimer(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitTimeoutOp("next-1", "inbox", "inbox", "1m"))

	id := newEventID()
	evt := event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}
	msg := messageStepID("inbox", id)

	h.crash.arm("after:SaveInboxMessage")
	_ = h.deliverID(id, evt)
	require.Equal(t, 1, timesIn(h.jobIDs(), h.timerJobID("next-1")), "the timer is still queued")

	require.NoError(t, h.deliverID(id, evt))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-1", msg)))
	require.Zero(t, timesIn(h.jobIDs(), h.timerJobID("next-1")))
}

func indexIn(ids []string, want string) int {
	for i, id := range ids {
		if id == want {
			return i
		}
	}

	return -1
}

// A read with no timeout still arms a timer, so the run holds a queue item
// while it waits (the function's singleton lock is released when it has none).
// The timer does nothing when it fires: the read keeps waiting and a message
// still wakes it.
func TestSubscribeUntimedReadHoldsAQueueItem(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))

	require.Equal(t, 1, timesIn(h.jobIDs(), h.timerJobID("next-1")), "the untimed read armed a timer")

	tm := h.timerOf("next-1")
	require.NoError(t, h.runTimer(tm))

	require.NotContains(t, h.steps(), "next-1", "the timer saved nothing")
	require.Zero(t, timesIn(h.jobIDs(), h.timeoutJobID("next-1")), "the timer woke nothing")

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	msg := messageStepID("inbox", id)

	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-1", msg)), "the read was still waiting")
	require.Zero(t, timesIn(h.jobIDs(), tm.jobID), "the message took the timer away")
}

// Closing the inbox ends an untimed read and takes its timer away.
func TestSubscribeCloseTakesAnUntimedTimerAway(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))
	tm := h.timerOf("next-1")

	h.report(closeOp("close", "inbox"))

	require.Equal(t, 1, timesIn(h.jobIDs(), h.closeJobID("next-1")), "the close woke the read")
	require.Zero(t, timesIn(h.jobIDs(), tm.jobID), "the close took the timer away")
}
