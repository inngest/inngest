package connect

import (
	"context"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/logger"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionHandlerPhaseHelpers(t *testing.T) {
	ch := &connectionHandler{
		log: logger.StdlibLogger(context.Background(), logger.WithLoggerLevel(logger.LevelEmergency)),
	}

	require.Equal(t, gatewayConnPhaseNew, ch.phase())

	ch.markHandshaking("test handshake")
	require.Equal(t, gatewayConnPhaseHandshaking, ch.phase())

	ch.markReady("test ready")
	require.Equal(t, gatewayConnPhaseReady, ch.phase())
	require.False(t, ch.draining.Load())

	ch.beginDrain("test drain")
	require.Equal(t, gatewayConnPhaseDraining, ch.phase())
	require.True(t, ch.draining.Load())

	ch.beginDisconnect("test disconnect")
	require.Equal(t, gatewayConnPhaseDisconnecting, ch.phase())

	ch.markClosed("test closed")
	require.Equal(t, gatewayConnPhaseClosed, ch.phase())

	ch.markClosed("test closed again")
	require.Equal(t, gatewayConnPhaseClosed, ch.phase())

	ch.beginDrain("late drain")
	require.Equal(t, gatewayConnPhaseClosed, ch.phase())
}

func TestGatewayConnectionPhaseHandshakeReady(t *testing.T) {
	res := createTestingGateway(t, testingParameters{silent: true})
	handshake(t, res)

	handler := awaitConnectionHandler(t, res)
	require.Equal(t, gatewayConnPhaseReady, handler.phase())
}

func TestGatewayConnectionPhaseWorkerPauseDraining(t *testing.T) {
	res := createTestingGateway(t, testingParameters{silent: true})
	handshake(t, res)

	handler := awaitConnectionHandler(t, res)
	require.Equal(t, gatewayConnPhaseReady, handler.phase())

	sendWorkerPauseMessage(t, res.ws)

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		assert.Equal(ct, gatewayConnPhaseDraining, handler.phase())
		assert.True(ct, handler.draining.Load())
	}, 2*time.Second, 100*time.Millisecond)
}

func TestGatewayConnectionPhaseGatewayDrainDraining(t *testing.T) {
	res := createTestingGateway(t, testingParameters{
		drainAckTimeout: 2 * time.Second,
		silent:          true,
	})
	handshake(t, res)

	handler := awaitConnectionHandler(t, res)
	require.Equal(t, gatewayConnPhaseReady, handler.phase())

	require.NoError(t, res.svc.DrainGateway())

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		assert.Equal(ct, gatewayConnPhaseDraining, handler.phase())
		assert.True(ct, handler.draining.Load())
	}, 2*time.Second, 100*time.Millisecond)

	msg := awaitNextMessage(t, res.ws, 3*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_GATEWAY_CLOSING, msg.Kind)
}

func TestGatewayConnectionPhaseReadLoopExitClosed(t *testing.T) {
	res := createTestingGateway(t, testingParameters{silent: true})
	handshake(t, res)

	handler := awaitConnectionHandler(t, res)
	require.Equal(t, gatewayConnPhaseReady, handler.phase())

	require.NoError(t, res.ws.Close(websocket.StatusNormalClosure, connectpb.WorkerDisconnectReason_WORKER_SHUTDOWN.String()))

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		assert.Equal(ct, gatewayConnPhaseClosed, handler.phase())
	}, 3*time.Second, 100*time.Millisecond)
}

func awaitConnectionHandler(t *testing.T, res testingResources) *connectionHandler {
	t.Helper()

	var handler *connectionHandler
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		val, ok := res.svc.wsConnections.Load(res.connID.String())
		if !assert.True(ct, ok) {
			return
		}

		var typeOK bool
		handler, typeOK = val.(*connectionHandler)
		assert.True(ct, typeOK)
	}, 2*time.Second, 100*time.Millisecond)

	return handler
}
