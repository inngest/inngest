package apiv2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSandboxesNotImplementedInOSS(t *testing.T) {
	service := NewService(ServiceOptions{})
	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "CreateSandbox",
			call: func() error {
				response, err := service.CreateSandbox(context.Background(), &apiv2.CreateSandboxRequest{})
				require.Nil(t, response)
				return err
			},
		},
		{
			name: "GetSandbox",
			call: func() error {
				response, err := service.GetSandbox(context.Background(), &apiv2.GetSandboxRequest{})
				require.Nil(t, response)
				return err
			},
		},
		{
			name: "ExecSandbox",
			call: func() error {
				response, err := service.ExecSandbox(context.Background(), &apiv2.ExecSandboxRequest{})
				require.Nil(t, response)
				return err
			},
		},
		{
			name: "DeleteSandbox",
			call: func() error {
				response, err := service.DeleteSandbox(context.Background(), &apiv2.DeleteSandboxRequest{})
				require.Nil(t, response)
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.call()
			require.Equal(t, codes.Unimplemented, status.Code(err))
			st := status.Convert(err)
			require.JSONEq(t, `{"errors":[{"code":"not_implemented","message":"Sandboxes are not implemented in OSS"}]}`, st.Message())
		})
	}
}

func TestHTTPGateway_SandboxesNotImplementedInOSS(t *testing.T) {
	handler, err := newTestHTTPHandler(context.Background(), ServiceOptions{}, HTTPHandlerOptions{})
	require.NoError(t, err)

	const vpcID = "11111111-1111-1111-1111-111111111111"
	const sandboxName = "test-sandbox"
	validCreateBody := `{
		"vpcId":"11111111-1111-1111-1111-111111111111",
		"name":"test-sandbox",
		"profileId":"small",
		"imageId":"image-v1",
		"runtimeTimeoutSeconds":60,
		"command":[],
		"environment":{},
		"secretReferences":[]
	}`
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create", method: http.MethodPost, path: "/api/v2/sandboxes", body: validCreateBody},
		{name: "get", method: http.MethodGet, path: "/api/v2/vpcs/" + vpcID + "/sandboxes/" + sandboxName},
		{name: "exec", method: http.MethodPost, path: "/api/v2/vpcs/" + vpcID + "/sandboxes/" + sandboxName + "/exec", body: `{"argv":["printf","%s","hello world"]}`},
		{name: "delete", method: http.MethodDelete, path: "/api/v2/vpcs/" + vpcID + "/sandboxes/" + sandboxName + "?graceful=true"},
		{name: "malformed VPC UUID reaches OSS method", method: http.MethodGet, path: "/api/v2/vpcs/not-a-uuid/sandboxes/" + sandboxName},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			if test.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			require.Equal(t, http.StatusNotImplemented, response.Code)
			require.JSONEq(t, `{"errors":[{"code":"not_implemented","message":"Sandboxes are not implemented in OSS"}]}`, response.Body.String())
		})
	}
}

func TestHTTPGateway_SandboxGeneratedTransportErrors(t *testing.T) {
	handler, err := newTestHTTPHandler(context.Background(), ServiceOptions{}, HTTPHandlerOptions{})
	require.NoError(t, err)

	t.Run("malformed JSON", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/v2/vpcs/11111111-1111-1111-1111-111111111111/sandboxes/test-sandbox/exec", strings.NewReader(`{"argv":`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		require.Equal(t, http.StatusBadRequest, response.Code)
		var body apiv2base.ErrorResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.NotEmpty(t, body.Errors)
	})

	t.Run("unsupported method", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPut, "/api/v2/vpcs/11111111-1111-1111-1111-111111111111/sandboxes/test-sandbox", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		require.Equal(t, http.StatusNotImplemented, response.Code)
	})

	t.Run("private field", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/v2/sandboxes", strings.NewReader(`{
			"vpcId":"11111111-1111-1111-1111-111111111111",
			"name":"test-sandbox",
			"profileId":"small",
			"imageId":"image-v1",
			"runtimeTimeoutSeconds":60,
			"command":[],
			"environment":{},
			"secretReferences":[],
			"accountId":"33333333-3333-3333-3333-333333333333"
		}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("malformed graceful query", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodDelete, "/api/v2/vpcs/11111111-1111-1111-1111-111111111111/sandboxes/test-sandbox?graceful=not-a-bool", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})
}

func TestHTTPGateway_SandboxRoutesRequireAuthz(t *testing.T) {
	authzCalls := 0
	authz := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			authzCalls++
			writer.WriteHeader(http.StatusForbidden)
		})
	}
	handler, err := newTestHTTPHandler(context.Background(), ServiceOptions{}, HTTPHandlerOptions{AuthzMiddleware: authz})
	require.NoError(t, err)

	requests := []*http.Request{
		httptest.NewRequest(http.MethodPost, "/api/v2/sandboxes", strings.NewReader(`{}`)),
		httptest.NewRequest(http.MethodGet, "/api/v2/vpcs/11111111-1111-1111-1111-111111111111/sandboxes/test-sandbox", nil),
		httptest.NewRequest(http.MethodPost, "/api/v2/vpcs/11111111-1111-1111-1111-111111111111/sandboxes/test-sandbox/exec", strings.NewReader(`{"argv":["true"]}`)),
		httptest.NewRequest(http.MethodDelete, "/api/v2/vpcs/11111111-1111-1111-1111-111111111111/sandboxes/test-sandbox", nil),
	}
	requests[0].Header.Set("Content-Type", "application/json")
	requests[2].Header.Set("Content-Type", "application/json")
	for _, request := range requests {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		require.Equal(t, http.StatusForbidden, response.Code)
	}
	require.Equal(t, len(requests), authzCalls)
}
