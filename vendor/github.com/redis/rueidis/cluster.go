package rueidis

import (
	"context"
	"errors"
	"net"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/rueidis/internal/cmds"
	"github.com/redis/rueidis/internal/util"
)

// ErrNoSlot indicates that there is no redis node owns the key slot.
var ErrNoSlot = errors.New("the slot has no redis node")

type retry struct {
	cIndexes []int
	commands []Completed
	aIndexes []int
	cAskings []Completed
}

func (r *retry) Capacity() int {
	return cap(r.commands)
}

func (r *retry) ResetLen(n int) {
	r.cIndexes = r.cIndexes[:n]
	r.commands = r.commands[:n]
	r.aIndexes = r.aIndexes[:0]
	r.cAskings = r.cAskings[:0]
}

var retryp = util.NewPool(func(capacity int) *retry {
	return &retry{
		cIndexes: make([]int, 0, capacity),
		commands: make([]Completed, 0, capacity),
	}
})

type retrycache struct {
	cIndexes []int
	commands []CacheableTTL
	aIndexes []int
	cAskings []CacheableTTL
}

func (r *retrycache) Capacity() int {
	return cap(r.commands)
}

func (r *retrycache) ResetLen(n int) {
	r.cIndexes = r.cIndexes[:n]
	r.commands = r.commands[:n]
	r.aIndexes = r.aIndexes[:0]
	r.cAskings = r.cAskings[:0]
}

var retrycachep = util.NewPool(func(capacity int) *retrycache {
	return &retrycache{
		cIndexes: make([]int, 0, capacity),
		commands: make([]CacheableTTL, 0, capacity),
	}
})

type clusterClient struct {
	slots  [16384]conn
	opt    *ClientOption
	conns  map[string]conn
	connFn connFn
	sc     call
	mu     sync.RWMutex
	stop   uint32
	cmd    Builder
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

type clusterslots struct {
	reply RedisResult
	addr  string
}

func (c *clusterClient) _refresh() (err error) {
	var reply RedisMessage
	var addr string

	c.mu.RLock()
	results := make(chan clusterslots, len(c.conns))
	pending := make([]conn, 0, len(c.conns))
	for _, cc := range c.conns {
		pending = append(pending, cc)
	}
	c.mu.RUnlock()

	for i := 0; i < cap(results); i++ {
		if i&3 == 0 { // batch CLUSTER SLOTS for every 4 connections
			for j := i; j < i+4 && j < len(pending); j++ {
				go func(c conn) {
					results <- clusterslots{reply: c.Do(context.Background(), cmds.SlotCmd), addr: c.Addr()}
				}(pending[j])
			}
		}
		r := <-results
		addr = r.addr
		reply, err = r.reply.ToMessage()
		if len(reply.values) != 0 {
			break
		}
	}
	pending = nil

	if err != nil {
		return err
	}

	groups := parseSlots(reply, addr)

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

	if len(removes) > 0 {
		go func(removes []conn) {
			time.Sleep(time.Second * 5)
			for _, cc := range removes {
				cc.Close()
			}
		}(removes)
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

// parseSlots - map redis slots for each redis nodes/addresses
// defaultAddr is needed in case the node does not know its own IP
func parseSlots(slots RedisMessage, defaultAddr string) map[string]group {
	groups := make(map[string]group, len(slots.values))
	for _, v := range slots.values {
		var master string
		switch v.values[2].values[0].string {
		case "":
			master = defaultAddr
		case "?":
			continue
		default:
			master = net.JoinHostPort(v.values[2].values[0].string, strconv.FormatInt(v.values[2].values[1].integer, 10))
		}
		g, ok := groups[master]
		if !ok {
			g.slots = make([][2]int64, 0)
			g.nodes = make([]string, 0, len(v.values)-2)
			for i := 2; i < len(v.values); i++ {
				var dst string
				switch v.values[i].values[0].string {
				case "":
					dst = defaultAddr
				case "?":
					continue
				default:
					dst = net.JoinHostPort(v.values[i].values[0].string, strconv.FormatInt(v.values[i].values[1].integer, 10))
				}
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

func (c *clusterClient) redirectOrNew(addr string, prev conn) (p conn) {
	c.mu.RLock()
	p = c.conns[addr]
	c.mu.RUnlock()
	if p != nil && prev != p {
		return p
	}
	c.mu.Lock()
	if p = c.conns[addr]; p == nil {
		p = c.connFn(addr, c.opt)
		c.conns[addr] = p
	} else if prev == p {
		// try reconnection if the MOVED redirects to the same host,
		// because the same hostname may actually be resolved into another destination
		// depending on the fail-over implementation. ex: AWS MemoryDB's resize process.
		go func(prev conn) {
			time.Sleep(time.Second * 5)
			prev.Close()
		}(prev)
		p = c.connFn(addr, c.opt)
		c.conns[addr] = p
	}
	c.mu.Unlock()
	return p
}

func (c *clusterClient) B() Builder {
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
		resp = c.redirectOrNew(addr, cc).Do(ctx, cmd)
		goto process
	case RedirectAsk:
		results := c.redirectOrNew(addr, cc).DoMulti(ctx, cmds.AskingCmd, cmd)
		resp = results.s[1]
		resultsp.Put(results)
		goto process
	case RedirectRetry:
		if c.retry && cmd.IsReadOnly() {
			runtime.Gosched()
			goto retry
		}
	}
	return resp
}

func (c *clusterClient) _pickMulti(multi []Completed) (retries *connretry, last uint16) {
	last = cmds.InitSlot
	init := false

	c.mu.RLock()
	defer c.mu.RUnlock()

	count := conncountp.Get(len(c.conns), len(c.conns))
	for _, cmd := range multi {
		if cmd.Slot() == cmds.InitSlot {
			init = true
			continue
		}
		p := c.slots[cmd.Slot()]
		if p == nil {
			return nil, 0
		}
		count.m[p]++
	}

	retries = connretryp.Get(len(count.m), len(count.m))
	for cc, n := range count.m {
		retries.m[cc] = retryp.Get(0, n)
	}
	conncountp.Put(count)

	for i, cmd := range multi {
		if cmd.Slot() != cmds.InitSlot {
			if last == cmds.InitSlot {
				last = cmd.Slot()
			} else if init && last != cmd.Slot() {
				panic(panicMixCxSlot)
			}
			cc := c.slots[cmd.Slot()]
			re := retries.m[cc]
			re.commands = append(re.commands, cmd)
			re.cIndexes = append(re.cIndexes, i)
		}
	}

	return retries, last
}

func (c *clusterClient) pickMulti(multi []Completed) (*connretry, uint16, error) {
	conns, slot := c._pickMulti(multi)
	if conns == nil {
		if err := c.refresh(); err != nil {
			return nil, 0, err
		}
		if conns, slot = c._pickMulti(multi); conns == nil {
			return nil, 0, ErrNoSlot
		}
	}
	return conns, slot, nil
}

func (c *clusterClient) doresultfn(ctx context.Context, results *redisresults, retries *connretry, mu *sync.Mutex, cc conn, cIndexes []int, commands []Completed, resps []RedisResult) {
	for i, resp := range resps {
		ii := cIndexes[i]
		cm := commands[i]
		results.s[ii] = resp
		addr, mode := c.shouldRefreshRetry(resp.Error(), ctx)
		if mode != RedirectNone {
			nc := cc
			if mode == RedirectRetry {
				if !c.retry || !cm.IsReadOnly() {
					continue
				}
			} else {
				nc = c.redirectOrNew(addr, cc)
			}
			mu.Lock()
			nr := retries.m[nc]
			if nr == nil {
				nr = retryp.Get(0, len(commands))
				retries.m[nc] = nr
			}
			if mode == RedirectAsk {
				nr.aIndexes = append(nr.aIndexes, ii)
				nr.cAskings = append(nr.cAskings, cm)
			} else {
				nr.cIndexes = append(nr.cIndexes, ii)
				nr.commands = append(nr.commands, cm)
			}
			mu.Unlock()
		}
	}
}

func (c *clusterClient) doretry(ctx context.Context, cc conn, results *redisresults, retries *connretry, re *retry, mu *sync.Mutex, wg *sync.WaitGroup) {
	if len(re.commands) != 0 {
		resps := cc.DoMulti(ctx, re.commands...)
		c.doresultfn(ctx, results, retries, mu, cc, re.cIndexes, re.commands, resps.s)
		resultsp.Put(resps)
	}
	if len(re.cAskings) != 0 {
		resps := askingMulti(cc, ctx, re.cAskings)
		c.doresultfn(ctx, results, retries, mu, cc, re.aIndexes, re.cAskings, resps.s)
		resultsp.Put(resps)
	}
	if ctx.Err() == nil {
		retryp.Put(re)
	}
	wg.Done()
}

func (c *clusterClient) DoMulti(ctx context.Context, multi ...Completed) []RedisResult {
	if len(multi) == 0 {
		return nil
	}

	retries, slot, err := c.pickMulti(multi)
	if err != nil {
		return fillErrs(len(multi), err)
	}
	defer connretryp.Put(retries)

	if len(retries.m) <= 1 {
		for _, re := range retries.m {
			retryp.Put(re)
		}
		return c.doMulti(ctx, slot, multi)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	results := resultsp.Get(len(multi), len(multi))

retry:
	wg.Add(len(retries.m))
	mu.Lock()
	for cc, re := range retries.m {
		delete(retries.m, cc)
		go c.doretry(ctx, cc, results, retries, re, &mu, &wg)
	}
	mu.Unlock()
	wg.Wait()

	if len(retries.m) != 0 {
		goto retry
	}

	for i, cmd := range multi {
		if results.s[i].NonRedisError() == nil {
			cmds.PutCompleted(cmd)
		}
	}
	return results.s
}

func fillErrs(n int, err error) (results []RedisResult) {
	results = resultsp.Get(n, n).s
	for i := range results {
		results[i] = newErrResult(err)
	}
	return results
}

func (c *clusterClient) doMulti(ctx context.Context, slot uint16, multi []Completed) []RedisResult {
retry:
	cc, err := c.pick(slot)
	if err != nil {
		return fillErrs(len(multi), err)
	}
	resps := cc.DoMulti(ctx, multi...)
process:
	for _, resp := range resps.s {
		switch addr, mode := c.shouldRefreshRetry(resp.Error(), ctx); mode {
		case RedirectMove:
			if c.retry && allReadOnly(multi) {
				resultsp.Put(resps)
				resps = c.redirectOrNew(addr, cc).DoMulti(ctx, multi...)
				goto process
			}
		case RedirectAsk:
			if c.retry && allReadOnly(multi) {
				resultsp.Put(resps)
				resps = askingMulti(c.redirectOrNew(addr, cc), ctx, multi)
				goto process
			}
		case RedirectRetry:
			if c.retry && allReadOnly(multi) {
				resultsp.Put(resps)
				runtime.Gosched()
				goto retry
			}
		}
	}
	return resps.s
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
		resp = c.redirectOrNew(addr, cc).DoCache(ctx, cmd, ttl)
		goto process
	case RedirectAsk:
		results := askingMultiCache(c.redirectOrNew(addr, cc), ctx, []CacheableTTL{CT(cmd, ttl)})
		resp = results.s[0]
		resultsp.Put(results)
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

func askingMulti(cc conn, ctx context.Context, multi []Completed) *redisresults {
	commands := make([]Completed, 0, len(multi)*2)
	for _, cmd := range multi {
		commands = append(commands, cmds.AskingCmd, cmd)
	}
	results := resultsp.Get(0, len(multi))
	resps := cc.DoMulti(ctx, commands...)
	for i := 1; i < len(resps.s); i += 2 {
		results.s = append(results.s, resps.s[i])
	}
	resultsp.Put(resps)
	return results
}

func askingMultiCache(cc conn, ctx context.Context, multi []CacheableTTL) *redisresults {
	commands := make([]Completed, 0, len(multi)*6)
	for _, cmd := range multi {
		ck, _ := cmds.CacheKey(cmd.Cmd)
		commands = append(commands, cmds.OptInCmd, cmds.MultiCmd, cmds.AskingCmd, cmds.NewCompleted([]string{"PTTL", ck}), Completed(cmd.Cmd), cmds.ExecCmd)
	}
	results := resultsp.Get(0, len(multi))
	resps := cc.DoMulti(ctx, commands...)
	for i := 5; i < len(resps.s); i += 6 {
		if arr, err := resps.s[i].ToArray(); err != nil {
			results.s = append(results.s, newErrResult(err))
		} else {
			results.s = append(results.s, newResult(arr[len(arr)-1], nil))
		}
	}
	resultsp.Put(resps)
	return results
}

func (c *clusterClient) _pickMultiCache(multi []CacheableTTL) *connretrycache {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := conncountp.Get(len(c.conns), len(c.conns))
	for _, cmd := range multi {
		p := c.slots[cmd.Cmd.Slot()]
		if p == nil {
			return nil
		}
		count.m[p]++
	}

	retries := connretrycachep.Get(len(count.m), len(count.m))
	for cc, n := range count.m {
		retries.m[cc] = retrycachep.Get(0, n)
	}
	conncountp.Put(count)

	for i, cmd := range multi {
		cc := c.slots[cmd.Cmd.Slot()]
		re := retries.m[cc]
		re.commands = append(re.commands, cmd)
		re.cIndexes = append(re.cIndexes, i)
	}

	return retries
}

func (c *clusterClient) pickMultiCache(multi []CacheableTTL) (*connretrycache, error) {
	conns := c._pickMultiCache(multi)
	if conns == nil {
		if err := c.refresh(); err != nil {
			return nil, err
		}
		if conns = c._pickMultiCache(multi); conns == nil {
			return nil, ErrNoSlot
		}
	}
	return conns, nil
}

func (c *clusterClient) resultcachefn(ctx context.Context, results *redisresults, retries *connretrycache, mu *sync.Mutex, cc conn, cIndexes []int, commands []CacheableTTL, resps []RedisResult) {
	for i, resp := range resps {
		ii := cIndexes[i]
		cm := commands[i]
		results.s[ii] = resp
		addr, mode := c.shouldRefreshRetry(resp.Error(), ctx)
		if mode != RedirectNone {
			nc := cc
			if mode == RedirectRetry {
				if !c.retry {
					continue
				}
			} else {
				nc = c.redirectOrNew(addr, cc)
			}
			mu.Lock()
			nr := retries.m[nc]
			if nr == nil {
				nr = retrycachep.Get(0, len(commands))
				retries.m[nc] = nr
			}
			if mode == RedirectAsk {
				nr.aIndexes = append(nr.aIndexes, ii)
				nr.cAskings = append(nr.cAskings, cm)
			} else {
				nr.cIndexes = append(nr.cIndexes, ii)
				nr.commands = append(nr.commands, cm)
			}
			mu.Unlock()
		}
	}
}

func (c *clusterClient) doretrycache(ctx context.Context, cc conn, results *redisresults, retries *connretrycache, re *retrycache, mu *sync.Mutex, wg *sync.WaitGroup) {
	if len(re.commands) != 0 {
		resps := cc.DoMultiCache(ctx, re.commands...)
		c.resultcachefn(ctx, results, retries, mu, cc, re.cIndexes, re.commands, resps.s)
		resultsp.Put(resps)
	}
	if len(re.cAskings) != 0 {
		resps := askingMultiCache(cc, ctx, re.cAskings)
		c.resultcachefn(ctx, results, retries, mu, cc, re.aIndexes, re.cAskings, resps.s)
		resultsp.Put(resps)
	}
	if ctx.Err() == nil {
		retrycachep.Put(re)
	}
	wg.Done()
}

func (c *clusterClient) DoMultiCache(ctx context.Context, multi ...CacheableTTL) []RedisResult {
	if len(multi) == 0 {
		return nil
	}

	retries, err := c.pickMultiCache(multi)
	if err != nil {
		return fillErrs(len(multi), err)
	}
	defer connretrycachep.Put(retries)

	var wg sync.WaitGroup
	var mu sync.Mutex

	results := resultsp.Get(len(multi), len(multi))

retry:
	wg.Add(len(retries.m))
	mu.Lock()
	for cc, re := range retries.m {
		delete(retries.m, cc)
		go c.doretrycache(ctx, cc, results, retries, re, &mu, &wg)
	}
	mu.Unlock()
	wg.Wait()

	if len(retries.m) != 0 {
		goto retry
	}

	for i, cmd := range multi {
		if err := results.s[i].NonRedisError(); err == nil || err == ErrDoCacheAborted {
			cmds.PutCacheable(cmd.Cmd)
		}
	}
	return results.s
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
		} else if ctx.Err() == nil {
			mode = RedirectRetry
		}
		if mode != RedirectNone {
			go c.refresh()
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
	cmd   Builder
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

func (c *dedicatedClusterClient) B() Builder {
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
		resp = w.DoMulti(ctx, multi...).s
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
		resp = resultsp.Get(len(multi), len(multi)).s
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

type conncount struct {
	m map[conn]int
	n int
}

func (r *conncount) Capacity() int {
	return r.n
}

func (r *conncount) ResetLen(n int) {
	for k := range r.m {
		delete(r.m, k)
	}
}

var conncountp = util.NewPool(func(capacity int) *conncount {
	return &conncount{m: make(map[conn]int, capacity), n: capacity}
})

type connretry struct {
	m map[conn]*retry
	n int
}

func (r *connretry) Capacity() int {
	return r.n
}

func (r *connretry) ResetLen(n int) {
	for k := range r.m {
		delete(r.m, k)
	}
}

var connretryp = util.NewPool(func(capacity int) *connretry {
	return &connretry{m: make(map[conn]*retry, capacity), n: capacity}
})

type connretrycache struct {
	m map[conn]*retrycache
	n int
}

func (r *connretrycache) Capacity() int {
	return r.n
}

func (r *connretrycache) ResetLen(n int) {
	for k := range r.m {
		delete(r.m, k)
	}
}

var connretrycachep = util.NewPool(func(capacity int) *connretrycache {
	return &connretrycache{m: make(map[conn]*retrycache, capacity), n: capacity}
})
