package memory

import (
	"math"
	"sync/atomic"
)

// deadCell marks a counter that housekeeping removed from its map at zero.
// any operation that sees it must look the key up again.  a live counter is
// never negative, so any value below deadThreshold is the sentinel plus a
// small transient add.
const (
	deadCell      int64 = math.MinInt64 / 2
	deadThreshold int64 = math.MinInt64 / 4
)

func isDead(v int64) bool {
	return v < deadThreshold
}

// semaphoreCell is one usage or capacity counter.  concurrency in progress
// counts, semaphore usage and semaphore capacity all use it.
type semaphoreCell struct {
	v atomic.Int64
}

// load returns the counter.  ok is false when the cell is dead.
func (c *semaphoreCell) load() (v int64, ok bool) {
	v = c.v.Load()
	if isDead(v) {
		return 0, false
	}
	return v, true
}

// take adds w*q and returns how many of q fit under capacity, rolling the
// rest back exactly, and the counter value once this take is accounted.  the
// counter overshoots by the rolled back amount for a few nanoseconds, which
// can make a concurrent take fit less, never more.  ok is false when the
// cell is dead and nothing was added.
func (c *semaphoreCell) take(capacity, w int64, q int) (fit int, after int64, ok bool) {
	if q <= 0 {
		v := c.v.Load()
		return 0, v, !isDead(v)
	}
	added := w * int64(q)
	after = c.v.Add(added)
	before := after - added
	if isDead(before) {
		c.v.Add(-added)
		return 0, 0, false
	}
	if avail := capacity - before; avail >= w {
		fit = int(avail / w)
		if fit > q {
			fit = q
		}
	}
	if fit < q {
		c.v.Add(-w * int64(q-fit))
		after = before + w*int64(fit)
	}
	return fit, after, true
}

// give subtracts w and clamps at zero.  a CAS loop, not Add and fix up,
// because two concurrent double releases fixing up would leave the counter
// at +1.  ok is false when the cell is dead and nothing changed.
func (c *semaphoreCell) give(w int64) (v int64, ok bool) {
	for {
		old := c.v.Load()
		if isDead(old) {
			return 0, false
		}
		next := old - w
		if next < 0 {
			next = 0
		}
		if c.v.CompareAndSwap(old, next) {
			return next, true
		}
	}
}

// set stores v.  ok is false when the cell is dead and nothing changed.
func (c *semaphoreCell) set(v int64) (ok bool) {
	for {
		old := c.v.Load()
		if isDead(old) {
			return false
		}
		if c.v.CompareAndSwap(old, v) {
			return true
		}
	}
}

// adjust adds delta and clamps at zero, returning the new value.  ok is false
// when the cell is dead and nothing changed.
func (c *semaphoreCell) adjust(delta int64) (v int64, ok bool) {
	for {
		old := c.v.Load()
		if isDead(old) {
			return 0, false
		}
		next := old + delta
		if next < 0 {
			next = 0
		}
		if c.v.CompareAndSwap(old, next) {
			return next, true
		}
	}
}

// kill turns a zero counter into the dead sentinel so it can be dropped from
// its map.  returns false when the counter is not zero.
func (c *semaphoreCell) kill() bool {
	return c.v.CompareAndSwap(0, deadCell)
}

// gcraState is one immutable rate limit or throttle state.  the cell swaps
// whole states so a reader never sees a TAT with another write's expiry.
type gcraState struct {
	tat       float64
	expiresAt int64
}

// deadGCRA marks a cell housekeeping dropped from its map.
var deadGCRA = &gcraState{}

// gcraCell is one rate limit or throttle state.  it reads as absent when
// expiresAt <= now, the way an expired Redis key does.
type gcraCell struct {
	state atomic.Pointer[gcraState]
}

// load returns the stored TAT and whether it is present at nowMS.  a dead
// cell reads as absent.
func (c *gcraCell) load(nowMS int64) (tat float64, present bool) {
	st := c.state.Load()
	if st == nil || st == deadGCRA || st.expiresAt <= nowMS {
		return 0, false
	}
	return st.tat, true
}

// alive is false once housekeeping dropped the cell.
func (c *gcraCell) alive() bool {
	return c.state.Load() != deadGCRA
}

// update runs f on the current TAT and stores the result with a CAS.  f is
// pure and may run more than once under contention.  when f returns store
// false nothing is written and update returns after one run.  ok is false
// when the cell is dead and nothing was written.
func (c *gcraCell) update(nowMS int64, f func(tat float64, present bool) (newTAT float64, store bool, ttlSec int64)) (ok bool) {
	for {
		old := c.state.Load()
		if old == deadGCRA {
			return false
		}
		var tat float64
		present := old != nil && old.expiresAt > nowMS
		if present {
			tat = old.tat
		}
		newTAT, store, ttlSec := f(tat, present)
		if !store {
			return true
		}
		next := &gcraState{tat: newTAT, expiresAt: nowMS + ttlSec*1000}
		if c.state.CompareAndSwap(old, next) {
			return true
		}
	}
}

// kill marks an expired or empty cell dead so it can be dropped from its
// map.  returns false when the cell still holds a live state.
func (c *gcraCell) kill(nowMS int64) bool {
	old := c.state.Load()
	if old == deadGCRA {
		return false
	}
	if old != nil && old.expiresAt > nowMS {
		return false
	}
	return c.state.CompareAndSwap(old, deadGCRA)
}
