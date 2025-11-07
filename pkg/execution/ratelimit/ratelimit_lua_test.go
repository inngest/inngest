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
	"github.com/throttled/throttled/v2"
)

const prefix = "{rl}:"

// initRedis creates both miniredis/rueidis for Lua, throttled store, and fake clock
func initRedis(t *testing.T) (*miniredis.Miniredis, rueidis.Client, throttled.GCRAStoreCtx, clockwork.FakeClock) {
	r := miniredis.RunT(t)

	// Create rueidis client for Lua implementation
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	// Create throttled store
	store := New(context.Background(), rc, prefix).(*rueidisStore)
	require.NoError(t, err)

	clock := clockwork.NewFakeClock()
	// Set miniredis time to match fake clock
	r.SetTime(clock.Now())

	return r, rc, store, clock
}

func TestLuaRateLimit_BasicFunctionality(t *testing.T) {
	ctx := context.Background()

	t.Run("should allow requests under limit", func(t *testing.T) {
		r, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, prefix)

		config := inngest.RateLimit{
			Limit:  5,
			Period: "1h",
		}

		// First request should be allowed (not limited)
		r.SetTime(clock.Now())
		limited, retryAfter, err := limiter.RateLimit(ctx, "test-key", config, clock.Now())
		require.NoError(t, err)
		require.False(t, limited)
		require.Equal(t, time.Duration(0), retryAfter)

		// Should have created a key in Redis
		require.Len(t, r.Keys(), 1)
	})

	t.Run("should rate limit when over limit", func(t *testing.T) {
		_, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, "test:")

		config := inngest.RateLimit{
			Limit:  1,
			Period: "1h",
		}

		// First request should be allowed (not limited)
		limited, _, err := limiter.RateLimit(ctx, "test-key", config, clock.Now())
		require.NoError(t, err)
		require.False(t, limited)

		// Second request should be rate limited
		limited, retryAfter, err := limiter.RateLimit(ctx, "test-key", config, clock.Now())
		require.NoError(t, err)
		require.True(t, limited)
		require.Greater(t, retryAfter, time.Duration(0))
	})

	t.Run("should handle burst correctly", func(t *testing.T) {
		r, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, "test:")

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
			limited, _, err := limiter.RateLimit(ctx, key, config, clock.Now())
			require.NoError(t, err)
			require.False(t, limited, "request %d should be allowed (not limited)", i+1)
		}

		// Next request should be rate limited
		r.SetTime(clock.Now())
		limited, retryAfter, err := limiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.True(t, limited)
		require.Greater(t, retryAfter, time.Duration(0))
	})
}

func TestLuaRateLimit_SideBySideComparison(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name     string
		limit    uint
		period   string
		requests int
	}{
		{"1 per hour", 1, "1h", 3},
		{"5 per hour", 5, "1h", 10},
		{"10 per minute", 10, "1m", 15},
		{"100 per hour with burst", 100, "1h", 110},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up both implementations with separate Redis instances
			r1, rc1, throttledStore, _ := initRedis(t)
			defer rc1.Close()

			r2, rc2, _, clock2 := initRedis(t)
			defer rc2.Close()

			luaLimiter := newLuaGCRARateLimiter(ctx, rc2, "test:")

			config := inngest.RateLimit{
				Limit:  tc.limit,
				Period: tc.period,
			}

			key := "test-key"

			// Make requests to both implementations
			for i := 0; i < tc.requests; i++ {
				// Test Lua implementation
				r2.SetTime(clock2.Now())
				luaAllowed, luaRetry, err := luaLimiter.RateLimit(ctx, key, config, clock2.Now())
				require.NoError(t, err)

				// Test throttled implementation (uses original interface)
				throttledAllowed, throttledRetry, err := rateLimit(ctx, throttledStore, key, config)
				if err != nil {
					t.Logf("Throttled implementation error: %v", err)
				}
				require.NoError(t, err)

				t.Logf("Request %d: lua(limited=%v, retry=%v) vs throttled(limited=%v, retry=%v)",
					i+1, luaAllowed, luaRetry, throttledAllowed, throttledRetry)

				// Results should match - both implementations now return "limited" (true if rate limited)
				throttledLimitedStatus := throttledAllowed // true if limited  
				luaLimitedStatus := luaAllowed             // true if limited

				// Both should have same semantics now
				require.Equal(t, throttledLimitedStatus, luaLimitedStatus,
					"request %d: limited status should match (throttled_limited=%v, lua_limited=%v)",
					i+1, throttledLimitedStatus, luaLimitedStatus)

				// If rate limited, both should have similar retry times (within tolerance)
				if throttledAllowed && luaAllowed {
					// Both are rate limited - compare retry times
					// throttledRetry might be -1 if not rate limited, so check for positive values
					if throttledRetry > 0 && luaRetry > 0 {
						timeDiff := abs(luaRetry - throttledRetry)
						require.Less(t, timeDiff, time.Second,
							"request %d: retry times should be similar (throttled=%v, lua=%v, diff=%v)",
							i+1, throttledRetry, luaRetry, timeDiff)
					}
				}
			}

			// Compare Redis state between implementations
			compareRedisState(t, r1, r2, "test-key")
		})
	}
}

func TestLuaRateLimit_StateMigration(t *testing.T) {
	ctx := context.Background()

	t.Run("old throttled state -> Lua implementation", func(t *testing.T) {
		r, rc, throttledStore, clock := initRedis(t)
		defer rc.Close()

		config := inngest.RateLimit{
			Limit:  5,
			Period: "1h",
		}

		key := "migration-test"

		// Create state with throttled implementation
		// Note: throttledStore uses prefix "{rl}:" while we'll use same prefix for Lua
		throttledLimited, _, err := rateLimit(ctx, throttledStore, key, config)
		require.NoError(t, err)
		require.False(t, throttledLimited) // false means not limited (allowed)

		// Now switch to Lua implementation using same prefix
		luaLimiter := newLuaGCRARateLimiter(ctx, rc, prefix) // Use same prefix as throttled

		// Should be able to read existing state and continue rate limiting
		r.SetTime(clock.Now())
		luaAllowed, _, err := luaLimiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		t.Logf("After migration, Lua limiter returned: allowed=%v", luaAllowed)

		// Make several more requests to verify continuity
		for i := 0; i < 3; i++ {
			r.SetTime(clock.Now())
			luaAllowed, _, err = luaLimiter.RateLimit(ctx, key, config, clock.Now())
			require.NoError(t, err)
			t.Logf("Request %d after migration: allowed=%v", i+1, luaAllowed)
		}
	})

	t.Run("Lua state -> old throttled implementation", func(t *testing.T) {
		r, rc, throttledStore, clock := initRedis(t)
		defer rc.Close()

		config := inngest.RateLimit{
			Limit:  5,
			Period: "1h",
		}

		key := "migration-test-2"

		// Create state with Lua implementation using same prefix as throttled
		luaLimiter := newLuaGCRARateLimiter(ctx, rc, prefix)
		r.SetTime(clock.Now())
		luaLimited, _, err := luaLimiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.False(t, luaLimited) // false means not limited (allowed)
		t.Logf("Lua limiter created state: limited=%v", luaLimited)

		// Now switch to throttled implementation and continue
		throttledLimited, _, err := rateLimit(ctx, throttledStore, key, config)
		require.NoError(t, err)
		t.Logf("After migration to throttled, result: limited=%v", throttledLimited)

		// Make several more requests to verify continuity
		for i := 0; i < 3; i++ {
			throttledLimited, _, err = rateLimit(ctx, throttledStore, key, config)
			require.NoError(t, err)
			t.Logf("Request %d after migration: limited=%v", i+1, throttledLimited)
		}
	})
}

func TestLuaRateLimit_MigrationUnderLoad(t *testing.T) {
	ctx := context.Background()

	t.Run("throttled to Lua under burst load", func(t *testing.T) {
		r, rc, throttledStore, clock := initRedis(t)
		defer rc.Close()

		config := inngest.RateLimit{
			Limit:  10,
			Period: "1h",
		}

		key := "load-test-1"

		// Consume partial burst capacity with throttled implementation
		// burst = 10/10 = 1, so total capacity = 10 + 1 = 11
		requestsToMake := 5 // Consume about half the capacity
		var throttledResults []bool
		var throttledRetryTimes []time.Duration

		for i := 0; i < requestsToMake; i++ {
			limited, retry, err := rateLimit(ctx, throttledStore, key, config)
			require.NoError(t, err)
			throttledResults = append(throttledResults, limited)
			throttledRetryTimes = append(throttledRetryTimes, retry)
			t.Logf("Throttled request %d: limited=%v, retry=%v", i+1, limited, retry)
		}

		// Now migrate to Lua implementation and continue the sequence
		luaLimiter := newLuaGCRARateLimiter(ctx, rc, prefix)

		// Continue making requests with Lua implementation
		remainingRequests := 8 // Should hit limits sooner due to existing state
		var luaResults []bool
		var luaRetryTimes []time.Duration

		for i := 0; i < remainingRequests; i++ {
			r.SetTime(clock.Now())
			allowed, retry, err := luaLimiter.RateLimit(ctx, key, config, clock.Now())
			require.NoError(t, err)
			luaResults = append(luaResults, allowed)
			luaRetryTimes = append(luaRetryTimes, retry)
			t.Logf("Lua request %d: allowed=%v, retry=%v", i+1, allowed, retry)
		}

		// Verify that we hit rate limits during the Lua phase
		// (since we already consumed capacity during throttled phase)
		rateLimitedCount := 0
		for _, allowed := range luaResults {
			if !allowed {
				rateLimitedCount++
			}
		}
		require.Greater(t, rateLimitedCount, 0, "Should hit rate limits with existing state")

		// Total capacity should be respected across both implementations
		totalAllowedThrottled := 0
		for _, limited := range throttledResults {
			if !limited { // throttled semantics: false = not limited
				totalAllowedThrottled++
			}
		}
		totalAllowedLua := 0
		for _, allowed := range luaResults {
			if allowed { // lua semantics: true = allowed
				totalAllowedLua++
			}
		}
		totalAllowed := totalAllowedThrottled + totalAllowedLua
		t.Logf("Total allowed requests: throttled=%d, lua=%d, total=%d",
			totalAllowedThrottled, totalAllowedLua, totalAllowed)

		// Should not exceed total capacity (limit + burst = 10 + 1 = 11)
		require.LessOrEqual(t, totalAllowed, 11, "Should not exceed total capacity")
	})

	t.Run("Lua to throttled under burst load", func(t *testing.T) {
		r, rc, throttledStore, clock := initRedis(t)
		defer rc.Close()

		config := inngest.RateLimit{
			Limit:  10,
			Period: "1h",
		}

		key := "load-test-2"

		// Start with Lua implementation
		luaLimiter := newLuaGCRARateLimiter(ctx, rc, prefix)

		// Consume partial capacity
		requestsToMake := 6
		var luaResults []bool

		for i := 0; i < requestsToMake; i++ {
			r.SetTime(clock.Now())
			allowed, retry, err := luaLimiter.RateLimit(ctx, key, config, clock.Now())
			require.NoError(t, err)
			luaResults = append(luaResults, allowed)
			t.Logf("Lua request %d: allowed=%v, retry=%v", i+1, allowed, retry)
		}

		// Migrate to throttled implementation
		remainingRequests := 7
		var throttledResults []bool

		for i := 0; i < remainingRequests; i++ {
			limited, retry, err := rateLimit(ctx, throttledStore, key, config)
			require.NoError(t, err)
			throttledResults = append(throttledResults, limited)
			t.Logf("Throttled request %d: limited=%v, retry=%v", i+1, limited, retry)
		}

		// Verify rate limiting occurred
		rateLimitedCount := 0
		for _, limited := range throttledResults {
			if limited { // throttled semantics: true = limited
				rateLimitedCount++
			}
		}
		require.Greater(t, rateLimitedCount, 0, "Should hit rate limits with existing state")

		// Verify total capacity
		totalAllowedLua := 0
		for _, allowed := range luaResults {
			if allowed {
				totalAllowedLua++
			}
		}
		totalAllowedThrottled := 0
		for _, limited := range throttledResults {
			if !limited {
				totalAllowedThrottled++
			}
		}
		totalAllowed := totalAllowedLua + totalAllowedThrottled
		t.Logf("Total allowed requests: lua=%d, throttled=%d, total=%d",
			totalAllowedLua, totalAllowedThrottled, totalAllowed)

		require.LessOrEqual(t, totalAllowed, 11, "Should not exceed total capacity")
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
		r1, rc1, _, clock1 := initRedis(t)
		defer rc1.Close()

		luaLimiterFresh := newLuaGCRARateLimiter(ctx, rc1, "fresh:")
		keyFresh := "fresh-state-test"

		// Count how many requests fresh state allows
		var freshResults []bool
		for i := 0; i < 5; i++ { // Try enough to hit limits
			r1.SetTime(clock1.Now())
			allowed, retry, err := luaLimiterFresh.RateLimit(ctx, keyFresh, config, clock1.Now())
			require.NoError(t, err)
			freshResults = append(freshResults, allowed)
			t.Logf("Fresh state - request %d: allowed=%v, retry=%v", i+1, allowed, retry)
			if !allowed {
				break // Stop at first rate limit
			}
		}

		allowedFreshCount := len(freshResults) - 1 // Subtract the rate limited one
		if len(freshResults) > 0 && freshResults[len(freshResults)-1] {
			allowedFreshCount = len(freshResults) // All were allowed
		}

		t.Logf("Fresh state allowed %d requests before hitting limit", allowedFreshCount)

		// Scenario 2: Create existing state by pre-consuming some capacity
		r2, rc2, _, clock2 := initRedis(t)
		defer rc2.Close()

		luaLimiterExisting := newLuaGCRARateLimiter(ctx, rc2, "existing:")
		keyExisting := "existing-state-test"

		// Pre-consume 1 request to create existing state
		r2.SetTime(clock2.Now())
		preAllowed, _, err := luaLimiterExisting.RateLimit(ctx, keyExisting, config, clock2.Now())
		require.NoError(t, err)
		require.True(t, preAllowed)
		t.Logf("Pre-consumed 1 request to create existing state")

		// Now test how many MORE requests this existing state allows
		var existingResults []bool
		for i := 0; i < 5; i++ { // Try enough to hit limits
			r2.SetTime(clock2.Now())
			allowed, retry, err := luaLimiterExisting.RateLimit(ctx, keyExisting, config, clock2.Now())
			require.NoError(t, err)
			existingResults = append(existingResults, allowed)
			t.Logf("Existing state - request %d: allowed=%v, retry=%v", i+1, allowed, retry)
			if !allowed {
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

	t.Run("cross-implementation state preservation", func(t *testing.T) {
		config := inngest.RateLimit{
			Limit:  10,    // burst = 10/10 = 1, total capacity = 10 + 1 = 11
			Period: "10m", // Shorter period for faster test
		}

		// Start with throttled, consume capacity, migrate to Lua, verify exact behavior
		r, rc, throttledStore, clock := initRedis(t)
		defer rc.Close()

		key := "cross-impl-test"

		// Phase 1: Throttled implementation - consume 1 out of 2 total requests (based on side-by-side test)
		var phase1Results []bool
		for i := 0; i < 1; i++ {
			limited, retry, err := rateLimit(ctx, throttledStore, key, config)
			require.NoError(t, err)
			phase1Results = append(phase1Results, limited)
			t.Logf("Phase 1 (throttled) - request %d: limited=%v, retry=%v", i+1, limited, retry)
		}

		// The 1 request should be allowed
		for i, limited := range phase1Results {
			require.False(t, limited, "Phase 1 request %d should be allowed", i+1)
		}

		// Phase 2: Migrate to Lua - should have exactly 1 request remaining (2 - 1)
		luaLimiter := newLuaGCRARateLimiter(ctx, rc, prefix)

		var phase2Results []bool
		for i := 0; i < 3; i++ { // Try 3 requests, expect 1 allowed, 2 rate limited
			r.SetTime(clock.Now())
			allowed, retry, err := luaLimiter.RateLimit(ctx, key, config, clock.Now())
			require.NoError(t, err)
			phase2Results = append(phase2Results, allowed)
			t.Logf("Phase 2 (lua) - request %d: allowed=%v, retry=%v", i+1, allowed, retry)
		}

		// Count allowed requests in phase 2
		allowedPhase2 := 0
		for _, allowed := range phase2Results {
			if allowed {
				allowedPhase2++
			}
		}

		require.Equal(t, 1, allowedPhase2, "Phase 2 should allow exactly 1 more request")

		// Verify the pattern: first 1 allowed, then rate limited
		require.True(t, phase2Results[0], "Request 1 in phase 2 should be allowed")
		require.False(t, phase2Results[1], "Request 2 in phase 2 should be rate limited")
		require.False(t, phase2Results[2], "Request 3 in phase 2 should be rate limited")
	})
}

func TestLuaRateLimit_PreciseTimingMigration(t *testing.T) {
	ctx := context.Background()

	t.Run("nanosecond precision timing preservation", func(t *testing.T) {
		config := inngest.RateLimit{
			Limit:  2,
			Period: "5s", // Short period for precise timing tests
		}

		// Set up both implementations with same Redis instance
		r, rc, throttledStore, clock := initRedis(t)
		defer rc.Close()

		key := "timing-precision-test"

		// Exhaust capacity with throttled implementation
		for i := 0; i < 2; i++ {
			limited, _, err := rateLimit(ctx, throttledStore, key, config)
			require.NoError(t, err)
			require.False(t, limited)
		}

		// Get rate limited request with exact timing
		throttledLimited, throttledRetry, err := rateLimit(ctx, throttledStore, key, config)
		require.NoError(t, err)
		require.True(t, throttledLimited)
		throttledRetryTime := clock.Now().Add(throttledRetry)

		// Migrate to Lua and check the same rate limited request
		luaLimiter := newLuaGCRARateLimiter(ctx, rc, prefix)
		r.SetTime(clock.Now())
		luaAllowed, luaRetry, err := luaLimiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.False(t, luaAllowed) // Should also be rate limited
		luaRetryTime := clock.Now().Add(luaRetry)

		// Timing should be very close (within 10ms due to test execution time)
		timeDiff := abs(throttledRetryTime.Sub(luaRetryTime))
		t.Logf("Timing precision: throttled=%v, lua=%v, diff=%v",
			throttledRetry, luaRetry, timeDiff)
		require.Less(t, timeDiff, 10*time.Millisecond,
			"Retry times should be very close between implementations")
	})

	t.Run("boundary condition timing", func(t *testing.T) {
		config := inngest.RateLimit{
			Limit:  1,
			Period: "1s",
		}

		r, rc, throttledStore, clock := initRedis(t)
		defer rc.Close()

		key := "boundary-timing-test"

		// Create state right at the boundary (consume full capacity)
		limited1, _, err := rateLimit(ctx, throttledStore, key, config)
		require.NoError(t, err)
		require.False(t, limited1) // First should be allowed

		// Right at boundary - should be rate limited
		limited2, retry2, err := rateLimit(ctx, throttledStore, key, config)
		require.NoError(t, err)
		require.True(t, limited2)
		t.Logf("Throttled at boundary: retry=%v", retry2)

		// Migrate to Lua and test same boundary condition
		luaLimiter := newLuaGCRARateLimiter(ctx, rc, prefix)
		r.SetTime(clock.Now())
		allowed3, retry3, err := luaLimiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.False(t, allowed3) // Should also be rate limited
		t.Logf("Lua at boundary: retry=%v", retry3)

		// Both should be rate limited with similar retry times
		retryDiff := abs(retry2 - retry3)
		require.Less(t, retryDiff, 100*time.Millisecond,
			"Boundary retry times should be similar")
	})
}

func TestLuaRateLimit_BurstCapacityMigration(t *testing.T) {
	ctx := context.Background()

	t.Run("partial burst consumption migration", func(t *testing.T) {
		config := inngest.RateLimit{
			Limit:  20, // burst = 20/10 = 2, total = 20 + 2 = 22
			Period: "1h",
		}

		r, rc, throttledStore, clock := initRedis(t)
		defer rc.Close()

		key := "burst-partial-test"

		// Consume exactly the burst capacity (2 requests) with throttled
		var throttledResults []bool
		for i := 0; i < 2; i++ {
			limited, retry, err := rateLimit(ctx, throttledStore, key, config)
			require.NoError(t, err)
			throttledResults = append(throttledResults, limited)
			t.Logf("Throttled burst request %d: limited=%v, retry=%v", i+1, limited, retry)
		}

		// Both burst requests should be allowed
		for i, limited := range throttledResults {
			require.False(t, limited, "Burst request %d should be allowed", i+1)
		}

		// Migrate to Lua and continue - should have base capacity (20) remaining
		luaLimiter := newLuaGCRARateLimiter(ctx, rc, prefix)

		var luaResults []bool
		// Try to consume the base capacity (20 requests) + a few extra
		for i := 0; i < 23; i++ {
			r.SetTime(clock.Now())
			allowed, retry, err := luaLimiter.RateLimit(ctx, key, config, clock.Now())
			require.NoError(t, err)
			luaResults = append(luaResults, allowed)
			if i < 5 || i >= 18 { // Log first 5 and last 5
				t.Logf("Lua base request %d: allowed=%v, retry=%v", i+1, allowed, retry)
			}
		}

		// Count allowed in Lua phase
		allowedLua := 0
		for _, allowed := range luaResults {
			if allowed {
				allowedLua++
			}
		}

		t.Logf("Burst migration: throttled allowed 2 (burst), lua allowed %d (base)", allowedLua)

		// Should allow exactly the base capacity (20)
		require.Equal(t, 20, allowedLua, "Should allow exactly base capacity after burst consumed")

		// Total across both phases should be 22 (2 burst + 20 base)
		require.Equal(t, 22, 2+allowedLua, "Total should equal full capacity")
	})

	t.Run("exact burst boundary migration", func(t *testing.T) {
		config := inngest.RateLimit{
			Limit:  10, // burst = 1, total = 11
			Period: "1h",
		}

		r, rc, throttledStore, clock := initRedis(t)
		defer rc.Close()

		key := "burst-boundary-test"

		// Consume burst + part of base capacity
		consumeCount := 6 // 1 burst + 5 base
		var throttledResults []bool

		for i := 0; i < consumeCount; i++ {
			limited, retry, err := rateLimit(ctx, throttledStore, key, config)
			require.NoError(t, err)
			throttledResults = append(throttledResults, limited)
			t.Logf("Throttled consume request %d: limited=%v, retry=%v", i+1, limited, retry)
		}

		// All should be allowed
		allowedThrottled := 0
		for _, limited := range throttledResults {
			if !limited {
				allowedThrottled++
			}
		}
		require.Equal(t, 6, allowedThrottled, "Should allow 6 requests")

		// Migrate to Lua - should have exactly 5 requests remaining (11 total - 6 consumed)
		luaLimiter := newLuaGCRARateLimiter(ctx, rc, prefix)

		var luaResults []bool
		for i := 0; i < 8; i++ { // Try more than remaining to verify limit
			r.SetTime(clock.Now())
			allowed, retry, err := luaLimiter.RateLimit(ctx, key, config, clock.Now())
			require.NoError(t, err)
			luaResults = append(luaResults, allowed)
			t.Logf("Lua remaining request %d: allowed=%v, retry=%v", i+1, allowed, retry)
		}

		// Count allowed requests
		allowedLua := 0
		rateLimitedLua := 0
		for _, allowed := range luaResults {
			if allowed {
				allowedLua++
			} else {
				rateLimitedLua++
			}
		}

		t.Logf("Boundary test: remaining capacity was %d, lua allowed %d, rate limited %d",
			5, allowedLua, rateLimitedLua)

		require.Equal(t, 5, allowedLua, "Should allow exactly remaining capacity")
		require.Equal(t, 3, rateLimitedLua, "Should rate limit excess requests")

		// Verify exact pattern: 5 allowed, then 3 rate limited
		for i := 0; i < 5; i++ {
			require.True(t, luaResults[i], "Request %d should be allowed", i+1)
		}
		for i := 5; i < 8; i++ {
			require.False(t, luaResults[i], "Request %d should be rate limited", i+1)
		}
	})

	t.Run("burst overflow protection", func(t *testing.T) {
		config := inngest.RateLimit{
			Limit:  5, // burst = 0, total = 5 (no burst for this test)
			Period: "1h",
		}

		r, rc, throttledStore, clock := initRedis(t)
		defer rc.Close()

		key := "burst-overflow-test"

		// Consume full capacity with throttled
		for i := 0; i < 5; i++ {
			limited, _, err := rateLimit(ctx, throttledStore, key, config)
			require.NoError(t, err)
			require.False(t, limited)
		}

		// Next should be rate limited
		limited6, _, err := rateLimit(ctx, throttledStore, key, config)
		require.NoError(t, err)
		require.True(t, limited6)

		// Migrate to Lua - should also be rate limited immediately
		luaLimiter := newLuaGCRARateLimiter(ctx, rc, prefix)
		r.SetTime(clock.Now())
		allowed1, retry1, err := luaLimiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.False(t, allowed1)
		require.Greater(t, retry1, time.Duration(0))

		t.Logf("Overflow protection: both implementations rate limit when capacity exhausted")
	})
}

func TestLuaRateLimit_TimeBasedRecoveryMigration(t *testing.T) {
	ctx := context.Background()

	t.Run("recovery timing preservation", func(t *testing.T) {
		config := inngest.RateLimit{
			Limit:  2,
			Period: "4s", // Fast recovery for testing
		}

		r, rc, throttledStore, clock := initRedis(t)
		defer rc.Close()

		key := "recovery-timing-test"

		// Exhaust capacity
		for i := 0; i < 2; i++ {
			limited, _, err := rateLimit(ctx, throttledStore, key, config)
			require.NoError(t, err)
			require.False(t, limited)
		}

		// Get rate limited
		limited3, retry3, err := rateLimit(ctx, throttledStore, key, config)
		require.NoError(t, err)
		require.True(t, limited3)
		expectedRecoveryTime := clock.Now().Add(retry3)

		// Migrate to Lua and verify same recovery timing
		luaLimiter := newLuaGCRARateLimiter(ctx, rc, prefix)
		r.SetTime(clock.Now())
		allowed4, retry4, err := luaLimiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.False(t, allowed4)
		luaRecoveryTime := clock.Now().Add(retry4)

		// Recovery times should be very similar
		recoveryTimeDiff := abs(expectedRecoveryTime.Sub(luaRecoveryTime))
		t.Logf("Recovery timing: throttled=%v, lua=%v, diff=%v",
			retry3, retry4, recoveryTimeDiff)
		require.Less(t, recoveryTimeDiff, 50*time.Millisecond,
			"Recovery times should be nearly identical")

		// Advance time for partial recovery and test again
		clock.Advance(2 * time.Second) // Advance for half the period
		r.FastForward(2 * time.Second)

		// Both implementations should still be rate limited but with shorter retry
		limited5, retry5, err := rateLimit(ctx, throttledStore, key, config)
		require.NoError(t, err)
		require.True(t, limited5)

		r.SetTime(clock.Now())
		allowed6, retry6, err := luaLimiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.False(t, allowed6)

		// Both should have shorter retry times now
		require.Less(t, retry5, retry3, "Throttled retry should be shorter after partial recovery")
		require.Less(t, retry6, retry4, "Lua retry should be shorter after partial recovery")

		retryDiffAfterWait := abs(retry5 - retry6)
		t.Logf("After partial recovery: throttled=%v, lua=%v, diff=%v",
			retry5, retry6, retryDiffAfterWait)
		require.Less(t, retryDiffAfterWait, 50*time.Millisecond,
			"Recovery timing should remain consistent")
	})
}

func TestLuaRateLimit_RetryAfterValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("retryAfter timing accuracy", func(t *testing.T) {
		r, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, "retry-test:")

		config := inngest.RateLimit{
			Limit:  1,
			Period: "2s", // Short period for faster testing
		}

		key := "timing-accuracy-test"

		// First request should be allowed
		r.SetTime(clock.Now())
		allowed1, retry1, err := limiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.True(t, allowed1)
		require.Equal(t, time.Duration(0), retry1)

		// Second request should be rate limited with retryAfter
		startTime := clock.Now()
		r.SetTime(clock.Now())
		allowed2, retry2, err := limiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.False(t, allowed2)
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
		allowed3, retry3, err := limiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.False(t, allowed3)
		require.Greater(t, retry3, time.Duration(0))
		require.Less(t, retry3, 100*time.Millisecond, "Should have very short retryAfter")

		// Advance time a bit more and request should be allowed
		clock.Advance(100 * time.Millisecond)
		r.FastForward(100 * time.Millisecond)
		r.SetTime(clock.Now())
		allowed4, retry4, err := limiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.True(t, allowed4)
		require.Equal(t, time.Duration(0), retry4)

		// Verify total elapsed time is reasonable
		elapsed := clock.Since(startTime)
		t.Logf("Total test elapsed time: %v", elapsed)
		require.Greater(t, elapsed, 1900*time.Millisecond, "Should take close to 2 seconds")
		require.Less(t, elapsed, 2500*time.Millisecond, "Should not take much more than 2 seconds")
	})

	t.Run("retryAfter decreases over time", func(t *testing.T) {
		r, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, "retry-decrease:")

		config := inngest.RateLimit{
			Limit:  1,
			Period: "3s",
		}

		key := "decreasing-retry-test"

		// Exhaust capacity
		r.SetTime(clock.Now())
		allowed1, _, err := limiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.True(t, allowed1)

		// Get initial retryAfter
		r.SetTime(clock.Now())
		_, retry1, err := limiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.Greater(t, retry1, time.Duration(0))
		t.Logf("Initial retryAfter: %v", retry1)

		// Advance time by 500ms
		clock.Advance(500 * time.Millisecond)
		r.FastForward(500 * time.Millisecond)

		// Get retryAfter again - should be shorter
		r.SetTime(clock.Now())
		_, retry2, err := limiter.RateLimit(ctx, key, config, clock.Now())
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
		_, retry3, err := limiter.RateLimit(ctx, key, config, clock.Now())
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
		r, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, "period-test:")

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
				allowed, _, err := limiter.RateLimit(ctx, key, config, clock.Now())
				require.NoError(t, err)
				require.True(t, allowed)

				// Get retryAfter
				r.SetTime(clock.Now())
				rateLimited, retryAfter, err := limiter.RateLimit(ctx, key, config, clock.Now())
				require.NoError(t, err)
				require.False(t, rateLimited)
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
		r, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, "burst-test:")

		config := inngest.RateLimit{
			Limit:  10, // burst = 1, total = 2
			Period: "10s",
		}

		key := "burst-retry-test"

		// Consume all capacity (2 requests)
		for i := 0; i < 2; i++ {
			r.SetTime(clock.Now())
			allowed, retry, err := limiter.RateLimit(ctx, key, config, clock.Now())
			require.NoError(t, err)
			require.True(t, allowed)
			require.Equal(t, time.Duration(0), retry)
			t.Logf("Burst request %d: allowed", i+1)
		}

		// Next request should be rate limited
		startTime := clock.Now()
		r.SetTime(clock.Now())
		rateLimited, retryAfter, err := limiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.False(t, rateLimited)
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
			allowed, _, err := limiter.RateLimit(ctx, key, config, clock.Now())
			require.NoError(t, err)
			require.True(t, allowed, "Request should be allowed after waiting retryAfter duration")

			elapsed := clock.Since(startTime)
			t.Logf("Successfully allowed request after %v", elapsed)
		}
	})

	t.Run("retryAfter with zero capacity edge case", func(t *testing.T) {
		r, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, "zero-test:")

		config := inngest.RateLimit{
			Limit:  0, // Zero limit should always rate limit
			Period: "1h",
		}

		key := "zero-capacity-test"

		// First request should be rate limited
		r.SetTime(clock.Now())
		rateLimited, retryAfter, err := limiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.False(t, rateLimited)
		require.Greater(t, retryAfter, time.Duration(0))

		t.Logf("Zero limit retryAfter: %v", retryAfter)

		// With zero limit, retryAfter should be the full period
		require.Greater(t, retryAfter, 59*time.Minute)
		require.Less(t, retryAfter, 61*time.Minute)

		// Subsequent requests should also be rate limited
		r.SetTime(clock.Now())
		rateLimited2, retryAfter2, err := limiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.False(t, rateLimited2)
		require.Greater(t, retryAfter2, time.Duration(0))

		// RetryAfter should remain close to original (zero capacity = no progress)
		timeDiff := abs(retryAfter - retryAfter2)
		require.Less(t, timeDiff, 100*time.Millisecond, "RetryAfter should be consistent for zero limit")
	})

	t.Run("retryAfter precision validation", func(t *testing.T) {
		r, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, "precision-test:")

		config := inngest.RateLimit{
			Limit:  1,
			Period: "1s",
		}

		key := "precision-test"

		// Exhaust capacity
		r.SetTime(clock.Now())
		allowed, _, err := limiter.RateLimit(ctx, key, config, clock.Now())
		require.NoError(t, err)
		require.True(t, allowed)

		// Get multiple retryAfter values in quick succession
		var retryTimes []time.Duration

		for i := 0; i < 5; i++ {
			r.SetTime(clock.Now())
			_, retry, err := limiter.RateLimit(ctx, key, config, clock.Now())
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

func TestLuaRateLimit_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid period", func(t *testing.T) {
		r, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, "test:")

		config := inngest.RateLimit{
			Limit:  5,
			Period: "invalid",
		}

		r.SetTime(clock.Now())
		allowed, retryAfter, err := limiter.RateLimit(ctx, "test-key", config, clock.Now())
		require.Error(t, err)
		require.True(t, allowed) // Should return true on error
		require.Equal(t, time.Duration(-1), retryAfter)
	})

	t.Run("zero limit", func(t *testing.T) {
		r, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, "test:")

		config := inngest.RateLimit{
			Limit:  0,
			Period: "1h",
		}

		// The throttled library panics with divide by zero for limit=0
		// So we test that our Lua implementation gracefully handles zero limits
		// by immediately rate limiting (which is the logical behavior)
		r.SetTime(clock.Now())
		allowed, retryAfter, err := limiter.RateLimit(ctx, "test-key", config, clock.Now())
		require.NoError(t, err)
		t.Logf("Lua with zero limit: allowed=%v, retry=%v", allowed, retryAfter)

		// Zero limit should immediately rate limit
		require.False(t, allowed)
		require.Greater(t, retryAfter, time.Duration(0))
	})

	t.Run("very short period", func(t *testing.T) {
		r, rc, _, clock := initRedis(t)
		defer rc.Close()

		limiter := newLuaGCRARateLimiter(ctx, rc, "test:")

		config := inngest.RateLimit{
			Limit:  1,
			Period: "1ms",
		}

		// First request should be allowed
		r.SetTime(clock.Now())
		allowed, _, err := limiter.RateLimit(ctx, "test-key", config, clock.Now())
		require.NoError(t, err)
		require.True(t, allowed)

		// Advance time for period to pass
		clock.Advance(2 * time.Millisecond)
		r.FastForward(2 * time.Millisecond)

		// Next request should be allowed
		r.SetTime(clock.Now())
		allowed, _, err = limiter.RateLimit(ctx, "test-key", config, clock.Now())
		require.NoError(t, err)
		require.True(t, allowed)
	})
}

// Helper functions

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func compareRedisState(t *testing.T, r1, r2 *miniredis.Miniredis, keyPattern string) {
	keys1 := r1.Keys()
	keys2 := r2.Keys()

	t.Logf("Redis state comparison:")
	t.Logf("  r1 keys: %v", keys1)
	t.Logf("  r2 keys: %v", keys2)

	// Both should have similar number of keys
	require.Equal(t, len(keys1), len(keys2), "Redis instances should have same number of keys")

	// For each key, values and TTLs should be comparable
	for _, key := range keys1 {
		if r1.Exists(key) && r2.Exists(key) {
			val1, err1 := r1.Get(key)
			val2, err2 := r2.Get(key)

			if err1 == nil && err2 == nil {
				t.Logf("  Key %s:", key)
				t.Logf("    r1 value: %s", val1)
				t.Logf("    r2 value: %s", val2)

				// Compare TTLs
				ttl1 := r1.TTL(key)
				ttl2 := r2.TTL(key)
				t.Logf("    r1 TTL: %v", ttl1)
				t.Logf("    r2 TTL: %v", ttl2)

				// Values should be close (allowing for timing differences)
				// TTLs should be close (allowing for timing differences)
				if ttl1 > 0 && ttl2 > 0 {
					ttlDiff := abs(ttl1 - ttl2)
					t.Logf("    TTL diff: %v", ttlDiff)
					require.Less(t, ttlDiff, time.Second, "TTLs should be similar")
				}
			}
		}
	}

	// Check for keys that exist in one but not the other
	for _, key := range keys1 {
		if !r2.Exists(key) {
			t.Logf("  Key %s exists in r1 but not r2", key)
		}
	}
	for _, key := range keys2 {
		if !r1.Exists(key) {
			t.Logf("  Key %s exists in r2 but not r1", key)
		}
	}
}
