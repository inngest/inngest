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

func TestConnectionHandlerPhaseEligibility(t *testing.T) {
	ch := &connectionHandler{
		log: logger.StdlibLogger(context.Background(), logger.WithLoggerLevel(logger.LevelEmergency)),
	}

	require.False(t, ch.canForward())
	require.False(t, ch.canWrite(connectpb.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST))

	ch.markReady("test ready")
	require.True(t, ch.canForward())
	require.True(t, ch.canWrite(connectpb.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST))
	require.True(t, ch.canWrite(connectpb.GatewayMessageType_GATEWAY_HEARTBEAT))

	ch.beginDrain("test drain")
	require.False(t, ch.canForward())
	require.False(t, ch.canWrite(connectpb.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST))
	require.False(t, ch.canWrite(connectpb.GatewayMessageType_GATEWAY_HEARTBEAT))
	require.True(t, ch.canWrite(connectpb.GatewayMessageType_WORKER_REPLY_ACK))
	require.True(t, ch.canWrite(connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE_ACK))

	ch.beginDisconnect("test disconnect")
	require.False(t, ch.canForward())
	require.True(t, ch.canWrite(connectpb.GatewayMessageType_WORKER_REPLY_ACK))
	require.False(t, ch.canWrite(connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE_ACK))

	ch.markClosed("test closed")
	require.False(t, ch.canForward())
	require.False(t, ch.canWrite(connectpb.GatewayMessageType_WORKER_REPLY_ACK))
}

func TestConnectionHandlerPhaseReleasesPendingAcks(t *testing.T) {
	ch := &connectionHandler{
		log: logger.StdlibLogger(context.Background(), logger.WithLoggerLevel(logger.LevelEmergency)),
	}
	ch.markReady("test ready")

	ackCh := make(chan error, 1)
	ch.pendingAcks.Store("req-1", ackCh)

	ch.beginDisconnect("test disconnect")

	select {
	case err := <-ackCh:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("pending ack should be released when leaving Ready")
	}

	_, ok := ch.pendingAcks.Load("req-1")
	require.False(t, ok)
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

func TestForwardRejectsConnectionOutsideReady(t *testing.T) {
	res := createTestingGateway(t, testingParameters{silent: true})
	handshake(t, res)

	handler := awaitConnectionHandler(t, res)
	handler.beginDisconnect("test disconnect")

	resp, err := res.svc.Forward(context.Background(), &connectpb.ForwardRequest{
		ConnectionID: res.connID.String(),
		Data:         testGatewayExecutorRequestData(res, "test-forward-not-ready"),
	})
	require.NoError(t, err)
	require.False(t, resp.Success)
}

func TestGatewayExecutorRequestNotWrittenOutsideReady(t *testing.T) {
	res := createTestingGateway(t, testingParameters{silent: true})
	handshake(t, res)

	handler := awaitConnectionHandler(t, res)
	handler.beginDisconnect("test disconnect")

	resultCh := make(chan error, 1)
	handler.messageChan <- forwardMessage{
		Data:   testGatewayExecutorRequestData(res, "test-write-not-ready"),
		Result: resultCh,
	}

	select {
	case err := <-resultCh:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("forward should fail when connection is not Ready")
	}

	assertNoMessage(t, res.ws, 200*time.Millisecond)
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

func testGatewayExecutorRequestData(res testingResources, requestID string) *connectpb.GatewayExecutorRequestData {
	return &connectpb.GatewayExecutorRequestData{
		RequestId:      requestID,
		AccountId:      res.accountID.String(),
		EnvId:          res.envID.String(),
		AppId:          res.appID.String(),
		AppName:        res.appName,
		FunctionId:     res.fnID.String(),
		FunctionSlug:   res.fnSlug,
		RequestPayload: []byte("test payload"),
		RunId:          res.runID.String(),
	}
}
