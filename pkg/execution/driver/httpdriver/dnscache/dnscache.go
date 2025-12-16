package dnscache

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"time"

	"github.com/inngest/inngest/pkg/logger"
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

	ErrNoIPsAvailable = fmt.Errorf("no ips available for domain")
)

type cacheType []net.IP

type DNSResolver interface {
	Lookup(ctx context.Context, host string) ([]net.IP, error)
	Dialer() Dialer
}

type (
	ResolverOpts func(r *resolver)
	Dialer       func(ctx context.Context, network, addr string) (net.Conn, error)
)

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

func WithLogger(l logger.Logger) ResolverOpts {
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
	l logger.Logger
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

		// XXX: When IPv6 and IPv4 are both specified, prefer IPv4.  This allows us to
		// prefer egress without NAT64 hops, which on some clouds is required.
		//
		// We cna also implement our own "happy eyes" dial mechanism which allows us
		// to dial both and eject after 300ms.

		six, four := sixfour(ips)

		var retErr error

		// First, dial the IPv4 addresses.
		for _, idx := range r.randPerm(len(four)) {
			ip := ips[idx]
			conn, err := r.dialer(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			if retErr == nil {
				retErr = err
			}
		}

		for _, idx := range r.randPerm(len(six)) {
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

		if len(ips) == 0 {
			return nil, ErrNoIPsAvailable
		}

		return ips, nil
	})
	if err != nil {
		if r.l != nil {
			r.l.Error("error fetching ips from cache", "host", host)
		}
		return nil, err
	}

	return item.Value(), nil
}

// sixfour returns the ips split by ipv6 and ipv4
func sixfour(ips []net.IP) (six []net.IP, four []net.IP) {
	for _, ip := range ips {
		if ip.To4() != nil {
			four = append(four, ip)
		} else {
			six = append(six, ip)
		}
	}
	return six, four
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
