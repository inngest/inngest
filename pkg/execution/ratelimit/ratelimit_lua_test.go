package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
		limited, retryAfter, err := limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, limited)
		require.Equal(t, time.Duration(0), retryAfter)

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
		limited, _, err := limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, limited)

		// Second request should be rate limited
		limited, retryAfter, err := limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, limited)
		require.Greater(t, retryAfter, time.Duration(0))
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
			limited, _, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
			require.NoError(t, err)
			require.False(t, limited, "request %d should be allowed (not limited)", i+1)
		}

		// Next request should be rate limited
		r.SetTime(clock.Now())
		limited, retryAfter, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, limited)
		require.Greater(t, retryAfter, time.Duration(0))
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
			limited, retry, err := luaLimiterFresh.RateLimit(ctx, keyFresh, config, WithNow(clock1.Now()))
			require.NoError(t, err)
			freshResults = append(freshResults, !limited) // Store as "allowed" for logic consistency
			t.Logf("Fresh state - request %d: limited=%v, retry=%v", i+1, limited, retry)
			if limited {
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
		preLimited, _, err := luaLimiterExisting.RateLimit(ctx, keyExisting, config, WithNow(clock2.Now()))
		require.NoError(t, err)
		require.False(t, preLimited)
		t.Logf("Pre-consumed 1 request to create existing state")

		// Now test how many MORE requests this existing state allows
		var existingResults []bool
		for i := 0; i < 5; i++ { // Try enough to hit limits
			r2.SetTime(clock2.Now())
			limited, retry, err := luaLimiterExisting.RateLimit(ctx, keyExisting, config, WithNow(clock2.Now()))
			require.NoError(t, err)
			existingResults = append(existingResults, !limited) // Store as "allowed" for logic consistency
			t.Logf("Existing state - request %d: limited=%v, retry=%v", i+1, limited, retry)
			if limited {
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
		limited1, retry1, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, limited1)
		require.Equal(t, time.Duration(0), retry1)

		// Second request should be rate limited with retryAfter
		startTime := clock.Now()
		r.SetTime(clock.Now())
		limited2, retry2, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, limited2)
		require.Greater(t, retry2, time.Duration(0))

		t.Logf("Rate limited with retryAfter: %v", retry2)

		// Verify retryAfter is reasonable (should be close to 2 seconds)
		require.Greater(t, retry2, 1500*time.Millisecond, "retryAfter should be at least 1.5s")
		require.Less(t, retry2, 2500*time.Millisecond, "retryAfter should be less than 2.5s")

		// Wait for the retryAfter duration (minus small buffer for test execution time)
		waitDuration := retry2 - 50*time.Millisecond
		if waitDuration > 0 {
			t.Logf("Advancing time by %v before next attempt", waitDuration)
			clock.Advance(waitDuration)
			r.FastForward(waitDuration)
		}

		// Request should still be rate limited (slightly early)
		r.SetTime(clock.Now())
		limited3, retry3, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, limited3)
		require.Greater(t, retry3, time.Duration(0))
		require.Less(t, retry3, 100*time.Millisecond, "Should have very short retryAfter")

		// Advance time a bit more and request should be allowed
		clock.Advance(100 * time.Millisecond)
		r.FastForward(100 * time.Millisecond)
		r.SetTime(clock.Now())
		limited4, retry4, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, limited4)
		require.Equal(t, time.Duration(0), retry4)

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
		limited1, _, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, limited1)

		// Get initial retryAfter
		r.SetTime(clock.Now())
		_, retry1, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.Greater(t, retry1, time.Duration(0))
		t.Logf("Initial retryAfter: %v", retry1)

		// Advance time by 500ms
		clock.Advance(500 * time.Millisecond)
		r.FastForward(500 * time.Millisecond)

		// Get retryAfter again - should be shorter
		r.SetTime(clock.Now())
		_, retry2, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.Greater(t, retry2, time.Duration(0))
		t.Logf("RetryAfter after 500ms advance: %v", retry2)

		// Verify retryAfter decreased
		decrease := retry1 - retry2
		t.Logf("RetryAfter decreased by: %v", decrease)
		require.Greater(t, decrease, 400*time.Millisecond, "Should decrease by approximately the wait time")
		require.Less(t, decrease, 600*time.Millisecond, "Should not decrease more than wait + tolerance")

		// Advance time by another 500ms
		clock.Advance(500 * time.Millisecond)
		r.FastForward(500 * time.Millisecond)

		// Get retryAfter again - should be even shorter
		r.SetTime(clock.Now())
		_, retry3, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.Greater(t, retry3, time.Duration(0))
		t.Logf("RetryAfter after total 1s advance: %v", retry3)

		// Verify continued decrease
		require.Less(t, retry3, retry2, "RetryAfter should continue to decrease")

		totalDecrease := retry1 - retry3
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
				limited, _, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
				require.NoError(t, err)
				require.False(t, limited)

				// Get retryAfter
				r.SetTime(clock.Now())
				rateLimited, retryAfter, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
				require.NoError(t, err)
				require.True(t, rateLimited)
				require.Greater(t, retryAfter, time.Duration(0))

				t.Logf("Period %s: retryAfter = %v", tc.period, retryAfter)

				// Verify retryAfter is within expected range
				require.Greater(t, retryAfter, tc.expectedMin,
					"retryAfter should be at least %v for period %s", tc.expectedMin, tc.period)
				require.Less(t, retryAfter, tc.expectedMax,
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
			limited, retry, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
			require.NoError(t, err)
			require.False(t, limited)
			require.Equal(t, time.Duration(0), retry)
			t.Logf("Burst request %d: allowed", i+1)
		}

		// Next request should be rate limited
		startTime := clock.Now()
		r.SetTime(clock.Now())
		rateLimited, retryAfter, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, rateLimited)
		require.Greater(t, retryAfter, time.Duration(0))

		t.Logf("After exhausting burst capacity, retryAfter: %v", retryAfter)

		// For GCRA with burst, retryAfter should be based on emission interval
		// emission_interval = period / limit = 10s / 10 = 1s per request
		// After consuming 2 requests instantly, should wait ~1s for next slot
		require.Greater(t, retryAfter, 500*time.Millisecond)
		require.Less(t, retryAfter, 2000*time.Millisecond)

		// Verify retryAfter leads to successful request
		if retryAfter < 5*time.Second { // Only wait if reasonable for test
			waitTime := retryAfter + 50*time.Millisecond // Add small buffer
			t.Logf("Advancing time by %v for capacity recovery", waitTime)
			clock.Advance(waitTime)
			r.FastForward(waitTime)

			r.SetTime(clock.Now())
			limited, _, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
			require.NoError(t, err)
			require.False(t, limited, "Request should be allowed (not limited) after waiting retryAfter duration")

			elapsed := clock.Since(startTime)
			t.Logf("Successfully allowed request after %v", elapsed)
		}
	})

	t.Run("retryAfter with zero capacity edge case", func(t *testing.T) {
		r, rc, clock := initRedis(t)
		defer rc.Close()

		limiter := New(ctx, rc, "{rl}:")

		config := inngest.RateLimit{
			Limit:  0, // Zero limit should always rate limit
			Period: "1h",
		}

		key := "zero-capacity-test"

		// First request should be rate limited
		r.SetTime(clock.Now())
		rateLimited, retryAfter, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, rateLimited)
		require.Greater(t, retryAfter, time.Duration(0))

		t.Logf("Zero limit retryAfter: %v", retryAfter)

		// With zero limit, retryAfter should be the full period
		require.Greater(t, retryAfter, 59*time.Minute)
		require.Less(t, retryAfter, 61*time.Minute)

		// Subsequent requests should also be rate limited
		r.SetTime(clock.Now())
		rateLimited2, retryAfter2, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, rateLimited2)
		require.Greater(t, retryAfter2, time.Duration(0))

		// RetryAfter should remain close to original (zero capacity = no progress)
		timeDiff := abs(retryAfter - retryAfter2)
		require.Less(t, timeDiff, 100*time.Millisecond, "RetryAfter should be consistent for zero limit")
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
		limited, _, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, limited)

		// Get multiple retryAfter values in quick succession
		var retryTimes []time.Duration

		for i := 0; i < 5; i++ {
			r.SetTime(clock.Now())
			_, retry, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
			require.NoError(t, err)
			require.Greater(t, retry, time.Duration(0))
			retryTimes = append(retryTimes, retry)

			t.Logf("Retry %d: %v", i+1, retry)
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
		limited1, retry1, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, limited1)
		require.Equal(t, time.Duration(0), retry1)
		t.Logf("Request 1 (no idempotency): limited=%v, retry=%v", limited1, retry1)

		// Second request should be rate limited (capacity exhausted)
		r.SetTime(clock.Now())
		limited2, retry2, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, limited2)
		require.Greater(t, retry2, time.Duration(0))
		t.Logf("Request 2 (no idempotency): limited=%v, retry=%v", limited2, retry2)
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
		limited1, retry1, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, limited1)
		require.Equal(t, time.Duration(0), retry1)
		t.Logf("First request with idempotency: limited=%v, retry=%v", limited1, retry1)

		// Subsequent request with same idempotency key should be allowed WITHOUT consuming capacity (idempotency bypass)
		r.SetTime(clock.Now())
		limited2, retry2, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, limited2)
		require.Equal(t, time.Duration(0), retry2)
		t.Logf("Duplicate request with same idempotency key: limited=%v, retry=%v", limited2, retry2)

		// Third request with same idempotency key should still be allowed (idempotency bypass)
		r.SetTime(clock.Now())
		limited3, retry3, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, limited3)
		require.Equal(t, time.Duration(0), retry3)
		t.Logf("Third request with same idempotency key: limited=%v, retry=%v", limited3, retry3)

		// New request without idempotency should be rate limited (capacity already exhausted by first request)
		r.SetTime(clock.Now())
		limited4, retry4, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, limited4)
		require.Greater(t, retry4, time.Duration(0))
		t.Logf("New request without idempotency: limited=%v, retry=%v", limited4, retry4)
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
		limited1, retry1, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, limited1)
		require.Equal(t, time.Duration(0), retry1)
		t.Logf("Setup request (consume capacity): limited=%v, retry=%v", limited1, retry1)

		// Request with idempotency should be rate limited
		r.SetTime(clock.Now())
		limited2, retry2, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.True(t, limited2)
		require.Greater(t, retry2, time.Duration(0))
		t.Logf("Rate limited request with idempotency: limited=%v, retry=%v", limited2, retry2)

		// Subsequent request with same idempotency key should STILL be rate limited
		// (idempotency key should NOT be set for rate limited requests)
		r.SetTime(clock.Now())
		limited3, retry3, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.True(t, limited3)
		require.Greater(t, retry3, time.Duration(0))
		t.Logf("Retry with same idempotency key after rate limit: limited=%v, retry=%v", limited3, retry3)

		// Advance time to allow capacity recovery
		waitTime := retry2 + 100*time.Millisecond
		clock.Advance(waitTime)
		r.FastForward(waitTime)
		t.Logf("Advanced time by %v for capacity recovery", waitTime)

		// Now the request with idempotency should succeed
		r.SetTime(clock.Now())
		limited4, retry4, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, limited4)
		require.Equal(t, time.Duration(0), retry4)
		t.Logf("Request after recovery with idempotency: limited=%v, retry=%v", limited4, retry4)

		// Subsequent request with same idempotency should now be allowed (idempotency enforced)
		r.SetTime(clock.Now())
		limited5, retry5, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, limited5)
		require.Equal(t, time.Duration(0), retry5)
		t.Logf("Duplicate after successful request: limited=%v, retry=%v", limited5, retry5)
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
		idempotencyTTL := 5 * time.Second // Short TTL for testing

		// First request with idempotency should be allowed and consume the 1 available capacity
		r.SetTime(clock.Now())
		limited1, retry1, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, limited1)
		require.Equal(t, time.Duration(0), retry1)
		t.Logf("Initial request with idempotency (TTL=%v): limited=%v, retry=%v", idempotencyTTL, limited1, retry1)

		// Request with same idempotency key should be allowed (idempotency active - bypass)
		r.SetTime(clock.Now())
		limited2, retry2, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.False(t, limited2)
		require.Equal(t, time.Duration(0), retry2)
		t.Logf("Duplicate within TTL: limited=%v, retry=%v", limited2, retry2)

		// Advance time to expire the idempotency key
		expiryWait := idempotencyTTL + 1*time.Second
		clock.Advance(expiryWait)
		r.FastForward(expiryWait)
		t.Logf("Advanced time by %v to expire idempotency key", expiryWait)

		// Request with same idempotency key should now be rate limited (capacity already exhausted by first request)
		r.SetTime(clock.Now())
		limited3, retry3, err := limiter.RateLimit(ctx, key, config,
			WithNow(clock.Now()),
			WithIdempotency(idempotencyKey, idempotencyTTL))
		require.NoError(t, err)
		require.True(t, limited3) // Should be rate limited since capacity was consumed by first request
		require.Greater(t, retry3, time.Duration(0))
		t.Logf("Request after idempotency expired: limited=%v, retry=%v", limited3, retry3)

		// Verify that any new request without idempotency is also rate limited (capacity exhausted)
		r.SetTime(clock.Now())
		limited4, retry4, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.True(t, limited4) // Should be rate limited (capacity exhausted)
		require.Greater(t, retry4, time.Duration(0))
		t.Logf("Verify capacity exhausted - new request: limited=%v, retry=%v", limited4, retry4)
	})
}

func TestLuaRateLimit_ScientificNotationParsing(t *testing.T) {
	ctx := context.Background()

	t.Run("large nanosecond timestamps causing scientific notation", func(t *testing.T) {
		t.Skip("this should produce scientific notation but does not -- the root cause is likely a more complex combination")

		r, rc, throttledStore, clock := initRedis(t)
		defer rc.Close()

		config := inngest.RateLimit{
			Limit:  100,
			Period: "1h",
		}

		key := "scientific-notation-test"

		// Phase 1: Create throttled state (first request)
		t.Logf("Phase 1: Creating initial throttled state at time %v (ns: %d)", clock.Now(), clock.Now().UnixNano())
		limited1, retry1, err := rateLimit(ctx, throttledStore, key, config)
		require.NoError(t, err)
		require.False(t, limited1) // Should be allowed (not limited)
		t.Logf("First request: limited=%v, retry=%v", limited1, retry1)

		currentVal, err := r.Get(prefix + key)
		require.NoError(t, err)
		t.Logf("First key: %v", currentVal)

		// Phase 2: Advance clock by 1 second
		t.Logf("Phase 2: Advancing clock by 1 second")
		clock.Advance(1 * time.Second)
		r.FastForward(1 * time.Second)
		r.SetTime(clock.Now())
		t.Logf("Clock advanced to %v (ns: %d)", clock.Now(), clock.Now().UnixNano())

		// Phase 3: Make request using Lua implementation (this should work)
		t.Logf("Phase 3: Making request with Lua implementation")
		luaLimiter := newLuaGCRARateLimiter(ctx, rc, prefix)
		limited2, retry2, err := luaLimiter.RateLimit(ctx, key, config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, limited2) // Should not be rate limited
		t.Logf("Lua request: limited=%v, retry=%v", limited2, retry2)

		currentVal, err = r.Get(prefix + key)
		require.NoError(t, err)
		t.Logf("Second key: %v", currentVal)

		// Phase 4: Advance clock by another second
		t.Logf("Phase 4: Advancing clock by another second")
		clock.Advance(1 * time.Second)
		r.FastForward(1 * time.Second)
		r.SetTime(clock.Now())
		t.Logf("Clock advanced to %v (ns: %d)", clock.Now(), clock.Now().UnixNano())

		// Phase 5: Continue with throttled state - this should trigger the scientific notation issue
		// The Lua script has stored a very large nanosecond timestamp that gets serialized in scientific notation
		t.Logf("Phase 5: Attempting throttled implementation (this may fail with scientific notation parsing)")

		// This is where the bug should manifest - AsInt64() trying to parse scientific notation
		_, _, err = rateLimit(ctx, throttledStore, key, config)
		require.Error(t, err)
		t.Logf("ERROR (expected): %v", err)
		// Check if it's the specific scientific notation parsing error
		if strings.Contains(err.Error(), "strconv.ParseInt") && strings.Contains(err.Error(), "invalid syntax") {
			t.Logf("SUCCESS: Reproduced the scientific notation parsing issue!")
			t.Logf("Error details: %v", err)
		} else {
			t.Fatalf("Unexpected error (not the scientific notation issue): %v", err)
		}

		// Additional verification: try to directly observe the Redis value that might be in scientific notation
		redisKey := prefix + key
		cmd := rc.B().Get().Key(redisKey).Build()
		result, err := rc.Do(ctx, cmd).ToString()
		if err == nil {
			t.Logf("Raw Redis value: %s", result)
			// Check if it's in scientific notation
			if strings.Contains(result, "e+") || strings.Contains(result, "E+") {
				t.Logf("CONFIRMED: Redis value is in scientific notation format!")
			}
		}
	})

	t.Run("direct scientific notation parsing failure", func(t *testing.T) {
		r, rc, throttledStore, _ := initRedis(t)

		defer rc.Close()

		// NOTE: Explicitly disable graceful parsing here so we get to see the error
		throttledStore.disableGracefulScientificNotationParsing = true

		config := inngest.RateLimit{
			Limit:  1,
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

		// Now try to use the throttled implementation which should fail when trying to parse this
		t.Logf("Attempting to use throttled implementation with scientific notation value in Redis...")

		// This should fail with the AsInt64() parsing error
		limited, retry, err := rateLimit(ctx, throttledStore, key, config)

		// We expect this to fail with a parsing error
		require.Error(t, err)
		t.Logf("Got expected error: %v", err)

		// Verify it's the specific scientific notation parsing error
		require.True(t, strings.Contains(err.Error(), "strconv.ParseInt") ||
			strings.Contains(err.Error(), "invalid syntax") ||
			strings.Contains(err.Error(), "failed to get key value"),
			"Expected parsing error, got: %v", err)

		t.Logf("SUCCESS: Reproduced scientific notation parsing failure!")
		t.Logf("Error details: %v", err)
		t.Logf("Limited: %v, Retry: %v", limited, retry)

		// Also test the direct Redis parsing that would happen in GetWithTime
		cmd := rc.B().Get().Key(redisKey).Build()
		result := rc.Do(ctx, cmd)

		// Try to parse as int64 - this should fail
		_, parseErr := result.AsInt64()
		require.Error(t, parseErr)
		t.Logf("Direct AsInt64() parsing also failed as expected: %v", parseErr)

		// But ToString should work
		strResult, err := result.ToString()
		require.NoError(t, err)
		t.Logf("ToString() works fine: %s", strResult)
	})

	t.Run("with graceful handling, no more syntax errors should be surfaced", func(t *testing.T) {
		r, rc, throttledStore, clock := initRedis(t)

		defer rc.Close()

		// With graceful parsing, we should be handled gracefully
		throttledStore.disableGracefulScientificNotationParsing = false

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
			limited, retry, err := rateLimit(ctx, throttledStore, key, config)
			require.NoError(t, err)
			require.False(t, limited)
			require.Equal(t, time.Duration(-1), retry)

			clock.Advance(1 * time.Second)
			r.FastForward(1 * time.Second)
			r.SetTime(clock.Now())
		}
	})

	t.Run("force lua to write scientific notation with artificially large number", func(t *testing.T) {
		r, rc, throttledStore, _ := initRedis(t)
		defer rc.Close()

		config := inngest.RateLimit{
			Limit:  10,
			Period: "1h",
		}

		key := "scientific-notation-direct-test"
		redisKey := prefix + key

		// Try to force Redis to store in scientific notation by using a very large number with decimals
		cmd := rc.B().Eval().Script(`local key = KEYS[1]
			-- Create a number that's too large for Redis to store as a normal integer
			-- Math operations that create very large floating-point results
			local base = 9223372036854775807  -- Max int64
			local multiplier = 1.5
			local very_large = base * multiplier  -- This should force floating-point representation
			redis.call("SET", key, very_large)
			return 0`).Numkeys(1).Key(redisKey).Build()
		err := rc.Do(ctx, cmd).Error()
		require.NoError(t, err)

		// Verify the value was set
		storedValue, err := r.Get(redisKey)
		require.NoError(t, err)
		t.Logf("Confirmed stored value: %s", storedValue)

		// Also test the direct Redis parsing that would happen in GetWithTime
		cmd = rc.B().Get().Key(redisKey).Build()
		result := rc.Do(ctx, cmd)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				r, rc, throttledStore, clock := initRedis(t)
				defer rc.Close()

		// But ToString should work
		strResult, err := result.ToString()
		require.NoError(t, err)
		t.Logf("ToString() works fine: %s", strResult)

		// Rate limit should gracefully handle value
		_, _, err = rateLimit(ctx, throttledStore, key, config)
		require.NoError(t, err)

		// Expect value to be normalized
		storedValue, err = r.Get(redisKey)
		require.NoError(t, err)
		t.Logf("Confirmed stored value: %s", storedValue)

		_, err = strconv.Atoi(storedValue)
		require.NoError(t, err)
	})

	t.Run("force lua to write scientific notation with artificially large number with new impl", func(t *testing.T) {
		r, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, prefix)

		config := inngest.RateLimit{
			Limit:  10,
			Period: "1h",
		}

		key := "scientific-notation-direct-test"
		redisKey := prefix + key

		// Try to force Redis to store in scientific notation by using a very large number with decimals
		cmd := rc.B().Eval().Script(`local key = KEYS[1]
			-- Create a number that's too large for Redis to store as a normal integer
			-- Math operations that create very large floating-point results
			local base = 9223372036854775807  -- Max int64
			local multiplier = 1.5
			local very_large = base * multiplier  -- This should force floating-point representation
			redis.call("SET", key, very_large)
			return 0`).Numkeys(1).Key(redisKey).Build()
		err := rc.Do(ctx, cmd).Error()
		require.NoError(t, err)

		// Verify the value was set
		storedValue, err := r.Get(redisKey)
		require.NoError(t, err)
		t.Logf("Confirmed stored value: %s", storedValue)

		// Also test the direct Redis parsing that would happen in GetWithTime
		cmd = rc.B().Get().Key(redisKey).Build()
		result := rc.Do(ctx, cmd)

		// Try to parse as int64 - this should fail
		_, parseErr := result.AsInt64()
		require.Error(t, parseErr)
		t.Logf("Direct AsInt64() parsing also failed as expected: %v", parseErr)

		// But ToString should work
		strResult, err := result.ToString()
		require.NoError(t, err)
		t.Logf("ToString() works fine: %s", strResult)

		// Should fail because it's clamped to the maximum
		limited, retry, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))

		// We expect this to fail with a parsing error
		require.NoError(t, err)

		// Expect value to be clamped
		storedValue, err = r.Get(redisKey)
		require.NoError(t, err)
		t.Logf("Confirmed stored value: %s", storedValue)

		normalizedValue, err := strconv.Atoi(storedValue)
		require.NoError(t, err)

		emissionInterval := time.Hour.Nanoseconds() / 10
		burst := 1
		totalCapacity := (burst + 1)
		delayVariationTolerance := emissionInterval * int64(totalCapacity)
		expectedMax := clock.Now().UnixNano() + time.Hour.Nanoseconds() + delayVariationTolerance

		require.InDelta(t, int(expectedMax), normalizedValue, 10)

		require.True(t, limited)
		require.Equal(t, time.Hour+6*time.Minute, retry.Round(time.Minute))

		clock.Advance(retry + time.Minute)
		r.FastForward(retry + time.Minute)
		r.SetTime(clock.Now())

		// Should allow another request
		limited, retry, err = limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))

		require.NoError(t, err)
		require.False(t, limited)
		require.Equal(t, time.Duration(0), retry)
	})

	t.Run("force lua to write scientific notation with artificially large number with new impl", func(t *testing.T) {
		r, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, prefix)

		config := inngest.RateLimit{
			Limit:  10,
			Period: "1h",
		}

		key := "scientific-notation-direct-test"
		redisKey := prefix + key

		// Try to force Redis to store in scientific notation by using a very large number with decimals
		cmd := rc.B().Eval().Script(`local key = KEYS[1]
			-- Create a number that's too large for Redis to store as a normal integer
			-- Math operations that create very large floating-point results
			local base = 9223372036854775807  -- Max int64
			local multiplier = 1.5
			local very_large = base * multiplier  -- This should force floating-point representation
			redis.call("SET", key, very_large)
			return 0`).Numkeys(1).Key(redisKey).Build()
		err := rc.Do(ctx, cmd).Error()
		require.NoError(t, err)

		// Verify the value was set
		storedValue, err := r.Get(redisKey)
		require.NoError(t, err)
		t.Logf("Confirmed stored value: %s", storedValue)

		// Also test the direct Redis parsing that would happen in GetWithTime
		cmd = rc.B().Get().Key(redisKey).Build()
		result := rc.Do(ctx, cmd)

		// Try to parse as int64 - this should fail
		_, parseErr := result.AsInt64()
		require.Error(t, parseErr)
		t.Logf("Direct AsInt64() parsing also failed as expected: %v", parseErr)

		// But ToString should work
		strResult, err := result.ToString()
		require.NoError(t, err)
		t.Logf("ToString() works fine: %s", strResult)

		// Should fail because it's clamped to the maximum
		limited, retry, err := limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))

		// We expect this to fail with a parsing error
		require.NoError(t, err)

		// Expect value to be clamped
		storedValue, err = r.Get(redisKey)
		require.NoError(t, err)
		t.Logf("Confirmed stored value: %s", storedValue)

		normalizedValue, err := strconv.Atoi(storedValue)
		require.NoError(t, err)

		emissionInterval := time.Hour.Nanoseconds() / 10
		burst := 1
		totalCapacity := (burst + 1)
		delayVariationTolerance := emissionInterval * int64(totalCapacity)
		expectedMax := clock.Now().UnixNano() + time.Hour.Nanoseconds() + delayVariationTolerance

		require.InDelta(t, int(expectedMax), normalizedValue, 10)

		require.True(t, limited)
		require.Equal(t, time.Hour+6*time.Minute, retry.Round(time.Minute))

		clock.Advance(retry + time.Minute)
		r.FastForward(retry + time.Minute)
		r.SetTime(clock.Now())

		// Should allow another request
		limited, retry, err = limiter.RateLimit(ctx, key, config, WithNow(clock.Now()))

		require.NoError(t, err)
		require.False(t, limited)
		require.Equal(t, time.Duration(0), retry)
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
		limited, retryAfter, err := limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.Error(t, err)
		require.True(t, limited) // Should return true (limited) on error
		require.Equal(t, time.Duration(-1), retryAfter)
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
		// by immediately rate limiting (which is the logical behavior)
		r.SetTime(clock.Now())
		limited, retryAfter, err := limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.NoError(t, err)
		t.Logf("Lua with zero limit: limited=%v, retry=%v", limited, retryAfter)

		// Zero limit should immediately rate limit
		require.True(t, limited)
		require.Greater(t, retryAfter, time.Duration(0))
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
		limited, _, err := limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, limited)

		// Advance time for period to pass
		clock.Advance(2 * time.Millisecond)
		r.FastForward(2 * time.Millisecond)

		// Next request should be allowed
		r.SetTime(clock.Now())
		limited, _, err = limiter.RateLimit(ctx, "test-key", config, WithNow(clock.Now()))
		require.NoError(t, err)
		require.False(t, limited)
	})
}

// Helper functions

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
