package redis_state

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGCRA(t *testing.T) {

	runScript := func(t *testing.T, rc rueidis.Client, key string, now time.Time, period time.Duration, limit, burst, capacity int) int {
		args, err := StrSlice([]any{
			key,
			now.UnixMilli(),
			limit,
			burst,
			period.Milliseconds(),
			capacity,
		})
		require.NoError(t, err)

		statusOrCapacity, err := scripts["queue/gcraTest"].Exec(context.Background(), rc, []string{}, args).ToInt64()
		require.NoError(t, err)

		switch statusOrCapacity {
		case -1:
			return 0
		default:
			return int(statusOrCapacity)
		}
	}

	t.Run("should reduce throttle capacity", func(t *testing.T) {
		clock := clockwork.NewFakeClock()
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		key := "test"

		period := 1 * time.Hour
		limit := 100
		burst := 10

		// Read initial capacity
		require.Equal(t, 110, runScript(t, rc, key, clock.Now(), period, limit, burst, 0))
		require.Len(t, r.Keys(), 0)

		// "Start" one run
		require.Equal(t, 0, runScript(t, rc, key, clock.Now(), period, limit, burst, 1))
		require.Len(t, r.Keys(), 1)
		require.True(t, r.Exists(key))
		require.Equal(t, 109, runScript(t, rc, key, clock.Now(), period, limit, burst, 0))

		clock.Advance(2 * time.Hour)
		r.FastForward(2 * time.Hour)

		// Should match initial capacity
		require.Equal(t, 109, runScript(t, rc, key, clock.Now(), period, limit, burst, 0))
		require.Len(t, r.Keys(), 0)
	})

	t.Run("should prevent overflowing", func(t *testing.T) {
		clock := clockwork.NewFakeClock()
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		defer rc.Close()

		key := "test"

		period := 1 * time.Hour
		limit := 5
		burst := 0

		// Read initial capacity
		require.Equal(t, 5, runScript(t, rc, key, clock.Now(), period, limit, burst, 0))
		require.Len(t, r.Keys(), 0)

		// "Start" 5 runs
		require.Equal(t, 0, runScript(t, rc, key, clock.Now(), period, limit, burst, 5))
		require.Len(t, r.Keys(), 1)
		require.True(t, r.Exists(key))
		require.Equal(t, 0, runScript(t, rc, key, clock.Now(), period, limit, burst, 0))

		clock.Advance(2 * time.Hour)
		r.FastForward(2 * time.Hour)

		// Should match initial capacity
		require.Equal(t, 5, runScript(t, rc, key, clock.Now(), period, limit, burst, 0))
		require.Len(t, r.Keys(), 0)
	})
}
