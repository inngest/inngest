package connect

import (
	"context"
	"testing"
	"time"

	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// Give the gateway time to process the WORKER_PAUSE message
	time.Sleep(200 * time.Millisecond)

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
	conn, err := res.stateManager.GetConnection(ctx, res.envID, res.connID)
	require.NoError(t, err)
	require.NotNil(t, conn, "connection should still exist in Redis")
	require.Equal(t, connectpb.ConnectionStatus_DRAINING, conn.Status,
		"WORKER_PAUSE handler should update Redis status to DRAINING "+
			"even when gateway isDraining is true")

	// Verify the connection was removed from the in-memory wsConnections
	// map by the WORKER_PAUSE handler, preventing further message forwarding.
	_, loaded := res.svc.wsConnections.Load(res.connID.String())
	require.False(t, loaded,
		"WORKER_PAUSE handler should remove connection from wsConnections map")
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
// closes the connection. Heartbeats during drain are rejected with
// ErrDraining, causing the connection to close.
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
