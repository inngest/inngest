package connect

import (
	"context"
	connect_sdk "github.com/inngest/inngestgo/connect"
	"github.com/redis/rueidis"
)

type ProxyResponse struct {
	SdkResponse *connect_sdk.SdkResponse
}

type RequestForwarder interface {
	Proxy(ctx context.Context, data connect_sdk.GatewayMessageTypeExecutorRequestData) (*connect_sdk.SdkResponse, error)
}

type RequestReceiver interface {
	OnProxy(ctx context.Context, data connect_sdk.GatewayMessageTypeExecutorRequestData) (*connect_sdk.SdkResponse, error)
}

type redisPubSubConnector struct {
	client rueidis.Client

	RequestForwarder
	RequestReceiver
}

func NewRedisPubSubConnector(client rueidis.Client) *redisPubSubConnector {
	return &redisPubSubConnector{
		client: client,
	}
}

func (i redisPubSubConnector) Proxy(ctx context.Context, data connect_sdk.GatewayMessageTypeExecutorRequestData) (*connect_sdk.SdkResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (i redisPubSubConnector) OnProxy(ctx context.Context, data connect_sdk.GatewayMessageTypeExecutorRequestData) (*connect_sdk.SdkResponse, error) {
	//TODO implement me
	panic("implement me")
}
