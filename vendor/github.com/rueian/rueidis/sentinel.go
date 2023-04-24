package rueidis

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rueian/rueidis/internal/cmds"
)

func newSentinelClient(opt *ClientOption, connFn connFn) (client *sentinelClient, err error) {
	client = &sentinelClient{
		cmd:       cmds.NewBuilder(cmds.NoSlot),
		mOpt:      opt,
		sOpt:      newSentinelOpt(opt),
		connFn:    connFn,
		sentinels: list.New(),
		retry:     !opt.DisableRetry,
	}

	for _, sentinel := range opt.InitAddress {
		client.sentinels.PushBack(sentinel)
	}

	if err = client.refresh(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

type sentinelClient struct {
	mConn     atomic.Value
	sConn     conn
	mOpt      *ClientOption
	sOpt      *ClientOption
	connFn    connFn
	sentinels *list.List
	mAddr     string
	sAddr     string
	sc        call
	mu        sync.Mutex
	stop      uint32
	cmd       cmds.Builder
	retry     bool
}

func (c *sentinelClient) B() cmds.Builder {
	return c.cmd
}

func (c *sentinelClient) Do(ctx context.Context, cmd Completed) (resp RedisResult) {
retry:
	resp = c.mConn.Load().(conn).Do(ctx, cmd)
	if c.retry && cmd.IsReadOnly() && c.isRetryable(resp.NonRedisError(), ctx) {
		goto retry
	}
	if resp.NonRedisError() == nil { // not recycle cmds if error, since cmds may be used later in pipe. consider recycle them by pipe
		cmds.PutCompleted(cmd)
	}
	return resp
}

func (c *sentinelClient) DoMulti(ctx context.Context, multi ...Completed) (resps []RedisResult) {
	if len(multi) == 0 {
		return nil
	}
retry:
	resps = c.mConn.Load().(conn).DoMulti(ctx, multi...)
	if c.retry && allReadOnly(multi) {
		for _, resp := range resps {
			if c.isRetryable(resp.NonRedisError(), ctx) {
				goto retry
			}
		}
	}
	for i, cmd := range multi {
		if resps[i].NonRedisError() == nil {
			cmds.PutCompleted(cmd)
		}
	}
	return resps
}

func (c *sentinelClient) DoCache(ctx context.Context, cmd Cacheable, ttl time.Duration) (resp RedisResult) {
retry:
	resp = c.mConn.Load().(conn).DoCache(ctx, cmd, ttl)
	if c.retry && c.isRetryable(resp.NonRedisError(), ctx) {
		goto retry
	}
	if err := resp.NonRedisError(); err == nil || err == ErrDoCacheAborted {
		cmds.PutCacheable(cmd)
	}
	return resp
}

func (c *sentinelClient) DoMultiCache(ctx context.Context, multi ...CacheableTTL) (resps []RedisResult) {
	if len(multi) == 0 {
		return nil
	}
retry:
	resps = c.mConn.Load().(conn).DoMultiCache(ctx, multi...)
	if c.retry {
		for _, resp := range resps {
			if c.isRetryable(resp.NonRedisError(), ctx) {
				goto retry
			}
		}
	}
	for i, cmd := range multi {
		if err := resps[i].NonRedisError(); err == nil || err == ErrDoCacheAborted {
			cmds.PutCacheable(cmd.Cmd)
		}
	}
	return resps
}

func (c *sentinelClient) Receive(ctx context.Context, subscribe Completed, fn func(msg PubSubMessage)) (err error) {
retry:
	err = c.mConn.Load().(conn).Receive(ctx, subscribe, fn)
	if c.retry {
		if _, ok := err.(*RedisError); !ok && c.isRetryable(err, ctx) {
			goto retry
		}
	}
	if err == nil {
		cmds.PutCompleted(subscribe)
	}
	return err
}

func (c *sentinelClient) Dedicated(fn func(DedicatedClient) error) (err error) {
	master := c.mConn.Load().(conn)
	wire := master.Acquire()
	dsc := &dedicatedSingleClient{cmd: c.cmd, conn: master, wire: wire, retry: c.retry}
	err = fn(dsc)
	dsc.release()
	return err
}

func (c *sentinelClient) Dedicate() (DedicatedClient, func()) {
	master := c.mConn.Load().(conn)
	wire := master.Acquire()
	dsc := &dedicatedSingleClient{cmd: c.cmd, conn: master, wire: wire, retry: c.retry}
	return dsc, dsc.release
}

func (c *sentinelClient) Nodes() map[string]Client {
	conn := c.mConn.Load().(conn)
	return map[string]Client{conn.Addr(): newSingleClientWithConn(conn, c.cmd, c.retry)}
}

func (c *sentinelClient) Close() {
	atomic.StoreUint32(&c.stop, 1)
	c.mu.Lock()
	if c.sConn != nil {
		c.sConn.Close()
	}
	if master := c.mConn.Load(); master != nil {
		master.(conn).Close()
	}
	c.mu.Unlock()
}

func (c *sentinelClient) isRetryable(err error, ctx context.Context) (should bool) {
	return err != nil && atomic.LoadUint32(&c.stop) == 0 && ctx.Err() == nil
}

func (c *sentinelClient) addSentinel(addr string) {
	c.mu.Lock()
	c._addSentinel(addr)
	c.mu.Unlock()
}

func (c *sentinelClient) _addSentinel(addr string) {
	for e := c.sentinels.Front(); e != nil; e = e.Next() {
		if e.Value.(string) == addr {
			return
		}
	}
	c.sentinels.PushFront(addr)
}

func (c *sentinelClient) switchMasterRetry(addr string) {
	c.mu.Lock()
	err := c._switchMaster(addr)
	c.mu.Unlock()
	if err != nil {
		go c.refreshRetry()
	}
}

func (c *sentinelClient) _switchMaster(addr string) (err error) {
	var master conn
	if atomic.LoadUint32(&c.stop) == 1 {
		return nil
	}
	if c.mAddr == addr {
		master = c.mConn.Load().(conn)
		if master.Error() != nil {
			master = nil
		}
	}
	if master == nil {
		master = c.connFn(addr, c.mOpt)
		if err = master.Dial(); err != nil {
			return err
		}
	}
	if resp, err := master.Do(context.Background(), cmds.RoleCmd).ToArray(); err != nil {
		master.Close()
		return err
	} else if resp[0].string != "master" {
		master.Close()
		return errNotMaster
	}
	c.mAddr = addr
	if old := c.mConn.Swap(master); old != nil {
		if prev := old.(conn); prev != master {
			prev.Close()
		}
	}
	return nil
}

func (c *sentinelClient) refreshRetry() {
retry:
	if err := c.refresh(); err != nil {
		goto retry
	}
}

func (c *sentinelClient) refresh() (err error) {
	return c.sc.Do(c._refresh)
}

func (c *sentinelClient) _refresh() (err error) {
	var master string
	var sentinels []string

	c.mu.Lock()
	head := c.sentinels.Front()
	for e := head; e != nil; {
		if atomic.LoadUint32(&c.stop) == 1 {
			c.mu.Unlock()
			return nil
		}
		addr := e.Value.(string)

		if c.sAddr != addr || c.sConn == nil || c.sConn.Error() != nil {
			if c.sConn != nil {
				c.sConn.Close()
			}
			c.sAddr = addr
			c.sConn = c.connFn(addr, c.sOpt)
			err = c.sConn.Dial()
		}
		if err == nil {
			if master, sentinels, err = c.listWatch(c.sConn); err == nil {
				for _, sentinel := range sentinels {
					c._addSentinel(sentinel)
				}
				if err = c._switchMaster(master); err == nil {
					break
				}
			}
			c.sConn.Close()
		}
		c.sentinels.MoveToBack(e)
		if e = c.sentinels.Front(); e == head {
			break
		}
	}
	c.mu.Unlock()

	if err == nil {
		if master := c.mConn.Load(); master == nil {
			err = ErrNoAddr
		} else {
			err = master.(conn).Error()
		}
	}
	return err
}

func (c *sentinelClient) listWatch(cc conn) (master string, sentinels []string, err error) {
	ctx := context.Background()
	sentinelsCMD := c.cmd.SentinelSentinels().Master(c.mOpt.Sentinel.MasterSet).Build()
	getMasterCMD := c.cmd.SentinelGetMasterAddrByName().Master(c.mOpt.Sentinel.MasterSet).Build()
	defer func() {
		if err == nil { // not recycle cmds if error, since cmds may be used later in pipe. consider recycle them by pipe
			cmds.PutCompleted(sentinelsCMD)
			cmds.PutCompleted(getMasterCMD)
		}
	}()

	go func(cc conn) {
		if err := cc.Receive(ctx, cmds.SentinelSubscribe, func(event PubSubMessage) {
			switch event.Channel {
			case "+sentinel":
				m := strings.SplitN(event.Message, " ", 4)
				c.addSentinel(fmt.Sprintf("%s:%s", m[2], m[3]))
			case "+switch-master":
				m := strings.SplitN(event.Message, " ", 5)
				if m[0] == c.sOpt.Sentinel.MasterSet {
					c.switchMasterRetry(fmt.Sprintf("%s:%s", m[3], m[4]))
				}
			case "+reboot":
				m := strings.SplitN(event.Message, " ", 4)
				if m[0] == "master" && m[1] == c.sOpt.Sentinel.MasterSet {
					c.switchMasterRetry(fmt.Sprintf("%s:%s", m[2], m[3]))
				}
			}
		}); err != nil && atomic.LoadUint32(&c.stop) == 0 {
			c.refreshRetry()
		}
	}(cc)

	resp := cc.DoMulti(ctx, sentinelsCMD, getMasterCMD)
	others, err := resp[0].ToArray()
	if err != nil {
		return "", nil, err
	}
	for _, other := range others {
		if m, err := other.AsStrMap(); err == nil {
			sentinels = append(sentinels, fmt.Sprintf("%s:%s", m["ip"], m["port"]))
		}
	}
	m, err := resp[1].AsStrSlice()
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("%s:%s", m[0], m[1]), sentinels, nil
}

func newSentinelOpt(opt *ClientOption) *ClientOption {
	o := *opt
	o.Username = o.Sentinel.Username
	o.Password = o.Sentinel.Password
	o.ClientName = o.Sentinel.ClientName
	o.Dialer = o.Sentinel.Dialer
	o.TLSConfig = o.Sentinel.TLSConfig
	o.SelectDB = 0 // https://github.com/rueian/rueidis/issues/138
	return &o
}

var errNotMaster = errors.New("the redis is not master")
