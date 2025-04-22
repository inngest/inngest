package dnscache

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net"
	"time"

	"github.com/karlseguin/ccache/v3"
	// "golang.org/x/sync/singleflight"
)

var (
	defaultRefreshInterval = 5 * time.Second
	defaultCacheTTL        = 5 * time.Second
	defaultLookupTimeout   = 5 * time.Second

	// default dialer to use if not provided
	defaultDialer = &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 15 * time.Second}

	// lookupGroup merges lookup calls together for lookups for the same host. The
	// lookupGroup key is is the LookupIPAddr.host argument.
	// lookupGroup singleflight.Group
)

type cacheType []net.IP

type DNSResolver interface {
	Lookup(ctx context.Context, host string) ([]net.IP, error)
	Dialer() Dialer
}

type ResolverOpts func(r *resolver)
type Dialer func(ctx context.Context, network, addr string) (net.Conn, error)

func New(opts ...ResolverOpts) *resolver {
	r := resolver{
		lookupTimeout:   defaultLookupTimeout,
		refreshInterval: defaultRefreshInterval,
		dialer:          defaultDialer.DialContext,
		cacheTTL:        defaultCacheTTL,
	}

	for _, apply := range opts {
		apply(&r)
	}

	// initialize the cache
	r.cache = ccache.New(ccache.Configure[cacheType]().MaxSize(10_000).ItemsToPrune(500))

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

func WithCacheTTL(ttl time.Duration) ResolverOpts {
	return func(r *resolver) {
		r.cacheTTL = ttl
	}
}

func WithLookupTimeout(t time.Duration) ResolverOpts {
	return func(r *resolver) {
		r.lookupTimeout = t
	}
}

func WithLogger(l *slog.Logger) ResolverOpts {
	return func(r *resolver) {
		r.l = l
	}
}

type resolver struct {
	// lookupTimeout defines the maximum allowed time allowed for a lookup.
	lookupTimeout time.Duration

	// cache stores the LRU cache for list of IPs
	cache *ccache.Cache[cacheType]

	// cacheTTL sets the time the cache is valid for
	cacheTTL time.Duration

	// refreshInterval defines the duration between refresh of IP addresses
	refreshInterval time.Duration

	// dialer is the function used to establish the connection
	dialer Dialer

	// l is an optional logger
	l *slog.Logger
}

func (r *resolver) Dialer() Dialer {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		lctx, cancel := context.WithTimeout(ctx, r.lookupTimeout)
		defer cancel()

		ips, err := r.Lookup(lctx, host)
		if err != nil {
			return nil, err
		}

		var retErr error
		for _, idx := range r.randPerm(len(ips)) {
			ip := ips[idx]
			conn, err := r.dialer(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			if retErr == nil {
				retErr = err
			}
		}
		return nil, retErr
	}
}

func (r *resolver) Lookup(ctx context.Context, host string) ([]net.IP, error) {
	key := fmt.Sprintf("h:%s", host)

	// should this utilize singleflight to reduce map lookups?
	item, err := r.cache.Fetch(key, r.cacheTTL, func() (cacheType, error) {
		// should this provide resolver override?
		addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}

		ips := make([]net.IP, len(addrs))
		for i, addr := range addrs {
			ips[i] = addr.IP
		}

		return ips, nil
	})
	if err != nil {
		if r.l != nil {
			r.l.Error("error fetching ips from cache", "host", host)
		}
		return nil, fmt.Errorf("error fetching ips from cache")
	}

	return item.Value(), nil
}

func (r *resolver) randPerm(n int) []int {
	return rand.Perm(n)
}

// isCached is mainly used to test if the addr is properly cached
func (r *resolver) isCached(host string) bool {
	key := fmt.Sprintf("h:%s", host)
	item := r.cache.Get(key)
	return item != nil && !item.Expired()
}
