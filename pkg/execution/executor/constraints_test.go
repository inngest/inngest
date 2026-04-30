package executor

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
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

// scriptedCapacityManager is a CapacityManager whose Acquire response is
// scripted per-call. All other operations no-op or fail.
type scriptedCapacityManager struct {
	calls     atomic.Int64
	responses []*constraintapi.CapacityAcquireResponse
}

func (s *scriptedCapacityManager) Acquire(ctx context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	idx := int(s.calls.Add(1)) - 1
	if idx >= len(s.responses) {
		return nil, errs.Wrap(0, false, "scriptedCapacityManager: unexpected call %d", idx)
	}
	return s.responses[idx], nil
}

func (s *scriptedCapacityManager) Check(context.Context, *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	return nil, nil, errs.Wrap(0, false, "not implemented")
}

func (s *scriptedCapacityManager) ExtendLease(context.Context, *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	// Return a fresh lease so the background extension goroutine can run without
	// thrashing or canceling the schedule context.
	id := ulid.MustNew(ulid.Now(), nil)
	return &constraintapi.CapacityExtendLeaseResponse{LeaseID: &id}, nil
}

func (s *scriptedCapacityManager) Release(context.Context, *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	return &constraintapi.CapacityReleaseResponse{}, nil
}

// TestScheduleConstraintCacheDoesNotDropRetriesOnExhaustion exercises the
// silent-drop-on-retry case (SYS-820): when a Schedule attempt is rate
// limited, the in-process constraint cache stores the "exhausted" decision.
// On retry of the same event the cache must not silently deny the request -
// the original event ReceivedAt predates the cache entry, so the cache should
// be bypassed and the actual constraint state re-checked.
func TestScheduleConstraintCacheDoesNotDropRetriesOnExhaustion(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

	accountID := uuid.New()
	envID := uuid.New()
	appID := uuid.New()
	fnID := uuid.New()

	rateLimit := constraintapi.ConstraintItem{
		Kind: constraintapi.ConstraintKindRateLimit,
		RateLimit: &constraintapi.RateLimitConstraint{
			Scope: 0, // RateLimitScopeFn (default)
		},
	}

	// First Acquire: rate limit exhausted, no leases. Cache will be populated.
	// Subsequent Acquires: capacity restored (e.g. window moved on), single
	// lease granted. The bug surfaces when the cache prevents the second call
	// from ever reaching the manager.
	leaseID := ulid.MustNew(ulid.Timestamp(clock.Now().Add(time.Minute)), nil)
	manager := &scriptedCapacityManager{
		responses: []*constraintapi.CapacityAcquireResponse{
			{
				RequestID:            ulid.MustNew(ulid.Now(), nil),
				Leases:               nil,
				ExhaustedConstraints: []constraintapi.ConstraintItem{rateLimit},
				LimitingConstraints:  []constraintapi.ConstraintItem{rateLimit},
				RetryAfter:           clock.Now().Add(30 * time.Second),
			},
			{
				RequestID: ulid.MustNew(ulid.Now(), nil),
				Leases: []constraintapi.CapacityLease{
					{LeaseID: leaseID, IdempotencyKey: "lease-key"},
				},
			},
		},
	}

	cache := constraintapi.NewConstraintCache(
		constraintapi.WithConstraintCacheClock(clock),
		constraintapi.WithConstraintCacheManager(manager),
		constraintapi.WithConstraintCacheEnable(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, time.Duration, time.Duration) {
			return true, constraintapi.MinCacheTTL, constraintapi.MaxCacheTTL
		}),
	)

	useConstraintAPI := func(context.Context, uuid.UUID) bool { return true }

	receivedAt := clock.Now() // T0: when the event was received
	clock.Advance(time.Second)
	firstAttemptNow := clock.Now() // T1 = T0 + 1s

	fn := inngest.Function{
		ID: fnID,
		RateLimit: &inngest.RateLimit{
			Limit:  1,
			Period: "1m",
		},
	}

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

	idempotencyKey := "schedule-idempotency"
	tracer := trace.NoopConditionalTracer()

	called := func() (any, error) { return nil, nil }
	scheduleFn := func(ctx context.Context, performChecks bool) (any, error) {
		return called()
	}

	// First attempt: rate limit hit. Cache is populated with addedAt = T1.
	_, err := WithConstraints(
		ctx,
		firstAttemptNow,
		receivedAt,
		cache,
		useConstraintAPI,
		req,
		tracer,
		idempotencyKey,
		scheduleFn,
	)
	require.ErrorIs(t, err, ErrFunctionRateLimited, "first attempt should be rate limited")
	require.Equal(t, int64(1), manager.calls.Load(), "first attempt should call Acquire once")

	// Retry: same event, same idempotency key. RequestTime stays at the
	// original ReceivedAt (T0), which is before the cache entry's addedAt
	// (T1). The cache must be bypassed so the underlying manager is consulted
	// again - otherwise the retry is silently dropped.
	clock.Advance(time.Second)
	retryNow := clock.Now() // T2 = T0 + 2s

	_, err = WithConstraints(
		ctx,
		retryNow,
		receivedAt, // unchanged across retries
		cache,
		useConstraintAPI,
		req,
		tracer,
		idempotencyKey,
		scheduleFn,
	)
	require.NoError(t, err, "retry should not be silently rate limited from cache")
	require.Equal(t, int64(2), manager.calls.Load(), "retry must reach the underlying capacity manager (cache must be bypassed)")
}
