package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/coreapi/apiutil"
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
			"propagated_sessions": {"conv_id": "propagated", "org_id": "42"}
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
			"propagated_sessions": {"conv_id": "propagated", "org_id": "42", "cut_me": "inherited"}
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

// postBatch drives ReceiveEvent with the given body and handler, returning the
// decoded response and the names of the events that reached the handler.
func postBatch(t *testing.T, body string, handler EventHandler) (*httptest.ResponseRecorder, apiutil.EventAPIResponse, []string) {
	t.Helper()

	var handled []string
	a := API{
		handler: func(ctx context.Context, e *event.Event, seed *event.SeededID) (string, error) {
			handled = append(handled, e.Name)
			return handler(ctx, e, seed)
		},
		log: logger.StdlibLogger(t.Context()),
	}

	req := httptest.NewRequest(http.MethodPost, "/e/test-key", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "test-key")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	a.ReceiveEvent(w, req)

	var resp apiutil.EventAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), w.Body.String())
	return w, resp, handled
}

func idFromName(_ context.Context, e *event.Event, _ *event.SeededID) (string, error) {
	return "id-" + e.Name, nil
}

func TestReceiveEvent_Batch(t *testing.T) {
	t.Run("all valid", func(t *testing.T) {
		w, resp, handled := postBatch(t, `[{"name": "a"}, {"name": "b"}]`, idFromName)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, []string{"id-a", "id-b"}, resp.IDs)
		require.Empty(t, resp.Error)
		require.Equal(t, []string{"a", "b"}, handled)
	})

	t.Run("invalid events are skipped", func(t *testing.T) {
		w, resp, handled := postBatch(t, `[
			{"name": "a"},
			{"name": ""},
			{"name": "b"},
			{"name": "inngest/reserved"},
			{"name": "c"}
		]`, idFromName)

		require.Equal(t, http.StatusNotAcceptable, w.Code)
		require.Equal(t, http.StatusNotAcceptable, resp.Status)
		require.Equal(t, []string{"id-a", "", "id-b", "", "id-c"}, resp.IDs)
		require.Contains(t, resp.Error, "event name is empty")
		require.Equal(t, []string{"a", "b", "c"}, handled)
	})

	t.Run("first failure picks the status", func(t *testing.T) {
		w, resp, _ := postBatch(t, `[{"name": "inngest/reserved"}, {"name": ""}]`, idFromName)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Equal(t, []string{"", ""}, resp.IDs)
		require.Contains(t, resp.Error, event.ErrInternalEventName.Error())
	})

	t.Run("handler failure", func(t *testing.T) {
		w, resp, handled := postBatch(t, `[{"name": "a"}, {"name": "fail"}, {"name": "b"}]`,
			func(ctx context.Context, e *event.Event, seed *event.SeededID) (string, error) {
				if e.Name == "fail" {
					return "", errors.New("publish failed")
				}
				return idFromName(ctx, e, seed)
			},
		)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.Equal(t, []string{"id-a", "", "id-b"}, resp.IDs)
		require.Equal(t, []string{"a", "fail", "b"}, handled)
	})

	t.Run("undecodable event rejects the whole batch", func(t *testing.T) {
		w, resp, handled := postBatch(t, `[{"name": "a"}, {"name": "b", "data": "nope"}]`, idFromName)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Equal(t, []string{}, resp.IDs)
		require.NotEmpty(t, resp.Error)
		require.Empty(t, handled)
	})
}
