package constraintapi

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestThrottleGCRA(t *testing.T) {
	type gcraScriptOptions struct {
		key      string
		now      time.Time
		period   time.Duration
		limit    int
		burst    int
		quantity int
	}

	type rateLimitResult struct {
		Limited bool `json:"limited"`

		// Limit is the maximum number of requests that could be permitted
		// instantaneously for this key starting from an empty state. For
		// example, if a rate limiter allows 10 requests per second per
		// key, Limit would always be 10.
		Limit int `json:"limit"`

		Usage int64 `json:"u"`

		// Remaining is the maximum number of requests that could be
		// permitted instantaneously for this key given the current
		// state. For example, if a rate limiter allows 10 requests per
		// second and has already received 6 requests for this key this
		// second, Remaining would be 4.
		Remaining int `json:"remaining"`

		// ResetAfter is the time until the RateLimiter returns to its
		// initial state for a given key. For example, if a rate limiter
		// manages requests per second and received one request 200ms ago,
		// Reset would return 800ms. You can also think of this as the time
		// until Limit and Remaining will be equal.
		ResetAfterMS int64 `json:"reset_after"`

		// RetryAfter is the time until the next request will be permitted.
		// It should be -1 unless the rate limit has been exceeded.
		RetryAfterMS int64 `json:"retry_after"`

		// RetryAt is the time the next request is permitted
		// This assumes all capacity is consumed in the request.
		RetryAtMS int64 `json:"retry_at"`

		EmissionInterval int64 `json:"ei"`
		DVT              int64 `json:"dvt"`

		TAT    int64 `json:"tat"`
		NewTAT int64 `json:"ntat"`

		Increment int64 `json:"inc"`
		AllowAt   int64 `json:"aat"`

		Diff int64 `json:"diff"`

		Next int64 `json:"next"`
	}

	initRedis := func(t *testing.T) (*miniredis.Miniredis, rueidis.Client) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		return r, rc
	}

	runScript := func(t *testing.T, rc rueidis.Client, opts gcraScriptOptions) rateLimitResult {
		nowMS := opts.now.UnixMilli()
		args, err := strSlice([]any{
			opts.key,
			nowMS,
			opts.limit,
			opts.burst,
			opts.period.Milliseconds(),
			opts.quantity,
		})
		require.NoError(t, err)

		rawRes, err := scripts["test/throttle"].Exec(t.Context(), rc, []string{}, args).ToString()
		require.NoError(t, err)

		var res rateLimitResult
		err = json.Unmarshal([]byte(rawRes), &res)
		require.NoError(t, err)

		return res
	}

	t.Run("should return gcra result struct", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 1 * time.Minute
		limit := 10
		burst := 0

		// Read initial capacity
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.False(t, res.Limited)

		require.Equal(t, (6 * time.Second).Milliseconds(), res.EmissionInterval)
		require.Equal(t, res.TAT, clock.Now().UnixMilli())
		require.WithinDuration(t, clock.Now().Add(6*time.Second), time.UnixMilli(res.NewTAT), time.Second)
		require.Equal(t, (6 * time.Second).Milliseconds(), res.DVT)
		require.Equal(t, (6 * time.Second).Milliseconds(), res.Increment)
		require.Equal(t, clock.Now().UnixMilli(), res.AllowAt)
		require.Equal(t, int64(0), res.Diff)

		require.Equal(t, 1, res.Limit)
		require.Equal(t, 1, res.Remaining)
		require.Equal(t, time.Duration(0), time.Duration(res.ResetAfterMS)*time.Millisecond)
		require.Equal(t, time.Duration(0), time.Duration(res.RetryAfterMS)*time.Millisecond)
		require.Equal(t, clock.Now().Add(6*time.Second).UnixMilli(), res.RetryAtMS)
	})

	t.Run("consume 1 should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 1 * time.Minute
		limit := 10
		burst := 0

		// First request should be admitted
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.False(t, res.Limited)
		require.Equal(t, (6 * time.Second).Milliseconds(), res.EmissionInterval)
		require.Equal(t, res.TAT, clock.Now().UnixMilli())
		require.WithinDuration(t, clock.Now().Add(6*time.Second), time.UnixMilli(res.NewTAT), time.Second)
		require.Equal(t, (6 * time.Second).Milliseconds(), res.DVT)
		require.Equal(t, (6 * time.Second).Milliseconds(), res.Increment)
		require.Equal(t, clock.Now().UnixMilli(), res.AllowAt)
		require.Equal(t, int64(0), res.Diff)

		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, 6*time.Second, time.Duration(res.ResetAfterMS)*time.Millisecond)
		require.Equal(t, time.Duration(0), time.Duration(res.RetryAfterMS)*time.Millisecond)
		require.Equal(t, clock.Now().Add(6*time.Second).UnixMilli(), res.RetryAtMS)
	})

	t.Run("consume 1 with burst 1 should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 1 * time.Minute
		limit := 10
		burst := 1

		// Read initial capacity
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 2,
		})

		require.False(t, res.Limited)
		require.Equal(t, (6 * time.Second).Milliseconds(), res.EmissionInterval)
		require.Equal(t, res.TAT, clock.Now().UnixMilli())
		require.WithinDuration(t, clock.Now().Add(2*6*time.Second), time.UnixMilli(res.NewTAT), time.Second)
		require.Equal(t, (12 * time.Second).Milliseconds(), res.DVT)
		require.Equal(t, (2 * 6 * time.Second).Milliseconds(), res.Increment)
		require.Equal(t, clock.Now().Add(2*6*time.Second).Add(-12*time.Second).UnixMilli(), res.AllowAt)
		require.Equal(t, (0 * time.Second).Milliseconds(), res.Diff)

		require.Equal(t, 2, res.Limit)
		require.Equal(t, 0, res.Remaining)
		// Accounts for burst
		require.Equal(t, 2*6*time.Second, time.Duration(res.ResetAfterMS)*time.Millisecond)
		require.Equal(t, time.Duration(0), time.Duration(res.RetryAfterMS)*time.Millisecond)
		require.Equal(t, clock.Now().Add(6*time.Second).UnixMilli(), res.RetryAtMS)
	})

	t.Run("being limited should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 1 * time.Minute
		limit := 10
		burst := 0

		// First request should be allowed
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.Equal(t, res.TAT, clock.Now().UnixMilli())
		require.WithinDuration(t, clock.Now().Add(1*6*time.Second), time.UnixMilli(res.NewTAT), time.Second)

		require.False(t, res.Limited)
		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, clock.Now().Add(6*time.Second).UnixMilli(), res.RetryAtMS)

		// Second request should be limited
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.Equal(t, (6 * time.Second).Milliseconds(), res.EmissionInterval)
		require.Equal(t, res.TAT, clock.Now().Add(6*time.Second).UnixMilli())
		require.WithinDuration(t, clock.Now().Add(2*6*time.Second), time.UnixMilli(res.NewTAT), time.Second)
		require.Equal(t, (6 * time.Second).Milliseconds(), res.DVT)
		require.Equal(t, (1 * 6 * time.Second).Milliseconds(), res.Increment)
		require.Equal(t, clock.Now().Add(2*6*time.Second).Add(-6*time.Second).UnixMilli(), res.AllowAt)
		require.Equal(t, -(6 * time.Second).Milliseconds(), res.Diff)

		require.True(t, res.Limited)
		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)
		// Accounts for burst
		require.Equal(t, 6*time.Second, time.Duration(res.ResetAfterMS)*time.Millisecond)
		require.Equal(t, 6*time.Second, time.Duration(res.RetryAfterMS)*time.Millisecond)
		require.Equal(t, clock.Now().Add(6*time.Second).UnixMilli(), res.RetryAtMS)
	})

	t.Run("1 request every 24h should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		r, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 24 * time.Hour
		limit := 1
		burst := 0

		// First request should work
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.False(t, res.Limited)
		require.Equal(t, res.TAT, clock.Now().UnixMilli())
		require.WithinDuration(t, clock.Now().Add(24*time.Hour), time.UnixMilli(res.NewTAT), time.Second)

		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, clock.Now().Add(24*time.Hour).UnixMilli(), res.RetryAtMS)

		// Second request should be limited
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.Equal(t, (24 * time.Hour).Milliseconds(), res.EmissionInterval)
		require.Equal(t, res.TAT, clock.Now().Add(24*time.Hour).UnixMilli())
		require.WithinDuration(t, clock.Now().Add(2*24*time.Hour), time.UnixMilli(res.NewTAT), time.Second)
		require.Equal(t, (24 * time.Hour * 1).Milliseconds(), res.DVT)
		require.Equal(t, (1 * 24 * time.Hour).Milliseconds(), res.Increment)
		require.Equal(t, clock.Now().Add(24*time.Hour).Add(1*24*time.Hour).Add(-24*time.Hour).UnixMilli(), res.AllowAt)
		require.Equal(t, -(24 * time.Hour).Milliseconds(), res.Diff)

		require.True(t, res.Limited)
		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, 24*time.Hour, time.Duration(res.ResetAfterMS)*time.Millisecond)
		require.Equal(t, 24*time.Hour, time.Duration(res.RetryAfterMS)*time.Millisecond)
		require.Equal(t, clock.Now().Add(24*time.Hour).UnixMilli(), res.RetryAtMS)

		// Waiting should reduce ttl but still reject

		clock.Advance(4 * time.Hour)
		r.FastForward(4 * time.Hour)
		r.SetTime(clock.Now())

		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.Equal(t, (24 * time.Hour).Milliseconds(), res.EmissionInterval)
		require.Equal(t, res.TAT, clock.Now().Add(20*time.Hour).UnixMilli())
		require.WithinDuration(t, clock.Now().Add(20*time.Hour+24*time.Hour), time.UnixMilli(res.NewTAT), time.Second)
		require.Equal(t, (24 * time.Hour * 1).Milliseconds(), res.DVT)
		require.Equal(t, (1 * 24 * time.Hour).Milliseconds(), res.Increment)
		require.Equal(t, clock.Now().Add(20*time.Hour).Add(1*24*time.Hour).Add(-24*time.Hour).UnixMilli(), res.AllowAt)
		require.Equal(t, -(20 * time.Hour).Milliseconds(), res.Diff)

		require.True(t, res.Limited)
		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, 20*time.Hour, time.Duration(res.ResetAfterMS)*time.Millisecond)
		require.Equal(t, 20*time.Hour, time.Duration(res.RetryAfterMS)*time.Millisecond)
		require.Equal(t, clock.Now().Add(20*time.Hour).UnixMilli(), res.RetryAtMS)
	})

	t.Run("3000 requests every minute should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := time.Minute
		limit := 3000
		burst := 0

		// First request should be allowed
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.Equal(t, res.TAT, clock.Now().UnixMilli())
		require.WithinDuration(t, clock.Now().Add(20*time.Millisecond), time.UnixMilli(res.NewTAT), time.Second)
		require.Equal(t, (20 * time.Millisecond).Milliseconds(), res.EmissionInterval)
		require.Equal(t, (20 * time.Millisecond).Milliseconds(), res.DVT)
		require.Equal(t, (20 * time.Millisecond).Milliseconds(), res.Increment)
		require.Equal(t, clock.Now().Add(20*time.Millisecond).UnixMilli(), res.RetryAtMS)
		// allow initial request
		require.Equal(t, clock.Now().Add(20*time.Millisecond).Add(-20*time.Millisecond).UnixMilli(), res.AllowAt)
		require.Equal(t, time.Duration(0).Milliseconds(), res.Diff)

		// Since we don't have a burst, only one request will be allowed every 20ms
		require.False(t, res.Limited)
		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)

		require.Equal(t, 20*time.Millisecond, time.Duration(res.ResetAfterMS)*time.Millisecond)
		require.Equal(t, time.Duration(0), time.Duration(res.RetryAfterMS)*time.Millisecond)
		require.Equal(t, clock.Now().Add(20*time.Millisecond).UnixMilli(), res.RetryAtMS)
	})

	t.Run("3000 requests every minute with burst should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		r, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := time.Minute
		limit := 3000
		burst := 1

		// First request should be allowed
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 2,
		})
		require.False(t, res.Limited)

		require.Equal(t, res.TAT, clock.Now().UnixMilli())
		require.WithinDuration(t, clock.Now().Add(2*20*time.Millisecond), time.UnixMilli(res.NewTAT), time.Second)
		require.Equal(t, (20 * time.Millisecond).Milliseconds(), res.EmissionInterval)
		require.Equal(t, (2 * 20 * time.Millisecond).Milliseconds(), res.DVT)
		require.Equal(t, (2 * 20 * time.Millisecond).Milliseconds(), res.Increment)
		// allow initial request
		require.Equal(t, clock.Now().Add(2*20*time.Millisecond).Add(-2*20*time.Millisecond).UnixMilli(), res.AllowAt)
		require.Equal(t, time.Duration(0).Milliseconds(), res.Diff)

		// Since we don't have a burst, only one request will be allowed every 20ms
		require.Equal(t, 2, res.Limit)
		require.Equal(t, 0, res.Remaining)

		// burst was applied
		require.Equal(t, 2*20*time.Millisecond, time.Duration(res.ResetAfterMS)*time.Millisecond)

		// request was allowed
		require.Equal(t, time.Duration(0), time.Duration(res.RetryAfterMS)*time.Millisecond)

		// second request should be blocked
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})
		require.True(t, res.Limited)
		require.Equal(t, 20*time.Millisecond, time.Duration(res.RetryAfterMS)*time.Millisecond)
		require.Equal(t, clock.Now().Add(2*20*time.Millisecond).UnixMilli(), res.TAT)

		// waiting for 20ms should unblock 1 request
		clock.Advance(20 * time.Millisecond)
		r.FastForward(20 * time.Millisecond)
		r.SetTime(clock.Now())

		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.False(t, res.Limited)
		require.Equal(t, 1, res.Remaining)
		require.Equal(t, 20*time.Millisecond, time.Duration(res.ResetAfterMS)*time.Millisecond)

		// waiting for 20ms should unblock the burst request
		clock.Advance(20 * time.Millisecond)
		r.FastForward(20 * time.Millisecond)
		r.SetTime(clock.Now())

		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.False(t, res.Limited)
		require.Equal(t, 2, res.Remaining)
		require.Equal(t, 0*time.Millisecond, time.Duration(res.ResetAfterMS)*time.Millisecond)
	})

	// NOTE: Key queues are not immediately supported by gcra. This is because we apply smoothing: We do not want
	// callers to be able to exhaust the complete capacity for a period within a single request.
	// This is why we break down the period into smaller chunks (the emission interval).
	//
	// For key queues, we should do the following: Instead of rewriting gcra to fit
	// the case where we need to consume multiple items at once while respecting the period limit,
	// we should make this a burst. This way, it naturally works. We just have to make sure burst = limit - 1
	// as we apply + 1 by default
	t.Run("capacity calculation should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := time.Minute
		limit := 20
		burst := limit - 1 // assume we can spend entire limit at once!

		// Read initial capacity
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.False(t, res.Limited)
		require.Equal(t, 20, res.Limit)
		require.Equal(t, 20, res.Remaining)
		require.Equal(t, clock.Now().Add(3*time.Second).UnixMilli(), res.RetryAtMS)

		// use half the capacity at once
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 10,
		})
		require.False(t, res.Limited)
		require.Equal(t, 20, res.Limit)
		require.Equal(t, 10, res.Remaining)
		require.Equal(t, clock.Now().Add(3*time.Second).UnixMilli(), res.RetryAtMS)

		// use remaining half
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 10,
		})
		require.False(t, res.Limited)
		require.Equal(t, 20, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, clock.Now().Add(3*time.Second).UnixMilli(), res.RetryAtMS)

		// no more capacity
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.Equal(t, 20, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, 3*time.Second, time.Duration(res.EmissionInterval)*time.Millisecond)
		require.WithinDuration(t, clock.Now().Add(time.Minute), time.UnixMilli(res.TAT), time.Second)
		require.WithinDuration(t, clock.Now().Add(time.Minute+3*time.Second), time.UnixMilli(res.NewTAT), time.Second)

		// it would take 3s until we can run another request
		require.Equal(t, -3*time.Second, time.Duration(res.Diff)*time.Millisecond)
		require.WithinDuration(t, clock.Now().Add(3*time.Second), time.UnixMilli(res.RetryAtMS), time.Second)

		// using for multiple items is impossible now
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 10,
		})
		require.True(t, res.Limited)
		require.Equal(t, 20, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, 3*time.Second, time.Duration(res.EmissionInterval)*time.Millisecond)
		require.WithinDuration(t, clock.Now().Add(time.Minute), time.UnixMilli(res.TAT), time.Second)
		require.WithinDuration(t, clock.Now().Add(time.Minute+10*3*time.Second), time.UnixMilli(res.NewTAT), time.Second)

		// it would take 30s until we could run all requests
		require.Equal(t, -30*time.Second, time.Duration(res.Diff)*time.Millisecond)
		require.WithinDuration(t, clock.Now().Add(30*time.Second), time.UnixMilli(res.RetryAtMS), time.Second)
	})

	t.Run("simulate gcraCapacity for key queues", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := time.Hour
		limit := 100

		burst := 10

		// simulate gcraUpdate beheavior
		maxBurst := limit + burst - 1

		// Read initial capacity
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    maxBurst,
			quantity: 0,
		})
		require.False(t, res.Limited)
		require.Equal(t, 110, res.Limit)
		require.Equal(t, 110, res.Remaining)
	})

	t.Run("simulate using up capacity and getting retryAt", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := time.Hour
		limit := 5
		burst := 0

		// simulate gcraUpdate beheavior
		maxBurst := limit + burst - 1

		// Read initial capacity
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    maxBurst,
			quantity: 0,
		})
		require.False(t, res.Limited)
		require.Equal(t, 5, res.Limit)
		require.Equal(t, int64(0), res.Usage)
		require.Equal(t, 5, res.Remaining)

		// Consume all
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    maxBurst,
			quantity: 5,
		})
		require.False(t, res.Limited)
		require.Equal(t, 5, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, int64(5), res.Usage)

		require.Equal(t, int64(0), res.RetryAfterMS)
		require.Equal(t, time.Hour.Milliseconds(), res.ResetAfterMS)
	})

	t.Run("retryAt should be properly calculated", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		r, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		// 10 every 60 minutes, 1 every 6s
		period := 1 * time.Minute
		limit := 10
		burst := 1

		// with full capacity, should show refill after 6s assuming that all capacity is consumed
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.WithinDuration(t, clock.Now().Add(6*time.Second), time.UnixMilli(res.RetryAtMS), time.Second)

		// First request should work
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 2,
		})

		require.Equal(t, (6 * time.Second).Milliseconds(), res.EmissionInterval)
		require.Equal(t, res.TAT, clock.Now().UnixMilli())
		require.WithinDuration(t, clock.Now().Add(2*6*time.Second), time.UnixMilli(res.NewTAT), time.Second)
		require.Equal(t, (12 * time.Second).Milliseconds(), res.DVT)
		require.Equal(t, (2 * 6 * time.Second).Milliseconds(), res.Increment)
		require.Equal(t, clock.Now().Add(2*6*time.Second).Add(-12*time.Second).UnixMilli(), res.AllowAt)
		require.Equal(t, (0 * time.Second).Milliseconds(), res.Diff)

		require.Equal(t, 2, res.Limit)
		require.Equal(t, 0, res.Remaining)
		// Accounts for burst
		require.Equal(t, 2*6*time.Second, time.Duration(res.ResetAfterMS)*time.Millisecond)
		require.Equal(t, time.Duration(0), time.Duration(res.RetryAfterMS)*time.Millisecond)
		require.Equal(t, clock.Now().Add(6*time.Second).UnixMilli(), res.RetryAtMS)

		// Advance time just a little so retryAt should go down

		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.Equal(t, clock.Now().Add(4*time.Second).UnixMilli(), res.AllowAt)
		require.Equal(t, -4*time.Second, time.Duration(res.Diff)*time.Millisecond)
		require.WithinDuration(t, clock.Now().Add(4*time.Second), time.UnixMilli(res.RetryAtMS), time.Second)

		// skip forward 4 seconds, so first request is "fully consumed"
		clock.Advance(4 * time.Second)
		r.FastForward(4 * time.Second)
		r.SetTime(clock.Now())

		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.Equal(t, 1, res.Remaining)
		require.WithinDuration(t, clock.Now().Add(6*time.Second), time.UnixMilli(res.TAT), time.Second)
		require.Equal(t, clock.Now().UnixMilli(), res.AllowAt)
		require.Equal(t, 0*time.Second, time.Duration(res.Diff)*time.Millisecond)
		require.WithinDuration(t, clock.Now().Add(6*time.Second), time.UnixMilli(res.RetryAtMS), time.Second)
	})

	t.Run("retry_after should be set properly", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 1 * time.Minute
		limit := 10
		burst := 0

		// Read initial capacity
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.False(t, res.Limited)

		// Can run the request right now!
		require.Equal(t, time.Duration(0), time.Duration(res.RetryAfterMS)*time.Millisecond)

		// NOTE: The read-only request is used to return the retry time
		// assuming the request went through and consumed all capacity.
		// If we just returned the current state BEFORE modifying, we would
		// have to return the current time here.
		require.WithinDuration(t, clock.Now().Add(6*time.Second), time.UnixMilli(res.RetryAtMS), time.Millisecond)

		// Can run 1 more
		require.Equal(t, int64(0), res.Usage)
		require.Equal(t, 1, res.Remaining)

		// Consume one
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})
		// Still not limited
		require.False(t, res.Limited)

		// No more capacity now
		require.Equal(t, int64(1), res.Usage)
		require.Equal(t, 0, res.Remaining)

		// Request was successful so retryAfter will be unset
		require.Equal(t, 0*time.Second, time.Duration(res.RetryAfterMS)*time.Millisecond)

		// RetryAtMS will be set to now + emission
		require.WithinDuration(t, clock.Now().Add(6*time.Second), time.UnixMilli(res.RetryAtMS), time.Millisecond)
	})

	// Regression test for: "attempt to compare nil with number" panic in acquire.lua / check.lua.
	//
	// The throttle function only sets result["remaining"] when next > -emission
	// (where next = dvt - ttl). When the stored TAT is far enough in the future
	// that ttl >= dvt+emission, the conditional is never entered and remaining
	// stays nil. The nil then reaches `if constraintCapacity <= 0` in the callers
	// and crashes Lua.
	//
	// With limit=10/min (emission=6s, dvt=6s, burst=0): the boundary is ttl >= 12s.
	// We inject a TAT 13s ahead to land just past it.
	t.Run("quantity=0 with TAT beyond dvt+emission returns 0 remaining, not nil", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		r, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 1 * time.Minute
		limit := 10
		burst := 0
		// emission=6s, dvt=6s, dvt+emission=12s — inject TAT 13s ahead to exceed boundary
		tat := clock.Now().Add(13 * time.Second).UnixMilli()
		require.NoError(t, r.Set(key, strconv.FormatInt(tat, 10)))

		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})

		// remaining must be 0 (not nil); nil caused a Lua panic before the fix
		require.Equal(t, 0, res.Remaining)
	})
}

func TestRateLimitGCRA(t *testing.T) {
	type gcraScriptOptions struct {
		key      string
		now      time.Time
		period   time.Duration
		limit    int
		burst    int
		quantity int
	}

	type rateLimitResult struct {
		Limited bool `json:"limited"`

		// Limit is the maximum number of requests that could be permitted
		// instantaneously for this key starting from an empty state. For
		// example, if a rate limiter allows 10 requests per second per
		// key, Limit would always be 10.
		Limit int `json:"limit"`

		Usage int64 `json:"u"`

		// Remaining is the maximum number of requests that could be
		// permitted instantaneously for this key given the current
		// state. For example, if a rate limiter allows 10 requests per
		// second and has already received 6 requests for this key this
		// second, Remaining would be 4.
		Remaining int `json:"remaining"`

		// ResetAfter is the time until the RateLimiter returns to its
		// initial state for a given key. For example, if a rate limiter
		// manages requests per second and received one request 200ms ago,
		// Reset would return 800ms. You can also think of this as the time
		// until Limit and Remaining will be equal.
		ResetAfterMS int64 `json:"reset_after"`

		// RetryAfter is the time until the next request will be permitted.
		// It should be -1 unless the rate limit has been exceeded.
		RetryAfterMS int64 `json:"retry_after"`

		// RetryAt is the time the next request is permitted
		// This assumes all capacity is consumed in the request.
		RetryAtMS int64 `json:"retry_at"`

		EmissionInterval int64 `json:"ei"`
		DVT              int64 `json:"dvt"`

		TAT    int64 `json:"tat"`
		NewTAT int64 `json:"ntat"`

		Increment int64 `json:"inc"`
		AllowAt   int64 `json:"aat"`

		Diff int64 `json:"diff"`

		Next int64 `json:"next"`
	}

	initRedis := func(t *testing.T) (*miniredis.Miniredis, rueidis.Client) {
		r := miniredis.RunT(t)
		rc, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{r.Addr()},
			DisableCache: true,
		})
		require.NoError(t, err)
		return r, rc
	}

	runScript := func(t *testing.T, rc rueidis.Client, opts gcraScriptOptions) rateLimitResult {
		nowNS := opts.now.UnixNano()
		args, err := strSlice([]any{
			opts.key,
			nowNS,
			opts.limit,
			opts.burst,
			opts.period.Nanoseconds(),
			opts.quantity,
		})
		require.NoError(t, err)

		rawRes, err := scripts["test/ratelimit"].Exec(t.Context(), rc, []string{}, args).ToString()
		require.NoError(t, err)

		var res rateLimitResult
		err = json.Unmarshal([]byte(rawRes), &res)
		require.NoError(t, err)

		return res
	}

	t.Run("should return gcra result struct", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 1 * time.Minute
		limit := 10
		burst := 0

		// Read initial capacity
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.False(t, res.Limited)

		require.Equal(t, (6 * time.Second).Nanoseconds(), res.EmissionInterval)
		require.WithinDuration(t, clock.Now(), time.Unix(0, res.TAT), time.Second)
		require.WithinDuration(t, clock.Now().Add(6*time.Second), time.Unix(0, res.NewTAT), time.Second)
		require.Equal(t, (6 * time.Second).Nanoseconds(), res.DVT)
		require.Equal(t, (6 * time.Second).Nanoseconds(), res.Increment)
		require.Equal(t, clock.Now().UnixNano(), res.AllowAt)
		require.Equal(t, int64(0), res.Diff)

		require.Equal(t, 1, res.Limit)
		require.Equal(t, 1, res.Remaining)
		require.Equal(t, time.Duration(0), time.Duration(res.ResetAfterMS)*time.Nanosecond)
		require.Equal(t, time.Duration(0), time.Duration(res.RetryAfterMS)*time.Nanosecond)
		require.Equal(t, clock.Now().Add(6*time.Second).UnixNano(), res.RetryAtMS)
	})

	t.Run("consume 1 should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 1 * time.Minute
		limit := 10
		burst := 0

		// First request should be admitted
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.False(t, res.Limited)
		require.Equal(t, (6 * time.Second).Nanoseconds(), res.EmissionInterval)
		require.WithinDuration(t, clock.Now(), time.Unix(0, res.TAT), time.Second)
		require.WithinDuration(t, clock.Now().Add(6*time.Second), time.Unix(0, res.NewTAT), time.Second)
		require.Equal(t, (6 * time.Second).Nanoseconds(), res.DVT)
		require.Equal(t, (6 * time.Second).Nanoseconds(), res.Increment)
		require.Equal(t, clock.Now().UnixNano(), res.AllowAt)
		require.Equal(t, int64(0), res.Diff)

		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, 6*time.Second, time.Duration(res.ResetAfterMS)*time.Nanosecond)
		require.Equal(t, time.Duration(0), time.Duration(res.RetryAfterMS)*time.Nanosecond)
		require.Equal(t, clock.Now().Add(6*time.Second).UnixNano(), res.RetryAtMS)
	})

	t.Run("consume 1 with burst 1 should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 1 * time.Minute
		limit := 10
		burst := 1

		// Read initial capacity
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 2,
		})

		require.False(t, res.Limited)
		require.Equal(t, (6 * time.Second).Nanoseconds(), res.EmissionInterval)
		require.WithinDuration(t, clock.Now(), time.Unix(0, res.TAT), time.Second)
		require.WithinDuration(t, clock.Now().Add(2*6*time.Second), time.Unix(0, res.NewTAT), time.Second)
		require.Equal(t, (12 * time.Second).Nanoseconds(), res.DVT)
		require.Equal(t, (2 * 6 * time.Second).Nanoseconds(), res.Increment)
		require.Equal(t, clock.Now().Add(2*6*time.Second).Add(-12*time.Second).UnixNano(), res.AllowAt)
		require.Equal(t, (0 * time.Second).Nanoseconds(), res.Diff)

		require.Equal(t, 2, res.Limit)
		require.Equal(t, 0, res.Remaining)
		// Accounts for burst
		require.Equal(t, 2*6*time.Second, time.Duration(res.ResetAfterMS)*time.Nanosecond)
		require.Equal(t, time.Duration(0), time.Duration(res.RetryAfterMS)*time.Nanosecond)
		require.Equal(t, clock.Now().Add(6*time.Second).UnixNano(), res.RetryAtMS)
	})

	t.Run("being limited should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 1 * time.Minute
		limit := 10
		burst := 0

		// First request should be allowed
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.WithinDuration(t, clock.Now(), time.Unix(0, res.TAT), time.Second)
		require.WithinDuration(t, clock.Now().Add(1*6*time.Second), time.Unix(0, res.NewTAT), time.Second)

		require.False(t, res.Limited)
		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, clock.Now().Add(6*time.Second).UnixNano(), res.RetryAtMS)

		// Second request should be limited
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.Equal(t, (6 * time.Second).Nanoseconds(), res.EmissionInterval)
		require.Equal(t, res.TAT, clock.Now().Add(6*time.Second).UnixNano())
		require.WithinDuration(t, clock.Now().Add(2*6*time.Second), time.Unix(0, res.NewTAT), time.Second)
		require.Equal(t, (6 * time.Second).Nanoseconds(), res.DVT)
		require.Equal(t, (1 * 6 * time.Second).Nanoseconds(), res.Increment)
		require.Equal(t, clock.Now().Add(2*6*time.Second).Add(-6*time.Second).UnixNano(), res.AllowAt)
		require.Equal(t, -(6 * time.Second).Nanoseconds(), res.Diff)

		require.True(t, res.Limited)
		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)
		// Accounts for burst
		require.Equal(t, 6*time.Second, time.Duration(res.ResetAfterMS)*time.Nanosecond)
		require.Equal(t, 6*time.Second, time.Duration(res.RetryAfterMS)*time.Nanosecond)
		require.Equal(t, clock.Now().Add(6*time.Second).UnixNano(), res.RetryAtMS)
	})

	t.Run("1 request every 24h should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		r, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 24 * time.Hour
		limit := 1
		burst := 0

		// First request should work
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.False(t, res.Limited)
		require.WithinDuration(t, clock.Now(), time.Unix(0, res.TAT), time.Second)
		require.WithinDuration(t, clock.Now().Add(24*time.Hour), time.Unix(0, res.NewTAT), time.Second)

		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, clock.Now().Add(24*time.Hour).UnixNano(), res.RetryAtMS)

		// Second request should be limited
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.Equal(t, (24 * time.Hour).Nanoseconds(), res.EmissionInterval)
		require.Equal(t, res.TAT, clock.Now().Add(24*time.Hour).UnixNano())
		require.WithinDuration(t, clock.Now().Add(2*24*time.Hour), time.Unix(0, res.NewTAT), time.Second)
		require.Equal(t, (24 * time.Hour * 1).Nanoseconds(), res.DVT)
		require.Equal(t, (1 * 24 * time.Hour).Nanoseconds(), res.Increment)
		require.Equal(t, clock.Now().Add(24*time.Hour).Add(1*24*time.Hour).Add(-24*time.Hour).UnixNano(), res.AllowAt)
		require.Equal(t, -(24 * time.Hour).Nanoseconds(), res.Diff)

		require.True(t, res.Limited)
		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, 24*time.Hour, time.Duration(res.ResetAfterMS)*time.Nanosecond)
		require.Equal(t, 24*time.Hour, time.Duration(res.RetryAfterMS)*time.Nanosecond)
		require.Equal(t, clock.Now().Add(24*time.Hour).UnixNano(), res.RetryAtMS)

		// Waiting should reduce ttl but still reject

		clock.Advance(4 * time.Hour)
		r.FastForward(4 * time.Hour)
		r.SetTime(clock.Now())

		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.Equal(t, (24 * time.Hour).Nanoseconds(), res.EmissionInterval)
		require.Equal(t, res.TAT, clock.Now().Add(20*time.Hour).UnixNano())
		require.WithinDuration(t, clock.Now().Add(20*time.Hour+24*time.Hour), time.Unix(0, res.NewTAT), time.Second)
		require.Equal(t, (24 * time.Hour * 1).Nanoseconds(), res.DVT)
		require.Equal(t, (1 * 24 * time.Hour).Nanoseconds(), res.Increment)
		require.Equal(t, clock.Now().Add(20*time.Hour).Add(1*24*time.Hour).Add(-24*time.Hour).UnixNano(), res.AllowAt)
		require.Equal(t, -(20 * time.Hour).Nanoseconds(), res.Diff)

		require.True(t, res.Limited)
		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, 20*time.Hour, time.Duration(res.ResetAfterMS)*time.Nanosecond)
		require.Equal(t, 20*time.Hour, time.Duration(res.RetryAfterMS)*time.Nanosecond)
		require.Equal(t, clock.Now().Add(20*time.Hour).UnixNano(), res.RetryAtMS)
	})

	t.Run("3000 requests every minute should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := time.Minute
		limit := 3000
		burst := 0

		// First request should be allowed
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})

		require.WithinDuration(t, clock.Now(), time.Unix(0, res.TAT), time.Second)
		require.WithinDuration(t, clock.Now().Add(20*time.Millisecond), time.Unix(0, res.NewTAT), time.Second)
		require.Equal(t, (20 * time.Millisecond).Nanoseconds(), res.EmissionInterval)
		require.Equal(t, (20 * time.Millisecond).Nanoseconds(), res.DVT)
		require.Equal(t, (20 * time.Millisecond).Nanoseconds(), res.Increment)
		require.Equal(t, clock.Now().Add(20*time.Millisecond).UnixNano(), res.RetryAtMS)
		// allow initial request
		require.Equal(t, clock.Now().Add(20*time.Millisecond).Add(-20*time.Millisecond).UnixNano(), res.AllowAt)
		require.Equal(t, time.Duration(0).Nanoseconds(), res.Diff)

		// Since we don't have a burst, only one request will be allowed every 20ms
		require.False(t, res.Limited)
		require.Equal(t, 1, res.Limit)
		require.Equal(t, 0, res.Remaining)

		require.Equal(t, 20*time.Millisecond, time.Duration(res.ResetAfterMS)*time.Nanosecond)
		require.Equal(t, time.Duration(0), time.Duration(res.RetryAfterMS)*time.Nanosecond)
		require.Equal(t, clock.Now().Add(20*time.Millisecond).UnixNano(), res.RetryAtMS)
	})

	t.Run("3000 requests every minute with burst should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		r, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := time.Minute
		limit := 3000
		burst := 1

		// First request should be allowed
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 2,
		})
		require.False(t, res.Limited)

		require.WithinDuration(t, clock.Now(), time.Unix(0, res.TAT), time.Second)
		require.WithinDuration(t, clock.Now().Add(2*20*time.Millisecond), time.Unix(0, res.NewTAT), time.Second)
		require.Equal(t, (20 * time.Millisecond).Nanoseconds(), res.EmissionInterval)
		require.Equal(t, (2 * 20 * time.Millisecond).Nanoseconds(), res.DVT)
		require.Equal(t, (2 * 20 * time.Millisecond).Nanoseconds(), res.Increment)
		// allow initial request
		require.Equal(t, clock.Now().Add(2*20*time.Millisecond).Add(-2*20*time.Millisecond).UnixNano(), res.AllowAt)
		require.Equal(t, time.Duration(0).Nanoseconds(), res.Diff)

		// Since we don't have a burst, only one request will be allowed every 20ms
		require.Equal(t, 2, res.Limit)
		require.Equal(t, 0, res.Remaining)

		// burst was applied
		require.Equal(t, 2*20*time.Millisecond, time.Duration(res.ResetAfterMS)*time.Nanosecond)

		// request was allowed
		require.Equal(t, time.Duration(0), time.Duration(res.RetryAfterMS)*time.Nanosecond)

		// second request should be blocked
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 1,
		})
		require.True(t, res.Limited)
		require.Equal(t, 20*time.Millisecond, time.Duration(res.RetryAfterMS)*time.Nanosecond)
		require.Equal(t, clock.Now().Add(2*20*time.Millisecond).UnixNano(), res.TAT)

		// waiting for 20ms should unblock 1 request
		clock.Advance(20 * time.Millisecond)
		r.FastForward(20 * time.Millisecond)
		r.SetTime(clock.Now())

		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.False(t, res.Limited)
		require.Equal(t, 1, res.Remaining)
		require.Equal(t, 20*time.Millisecond, time.Duration(res.ResetAfterMS)*time.Nanosecond)

		// waiting for 20ms should unblock the burst request
		clock.Advance(20 * time.Millisecond)
		r.FastForward(20 * time.Millisecond)
		r.SetTime(clock.Now())

		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.False(t, res.Limited)
		require.Equal(t, 2, res.Remaining)
		require.Equal(t, 0*time.Millisecond, time.Duration(res.ResetAfterMS)*time.Millisecond)
	})

	// NOTE: Key queues are not immediately supported by gcra. This is because we apply smoothing: We do not want
	// callers to be able to exhaust the complete capacity for a period within a single request.
	// This is why we break down the period into smaller chunks (the emission interval).
	//
	// For key queues, we should do the following: Instead of rewriting gcra to fit
	// the case where we need to consume multiple items at once while respecting the period limit,
	// we should make this a burst. This way, it naturally works. We just have to make sure burst = limit - 1
	// as we apply + 1 by default
	t.Run("capacity calculation should work", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := time.Minute
		limit := 20
		burst := limit - 1 // assume we can spend entire limit at once!

		// Read initial capacity
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.False(t, res.Limited)
		require.Equal(t, 20, res.Limit)
		require.Equal(t, 20, res.Remaining)
		require.WithinDuration(t, clock.Now().Add(3*time.Second), time.Unix(0, res.RetryAtMS), time.Second)

		// use half the capacity at once
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 10,
		})
		require.False(t, res.Limited)
		require.Equal(t, 20, res.Limit)
		require.Equal(t, 10, res.Remaining)
		require.Equal(t, clock.Now().Add(3*time.Second).UnixNano(), res.RetryAtMS)

		// use remaining half
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 10,
		})
		require.False(t, res.Limited)
		require.Equal(t, 20, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, clock.Now().Add(3*time.Second).UnixNano(), res.RetryAtMS)

		// no more capacity
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.Equal(t, 20, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, 3*time.Second, time.Duration(res.EmissionInterval)*time.Nanosecond)
		require.WithinDuration(t, clock.Now().Add(time.Minute), time.Unix(0, res.TAT), time.Second)
		require.WithinDuration(t, clock.Now().Add(time.Minute+3*time.Second), time.Unix(0, res.NewTAT), time.Second)

		// it would take 3s until we can run another request
		require.Equal(t, -3*time.Second, time.Duration(res.Diff)*time.Nanosecond)
		require.WithinDuration(t, clock.Now().Add(3*time.Second), time.Unix(0, res.RetryAtMS), time.Second)

		// using for multiple items is impossible now
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 10,
		})
		require.True(t, res.Limited)
		require.Equal(t, 20, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, 3*time.Second, time.Duration(res.EmissionInterval)*time.Nanosecond)
		require.WithinDuration(t, clock.Now().Add(time.Minute), time.Unix(0, res.TAT), time.Second)
		require.WithinDuration(t, clock.Now().Add(time.Minute+10*3*time.Second), time.Unix(0, res.NewTAT), time.Second)

		// it would take 30s until we could run all requests
		require.Equal(t, -30*time.Second, time.Duration(res.Diff)*time.Nanosecond)
		require.WithinDuration(t, clock.Now().Add(30*time.Second), time.Unix(0, res.RetryAtMS), time.Second)
	})

	t.Run("simulate gcraCapacity for key queues", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := time.Hour
		limit := 100

		burst := 10

		// simulate gcraUpdate beheavior
		maxBurst := limit + burst - 1

		// Read initial capacity
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    maxBurst,
			quantity: 0,
		})
		require.False(t, res.Limited)
		require.Equal(t, 110, res.Limit)
		require.Equal(t, 110, res.Remaining)
	})

	t.Run("simulate using up capacity and getting retryAt", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		_, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := time.Hour
		limit := 5
		burst := 0

		// simulate gcraUpdate beheavior
		maxBurst := limit + burst - 1

		// Read initial capacity
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    maxBurst,
			quantity: 0,
		})
		require.False(t, res.Limited)
		require.Equal(t, 5, res.Limit)
		require.Equal(t, int64(0), res.Usage)
		require.Equal(t, 5, res.Remaining)

		// Consume all
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    maxBurst,
			quantity: 5,
		})
		require.False(t, res.Limited)
		require.Equal(t, 5, res.Limit)
		require.Equal(t, 0, res.Remaining)
		require.Equal(t, int64(5), res.Usage)

		require.Equal(t, int64(0), res.RetryAfterMS)
		require.Equal(t, time.Hour.Nanoseconds(), res.ResetAfterMS)
	})

	t.Run("retryAt should be properly calculated", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		r, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		// 10 every 60 minutes, 1 every 6s
		period := 1 * time.Minute
		limit := 10
		burst := 1

		// with full capacity, should show refill after 6s assuming that all capacity is consumed
		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.WithinDuration(t, clock.Now().Add(6*time.Second), time.Unix(0, res.RetryAtMS), time.Second)

		// First request should work
		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 2,
		})

		require.Equal(t, (6 * time.Second).Nanoseconds(), res.EmissionInterval)
		require.Equal(t, res.TAT, clock.Now().UnixNano())
		require.WithinDuration(t, clock.Now().Add(2*6*time.Second), time.Unix(0, res.NewTAT), time.Second)
		require.Equal(t, (12 * time.Second).Nanoseconds(), res.DVT)
		require.Equal(t, (2 * 6 * time.Second).Nanoseconds(), res.Increment)
		require.Equal(t, clock.Now().Add(2*6*time.Second).Add(-12*time.Second).UnixNano(), res.AllowAt)
		require.Equal(t, (0 * time.Second).Nanoseconds(), res.Diff)

		require.Equal(t, 2, res.Limit)
		require.Equal(t, 0, res.Remaining)
		// Accounts for burst
		require.Equal(t, 2*6*time.Second, time.Duration(res.ResetAfterMS)*time.Nanosecond)
		require.Equal(t, time.Duration(0), time.Duration(res.RetryAfterMS)*time.Nanosecond)
		require.Equal(t, clock.Now().Add(6*time.Second).UnixNano(), res.RetryAtMS)

		// Advance time just a little so retryAt should go down

		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.Equal(t, clock.Now().Add(4*time.Second).UnixNano(), res.AllowAt)
		require.Equal(t, -4*time.Second, time.Duration(res.Diff)*time.Nanosecond)
		require.WithinDuration(t, clock.Now().Add(4*time.Second), time.Unix(0, res.RetryAtMS), time.Second)

		// skip forward 4 seconds, so first request is "fully consumed"
		clock.Advance(4 * time.Second)
		r.FastForward(4 * time.Second)
		r.SetTime(clock.Now())

		res = runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})
		require.Equal(t, 1, res.Remaining)
		require.WithinDuration(t, clock.Now().Add(6*time.Second), time.Unix(0, res.TAT), time.Second)
		require.Equal(t, clock.Now().UnixNano(), res.AllowAt)
		require.Equal(t, 0*time.Second, time.Duration(res.Diff)*time.Nanosecond)
		require.WithinDuration(t, clock.Now().Add(6*time.Second), time.Unix(0, res.RetryAtMS), time.Second)
	})

	// Regression test for: "attempt to compare nil with number" panic in acquire.lua / check.lua.
	//
	// The rateLimit function only sets result["remaining"] when next > -emission
	// (where next = dvt - ttl). When the stored TAT is far enough in the future
	// that ttl >= dvt+emission, the conditional is never entered and remaining
	// stays nil. The nil then reaches `if constraintCapacity <= 0` in the callers
	// and crashes Lua.
	//
	// With limit=10/min (emission=6s, dvt=6s, burst=0): the boundary is ttl >= 12s.
	// We inject a TAT 13s ahead to land just past it.
	t.Run("quantity=0 with TAT beyond dvt+emission returns 0 remaining, not nil", func(t *testing.T) {
		t.Parallel()

		clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

		r, rc := initRedis(t)
		defer rc.Close()

		key := "test"

		period := 1 * time.Minute
		limit := 10
		burst := 0
		// emission=6s, dvt=6s, dvt+emission=12s — inject TAT 13s ahead to exceed boundary
		tat := clock.Now().Add(13 * time.Second).UnixNano()
		require.NoError(t, r.Set(key, strconv.FormatInt(tat, 10)))

		res := runScript(t, rc, gcraScriptOptions{
			key:      key,
			now:      clock.Now(),
			period:   period,
			limit:    limit,
			burst:    burst,
			quantity: 0,
		})

		// remaining must be 0 (not nil); nil caused a Lua panic before the fix
		require.Equal(t, 0, res.Remaining)
	})
}
