package connect

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	connect_sdk "github.com/inngest/inngestgo/connect"
	"github.com/redis/rueidis"
)

type ProxyResponse struct {
	Status string

	SdkResponse *connect_sdk.SdkResponse
}

type RequestForwarder interface {
	Proxy(ctx context.Context, data connect_sdk.GatewayMessageTypeExecutorRequestData) (*connect_sdk.SdkResponse, error)
}

type RequestReceiver interface {
	ReceiveExecutorMessages(ctx context.Context, appId uuid.UUID, onMessage func(data connect_sdk.GatewayMessageTypeExecutorRequestData)) error
	NotifyExecutor(ctx context.Context, replyId string, resp ProxyResponse) error

	Wait() error
}

type redisPubSubConnector struct {
	client       rueidis.Client
	pubSubClient rueidis.DedicatedClient

	subscribers map[uuid.UUID][]chan connect_sdk.GatewayMessageTypeExecutorRequestData

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

func (i redisPubSubConnector) channelAppRequests(appId uuid.UUID) string {
	return fmt.Sprintf("app_requests:%s", appId)
}

func (i redisPubSubConnector) ReceiveExecutorMessages(ctx context.Context, appId uuid.UUID, onMessage func(data connect_sdk.GatewayMessageTypeExecutorRequestData)) error {
	msgs := make(chan connect_sdk.GatewayMessageTypeExecutorRequestData)

	if _, ok := i.subscribers[appId]; !ok {
		// subscribe to channel
		i.pubSubClient.Do(ctx, i.pubSubClient.B().Subscribe().Channel(i.channelAppRequests(appId)).Build())
		i.subscribers[appId] = make([]chan connect_sdk.GatewayMessageTypeExecutorRequestData, 0)
	}

	i.subscribers[appId] = append(i.subscribers[appId], msgs)

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-msgs:
			onMessage(msg)
		}
	}

	return nil
}

func (i redisPubSubConnector) Wait() error {
	c, cancel := i.client.Dedicate()
	defer cancel()

	i.pubSubClient = c

	wait := c.SetPubSubHooks(rueidis.PubSubHooks{
		OnMessage: func(m rueidis.PubSubMessage) {
			// Handle the message. Note that if you want to call another `c.Do()` here, you need to do it in another goroutine or the `c` will be blocked.
			go func() {

			}()
		},
	})
	c.Do(ctx, c.B().Subscribe().Channel("ch").Build())
	err := <-wait // disconnected with err
}
