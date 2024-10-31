package pubsub

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	connect_sdk "github.com/inngest/inngestgo/connect"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"sync"
	"time"
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
	NotifyExecutor(ctx context.Context, appId uuid.UUID, resp ProxyResponse) error
	AckMessage(ctx context.Context, appId uuid.UUID, requestId string) error

	Wait(ctx context.Context) error
}

type redisPubSubConnector struct {
	client       rueidis.Client
	pubSubClient rueidis.DedicatedClient

	subscribers     map[string]map[string]chan string
	subscribersLock sync.RWMutex

	RequestForwarder
	RequestReceiver
}

func NewRedisPubSubConnector(client rueidis.Client) *redisPubSubConnector {
	return &redisPubSubConnector{
		client:          client,
		subscribers:     make(map[string]map[string]chan string),
		subscribersLock: sync.RWMutex{},
	}
}

func (i *redisPubSubConnector) Proxy(ctx context.Context, data connect_sdk.GatewayMessageTypeExecutorRequestData) (*connect_sdk.SdkResponse, error) {
	if data.RequestId == "" {
		data.RequestId = ulid.MustNew(ulid.Now(), rand.Reader).String()
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("could not marshal executor request: %w", err)
	}

	ackErrChan := make(chan error)
	var acked bool
	{
		withAckTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		go func() {
			err = i.subscribe(withAckTimeout, i.channelAppRequestsAck(data.AppId, data.RequestId), func(msg string) {
				acked = true
			}, true)
			ackErrChan <- err
		}()
	}

	replyErrChan := make(chan error)
	var reply ProxyResponse
	go func() {
		// This may take a while: This waits until we receive the SDK response, and we allow for up to 2h in the serverless execution model
		err = i.subscribe(ctx, i.channelAppRequestsReply(data.AppId, data.RequestId), func(msg string) {
			err := json.Unmarshal([]byte(msg), &reply)
			if err != nil {
				// TODO This should never happen, push message into dead-letter channel and report
			}
		}, true)
		replyErrChan <- err
	}()

	channelName := i.channelAppRequests(data.AppId)
	fmt.Println("published to", channelName, data.RequestId)
	err = i.client.Do(ctx, i.client.B().Publish().Channel(channelName).Message(string(dataBytes)).Build()).Error()
	if err != nil {
		return nil, fmt.Errorf("could not publish executor request: %w", err)
	}

	// Sanity check: Ensure the gateway received the message using a request-specific ack channel
	{
		err := <-ackErrChan
		close(ackErrChan)
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("could not receive executor request ack: %w", err)
		}

		if !acked {
			return nil, fmt.Errorf("gateway did not ack in time")
		}
	}

	{
		err := <-replyErrChan
		close(replyErrChan)
		if err != nil {
			return nil, fmt.Errorf("could not receive executor response: %w", err)
		}
	}

	return reply.SdkResponse, nil
}

// channelAppRequests returns the channel name for executor requests for a specific app.
func (i *redisPubSubConnector) channelAppRequests(appId uuid.UUID) string {
	return fmt.Sprintf("app_requests:%s", appId)
}

func (i *redisPubSubConnector) channelAppRequestsAck(appId uuid.UUID, requestId string) string {
	return fmt.Sprintf("app_requests_ack:%s:%s", appId, requestId)
}

func (i *redisPubSubConnector) channelAppRequestsReply(appId uuid.UUID, requestId string) string {
	return fmt.Sprintf("app_requests_reply:%s:%s", appId, requestId)
}

func (i *redisPubSubConnector) subscribe(ctx context.Context, channel string, onMessage func(msg string), once bool) error {
	msgs := make(chan string)

	subId := ulid.MustNew(ulid.Now(), rand.Reader).String()

	i.subscribersLock.Lock()

	if _, ok := i.subscribers[channel]; !ok {
		// subscribe to channel
		i.subscribers[channel] = make(map[string]chan string)
	}

	i.subscribers[channel][subId] = msgs

	i.subscribersLock.Unlock()

	// This function is blocking, so whenever we return, we want to clean up the subscription handler and potentially
	// remove the subscription, if it's no longer used.
	defer func() {
		i.subscribersLock.Lock()
		defer i.subscribersLock.Unlock()

		close(msgs)
		fmt.Println("removing subscription", channel, subId)
		delete(i.subscribers[channel], subId)
		if len(i.subscribers[channel]) == 0 {
			fmt.Println("unsubscribing from", channel)
			delete(i.subscribers, channel)
			i.pubSubClient.Do(ctx, i.pubSubClient.B().Unsubscribe().Channel(channel).Build())
		}
	}()

	done := make(chan struct{})
	go func() {
		defer func() {
			done <- struct{}{}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-msgs:
				onMessage(msg)
				if once {
					return
				}
			}
		}
	}()

	i.pubSubClient.Do(ctx, i.pubSubClient.B().Subscribe().Channel(channel).Build())

	fmt.Println("subscribed to", channel)

	<-done
	return nil
}

func (i *redisPubSubConnector) ReceiveExecutorMessages(ctx context.Context, appId uuid.UUID, onMessage func(data connect_sdk.GatewayMessageTypeExecutorRequestData)) error {
	return i.subscribe(ctx, i.channelAppRequests(appId), func(msg string) {
		var data connect_sdk.GatewayMessageTypeExecutorRequestData
		err := json.Unmarshal([]byte(msg), &data)
		if err != nil {
			// TODO This should never happen, but PubSub will not redeliver, should we push the message into a dead-letter channel?
			return
		}

		onMessage(data)
	}, false)
}

func (i *redisPubSubConnector) Wait(ctx context.Context) error {
	c, cancel := i.client.Dedicate()
	defer cancel()

	// TODO: Check whether this graceful shutdown routine makes sense here
	go func() {
		<-ctx.Done()

		fmt.Println("gracefully shutting down pubsub subscriber")

		i.subscribersLock.Lock()
		defer i.subscribersLock.Unlock()

		// TODO Should we prevent other executors from subscribing while we're in "shutting down" state?

		// Unsubscribe from all channels
		subs := i.subscribers
		for channelName, _ := range subs {
			c.Do(ctx, c.B().Unsubscribe().Channel(channelName).Build())
		}

		c.Close()
	}()

	i.pubSubClient = c

	wait := c.SetPubSubHooks(rueidis.PubSubHooks{
		OnMessage: func(m rueidis.PubSubMessage) {
			fmt.Println("received message", m.Channel)

			// Handle the message. Note that if you want to call another `c.Do()` here, you need to do it in another goroutine or the `c` will be blocked.
			go func() {
				i.subscribersLock.RLock()
				subs := i.subscribers[m.Channel]
				i.subscribersLock.RUnlock()

				for _, receiverChan := range subs {
					receiverChan <- m.Message
				}
			}()
		},
	})
	err := <-wait // disconnected with err
	if err != nil {
		return err
	}

	return nil
}

func (i *redisPubSubConnector) NotifyExecutor(ctx context.Context, appId uuid.UUID, resp ProxyResponse) error {
	serialized, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("could not serialize response: %w", err)
	}

	channelName := i.channelAppRequestsReply(appId, resp.SdkResponse.RequestId)

	err = i.client.Do(
		ctx,
		i.client.B().
			Publish().
			Channel(channelName).
			Message(string(serialized)).
			Build()).
		Error()
	if err != nil {
		return fmt.Errorf("could not publish response: %w", err)
	}

	fmt.Println("sent reply to", channelName)

	return nil
}

func (i *redisPubSubConnector) AckMessage(ctx context.Context, appId uuid.UUID, requestId string) error {
	err := i.client.Do(
		ctx,
		i.client.B().
			Publish().
			Channel(i.channelAppRequestsAck(appId, requestId)).
			Message(time.Now().Format(time.RFC3339)).
			Build()).
		Error()
	if err != nil {
		return fmt.Errorf("could not publish response: %w", err)
	}

	return nil
}
