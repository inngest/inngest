package executor

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

// closeOp is the op an SDK reports for handle.close(): the unread ids it
// computed, the source pauses to delete, and the reads that were waiting under
// the handle.
func closeOp(close string, inbox string, ids []string, pauses []map[string]any, waits ...string) *state.GeneratorOpcode {
	return closeRowsOp(close, inbox, nil, ids, pauses, waits...)
}

// closeRowsOp is closeOp for a handle whose rows are the open and listen ops
// it closes.
func closeRowsOp(close string, inbox string, rows []string, ids []string, pauses []map[string]any, waits ...string) *state.GeneratorOpcode {
	if rows == nil {
		rows = []string{}
	}
	if ids == nil {
		ids = []string{}
	}
	if pauses == nil {
		pauses = []map[string]any{}
	}
	if waits == nil {
		waits = []string{}
	}

	return &state.GeneratorOpcode{
		Op:   enums.OpcodeSubscribeClose,
		ID:   close,
		Name: close,
		Opts: map[string]any{"inbox": inbox, "ids": ids, "pauses": pauses, "waits": waits, "rows": rows},
	}
}

// closeJobID is the id of the job that runs the step after a read that a close
// ended, as the queue stores it (hashed).
func (h *subscribeHarness) closeJobID(wait string) string {
	return queue.HashID(h.infra.ctx, fmt.Sprintf("%s-%s-close", h.run.IdempotencyKey(), wait))
}

// closedIDs is the ids a close op saved as its result.
func (h *subscribeHarness) closedIDs(close string) []string {
	h.t.Helper()

	saved, ok := h.steps()[close]
	require.True(h.t, ok, "the record of %s is saved", close)

	var record struct {
		Data []string `json:"data"`
	}
	require.NoError(h.t, json.Unmarshal([]byte(saved), &record))

	return record.Data
}

// Closing a view deletes the pauses it names and saves the ids the SDK gave as
// the op's result. The inbox's own source keeps receiving.
func TestSubscribeCloseDeletesItsPausesAndSavesTheIds(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/a"))
	h.report(listenOp("inbox:listen", "inbox", "test/b"))

	listen := h.listenPause("inbox:listen", "test/b")
	require.NotNil(t, listen)

	a1 := newEventID()
	require.NoError(t, h.deliverID(a1, event.Event{Name: "test/a", Data: map[string]any{"n": 1}}))

	h.report(closeOp("inbox:listen:close", "inbox", []string{"one", "two"}, []map[string]any{
		{"id": listen.ID.String(), "event": "test/b"},
	}))

	require.Nil(t, h.listenPause("inbox:listen", "test/b"), "the view's pause is deleted")
	require.NotNil(t, h.sourcePause("inbox", "test/a"), "the inbox's own pause stays")
	require.Equal(t, []string{"one", "two"}, h.closedIDs("inbox:listen:close"), "the SDK's ids are the result")

	b := newEventID()
	require.NoError(t, h.deliverID(b, event.Event{Name: "test/b", Data: map[string]any{"n": 2}}))
	require.NotContains(t, h.steps(), messageStepID("inbox", b), "nothing arrives through a closed view's source")

	a2 := newEventID()
	require.NoError(t, h.deliverID(a2, event.Event{Name: "test/a", Data: map[string]any{"n": 3}}))
	require.Contains(t, h.steps(), messageStepID("inbox", a2), "the inbox still receives its own source")
}

// A message that lands before the close is handled is not in the ids the SDK
// computed, and the executor does not add it.
func TestSubscribeCloseIdsAreTheSDKs(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/a"))
	pause := h.sourcePause("inbox", "test/a")
	require.NotNil(t, pause)

	first := newEventID()
	require.NoError(t, h.deliverID(first, event.Event{Name: "test/a", Data: map[string]any{"n": 1}}))
	computed := []string{messageStepID("inbox", first)}

	late := newEventID()
	require.NoError(t, h.deliverID(late, event.Event{Name: "test/a", Data: map[string]any{"n": 2}}))

	h.report(closeOp("inbox:close", "inbox", computed, []map[string]any{{"id": pause.ID.String(), "event": "test/a"}}))

	require.Contains(t, h.steps(), messageStepID("inbox", late), "the late message is in state")
	require.Equal(t, computed, h.closedIDs("inbox:close"), "but it is not in the ids")
	require.Nil(t, h.sourcePause("inbox", "test/a"))
}

// A read that is waiting when its handle closes loses its record and its timer,
// and the run is woken once with a job named by the read and the close.
func TestSubscribeCloseEndsWaitingReads(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	pause := h.sourcePause("inbox", "test/msg")
	h.report(subscribeWaitTimeoutOp("next-1", "inbox", "inbox", "1m"))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.timerJobID("next-1")), "the read armed a timer")

	h.report(closeOp("inbox:close", "inbox", nil, []map[string]any{{"id": pause.ID.String(), "event": "test/msg"}}, "next-1"))

	require.Equal(t, 1, timesIn(h.jobIDs(), h.closeJobID("next-1")), "the run was woken for the read")
	require.Zero(t, timesIn(h.jobIDs(), h.timerJobID("next-1")), "the read's timer is gone")
	require.NotContains(t, h.steps(), "next-1", "nothing is saved under the read")
}

// Closing is repaired by the same op reported again: a crash after the close is
// saved but before the wake is queued loses nothing.
func TestSubscribeCloseCrashBeforeWakeIsRepaired(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	pause := h.sourcePause("inbox", "test/msg")
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))

	op := closeOp("inbox:close", "inbox", nil, []map[string]any{{"id": pause.ID.String(), "event": "test/msg"}}, "next-1")

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
	pause := h.sourcePause("inbox", "test/msg")

	h.report(closeOp("inbox:close", "inbox", nil, []map[string]any{{"id": pause.ID.String(), "event": "test/msg"}}, "next-1"))
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
	pause := h.sourcePause("inbox", "test/msg")
	h.report(plannedOp("work"), subscribeWaitOp("next-1", "inbox", "inbox"))
	queued := len(h.jobIDs())

	h.report(closeOp("inbox:close", "inbox", nil, []map[string]any{{"id": pause.ID.String(), "event": "test/msg"}}, "next-1"))
	require.Equal(t, queued, len(h.jobIDs()), "the step is still pending, so the close queued no discovery")

	hasPending, err := h.infra.smv2.SaveStep(h.infra.ctx, h.run.ID, "work", []byte(`{"data":"done"}`))
	require.NoError(t, err)
	require.False(t, hasPending, "the ended read left the pending set")
}

// A close with no read waiting still wakes the run: the code after await
// close() has to continue.
func TestSubscribeCloseWithNoReadsWakesTheRun(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	pause := h.sourcePause("inbox", "test/msg")
	before := len(h.jobIDs())

	h.report(closeOp("inbox:close", "inbox", nil, []map[string]any{{"id": pause.ID.String(), "event": "test/msg"}}))
	require.Equal(t, before+1, len(h.jobIDs()), "one job continues the run")
}
