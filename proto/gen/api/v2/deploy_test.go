package apiv2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

type deployRouteServer struct {
	UnimplementedV2Server
	created   *CreateDeployRequest
	committed *CommitDeployRequest
}

func TestDeployConfigJSON(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		invalid bool
		check   func(*testing.T, *DeployConfig)
	}{
		{
			name: "serve endpoint stays omitted",
			body: `{"config":{"serve":{}}}`,
			check: func(t *testing.T, config *DeployConfig) {
				require.NotNil(t, config.GetServe())
				require.Nil(t, config.Description)
				require.Nil(t, config.GetServe().Port)
				require.Nil(t, config.GetServe().Path)
			},
		},
		{
			name: "empty description retains presence",
			body: `{"config":{"description":"","serve":{}}}`,
			check: func(t *testing.T, config *DeployConfig) {
				require.NotNil(t, config.Description)
				require.Empty(t, config.GetDescription())
			},
		},
		{
			name: "serve settings stay omitted",
			body: `{"config":{"serve":{"port":8080,"path":"/api/inngest"}}}`,
			check: func(t *testing.T, config *DeployConfig) {
				require.EqualValues(t, 8080, config.GetServe().GetPort())
				require.Equal(t, "/api/inngest", config.GetServe().GetPath())
				require.Nil(t, config.ServeConfig)
			},
		},
		{
			name: "partial serve settings stay omitted",
			body: `{"config":{"serve":{"port":8080,"path":"/api/inngest"},"serve_config":{"multiplex":10,"idle":"15s"}}}`,
			check: func(t *testing.T, config *DeployConfig) {
				require.EqualValues(t, 10, config.ServeConfig.GetMultiplex())
				require.Equal(t, "15s", config.ServeConfig.GetIdle())
				require.Nil(t, config.ServeConfig.MaxRequests)
				require.Nil(t, config.ServeConfig.Lifetime)
			},
		},
		{
			name: "connect replicas stay omitted",
			body: `{"config":{"connect":{}}}`,
			check: func(t *testing.T, config *DeployConfig) {
				require.NotNil(t, config.GetConnect())
				require.Nil(t, config.GetConnect().Replicas)
			},
		},
		{
			name: "explicit zero replicas retain presence",
			body: `{"config":{"connect":{"replicas":0}}}`,
			check: func(t *testing.T, config *DeployConfig) {
				require.NotNil(t, config.GetConnect().Replicas)
				require.Zero(t, config.GetConnect().GetReplicas())
			},
		},
		{
			name:    "serve and connect are mutually exclusive",
			body:    `{"config":{"connect":{},"serve":{"port":8080,"path":"/api/inngest"}}}`,
			invalid: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var request CreateDeployRequest
			err := protojson.Unmarshal([]byte(tc.body), &request)
			if tc.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			encoded, err := protojson.Marshal(&request)
			require.NoError(t, err)
			var decoded CreateDeployRequest
			require.NoError(t, protojson.Unmarshal(encoded, &decoded))
			tc.check(t, decoded.Config)
		})
	}
}

func (s *deployRouteServer) CreateDeploy(_ context.Context, req *CreateDeployRequest) (*CreateDeployResponse, error) {
	s.created = req
	return &CreateDeployResponse{}, nil
}

func (s *deployRouteServer) CommitDeploy(_ context.Context, req *CommitDeployRequest) (*DeployResponse, error) {
	s.committed = req
	return &DeployResponse{}, nil
}

func TestDeployRoutePathParameters(t *testing.T) {
	for _, tc := range []struct {
		name   string
		path   string
		body   string
		commit bool
	}{
		{
			"create",
			"/deploys/python-tools",
			`{"tag":"latest","uploadSizeBytes":"123"}`,
			false,
		},
		{
			"create path wins",
			"/deploys/python-tools",
			`{"name":"other","tag":"latest","uploadSizeBytes":"123"}`,
			false,
		},
		{
			"commit",
			"/deploys/python-tools/deploy-id/commit",
			`{}`,
			true,
		},
		{
			"commit path wins",
			"/deploys/python-tools/deploy-id/commit",
			`{"name":"other","deployId":"other-id"}`,
			true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &deployRouteServer{}
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
				require.Equal(t, "deploy-id", s.committed.DeployId)
			} else {
				require.NotNil(t, s.created)
				require.Equal(t, "python-tools", s.created.Name)
				require.Equal(t, "latest", s.created.Tag)
				require.EqualValues(t, 123, s.created.UploadSizeBytes)
			}
		})
	}
}

func (s *deployRouteServer) GetDeploy(_ context.Context, req *GetDeployRequest) (*DeployResponse, error) {
	return &DeployResponse{
		Data: &Deploy{
			Id: req.DeployId,
		},
	}, nil
}

func (s *deployRouteServer) ListDeploys(context.Context, *ListDeploysRequest) (*ListDeploysResponse, error) {
	return &ListDeploysResponse{
		Data: []*Deploy{{
			Id: "listed",
		}},
	}, nil
}

func (s *deployRouteServer) CancelDeploy(_ context.Context, req *CancelDeployRequest) (*DeployResponse, error) {
	return &DeployResponse{
		Data: &Deploy{
			Id: req.DeployId,
		},
	}, nil
}

func TestDeployReadAndCancelRoutes(t *testing.T) {
	for _, tc := range []struct {
		method string
		path   string
		body   string
		id     string
	}{
		{
			http.MethodGet,
			"/deploys/deploy-id",
			"",
			"deploy-id",
		},
		{
			http.MethodGet,
			"/deploys",
			"",
			"listed",
		},
		{
			http.MethodPost,
			"/deploys/deploy-id/cancel",
			`{"deployId":"body-id"}`,
			"deploy-id",
		},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			mux := runtime.NewServeMux()
			require.NoError(t, RegisterV2HandlerServer(context.Background(), mux, &deployRouteServer{}))
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, req)
			require.Equal(t, http.StatusOK, res.Code, res.Body.String())
			require.Contains(t, res.Body.String(), tc.id)
		})
	}
}

func TestDeployUnimplemented(t *testing.T) {
	mux := runtime.NewServeMux()
	require.NoError(t, RegisterV2HandlerServer(context.Background(), mux, &UnimplementedV2Server{}))
	for _, tc := range []struct {
		method string
		path   string
	}{
		{
			http.MethodPost,
			"/deploys/name",
		},
		{
			http.MethodPost,
			"/deploys/name/deploy-id/commit",
		},
		{
			http.MethodGet,
			"/deploys/deploy-id",
		},
		{
			http.MethodGet,
			"/deploys",
		},
		{
			http.MethodPost,
			"/deploys/deploy-id/cancel",
		},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			body := ""
			if tc.method == http.MethodPost {
				body = `{}`
			}
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, req)
			require.Equal(t, http.StatusNotImplemented, res.Code, res.Body.String())
		})
	}
}
