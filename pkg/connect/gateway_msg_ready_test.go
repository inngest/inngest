package connect

import (
	"errors"
	"testing"

	"github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
)

func TestHandleWorkerReadyMarksConnectionReady(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)

	serr := ch.handleWorkerReady()
	require.Nil(t, serr)
	res.lifecycles.Assert(t, testRecorderAssertion{
		onConnectedCount: 1,
		onSyncedCount:    1,
		onReadyCount:     2,
	})
}

func TestHandleWorkerReadyReturnsErrDrainingWhenGatewayIsDraining(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)
	res.svc.isDraining.Store(true)

	serr := ch.handleWorkerReady()
	require.NotNil(t, serr)
	require.Equal(t, ErrDraining.SysCode, serr.SysCode)
}

func TestHandleWorkerReadyIgnoresDrainingConnection(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)
	ch.draining.Store(true)

	serr := ch.handleWorkerReady()
	require.Nil(t, serr)

	conn, err := res.stateManager.GetConnection(t.Context(), res.envID, res.connID)
	require.NoError(t, err)
	require.Equal(t, connectpb.ConnectionStatus_READY, conn.Status)
}

func TestHandleWorkerReadyStatusUpdateFailureIsNonFatalUntilThreshold(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	res.svc.stateManager = upsertConnectionErrorStateManager{
		StateManager: res.svc.stateManager,
		err:          errors.New("upsert connection failed"),
	}

	ch := newTestConnectionHandler(t, res)

	for range maxConsecutiveConnStatusUpdateFailures - 1 {
		serr := ch.handleWorkerReady()
		require.Nil(t, serr)
	}

	serr := ch.handleWorkerReady()
	require.NotNil(t, serr)
	require.Equal(t, syscode.CodeConnectInternal, serr.SysCode)
	require.Contains(t, serr.Msg, "could not update connection status")
}

func TestHandleWorkerReadyMissingInstanceIDIsFatal(t *testing.T) {
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

	serr := ch.handleWorkerReady()
	require.NotNil(t, serr)
	require.Equal(t, syscode.CodeConnectInternal, serr.SysCode)
	require.Contains(t, serr.Msg, "missing instanceId")
}
