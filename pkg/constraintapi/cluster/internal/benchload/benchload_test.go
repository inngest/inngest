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

// TestWatch checks that watch returns when the measured time ends, records
// the longest late wake of the measured time, and records nothing for a
// sleep that began before it.
func TestWatch(t *testing.T) {
	begin := time.Now()
	parts := make([]part, 1)
	watch(parts, begin, begin.Add(30*time.Millisecond))
	took := time.Since(begin)
	require.GreaterOrEqual(t, took, 30*time.Millisecond)
	require.Less(t, took, 5*time.Second)
	require.Less(t, parts[0].late.Load(), int64(5*time.Second))

	parts[0].late.Store(0)
	parts[0].wokeLate(5 * time.Millisecond)
	parts[0].wokeLate(3 * time.Millisecond)
	require.EqualValues(t, 5*time.Millisecond, parts[0].late.Load())
	parts[0].wokeLate(9 * time.Millisecond)
	require.EqualValues(t, 9*time.Millisecond, parts[0].late.Load())

	// the measured time has not begun, so no sleep counts.
	begin = time.Now()
	parts = make([]part, 1)
	watch(parts, begin.Add(time.Hour), begin.Add(20*time.Millisecond))
	require.Zero(t, parts[0].late.Load())
}

// TestLogLimit checks that one second allows slowLogsPerSecond lines and
// that the next second allows them again.
func TestLogLimit(t *testing.T) {
	var l logLimit
	at := time.Unix(1_000, 0)
	for i := range 3 * slowLogsPerSecond {
		require.Equal(t, i < slowLogsPerSecond, l.allow(at.Add(time.Duration(i)*time.Millisecond)), "line %d", i)
	}
	require.True(t, l.allow(at.Add(time.Second)))
}
