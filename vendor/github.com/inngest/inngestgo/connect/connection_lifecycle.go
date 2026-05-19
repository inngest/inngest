package connect

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/coder/websocket"
)

type connPhase uint8

const (
	// connPhaseNew is the zero-value phase for a connection object that exists
	// locally but has not started the gateway websocket handshake.
	connPhaseNew connPhase = iota
	// connPhaseHandshaking allows only the connect handshake protocol. Request
	// protocol writes are still forbidden in this phase.
	connPhaseHandshaking
	// connPhaseActive is the normal read-loop phase. Gateway requests may be
	// ACKed, heartbeats and lease extensions may be sent, and replies may use
	// the websocket.
	connPhaseActive
	// connPhaseDraining is gateway-initiated replacement. New request ACKs and
	// heartbeats stop, but already-ACKed work keeps its lease and may still try
	// to reply before the generation is retired.
	connPhaseDraining
	// connPhaseClosing is local worker shutdown. It permits the worker pause
	// message, already-ACKed lease extensions, and replies while the worker pool
	// drains.
	connPhaseClosing
	// connPhaseRetired is the hard no-write boundary. Queued pre-ACK work skips
	// and already-ACKed work buffers replies instead of using this websocket.
	connPhaseRetired
	// connPhaseClosed means the transport has been closed or best-effort closed.
	// It has the same write policy as Retired.
	connPhaseClosed
)

func (p connPhase) String() string {
	switch p {
	case connPhaseNew:
		return "New"
	case connPhaseHandshaking:
		return "Handshaking"
	case connPhaseActive:
		return "Active"
	case connPhaseDraining:
		return "Draining"
	case connPhaseClosing:
		return "Closing"
	case connPhaseRetired:
		return "Retired"
	case connPhaseClosed:
		return "Closed"
	default:
		return fmt.Sprintf("connPhase(%d)", p)
	}
}

type connLifecycle struct {
	mu     sync.Mutex
	phase  connPhase
	reason string
	attrs  []any
	// noWrites is closed exactly once when a generation reaches Retired or
	// Closed. Later phases can select on this channel instead of polling phase.
	noWrites chan struct{}
	logger   *slog.Logger
	// flushNotify is intentionally best-effort and non-blocking. Retiring a
	// generation tells the manager buffered replies may need API flush, but the
	// connection lifecycle never performs API I/O directly.
	flushNotify chan struct{}
}

func (c *connection) initLifecycle(logger *slog.Logger, flushNotify chan struct{}) {
	c.lifecycle.mu.Lock()
	defer c.lifecycle.mu.Unlock()

	c.lifecycle.logger = logger
	c.lifecycle.flushNotify = flushNotify
	c.lifecycle.ensureLocked()
}

// phase returns the observable lifecycle phase for this websocket generation.
// Manager state is intentionally separate and coarser-grained.
func (c *connection) phase() connPhase {
	c.lifecycle.mu.Lock()
	defer c.lifecycle.mu.Unlock()

	return c.lifecycle.phase
}

func (c *connection) markActive(reason string, attrs ...any) error {
	return c.transition(connPhaseActive, reason, attrs...)
}

func (c *connection) beginDrain(reason string, attrs ...any) error {
	return c.transition(connPhaseDraining, reason, attrs...)
}

func (c *connection) beginClose(reason string, attrs ...any) error {
	return c.transition(connPhaseClosing, reason, attrs...)
}

func (c *connection) retire(reason string, attrs ...any) bool {
	changed, err := c.transitionLocked(connPhaseRetired, reason, attrs...)
	return err == nil && changed
}

func (c *connection) closeNormal(reason string, attrs ...any) bool {
	changed, err := c.transitionLocked(connPhaseClosed, reason, attrs...)
	if err != nil || !changed {
		return false
	}
	if c.ws != nil {
		_ = c.ws.Close(websocket.StatusNormalClosure, reason)
	}
	return true
}

func (c *connection) closeNow(reason string, attrs ...any) bool {
	changed, err := c.transitionLocked(connPhaseClosed, reason, attrs...)
	if err != nil || !changed {
		return false
	}
	if c.ws != nil {
		_ = c.ws.CloseNow()
	}
	return true
}

func (c *connection) isRetired() bool {
	switch c.phase() {
	case connPhaseRetired, connPhaseClosed:
		return true
	default:
		return false
	}
}

// transition applies a validated phase change and its lifecycle side effects.
// All phase mutations go through this path so write permissions, no-write
// notification, manager flush notification, and transition logs stay coupled.
func (c *connection) transition(to connPhase, reason string, attrs ...any) error {
	_, err := c.transitionLocked(to, reason, attrs...)
	return err
}

func (c *connection) transitionLocked(to connPhase, reason string, attrs ...any) (bool, error) {
	c.lifecycle.mu.Lock()
	defer c.lifecycle.mu.Unlock()

	c.lifecycle.ensureLocked()
	from := c.lifecycle.phase
	if from == to {
		return false, nil
	}
	if !validConnPhaseTransition(from, to) {
		err := fmt.Errorf("invalid connection phase transition from %s to %s", from, to)
		if c.lifecycle.logger != nil {
			c.lifecycle.logger.Warn("invalid connection phase transition", append(c.logAttrs(), "from", from, "to", to, "reason", reason)...)
		}
		return false, err
	}

	c.lifecycle.phase = to
	c.lifecycle.reason = reason
	c.lifecycle.attrs = append(c.lifecycle.attrs[:0], attrs...)

	if entersNoWritePhase(from, to) {
		close(c.lifecycle.noWrites)
		c.lifecycle.noWrites = nil
		c.lifecycle.notifyFlushLocked()
	}

	if c.lifecycle.logger != nil {
		logAttrs := append(c.logAttrs(), "from", from, "to", to, "reason", reason)
		logAttrs = append(logAttrs, attrs...)
		c.lifecycle.logger.Debug("connection phase transition", logAttrs...)
	}

	return true, nil
}

func (l *connLifecycle) ensureLocked() {
	if l.noWrites == nil && l.phase != connPhaseRetired && l.phase != connPhaseClosed {
		l.noWrites = make(chan struct{})
	}
}

func (l *connLifecycle) notifyFlushLocked() {
	if l.flushNotify == nil {
		return
	}
	select {
	case l.flushNotify <- struct{}{}:
	default:
	}
}

func entersNoWritePhase(from, to connPhase) bool {
	switch to {
	case connPhaseRetired, connPhaseClosed:
		return from != connPhaseRetired && from != connPhaseClosed
	default:
		return false
	}
}

// validConnPhaseTransition describes only websocket-generation lifecycle, not
// public WorkerConnection manager state. Keeping these separate allows the
// manager to remain ACTIVE while an older generation is Draining or Retired.
func validConnPhaseTransition(from, to connPhase) bool {
	switch from {
	case connPhaseNew:
		return to == connPhaseHandshaking || to == connPhaseRetired || to == connPhaseClosed
	case connPhaseHandshaking:
		return to == connPhaseActive || to == connPhaseRetired || to == connPhaseClosed
	case connPhaseActive:
		return to == connPhaseDraining || to == connPhaseClosing || to == connPhaseRetired || to == connPhaseClosed
	case connPhaseDraining:
		return to == connPhaseRetired || to == connPhaseClosed
	case connPhaseClosing:
		return to == connPhaseRetired || to == connPhaseClosed
	case connPhaseRetired:
		return to == connPhaseClosed
	case connPhaseClosed:
		return false
	default:
		return false
	}
}
