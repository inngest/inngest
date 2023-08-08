package rueidis

import "sync"

func newPool(cap int, dead wire, makeFn func() wire) *pool {
	if cap <= 0 {
		cap = DefaultPoolSize
	}

	return &pool{
		size: 0,
		dead: dead,
		make: makeFn,
		list: make([]wire, 0, cap),
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

type pool struct {
	dead wire
	cond *sync.Cond
	make func() wire
	list []wire
	size int
	down bool
}

func (p *pool) Acquire() (v wire) {
	p.cond.L.Lock()
	for len(p.list) == 0 && p.size == cap(p.list) && !p.down {
		p.cond.Wait()
	}
	if p.down {
		v = p.dead
	} else if len(p.list) == 0 {
		p.size++
		v = p.make()
	} else {
		i := len(p.list) - 1
		v = p.list[i]
		p.list = p.list[:i]
	}
	p.cond.L.Unlock()
	return v
}

func (p *pool) Store(v wire) {
	p.cond.L.Lock()
	if !p.down && v.Error() == nil {
		p.list = append(p.list, v)
	} else {
		p.size--
		v.Close()
	}
	p.cond.L.Unlock()
	p.cond.Signal()
}

func (p *pool) Close() {
	p.cond.L.Lock()
	p.down = true
	for _, w := range p.list {
		w.Close()
	}
	p.cond.L.Unlock()
	p.cond.Broadcast()
}
