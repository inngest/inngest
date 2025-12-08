package redis_state

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func oldLuaGCRARateLimit(ctx context.Context, rc rueidis.Client, key string, now time.Time, period time.Duration, limit, burst int) (bool, error) {
	nowMS := now.UnixMilli()
	args, err := StrSlice([]any{
		key,
		nowMS,
		period.Milliseconds(),
		limit,
		burst,
	})
	if err != nil {
		return false, err
	}
	res, err := scripts["test/gcra_ratelimit"].Exec(ctx, rc, []string{}, args).AsInt64()
	if err != nil {
		return false, err
	}
	// lua gcra() returns 1 on success (allowed), 0 if rate limited
	// we return true if limited, false if allowed (to match throttled interface)
	return res == 0, nil
}

func TestOldLuaGCRA(t *testing.T) {
	getThrottleState := func(t *testing.T, r *miniredis.Miniredis, key string) time.Time {
		value, err := r.Get(key)
		require.NoError(t, err)

		parsed, err := strconv.Atoi(value)
		require.NoError(t, err)

		return time.UnixMilli(int64(parsed))
	}

	ctx := context.Background()

	t.Run("should allow initial request", func(t *testing.T) {
		clock := clockwork.NewFakeClock()
		r, rc := initRedis(t)
		defer rc.Close()
		key := "test:throttle:1"
		period := 1 * time.Hour
		limit := 10
		burst := 0
		// First request should succeed (not be limited)
		limited, err := oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.False(t, limited, "first request should not be limited")

		require.WithinDuration(t, clock.Now().Add(6*time.Minute), getThrottleState(t, r, key), time.Second)
	})

	t.Run("should rate limit with minimum burst", func(t *testing.T) {
		clock := clockwork.NewFakeClock()
		r, rc := initRedis(t)
		defer rc.Close()
		key := "test:throttle:2"
		period := 1 * time.Hour
		limit := 5
		burst := 0 // Due to math.max(burst, 1), this behaves as burst=1
		// First request should succeed (not be limited)
		limited, err := oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.False(t, limited, "first request should not be limited")

		require.WithinDuration(t, clock.Now().Add(time.Hour/5), getThrottleState(t, r, key), time.Second)

		// Second immediate request should be rate limited (burst=1 allows only 1 request)
		limited, err = oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.True(t, limited, "second immediate request should be rate limited with burst=0")
	})

	t.Run("should allow requests after period expires", func(t *testing.T) {
		clock := clockwork.NewFakeClock()
		r, rc := initRedis(t)
		defer rc.Close()
		key := "test:throttle:3"
		period := 1 * time.Second
		limit := 1
		burst := 0
		// Make first request
		limited, err := oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.False(t, limited, "first request should not be limited")

		require.WithinDuration(t, clock.Now().Add(time.Second), getThrottleState(t, r, key), 10*time.Millisecond)

		// Immediate second request should be rate limited
		limited, err = oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.True(t, limited, "immediate second request should be rate limited")

		// Advance time past the period
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		// Request should be allowed again
		limited, err = oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.False(t, limited, "request after period should not be limited")
	})

	t.Run("should handle burst allowance", func(t *testing.T) {
		clock := clockwork.NewFakeClock()
		_, rc := initRedis(t)
		defer rc.Close()
		key := "test:throttle:4"
		period := 1 * time.Second
		limit := 5
		burst := 3
		// With burst=3, we can make approximately 3 immediate requests
		// (the exact behavior depends on GCRA variance calculation)
		for i := 0; i < 3; i++ {
			limited, err := oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
			require.NoError(t, err)
			require.False(t, limited, "request %d should not be limited (within burst)", i+1)
		}
		// Next immediate request should be rate limited
		limited, err := oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.True(t, limited, "request exceeding burst should be rate limited")
	})
	t.Run("should handle different keys independently", func(t *testing.T) {
		clock := clockwork.NewFakeClock()
		_, rc := initRedis(t)
		defer rc.Close()
		key1 := "test:throttle:key1"
		key2 := "test:throttle:key2"
		period := 1 * time.Hour
		limit := 5
		burst := 0
		// Make one request on key1 (should succeed)
		limited, err := oldLuaGCRARateLimit(ctx, rc, key1, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.False(t, limited, "key1 first request should not be limited")
		// Second immediate request on key1 should be rate limited (burst=1)
		limited, err = oldLuaGCRARateLimit(ctx, rc, key1, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.True(t, limited, "key1 should be rate limited")
		// key2 should still work independently
		limited, err = oldLuaGCRARateLimit(ctx, rc, key2, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.False(t, limited, "key2 should not be limited")
	})
	t.Run("should allow requests after emission interval", func(t *testing.T) {
		clock := clockwork.NewFakeClock()
		r, rc := initRedis(t)
		defer rc.Close()
		key := "test:throttle:5"
		period := 10 * time.Second
		limit := 10
		burst := 0
		// First request should be allowed
		limited, err := oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.False(t, limited, "first request should not be limited")
		// Immediate second request should be rate limited
		limited, err = oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.True(t, limited, "immediate second request should be rate limited")
		// Advance time by emission interval (period/limit)
		emissionInterval := period / time.Duration(limit)
		clock.Advance(emissionInterval)
		r.FastForward(emissionInterval)
		// Request should be allowed after emission interval
		limited, err = oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.False(t, limited, "request after emission interval should not be limited")
	})
	t.Run("should handle larger burst values", func(t *testing.T) {
		clock := clockwork.NewFakeClock()
		r, rc := initRedis(t)
		defer rc.Close()
		key := "test:throttle:6"
		period := 1 * time.Second
		limit := 10
		burst := 10
		// With burst=10, we should be able to make 10 immediate requests
		for i := 0; i < 10; i++ {
			limited, err := oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
			require.NoError(t, err)
			require.False(t, limited, "request %d should not be limited within burst", i+1)
		}
		// 11th immediate request should be denied
		limited, err := oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.True(t, limited, "request beyond burst should be limited")
		// After waiting the full period, should be able to make requests again
		clock.Advance(period)
		r.FastForward(period)
		limited, err = oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
		require.NoError(t, err)
		require.False(t, limited, "request after full period should not be limited")
	})
}
