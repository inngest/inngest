package telemetry

import (
	"sync"
	"sync/atomic"
)

// Ring is a bounded drop-oldest ring buffer. It exists on the worker side so
// that a slow harness consumer cannot block the SDK hot path.
type Ring struct {
	mu      sync.Mutex
	buf     []Frame
	head    int
	size    int
	cap     int
	dropped uint64
	cond    *sync.Cond
	closed  bool
}

// NewRing returns a ring that holds up to capacity frames.
func NewRing(capacity int) *Ring {
	r := &Ring{buf: make([]Frame, capacity), cap: capacity}
	r.cond = sync.NewCond(&r.mu)
	return r
}

// Push appends f. If the ring is full, the oldest frame is evicted and the
// dropped counter is incremented.
func (r *Ring) Push(f Frame) {
	r.mu.Lock()
	if r.size == r.cap {
		r.head = (r.head + 1) % r.cap
		r.size--
		atomic.AddUint64(&r.dropped, 1)
	}
	tail := (r.head + r.size) % r.cap
	r.buf[tail] = f
	r.size++
	r.cond.Signal()
	r.mu.Unlock()
}

// Pop blocks until at least one frame is available or the ring is closed.
// It returns the oldest frame and ok=true, or a zero frame and ok=false once
// closed and drained.
func (r *Ring) Pop() (Frame, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for r.size == 0 && !r.closed {
		r.cond.Wait()
	}
	if r.size == 0 {
		return Frame{}, false
	}
	f := r.buf[r.head]
	r.head = (r.head + 1) % r.cap
	r.size--
	return f, true
}

// Close wakes any pending Pop callers; subsequent pops drain remaining frames
// and then return ok=false.
func (r *Ring) Close() {
	r.mu.Lock()
	r.closed = true
	r.cond.Broadcast()
	r.mu.Unlock()
}

// Dropped returns the current dropped-frame counter.
func (r *Ring) Dropped() uint64 { return atomic.LoadUint64(&r.dropped) }
