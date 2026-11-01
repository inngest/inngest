package executor

import (
	"testing"

	"github.com/inngest/inngest/pkg/event"
	"github.com/stretchr/testify/require"
)

// sourceIfOp is subscribeOp for one source with an if expression.
func sourceIfOp(inbox string, eventName string, expr string) map[string]any {
	return map[string]any{"event": eventName, "if": expr}
}

// A source's if can reference the run's trigger event: it matches an arrival
// with the trigger's value and not another value.
func TestSubscribeSourceIfReferencesTriggerEvent(t *testing.T) {
	op := subscribeOp("inbox")
	op.Opts = map[string]any{"sources": []map[string]any{
		sourceIfOp("inbox", "test/msg", "async.data.n == event.data.n"),
	}}

	h := newSubscribeHarnessTriggered(t, map[string]any{"n": 1}, nil, op)

	miss := newEventID()
	require.NoError(t, h.deliverID(miss, event.Event{Name: "test/msg", Data: map[string]any{"n": 2}}))
	require.NotContains(t, h.steps(), messageStepID("inbox", miss), "another value does not match")

	hit := newEventID()
	require.NoError(t, h.deliverID(hit, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	require.Contains(t, h.steps(), messageStepID("inbox", hit), "the trigger's value matches")
}

// The same holds for a source added by listen.
func TestSubscribeListenIfReferencesTriggerEvent(t *testing.T) {
	h := newSubscribeHarnessTriggered(t, map[string]any{"n": 1}, nil, subscribeOp("inbox", "test/other"))

	op := listenOp("inbox:listen", "inbox")
	op.Opts.(map[string]any)["sources"] = []map[string]any{
		sourceIfOp("inbox", "test/msg", "async.data.n == event.data.n"),
	}
	h.report(op)

	miss := newEventID()
	require.NoError(t, h.deliverID(miss, event.Event{Name: "test/msg", Data: map[string]any{"n": 2}}))
	require.NotContains(t, h.steps(), messageStepID("inbox", miss))

	hit := newEventID()
	require.NoError(t, h.deliverID(hit, event.Event{Name: "test/msg", Data: map[string]any{"n": 1}}))
	require.Contains(t, h.steps(), messageStepID("inbox", hit))
}
