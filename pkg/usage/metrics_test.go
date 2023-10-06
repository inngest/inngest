package usage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetricsRequestValid(t *testing.T) {
	tests := []struct {
		name string
		req  *MetricsRequest
		err  error
	}{
		{
			name: "valid request returns true",
			req: &MetricsRequest{
				Name: "something",
				From: time.Now().Add(-1 * time.Hour),
				To:   time.Now(),
			},
			err: nil,
		},
		{
			name: "missing name returns false",
			req:  &MetricsRequest{},
			err:  NoMetricsNameErr,
		},
		{
			name: "missing time range will returns false",
			req: &MetricsRequest{
				Name: "something",
			},
			err: NoMetricsTimeRangeErr,
		},
		{
			name: "opposite time range (to < from) returns false",
			req: &MetricsRequest{
				Name: "something",
				From: time.Now().Add(-1 * time.Hour),
				To:   time.Now().Add(-2 * time.Hour),
			},
			err: InvalidMetricsTimeRangeErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.err, test.req.Valid())
		})
	}
}

func TestMetricsRequestGranularity(t *testing.T) {
	tests := []struct {
		name     string
		req      *MetricsRequest
		expected time.Duration
	}{
		{
			name:     "empty req defaults to 1m",
			req:      &MetricsRequest{},
			expected: time.Minute,
		},
		{
			name:     ">= 3h range returns 5m granularity",
			req:      &MetricsRequest{From: time.Now().Add(-4 * time.Hour), To: time.Now()},
			expected: 5 * time.Minute,
		},
		{
			name:     ">= 6h range returns 10m granularity",
			req:      &MetricsRequest{From: time.Now().Add(-7 * time.Hour), To: time.Now()},
			expected: 10 * time.Minute,
		},
		{
			name:     ">= 12h range returns 15m granularity",
			req:      &MetricsRequest{From: time.Now().Add(-12 * time.Hour), To: time.Now()},
			expected: 15 * time.Minute,
		},
		{
			name:     ">= 1d range returns 30m granularity",
			req:      &MetricsRequest{From: time.Now().Add(-24 * time.Hour), To: time.Now()},
			expected: 30 * time.Minute,
		},
		{
			name:     ">= 3d range returns 1h granularity",
			req:      &MetricsRequest{From: time.Now().Add(4 * -24 * time.Hour), To: time.Now()},
			expected: time.Hour,
		},
		{
			name:     ">= 7d returns 3h granularity",
			req:      &MetricsRequest{From: time.Now().Add(8 * -24 * time.Hour), To: time.Now()},
			expected: 3 * time.Hour,
		},
		{
			name:     ">= 14d range returns 6h granularity",
			req:      &MetricsRequest{From: time.Now().Add(15 * -24 * time.Hour), To: time.Now()},
			expected: 6 * time.Hour,
		},
		{
			name:     ">= 30d range returns 12h granularity",
			req:      &MetricsRequest{From: time.Now().Add(31 * -24 * time.Hour), To: time.Now()},
			expected: 12 * time.Hour,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.req.Granularity())
		})
	}
}
