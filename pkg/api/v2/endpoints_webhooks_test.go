package apiv2

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
		{"whitespace transform", id, "production", `{"transform":" "}`, "not_implemented", "Webhooks not implemented in OSS", 501},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v2/env/webhooks/"+tc.id, strings.NewReader(tc.body))
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
