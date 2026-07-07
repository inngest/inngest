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
