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
		require.Equal(t, 110, runScript(t, rc, key, clock.Now(), period, limit, burst, 0))
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
		require.Equal(t, 0, runScript(t, rc, key, clock.Now(), period, limit, burst, 2))

		clock.Advance(2 * time.Hour)
		r.FastForward(2 * time.Hour)

		// Should match initial capacity
		require.Equal(t, 5, runScript(t, rc, key, clock.Now(), period, limit, burst, 0))
		require.Len(t, r.Keys(), 0)
	})

	type action struct {
		delay time.Duration

		consumeCapacity int

		capacityBefore int
		capacityAfter  int
	}

	type tableTest struct {
		name string

		actions []action
		period  time.Duration
		limit   int
		burst   int
	}

	tests := []tableTest{
		{
			name:   "basic limits without passing time",
			period: time.Hour,
			limit:  10,
			burst:  0,
			actions: []action{
				{
					delay:           0,
					capacityBefore:  10,
					consumeCapacity: 0,
					capacityAfter:   10,
				},
				{
					delay:           0,
					capacityBefore:  10,
					consumeCapacity: 5,
					capacityAfter:   5,
				},
				{
					delay:           0,
					capacityBefore:  5,
					consumeCapacity: 5,
					capacityAfter:   0,
				},
				{
					delay:           1 * time.Hour,
					capacityBefore:  10,
					consumeCapacity: 0,
					capacityAfter:   10,
				},
			},
		},
		{
			name:   "basic limits with passing time",
			period: 10 * time.Hour,
			limit:  100,
			// refill 10 every hour
			burst: 0,
			actions: []action{
				{
					delay:           0,
					capacityBefore:  100,
					consumeCapacity: 0,
					capacityAfter:   100,
				},
				{
					delay:           0,
					capacityBefore:  100,
					consumeCapacity: 20,
					capacityAfter:   80,
				},
				{
					delay:           1 * time.Hour,
					capacityBefore:  90, // assume 10 items got refilled since 1 hour passed
					consumeCapacity: 90,
					capacityAfter:   0,
				},
				{
					delay:           1 * time.Hour,
					capacityBefore:  10, // assume 10 items got refilled again
					consumeCapacity: 0,
					capacityAfter:   10,
				},
			},
		},
		{
			name:   "with burst",
			period: 1 * time.Hour,
			limit:  10,
			// 10 are refilled per hour, or 1 every 10 minutes
			burst: 2,
			actions: []action{
				{
					delay:           0,
					capacityBefore:  12,
					consumeCapacity: 0,
					capacityAfter:   12,
				},
				{
					delay:           0,
					capacityBefore:  12,
					consumeCapacity: 5,
					capacityAfter:   7,
				},
				{
					delay:           10 * time.Minute,
					capacityBefore:  8, // assume 1 item got refilled
					consumeCapacity: 0,
					capacityAfter:   8,
				},
				{
					delay:           5 * time.Minute,
					capacityBefore:  9, // assume 1 item got refilled
					consumeCapacity: 0,
					capacityAfter:   9,
				},
				{
					delay:           0,
					capacityBefore:  9,
					consumeCapacity: 9,
					capacityAfter:   0,
				},
				{
					delay:           60 * time.Minute,
					capacityBefore:  10,
					consumeCapacity: 0,
					capacityAfter:   10,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clock := clockwork.NewFakeClock()
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{
				InitAddress:  []string{r.Addr()},
				DisableCache: true,
			})
			require.NoError(t, err)
			defer rc.Close()

			key := "test"

			current := clock.Now()
			for i, a := range test.actions {
				current = current.Add(a.delay)

				if a.capacityBefore > 0 {
					require.Equal(t, a.capacityBefore, runScript(t, rc, key, current, test.period, test.limit, test.burst, 0), "capacity before in action %d failed", i)
				}

				if a.consumeCapacity > 0 {
					require.Equal(t, 0, runScript(t, rc, key, current, test.period, test.limit, test.burst, a.consumeCapacity), "gcra update in action %d failed", i)
				}
				require.Equal(t, a.capacityAfter, runScript(t, rc, key, current, test.period, test.limit, test.burst, 0), "capacity after in action %d failed", i)
			}
		})
	}
}
