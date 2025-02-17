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

type connectRouterSvc struct {
	logger *slog.Logger

	stateManager state.StateManager
	receiver     pubsub.RequestReceiver

	rnd *util.FrandRNG
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

			appId, err := uuid.Parse(data.AppId)
			if err != nil {
				log.Error("could not parse app ID")
				return
			}

			envId, err := uuid.Parse(data.EnvId)
			if err != nil {
				log.Error("could not parse env ID")
				return
			}

			log.Debug("router received msg")

			// We need to add an idempotency key to ensure only one router instance processes the message
			err = c.stateManager.SetRequestIdempotency(ctx, appId, data.RequestId)
			if err != nil {
				if errors.Is(err, state.ErrIdempotencyKeyExists) {
					// Another connection was faster than us, we can ignore this message
					return
				}

				log.Error("could not store idempotency key", "err", err)
				return
			}

			err = c.receiver.AckMessage(ctx, appId, data.RequestId, pubsub.AckSourceRouter)
			if err != nil {
				log.Error("failed to ack message", "err", err)
				// TODO Log error, retry?
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
			ctx, span := trace.ConnectTracer().Start(ctx, "RouteExecutorRequest")
			defer span.End()

			routeTo, err := c.getSuitableConnection(ctx, envId, appId, data.FunctionSlug, log)
			if err != nil {
				log.Error("could not retrieve suitable connection", "err", err)
				return
			}

			if routeTo == nil {
				log.Warn("no healthy connections")
				metrics.IncrConnectRouterNoHealthyConnectionCounter(ctx, 1, metrics.CounterOpt{
					PkgName: pkgNameRouter,
				})
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

			log = log.With("gateway_id", routeTo.GatewayId, "conn_id", routeTo.Id, "group_hash", routeTo.GroupId)
			span.SetAttributes(
				attribute.String("route_to_gateway_id", gatewayId.String()),
				attribute.String("route_to_conn_id", connId.String()),
			)

			// TODO What if something goes wrong inbetween setting idempotency (claiming exclusivity) and forwarding the req?
			// We'll potentially lose data here

			// Forward message to the gateway
			err = c.receiver.RouteExecutorRequest(ctx, gatewayId, appId, connId, data)
			if err != nil {
				// TODO Should we retry? Log error?
				log.Error("failed to route request to gateway", "err", err)
				span.RecordError(err)
				span.SetStatus(codes.Error, " could not forward request to gateway")
				return
			}

			log.Debug("forwarded executor request to gateway")
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

func (c *connectRouterSvc) getSuitableConnection(ctx context.Context, envId uuid.UUID, appId uuid.UUID, fnSlug string, log *slog.Logger) (*connect.ConnMetadata, error) {
	conns, err := c.stateManager.GetConnectionsByAppID(ctx, envId, appId)
	if err != nil {
		return nil, fmt.Errorf("could not get connections by app ID: %w", err)
	}

	if len(conns) == 0 {
		return nil, nil
	}

	healthy := make([]*connect.ConnMetadata, 0, len(conns))
	for _, conn := range conns {
		isHealthy, err := c.isHealthy(ctx, envId, appId, fnSlug, conn, log)
		if err != nil {
			return nil, err
		}

		if isHealthy {
			healthy = append(healthy, conn)
		}
	}

	if len(healthy) == 0 {
		return nil, fmt.Errorf("no healthy connections found")
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

func (c *connectRouterSvc) isHealthy(ctx context.Context, envId uuid.UUID, appId uuid.UUID, fnSlug string, conn *connect.ConnMetadata, log *slog.Logger) (isHealthy bool, err error) {
	defer func() {
		if isHealthy {
			return
		}

		connId, err := ulid.Parse(conn.Id)
		if err != nil {
			log.Error("could not clean up inactive connection, invalid connection ID", "conn_id", conn.Id, "err", err)
		}

		// Clean up unhealthy connection
		err = c.stateManager.DeleteConnection(context.Background(), envId, &appId, conn.GroupId, connId)
		if err != nil && !errors.Is(err, state.ConnDeletedWithGroupErr) {
			log.Error("could not clean up inactive connection", "conn_id", conn.Id, "err", err)
		}
	}()

	log.Debug("evaluating connection", "connection_id", conn.Id, "status", conn.Status, "last_heartbeat_at", conn.LastHeartbeatAt.AsTime())

	gatewayId, err := ulid.Parse(conn.GatewayId)
	if err != nil {
		log.Error("connection gateway id could not be parsed", "err", err, "gateway_id", conn.GatewayId)

		return
	}

	if conn.Status != connect.ConnectionStatus_READY {
		log.Debug("connection is not ready")

		return
	}

	if conn.LastHeartbeatAt.AsTime().Before(time.Now().Add(-2 * WorkerHeartbeatInterval)) {
		log.Debug("last heartbeat is too old")

		return
	}

	group, err := c.stateManager.GetWorkerGroupByHash(ctx, envId, conn.GroupId)
	if err != nil {
		log.Error("could not get worker group for connection", "group_id", conn.GroupId)

		return
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

		return
	}

	// Ensure gateway is healthy
	gw, err := c.stateManager.GetGateway(ctx, gatewayId)
	if err != nil {
		log.Error("could not get gateway", "gateway_id", gatewayId.String())

		return
	}

	log.Debug("retrieved gateway for connection", "conn_id", conn.Id, "gateway_id", gatewayId.String(), "status", gw.Status, "last_heartbeat_at", gw.LastHeartbeatAt)

	if gw.Status != state.GatewayStatusActive || gw.LastHeartbeatAt.Before(time.Now().Add(-2*GatewayHeartbeatInterval)) {
		log.Debug("gateway is unhealthy", "conn_id", conn.Id, "gateway_id", gatewayId.String(), "status", gw.Status, "last_heartbeat_at", gw.LastHeartbeatAt)

		// Clean up unhealthy gateway
		err = c.stateManager.DeleteGateway(ctx, gatewayId)
		if err != nil {
			c.logger.Error("could not clean up inactive gateway", "gateway_id", conn.GatewayId, "err", err)
		}

		return
	}

	isHealthy = true
	return
}

func (c *connectRouterSvc) Stop(ctx context.Context) error {
	return nil
}

func NewConnectMessageRouterService(stateManager state.StateManager, receiver pubsub.RequestReceiver) *connectRouterSvc {
	return &connectRouterSvc{
		stateManager: stateManager,
		receiver:     receiver,
	}
}
