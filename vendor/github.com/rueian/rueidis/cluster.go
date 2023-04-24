package rueidis

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rueian/rueidis/internal/cmds"
	"github.com/rueian/rueidis/internal/util"
)

// ErrNoSlot indicates that there is no redis node owns the key slot.
var ErrNoSlot = errors.New("the slot has no redis node")

type clusterClient struct {
	slots  [16384]conn
	opt    *ClientOption
	conns  map[string]conn
	connFn connFn
	sc     call
	mu     sync.RWMutex
	stop   uint32
	cmd    cmds.Builder
	retry  bool
}

func newClusterClient(opt *ClientOption, connFn connFn) (client *clusterClient, err error) {
	client = &clusterClient{
		cmd:    cmds.NewBuilder(cmds.InitSlot),
		opt:    opt,
		connFn: connFn,
		conns:  make(map[string]conn),
		retry:  !opt.DisableRetry,
	}

	if err = client.init(); err != nil {
		return nil, err
	}

	if err = client.refresh(); err != nil {
		return client, err
	}

	return client, nil
}

func (c *clusterClient) init() error {
	if len(c.opt.InitAddress) == 0 {
		return ErrNoAddr
	}
	results := make(chan error, len(c.opt.InitAddress))
	for _, addr := range c.opt.InitAddress {
		cc := c.connFn(addr, c.opt)
		go func(addr string, cc conn) {
			if err := cc.Dial(); err == nil {
				c.mu.Lock()
				if _, ok := c.conns[addr]; ok {
					go cc.Close() // abort the new connection instead of closing the old one which may already been used
				} else {
					c.conns[addr] = cc
				}
				c.mu.Unlock()
				results <- nil
			} else {
				results <- err
			}
		}(addr, cc)
	}
	es := make([]error, cap(results))
	for i := 0; i < cap(results); i++ {
		if err := <-results; err == nil {
			return nil
		} else {
			es[i] = err
		}
	}
	return es[0]
}

func (c *clusterClient) refresh() (err error) {
	return c.sc.Do(c._refresh)
}

func (c *clusterClient) _refresh() (err error) {
	var reply RedisMessage

retry:
	c.mu.RLock()
	results := make(chan RedisResult, len(c.conns))
	for _, cc := range c.conns {
		go func(c conn) { results <- c.Do(context.Background(), cmds.SlotCmd) }(cc)
	}
	c.mu.RUnlock()

	for i := 0; i < cap(results); i++ {
		if reply, err = (<-results).ToMessage(); len(reply.values) != 0 {
			break
		}
	}

	if err != nil {
		return err
	}

	if len(reply.values) == 0 {
		if err = c.init(); err != nil {
			return err
		}
		goto retry
	}

	groups := parseSlots(reply)

	// TODO support read from replicas
	conns := make(map[string]conn, len(groups))
	for _, g := range groups {
		for _, addr := range g.nodes {
			conns[addr] = c.connFn(addr, c.opt)
		}
	}
	// make sure InitAddress always be present
	for _, addr := range c.opt.InitAddress {
		if _, ok := conns[addr]; !ok {
			conns[addr] = c.connFn(addr, c.opt)
		}
	}

	var removes []conn

	c.mu.RLock()
	for addr, cc := range c.conns {
		if _, ok := conns[addr]; ok {
			conns[addr] = cc
		} else {
			removes = append(removes, cc)
		}
	}
	c.mu.RUnlock()

	slots := [16384]conn{}
	for master, g := range groups {
		cc := conns[master]
		for _, slot := range g.slots {
			for i := slot[0]; i <= slot[1]; i++ {
				slots[i] = cc
			}
		}
	}

	c.mu.Lock()
	c.slots = slots
	c.conns = conns
	c.mu.Unlock()

	for _, cc := range removes {
		go cc.Close()
	}

	return nil
}

func (c *clusterClient) single() conn {
	return c._pick(cmds.InitSlot)
}

func (c *clusterClient) nodes() []string {
	c.mu.RLock()
	nodes := make([]string, 0, len(c.conns))
	for addr := range c.conns {
		nodes = append(nodes, addr)
	}
	c.mu.RUnlock()
	return nodes
}

type group struct {
	nodes []string
	slots [][2]int64
}

func parseSlots(slots RedisMessage) map[string]group {
	groups := make(map[string]group, len(slots.values))
	for _, v := range slots.values {
		master := fmt.Sprintf("%s:%d", v.values[2].values[0].string, v.values[2].values[1].integer)
		g, ok := groups[master]
		if !ok {
			g.slots = make([][2]int64, 0)
			g.nodes = make([]string, 0, len(v.values)-2)
			for i := 2; i < len(v.values); i++ {
				dst := fmt.Sprintf("%s:%d", v.values[i].values[0].string, v.values[i].values[1].integer)
				g.nodes = append(g.nodes, dst)
			}
		}
		g.slots = append(g.slots, [2]int64{v.values[0].integer, v.values[1].integer})
		groups[master] = g
	}
	return groups
}

func (c *clusterClient) _pick(slot uint16) (p conn) {
	c.mu.RLock()
	if slot == cmds.InitSlot {
		for _, cc := range c.conns {
			p = cc
			break
		}
	} else {
		p = c.slots[slot]
	}
	c.mu.RUnlock()
	return p
}

func (c *clusterClient) pick(slot uint16) (p conn, err error) {
	if p = c._pick(slot); p == nil {
		if err := c.refresh(); err != nil {
			return nil, err
		}
		if p = c._pick(slot); p == nil {
			return nil, ErrNoSlot
		}
	}
	return p, nil
}

func (c *clusterClient) redirectOrNew(addr string) (p conn) {
	c.mu.RLock()
	p = c.conns[addr]
	c.mu.RUnlock()
	if p != nil && p.Addr() != addr {
		return p
	}
	c.mu.Lock()
	if p = c.conns[addr]; p == nil {
		p = c.connFn(addr, c.opt)
		c.conns[addr] = p
	} else if p.Addr() == addr {
		// try reconnection if the MOVED redirects to the same host,
		// because the same hostname may actually be resolved into another destination
		// depending on the fail-over implementation. ex: AWS MemoryDB's resize process.
		go p.Close()
		p = c.connFn(addr, c.opt)
		c.conns[addr] = p
	}
	c.mu.Unlock()
	return p
}

func (c *clusterClient) B() cmds.Builder {
	return c.cmd
}

func (c *clusterClient) Do(ctx context.Context, cmd Completed) (resp RedisResult) {
	if resp = c.do(ctx, cmd); resp.NonRedisError() == nil { // not recycle cmds if error, since cmds may be used later in pipe. consider recycle them by pipe
		cmds.PutCompleted(cmd)
	}
	return resp
}

func (c *clusterClient) do(ctx context.Context, cmd Completed) (resp RedisResult) {
retry:
	cc, err := c.pick(cmd.Slot())
	if err != nil {
		return newErrResult(err)
	}
	resp = cc.Do(ctx, cmd)
process:
	switch addr, mode := c.shouldRefreshRetry(resp.Error(), ctx); mode {
	case RedirectMove:
		resp = c.redirectOrNew(addr).Do(ctx, cmd)
		goto process
	case RedirectAsk:
		resp = c.redirectOrNew(addr).DoMulti(ctx, cmds.AskingCmd, cmd)[1]
		goto process
	case RedirectRetry:
		if c.retry && cmd.IsReadOnly() {
			runtime.Gosched()
			goto retry
		}
	}
	return resp
}

func (c *clusterClient) DoMulti(ctx context.Context, multi ...Completed) (results []RedisResult) {
	if len(multi) == 0 {
		return nil
	}
	slots := make(map[uint16]int, 8)
	for _, cmd := range multi {
		slots[cmd.Slot()]++
	}
	if len(slots) > 2 && slots[cmds.InitSlot] > 0 {
		panic(panicMixCxSlot)
	}
	commands := make(map[uint16][]Completed, len(slots))
	cIndexes := make(map[uint16][]int, len(slots))
	if len(slots) == 2 && slots[cmds.InitSlot] > 0 {
		delete(slots, cmds.InitSlot)
		for slot := range slots {
			commands[slot] = multi
		}
	} else {
		for slot, count := range slots {
			cIndexes[slot] = make([]int, 0, count)
			commands[slot] = make([]Completed, 0, count)
		}
		for i, cmd := range multi {
			slot := cmd.Slot()
			commands[slot] = append(commands[slot], cmd)
			cIndexes[slot] = append(cIndexes[slot], i)
		}
	}

	results = make([]RedisResult, len(multi))
	util.ParallelKeys(commands, func(slot uint16) {
		c.doMulti(ctx, slot, commands[slot], cIndexes[slot], results)
	})

	for i, cmd := range multi {
		if results[i].NonRedisError() == nil {
			cmds.PutCompleted(cmd)
		}
	}
	return results
}

func fillErrs(idx []int, results []RedisResult, err error) {
	if idx == nil {
		for i := range results {
			results[i] = newErrResult(err)
		}
	} else {
		for _, i := range idx {
			results[i] = newErrResult(err)
		}
	}
}

func (c *clusterClient) doMulti(ctx context.Context, slot uint16, multi []Completed, idx []int, results []RedisResult) {
retry:
	cc, err := c.pick(slot)
	if err != nil {
		fillErrs(idx, results, err)
		return
	}
	resps := cc.DoMulti(ctx, multi...)
process:
	for _, resp := range resps {
		switch addr, mode := c.shouldRefreshRetry(resp.Error(), ctx); mode {
		case RedirectMove:
			if c.retry && allReadOnly(multi) {
				resps = c.redirectOrNew(addr).DoMulti(ctx, multi...)
				goto process
			}
		case RedirectAsk:
			if c.retry && allReadOnly(multi) {
				resps = c.redirectOrNew(addr).DoMulti(ctx, append([]Completed{cmds.AskingCmd}, multi...)...)[1:]
				goto process
			}
		case RedirectRetry:
			if c.retry && allReadOnly(multi) {
				runtime.Gosched()
				goto retry
			}
		}
	}
	if idx == nil {
		for i, res := range resps {
			results[i] = res
		}
	} else {
		for i, res := range resps {
			results[idx[i]] = res
		}
	}
}

func (c *clusterClient) doCache(ctx context.Context, cmd Cacheable, ttl time.Duration) (resp RedisResult) {
retry:
	cc, err := c.pick(cmd.Slot())
	if err != nil {
		return newErrResult(err)
	}
	resp = cc.DoCache(ctx, cmd, ttl)
process:
	switch addr, mode := c.shouldRefreshRetry(resp.Error(), ctx); mode {
	case RedirectMove:
		resp = c.redirectOrNew(addr).DoCache(ctx, cmd, ttl)
		goto process
	case RedirectAsk:
		// TODO ASKING OPT-IN Caching
		resp = c.redirectOrNew(addr).DoMulti(ctx, cmds.AskingCmd, Completed(cmd))[1]
		goto process
	case RedirectRetry:
		if c.retry {
			runtime.Gosched()
			goto retry
		}
	}
	return resp
}

func (c *clusterClient) DoCache(ctx context.Context, cmd Cacheable, ttl time.Duration) (resp RedisResult) {
	resp = c.doCache(ctx, cmd, ttl)
	if err := resp.NonRedisError(); err == nil || err == ErrDoCacheAborted {
		cmds.PutCacheable(cmd)
	}
	return resp
}

func (c *clusterClient) doMultiCache(ctx context.Context, slot uint16, multi []CacheableTTL, idx []int, results []RedisResult) {
retry:
	cc, err := c.pick(slot)
	if err != nil {
		fillErrs(idx, results, err)
		return
	}
	resps := cc.DoMultiCache(ctx, multi...)
process:
	for _, resp := range resps {
		switch addr, mode := c.shouldRefreshRetry(resp.Error(), ctx); mode {
		case RedirectMove:
			resps = c.redirectOrNew(addr).DoMultiCache(ctx, multi...)
			goto process
		case RedirectAsk:
			commands := make([]Completed, 0, len(multi)+3)
			commands = append(commands, cmds.AskingCmd, cmds.MultiCmd)
			for _, cmd := range multi {
				commands = append(commands, Completed(cmd.Cmd))
			}
			commands = append(commands, cmds.ExecCmd)
			if asked, err := c.redirectOrNew(addr).DoMulti(ctx, commands...)[len(commands)-1].ToArray(); err != nil {
				for i := range resps {
					resps[i] = newErrResult(err)
				}
			} else {
				for i, ret := range asked {
					resps[i] = newResult(ret, nil)
				}
			}
			goto process
		case RedirectRetry:
			if c.retry {
				runtime.Gosched()
				goto retry
			}
		}
	}
	if idx == nil {
		copy(results, resps)
	} else {
		for i, resp := range resps {
			results[idx[i]] = resp
		}
	}
}

func (c *clusterClient) DoMultiCache(ctx context.Context, multi ...CacheableTTL) (results []RedisResult) {
	if len(multi) == 0 {
		return nil
	}
	slots := make(map[uint16]int, 8)
	for _, cmd := range multi {
		slots[cmd.Cmd.Slot()]++
	}
	commands := make(map[uint16][]CacheableTTL, len(slots))
	cIndexes := make(map[uint16][]int, len(slots))
	if len(slots) == 1 {
		for slot := range slots {
			commands[slot] = multi
		}
	} else {
		for slot, count := range slots {
			cIndexes[slot] = make([]int, 0, count)
			commands[slot] = make([]CacheableTTL, 0, count)
		}
		for i, cmd := range multi {
			slot := cmd.Cmd.Slot()
			commands[slot] = append(commands[slot], cmd)
			cIndexes[slot] = append(cIndexes[slot], i)
		}
	}

	results = make([]RedisResult, len(multi))
	util.ParallelKeys(commands, func(slot uint16) {
		c.doMultiCache(ctx, slot, commands[slot], cIndexes[slot], results)
	})

	for i, cmd := range multi {
		if err := results[i].NonRedisError(); err == nil || err == ErrDoCacheAborted {
			cmds.PutCacheable(cmd.Cmd)
		}
	}
	return results
}

func (c *clusterClient) Receive(ctx context.Context, subscribe Completed, fn func(msg PubSubMessage)) (err error) {
retry:
	cc, err := c.pick(subscribe.Slot())
	if err != nil {
		goto ret
	}
	err = cc.Receive(ctx, subscribe, fn)
	if _, mode := c.shouldRefreshRetry(err, ctx); c.retry && mode != RedirectNone {
		runtime.Gosched()
		goto retry
	}
ret:
	if err == nil {
		cmds.PutCompleted(subscribe)
	}
	return err
}

func (c *clusterClient) Dedicated(fn func(DedicatedClient) error) (err error) {
	dcc := &dedicatedClusterClient{cmd: c.cmd, client: c, slot: cmds.NoSlot, retry: c.retry}
	err = fn(dcc)
	dcc.release()
	return err
}

func (c *clusterClient) Dedicate() (DedicatedClient, func()) {
	dcc := &dedicatedClusterClient{cmd: c.cmd, client: c, slot: cmds.NoSlot, retry: c.retry}
	return dcc, dcc.release
}

func (c *clusterClient) Nodes() map[string]Client {
	c.mu.RLock()
	nodes := make(map[string]Client, len(c.conns))
	for addr, conn := range c.conns {
		nodes[addr] = newSingleClientWithConn(conn, c.cmd, c.retry)
	}
	c.mu.RUnlock()
	return nodes
}

func (c *clusterClient) Close() {
	atomic.StoreUint32(&c.stop, 1)
	c.mu.RLock()
	for _, cc := range c.conns {
		go cc.Close()
	}
	c.mu.RUnlock()
}

func (c *clusterClient) shouldRefreshRetry(err error, ctx context.Context) (addr string, mode RedirectMode) {
	if err != nil && atomic.LoadUint32(&c.stop) == 0 {
		if err, ok := err.(*RedisError); ok {
			if addr, ok = err.IsMoved(); ok {
				mode = RedirectMove
			} else if addr, ok = err.IsAsk(); ok {
				mode = RedirectAsk
			} else if err.IsClusterDown() || err.IsTryAgain() {
				mode = RedirectRetry
			}
		} else {
			mode = RedirectRetry
		}
		if mode != RedirectNone {
			go c.refresh()
		}
		if mode == RedirectRetry && ctx.Err() != nil {
			mode = RedirectNone
		}
	}
	return
}

type dedicatedClusterClient struct {
	client *clusterClient
	conn   conn
	wire   wire
	pshks  *pshks

	mu    sync.Mutex
	cmd   cmds.Builder
	slot  uint16
	mark  bool
	retry bool
}

func (c *dedicatedClusterClient) acquire(slot uint16) (wire wire, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.slot == cmds.NoSlot {
		c.slot = slot
	} else if c.slot != slot && slot != cmds.InitSlot {
		panic(panicMsgCxSlot)
	}
	if c.wire != nil {
		return c.wire, nil
	}
	if c.conn, err = c.client.pick(c.slot); err != nil {
		if p := c.pshks; p != nil {
			c.pshks = nil
			p.close <- err
			close(p.close)
		}
		return nil, err
	}
	c.wire = c.conn.Acquire()
	if p := c.pshks; p != nil {
		c.pshks = nil
		ch := c.wire.SetPubSubHooks(p.hooks)
		go func(ch <-chan error) {
			for e := range ch {
				p.close <- e
			}
			close(p.close)
		}(ch)
	}
	return c.wire, nil
}

func (c *dedicatedClusterClient) release() {
	c.mu.Lock()
	if !c.mark {
		if p := c.pshks; p != nil {
			c.pshks = nil
			close(p.close)
		}
		if c.wire != nil {
			c.conn.Store(c.wire)
		}
	}
	c.mark = true
	c.mu.Unlock()
}

func (c *dedicatedClusterClient) B() cmds.Builder {
	return c.cmd
}

func (c *dedicatedClusterClient) Do(ctx context.Context, cmd Completed) (resp RedisResult) {
retry:
	if w, err := c.acquire(cmd.Slot()); err != nil {
		resp = newErrResult(err)
	} else {
		resp = w.Do(ctx, cmd)
		switch _, mode := c.client.shouldRefreshRetry(resp.Error(), ctx); mode {
		case RedirectRetry:
			if c.retry && cmd.IsReadOnly() && w.Error() == nil {
				runtime.Gosched()
				goto retry
			}
		}
	}
	if resp.NonRedisError() == nil {
		cmds.PutCompleted(cmd)
	}
	return resp
}

func (c *dedicatedClusterClient) DoMulti(ctx context.Context, multi ...Completed) (resp []RedisResult) {
	if len(multi) == 0 {
		return nil
	}
	slot := chooseSlot(multi)
	if slot == cmds.NoSlot {
		panic(panicMsgCxSlot)
	}
	retryable := c.retry
	if retryable {
		retryable = allReadOnly(multi)
	}
retry:
	if w, err := c.acquire(slot); err == nil {
		resp = w.DoMulti(ctx, multi...)
		for _, r := range resp {
			_, mode := c.client.shouldRefreshRetry(r.Error(), ctx)
			if mode == RedirectRetry && retryable && w.Error() == nil {
				runtime.Gosched()
				goto retry
			}
			if mode != RedirectNone {
				break
			}
		}
	} else {
		resp = make([]RedisResult, len(multi))
		for i := range resp {
			resp[i] = newErrResult(err)
		}
	}
	for i, cmd := range multi {
		if resp[i].NonRedisError() == nil {
			cmds.PutCompleted(cmd)
		}
	}
	return resp
}

func (c *dedicatedClusterClient) Receive(ctx context.Context, subscribe Completed, fn func(msg PubSubMessage)) (err error) {
	var w wire
retry:
	if w, err = c.acquire(subscribe.Slot()); err == nil {
		err = w.Receive(ctx, subscribe, fn)
		if _, mode := c.client.shouldRefreshRetry(err, ctx); c.retry && mode == RedirectRetry && w.Error() == nil {
			runtime.Gosched()
			goto retry
		}
	}
	if err == nil {
		cmds.PutCompleted(subscribe)
	}
	return err
}

func (c *dedicatedClusterClient) SetPubSubHooks(hooks PubSubHooks) <-chan error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p := c.pshks; p != nil {
		c.pshks = nil
		close(p.close)
	}
	if c.wire != nil {
		return c.wire.SetPubSubHooks(hooks)
	}
	if hooks.isZero() {
		return nil
	}
	ch := make(chan error, 1)
	c.pshks = &pshks{hooks: hooks, close: ch}
	return ch
}

func (c *dedicatedClusterClient) Close() {
	c.mu.Lock()
	if p := c.pshks; p != nil {
		c.pshks = nil
		p.close <- ErrClosing
		close(p.close)
	}
	if c.wire != nil {
		c.wire.Close()
	}
	c.mu.Unlock()
	c.release()
}

type RedirectMode int

const (
	RedirectNone RedirectMode = iota
	RedirectMove
	RedirectAsk
	RedirectRetry

	panicMsgCxSlot = "cross slot command in Dedicated is prohibited"
	panicMixCxSlot = "Mixing no-slot and cross slot commands in DoMulti is prohibited"
)
