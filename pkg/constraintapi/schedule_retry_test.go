package constraintapi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

func TestAcquireRetryProtection(t *testing.T) {
	for _, tc := range []struct {
		name     string
		location CallerLocation
		gap      time.Duration
		allowed  bool
		override time.Duration
	}{
		{"schedule survives maximum delivery backoff", CallerLocationSchedule, 10 * time.Minute, true, 0},
		{"schedule protection remains bounded", CallerLocationSchedule, 31 * time.Minute, false, 0},
		{"queue lease retains shorter protection", CallerLocationItemLease, 10 * time.Minute, false, 0},
		{"explicit TTL overrides schedule default", CallerLocationSchedule, 2 * time.Minute, false, time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cm, redis, clock, ctx := newTestSetup(t, nil)
			if tc.override != 0 {
				WithConstraintCheckIdempotencyTTL(tc.override)(cm)
			}
			manager := NewConstraintCache(
				WithConstraintCacheClock(clock),
				WithConstraintCacheManager(cm),
				WithConstraintCacheEnable(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, time.Duration, time.Duration) {
					return true, time.Second, time.Minute
				}),
			)
			req := makeAcquireRequest(uuid.New(), uuid.New(), uuid.New(), clock,
				ConstraintConfig{
					FunctionVersion: 1,
					RateLimit:       []RateLimitConfig{{KeyExpressionHash: "message-id", Limit: 1, Period: 86400}},
				},
				[]ConstraintItem{{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{
					KeyExpressionHash: "message-id", EvaluatedKeyHash: "same-message",
				}}}, "original-event")
			req.Source.Location = tc.location
			req.RequestTime = clock.Now().Add(-time.Second)

			acquireAndRelease := func() *CapacityAcquireResponse {
				t.Helper()
				req.CurrentTime = clock.Now()
				res, err := manager.Acquire(ctx, req)
				require.NoError(t, err)
				for _, lease := range res.Leases {
					// Run creation failed after capacity was consumed. Scheduling
					// releases its lease, but the 24-hour rate limit remains used.
					_, err := manager.Release(ctx, &CapacityReleaseRequest{
						AccountID: req.AccountID, LeaseID: lease.LeaseID,
						IdempotencyKey: lease.LeaseID.String(), Source: req.Source,
						LeaseIssuedAt: clock.Now(),
					})
					require.NoError(t, err)
				}
				return res
			}
			require.Len(t, acquireAndRelease().Leases, 1)
			for range 3 {
				clock.Advance(tc.gap)
				redis.FastForward(tc.gap)
				res := acquireAndRelease()
				if !tc.allowed {
					require.Empty(t, res.Leases)
					break
				}
				require.Len(t, res.Leases, 1, "the original event must remain retryable across delivery delays")
			}

			// Regression guard: a separate event with the same message key remains
			// blocked by the 24-hour rate limit, independently of marker retention.
			req.IdempotencyKey = "another-event"
			req.LeaseIdempotencyKeys = []string{"another-event"}
			require.Empty(t, acquireAndRelease().Leases)
		})
	}
}

func TestAcquireRetryMarkerTTL(t *testing.T) {
	for _, tc := range []struct {
		name        string
		kind        ConstraintKind
		location    CallerLocation
		scheduleTTL time.Duration
		want        time.Duration
	}{
		{"schedule with concurrency only", ConstraintKindConcurrency, CallerLocationSchedule, 0, 5 * time.Minute},
		{"schedule with rate limit", ConstraintKindRateLimit, CallerLocationSchedule, 0, 30 * time.Minute},
		{"schedule with throttle", ConstraintKindThrottle, CallerLocationSchedule, 0, 30 * time.Minute},
		{"queue with rate limit", ConstraintKindRateLimit, CallerLocationItemLease, 0, 5 * time.Minute},
		{"checkpoint with rate limit", ConstraintKindRateLimit, CallerLocationCheckpoint, 0, 5 * time.Minute},
		{"schedule-only override", ConstraintKindRateLimit, CallerLocationSchedule, 20 * time.Minute, 20 * time.Minute},
		{"schedule override leaves concurrency unchanged", ConstraintKindConcurrency, CallerLocationSchedule, 20 * time.Minute, 5 * time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cm, redis, clock, ctx := newTestSetup(t, nil)
			if tc.scheduleTTL != 0 {
				WithScheduleConstraintCheckIdempotencyTTL(tc.scheduleTTL)(cm)
			}
			config := ConstraintConfig{FunctionVersion: 1}
			var constraints []ConstraintItem
			switch tc.kind {
			case ConstraintKindRateLimit:
				config.RateLimit = []RateLimitConfig{{Limit: 1, Period: 86400}}
				constraints = []ConstraintItem{{Kind: tc.kind, RateLimit: &RateLimitConstraint{EvaluatedKeyHash: "message"}}}
			case ConstraintKindThrottle:
				config.Throttle = []ThrottleConfig{{Scope: enums.ThrottleScopeFn, Limit: 1, Burst: 1, Period: 86400}}
				constraints = []ConstraintItem{{Kind: tc.kind, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn, EvaluatedKeyHash: "message"}}}
			case ConstraintKindConcurrency:
				config.Concurrency.FunctionConcurrency = 1
				constraints = []ConstraintItem{{Kind: tc.kind, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn, Mode: enums.ConcurrencyModeStep}}}
			}
			req := makeAcquireRequest(uuid.New(), uuid.New(), uuid.New(), clock, config, constraints, "request")
			req.Source.Location = tc.location
			res, err := cm.Acquire(ctx, req)
			require.NoError(t, err)
			require.Len(t, res.Leases, 1)
			// Use distinct request and lease keys so both Lua SET paths are checked.
			require.NotEqual(t, req.IdempotencyKey, req.LeaseIdempotencyKeys[0])
			for _, key := range []string{req.IdempotencyKey, req.LeaseIdempotencyKeys[0]} {
				require.Equal(t, tc.want, redis.TTL(cm.keyConstraintCheckIdempotency(req.AccountID, key)))
			}
		})
	}
}
