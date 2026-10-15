package constraintapi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
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

			// A separate event with the same message key is still a duplicate.
			req.IdempotencyKey = "another-event"
			req.LeaseIdempotencyKeys = []string{"another-event"}
			require.Empty(t, acquireAndRelease().Leases)
		})
	}
}
