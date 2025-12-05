package redis_state

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/jonboulle/clockwork"
	goredis "github.com/redis/go-redis/v9"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/goredisstore.v9"
	"github.com/xhit/go-str2duration/v2"
)

func throttledRateLimit(ctx context.Context, store throttled.GCRAStoreCtx, key string, period time.Duration, limit, burst int) (bool, error) {
	quota := throttled.RateQuota{
		MaxRate:  throttled.PerDuration(limit, period),
		MaxBurst: burst,
	}

	limiter, err := throttled.NewGCRARateLimiterCtx(store, quota)
	if err != nil {
		log.Fatal(err)
	}

	limited, _, err := limiter.RateLimitCtx(ctx, key, 1)
	if err != nil {
		return false, err
	}

	return limited, nil
}

func initRedisWithThrottledStore(t *testing.T) (*miniredis.Miniredis, rueidis.Client, throttled.GCRAStoreCtx, clockwork.FakeClock) {
	r := miniredis.RunT(t)

	// Create rueidis client for Lua implementation
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	// Create go-redis client for throttled store
	goredisClient := goredis.NewClient(&goredis.Options{
		Addr: r.Addr(),
	})

	// Create throttled store using goredisstore
	store, err := goredisstore.NewCtx(goredisClient, "throttled:")
	require.NoError(t, err)

	clock := clockwork.NewFakeClock()
	// Set miniredis time to match fake clock
	r.SetTime(clock.Now())

	return r, rc, store, clock
}

func TestOldLuaGCRA_SideBySideComparison(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name              string
		limit             int
		period            string
		burst             int
		requests          int
		advancePerRequest string // time to advance clock between requests (e.g., "0s", "1s", "emission")
	}{
		// Burst tests (no time advancement)
		{"1 per hour, no burst", 1, "1h", 0, 3, "0s"},
		{"5 per hour, burst 0", 5, "1h", 0, 10, "0s"},
		{"10 per minute, burst 1", 10, "1m", 1, 15, "0s"},
		{"10 per minute, burst 5", 10, "1m", 5, 20, "0s"},
		{"100 per hour, burst 10", 100, "1h", 10, 120, "0s"},
		{"1 per minute, no burst", 1, "1m", 0, 3, "0s"},
		{"1 per second, no burst", 1, "1s", 0, 3, "0s"},
		{"20 per second, burst 2", 20, "1s", 2, 25, "0s"},
		{"50 per minute, burst 5", 50, "1m", 5, 60, "0s"},
		{"5 per second, burst 10", 5, "1s", 10, 20, "0s"},
		{"100 per minute, burst 20", 100, "1m", 20, 130, "0s"},
		{"3 per hour, burst 1", 3, "1h", 1, 5, "0s"},
		{"10 per hour, burst 3", 10, "1h", 3, 15, "0s"},

		// Time-based tests (advancing by emission interval - should allow all requests)
		{"10 per second, advance by emission", 10, "1s", 0, 15, "emission"},
		{"5 per minute, advance by emission", 5, "1m", 0, 10, "emission"},
		{"100 per hour, advance by emission", 100, "1h", 0, 120, "emission"},
		{"20 per second with burst, advance by emission", 20, "1s", 5, 30, "emission"},

		// Time-based tests (advancing by fixed duration)
		{"10 per second, advance 100ms (should limit)", 10, "1s", 0, 15, "100ms"},
		{"5 per minute, advance 10s (should allow)", 5, "1m", 0, 8, "10s"},
		{"100 per hour, advance 30s (should limit)", 100, "1h", 0, 10, "30s"},
		{"1 per second, advance 1s (should allow)", 1, "1s", 0, 5, "1s"},
		{"1 per second, advance 500ms (should limit)", 1, "1s", 0, 5, "500ms"},

		// Mixed burst and time tests
		{"10 per minute, burst 3, advance 6s", 10, "1m", 3, 15, "6s"},
		{"5 per second, burst 2, advance 200ms", 5, "1s", 2, 12, "200ms"},
		{"100 per hour, burst 10, advance 36s", 100, "1h", 10, 20, "36s"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up both implementations with separate Redis instances
			rThrottled, rcThrottled, throttledStore, _ := initRedisWithThrottledStore(t)
			defer rcThrottled.Close()

			rLua, rcLua, _, clock := initRedisWithThrottledStore(t)
			defer rcLua.Close()

			// Parse period for both implementations
			period, err := str2duration.ParseDuration(tc.period)
			require.NoError(t, err)

			key := "test-key"

			// normalize burst. lua treats burst as total requests allowed, whereas throttled considers it additional
			luaBurst := tc.burst
			throttledBurst := tc.burst - 1
			if throttledBurst < 0 {
				throttledBurst = 0
			}

			// Parse time advancement
			var advancePerRequest time.Duration
			if tc.advancePerRequest == "emission" {
				// Calculate emission interval (period / limit)
				advancePerRequest = period / time.Duration(tc.limit)
				t.Logf("Using emission interval: %v", advancePerRequest)
			} else {
				advancePerRequest, err = str2duration.ParseDuration(tc.advancePerRequest)
				require.NoError(t, err)
			}

			// Make requests to both implementations
			for i := 0; i < tc.requests; i++ {
				// Test Lua implementation using oldLuaGCRARateLimit
				fakeNow := clock.Now()
				rLua.SetTime(fakeNow)
				luaLimited, err := oldLuaGCRARateLimit(ctx, rcLua, key, fakeNow, period, tc.limit, luaBurst)
				require.NoError(t, err)

				// Test throttled implementation
				rThrottled.SetTime(fakeNow)
				throttledLimited, err := throttledRateLimit(ctx, throttledStore, key, period, tc.limit, throttledBurst)
				require.NoError(t, err)

				t.Logf("Request %d (at %v): lua(limited=%v) vs throttled(limited=%v)",
					i+1, fakeNow, luaLimited, throttledLimited)

				require.Equal(t, throttledLimited, luaLimited,
					"request %d: limited status should match (throttled_limited=%v, lua_limited=%v)",
					i+1, throttledLimited, luaLimited)

				// Advance clocks for next request
				if i < tc.requests-1 && advancePerRequest > 0 {
					clock.Advance(advancePerRequest)
					rThrottled.FastForward(advancePerRequest)
					rLua.FastForward(advancePerRequest)
				}
			}

			// Compare Redis state between implementations
			compareRedisState(t, rThrottled, rLua, key)
		})
	}
}

func compareRedisState(t *testing.T, rThrottled, rLua *miniredis.Miniredis, baseKey string) {
	keysThrottled := rThrottled.Keys()
	keysLua := rLua.Keys()

	t.Logf("Redis state comparison:")
	t.Logf("  throttled keys: %v", keysThrottled)
	t.Logf("  lua keys: %v", keysLua)

	// Both should have same number of keys
	require.Equal(t, len(keysThrottled), len(keysLua), "Redis instances should have same number of keys")

	// throttled stores keys with "throttled:" prefix, lua uses plain keys
	throttledKey := "throttled:" + baseKey
	luaKey := baseKey

	// Both should have their respective keys
	require.True(t, rThrottled.Exists(throttledKey), "throttled should have key: %s", throttledKey)
	require.True(t, rLua.Exists(luaKey), "lua should have key: %s", luaKey)

	// Compare values and TTLs
	valThrottled, errThrottled := rThrottled.Get(throttledKey)
	valLua, errLua := rLua.Get(luaKey)

	require.NoError(t, errThrottled, "should get throttled value")
	require.NoError(t, errLua, "should get lua value")

	t.Logf("  Comparing key: throttled:'%s' vs lua:'%s'", throttledKey, luaKey)
	t.Logf("    throttled value: %s", valThrottled)
	t.Logf("    lua value: %s", valLua)

	// Compare TTLs (for informational purposes - implementations may differ)
	ttlThrottled := rThrottled.TTL(throttledKey)
	ttlLua := rLua.TTL(luaKey)
	t.Logf("    throttled TTL: %v", ttlThrottled)
	t.Logf("    lua TTL: %v", ttlLua)

	// Log TTL difference for reference (don't assert - implementations calculate TTLs differently)
	if ttlThrottled > 0 && ttlLua > 0 {
		ttlDiff := abs(ttlThrottled - ttlLua)
		t.Logf("    TTL diff: %v", ttlDiff)
	}
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
