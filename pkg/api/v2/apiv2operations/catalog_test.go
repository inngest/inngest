package apiv2operations

import (
	"testing"

	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	endpoints := []apiv2endpoint.Endpoint{{
		MethodName:  "GetFunctionRun",
		CommandName: "get-function-run",
		ToolName:    "get_run",
		HTTPMethod:  "GET",
		Path:        "/runs/{run_id}",
		Summary:     "Get a function run",
	}}
	tools := []*mcp.Tool{
		{Name: "get_run", InputSchema: map[string]any{"type": "object"}},
		{Name: "grep_docs", Title: "Search documentation", InputSchema: map[string]any{"type": "object"}},
	}

	catalog := New(endpoints, tools)

	require.Len(t, catalog.Operations, 2)
	require.Equal(t, "GetFunctionRun", catalog.Operations[0].ID)
	require.Equal(t, "GET", catalog.Operations[0].HTTP.Method)
	require.Equal(t, "get-function-run", catalog.Operations[0].CLI.Command)
	require.Equal(t, "get_run", catalog.Operations[0].MCP.Name)
	require.Equal(t, "mcp.grep_docs", catalog.Operations[1].ID)
	require.Nil(t, catalog.Operations[1].HTTP)
	require.Nil(t, catalog.Operations[1].CLI)
}
