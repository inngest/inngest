package connectdriver

import (
	"context"
	connect_sdk "github.com/inngest/inngestgo/connect"
)

type ProxyResponse struct {
	SdkResponse *connect_sdk.SdkResponse
}

type RequestForwarder interface {
	Proxy(ctx context.Context, data connect_sdk.GatewayMessageTypeExecutorRequestData) (*connect_sdk.SdkResponse, error)
}

type SubscribeFunc func(data connect_sdk.GatewayMessageTypeExecutorRequestData) error

type inProcessForwarder struct {
	subscribers map[string][]SubscribeFunc
}

func NewInProcessForwarder() RequestForwarder {
	return &inProcessForwarder{}
}

func (i inProcessForwarder) Proxy(ctx context.Context, data connect_sdk.GatewayMessageTypeExecutorRequestData) (*connect_sdk.SdkResponse, error) {
	//TODO implement me
	panic("implement me")
}
