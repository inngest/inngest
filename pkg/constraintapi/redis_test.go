package constraintapi

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestRedisCapacityManager(t *testing.T) {
	r := miniredis.RunT(t)
	ctx := context.Background()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	cm, err := NewRedisCapacityManager(
		WithRateLimitClient(rc),
		WithQueueShards(map[string]rueidis.Client{}),
		WithClock(clock),
		WithNumScavengerShards(4),
		WithQueueStateKeyPrefix("q:v1"),
		WithRateLimitKeyPrefix("rl"),
	)
	require.NoError(t, err)
	require.NotNil(t, cm)

	// The following tests are essential functionality. We also have detailed test for each method,
	// to cover edge cases.

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	var leaseID ulid.ULID
	leaseIdempotencyKey := "event1"

	t.Run("Acquire", func(t *testing.T) {
		enableDebugLogs = true
		opIdempotencyKey := "event1"
		resp, err := cm.Acquire(ctx, &CapacityAcquireRequest{
			AccountID:            accountID,
			EnvID:                envID,
			FunctionID:           fnID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{leaseIdempotencyKey},
			IdempotencyKey:       "event1",
			LeaseRunIDs:          nil,
			Duration:             5 * time.Second,
			Source: LeaseSource{
				Service:           ServiceExecutor,
				Location:          LeaseLocationScheduleRun,
				RunProcessingMode: RunProcessingModeBackground,
			},
			Configuration: ConstraintConfig{
				FunctionVersion: 1,
				RateLimit: []RateLimitConfig{
					{
						KeyExpressionHash: "expr-hash",
						Limit:             120,
						Period:            60,
					},
				},
			},
			Constraints: []ConstraintItem{
				{
					Kind: ConstraintKindRateLimit,
					RateLimit: &RateLimitConstraint{
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "test-value",
					},
				},
			},
			CurrentTime:     clock.Now(),
			MaximumLifetime: time.Minute,
			Migration: MigrationIdentifier{
				IsRateLimit: true,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// internal state should match
		t.Log(resp.internalDebugState.Debug)
		require.Equal(t, 3, resp.internalDebugState.Status)
		require.Equal(t, 1, resp.internalDebugState.Granted)

		// One lease should have been granted
		require.Len(t, resp.Leases, 1)

		// Don't expect limiting constraint
		require.Nil(t, resp.LimitingConstraints)

		// RetryAfter should not be set
		require.Zero(t, resp.RetryAfter)

		// TODO: Verify all keys have been created as expected + TTLs set
		require.Len(t, r.Keys(), 7)
		require.True(t, r.Exists("{rl}:test-value")) // rate limit state exists
		require.True(t, r.Exists(cm.keyScavengerShard(cm.rateLimitKeyPrefix, 0)))
		require.True(t, r.Exists(cm.keyAccountLeases(cm.rateLimitKeyPrefix, accountID)))
		require.True(t, r.Exists(cm.keyLeaseDetails(cm.rateLimitKeyPrefix, accountID, leaseIdempotencyKey)))
		require.True(t, r.Exists(cm.keyConstraintCheckIdempotency(cm.rateLimitKeyPrefix, accountID, leaseIdempotencyKey)))
		require.True(t, r.Exists(cm.keyOperationIdempotency(cm.rateLimitKeyPrefix, accountID, "acq", opIdempotencyKey)))

		leaseID = resp.Leases[0].LeaseID
	})

	t.Run("Check", func(t *testing.T) {
		resp, userErr, internalErr := cm.Check(ctx, &CapacityCheckRequest{})
		require.NoError(t, userErr)
		require.NoError(t, internalErr)
		require.NotNil(t, resp)
	})

	t.Run("Extend", func(t *testing.T) {
		enableDebugLogs = true

		// Simulate that 2s have passed
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		opIdempotencyKey := "extend-test"

		resp, err := cm.ExtendLease(ctx, &CapacityExtendLeaseRequest{
			IdempotencyKey:      opIdempotencyKey,
			LeaseIdempotencyKey: leaseIdempotencyKey,
			Duration:            5 * time.Second,
			AccountID:           accountID,
			LeaseID:             leaseID,
			Migration: MigrationIdentifier{
				IsRateLimit: true,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		t.Log(resp.internalDebugState.Debug)

		require.Equal(t, 5, resp.internalDebugState.Status, r.Dump())
		require.NotEqual(t, ulid.Zero, resp.internalDebugState.LeaseID)

		require.NotNil(t, resp.LeaseID)

		// TODO: Verify all respective keys have been updated

		leaseID = *resp.LeaseID
	})

	t.Run("Release", func(t *testing.T) {
		enableDebugLogs = true

		// Simulate that 2s have passed
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		opIdempotencyKey := "release-test"

		resp, err := cm.Release(ctx, &CapacityReleaseRequest{
			IdempotencyKey:      opIdempotencyKey,
			LeaseIdempotencyKey: leaseIdempotencyKey,
			AccountID:           accountID,
			LeaseID:             leaseID,
			Migration: MigrationIdentifier{
				IsRateLimit: true,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		t.Log(resp.internalDebugState.Debug)

		require.Equal(t, 5, resp.internalDebugState.Status, r.Dump())
		require.Equal(t, 0, resp.internalDebugState.Remaining)

		// TODO: Verify all respective keys have been updated
		require.Len(t, r.Keys(), 0, r.Dump())
	})
}
