package apiv2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
)

type buildRouteServer struct {
	UnimplementedV2Server
	created   *CreateBuildRequest
	committed *CommitBuildRequest
}

func (s *buildRouteServer) CreateBuild(_ context.Context, req *CreateBuildRequest) (*CreateBuildResponse, error) {
	s.created = req
	return &CreateBuildResponse{}, nil
}

func (s *buildRouteServer) CommitBuild(_ context.Context, req *CommitBuildRequest) (*BuildResponse, error) {
	s.committed = req
	return &BuildResponse{}, nil
}

func TestBuildRoutePathParameters(t *testing.T) {
	for _, tc := range []struct {
		name, path, body string
		commit           bool
	}{
		{"create", "/builds/python-tools", `{"tag":"latest","uploadSizeBytes":"123"}`, false},
		{"create path wins", "/builds/python-tools", `{"name":"other","tag":"latest","uploadSizeBytes":"123"}`, false},
		{"commit", "/builds/python-tools/build-id/commit", `{}`, true},
		{"commit path wins", "/builds/python-tools/build-id/commit", `{"name":"other","buildId":"other-id"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &buildRouteServer{}
			mux := runtime.NewServeMux()
			require.NoError(t, RegisterV2HandlerServer(context.Background(), mux, s))
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, req)
			require.Equal(t, http.StatusOK, res.Code, res.Body.String())
			if tc.commit {
				require.NotNil(t, s.committed)
				require.Equal(t, "python-tools", s.committed.Name)
				require.Equal(t, "build-id", s.committed.BuildId)
			} else {
				require.NotNil(t, s.created)
				require.Equal(t, "python-tools", s.created.Name)
				require.Equal(t, "latest", s.created.Tag)
				require.EqualValues(t, 123, s.created.UploadSizeBytes)
			}
		})
	}
}
