package event

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/require"
)

// TestNewInvocationEventResolveSessions verifies the step.invoke server path:
// the invocation payload carries two session layers (manual meta.sessions and
// SDK-stamped meta.propagatedSessions), NewInvocationEvent copies both onto the
// built event, and ResolveSessions (called by the executor before publishing)
// folds them together the same way API ingest does for step.sendEvent.
func TestNewInvocationEventResolveSessions(t *testing.T) {
	// Simulate the wire payload the SDK stamps for an invoke: a manual key plus a
	// propagated layer inherited from the parent run. Decoding through
	// event.Event exercises EventMeta.UnmarshalJSON (tombstone capture).
	raw := []byte(`{
		"name": "some/event",
		"data": {"foo": "bar"},
		"meta": {
			"sessions": {"conv_id": "manual", "cut_me": null},
			"propagatedSessions": {"conv_id": "parent", "org_id": "42", "cut_me": "inherited"}
		}
	}`)

	var payload Event
	require.NoError(t, json.Unmarshal(raw, &payload))

	evt := NewInvocationEvent(NewInvocationEventOpts{
		Event: payload,
		FnID:  "target-fn",
	})

	// The point of this test: NewInvocationEvent must carry BOTH un-merged
	// session layers (and the captured tombstone) through onto the built event
	// untouched — it must not merge or drop them. That passthrough is what lets
	// the executor resolve later; the merge itself is covered exhaustively by
	// TestResolveSessions / TestResolveSessionsTombstones, so we don't re-assert
	// it here beyond a smoke check.
	require.Equal(t, Sessions{"conv_id": "manual"}, evt.Event.Meta.Sessions,
		"manual layer preserved (the null tombstone is captured off-map, not stored)")
	require.Equal(t, Sessions{"conv_id": "parent", "org_id": "42", "cut_me": "inherited"},
		evt.Event.Meta.PropagatedSessions, "propagated layer preserved untouched")
	require.Equal(t, []string{"cut_me"}, evt.Event.Meta.sessionTombstones,
		"null tombstone captured during Event unmarshal and carried through")
	require.Equal(t, InvokeFnName, evt.Event.Name)
}

func TestEvent_SetInvokeSpanRef(t *testing.T) {
	ref := &meta.SpanReference{TraceParent: "00-00112233445566778899aabbccddeeff-0011223344556677-01"}
	evt := Event{
		Name: InvokeFnName,
		Data: map[string]any{
			consts.InngestEventDataPrefix: InngestMetadata{InvokeFnID: "app-fn"},
		},
	}

	require.True(t, evt.SetInvokeSpanRef(ref))

	md := evt.Data[consts.InngestEventDataPrefix].(InngestMetadata)
	require.Equal(t, ref, md.InvokeSpanRef)
	require.Equal(t, "app-fn", md.InvokeFnID, "unrelated fields must be preserved")
}

func TestEvent_SetInvokeSpanRef_ReturnsFalseWhenNoMetadata(t *testing.T) {
	evt := Event{Name: "user/thing.happened", Data: map[string]any{"foo": "bar"}}
	require.False(t, evt.SetInvokeSpanRef(&meta.SpanReference{}))

	nilData := Event{Name: "user/thing.happened"}
	require.False(t, nilData.SetInvokeSpanRef(&meta.SpanReference{}))
}
