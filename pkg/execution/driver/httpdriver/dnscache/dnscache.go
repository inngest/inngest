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

	happyEyeballsDelay = 100 * time.Millisecond

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
		// We also implement our own "happy eyes" dial mechanism which allows us
		// to dial both and eject after some delay.
		six, four := sixfour(ips)
		return r.dialParallel(ctx, network, port, four, six)
	}
}

// dialSerial takes a list of IPs and dials them sequentially, without happy eyeballs
// or concurrent connections.  It returns the first available Conn.
//
// This defers to the default dialer, and so we don't need to handle TCP, unix sockets, etc.
func (r *resolver) dialSerial(ctx context.Context, network, port string, ips []net.IP) (net.Conn, error) {
	if len(ips) == 0 {
		return nil, ErrNoIPsAvailable
	}

	var firstErr error
	for _, idx := range r.randPerm(len(ips)) {
		ip := ips[idx]
		addr := net.JoinHostPort(ip.String(), port)

		conn, err := r.dialer(ctx, network, addr)
		if err == nil {
			return conn, nil
		}

		if firstErr == nil {
			firstErr = err
		}

		if ctx.Err() != nil {
			// check if context was cancelled
			return nil, ctx.Err()
		}
	}

	return nil, firstErr
}

// dialParallel races two copies of dialSerial, giving the first a
// head start. It returns the first established connection and
// closes the others. Otherwise it returns an error from the first
// primary address.
func (r *resolver) dialParallel(ctx context.Context, network, port string, primaries, fallbacks []net.IP) (net.Conn, error) {
	if len(fallbacks) == 0 {
		return r.dialSerial(ctx, network, port, primaries)
	}

	// create a chan that tells us if we've already finished the dialer.  this ensures that
	// we don't forever poll on a chan that won't be sent/received.
	returned := make(chan struct{})
	defer close(returned)

	type dialResult struct {
		net.Conn
		error
		primary bool
		done    bool
	}
	results := make(chan dialResult) // unbuffered

	startRacer := func(ctx context.Context, primary bool) {
		ras := primaries
		if !primary {
			ras = fallbacks
		}
		c, err := r.dialSerial(ctx, network, port, ras)
		select {
		case results <- dialResult{Conn: c, error: err, primary: primary, done: true}:
		case <-returned:
			if c != nil {
				c.Close()
			}
		}
	}

	var primary, fallback dialResult

	// Start the main racer.
	primaryCtx, primaryCancel := context.WithCancel(ctx)
	defer primaryCancel()
	go startRacer(primaryCtx, true)

	// Start the timer for the fallback racer.
	fallbackTimer := time.NewTimer(happyEyeballsDelay)
	defer fallbackTimer.Stop()

	for {
		select {
		// after N milliseconds hit ipv6
		case <-fallbackTimer.C:
			fallbackCtx, fallbackCancel := context.WithCancel(ctx)
			defer fallbackCancel()
			go startRacer(fallbackCtx, false)

		case res := <-results:
			if res.error == nil {
				return res.Conn, nil
			}
			if res.primary {
				primary = res
			} else {
				fallback = res
			}
			if primary.done && fallback.done {
				// err != nil and both are done, so return the primary err.
				return nil, primary.error
			}
			if res.primary && fallbackTimer.Stop() {
				// If we were able to stop the timer, that means it
				// was running (hadn't yet started the fallback), but
				// we just got an error on the primary path, so start
				// the fallback immediately (in 0 nanoseconds).
				fallbackTimer.Reset(0)
			}
		}
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
