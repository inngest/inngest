package rueidis

import (
	"context"
	"math/rand"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/rueidis/internal/cmds"
	"github.com/redis/rueidis/internal/util"
)

type connFn func(dst string, opt *ClientOption) conn
type dialFn func(dst string, opt *ClientOption) (net.Conn, error)
type wireFn func() wire

type singleconnect struct {
	w wire
	e error
	g sync.WaitGroup
}

type batchcache struct {
	cIndexes []int
	commands []CacheableTTL
}

func (r *batchcache) Capacity() int {
	return cap(r.commands)
}

func (r *batchcache) ResetLen(n int) {
	r.cIndexes = r.cIndexes[:n]
	r.commands = r.commands[:n]
}

var batchcachep = util.NewPool(func(capacity int) *batchcache {
	return &batchcache{
		cIndexes: make([]int, 0, capacity),
		commands: make([]CacheableTTL, 0, capacity),
	}
})

type conn interface {
	Do(ctx context.Context, cmd Completed) RedisResult
	DoCache(ctx context.Context, cmd Cacheable, ttl time.Duration) RedisResult
	DoMulti(ctx context.Context, multi ...Completed) *redisresults
	DoMultiCache(ctx context.Context, multi ...CacheableTTL) *redisresults
	Receive(ctx context.Context, subscribe Completed, fn func(message PubSubMessage)) error
	Info() map[string]RedisMessage
	Error() error
	Close()
	Dial() error
	Override(conn)
	Acquire() wire
	Store(w wire)
	Addr() string
}

var _ conn = (*mux)(nil)

type mux struct {
	init   wire
	dead   wire
	pool   *pool
	wireFn wireFn
	dst    string
	wire   []atomic.Value
	sc     []*singleconnect
	mu     []sync.Mutex
	maxp   int
}

func makeMux(dst string, option *ClientOption, dialFn dialFn) *mux {
	dead := deadFn()
	return newMux(dst, option, (*pipe)(nil), dead, func() (w wire) {
		w, err := newPipe(func() (net.Conn, error) {
			return dialFn(dst, option)
		}, option)
		if err != nil {
			dead.error.Store(&errs{error: err})
			w = dead
		}
		return w
	})
}

func newMux(dst string, option *ClientOption, init, dead wire, wireFn wireFn) *mux {
	var multiplex int
	if option.PipelineMultiplex >= 0 {
		multiplex = 1 << option.PipelineMultiplex
	} else {
		multiplex = 1
	}
	m := &mux{dst: dst, init: init, dead: dead, wireFn: wireFn,
		wire: make([]atomic.Value, multiplex),
		mu:   make([]sync.Mutex, multiplex),
		sc:   make([]*singleconnect, multiplex),
		maxp: runtime.GOMAXPROCS(0),
	}
	for i := 0; i < len(m.wire); i++ {
		m.wire[i].Store(init)
	}
	m.pool = newPool(option.BlockingPoolSize, dead, wireFn)
	return m
}

func (m *mux) Override(cc conn) {
	if m2, ok := cc.(*mux); ok {
		for i := 0; i < len(m.wire) && i < len(m2.wire); i++ {
			m.wire[i].CompareAndSwap(m.init, m2.wire[i].Load())
		}
	}
}

func (m *mux) _pipe(i uint16) (w wire, err error) {
	if w = m.wire[i].Load().(wire); w != m.init {
		return w, nil
	}

	m.mu[i].Lock()
	sc := m.sc[i]
	if m.sc[i] == nil {
		m.sc[i] = &singleconnect{}
		m.sc[i].g.Add(1)
	}
	m.mu[i].Unlock()

	if sc != nil {
		sc.g.Wait()
		return sc.w, sc.e
	}

	if w = m.wire[i].Load().(wire); w == m.init {
		if w = m.wireFn(); w != m.dead {
			i := i
			w := w
			w.SetOnCloseHook(func(err error) {
				if err != ErrClosing {
					m.wire[i].CompareAndSwap(w, m.init)
				}
			})
			m.wire[i].Store(w)
		} else {
			err = w.Error()
		}
	}

	m.mu[i].Lock()
	sc = m.sc[i]
	m.sc[i] = nil
	m.mu[i].Unlock()

	sc.w = w
	sc.e = err
	sc.g.Done()

	return w, err
}

func (m *mux) pipe(i uint16) wire {
	w, _ := m._pipe(i)
	return w // this should never be nil
}

func (m *mux) Dial() error {
	_, err := m._pipe(0)
	return err
}

func (m *mux) Info() map[string]RedisMessage {
	return m.pipe(0).Info()
}

func (m *mux) Error() error {
	return m.pipe(0).Error()
}

func (m *mux) Do(ctx context.Context, cmd Completed) (resp RedisResult) {
	if cmd.IsBlock() {
		resp = m.blocking(ctx, cmd)
	} else {
		resp = m.pipeline(ctx, cmd)
	}
	return resp
}

func (m *mux) DoMulti(ctx context.Context, multi ...Completed) (resp *redisresults) {
	for _, cmd := range multi {
		if cmd.IsBlock() {
			goto block
		}
	}
	return m.pipelineMulti(ctx, multi)
block:
	cmds.ToBlock(&multi[0]) // mark the first cmd as block if one of them is block to shortcut later check.
	return m.blockingMulti(ctx, multi)
}

func (m *mux) blocking(ctx context.Context, cmd Completed) (resp RedisResult) {
	wire := m.pool.Acquire()
	resp = wire.Do(ctx, cmd)
	if resp.NonRedisError() != nil { // abort the wire if blocking command return early (ex. context.DeadlineExceeded)
		wire.Close()
	}
	m.pool.Store(wire)
	return resp
}

func (m *mux) blockingMulti(ctx context.Context, cmd []Completed) (resp *redisresults) {
	wire := m.pool.Acquire()
	resp = wire.DoMulti(ctx, cmd...)
	for _, res := range resp.s {
		if res.NonRedisError() != nil { // abort the wire if blocking command return early (ex. context.DeadlineExceeded)
			wire.Close()
			break
		}
	}
	m.pool.Store(wire)
	return resp
}

func (m *mux) pipeline(ctx context.Context, cmd Completed) (resp RedisResult) {
	slot := slotfn(len(m.wire), cmd.Slot(), cmd.NoReply())
	wire := m.pipe(slot)
	if resp = wire.Do(ctx, cmd); isBroken(resp.NonRedisError(), wire) {
		m.wire[slot].CompareAndSwap(wire, m.init)
	}
	return resp
}

func (m *mux) pipelineMulti(ctx context.Context, cmd []Completed) (resp *redisresults) {
	slot := slotfn(len(m.wire), cmd[0].Slot(), cmd[0].NoReply())
	wire := m.pipe(slot)
	resp = wire.DoMulti(ctx, cmd...)
	for _, r := range resp.s {
		if isBroken(r.NonRedisError(), wire) {
			m.wire[slot].CompareAndSwap(wire, m.init)
			return resp
		}
	}
	return resp
}

func (m *mux) DoCache(ctx context.Context, cmd Cacheable, ttl time.Duration) RedisResult {
	slot := cmd.Slot() & uint16(len(m.wire)-1)
	wire := m.pipe(slot)
	resp := wire.DoCache(ctx, cmd, ttl)
	if isBroken(resp.NonRedisError(), wire) {
		m.wire[slot].CompareAndSwap(wire, m.init)
	}
	return resp
}

func (m *mux) DoMultiCache(ctx context.Context, multi ...CacheableTTL) (results *redisresults) {
	var slots *muxslots
	var mask = uint16(len(m.wire) - 1)

	if mask == 0 {
		return m.doMultiCache(ctx, 0, multi)
	}

	slots = muxslotsp.Get(len(m.wire), len(m.wire))
	for _, cmd := range multi {
		slots.s[cmd.Cmd.Slot()&mask]++
	}

	if slots.LessThen(2) {
		return m.doMultiCache(ctx, multi[0].Cmd.Slot()&mask, multi)
	}

	batches := batchcachemaps.Get(len(m.wire), len(m.wire))
	for slot, count := range slots.s {
		if count > 0 {
			batches.m[uint16(slot)] = batchcachep.Get(0, count)
		}
	}
	muxslotsp.Put(slots)

	for i, cmd := range multi {
		batch := batches.m[cmd.Cmd.Slot()&mask]
		batch.commands = append(batch.commands, cmd)
		batch.cIndexes = append(batch.cIndexes, i)
	}

	results = resultsp.Get(len(multi), len(multi))
	util.ParallelKeys(m.maxp, batches.m, func(slot uint16) {
		batch := batches.m[slot]
		resp := m.doMultiCache(ctx, slot, batch.commands)
		for i, r := range resp.s {
			results.s[batch.cIndexes[i]] = r
		}
		resultsp.Put(resp)
	})

	for _, batch := range batches.m {
		batchcachep.Put(batch)
	}
	batchcachemaps.Put(batches)

	return results
}

func (m *mux) doMultiCache(ctx context.Context, slot uint16, multi []CacheableTTL) (resps *redisresults) {
	wire := m.pipe(slot)
	resps = wire.DoMultiCache(ctx, multi...)
	for _, r := range resps.s {
		if isBroken(r.NonRedisError(), wire) {
			m.wire[slot].CompareAndSwap(wire, m.init)
			return resps
		}
	}
	return resps
}

func (m *mux) Receive(ctx context.Context, subscribe Completed, fn func(message PubSubMessage)) error {
	slot := slotfn(len(m.wire), subscribe.Slot(), subscribe.NoReply())
	wire := m.pipe(slot)
	err := wire.Receive(ctx, subscribe, fn)
	if isBroken(err, wire) {
		m.wire[slot].CompareAndSwap(wire, m.init)
	}
	return err
}

func (m *mux) Acquire() wire {
	return m.pool.Acquire()
}

func (m *mux) Store(w wire) {
	w.SetPubSubHooks(PubSubHooks{})
	w.CleanSubscriptions()
	m.pool.Store(w)
}

func (m *mux) Close() {
	for i := 0; i < len(m.wire); i++ {
		if prev := m.wire[i].Swap(m.dead).(wire); prev != m.init && prev != m.dead {
			prev.Close()
		}
	}
	m.pool.Close()
}

func (m *mux) Addr() string {
	return m.dst
}

func isBroken(err error, w wire) bool {
	return err != nil && err != ErrClosing && w.Error() != nil
}

var rngPool = sync.Pool{
	New: func() any {
		return rand.New(rand.NewSource(time.Now().UnixNano()))
	},
}

func fastrand(n int) (r int) {
	s := rngPool.Get().(*rand.Rand)
	r = s.Intn(n)
	rngPool.Put(s)
	return
}

func slotfn(n int, ks uint16, noreply bool) uint16 {
	if n == 1 || ks == cmds.NoSlot || noreply {
		return 0
	}
	return uint16(fastrand(n))
}

type muxslots struct {
	s []int
}

func (r *muxslots) Capacity() int {
	return cap(r.s)
}

func (r *muxslots) ResetLen(n int) {
	r.s = r.s[:n]
	for i := 0; i < n; i++ {
		r.s[i] = 0
	}
}

func (r *muxslots) LessThen(n int) bool {
	count := 0
	for _, value := range r.s {
		if value > 0 {
			if count++; count == n {
				return false
			}
		}
	}
	return true
}

var muxslotsp = util.NewPool(func(capacity int) *muxslots {
	return &muxslots{s: make([]int, 0, capacity)}
})

type batchcachemap struct {
	m map[uint16]*batchcache
	n int
}

func (r *batchcachemap) Capacity() int {
	return r.n
}

func (r *batchcachemap) ResetLen(n int) {
	for k := range r.m {
		delete(r.m, k)
	}
}

var batchcachemaps = util.NewPool(func(capacity int) *batchcachemap {
	return &batchcachemap{m: make(map[uint16]*batchcache, capacity), n: capacity}
})
