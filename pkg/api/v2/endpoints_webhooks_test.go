package apiv2

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
)

func TestHTTPGateway_UpdateWebhook(t *testing.T) {
	const id = "11111111-1111-1111-1111-111111111111"
	handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{}, HTTPHandlerOptions{})
	require.NoError(t, err)

	for _, tc := range []struct {
		name, id, env, body, code, message string
		status                             int
	}{
		{"missing environment first", "bad", "", `{}`, "missing_field", "X-Inngest-Env header is required", 400},
		{"invalid id before fields", "bad", "production", `{}`, "invalid_field_format", "Invalid webhook ID", 400},
		{"neither field", id, "production", `{}`, "missing_field", "Name or transform is required", 400},
		{"null fields", id, "production", `{"name":null,"transform":null}`, "missing_field", "Name or transform is required", 400},
		{"empty name", id, "production", `{"name":""}`, "invalid_field_format", "Webhook name cannot be empty", 400},
		{"empty name with transform", id, "production", `{"name":"","transform":"function transform(evt) { return evt; }"}`, "invalid_field_format", "Webhook name cannot be empty", 400},
		{"empty transform", id, "production", `{"transform":""}`, "invalid_field_format", "Transform function cannot be empty", 400},
		{"empty transform with name", id, "production", `{"name":"Renamed","transform":""}`, "invalid_field_format", "Transform function cannot be empty", 400},
		{"empty name before transform", id, "production", `{"name":"","transform":""}`, "invalid_field_format", "Webhook name cannot be empty", 400},
		{"name only", id, "production", `{"name":"Renamed"}`, "not_implemented", "Webhooks not implemented in OSS", 501},
		{"transform only", id, "production", `{"transform":"function transform(evt) { return evt; }"}`, "not_implemented", "Webhooks not implemented in OSS", 501},
		{"both fields", id, "production", `{"name":"Renamed","transform":"function transform(evt) { return evt; }"}`, "not_implemented", "Webhooks not implemented in OSS", 501},
		{"whitespace name", id, "production", `{"name":" "}`, "not_implemented", "Webhooks not implemented in OSS", 501},
		{"name too long", id, "production", `{"name":"` + strings.Repeat("é", 129) + `"}`, "invalid_field_format", "Webhook name must be at most 128 characters", 400},
		{"name at limit", id, "production", `{"name":"` + strings.Repeat("é", 128) + `"}`, "not_implemented", "Webhooks not implemented in OSS", 501},
		{"unknown field", id, "production", `{"name":"Renamed","url":"https://example.com"}`, "invalid_request", "Unexpected request body field \"url\"", 400},
		{"whitespace transform", id, "production", `{"transform":" "}`, "not_implemented", "Webhooks not implemented in OSS", 501},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v2/webhooks/"+tc.id, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.env != "" {
				req.Header.Set("X-Inngest-Env", tc.env)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			require.JSONEq(t, fmt.Sprintf(`{"errors":[{"code":%q,"message":%q}]}`, tc.code, tc.message), rec.Body.String())
		})
	}
}

func TestHTTPGateway_UpdateWebhookRateLimited(t *testing.T) {
	limiter := &testEventRateLimiter{limited: true}
	handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{RateLimitProvider: limiter}, HTTPHandlerOptions{})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v2/webhooks/11111111-1111-1111-1111-111111111111", strings.NewReader(`{"name":"Renamed"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inngest-Env", "production")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusTooManyRequests, rec.Code, rec.Body.String())
	require.Equal(t, apiv2.V2_UpdateWebhook_FullMethodName, limiter.method)
}

func TestHTTPGateway_WebhookBodyLimit(t *testing.T) {
	for _, tc := range []struct {
		path, body string
		limit      int
	}{
		{"/v2/env/webhooks", `{"name":"hook","transform":"function transform(e) { return e; }"}`, 64 << 10},
		{"/v2/webhooks/11111111-1111-1111-1111-111111111111", `{"name":"Renamed"}`, 16 << 10},
	} {
		t.Run(tc.path, func(t *testing.T) {
			handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{}, HTTPHandlerOptions{})
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(strings.Repeat(" ", tc.limit)+tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Inngest-Env", "production")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
		})
	}
}
