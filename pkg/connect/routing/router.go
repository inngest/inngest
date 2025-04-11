package routing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/util"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"gonum.org/v1/gonum/stat/sampleuv"
	"log/slog"
	"slices"
	"time"
)

const (
	pkgNameRouter = "connect.router"
)

var ErrNoHealthyConnection = fmt.Errorf("no healthy connection")

type RouteResult struct {
	GatewayID    ulid.ULID
	ConnectionID ulid.ULID
}

func GetRoute(ctx context.Context, stateMgr state.StateManager, rnd *util.FrandRNG, tracer trace.ConditionalTracer, log *slog.Logger, data *connectpb.GatewayExecutorRequestData) (*RouteResult, error) {
	appID, err := uuid.Parse(data.AppId)
	if err != nil {
		return nil, fmt.Errorf("could not parse app ID: %w", err)
	}

	accountID, err := uuid.Parse(data.AccountId)
	if err != nil {
		return nil, fmt.Errorf("could not parse account ID: %w", err)
	}

	envID, err := uuid.Parse(data.EnvId)
	if err != nil {
		return nil, fmt.Errorf("could not parse env ID: %w", err)
	}

	{
		systemTraceCtx := propagation.MapCarrier{}
		if err := json.Unmarshal(data.SystemTraceCtx, &systemTraceCtx); err != nil {
			return nil, fmt.Errorf("could not unmarshal system trace ctx: %w", err)
		}

		ctx = trace.SystemTracer().Propagator().Extract(ctx, systemTraceCtx)
	}
	ctx, span := tracer.NewSpan(ctx, "RouteExecutorRequest", accountID, envID)
	defer span.End()

	routeTo, err := getSuitableConnection(ctx, rnd, stateMgr, envID, appID, data.FunctionSlug, log)
	if err != nil && !errors.Is(err, ErrNoHealthyConnection) {
		return nil, fmt.Errorf("could not retrieve suitable connection: %w", err)
	}

	if routeTo == nil {
		log.Warn("no healthy connections")
		metrics.IncrConnectRouterNoHealthyConnectionCounter(ctx, 1, metrics.CounterOpt{
			PkgName: pkgNameRouter,
		})

		return nil, ErrNoHealthyConnection
	}

	gatewayId, err := ulid.Parse(routeTo.GatewayId)
	if err != nil {
		return nil, fmt.Errorf("invalid gatewayId %q: %w", routeTo.GatewayId, err)
	}

	connId, err := ulid.Parse(routeTo.Id)
	if err != nil {
		return nil, fmt.Errorf("invalid connectionID %q: %w", routeTo.Id, err)
	}

	groupHash := routeTo.SyncedWorkerGroups[data.AppId]

	group, err := stateMgr.GetWorkerGroupByHash(ctx, envID, groupHash)
	if err != nil {
		return nil, fmt.Errorf("failed to load worker group after successful connection selection: %w", err)
	}

	// Set app name: This is important to help the SDK find the respective function to invoke
	data.AppName = group.AppName

	span.SetAttributes(
		attribute.String("route_to_gateway_id", gatewayId.String()),
		attribute.String("route_to_conn_id", connId.String()),
	)

	return &RouteResult{
		GatewayID:    gatewayId,
		ConnectionID: connId,
	}, nil
}

type connWithGroup struct {
	conn  *connectpb.ConnMetadata
	group *state.WorkerGroup
}

func getSuitableConnection(ctx context.Context, rnd *util.FrandRNG, stateMgr state.StateManager, envID uuid.UUID, appID uuid.UUID, fnSlug string, log *slog.Logger) (*connectpb.ConnMetadata, error) {
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

func cleanupUnhealthyGateway(stateManager state.StateManager, conn *connectpb.ConnMetadata, log *slog.Logger) {
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

func cleanupUnhealthyConnection(stateManager state.StateManager, envID uuid.UUID, conn *connectpb.ConnMetadata, log *slog.Logger) {
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

func pickConnection(candidates []connWithGroup, rnd *util.FrandRNG) (*connectpb.ConnMetadata, error) {
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

func isHealthy(ctx context.Context, stateManager state.StateManager, envID uuid.UUID, appID uuid.UUID, fnSlug string, conn *connectpb.ConnMetadata, log *slog.Logger) isHealthyRes {
	log.Debug("evaluating connection", "conn_id", conn.Id, "status", conn.Status, "last_heartbeat_at", conn.LastHeartbeatAt.AsTime())

	gatewayId, err := ulid.Parse(conn.GatewayId)
	if err != nil {
		log.Error("connection gateway id could not be parsed", "err", err, "gateway_id", conn.GatewayId)

		// Clean up invalid connection
		return isHealthyRes{
			shouldDeleteUnhealthyConnection: true,
		}
	}

	if conn.Status != connectpb.ConnectionStatus_READY {
		log.Debug("connection is not ready")

		if conn.Status == connectpb.ConnectionStatus_DISCONNECTED {
			// Clean up disconnected connection
			return isHealthyRes{
				shouldDeleteUnhealthyConnection: true,
			}
		}

		return isHealthyRes{}
	}

	// If more than two consecutive heartbeats were missed, the connection is not healthy
	connectionHeartbeatMissed := conn.LastHeartbeatAt.AsTime().Before(time.Now().Add(-2 * consts.ConnectWorkerHeartbeatInterval))
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
	gatewayHeartbeatTimedOut := gw.LastHeartbeatAt.Before(time.Now().Add(-2 * consts.ConnectGatewayHeartbeatInterval))
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
