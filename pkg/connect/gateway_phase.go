package connect

import (
	"fmt"

	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

// gatewayConnPhase tracks the in-memory lifecycle of a single websocket
// handler. It is intentionally separate from Redis connection status: Redis is
// router-visible state, while this phase answers what the local handler may do.
type gatewayConnPhase int32

const (
	gatewayConnPhaseNew gatewayConnPhase = iota
	gatewayConnPhaseHandshaking
	gatewayConnPhaseReady
	gatewayConnPhaseDraining
	gatewayConnPhaseDisconnecting
	gatewayConnPhaseClosed
)

func (p gatewayConnPhase) String() string {
	switch p {
	case gatewayConnPhaseNew:
		return "New"
	case gatewayConnPhaseHandshaking:
		return "Handshaking"
	case gatewayConnPhaseReady:
		return "Ready"
	case gatewayConnPhaseDraining:
		return "Draining"
	case gatewayConnPhaseDisconnecting:
		return "Disconnecting"
	case gatewayConnPhaseClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

func (c *connectionHandler) phase() gatewayConnPhase {
	return gatewayConnPhase(c.connPhase.Load())
}

func (c *connectionHandler) canForward() bool {
	return c.phase() == gatewayConnPhaseReady
}

func (c *connectionHandler) canWrite(kind connectpb.GatewayMessageType) bool {
	phase := c.phase()

	switch kind {
	case connectpb.GatewayMessageType_GATEWAY_EXECUTOR_REQUEST:
		return phase == gatewayConnPhaseReady
	case connectpb.GatewayMessageType_GATEWAY_HEARTBEAT:
		return phase == gatewayConnPhaseReady ||
			phase == gatewayConnPhaseDraining
	case connectpb.GatewayMessageType_WORKER_REPLY_ACK:
		// Replies are only handled after the run loop has read a worker message.
		// Allow the ACK while cleanup is starting, but never after final close.
		return phase == gatewayConnPhaseReady ||
			phase == gatewayConnPhaseDraining ||
			phase == gatewayConnPhaseDisconnecting
	case connectpb.GatewayMessageType_WORKER_REQUEST_EXTEND_LEASE_ACK:
		return phase == gatewayConnPhaseReady ||
			phase == gatewayConnPhaseDraining
	case connectpb.GatewayMessageType_GATEWAY_CLOSING:
		return phase == gatewayConnPhaseReady ||
			phase == gatewayConnPhaseDraining
	default:
		return phase != gatewayConnPhaseClosed
	}
}

// transition records phase movement. Most external side effects still live at
// existing call sites so phase checks can be introduced without collapsing the
// Redis, lifecycle listener, and transport-close responsibilities together.
func (c *connectionHandler) transition(to gatewayConnPhase, reason string, attrs ...any) (gatewayConnPhase, bool) {
	for {
		from := c.phase()
		logAttrs := []any{
			"from_phase", from.String(),
			"to_phase", to.String(),
			"reason", reason,
		}
		logAttrs = append(logAttrs, attrs...)

		if from > to {
			c.log.Trace("worker connection phase transition ignored", logAttrs...)
			return from, false
		}

		if c.connPhase.CompareAndSwap(int32(from), int32(to)) {
			if from == to {
				c.log.Trace("worker connection phase refreshed", logAttrs...)
				return from, true
			}

			c.log.Debug("worker connection phase transition", logAttrs...)
			return from, true
		}
	}
}

func (c *connectionHandler) markHandshaking(reason string, attrs ...any) {
	c.transition(gatewayConnPhaseHandshaking, reason, attrs...)
}

func (c *connectionHandler) markReady(reason string, attrs ...any) {
	// Keep the legacy draining flag in sync while it remains part of the
	// handler state observed by tests and drain cleanup.
	if _, ok := c.transition(gatewayConnPhaseReady, reason, attrs...); ok {
		c.draining.Store(false)
	}
}

func (c *connectionHandler) beginDrain(reason string, attrs ...any) {
	from, ok := c.transition(gatewayConnPhaseDraining, reason, attrs...)
	if !ok {
		return
	}

	// Keep c.draining synchronized with the phase for compatibility with
	// existing drain state observers.
	c.draining.Store(true)
	if from == gatewayConnPhaseReady {
		c.releasePendingAcks(fmt.Errorf("connection entered drain: %s", reason))
	}
}

func (c *connectionHandler) beginDisconnect(reason string, attrs ...any) {
	from, ok := c.transition(gatewayConnPhaseDisconnecting, reason, attrs...)
	if !ok {
		return
	}

	if from == gatewayConnPhaseReady {
		c.releasePendingAcks(fmt.Errorf("connection started disconnecting: %s", reason))
	}
}

func (c *connectionHandler) markClosed(reason string, attrs ...any) {
	from, ok := c.transition(gatewayConnPhaseClosed, reason, attrs...)
	if !ok {
		return
	}

	if from == gatewayConnPhaseReady {
		c.releasePendingAcks(fmt.Errorf("connection closed: %s", reason))
	}
}
