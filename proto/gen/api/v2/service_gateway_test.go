package apiv2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
)

type experimentRouteServer struct {
	UnimplementedV2Server
}

func (experimentRouteServer) GetExperiment(_ context.Context, req *GetExperimentRequest) (*GetExperimentResponse, error) {
	return &GetExperimentResponse{
		Data: &ExperimentDetail{
			Id: req.ExperimentId,
		},
	}, nil
}

func TestGetExperimentRouteSupportsSlashInExperimentID(t *testing.T) {
	mux := runtime.NewServeMux()
	require.NoError(t, RegisterV2HandlerServer(context.Background(), mux, experimentRouteServer{}))

	req := httptest.NewRequest(http.MethodGet, "/apps/app/functions/fn/experiments/A%2FB%20rollout", nil)
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	require.Contains(t, res.Body.String(), `"id":"A/B rollout"`)
}

type functionRunsRouteServer struct {
	UnimplementedV2Server
	request *ListFunctionRunsRequest
}

func (s *functionRunsRouteServer) ListFunctionRuns(_ context.Context, req *ListFunctionRunsRequest) (*ListFunctionRunsResponse, error) {
	s.request = req
	return &ListFunctionRunsResponse{}, nil
}

func TestListFunctionRunsRouteSupportsSlashInFunctionID(t *testing.T) {
	for _, test := range []struct {
		name        string
		path        string
		wantPath    string
		wantRawPath string
		functionID  string
	}{
		{
			name:        "encoded infra function ID",
			path:        "/apps/resend/functions/infra%2Fanti-abuse.create-partition/runs",
			wantPath:    "/apps/resend/functions/infra/anti-abuse.create-partition/runs",
			wantRawPath: "/apps/resend/functions/infra%2Fanti-abuse.create-partition/runs",
			functionID:  "infra/anti-abuse.create-partition",
		},
		{
			name:        "encoded users function ID",
			path:        "/apps/resend/functions/users%2Faccount.delete/runs",
			wantPath:    "/apps/resend/functions/users/account.delete/runs",
			wantRawPath: "/apps/resend/functions/users%2Faccount.delete/runs",
			functionID:  "users/account.delete",
		},
		{
			name:       "literal slash in function ID",
			path:       "/apps/resend/functions/infra/anti-abuse.create-partition/runs",
			wantPath:   "/apps/resend/functions/infra/anti-abuse.create-partition/runs",
			functionID: "infra/anti-abuse.create-partition",
		},
		{
			name:       "normal function ID",
			path:       "/apps/resend/functions/send-email/runs",
			wantPath:   "/apps/resend/functions/send-email/runs",
			functionID: "send-email",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := &functionRunsRouteServer{}
			mux := runtime.NewServeMux()
			require.NoError(t, RegisterV2HandlerServer(context.Background(), mux, server))

			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			require.Equal(t, test.wantPath, req.URL.Path)
			require.Equal(t, test.wantRawPath, req.URL.RawPath)
			res := httptest.NewRecorder()

			mux.ServeHTTP(res, req)

			require.Equal(t, http.StatusOK, res.Code)
			require.Equal(t, "resend", server.request.GetAppId())
			require.Equal(t, test.functionID, server.request.GetFunctionId())
		})
	}
}

func TestListFunctionRunsRouteRejectsUnmatchedPaths(t *testing.T) {
	mux := runtime.NewServeMux()
	require.NoError(t, RegisterV2HandlerServer(context.Background(), mux, &functionRunsRouteServer{}))

	for _, path := range []string{
		"/apps/resend/functions/infra/anti-abuse.create-partition",
		"/apps/resend/functions/infra/anti-abuse.create-partition/runs/extra",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		res := httptest.NewRecorder()

		mux.ServeHTTP(res, req)

		require.Equal(t, http.StatusNotFound, res.Code)
	}
}
