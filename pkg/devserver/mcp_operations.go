package devserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2operations"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type operationsResponse struct {
	Data apiv2operations.Catalog `json:"data"`
}

func (h *MCPHandler) Operations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Del("Access-Control-Allow-Credentials")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	content, err := h.operationCatalog()
	if err != nil {
		http.Error(w, "unable to list operations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=3600")
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(content)
}

func (h *MCPHandler) operationCatalog() ([]byte, error) {
	h.operationsMu.Lock()
	defer h.operationsMu.Unlock()
	if h.operationsJSON != nil {
		return h.operationsJSON, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tools, err := listServerTools(ctx, h.getMCPServer())
	if err != nil {
		return nil, err
	}

	content, err := json.Marshal(operationsResponse{
		Data: apiv2operations.New(h.endpoints, tools),
	})
	if err != nil {
		return nil, err
	}
	h.operationsJSON = content
	return h.operationsJSON, nil
}

func listServerTools(ctx context.Context, server *mcp.Server) ([]*mcp.Tool, error) {
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, err
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "inngest-operation-catalog",
		Version: "1.0.0",
	}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return nil, err
	}
	defer clientSession.Close()

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	return result.Tools, nil
}
