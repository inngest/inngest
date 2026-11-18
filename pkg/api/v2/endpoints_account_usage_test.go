package apiv2

import (
	"net/http"
	"net/http/httptest"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
)

func TestHTTPGateway_GetAccountUsageCloudOnly(t *testing.T) {
	handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{}, HTTPHandlerOptions{})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v2/account/usage", nil))
	require.Equal(t, http.StatusNotImplemented, rec.Code, rec.Body.String())
	require.JSONEq(t, `{"errors":[{"code":"not_implemented","message":"Account usage is only available in Inngest Cloud. Run: inngest api --prod get-account-usage"}]}`, rec.Body.String())
}

func TestHTTPGateway_GetAccountUsageRateLimited(t *testing.T) {
	limiter := &testEventRateLimiter{limited: true}
	handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{RateLimitProvider: limiter}, HTTPHandlerOptions{})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v2/account/usage", nil))
	require.Equal(t, http.StatusTooManyRequests, rec.Code, rec.Body.String())
	require.Equal(t, apiv2.V2_GetAccountUsage_FullMethodName, limiter.method)
	require.JSONEq(t, `{"errors":[{"code":"rate_limited","message":"API rate limit exceeded. The request was rejected and no account usage was fetched."}]}`, rec.Body.String())
}
