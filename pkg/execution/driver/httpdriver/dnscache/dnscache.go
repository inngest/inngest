package dnscache

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"syscall"
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
	DialerWithNAT64Fallback() Dialer
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
	return r.dialerWithNAT64Fallback(false)
}

func (r *resolver) DialerWithNAT64Fallback() Dialer {
	return r.dialerWithNAT64Fallback(true)
}

func (r *resolver) dialerWithNAT64Fallback(allowNAT64Fallback bool) Dialer {
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
		return r.dialParallelWithNAT64Fallback(ctx, network, port, four, six, allowNAT64Fallback)
	}
}

type dialResult struct {
	net.Conn
	error
	primary              bool
	done                 bool
	allTimedOutOrNoRoute bool
	allNoRoute           bool
}

// dialSerial takes a list of IPs and dials them sequentially, without happy eyeballs
// or concurrent connections.  It returns the first available Conn.
//
// This defers to the default dialer, and so we don't need to handle TCP, unix sockets, etc.
func (r *resolver) dialSerial(ctx context.Context, network, port string, ips []net.IP) (net.Conn, error) {
	result := r.dialSerialResult(ctx, network, port, ips)
	return result.Conn, result.error
}

func (r *resolver) dialSerialResult(ctx context.Context, network, port string, ips []net.IP) dialResult {
	if len(ips) == 0 {
		return dialResult{error: ErrNoIPsAvailable}
	}

	var firstErr error
	allTimedOutOrNoRoute := true
	allNoRoute := true
	for _, idx := range r.randPerm(len(ips)) {
		ip := ips[idx]
		addr := net.JoinHostPort(ip.String(), port)

		conn, err := r.dialer(ctx, network, addr)
		if err == nil {
			return dialResult{Conn: conn}
		}

		if firstErr == nil {
			firstErr = err
		}
		var netErr net.Error
		noRoute := errors.Is(err, syscall.ENETUNREACH)
		timedOut := errors.As(err, &netErr) && netErr.Timeout()
		allTimedOutOrNoRoute = allTimedOutOrNoRoute && (timedOut || noRoute)
		allNoRoute = allNoRoute && noRoute

		if ctx.Err() != nil {
			// check if context was cancelled
			return dialResult{error: ctx.Err()}
		}
	}

	return dialResult{
		error:                firstErr,
		allTimedOutOrNoRoute: allTimedOutOrNoRoute,
		allNoRoute:           allNoRoute,
	}
}

// dialParallel races two copies of dialSerial, giving the first a
// head start. It returns the first established connection and
// closes the others. Otherwise it returns an error from the first
// primary address.
func (r *resolver) dialParallel(ctx context.Context, network, port string, primaries, fallbacks []net.IP) (net.Conn, error) {
	return r.dialParallelWithNAT64Fallback(ctx, network, port, primaries, fallbacks, false)
}

func (r *resolver) dialParallelWithNAT64Fallback(ctx context.Context, network, port string, primaries, fallbacks []net.IP, allowNAT64Fallback bool) (net.Conn, error) {
	// a pod with only IPv6 has no route to the A records.  NAT64 can still reach them.
	ipv4HasNoRoute := func(v4 dialResult) bool {
		return allowNAT64Fallback && network == "tcp" && ctx.Err() == nil && v4.allNoRoute
	}

	if len(fallbacks) == 0 {
		result := r.dialSerialResult(ctx, network, port, primaries)
		if ipv4HasNoRoute(result) {
			return r.dialNAT64(ctx, port, primaries, result.error)
		}
		return result.Conn, result.error
	}
	if len(primaries) == 0 {
		// also handle ipv6 only, if someone is brave enough to do so!
		return r.dialSerial(ctx, network, port, fallbacks)
	}

	// create a chan that tells us if we've already finished the dialer.  this ensures that
	// we don't forever poll on a chan that won't be sent/received.
	returned := make(chan struct{})
	defer close(returned)

	results := make(chan dialResult) // unbuffered

	startRacer := func(ctx context.Context, primary bool) {
		ras := primaries
		if !primary {
			ras = fallbacks
		}
		result := r.dialSerialResult(ctx, network, port, ras)
		result.primary = primary
		result.done = true
		select {
		case results <- result:
		case <-returned:
			if result.Conn != nil {
				result.Conn.Close()
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
				if ipv4HasNoRoute(primary) && fallback.allTimedOutOrNoRoute {
					return r.dialNAT64(ctx, port, primaries, primary.error)
				}
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

// dialNAT64 dials each IPv4 address through the NAT64 prefix 64:ff9b::/96.
// if every NAT64 dial fails, it returns dialErr so the caller sees the error
// from the original IPv4 dial.
func (r *resolver) dialNAT64(ctx context.Context, port string, ips []net.IP, dialErr error) (net.Conn, error) {
	conn, err := r.dialSerial(ctx, "tcp6", port, nat64Addresses(ips))
	if err != nil {
		return nil, dialErr
	}
	return conn, nil
}

func nat64Addresses(ips []net.IP) []net.IP {
	addresses := make([]net.IP, 0, len(ips))
	for _, ip := range ips {
		address := make(net.IP, net.IPv6len)
		address[1] = 0x64
		address[2] = 0xff
		address[3] = 0x9b
		copy(address[12:], ip.To4())
		addresses = append(addresses, address)
	}
	return addresses
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
