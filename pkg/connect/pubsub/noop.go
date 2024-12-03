package pubsub

import (
	"context"

	"github.com/google/uuid"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

// noopConnector is a blank implementation of the Connector interface
type noopConnector struct{}

func (noopConnector) Proxy(ctx context.Context, appID uuid.UUID, data *connpb.GatewayExecutorRequestData) (*connpb.SDKResponse, error) {
	return &connpb.SDKResponse{}, nil
}

func (noopConnector) ReceiveExecutorMessages(ctx context.Context, onMessage func(byt []byte, data *connpb.GatewayExecutorRequestData)) error {
	return nil
}

func (noopConnector) RouteExecutorRequest(ctx context.Context, gatewayId string, appId uuid.UUID, connId string, data *connpb.GatewayExecutorRequestData) error {
	return nil
}

func (noopConnector) ReceiveRoutedRequest(ctx context.Context, gatewayId string, appId uuid.UUID, connId string, onMessage func(rawBytes []byte, data *connpb.GatewayExecutorRequestData)) error {
	return nil
}

func (noopConnector) AckMessage(ctx context.Context, appId uuid.UUID, requestId string, source AckSource) error {
	return nil
}

func (noopConnector) NotifyExecutor(ctx context.Context, appId uuid.UUID, resp *connpb.SDKResponse) error {
	return nil
}

func (noopConnector) Wait(ctx context.Context) error {
	return nil
}
