package state

import (
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/require"
)

func TestWorkerCapacity_IsUnlimited(t *testing.T) {
	tests := []struct {
		name     string
		capacity WorkerCapacity
		expected bool
	}{
		{
			name: "returns true when Total is 0",
			capacity: WorkerCapacity{
				Total:     0,
				Available: 5,
			},
			expected: true,
		},
		{
			name: "returns false when Total is positive",
			capacity: WorkerCapacity{
				Total:     10,
				Available: 5,
			},
			expected: false,
		},
		{
			name: "returns true when Total is negative",
			capacity: WorkerCapacity{
				Total:     -1,
				Available: 5,
			},
			expected: true,
		},
		{
			name: "returns true when both Total and Available are 0",
			capacity: WorkerCapacity{
				Total:     0,
				Available: 0,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.capacity.IsUnlimited()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkerCapacity_IsAvailable(t *testing.T) {
	tests := []struct {
		name     string
		capacity WorkerCapacity
		expected bool
	}{
		{
			name: "returns true when Available is positive",
			capacity: WorkerCapacity{
				Total:     10,
				Available: 5,
			},
			expected: true,
		},
		{
			name: "returns false when Available is 0",
			capacity: WorkerCapacity{
				Total:     10,
				Available: 0,
			},
			expected: false,
		},
		{
			name: "returns true when Available equals no concurrency limit constant",
			capacity: WorkerCapacity{
				Total:     10,
				Available: consts.ConnectWorkerCapacityForNoConcurrencyLimit,
			},
			expected: true,
		},
		{
			name: "returns false when Available is negative but not the special constant",
			capacity: WorkerCapacity{
				Total:     10,
				Available: -2,
			},
			expected: false,
		},
		{
			name: "returns true when Available is 1",
			capacity: WorkerCapacity{
				Total:     10,
				Available: 1,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.capacity.IsAvailable()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkerCapacity_IsAtCapacity(t *testing.T) {
	tests := []struct {
		name     string
		capacity WorkerCapacity
		expected bool
	}{
		{
			name: "returns true when Available is 0",
			capacity: WorkerCapacity{
				Total:     10,
				Available: 0,
			},
			expected: true,
		},
		{
			name: "returns false when Available is positive",
			capacity: WorkerCapacity{
				Total:     10,
				Available: 5,
			},
			expected: false,
		},
		{
			name: "returns false when Available is negative",
			capacity: WorkerCapacity{
				Total:     10,
				Available: -1,
			},
			expected: false,
		},
		{
			name: "returns false when Available equals no concurrency limit constant",
			capacity: WorkerCapacity{
				Total:     10,
				Available: consts.ConnectWorkerCapacityForNoConcurrencyLimit,
			},
			expected: false,
		},
		{
			name: "returns false when both Total and Available are 0",
			capacity: WorkerCapacity{
				Total:     0,
				Available: 0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.capacity.IsAtCapacity()
			require.Equal(t, tt.expected, result)
		})
	}
}
