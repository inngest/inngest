package connect

import (
	connecterrors "github.com/inngest/inngest/pkg/connect/errors"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

func (c *connectionHandler) handleIncomingWebSocketMessage(msg *connectpb.ConnectMessage) *connecterrors.SocketError {
	c.log.Trace("received WebSocket message", "kind", msg.Kind.String())

	switch msg.Kind {
	case connectpb.GatewayMessageType_WORKER_READY:
		return c.handleWorkerReady()
	case connectpb.GatewayMessageType_WORKER_HEARTBEAT:
		return c.handleWorkerHeartbeat()
	case connectpb.GatewayMessageType_WORKER_STATUS:
		return c.handleWorkerStatus(msg)
	case connectpb.GatewayMessageType_WORKER_PAUSE:
		return c.handleWorkerPause()
	case connectpb.GatewayMessageType_WORKER_REQUEST_ACK:
		return c.handleWorkerRequestAck(msg)
	case connectpb.GatewayMessageType_WORKER_REPLY:
		return c.handleWorkerReply(msg)
	case connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE:
		return c.handleWorkerRequestExtendLease(msg)
	default:
		c.log.Warn("unexpected message kind received", "kind", msg.Kind.String())
		return nil
	}
}
