package apiv2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/logger"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type testEventProvider struct {
	get  func(context.Context, ulid.ULID) (*cqrs.Event, error)
	list func(context.Context, *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error)
}

func (p testEventProvider) GetEvent(ctx context.Context, id ulid.ULID) (*cqrs.Event, error) {
	return p.get(ctx, id)
}
func (p testEventProvider) GetEvents(ctx context.Context, opts *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
	return p.list(ctx, opts)
}

func TestListEventsOptions(t *testing.T) {
	now := time.Now().UTC()
	from, until := now.Add(-30*time.Minute), now.Add(-time.Minute)
	id := ulid.Make()
	for _, tc := range []struct {
		name  string
		req   *apiv2.ListEventsRequest
		limit int
	}{
		{"default", &apiv2.ListEventsRequest{}, 40},
		{"zero", &apiv2.ListEventsRequest{Limit: new(int32(0))}, 40},
		{"maximum", &apiv2.ListEventsRequest{Limit: new(int32(50))}, 50},
		{"payload default", &apiv2.ListEventsRequest{Include: []string{"payload"}}, 20},
		{"payload zero", &apiv2.ListEventsRequest{Limit: new(int32(0)), Include: []string{"payload"}}, 20},
		{"payload maximum", &apiv2.ListEventsRequest{Limit: new(int32(20)), Include: []string{"payload"}}, 20},
		{"filters", &apiv2.ListEventsRequest{Cursor: new(id.String()), From: timestamppb.New(from), Until: timestamppb.New(until), Name: []string{"order/placed", "order/paid"}}, 40},
		{"equal bounds", &apiv2.ListEventsRequest{From: timestamppb.New(from), Until: timestamppb.New(from)}, 40},
		{"historical until", &apiv2.ListEventsRequest{Until: timestamppb.New(now.Add(-24 * time.Hour))}, 40},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := time.Now()
			var received cqrs.WorkspaceEventsOpts
			svc := NewService(ServiceOptions{Events: testEventProvider{list: func(_ context.Context, opts *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
				received = *opts
				return []*cqrs.Event{{ID: id, EventName: "order/placed", ReceivedAt: now}}, nil
			}}})
			resp, err := svc.ListEvents(t.Context(), tc.req)
			require.NoError(t, err)
			require.Equal(t, int32(tc.limit), resp.Page.Limit)
			require.Equal(t, tc.limit+1, received.Limit)
			require.False(t, received.IncludeInternalEvents)
			require.Equal(t, tc.req.Name, received.Names)
			if tc.req.Cursor != nil {
				require.Equal(t, id, *received.Cursor)
			} else {
				require.Nil(t, received.Cursor)
			}
			if tc.req.From != nil {
				require.Equal(t, tc.req.From.AsTime(), received.Oldest)
			} else if tc.req.Until != nil {
				require.Equal(t, tc.req.Until.AsTime().Add(-time.Hour), received.Oldest)
			} else {
				require.WithinRange(t, received.Oldest, before.Add(-time.Hour), time.Now().Add(-time.Hour))
			}
			if tc.req.Until != nil {
				require.Equal(t, tc.req.Until.AsTime(), received.Newest)
			} else {
				require.WithinRange(t, received.Newest, before, time.Now())
			}
			require.Equal(t, received.Oldest.UTC(), resp.Metadata.TimeRange.From.AsTime())
			require.Equal(t, received.Newest.UTC(), resp.Metadata.TimeRange.Until.AsTime())
			require.Equal(t, "order/placed", resp.Data[0].Name)
			require.False(t, resp.Page.HasMore)
			require.Nil(t, resp.Page.Cursor)
		})
	}
}

func TestListEventsTimestampBounds(t *testing.T) {
	for _, tc := range []struct {
		name    string
		from    string
		until   string
		message string
	}{
		{"from zero", "0001-01-01T00:00:00Z", "", "from must not be the zero time"},
		{"until zero", "", "0001-01-01T00:00:00Z", "until must not be the zero time"},
		{"both zero", "0001-01-01T00:00:00Z", "0001-01-01T00:00:00Z", "from must not be the zero time"},
		{"valid nonzero", "1970-01-01T00:00:00Z", "1970-01-01T01:00:00Z", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var received cqrs.WorkspaceEventsOpts
			provider := testEventProvider{list: func(_ context.Context, opts *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
				calls++
				received = *opts
				return []*cqrs.Event{{ID: ulid.MustParse("01J00000000000000000000000"), EventName: "app/event", ReceivedAt: time.Unix(0, 0)}}, nil
			}}
			handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{Events: provider}, HTTPHandlerOptions{})
			require.NoError(t, err)
			query := url.Values{}
			if tc.from != "" {
				query.Set("from", tc.from)
			}
			if tc.until != "" {
				query.Set("until", tc.until)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v2/events?"+query.Encode(), nil))
			if tc.message != "" {
				require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
				require.Contains(t, rec.Body.String(), `"code":"invalid_field_format"`)
				require.Contains(t, rec.Body.String(), tc.message)
				require.Zero(t, calls)
				return
			}
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"name":"app/event"`)
			require.Equal(t, 1, calls)
			require.Equal(t, tc.from, received.Oldest.Format(time.RFC3339))
			require.Equal(t, tc.until, received.Newest.Format(time.RFC3339))
		})
	}
}

func TestListEventsNames(t *testing.T) {
	for _, tc := range []struct {
		name  string
		names []string
		valid bool
	}{
		{"omitted", nil, true},
		{"quotes unicode and slash", []string{"app/it's ok", "café/注文"}, true},
		{"20 names", strings.Split(strings.Repeat("event,", 19)+"event", ","), true},
		{"21 names", strings.Split(strings.Repeat("event,", 20)+"event", ","), false},
		{"256 bytes", []string{strings.Repeat("a", 256)}, true},
		{"257 bytes", []string{strings.Repeat("a", 257)}, false},
		{"256 unicode bytes", []string{strings.Repeat("é", 128)}, true},
		{"258 unicode bytes", []string{strings.Repeat("é", 129)}, false},
		{"empty", []string{""}, false},
		{"empty after valid", []string{"app/valid", ""}, false},
		{"backslash", []string{`app\event`}, false},
		{"injection", []string{`x\')) OR 1=1 --`}, false},
		{"nul", []string{"app/\x00event"}, false},
		{"newline", []string{"app/\nevent"}, false},
		{"tab", []string{"app/\tevent"}, false},
		{"delete", []string{"app/\x7fevent"}, false},
		{"unicode control", []string{"app/\u0085event"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			provider := testEventProvider{list: func(_ context.Context, opts *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
				calls++
				require.Equal(t, tc.names, opts.Names)
				return []*cqrs.Event{{ID: ulid.MustParse("01J00000000000000000000000"), EventName: "app/it's ok", ReceivedAt: time.Now()}}, nil
			}}
			handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{Events: provider}, HTTPHandlerOptions{})
			require.NoError(t, err)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v2/events?"+url.Values{"name": tc.names}.Encode(), nil))
			if !tc.valid {
				require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
				require.Contains(t, rec.Body.String(), `"code":"invalid_field_format"`)
				require.Zero(t, calls)
				return
			}
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"name":"app/it's ok"`)
			require.Equal(t, 1, calls)
		})
	}
}

func TestListEventsOmitEventData(t *testing.T) {
	for _, tc := range []struct {
		name    string
		include []string
		omit    bool
	}{
		{"default", nil, true},
		{"payload", []string{"payload"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(ServiceOptions{Events: testEventProvider{list: func(_ context.Context, opts *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
				require.Equal(t, tc.omit, opts.OmitEventData)
				return []*cqrs.Event{{ID: ulid.Make(), EventName: "app/event", ReceivedAt: time.Now(), EventData: map[string]any{"message": "hello"}}}, nil
			}}})
			resp, err := svc.ListEvents(t.Context(), &apiv2.ListEventsRequest{Include: tc.include})
			require.NoError(t, err)
			require.Len(t, resp.Data, 1)
			require.Equal(t, "app/event", resp.Data[0].Name)
			if tc.omit {
				require.Nil(t, resp.Data[0].Payload)
			} else {
				require.Equal(t, map[string]any{"message": "hello"}, resp.Data[0].Payload.AsMap()["data"])
			}
		})
	}
}

func TestGetEventExcludesInternalEvents(t *testing.T) {
	id := ulid.Make()
	missing := NewService(ServiceOptions{Events: testEventProvider{get: func(context.Context, ulid.ULID) (*cqrs.Event, error) {
		return nil, ErrEventNotFound
	}}})
	_, notFound := missing.GetEvent(t.Context(), &apiv2.GetEventRequest{EventId: id.String()})
	requireAPIError(t, notFound, http.StatusNotFound, "not_found")
	for _, tc := range []struct {
		name     string
		internal bool
	}{
		{"inngest/function.finished", true},
		{"inngest/", true},
		{"app/it's ok", false},
		{"inngest-custom/event", false},
	} {
		for _, include := range [][]string{nil, {"payload"}} {
			t.Run(fmt.Sprintf("%s/%v", tc.name, include), func(t *testing.T) {
				svc := NewService(ServiceOptions{Events: testEventProvider{get: func(_ context.Context, got ulid.ULID) (*cqrs.Event, error) {
					require.Equal(t, id, got)
					return &cqrs.Event{ID: id, EventName: tc.name, ReceivedAt: time.Now(), EventData: map[string]any{"output": "private"}}, nil
				}}})
				resp, err := svc.GetEvent(t.Context(), &apiv2.GetEventRequest{EventId: id.String(), Include: include})
				if tc.internal {
					requireAPIError(t, err, http.StatusNotFound, "not_found")
					require.Equal(t, notFound.Error(), err.Error())
					require.Nil(t, resp)
					return
				}
				require.NoError(t, err)
				require.Equal(t, id.String(), resp.Data.Id)
				require.Equal(t, tc.name, resp.Data.Name)
			})
		}
	}
}

func TestEventReadProviderErrorsLogged(t *testing.T) {
	var logs bytes.Buffer
	ctx := logger.WithStdlib(t.Context(), logger.FromSlog(slog.New(slog.NewJSONHandler(&logs, nil)), slog.LevelInfo))
	providerErr := errors.New("database unavailable")
	evt := &cqrs.Event{EventName: "private-event-name", EventData: map[string]any{"secret": "private-payload"}}
	svc := NewService(ServiceOptions{Events: testEventProvider{
		get: func(context.Context, ulid.ULID) (*cqrs.Event, error) { return evt, providerErr },
		list: func(context.Context, *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
			return []*cqrs.Event{evt}, providerErr
		},
	}})
	for _, list := range []bool{true, false} {
		logs.Reset()
		var err error
		message := "unable to fetch event"
		if list {
			_, err = svc.ListEvents(ctx, &apiv2.ListEventsRequest{})
			message = "unable to fetch events"
		} else {
			_, err = svc.GetEvent(ctx, &apiv2.GetEventRequest{EventId: ulid.Make().String()})
		}
		requireAPIError(t, err, http.StatusInternalServerError, "internal_error")
		require.NotContains(t, err.Error(), providerErr.Error())
		var entry map[string]any
		require.NoError(t, json.Unmarshal(logs.Bytes(), &entry))
		require.Equal(t, message, entry["msg"])
		require.Equal(t, "database unavailable", entry["error"])
		require.NotContains(t, logs.String(), "private-event-name")
		require.NotContains(t, logs.String(), "private-payload")
	}
}

func TestEventReadValidation(t *testing.T) {
	now := timestamppb.Now()
	for _, req := range []*apiv2.ListEventsRequest{
		{Limit: new(int32(-1))}, {Limit: new(int32(51))},
		{Limit: new(int32(21)), Include: []string{"payload"}},
		{Cursor: new("bad")}, {Cursor: new("01J0000000000000000000000!")},
		{From: &timestamppb.Timestamp{Seconds: 253402300800}},
		{Until: &timestamppb.Timestamp{Nanos: -1}},
		{From: now, Until: timestamppb.New(now.AsTime().Add(-time.Second))},
		{Include: []string{"unknown"}},
	} {
		svc := NewService(ServiceOptions{Events: testEventProvider{}})
		_, err := svc.ListEvents(t.Context(), req)
		requireAPIError(t, err, 400, "invalid_field_format")
	}
	for _, req := range []*apiv2.GetEventRequest{
		{}, {EventId: "bad"}, {EventId: "01J0000000000000000000000!"},
		{EventId: ulid.Make().String(), Include: []string{"unknown"}},
	} {
		_, err := NewService(ServiceOptions{Events: testEventProvider{}}).GetEvent(t.Context(), req)
		requireAPIError(t, err, 400, "invalid_field_format")
	}
}

func requireAPIError(t *testing.T, err error, status int, code string) {
	t.Helper()
	require.Error(t, err)
	var httpErr interface{ HTTPStatus() int }
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, status, httpErr.HTTPStatus())
	require.Contains(t, err.Error(), code)
}

func TestEventReadErrors(t *testing.T) {
	id := ulid.Make()
	for _, tc := range []struct {
		name                string
		err                 error
		noProvider, limited bool
		status              int
		code                string
	}{
		{name: "unavailable", err: errors.New("private database detail"), status: 500, code: "internal_error"},
		{name: "nil provider", noProvider: true, status: 501, code: "not_implemented"},
		{name: "rate limited", limited: true, status: 429, code: "rate_limited"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limiter := &testEventRateLimiter{limited: tc.limited}
			calls := 0
			opts := ServiceOptions{RateLimitProvider: limiter}
			if !tc.noProvider {
				opts.Events = testEventProvider{
					get:  func(context.Context, ulid.ULID) (*cqrs.Event, error) { calls++; return nil, tc.err },
					list: func(context.Context, *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) { calls++; return nil, tc.err },
				}
			}
			svc := NewService(opts)
			_, err := svc.ListEvents(t.Context(), &apiv2.ListEventsRequest{})
			requireAPIError(t, err, tc.status, tc.code)
			require.NotContains(t, err.Error(), "private database detail")
			require.Equal(t, apiv2.V2_ListEvents_FullMethodName, limiter.method)
			_, err = svc.GetEvent(t.Context(), &apiv2.GetEventRequest{EventId: id.String()})
			requireAPIError(t, err, tc.status, tc.code)
			require.NotContains(t, err.Error(), "private database detail")
			require.Equal(t, apiv2.V2_GetEvent_FullMethodName, limiter.method)
			if tc.limited || tc.noProvider {
				require.Zero(t, calls)
			} else {
				require.Equal(t, 2, calls)
			}
		})
	}
	svc := NewService(ServiceOptions{Events: testEventProvider{get: func(context.Context, ulid.ULID) (*cqrs.Event, error) {
		return nil, fmt.Errorf("lookup: %w", ErrEventNotFound)
	}}})
	_, err := svc.GetEvent(t.Context(), &apiv2.GetEventRequest{EventId: id.String()})
	requireAPIError(t, err, 404, "not_found")
	require.Contains(t, err.Error(), "Event not found")
}

func TestHTTPGateway_EventReads(t *testing.T) {
	id := ulid.MustParse("01J00000000000000000000000")
	received := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	sourceID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	evt := &cqrs.Event{ID: id, EventName: "FUNCTION_RUN_STATUS_literal", ReceivedAt: received,
		EventID: "APP_METHOD_sender", EventVersion: "TRACE_STEP_OP_version", EventTS: received.Add(-time.Minute).UnixMilli(),
		Source: "SANDBOX_STATUS_source", SourceID: &sourceID,
		EventData: map[string]any{"literal": "FUNCTION_RUN_STATUS_FAILED", "nested": []any{"APP_METHOD_SERVE"}}, EventUser: map[string]any{"id": "user-1"},
	}
	provider := testEventProvider{
		get: func(_ context.Context, got ulid.ULID) (*cqrs.Event, error) {
			require.Equal(t, id, got)
			return evt, nil
		},
		list: func(_ context.Context, opts *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
			require.Equal(t, []string{"first", "second"}, opts.Names)
			require.Equal(t, received.Add(-time.Hour), opts.Oldest)
			require.Equal(t, received, opts.Newest)
			return []*cqrs.Event{evt}, nil
		},
	}
	handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{Events: provider}, HTTPHandlerOptions{})
	require.NoError(t, err)
	for _, include := range []bool{false, true} {
		for _, list := range []bool{false, true} {
			path := "/api/v2/events/" + id.String() + "?"
			if list {
				path = "/api/v2/events?name=first&name=second&from=2026-10-27T11:00:00Z&until=2026-10-27T12:00:00Z&"
			}
			if include {
				path += "include=payload"
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			require.Equal(t, 200, rec.Code, rec.Body.String())
			var body map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			data := body["data"]
			if list {
				data = data.([]any)[0]
				require.Equal(t, false, body["page"].(map[string]any)["hasMore"])
				require.NotContains(t, body["page"], "cursor")
			}
			expected := map[string]any{"id": id.String(), "name": "FUNCTION_RUN_STATUS_literal", "receivedAt": "2026-10-27T12:00:00Z", "occurredAt": "2026-10-27T11:59:00Z", "idempotencyKey": "APP_METHOD_sender", "version": "TRACE_STEP_OP_version", "source": map[string]any{"id": sourceID.String(), "name": "SANDBOX_STATUS_source"}}
			if include {
				expected["payload"] = map[string]any{
					"name": "FUNCTION_RUN_STATUS_literal",
					"data": map[string]any{"literal": "FUNCTION_RUN_STATUS_FAILED", "nested": []any{"APP_METHOD_SERVE"}},
					"user": map[string]any{"id": "user-1"},
					"ts":   float64(received.Add(-time.Minute).UnixMilli()),
					"id":   "APP_METHOD_sender",
					"v":    "TRACE_STEP_OP_version",
				}
			}
			require.Equal(t, expected, data)
		}
	}
	evt.EventTS = 0
	evt.Source = ""
	evt.SourceID = nil
	evt.EventVersion = ""
	evt.EventID = ""
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v2/events/"+id.String(), nil))
	require.Equal(t, 200, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, map[string]any{"id": id.String(), "name": "FUNCTION_RUN_STATUS_literal", "receivedAt": "2026-10-27T12:00:00Z", "idempotencyKey": ""}, body["data"])
}

func TestHTTPGateway_EventRoutes(t *testing.T) {
	provider := testEventProvider{list: func(context.Context, *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) { return nil, nil }}
	handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{Events: provider}, HTTPHandlerOptions{})
	require.NoError(t, err)
	for _, tc := range []struct {
		path   string
		status int
		code   string
	}{
		{"/events", 200, ""}, {"/events?limit=51", 400, "invalid_field_format"},
		{"/events?cursor=invalid", 400, "invalid_field_format"}, {"/events?from=invalid", 400, ""},
		{"/events?includeInternalEvents=true", 400, ""}, {"/events?if=true", 400, ""},
		{"/events/bad", 400, "invalid_field_format"},
		{"/events/01J00000000000000000000000/runs", 501, "not_implemented"},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		require.Equal(t, tc.status, rec.Code, rec.Body.String())
		if tc.code != "" {
			require.Contains(t, rec.Body.String(), tc.code)
		}
		if tc.status == 200 {
			var body map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, []any{}, body["data"])
			require.Equal(t, map[string]any{"hasMore": false, "limit": float64(40)}, body["page"])
		}
	}
	for path, status := range map[string]int{
		"/events/":                           http.StatusNotImplemented,
		"/events/01J00000000000000000000000": http.StatusNotImplemented,
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		require.Equal(t, status, rec.Code, rec.Body.String())
	}
	denied, err := newTestHTTPHandler(t.Context(), ServiceOptions{}, HTTPHandlerOptions{AuthzMiddleware: func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "Access denied", http.StatusForbidden) })
	}})
	require.NoError(t, err)
	for _, prefix := range []string{"", "/v2", "/api/v2"} {
		for _, path := range []string{"/events", "/events/01J00000000000000000000000"} {
			rec := httptest.NewRecorder()
			denied.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, prefix+path, nil))
			require.Equal(t, 403, rec.Code)
			require.Equal(t, "Access denied\n", rec.Body.String())
		}
	}
}
