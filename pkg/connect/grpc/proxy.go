package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	mathRand "math/rand"
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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
)

const (
	pkgName = "connect.execution.proxy"
)

/*
	This package provides a gRPC-based request forwarding mechanism for the Connect SDK.

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

type EnforceLeaseExpiryFunc func(ctx context.Context, accountID uuid.UUID) bool

type grpcConnector struct {
	logger logger.Logger
	tracer trace.ConditionalTracer

	stateManager state.StateManager
	rnd          *util.FrandRNG

	enforceLeaseExpiry EnforceLeaseExpiryFunc

	gatewayGRPCManager GatewayGRPCManager
}

type GRPCConnectorOpts struct {
	Tracer             trace.ConditionalTracer
	StateManager       state.StateManager
	EnforceLeaseExpiry EnforceLeaseExpiryFunc
}

type GRPCConnectorOption func(*grpcConnector)

func WithConnectorLogger(logger logger.Logger) GRPCConnectorOption {
	return func(c *grpcConnector) {
		c.logger = logger
	}
}

func WithGatewayManager(manager GatewayGRPCManager) GRPCConnectorOption {
	return func(c *grpcConnector) {
		c.gatewayGRPCManager = manager
	}
}

func newGRPCConnector(ctx context.Context, opts GRPCConnectorOpts, options ...GRPCConnectorOption) *grpcConnector {
	connector := &grpcConnector{
		logger:             logger.StdlibLogger(ctx), // Default logger
		tracer:             opts.Tracer,
		enforceLeaseExpiry: opts.EnforceLeaseExpiry,
		stateManager:       opts.StateManager,
		rnd:                util.NewFrandRNG(),
	}
	
	// Apply functional options
	for _, option := range options {
		option(connector)
	}
	
	// Create default gateway manager if not provided via options
	if connector.gatewayGRPCManager == nil {
		connector.gatewayGRPCManager = newGatewayGRPCManager(ctx, opts.StateManager, WithGatewayLogger(connector.logger))
	}
	
	return connector
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
func (i *grpcConnector) Proxy(ctx, traceCtx context.Context, opts ProxyOpts) (*connectpb.SDKResponse, error) {
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

		metrics.IncrConnectRouterGRPCMessageSentCounter(ctx, 1, metrics.CounterOpt{
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
