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

func oldLuaGCRARateLimit(ctx context.Context, rc rueidis.Client, key string, now time.Time, period time.Duration, limit, burst int) (bool, bool, error) {
	nowMS := now.UnixMilli()
	args, err := StrSlice([]any{
		key,
		nowMS,
		period.Milliseconds(),
		limit,
		burst,
	})
	if err != nil {
		return false, false, err
	}

	// lua gcra() returns 1 on success (allowed), 2 on success with burst used (allowed), and 0 if rate limited
	res, err := scripts["test/gcra_ratelimit"].Exec(ctx, rc, []string{}, args).AsInt64()
	if err != nil {
		return false, false, err
	}

	usedBurst := res == 2

	// we return true if limited, false if allowed (to match throttled interface)
	return res == 0, usedBurst, nil
}

func TestBurstUsage(t *testing.T) {
	getThrottleState := func(t *testing.T, r *miniredis.Miniredis, key string) time.Time {
		value, err := r.Get(key)
		require.NoError(t, err)

		parsed, err := strconv.Atoi(value)
		require.NoError(t, err)

		return time.UnixMilli(int64(parsed))
	}

	ctx := context.Background()

	matrix := []struct {
		name string
	}{
		{
			name: "with fix",
		},
	}

	for _, tc := range matrix {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("should not use burst when not needed", func(t *testing.T) {
				clock := clockwork.NewFakeClock()
				r, rc := initRedis(t)
				defer rc.Close()
				key := "test:throttle:1"
				period := 1 * time.Hour
				limit := 10
				burst := 0
				// First request should succeed (not be limited)
				limited, usedBurst, err := oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
				require.NoError(t, err)
				require.False(t, limited, "first request should not be limited")
				require.False(t, usedBurst)

				require.WithinDuration(t, clock.Now().Add(6*time.Minute), getThrottleState(t, r, key), time.Second)
			})

			t.Run("should use burst capacity", func(t *testing.T) {
				clock := clockwork.NewFakeClock()
				r, rc := initRedis(t)
				defer rc.Close()
				key := "test:throttle:1"
				period := time.Minute
				limit := 10
				burst := 5

				// First request should succeed (not be limited)
				limited, usedBurst, err := oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
				require.NoError(t, err)
				require.False(t, limited)
				require.False(t, usedBurst)

				require.WithinDuration(t, clock.Now().Add(6*time.Second), getThrottleState(t, r, key), time.Second)

				limited, usedBurst, err = oldLuaGCRARateLimit(ctx, rc, key, clock.Now(), period, limit, burst)
				require.NoError(t, err)
				require.False(t, limited)
				require.True(t, usedBurst)
			})
		})
	}
}
