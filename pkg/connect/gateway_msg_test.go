package connect

import (
	"testing"

	"github.com/inngest/inngest/pkg/syscode"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
)

func TestHandleIncomingWebSocketMessageUnknownKindIsIgnored(t *testing.T) {
	res := createTestingGateway(t)
	handshake(t, res)

	ch := newTestConnectionHandler(t, res)

	serr := ch.handleIncomingWebSocketMessage(&connectpb.ConnectMessage{})
	require.Nil(t, serr)
}

func TestHandleIncomingWebSocketMessageDispatchesHandlerOutcomes(t *testing.T) {
	t.Run("worker ready rejects gateway drain", func(t *testing.T) {
		res := createTestingGateway(t)
		handshake(t, res)

		ch := newTestConnectionHandler(t, res)
		res.svc.isDraining.Store(true)

		serr := ch.handleIncomingWebSocketMessage(&connectpb.ConnectMessage{
			Kind: connectpb.GatewayMessageType_WORKER_READY,
		})
		require.NotNil(t, serr)
		require.Equal(t, ErrDraining.SysCode, serr.SysCode)
	})

	t.Run("worker pause accepts gateway drain", func(t *testing.T) {
		res := createTestingGateway(t)
		handshake(t, res)

		ch := newTestConnectionHandler(t, res)
		res.svc.wsConnections.Store(res.connID.String(), ch)
		res.svc.isDraining.Store(true)

		serr := ch.handleIncomingWebSocketMessage(&connectpb.ConnectMessage{
			Kind: connectpb.GatewayMessageType_WORKER_PAUSE,
		})
		require.Nil(t, serr)
		require.True(t, ch.draining.Load())
	})

	t.Run("worker request ack invalid payload is fatal", func(t *testing.T) {
		res := createTestingGateway(t)
		handshake(t, res)

		ch := newTestConnectionHandler(t, res)

		serr := ch.handleIncomingWebSocketMessage(&connectpb.ConnectMessage{
			Kind:    connectpb.GatewayMessageType_WORKER_REQUEST_ACK,
			Payload: []byte("invalid protobuf"),
		})
		require.NotNil(t, serr)
		require.Equal(t, syscode.CodeConnectWorkerRequestAckInvalidPayload, serr.SysCode)
	})

	t.Run("worker request extend lease invalid payload is fatal", func(t *testing.T) {
		res := createTestingGateway(t)
		handshake(t, res)

		ch := newTestConnectionHandler(t, res)

		serr := ch.handleIncomingWebSocketMessage(&connectpb.ConnectMessage{
			Kind:    connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE,
			Payload: []byte("invalid protobuf"),
		})
		require.NotNil(t, serr)
		require.Equal(t, syscode.CodeConnectWorkerRequestExtendLeaseInvalidPayload, serr.SysCode)
	})

	t.Run("worker reply invalid payload is fatal", func(t *testing.T) {
		res := createTestingGateway(t)
		handshake(t, res)

		ch := newTestConnectionHandler(t, res)

		serr := ch.handleIncomingWebSocketMessage(&connectpb.ConnectMessage{
			Kind:    connectpb.GatewayMessageType_WORKER_REPLY,
			Payload: []byte("invalid protobuf"),
		})
		require.NotNil(t, serr)
		require.Equal(t, syscode.CodeConnectInternal, serr.SysCode)
	})
}
