package grpc

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type assignRaceStateManager struct {
	state.StateManager
	mu     sync.Mutex
	onCall func(ctx context.Context, envID uuid.UUID, instanceID, requestID string) error
}

func (s *assignRaceStateManager) AssignRequestToWorker(ctx context.Context, envID uuid.UUID, instanceID, requestID string) error {
	s.mu.Lock()
	fn := s.onCall
	s.mu.Unlock()
	if fn != nil {
		if err := fn(ctx, envID, instanceID, requestID); err != nil {
			return err
		}
	}
	return s.StateManager.AssignRequestToWorker(ctx, envID, instanceID, requestID)
}

type failConnectionOnceForwarder struct {
	mockGatewayGRPCManager
	failConnID ulid.ULID
	onFail     func()
}

func (f *failConnectionOnceForwarder) Forward(ctx context.Context, gatewayID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error {
	if connectionID == f.failConnID {
		f.failConnID = ulid.ULID{}
		if f.onFail != nil {
			f.onFail()
		}
		return status.Error(codes.Unavailable, "worker disconnected")
	}
	if err := f.mockGatewayGRPCManager.Forward(ctx, gatewayID, connectionID, data); err != nil {
		return err
	}
	f.sendResponse(data.RequestId, &connectpb.SDKResponse{RequestId: data.RequestId, Status: connectpb.SDKResponseStatus_DONE})
	return nil
}

func TestProxyReRoutesWhenWorkerReachesCapacityBeforeAssignment(t *testing.T) {
	t.Run("initial route loses capacity race and dispatches to free worker", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
		require.NoError(t, err)
		defer rc.Close()

		sm := state.NewRedisConnectionStateManager(rc)
		accountID, envID, appID, fnID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		gatewayID := ulid.Make()
		require.NoError(t, sm.UpsertGateway(ctx, &state.Gateway{
			Id: gatewayID, Status: state.GatewayStatusActive, LastHeartbeatAtMS: time.Now().UnixMilli(),
		}))

		group := &state.WorkerGroup{
			AccountID: accountID, EnvID: envID, AppID: &appID,
			AppName: "test-app", Hash: "test-group", FunctionSlugs: []string{"test-fn"},
		}
		addWorker := func(instanceID string) ulid.ULID {
			connID := ulid.Make()
			require.NoError(t, sm.UpsertConnection(ctx, &state.Connection{
				AccountID: accountID, EnvID: envID, ConnectionId: connID, GatewayId: gatewayID,
				Data:   &connectpb.WorkerConnectRequestData{InstanceId: instanceID},
				Groups: map[string]*state.WorkerGroup{group.Hash: group},
			}, connectpb.ConnectionStatus_READY, time.Now()))
			require.NoError(t, sm.SetWorkerTotalCapacity(ctx, envID, instanceID, 1))
			return connID
		}

		connByWorker := map[string]ulid.ULID{
			"worker-1": addWorker("worker-1"),
			"worker-2": addWorker("worker-2"),
		}

		racedOnce := false
		var racedWorker string
		manager := &assignRaceStateManager{
			StateManager: sm,
			onCall: func(ctx context.Context, envID uuid.UUID, instanceID, requestID string) error {
				if !racedOnce && requestID == "req-1" {
					racedOnce = true
					racedWorker = instanceID
					return sm.AssignRequestToWorker(ctx, envID, instanceID, "concurrent-req")
				}
				return nil
			},
		}

		gateway := &failConnectionOnceForwarder{}
		connector := newGRPCConnector(ctx, GRPCConnectorOpts{
			Tracer: trace.NoopConditionalTracer(), StateManager: manager,
		}, WithGatewayManager(gateway))
		connector.grpcConfig.Executor.IP = net.IPv4(127, 0, 0, 1)

		resp, err := connector.Proxy(ctx, context.Background(), ProxyOpts{
			AccountID: accountID, EnvID: envID, AppID: appID, FunctionID: fnID,
			Data: &connectpb.GatewayExecutorRequestData{
				RequestId: "req-1", JobId: "req-1", RunId: ulid.Make().String(),
				AccountId: accountID.String(), EnvId: envID.String(), AppId: appID.String(),
				FunctionId: fnID.String(), FunctionSlug: "test-fn",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, connectpb.SDKResponseStatus_DONE, resp.Status)

		calls := gateway.getForwardCalls()
		require.Len(t, calls, 1)
		otherWorker := "worker-1"
		if racedWorker == "worker-1" {
			otherWorker = "worker-2"
		}
		require.Equal(t, connByWorker[otherWorker], calls[0].connectionID)
	})

	t.Run("initial route loses capacity race and all workers are now full", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
		require.NoError(t, err)
		defer rc.Close()

		sm := state.NewRedisConnectionStateManager(rc)
		accountID, envID, appID, fnID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		gatewayID, connID := ulid.Make(), ulid.Make()
		require.NoError(t, sm.UpsertGateway(ctx, &state.Gateway{
			Id: gatewayID, Status: state.GatewayStatusActive, LastHeartbeatAtMS: time.Now().UnixMilli(),
		}))
		group := &state.WorkerGroup{
			AccountID: accountID, EnvID: envID, AppID: &appID,
			AppName: "test-app", Hash: "test-group", FunctionSlugs: []string{"test-fn"},
		}
		require.NoError(t, sm.UpsertConnection(ctx, &state.Connection{
			AccountID: accountID, EnvID: envID, ConnectionId: connID, GatewayId: gatewayID,
			Data:   &connectpb.WorkerConnectRequestData{InstanceId: "worker-1"},
			Groups: map[string]*state.WorkerGroup{group.Hash: group},
		}, connectpb.ConnectionStatus_READY, time.Now()))
		require.NoError(t, sm.SetWorkerTotalCapacity(ctx, envID, "worker-1", 1))

		racedOnce := false
		manager := &assignRaceStateManager{
			StateManager: sm,
			onCall: func(ctx context.Context, envID uuid.UUID, instanceID, requestID string) error {
				if !racedOnce && requestID == "req-full" {
					racedOnce = true
					return sm.AssignRequestToWorker(ctx, envID, instanceID, "concurrent-req")
				}
				return nil
			},
		}

		gateway := &mockGatewayGRPCManager{}
		connector := newGRPCConnector(ctx, GRPCConnectorOpts{
			Tracer: trace.NoopConditionalTracer(), StateManager: manager,
		}, WithGatewayManager(gateway))
		connector.grpcConfig.Executor.IP = net.IPv4(127, 0, 0, 1)

		resp, err := connector.Proxy(ctx, context.Background(), ProxyOpts{
			AccountID: accountID, EnvID: envID, AppID: appID, FunctionID: fnID,
			Data: &connectpb.GatewayExecutorRequestData{
				RequestId: "req-full", JobId: "req-full", RunId: ulid.Make().String(),
				AccountId: accountID.String(), EnvId: envID.String(), AppId: appID.String(),
				FunctionId: fnID.String(), FunctionSlug: "test-fn",
			},
		})
		require.Nil(t, resp)
		var serr syscode.Error
		require.ErrorAs(t, err, &serr)
		require.Equal(t, syscode.CodeConnectRequestAssignWorkerReachedCapacity, serr.Code)
		require.Empty(t, gateway.getForwardCalls())

		leased, err := sm.IsRequestLeased(ctx, envID, "req-full")
		require.NoError(t, err)
		require.False(t, leased, "unsent request lease must be released when capacity race exhausts workers")
	})

	t.Run("forward failure re-route does not forward to a worker that reached capacity", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
		require.NoError(t, err)
		defer rc.Close()

		sm := state.NewRedisConnectionStateManager(rc)
		accountID, envID, appID, fnID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		gatewayID := ulid.Make()
		require.NoError(t, sm.UpsertGateway(ctx, &state.Gateway{
			Id: gatewayID, Status: state.GatewayStatusActive, LastHeartbeatAtMS: time.Now().UnixMilli(),
		}))
		group := &state.WorkerGroup{
			AccountID: accountID, EnvID: envID, AppID: &appID,
			AppName: "test-app", Hash: "test-group", FunctionSlugs: []string{"test-fn"},
		}
		addWorker := func(instanceID string) ulid.ULID {
			connID := ulid.Make()
			require.NoError(t, sm.UpsertConnection(ctx, &state.Connection{
				AccountID: accountID, EnvID: envID, ConnectionId: connID, GatewayId: gatewayID,
				Data:   &connectpb.WorkerConnectRequestData{InstanceId: instanceID},
				Groups: map[string]*state.WorkerGroup{group.Hash: group},
			}, connectpb.ConnectionStatus_READY, time.Now()))
			require.NoError(t, sm.SetWorkerTotalCapacity(ctx, envID, instanceID, 1))
			return connID
		}

		conn1 := addWorker("worker-1")
		conn2 := addWorker("worker-2")
		conn3 := addWorker("worker-3")
		require.NoError(t, sm.AssignRequestToWorker(ctx, envID, "worker-2", "busy-2"))
		require.NoError(t, sm.AssignRequestToWorker(ctx, envID, "worker-3", "busy-3"))

		racedOnRetry := false
		var retryRacedConn ulid.ULID
		manager := &assignRaceStateManager{
			StateManager: sm,
			onCall: func(ctx context.Context, envID uuid.UUID, instanceID, requestID string) error {
				if !racedOnRetry && requestID == "req-reroute" && instanceID != "worker-1" {
					racedOnRetry = true
					if instanceID == "worker-2" {
						retryRacedConn = conn2
					} else {
						retryRacedConn = conn3
					}
					return sm.AssignRequestToWorker(ctx, envID, instanceID, "concurrent-retry-req")
				}
				return nil
			},
		}

		gateway := &failConnectionOnceForwarder{
			failConnID: conn1,
			onFail: func() {
				require.NoError(t, sm.DeleteConnection(ctx, envID, conn1))
				require.NoError(t, sm.DeleteRequestFromWorker(ctx, envID, "worker-2", "busy-2"))
				require.NoError(t, sm.DeleteRequestFromWorker(ctx, envID, "worker-3", "busy-3"))
			},
		}
		connector := newGRPCConnector(ctx, GRPCConnectorOpts{
			Tracer: trace.NoopConditionalTracer(), StateManager: manager,
		}, WithGatewayManager(gateway))
		connector.grpcConfig.Executor.IP = net.IPv4(127, 0, 0, 1)

		resp, err := connector.Proxy(ctx, context.Background(), ProxyOpts{
			AccountID: accountID, EnvID: envID, AppID: appID, FunctionID: fnID,
			Data: &connectpb.GatewayExecutorRequestData{
				RequestId: "req-reroute", JobId: "req-reroute", RunId: ulid.Make().String(),
				AccountId: accountID.String(), EnvId: envID.String(), AppId: appID.String(),
				FunctionId: fnID.String(), FunctionSlug: "test-fn",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, connectpb.SDKResponseStatus_DONE, resp.Status)

		calls := gateway.getForwardCalls()
		require.Len(t, calls, 1, "must not forward to the worker that rejected capacity assignment")
		require.NotEqual(t, retryRacedConn, calls[0].connectionID, "must re-route away from worker at capacity")
	})
}
