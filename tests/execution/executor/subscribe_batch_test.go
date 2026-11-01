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

// answerOf is the answer a batch read was saved with: the step it was answered
// after and the message step ids.
func (h *subscribeHarness) answerOf(wait string) (string, []string) {
	h.t.Helper()

	saved, ok := h.steps()[wait]
	require.True(h.t, ok, "the batch read %s has an answer", wait)

	var answer struct {
		Seen string   `json:"seen"`
		IDs  []string `json:"ids"`
	}
	require.NoError(h.t, json.Unmarshal([]byte(saved), &answer))

	return answer.Seen, answer.IDs
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

// A drain is answered at once with the messages saved after the step the SDK
// last saw, and wakes the run. It queues no timer: it never waits.
func TestSubscribeDrainIsAnsweredWithWhatArrivedSince(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	msgs := h.deliverN(2)

	h.report(subscribeBatchOp("drain-1", "inbox", "inbox", map[string]any{"drain": true}))

	seen, ids := h.answerOf("drain-1")
	require.Equal(t, "inbox", seen)
	require.Equal(t, msgs, ids, "the answer lists the messages in the order they were saved")
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("drain-1", "")), "the run was woken")
	require.Zero(t, timesIn(h.jobIDs(), h.timerJobID("drain-1")), "a drain queues no timer")
}

// With nothing saved since, a drain is answered with an empty list.
func TestSubscribeDrainWithNothingIsAnsweredEmpty(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))

	h.report(subscribeBatchOp("drain-1", "inbox", "inbox", map[string]any{"drain": true}))

	_, ids := h.answerOf("drain-1")
	require.Empty(t, ids)
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("drain-1", "")))
}

// Messages the SDK saw already are not in the answer, and the room bounds it,
// oldest first.
func TestSubscribeBatchAnswerStartsAfterSeenAndFitsTheRoom(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	msgs := h.deliverN(3)

	h.report(subscribeBatchOp("drain-1", "inbox", msgs[0], map[string]any{"drain": true, "room": 1}))

	seen, ids := h.answerOf("drain-1")
	require.Equal(t, msgs[0], seen)
	require.Equal(t, []string{msgs[1]}, ids)
}

// A nextBatch with nothing unread waits like a read: it is recorded with a
// timer, and the first message answers it with the messages saved since.
func TestSubscribeNextBatchWaitsAndTheFirstMessageAnswersIt(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))

	h.report(subscribeBatchOp("batch-1", "inbox", "inbox", map[string]any{}))

	require.NotContains(t, h.steps(), "batch-1", "nothing is unread, so the read waits")
	require.Equal(t, 1, timesIn(h.jobIDs(), h.timerJobID("batch-1")), "the read has a timer")

	msgs := h.deliverN(1)

	_, ids := h.answerOf("batch-1")
	require.Equal(t, msgs, ids)
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("batch-1", "")))

	// Later messages are not added to the answer.
	h.deliverN(1)
	_, ids = h.answerOf("batch-1")
	require.Equal(t, msgs, ids)
}

// A nextBatch that finds messages saved since is answered at once, and one
// whose SDK holds unread messages is answered at once even with none.
func TestSubscribeNextBatchIsAnsweredAtOnceWhenSomethingIsUnread(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	msgs := h.deliverN(1)

	h.report(subscribeBatchOp("batch-1", "inbox", "inbox", map[string]any{}))
	_, ids := h.answerOf("batch-1")
	require.Equal(t, msgs, ids, "found in the gap")

	h.report(subscribeBatchOp("batch-2", "inbox", msgs[0], map[string]any{"unread": true, "room": 0}))
	_, ids = h.answerOf("batch-2")
	require.Empty(t, ids, "the SDK holds unread messages, so it does not wait")
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
