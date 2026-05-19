package connect

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/coder/websocket"
)

type connPhase uint8

const (
	connPhaseNew connPhase = iota
	connPhaseHandshaking
	connPhaseActive
	connPhaseDraining
	connPhaseClosing
	connPhaseRetired
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
	mu          sync.Mutex
	phase       connPhase
	reason      string
	attrs       []any
	noWrites    chan struct{}
	logger      *slog.Logger
	flushNotify chan struct{}
}

func (c *connection) initLifecycle(logger *slog.Logger, flushNotify chan struct{}) {
	c.lifecycle.mu.Lock()
	defer c.lifecycle.mu.Unlock()

	c.lifecycle.logger = logger
	c.lifecycle.flushNotify = flushNotify
	c.lifecycle.ensureLocked()
}

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
