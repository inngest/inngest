package connect

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

// transition records phase movement without owning external side effects yet.
// Phase 1 keeps behavior unchanged, so Redis writes, lifecycle callbacks, and
// transport closes still happen at their existing call sites.
func (c *connectionHandler) transition(to gatewayConnPhase, reason string, attrs ...any) gatewayConnPhase {
	from := gatewayConnPhase(c.connPhase.Swap(int32(to)))

	logAttrs := []any{
		"from_phase", from.String(),
		"to_phase", to.String(),
		"reason", reason,
	}
	logAttrs = append(logAttrs, attrs...)

	if from == to {
		c.log.Trace("worker connection phase refreshed", logAttrs...)
		return from
	}

	c.log.Debug("worker connection phase transition", logAttrs...)
	return from
}

func (c *connectionHandler) markHandshaking(reason string, attrs ...any) {
	c.transition(gatewayConnPhaseHandshaking, reason, attrs...)
}

func (c *connectionHandler) markReady(reason string, attrs ...any) {
	// Keep the legacy draining flag in sync until routing/write eligibility can
	// move fully to phase checks in a later plan phase.
	c.draining.Store(false)
	c.transition(gatewayConnPhaseReady, reason, attrs...)
}

func (c *connectionHandler) beginDrain(reason string, attrs ...any) {
	switch c.phase() {
	case gatewayConnPhaseDisconnecting, gatewayConnPhaseClosed:
		// Drain is a routing concern. Once disconnect cleanup has started, do not
		// move the observable phase backward or revive legacy drain behavior.
		c.log.Trace("worker connection phase not moved to draining after disconnect started",
			append([]any{"phase", c.phase().String(), "reason", reason}, attrs...)...)
		return
	default:
		// During Phase 1 the existing Forward and heartbeat paths still read
		// c.draining, so set it as part of entering the drain phase.
		c.draining.Store(true)
		c.transition(gatewayConnPhaseDraining, reason, attrs...)
	}
}

func (c *connectionHandler) beginDisconnect(reason string, attrs ...any) {
	if c.phase() == gatewayConnPhaseClosed {
		// Cleanup can be reached from several defers. Preserve idempotence and
		// keep Closed as the terminal observable phase.
		c.log.Trace("worker connection phase already closed",
			append([]any{"reason", reason}, attrs...)...)
		return
	}

	c.transition(gatewayConnPhaseDisconnecting, reason, attrs...)
}

func (c *connectionHandler) markClosed(reason string, attrs ...any) {
	c.transition(gatewayConnPhaseClosed, reason, attrs...)
}
