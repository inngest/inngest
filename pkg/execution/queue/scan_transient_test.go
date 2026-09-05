package queue

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsTransientScanError covers the failure modes that previously took the
// whole server down. The production signature is a socket read deadline from
// the Redis state store, wrapped by the scan call chain, which prints as:
//
//	error scanning continuations: error leasing partition: i/o timeout
//
// os.ErrDeadlineExceeded is a different sentinel than context.DeadlineExceeded,
// so the original errors.Is check missed it and fell through to the fatal path.
func TestIsTransientScanError(t *testing.T) {
	// The exact shape the net package produces for a socket read deadline.
	socketTimeout := &net.OpError{
		Op:  "read",
		Net: "tcp",
		Err: os.ErrDeadlineExceeded,
	}
	require.Equal(t, "i/o timeout", os.ErrDeadlineExceeded.Error(),
		"guard: the production log line is this sentinel's message")

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "socket read deadline, wrapped by the scan call chain",
			err: fmt.Errorf("error scanning continuations: %w",
				fmt.Errorf("error leasing partition: %w", socketTimeout)),
			want: true,
		},
		{"bare socket read deadline", socketTimeout, true},
		{"os.ErrDeadlineExceeded sentinel", os.ErrDeadlineExceeded, true},
		{"context deadline still retried", context.DeadlineExceeded, true},
		{
			name: "wrapped context deadline",
			err:  fmt.Errorf("could not peek global partitions: %w", context.DeadlineExceeded),
			want: true,
		},
		{
			name: "connection reset by peer",
			err:  &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET},
			want: true,
		},
		{
			name: "connection refused",
			err:  &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED},
			want: true,
		},
		{
			// SERVFAIL from a resolver hiccup. This is the shape actually
			// observed when a container network's DNS went bad for ~30 minutes.
			name: "transient DNS failure (SERVFAIL sets IsTemporary)",
			err: &net.DNSError{
				Err:         "server misbehaving",
				Name:        "example.redis.azure.net",
				IsTemporary: true,
			},
			want: true,
		},
		{
			name: "DNS timeout",
			err: &net.DNSError{
				Err:       "i/o timeout",
				Name:      "example.redis.azure.net",
				IsTimeout: true,
			},
			want: true,
		},
		// A permanent NXDOMAIN means the endpoint is misconfigured or gone.
		// Retrying forever would keep the process up and passing health checks
		// while it can never reach its state store, which hides the fault
		// instead of surfacing it.
		{
			name: "permanent DNS not-found is NOT transient",
			err: &net.DNSError{
				Err:        "no such host",
				Name:       "typo.redis.azure.net",
				IsNotFound: true,
			},
			want: false,
		},
		{
			name: "wrapped permanent DNS not-found is NOT transient",
			err: fmt.Errorf("could not peek global partitions: %w",
				&net.DNSError{Err: "no such host", Name: "typo.redis.azure.net", IsNotFound: true}),
			want: false,
		},
		// Cancellation is a shutdown signal, not a transient fault: the loop
		// must still exit so a real SIGTERM drains cleanly.
		{"context canceled is not transient", context.Canceled, false},
		{"application error is not transient", errors.New("malformed partition"), false},
		{"nil", nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isTransientScanError(tc.err))
		})
	}
}
