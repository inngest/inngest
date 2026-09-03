package memory

import (
	"sync"
	"sync/atomic"
)

const (
	pageBits = 16
	pageSize = 1 << pageBits

	slotEmpty uint32 = 0
	slotLive  uint32 = 1
	slotTaken uint32 = 2
)

// slot is one lease record.  it is written once by alloc and taken once by
// the release, extend or sweep that wins the CAS.  seqs never repeat, so a
// slot never goes back to live.
type slot struct {
	state       atomic.Uint32
	expiresAtMS int64
	req         *requestState
}

// take moves the slot from live to taken.  the winner owns the record and is
// the only caller allowed to give back the counters it holds.
func (s *slot) take() bool {
	return s.state.CompareAndSwap(slotLive, slotTaken)
}

// page holds pageSize slots.  live counts slots in state live so the sweeper
// can free a page nothing points at.
type page struct {
	createdAtMS int64
	live        atomic.Int64
	slots       [pageSize]slot
}

// slab hands out lease records by sequence number.  pages are created on
// first use and freed by housekeeping when every slot is taken and the page
// is older than the maximum lease lifetime.
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
	sl.req = req
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

// freePages drops every page that is not the current allocation page, has no
// live slot, and was created more than maxLifetimeMS ago.  a late release for
// a freed page finds no record, the same as Redis after cleanup.
func (s *slab) freePages(nowMS, maxLifetimeMS int64) (freed int) {
	current := s.next.Load() >> pageBits
	s.pages.Range(func(k, v any) bool {
		idx, p := k.(uint64), v.(*page)
		if idx >= current || p.live.Load() != 0 || p.createdAtMS+maxLifetimeMS >= nowMS {
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
