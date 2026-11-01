package executor

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// subscribeOp is the op an SDK reports for step.subscribe(inbox, sources).
func subscribeOp(inbox string, eventNames ...string) *state.GeneratorOpcode {
	sources := []map[string]any{}
	for _, name := range eventNames {
		sources = append(sources, map[string]any{"event": name})
	}

	return &state.GeneratorOpcode{
		Op:   enums.OpcodeSubscribe,
		ID:   inbox,
		Name: inbox,
		Opts: map[string]any{"sources": sources},
	}
}

// subscribeWaitOp is the op an SDK reports for a read with nothing unread.
// seen is the last step id of the stack the SDK loaded.
func subscribeWaitOp(wait string, inbox string, seen string) *state.GeneratorOpcode {
	return &state.GeneratorOpcode{
		Op:   enums.OpcodeSubscribeWait,
		ID:   wait,
		Name: wait,
		Opts: map[string]any{"inbox": inbox, "seen": seen},
	}
}

// sourcePause loads the pause the executor wrote for an inbox's source.
func (h *subscribeHarness) sourcePause(inbox string, eventName string) *state.Pause {
	h.t.Helper()

	pauseID := inngest.DeterministicSha1UUID(h.run.ID.RunID.String() + inbox + eventName + "")
	p, err := h.infra.pauseMgr.PauseByID(h.infra.ctx, pauses.Index{
		WorkspaceID: h.run.ID.Tenant.EnvID,
		EventName:   eventName,
	}, pauseID)
	if err != nil {
		return nil
	}

	return p
}

func messageStepID(inbox string, eventID ulid.ULID) string {
	sum := sha1.Sum([]byte(inbox + eventID.String()))

	return hex.EncodeToString(sum[:])
}

// wakeJobID is the id of the job that runs the step after a read is woken by
// a message, as the queue stores it (hashed). A read has one such job whichever
// message woke it.
func (h *subscribeHarness) wakeJobID(wait string, _ string) string {
	return queue.HashID(h.infra.ctx, fmt.Sprintf("%s-%s-message", h.run.IdempotencyKey(), wait))
}

func newEventID() ulid.ULID {
	return ulid.MustNew(ulid.Now(), rand.Reader)
}

func TestSubscribeOpenWritesSourcePausesAndRecord(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/a", "test/b"))

	for _, name := range []string{"test/a", "test/b"} {
		pause := h.sourcePause("inbox", name)
		require.NotNil(t, pause, "a pause for source %s", name)
		require.Greater(t, time.Until(pause.Expires.Time()), 300*24*time.Hour, "a source outlives any wait")
		require.LessOrEqual(t, time.Until(pause.Expires.Time()), consts.MaxWaitForEventTimeout)
	}

	require.Contains(t, h.steps(), "inbox", "the open record is a step")
}

// A message that arrives while a read waits is saved as its own step and wakes
// the read once; a second message finds no read waiting and wakes nothing.
func TestSubscribeArrivalWakesWait(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))

	pause := h.sourcePause("inbox", "test/msg")
	require.NotNil(t, pause)

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))

	msg := messageStepID("inbox", id)
	saved := h.steps()
	require.Contains(t, saved, msg)

	var value struct {
		Data  map[string]any `json:"data"`
		Inbox string         `json:"inngest/inbox"`
		Src   []string       `json:"src"`
	}
	require.NoError(t, json.Unmarshal([]byte(saved[msg]), &value))
	require.Equal(t, "test/msg", value.Data["name"])
	require.Equal(t, "inbox", value.Inbox)
	require.Empty(t, value.Src, "a message does not record the pauses it came through")

	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-1", msg)), "the read was woken once")
	require.NotContains(t, saved, "next-1", "nothing is saved under the read")

	jobs := h.jobIDs()
	second := newEventID()
	require.NoError(t, h.deliverID(second, event.Event{Name: "test/msg", Data: map[string]any{"n": 2}}))

	require.Contains(t, h.steps(), messageStepID("inbox", second), "the message is saved whether or not a read waits")
	require.Equal(t, len(jobs), len(h.jobIDs()), "no read was waiting, so nothing was queued")
}

// A message saved after the SDK loaded state and before the wait was recorded
// is found by the wait's scan of the stack and wakes the read.
func TestSubscribeMessageBeforeWaitWakesTheRead(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	msg := messageStepID("inbox", id)
	require.Contains(t, h.steps(), msg)

	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))

	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-1", msg)), "recording the wait woke the read")

	// Reporting the same wait again (the SDK does on each request) finds the
	// same message and queues the same job.
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-1", msg)))
}

// A message the SDK already saw (it is at or before "seen") does not wake.
func TestSubscribeSeenMessageDoesNotWake(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	msg := messageStepID("inbox", id)

	h.report(subscribeWaitOp("next-1", "inbox", msg))

	require.Zero(t, timesIn(h.jobIDs(), h.wakeJobID("next-1", msg)))
}

// A crash after the message is saved and before the read is woken is repaired
// by the event being delivered again: W outlives the wake.
func TestSubscribeCrashAfterMessageSave(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))

	id := newEventID()
	msg := messageStepID("inbox", id)
	evt := event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}

	h.crash.arm("after:SaveInboxMessage")
	_ = h.deliverID(id, evt)

	saved := h.steps()
	require.Contains(t, saved, msg, "the message was saved before the crash")
	require.Zero(t, timesIn(h.jobIDs(), h.wakeJobID("next-1", msg)), "the crash came before the wake")

	require.NoError(t, h.deliverID(id, evt))

	require.Equal(t, saved, h.steps(), "the redelivery saved the same message")
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-1", msg)), "the redelivery woke the read once")
}

// Two messages that both find a read still recorded (the first one's delete
// of it has not happened yet) wake it once: the read has one wake job, not one
// per message, so a burst does not start several replays of the run at once.
func TestSubscribeTwoMessagesWakeAReadOnce(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/msg"))
	h.report(subscribeWaitOp("next-1", "inbox", "inbox"))

	first := newEventID()
	second := newEventID()

	// The first message queues the wake and then crashes before the read is
	// deleted, as a second arrival running at the same time would find it.
	before := len(h.jobIDs())
	h.crash.arm("after:Enqueue")
	_ = h.deliverID(first, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}})
	require.Equal(t, before+1, len(h.jobIDs()), "the first message queued the wake")

	queued := map[string]bool{}
	for _, id := range h.jobIDs() {
		queued[id] = true
	}

	require.NoError(t, h.deliverID(second, event.Event{Name: "test/msg", Data: map[string]any{"n": 2}}))

	for _, id := range h.jobIDs() {
		require.True(t, queued[id], "the second message queued no job of its own, got %s", id)
	}
}

// An event that matches sources of two inboxes is saved once per inbox, and
// wakes the read waiting on each.
func TestSubscribeEventInTwoInboxesIsSavedOncePerInbox(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("one", "test/msg"), subscribeOp("two", "test/msg"))
	h.report(subscribeWaitOp("next-one", "one", "one"))
	h.report(subscribeWaitOp("next-two", "two", "two"))

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))

	saved := h.steps()
	for _, inbox := range []string{"one", "two"} {
		msg := messageStepID(inbox, id)
		require.Contains(t, saved, msg, "the event is a message of %s", inbox)
		require.Contains(t, saved[msg], `"inngest/inbox":"`+inbox+`"`)
	}

	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-one", "")), "the read of the first inbox was woken")
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-two", "")), "the read of the second inbox was woken")
}
