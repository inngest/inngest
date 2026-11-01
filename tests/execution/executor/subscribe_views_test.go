package executor

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/require"
)

// subscribeWaitWantOp is the op an SDK reports for a read of a view: a read of
// an inbox that has nothing unread, with the filter of the view.
func subscribeWaitWantOp(wait string, inbox string, seen string, want map[string]any) *state.GeneratorOpcode {
	return &state.GeneratorOpcode{
		Op:   enums.OpcodeSubscribeWait,
		ID:   wait,
		Name: wait,
		Opts: map[string]any{"inbox": inbox, "seen": seen, "want": want},
	}
}

// listenOp is the op an SDK reports for inbox.listen(sources): the sources of
// a view, opened on an inbox that is already open.
func listenOp(listen string, inbox string, eventNames ...string) *state.GeneratorOpcode {
	op := subscribeOp(listen, eventNames...)
	op.Opts.(map[string]any)["inbox"] = inbox

	return op
}

// listenPause loads the pause the executor wrote for a source of a listen op.
func (h *subscribeHarness) listenPause(listen string, eventName string) *state.Pause {
	h.t.Helper()

	return h.sourcePause(listen, eventName)
}

// pausesOf is the pause ids a listen op's record holds.
func (h *subscribeHarness) pausesOf(op string) []string {
	h.t.Helper()

	var record struct {
		Data struct {
			Pauses []string `json:"pauses"`
		} `json:"data"`
	}
	saved, ok := h.steps()[op]
	require.True(h.t, ok, "the record of %s is saved", op)
	require.NoError(h.t, json.Unmarshal([]byte(saved), &record))

	return record.Data.Pauses
}

// A read that wants one event type is woken by that type and not by another.
func TestSubscribeWaitWantsEventNames(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/a", "test/b"))
	h.report(subscribeWaitWantOp("next-b", "inbox", "inbox", map[string]any{"events": []string{"test/b"}}))

	a := newEventID()
	require.NoError(t, h.deliverID(a, event.Event{Name: "test/a", Data: map[string]any{"n": 1}}))
	require.Contains(t, h.steps(), messageStepID("inbox", a), "the message is saved whoever wants it")
	require.Zero(t, timesIn(h.jobIDs(), h.wakeJobID("next-b", messageStepID("inbox", a))), "a message of another type does not wake the read")

	b := newEventID()
	require.NoError(t, h.deliverID(b, event.Event{Name: "test/b", Data: map[string]any{"n": 2}}))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-b", messageStepID("inbox", b))), "a message of the type wakes the read")
}

// Two reads of one inbox wait for different types; a message wakes only the
// read that wants it, and the other keeps waiting.
func TestSubscribeTwoViewsWaitInParallel(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/a", "test/b"))
	h.report(subscribeWaitWantOp("next-a", "inbox", "inbox", map[string]any{"events": []string{"test/a"}}))
	h.report(subscribeWaitWantOp("next-b", "inbox", "inbox", map[string]any{"events": []string{"test/b"}}))

	b := newEventID()
	require.NoError(t, h.deliverID(b, event.Event{Name: "test/b", Data: map[string]any{"n": 1}}))
	require.Zero(t, timesIn(h.jobIDs(), h.wakeJobID("next-a", messageStepID("inbox", b))), "the view of a is not woken by b")

	a := newEventID()
	require.NoError(t, h.deliverID(a, event.Event{Name: "test/a", Data: map[string]any{"n": 2}}))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-a", messageStepID("inbox", a))), "the view of a was still waiting")
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-b", messageStepID("inbox", b))), "so was the view of b, and each message woke only its own")
}

// A wait recorded after messages it does not want finds nothing in its scan.
func TestSubscribeRecordedWaitSkipsMessagesItDoesNotWant(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/a", "test/b"))

	a := newEventID()
	require.NoError(t, h.deliverID(a, event.Event{Name: "test/a", Data: map[string]any{"n": 1}}))
	b := newEventID()
	require.NoError(t, h.deliverID(b, event.Event{Name: "test/b", Data: map[string]any{"n": 2}}))

	h.report(subscribeWaitWantOp("next-b", "inbox", "inbox", map[string]any{"events": []string{"test/b"}}))

	// A read has one wake job whichever message woke it, so the scan woke the
	// read once: it passed the message of the other type and found the one it
	// wants.
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-b", messageStepID("inbox", b))), "the scan found the message it wants")
}

// Listening on an open inbox writes a pause for each source, keyed by the
// listen op, saves a record that names them, and counts as one step.
func TestSubscribeListenWritesItsOwnPauses(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/a"))
	before := len(h.steps())

	h.report(listenOp("inbox:listen", "inbox", "test/a"))

	root := h.sourcePause("inbox", "test/a")
	listen := h.listenPause("inbox:listen", "test/a")
	require.NotNil(t, root)
	require.NotNil(t, listen)
	require.NotEqual(t, root.ID, listen.ID, "a view's source is a pause of its own")
	require.Equal(t, "inbox", listen.Subscribe.Inbox, "the pause belongs to the inbox, not to the view")
	require.Equal(t, []string{listen.ID.String()}, h.pausesOf("inbox:listen"))
	require.Equal(t, before+1, len(h.steps()), "listen is one step")
}

// An event that matches a source of the inbox and a source of a view is one
// message listing both.
func TestSubscribeListenAndInboxSourceMakeOneMessage(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/a"))
	h.report(listenOp("inbox:listen", "inbox", "test/a"))

	root := h.sourcePause("inbox", "test/a")
	listen := h.listenPause("inbox:listen", "test/a")

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/a", Data: map[string]any{"n": 1}}))

	var value struct {
		Src []string `json:"src"`
	}
	saved := h.steps()[messageStepID("inbox", id)]
	require.NoError(t, json.Unmarshal([]byte(saved), &value))
	require.ElementsMatch(t, []string{root.ID.String(), listen.ID.String()}, value.Src)
}

// A read of a view over a listen op's sources is woken by messages that came
// through those sources and not by messages that came through others.
func TestSubscribeWaitWantsSources(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/a"))
	h.report(listenOp("inbox:listen", "inbox", "test/b"))

	listen := h.listenPause("inbox:listen", "test/b")
	require.NotNil(t, listen)

	h.report(subscribeWaitWantOp("next-view", "inbox", "inbox:listen", map[string]any{"src": []string{listen.ID.String()}}))

	a := newEventID()
	require.NoError(t, h.deliverID(a, event.Event{Name: "test/a", Data: map[string]any{"n": 1}}))
	require.Zero(t, timesIn(h.jobIDs(), h.wakeJobID("next-view", messageStepID("inbox", a))), "a message from the inbox's own source does not wake the view")

	b := newEventID()
	require.NoError(t, h.deliverID(b, event.Event{Name: "test/b", Data: map[string]any{"n": 2}}))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-view", messageStepID("inbox", b))))
}

// A filter with both parts wants a message that passes both: an of() view
// under a listen() view.
func TestSubscribeWaitWantsSourcesAndEventNames(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/other"))
	h.report(listenOp("inbox:listen", "inbox", "test/a", "test/b"))

	a := h.listenPause("inbox:listen", "test/a")
	require.NotNil(t, a)
	b := h.listenPause("inbox:listen", "test/b")
	require.NotNil(t, b)

	want := map[string]any{"src": []string{a.ID.String(), b.ID.String()}, "events": []string{"test/b"}}
	h.report(subscribeWaitWantOp("next-nested", "inbox", "inbox:listen", want))

	for _, name := range []string{"test/a", "test/other"} {
		id := newEventID()
		require.NoError(t, h.deliverID(id, event.Event{Name: name, Data: map[string]any{"n": 1}}))
		require.Zero(t, timesIn(h.jobIDs(), h.wakeJobID("next-nested", messageStepID("inbox", id))), "%s fails one part of the filter", name)
	}

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/b", Data: map[string]any{"n": 2}}))
	require.Equal(t, 1, timesIn(h.jobIDs(), h.wakeJobID("next-nested", messageStepID("inbox", id))))
}

// Two listen ops on one event with different expressions: an event that passes
// one expression is a message of that listen's pause only.
func TestSubscribeListensWithDifferentExpressions(t *testing.T) {
	h := newSubscribeHarness(t, subscribeOp("inbox", "test/other"))

	ifOne := `async.data.k == "one"`
	ifTwo := `async.data.k == "two"`
	h.report(
		&state.GeneratorOpcode{Op: enums.OpcodeSubscribe, ID: "inbox:listen", Name: "inbox:listen", Opts: map[string]any{
			"inbox":   "inbox",
			"sources": []map[string]any{{"event": "test/a", "if": ifOne}},
		}},
		&state.GeneratorOpcode{Op: enums.OpcodeSubscribe, ID: "inbox:listen:1", Name: "inbox:listen:1", Opts: map[string]any{
			"inbox":   "inbox",
			"sources": []map[string]any{{"event": "test/a", "if": ifTwo}},
		}},
	)

	pauseFor := func(listen string, expr string) *state.Pause {
		id := inngest.DeterministicSha1UUID(h.run.ID.RunID.String() + listen + "test/a" + expr)
		p, err := h.infra.pauseMgr.PauseByID(h.infra.ctx, pauses.Index{WorkspaceID: h.run.ID.Tenant.EnvID, EventName: "test/a"}, id)
		require.NoError(t, err)

		return p
	}
	one := pauseFor("inbox:listen", ifOne)
	two := pauseFor("inbox:listen:1", ifTwo)

	id := newEventID()
	require.NoError(t, h.deliverID(id, event.Event{Name: "test/a", Data: map[string]any{"k": "two"}}))

	var value struct {
		Src []string `json:"src"`
	}
	require.NoError(t, json.Unmarshal([]byte(h.steps()[messageStepID("inbox", id)]), &value))
	require.Equal(t, []string{two.ID.String()}, value.Src)
	require.NotContains(t, value.Src, one.ID.String())
}
