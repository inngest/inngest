package dnscache

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestDNSCache(t *testing.T) {
	ctx := context.Background()
	l := logger.StdlibLogger(ctx)

	ttl := 2 * time.Second

	cachedResolver := New(
		WithCacheTTL(ttl),
		WithLogger(l),
	)

	c := http.Client{
		Transport: &http.Transport{
			DialContext: cachedResolver.Dialer(),
		},
	}

	// inngest and vercel use SNI
	hosts := []string{"www.example.com", "www.inngest.com", "vercel.com"}

	for _, host := range hosts {
		t.Run(host, func(t *testing.T) {
			t.Run("non cached", func(t *testing.T) {
				require.False(t, cachedResolver.isCached(host))
			})

			t.Run("request", func(t *testing.T) {
				resp, err := c.Get(fmt.Sprintf("https://%s", host))
				require.NoError(t, err)
				require.EqualValues(t, 200, resp.StatusCode)
			})

			t.Run("cached", func(t *testing.T) {
				require.True(t, cachedResolver.isCached(host))
			})

			<-time.After(ttl)

			t.Run("cache expired", func(t *testing.T) {
				require.False(t, cachedResolver.isCached(host))
			})
		})
	}
}

func TestHappyEyeballs(t *testing.T) {
	ctx := context.Background()

	t.Run("PreferIPv4", func(t *testing.T) {
		var dialOrder []string
		var mu sync.Mutex

		mockDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			mu.Lock()
			dialOrder = append(dialOrder, addr)
			mu.Unlock()

			host, _, _ := net.SplitHostPort(addr)
			ip := net.ParseIP(host)
			
			// Simulate successful IPv4 connection, failed IPv6
			if ip.To4() != nil {
				// IPv4 - return a mock connection
				return &mockConn{}, nil
			}
			// IPv6 - simulate failure
			return nil, fmt.Errorf("connection refused")
		}

		resolver := New(WithDialer(mockDialer))
		
		// Create test IPs (IPv6 first, IPv4 second to test preference)
		testIPs := []net.IP{
			net.ParseIP("2001:db8::1"), // IPv6
			net.ParseIP("192.0.2.1"),   // IPv4
		}

		six, four := sixfour(testIPs)
		conn, err := resolver.dialParallel(ctx, "tcp", "80", four, six)
		
		require.NoError(t, err)
		require.NotNil(t, conn)
		conn.Close()

		// Verify IPv4 was dialed first (primary)
		require.Greater(t, len(dialOrder), 0)
		host, _, _ := net.SplitHostPort(dialOrder[0])
		firstIP := net.ParseIP(host)
		require.NotNil(t, firstIP.To4(), "First dial should be IPv4")
	})

	t.Run("FallbackToIPv6", func(t *testing.T) {
		var dialOrder []string
		var dialResults []bool // true = success, false = failure
		var mu sync.Mutex

		mockDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			mu.Lock()
			dialOrder = append(dialOrder, addr)
			mu.Unlock()

			host, _, _ := net.SplitHostPort(addr)
			ip := net.ParseIP(host)
			
			// Simulate IPv4 failure, IPv6 success
			if ip.To4() != nil {
				// IPv4 - simulate failure
				mu.Lock()
				dialResults = append(dialResults, false)
				mu.Unlock()
				return nil, fmt.Errorf("connection refused")
			}
			// IPv6 - return successful connection
			mu.Lock()
			dialResults = append(dialResults, true)
			mu.Unlock()
			return &mockConn{}, nil
		}

		resolver := New(WithDialer(mockDialer))
		
		testIPs := []net.IP{
			net.ParseIP("2001:db8::1"), // IPv6
			net.ParseIP("192.0.2.1"),   // IPv4
		}

		six, four := sixfour(testIPs)
		conn, err := resolver.dialParallel(ctx, "tcp", "80", four, six)
		
		require.NoError(t, err)
		require.NotNil(t, conn)
		conn.Close()

		mu.Lock()
		defer mu.Unlock()

		// Should have tried both IPv4 (failed) and IPv6 (succeeded)
		require.Equal(t, 2, len(dialOrder))
		require.Equal(t, 2, len(dialResults))
		
		// First attempt should be IPv4 (primary), should fail
		host1, _, _ := net.SplitHostPort(dialOrder[0])
		firstIP := net.ParseIP(host1)
		require.NotNil(t, firstIP.To4(), "First dial should be IPv4")
		require.False(t, dialResults[0], "IPv4 should fail")
		
		// Second attempt should be IPv6 (fallback), should succeed
		host2, _, _ := net.SplitHostPort(dialOrder[1])
		secondIP := net.ParseIP(host2)
		require.Nil(t, secondIP.To4(), "Second dial should be IPv6")
		require.True(t, dialResults[1], "IPv6 should succeed")
	})

	t.Run("IPv4OnlySuccess", func(t *testing.T) {
		mockDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return &mockConn{}, nil
		}

		resolver := New(WithDialer(mockDialer))
		
		// Only IPv4 addresses
		testIPs := []net.IP{
			net.ParseIP("192.0.2.1"),
			net.ParseIP("192.0.2.2"),
		}

		six, four := sixfour(testIPs)
		require.Empty(t, six, "Should have no IPv6 addresses")
		require.Len(t, four, 2, "Should have 2 IPv4 addresses")

		conn, err := resolver.dialParallel(ctx, "tcp", "80", four, six)
		
		require.NoError(t, err)
		require.NotNil(t, conn)
		conn.Close()
	})

	t.Run("IPv6OnlySuccess", func(t *testing.T) {
		mockDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return &mockConn{}, nil
		}

		resolver := New(WithDialer(mockDialer))
		
		// Only IPv6 addresses
		testIPs := []net.IP{
			net.ParseIP("2001:db8::1"),
			net.ParseIP("2001:db8::2"),
		}

		six, four := sixfour(testIPs)
		require.Len(t, six, 2, "Should have 2 IPv6 addresses")
		require.Empty(t, four, "Should have no IPv4 addresses")

		conn, err := resolver.dialParallel(ctx, "tcp", "80", four, six)
		
		require.NoError(t, err)
		require.NotNil(t, conn)
		conn.Close()
	})

	t.Run("BothFail", func(t *testing.T) {
		mockDialer := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, fmt.Errorf("connection refused")
		}

		resolver := New(WithDialer(mockDialer))
		
		testIPs := []net.IP{
			net.ParseIP("2001:db8::1"), // IPv6
			net.ParseIP("192.0.2.1"),   // IPv4
		}

		six, four := sixfour(testIPs)
		conn, err := resolver.dialParallel(ctx, "tcp", "80", four, six)
		
		require.Error(t, err)
		require.Nil(t, conn)
		require.Contains(t, err.Error(), "connection refused")
	})
}

func TestSixFour(t *testing.T) {
	testIPs := []net.IP{
		net.ParseIP("192.0.2.1"),     // IPv4
		net.ParseIP("2001:db8::1"),   // IPv6
		net.ParseIP("203.0.113.1"),   // IPv4
		net.ParseIP("2001:db8::2"),   // IPv6
	}

	six, four := sixfour(testIPs)
	
	require.Len(t, four, 2, "Should have 2 IPv4 addresses")
	require.Len(t, six, 2, "Should have 2 IPv6 addresses")
	
	// Verify IPv4 addresses
	for _, ip := range four {
		require.NotNil(t, ip.To4(), "Should be IPv4")
	}
	
	// Verify IPv6 addresses
	for _, ip := range six {
		require.Nil(t, ip.To4(), "Should be IPv6")
	}
}

// mockConn is a mock net.Conn for testing
type mockConn struct{}

func (m *mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr               { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0} }
func (m *mockConn) RemoteAddr() net.Addr              { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 80} }
func (m *mockConn) SetDeadline(t time.Time) error     { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }
