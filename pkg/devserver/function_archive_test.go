package devserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/stretchr/testify/require"
)

func TestFunctionArchiveUnsupported(t *testing.T) {
	ds := newTestDevServer(t)
	appID, fnID := uuid.New(), uuid.New()
	_, err := ds.Data.UpsertApp(t.Context(), cqrs.UpsertAppParams{ID: appID, Name: "app"})
	require.NoError(t, err)
	_, err = ds.Data.UpsertFunction(t.Context(), cqrs.UpsertFunctionParams{
		ID: fnID, AppID: appID, Slug: "app-fn", Name: "Function",
		Config: `{"name":"Function","slug":"fn","triggers":[{"event":"test/event"}]}`,
	})
	require.NoError(t, err)
	provider := NewFunctionProvider(ds.Data, nil, nil)
	provider.pauses.pause(fnID)
	before, err := provider.GetFunctionByApp(t.Context(), "app", "fn")
	require.NoError(t, err)
	params := apiParams{
		CQRS: ds.Data, Functions: provider, Clients: &redisClients{},
		DualWrite: &dualWrite{data: ds.Data},
	}
	handler, err := apiv2.NewHTTPHandler(t.Context(), params.v2ServiceOptions(), apiv2.HTTPHandlerOptions{}, apiv2base.NewBase())
	require.NoError(t, err)
	for _, action := range []string{"archive", "unarchive"} {
		for _, tc := range []struct {
			body   string
			status int
		}{
			{"", http.StatusNotImplemented},
			{"{}", http.StatusBadRequest},
		} {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v2/apps/app/functions/fn/"+action, strings.NewReader(tc.body)))
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			if tc.status == http.StatusNotImplemented {
				require.Contains(t, rec.Body.String(), `"code":"not_implemented"`)
			}
			after, err := provider.GetFunctionByApp(t.Context(), "app", "fn")
			require.NoError(t, err)
			require.Equal(t, before, after)
		}
	}
}
