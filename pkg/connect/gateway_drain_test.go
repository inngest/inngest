package connect

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/connect/types"
	"github.com/inngest/inngest/pkg/connect/wsproto"
	"github.com/inngest/inngest/pkg/consts"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

// TestWorkerPauseDuringGatewayDrain_UpdatesRedisStatus verifies that
// when a worker sends WORKER_PAUSE while the gateway is draining, the gateway
// does NOT reject the message with ErrDraining (which would close the
// connection abruptly). Instead, it should process the pause normally so
// the worker can finish in-progress work and shut down gracefully.
func TestWorkerPauseDuringGatewayDrain_UpdatesRedisStatus(t *testing.T) {
	ctx := context.Background()
	res := createTestingGateway(t, testingParameters{
		consecutiveMissesBeforeClose: 10,
		heartbeatInterval:            1 * time.Second,
		silent:                       true,
	})
	handshake(t, res)

	// Set isDraining on the gateway WITHOUT triggering drainListener.Notify().
	// This simulates the exact race condition: the gateway's isDraining flag
	// is set (e.g. by DrainGateway()), but we isolate the WORKER_PAUSE
	// handling from the gateway-initiated drain goroutine that would
	// independently set the connection status to DRAINING.
	res.svc.isDraining.Store(true)

	// Worker sends WORKER_PAUSE (e.g. graceful shutdown via SIGTERM).
	// With the bug, handleIncomingWebSocketMessage returns &ErrDraining,
	// which causes closeWithConnectError to send a StatusGoingAway close
	// frame. Without the bug, the pause is handled normally.
	sendWorkerPauseMessage(t, res.ws)

	// If the bug is fixed: the WORKER_PAUSE handler updates Redis to
	// DRAINING and removes the connection from wsConnections. The
	// connection-level draining flag (ch.draining) is set by the
	// WORKER_PAUSE handler specifically.
	//
	// If the bug is present: WORKER_PAUSE returns ErrDraining, the
	// connection is force-closed, and the status in Redis is NOT updated
	// by the WORKER_PAUSE handler (the gateway drain goroutine was not
	// triggered since we only set isDraining without calling Notify()).

	// Verify the connection status was set to DRAINING in Redis by the
	// WORKER_PAUSE handler. Without the fix, the status remains READY
	// because we only set isDraining (no drain goroutine was triggered).
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		conn, err := res.stateManager.GetConnection(ctx, res.envID, res.connID)
		assert.NoError(ct, err)
		if conn != nil {
			assert.Equal(ct, connectpb.ConnectionStatus_DRAINING, conn.Status,
				"WORKER_PAUSE handler should update Redis status to DRAINING "+
					"even when gateway isDraining is true")
		}
	}, 2*time.Second, 100*time.Millisecond)

	// Verify the connection was removed from the in-memory wsConnections
	// map by the WORKER_PAUSE handler, preventing further message forwarding.
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		_, loaded := res.svc.wsConnections.Load(res.connID.String())
		assert.False(ct, loaded,
			"WORKER_PAUSE handler should remove connection from wsConnections map")
	}, 2*time.Second, 100*time.Millisecond)
}

// TestWorkerPauseWhenGatewayNotDraining_ShouldWork is a control test verifying
// that WORKER_PAUSE works correctly when the gateway is NOT draining.
func TestWorkerPauseWhenGatewayNotDraining_ShouldWork(t *testing.T) {
	ctx := context.Background()
	res := createTestingGateway(t, testingParameters{
		consecutiveMissesBeforeClose: 10,
		heartbeatInterval:            1 * time.Second,
		silent:                       true,
	})
	handshake(t, res)

	// Verify connection is READY
	conn, err := res.stateManager.GetConnection(ctx, res.envID, res.connID)
	require.NoError(t, err)
	require.Equal(t, connectpb.ConnectionStatus_READY, conn.Status)

	// Worker sends WORKER_PAUSE without gateway draining (normal graceful shutdown)
	sendWorkerPauseMessage(t, res.ws)

	// Connection status should be updated to DRAINING in Redis
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		conn, err := res.stateManager.GetConnection(ctx, res.envID, res.connID)
		assert.NoError(ct, err)
		if conn != nil {
			assert.Equal(ct, connectpb.ConnectionStatus_DRAINING, conn.Status)
		}
	}, 2*time.Second, 100*time.Millisecond)

	// Connection should be removed from in-memory wsConnections map
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		_, loaded := res.svc.wsConnections.Load(res.connID.String())
		assert.False(ct, loaded, "connection should be removed from wsConnections map")
	}, 2*time.Second, 100*time.Millisecond)

	// Draining lifecycle should have been called
	res.lifecycles.Assert(t, testRecorderAssertion{
		onConnectedCount:     1,
		onSyncedCount:        1,
		onReadyCount:         1,
		onStartDrainingCount: 1,
	})
}

// TestHeartbeatDuringGatewayDrain_ClosesConnection verifies that when a
// gateway drain starts, the gateway sends GATEWAY_CLOSING and eventually
// force-closes the connection after the drain timeout if the worker
// doesn't close voluntarily.
func TestHeartbeatDuringGatewayDrain_ClosesConnection(t *testing.T) {
	res := createTestingGateway(t, testingParameters{
		consecutiveMissesBeforeClose: 10,
		heartbeatInterval:            1 * time.Second,
		silent:                       true,
	})
	handshake(t, res)

	// Start gateway drain
	err := res.svc.DrainGateway()
	require.NoError(t, err)

	// Gateway sends GATEWAY_CLOSING to the worker
	msg := awaitNextMessage(t, res.ws, 3*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_GATEWAY_CLOSING, msg.Kind)

	// The drain goroutine waits up to 5s for the worker to close, then
	// force-closes. Since the test client doesn't close the WS, we need
	// to wait past the 5s timeout for the full disconnect lifecycle.
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		res.lifecycles.lock.Lock()
		defer res.lifecycles.lock.Unlock()

		assert.Equal(ct, 1, len(res.lifecycles.onConnected))
		assert.Equal(ct, 1, len(res.lifecycles.onSynced))
		assert.Equal(ct, 1, len(res.lifecycles.onReady))
		assert.GreaterOrEqual(ct, len(res.lifecycles.onStartDraining), 1)
		assert.Equal(ct, 1, len(res.lifecycles.onDisconnected))
	}, 10*time.Second, 200*time.Millisecond)
}

// TestWorkerReadyDuringGatewayDrain_ClosesConnection verifies that when a
// gateway drain starts, the connection is eventually fully cleaned up.
func TestWorkerReadyDuringGatewayDrain_ClosesConnection(t *testing.T) {
	res := createTestingGateway(t, testingParameters{
		consecutiveMissesBeforeClose: 10,
		heartbeatInterval:            1 * time.Second,
		silent:                       true,
	})
	handshake(t, res)

	// Start gateway drain
	err := res.svc.DrainGateway()
	require.NoError(t, err)

	// Gateway sends GATEWAY_CLOSING to the worker
	msg := awaitNextMessage(t, res.ws, 3*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_GATEWAY_CLOSING, msg.Kind)

	// Worker closes the connection in response to GATEWAY_CLOSING
	err = res.ws.Close(1000, connectpb.WorkerDisconnectReason_WORKER_SHUTDOWN.String())
	require.NoError(t, err)

	// Connection should eventually be fully cleaned up.
	res.lifecycles.Assert(t, testRecorderAssertion{
		onConnectedCount:          1,
		onSyncedCount:             1,
		onReadyCount:              1,
		onStartDrainingCount:      1,
		onStartDisconnectingCount: 1,
		onDisconnectedCount:       1,
	})
}

// TestHeartbeatDuringGatewayDrain_StatusRemainsDraining verifies that an
// existing connection can continue sending heartbeats during drain, and the
// connection status stays DRAINING (not reset to READY).
func TestHeartbeatDuringGatewayDrain_StatusRemainsDraining(t *testing.T) {
	ctx := context.Background()
	res := createTestingGateway(t, testingParameters{
		consecutiveMissesBeforeClose: 10,
		heartbeatInterval:            1 * time.Second,
		drainAckTimeout:              500 * time.Millisecond,
		silent:                       true,
	})
	handshake(t, res)

	// Start gateway drain
	err := res.svc.DrainGateway()
	require.NoError(t, err)

	// Gateway sends GATEWAY_CLOSING to the worker
	msg := awaitNextMessage(t, res.ws, 3*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_GATEWAY_CLOSING, msg.Kind)

	// Worker sends a heartbeat during drain
	sendWorkerHeartbeatMessage(t, res.ws)

	// Expect GATEWAY_HEARTBEAT response (heartbeats are still processed)
	msg = awaitNextMessage(t, res.ws, 3*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_GATEWAY_HEARTBEAT, msg.Kind)

	// Verify connection status in Redis is DRAINING (not reset to READY)
	conn, err := res.stateManager.GetConnection(ctx, res.envID, res.connID)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, connectpb.ConnectionStatus_DRAINING, conn.Status,
		"heartbeat during drain should keep status as DRAINING, not reset to READY")

	// Verify lifecycle: heartbeat was recorded and draining started
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		res.lifecycles.lock.Lock()
		defer res.lifecycles.lock.Unlock()

		assert.Equal(ct, 1, len(res.lifecycles.onHeartbeat))
		assert.GreaterOrEqual(ct, len(res.lifecycles.onStartDraining), 1)
	}, 3*time.Second, 100*time.Millisecond)
}

// TestNoNewConnectionsDuringDrain verifies the gateway rejects new WebSocket
// connections when draining.
func TestNoNewConnectionsDuringDrain(t *testing.T) {
	res := createTestingGateway(t, testingParameters{
		consecutiveMissesBeforeClose: 10,
		heartbeatInterval:            1 * time.Second,
		noConnect:                    true,
		silent:                       true,
	})

	// Start draining before any connection
	err := res.svc.DrainGateway()
	require.NoError(t, err)

	// Try to connect — the gateway should reject the connection
	ws, _, dialErr := websocket.Dial(context.Background(), res.websocketUrl, &websocket.DialOptions{
		Subprotocols: []string{types.GatewaySubProtocol},
	})

	if dialErr != nil {
		// Connection rejected at dial — this is acceptable
		t.Logf("dial rejected: %v", dialErr)
	} else {
		// Connection accepted but should be closed immediately
		defer func() { _ = ws.CloseNow() }()

		// Try to read — expect an error (close frame or EOF)
		var parsed connectpb.ConnectMessage
		readErr := wsproto.Read(context.Background(), ws, &parsed)
		require.Error(t, readErr, "new connection during drain should be rejected")
		t.Logf("read error: %v", readErr)
	}

	// Verify no connections were established
	require.Equal(t, uint64(0), res.svc.connectionCount.Count(),
		"no connections should be accepted during drain")

	// Verify no lifecycle events fired for the rejected connection
	res.lifecycles.lock.Lock()
	connectedCount := len(res.lifecycles.onConnected)
	readyCount := len(res.lifecycles.onReady)
	res.lifecycles.lock.Unlock()

	require.Equal(t, 0, connectedCount, "no onConnected events should fire")
	require.Equal(t, 0, readyCount, "no onReady events should fire")
}

// TestWorkerReplyDuringGatewayDrain_IsProcessed verifies that worker replies
// are accepted, saved to Redis, and acknowledged during drain.
func TestWorkerReplyDuringGatewayDrain_IsProcessed(t *testing.T) {
	ctx := context.Background()
	res := createTestingGateway(t, testingParameters{
		consecutiveMissesBeforeClose: 10,
		heartbeatInterval:            1 * time.Second,
		silent:                       true,
	})
	handshake(t, res)

	// Start gateway drain
	err := res.svc.DrainGateway()
	require.NoError(t, err)

	// Gateway sends GATEWAY_CLOSING
	msg := awaitNextMessage(t, res.ws, 3*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_GATEWAY_CLOSING, msg.Kind)

	// Send WORKER_REPLY with an SDKResponse payload
	requestID := "test-drain-reply-req"
	runID := ulid.MustNew(ulid.Now(), rand.Reader)

	sdkResponse := &connectpb.SDKResponse{
		RequestId:      requestID,
		AccountId:      res.accountID.String(),
		EnvId:          res.envID.String(),
		AppId:          res.appID.String(),
		Status:         connectpb.SDKResponseStatus_DONE,
		Body:           []byte("drain reply body"),
		SdkVersion:     "test-version",
		RequestVersion: 1,
		RunId:          runID.String(),
	}

	responseBytes, err := proto.Marshal(sdkResponse)
	require.NoError(t, err)

	err = wsproto.Write(ctx, res.ws, &connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_REPLY,
		Payload: responseBytes,
	})
	require.NoError(t, err)

	// Expect WORKER_REPLY_ACK with matching requestId
	msg = awaitNextMessage(t, res.ws, 3*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_WORKER_REPLY_ACK, msg.Kind)

	ackData := &connectpb.WorkerReplyAckData{}
	err = proto.Unmarshal(msg.Payload, ackData)
	require.NoError(t, err)
	require.Equal(t, requestID, ackData.RequestId)

	// Verify response was saved to Redis
	savedResponse, err := res.stateManager.GetResponse(ctx, res.envID, requestID)
	require.NoError(t, err)
	require.NotNil(t, savedResponse)
	require.Equal(t, requestID, savedResponse.RequestId)
	require.Equal(t, connectpb.SDKResponseStatus_DONE, savedResponse.Status)
}

// mockExecutorServer implements ConnectExecutorServer for testing the ack flow.
type mockExecutorServer struct {
	connectpb.UnimplementedConnectExecutorServer
	ackReceived chan *connectpb.AckMessage
}

func (s *mockExecutorServer) Ack(_ context.Context, msg *connectpb.AckMessage) (*connectpb.AckResponse, error) {
	s.ackReceived <- msg
	return &connectpb.AckResponse{Success: true}, nil
}

func (s *mockExecutorServer) Ping(_ context.Context, _ *connectpb.PingRequest) (*connectpb.PingResponse, error) {
	return &connectpb.PingResponse{Message: "ok"}, nil
}

// TestWorkerAckDuringGatewayDrain_IsProcessed verifies that worker request
// acks are accepted and forwarded via gRPC during drain.
func TestWorkerAckDuringGatewayDrain_IsProcessed(t *testing.T) {
	ctx := context.Background()
	res := createTestingGateway(t, testingParameters{
		consecutiveMissesBeforeClose: 10,
		heartbeatInterval:            1 * time.Second,
		silent:                       true,
	})
	handshake(t, res)

	// Start a mock gRPC executor server on a free port
	mockServer := &mockExecutorServer{
		ackReceived: make(chan *connectpb.AckMessage, 1),
	}
	grpcServer := grpc.NewServer()
	connectpb.RegisterConnectExecutorServer(grpcServer, mockServer)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	executorPort := lis.Addr().(*net.TCPAddr).Port

	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.Stop)

	// Override the executor port so the gateway connects to our mock
	res.svc.grpcConfig.Executor.Port = executorPort

	// Verify mock gRPC server is reachable
	testConn, err := grpc.NewClient(
		fmt.Sprintf("127.0.0.1:%d", executorPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	testConn.Close()

	// Lease a request so GetExecutorIP returns 127.0.0.1
	requestID := "test-drain-ack-req"
	_, err = res.svc.stateManager.LeaseRequest(ctx, res.envID, requestID, 5*time.Second, testExecutorIP)
	require.NoError(t, err)

	// Forward a request to the worker via wsConnections channel
	expectedPayload := &connectpb.GatewayExecutorRequestData{
		RequestId:      requestID,
		AccountId:      res.accountID.String(),
		EnvId:          res.envID.String(),
		AppId:          res.appID.String(),
		AppName:        res.appName,
		FunctionId:     res.fnID.String(),
		FunctionSlug:   res.fnSlug,
		StepId:         ptr.String("step"),
		RequestPayload: []byte("ack test payload"),
		RunId:          res.runID.String(),
		LeaseId:        "test-lease",
	}

	messageChan, ok := res.svc.wsConnections.Load(res.connID.String())
	require.True(t, ok, "connection should be registered for gRPC delivery")

	go func() {
		messageChan.(*connectionHandler).messageChan <- forwardMessage{Data: expectedPayload, Result: make(chan error, 1)}
	}()

	// Worker receives GATEWAY_EXECUTOR_REQUEST
	msg := awaitNextMessage(t, res.ws, 3*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST, msg.Kind)

	// Start gateway drain
	err = res.svc.DrainGateway()
	require.NoError(t, err)

	// Gateway sends GATEWAY_CLOSING
	msg = awaitNextMessage(t, res.ws, 3*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_GATEWAY_CLOSING, msg.Kind)

	// Worker sends WORKER_REQUEST_ACK
	ackPayload, err := proto.Marshal(&connectpb.WorkerRequestAckData{
		RequestId: requestID,
		AccountId: res.accountID.String(),
		EnvId:     res.envID.String(),
		AppId:     res.appID.String(),
	})
	require.NoError(t, err)

	err = wsproto.Write(ctx, res.ws, &connectpb.ConnectMessage{
		Kind:    connectpb.GatewayMessageType_WORKER_REQUEST_ACK,
		Payload: ackPayload,
	})
	require.NoError(t, err)

	// Verify mock gRPC server received the ack
	select {
	case ackMsg := <-mockServer.ackReceived:
		require.Equal(t, requestID, ackMsg.RequestId,
			"mock executor should receive ack with correct request ID")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for ack to reach mock executor")
	}
}

// TestLeaseExtensionDuringGatewayDrain_IsProcessed verifies that lease
// extensions are accepted during drain so in-flight requests can continue
// processing without losing their lease.
func TestLeaseExtensionDuringGatewayDrain_IsProcessed(t *testing.T) {
	ctx := context.Background()
	res := createTestingGateway(t, testingParameters{
		consecutiveMissesBeforeClose: 10,
		heartbeatInterval:            1 * time.Second,
		silent:                       true,
	})
	handshake(t, res)

	requestID := "test-drain-lease-ext-req"

	// Lease a request
	leaseID, err := res.svc.stateManager.LeaseRequest(ctx, res.envID, requestID, 5*time.Second, testExecutorIP)
	require.NoError(t, err)

	// Forward a request to the worker
	expectedPayload := &connectpb.GatewayExecutorRequestData{
		RequestId:      requestID,
		AccountId:      res.accountID.String(),
		EnvId:          res.envID.String(),
		AppId:          res.appID.String(),
		AppName:        res.appName,
		FunctionId:     res.fnID.String(),
		FunctionSlug:   res.fnSlug,
		StepId:         ptr.String("step"),
		RequestPayload: []byte("lease ext test"),
		RunId:          res.runID.String(),
		LeaseId:        leaseID.String(),
	}

	messageChan, ok := res.svc.wsConnections.Load(res.connID.String())
	require.True(t, ok, "connection should be registered for gRPC delivery")

	go func() {
		messageChan.(*connectionHandler).messageChan <- forwardMessage{Data: expectedPayload, Result: make(chan error, 1)}
	}()

	// Worker receives GATEWAY_EXECUTOR_REQUEST
	msg := awaitNextMessage(t, res.ws, 3*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST, msg.Kind)

	payload := &connectpb.GatewayExecutorRequestData{}
	err = proto.Unmarshal(msg.Payload, payload)
	require.NoError(t, err)

	// Start gateway drain
	err = res.svc.DrainGateway()
	require.NoError(t, err)

	// Gateway sends GATEWAY_CLOSING
	msg = awaitNextMessage(t, res.ws, 3*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_GATEWAY_CLOSING, msg.Kind)

	// Worker sends lease extension during drain
	sendWorkerExtendLeaseMessage(t, res, &connectpb.WorkerRequestExtendLeaseData{
		RequestId:    payload.RequestId,
		AccountId:    payload.AccountId,
		EnvId:        payload.EnvId,
		AppId:        payload.AppId,
		FunctionSlug: payload.FunctionSlug,
		StepId:       payload.StepId,
		RunId:        payload.RunId,
		LeaseId:      payload.LeaseId,
	})

	// Expect lease extension ack (not a close/error)
	msg = awaitNextMessage(t, res.ws, 3*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE_ACK, msg.Kind)

	ackPayload := connectpb.WorkerRequestExtendLeaseAckData{}
	err = proto.Unmarshal(msg.Payload, &ackPayload)
	require.NoError(t, err)

	require.Equal(t, payload.RequestId, ackPayload.RequestId)
	require.Equal(t, payload.AccountId, ackPayload.AccountId)
	require.NotNil(t, ackPayload.NewLeaseId,
		"lease extension during drain should succeed and return a new lease ID")

	// Verify the new lease ID is valid and has a reasonable expiry
	parsed, err := ulid.Parse(*ackPayload.NewLeaseId)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(consts.ConnectWorkerRequestLeaseDuration), ulid.Time(parsed.Time()), 2*time.Second,
		"new lease should have a future expiry")

	// Verify connection is still alive by exchanging a heartbeat
	sendWorkerHeartbeatMessage(t, res.ws)
	msg = awaitNextMessage(t, res.ws, 3*time.Second)
	require.Equal(t, connectpb.GatewayMessageType_GATEWAY_HEARTBEAT, msg.Kind,
		"connection should still be alive after lease extension during drain")
}
