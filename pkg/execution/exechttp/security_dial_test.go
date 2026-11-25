package exechttp

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/inngest/inngest/pkg/execution/driver/httpdriver/dnscache"
	"github.com/stretchr/testify/require"
)

var errDialed = errors.New("dialed")

// stubResolver answers every lookup with one address and records what is
// dialed.  its Dialer methods do their own lookup, which SecureDialer must not use.
type stubResolver struct {
	ip     net.IP
	dialed []net.IP
}

func (s *stubResolver) Lookup(ctx context.Context, host string) ([]net.IP, error) {
	return []net.IP{s.ip}, nil
}

func (s *stubResolver) Dialer() dnscache.Dialer {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		panic("SecureDialer must dial the checked addresses, not look up again")
	}
}

func (s *stubResolver) DialerWithNAT64Fallback() dnscache.Dialer { return s.Dialer() }

func (s *stubResolver) DialIPs(ctx context.Context, network, port string, ips []net.IP, allowNAT64Fallback bool) (net.Conn, error) {
	s.dialed = append(s.dialed, ips...)
	return nil, errDialed
}

func TestSecureDialerDialsCheckedAddresses(t *testing.T) {
	tests := []struct {
		name       string
		ip         string
		wantDialed []string
		wantErr    error
	}{
		{name: "public address is dialed as checked", ip: "203.0.113.10", wantDialed: []string{"203.0.113.10"}, wantErr: errDialed},
		{name: "private address is refused before dialing", ip: "10.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &stubResolver{ip: net.ParseIP(tt.ip)}
			dial := SecureDialer(SecureDialerOpts{resolver: r})

			_, err := dial(context.Background(), "tcp", "api.example.com:443")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.ErrorContains(t, err, "private IP range")
			}

			var dialed []string
			for _, ip := range r.dialed {
				dialed = append(dialed, ip.String())
			}
			require.Equal(t, tt.wantDialed, dialed)
		})
	}
}
