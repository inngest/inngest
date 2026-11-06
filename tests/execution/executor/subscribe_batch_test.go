package executor

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

// subscribeBatchOp is the op an SDK reports for a batch read: drain() never
// waits, nextBatch() waits only if nothing at all is unread.
func subscribeBatchOp(wait string, inbox string, seen string, batch map[string]any) *state.GeneratorOpcode {
	op := subscribeWaitOp(wait, inbox, seen)
	op.Opts.(map[string]any)["batch"] = batch

	return op
}

// cutOfAnswer is the cut a batch read was saved with: the last message step
// it covers, empty when it covers none.
func (h *subscribeHarness) cutOfAnswer(wait string) string {
	h.t.Helper()

	saved, ok := h.steps()[wait]
	require.True(h.t, ok, "the batch read %s has an answer", wait)

	var answer struct {
		Cut *string `json:"cut"`
	}
	require.NoError(h.t, json.Unmarshal([]byte(saved), &answer))
	if answer.Cut == nil {
		return ""
	}

	return *answer.Cut
}

func (h *subscribeHarness) deliverN(n int) []string {
	h.t.Helper()

	ids := []string{}
	for i := 0; i < n; i++ {
		id := newEventID()
		require.NoError(h.t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": i}}))
		ids = append(ids, messageStepID("inbox", id))
	}

	return ids
}

// A drain is answered at once with a cut at the last message saved, and wakes
// the run. It queues no timer: it never waits.
func TestSubscribeDrainIsAnsweredWithACut(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	msgs := h.deliverN(2)

	h.report(subscribeBatchOp("drain-1", "inbox", "inbox", map[string]any{"drain": true}))

	require.Equal(t, msgs[1], h.cutOfAnswer("drain-1"))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("drain-1", "")), "the run was woken")
	require.Zero(t, timesIn(h.jobIDs(), h.timerJobID("drain-1")), "a drain queues no timer")
}

// With nothing saved, a drain is answered with no cut.
func TestSubscribeDrainWithNothingIsAnsweredEmpty(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))

	h.report(subscribeBatchOp("drain-1", "inbox", "inbox", map[string]any{"drain": true}))

	require.Equal(t, `{"cut":null}`, h.steps()["drain-1"])
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("drain-1", "")))
}

// The cut covers messages saved after the step the SDK last saw, and no more
// than max of them.
func TestSubscribeBatchCutStartsAfterSeenAndFitsMax(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	msgs := h.deliverN(3)

	h.report(subscribeBatchOp("drain-1", "inbox", msgs[0], map[string]any{"drain": true, "max": 1}))

	require.Equal(t, msgs[1], h.cutOfAnswer("drain-1"))
}

// A nextBatch with nothing unread waits like a read: it is recorded with a
// timer, and the first message answers it with a cut at that message.
func TestSubscribeNextBatchWaitsAndTheFirstMessageAnswersIt(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))

	h.report(subscribeBatchOp("batch-1", "inbox", "inbox", map[string]any{}))

	require.NotContains(t, h.steps(), "batch-1", "nothing is unread, so the read waits")
	require.Equal(t, 1, timesIn(h.jobIDs(), h.timerJobID("batch-1")), "the read has a timer")

	msgs := h.deliverN(1)

	require.Equal(t, msgs[0], h.cutOfAnswer("batch-1"))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("batch-1", "")))

	// Later messages do not move the cut.
	h.deliverN(1)
	require.Equal(t, msgs[0], h.cutOfAnswer("batch-1"))
}

// A nextBatch that finds messages saved since is answered at once, and one
// whose SDK holds unread messages is answered at once even with none saved
// since.
func TestSubscribeNextBatchIsAnsweredAtOnceWhenSomethingIsUnread(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	msgs := h.deliverN(1)

	h.report(subscribeBatchOp("batch-1", "inbox", "inbox", map[string]any{}))
	require.Equal(t, msgs[0], h.cutOfAnswer("batch-1"), "found in the gap")

	h.report(subscribeBatchOp("batch-2", "inbox", msgs[0], map[string]any{"unread": true}))
	require.Equal(t, msgs[0], h.cutOfAnswer("batch-2"), "the SDK holds unread messages, so it does not wait and the cut covers them")
}

// A timer and a message race for one nextBatch: the first write wins.
func TestSubscribeNextBatchTimerAndMessageFirstWriteWins(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	op := subscribeBatchOp("batch-1", "inbox", "inbox", map[string]any{})
	op.Opts.(map[string]any)["timeout"] = "1m"
	h.report(op)

	tm := h.timerOf("batch-1")
	require.NoError(t, h.runTimer(tm))
	require.Equal(t, "null", h.steps()["batch-1"], "the timer answered the read with a null")

	h.deliverN(1)
	require.Equal(t, "null", h.steps()["batch-1"], "a message does not overwrite it")
}
