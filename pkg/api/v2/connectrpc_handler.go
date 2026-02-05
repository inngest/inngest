package apiv2

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/inngest/inngest/proto/gen/api/v2/apiv2connect"
	"github.com/oklog/ulid/v2"
)

// ConnectRpcHandler adapts the Service to the apiv2connect.V2Handler interface.
// This allows the same Service implementation to be used for both grpc-gateway (REST)
// and ConnectRPC protocol requests.
type ConnectRpcHandler struct {
	apiv2connect.UnimplementedV2Handler
	service *Service
}

func NewConnectRpcHandler(service *Service) *ConnectRpcHandler {
	return &ConnectRpcHandler{service: service}
}

func (h *ConnectRpcHandler) Health(ctx context.Context, req *connect.Request[apiv2.HealthRequest]) (*connect.Response[apiv2.HealthResponse], error) {
	resp, err := h.service.Health(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *ConnectRpcHandler) CreatePartnerAccount(ctx context.Context, req *connect.Request[apiv2.CreateAccountRequest]) (*connect.Response[apiv2.CreateAccountResponse], error) {
	resp, err := h.service.CreatePartnerAccount(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *ConnectRpcHandler) CreateEnv(ctx context.Context, req *connect.Request[apiv2.CreateEnvRequest]) (*connect.Response[apiv2.CreateEnvResponse], error) {
	resp, err := h.service.CreateEnv(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *ConnectRpcHandler) FetchPartnerAccounts(ctx context.Context, req *connect.Request[apiv2.FetchAccountsRequest]) (*connect.Response[apiv2.FetchAccountsResponse], error) {
	resp, err := h.service.FetchPartnerAccounts(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *ConnectRpcHandler) FetchAccount(ctx context.Context, req *connect.Request[apiv2.FetchAccountRequest]) (*connect.Response[apiv2.FetchAccountResponse], error) {
	resp, err := h.service.FetchAccount(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *ConnectRpcHandler) FetchAccountEnvs(ctx context.Context, req *connect.Request[apiv2.FetchAccountEnvsRequest]) (*connect.Response[apiv2.FetchAccountEnvsResponse], error) {
	resp, err := h.service.FetchAccountEnvs(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *ConnectRpcHandler) FetchAccountEventKeys(ctx context.Context, req *connect.Request[apiv2.FetchAccountEventKeysRequest]) (*connect.Response[apiv2.FetchAccountEventKeysResponse], error) {
	resp, err := h.service.FetchAccountEventKeys(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *ConnectRpcHandler) FetchAccountSigningKeys(ctx context.Context, req *connect.Request[apiv2.FetchAccountSigningKeysRequest]) (*connect.Response[apiv2.FetchAccountSigningKeysResponse], error) {
	resp, err := h.service.FetchAccountSigningKeys(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *ConnectRpcHandler) CreateWebhook(ctx context.Context, req *connect.Request[apiv2.CreateWebhookRequest]) (*connect.Response[apiv2.CreateWebhookResponse], error) {
	resp, err := h.service.CreateWebhook(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *ConnectRpcHandler) ListWebhooks(ctx context.Context, req *connect.Request[apiv2.ListWebhooksRequest]) (*connect.Response[apiv2.ListWebhooksResponse], error) {
	resp, err := h.service.ListWebhooks(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *ConnectRpcHandler) PatchEnv(ctx context.Context, req *connect.Request[apiv2.PatchEnvRequest]) (*connect.Response[apiv2.PatchEnvsResponse], error) {
	resp, err := h.service.PatchEnv(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// StreamRun implements server-streaming for run trace updates.
// It polls the backend and sends updates over the stream until the run completes.
func (h *ConnectRpcHandler) StreamRun(ctx context.Context, req *connect.Request[apiv2.StreamRunRequest], stream *connect.ServerStream[apiv2.RunStreamItem]) error {
	if h.service.rpcProvider == nil {
		return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("run streaming not configured"))
	}

	envID, err := uuid.Parse(req.Msg.EnvId)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid env_id: %w", err))
	}

	runID, err := ulid.Parse(req.Msg.RunId)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid run_id: %w", err))
	}

	//
	// Get account ID from context (set by auth middleware)
	accountID, ok := ctx.Value("account_id").(uuid.UUID)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing account_id in context"))
	}

	//
	// Poll interval for streaming updates
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastStatus string

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			runData, err := h.service.rpcProvider.GetRunData(ctx, accountID, envID, runID)
			if err != nil {
				return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get run data: %w", err))
			}

			//
			// Send update to client
			if err := stream.Send(&apiv2.RunStreamItem{Run: runData}); err != nil {
				return err
			}

			//
			// Check if run has completed (stop streaming if terminal status)
			if runData.Status != lastStatus {
				lastStatus = runData.Status
			}

			//
			// Terminal statuses - stop streaming
			switch runData.Status {
			case "COMPLETED", "FAILED", "CANCELLED":
				return nil
			}
		}
	}
}
