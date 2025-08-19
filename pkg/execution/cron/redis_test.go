package cron

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGenerateJitter(t *testing.T) {
	testcases := []struct {
		name      string
		min       time.Duration
		max       time.Duration
		expectMin time.Duration
		expectMax time.Duration
	}{
		{
			name:      "normal range 1-3 seconds",
			min:       1000 * time.Millisecond,
			max:       3000 * time.Millisecond,
			expectMin: 1000 * time.Millisecond,
			expectMax: 3000 * time.Millisecond,
		},
		{
			name:      "equal min and max",
			min:       2000 * time.Millisecond,
			max:       2000 * time.Millisecond,
			expectMin: 2000 * time.Millisecond,
			expectMax: 2000 * time.Millisecond,
		},
		{
			name:      "small range",
			min:       1000 * time.Millisecond,
			max:       1001 * time.Millisecond,
			expectMin: 1000 * time.Millisecond,
			expectMax: 1001 * time.Millisecond,
		},
		{
			name:      "min greater than max returns zero",
			min:       5000 * time.Millisecond,
			max:       1000 * time.Millisecond,
			expectMin: 0,
			expectMax: 0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.min > tc.max {
				// Special case: should return zero
				jitter := generateJitter(tc.min, tc.max)
				require.Equal(t, time.Duration(0), jitter)
				return
			}

			// Run multiple times to test range
			for i := 0; i < 50; i++ {
				jitter := generateJitter(tc.min, tc.max)
				
				require.GreaterOrEqual(t, jitter, tc.expectMin, "jitter should be >= min")
				require.LessOrEqual(t, jitter, tc.expectMax, "jitter should be <= max")
			}

			// Test for some variation (unless min == max)
			if tc.min != tc.max {
				values := make(map[time.Duration]bool)
				for i := 0; i < 20; i++ {
					jitter := generateJitter(tc.min, tc.max)
					values[jitter] = true
				}
				require.Greater(t, len(values), 1, "expected some variation in jitter values")
			}
		})
	}
}