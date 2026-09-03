package memory

import (
	"sort"
	"sync"
	"sync/atomic"
)

// expiryBucket holds the seqs of every lease that expires within one second.
// drained is set under mu when the sweeper has taken the bucket out of the
// index, so a late add moves on to the next second instead of being lost.
type expiryBucket struct {
	mu      sync.Mutex
	seqs    []uint64
	drained bool
}

// expiryIndex finds expired leases without scanning the slab.  buckets are
// keyed by expiry second.  swept is the last second a drain covered.  add
// never targets a second at or below it, so a drained bucket is never
// refilled.
type expiryIndex struct {
	buckets sync.Map // int64 second -> *expiryBucket
	swept   atomic.Int64
}

func newExpiryIndex(nowMS int64) *expiryIndex {
	e := &expiryIndex{}
	e.swept.Store(nowMS/1000 - 1)
	return e
}

func (e *expiryIndex) bucket(sec int64) *expiryBucket {
	if v, ok := e.buckets.Load(sec); ok {
		return v.(*expiryBucket)
	}
	v, _ := e.buckets.LoadOrStore(sec, &expiryBucket{})
	return v.(*expiryBucket)
}

// add records that seq expires at expiresAtMS.  a seq whose second has
// already been drained lands in the first second the sweeper has not reached.
func (e *expiryIndex) add(expiresAtMS int64, seq uint64) {
	sec := expiresAtMS / 1000
	for {
		if first := e.swept.Load() + 1; sec < first {
			sec = first
		}
		b := e.bucket(sec)
		b.mu.Lock()
		if b.drained || sec <= e.swept.Load() {
			// the sweeper owns this second.  an empty bucket here was created
			// after the sweeper removed the real one, so drop it.
			if !b.drained && len(b.seqs) == 0 {
				e.buckets.CompareAndDelete(sec, b)
			}
			b.mu.Unlock()
			sec++
			continue
		}
		b.seqs = append(b.seqs, seq)
		b.mu.Unlock()
		return
	}
}

// advanceSwept moves swept forward to sec and never backwards, so two
// concurrent drains cannot reopen a second for add.
func (e *expiryIndex) advanceSwept(sec int64) {
	for {
		cur := e.swept.Load()
		if cur >= sec || e.swept.CompareAndSwap(cur, sec) {
			return
		}
	}
}

// drain removes every bucket from swept+1 through nowSec-1 and calls fn for
// each seq in them, oldest second first.  every lease in these buckets has
// expired.  it walks the buckets that exist, not the seconds in between, so a
// long idle gap costs nothing.  a bucket created for a passed second after the
// walk started is picked up by the next drain.
func (e *expiryIndex) drain(nowMS int64, fn func(seq uint64)) {
	last := nowMS/1000 - 1
	if e.swept.Load() >= last {
		return
	}

	var secs []int64
	e.buckets.Range(func(k, _ any) bool {
		if sec := k.(int64); sec <= last {
			secs = append(secs, sec)
		}
		return true
	})
	sort.Slice(secs, func(i, j int) bool { return secs[i] < secs[j] })

	for _, sec := range secs {
		e.advanceSwept(sec)
		v, ok := e.buckets.LoadAndDelete(sec)
		if !ok {
			continue
		}
		b := v.(*expiryBucket)
		b.mu.Lock()
		b.drained = true
		seqs := b.seqs
		b.seqs = nil
		b.mu.Unlock()
		for _, seq := range seqs {
			fn(seq)
		}
	}
	e.advanceSwept(last)
}

// scan calls fn for every seq in the bucket for the current second without
// removing anything.  the caller checks each slot's expiry itself.
func (e *expiryIndex) scan(nowMS int64, fn func(seq uint64)) {
	v, ok := e.buckets.Load(nowMS / 1000)
	if !ok {
		return
	}
	b := v.(*expiryBucket)
	b.mu.Lock()
	seqs := append([]uint64(nil), b.seqs...)
	b.mu.Unlock()
	for _, seq := range seqs {
		fn(seq)
	}
}

// bucketCount is for tests.
func (e *expiryIndex) bucketCount() int {
	n := 0
	e.buckets.Range(func(any, any) bool { n++; return true })
	return n
}
