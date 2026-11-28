package grpc

import (
	"context"
	"net"
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
)

type assignmentCapacityRaceStateManager struct {
	state.StateManager
	failNextAssignment bool
}

func (s *assignmentCapacityRaceStateManager) AssignRequestToWorker(ctx context.Context, envID uuid.UUID, instanceID, requestID string) error {
	if s.failNextAssignment {
		s.failNextAssignment = false
		return state.ErrWorkerCapacityExceeded
	}
	return s.StateManager.AssignRequestToWorker(ctx, envID, instanceID, requestID)
}

func TestProxyRetriesUnsentRequestWithoutWaitingForLeaseExpiry(t *testing.T) {
	for _, tt := range []struct {
		name              string
		missingConnection bool
		capacityRace      bool
		errorCode         string
	}{
		{name: "no healthy connection", missingConnection: true, errorCode: syscode.CodeConnectNoHealthyConnection},
		{name: "all workers at capacity", errorCode: syscode.CodeConnectAllWorkersAtCapacity},
		{name: "capacity taken after routing", capacityRace: true, errorCode: syscode.CodeConnectRequestAssignWorkerReachedCapacity},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			defer rc.Close()
			sm := state.NewRedisConnectionStateManager(rc)
			manager := &assignmentCapacityRaceStateManager{StateManager: sm, failNextAssignment: tt.capacityRace}
			gateway := &mockGatewayGRPCManager{}
			connector := newGRPCConnector(ctx, GRPCConnectorOpts{
				Tracer: trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace), StateManager: manager,
			}, WithGatewayManager(gateway))
			connector.grpcConfig.Executor.IP = net.IPv4(127, 0, 0, 1)

			accountID, envID, appID, fnID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			const requestID, instanceID = "stable-job-id", "worker"
			addWorker := func() {
				gatewayID, connectionID := ulid.Make(), ulid.Make()
				group := &state.WorkerGroup{AccountID: accountID, EnvID: envID, AppID: &appID,
					AppName: "test-app", Hash: "test-group", FunctionSlugs: []string{"test-fn"}}
				require.NoError(t, sm.UpsertConnection(ctx, &state.Connection{
					AccountID: accountID, EnvID: envID, ConnectionId: connectionID, GatewayId: gatewayID,
					Data:   &connectpb.WorkerConnectRequestData{InstanceId: instanceID},
					Groups: map[string]*state.WorkerGroup{group.Hash: group},
				}, connectpb.ConnectionStatus_READY, time.Now()))
				require.NoError(t, sm.UpsertGateway(ctx, &state.Gateway{
					Id: gatewayID, Status: state.GatewayStatusActive, LastHeartbeatAtMS: time.Now().UnixMilli(),
				}))
				require.NoError(t, sm.SetWorkerTotalCapacity(ctx, envID, instanceID, 1))
			}
			if !tt.missingConnection {
				addWorker()
				if !tt.capacityRace {
					require.NoError(t, sm.AssignRequestToWorker(ctx, envID, instanceID, "other-work"))
				}
			}
			opts := ProxyOpts{AccountID: accountID, EnvID: envID, AppID: appID, FunctionID: fnID,
				Data: &connectpb.GatewayExecutorRequestData{
					RequestId: requestID, JobId: requestID, RunId: ulid.Make().String(), AccountId: accountID.String(),
					EnvId: envID.String(), AppId: appID.String(), FunctionId: fnID.String(), FunctionSlug: "test-fn",
				}}

			response, err := connector.Proxy(ctx, context.Background(), opts)
			require.Nil(t, response)
			var routeErr syscode.Error
			require.ErrorAs(t, err, &routeErr)
			require.Equal(t, tt.errorCode, routeErr.Code)
			require.Empty(t, gateway.getForwardCalls(), "routing failure must not dispatch SDK work")

			if tt.missingConnection {
				addWorker()
			} else if !tt.capacityRace {
				require.NoError(t, sm.DeleteRequestFromWorker(ctx, envID, instanceID, "other-work"))
			}
			capacity, err := sm.GetWorkerCapacities(ctx, envID, instanceID)
			require.NoError(t, err)
			require.EqualValues(t, 1, capacity.Available)

			// Retry the same stable ID while the abandoned two-minute lease
			// would still be live. It must forward as soon as a worker is ready.
			type result struct {
				response *connectpb.SDKResponse
				err      error
			}
			results := make(chan result, 1)
			go func() {
				response, err := connector.Proxy(ctx, context.Background(), opts)
				results <- result{response, err}
			}()
			require.Eventually(t, func() bool { return len(gateway.getForwardCalls()) == 1 }, time.Second, time.Millisecond,
				"retry waited on an unsent request lease despite available capacity")
			expected := &connectpb.SDKResponse{RequestId: requestID, Status: connectpb.SDKResponseStatus_DONE}
			require.NoError(t, sm.SaveResponse(ctx, envID, requestID, expected))
			gateway.sendResponse(requestID, expected)
			select {
			case res := <-results:
				require.NoError(t, res.err)
				require.Equal(t, expected.Status, res.response.Status)
			case <-ctx.Done():
				t.Fatal("retry did not receive the SDK response")
			}
			leased, err := sm.IsRequestLeased(ctx, envID, requestID)
			require.NoError(t, err)
			require.False(t, leased)
			capacity, err = sm.GetWorkerCapacities(ctx, envID, instanceID)
			require.NoError(t, err)
			require.EqualValues(t, 1, capacity.Available)
		})
	}
}
