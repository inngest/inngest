package apiv2

import (
	"context"

	"connectrpc.com/connect"
	"github.com/inngest/inngest/proto/gen/api/v2/apiv2connect"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
)

//
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
