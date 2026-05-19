package connect

func (h *connectHandler) flushBufferedMessages(reason string) {
	// Flushes are manager-owned because they need the current auth context.
	// Websocket generations only send a best-effort notification when their
	// lifecycle reaches a no-write phase.
	if h.messageBuffer == nil || !h.messageBuffer.hasMessages() {
		return
	}

	if err := h.messageBuffer.flush(h.auth.hashedSigningKey); err != nil {
		h.logger.Error("could not send buffered messages", "err", err, "reason", reason)
	}
}
