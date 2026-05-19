package connect

import connectproto "github.com/inngest/inngest/proto/gen/connect/v1"

// These helpers centralize websocket write policy for one connection
// generation. Phase 2 keeps request behavior unchanged; it only makes each
// protocol write ask lifecycle state before attempting the websocket write.
func (c *connection) canWriteHeartbeat() bool {
	return c.canWrite(connectproto.GatewayMessageType_WORKER_HEARTBEAT)
}

func (c *connection) canWriteRequestAck() bool {
	return c.canWrite(connectproto.GatewayMessageType_WORKER_REQUEST_ACK)
}

func (c *connection) canWriteReply() bool {
	return c.canWrite(connectproto.GatewayMessageType_WORKER_REPLY)
}

func (c *connection) canWriteExtendLease() bool {
	return c.canWrite(connectproto.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE)
}

func (c *connection) canWritePause() bool {
	return c.canWrite(connectproto.GatewayMessageType_WORKER_PAUSE)
}

// canWrite maps lifecycle phase to allowed protocol writes. Handshake writes
// are intentionally not modeled here because they happen before a connection is
// active and still use the raw websocket during performConnectHandshake.
func (c *connection) canWrite(kind connectproto.GatewayMessageType) bool {
	switch c.phase() {
	case connPhaseActive:
		// Active is the only phase that can claim new work from the gateway,
		// keep the socket alive, and extend active request ownership.
		switch kind {
		case connectproto.GatewayMessageType_WORKER_HEARTBEAT,
			connectproto.GatewayMessageType_WORKER_REQUEST_ACK,
			connectproto.GatewayMessageType_WORKER_REPLY,
			connectproto.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE,
			connectproto.GatewayMessageType_WORKER_PAUSE:
			return true
		default:
			return false
		}
	case connPhaseDraining:
		// Gateway drain means this generation is being replaced. It must not
		// ACK new work, but already-ACKed work is still owned here. Keep
		// extending leases while that work finishes so it is not retried
		// elsewhere before the original worker replies.
		switch kind {
		case connectproto.GatewayMessageType_WORKER_REPLY,
			connectproto.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE:
			return true
		default:
			return false
		}
	case connPhaseClosing:
		// Local close tells the gateway to pause new work while allowing replies
		// and lease extensions for requests already owned by this worker.
		switch kind {
		case connectproto.GatewayMessageType_WORKER_REPLY,
			connectproto.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE,
			connectproto.GatewayMessageType_WORKER_PAUSE:
			return true
		default:
			return false
		}
	default:
		return false
	}
}
