package memory

import (
	"sort"
	"sync"
	"sync/atomic"
)

// expiryShards is the number of independent slices in one bucket.  every
// lease granted in a second lands in the same bucket, so one lock there would
// serialize every acquire.
const expiryShards = 16

// expiryShard is one padded slice of seqs.
type expiryShard struct {
	mu   sync.Mutex
	seqs []uint64
	_    [32]byte
}

// expiryBucket holds the seqs of every lease that expires within one second.
// drained is set when the sweeper has taken the bucket out of the index.  a
// late add sees it under the shard lock and moves on to the next second, so
// nothing is lost.
type expiryBucket struct {
	shards  [expiryShards]expiryShard
	drained atomic.Bool
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
		sh := &b.shards[seq%expiryShards]
		sh.mu.Lock()
		if b.drained.Load() {
			sh.mu.Unlock()
			sec++
			continue
		}
		sh.seqs = append(sh.seqs, seq)
		sh.mu.Unlock()
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
		b.drained.Store(true)
		for i := range b.shards {
			sh := &b.shards[i]
			sh.mu.Lock()
			seqs := sh.seqs
			sh.seqs = nil
			sh.mu.Unlock()
			for _, seq := range seqs {
				fn(seq)
			}
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
	for i := range b.shards {
		sh := &b.shards[i]
		sh.mu.Lock()
		seqs := append([]uint64(nil), sh.seqs...)
		sh.mu.Unlock()
		for _, seq := range seqs {
			fn(seq)
		}
	}
}

// bucketCount is for tests.
func (e *expiryIndex) bucketCount() int {
	n := 0
	e.buckets.Range(func(any, any) bool { n++; return true })
	return n
}
