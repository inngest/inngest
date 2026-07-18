package apiv2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	openapiv2 "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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

type sandboxRouteServer struct {
	UnimplementedV2Server
	createRequest *CreateSandboxRequest
	getRequest    *GetSandboxRequest
	execRequest   *ExecSandboxRequest
	deleteRequest *DeleteSandboxRequest
}

func (s *sandboxRouteServer) CreateSandbox(_ context.Context, req *CreateSandboxRequest) (*CreateSandboxResponse, error) {
	s.createRequest = req
	return &CreateSandboxResponse{Data: &Sandbox{Id: req.VpcId, Name: req.Name}}, nil
}

func (s *sandboxRouteServer) GetSandbox(_ context.Context, req *GetSandboxRequest) (*GetSandboxResponse, error) {
	s.getRequest = req
	return &GetSandboxResponse{Data: &Sandbox{Id: req.SandboxId}}, nil
}

func (s *sandboxRouteServer) ExecSandbox(_ context.Context, req *ExecSandboxRequest) (*ExecSandboxResponse, error) {
	s.execRequest = req
	return &ExecSandboxResponse{Data: &ExecSandboxData{}}, nil
}

func (s *sandboxRouteServer) DeleteSandbox(_ context.Context, req *DeleteSandboxRequest) (*DeleteSandboxResponse, error) {
	s.deleteRequest = req
	return &DeleteSandboxResponse{Data: &Sandbox{Id: req.SandboxId}}, nil
}

func TestSandboxRoutesBindBodyPathAndQuery(t *testing.T) {
	server := &sandboxRouteServer{}
	mux := runtime.NewServeMux()
	require.NoError(t, RegisterV2HandlerServer(context.Background(), mux, server))

	const vpcID = "11111111-1111-1111-1111-111111111111"
	const sandboxID = "22222222-2222-2222-2222-222222222222"
	createBody := `{
		"vpcId":"11111111-1111-1111-1111-111111111111",
		"name":"test-sandbox",
		"profileId":"small",
		"imageId":"image-v1",
		"runtimeTimeoutSeconds":60,
		"command":["/bin/sh","-lc","sleep 60"],
		"environment":{"MODE":"test"},
		"secretReferences":[{"id":"33333333-3333-3333-3333-333333333333","environmentName":"TOKEN","versionRef":"1"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/sandboxes", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)
	require.True(t, proto.Equal(&CreateSandboxRequest{
		VpcId:                 vpcID,
		Name:                  "test-sandbox",
		ProfileId:             "small",
		ImageId:               "image-v1",
		RuntimeTimeoutSeconds: 60,
		Command:               []string{"/bin/sh", "-lc", "sleep 60"},
		Environment:           map[string]string{"MODE": "test"},
		SecretReferences: []*SandboxSecretReference{{
			Id:              "33333333-3333-3333-3333-333333333333",
			EnvironmentName: "TOKEN",
			VersionRef:      "1",
		}},
	}, server.createRequest))

	req = httptest.NewRequest(http.MethodGet, "/sandboxes/"+sandboxID, nil)
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)
	require.True(t, proto.Equal(&GetSandboxRequest{SandboxId: sandboxID}, server.getRequest))

	req = httptest.NewRequest(http.MethodPost, "/sandboxes/"+sandboxID+"/exec", strings.NewReader(`{"argv":["printf","%s","hello world"]}`))
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)
	require.True(t, proto.Equal(&ExecSandboxRequest{
		SandboxId: sandboxID,
		Argv:      []string{"printf", "%s", "hello world"},
	}, server.execRequest))

	req = httptest.NewRequest(http.MethodDelete, "/sandboxes/"+sandboxID+"?graceful=true", nil)
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)
	require.True(t, proto.Equal(&DeleteSandboxRequest{SandboxId: sandboxID, Graceful: true}, server.deleteRequest))

	req = httptest.NewRequest(http.MethodDelete, "/sandboxes/"+sandboxID, strings.NewReader(`{"graceful":true}`))
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)
	require.True(t, proto.Equal(&DeleteSandboxRequest{SandboxId: sandboxID}, server.deleteRequest), "DELETE body must not bind")
}

func TestSandboxOpenAPIAndAuthzMetadata(t *testing.T) {
	service := File_api_v2_service_proto.Services().ByName("V2")
	require.NotNil(t, service)

	tests := []struct {
		method protoreflect.Name
		status string
	}{
		{method: "CreateSandbox", status: "202"},
		{method: "GetSandbox", status: "200"},
		{method: "ExecSandbox", status: "200"},
		{method: "DeleteSandbox", status: "202"},
	}
	for _, test := range tests {
		t.Run(string(test.method), func(t *testing.T) {
			method := service.Methods().ByName(test.method)
			require.NotNil(t, method)
			require.True(t, proto.HasExtension(method.Options(), E_Authz))
			authz := proto.GetExtension(method.Options(), E_Authz).(*AuthzOptions)
			require.True(t, authz.RequireAuthz)

			require.True(t, proto.HasExtension(method.Options(), openapiv2.E_Openapiv2Operation))
			operation := proto.GetExtension(method.Options(), openapiv2.E_Openapiv2Operation).(*openapiv2.Operation)
			require.Contains(t, operation.Responses, test.status)
		})
	}

	execMethod := service.Methods().ByName("ExecSandbox")
	execOperation := proto.GetExtension(execMethod.Options(), openapiv2.E_Openapiv2Operation).(*openapiv2.Operation)
	require.Contains(t, execOperation.Responses["409"].Description, "non-replayable command is ambiguous")
}

func TestSandboxMessagesExposeOnlyPublicFields(t *testing.T) {
	fieldNames := func(message protoreflect.MessageDescriptor) []string {
		fields := message.Fields()
		names := make([]string, fields.Len())
		for i := 0; i < fields.Len(); i++ {
			names[i] = string(fields.Get(i).Name())
		}
		return names
	}

	request := (&CreateSandboxRequest{}).ProtoReflect().Descriptor()
	require.Equal(t, []string{
		"vpc_id",
		"name",
		"profile_id",
		"image_id",
		"runtime_timeout_seconds",
		"command",
		"environment",
		"secret_references",
	}, fieldNames(request))
	require.Equal(t, []string{"sandbox_id"}, fieldNames((&GetSandboxRequest{}).ProtoReflect().Descriptor()))
	require.Equal(t, []string{"sandbox_id", "argv"}, fieldNames((&ExecSandboxRequest{}).ProtoReflect().Descriptor()))
	require.Equal(t, []string{"sandbox_id", "graceful"}, fieldNames((&DeleteSandboxRequest{}).ProtoReflect().Descriptor()))

	sandbox := (&Sandbox{}).ProtoReflect().Descriptor()
	require.Equal(t, []string{"id", "name", "desired_state", "phase", "outcome", "cleanup_state"}, fieldNames(sandbox))
	require.True(t, sandbox.Fields().ByName("outcome").HasOptionalKeyword())

	execData := (&ExecSandboxData{}).ProtoReflect().Descriptor()
	require.Equal(t, []string{
		"stdout",
		"stderr",
		"exit_code",
		"duration_ms",
		"stdout_truncated",
		"stderr_truncated",
	}, fieldNames(execData))
	for i := 0; i < execData.Fields().Len(); i++ {
		require.True(t, execData.Fields().Get(i).HasOptionalKeyword())
	}

	for _, response := range []protoreflect.MessageDescriptor{
		(&CreateSandboxResponse{}).ProtoReflect().Descriptor(),
		(&GetSandboxResponse{}).ProtoReflect().Descriptor(),
		(&ExecSandboxResponse{}).ProtoReflect().Descriptor(),
		(&DeleteSandboxResponse{}).ProtoReflect().Descriptor(),
	} {
		require.Equal(t, []string{"data", "metadata"}, fieldNames(response))
	}

	require.Equal(t, "/api.v2.V2/CreateSandbox", V2_CreateSandbox_FullMethodName)
	require.Equal(t, "/api.v2.V2/GetSandbox", V2_GetSandbox_FullMethodName)
	require.Equal(t, "/api.v2.V2/ExecSandbox", V2_ExecSandbox_FullMethodName)
	require.Equal(t, "/api.v2.V2/DeleteSandbox", V2_DeleteSandbox_FullMethodName)
}
