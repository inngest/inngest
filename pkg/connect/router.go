package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/pubsub"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"gonum.org/v1/gonum/stat/sampleuv"
	"log/slog"
	"time"
)

const (
	pkgNameRouter = "connect.router"
)

var ErrNoHealthyConnection = fmt.Errorf("no healthy connection")

type connectRouterSvc struct {
	logger *slog.Logger

	stateManager state.StateManager
	receiver     pubsub.RequestReceiver

	rnd *util.FrandRNG

	tracer trace.ConditionalTracer
}

func (c *connectRouterSvc) Name() string {
	return "connect-router"
}

func (c *connectRouterSvc) Pre(ctx context.Context) error {
	// For weighted shuffles generate a new rand.
	c.rnd = util.NewFrandRNG()

	// Set up router-specific logger with info for correlations
	c.logger = logger.StdlibLogger(ctx)

	return nil
}

func (c *connectRouterSvc) Run(ctx context.Context) error {
	go func() {
		err := c.receiver.ReceiveExecutorMessages(ctx, func(rawBytes []byte, data *connect.GatewayExecutorRequestData) {
			log := c.logger.With("env_id", data.EnvId, "app_id", data.AppId, "req_id", data.RequestId)

			appID, err := uuid.Parse(data.AppId)
			if err != nil {
				log.Error("could not parse app ID")
				return
			}

			accountID, err := uuid.Parse(data.AccountId)
			if err != nil {
				log.Error("could not parse account ID")
				return
			}

			envID, err := uuid.Parse(data.EnvId)
			if err != nil {
				log.Error("could not parse env ID")
				return
			}

			log.Debug("router received msg")

			// We need to add an idempotency key to ensure only one router instance processes the message
			err = c.stateManager.SetRequestIdempotency(ctx, appID, data.RequestId)
			if err != nil {
				if errors.Is(err, state.ErrIdempotencyKeyExists) {
					// Another connection was faster than us, we can ignore this message
					return
				}

				log.Error("could not store idempotency key", "err", err)
				return
			}

			// Now we're guaranteed to be the exclusive connection processing this message!

			{
				systemTraceCtx := propagation.MapCarrier{}
				if err := json.Unmarshal(data.SystemTraceCtx, &systemTraceCtx); err != nil {
					log.Error("could not unmarshal system trace ctx", "err", err)

					return
				}

				ctx = trace.SystemTracer().Propagator().Extract(ctx, systemTraceCtx)
			}
			ctx, span := c.tracer.NewSpan(ctx, "RouteExecutorRequest", accountID, envID)
			defer span.End()

			routeTo, err := c.getSuitableConnection(ctx, envID, appID, data.AppName, data.FunctionSlug, log)
			if err != nil && !errors.Is(err, ErrNoHealthyConnection) {
				log.Error("could not retrieve suitable connection", "err", err)
				return
			}

			if routeTo == nil {
				log.Warn("no healthy connections")
				metrics.IncrConnectRouterNoHealthyConnectionCounter(ctx, 1, metrics.CounterOpt{
					PkgName: pkgNameRouter,
				})

				err = c.receiver.NackMessage(ctx, data.RequestId, pubsub.AckSourceRouter, syscode.Error{
					Code:    syscode.CodeConnectNoHealthyConnection,
					Message: "Could not find a healthy connection",
				})
				if err != nil {
					log.Error("failed to nack message", "err", err)
					// TODO Log error, retry?
					return
				}

				return
			}

			gatewayId, err := ulid.Parse(routeTo.GatewayId)
			if err != nil {
				log.Error("invalid gatewayId", "gatewayId", gatewayId, "err", err)
				return
			}

			connId, err := ulid.Parse(routeTo.Id)
			if err != nil {
				c.logger.Error("invalid connection ID", "conn_id", routeTo.Id, "err", err)
				return
			}

			log = log.With("gateway_id", routeTo.GatewayId, "conn_id", routeTo.Id, "group_hash", routeTo.WorkerGroups[data.AppName])
			span.SetAttributes(
				attribute.String("route_to_gateway_id", gatewayId.String()),
				attribute.String("route_to_conn_id", connId.String()),
			)

			// TODO What if something goes wrong inbetween setting idempotency (claiming exclusivity) and forwarding the req?
			// We'll potentially lose data here

			// Forward message to the gateway
			err = c.receiver.RouteExecutorRequest(ctx, gatewayId, connId, data)
			if err != nil {
				// TODO Should we retry? Log error?
				log.Error("failed to route request to gateway", "err", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, " could not forward request to gateway")
				return
			}

			log.Debug("forwarded executor request to gateway")

			err = c.receiver.AckMessage(ctx, data.RequestId, pubsub.AckSourceRouter)
			if err != nil {
				log.Error("failed to ack message", "err", err)
				// TODO Log error, retry?
				return
			}
		})
		if err != nil {
			// TODO Log error, retry?
			return
		}
	}()

	// TODO Periodically ping random gateways via PubSub and only consider them active if they respond in time -> Multiple routers will do this

	err := c.receiver.Wait(ctx)
	if err != nil {
		return fmt.Errorf("could not listen for pubsub messages: %w", err)
	}

	return nil

}

func (c *connectRouterSvc) getSuitableConnection(ctx context.Context, envID uuid.UUID, appID uuid.UUID, appName string, fnSlug string, log *slog.Logger) (*connect.ConnMetadata, error) {
	conns, err := c.stateManager.GetConnectionsByAppID(ctx, envID, appID)
	if err != nil {
		return nil, fmt.Errorf("could not get connections by app ID: %w", err)
	}

	if len(conns) == 0 {
		return nil, ErrNoHealthyConnection
	}

	healthy := make([]*connect.ConnMetadata, 0, len(conns))
	for _, conn := range conns {
		if c.isHealthy(ctx, envID, appName, fnSlug, conn, log) {
			healthy = append(healthy, conn)
		}
	}

	if len(healthy) == 0 {
		return nil, ErrNoHealthyConnection
	}

	weights := make([]float64, len(healthy))
	for i := range healthy {
		// TODO Calculate weights based on factors like version
		weights[i] = float64(10)
	}

	w := sampleuv.NewWeighted(weights, c.rnd)
	idx, ok := w.Take()
	if !ok {
		return nil, util.ErrWeightedSampleRead
	}
	chosen := healthy[idx]

	return chosen, nil
}

func (c *connectRouterSvc) isHealthy(ctx context.Context, envID uuid.UUID, appName string, fnSlug string, conn *connect.ConnMetadata, log *slog.Logger) bool {
	var shouldDeleteConnection bool

	defer func() {
		if !shouldDeleteConnection {
			return
		}

		connId, err := ulid.Parse(conn.Id)
		if err != nil {
			log.Error("could not clean up inactive connection, invalid connection ID", "conn_id", conn.Id, "err", err)
		}

		// Clean up unhealthy connection
		err = c.stateManager.DeleteConnection(context.Background(), envID, connId)
		if err != nil {
			log.Error("could not clean up inactive connection", "conn_id", conn.Id, "err", err)
		}
	}()

	log.Debug("evaluating connection", "connection_id", conn.Id, "status", conn.Status, "last_heartbeat_at", conn.LastHeartbeatAt.AsTime())

	gatewayId, err := ulid.Parse(conn.GatewayId)
	if err != nil {
		log.Error("connection gateway id could not be parsed", "err", err, "gateway_id", conn.GatewayId)

		// Clean up invalid connection
		shouldDeleteConnection = true

		return false
	}

	if conn.Status != connect.ConnectionStatus_READY {
		log.Debug("connection is not ready")

		if conn.Status == connect.ConnectionStatus_DISCONNECTED {
			// Clean up disconnected connection
			shouldDeleteConnection = true
		}

		return false
	}

	// If more than two consecutive heartbeats were missed, the connection is not healthy
	connectionHeartbeatMissed := conn.LastHeartbeatAt.AsTime().Before(time.Now().Add(-2 * WorkerHeartbeatInterval))
	if connectionHeartbeatMissed {
		log.Debug("last heartbeat is too old")

		// Clean up outdated connection
		shouldDeleteConnection = true

		return false
	}

	groupHash, ok := conn.WorkerGroups[appName]
	if !ok {
		log.Error("connection missing worker group hash for app", "app_name", appName, "worker_groups", conn.WorkerGroups)

		return false
	}

	group, err := c.stateManager.GetWorkerGroupByHash(ctx, envID, groupHash)
	if err != nil {
		log.Error("could not get worker group for connection", "group_id", groupHash)

		shouldDeleteConnection = true

		return false
	}

	var hasFn bool
	for _, slug := range group.FunctionSlugs {
		if slug == fnSlug {
			hasFn = true
			break
		}
	}

	if !hasFn {
		log.Debug("connection does not have function", "slug", fnSlug, "available", group.FunctionSlugs)

		return false
	}

	// Ensure gateway is healthy
	gw, err := c.stateManager.GetGateway(ctx, gatewayId)
	if err != nil {
		log.Error("could not get gateway", "gateway_id", gatewayId.String())

		shouldDeleteConnection = true

		return false
	}

	log.Debug("retrieved gateway for connection", "conn_id", conn.Id, "gateway_id", gatewayId.String(), "status", gw.Status, "last_heartbeat_at", gw.LastHeartbeatAt)

	gatewayIsActive := gw.Status == state.GatewayStatusActive
	gatewayHeartbeatTimedOut := gw.LastHeartbeatAt.Before(time.Now().Add(-2 * GatewayHeartbeatInterval))
	if !gatewayIsActive || gatewayHeartbeatTimedOut {
		log.Debug("gateway is unhealthy", "conn_id", conn.Id, "gateway_id", gatewayId.String(), "status", gw.Status, "last_heartbeat_at", gw.LastHeartbeatAt)

		if gatewayHeartbeatTimedOut {
			// Clean up unhealthy gateway
			err = c.stateManager.DeleteGateway(ctx, gatewayId)
			if err != nil {
				c.logger.Error("could not clean up inactive gateway", "gateway_id", conn.GatewayId, "err", err)
			}
		}

		// Drop associated connection
		shouldDeleteConnection = true

		return false
	}

	return true
}

func (c *connectRouterSvc) Stop(ctx context.Context) error {
	return nil
}

func NewConnectMessageRouterService(stateManager state.StateManager, receiver pubsub.RequestReceiver, tracer trace.ConditionalTracer) *connectRouterSvc {
	return &connectRouterSvc{
		stateManager: stateManager,
		receiver:     receiver,
		tracer:       tracer,
	}
}
