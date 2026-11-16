package apiv2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPGateway_GetAccountUsage(t *testing.T) {
	handler, err := newTestHTTPHandler(t.Context(), ServiceOptions{}, HTTPHandlerOptions{})
	require.NoError(t, err)

	before := time.Now().UTC()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v2/account/usage", nil))
	after := time.Now().UTC()
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var response struct {
		Data     map[string]any `json:"data"`
		Metadata struct {
			FetchedAt time.Time `json:"fetchedAt"`
		} `json:"metadata"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	fetchedAt := response.Metadata.FetchedAt
	require.False(t, fetchedAt.Before(before))
	require.False(t, fetchedAt.After(after))
	start := time.Date(fetchedAt.Year(), fetchedAt.Month(), 1, 0, 0, 0, 0, time.UTC)
	require.Equal(t, map[string]any{
		"executions":  "0",
		"periodStart": start.Format(time.RFC3339),
		"periodEnd":   start.AddDate(0, 1, 0).Format(time.RFC3339),
	}, response.Data)
}

func TestAccountUsagePeriod(t *testing.T) {
	for _, tc := range []struct {
		now, start, end string
	}{
		{"2026-11-30T23:59:59Z", "2026-11-01T00:00:00Z", "2026-12-01T00:00:00Z"},
		{"2026-12-01T00:00:00Z", "2026-12-01T00:00:00Z", "2027-01-01T00:00:00Z"},
		{"2026-12-31T23:59:59Z", "2026-12-01T00:00:00Z", "2027-01-01T00:00:00Z"},
		{"2027-01-01T00:00:00Z", "2027-01-01T00:00:00Z", "2027-02-01T00:00:00Z"},
		{"2026-12-31T20:00:00-05:00", "2027-01-01T00:00:00Z", "2027-02-01T00:00:00Z"},
		{"2028-02-29T12:00:00Z", "2028-02-01T00:00:00Z", "2028-03-01T00:00:00Z"},
	} {
		t.Run(tc.now, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tc.now)
			require.NoError(t, err)
			start, end := accountUsagePeriod(now)
			require.Equal(t, tc.start, start.Format(time.RFC3339))
			require.Equal(t, tc.end, end.Format(time.RFC3339))
		})
	}
}
