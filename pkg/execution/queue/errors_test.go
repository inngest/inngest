package queue

import (
	"fmt"
	"io"
	"net"
	"syscall"
	"testing"
)

func TestIsTransientDBError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"EOF", io.EOF, true},
		{"unexpected EOF", io.ErrUnexpectedEOF, true},
		{"connection refused", syscall.ECONNREFUSED, true},
		{"connection reset", syscall.ECONNRESET, true},
		{"broken pipe", syscall.EPIPE, true},
		{"wrapped connection refused", fmt.Errorf("could not peek: %w", syscall.ECONNREFUSED), true},
		{"net.OpError", &net.OpError{Op: "dial", Err: fmt.Errorf("connection refused")}, true},
		{"string match connection refused", fmt.Errorf("dial tcp 127.0.0.1:5432: connection refused"), true},
		{"string match driver bad connection", fmt.Errorf("driver: bad connection"), true},
		{"non-transient error", fmt.Errorf("partition not found"), false},
		{"non-transient queue error", ErrPartitionNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTransientDBError(tt.err)
			if got != tt.expected {
				t.Errorf("IsTransientDBError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}
