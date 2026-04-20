package telemetry

import (
	"context"
	"net"
	"sync/atomic"
	"time"
)

// Client is the worker-side telemetry sender. It owns a Ring and a goroutine
// that drains the ring onto a unix-socket connection.
type Client struct {
	workerID string
	ring     *Ring
	seq      uint64
	conn     net.Conn
}

// Dial connects to the harness unix socket and starts the drain goroutine.
func Dial(ctx context.Context, socketPath, workerID string, ringCap int) (*Client, error) {
	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return nil, err
	}
	c := &Client{workerID: workerID, ring: NewRing(ringCap), conn: conn}
	go c.drain()
	return c, nil
}

func (c *Client) drain() {
	for {
		f, ok := c.ring.Pop()
		if !ok {
			_ = c.conn.Close()
			return
		}
		if err := WriteFrame(c.conn, f); err != nil {
			// Drop silently — the harness consumer is gone, no point accumulating.
			_ = c.conn.Close()
			c.ring.Close()
			return
		}
	}
}

// Emit enqueues a frame. It never blocks the caller.
func (c *Client) Emit(phase Phase, runID, fnSlug, stepID string, attempt int) {
	c.EmitWithCorr(phase, runID, "", fnSlug, stepID, attempt)
}

// EmitWithCorr is like Emit but also sets a correlation ID on the frame. The
// correlation ID is how the harness joins fired events to the run that
// actually handled them.
func (c *Client) EmitWithCorr(phase Phase, runID, corr, fnSlug, stepID string, attempt int) {
	seq := atomic.AddUint64(&c.seq, 1)
	c.ring.Push(Frame{
		WorkerID:      c.workerID,
		Seq:           seq,
		InngestRunID:  runID,
		CorrelationID: corr,
		FunctionSlug:  fnSlug,
		StepID:        stepID,
		Attempt:       attempt,
		Phase:         phase,
		TSNanos:       time.Now().UnixNano(),
	})
}

// Dropped returns the number of frames that have been dropped by the ring.
func (c *Client) Dropped() uint64 { return c.ring.Dropped() }

// Close drains and closes the underlying connection.
func (c *Client) Close() { c.ring.Close() }
