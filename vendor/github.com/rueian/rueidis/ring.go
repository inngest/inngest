package rueidis

import (
	"sync"
	"sync/atomic"
)

type queue interface {
	PutOne(m Completed) chan RedisResult
	PutMulti(m []Completed) chan RedisResult
	NextWriteCmd() (Completed, []Completed, chan RedisResult)
	WaitForWrite() (Completed, []Completed, chan RedisResult)
	NextResultCh() (Completed, []Completed, chan RedisResult, *sync.Cond)
}

var _ queue = (*ring)(nil)

func newRing(factor int) *ring {
	if factor <= 0 {
		factor = DefaultRingScale
	}
	r := &ring{store: make([]node, 2<<(factor-1))}
	r.mask = uint64(len(r.store) - 1)
	for i := range r.store {
		m := &sync.Mutex{}
		r.store[i].c1 = sync.NewCond(m)
		r.store[i].c2 = sync.NewCond(m)
		r.store[i].ch = make(chan RedisResult, 0) // this channel can't be buffered
	}
	return r
}

type ring struct {
	store []node // store's size must be 2^N to work with the mask
	_     [5]uint64
	write uint64
	_     [7]uint64
	read1 uint64
	read2 uint64
	mask  uint64
}

type node struct {
	c1    *sync.Cond
	c2    *sync.Cond
	ch    chan RedisResult
	one   Completed
	multi []Completed
	mark  uint32
	slept bool
}

func (r *ring) PutOne(m Completed) chan RedisResult {
	n := &r.store[atomic.AddUint64(&r.write, 1)&r.mask]
	n.c1.L.Lock()
	for n.mark != 0 {
		n.c1.Wait()
	}
	n.one = m
	n.multi = nil
	n.mark = 1
	s := n.slept
	n.c1.L.Unlock()
	if s {
		n.c2.Broadcast()
	}
	return n.ch
}

func (r *ring) PutMulti(m []Completed) chan RedisResult {
	n := &r.store[atomic.AddUint64(&r.write, 1)&r.mask]
	n.c1.L.Lock()
	for n.mark != 0 {
		n.c1.Wait()
	}
	n.one = Completed{}
	n.multi = m
	n.mark = 1
	s := n.slept
	n.c1.L.Unlock()
	if s {
		n.c2.Broadcast()
	}
	return n.ch
}

// NextWriteCmd should be only called by one dedicated thread
func (r *ring) NextWriteCmd() (one Completed, multi []Completed, ch chan RedisResult) {
	r.read1++
	p := r.read1 & r.mask
	n := &r.store[p]
	n.c1.L.Lock()
	if n.mark == 1 {
		one, multi, ch = n.one, n.multi, n.ch
		n.mark = 2
	} else {
		r.read1--
	}
	n.c1.L.Unlock()
	return
}

// WaitForWrite should be only called by one dedicated thread
func (r *ring) WaitForWrite() (one Completed, multi []Completed, ch chan RedisResult) {
	r.read1++
	p := r.read1 & r.mask
	n := &r.store[p]
	n.c1.L.Lock()
	for n.mark != 1 {
		n.slept = true
		n.c2.Wait() // c1 and c2 share the same mutex
		n.slept = false
	}
	one, multi, ch = n.one, n.multi, n.ch
	n.mark = 2
	n.c1.L.Unlock()
	return
}

// NextResultCh should be only called by one dedicated thread
func (r *ring) NextResultCh() (one Completed, multi []Completed, ch chan RedisResult, cond *sync.Cond) {
	r.read2++
	p := r.read2 & r.mask
	n := &r.store[p]
	cond = n.c1
	n.c1.L.Lock()
	if n.mark == 2 {
		one, multi, ch = n.one, n.multi, n.ch
		n.mark = 0
	} else {
		r.read2--
	}
	return
}
