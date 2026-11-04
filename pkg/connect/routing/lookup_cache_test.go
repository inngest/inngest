package routing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestGetRouteReusesLookupsAndRefreshesBetweenCalls(t *testing.T) {
	ctx := context.Background()
	mgr, cleanup := setupRedis(t)
	defer cleanup()
	conns := make([]testConnection, 100)
	for i := range conns {
		conns[i] = newTestConn(connectpb.ConnectionStatus_READY, time.Now())
	}
	s := setup(t, mgr, setupOpts{}, conns...)
	counted := newCountingRoutingState(mgr)
	data := &connectpb.GatewayExecutorRequestData{
		AccountId: s.acctId.String(), AppId: s.appId.String(), EnvId: s.envId.String(),
		FunctionId: uuid.NewString(), FunctionSlug: s.fnSlug, SystemTraceCtx: []byte("{}"),
	}
	route, err := GetRoute(ctx, counted, rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), data)
	require.NoError(t, err)
	require.Equal(t, s.appName, data.AppName)
	require.Equal(t, s.gatewayId, route.GatewayID)
	require.Equal(t, 1, counted.gatewayReads[s.gatewayId])
	require.Equal(t, 1, counted.capacityReads)
	require.Len(t, counted.groupReads, 1)
	var key workerGroupKey
	for k, reads := range counted.groupReads {
		require.Equal(t, 1, reads)
		key = k
	}

	group, err := mgr.GetWorkerGroupByHash(ctx, s.envId, key.hash)
	require.NoError(t, err)
	group.AppName = "updated-app"
	require.NoError(t, mgr.UpdateWorkerGroup(ctx, s.envId, group))
	_, err = GetRoute(ctx, counted, rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), data)
	require.NoError(t, err)
	require.Equal(t, group.AppName, data.AppName)
	require.Equal(t, 2, counted.groupReads[key])
	require.Equal(t, 2, counted.gatewayReads[s.gatewayId])
	require.Equal(t, 2, counted.capacityReads)

	require.NoError(t, mgr.UpsertGateway(ctx, &state.Gateway{
		Id: s.gatewayId, Status: state.GatewayStatusActive,
		LastHeartbeatAtMS: time.Now().Add(-3 * consts.ConnectGatewayHeartbeatInterval).UnixMilli(),
	}))
	_, err = getSuitableConnection(ctx, rnd, counted, s.envId, s.appId, s.fnSlug, logger.VoidLogger())
	require.ErrorIs(t, err, ErrNoHealthyConnection)
}

func TestRoutingLookupsUseIndependentRecordKeys(t *testing.T) {
	ctx := context.Background()
	counted := newCountingRoutingState(nil)
	counted.getGroup = func(_ context.Context, env uuid.UUID, hash string) (*state.WorkerGroup, error) {
		return &state.WorkerGroup{EnvID: env, Hash: hash}, nil
	}
	counted.getGateway = func(_ context.Context, id ulid.ULID) (*state.Gateway, error) {
		return &state.Gateway{Id: id, LastHeartbeatAtMS: time.Now().UnixMilli()}, nil
	}
	lookups := newTestRoutingLookups(counted)
	envA, envB := uuid.New(), uuid.New()
	for _, key := range []workerGroupKey{{envA, "group-a"}, {envB, "group-a"}, {envA, "group-b"}} {
		for range 2 {
			group, err := lookups.GetWorkerGroupByHash(ctx, key.envID, key.hash)
			require.NoError(t, err)
			require.Equal(t, key.envID, group.EnvID)
			require.Equal(t, key.hash, group.Hash)
		}
		require.Equal(t, 1, counted.groupReads[key])
	}
	for range 2 {
		id := ulid.Make()
		for range 2 {
			gateway, err := lookups.GetGateway(ctx, id)
			require.NoError(t, err)
			require.Equal(t, id, gateway.Id)
		}
		require.Equal(t, 1, counted.gatewayReads[id])
	}
}

func TestRoutingLookupsRetryFailedReads(t *testing.T) {
	ctx := context.Background()
	counted := newCountingRoutingState(nil)
	lookupErr := errors.New("lookup failed")
	counted.getGroup = func(_ context.Context, env uuid.UUID, hash string) (*state.WorkerGroup, error) {
		if counted.groupReads[workerGroupKey{envID: env, hash: hash}] == 1 {
			return nil, lookupErr
		}
		return &state.WorkerGroup{EnvID: env, Hash: hash}, nil
	}
	counted.getGateway = func(_ context.Context, id ulid.ULID) (*state.Gateway, error) {
		if counted.gatewayReads[id] == 1 {
			return nil, lookupErr
		}
		return &state.Gateway{Id: id, LastHeartbeatAtMS: time.Now().UnixMilli()}, nil
	}
	lookups := newTestRoutingLookups(counted)
	env, id := uuid.New(), ulid.Make()
	_, err := lookups.GetWorkerGroupByHash(ctx, env, "group")
	require.ErrorIs(t, err, lookupErr)
	group, err := lookups.GetWorkerGroupByHash(ctx, env, "group")
	require.NoError(t, err)
	require.NotNil(t, group)
	_, err = lookups.GetGateway(ctx, id)
	require.ErrorIs(t, err, lookupErr)
	gateway, err := lookups.GetGateway(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, gateway)
}

func TestRoutingLookupsRefreshExpiredGateway(t *testing.T) {
	ctx := context.Background()
	counted := newCountingRoutingState(nil)
	counted.getGateway = func(_ context.Context, id ulid.ULID) (*state.Gateway, error) {
		return &state.Gateway{Id: id, LastHeartbeatAtMS: time.Now().UnixMilli()}, nil
	}
	lookups := newTestRoutingLookups(counted)
	id := ulid.Make()
	gateway, err := lookups.GetGateway(ctx, id)
	require.NoError(t, err)
	gateway.LastHeartbeatAtMS = time.Now().Add(-3 * consts.ConnectGatewayHeartbeatInterval).UnixMilli()
	fresh, err := lookups.GetGateway(ctx, id)
	require.NoError(t, err)
	require.NotSame(t, gateway, fresh)
	require.Equal(t, 2, counted.gatewayReads[id])
}

func TestRoutingLookupsInvalidateRecordsAfterCleanup(t *testing.T) {
	ctx := context.Background()
	mgr, cleanup := setupRedis(t)
	defer cleanup()
	s := setup(t, mgr, setupOpts{}, newTestConn(connectpb.ConnectionStatus_READY, time.Now()))
	counted := newCountingRoutingState(mgr)
	lookups := newTestRoutingLookups(counted)
	conn, err := mgr.GetConnection(ctx, s.envId, s.connIds[0])
	require.NoError(t, err)
	hash := conn.SyncedWorkerGroups[s.appId.String()]
	_, err = lookups.GetWorkerGroupByHash(ctx, s.envId, hash)
	require.NoError(t, err)
	_, err = lookups.GetGateway(ctx, s.gatewayId)
	require.NoError(t, err)

	deleted, err := lookups.DeleteConnectionIfUnchanged(ctx, s.envId, s.connIds[0], conn)
	require.NoError(t, err)
	require.True(t, deleted)
	_, err = lookups.GetWorkerGroupByHash(ctx, s.envId, hash)
	require.ErrorIs(t, err, state.ErrWorkerGroupNotFound)
	require.Equal(t, 2, counted.groupReads[workerGroupKey{envID: s.envId, hash: hash}])

	require.NoError(t, lookups.DeleteGateway(ctx, s.gatewayId))
	_, err = lookups.GetGateway(ctx, s.gatewayId)
	require.Error(t, err)
	require.Equal(t, 2, counted.gatewayReads[s.gatewayId])
}

func TestGetRouteRefreshesLookupsAfterDrainingRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mgr, cleanup := setupRedis(t)
	defer cleanup()
	s := setup(t, mgr, setupOpts{}, newTestConn(connectpb.ConnectionStatus_READY, time.Now()))
	counted := newCountingRoutingState(mgr)
	counted.getGroup = func(ctx context.Context, env uuid.UUID, hash string) (*state.WorkerGroup, error) {
		group, err := mgr.GetWorkerGroupByHash(ctx, env, hash)
		if counted.groupReads[workerGroupKey{envID: env, hash: hash}] == 1 {
			group.FunctionSlugs = nil
		}
		return group, err
	}
	counted.getConnections = func(ctx context.Context, env, app uuid.UUID) ([]*connectpb.ConnMetadata, error) {
		conns, err := mgr.GetConnectionsByAppID(ctx, env, app)
		if counted.connectionReads == 1 {
			conn := proto.Clone(conns[0]).(*connectpb.ConnMetadata)
			conn.Id = ulid.Make().String()
			conn.Status = connectpb.ConnectionStatus_DRAINING
			conns = append(conns, conn)
		}
		return conns, err
	}
	data := &connectpb.GatewayExecutorRequestData{
		AccountId: s.acctId.String(), AppId: s.appId.String(), EnvId: s.envId.String(),
		FunctionId: uuid.NewString(), FunctionSlug: s.fnSlug, SystemTraceCtx: []byte("{}"),
	}
	_, err := GetRoute(ctx, counted, rnd, trace.NoopConditionalTracer(), logger.VoidLogger(), data)
	require.NoError(t, err)
	require.Equal(t, 2, counted.connectionReads)
	for _, reads := range counted.groupReads {
		require.Equal(t, 2, reads)
	}
}

func TestRoutingLookupsPreserveCapacityExhaustion(t *testing.T) {
	ctx := context.Background()
	mgr, cleanup := setupRedis(t)
	defer cleanup()
	s := setup(t, mgr, setupOpts{},
		newTestConn(connectpb.ConnectionStatus_READY, time.Now()),
		newTestConn(connectpb.ConnectionStatus_READY, time.Now()),
	)
	require.NoError(t, mgr.SetWorkerTotalCapacity(ctx, s.envId, "my-worker", 1))
	require.NoError(t, mgr.AssignRequestToWorker(ctx, s.envId, "my-worker", "request"))
	counted := newCountingRoutingState(mgr)
	_, err := getSuitableConnection(ctx, rnd, counted, s.envId, s.appId, s.fnSlug, logger.VoidLogger())
	require.ErrorIs(t, err, ErrAllWorkersAtCapacity)
	require.Equal(t, 1, counted.capacityReads)
	require.Equal(t, 1, counted.gatewayReads[s.gatewayId])
}

func TestRoutingLookupsPreserveMissingRecordCleanup(t *testing.T) {
	for _, missing := range []string{"worker group", "gateway"} {
		t.Run(missing, func(t *testing.T) {
			ctx := context.Background()
			mgr, cleanup := setupRedis(t)
			defer cleanup()
			s := setup(t, mgr, setupOpts{},
				newTestConn(connectpb.ConnectionStatus_READY, time.Now()),
				newTestConn(connectpb.ConnectionStatus_READY, time.Now()),
			)
			counted := newCountingRoutingState(mgr)
			if missing == "worker group" {
				counted.getGroup = func(context.Context, uuid.UUID, string) (*state.WorkerGroup, error) {
					return nil, state.ErrWorkerGroupNotFound
				}
			} else {
				require.NoError(t, mgr.DeleteGateway(ctx, s.gatewayId))
			}
			_, err := getSuitableConnection(ctx, rnd, counted, s.envId, s.appId, s.fnSlug, logger.VoidLogger())
			require.ErrorIs(t, err, ErrNoHealthyConnection)
			conns, err := mgr.GetConnectionsByAppID(ctx, s.envId, s.appId)
			require.NoError(t, err)
			require.Empty(t, conns)
		})
	}
}

type countingRoutingState struct {
	state.StateManager
	groupReads      map[workerGroupKey]int
	gatewayReads    map[ulid.ULID]int
	capacityReads   int
	connectionReads int
	getGroup        func(context.Context, uuid.UUID, string) (*state.WorkerGroup, error)
	getGateway      func(context.Context, ulid.ULID) (*state.Gateway, error)
	getConnections  func(context.Context, uuid.UUID, uuid.UUID) ([]*connectpb.ConnMetadata, error)
}

func newCountingRoutingState(mgr state.StateManager) *countingRoutingState {
	return &countingRoutingState{
		StateManager: mgr,
		groupReads:   make(map[workerGroupKey]int),
		gatewayReads: make(map[ulid.ULID]int),
	}
}

func newTestRoutingLookups(mgr state.StateManager) *routingLookups {
	return &routingLookups{
		StateManager: mgr,
		groups:       make(map[workerGroupKey]*state.WorkerGroup),
		gateways:     make(map[ulid.ULID]*state.Gateway),
	}
}

func (m *countingRoutingState) GetConnectionsByAppID(ctx context.Context, env, app uuid.UUID) ([]*connectpb.ConnMetadata, error) {
	m.connectionReads++
	if m.getConnections != nil {
		return m.getConnections(ctx, env, app)
	}
	return m.StateManager.GetConnectionsByAppID(ctx, env, app)
}

func (m *countingRoutingState) GetWorkerGroupByHash(ctx context.Context, env uuid.UUID, hash string) (*state.WorkerGroup, error) {
	m.groupReads[workerGroupKey{envID: env, hash: hash}]++
	if m.getGroup != nil {
		return m.getGroup(ctx, env, hash)
	}
	return m.StateManager.GetWorkerGroupByHash(ctx, env, hash)
}

func (m *countingRoutingState) GetGateway(ctx context.Context, id ulid.ULID) (*state.Gateway, error) {
	m.gatewayReads[id]++
	if m.getGateway != nil {
		return m.getGateway(ctx, id)
	}
	return m.StateManager.GetGateway(ctx, id)
}

func (m *countingRoutingState) GetWorkerCapacities(ctx context.Context, env uuid.UUID, instance string) (*state.WorkerCapacity, error) {
	m.capacityReads++
	return m.StateManager.GetWorkerCapacities(ctx, env, instance)
}
