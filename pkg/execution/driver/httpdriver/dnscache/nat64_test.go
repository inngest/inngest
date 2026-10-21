package dnscache

import (
	"context"
	"net"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNAT64Fallback(t *testing.T) {
	v4 := net.ParseIP("192.0.2.1")
	v6 := net.ParseIP("2001:db8::1")
	nat64 := net.ParseIP("64:ff9b::c000:201")

	tests := []struct {
		name         string
		allow        bool
		network      string
		v4Error      error
		v6Error      error
		otherV6Error error
		wantNAT64    bool
		wantSuccess  bool
	}{
		{
			name:  "native IPv6 timeout after IPv4 has no route",
			allow: true, network: "tcp",
			v4Error: syscall.ENETUNREACH, v6Error: context.DeadlineExceeded,
			wantNAT64: true, wantSuccess: true,
		},
		{
			name:    "NAT64 disabled",
			network: "tcp",
			v4Error: syscall.ENETUNREACH, v6Error: context.DeadlineExceeded,
		},
		{
			name:  "native IPv6 refused",
			allow: true, network: "tcp",
			v4Error: syscall.ENETUNREACH, v6Error: syscall.ECONNREFUSED,
		},
		{
			name:  "one native IPv6 address refused",
			allow: true, network: "tcp",
			v4Error: syscall.ENETUNREACH, v6Error: context.DeadlineExceeded,
			otherV6Error: syscall.ECONNREFUSED,
		},
		{
			name:  "IPv4 refused",
			allow: true, network: "tcp",
			v4Error: syscall.ECONNREFUSED, v6Error: context.DeadlineExceeded,
		},
		{
			name:  "explicit IPv4 network",
			allow: true, network: "tcp4",
			v4Error: syscall.ENETUNREACH, v6Error: context.DeadlineExceeded,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var attempts []string
			var mu sync.Mutex
			resolver := New(WithDialer(func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, _, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				mu.Lock()
				attempts = append(attempts, host)
				mu.Unlock()
				ip := net.ParseIP(host)
				switch {
				case ip.Equal(nat64):
					return &mockConn{}, nil
				case ip.To4() != nil:
					return nil, &net.OpError{Op: "dial", Net: network, Err: tc.v4Error}
				case tc.otherV6Error != nil && ip.Equal(net.ParseIP("2001:db8::2")):
					return nil, &net.OpError{Op: "dial", Net: network, Err: tc.otherV6Error}
				default:
					return nil, &net.OpError{Op: "dial", Net: network, Err: tc.v6Error}
				}
			}))

			v6Addresses := []net.IP{v6}
			if tc.otherV6Error != nil {
				v6Addresses = append(v6Addresses, net.ParseIP("2001:db8::2"))
			}
			conn, err := resolver.dialParallelWithNAT64Fallback(context.Background(), tc.network, "443", []net.IP{v4}, v6Addresses, tc.allow)
			if tc.wantSuccess {
				require.NoError(t, err)
				require.NotNil(t, conn)
				require.NoError(t, conn.Close())
			} else {
				require.Nil(t, conn)
				require.ErrorIs(t, err, tc.v4Error)
			}
			require.Contains(t, attempts, v4.String())
			require.Contains(t, attempts, v6.String())
			if tc.wantNAT64 {
				require.Contains(t, attempts, nat64.String())
			} else {
				require.NotContains(t, attempts, nat64.String())
			}
		})
	}
}

func TestNAT64FallbackWithoutAAAA(t *testing.T) {
	v4 := net.ParseIP("192.0.2.1")
	nat64 := net.ParseIP("64:ff9b::c000:201")

	tests := []struct {
		name      string
		allow     bool
		network   string
		v4Error   error
		nat64Err  error
		wantNAT64 bool
	}{
		{
			name:      "IPv4 has no route",
			allow:     true,
			network:   "tcp",
			v4Error:   syscall.ENETUNREACH,
			wantNAT64: true,
		},
		{
			name:      "NAT64 dial fails",
			allow:     true,
			network:   "tcp",
			v4Error:   syscall.ENETUNREACH,
			nat64Err:  syscall.ECONNREFUSED,
			wantNAT64: true,
		},
		{
			name:    "NAT64 disabled",
			network: "tcp",
			v4Error: syscall.ENETUNREACH,
		},
		{
			name:    "IPv4 refused",
			allow:   true,
			network: "tcp",
			v4Error: syscall.ECONNREFUSED,
		},
		{
			name:    "explicit IPv4 network",
			allow:   true,
			network: "tcp4",
			v4Error: syscall.ENETUNREACH,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var attempts []string
			resolver := New(WithDialer(func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, _, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				attempts = append(attempts, host)
				if net.ParseIP(host).Equal(nat64) {
					if tc.nat64Err != nil {
						return nil, &net.OpError{Op: "dial", Net: network, Err: tc.nat64Err}
					}
					return &mockConn{}, nil
				}
				return nil, &net.OpError{Op: "dial", Net: network, Err: tc.v4Error}
			}))

			conn, err := resolver.dialParallelWithNAT64Fallback(context.Background(), tc.network, "443", []net.IP{v4}, nil, tc.allow)
			if tc.wantNAT64 && tc.nat64Err == nil {
				require.NoError(t, err)
				require.NotNil(t, conn)
				require.NoError(t, conn.Close())
			} else {
				require.Nil(t, conn)
				require.ErrorIs(t, err, tc.v4Error)
			}
			require.Contains(t, attempts, v4.String())
			if tc.wantNAT64 {
				require.Contains(t, attempts, nat64.String())
			} else {
				require.NotContains(t, attempts, nat64.String())
			}
		})
	}
}
