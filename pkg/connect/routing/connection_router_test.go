package routing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func routeData(s setupRes) *connectpb.GatewayExecutorRequestData {
	return &connectpb.GatewayExecutorRequestData{
		AccountId: s.acctId.String(), AppId: s.appId.String(), EnvId: s.envId.String(),
		FunctionId: uuid.NewString(), FunctionSlug: s.fnSlug, SystemTraceCtx: []byte("{}"),
	}
}

func TestRouterCachesConnectionsFor250Milliseconds(t *testing.T) {
	mgr, cleanup := setupRedis(t)
	defer cleanup()
	s := setup(t, mgr, setupOpts{}, newTestConn(connectpb.ConnectionStatus_READY, time.Now()))
	counted := newCountingRoutingState(mgr)
	router := NewRouter(counted)
	clock := clockwork.NewFakeClock()
	router.snapshots.clock = clock
	for range 2 {
		route, err := router.GetRoute(context.Background(), rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), routeData(s))
		require.NoError(t, err)
		require.Equal(t, s.connIds[0], route.ConnectionID)
	}
	require.Equal(t, 1, counted.connectionReads)
	require.Equal(t, 2, counted.capacityReads)
	require.Equal(t, 2, counted.gatewayReads[s.gatewayId])
	clock.Advance(250 * time.Millisecond)
	_, err := router.GetRoute(context.Background(), rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), routeData(s))
	require.NoError(t, err)
	require.Equal(t, 2, counted.connectionReads)
}

func TestRouterKeepsCapacityChecksLive(t *testing.T) {
	ctx := context.Background()
	mgr, cleanup := setupRedis(t)
	defer cleanup()
	s := setup(t, mgr, setupOpts{}, newTestConn(connectpb.ConnectionStatus_READY, time.Now()))
	require.NoError(t, mgr.SetWorkerTotalCapacity(ctx, s.envId, "my-worker", 1))
	counted := newCountingRoutingState(mgr)
	router := NewRouter(counted)
	router.snapshots.clock = clockwork.NewFakeClock()
	_, err := router.GetRoute(ctx, rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), routeData(s))
	require.NoError(t, err)
	require.NoError(t, mgr.AssignRequestToWorker(ctx, s.envId, "my-worker", "other-request"))
	_, err = router.GetRoute(ctx, rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), routeData(s))
	require.ErrorIs(t, err, ErrAllWorkersAtCapacity)
	require.NoError(t, mgr.DeleteRequestFromWorker(ctx, s.envId, "my-worker", "other-request"))
	_, err = router.GetRoute(ctx, rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), routeData(s))
	require.NoError(t, err)
}

func TestRouterRefreshesSnapshotBeforeCleanup(t *testing.T) {
	ctx := context.Background()
	mgr, cleanup := setupRedis(t)
	defer cleanup()
	s := setup(t, mgr, setupOpts{}, newTestConn(connectpb.ConnectionStatus_READY, time.Now()))
	counted := newCountingRoutingState(mgr)
	counted.getConnections = func(ctx context.Context, env, app uuid.UUID) ([]*connectpb.ConnMetadata, error) {
		connections, err := mgr.GetConnectionsByAppID(ctx, env, app)
		if err == nil && counted.connectionReads == 1 {
			connections[0].LastHeartbeatAt = timestamppb.New(time.Now().Add(-time.Minute))
		}
		return connections, err
	}
	router := NewRouter(counted)
	router.snapshots.clock = clockwork.NewFakeClock()
	_, _, err := router.snapshots.get(ctx, counted, connectionSnapshotKey{envID: s.envId, appID: s.appId})
	require.NoError(t, err)
	route, err := router.GetRoute(ctx, rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), routeData(s))
	require.NoError(t, err)
	require.Equal(t, s.connIds[0], route.ConnectionID)
	require.Equal(t, 2, counted.connectionReads)
	connection, err := mgr.GetConnection(ctx, s.envId, s.connIds[0])
	require.NoError(t, err)
	require.NotNil(t, connection)
}

func TestRouterRefreshesExhaustedSnapshotAndInvalidatesCleanup(t *testing.T) {
	ctx := context.Background()
	mgr, cleanup := setupRedis(t)
	defer cleanup()
	s := setup(t, mgr, setupOpts{}, newTestConn(connectpb.ConnectionStatus_READY, time.Now()))
	counted := newCountingRoutingState(mgr)
	router := NewRouter(counted)
	router.snapshots.clock = clockwork.NewFakeClock()
	_, err := router.GetRoute(ctx, rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), routeData(s))
	require.NoError(t, err)
	require.NoError(t, mgr.UpsertGateway(ctx, &state.Gateway{
		Id: s.gatewayId, Status: state.GatewayStatusActive,
		LastHeartbeatAtMS: time.Now().Add(-3 * consts.ConnectGatewayHeartbeatInterval).UnixMilli(),
	}))
	replacement := setup(t, mgr, setupOpts{acctId: s.acctId, envId: s.envId, appId: s.appId, syncId: s.syncId},
		newTestConn(connectpb.ConnectionStatus_READY, time.Now()))
	route, err := router.GetRoute(ctx, rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), routeData(s))
	require.NoError(t, err)
	require.Equal(t, replacement.connIds[0], route.ConnectionID)
	require.Equal(t, 2, counted.connectionReads)
	_, err = router.GetRoute(ctx, rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), routeData(s))
	require.NoError(t, err)
	require.Equal(t, 3, counted.connectionReads)
}

func TestRouterInvalidationObservesReplacementImmediately(t *testing.T) {
	ctx := context.Background()
	mgr, cleanup := setupRedis(t)
	defer cleanup()
	s := setup(t, mgr, setupOpts{}, newTestConn(connectpb.ConnectionStatus_READY, time.Now()))
	counted := newCountingRoutingState(mgr)
	router := NewRouter(counted)
	router.snapshots.clock = clockwork.NewFakeClock()
	_, err := router.GetRoute(ctx, rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), routeData(s))
	require.NoError(t, err)
	require.NoError(t, mgr.DeleteConnection(ctx, s.envId, s.connIds[0]))
	replacement := setup(t, mgr, setupOpts{acctId: s.acctId, envId: s.envId, appId: s.appId, syncId: s.syncId},
		newTestConn(connectpb.ConnectionStatus_READY, time.Now()))
	router.InvalidateConnections(s.envId, s.appId)
	route, err := router.GetRoute(ctx, rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), routeData(s))
	require.NoError(t, err)
	require.Equal(t, replacement.connIds[0], route.ConnectionID)
	require.Equal(t, 2, counted.connectionReads)
}
