package pubsub

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"google.golang.org/protobuf/proto"
)

const (
	pkgName = "connect.execution.proxy"
)

/*
	This package provides a PubSub-based request forwarding mechanism for the Connect SDK.

	Execution requests are forwarded from the executor to the SDK via the connect infrastructure, including the following components:

	Executor -> Router -> Gateway -> SDK

	The router is responsible for selecting a gateway with an active, healthy connection for a given app. It is only responsible for
	routing requests to the correct gateway, not for returning the SDK response back to the executor. This is directly handled by the gateway.

	The gateway will acknowledge the request, forward it to the SDK, and return the response to the executor.
*/

type RequestForwarder interface {
	// Proxy forwards a request from the executor to the SDK via the connect infrastructure and waits for a response.
	//
	// If no responsible gateway ack's the message within a 10-second timeout, an error is returned.
	// If no response is received before the context is canceled, an error is returned.
	Proxy(ctx, traceCtx context.Context, appId uuid.UUID, data *connect.GatewayExecutorRequestData) (*connect.SDKResponse, error)
}

type AckSource string

const (
	AckSourceWorker  AckSource = "worker"
	AckSourceGateway AckSource = "gateway"
	AckSourceRouter  AckSource = "router"
)

type ResponseNotifier interface {
	// NotifyExecutor sends a response to the executor for a specific request.
	NotifyExecutor(ctx context.Context, resp *connect.SDKResponse) error
}

type RequestReceiver interface {
	ResponseNotifier

	// ReceiveExecutorMessages listens for incoming PubSub messages for the connect router.
	// This is a blocking call which only stops once the context is canceled.
	ReceiveExecutorMessages(ctx context.Context, onMessage func(rawBytes []byte, data *connect.GatewayExecutorRequestData)) error

	// RouteExecutorRequest forwards an executor request to the respective gateway
	RouteExecutorRequest(ctx context.Context, gatewayId ulid.ULID, appId uuid.UUID, connId ulid.ULID, data *connect.GatewayExecutorRequestData) error

	// ReceiveRouterMessages listens for incoming PubSub messages for a specific gateway and app and calls the provided callback.
	// This is a blocking call which only stops once the context is canceled.
	ReceiveRoutedRequest(ctx context.Context, gatewayId ulid.ULID, appId uuid.UUID, connId ulid.ULID, onMessage func(rawBytes []byte, data *connect.GatewayExecutorRequestData)) error

	// AckMessage sends an acknowledgment for a specific request.
	AckMessage(ctx context.Context, appId uuid.UUID, requestId string, source AckSource) error

	// Wait blocks and listens for incoming PubSub messages for the internal subscribers. This must be run before
	// subscribing to any channels to ensure that the PubSub client is connected and ready to receive messages.
	Wait(ctx context.Context) error
}

type redisPubSubConnector struct {
	client       rueidis.Client
	pubSubClient rueidis.DedicatedClient

	subscribers     map[string]map[string]chan string
	subscribersLock sync.RWMutex

	logger *slog.Logger

	RequestForwarder
	RequestReceiver
}

func newRedisPubSubConnector(client rueidis.Client, logger *slog.Logger) *redisPubSubConnector {
	return &redisPubSubConnector{
		client:          client,
		subscribers:     make(map[string]map[string]chan string),
		subscribersLock: sync.RWMutex{},
		logger:          logger,
	}
}

// Proxy forwards a request to the executor and waits for a response.
//
// If the gateway does not ack the message within a 10-second timeout, an error is returned.
// If no response is received before the context is canceled, an error is returned.
func (i *redisPubSubConnector) Proxy(ctx, traceCtx context.Context, appId uuid.UUID, data *connect.GatewayExecutorRequestData) (*connect.SDKResponse, error) {
	traceCtx, span := trace.ConnectTracer().Start(traceCtx, "Proxy")
	span.SetAttributes(attribute.Bool("inngest.system", true))
	defer span.End()

	proxyStartTime := time.Now()

	if data.RequestId == "" {
		data.RequestId = ulid.MustNew(ulid.Now(), rand.Reader).String()
	}

	span.SetAttributes(
		attribute.String("request_id", data.RequestId),
	)

	// Include trace context
	{
		// Add `traceparent` and `tracestate` headers to the request from `traceCtx`
		systemTraceCtx := propagation.MapCarrier{}
		// Note: The system context is stored in `traceCtx`
		trace.SystemTracer().Propagator().Inject(traceCtx, systemTraceCtx)
		marshaled, err := json.Marshal(systemTraceCtx)
		if err != nil {
			return nil, fmt.Errorf("could not marshal system trace ctx: %w", err)
		}
		data.SystemTraceCtx = marshaled
	}

	{
		userTraceCtx := propagation.MapCarrier{}
		// Note: The user context is stored in `ctx`
		trace.UserTracer().Propagator().Inject(ctx, userTraceCtx)
		marshaled, err := json.Marshal(userTraceCtx)
		if err != nil {
			return nil, fmt.Errorf("could not marshal user trace ctx: %w", err)
		}
		// Include in request
		data.UserTraceCtx = marshaled
	}

	dataBytes, err := proto.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("could not marshal executor request: %w", err)
	}

	// Await ack from router BEFORE response
	routerAckErrChan := make(chan error)
	var routerAcked bool
	{
		withAckTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		go func() {
			err = i.subscribe(withAckTimeout, i.channelAppRequestsAck(appId, data.RequestId, AckSourceRouter), func(_ string) {
				routerAcked = true

				span.AddEvent("RouterAck")
				metrics.HistogramConnectProxyAckTime(ctx, time.Since(proxyStartTime).Milliseconds(), metrics.HistogramOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"kind": "router",
					},
				})
			}, true)
			routerAckErrChan <- err
		}()
	}

	gatewayAckErrChan := make(chan error)
	var gatewayAcked bool
	{
		withAckTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		go func() {
			err = i.subscribe(withAckTimeout, i.channelAppRequestsAck(appId, data.RequestId, AckSourceGateway), func(_ string) {
				gatewayAcked = true

				span.AddEvent("GatewayAck")
				metrics.HistogramConnectProxyAckTime(ctx, time.Since(proxyStartTime).Milliseconds(), metrics.HistogramOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"kind": "gateway",
					},
				})
			}, true)
			gatewayAckErrChan <- err
		}()
	}

	workerAckErrChan := make(chan error)
	var workerAcked bool
	{
		withAckTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		go func() {
			err = i.subscribe(withAckTimeout, i.channelAppRequestsAck(appId, data.RequestId, AckSourceWorker), func(_ string) {
				workerAcked = true

				span.AddEvent("WorkerAck")
				metrics.HistogramConnectProxyAckTime(ctx, time.Since(proxyStartTime).Milliseconds(), metrics.HistogramOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"kind": "worker",
					},
				})
			}, true)
			workerAckErrChan <- err
		}()
	}

	// Await SDK response forwarded by gateway
	replyErrChan := make(chan error)
	var reply connect.SDKResponse
	go func() {
		// This may take a while: This waits until we receive the SDK response, and we allow for up to 2h in the serverless execution model
		err = i.subscribe(ctx, i.channelAppRequestsReply(appId, data.RequestId), func(msg string) {
			span.AddEvent("ReplyReceived")

			err := proto.Unmarshal([]byte(msg), &reply)
			if err != nil {
				// TODO This should never happen, push message into dead-letter channel and report
				return
			}
		}, true)
		replyErrChan <- err
	}()

	// After setting up ack and reply subscriptions, publish the request to the router, which forwards to the most suitable gateway
	channelName := i.channelExecutorRequests()

	// TODO Test whether this works with marshaled Protobuf bytes
	err = i.client.Do(ctx, i.client.B().Publish().Channel(channelName).Message(string(dataBytes)).Build()).Error()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "publishing message to router failed")
		return nil, fmt.Errorf("could not publish executor request: %w", err)
	}

	i.logger.Debug("published connect pubsub message", "channel", channelName, "request_id", data.RequestId)

	// Sanity check: Ensure the router received the message using a request-specific ack channel (ack must come in before SDK response)
	{
		err := <-routerAckErrChan
		close(routerAckErrChan)
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "unexpected error waiting for router ack")
			return nil, fmt.Errorf("could not receive executor request ack by router: %w", err)
		}

		if !routerAcked {
			span.SetStatus(codes.Error, "router did not ack")
			return nil, fmt.Errorf("router did not ack in time")
		}
	}

	{
		err := <-gatewayAckErrChan
		close(gatewayAckErrChan)
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "unexpected error waiting for gateway ack")
			return nil, fmt.Errorf("could not receive executor request ack by gateway: %w", err)
		}

		if !gatewayAcked {
			span.SetStatus(codes.Error, "gateway did not ack")
			return nil, fmt.Errorf("gateway did not ack in time")
		}
	}

	{
		err := <-workerAckErrChan
		close(workerAckErrChan)
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "unexpected error waiting for worker ack")

			return nil, fmt.Errorf("could not receive executor request ack by worker: %w", err)
		}

		if !workerAcked {
			span.SetStatus(codes.Error, "worker did not ack")

			return nil, fmt.Errorf("worker did not ack in time")
		}
	}

	// Await SDK response forwarded by gateway
	// This may take a while: This waits until we receive the SDK response, and we allow for up to 2h in the serverless execution model
	{
		err := <-replyErrChan
		close(replyErrChan)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not receive worker response")

			return nil, fmt.Errorf("could not receive executor response: %w", err)
		}
	}

	return &reply, nil
}

// channelExecutorRequests returns the channel name for executor requests to be processed by the router.
func (i *redisPubSubConnector) channelExecutorRequests() string {
	return "executor_requests"
}

// channelGatewayAppRequests returns the channel name for routed executor requests received by the gateway for a specific app and connection.
func (i *redisPubSubConnector) channelGatewayAppRequests(gatewayId ulid.ULID, appId uuid.UUID, connId ulid.ULID) string {
	return fmt.Sprintf("app_requests:%s:%s:%s", gatewayId, appId, connId)
}

func (i *redisPubSubConnector) channelAppRequestsAck(appId uuid.UUID, requestId string, source AckSource) string {
	return fmt.Sprintf("app_requests_ack:%s:%s:%s", appId, requestId, source)
}

func (i *redisPubSubConnector) channelAppRequestsReply(appId uuid.UUID, requestId string) string {
	return fmt.Sprintf("app_requests_reply:%s:%s", appId, requestId)
}

// subscribe sets up a subscription to a specific channel and calls the provided callback whenever a message is received.
// This method is blocking and will only return once the context is canceled.
//
// Upon return, the subscription is cleaned up and if the subscription was the last one for the channel, the PubSub client
// is unsubscribed from the channel.
func (i *redisPubSubConnector) subscribe(ctx context.Context, channel string, onMessage func(msg string), once bool) error {
	msgs := make(chan string)

	subId := ulid.MustNew(ulid.Now(), rand.Reader).String()

	// Set up internal subscription handler
	redisSubscribed := false
	{
		i.subscribersLock.Lock()

		if _, ok := i.subscribers[channel]; !ok {
			// subscribe to channel
			i.subscribers[channel] = make(map[string]chan string)
		} else {
			redisSubscribed = true
		}

		i.subscribers[channel][subId] = msgs

		i.subscribersLock.Unlock()
	}

	// This function is blocking, so whenever we return, we want to clean up the subscription handler and potentially
	// remove the subscription, if it's no longer used.
	defer func() {
		i.subscribersLock.Lock()
		defer i.subscribersLock.Unlock()

		close(msgs)
		i.logger.Debug("connect pubsub removing in-memory subscription", "channel", channel, "sub_id", subId)
		delete(i.subscribers[channel], subId)
		if len(i.subscribers[channel]) == 0 {
			i.logger.Debug("unsubscribing pubsub client from channel", "channel", channel)
			delete(i.subscribers, channel)
			i.pubSubClient.Do(ctx, i.pubSubClient.B().Unsubscribe().Channel(channel).Build())
		}
	}()

	// Set up receiver for incoming messages _before_ subscribing
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

	// If Redis client is not subscribed to channel already, send SUBSCRIBE command
	if !redisSubscribed {
		i.pubSubClient.Do(ctx, i.pubSubClient.B().Subscribe().Channel(channel).Build())
		i.logger.Debug("connect pubsub client subscribed to channel", "channel", channel)
	}

	<-done
	return nil
}

// ReceiveExecutorMessages listens for incoming PubSub messages for a specific app and calls the provided callback.
// This is a blocking call which only stops once the context is canceled.
func (i *redisPubSubConnector) ReceiveRoutedRequest(ctx context.Context, gatewayId ulid.ULID, appId uuid.UUID, connId ulid.ULID, onMessage func(rawBytes []byte, data *connect.GatewayExecutorRequestData)) error {
	return i.subscribe(ctx, i.channelGatewayAppRequests(gatewayId, appId, connId), func(msg string) {
		// TODO Test whether this works with marshaled Protobuf bytes
		msgBytes := []byte(msg)

		var data connect.GatewayExecutorRequestData
		err := proto.Unmarshal(msgBytes, &data)
		if err != nil {
			// TODO This should never happen, but PubSub will not redeliver, should we push the message into a dead-letter channel?
			i.logger.Error("invalid protobuf received by gateway", "err", err, "msg", msgBytes, "gateway_id", gatewayId, "app_id", appId, "conn_id", connId)
			return
		}

		onMessage(msgBytes, &data)
	}, false)
}

// ReceiveExecutorMessages listens for incoming PubSub messages for a specific app and calls the provided callback.
// This is a blocking call which only stops once the context is canceled.
func (i *redisPubSubConnector) ReceiveExecutorMessages(ctx context.Context, onMessage func(rawBytes []byte, data *connect.GatewayExecutorRequestData)) error {
	return i.subscribe(ctx, i.channelExecutorRequests(), func(msg string) {
		// TODO Test whether this works with marshaled Protobuf bytes
		msgBytes := []byte(msg)

		var data connect.GatewayExecutorRequestData
		err := proto.Unmarshal(msgBytes, &data)
		if err != nil {
			// TODO This should never happen, but PubSub will not redeliver, should we push the message into a dead-letter channel?
			return
		}

		onMessage(msgBytes, &data)
	}, false)
}

// Wait blocks and listens for incoming PubSub messages for the internal subscribers. This must be run before
// subscribing to any channels to ensure that the PubSub client is connected and ready to receive messages.
func (i *redisPubSubConnector) Wait(ctx context.Context) error {
	c, cancel := i.client.Dedicate()
	defer cancel()

	// TODO: Check whether this graceful shutdown routine makes sense here
	go func() {
		<-ctx.Done()

		i.logger.Debug("gracefully shutting down connect pubsub subscriber")

		i.subscribersLock.Lock()
		defer i.subscribersLock.Unlock()

		// TODO Should we prevent other executors from subscribing while we're in "shutting down" state?

		// Unsubscribe from all channels
		subs := i.subscribers
		for channelName := range subs {
			c.Do(ctx, c.B().Unsubscribe().Channel(channelName).Build())
		}

		c.Close()
	}()

	i.pubSubClient = c

	wait := c.SetPubSubHooks(rueidis.PubSubHooks{
		OnMessage: func(m rueidis.PubSubMessage) {
			i.logger.Debug("connect pubsub received message", "channel", m.Channel)

			// Run in another goroutine to avoid blocking `c`
			go func() {
				i.subscribersLock.RLock()
				// NOTE:  We have to keep this lock as we send in channels, otherwise we may attempt
				// to send on a closed channel that's unsubscribed.  Therefore, we keep the read lock
				// until we're done sending to all chans.
				defer i.subscribersLock.RUnlock()

				subs := i.subscribers[m.Channel]
				if len(subs) == 0 {
					// This should not happen: In subscribe, we UNSUBSCRIBE once the last subscriber is removed
					i.logger.Debug("no subscribers for connect pubsub channel", "channel", m.Channel)
					return
				}

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

// NotifyExecutor sends a response to the executor for a specific request.
func (i *redisPubSubConnector) NotifyExecutor(ctx context.Context, resp *connect.SDKResponse) error {
	serialized, err := proto.Marshal(resp)
	if err != nil {
		return fmt.Errorf("could not serialize response: %w", err)
	}

	appId, err := uuid.Parse(resp.AppId)
	if err != nil {
		return fmt.Errorf("missing appId in sdk response: %w", err)
	}

	channelName := i.channelAppRequestsReply(appId, resp.RequestId)

	// TODO Test whether this works with marshaled Protobuf bytes
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

	i.logger.Debug("sent connect pubsub reply", "channel", channelName)

	return nil
}

// AckMessage sends an acknowledgment for a specific request.
func (i *redisPubSubConnector) AckMessage(ctx context.Context, appId uuid.UUID, requestId string, source AckSource) error {
	err := i.client.Do(
		ctx,
		i.client.B().
			Publish().
			Channel(i.channelAppRequestsAck(appId, requestId, source)).
			Message(time.Now().Format(time.RFC3339)).
			Build()).
		Error()
	if err != nil {
		return fmt.Errorf("could not publish response: %w", err)
	}

	return nil
}

// RouteExecutorRequest forwards an executor request to the respective gateway
func (i *redisPubSubConnector) RouteExecutorRequest(ctx context.Context, gatewayId ulid.ULID, appId uuid.UUID, connId ulid.ULID, data *connect.GatewayExecutorRequestData) error {
	dataBytes, err := proto.Marshal(data)
	if err != nil {
		return fmt.Errorf("could not marshal executor request: %w", err)
	}

	channelName := i.channelGatewayAppRequests(gatewayId, appId, connId)

	// TODO Test whether this works with marshaled Protobuf bytes
	err = i.client.Do(ctx, i.client.B().Publish().Channel(channelName).Message(string(dataBytes)).Build()).Error()
	if err != nil {
		i.logger.Error("could not forward request to gateway", "err", err, "gateway_id", gatewayId, "channel", channelName, "request_id", data.RequestId, "conn_id", connId, "app_id", appId)
		return fmt.Errorf("could not publish executor request: %w", err)
	}

	i.logger.Debug("forwarded connect request to gateway", "gateway_id", gatewayId, "channel", channelName, "request_id", data.RequestId, "conn_id", connId, "app_id", appId)

	return nil
}
