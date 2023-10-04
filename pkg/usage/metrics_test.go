package usage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetricsRequestGranularity(t *testing.T) {
	tests := []struct {
		name     string
		req      *MetricsRequest
		expected string
	}{
		{
			name:     "empty req defaults to 5m",
			req:      &MetricsRequest{},
			expected: "5m",
		},
		{
			name:     ">= 1h range returns 5m granularity",
			req:      &MetricsRequest{From: time.Now().Add(-2 * time.Hour), To: time.Now()},
			expected: "5m",
		},
		{
			name:     ">= 6h range returns 10m granularity",
			req:      &MetricsRequest{From: time.Now().Add(-7 * time.Hour), To: time.Now()},
			expected: "10m",
		},
		{
			name:     ">= 12h range returns 15m granularity",
			req:      &MetricsRequest{From: time.Now().Add(-12 * time.Hour), To: time.Now()},
			expected: "15m",
		},
		{
			name:     ">= 1d range returns 30m granularity",
			req:      &MetricsRequest{From: time.Now().Add(-24 * time.Hour), To: time.Now()},
			expected: "30m",
		},
		{
			name:     ">= 3d range returns 1h granularity",
			req:      &MetricsRequest{From: time.Now().Add(4 * -24 * time.Hour), To: time.Now()},
			expected: "1h",
		},
		{
			name:     ">= 7d returns 3h granularity",
			req:      &MetricsRequest{From: time.Now().Add(8 * -24 * time.Hour), To: time.Now()},
			expected: "3h",
		},
		{
			name:     ">= 14d range returns 6h granularity",
			req:      &MetricsRequest{From: time.Now().Add(15 * -24 * time.Hour), To: time.Now()},
			expected: "6h",
		},
		{
			name:     ">= 30d range returns 12h granularity",
			req:      &MetricsRequest{From: time.Now().Add(31 * -24 * time.Hour), To: time.Now()},
			expected: "12h",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.req.Granularity())
		})
	}
}
