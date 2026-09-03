package memory

import (
	"sync"
	"sync/atomic"
)

const (
	// pageBits sizes a page at 8192 slots, about 200 KB.  a page is freed once
	// every slot in it is taken, so smaller pages return memory sooner.
	pageBits = 13
	pageSize = 1 << pageBits

	slotEmpty uint32 = 0
	slotLive  uint32 = 1
	slotTaken uint32 = 2
)

// slot is one lease record.  it is written once by alloc and taken once by
// the release, extend or sweep that wins the CAS.  seqs never repeat, so a
// slot never goes back to live.  the owner clears req when it is done so the
// request state is collected as soon as its last lease is gone, instead of
// living as long as the page.
type slot struct {
	state       atomic.Uint32
	expiresAtMS int64
	req         atomic.Pointer[requestState]
}

// take moves the slot from live to taken.  the winner owns the record and is
// the only caller allowed to give back the counters it holds.
func (s *slot) take() bool {
	return s.state.CompareAndSwap(slotLive, slotTaken)
}

// page holds pageSize slots.  live counts slots in state live so the sweeper
// can free a page nothing points at.  emptySinceMS is when housekeeping first
// saw the page with no live slot, or 0.
type page struct {
	createdAtMS  int64
	live         atomic.Int64
	emptySinceMS atomic.Int64
	slots        [pageSize]slot
}

// slab hands out lease records by sequence number.  pages are created on
// first use and freed by housekeeping once every slot in them is taken.
type slab struct {
	// next is the last allocated seq.  seq 0 is never allocated.
	next  atomic.Uint64
	pages sync.Map // uint64 page index -> *page
}

// alloc returns a live slot for a lease that expires at expiresAtMS.  live is
// raised before the slot is published so a page never reads as empty while a
// slot in it is about to go live.
func (s *slab) alloc(nowMS, expiresAtMS int64, req *requestState) (seq uint64, sl *slot) {
	seq = s.next.Add(1)
	idx := seq >> pageBits
	var p *page
	if v, ok := s.pages.Load(idx); ok {
		p = v.(*page)
	} else {
		v, _ := s.pages.LoadOrStore(idx, &page{createdAtMS: nowMS})
		p = v.(*page)
	}
	sl = &p.slots[seq&(pageSize-1)]
	sl.expiresAtMS = expiresAtMS
	sl.req.Store(req)
	p.live.Add(1)
	sl.state.Store(slotLive)
	return seq, sl
}

// get returns the slot for seq while it is live, else nil.
func (s *slab) get(seq uint64) *slot {
	p := s.page(seq)
	if p == nil {
		return nil
	}
	sl := &p.slots[seq&(pageSize-1)]
	if sl.state.Load() != slotLive {
		return nil
	}
	return sl
}

// page returns the page holding seq, or nil when it was freed or never
// allocated.
func (s *slab) page(seq uint64) *page {
	v, ok := s.pages.Load(seq >> pageBits)
	if !ok {
		return nil
	}
	return v.(*page)
}

// freePages drops every page that is not the current allocation page and has
// had no live slot for graceMS across two calls.  the grace covers an alloc
// that has taken a seq in the page but not yet raised live.  a late release
// for a freed page finds no record, the same as for a taken slot.
func (s *slab) freePages(nowMS, graceMS int64) (freed int) {
	current := s.next.Load() >> pageBits
	s.pages.Range(func(k, v any) bool {
		idx, p := k.(uint64), v.(*page)
		if idx >= current {
			return true
		}
		if p.live.Load() != 0 {
			p.emptySinceMS.Store(0)
			return true
		}
		since := p.emptySinceMS.Load()
		if since == 0 {
			p.emptySinceMS.Store(nowMS)
			return true
		}
		if nowMS <= since+graceMS {
			return true
		}
		if s.pages.CompareAndDelete(k, v) {
			freed++
		}
		return true
	})
	return freed
}

// pageCount is for tests.
func (s *slab) pageCount() int {
	n := 0
	s.pages.Range(func(any, any) bool { n++; return true })
	return n
}
