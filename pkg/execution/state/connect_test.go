package state

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/syscode"
	"github.com/stretchr/testify/require"
)

type customError struct {
	inner error
}

func (c *customError) Error() string {
	return c.inner.Error()
}

func TestIsConnectWorkerAtCapacityCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "CodeConnectAllWorkersAtCapacity returns true",
			code:     syscode.CodeConnectAllWorkersAtCapacity,
			expected: true,
		},
		{
			name:     "CodeConnectRequestAssignWorkerReachedCapacity returns true",
			code:     syscode.CodeConnectRequestAssignWorkerReachedCapacity,
			expected: true,
		},
		{
			name:     "empty string returns false",
			code:     "",
			expected: false,
		},
		{
			name:     "unrelated error code returns false",
			code:     "some_other_error_code",
			expected: false,
		},
		{
			name:     "partial match returns false",
			code:     "connect_all_workers",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConnectWorkerAtCapacityCode(tt.code)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsConnectWorkerAtCapacityError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error returns false",
			err:      nil,
			expected: false,
		},
		{
			name:     "ErrConnectWorkerCapacity returns true",
			err:      ErrConnectWorkerCapacity,
			expected: true,
		},
		{
			name:     "wrapped ErrConnectWorkerCapacity returns true",
			err:      fmt.Errorf("wrapped error: %w", ErrConnectWorkerCapacity),
			expected: true,
		},
		{
			name:     "error with CodeConnectAllWorkersAtCapacity message returns true",
			err:      errors.New(syscode.CodeConnectAllWorkersAtCapacity),
			expected: true,
		},
		{
			name:     "error with CodeConnectRequestAssignWorkerReachedCapacity message returns true",
			err:      errors.New(syscode.CodeConnectRequestAssignWorkerReachedCapacity),
			expected: true,
		},
		{
			name:     "unrelated error returns false",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "error with partial match returns false",
			err:      errors.New("connect_all_workers"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConnectWorkerAtCapacityError(tt.err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsConnectWorkerAtCapacityError_EdgeCases(t *testing.T) {
	t.Run("error with capacity code in the middle of message", func(t *testing.T) {
		err := errors.New("prefix " + syscode.CodeConnectAllWorkersAtCapacity + " suffix")
		// This should return false because Error() returns the full message, not just the code
		require.False(t, IsConnectWorkerAtCapacityError(err))
	})

	t.Run("custom error type that wraps ErrConnectWorkerCapacity", func(t *testing.T) {
		customErr := &customError{inner: ErrConnectWorkerCapacity}

		// This should return false because errors.Is only works if Unwrap is implemented
		// Since customError doesn't implement Unwrap(), errors.Is() will return false
		require.False(t, IsConnectWorkerAtCapacityError(customErr))
	})
}

func TestIsConnectNoHealthyConnectionCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "CodeConnectNoHealthyConnection returns true",
			code:     syscode.CodeConnectNoHealthyConnection,
			expected: true,
		},
		{
			name:     "CodeConnectWorkerStoppedResponding returns false",
			code:     syscode.CodeConnectWorkerStoppedResponding,
			expected: false,
		},
		{
			name:     "capacity code returns false",
			code:     syscode.CodeConnectAllWorkersAtCapacity,
			expected: false,
		},
		{
			name:     "empty string returns false",
			code:     "",
			expected: false,
		},
		{
			name:     "partial match returns false",
			code:     "connect_no_healthy",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConnectNoHealthyConnectionCode(tt.code)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsConnectNoHealthyConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error returns false",
			err:      nil,
			expected: false,
		},
		{
			name:     "ErrConnectNoHealthyConnection returns true",
			err:      ErrConnectNoHealthyConnection,
			expected: true,
		},
		{
			name:     "wrapped ErrConnectNoHealthyConnection returns true",
			err:      fmt.Errorf("wrapped error: %w", ErrConnectNoHealthyConnection),
			expected: true,
		},
		{
			name:     "error with CodeConnectNoHealthyConnection message returns true",
			err:      errors.New(syscode.CodeConnectNoHealthyConnection),
			expected: true,
		},
		{
			name:     "error with CodeConnectWorkerStoppedResponding message returns false",
			err:      errors.New(syscode.CodeConnectWorkerStoppedResponding),
			expected: false,
		},
		{
			name:     "ErrConnectWorkerCapacity returns false",
			err:      ErrConnectWorkerCapacity,
			expected: false,
		},
		{
			name:     "unrelated error returns false",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConnectNoHealthyConnectionError(tt.err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestConnectNoHealthyConnectionRetryAt(t *testing.T) {
	now := time.Now()
	minAt := now.Add(ConnectNoHealthyConnectionRetryDelay)
	maxAt := minAt.Add(ConnectNoHealthyConnectionRetryJitter)

	for range 1000 {
		at := ConnectNoHealthyConnectionRetryAt(now)
		require.False(t, at.Before(minAt), "retry at %s is before the fixed delay", at)
		require.True(t, at.Before(maxAt), "retry at %s exceeds the jitter window", at)
	}
}
