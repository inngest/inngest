package dnscache

import (
	"context"
	"net"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"golang.org/x/sync/singleflight"
)

var (
	defaultRefreshInterval = 5 * time.Second
	defaultDialer          = &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 15 * time.Second}

	// lookupGroup merges lookup calls together for lookups for the same host. The
	// lookupGroup key is is the LookupIPAddr.host argument.
	lookupGroup singleflight.Group
)

type DNSResolver interface {
	LookupHost(ctx context.Context, host string) (addrs []string, err error)
	LookupAddr(ctx context.Context, addr string) (names []string, err error)
	Dialer() Dialer
}

type ResolverOpts func(r *resolver)
type Dialer func(ctx context.Context, network, addr string) (net.Conn, error)

func New(ctx context.Context, opts ...ResolverOpts) DNSResolver {
	r := resolver{
		refreshInterval: defaultRefreshInterval,
		dialer:          defaultDialer.DialContext,
	}

	for _, apply := range opts {
		apply(&r)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.StdlibLogger(ctx).Error("panic in resolver refresh", "panic", r)
			}
		}()

		t := time.NewTicker(r.refreshInterval)
		defer t.Stop()

		for range t.C {
			select {
			case <-ctx.Done():
				return

			case <-t.C:
				r.refresh(true)
			}
		}
	}()

	return &r
}

func WithCacheRefreshInterval(dur time.Duration) ResolverOpts {
	return func(r *resolver) {
		r.refreshInterval = dur
	}
}

func WithDialer(dialer Dialer) ResolverOpts {
	return func(r *resolver) {
		r.dialer = dialer
	}
}

type resolver struct {
	// Timeout defines the maximum allowed time allowed for a lookup.
	Timeout time.Duration

	lookups int64

	once  sync.Once
	mu    sync.RWMutex
	cache *cache

	// OnCacheMiss is executed if the host or address is not included in
	// the cache and the default lookup is executed.
	OnCacheMiss func()

	// refreshInterval defines the duration between refresh of IP addresses
	refreshInterval time.Duration

	// dialer is the function used to establish the connection
	dialer Dialer
}

type cacheEntry struct {
	rrs  []string
	err  error
	used bool
}

func (r *resolver) Dialer() Dialer {
	return r.dialer
}

// LookupAddr performs a reverse lookup for the given address, returning a list
// of names mapping to that address.
func (r *resolver) LookupAddr(ctx context.Context, addr string) (names []string, err error) {
	r.once.Do(r.init)
	return r.lookup(ctx, "r"+addr)
}

// LookupHost looks up the given host using the local resolver. It returns a
// slice of that host's addresses.
func (r *resolver) LookupHost(ctx context.Context, host string) (addrs []string, err error) {
	r.once.Do(r.init)
	return r.lookup(ctx, "h"+host)
}

// refreshRecords refreshes cached entries which have been used at least once since
// the last Refresh. If clearUnused is true, entries which haven't be used since the
// last Refresh are removed from the cache. If persistOnFailure is true, stale
// entries will not be removed on failed lookups
func (r *resolver) refreshRecords(clearUnused bool, persistOnFailure bool) {
	r.once.Do(r.init)
	r.mu.RLock()
	update := make([]string, 0, r.cache.cache.ItemCount())
	del := make([]string, 0, r.cache.cache.ItemCount())

	r.cache.ForEachFunc(func(key string, entry *cacheEntry) bool {
		if entry.used {
			update = append(update, key)
		} else if clearUnused {
			del = append(del, key)
		}
		return true
	})
	r.mu.RUnlock()

	if len(del) > 0 {
		r.mu.Lock()
		for _, key := range del {
			r.cache.Delete(key)
		}
		r.mu.Unlock()
	}

	for _, key := range update {
		_, _ = r.update(context.Background(), key, false, persistOnFailure)
	}
}

func (r *resolver) refresh(clearUnused bool) {
	r.refreshRecords(clearUnused, false)
}

func (r *resolver) init() {
	r.cache = newCache()
}

func (r *resolver) lookup(ctx context.Context, key string) (rrs []string, err error) {
	var found bool
	rrs, err, found = r.load(key)
	if !found {
		if r.OnCacheMiss != nil {
			r.OnCacheMiss()
		}
		rrs, err = r.update(ctx, key, true, false)
	}
	return
}

func (r *resolver) update(ctx context.Context, key string, used bool, persistOnFailure bool) (rrs []string, err error) {
	c := lookupGroup.DoChan(key, r.lookupFunc(ctx, key))
	select {
	case <-ctx.Done():
		err = ctx.Err()
		if err == context.DeadlineExceeded {
			// If DNS request timed out for some reason, force future
			// request to start the DNS lookup again rather than waiting
			// for the current lookup to complete.
			lookupGroup.Forget(key)
		}
	case res := <-c:
		if res.Shared {
			// We had concurrent lookups, check if the cache is already updated
			// by a friend.
			var found bool
			rrs, err, found = r.load(key)
			if found {
				return
			}
		}
		err = res.Err
		if err == nil {
			rrs, _ = res.Val.([]string)
		}

		if err != nil && persistOnFailure {
			var found bool
			rrs, err, found = r.load(key)
			if found {
				return
			}
		}

		r.mu.Lock()
		r.storeLocked(key, rrs, used, err)
		r.mu.Unlock()
	}
	return
}

// lookupFunc returns lookup function for key. The type of the key is stored as
// the first char and the lookup subject is the rest of the key.
func (r *resolver) lookupFunc(ctx context.Context, key string) func() (interface{}, error) {
	if len(key) == 0 {
		panic("lookupFunc with empty key")
	}

	switch key[0] {
	case 'h':
		return func() (interface{}, error) {
			ctx, cancel := r.prepareCtx(ctx)
			defer cancel()

			atomic.AddInt64(&r.lookups, 1)
			return r.LookupHost(ctx, key[1:])
		}
	case 'r':
		return func() (interface{}, error) {
			ctx, cancel := r.prepareCtx(ctx)
			defer cancel()

			atomic.AddInt64(&r.lookups, 1)
			return r.LookupAddr(ctx, key[1:])
		}
	default:
		panic("lookupFunc invalid key type: " + key)
	}
}

func (r *resolver) prepareCtx(origContext context.Context) (ctx context.Context, cancel context.CancelFunc) {
	ctx = context.Background()
	if r.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, r.Timeout)
	} else {
		cancel = func() {}
	}

	// If a httptrace has been attached to the given context it will be copied over to the newly created context. We only need to copy pointers
	// to DNSStart and DNSDone hooks
	if trace := httptrace.ContextClientTrace(origContext); trace != nil {
		derivedTrace := &httptrace.ClientTrace{
			DNSStart: trace.DNSStart,
			DNSDone:  trace.DNSDone,
		}

		ctx = httptrace.WithClientTrace(ctx, derivedTrace)
	}

	return
}

func (r *resolver) load(key string) (rrs []string, err error, found bool) {
	r.mu.RLock()
	var entry *cacheEntry
	entry, found = r.cache.Get(key)
	if !found {
		r.mu.RUnlock()
		return
	}
	rrs = entry.rrs
	err = entry.err
	used := entry.used
	r.mu.RUnlock()
	if !used {
		r.mu.Lock()
		entry.used = true
		r.mu.Unlock()
	}
	return rrs, err, true
}

func (r *resolver) storeLocked(key string, rrs []string, used bool, err error) {
	if entry, found := r.cache.Get(key); found {
		// Update existing entry in place
		entry.rrs = rrs
		entry.err = err
		entry.used = used
		return
	}
	r.cache.Set(key, &cacheEntry{
		rrs:  rrs,
		err:  err,
		used: used,
	})
}
