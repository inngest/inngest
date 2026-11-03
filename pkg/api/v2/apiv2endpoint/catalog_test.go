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
	require.Equal(t, "functions:write:invoke", byMethod["InvokeFunction"].AuthzPermission)
	require.Equal(t, "functions:write:pause", byMethod["PauseFunction"].AuthzPermission)
	require.Equal(t, "pause-function", byMethod["PauseFunction"].CommandName)
	require.Equal(t, "pause_function", byMethod["PauseFunction"].ToolName)
	require.Equal(t, http.MethodPost, byMethod["PauseFunction"].HTTPMethod)
	require.Equal(t, "/apps/{app_id}/functions/{function_id}/pause", byMethod["PauseFunction"].Path)
	require.Empty(t, byMethod["PauseFunction"].Body)
	require.Equal(t, "functions:write:resume", byMethod["ResumeFunction"].AuthzPermission)
	require.Equal(t, "resume-function", byMethod["ResumeFunction"].CommandName)
	require.Equal(t, "resume_function", byMethod["ResumeFunction"].ToolName)
	require.Equal(t, http.MethodPost, byMethod["ResumeFunction"].HTTPMethod)
	require.Equal(t, "/apps/{app_id}/functions/{function_id}/resume", byMethod["ResumeFunction"].Path)
	require.Empty(t, byMethod["ResumeFunction"].Body)
	require.Equal(t, "runs:read:list", byMethod["ListRuns"].AuthzPermission)
	require.Equal(t, "runs:write:replay", byMethod["CreateReplay"].AuthzPermission)
	require.Equal(t, "create-replay", byMethod["CreateReplay"].CommandName)
	require.Equal(t, "create_replay", byMethod["CreateReplay"].ToolName)
	require.Equal(t, http.MethodPost, byMethod["CreateReplay"].HTTPMethod)
	require.Equal(t, "/apps/{app_id}/functions/{function_id}/replays", byMethod["CreateReplay"].Path)
	require.Equal(t, "*", byMethod["CreateReplay"].Body)
	require.Equal(t, "sandboxes:write:exec", byMethod["ExecSandbox"].AuthzPermission)
	require.Equal(t, "send-event", byMethod["SendEvent"].CommandName)
	require.Equal(t, "send_event", byMethod["SendEvent"].ToolName)
	require.Equal(t, http.MethodPost, byMethod["SendEvent"].HTTPMethod)
	require.Equal(t, "/events", byMethod["SendEvent"].Path)
	require.Equal(t, "events:write:send", byMethod["SendEvent"].AuthzPermission)
}

func TestEventReadCatalog(t *testing.T) {
	byMethod := map[string]Endpoint{}
	for _, endpoint := range Discover() {
		byMethod[endpoint.MethodName] = endpoint
	}
	for _, tc := range []struct{ method, command, tool, path, permission string }{
		{"ListEvents", "list-events", "list_events", "/events", "events:read:list"},
		{"GetEvent", "get-event", "get_event", "/events/{event_id}", "events:read:get"},
	} {
		endpoint := byMethod[tc.method]
		require.Equal(t, tc.command, endpoint.CommandName)
		require.Equal(t, tc.tool, endpoint.ToolName)
		require.Equal(t, http.MethodGet, endpoint.HTTPMethod)
		require.Equal(t, tc.path, endpoint.Path)
		require.Equal(t, tc.permission, endpoint.AuthzPermission)
		require.Equal(t, []string{"payload"}, FieldEnum(endpoint.Input.Fields().ByName("include")))
	}
}
