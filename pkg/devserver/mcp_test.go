package devserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestMCPToolsMatchGeneratedContractWithCompatibilityTools(t *testing.T) {
	handler := NewMCPHandler(nil, nil, 0, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	tools := listMCPTools(t, handler)

	expected := map[string]struct{}{
		"grep_docs":            {},
		"read_doc":             {},
		"list_docs":            {},
		"send_event":           {},
		"get_run_status":       {},
		"poll_run_status":      {},
		"invoke_function_sync": {},
	}
	generated := map[string]struct{}{}
	for _, endpoint := range apiv2endpoint.Discover() {
		if _, unsupported := unsupportedDevServerMCPMethods[endpoint.MethodName]; unsupported {
			continue
		}
		expected[endpoint.ToolName] = struct{}{}
		generated[endpoint.ToolName] = struct{}{}
	}

	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
		if _, ok := generated[tool.Name]; ok {
			require.IsType(t, []any{}, tool.InputSchema["required"], tool.Name)
			properties := tool.InputSchema["properties"].(map[string]any)
			require.NotContains(t, properties, "env", tool.Name)
		}
	}
	require.Len(t, names, len(expected))
	for name := range expected {
		require.Contains(t, names, name)
	}
	require.Contains(t, names, "list_functions")
	require.Contains(t, names, "invoke_function")
	for method := range unsupportedDevServerMCPMethods {
		require.NotContains(t, names, apiv2endpoint.ToolName(method))
	}

	titles := map[string]string{}
	descriptions := map[string]string{}
	for _, tool := range tools {
		titles[tool.Name] = tool.Title
		descriptions[tool.Name] = tool.Description
	}
	require.Equal(t, "Search documentation", titles["grep_docs"])
	require.Equal(t, "Read documentation", titles["read_doc"])
	require.Equal(t, "List documentation", titles["list_docs"])
	for _, name := range []string{
		"send_event",
		"get_run_status",
		"poll_run_status",
		"invoke_function_sync",
	} {
		require.Contains(t, strings.ToLower(titles[name]), "deprecated", name)
		require.Contains(t, strings.ToLower(descriptions[name]), "deprecated", name)
	}
}

func TestMCPRoutesDoNotCaptureSetupPage(t *testing.T) {
	router := chi.NewRouter()
	AddMCPRoute(router, nil, nil, 0, nil)
	router.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	req := httptest.NewRequest(http.MethodGet, "/mcp/setup", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusTeapot, recorder.Code)
}

func TestMCPBrowserRequests(t *testing.T) {
	handler := NewMCPHandler(nil, nil, 0, nil)
	body := `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"initialize",
		"params":{
			"protocolVersion":"2025-06-18",
			"capabilities":{},
			"clientInfo":{"name":"test","version":"1.0.0"}
		}
	}`

	for _, tt := range []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "MCP endpoint", path: "/mcp", wantStatus: http.StatusOK},
		{name: "other endpoint", path: "/other", wantStatus: http.StatusForbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(body))
			req.Header.Set("Accept", "application/json, text/event-stream")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "https://example.com")
			req.Header.Set("Sec-Fetch-Site", "cross-site")
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			require.Equal(t, tt.wantStatus, recorder.Code)
		})
	}
}

func TestMCPNotImplementedIsToolError(t *testing.T) {
	handler := &MCPHandler{v2: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte(`{"errors":[{"code":"not_implemented","message":"Sessions not implemented in OSS"}]}`))
	})}
	req := httptest.NewRequest(http.MethodGet, "/api/v2/sessions", nil)

	result, err := handler.executeV2(context.Background(), apiv2endpoint.Endpoint{}, req)
	require.NoError(t, err)
	require.True(t, result.IsError)
	require.Contains(t, result.Content[0].(*mcp.TextContent).Text, "HTTP 501")
	require.Equal(t, map[string]any{
		"errors": []any{map[string]any{
			"code":    "not_implemented",
			"message": "Sessions not implemented in OSS",
		}},
	}, result.StructuredContent)
}

type listedMCPTool struct {
	Name        string         `json:"name"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func listMCPTools(t *testing.T, handler http.Handler) []listedMCPTool {
	t.Helper()

	call := func(body, sessionID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Content-Type", "application/json")
		if sessionID != "" {
			req.Header.Set("Mcp-Session-Id", sessionID)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		return recorder
	}

	initialize := call(`{
		"jsonrpc":"2.0",
		"id":1,
		"method":"initialize",
		"params":{
			"protocolVersion":"2025-06-18",
			"capabilities":{},
			"clientInfo":{"name":"test","version":"1.0.0"}
		}
	}`, "")
	require.Equal(t, http.StatusOK, initialize.Code)
	var initializeResponse struct {
		Result struct {
			Capabilities struct {
				Tools map[string]any `json:"tools"`
			} `json:"capabilities"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(initialize.Body.Bytes(), &initializeResponse))
	require.NotNil(t, initializeResponse.Result.Capabilities.Tools)
	require.NotContains(t, initializeResponse.Result.Capabilities.Tools, "listChanged")
	sessionID := initialize.Header().Get("Mcp-Session-Id")
	require.NotEmpty(t, sessionID)

	initialized := call(`{"jsonrpc":"2.0","method":"notifications/initialized"}`, sessionID)
	require.Less(t, initialized.Code, http.StatusBadRequest)
	recorder := call(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, sessionID)
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Result struct {
			Tools []listedMCPTool `json:"tools"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return response.Result.Tools
}
