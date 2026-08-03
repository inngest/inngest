package apiv2mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestRequest(t *testing.T) {
	endpoint := endpointByMethod(t, "InvokeFunction")
	req, err := Request(context.Background(), endpoint, map[string]any{
		"env":            "branch-a",
		"appId":          "my app",
		"functionId":     "hello/world",
		"data":           map[string]any{"message": "hi"},
		"idempotencyKey": "idem-1",
	}, Options{EnvHeader: "X-Inngest-Env"})
	require.NoError(t, err)
	require.Equal(t, http.MethodPost, req.Method)
	require.Equal(t, "/api/v2/apps/my%20app/functions/hello%2Fworld/invoke", req.URL.EscapedPath())
	require.Equal(t, "branch-a", req.Header.Get("X-Inngest-Env"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
	require.Equal(t, map[string]any{
		"data":           map[string]any{"message": "hi"},
		"idempotencyKey": "idem-1",
	}, body)
}

func TestInputSchema(t *testing.T) {
	endpoint := endpointByMethod(t, "InvokeFunction")
	schema := InputSchema(endpoint)
	properties := schema["properties"].(map[string]any)
	require.Contains(t, properties, "env")
	require.Contains(t, properties, "appId")
	require.Contains(t, properties, "functionId")
	require.Contains(t, properties, "data")
	require.ElementsMatch(t, []string{"appId", "functionId"}, schema["required"])
}

func TestToolAnnotations(t *testing.T) {
	for _, tt := range []struct {
		method      string
		readOnly    bool
		destructive bool
	}{
		{method: "GetFunctions", readOnly: true, destructive: false},
		{method: "InvokeFunction", readOnly: false, destructive: true},
		{method: "CancelRun", readOnly: false, destructive: true},
	} {
		t.Run(tt.method, func(t *testing.T) {
			annotations := Tool(endpointByMethod(t, tt.method)).Annotations
			require.Equal(t, tt.readOnly, annotations.ReadOnlyHint)
			require.Equal(t, tt.destructive, *annotations.DestructiveHint)
		})
	}
}

func TestToolOmitsEnvWithoutHeader(t *testing.T) {
	endpoint := endpointByMethod(t, "GetFunctions")
	schema := tool(endpoint, false).InputSchema.(map[string]any)
	properties := schema["properties"].(map[string]any)
	require.NotContains(t, properties, "env")

	req, err := Request(context.Background(), endpoint, map[string]any{
		"env":   "branch-a",
		"appId": "my-app",
	}, Options{})
	require.NoError(t, err)
	require.Empty(t, req.Header.Get("X-Inngest-Env"))
}

func TestCallPreservesIntegerArguments(t *testing.T) {
	endpoint := endpointByMethod(t, "GetFunctions")
	var gotQuery string
	result, err := call(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(`{"appId":"my-app","limit":1000000}`),
		},
	}, endpoint, Options{
		Execute: func(_ context.Context, _ apiv2endpoint.Endpoint, req *http.Request) (*mcp.CallToolResult, error) {
			gotQuery = req.URL.RawQuery
			return ToolResult(map[string]any{}, "{}"), nil
		},
	})

	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Equal(t, "limit=1000000", gotQuery)
}

func TestEndpointInputsDoNotUseReservedEnvField(t *testing.T) {
	for _, endpoint := range apiv2endpoint.Discover() {
		require.Nil(t, endpoint.Input.Fields().ByJSONName("env"), endpoint.MethodName)
	}
}

func TestRegisterExcludesEndpoints(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	registered := Register(server, Options{
		Exclude: func(endpoint apiv2endpoint.Endpoint) bool {
			return endpoint.MethodName == "ListSessions"
		},
	})

	methods := make([]string, 0, len(registered))
	for _, endpoint := range registered {
		methods = append(methods, endpoint.MethodName)
	}
	require.NotContains(t, methods, "ListSessions")
	require.Contains(t, methods, "GetFunctions")
}

func endpointByMethod(t *testing.T, method string) apiv2endpoint.Endpoint {
	t.Helper()
	for _, endpoint := range apiv2endpoint.Discover() {
		if endpoint.MethodName == method {
			return endpoint
		}
	}
	t.Fatalf("endpoint %s not found", method)
	return apiv2endpoint.Endpoint{}
}
