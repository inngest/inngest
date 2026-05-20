package connect

func (h *connectHandler) setState(state ConnectionState, reason string, attrs ...any) {
	previous, valid := h.setStateLocked(state)

	logAttrs := []any{
		"from", previous,
		"to", state,
		"reason", reason,
	}
	logAttrs = append(logAttrs, attrs...)

	if !valid {
		h.logger.Warn("invalid worker connection state transition", logAttrs...)
		return
	}

	h.logger.Debug("worker connection state transition", logAttrs...)
}

func (h *connectHandler) setStateLocked(state ConnectionState) (ConnectionState, bool) {
	h.stateLock.Lock()
	defer h.stateLock.Unlock()

	previous := h.state
	if previous != state && !validConnectionStateTransition(previous, state) {
		// Manager state is user-visible, so invalid transitions are logged and
		// ignored instead of silently reshaping what State() reports.
		return previous, false
	}

	h.state = state
	return previous, true
}

// validConnectionStateTransition validates only the user-visible manager
// lifecycle. Websocket generation states such as Draining and Retired are
// intentionally modeled separately in connection_lifecycle.go.
func validConnectionStateTransition(from, to ConnectionState) bool {
	switch from {
	case ConnectionStateConnecting:
		// Initial startup can establish a worker, be closed by the caller, or
		// fail permanently before ever becoming active.
		return to == ConnectionStateActive ||
			to == ConnectionStateClosing ||
			to == ConnectionStateClosed
	case ConnectionStateActive:
		// Active covers both the current websocket generation and gateway-drain
		// overlap. Draining older generations must not change manager state.
		return to == ConnectionStateReconnecting ||
			to == ConnectionStateClosing ||
			to == ConnectionStateClosed
	case ConnectionStateReconnecting:
		// Reconnect can succeed, keep retrying, be closed by the caller, or end
		// permanently after max attempts or a non-reconnectable failure.
		return to == ConnectionStateActive ||
			to == ConnectionStateReconnecting ||
			to == ConnectionStateClosing ||
			to == ConnectionStateClosed
	case ConnectionStateClosing:
		// Close is terminal from the public manager point of view.
		return to == ConnectionStateClosed
	case ConnectionStateClosed:
		return false
	default:
		return false
	}
}
