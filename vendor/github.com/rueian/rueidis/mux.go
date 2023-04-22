package rueidis

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rueian/rueidis/internal/cmds"
	"github.com/rueian/rueidis/internal/util"
)

type connFn func(dst string, opt *ClientOption) conn
type dialFn func(dst string, opt *ClientOption) (net.Conn, error)
type wireFn func() wire

type singleconnect struct {
	w wire
	e error
	g sync.WaitGroup
}

type conn interface {
	Do(ctx context.Context, cmd Completed) RedisResult
	DoCache(ctx context.Context, cmd Cacheable, ttl time.Duration) RedisResult
	DoMulti(ctx context.Context, multi ...Completed) []RedisResult
	DoMultiCache(ctx context.Context, multi ...CacheableTTL) []RedisResult
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

func (m *mux) DoMulti(ctx context.Context, multi ...Completed) (resp []RedisResult) {
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

func (m *mux) blockingMulti(ctx context.Context, cmd []Completed) (resp []RedisResult) {
	wire := m.pool.Acquire()
	resp = wire.DoMulti(ctx, cmd...)
	for _, res := range resp {
		if res.NonRedisError() != nil { // abort the wire if blocking command return early (ex. context.DeadlineExceeded)
			wire.Close()
			break
		}
	}
	m.pool.Store(wire)
	return resp
}

func (m *mux) pipeline(ctx context.Context, cmd Completed) (resp RedisResult) {
	slot := cmd.Slot() & uint16(len(m.wire)-1)
	wire := m.pipe(slot)
	if resp = wire.Do(ctx, cmd); isBroken(resp.NonRedisError(), wire) {
		m.wire[slot].CompareAndSwap(wire, m.init)
	}
	return resp
}

func (m *mux) pipelineMulti(ctx context.Context, cmd []Completed) (resp []RedisResult) {
	slot := cmd[0].Slot() & uint16(len(m.wire)-1)
	wire := m.pipe(slot)
	resp = wire.DoMulti(ctx, cmd...)
	for _, r := range resp {
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

func (m *mux) DoMultiCache(ctx context.Context, multi ...CacheableTTL) (results []RedisResult) {
	var slots map[uint16]int
	var mask = uint16(len(m.wire) - 1)

	if mask == 0 {
		return m.doMultiCache(ctx, 0, multi)
	}

	slots = make(map[uint16]int, len(m.wire))
	for _, cmd := range multi {
		slots[cmd.Cmd.Slot()&mask]++
	}

	if len(slots) == 1 {
		return m.doMultiCache(ctx, multi[0].Cmd.Slot()&mask, multi)
	}

	commands := make(map[uint16][]CacheableTTL, len(slots))
	cIndexes := make(map[uint16][]int, len(slots))
	for slot, count := range slots {
		cIndexes[slot] = make([]int, 0, count)
		commands[slot] = make([]CacheableTTL, 0, count)
	}
	for i, cmd := range multi {
		slot := cmd.Cmd.Slot() & mask
		commands[slot] = append(commands[slot], cmd)
		cIndexes[slot] = append(cIndexes[slot], i)
	}

	results = make([]RedisResult, len(multi))
	util.ParallelKeys(commands, func(slot uint16) {
		for i, r := range m.doMultiCache(ctx, slot, commands[slot]) {
			results[cIndexes[slot][i]] = r
		}
	})

	return results
}

func (m *mux) doMultiCache(ctx context.Context, slot uint16, multi []CacheableTTL) (resps []RedisResult) {
	wire := m.pipe(slot)
	resps = wire.DoMultiCache(ctx, multi...)
	for _, r := range resps {
		if isBroken(r.NonRedisError(), wire) {
			m.wire[slot].CompareAndSwap(wire, m.init)
			return resps
		}
	}
	return resps
}

func (m *mux) Receive(ctx context.Context, subscribe Completed, fn func(message PubSubMessage)) error {
	slot := subscribe.Slot() & uint16(len(m.wire)-1)
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
