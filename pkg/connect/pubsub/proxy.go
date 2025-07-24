package pubsub

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	mathRand "math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/routing"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/util"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	pkgName = "connect.execution.proxy"
)

/*
	This package provides a PubSub-based request forwarding mechanism for the Connect SDK.

	Execution requests are forwarded from the executor to the SDK via the connect infrastructure, including the following components:

	Executor -> Embedded Router -> Gateway -> SDK

	The embedded routing layer is responsible for selecting a gateway with an active, healthy connection for a given app. It is only responsible for
	routing requests to the correct gateway, not for returning the SDK response back to the executor. This is directly handled by the gateway.

	The gateway will acknowledge the request, forward it to the SDK, and return the response to the executor.
*/

type RequestForwarder interface {
	// Proxy forwards a request from the executor to the SDK via the connect infrastructure and waits for a response.
	//
	// If no responsible gateway ack's the message within a 10-second timeout, an error is returned.
	// If no response is received before the context is canceled, an error is returned.
	Proxy(ctx, traceCtx context.Context, opts ProxyOpts) (*connectpb.SDKResponse, error)
}

type AckSource string

const (
	AckSourceWorker  AckSource = "worker"
	AckSourceGateway AckSource = "gateway"
)

type ResponseNotifier interface {
	// NotifyExecutor sends a response to the executor for a specific request.
	NotifyExecutor(ctx context.Context, resp *connectpb.SDKResponse) error
}

type RequestReceiver interface {
	ResponseNotifier

	// RouteExecutorRequest forwards an executor request to the respective gateway
	RouteExecutorRequest(ctx context.Context, gatewayId ulid.ULID, connId ulid.ULID, data *connectpb.GatewayExecutorRequestData) error

	// ReceiveRoutedRequest listens for incoming PubSub messages for a specific gateway and app and calls the provided callback.
	// This is a blocking call which only stops once the context is canceled.
	ReceiveRoutedRequest(ctx context.Context, gatewayId ulid.ULID, connId ulid.ULID, onMessage func(rawBytes []byte, data *connectpb.GatewayExecutorRequestData), onSubscribed chan struct{}) error

	// AckMessage sends an acknowledgment for a specific request.
	AckMessage(ctx context.Context, requestId string, source AckSource) error

	// Wait blocks and listens for incoming PubSub messages for the internal subscribers. This must be run before
	// subscribing to any channels to ensure that the PubSub client is connected and ready to receive messages.
	Wait(ctx context.Context) error
}

type subscription struct {
	ch  chan string
	ctx context.Context
}

type EnforceLeaseExpiryFunc func(ctx context.Context, accountID uuid.UUID) bool

type redisPubSubConnector struct {
	client       rueidis.Client
	pubSubClient rueidis.DedicatedClient
	setup        chan struct{}

	subscribers     map[string]map[string]*subscription
	subscribersLock sync.RWMutex

	logger logger.Logger
	tracer trace.ConditionalTracer

	stateManager state.StateManager
	rnd          *util.FrandRNG

	enforceLeaseExpiry EnforceLeaseExpiryFunc

	gatewayGRPCManager GatewayGRPCManager

	RequestReceiver
}


type RedisPubSubConnectorOpts struct {
	Logger             logger.Logger
	Tracer             trace.ConditionalTracer
	StateManager       state.StateManager
	EnforceLeaseExpiry EnforceLeaseExpiryFunc

	GatewayGRPCManager GatewayGRPCManager
}

func newRedisPubSubConnector(client rueidis.Client, opts RedisPubSubConnectorOpts) *redisPubSubConnector {
	return &redisPubSubConnector{
		client:             client,
		subscribers:        make(map[string]map[string]*subscription),
		subscribersLock:    sync.RWMutex{},
		logger:             opts.Logger,
		tracer:             opts.Tracer,
		setup:              make(chan struct{}),
		enforceLeaseExpiry: opts.EnforceLeaseExpiry,

		gatewayGRPCManager: opts.GatewayGRPCManager,

		// For routing
		stateManager: opts.StateManager,
		rnd:          util.NewFrandRNG(),
	}
}

type ProxyOpts struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
	AppID     uuid.UUID
	SpanID    string
	Data      *connectpb.GatewayExecutorRequestData
	logger    logger.Logger
}

// Proxy forwards a request to the executor and waits for a response.
//
// If the gateway does not ack the message within a 10-second timeout, an error is returned.
// If no response is received before the context is canceled, an error is returned.
func (i *redisPubSubConnector) Proxy(ctx, traceCtx context.Context, opts ProxyOpts) (*connectpb.SDKResponse, error) {
	select {
	case <-i.setup:
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("expected setup to be completed within 10s")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	l := logger.StdlibLogger(ctx)
	if opts.logger != nil {
		l = opts.logger
	}

	l = l.With(
		"scope", "connect_proxy",
		"app_id", opts.AppID.String(),
		"env_id", opts.EnvID.String(),
		"account_id", opts.AccountID.String(),
		"run_id", opts.Data.RunId,
		"req_id", opts.Data.RequestId,
	)

	traceCtx, span := i.tracer.NewSpan(traceCtx, "Proxy", opts.AccountID, opts.EnvID)
	span.SetAttributes(attribute.Bool("inngest.system", true))
	defer span.End()

	proxyStartTime := time.Now()

	span.SetAttributes(
		attribute.String("request_id", opts.Data.RequestId),
	)

	{
		// Check if previous request finished. Even if the lease is expired, it's possible for the worker
		// to have already sent a response and completed the request.
		resp, err := i.stateManager.GetResponse(ctx, opts.EnvID, opts.Data.RequestId)
		if err != nil {
			span.RecordError(err)
			l.ReportError(err, "could not check for buffered response")
			return nil, fmt.Errorf("could not check for buffered response: %w", err)
		}

		if resp != nil {
			// We have a response already, return it
			l.Debug("found buffered response")

			// The response has a short TTL so it will be cleaned up, but we should try
			// to garbage-collect unused state as quickly as possible
			err := i.stateManager.DeleteResponse(ctx, opts.EnvID, opts.Data.RequestId)
			if err != nil {
				span.RecordError(err)
				l.ReportError(err, "could not delete buffered response")
			}

			return resp, nil
		}
	}

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
		opts.Data.SystemTraceCtx = marshaled
	}

	{

		userTraceCtx, err := trace.HeadersFromTraceState(
			ctx,
			opts.SpanID,
			opts.AppID.String(),
			opts.Data.FunctionId,
		)
		if err != nil {
			span.RecordError(err)
			l.Error("could not get user trace ctx", "err", err)
			return nil, fmt.Errorf("could not get user trace ctx: %w", err)
		}

		marshaled, err := json.Marshal(userTraceCtx)
		if err != nil {
			return nil, fmt.Errorf("could not marshal user trace ctx: %w", err)
		}
		// Include in request
		opts.Data.UserTraceCtx = marshaled
	}

	{
		// Receive worker acknowledgement for o11y
		ch := i.gatewayGRPCManager.SubscribeWorkerAck(ctx, opts.Data.RequestId)
		defer i.gatewayGRPCManager.UnsubscribeWorkerAck(ctx, opts.Data.RequestId)

		go func() {
			<-ch

			span.AddEvent("WorkerAck")
			metrics.HistogramConnectProxyAckTime(ctx, time.Since(proxyStartTime).Milliseconds(), metrics.HistogramOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"kind":      "worker",
					"transport": "grpc",
				},
			})

		}()
	}

	// Await SDK response forwarded by gateway

	reply := &connectpb.SDKResponse{}

	waitForResponseCtx, cancelWaitForResponseCtx := context.WithCancel(ctx)
	defer cancelWaitForResponseCtx()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-waitForResponseCtx.Done():
				return
			// Poll every two seconds with a jitter of up to 3 seconds
			case <-time.After(2*time.Second + time.Duration(mathRand.Int63n(3))*time.Second):
			}

			resp, err := i.stateManager.GetResponse(ctx, opts.EnvID, opts.Data.RequestId)
			if err != nil {
				span.RecordError(err)
				l.ReportError(err, "could not check for response")
				continue
			}

			if resp != nil {
				span.AddEvent("ReplyReceivedPoll")

				l.Debug("received response via polling")

				reply = resp

				cancelWaitForResponseCtx()
				return
			}

			span.AddEvent("ReplyPollOk")
		}
	}()

	{
		// Alternatively, the gateway will send the response as soon as it comes in.
		// This is unreliable but quicker than polling for the response, so we use this
		// as a best-effort notification mechanism.
		replySubscribed := make(chan struct{})
		go func() {
			replyReceived := i.gatewayGRPCManager.Subscribe(ctx, opts.Data.RequestId)

			close(replySubscribed)

			reply = <-replyReceived
			span.AddEvent("ReplyReceivedGRPC")
			l.Debug("received response via gRPC")

			metrics.IncrConnectGatewayGRPCReplyCounter(ctx, 1, metrics.CounterOpt{})

			cancelWaitForResponseCtx()
		}()

		select {
		case <-replySubscribed:
			defer i.gatewayGRPCManager.Unsubscribe(ctx, opts.Data.RequestId)
		case <-time.After(5 * time.Second):
			return nil, fmt.Errorf("did not subscribe to grpc reply within 5s")
		}
	}

	// Attempt to lease the request. If the request is still running on a worker,
	// this will fail with ErrRequestLeased. In this case, we can just wait for the request to complete.
	// Otherwise, we acquired the lease and need to forward the request to the worker.
	leaseID, err := i.stateManager.LeaseRequest(ctx, opts.EnvID, opts.Data.RequestId, consts.ConnectWorkerRequestLeaseDuration)
	if err != nil && !errors.Is(err, state.ErrRequestLeased) {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to lease request")

		return nil, fmt.Errorf("failed to lease request: %w", err)
	}

	if leaseID == nil && !errors.Is(err, state.ErrRequestLeased) {
		span.SetStatus(codes.Error, "missing initial lease ID")

		return nil, fmt.Errorf("missing initial lease ID")
	}

	if leaseID != nil {
		// Include initial Lease ID in request
		opts.Data.LeaseId = leaseID.String()
	}

	// Periodically check for lease health, if lease expired, we need to retry
	leaseCtx, cancelLeaseCtx := context.WithCancel(ctx)
	defer cancelLeaseCtx()
	go func() {
		for {
			select {
			case <-leaseCtx.Done():
				return
			// Verify lease did not expire
			case <-time.After(consts.ConnectWorkerRequestExtendLeaseInterval):
			}

			leased, err := i.stateManager.IsRequestLeased(ctx, opts.EnvID, opts.Data.RequestId)
			if err != nil {
				span.RecordError(err)
				l.ReportError(err, "could not get lease status")
				continue
			}

			if !leased {
				// Selectively enable lease enforcement to create gradual rollout for existing connect users
				if i.enforceLeaseExpiry != nil && !i.enforceLeaseExpiry(ctx, opts.AccountID) {
					continue
				}

				// Grace period to wait for the worker to send the response
				select {
				case <-waitForResponseCtx.Done():
					l.Debug("response arrived during lease expiry grace period")
					return
				case <-time.After(consts.ConnectWorkerRequestGracePeriod):
				}

				l.Debug("request lease expired")
				span.RecordError(fmt.Errorf("item is no longer leased"))

				cancelLeaseCtx()
				return
			}

			l.Debug("request is still leased by worker")
			span.AddEvent("RequestLeaseOk")
		}
	}()

	// Forward message to the gateway if the request wasn't already running
	if leaseID != nil {
		// Determine the most suitable connection
		route, err := routing.GetRoute(ctx, i.stateManager, i.rnd, i.tracer, l, opts.Data)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to route message")

			if errors.Is(err, routing.ErrNoHealthyConnection) {
				return nil, syscode.Error{
					Code:    syscode.CodeConnectNoHealthyConnection,
					Message: "Could not find a healthy connection",
				}
			}

			return nil, fmt.Errorf("failed to route message: %w", err)
		}

		transport := "grpc"

		// Forward the request
		err = i.gatewayGRPCManager.Forward(ctx, route.GatewayID, route.ConnectionID, opts.Data)
		if err != nil {
			// Handle gateway ack
			metrics.HistogramConnectProxyAckTime(ctx, time.Since(proxyStartTime).Milliseconds(), metrics.HistogramOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"kind": "gateway",
				},
			})
		}

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not forward request to gateway")

			return nil, fmt.Errorf("failed to route request to gateway: %w", err)
		}

		l.Debug("forwarded executor request to gateway", "gateway_id", route.GatewayID, "conn_id", route.ConnectionID)

		metrics.IncrConnectRouterPubSubMessageSentCounter(ctx, 1, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"transport": transport},
		})
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("parent context was closed unexpectedly")
	// Handle maximum function timeout
	case <-time.After(consts.MaxFunctionTimeout):
		return nil, syscode.Error{
			Code:    syscode.CodeRequestTooLong,
			Message: "The worker took longer than the maximum request duration to respond to the request.",
		}
	// Await SDK response forwarded by gateway
	// This may take a while: This waits until we receive the SDK response, and we allow for up to 2h in the serverless execution model
	case <-waitForResponseCtx.Done():
		// Stop checking for lease
		cancelLeaseCtx()

		// The lease has a short TTL so it will be cleaned up, but we should try
		// to garbage-collect unused state as quickly as possible
		err = i.stateManager.DeleteLease(ctx, opts.EnvID, opts.Data.RequestId)
		if err != nil {
			span.RecordError(err)
			l.ReportError(err, "could not delete lease")
		}

		if reply.RequestId == "" {
			span.SetStatus(codes.Error, "missing response")

			return nil, fmt.Errorf("did not receive worker response")
		}

		// The response has a short TTL so it will be cleaned up, but we should try
		// to garbage-collect unused state as quickly as possible
		err := i.stateManager.DeleteResponse(ctx, opts.EnvID, opts.Data.RequestId)
		if err != nil {
			span.RecordError(err)
			l.ReportError(err, "could not delete response")
		}

		l.Debug("returning reply", "status", reply.Status)
		return reply, nil
	// If the worker terminates or otherwise fails to continue extending the lease,
	// we must retry the step as soon as possible.
	case <-leaseCtx.Done():
		span.SetStatus(codes.Error, "lease expired")

		return nil, syscode.Error{
			Code:    syscode.CodeConnectWorkerStoppedResponding,
			Message: "The worker stopped responding to the request.",
		}
	}
}

// channelGatewayAppRequests returns the channel name for routed executor requests received by the gateway for a specific app and connection.
func (i *redisPubSubConnector) channelGatewayAppRequests(gatewayId ulid.ULID, connId ulid.ULID) string {
	return fmt.Sprintf("app_requests:%s:%s", gatewayId, connId)
}

func (i *redisPubSubConnector) channelAppRequestsAck(requestId string, source AckSource) string {
	return fmt.Sprintf("app_requests_ack:%s:%s", requestId, source)
}

func (i *redisPubSubConnector) channelAppRequestsReply(requestId string) string {
	return fmt.Sprintf("app_requests_reply:%s", requestId)
}

// subscribe sets up a subscription to a specific channel and calls the provided callback whenever a message is received.
// This method is blocking and will only return once the context is canceled.
//
// Upon return, the subscription is cleaned up and if the subscription was the last one for the channel, the PubSub client
// is unsubscribed from the channel.
func (i *redisPubSubConnector) subscribe(ctx context.Context, channel string, onMessage func(msg string), once bool, onSubscribed chan struct{}) {
	<-i.setup

	msgs := make(chan string)

	subId := ulid.MustNew(ulid.Now(), rand.Reader).String()

	// Set up internal subscription handler
	redisSubscribed := false
	{
		i.subscribersLock.Lock()

		if _, ok := i.subscribers[channel]; !ok {
			// subscribe to channel
			i.subscribers[channel] = make(map[string]*subscription)
		} else {
			redisSubscribed = true
		}

		i.subscribers[channel][subId] = &subscription{
			ch:  msgs,
			ctx: ctx,
		}

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

	if onSubscribed != nil {
		close(onSubscribed)
	}

	<-done
}

// ReceiveRoutedRequest listens for incoming PubSub messages for a specific app and calls the provided callback.
// This is a blocking call which only stops once the context is canceled.
func (i *redisPubSubConnector) ReceiveRoutedRequest(ctx context.Context, gatewayId ulid.ULID, connId ulid.ULID, onMessage func(rawBytes []byte, data *connectpb.GatewayExecutorRequestData), onSubscribed chan struct{}) error {
	i.subscribe(ctx, i.channelGatewayAppRequests(gatewayId, connId), func(msg string) {
		// TODO Test whether this works with marshaled Protobuf bytes
		msgBytes := []byte(msg)

		var data connectpb.GatewayExecutorRequestData
		err := proto.Unmarshal(msgBytes, &data)
		if err != nil {
			// TODO This should never happen, but PubSub will not redeliver, should we push the message into a dead-letter channel?
			i.logger.Error("invalid protobuf received by gateway", "err", err, "msg", msgBytes, "gateway_id", gatewayId, "conn_id", connId)
			return
		}

		onMessage(msgBytes, &data)
	}, false, onSubscribed)
	return nil
}

// Wait blocks and listens for incoming PubSub messages for the internal subscribers. This must be run before
// subscribing to any channels to ensure that the PubSub client is connected and ready to receive messages.
func (i *redisPubSubConnector) Wait(ctx context.Context) error {
	// If we already set this up, return immediately
	if i.pubSubClient != nil {
		return nil
	}

	c, cancel := i.client.Dedicate()
	defer cancel()

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
	close(i.setup)

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

				for _, sub := range subs {
					select {
					case sub.ch <- m.Message:
						// Message successfully sent
					case <-sub.ctx.Done():
						// Subscriber's context is cancelled; stop processing further
						continue
					}
				}
			}()
		},
	})

	if i.gatewayGRPCManager != nil {
		if err := i.gatewayGRPCManager.ConnectToGateways(ctx); err != nil {
			return err
		}
	}

	err := <-wait // disconnected with err
	if err != nil && !errors.Is(err, rueidis.ErrClosing) {
		return err
	}
	return nil
}

// NotifyExecutor sends a response to the executor for a specific request.
func (i *redisPubSubConnector) NotifyExecutor(ctx context.Context, resp *connectpb.SDKResponse) error {
	serialized, err := proto.Marshal(resp)
	if err != nil {
		return fmt.Errorf("could not serialize response: %w", err)
	}

	channelName := i.channelAppRequestsReply(resp.RequestId)

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
func (i *redisPubSubConnector) AckMessage(ctx context.Context, requestId string, source AckSource) error {
	msgBytes, err := proto.Marshal(&connectpb.PubSubAckMessage{
		Ts: timestamppb.Now(),
	})
	if err != nil {
		return fmt.Errorf("could not marshal ack message: %w", err)
	}

	err = i.client.Do(
		ctx,
		i.client.B().
			Publish().
			Channel(i.channelAppRequestsAck(requestId, source)).
			Message(string(msgBytes)).
			Build()).
		Error()
	if err != nil {
		return fmt.Errorf("could not publish response: %w", err)
	}

	return nil
}

// RouteExecutorRequest forwards an executor request to the respective gateway
func (i *redisPubSubConnector) RouteExecutorRequest(ctx context.Context, gatewayId ulid.ULID, connId ulid.ULID, data *connectpb.GatewayExecutorRequestData) error {
	dataBytes, err := proto.Marshal(data)
	if err != nil {
		return fmt.Errorf("could not marshal executor request: %w", err)
	}

	channelName := i.channelGatewayAppRequests(gatewayId, connId)

	err = i.client.Do(ctx, i.client.B().Publish().Channel(channelName).Message(string(dataBytes)).Build()).Error()
	if err != nil {
		i.logger.Error("could not forward request to gateway", "err", err, "gateway_id", gatewayId, "channel", channelName, "req_id", data.RequestId, "conn_id", connId, "app_id", data.AppId)
		return fmt.Errorf("could not publish executor request: %w", err)
	}

	i.logger.Debug("forwarded connect request to gateway", "gateway_id", gatewayId, "channel", channelName, "req_id", data.RequestId, "conn_id", connId, "app_id", data.AppId)

	return nil
}
