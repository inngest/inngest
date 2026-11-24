package cluster

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"
)

// ErrUnavailable is returned when a node cannot reach the coordinator.
var ErrUnavailable = errors.New("cluster: coordinator unavailable")

// Coordinator is one node's handle to the shard table.
type Coordinator interface {
	// Table returns the newest table this node has and a channel that is
	// closed when a newer one arrives.  the caller must not change the
	// table.
	Table() (*Table, <-chan struct{})
	// Propose applies one command.  an error that wraps ErrRejected means
	// the command did not fit the table.
	Propose(ctx context.Context, cmd Command) error
	// Report says this node is alive and how far each replica has got.  an
	// error that wraps ErrRemoved means the table replaced the node.
	Report(ctx context.Context, r Report) error
}

// MemCoordinator keeps one table in process memory and runs the controller
// loop on a goroutine.  tests and benchmarks use it.
type MemCoordinator struct {
	interval time.Duration
	timeout  time.Duration

	mu sync.Mutex
	// table is replaced, never changed, so a caller of Table can read its
	// copy with no lock.
	table   *Table
	changed chan struct{}
	reports map[string]memReport
	cut     map[string]*memCut
	delay   time.Duration
	// graceUntil is when the controller starts to treat a node with no
	// fresh report as dead.
	graceUntil time.Time
	stop       chan struct{}
	done       chan struct{}
}

type memReport struct {
	at     time.Time
	bootID string
	lsn    map[int]uint64
}

// memCut is the table a node had when it was cut off.
type memCut struct {
	table *Table
	ch    chan struct{}
}

// NewMemCoordinator returns a coordinator with an empty table and a running
// controller.  a node is live while its last report is younger than
// heartbeatTimeout.  the controller runs every half heartbeatInterval.
func NewMemCoordinator(heartbeatInterval, heartbeatTimeout time.Duration) *MemCoordinator {
	c := &MemCoordinator{
		interval: heartbeatInterval,
		timeout:  heartbeatTimeout,
		table:    &Table{},
		changed:  make(chan struct{}),
		reports:  map[string]memReport{},
		cut:      map[string]*memCut{},
	}
	c.StartController()
	return c
}

// Node returns the handle of one node.
func (c *MemCoordinator) Node(id string) Coordinator {
	return &memHandle{
		c:  c,
		id: id,
	}
}

// Cut makes the node's Propose and Report return ErrUnavailable and stops
// its Table from advancing.
func (c *MemCoordinator) Cut(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.cut[id]; ok {
		return
	}
	c.cut[id] = &memCut{
		table: c.table,
		ch:    make(chan struct{}),
	}
}

// Restore undoes Cut.  it closes the channel the node got from Table while
// it was cut, also when the table did not change in that time.
func (c *MemCoordinator) Restore(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cut, ok := c.cut[id]; ok {
		close(cut.ch)
		delete(c.cut, id)
	}
}

// SetProposeDelay makes every Propose from a node wait d before it applies.
// commands from the controller do not wait.
func (c *MemCoordinator) SetProposeDelay(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.delay = d
}

// StartController starts the controller loop if it is not running.  for one
// heartbeat timeout after the start it treats no node as dead, so nodes have
// time to report.
func (c *MemCoordinator) StartController() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stop != nil {
		return
	}
	c.graceUntil = time.Now().Add(c.timeout)
	c.stop = make(chan struct{})
	c.done = make(chan struct{})
	go c.control(c.stop, c.done)
}

// StopController stops the controller loop.  no primary is elected until
// StartController.
func (c *MemCoordinator) StopController() {
	c.mu.Lock()
	stop, done := c.stop, c.done
	c.stop, c.done = nil, nil
	c.mu.Unlock()
	if stop == nil {
		return
	}
	close(stop)
	<-done
}

// Close stops the controller goroutine.
func (c *MemCoordinator) Close() error {
	c.StopController()
	return nil
}

func (c *MemCoordinator) control(stop, done chan struct{}) {
	defer close(done)
	tick := time.NewTicker(max(c.interval/2, time.Microsecond))
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			c.plan(time.Now())
		}
	}
}

// plan runs the controller once.
func (c *MemCoordinator) plan(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if now.Before(c.graceUntil) {
		return
	}
	live := func(node string) bool {
		r, ok := c.reports[node]
		return ok && r.bootID == c.table.Nodes[node].BootID && now.Sub(r.at) <= c.timeout
	}
	lastLSN := func(node string, shard int) uint64 {
		return c.reports[node].lsn[shard]
	}
	for _, cmd := range Plan(c.table, live, lastLSN) {
		_ = c.apply(cmd)
	}
}

// apply runs a command on a copy of the table and publishes the copy.  the
// caller has the mutex.
func (c *MemCoordinator) apply(cmd Command) error {
	next := c.table.Clone()
	if err := next.Apply(cmd); err != nil {
		return err
	}
	c.table = next
	close(c.changed)
	c.changed = make(chan struct{})
	return nil
}

type memHandle struct {
	c  *MemCoordinator
	id string
}

func (h *memHandle) Table() (*Table, <-chan struct{}) {
	c := h.c
	c.mu.Lock()
	defer c.mu.Unlock()
	if cut, ok := c.cut[h.id]; ok {
		return cut.table, cut.ch
	}
	return c.table, c.changed
}

func (h *memHandle) Propose(ctx context.Context, cmd Command) error {
	c := h.c
	c.mu.Lock()
	_, cut := c.cut[h.id]
	delay := c.delay
	c.mu.Unlock()
	if cut {
		return ErrUnavailable
	}
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if _, cut := c.cut[h.id]; cut {
		return ErrUnavailable
	}
	return c.apply(cmd)
}

func (h *memHandle) Report(ctx context.Context, r Report) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c := h.c
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, cut := c.cut[h.id]; cut {
		return ErrUnavailable
	}
	if by, ok := c.table.Removed[r.Node]; ok {
		return fmt.Errorf("%w by node %q", ErrRemoved, by)
	}
	// a report from a boot ID the table does not have comes from a process
	// that has not registered or that was replaced.  it must not keep the
	// node live.
	if info, ok := c.table.Nodes[r.Node]; !ok || info.BootID != r.BootID {
		return nil
	}
	c.reports[r.Node] = memReport{
		at:     time.Now(),
		bootID: r.BootID,
		lsn:    maps.Clone(r.LSN),
	}
	return nil
}
