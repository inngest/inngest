package devserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/stretchr/testify/require"
)

func TestAppSyncProvider(t *testing.T) {
	provider := NewAppProvider(&fakeFunctionStore{app: &cqrs.App{
		ID: uuid.New(), Name: "my-app", SdkVersion: "3.22.0",
	}})
	app, err := provider.GetApp(t.Context(), "my-app")
	require.NoError(t, err)
	require.Equal(t, "3.22.0", app.LatestSync.SdkVersion)
	result, err := provider.ListAppSyncs(t.Context(), app, apiv2.ListAppSyncsOpts{Limit: 20})
	require.NoError(t, err)
	require.Empty(t, result.Syncs)
	require.False(t, result.HasMore)
	_, err = provider.ListAppSyncs(t.Context(), app, apiv2.ListAppSyncsOpts{Limit: 20, Cursor: uuid.New()})
	require.ErrorIs(t, err, apiv2.ErrInvalidCursor)
	_, err = provider.GetAppSync(t.Context(), app, uuid.New())
	require.ErrorIs(t, err, apiv2.ErrAppSyncNotFound)
}

func TestAppSyncHistoryHTTP(t *testing.T) {
	provider := NewAppProvider(&fakeFunctionStore{app: &cqrs.App{ID: uuid.New(), Name: "my-app"}})
	handler, err := apiv2.NewHTTPHandler(t.Context(), apiv2.ServiceOptions{Apps: provider, AppSyncs: provider}, apiv2.HTTPHandlerOptions{}, apiv2base.NewBase())
	require.NoError(t, err)
	id := uuid.New().String()
	for _, tc := range []struct {
		path    string
		status  int
		message string
	}{
		{"/v2/apps/my-app/syncs", 200, ""},
		{"/v2/apps/my-app/syncs?cursor=" + id, 400, "Cursor is invalid"},
		{"/v2/apps/my-app/syncs/" + id, 404, "Sync not found"},
		{"/v2/apps/missing/syncs", 404, "App not found"},
		{"/v2/apps/missing/syncs/" + id, 404, "App not found"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			if tc.status == 200 {
				var body map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				require.JSONEq(t, `[]`, string(body["data"]))
				require.JSONEq(t, `{"limit":20,"hasMore":false}`, string(body["page"]))
			} else if tc.status == http.StatusBadRequest {
				require.JSONEq(t, `{"errors":[{"code":"invalid_field_format","message":"Cursor is invalid"}]}`, rec.Body.String())
			} else {
				require.Contains(t, rec.Body.String(), tc.message)
			}
		})
	}
}
