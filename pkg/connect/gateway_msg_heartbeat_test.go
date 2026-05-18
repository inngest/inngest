package connect

import (
	"errors"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
)

func TestHandleWorkerHeartbeatWritesGatewayHeartbeat(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	exchangeHeartbeat(t, res.ws, 2*time.Second)

	res.lifecycles.Assert(t, testRecorderAssertion{
		onConnectedCount: 1,
		onSyncedCount:    1,
		onReadyCount:     1,
		onHeartbeatCount: 1,
	})
}

func TestHandleWorkerHeartbeatKeepsDrainingStatus(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	sendWorkerPauseMessage(t, res.ws)
	exchangeHeartbeat(t, res.ws, 2*time.Second)

	conn, err := res.stateManager.GetConnection(t.Context(), res.envID, res.connID)
	require.NoError(t, err)
	require.Equal(t, connectpb.ConnectionStatus_DRAINING, conn.Status)

	res.lifecycles.Assert(t, testRecorderAssertion{
		onConnectedCount:     1,
		onSyncedCount:        1,
		onReadyCount:         1,
		onHeartbeatCount:     1,
		onStartDrainingCount: 1,
	})
}

func TestHandleWorkerHeartbeatStatusUpdateFailureWritesGatewayHeartbeatUntilThreshold(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	res.svc.stateManager = upsertConnectionErrorStateManager{
		StateManager: res.svc.stateManager,
		err:          errors.New("upsert connection failed"),
	}

	for range maxConsecutiveConnStatusUpdateFailures - 1 {
		exchangeHeartbeat(t, res.ws, 2*time.Second)
	}

	sendWorkerHeartbeatMessage(t, res.ws)
	status, reason := awaitClosure(t, res.ws, 2*time.Second)
	require.Equal(t, websocket.StatusInternalError, status)
	require.Equal(t, syscode.CodeConnectInternal, reason)
}

func TestHandleWorkerHeartbeatMissingInstanceIDIsFatal(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)
	ch.conn = &state.Connection{
		AccountID:    res.accountID,
		EnvID:        res.envID,
		ConnectionId: res.connID,
		GatewayId:    res.svc.gatewayId,
		Data: &connectpb.WorkerConnectRequestData{
			InstanceId: "",
		},
	}

	serr := ch.handleWorkerHeartbeat()
	require.NotNil(t, serr)
	require.Contains(t, serr.Msg, "missing instanceId")
}
