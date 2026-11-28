package executor

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/stretchr/testify/require"
)

// closeOp is the op an SDK reports for inbox.close(): it names the inbox and
// nothing else. The executor knows the inbox's pauses, its waiting read and its
// last message.
func closeOp(close string, inbox string) *state.GeneratorOpcode {
	return &state.GeneratorOpcode{
		Op:   enums.OpcodeSubscribeClose,
		ID:   close,
		Name: close,
		Opts: map[string]any{"inbox": inbox},
	}
}

// closeJobID is the id of the job that runs the step after a read that a close
// ended, as the queue stores it (hashed).
func (h *subscribeHarness) closeJobID(wait string) string {
	return queue.HashID(h.infra.ctx, fmt.Sprintf("%s-%s-wake", h.run.IdempotencyKey(), wait))
}

// cutOf is the cut a close op saved as its result: the last message of the
// inbox at the time of the close, empty if the inbox had none.
func (h *subscribeHarness) cutOf(close string) string {
	h.t.Helper()

	saved, ok := h.steps()[close]
	require.True(h.t, ok, "the record of %s is saved", close)

	var record struct {
		Cut *string `json:"cut"`
	}
	require.NoError(h.t, json.Unmarshal([]byte(saved), &record))
	if record.Cut == nil {
		return ""
	}

	return *record.Cut
}

// Closing deletes the pauses of every source of the inbox, and saves the last
// message the inbox had as the op's cut.
func TestSubscribeCloseDeletesTheInboxPausesAndSavesTheCut(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/a", "test/b"))
	require.NotNil(t, h.sourcePause("inbox", "test/a"))
	require.NotNil(t, h.sourcePause("inbox", "test/b"))

	first := newEventID()
	require.NoError(t, h.deliverID(first, event.Event{Name: "test/a", Data: map[string]any{"n": 1}}))
	last := newEventID()
	require.NoError(t, h.deliverID(last, event.Event{Name: "test/b", Data: map[string]any{"n": 2}}))

	h.report(closeOp("inbox:close", "inbox"))

	require.Nil(t, h.sourcePause("inbox", "test/a"), "every source pause is deleted")
	require.Nil(t, h.sourcePause("inbox", "test/b"), "every source pause is deleted")
	require.Equal(t, messageStepID("inbox", last), h.cutOf("inbox:close"), "the cut is the last message")

	after := newEventID()
	require.NoError(t, h.deliverID(after, event.Event{Name: "test/a", Data: map[string]any{"n": 3}}))
	require.NotContains(t, h.steps(), messageStepID("inbox", after), "nothing arrives through a closed inbox")
}

// A close of an inbox that received nothing has no cut.
func TestSubscribeCloseOfAnEmptyInboxHasNoCut(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/a"))

	h.report(closeOp("inbox:close", "inbox"))

	require.Equal(t, `{"cut":null}`, h.steps()["inbox:close"])
}

// An arrival that matched a pause before the close deleted it is dropped when
// it reaches state after the close: the inbox is closed.
func TestSubscribeMessageAfterCloseIsDropped(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/a"))
	h.report(closeOp("inbox:close", "inbox"))

	id := newEventID()
	_, err := h.infra.smv2.SaveInboxMessage(h.infra.ctx, h.run.ID, statev2.InboxMessage{
		StepID: messageStepID("inbox", id),
		Inbox:  "inbox",
		Value:  []byte(`{"data":{"name":"test/a"},"inngest/inbox":"inbox"}`),
	})
	require.ErrorIs(t, err, statev2.ErrInboxClosed)
	require.NotContains(t, h.steps(), messageStepID("inbox", id))
}

// An arrival for a run that has ended is dropped and does not bring its keys
// back.
func TestSubscribeMessageAfterRunEndIsDropped(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/a"))
	require.NoError(t, h.infra.smv2.Delete(h.infra.ctx, h.run.ID))

	id := newEventID()
	_, err := h.infra.smv2.SaveInboxMessage(h.infra.ctx, h.run.ID, statev2.InboxMessage{
		StepID: messageStepID("inbox", id),
		Inbox:  "inbox",
		Value:  []byte(`{"data":{"name":"test/a"},"inngest/inbox":"inbox"}`),
	})
	require.ErrorIs(t, err, state.ErrRunNotFound)

	exists, err := h.infra.smv2.Exists(h.infra.ctx, h.run.ID)
	require.NoError(t, err)
	require.False(t, exists)
}

// A read that is waiting when its inbox closes loses its record and its timer,
// and the run is woken once with a job named by the read and the close.
func TestSubscribeCloseEndsWaitingReads(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitTimeoutOp("next-1", "inbox", "inbox", "1m"))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.timerJobID("next-1")), "the read armed a timer")

	h.report(closeOp("inbox:close", "inbox"))

	require.Equal(t, 1, timesIn(h.jobIDs(), h.closeJobID("next-1")), "the run was woken for the read")
	require.Zero(t, timesIn(h.jobIDs(), h.timerJobID("next-1")), "the read's timer is gone")
	require.NotContains(t, h.steps(), "next-1", "nothing is saved under the read")
}

// Closing is repaired by the same op reported again: a crash after the close is
// saved but before the wake is queued loses nothing.
func TestSubscribeCloseCrashBeforeWakeIsRepaired(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))

	op := closeOp("inbox:close", "inbox")

	h.crash.arm("after:CloseInbox")
	h.driver.response = &state.DriverResponse{StatusCode: 206, RequestVersion: 2, Generator: []*state.GeneratorOpcode{op}}
	require.Error(t, h.tryExecute(), "the injected crash fails the request")
	require.Zero(t, timesIn(h.jobIDs(), h.closeJobID("next-1")), "the wake was not queued")

	// The queue runs the request again.
	h.execute()
	require.Equal(t, 1, timesIn(h.jobIDs(), h.closeJobID("next-1")), "the redelivery queued the wake")
	require.Contains(t, h.steps(), "inbox:close")
}

// Closing with a read in the same response: whichever of the two the executor
// handles first, the read is ended and the run is woken.
func TestSubscribeCloseBeforeTheReadItEnds(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))

	h.report(closeOp("inbox:close", "inbox"))
	require.Zero(t, timesIn(h.jobIDs(), h.closeJobID("next-1")), "no read was waiting yet")

	// The read is recorded after the close was handled.
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.closeJobID("next-1")), "a read reported after its close is ended at once")
}

// A read beside a step is in the pending set. The close takes it out in the
// same call that saves the close, so the step's own completion finds nothing
// pending and the run is woken by the close.
func TestSubscribeCloseTakesReadsOutOfThePendingSet(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(plannedOp("work"), subscribeWaitOp("next-1", "inbox", "inbox"))
	// The read's timer leaves the queue when the read is satisfied or closed.
	queued := len(h.jobIDs()) - 1

	h.report(closeOp("inbox:close", "inbox"))
	require.Equal(t, queued, len(h.jobIDs()), "the step is still pending, so the close queued no discovery")

	hasPending, err := h.infra.smv2.SaveStep(h.infra.ctx, h.run.ID, "work", []byte(`{"data":"done"}`))
	require.NoError(t, err)
	require.False(t, hasPending, "the ended read left the pending set")
}

// A close with no read waiting still wakes the run: the code after await
// close() has to continue.
func TestSubscribeCloseWithNoReadsWakesTheRun(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	before := len(h.jobIDs())

	h.report(closeOp("inbox:close", "inbox"))
	require.Equal(t, before+1, len(h.jobIDs()), "one job continues the run")
}
