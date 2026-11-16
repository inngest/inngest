package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type replacingConnectionForwarder struct {
	mockGatewayGRPCManager
	manager     state.StateManager
	oldID       ulid.ULID
	replacement *state.Connection
}

func (f *replacingConnectionForwarder) Forward(ctx context.Context, gatewayID, connectionID ulid.ULID, data *connectpb.GatewayExecutorRequestData) error {
	if err := f.mockGatewayGRPCManager.Forward(ctx, gatewayID, connectionID, data); err != nil {
		return err
	}
	if connectionID == f.oldID {
		if len(f.getForwardCalls()) > 1 {
			return status.Error(codes.Unavailable, "connection snapshot was not refreshed")
		}
		if err := f.manager.DeleteConnection(ctx, f.replacement.EnvID, f.oldID); err != nil {
			return err
		}
		if err := f.manager.UpsertConnection(ctx, f.replacement, connectpb.ConnectionStatus_READY, time.Now()); err != nil {
			return err
		}
		return status.Error(codes.Unavailable, "worker connection was replaced")
	}
	f.sendResponse(data.RequestId, &connectpb.SDKResponse{RequestId: data.RequestId, Status: connectpb.SDKResponseStatus_DONE})
	return nil
}

func TestProxyRefreshesConnectionSnapshotAfterForwardFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	redis := miniredis.RunT(t)
	client, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{redis.Addr()}, DisableCache: true})
	require.NoError(t, err)
	defer client.Close()
	manager := state.NewRedisConnectionStateManager(client)
	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	gatewayID, oldID, newID := ulid.Make(), ulid.Make(), ulid.Make()
	group := &state.WorkerGroup{
		AccountID: accountID, EnvID: envID, AppID: &appID, AppName: "test-app",
		Hash: "group", FunctionSlugs: []string{"test-function"},
	}
	connection := func(id ulid.ULID, instance string) *state.Connection {
		return &state.Connection{
			AccountID: accountID, EnvID: envID, ConnectionId: id, GatewayId: gatewayID,
			Data: &connectpb.WorkerConnectRequestData{InstanceId: instance},
			Groups: map[string]*state.WorkerGroup{
				group.Hash: group,
			},
		}
	}
	require.NoError(t, manager.UpsertGateway(ctx, &state.Gateway{
		Id: gatewayID, Status: state.GatewayStatusActive, LastHeartbeatAtMS: time.Now().UnixMilli(),
	}))
	require.NoError(t, manager.UpsertConnection(ctx, connection(oldID, "old-worker"), connectpb.ConnectionStatus_READY, time.Now()))
	forwarder := &replacingConnectionForwarder{manager: manager, oldID: oldID, replacement: connection(newID, "new-worker")}
	connector := newGRPCConnector(ctx, GRPCConnectorOpts{
		Tracer: trace.NoopConditionalTracer(), StateManager: manager,
	}, WithGatewayManager(forwarder))
	connector.grpcConfig.Executor.IP = net.IPv4(127, 0, 0, 1)
	response, servedID, err := connector.Proxy(ctx, context.Background(), ProxyOpts{
		AccountID: accountID, EnvID: envID, AppID: appID, FunctionID: functionID,
		Data: &connectpb.GatewayExecutorRequestData{
			RequestId: "request", JobId: "job", RunId: ulid.Make().String(),
			AccountId: accountID.String(), EnvId: envID.String(), AppId: appID.String(),
			FunctionId: functionID.String(), FunctionSlug: "test-function",
		},
	})
	require.NoError(t, err)
	require.Equal(t, connectpb.SDKResponseStatus_DONE, response.Status)
	require.Equal(t, "new-worker", servedID, "the worker that got the re-routed request must be reported")
	calls := forwarder.getForwardCalls()
	require.Len(t, calls, 2)
	require.Equal(t, oldID, calls[0].connectionID)
	require.Equal(t, newID, calls[1].connectionID)
}
