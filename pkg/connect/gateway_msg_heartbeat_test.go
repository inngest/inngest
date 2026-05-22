package connect

import (
	"errors"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/assert"
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
	sendWorkerHeartbeatMessage(t, res.ws)

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		conn, err := res.stateManager.GetConnection(t.Context(), res.envID, res.connID)
		assert.NoError(ct, err)
		if conn != nil {
			assert.Equal(ct, connectpb.ConnectionStatus_DRAINING, conn.Status)
		}
	}, 2*time.Second, 100*time.Millisecond)

	res.lifecycles.Assert(t, testRecorderAssertion{
		onConnectedCount:     1,
		onSyncedCount:        1,
		onReadyCount:         1,
		onHeartbeatCount:     1,
		onStartDrainingCount: 1,
	})
}

func TestHandleWorkerHeartbeatKeepsManualReadinessConnectionConnected(t *testing.T) {
	res := createTestingGateway(t)

	msg := awaitNextMessage(t, res.ws, 2*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_GATEWAY_HELLO, msg.Kind)

	res.reqData.WorkerManualReadinessAck = true
	sendWorkerConnectMessage(t, res)

	msg = awaitNextMessage(t, res.ws, 5*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_GATEWAY_CONNECTION_READY, msg.Kind)

	handler := awaitConnectionHandler(t, res)
	require.Equal(t, gatewayConnPhaseHandshaking, handler.phase())

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		conn, err := res.stateManager.GetConnection(t.Context(), res.envID, res.connID)
		assert.NoError(ct, err)
		if conn != nil {
			assert.Equal(ct, connectpb.ConnectionStatus_CONNECTED, conn.Status)
		}
	}, 2*time.Second, 100*time.Millisecond)

	sendWorkerHeartbeatMessage(t, res.ws)

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		res.lifecycles.lock.Lock()
		defer res.lifecycles.lock.Unlock()

		assert.Equal(ct, 1, len(res.lifecycles.onHeartbeat))
	}, 2*time.Second, 100*time.Millisecond)

	require.Equal(t, gatewayConnPhaseHandshaking, handler.phase())

	conn, err := res.stateManager.GetConnection(t.Context(), res.envID, res.connID)
	require.NoError(t, err)
	require.Equal(t, connectpb.ConnectionStatus_CONNECTED, conn.Status)

	res.lifecycles.Assert(t, testRecorderAssertion{
		onConnectedCount: 1,
		onSyncedCount:    1,
		onReadyCount:     0,
		onHeartbeatCount: 1,
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
