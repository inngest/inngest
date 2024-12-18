package rueidis

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/rueidis/internal/cmds"
	"github.com/redis/rueidis/internal/util"
)

// ErrNoSlot indicates that there is no redis node owns the key slot.
var ErrNoSlot = errors.New("the slot has no redis node")
var ErrReplicaOnlyConflict = errors.New("ReplicaOnly conflicts with SendToReplicas option")
var ErrInvalidShardsRefreshInterval = errors.New("ShardsRefreshInterval must be greater than or equal to 0")

type clusterClient struct {
	pslots       [16384]conn
	rslots       []conn
	sc           call
	rOpt         *ClientOption
	conns        map[string]connrole
	connFn       connFn
	opt          *ClientOption
	retryHandler retryHandler
	stopCh       chan struct{}
	cmd          Builder
	mu           sync.RWMutex
	stop         uint32
	retry        bool
}

// NOTE: connrole and conn must be initialized at the same time
type connrole struct {
	conn    conn
	replica bool
}

func newClusterClient(opt *ClientOption, connFn connFn, retryer retryHandler) (*clusterClient, error) {
	client := &clusterClient{
		cmd:          cmds.NewBuilder(cmds.InitSlot),
		connFn:       connFn,
		opt:          opt,
		conns:        make(map[string]connrole),
		retry:        !opt.DisableRetry,
		retryHandler: retryer,
		stopCh:       make(chan struct{}),
	}

	if opt.ReplicaOnly && opt.SendToReplicas != nil {
		return nil, ErrReplicaOnlyConflict
	}

	if opt.SendToReplicas != nil {
		rOpt := *opt
		rOpt.ReplicaOnly = true
		client.rOpt = &rOpt
	}

	client.connFn = func(dst string, opt *ClientOption) conn {
		cc := connFn(dst, opt)
		cc.SetOnCloseHook(func(err error) {
			client.lazyRefresh()
		})
		return cc
	}

	if err := client.init(); err != nil {
		return nil, err
	}

	if err := client.refresh(context.Background()); err != nil {
		return client, err
	}

	if opt.ClusterOption.ShardsRefreshInterval > 0 {
		go client.runClusterTopologyRefreshment()
	} else if opt.ClusterOption.ShardsRefreshInterval < 0 {
		return nil, ErrInvalidShardsRefreshInterval
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
					c.conns[addr] = connrole{
						conn: cc,
					}
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

func (c *clusterClient) refresh(ctx context.Context) (err error) {
	return c.sc.Do(ctx, c._refresh)
}

func (c *clusterClient) lazyRefresh() {
	c.sc.LazyDo(time.Second, c._refresh)
}

type clusterslots struct {
	addr  string
	reply RedisResult
	ver   int
}

func (s clusterslots) parse(tls bool) map[string]group {
	if s.ver < 8 {
		return parseSlots(s.reply.val, s.addr)
	}
	return parseShards(s.reply.val, s.addr, tls)
}

func getClusterSlots(c conn, timeout time.Duration) clusterslots {
	var ctx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}
	v := c.Version()
	if v < 8 {
		return clusterslots{reply: c.Do(ctx, cmds.SlotCmd), addr: c.Addr(), ver: v}
	}
	return clusterslots{reply: c.Do(ctx, cmds.ShardsCmd), addr: c.Addr(), ver: v}
}

func (c *clusterClient) _refresh() (err error) {
	c.mu.RLock()
	results := make(chan clusterslots, len(c.conns))
	pending := make([]conn, 0, len(c.conns))
	for _, cc := range c.conns {
		pending = append(pending, cc.conn)
	}
	c.mu.RUnlock()

	var result clusterslots
	for i := 0; i < cap(results); i++ {
		if i&3 == 0 { // batch CLUSTER SLOTS/CLUSTER SHARDS for every 4 connections
			for j := i; j < i+4 && j < len(pending); j++ {
				go func(c conn, timeout time.Duration) {
					results <- getClusterSlots(c, timeout)
				}(pending[j], c.opt.ConnWriteTimeout)
			}
		}
		result = <-results
		err = result.reply.Error()
		if len(result.reply.val.values) != 0 {
			break
		}
	}
	if err != nil {
		return err
	}
	pending = nil

	groups := result.parse(c.opt.TLSConfig != nil)
	conns := make(map[string]connrole, len(groups))
	for master, g := range groups {
		conns[master] = connrole{conn: c.connFn(master, c.opt), replica: false}
		for _, addr := range g.nodes[1:] {
			if c.rOpt != nil {
				conns[addr] = connrole{conn: c.connFn(addr, c.rOpt), replica: true}
			} else {
				conns[addr] = connrole{conn: c.connFn(addr, c.opt), replica: true}
			}
		}
	}
	// make sure InitAddress always be present
	for _, addr := range c.opt.InitAddress {
		if _, ok := conns[addr]; !ok {
			conns[addr] = connrole{
				conn: c.connFn(addr, c.opt),
			}
		}
	}

	var removes []conn

	c.mu.RLock()
	for addr, cc := range c.conns {
		fresh, ok := conns[addr]
		if ok && (cc.replica == fresh.replica || c.rOpt == nil) {
			conns[addr] = connrole{
				conn:    cc.conn,
				replica: fresh.replica,
			}
		} else {
			removes = append(removes, cc.conn)
		}
	}
	c.mu.RUnlock()

	pslots := [16384]conn{}
	var rslots []conn
	for master, g := range groups {
		switch {
		case c.opt.ReplicaOnly && len(g.nodes) > 1:
			nodesCount := len(g.nodes)
			for _, slot := range g.slots {
				for i := slot[0]; i <= slot[1]; i++ {
					pslots[i] = conns[g.nodes[1+util.FastRand(nodesCount-1)]].conn
				}
			}
		case c.rOpt != nil: // implies c.opt.SendToReplicas != nil
			if len(rslots) == 0 { // lazy init
				rslots = make([]conn, 16384)
			}
			if len(g.nodes) > 1 {
				for _, slot := range g.slots {
					for i := slot[0]; i <= slot[1]; i++ {
						pslots[i] = conns[master].conn
						rslots[i] = conns[g.nodes[1+util.FastRand(len(g.nodes)-1)]].conn
					}
				}
			} else {
				for _, slot := range g.slots {
					for i := slot[0]; i <= slot[1]; i++ {
						pslots[i] = conns[master].conn
						rslots[i] = conns[master].conn
					}
				}
			}
		default:
			for _, slot := range g.slots {
				for i := slot[0]; i <= slot[1]; i++ {
					pslots[i] = conns[master].conn
				}
			}
		}
	}

	c.mu.Lock()
	c.pslots = pslots
	c.rslots = rslots
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

func (c *clusterClient) single() (conn conn) {
	return c._pick(cmds.InitSlot, false)
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

func parseEndpoint(fallback, endpoint string, port int64) string {
	switch endpoint {
	case "":
		endpoint, _, _ = net.SplitHostPort(fallback)
	case "?":
		return ""
	}
	return net.JoinHostPort(endpoint, strconv.FormatInt(port, 10))
}

// parseSlots - map redis slots for each redis nodes/addresses
// defaultAddr is needed in case the node does not know its own IP
func parseSlots(slots RedisMessage, defaultAddr string) map[string]group {
	groups := make(map[string]group, len(slots.values))
	for _, v := range slots.values {
		master := parseEndpoint(defaultAddr, v.values[2].values[0].string, v.values[2].values[1].integer)
		if master == "" {
			continue
		}
		g, ok := groups[master]
		if !ok {
			g.slots = make([][2]int64, 0)
			g.nodes = make([]string, 0, len(v.values)-2)
			for i := 2; i < len(v.values); i++ {
				if dst := parseEndpoint(defaultAddr, v.values[i].values[0].string, v.values[i].values[1].integer); dst != "" {
					g.nodes = append(g.nodes, dst)
				}
			}
		}
		g.slots = append(g.slots, [2]int64{v.values[0].integer, v.values[1].integer})
		groups[master] = g
	}
	return groups
}

// parseShards - map redis shards for each redis nodes/addresses
// defaultAddr is needed in case the node does not know its own IP
func parseShards(shards RedisMessage, defaultAddr string, tls bool) map[string]group {
	groups := make(map[string]group, len(shards.values))
	for _, v := range shards.values {
		m := -1
		shard, _ := v.AsMap()
		slots := shard["slots"].values
		nodes := shard["nodes"].values
		g := group{
			nodes: make([]string, 0, len(nodes)),
			slots: make([][2]int64, len(slots)/2),
		}
		for i := range g.slots {
			g.slots[i][0], _ = slots[i*2].AsInt64()
			g.slots[i][1], _ = slots[i*2+1].AsInt64()
		}
		for _, n := range nodes {
			dict, _ := n.AsMap()
			if dict["health"].string == "fail" {
				continue
			}
			port := dict["port"].integer
			if tls && dict["tls-port"].integer > 0 {
				port = dict["tls-port"].integer
			}
			if dst := parseEndpoint(defaultAddr, dict["endpoint"].string, port); dst != "" {
				if dict["role"].string == "master" && dict["health"].string == "online" {
					m = len(g.nodes)
				}
				g.nodes = append(g.nodes, dst)
			}
		}
		if m >= 0 {
			g.nodes[0], g.nodes[m] = g.nodes[m], g.nodes[0]
			groups[g.nodes[0]] = g
		}
	}
	return groups
}

func (c *clusterClient) runClusterTopologyRefreshment() {
	ticker := time.NewTicker(c.opt.ClusterOption.ShardsRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.lazyRefresh()
		}
	}
}

func (c *clusterClient) _pick(slot uint16, toReplica bool) (p conn) {
	c.mu.RLock()
	if slot == cmds.InitSlot {
		for _, cc := range c.conns {
			if cc.replica {
				continue
			}
			p = cc.conn
			break
		}
	} else if toReplica && c.rslots != nil {
		p = c.rslots[slot]
	} else {
		p = c.pslots[slot]
	}
	c.mu.RUnlock()
	return p
}

func (c *clusterClient) pick(ctx context.Context, slot uint16, toReplica bool) (p conn, err error) {
	if p = c._pick(slot, toReplica); p == nil {
		if err := c.refresh(ctx); err != nil {
			return nil, err
		}
		if p = c._pick(slot, toReplica); p == nil {
			return nil, ErrNoSlot
		}
	}
	return p, nil
}

func (c *clusterClient) redirectOrNew(addr string, prev conn, slot uint16, mode RedirectMode) conn {
	c.mu.RLock()
	cc := c.conns[addr]
	c.mu.RUnlock()
	if cc.conn != nil && prev != cc.conn {
		return cc.conn
	}
	c.mu.Lock()
	if cc = c.conns[addr]; cc.conn == nil {
		p := c.connFn(addr, c.opt)
		cc = connrole{conn: p, replica: false}
		c.conns[addr] = cc
		if mode == RedirectMove {
			c.pslots[slot] = p
		}
	} else if prev == cc.conn {
		// try reconnection if the MOVED redirects to the same host,
		// because the same hostname may actually be resolved into another destination
		// depending on the fail-over implementation. ex: AWS MemoryDB's resize process.
		go func(prev conn) {
			time.Sleep(time.Second * 5)
			prev.Close()
		}(prev)
		p := c.connFn(addr, c.opt)
		cc = connrole{conn: p, replica: cc.replica}
		c.conns[addr] = cc
		if mode == RedirectMove {
			if cc.replica {
				c.rslots[slot] = p
			} else {
				c.pslots[slot] = p
			}
		}
	}
	c.mu.Unlock()
	return cc.conn
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
	attempts := 1
retry:
	cc, err := c.pick(ctx, cmd.Slot(), c.toReplica(cmd))
	if err != nil {
		return newErrResult(err)
	}
	resp = cc.Do(ctx, cmd)
process:
	switch addr, mode := c.shouldRefreshRetry(resp.Error(), ctx); mode {
	case RedirectMove:
		resp = c.redirectOrNew(addr, cc, cmd.Slot(), mode).Do(ctx, cmd)
		goto process
	case RedirectAsk:
		results := c.redirectOrNew(addr, cc, cmd.Slot(), mode).DoMulti(ctx, cmds.AskingCmd, cmd)
		resp = results.s[1]
		resultsp.Put(results)
		goto process
	case RedirectRetry:
		if c.retry && cmd.IsReadOnly() {
			shouldRetry := c.retryHandler.WaitOrSkipRetry(ctx, attempts, cmd, resp.Error())
			if shouldRetry {
				attempts++
				goto retry
			}
		}
	}
	return resp
}

func (c *clusterClient) toReplica(cmd Completed) bool {
	if c.opt.SendToReplicas != nil {
		return c.opt.SendToReplicas(cmd)
	}
	return false
}

func (c *clusterClient) _pickMulti(multi []Completed) (retries *connretry, last uint16, toReplica bool) {
	last = cmds.InitSlot
	init := false

	for _, cmd := range multi {
		if cmd.Slot() == cmds.InitSlot {
			init = true
			break
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	count := conncountp.Get(len(c.conns), len(c.conns))

	if !init && c.rslots != nil && c.opt.SendToReplicas != nil {
		for _, cmd := range multi {
			var p conn
			if c.opt.SendToReplicas(cmd) {
				p = c.rslots[cmd.Slot()]
			} else {
				p = c.pslots[cmd.Slot()]
			}
			if p == nil {
				return nil, 0, false
			}
			count.m[p]++
		}

		retries = connretryp.Get(len(count.m), len(count.m))
		for cc, n := range count.m {
			retries.m[cc] = retryp.Get(0, n)
		}
		conncountp.Put(count)

		for i, cmd := range multi {
			last = cmd.Slot()

			var cc conn
			if c.opt.SendToReplicas(cmd) {
				toReplica = true
				cc = c.rslots[cmd.Slot()]
			} else {
				cc = c.pslots[cmd.Slot()]
			}

			re := retries.m[cc]
			re.commands = append(re.commands, cmd)
			re.cIndexes = append(re.cIndexes, i)
		}
		return retries, last, toReplica
	}

	for _, cmd := range multi {
		if cmd.Slot() == cmds.InitSlot {
			continue
		}
		p := c.pslots[cmd.Slot()]
		if p == nil {
			return nil, 0, false
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
			cc := c.pslots[cmd.Slot()]
			re := retries.m[cc]
			re.commands = append(re.commands, cmd)
			re.cIndexes = append(re.cIndexes, i)
		}
	}
	return retries, last, false
}

func (c *clusterClient) pickMulti(ctx context.Context, multi []Completed) (*connretry, uint16, bool, error) {
	conns, slot, toReplica := c._pickMulti(multi)
	if conns == nil {
		if err := c.refresh(ctx); err != nil {
			return nil, 0, false, err
		}
		if conns, slot, toReplica = c._pickMulti(multi); conns == nil {
			return nil, 0, false, ErrNoSlot
		}
	}
	return conns, slot, toReplica, nil
}

func (c *clusterClient) doresultfn(
	ctx context.Context, results *redisresults, retries *connretry, mu *sync.Mutex, cc conn, cIndexes []int, commands []Completed, resps []RedisResult, attempts int,
) {
	for i, resp := range resps {
		ii := cIndexes[i]
		cm := commands[i]
		results.s[ii] = resp
		addr, mode := c.shouldRefreshRetry(resp.Error(), ctx)
		if mode != RedirectNone {
			nc := cc
			retryDelay := time.Duration(-1)
			if mode == RedirectRetry {
				if !c.retry || !cm.IsReadOnly() {
					continue
				}

				retryDelay = c.retryHandler.RetryDelay(attempts, cm, resp.Error())
			} else {
				nc = c.redirectOrNew(addr, cc, cm.Slot(), mode)
			}
			mu.Lock()
			if mode != RedirectRetry {
				retries.Redirects++
			}
			if mode == RedirectRetry && retryDelay >= 0 {
				retries.RetryDelay = max(retries.RetryDelay, retryDelay)
			}
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

func (c *clusterClient) doretry(
	ctx context.Context, cc conn, results *redisresults, retries *connretry, re *retry, mu *sync.Mutex, wg *sync.WaitGroup, attempts int,
) {
	if len(re.commands) != 0 {
		resps := cc.DoMulti(ctx, re.commands...)
		c.doresultfn(ctx, results, retries, mu, cc, re.cIndexes, re.commands, resps.s, attempts)
		resultsp.Put(resps)
	}
	if len(re.cAskings) != 0 {
		resps := askingMulti(cc, ctx, re.cAskings)
		c.doresultfn(ctx, results, retries, mu, cc, re.aIndexes, re.cAskings, resps.s, attempts)
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

	retries, slot, toReplica, err := c.pickMulti(ctx, multi)
	if err != nil {
		return fillErrs(len(multi), err)
	}
	defer connretryp.Put(retries)

	if len(retries.m) <= 1 {
		for _, re := range retries.m {
			retryp.Put(re)
		}
		return c.doMulti(ctx, slot, multi, toReplica)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	results := resultsp.Get(len(multi), len(multi))

	attempts := 1

retry:
	retries.RetryDelay = -1 // Assume no retry. Because client retry flag can be set to false.

	wg.Add(len(retries.m))
	mu.Lock()
	for cc, re := range retries.m {
		delete(retries.m, cc)
		go c.doretry(ctx, cc, results, retries, re, &mu, &wg, attempts)
	}
	mu.Unlock()
	wg.Wait()

	if len(retries.m) != 0 {
		if retries.Redirects > 0 {
			retries.Redirects = 0
			goto retry
		}

		if retries.RetryDelay >= 0 {
			c.retryHandler.WaitForRetry(ctx, retries.RetryDelay)
			attempts++
			goto retry
		}
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

func (c *clusterClient) doMulti(ctx context.Context, slot uint16, multi []Completed, toReplica bool) []RedisResult {
	attempts := 1

retry:
	cc, err := c.pick(ctx, slot, toReplica)
	if err != nil {
		return fillErrs(len(multi), err)
	}
	resps := cc.DoMulti(ctx, multi...)
process:
	for i, resp := range resps.s {
		switch addr, mode := c.shouldRefreshRetry(resp.Error(), ctx); mode {
		case RedirectMove:
			if c.retry && allReadOnly(multi) {
				resultsp.Put(resps)
				resps = c.redirectOrNew(addr, cc, multi[i].Slot(), mode).DoMulti(ctx, multi...)
				goto process
			}
		case RedirectAsk:
			if c.retry && allReadOnly(multi) {
				resultsp.Put(resps)
				resps = askingMulti(c.redirectOrNew(addr, cc, multi[i].Slot(), mode), ctx, multi)
				goto process
			}
		case RedirectRetry:
			if c.retry && allReadOnly(multi) {
				shouldRetry := c.retryHandler.WaitOrSkipRetry(ctx, attempts, multi[i], resp.Error())
				if shouldRetry {
					resultsp.Put(resps)
					attempts++
					goto retry
				}
			}
		}
	}
	return resps.s
}

func (c *clusterClient) doCache(ctx context.Context, cmd Cacheable, ttl time.Duration) (resp RedisResult) {
	attempts := 1

retry:
	cc, err := c.pick(ctx, cmd.Slot(), c.toReplica(Completed(cmd)))
	if err != nil {
		return newErrResult(err)
	}
	resp = cc.DoCache(ctx, cmd, ttl)
process:
	switch addr, mode := c.shouldRefreshRetry(resp.Error(), ctx); mode {
	case RedirectMove:
		resp = c.redirectOrNew(addr, cc, cmd.Slot(), mode).DoCache(ctx, cmd, ttl)
		goto process
	case RedirectAsk:
		results := askingMultiCache(c.redirectOrNew(addr, cc, cmd.Slot(), mode), ctx, []CacheableTTL{CT(cmd, ttl)})
		resp = results.s[0]
		resultsp.Put(results)
		goto process
	case RedirectRetry:
		if c.retry {
			shouldRetry := c.retryHandler.WaitOrSkipRetry(ctx, attempts, Completed(cmd), resp.Error())
			if shouldRetry {
				attempts++
				goto retry
			}
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
	if c.opt.SendToReplicas == nil || c.rslots == nil {
		for _, cmd := range multi {
			p := c.pslots[cmd.Cmd.Slot()]
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
			cc := c.pslots[cmd.Cmd.Slot()]
			re := retries.m[cc]
			re.commands = append(re.commands, cmd)
			re.cIndexes = append(re.cIndexes, i)
		}

		return retries
	} else {
		for _, cmd := range multi {
			var p conn
			if c.opt.SendToReplicas(Completed(cmd.Cmd)) {
				p = c.rslots[cmd.Cmd.Slot()]
			} else {
				p = c.pslots[cmd.Cmd.Slot()]
			}
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
			var cc conn
			if c.opt.SendToReplicas(Completed(cmd.Cmd)) {
				cc = c.rslots[cmd.Cmd.Slot()]
			} else {
				cc = c.pslots[cmd.Cmd.Slot()]
			}
			re := retries.m[cc]
			re.commands = append(re.commands, cmd)
			re.cIndexes = append(re.cIndexes, i)
		}

		return retries
	}
}

func (c *clusterClient) pickMultiCache(ctx context.Context, multi []CacheableTTL) (*connretrycache, error) {
	conns := c._pickMultiCache(multi)
	if conns == nil {
		if err := c.refresh(ctx); err != nil {
			return nil, err
		}
		if conns = c._pickMultiCache(multi); conns == nil {
			return nil, ErrNoSlot
		}
	}
	return conns, nil
}

func (c *clusterClient) resultcachefn(
	ctx context.Context, results *redisresults, retries *connretrycache, mu *sync.Mutex, cc conn, cIndexes []int, commands []CacheableTTL, resps []RedisResult, attempts int,
) {
	for i, resp := range resps {
		ii := cIndexes[i]
		cm := commands[i]
		results.s[ii] = resp
		addr, mode := c.shouldRefreshRetry(resp.Error(), ctx)
		if mode != RedirectNone {
			nc := cc
			retryDelay := time.Duration(-1)
			if mode == RedirectRetry {
				if !c.retry {
					continue
				}

				retryDelay = c.retryHandler.RetryDelay(attempts, Completed(cm.Cmd), resp.Error())
			} else {
				nc = c.redirectOrNew(addr, cc, cm.Cmd.Slot(), mode)
			}
			mu.Lock()
			if mode != RedirectRetry {
				retries.Redirects++
			}
			if mode == RedirectRetry && retryDelay >= 0 {
				retries.RetryDelay = max(retries.RetryDelay, retryDelay)
			}
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

func (c *clusterClient) doretrycache(
	ctx context.Context, cc conn, results *redisresults, retries *connretrycache, re *retrycache, mu *sync.Mutex, wg *sync.WaitGroup, attempts int,
) {
	if len(re.commands) != 0 {
		resps := cc.DoMultiCache(ctx, re.commands...)
		c.resultcachefn(ctx, results, retries, mu, cc, re.cIndexes, re.commands, resps.s, attempts)
		resultsp.Put(resps)
	}
	if len(re.cAskings) != 0 {
		resps := askingMultiCache(cc, ctx, re.cAskings)
		c.resultcachefn(ctx, results, retries, mu, cc, re.aIndexes, re.cAskings, resps.s, attempts)
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

	retries, err := c.pickMultiCache(ctx, multi)
	if err != nil {
		return fillErrs(len(multi), err)
	}
	defer connretrycachep.Put(retries)

	var wg sync.WaitGroup
	var mu sync.Mutex

	results := resultsp.Get(len(multi), len(multi))

	attempts := 1

retry:
	retries.RetryDelay = -1 // Assume no retry. Because client retry flag can be set to false.

	wg.Add(len(retries.m))
	mu.Lock()
	for cc, re := range retries.m {
		delete(retries.m, cc)
		go c.doretrycache(ctx, cc, results, retries, re, &mu, &wg, attempts)
	}
	mu.Unlock()
	wg.Wait()

	if len(retries.m) != 0 {
		if retries.Redirects > 0 {
			retries.Redirects = 0
			goto retry
		}

		if retries.RetryDelay >= 0 {
			c.retryHandler.WaitForRetry(ctx, retries.RetryDelay)
			attempts++
			goto retry
		}
	}

	for i, cmd := range multi {
		if err := results.s[i].NonRedisError(); err == nil || err == ErrDoCacheAborted {
			cmds.PutCacheable(cmd.Cmd)
		}
	}
	return results.s
}

func (c *clusterClient) Receive(ctx context.Context, subscribe Completed, fn func(msg PubSubMessage)) (err error) {
	attempts := 1
retry:
	cc, err := c.pick(ctx, subscribe.Slot(), c.toReplica(subscribe))
	if err != nil {
		goto ret
	}
	err = cc.Receive(ctx, subscribe, fn)
	if _, mode := c.shouldRefreshRetry(err, ctx); c.retry && mode != RedirectNone {
		shouldRetry := c.retryHandler.WaitOrSkipRetry(ctx, attempts, subscribe, err)
		if shouldRetry {
			attempts++
			goto retry
		}
	}
ret:
	if err == nil {
		cmds.PutCompleted(subscribe)
	}
	return err
}

func (c *clusterClient) DoStream(ctx context.Context, cmd Completed) RedisResultStream {
	cc, err := c.pick(ctx, cmd.Slot(), c.toReplica(cmd))
	if err != nil {
		return RedisResultStream{e: err}
	}
	ret := cc.DoStream(ctx, cmd)
	cmds.PutCompleted(cmd)
	return ret
}

func (c *clusterClient) DoMultiStream(ctx context.Context, multi ...Completed) MultiRedisResultStream {
	if len(multi) == 0 {
		return RedisResultStream{e: io.EOF}
	}
	slot := multi[0].Slot()
	repl := c.toReplica(multi[0])
	for i := 1; i < len(multi); i++ {
		if s := multi[i].Slot(); s != cmds.InitSlot {
			if slot == cmds.InitSlot {
				slot = s
			} else if slot != s {
				panic("DoMultiStream across multiple slots is not supported")
			}
		}
		repl = repl && c.toReplica(multi[i])
	}
	cc, err := c.pick(ctx, slot, repl)
	if err != nil {
		return RedisResultStream{e: err}
	}
	ret := cc.DoMultiStream(ctx, multi...)
	for _, cmd := range multi {
		cmds.PutCompleted(cmd)
	}
	return ret
}

func (c *clusterClient) Dedicated(fn func(DedicatedClient) error) (err error) {
	dcc := &dedicatedClusterClient{cmd: c.cmd, client: c, slot: cmds.NoSlot, retry: c.retry, retryHandler: c.retryHandler}
	err = fn(dcc)
	dcc.release()
	return err
}

func (c *clusterClient) Dedicate() (DedicatedClient, func()) {
	dcc := &dedicatedClusterClient{cmd: c.cmd, client: c, slot: cmds.NoSlot, retry: c.retry, retryHandler: c.retryHandler}
	return dcc, dcc.release
}

func (c *clusterClient) Nodes() map[string]Client {
	c.mu.RLock()
	nodes := make(map[string]Client, len(c.conns))
	disableCache := c.opt != nil && c.opt.DisableCache
	for addr, cc := range c.conns {
		nodes[addr] = newSingleClientWithConn(cc.conn, c.cmd, c.retry, disableCache, c.retryHandler)
	}
	c.mu.RUnlock()
	return nodes
}

func (c *clusterClient) Close() {
	if atomic.CompareAndSwapUint32(&c.stop, 0, 1) {
		close(c.stopCh)
	}

	c.mu.RLock()
	for _, cc := range c.conns {
		go cc.conn.Close()
	}
	c.mu.RUnlock()
}

func (c *clusterClient) shouldRefreshRetry(err error, ctx context.Context) (addr string, mode RedirectMode) {
	if err != nil && err != Nil && err != ErrDoCacheAborted && atomic.LoadUint32(&c.stop) == 0 {
		if err, ok := err.(*RedisError); ok {
			if addr, ok = err.IsMoved(); ok {
				mode = RedirectMove
			} else if addr, ok = err.IsAsk(); ok {
				mode = RedirectAsk
			} else if err.IsClusterDown() || err.IsTryAgain() || err.IsLoading() {
				mode = RedirectRetry
			}
		} else if ctx.Err() == nil {
			mode = RedirectRetry
		}
		if mode != RedirectNone {
			c.lazyRefresh()
		}
	}
	return
}

type dedicatedClusterClient struct {
	client       *clusterClient
	conn         conn
	wire         wire
	pshks        *pshks
	mu           sync.Mutex
	cmd          Builder
	retryHandler retryHandler
	retry        bool
	slot         uint16
	mark         bool
}

func (c *dedicatedClusterClient) acquire(ctx context.Context, slot uint16) (wire wire, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.mark {
		return nil, ErrDedicatedClientRecycled
	}
	if c.slot == cmds.NoSlot {
		c.slot = slot
	} else if c.slot != slot && slot != cmds.InitSlot {
		panic(panicMsgCxSlot)
	}
	if c.wire != nil {
		return c.wire, nil
	}
	if c.conn, err = c.client.pick(ctx, c.slot, false); err != nil {
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
	attempts := 1
retry:
	if w, err := c.acquire(ctx, cmd.Slot()); err != nil {
		resp = newErrResult(err)
	} else {
		resp = w.Do(ctx, cmd)
		switch _, mode := c.client.shouldRefreshRetry(resp.Error(), ctx); mode {
		case RedirectRetry:
			if c.retry && cmd.IsReadOnly() && w.Error() == nil {
				shouldRetry := c.retryHandler.WaitOrSkipRetry(
					ctx, attempts, cmd, resp.Error(),
				)
				if shouldRetry {
					attempts++
					goto retry
				}
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
	attempts := 1
retry:
	if w, err := c.acquire(ctx, slot); err == nil {
		resp = w.DoMulti(ctx, multi...).s
		for i, r := range resp {
			_, mode := c.client.shouldRefreshRetry(r.Error(), ctx)
			if mode == RedirectRetry && retryable && w.Error() == nil {
				shouldRetry := c.retryHandler.WaitOrSkipRetry(
					ctx, attempts, multi[i], r.Error(),
				)
				if shouldRetry {
					attempts++
					goto retry
				}
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
	var (
		w        wire
		attempts = 1
	)
retry:
	if w, err = c.acquire(ctx, subscribe.Slot()); err == nil {
		err = w.Receive(ctx, subscribe, fn)
		if _, mode := c.client.shouldRefreshRetry(err, ctx); c.retry && mode == RedirectRetry && w.Error() == nil {
			shouldRetry := c.retryHandler.WaitOrSkipRetry(ctx, attempts, subscribe, err)
			if shouldRetry {
				attempts++
				goto retry
			}
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
	if c.mark {
		ch := make(chan error, 1)
		ch <- ErrDedicatedClientRecycled
		return ch
	}
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
