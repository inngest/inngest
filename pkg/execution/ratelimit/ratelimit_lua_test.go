package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

const prefix = "{rl}:"

// initRedis creates both miniredis/rueidis for Lua, throttled store, and fake clock
func initRedis(t *testing.T) (*miniredis.Miniredis, rueidis.Client, clockwork.FakeClock) {
	r := miniredis.RunT(t)

	// Create rueidis client for Lua implementation
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	clock := clockwork.NewFakeClock()
	// Set miniredis time to match fake clock
	r.SetTime(clock.Now())

	return r, rc, clock
}

func TestLuaRateLimit_BasicFunctionality(t *testing.T) {
	ctx := context.Background()

	t.Run("should allow requests under limit", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, prefix)

		config := inngest.RateLimit{
			Limit:  5,
			Period: "1h",
		}

		// First request should be allowed (not limited)
		r.SetTime(clock.Now())
		res, err := limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, res.Limited)
		require.Equal(t, time.Duration(0), res.RetryAfter)

		// Should have created a key in Redis
		require.Len(t, r.Keys(), 1)
	})

	t.Run("should rate limit when over limit", func(t *testing.T) {
		_, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  1,
			Period: "1h",
		}

		// First request should be allowed (not limited)
		res, err := limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, res.Limited)

		// Second request should be rate limited
		res, err = limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, res.Limited)
		require.Greater(t, res.RetryAfter, time.Duration(0))
	})

	t.Run("should handle burst correctly", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		// 10 requests per hour with burst of 1 (10/10)
		config := inngest.RateLimit{
			Limit:  10,
			Period: "1h",
		}

		key := "test-burst"

		// Should be able to make burst + limit requests initially
		// In throttled library: maxBurst = limit/10, capacity = maxBurst + 1
		// So for limit=10: maxBurst=1, capacity=2 total (1 burst + 1 base)
		for i := 0; i < 2; i++ {
			r.SetTime(clock.Now())
			res, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
			require.NoError(t, err)
			require.False(t, res.Limited, "request %d should be allowed (not limited)", i+1)
		}

		// Next request should be rate limited
		r.SetTime(clock.Now())
		res, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, res.Limited)
		require.Greater(t, res.RetryAfter, time.Duration(0))
	})
}

func TestLuaRateLimit_ExistingVsFreshState(t *testing.T) {
	ctx := context.Background()

	t.Run("existing state hits limits faster than fresh state", func(t *testing.T) {
		config := inngest.RateLimit{
			Limit:  10, // This gives us a small, predictable capacity for testing
			Period: "1h",
		}

		// Test with same Lua implementation but different states
		// Scenario 1: Fresh state
		r1, rc1, clock1 := initRedis(t)
		defer rc1.Close()

		luaLimiterFresh := New(ctx, rc1, "{rl}:")
		keyFresh := "fresh-state-test"

		// Count how many requests fresh state allows
		var freshResults []bool
		for i := 0; i < 5; i++ { // Try enough to hit limits
			r1.SetTime(clock1.Now())
			res, err := luaLimiterFresh.RateLimit(ctx, keyFresh, config, WithNow(clock1.Now()))
			require.NoError(t, err)
			freshResults = append(freshResults, !res.Limited) // Store as "allowed" for logic consistency
			t.Logf("Fresh state - request %d: limited=%v, retry=%v", i+1, res.Limited, res.RetryAfter)
			if res.Limited {
				break // Stop at first rate limit
			}
		}

		allowedFreshCount := len(freshResults) - 1 // Subtract the rate limited one
		if len(freshResults) > 0 && freshResults[len(freshResults)-1] {
			allowedFreshCount = len(freshResults) // All were allowed
		}

		t.Logf("Fresh state allowed %d requests before hitting limit", allowedFreshCount)

		// Scenario 2: Create existing state by pre-consuming some capacity
		r2, rc2, clock2 := initRedis(t)
		defer rc2.Close()

		luaLimiterExisting := New(ctx, rc2, "{rl}:")
		keyExisting := "existing-state-test"

		// Pre-consume 1 request to create existing state
		r2.SetTime(clock2.Now())
		preRes, err := luaLimiterExisting.RateLimit(ctx, keyExisting, config, WithNow(clock2.Now()))
		require.NoError(t, err)
		require.False(t, preRes.Limited)
		t.Logf("Pre-consumed 1 request to create existing state")

		// Now test how many MORE requests this existing state allows
		var existingResults []bool
		for i := 0; i < 5; i++ { // Try enough to hit limits
			r2.SetTime(clock2.Now())
			res, err := luaLimiterExisting.RateLimit(ctx, keyExisting, config, WithNow(clock2.Now()))
			require.NoError(t, err)
			existingResults = append(existingResults, !res.Limited) // Store as "allowed" for logic consistency
			t.Logf("Existing state - request %d: limited=%v, retry=%v", i+1, res.Limited, res.RetryAfter)
			if res.Limited {
				break // Stop at first rate limit
			}
		}

		allowedExistingCount := len(existingResults) - 1 // Subtract the rate limited one
		if len(existingResults) > 0 && existingResults[len(existingResults)-1] {
			allowedExistingCount = len(existingResults) // All were allowed
		}

		t.Logf("Existing state (after pre-consuming 1) allowed %d more requests", allowedExistingCount)

		// Key insight: existing state should allow fewer additional requests than fresh state total
		t.Logf("Comparison: fresh state=%d total, existing state=%d additional (after pre-consuming 1)",
			allowedFreshCount, allowedExistingCount)

		require.Less(t, allowedExistingCount, allowedFreshCount,
			"Existing state should allow fewer additional requests than fresh state total - this proves existing state affects rate limiting speed")

		// Total requests for existing state should equal fresh state (1 pre + remaining = total)
		totalExisting := 1 + allowedExistingCount
		require.Equal(t, allowedFreshCount, totalExisting,
			"Total capacity should be the same, but distributed differently")
	})
}

func TestLuaRateLimit_RetryAfterValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("retryAfter timing accuracy", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  1,
			Period: "2s", // Short period for faster testing
		}

		key := "timing-accuracy-test"

		// First request should be allowed
		r.SetTime(clock.Now())
		res1, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, res1.Limited)
		require.Equal(t, time.Duration(0), res1.RetryAfter)

		// Second request should be rate limited with retryAfter
		startTime := clock.Now()
		r.SetTime(clock.Now())
		res2, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, res2.Limited)
		require.Greater(t, res2.RetryAfter, time.Duration(0))

		t.Logf("Rate limited with retryAfter: %v", res2.RetryAfter)

		// Verify retryAfter is reasonable (should be close to 2 seconds)
		require.Greater(t, res2.RetryAfter, 1500*time.Millisecond, "retryAfter should be at least 1.5s")
		require.Less(t, res2.RetryAfter, 2500*time.Millisecond, "retryAfter should be less than 2.5s")

		// Wait for the retryAfter duration (minus small buffer for test execution time)
		waitDuration := res2.RetryAfter - 50*time.Millisecond
		if waitDuration > 0 {
			t.Logf("Advancing time by %v before next attempt", waitDuration)
			clock.Advance(waitDuration)
			r.FastForward(waitDuration)
		}

		// Request should still be rate limited (slightly early)
		r.SetTime(clock.Now())
		res3, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, res3.Limited)
		require.Greater(t, res3.RetryAfter, time.Duration(0))
		require.Less(t, res3.RetryAfter, 100*time.Millisecond, "Should have very short retryAfter")

		// Advance time a bit more and request should be allowed
		clock.Advance(100 * time.Millisecond)
		r.FastForward(100 * time.Millisecond)
		r.SetTime(clock.Now())
		res4, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, res4.Limited)
		require.Equal(t, time.Duration(0), res4.RetryAfter)

		// Verify total elapsed time is reasonable
		elapsed := clock.Since(startTime)
		t.Logf("Total test elapsed time: %v", elapsed)
		require.Greater(t, elapsed, 1900*time.Millisecond, "Should take close to 2 seconds")
		require.Less(t, elapsed, 2500*time.Millisecond, "Should not take much more than 2 seconds")
	})

	t.Run("retryAfter decreases over time", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  1,
			Period: "3s",
		}

		key := "decreasing-retry-test"

		// Exhaust capacity
		r.SetTime(clock.Now())
		res1, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, res1.Limited)

		// Get initial retryAfter
		r.SetTime(clock.Now())
		res2, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.Greater(t, res2.RetryAfter, time.Duration(0))
		t.Logf("Initial retryAfter: %v", res2.RetryAfter)

		// Advance time by 500ms
		clock.Advance(500 * time.Millisecond)
		r.FastForward(500 * time.Millisecond)

		// Get retryAfter again - should be shorter
		r.SetTime(clock.Now())
		res3, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.Greater(t, res3.RetryAfter, time.Duration(0))
		t.Logf("RetryAfter after 500ms advance: %v", res3.RetryAfter)

		// Verify retryAfter decreased
		decrease := res2.RetryAfter - res3.RetryAfter
		t.Logf("RetryAfter decreased by: %v", decrease)
		require.Greater(t, decrease, 400*time.Millisecond, "Should decrease by approximately the wait time")
		require.Less(t, decrease, 600*time.Millisecond, "Should not decrease more than wait + tolerance")

		// Advance time by another 500ms
		clock.Advance(500 * time.Millisecond)
		r.FastForward(500 * time.Millisecond)

		// Get retryAfter again - should be even shorter
		r.SetTime(clock.Now())
		res4, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.Greater(t, res4.RetryAfter, time.Duration(0))
		t.Logf("RetryAfter after total 1s advance: %v", res4.RetryAfter)

		// Verify continued decrease
		require.Less(t, res4.RetryAfter, res3.RetryAfter, "RetryAfter should continue to decrease")

		totalDecrease := res2.RetryAfter - res4.RetryAfter
		require.Greater(t, totalDecrease, 900*time.Millisecond, "Should decrease by approximately 1s")
		require.Less(t, totalDecrease, 1100*time.Millisecond, "Should not decrease more than 1s + tolerance")
	})

	t.Run("retryAfter with different periods", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		testCases := []struct {
			name        string
			period      string
			expectedMin time.Duration
			expectedMax time.Duration
		}{
			{"1 second", "1s", 800 * time.Millisecond, 1200 * time.Millisecond},
			{"5 seconds", "5s", 4800 * time.Millisecond, 5200 * time.Millisecond},
			{"30 seconds", "30s", 29800 * time.Millisecond, 30200 * time.Millisecond},
			{"1 minute", "1m", 59800 * time.Millisecond, 60200 * time.Millisecond},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := inngest.RateLimit{
					Limit:  1,
					Period: tc.period,
				}

				key := fmt.Sprintf("period-test-%s", tc.name)

				// Exhaust capacity
				r.SetTime(clock.Now())
				res, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
				require.NoError(t, err)
				require.False(t, res.Limited)

				// Get retryAfter
				r.SetTime(clock.Now())
				res, err = limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
				require.NoError(t, err)
				require.True(t, res.Limited)
				require.Greater(t, res.RetryAfter, time.Duration(0))

				t.Logf("Period %s: retryAfter = %v", tc.period, res.RetryAfter)

				// Verify retryAfter is within expected range
				require.Greater(t, res.RetryAfter, tc.expectedMin,
					"retryAfter should be at least %v for period %s", tc.expectedMin, tc.period)
				require.Less(t, res.RetryAfter, tc.expectedMax,
					"retryAfter should be less than %v for period %s", tc.expectedMax, tc.period)
			})
		}
	})

	t.Run("retryAfter with burst capacity", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  10, // burst = 1, total = 2
			Period: "10s",
		}

		key := "burst-retry-test"

		// Consume all capacity (2 requests)
		for i := 0; i < 2; i++ {
			r.SetTime(clock.Now())
			res, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
			require.NoError(t, err)
			require.False(t, res.Limited)
			require.Equal(t, time.Duration(0), res.RetryAfter)
			t.Logf("Burst request %d: allowed", i+1)
		}

		// Next request should be rate limited
		startTime := clock.Now()
		r.SetTime(clock.Now())
		res, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, res.Limited)
		require.Greater(t, res.RetryAfter, time.Duration(0))

		t.Logf("After exhausting burst capacity, retryAfter: %v", res.RetryAfter)

		// For GCRA with burst, retryAfter should be based on emission interval
		// emission_interval = period / limit = 10s / 10 = 1s per request
		// After consuming 2 requests instantly, should wait ~1s for next slot
		require.Greater(t, res.RetryAfter, 500*time.Millisecond)
		require.Less(t, res.RetryAfter, 2000*time.Millisecond)

		// Verify retryAfter leads to successful request
		if res.RetryAfter < 5*time.Second { // Only wait if reasonable for test
			waitTime := res.RetryAfter + 50*time.Millisecond // Add small buffer
			t.Logf("Advancing time by %v for capacity recovery", waitTime)
			clock.Advance(waitTime)
			r.FastForward(waitTime)

			r.SetTime(clock.Now())
			res, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
			require.NoError(t, err)
			require.False(t, res.Limited, "Request should be allowed (not limited) after waiting retryAfter duration")

			elapsed := clock.Since(startTime)
			t.Logf("Successfully allowed request after %v", elapsed)
		}
	})

	t.Run("retryAfter with zero capacity edge case", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  0, // Zero limit should be converted to 1
			Period: "1h",
		}

		key := "zero-capacity-test"

		// First request should not be rate limited
		r.SetTime(clock.Now())
		res, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, res.Limited)
		require.Equal(t, time.Duration(0), res.RetryAfter)

		// Subsequent requests should be rate limited
		r.SetTime(clock.Now())
		res2, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, res2.Limited)
		require.Greater(t, res2.RetryAfter, time.Duration(0))

		// RetryAfter should not remain close to original (zero capacity = no progress)
		require.WithinDuration(
			t,
			time.Unix(0, int64(res2.RetryAfter)),
			time.Unix(0, int64(res.RetryAfter)).Add(time.Hour),
			100*time.Millisecond,
			"RetryAfter should be consistent for zero limit",
		)
	})

	t.Run("retryAfter precision validation", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  1,
			Period: "1s",
		}

		key := "precision-test"

		// Exhaust capacity
		r.SetTime(clock.Now())
		res, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, res.Limited)

		// Get multiple retryAfter values in quick succession
		var retryTimes []time.Duration

		for i := 0; i < 5; i++ {
			r.SetTime(clock.Now())
			res, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
			require.NoError(t, err)
			require.Greater(t, res.RetryAfter, time.Duration(0))
			retryTimes = append(retryTimes, res.RetryAfter)

			t.Logf("Retry %d: %v", i+1, res.RetryAfter)
			clock.Advance(10 * time.Millisecond) // Small time advance between calls
			r.FastForward(10 * time.Millisecond)
		}

		// Verify retryAfter values are decreasing (accounting for time passage)
		for i := 1; i < len(retryTimes); i++ {
			timeDiff := retryTimes[i-1] - retryTimes[i]
			t.Logf("Time difference %d->%d: %v", i, i+1, timeDiff)

			// Should decrease by approximately the sleep time between calls
			require.Greater(t, timeDiff, 5*time.Millisecond, "Should decrease between calls")
			require.Less(t, timeDiff, 20*time.Millisecond, "Should not decrease too much")
		}

		// All retry times should be reasonable
		for i, retry := range retryTimes {
			require.Greater(t, retry, 900*time.Millisecond-(time.Duration(i+1)*15*time.Millisecond),
				"Retry %d should be close to remaining period", i+1)
			require.Less(t, retry, 1100*time.Millisecond,
				"Retry %d should not exceed period significantly", i+1)
		}
	})
}

func TestLuaRateLimit_Idempotency(t *testing.T) {
	ctx := context.Background()

	t.Run("no idempotency baseline", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  2, // burst = 2/10 = 0, capacity = 0 + 1 = 1 request total
			Period: "10s",
		}

		key := "baseline-test"

		// First request should be allowed (no idempotency) - consumes the 1 available capacity
		r.SetTime(clock.Now())
		res1, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, res1.Limited)
		require.Equal(t, time.Duration(0), res1.RetryAfter)
		t.Logf("Request 1 (no idempotency): limited=%v, retry=%v", res1.Limited, res1.RetryAfter)

		// Second request should be rate limited (capacity exhausted)
		r.SetTime(clock.Now())
		res2, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, res2.Limited)
		require.Greater(t, res2.RetryAfter, time.Duration(0))
		t.Logf("Request 2 (no idempotency): limited=%v, retry=%v", res2.Limited, res2.RetryAfter)
	})

	t.Run("idempotency enforced after successful request", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  2, // burst = 2/10 = 0, capacity = 0 + 1 = 1 request total
			Period: "10s",
		}

		key := "idempotency-test"
		idempotencyKey := "request-123"
		idempotencyTTL := 30 * time.Second

		// First request with idempotency should be allowed and consume the 1 available capacity
		r.SetTime(clock.Now())
		res1, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, res1.Limited)
		require.Equal(t, time.Duration(0), res1.RetryAfter)
		t.Logf("First request with idempotency: limited=%v, retry=%v", res1.Limited, res1.RetryAfter)

		// Subsequent request with same idempotency key should be allowed WITHOUT consuming capacity (idempotency bypass)
		r.SetTime(clock.Now())
		res2, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, res2.Limited)
		require.Equal(t, time.Duration(0), res2.RetryAfter)
		t.Logf("Duplicate request with same idempotency key: limited=%v, retry=%v", res2.Limited, res2.RetryAfter)

		// Third request with same idempotency key should still be allowed (idempotency bypass)
		r.SetTime(clock.Now())
		res3, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, res3.Limited)
		require.Equal(t, time.Duration(0), res3.RetryAfter)
		t.Logf("Third request with same idempotency key: limited=%v, retry=%v", res3.Limited, res3.RetryAfter)

		// New request without idempotency should be rate limited (capacity already exhausted by first request)
		r.SetTime(clock.Now())
		res4, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, res4.Limited)
		require.Greater(t, res4.RetryAfter, time.Duration(0))
		t.Logf("New request without idempotency: limited=%v, retry=%v", res4.Limited, res4.RetryAfter)
	})

	t.Run("idempotency not enforced after rate limited request", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  1, // Only 1 request allowed
			Period: "10s",
		}

		key := "rate-limited-test"
		idempotencyKey := "request-456"
		idempotencyTTL := 30 * time.Second

		// First request without idempotency to consume capacity
		r.SetTime(clock.Now())
		res1, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, res1.Limited)
		require.Equal(t, time.Duration(0), res1.RetryAfter)
		t.Logf("Setup request (consume capacity): limited=%v, retry=%v", res1.Limited, res1.RetryAfter)

		// Request with idempotency should be rate limited
		r.SetTime(clock.Now())
		res2, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.True(t, res2.Limited)
		require.Greater(t, res2.RetryAfter, time.Duration(0))
		t.Logf("Rate limited request with idempotency: limited=%v, retry=%v", res2.Limited, res2.RetryAfter)

		// Subsequent request with same idempotency key should STILL be rate limited
		// (idempotency key should NOT be set for rate limited requests)
		r.SetTime(clock.Now())
		res3, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.True(t, res3.Limited)
		require.Greater(t, res3.RetryAfter, time.Duration(0))
		t.Logf("Retry with same idempotency key after rate limit: limited=%v, retry=%v", res3.Limited, res3.RetryAfter)

		// Advance time to allow capacity recovery
		waitTime := res2.RetryAfter + 100*time.Millisecond
		clock.Advance(waitTime)
		r.FastForward(waitTime)
		t.Logf("Advanced time by %v for capacity recovery", waitTime)

		// Now the request with idempotency should succeed
		r.SetTime(clock.Now())
		res4, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, res4.Limited)
		require.Equal(t, time.Duration(0), res4.RetryAfter)
		t.Logf("Request after recovery with idempotency: limited=%v, retry=%v", res4.Limited, res4.RetryAfter)

		// Subsequent request with same idempotency should now be allowed (idempotency enforced)
		r.SetTime(clock.Now())
		res5, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, res5.Limited)
		require.Equal(t, time.Duration(0), res5.RetryAfter)
		t.Logf("Duplicate after successful request: limited=%v, retry=%v", res5.Limited, res5.RetryAfter)
	})

	t.Run("idempotency no longer enforced once expired", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  3, // burst = 3/10 = 0, capacity = 0 + 1 = 1 request total
			Period: "20s",
		}

		key := "expiry-test"
		idempotencyKey := "request-789"
		idempotencyTTL := 2 * time.Second // Short TTL for testing

		// First request with idempotency should be allowed and consume the 1 available capacity
		r.SetTime(clock.Now())
		res1, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, res1.Limited)
		require.Equal(t, time.Duration(0), res1.RetryAfter)
		t.Logf("Initial request with idempotency (TTL=%v): limited=%v, retry=%v", idempotencyTTL, res1.Limited, res1.RetryAfter)

		// Request with same idempotency key should be allowed (idempotency active - bypass)
		r.SetTime(clock.Now())
		res2, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, res2.Limited)
		require.Equal(t, time.Duration(0), res2.RetryAfter)
		t.Logf("Duplicate within TTL: limited=%v, retry=%v", res2.Limited, res2.RetryAfter)

		require.True(t, r.Exists("{rl}:"+idempotencyKey), r.Dump())
		require.True(t, r.Exists("{rl}:"+key), r.Dump())

		require.Equal(t, 6*time.Second, r.TTL("{rl}:"+key))

		// Advance time to expire the idempotency key
		expiryWait := idempotencyTTL + 1*time.Second
		clock.Advance(expiryWait)
		r.FastForward(expiryWait)
		t.Logf("Advanced time by %v to expire idempotency key", expiryWait)

		require.False(t, r.Exists("{rl}:"+idempotencyKey))
		require.True(t, r.Exists("{rl}:"+key), r.Dump())

		// Request with same idempotency key should now be rate limited (capacity already exhausted by first request)
		r.SetTime(clock.Now())
		res3, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.True(t, res3.Limited) // Should be rate limited since capacity was consumed by first request
		require.Greater(t, res3.RetryAfter, time.Duration(0))
		t.Logf("Request after idempotency expired: limited=%v, retry=%v", res3.Limited, res3.RetryAfter)

		// Verify that any new request without idempotency is also rate limited (capacity exhausted)
		r.SetTime(clock.Now())
		res4, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, res4.Limited) // Should be rate limited (capacity exhausted)
		require.Greater(t, res4.RetryAfter, time.Duration(0))
		t.Logf("Verify capacity exhausted - new request: limited=%v, retry=%v", res4.Limited, res4.RetryAfter)
	})
}

func TestLuaRateLimit_ScientificNotationParsing(t *testing.T) {
	ctx := context.Background()

	t.Run("with graceful handling, no more syntax errors should be surfaced", func(t *testing.T) {
		r, rc, clock := initRedis(t)

		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")
		config := inngest.RateLimit{
			Limit:  50,
			Period: "1h",
		}

		key := "scientific-notation-direct-test"
		redisKey := prefix + key

		// Directly set a scientific notation value in Redis that mimics what we observed
		// This is the exact value format that caused the issue: "1.7628952937785e+18"
		scientificValue := "1.7628952937785e+18"

		t.Logf("Manually setting Redis key %s to scientific notation value: %s", redisKey, scientificValue)
		err := r.Set(redisKey, scientificValue)
		require.NoError(t, err)

		// Verify the value was set
		storedValue, err := r.Get(redisKey)
		require.NoError(t, err)
		t.Logf("Confirmed stored value: %s", storedValue)

		// Run a couple rate limit operations in sequence to ensure we keep using the valid value
		for range 5 {
			res, err := limiter.RateLimit(ctx, key, config)
			require.NoError(t, err)
			require.False(t, res.Limited)
			require.Equal(t, time.Duration(0), res.RetryAfter)

			clock.Advance(1 * time.Second)
			r.FastForward(1 * time.Second)
			r.SetTime(clock.Now())
		}
	})
}

func TestLuaRateLimit_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid period", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  5,
			Period: "invalid",
		}

		r.SetTime(clock.Now())
		res, err := limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.Error(t, err)
		require.Nil(t, res)
	})

	t.Run("zero limit", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  0,
			Period: "1h",
		}

		// The throttled library panics with divide by zero for limit=0
		// So we test that our Lua implementation gracefully handles zero limits
		// by falling back to 1
		r.SetTime(clock.Now())
		res, err := limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.NoError(t, err)
		t.Logf("Lua with zero limit: limited=%v, retry=%v", res.Limited, res.RetryAfter)

		// Should allow the first request
		require.False(t, res.Limited)
		require.Equal(t, res.RetryAfter, time.Duration(0))

		// Second call should fail
		res, err = limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, res.Limited)
	})

	t.Run("very short period", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  1,
			Period: "1ms",
		}

		// First request should be allowed
		r.SetTime(clock.Now())
		res, err := limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, res.Limited)

		// Advance time for period to pass
		clock.Advance(2 * time.Millisecond)
		r.FastForward(2 * time.Millisecond)

		// Next request should be allowed
		r.SetTime(clock.Now())
		res, err = limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, res.Limited)
	})
}

func TestLuaRateLimit_NilRemainingEdgeCase(t *testing.T) {
	ctx := context.Background()

	t.Run("heavily exceeded rate limit should not cause nil comparison error", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		// Use limit=10 to get burst=1, capacity=2
		// emission = 1s / 10 = 100ms
		// For next <= -emission: dvt + emission <= tat - now
		// dvt = emission * (burst + 1) = 100ms * 2 = 200ms
		// We need tat - now >= 300ms
		config := inngest.RateLimit{
			Limit:  10,
			Period: "1s",
		}

		key := "nil-remaining-test"
		redisKey := "{rl}:" + key

		// Manually set TAT to be far in the future to trigger the edge case
		// Set TAT to be 500ms in the future (500,000,000 nanoseconds)
		now := clock.Now()
		futureTAT := now.Add(500 * time.Millisecond)
		tatNs := fmt.Sprintf("%d", futureTAT.UnixNano())

		t.Logf("Setting Redis key %s to TAT: %s (500ms in future)", redisKey, tatNs)
		err := r.Set(redisKey, tatNs)
		require.NoError(t, err)

		// Now when we check capacity:
		// - tat = now + 500ms
		// - ttl = tat - now = 500ms
		// - dvt = 200ms
		// - next = dvt - ttl = 200ms - 500ms = -300ms
		// - Is -300ms > -100ms? NO! So remaining should NOT be set (nil bug)

		// This should trigger the nil comparison error at line 173
		r.SetTime(clock.Now())
		res, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))

		// Without the fix, this should error with "attempt to compare number with nil"
		// With the fix, it should succeed and show we're rate limited
		require.NoError(t, err, "should not error even when TAT is far in future")
		require.True(t, res.Limited, "should be rate limited")
		require.Greater(t, res.RetryAfter, time.Duration(0), "should have positive retry time")

		t.Logf("Result: limited=%v, retry=%v", res.Limited, res.RetryAfter)
	})
}
