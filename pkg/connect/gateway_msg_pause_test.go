package connect

import (
	"testing"

	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
)

func TestHandleWorkerPauseMarksConnectionDrainingAndStopsForwarding(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)
	res.svc.wsConnections.Store(res.connID.String(), ch)

	serr := ch.handleWorkerPause()
	require.Nil(t, serr)
	require.True(t, ch.draining.Load())

	_, ok := res.svc.wsConnections.Load(res.connID.String())
	require.False(t, ok)

	select {
	case <-ch.stopForwarding:
	default:
		t.Fatal("stopForwarding should be closed")
	}

	conn, err := res.stateManager.GetConnection(t.Context(), res.envID, res.connID)
	require.NoError(t, err)
	require.Equal(t, connectpb.ConnectionStatus_DRAINING, conn.Status)
}
