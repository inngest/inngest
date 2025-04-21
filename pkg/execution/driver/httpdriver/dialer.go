package httpdriver

import (
	"context"
	"fmt"
	// "math/rand/v2"
	"net"
	"time"

	"github.com/inngest/inngest/pkg/execution/driver/httpdriver/dnscache"
	"github.com/inngest/inngest/pkg/logger"
	mdnscache "go.mercari.io/go-dnscache"
)

const (
	dnsCacheRefreshInterval = 5 * time.Minute
	defaultLookupTimeout    = 5 * time.Second
)

var (
	privateIPBlocks []*net.IPNet
	nat64blocks     []*net.IPNet
	cachedResolver  *dnscache.Resolver
)

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr (RFC4193)
		"ff00::/8",       // multicast
		"fec0::/10",      // deprecated
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}

	for _, cidr := range []string{
		"64:ff9b::/96",       // RFC 6052 suffix
		"2001:db8::/32",      // RFC 6052 bits 32-63
		"2001:db8:aaaa::/48", // RFC 6052 bits 48-87
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		nat64blocks = append(nat64blocks, block)
	}

	// Resolver for caching DNS lookups.
	cachedResolver = &dnscache.Resolver{}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.StdlibLogger(context.Background()).
					Error("panic in resolver refresh", "error", r)
			}
		}()

		t := time.NewTicker(dnsCacheRefreshInterval)
		defer t.Stop()
		for range t.C {
			// Remove entries that haven't been used since the last refresh.
			removeUnused := true

			cachedResolver.Refresh(removeUnused)
		}
	}()
}

type SecureDialerOpts struct {
	AllowHostDocker bool
	AllowPrivate    bool
	AllowNAT64      bool

	// log is used in testing.
	log bool
	// dial is a function used to actually dial, allowed to override in testing
	// for success.
	dial DialFunc
}

type DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error)

func SecureDialer(o SecureDialerOpts) DialFunc {
	resolver, err := mdnscache.New(dnsCacheRefreshInterval, defaultLookupTimeout)
	if err != nil {
		return nil
	}

	if o.dial == nil {
		// Always use the default dialer.  Only allow overrides in testing.
		o.dial = mdnscache.DialFunc(resolver, nil)
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		// network will be one of the well defined networks as per
		// https://pkg.go.dev/net#Dial, eg "tcp", "tcp4", "tcp6", etc.
		//
		// addr may be a domain or ip and port: "example.com:443", "192.0.2.1:http",
		// "[fe80::1%lo0]:53".
		//
		// We always want to ensure we translate the domains to IP addresses.
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		if !o.AllowHostDocker && isDockerHost(host) {
			return nil, fmt.Errorf("Unable to make request to %s at IP %s: accessing docker host", addr, host)
		}

		// Ensure that the current hostname is not a domain name.
		// ips, err := cachedResolver.LookupHost(ctx, host)
		ips, err := resolver.Fetch(ctx, addr)
		if err != nil {
			return nil, err
		}

		if o.log {
			logger.StdlibLogger(ctx).Info("domain resolved",
				"address", addr,
				"hosts", ips,
			)
		}

		for _, ip := range ips {
			if !o.AllowPrivate && isPrivateHost(ip.String()) {
				return nil, fmt.Errorf("Unable to make request to %s at IP %s: private IP range", addr, ip)
			}
			if !o.AllowNAT64 && isNat64(ip.String()) {
				return nil, fmt.Errorf("Unable to make request to %s at IP %s: NAT64 address", addr, ip)
			}
		}

		return o.dial(ctx, network, addr)

		// // Randomize the order of the IPs. The purpose is to evenly distribute
		// // load across the IPs.
		// rand.Shuffle(len(ips), func(i, j int) {
		// 	ips[i], ips[j] = ips[j], ips[i]
		// })

		// // Try each IP until we get a connection.
		// var conn net.Conn
		// for _, ip := range ips {
		// 	// We need to give the dialer an IP address. Otherwise, it will do
		// 	// DNS lookup that doesn't use the cached resolver.
		// 	addr := net.JoinHostPort(ip, port)

		// 	conn, err = o.dial(ctx, network, addr)
		// 	if err == nil {
		// 		break
		// 	}
		// }
		// return conn, err
	}
}

func isDockerHost(host string) bool {
	return host == "host.docker.internal"
}

func isPrivateHost(host string) bool {
	// fast path;  non-exhaustive for fast lookups.  Basic string matching.
	if host == "localhost" || host == "0.0.0.0" || host == "localhost.localdomain" {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return isPrivateIP(ip)
	}
	return false
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() ||
		ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

func isNat64(host string) bool {
	ip := net.ParseIP(host)
	for _, block := range nat64blocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}
