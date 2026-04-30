package executor

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestRateLimitKeyExpressionHashConsistency(t *testing.T) {
	ptr := func(s string) *string { return &s }

	tests := []struct {
		name                      string
		rateLimitKey              *string
		expectedKeyExpressionHash string
	}{
		{
			name:                      "with key expression",
			rateLimitKey:              ptr("event.data.userId"),
			expectedKeyExpressionHash: util.XXHash("event.data.userId"),
		},
		{
			name:                      "without key expression",
			rateLimitKey:              nil,
			expectedKeyExpressionHash: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fnID := uuid.New()
			fn := inngest.Function{
				ID: fnID,
				RateLimit: &inngest.RateLimit{
					Limit:  1,
					Period: "1m",
					Key:    tt.rateLimitKey,
				},
			}

			// Get KeyExpressionHash from ConvertToConstraintConfiguration
			config, err := queue.ConvertToConstraintConfiguration(0, fn)
			require.NoError(t, err)
			require.Len(t, config.RateLimit, 1)
			configHash := config.RateLimit[0].KeyExpressionHash

			// Get KeyExpressionHash from getScheduleConstraints
			req := execution.ScheduleRequest{
				Function: fn,
				Events: []event.TrackedEvent{
					event.InternalEvent{
						Event: event.Event{
							Name: "test",
							Data: map[string]any{"userId": "test-user"},
						},
					},
				},
			}

			constraints, err := getScheduleConstraints(context.Background(), req)
			require.NoError(t, err)
			require.Len(t, constraints, 1)
			constraintHash := constraints[0].RateLimit.KeyExpressionHash

			// Both must match each other and the expected value
			require.Equal(t, tt.expectedKeyExpressionHash, configHash, "config KeyExpressionHash mismatch")
			require.Equal(t, tt.expectedKeyExpressionHash, constraintHash, "constraint KeyExpressionHash mismatch")
			require.Equal(t, configHash, constraintHash, "config and constraint KeyExpressionHash must be equal")
		})
	}
}

// TestScheduleConstraintCacheDoesNotDropRetriesOnExhaustion exercises the
// silent-drop-on-retry case (SYS-820): when a Schedule attempt is rate
// limited, the in-process constraint cache stores the "exhausted" decision.
// On retry of the same event the cache must not silently deny the request -
// the original event ReceivedAt predates the cache entry, so the cache must
// be bypassed and the actual constraint state re-checked.
//
// This uses a real redisCapacityManager backed by miniredis (matching
// production), wrapped in the in-process constraintCache. The rate limit
// period is short (1s) so the GCRA window is past the rate limit by the time
// we retry; without the fix the cache would still deny the request and the
// event would be silently dropped.
func TestScheduleConstraintCacheDoesNotDropRetriesOnExhaustion(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(rc.Close)

	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

	// Lifecycle hooks let us count how many times the underlying manager's
	// Acquire actually ran - useful to distinguish "cache hit" (no manager
	// call) from "cache bypass + manager limited" (manager call returns no
	// leases).
	lifecycles := constraintapi.NewConstraintAPIDebugLifecycles()

	// Production-shaped capacity manager: real Lua-driven Redis logic, fake
	// clock so we can advance past the rate-limit window deterministically.
	// Idempotency TTLs are set to the minimum (1s, the floor for Redis EX);
	// we expire them between attempts via miniredis.FastForward so the second
	// Acquire actually re-runs GCRA instead of replaying a cached response.
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName("test"),
		constraintapi.WithClock(clock),
		constraintapi.WithOperationIdempotencyTTL(time.Second),
		constraintapi.WithConstraintCheckIdempotencyTTL(time.Second),
		constraintapi.WithCheckIdempotencyTTL(time.Second),
		constraintapi.WithLifecycles(lifecycles),
	)
	require.NoError(t, err)

	// In-process cache wrapping the real manager. Min TTL 1s, max TTL 1m -
	// matching production constants and the request's instructions.
	cache := constraintapi.NewConstraintCache(
		constraintapi.WithConstraintCacheClock(clock),
		constraintapi.WithConstraintCacheManager(cm),
		constraintapi.WithConstraintCacheEnable(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, time.Duration, time.Duration) {
			return true, time.Second, time.Minute
		}),
	)

	useConstraintAPI := func(context.Context, uuid.UUID) bool { return true }

	accountID := uuid.New()
	envID := uuid.New()
	appID := uuid.New()
	fnID := uuid.New()

	// Rate limit of 1 per second. With period=1s, advancing the clock by >1s
	// puts us past the GCRA window so the underlying manager will allow a
	// retry - if the cache lets us reach it.
	fn := inngest.Function{
		ID:              fnID,
		FunctionVersion: 1,
		RateLimit: &inngest.RateLimit{
			Limit:  1,
			Period: "1s",
		},
	}

	receivedAt := clock.Now() // T0: when the event was received

	// Advance the clock so the first attempt's wall-clock is strictly after
	// receivedAt - this is what makes addedAt > requestTime on the cache
	// entry populated by the first attempt.
	clock.Advance(100 * time.Millisecond)
	firstAttemptNow := clock.Now() // T0 + 100ms

	evt := event.InternalEvent{
		ID:          ulid.MustNew(ulid.Now(), nil),
		AccountID:   accountID,
		WorkspaceID: envID,
		Event: event.Event{
			Name: "test",
			Data: map[string]any{},
		},
		ReceivedAt: receivedAt,
	}

	req := execution.ScheduleRequest{
		Function:    fn,
		AccountID:   accountID,
		WorkspaceID: envID,
		AppID:       appID,
		Events:      []event.TrackedEvent{evt},
	}

	tracer := trace.NoopConditionalTracer()

	scheduleCalls := 0
	scheduleFn := func(ctx context.Context, performChecks bool) (any, error) {
		scheduleCalls++
		return nil, nil
	}

	runAttempt := func(now, requestTime time.Time, idempotencyKey string) error {
		_, err := WithConstraints(
			ctx,
			now,
			requestTime,
			cache,
			useConstraintAPI,
			req,
			tracer,
			idempotencyKey,
			scheduleFn,
		)
		return err
	}

	// Attempt 1 - warmup. Consumes the only allowed call in the rate-limit
	// window, exhausts the constraint, and populates the in-process cache
	// with addedAt = T0 + 100ms.
	require.NoError(t, runAttempt(firstAttemptNow, receivedAt, "warmup-key"),
		"warmup attempt should pass (lease granted, capacity then exhausted)")
	require.Equal(t, 1, scheduleCalls, "warmup attempt should run the schedule fn once")
	require.Equal(t, 1, len(lifecycles.AcquireCalls), "warmup must reach the manager")

	// Attempt 2 - old event (RequestTime predates addedAt) with a fresh
	// idempotency key. The cache is bypassed because RequestTime < addedAt,
	// but the rate-limit window is still active in Redis: GCRA inside the
	// capacity manager rejects the request, so the schedule is denied.
	// This verifies the manager's per-request handling kicks in once the
	// cache has stepped aside.
	require.ErrorIs(t,
		runAttempt(clock.Now(), receivedAt, "fresh-key"),
		ErrFunctionRateLimited,
		"old event with different idempotency key must be limited by the capacity manager (GCRA window active)",
	)
	require.Equal(t, 1, scheduleCalls, "manager-limited attempt must not run the schedule fn")
	require.Equal(t, 2, len(lifecycles.AcquireCalls),
		"old event with different idempotency key must reach the manager (cache bypassed)")

	// Attempt 3 - new event whose RequestTime is later than the cache
	// entry's addedAt. The cache must answer here without consulting the
	// manager: the AcquireCalls counter does not advance.
	newRequestTime := firstAttemptNow.Add(50 * time.Millisecond)
	require.ErrorIs(t,
		runAttempt(clock.Now(), newRequestTime, "new-event-key"),
		ErrFunctionRateLimited,
		"new event (RequestTime > addedAt) must be limited by the in-process cache",
	)
	require.Equal(t, 1, scheduleCalls, "cache-limited attempt must not run the schedule fn")
	require.Equal(t, 2, len(lifecycles.AcquireCalls),
		"new event must NOT reach the manager (cache hit)")

	// Attempt 4 - retry of the original event after the rate-limit window
	// has elapsed. Advance past the rate-limit window in the manager's fake
	// clock; FastForward miniredis so the Lua-script idempotency keys
	// (EX 1s) have expired and the second Acquire actually re-runs GCRA.
	// The in-process cache entry is still alive: ccache uses real
	// wall-clock time and only milliseconds have elapsed in real time.
	//
	// Because RequestTime (T0) < addedAt (T0 + 100ms), the cache is
	// bypassed and the underlying manager grants a fresh lease. Without
	// the fix this hits the cached "exhausted" entry and silently drops
	// the event with ErrFunctionRateLimited.
	clock.Advance(2 * time.Second)
	r.FastForward(2 * time.Second)
	require.NoError(t,
		runAttempt(clock.Now(), receivedAt, "warmup-key"),
		"retry must not be silently rate limited from a stale cache entry",
	)
	require.Equal(t, 2, scheduleCalls, "retry should run the schedule fn (cache bypassed, manager re-consulted)")
	require.Equal(t, 3, len(lifecycles.AcquireCalls), "retry must reach the manager")
}
