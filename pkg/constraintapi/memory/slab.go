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
	// shard sits in the top 4 of the seq's 48 bits, so each shard fills its
	// own pages.  a shard's 44 bit counter wraps after 1.7e13 leases, 5 years
	// at 100k leases per second on that shard alone.
	slabShards = 16
	shardShift = 44

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
	p := s.pageFor(seq)
	sl = &p.slots[seq&(pageSize-1)]
	sl.expiresAtMS = expiresAtMS
	sl.req.Store(req)
	sl.state.Store(slotLive)
	return seq, sl
}

// pageFor returns the page that contains seq, creating it on first use.
func (s *slab) pageFor(seq uint64) *page {
	idx := seq >> pageBits
	if v, ok := s.pages.Load(idx); ok {
		return v.(*page)
	}
	np := &page{}
	shard := seq >> shardShift
	if idx == shard<<(shardShift-pageBits) {
		np.slots[0].state.Store(slotTaken)
	}
	v, _ := s.pages.LoadOrStore(idx, np)
	return v.(*page)
}

// seqMask is the counter part of a seq, below the shard bits.
const seqMask = 1<<shardShift - 1

// allocAt places a live slot at a seq another Manager handed out, and raises
// the shard counter to it.  a follower calls it for every replicated lease.
// it returns false when the slot was already used and nothing changed.
func (s *slab) allocAt(seq uint64, expiresAtMS int64, req *requestState) bool {
	s.raise(int(seq>>shardShift)%slabShards, seq&seqMask)
	sl := &s.pageFor(seq).slots[seq&(pageSize-1)]
	if sl.state.Load() != slotEmpty {
		return false
	}
	sl.expiresAtMS = expiresAtMS
	sl.req.Store(req)
	sl.state.Store(slotLive)
	return true
}

// raise moves a shard counter up to at least n.
func (s *slab) raise(shard int, n uint64) {
	next := &s.shards[shard].next
	for {
		cur := next.Load()
		if cur >= n || next.CompareAndSwap(cur, n) {
			return
		}
	}
}

// seqJump is how far past the highest known seq a promoted follower starts.
// the old primary may have handed out seqs it never replicated, and a seq
// must never name two leases.
const seqJump = 1 << 20

// seal makes the slab ready to hand out seqs after it only received them.
// every empty slot in an existing page is marked taken so the page can be
// freed, and every shard counter moves to a page boundary at least seqJump
// past the highest seq seen.
func (s *slab) seal() {
	s.pages.Range(func(_, v any) bool {
		p := v.(*page)
		for i := range p.slots {
			p.slots[i].state.CompareAndSwap(slotEmpty, slotTaken)
		}
		return true
	})
	for i := range s.shards {
		next := &s.shards[i].next
		// the first seq handed out is next+1, which must be slot 0 of a page
		target := (next.Load()+seqJump+pageSize)&^uint64(pageSize-1) - 1
		next.Store(target)
	}
}

// each calls fn for every live slot.
func (s *slab) each(fn func(seq uint64, sl *slot)) {
	s.pages.Range(func(k, v any) bool {
		idx, p := k.(uint64), v.(*page)
		for i := range p.slots {
			if p.slots[i].state.Load() == slotLive {
				fn(idx<<pageBits|uint64(i), &p.slots[i])
			}
		}
		return true
	})
}

// noLive is true when no slot in the page is live.
func (p *page) noLive() bool {
	for i := range p.slots {
		if p.slots[i].state.Load() == slotLive {
			return false
		}
	}
	return true
}

// freeIdlePages is freePages for a follower.  a follower receives seqs out
// of order and never sees the seqs of leases that ended before it joined, so
// its pages keep empty slots.  it drops every page with no live slot that is
// below its shard's allocation page.  a late lease for a dropped page creates
// the page again.
func (s *slab) freeIdlePages() (freed int) {
	s.pages.Range(func(k, v any) bool {
		idx, p := k.(uint64), v.(*page)
		shard := int(idx>>(shardShift-pageBits)) % slabShards
		if idx >= s.currentPage(shard) || !p.noLive() {
			return true
		}
		if s.pages.CompareAndDelete(k, v) {
			freed++
		}
		return true
	})
	return freed
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
