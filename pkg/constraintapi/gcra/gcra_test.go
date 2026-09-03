package gcra

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// the expectations below are the values lua/helper/gcra.lua returns for the
// same inputs, taken from the Lua tables in constraintapi/gcra_test.go and
// worked by hand for the non divisible and past TAT cases.

func TestThrottle(t *testing.T) {
	now := float64(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli())
	minute := float64(time.Minute.Milliseconds())

	t.Run("empty state simulate", func(t *testing.T) {
		res, _, _, store := Throttle(0, false, now, minute, 10, 0, 0)
		require.False(t, store)
		require.False(t, res.Limited)
		require.Equal(t, 6000.0, res.EmissionInterval)
		require.Equal(t, now, res.TAT)
		require.Equal(t, now+6000, res.NewTAT)
		require.Equal(t, 6000.0, res.DVT)
		require.Equal(t, 6000.0, res.Increment)
		require.Equal(t, now, res.AllowAt)
		require.Equal(t, 0.0, res.Diff)
		require.Equal(t, 1, res.Limit)
		require.Equal(t, 1, res.Remaining)
		require.Equal(t, 0.0, res.ResetAfter)
		require.Equal(t, 0.0, res.RetryAfter)
		require.Equal(t, now+6000, res.RetryAt)
		require.Equal(t, int64(0), res.Usage)
	})

	t.Run("consume one stores tat", func(t *testing.T) {
		res, newTAT, ttl, store := Throttle(0, false, now, minute, 10, 0, 1)
		require.True(t, store)
		require.False(t, res.Limited)
		require.Equal(t, now+6000, newTAT)
		require.Equal(t, int64(6), ttl)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, 6000.0, res.ResetAfter)
		require.Equal(t, int64(1), res.Usage)
		require.Equal(t, now+6000, res.RetryAt)
	})

	t.Run("consume one with burst one", func(t *testing.T) {
		res, newTAT, ttl, store := Throttle(0, false, now, minute, 10, 1, 1)
		require.True(t, store)
		require.Equal(t, 2, res.Limit)
		require.Equal(t, 12000.0, res.DVT)
		require.Equal(t, now+6000, newTAT)
		require.Equal(t, int64(6), ttl)
		require.Equal(t, 1, res.Remaining)
	})

	t.Run("second request within emission is limited", func(t *testing.T) {
		_, tat, _, _ := Throttle(0, false, now, minute, 10, 0, 1)
		res, _, _, store := Throttle(tat, true, now+1000, minute, 10, 0, 1)
		require.False(t, store, "the limited path never writes")
		require.True(t, res.Limited)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, -5000.0, res.Diff)
		require.Equal(t, 5000.0, res.RetryAfter)
		require.Equal(t, now+6000, res.RetryAt)
		require.Equal(t, int64(1), res.Usage)
	})

	t.Run("limit does not divide period", func(t *testing.T) {
		emission := minute / 7
		res, newTAT, ttl, store := Throttle(0, false, now, minute, 7, 0, 1)
		require.True(t, store)
		require.Equal(t, emission, res.EmissionInterval)
		require.Equal(t, now+emission, newTAT)
		require.Equal(t, int64(8), ttl, "ttl is %%d of ttl/1000, which truncates 8.571")
		require.Equal(t, now+emission, res.RetryAt)

		// the fraction must survive a round trip through the stored TAT.  at
		// millisecond magnitude a double resolves to about a microsecond, so
		// now + emission - now is not exactly emission, in Lua or here.
		res2, _, _, _ := Throttle(newTAT, true, now, minute, 7, 0, 0)
		require.Equal(t, newTAT, res2.TAT)
		require.InDelta(t, emission, res2.ResetAfter, 1e-3)
	})

	t.Run("tat in the past inflates remaining", func(t *testing.T) {
		// a TAT 10s in the past gives a negative ttl.  next = dvt - ttl is
		// 16000 and remaining is floor(16000 / 6000) = 2, above burst + 1.
		// acquire pass two then runs with quantity 2 and hits the limited
		// path without writing.  both backends keep this.
		tat := now - 10000
		res, _, _, store := Throttle(tat, true, now, minute, 10, 0, 0)
		require.False(t, store)
		require.False(t, res.Limited)
		require.Equal(t, 2, res.Remaining)
		require.Equal(t, int64(-1), res.Usage)
		require.Equal(t, -10000.0, res.ResetAfter)

		res2, _, _, store2 := Throttle(tat, true, now, minute, 10, 0, 2)
		require.False(t, store2)
		require.True(t, res2.Limited)
		require.Equal(t, 0, res2.Remaining)
		require.Equal(t, now+6000, res2.RetryAt, "increment exceeds dvt so retry_at keeps the default")
	})

	t.Run("limit below one is clamped", func(t *testing.T) {
		res, _, _, _ := Throttle(0, false, now, minute, 0, 0, 0)
		require.Equal(t, minute, res.EmissionInterval)
	})

	t.Run("ttl floor is one second", func(t *testing.T) {
		_, _, ttl, store := Throttle(0, false, now, 100, 1000, 0, 1)
		require.True(t, store)
		require.Equal(t, int64(1), ttl)
	})
}

func TestRateLimit(t *testing.T) {
	now := float64(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano())
	minute := float64(time.Minute.Nanoseconds())

	t.Run("empty state simulate", func(t *testing.T) {
		res, _, _, store := RateLimit(0, false, now, minute, 10, 0, 0)
		require.False(t, store)
		require.Equal(t, float64(6*time.Second), res.EmissionInterval)
		require.Equal(t, 1, res.Remaining)
		require.Equal(t, now+float64(6*time.Second), res.RetryAt)
	})

	t.Run("consume one stores tat with ttl in seconds", func(t *testing.T) {
		res, newTAT, ttl, store := RateLimit(0, false, now, minute, 10, 0, 1)
		require.True(t, store)
		require.Equal(t, now+float64(6*time.Second), newTAT)
		require.Equal(t, int64(6), ttl)
		require.Equal(t, 0, res.Remaining)
	})

	t.Run("burst is limit over ten", func(t *testing.T) {
		// callers pass burst = limit / 10, the helper only adds one
		res, _, _, _ := RateLimit(0, false, now, minute, 120, 12, 0)
		require.Equal(t, 13, res.Limit)
		require.Equal(t, 13, res.Remaining)
		require.Equal(t, math.Floor(minute/120*13), res.DVT)
	})

	t.Run("limit does not divide period", func(t *testing.T) {
		emission := minute / 7
		_, newTAT, _, _ := RateLimit(0, false, now, minute, 7, 0, 1)
		require.Equal(t, now+emission, newTAT)
	})
}
