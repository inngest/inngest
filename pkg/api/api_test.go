package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/require"
)

// postEvent drives ReceiveEvent directly (white-box) with the chi "key" route
// param injected, capturing the event the stub handler receives. This exercises
// the real ingest path — including the ResolveSessions() call before Validate —
// without standing up the full router/config.
func postEvent(t *testing.T, body string) (*httptest.ResponseRecorder, *event.Event) {
	t.Helper()

	var got *event.Event
	a := API{
		handler: func(_ context.Context, e *event.Event, _ *event.SeededID) (string, error) {
			got = e
			return "01HZTESTEVENTID", nil
		},
		log: logger.StdlibLogger(t.Context()),
	}

	req := httptest.NewRequest(http.MethodPost, "/e/test-key", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	// ReceiveEvent reads the event key from the chi route param.
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "test-key")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	a.ReceiveEvent(w, req)
	return w, got
}

// TestReceiveEvent_ResolvesPropagatedSessions proves the ingest path folds the
// propagated session layer into meta.sessions (manual wins per key, disjoint
// keys fill) and clears the propagated layer before the event reaches the
// handler
func TestReceiveEvent_ResolvesPropagatedSessions(t *testing.T) {
	w, got := postEvent(t, `{
		"name": "test/session-prop",
		"data": {},
		"meta": {
			"sessions": {"conv_id": "manual"},
			"propagatedSessions": {"conv_id": "propagated", "org_id": "42"}
		}
	}`)

	require.Equal(t, 200, w.Code, w.Body.String())
	require.NotNil(t, got)
	require.Equal(t, event.Sessions{"conv_id": "manual", "org_id": "42"}, got.Meta.Sessions)
	require.Nil(t, got.Meta.PropagatedSessions, "propagated layer must not persist past ingest")
}

// postInvoke drives Invoke directly with the chi "slug" route param
// injected, capturing the invocation event the stub handler receives.
func postInvoke(t *testing.T, body string) (*httptest.ResponseRecorder, *event.Event) {
	t.Helper()

	var got *event.Event
	a := API{
		handler: func(_ context.Context, e *event.Event, _ *event.SeededID) (string, error) {
			got = e
			return "01HZTESTEVENTID", nil
		},
		log: logger.StdlibLogger(t.Context()),
	}

	req := httptest.NewRequest(http.MethodPost, "/invoke/my-fn", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	// Invoke reads the target function slug from the chi route param.
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", "my-fn")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	a.Invoke(w, req)
	return w, got
}

// TestInvoke_ResolvesPropagatedSessions proves the HTTP invoke path folds the
// propagated session layer into meta.sessions (manual wins per key, disjoint
// keys fill), applies null tombstones, and clears the propagated layer before
// the invocation event reaches the handler — mirroring ReceiveEvent, which this
// entrypoint bypasses.
func TestInvoke_ResolvesPropagatedSessions(t *testing.T) {
	w, got := postInvoke(t, `{
		"data": {},
		"meta": {
			"sessions": {"conv_id": "manual", "cut_me": null},
			"propagatedSessions": {"conv_id": "propagated", "org_id": "42", "cut_me": "inherited"}
		}
	}`)

	require.Equal(t, 200, w.Code, w.Body.String())
	require.NotNil(t, got)
	require.Equal(t, event.Sessions{"conv_id": "manual", "org_id": "42"}, got.Meta.Sessions)
	require.Nil(t, got.Meta.PropagatedSessions, "propagated layer must not persist past invoke")
	require.Equal(t, event.InvokeFnName, got.Name, "invoke path rewrites the event name")
}

// TestInvoke_ValidatesAfterResolve proves the HTTP invoke path validates the
// merged sessions and rejects an oversized manual layer, mirroring the
// ReceiveEvent behaviour this entrypoint bypasses.
func TestInvoke_ValidatesAfterResolve(t *testing.T) {
	// 6 manual keys > MaxEventSessions (5); all survive the merge.
	w, got := postInvoke(t, `{
		"data": {},
		"meta": {
			"sessions": {"a":"1","b":"1","c":"1","d":"1","e":"1","f":"1"}
		}
	}`)

	require.Equal(t, 400, w.Code, w.Body.String())
	require.Nil(t, got, "an invalid invocation event must never reach the handler")
}
