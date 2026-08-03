package apiv2endpoint

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscover(t *testing.T) {
	endpoints := Discover()
	require.NotEmpty(t, endpoints)

	byMethod := map[string]Endpoint{}
	toolNames := map[string]string{}
	for _, endpoint := range endpoints {
		byMethod[endpoint.MethodName] = endpoint
		if method, ok := toolNames[endpoint.ToolName]; ok {
			t.Fatalf("tool name %q is shared by %s and %s", endpoint.ToolName, method, endpoint.MethodName)
		}
		toolNames[endpoint.ToolName] = endpoint.MethodName
	}

	require.NotContains(t, byMethod, "_SchemaOnly")
	require.NotContains(t, byMethod, "CreatePartnerAccount")
	require.NotContains(t, byMethod, "FetchPartnerAccounts")
	require.Contains(t, byMethod, "ListFunctionRuns")
	require.True(t, byMethod["ListRuns"].CommandNameExplicit)
	require.False(t, byMethod["ListFunctionRuns"].CommandNameExplicit)
	require.Equal(t, byMethod["ListRuns"].CommandName, byMethod["ListFunctionRuns"].CommandName)
	require.Equal(t, "list_envs", byMethod["FetchAccountEnvs"].ToolName)
	require.Equal(t, "get-account-envs", byMethod["FetchAccountEnvs"].CommandName)
	require.Equal(t, "list_functions", byMethod["GetFunctions"].ToolName)
	require.Equal(t, "get_run", byMethod["GetFunctionRun"].ToolName)
	require.Equal(t, "get_run_trace", byMethod["GetFunctionTrace"].ToolName)
	require.Equal(t, http.MethodPost, byMethod["InvokeFunction"].HTTPMethod)
	require.Equal(t, "/apps/{app_id}/functions/{function_id}/invoke", byMethod["InvokeFunction"].Path)
}
