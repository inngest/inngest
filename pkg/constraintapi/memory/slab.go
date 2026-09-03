package memory

import (
	"math/rand/v2"
	"sync"
	"sync/atomic"
)

const (
	// pageBits sizes a page at 8192 slots, about 200 KB.  a page is freed once
	// every slot in it is taken, so smaller pages return memory sooner.
	pageBits = 13
	pageSize = 1 << pageBits

	// slabShards is the number of independent sequence counters.  an alloc
	// picks one at random, so cores rarely contend on the same counter.  the
	// shard sits in the top byte of the seq, so each shard fills its own
	// pages.
	slabShards = 16
	shardShift = 56

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

// page holds pageSize slots.
type page struct {
	slots [pageSize]slot
}

// allTaken is true once every slot was allocated and taken.  a slot still
// empty belongs to an alloc that has claimed its seq but not published yet,
// so the page stays.
func (p *page) allTaken() bool {
	for i := range p.slots {
		if p.slots[i].state.Load() != slotTaken {
			return false
		}
	}
	return true
}

// seqShard is one sequence counter on its own cache line.
type seqShard struct {
	next atomic.Uint64
	_    [56]byte
}

// slab hands out lease records by sequence number.  pages are created on
// first use and freed by housekeeping once every slot in them is taken.
type slab struct {
	shards [slabShards]seqShard
	pages  sync.Map // uint64 page index -> *page
}

// alloc returns a live slot for a lease that expires at expiresAtMS.
func (s *slab) alloc(nowMS, expiresAtMS int64, req *requestState) (seq uint64, sl *slot) {
	return s.allocIn(int(rand.Uint32()%slabShards), expiresAtMS, req)
}

// allocIn allocates from one shard.  every shard counts from 1, so seq 0 and
// the first slot of each shard's first page are never handed out.  that slot
// is marked taken when the page is created so the page can still be freed.
func (s *slab) allocIn(shard int, expiresAtMS int64, req *requestState) (seq uint64, sl *slot) {
	seq = uint64(shard)<<shardShift | s.shards[shard].next.Add(1)
	idx := seq >> pageBits
	var p *page
	if v, ok := s.pages.Load(idx); ok {
		p = v.(*page)
	} else {
		np := &page{}
		if idx == uint64(shard)<<(shardShift-pageBits) {
			np.slots[0].state.Store(slotTaken)
		}
		v, _ := s.pages.LoadOrStore(idx, np)
		p = v.(*page)
	}
	sl = &p.slots[seq&(pageSize-1)]
	sl.expiresAtMS = expiresAtMS
	sl.req.Store(req)
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

// currentPage is the page index the shard allocates into.
func (s *slab) currentPage(shard int) uint64 {
	return (uint64(shard)<<shardShift | s.shards[shard].next.Load()) >> pageBits
}

// freePages drops every page that is not its shard's allocation page and has
// every slot taken.  a late release for a freed page finds no record, the
// same as for a taken slot.
func (s *slab) freePages() (freed int) {
	s.pages.Range(func(k, v any) bool {
		idx, p := k.(uint64), v.(*page)
		shard := int(idx >> (shardShift - pageBits))
		if idx == s.currentPage(shard) || !p.allTaken() {
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
