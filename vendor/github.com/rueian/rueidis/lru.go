package rueidis

import (
	"container/list"
	"context"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	entrySize    = int(unsafe.Sizeof(entry{})) + int(unsafe.Sizeof(&entry{}))
	keyCacheSize = int(unsafe.Sizeof(keyCache{})) + int(unsafe.Sizeof(&keyCache{}))
	elementSize  = int(unsafe.Sizeof(list.Element{})) + int(unsafe.Sizeof(&list.Element{}))
	stringSSize  = int(unsafe.Sizeof(""))

	entryBaseSize = (keyCacheSize + entrySize + elementSize + stringSSize*2) * 3 / 2
	entryMinSize  = entryBaseSize + messageStructSize

	moveThreshold = uint64(1024 - 1)
)

type cache interface {
	GetOrPrepare(key, cmd string, now time.Time, ttl time.Duration) (v RedisMessage, entry *entry)
	Update(key, cmd string, value RedisMessage, pttl int64)
	Cancel(key, cmd string, err error)
	Delete(keys []RedisMessage)
	GetTTL(key string) time.Duration
	FreeAndClose(err error)
}

type entry struct {
	err  error
	ch   chan struct{}
	kc   *keyCache
	cmd  string
	val  RedisMessage
	size int
}

func (e *entry) Wait(ctx context.Context) (RedisMessage, error) {
	if ch := ctx.Done(); ch == nil {
		<-e.ch
	} else {
		select {
		case <-ch:
			return RedisMessage{}, ctx.Err()
		case <-e.ch:
		}
	}
	return e.val, e.err
}

type keyCache struct {
	ttl   time.Time
	cache map[string]*list.Element
	key   string
	hits  uint64
	miss  uint64
}

var _ cache = (*lru)(nil)

type lru struct {
	store map[string]*keyCache
	list  *list.List
	mu    sync.RWMutex
	size  int
	max   int
}

func newLRU(max int) *lru {
	return &lru{
		max:   max,
		store: make(map[string]*keyCache),
		list:  list.New(),
	}
}

func (c *lru) GetOrPrepare(key, cmd string, now time.Time, ttl time.Duration) (v RedisMessage, e *entry) {
	var ok bool
	var kc *keyCache
	var kcTTL time.Time
	var ele, back *list.Element

	c.mu.RLock()
	if kc, ok = c.store[key]; ok {
		kcTTL = kc.ttl
		if ele, ok = kc.cache[cmd]; ok {
			e = ele.Value.(*entry)
			v = e.val
			back = c.list.Back()
		}
	}
	c.mu.RUnlock()

	if e != nil && (v.typ == 0 || kcTTL.After(now)) {
		hits := atomic.AddUint64(&kc.hits, 1)
		if ele != back && hits&moveThreshold == 0 {
			c.mu.Lock()
			if c.list != nil {
				c.list.MoveToBack(ele)
			}
			c.mu.Unlock()
		}
		return v, e
	}

	v = RedisMessage{}
	e = nil

	c.mu.Lock()
	if kc, ok = c.store[key]; !ok {
		if c.store == nil {
			goto miss
		}
		kc = &keyCache{cache: make(map[string]*list.Element, 1), key: key, ttl: now.Add(ttl)}
		c.store[key] = kc
	}
	if ele, ok = kc.cache[cmd]; ok {
		if e = ele.Value.(*entry); e.val.typ == 0 || kc.ttl.After(now) {
			atomic.AddUint64(&kc.hits, 1)
			v = e.val
			c.list.MoveToBack(ele)
		} else {
			c.list.Remove(ele)
			c.size -= e.size
			e = nil
		}
	}
	if e == nil {
		atomic.AddUint64(&kc.miss, 1)
		c.list.PushBack(&entry{
			cmd: cmd,
			kc:  kc,
			ch:  make(chan struct{}, 1),
		})
		kc.ttl = now.Add(ttl)
		kc.cache[cmd] = c.list.Back()
	}
miss:
	c.mu.Unlock()
	return v, e
}

func (c *lru) Update(key, cmd string, value RedisMessage, pttl int64) {
	var ch chan struct{}
	c.mu.Lock()
	if kc, ok := c.store[key]; ok {
		if pttl >= 0 {
			// server side ttl should only shorten client side ttl
			if ttl := time.Now().Add(time.Duration(pttl) * time.Millisecond); ttl.Before(kc.ttl) {
				kc.ttl = ttl
			}
		}
		if ele, ok := kc.cache[cmd]; ok {
			if e := ele.Value.(*entry); e.val.typ == 0 {
				e.val = value
				e.val.setTTL(kc.ttl.Unix())
				e.size = entryBaseSize + 2*(len(key)+len(cmd)) + value.approximateSize()
				c.size += e.size
				ch = e.ch
			}

			ele = c.list.Front()
			for c.size > c.max && ele != nil {
				if e := ele.Value.(*entry); e.val.typ != 0 { // do not delete pending entries
					kc := e.kc
					if delete(kc.cache, e.cmd); len(kc.cache) == 0 {
						delete(c.store, kc.key)
					}
					c.list.Remove(ele)
					c.size -= e.size
				}
				ele = ele.Next()
			}
		}
	}
	c.mu.Unlock()
	if ch != nil {
		close(ch)
	}
}

func (c *lru) Cancel(key, cmd string, err error) {
	var ch chan struct{}
	c.mu.Lock()
	if kc, ok := c.store[key]; ok {
		if ele, ok := kc.cache[cmd]; ok {
			if e := ele.Value.(*entry); e.val.typ == 0 {
				e.err = err
				ch = e.ch
				if delete(kc.cache, cmd); len(kc.cache) == 0 {
					delete(c.store, key)
				}
				c.list.Remove(ele)
			}
		}
	}
	c.mu.Unlock()
	if ch != nil {
		close(ch)
	}
}

func (c *lru) GetTTL(key string) (ttl time.Duration) {
	c.mu.Lock()
	if kc, ok := c.store[key]; ok && len(kc.cache) != 0 {
		ttl = kc.ttl.Sub(time.Now())
	}
	if ttl <= 0 {
		ttl = -2
	}
	c.mu.Unlock()
	return
}

func (c *lru) purge(key string, kc *keyCache) {
	if kc != nil {
		for cmd, ele := range kc.cache {
			if e := ele.Value.(*entry); e.val.typ != 0 { // do not delete pending entries
				if delete(kc.cache, cmd); len(kc.cache) == 0 {
					delete(c.store, key)
				}
				c.list.Remove(ele)
				c.size -= e.size
			}
		}
	}
}

func (c *lru) Delete(keys []RedisMessage) {
	c.mu.Lock()
	if keys == nil {
		for key, kc := range c.store {
			c.purge(key, kc)
		}
	} else {
		for _, k := range keys {
			c.purge(k.string, c.store[k.string])
		}
	}
	c.mu.Unlock()
}

func (c *lru) FreeAndClose(err error) {
	c.mu.Lock()
	for _, kc := range c.store {
		for _, ele := range kc.cache {
			if e := ele.Value.(*entry); e.val.typ == 0 {
				e.err = err
				close(e.ch)
			}
		}
	}
	c.store = nil
	c.list = nil
	c.mu.Unlock()
}
