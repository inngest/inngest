package apiv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testAppSyncProvider struct {
	list func(context.Context, App, ListAppSyncsOpts) (*ListAppSyncsResult, error)
	get  func(context.Context, App, uuid.UUID) (AppSyncRecord, error)
}

func (p testAppSyncProvider) ListAppSyncs(ctx context.Context, app App, opts ListAppSyncsOpts) (*ListAppSyncsResult, error) {
	return p.list(ctx, app, opts)
}

func (p testAppSyncProvider) GetAppSync(ctx context.Context, app App, id uuid.UUID) (AppSyncRecord, error) {
	return p.get(ctx, app, id)
}

func TestHTTPGateway_ListAppSyncs(t *testing.T) {
	app := App{ID: "my app", InternalID: uuid.New()}
	first := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	last := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	created := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	apps := &mockAppProvider{}
	apps.On("GetApp", mock.Anything, "my app").Return(app, nil).Twice()
	provider := testAppSyncProvider{list: func(_ context.Context, got App, opts ListAppSyncsOpts) (*ListAppSyncsResult, error) {
		require.Equal(t, app, got)
		require.Equal(t, 1, opts.Limit)
		if opts.Cursor == uuid.Nil {
			return &ListAppSyncsResult{Syncs: []AppSyncRecord{{ID: first, Status: "success", CreatedAt: created}}, HasMore: true}, nil
		}
		require.Equal(t, first, opts.Cursor)
		return &ListAppSyncsResult{Syncs: []AppSyncRecord{{ID: last, Status: "duplicate", CreatedAt: created.Add(-time.Hour)}}}, nil
	}}
	handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{Apps: apps, AppSyncs: provider}, HTTPHandlerOptions{})
	require.NoError(t, err)
	cursor := ""
	for i, id := range []uuid.UUID{first, last} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v2/apps/my%20app/syncs?limit=1&cursor="+cursor, nil))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body struct {
			Data []map[string]any `json:"data"`
			Page struct {
				Limit   int     `json:"limit"`
				HasMore bool    `json:"hasMore"`
				Cursor  *string `json:"cursor"`
			} `json:"page"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Data, 1)
		require.Equal(t, id.String(), body.Data[0]["id"])
		require.Equal(t, "my app", body.Data[0]["appId"])
		for _, key := range []string{"url", "sdkLanguage", "sdkVersion", "framework", "appVersion", "error"} {
			require.NotContains(t, body.Data[0], key)
		}
		require.Equal(t, 1, body.Page.Limit)
		require.Equal(t, i == 0, body.Page.HasMore)
		if i == 0 {
			require.NotNil(t, body.Page.Cursor)
			require.Equal(t, first.String(), *body.Page.Cursor)
			cursor = *body.Page.Cursor
		} else {
			require.Nil(t, body.Page.Cursor)
		}
	}
	apps.AssertExpectations(t)
}

func TestHTTPGateway_ListAppSyncsEmpty(t *testing.T) {
	for _, query := range []string{"", "?limit=0", "?limit=100"} {
		t.Run(query, func(t *testing.T) {
			limit := 20
			if query == "?limit=100" {
				limit = 100
			}
			app := App{ID: "my-app", InternalID: uuid.New()}
			apps := &mockAppProvider{}
			apps.On("GetApp", mock.Anything, "my-app").Return(app, nil).Once()
			provider := testAppSyncProvider{list: func(_ context.Context, got App, opts ListAppSyncsOpts) (*ListAppSyncsResult, error) {
				require.Equal(t, app, got)
				require.Equal(t, ListAppSyncsOpts{Limit: limit}, opts)
				return &ListAppSyncsResult{}, nil
			}}
			handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{Apps: apps, AppSyncs: provider}, HTTPHandlerOptions{})
			require.NoError(t, err)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v2/apps/my-app/syncs"+query, nil))
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var body map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.JSONEq(t, `[]`, string(body["data"]))
			require.JSONEq(t, fmt.Sprintf(`{"limit":%d,"hasMore":false}`, limit), string(body["page"]))
			require.Contains(t, string(body["metadata"]), `"fetchedAt"`)
			apps.AssertExpectations(t)
		})
	}
}

func TestHTTPGateway_GetAppSync(t *testing.T) {
	app := App{ID: "my-app", InternalID: uuid.New()}
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	apps := &mockAppProvider{}
	apps.On("GetApp", mock.Anything, app.InternalID.String()).Return(app, nil).Twice()
	for _, full := range []bool{true, false} {
		provider := testAppSyncProvider{get: func(_ context.Context, got App, gotID uuid.UUID) (AppSyncRecord, error) {
			require.Equal(t, app, got)
			require.Equal(t, id, gotID)
			record := AppSyncRecord{ID: id, Status: "duplicate", CreatedAt: time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)}
			if full {
				record.Status = "error"
				record.URL = "https://example.com/api/inngest"
				record.SDKLanguage = "typescript"
				record.SDKVersion = "3.22.0"
				record.Framework = "nextjs"
				record.AppVersion = "APP_METHOD_literal"
				record.Error = &SyncError{Code: "sync_failed", Message: "FUNCTION_RUN_STATUS_literal"}
			}
			return record, nil
		}}
		handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{Apps: apps, AppSyncs: provider}, HTTPHandlerOptions{})
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v2/apps/"+app.InternalID.String()+"/syncs/"+id.String(), nil))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		if full {
			require.JSONEq(t, `{"id":"11111111-1111-1111-1111-111111111111","appId":"my-app","status":"error","url":"https://example.com/api/inngest","sdkLanguage":"typescript","sdkVersion":"3.22.0","framework":"nextjs","appVersion":"APP_METHOD_literal","error":{"code":"sync_failed","message":"FUNCTION_RUN_STATUS_literal"},"createdAt":"2026-11-02T12:00:00Z"}`, string(body["data"]))
		} else {
			require.JSONEq(t, `{"id":"11111111-1111-1111-1111-111111111111","appId":"my-app","status":"duplicate","createdAt":"2026-11-02T12:00:00Z"}`, string(body["data"]))
		}
		require.Contains(t, string(body["metadata"]), `"fetchedAt"`)
	}
	apps.AssertExpectations(t)
}

func TestHTTPGateway_AppSyncErrors(t *testing.T) {
	id := uuid.New().String()
	for _, tc := range []struct {
		name, suffix, message    string
		status                   int
		appErr, syncErr          error
		noApps, noSyncs, limited bool
	}{
		{name: "limit too high", suffix: "?limit=101", status: 400, message: "Limit cannot exceed 100"},
		{name: "negative limit", suffix: "?limit=-1", status: 400, message: "Limit must be at least 1"},
		{name: "bad cursor", suffix: "?cursor=bad", status: 400, message: "Cursor is invalid"},
		{name: "unresolved cursor", suffix: "?cursor=" + id, status: 400, message: "Cursor is invalid", syncErr: fmt.Errorf("lookup: %w", ErrInvalidCursor)},
		{name: "list unknown app", status: 404, message: "App not found", appErr: fmt.Errorf("lookup: %w", ErrAppNotFound)},
		{name: "get unknown app", suffix: "/" + id, status: 404, message: "App not found", appErr: ErrAppNotFound},
		{name: "unknown sync", suffix: "/" + id, status: 404, message: "Sync not found", syncErr: fmt.Errorf("lookup: %w", ErrAppSyncNotFound)},
		{name: "garbage sync", suffix: "/bad", status: 400, message: "Sync ID must be a valid UUID"},
		{name: "list app failure", status: 500, message: "Unable to fetch app", appErr: errors.New("private database detail")},
		{name: "get app failure", suffix: "/" + id, status: 500, message: "Unable to fetch app", appErr: errors.New("private database detail")},
		{name: "list sync failure", status: 500, message: "Unable to fetch app syncs", syncErr: errors.New("private database detail")},
		{name: "get sync failure", suffix: "/" + id, status: 500, message: "Unable to fetch app sync", syncErr: errors.New("private database detail")},
		{name: "list no sync provider", status: 501, noSyncs: true},
		{name: "get no sync provider", suffix: "/" + id, status: 501, noSyncs: true},
		{name: "list no app provider", status: 501, noApps: true},
		{name: "get no app provider", suffix: "/" + id, status: 501, noApps: true},
		{name: "list rate limited", suffix: "?limit=101", status: 429, limited: true, noSyncs: true},
		{name: "get rate limited", suffix: "/bad", status: 429, limited: true, noSyncs: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			apps := &mockAppProvider{}
			apps.On("GetApp", mock.Anything, "my-app").Return(App{ID: "my-app"}, tc.appErr)
			opts := ServiceOptions{RateLimitProvider: &testEventRateLimiter{limited: tc.limited}}
			if !tc.noApps {
				opts.Apps = apps
			}
			if !tc.noSyncs {
				opts.AppSyncs = testAppSyncProvider{
					list: func(context.Context, App, ListAppSyncsOpts) (*ListAppSyncsResult, error) {
						require.NotNil(t, tc.syncErr, "unexpected provider call")
						return nil, tc.syncErr
					},
					get: func(context.Context, App, uuid.UUID) (AppSyncRecord, error) {
						require.NotNil(t, tc.syncErr, "unexpected provider call")
						return AppSyncRecord{}, tc.syncErr
					},
				}
			}
			handler, err := newTestHTTPHandler(t.Context(), opts, HTTPHandlerOptions{})
			require.NoError(t, err)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v2/apps/my-app/syncs"+tc.suffix, nil))
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			if tc.status == http.StatusBadRequest {
				require.JSONEq(t, fmt.Sprintf(`{"errors":[{"code":"invalid_field_format","message":%q}]}`, tc.message), rec.Body.String())
			} else if tc.message != "" {
				require.Contains(t, rec.Body.String(), tc.message)
			}
			require.NotContains(t, rec.Body.String(), "private database detail")
		})
	}
}

func TestAppSyncMissingAppAndRateLimit(t *testing.T) {
	for _, limited := range []bool{false, true} {
		limiter := &testEventRateLimiter{limited: limited}
		svc := NewService(ServiceOptions{RateLimitProvider: limiter})
		status, code := 400, "missing_field"
		if limited {
			status, code = 429, "rate_limited"
		}
		_, err := svc.ListAppSyncs(t.Context(), &apiv2.ListAppSyncsRequest{})
		requireAPIError(t, err, status, code)
		require.Equal(t, apiv2.V2_ListAppSyncs_FullMethodName, limiter.method)
		_, err = svc.GetAppSync(t.Context(), &apiv2.GetAppSyncRequest{})
		requireAPIError(t, err, status, code)
		require.Equal(t, apiv2.V2_GetAppSync_FullMethodName, limiter.method)
	}
}
