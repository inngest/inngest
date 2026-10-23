package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestProxyRecoversRequestAfterExecutorDeath(t *testing.T) {
	for _, tt := range []struct {
		name     string
		buffered bool
		status   connectpb.SDKResponseStatus
	}{
		{name: "running request", status: connectpb.SDKResponseStatus_DONE},
		{name: "buffered success", buffered: true, status: connectpb.SDKResponseStatus_DONE},
		{name: "running request returns error", status: connectpb.SDKResponseStatus_ERROR},
		{name: "buffered error", buffered: true, status: connectpb.SDKResponseStatus_ERROR},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			defer rc.Close()
			sm := state.NewRedisConnectionStateManager(rc)
			gateway := &mockGatewayGRPCManager{}
			connector := newGRPCConnector(ctx, GRPCConnectorOpts{
				Tracer: trace.NewConditionalTracer(trace.ConnectTracer(), trace.AlwaysTrace), StateManager: sm,
			}, WithGatewayManager(gateway))
			connector.grpcConfig.Executor.IP = net.IPv4(127, 0, 0, 2)

			accountID, envID, appID, fnID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			const requestID, instanceID = "stable-job-id", "original-worker"
			// Seed the request dispatched by the executor that subsequently died.
			require.NoError(t, sm.SetWorkerTotalCapacity(ctx, envID, instanceID, 1))
			leaseID, err := sm.LeaseRequest(ctx, envID, requestID, consts.ConnectWorkerRequestLeaseDuration, net.IPv4(127, 0, 0, 1))
			require.NoError(t, err)
			require.NoError(t, sm.AssignRequestToWorker(ctx, envID, instanceID, requestID))
			expected := &connectpb.SDKResponse{RequestId: requestID, Status: tt.status, Body: []byte(`{"ok":true}`)}
			opts := ProxyOpts{AccountID: accountID, EnvID: envID, AppID: appID, FunctionID: fnID,
				Data: &connectpb.GatewayExecutorRequestData{
					RequestId: requestID, JobId: requestID, RunId: ulid.Make().String(), AccountId: accountID.String(),
					EnvId: envID.String(), AppId: appID.String(), FunctionId: fnID.String(), FunctionSlug: "test-fn",
				}}

			type result struct {
				response *connectpb.SDKResponse
				err      error
			}
			callProxy := func() <-chan result {
				results := make(chan result, 1)
				go func() {
					response, err := connector.Proxy(ctx, context.Background(), opts)
					results <- result{response, err}
				}()
				return results
			}
			await := func(results <-chan result) *connectpb.SDKResponse {
				select {
				case res := <-results:
					require.NoError(t, res.err)
					return res.response
				case <-ctx.Done():
					t.Fatal("proxy did not receive the SDK response")
					return nil
				}
			}

			if tt.buffered {
				require.NoError(t, sm.SaveResponse(ctx, envID, requestID, expected))
			}
			results := callProxy()
			if !tt.buffered {
				require.Eventually(t, func() bool {
					ip, err := sm.GetExecutorIP(ctx, envID, requestID)
					return err == nil && ip.Equal(connector.grpcConfig.Executor.IP)
				}, time.Second, time.Millisecond)
				// The worker still owns its original lease after executor takeover.
				_, err = sm.ExtendRequestLease(ctx, envID, instanceID, requestID, *leaseID, consts.ConnectWorkerRequestLeaseDuration, false)
				require.NoError(t, err)
				capacity, err := sm.GetWorkerCapacities(ctx, envID, instanceID)
				require.NoError(t, err)
				require.Zero(t, capacity.Available)
				require.NoError(t, sm.SaveResponse(ctx, envID, requestID, expected))
				gateway.sendResponse(requestID, expected)
			}
			require.Equal(t, expected.Status, await(results).Status)
			require.Empty(t, gateway.getForwardCalls(), "recovery must not dispatch another SDK request")
			leased, err := sm.IsRequestLeased(ctx, envID, requestID)
			require.NoError(t, err)
			require.False(t, leased)
			capacity, err := sm.GetWorkerCapacities(ctx, envID, instanceID)
			require.NoError(t, err)
			require.EqualValues(t, 1, capacity.Available)
			buffered, err := sm.GetResponse(ctx, envID, requestID)
			require.NoError(t, err)
			require.Nil(t, buffered)

			if tt.status == connectpb.SDKResponseStatus_ERROR {
				// A genuine retry must forward fresh work after consuming the error.
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
				opts.Data.RequestPayload = []byte(`{"ctx":{"attempt":1}}`)
				results = callProxy()
				require.Eventually(t, func() bool { return len(gateway.getForwardCalls()) == 1 }, time.Second, time.Millisecond)
				require.Equal(t, opts.Data.RequestPayload, gateway.getForwardCalls()[0].data.RequestPayload)
				success := &connectpb.SDKResponse{RequestId: requestID, Status: connectpb.SDKResponseStatus_DONE}
				require.NoError(t, sm.SaveResponse(ctx, envID, requestID, success))
				gateway.sendResponse(requestID, success)
				require.Equal(t, connectpb.SDKResponseStatus_DONE, await(results).Status)
			}
		})
	}
}
