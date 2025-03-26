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
	"golang.org/x/sync/errgroup"
	"gonum.org/v1/gonum/stat/sampleuv"
	"log/slog"
	"slices"
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

func Route(ctx context.Context, stateMgr state.StateManager, publisher pubsub.RequestReceiver, rnd *util.FrandRNG, tracer trace.ConditionalTracer, log *slog.Logger, data *connect.GatewayExecutorRequestData) error {
	appID, err := uuid.Parse(data.AppId)
	if err != nil {
		return fmt.Errorf("could not parse app ID: %w", err)
	}

	accountID, err := uuid.Parse(data.AccountId)
	if err != nil {
		return fmt.Errorf("could not parse account ID: %w", err)
	}

	envID, err := uuid.Parse(data.EnvId)
	if err != nil {
		return fmt.Errorf("could not parse env ID: %w", err)
	}

	log.Debug("router received msg")

	// We need to add an idempotency key to ensure only one router instance processes the message
	err = stateMgr.SetRequestIdempotency(ctx, appID, data.RequestId)
	if err != nil {
		if errors.Is(err, state.ErrIdempotencyKeyExists) {
			// Another connection was faster than us, we can ignore this message
			return nil
		}

		return fmt.Errorf("could not store idempotency key: %w", err)
	}

	// Now we're guaranteed to be the exclusive connection processing this message!

	{
		systemTraceCtx := propagation.MapCarrier{}
		if err := json.Unmarshal(data.SystemTraceCtx, &systemTraceCtx); err != nil {
			return fmt.Errorf("could not unmarshal system trace ctx: %w", err)
		}

		ctx = trace.SystemTracer().Propagator().Extract(ctx, systemTraceCtx)
	}
	ctx, span := tracer.NewSpan(ctx, "RouteExecutorRequest", accountID, envID)
	defer span.End()

	routeTo, err := getSuitableConnection(ctx, rnd, stateMgr, envID, appID, data.FunctionSlug, log)
	if err != nil && !errors.Is(err, ErrNoHealthyConnection) {
		return fmt.Errorf("could not retrieve suitable connection: %w", err)
	}

	if routeTo == nil {
		log.Warn("no healthy connections")
		metrics.IncrConnectRouterNoHealthyConnectionCounter(ctx, 1, metrics.CounterOpt{
			PkgName: pkgNameRouter,
		})

		return ErrNoHealthyConnection
	}

	gatewayId, err := ulid.Parse(routeTo.GatewayId)
	if err != nil {
		return fmt.Errorf("invalid gatewayId %q: %w", routeTo.GatewayId, err)
	}

	connId, err := ulid.Parse(routeTo.Id)
	if err != nil {
		return fmt.Errorf("invalid connectionID %q: %w", routeTo.Id, err)
	}

	groupHash := routeTo.SyncedWorkerGroups[data.AppId]
	log = log.With("gateway_id", routeTo.GatewayId, "conn_id", routeTo.Id, "group_hash", groupHash)

	group, err := stateMgr.GetWorkerGroupByHash(ctx, envID, groupHash)
	if err != nil {
		return fmt.Errorf("failed to load worker group after successful connection selection: %w", err)
	}

	// Set app name: This is important to help the SDK find the respective function to invoke
	data.AppName = group.AppName

	span.SetAttributes(
		attribute.String("route_to_gateway_id", gatewayId.String()),
		attribute.String("route_to_conn_id", connId.String()),
	)

	// TODO What if something goes wrong inbetween setting idempotency (claiming exclusivity) and forwarding the req?
	// We'll potentially lose data here

	// Forward message to the gateway
	err = publisher.RouteExecutorRequest(ctx, gatewayId, connId, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, " could not forward request to gateway")
		return fmt.Errorf("failed to route request to gateway: %w", err)
	}

	log.Debug("forwarded executor request to gateway")

	return nil
}

func (c *connectRouterSvc) Run(ctx context.Context) error {
	onSubscribed := make(chan struct{})

	go func() {
		err := c.receiver.ReceiveExecutorMessages(ctx, func(_ []byte, data *connect.GatewayExecutorRequestData) {
			log := c.logger.With("env_id", data.EnvId, "app_id", data.AppId, "req_id", data.RequestId, "run_id", data.RunId)

			err := Route(ctx, c.stateManager, c.receiver, c.rnd, c.tracer, log, data)
			if err != nil {
				if errors.Is(err, ErrNoHealthyConnection) {
					err = c.receiver.NackMessage(ctx, data.RequestId, pubsub.AckSourceRouter, syscode.Error{
						Code:    syscode.CodeConnectNoHealthyConnection,
						Message: "Could not find a healthy connection",
					})
					if err != nil {
						log.Error("failed to nack message", "err", err)
					}
					return
				}

				log.Error("failed to route message", "err", err)
			}

			err = c.receiver.AckMessage(ctx, data.RequestId, pubsub.AckSourceRouter)
			if err != nil {
				log.Error("failed to ack message", "err", err)
				// TODO Log error, retry?
				return
			}
		}, onSubscribed)
		if err != nil {
			// TODO Log error, retry?
			return
		}
	}()

	// TODO Periodically ping random gateways via PubSub and only consider them active if they respond in time -> Multiple routers will do this

	eg := errgroup.Group{}
	eg.Go(func() error {
		err := c.receiver.Wait(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("could not listen for pubsub messages: %w", err)
		}
		return nil
	})

	<-onSubscribed

	c.logger.Debug("connect router is ready")

	return eg.Wait()

}

type connWithGroup struct {
	conn  *connect.ConnMetadata
	group *state.WorkerGroup
}

func getSuitableConnection(ctx context.Context, rnd *util.FrandRNG, stateMgr state.StateManager, envID uuid.UUID, appID uuid.UUID, fnSlug string, log *slog.Logger) (*connect.ConnMetadata, error) {
	conns, err := stateMgr.GetConnectionsByAppID(ctx, envID, appID)
	if err != nil {
		return nil, fmt.Errorf("could not get connections by app ID: %w", err)
	}

	if len(conns) == 0 {
		return nil, ErrNoHealthyConnection
	}

	healthy := make([]connWithGroup, 0, len(conns))
	for _, conn := range conns {
		res := isHealthy(ctx, stateMgr, envID, appID, fnSlug, conn, log)
		if res.isHealthy {
			healthy = append(healthy, connWithGroup{
				conn:  conn,
				group: res.workerGroup,
			})
			continue
		}

		if res.shouldDeleteUnhealthyGateway {
			cleanupUnhealthyGateway(stateMgr, conn, log)
		}

		if res.shouldDeleteUnhealthyConnection {
			cleanupUnhealthyConnection(stateMgr, envID, conn, log)
		}
	}

	if len(healthy) == 0 {
		return nil, ErrNoHealthyConnection
	}

	if len(healthy) == 1 {
		return healthy[0].conn, nil
	}

	return pickConnection(healthy, rnd)
}

func cleanupUnhealthyGateway(stateManager state.StateManager, conn *connect.ConnMetadata, log *slog.Logger) {
	gatewayId, err := ulid.Parse(conn.GatewayId)
	if err != nil {
		log.Error("could not clean up unhealthy gateway, invalid gateway ID", "gateway_id", conn.GatewayId, "err", err)
		return
	}

	// Clean up unhealthy gateway
	err = stateManager.DeleteGateway(context.Background(), gatewayId)
	if err != nil {
		log.Error("could not clean up inactive gateway", "gateway_id", conn.GatewayId, "err", err)
	}
}

func cleanupUnhealthyConnection(stateManager state.StateManager, envID uuid.UUID, conn *connect.ConnMetadata, log *slog.Logger) {
	connId, err := ulid.Parse(conn.Id)
	if err != nil {
		log.Error("could not clean up inactive connection, invalid connection ID", "conn_id", conn.Id, "err", err)
		return
	}

	// Clean up unhealthy connection
	err = stateManager.DeleteConnection(context.Background(), envID, connId)
	if err != nil {
		log.Error("could not clean up inactive connection", "conn_id", conn.Id, "err", err)
	}
}

func getConnectionWeight(timeRange float64, oldestVersion time.Time, h connWithGroup) float64 {
	weight := 1.0

	// Calculate weights based on factors like version
	if !h.group.CreatedAt.IsZero() && timeRange > 0 {
		// Normalize to range [1, 10] where newer timestamps get higher weights
		timeDiff := h.group.CreatedAt.Sub(oldestVersion).Seconds()
		normalizedWeight := 1.0 + 9.0*(timeDiff/timeRange)
		weight = normalizedWeight
	}

	return weight
}

type versionTimeDistribution struct {
	oldestVersionCreatedAt time.Time
	newestVersionCreatedAt time.Time
	timeRange              float64
}

func getVersionTimeDistribution(sortedCandidates []connWithGroup) versionTimeDistribution {
	oldestVersion := sortedCandidates[0].group.CreatedAt
	newestVersion := sortedCandidates[len(sortedCandidates)-1].group.CreatedAt

	timeRange := newestVersion.Sub(oldestVersion).Seconds()

	return versionTimeDistribution{
		oldestVersionCreatedAt: oldestVersion,
		newestVersionCreatedAt: newestVersion,
		timeRange:              timeRange,
	}
}

func sortByGroupCreatedAt(candidates []connWithGroup) {
	slices.SortStableFunc(candidates, func(a, b connWithGroup) int {
		if a.group.CreatedAt.After(b.group.CreatedAt) {
			return 1
		}

		return -1
	})
}

func pickConnection(candidates []connWithGroup, rnd *util.FrandRNG) (*connect.ConnMetadata, error) {
	// First, sort candidate connections by CreatedAt timestamp (newest first)
	sortByGroupCreatedAt(candidates)

	// Clamp candidates
	if len(candidates) > 5 {
		candidates = candidates[:5]
	}

	// Get range of versions
	versionTimeDistribution := getVersionTimeDistribution(candidates)

	weights := make([]float64, len(candidates))
	for i, h := range candidates {
		weights[i] = getConnectionWeight(versionTimeDistribution.timeRange, versionTimeDistribution.oldestVersionCreatedAt, h)
	}

	w := sampleuv.NewWeighted(weights, rnd)
	idx, ok := w.Take()
	if !ok {
		return nil, util.ErrWeightedSampleRead
	}
	chosen := candidates[idx]

	return chosen.conn, nil
}

type isHealthyRes struct {
	isHealthy                       bool
	shouldDeleteUnhealthyConnection bool
	shouldDeleteUnhealthyGateway    bool
	workerGroup                     *state.WorkerGroup
}

func isHealthy(ctx context.Context, stateManager state.StateManager, envID uuid.UUID, appID uuid.UUID, fnSlug string, conn *connect.ConnMetadata, log *slog.Logger) isHealthyRes {
	log.Debug("evaluating connection", "connection_id", conn.Id, "status", conn.Status, "last_heartbeat_at", conn.LastHeartbeatAt.AsTime())

	gatewayId, err := ulid.Parse(conn.GatewayId)
	if err != nil {
		log.Error("connection gateway id could not be parsed", "err", err, "gateway_id", conn.GatewayId)

		// Clean up invalid connection
		return isHealthyRes{
			shouldDeleteUnhealthyConnection: true,
		}
	}

	if conn.Status != connect.ConnectionStatus_READY {
		log.Debug("connection is not ready")

		if conn.Status == connect.ConnectionStatus_DISCONNECTED {
			// Clean up disconnected connection
			return isHealthyRes{
				shouldDeleteUnhealthyConnection: true,
			}
		}

		return isHealthyRes{}
	}

	// If more than two consecutive heartbeats were missed, the connection is not healthy
	connectionHeartbeatMissed := conn.LastHeartbeatAt.AsTime().Before(time.Now().Add(-2 * WorkerHeartbeatInterval))
	if connectionHeartbeatMissed {
		log.Debug("last heartbeat is too old")

		// Clean up outdated connection
		return isHealthyRes{
			shouldDeleteUnhealthyConnection: true,
		}
	}

	groupHash, ok := conn.SyncedWorkerGroups[appID.String()]
	if !ok {
		log.Error("connection missing worker group hash for app", "app_id", appID.String(), "synced_worker_groups", conn.SyncedWorkerGroups)

		return isHealthyRes{}
	}

	group, err := stateManager.GetWorkerGroupByHash(ctx, envID, groupHash)
	if err != nil {
		log.Error("could not get worker group for connection", "group_id", groupHash)

		return isHealthyRes{
			shouldDeleteUnhealthyConnection: true,
		}
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

		return isHealthyRes{}
	}

	// Ensure gateway is healthy
	gw, err := stateManager.GetGateway(ctx, gatewayId)
	if err != nil {
		log.Error("could not get gateway", "gateway_id", gatewayId.String())

		return isHealthyRes{
			shouldDeleteUnhealthyConnection: true,
		}
	}

	log.Debug("retrieved gateway for connection", "conn_id", conn.Id, "gateway_id", gatewayId.String(), "status", gw.Status, "last_heartbeat_at", gw.LastHeartbeatAt)

	gatewayIsActive := gw.Status == state.GatewayStatusActive
	gatewayHeartbeatTimedOut := gw.LastHeartbeatAt.Before(time.Now().Add(-2 * GatewayHeartbeatInterval))
	if !gatewayIsActive || gatewayHeartbeatTimedOut {
		log.Debug("gateway is unhealthy", "conn_id", conn.Id, "gateway_id", gatewayId.String(), "status", gw.Status, "last_heartbeat_at", gw.LastHeartbeatAt)

		// Only drop gateway if it's no longer heart-beating, as an inactive gateway may be draining
		if gatewayHeartbeatTimedOut {
			return isHealthyRes{
				shouldDeleteUnhealthyConnection: true,
				shouldDeleteUnhealthyGateway:    true,
			}
		}

		// Drop associated connection
		return isHealthyRes{
			shouldDeleteUnhealthyConnection: true,
		}
	}

	return isHealthyRes{
		isHealthy:   true,
		workerGroup: group,
	}
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
