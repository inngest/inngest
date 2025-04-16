package dnscache

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDNSCache(t *testing.T) {
	cachedResolver := &Resolver{}

	dialer := &net.Dialer{KeepAlive: 15 * time.Second}

	c := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// network will be one of the well defined networks as per
				// https://pkg.go.dev/net#Dial, eg "tcp", "tcp4", "tcp6", etc.
				//
				// addr may be a domain or ip and port: "example.com:443", "192.0.2.1:http",
				// "[fe80::1%lo0]:53".
				//
				// We always want to ensure we translate the domains to IP addresses.
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}

				addrs, err := cachedResolver.LookupHost(t.Context(), host)
				require.NoError(t, err)

				// Try each IP until we get a connection.
				var conn net.Conn
				for _, ip := range addrs {
					// We need to give the dialer an IP address. Otherwise, it will do
					// DNS lookup that doesn't use the cached resolver.
					addr := net.JoinHostPort(ip, port)

					conn, err = dialer.DialContext(ctx, network, addr)
					if err == nil {
						break
					}
				}
				return conn, err
			},
		},
	}

	// inngest and vercel use SNI
	addrs := []string{"https://www.example.com", "https://www.inngest.com", "https://vercel.com"}

	for _, host := range addrs {
		resp, err := c.Get(host)
		require.NoError(t, err)
		require.EqualValues(t, 200, resp.StatusCode)

		entry, ok := cachedResolver.cache.Get("h" + strings.ReplaceAll(host, "https://", ""))
		require.True(t, ok)
		require.NotNil(t, entry)
		require.True(t, entry.used)
	}

	require.EqualValues(t, len(addrs), cachedResolver.lookups)

	// These shouldn't incur lookups, as we just looked them up.
	for _, host := range addrs {
		resp, err := c.Get(host)
		require.NoError(t, err)
		require.EqualValues(t, 200, resp.StatusCode)
	}
	require.EqualValues(t, len(addrs), cachedResolver.lookups)
}
