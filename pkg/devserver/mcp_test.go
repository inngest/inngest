package devserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPToolInputSchemasDoNotUseBooleanProperties(t *testing.T) {
	ctx := context.Background()
	server := (&MCPHandler{}).createMCPServer()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("connect server: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Fatalf("close session: %v", err)
		}
	})

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	for _, tool := range result.Tools {
		var inputSchema struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		schemaJSON, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatalf("marshal %s input schema: %v", tool.Name, err)
		}
		if err := json.Unmarshal(schemaJSON, &inputSchema); err != nil {
			t.Fatalf("unmarshal %s input schema: %v", tool.Name, err)
		}

		for name, propertySchema := range inputSchema.Properties {
			var isBoolean bool
			if err := json.Unmarshal(propertySchema, &isBoolean); err == nil {
				t.Fatalf("%s.%s input schema is boolean: %s", tool.Name, name, propertySchema)
			}
		}
	}
}
