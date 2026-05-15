package connect

import (
	"testing"

	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
)

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
