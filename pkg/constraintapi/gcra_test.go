package constraintapi

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/inngest/inngest/pkg/constraintapi/gcra"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// the GCRA tables below run against two implementations of the same helper:
// the Lua in lua/helper/gcra.lua executed in miniredis, and the Go port in
// pkg/constraintapi/gcra.  a case that passes on both pins the Lua behaviour
// and proves the port matches it.

type gcraKind int

const (
	gcraThrottle gcraKind = iota
	gcraRateLimit
)

type gcraScriptOptions struct {
	key      string
	now      time.Time
	period   time.Duration
	limit    int
	burst    int
	quantity int
}

// rateLimitResult is the helper's result table.  every number is read as a
// whole unit of the call, milliseconds for throttle and nanoseconds for rate
// limit, by truncating the double the helper computes.
type rateLimitResult struct {
	Limited bool

	// Limit is the maximum number of requests that could be permitted
	// instantaneously for this key starting from an empty state.
	Limit int

	Usage int64

	// Remaining is the maximum number of requests that could be permitted
	// instantaneously for this key given the current state.
	Remaining int

	// ResetAfter is the time until the limiter returns to its initial state
	// for a given key.
	ResetAfterMS int64

	// RetryAfter is the time until the next request will be permitted.
	RetryAfterMS int64

	// RetryAt is the time the next request is permitted, assuming all
	// capacity is consumed in the request.
	RetryAtMS int64

	EmissionInterval int64
	DVT              int64

	TAT    int64
	NewTAT int64

	Increment int64
	AllowAt   int64

	Diff int64

	Next int64
}

// gcraRunner drives one GCRA implementation with one key store.
type gcraRunner interface {
	run(t *testing.T, opts gcraScriptOptions) rateLimitResult
	// seed stores tat for key with no expiry, the way a test writes the
	// Redis key directly.
	seed(t *testing.T, key string, tat int64)
	// advance moves the store's clock to now so stored TTLs expire.
	advance(now time.Time, d time.Duration)
}

// forEachGCRARunner runs body once per implementation as a parallel subtest.
func forEachGCRARunner(t *testing.T, kind gcraKind, body func(t *testing.T, gr gcraRunner)) {
	t.Run("lua", func(t *testing.T) {
		t.Parallel()
		body(t, newLuaGCRARunner(t, kind))
	})
	t.Run("go", func(t *testing.T) {
		t.Parallel()
		body(t, &goGCRARunner{kind: kind, states: map[string]goGCRAState{}})
	})
}

func fromGCRAResult(limited bool, limit, remaining int, usage int64, f ...float64) rateLimitResult {
	tr := func(v float64) int64 { return int64(v) }
	return rateLimitResult{
		Limited:          limited,
		Limit:            limit,
		Usage:            usage,
		Remaining:        remaining,
		ResetAfterMS:     tr(f[0]),
		RetryAfterMS:     tr(f[1]),
		RetryAtMS:        tr(f[2]),
		EmissionInterval: tr(f[3]),
		DVT:              tr(f[4]),
		TAT:              tr(f[5]),
		NewTAT:           tr(f[6]),
		Increment:        tr(f[7]),
		AllowAt:          tr(f[8]),
		Diff:             tr(f[9]),
		Next:             tr(f[10]),
	}
}

type luaGCRARunner struct {
	kind gcraKind
	r    *miniredis.Miniredis
	rc   rueidis.Client
}

func newLuaGCRARunner(t *testing.T, kind gcraKind) *luaGCRARunner {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	return &luaGCRARunner{kind: kind, r: r, rc: rc}
}

func (l *luaGCRARunner) run(t *testing.T, opts gcraScriptOptions) rateLimitResult {
	var now, period int64
	var script string
	switch l.kind {
	case gcraThrottle:
		now, period, script = opts.now.UnixMilli(), opts.period.Milliseconds(), "test/throttle"
	case gcraRateLimit:
		now, period, script = opts.now.UnixNano(), opts.period.Nanoseconds(), "test/ratelimit"
	}
	args, err := strSlice([]any{opts.key, now, opts.limit, opts.burst, period, opts.quantity})
	require.NoError(t, err)

	rawRes, err := scripts[script].Exec(t.Context(), l.rc, []string{}, args).ToString()
	require.NoError(t, err)

	var raw struct {
		Limited    bool    `json:"limited"`
		Limit      float64 `json:"limit"`
		Usage      float64 `json:"u"`
		Remaining  float64 `json:"remaining"`
		ResetAfter float64 `json:"reset_after"`
		RetryAfter float64 `json:"retry_after"`
		RetryAt    float64 `json:"retry_at"`
		EI         float64 `json:"ei"`
		DVT        float64 `json:"dvt"`
		TAT        float64 `json:"tat"`
		NewTAT     float64 `json:"ntat"`
		Increment  float64 `json:"inc"`
		AllowAt    float64 `json:"aat"`
		Diff       float64 `json:"diff"`
		Next       float64 `json:"next"`
	}
	require.NoError(t, json.Unmarshal([]byte(rawRes), &raw))
	return fromGCRAResult(raw.Limited, int(raw.Limit), int(raw.Remaining), int64(raw.Usage),
		raw.ResetAfter, raw.RetryAfter, raw.RetryAt, raw.EI, raw.DVT, raw.TAT, raw.NewTAT, raw.Increment, raw.AllowAt, raw.Diff, raw.Next)
}

func (l *luaGCRARunner) seed(t *testing.T, key string, tat int64) {
	require.NoError(t, l.r.Set(key, strconv.FormatInt(tat, 10)))
}

func (l *luaGCRARunner) advance(now time.Time, d time.Duration) {
	l.r.FastForward(d)
	l.r.SetTime(now)
}

type goGCRAState struct {
	tat       float64
	expiresAt float64
	noExpiry  bool
}

// goGCRARunner keeps one state per key and expires it by the time each call
// carries, which is what a Redis TTL does once the clock is fast forwarded.
type goGCRARunner struct {
	kind   gcraKind
	states map[string]goGCRAState
}

func (g *goGCRARunner) run(t *testing.T, opts gcraScriptOptions) rateLimitResult {
	var now, period, perSecond float64
	switch g.kind {
	case gcraThrottle:
		now, period, perSecond = float64(opts.now.UnixMilli()), float64(opts.period.Milliseconds()), 1_000
	case gcraRateLimit:
		now, period, perSecond = float64(opts.now.UnixNano()), float64(opts.period.Nanoseconds()), 1_000_000_000
	}
	st, ok := g.states[opts.key]
	present := ok && (st.noExpiry || now < st.expiresAt)

	var res gcra.Result
	var newTAT float64
	var ttl int64
	var store bool
	switch g.kind {
	case gcraThrottle:
		res, newTAT, ttl, store = gcra.Throttle(st.tat, present, now, period, opts.limit, opts.burst, opts.quantity)
	case gcraRateLimit:
		res, newTAT, ttl, store = gcra.RateLimit(st.tat, present, now, period, opts.limit, opts.burst, opts.quantity)
	}
	if store {
		g.states[opts.key] = goGCRAState{tat: newTAT, expiresAt: now + float64(ttl)*perSecond}
	}
	return fromGCRAResult(res.Limited, res.Limit, res.Remaining, res.Usage,
		res.ResetAfter, res.RetryAfter, res.RetryAt, res.EmissionInterval, res.DVT, res.TAT, res.NewTAT, res.Increment, res.AllowAt, res.Diff, res.Next)
}

func (g *goGCRARunner) seed(t *testing.T, key string, tat int64) {
	g.states[key] = goGCRAState{tat: float64(tat), noExpiry: true}
}

func (g *goGCRARunner) advance(time.Time, time.Duration) {}

func TestThrottleGCRA(t *testing.T) {
	t.Run("should return gcra result struct", func(t *testing.T) {
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := 1 * time.Minute
			limit := 10
			burst := 0

			// Read initial capacity
			res := gr.run(t, gcraScriptOptions{
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
	})

	t.Run("consume 1 should work", func(t *testing.T) {
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := 1 * time.Minute
			limit := 10
			burst := 0

			// First request should be admitted
			res := gr.run(t, gcraScriptOptions{
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
	})

	t.Run("consume 1 with burst 1 should work", func(t *testing.T) {
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := 1 * time.Minute
			limit := 10
			burst := 1

			// Read initial capacity
			res := gr.run(t, gcraScriptOptions{
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
	})

	t.Run("being limited should work", func(t *testing.T) {
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := 1 * time.Minute
			limit := 10
			burst := 0

			// First request should be allowed
			res := gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
	})

	t.Run("1 request every 24h should work", func(t *testing.T) {
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := 24 * time.Hour
			limit := 1
			burst := 0

			// First request should work
			res := gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
			gr.advance(clock.Now(), 4*time.Hour)

			res = gr.run(t, gcraScriptOptions{
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
	})

	t.Run("3000 requests every minute should work", func(t *testing.T) {
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := time.Minute
			limit := 3000
			burst := 0

			// First request should be allowed
			res := gr.run(t, gcraScriptOptions{
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
	})

	t.Run("3000 requests every minute with burst should work", func(t *testing.T) {
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := time.Minute
			limit := 3000
			burst := 1

			// First request should be allowed
			res := gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
			gr.advance(clock.Now(), 20*time.Millisecond)

			res = gr.run(t, gcraScriptOptions{
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
			gr.advance(clock.Now(), 20*time.Millisecond)

			res = gr.run(t, gcraScriptOptions{
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
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := time.Minute
			limit := 20
			burst := limit - 1 // assume we can spend entire limit at once!

			// Read initial capacity
			res := gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
	})

	t.Run("simulate gcraCapacity for key queues", func(t *testing.T) {
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := time.Hour
			limit := 100

			burst := 10

			// simulate gcraUpdate beheavior
			maxBurst := limit + burst - 1

			// Read initial capacity
			res := gr.run(t, gcraScriptOptions{
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
	})

	t.Run("simulate using up capacity and getting retryAt", func(t *testing.T) {
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := time.Hour
			limit := 5
			burst := 0

			// simulate gcraUpdate beheavior
			maxBurst := limit + burst - 1

			// Read initial capacity
			res := gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
	})

	t.Run("retryAt should be properly calculated", func(t *testing.T) {
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			// 10 every 60 minutes, 1 every 6s
			period := 1 * time.Minute
			limit := 10
			burst := 1

			// with full capacity, should show refill after 6s assuming that all capacity is consumed
			res := gr.run(t, gcraScriptOptions{
				key:      key,
				now:      clock.Now(),
				period:   period,
				limit:    limit,
				burst:    burst,
				quantity: 0,
			})
			require.WithinDuration(t, clock.Now().Add(6*time.Second), time.UnixMilli(res.RetryAtMS), time.Second)

			// First request should work
			res = gr.run(t, gcraScriptOptions{
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
			gr.advance(clock.Now(), 2*time.Second)

			res = gr.run(t, gcraScriptOptions{
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
			gr.advance(clock.Now(), 4*time.Second)

			res = gr.run(t, gcraScriptOptions{
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
	})

	t.Run("retry_after should be set properly", func(t *testing.T) {
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := 1 * time.Minute
			limit := 10
			burst := 0

			// Read initial capacity
			res := gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := 1 * time.Minute
			limit := 10
			burst := 0
			// emission=6s, dvt=6s, dvt+emission=12s — inject TAT 13s ahead to exceed boundary
			tat := clock.Now().Add(13 * time.Second).UnixMilli()
			gr.seed(t, key, tat)

			res := gr.run(t, gcraScriptOptions{
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
	})

	t.Run("limit that does not divide the period keeps its fraction", func(t *testing.T) {
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {
			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))
			key := "test"
			period := 1 * time.Minute
			limit := 7
			burst := 0

			// emission is 8571.428ms.  the runner truncates every value to
			// whole milliseconds; the fraction itself is checked in the gcra
			// package tests.
			res := gr.run(t, gcraScriptOptions{key: key, now: clock.Now(), period: period, limit: limit, burst: burst, quantity: 1})
			require.False(t, res.Limited)
			require.Equal(t, int64(8571), res.EmissionInterval)
			require.Equal(t, clock.Now().UnixMilli()+8571, res.NewTAT)
			require.Equal(t, 0, res.Remaining)

			res = gr.run(t, gcraScriptOptions{key: key, now: clock.Now(), period: period, limit: limit, burst: burst, quantity: 0})
			require.Equal(t, clock.Now().UnixMilli()+8571, res.TAT, "the stored TAT keeps the fraction")
			require.Equal(t, int64(8571), res.ResetAfterMS)
			require.Equal(t, 0, res.Remaining)
			require.Equal(t, clock.Now().UnixMilli()+8571, res.RetryAtMS)

			// the stored TTL truncates 8.571s to 8s, so at 8.5s the key is gone
			// while the TAT is still ahead, and a new request starts from now
			clock.Advance(8500 * time.Millisecond)
			gr.advance(clock.Now(), 8500*time.Millisecond)
			res = gr.run(t, gcraScriptOptions{key: key, now: clock.Now(), period: period, limit: limit, burst: burst, quantity: 0})
			require.Equal(t, clock.Now().UnixMilli(), res.TAT, "the key expired before the TAT was reached")
			require.Equal(t, 1, res.Remaining)
		})
	})

	t.Run("a TAT in the past inflates remaining and the limited path writes nothing", func(t *testing.T) {
		forEachGCRARunner(t, gcraThrottle, func(t *testing.T, gr gcraRunner) {
			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))
			key := "test"
			period := 1 * time.Minute
			limit := 10
			burst := 0

			// emission 6s, dvt 6s.  a TAT 10s in the past gives ttl -10s, so
			// next = dvt - ttl is 16s and remaining is 2, above burst + 1.
			tat := clock.Now().Add(-10 * time.Second).UnixMilli()
			gr.seed(t, key, tat)

			res := gr.run(t, gcraScriptOptions{key: key, now: clock.Now(), period: period, limit: limit, burst: burst, quantity: 0})
			require.False(t, res.Limited)
			require.Equal(t, 2, res.Remaining)
			require.Equal(t, int64(-1), res.Usage)
			require.Equal(t, -(10 * time.Second).Milliseconds(), res.ResetAfterMS)

			// acquire pass two asks for the inflated remaining, is refused
			// with the default retry_at because the increment exceeds dvt,
			// and writes nothing
			res = gr.run(t, gcraScriptOptions{key: key, now: clock.Now(), period: period, limit: limit, burst: burst, quantity: 2})
			require.True(t, res.Limited)
			require.Equal(t, 0, res.Remaining)
			require.Equal(t, clock.Now().Add(6*time.Second).UnixMilli(), res.RetryAtMS)

			res = gr.run(t, gcraScriptOptions{key: key, now: clock.Now(), period: period, limit: limit, burst: burst, quantity: 0})
			require.Equal(t, tat, res.TAT, "the limited path did not write")
			require.Equal(t, 2, res.Remaining)
		})
	})
}

func TestRateLimitGCRA(t *testing.T) {
	t.Run("should return gcra result struct", func(t *testing.T) {
		forEachGCRARunner(t, gcraRateLimit, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := 1 * time.Minute
			limit := 10
			burst := 0

			// Read initial capacity
			res := gr.run(t, gcraScriptOptions{
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
	})

	t.Run("consume 1 should work", func(t *testing.T) {
		forEachGCRARunner(t, gcraRateLimit, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := 1 * time.Minute
			limit := 10
			burst := 0

			// First request should be admitted
			res := gr.run(t, gcraScriptOptions{
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
	})

	t.Run("consume 1 with burst 1 should work", func(t *testing.T) {
		forEachGCRARunner(t, gcraRateLimit, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := 1 * time.Minute
			limit := 10
			burst := 1

			// Read initial capacity
			res := gr.run(t, gcraScriptOptions{
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
	})

	t.Run("being limited should work", func(t *testing.T) {
		forEachGCRARunner(t, gcraRateLimit, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := 1 * time.Minute
			limit := 10
			burst := 0

			// First request should be allowed
			res := gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
	})

	t.Run("1 request every 24h should work", func(t *testing.T) {
		forEachGCRARunner(t, gcraRateLimit, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := 24 * time.Hour
			limit := 1
			burst := 0

			// First request should work
			res := gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
			gr.advance(clock.Now(), 4*time.Hour)

			res = gr.run(t, gcraScriptOptions{
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
	})

	t.Run("3000 requests every minute should work", func(t *testing.T) {
		forEachGCRARunner(t, gcraRateLimit, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := time.Minute
			limit := 3000
			burst := 0

			// First request should be allowed
			res := gr.run(t, gcraScriptOptions{
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
	})

	t.Run("3000 requests every minute with burst should work", func(t *testing.T) {
		forEachGCRARunner(t, gcraRateLimit, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := time.Minute
			limit := 3000
			burst := 1

			// First request should be allowed
			res := gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
			gr.advance(clock.Now(), 20*time.Millisecond)

			res = gr.run(t, gcraScriptOptions{
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
			gr.advance(clock.Now(), 20*time.Millisecond)

			res = gr.run(t, gcraScriptOptions{
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
		forEachGCRARunner(t, gcraRateLimit, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := time.Minute
			limit := 20
			burst := limit - 1 // assume we can spend entire limit at once!

			// Read initial capacity
			res := gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
	})

	t.Run("simulate gcraCapacity for key queues", func(t *testing.T) {
		forEachGCRARunner(t, gcraRateLimit, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := time.Hour
			limit := 100

			burst := 10

			// simulate gcraUpdate beheavior
			maxBurst := limit + burst - 1

			// Read initial capacity
			res := gr.run(t, gcraScriptOptions{
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
	})

	t.Run("simulate using up capacity and getting retryAt", func(t *testing.T) {
		forEachGCRARunner(t, gcraRateLimit, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := time.Hour
			limit := 5
			burst := 0

			// simulate gcraUpdate beheavior
			maxBurst := limit + burst - 1

			// Read initial capacity
			res := gr.run(t, gcraScriptOptions{
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
			res = gr.run(t, gcraScriptOptions{
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
	})

	t.Run("retryAt should be properly calculated", func(t *testing.T) {
		forEachGCRARunner(t, gcraRateLimit, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			// 10 every 60 minutes, 1 every 6s
			period := 1 * time.Minute
			limit := 10
			burst := 1

			// with full capacity, should show refill after 6s assuming that all capacity is consumed
			res := gr.run(t, gcraScriptOptions{
				key:      key,
				now:      clock.Now(),
				period:   period,
				limit:    limit,
				burst:    burst,
				quantity: 0,
			})
			require.WithinDuration(t, clock.Now().Add(6*time.Second), time.Unix(0, res.RetryAtMS), time.Second)

			// First request should work
			res = gr.run(t, gcraScriptOptions{
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
			gr.advance(clock.Now(), 2*time.Second)

			res = gr.run(t, gcraScriptOptions{
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
			gr.advance(clock.Now(), 4*time.Second)

			res = gr.run(t, gcraScriptOptions{
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
		forEachGCRARunner(t, gcraRateLimit, func(t *testing.T, gr gcraRunner) {

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

			key := "test"

			period := 1 * time.Minute
			limit := 10
			burst := 0
			// emission=6s, dvt=6s, dvt+emission=12s — inject TAT 13s ahead to exceed boundary
			tat := clock.Now().Add(13 * time.Second).UnixNano()
			gr.seed(t, key, tat)

			res := gr.run(t, gcraScriptOptions{
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
	})
}
