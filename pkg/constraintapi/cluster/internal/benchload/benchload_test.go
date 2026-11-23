package benchload

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestHistPercentile checks that a percentile is the upper edge of the
// bucket of the sample at that rank, at most 1.6% above the sample.
func TestHistPercentile(t *testing.T) {
	h := &hist{}
	for i := 1; i <= 1000; i++ {
		h.add(time.Duration(i) * time.Microsecond)
	}
	s := summarize(h)
	require.EqualValues(t, 1000, s.total)
	for _, c := range []struct {
		q    float64
		want float64
	}{
		{
			q:    0.5,
			want: 500,
		},
		{
			q:    0.99,
			want: 990,
		},
		{
			q:    1,
			want: 1000,
		},
	} {
		got := s.at(c.q)
		require.GreaterOrEqual(t, got, c.want, "quantile %v", c.q)
		require.LessOrEqual(t, got, c.want*1.016, "quantile %v", c.q)
	}

	// every duration is in a bucket whose upper edge is not below it.
	for _, d := range []time.Duration{0, 1, 63, 64, 127, 128, time.Millisecond, time.Hour} {
		one := &hist{}
		one.add(d)
		got := time.Duration(summarize(one).at(1) * 1e3)
		if d < 1<<40 {
			require.GreaterOrEqual(t, got, d)
			require.LessOrEqual(t, float64(got), float64(d)*1.016+1)
		}
	}
}
