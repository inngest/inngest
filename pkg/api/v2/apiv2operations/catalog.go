package apiv2operations

import (
	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Catalog struct {
	Operations []Operation `json:"operations"`
}

type Operation struct {
	ID          string       `json:"id"`
	Summary     string       `json:"summary,omitempty"`
	Description string       `json:"description,omitempty"`
	HTTP        *HTTPBinding `json:"http,omitempty"`
	CLI         *CLIBinding  `json:"cli,omitempty"`
	MCP         *mcp.Tool    `json:"mcp,omitempty"`
}

type HTTPBinding struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type CLIBinding struct {
	Command string `json:"command"`
}

func New(endpoints []apiv2endpoint.Endpoint, tools []*mcp.Tool) Catalog {
	toolsByName := make(map[string]*mcp.Tool, len(tools))
	for _, tool := range tools {
		toolsByName[tool.Name] = tool
	}

	operations := make([]Operation, 0, len(tools))
	for _, endpoint := range endpoints {
		tool, ok := toolsByName[endpoint.ToolName]
		if !ok {
			continue
		}

		operations = append(operations, Operation{
			ID:          endpoint.MethodName,
			Summary:     endpoint.Summary,
			Description: endpoint.Description,
			HTTP: &HTTPBinding{
				Method: endpoint.HTTPMethod,
				Path:   endpoint.Path,
			},
			CLI: &CLIBinding{
				Command: endpoint.CommandName,
			},
			MCP: tool,
		})
		delete(toolsByName, endpoint.ToolName)
	}

	for _, tool := range tools {
		if _, ok := toolsByName[tool.Name]; !ok {
			continue
		}
		operations = append(operations, Operation{
			ID:          "mcp." + tool.Name,
			Summary:     tool.Title,
			Description: tool.Description,
			MCP:         tool,
		})
	}

	return Catalog{Operations: operations}
}
