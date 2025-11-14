package constraintapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestRedisCapacityManager_RateLimit(t *testing.T) {
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

	config := ConstraintConfig{
		FunctionVersion: 1,
		RateLimit: []RateLimitConfig{
			{
				KeyExpressionHash: "expr-hash",
				Limit:             120,
				Period:            60,
			},
		},
	}

	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindRateLimit,
			RateLimit: &RateLimitConstraint{
				KeyExpressionHash: "expr-hash",
				EvaluatedKeyHash:  "test-value",
			},
		},
	}

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
			Configuration:   config,
			Constraints:     constraints,
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

		leaseID = resp.Leases[0].LeaseID

		// TODO: Verify all keys have been created as expected + TTLs set
		require.Len(t, r.Keys(), 7)
		require.True(t, r.Exists("{rl}:test-value")) // rate limit state exists
		require.True(t, r.Exists(cm.keyScavengerShard(cm.rateLimitKeyPrefix, 0)))
		require.True(t, r.Exists(cm.keyAccountLeases(cm.rateLimitKeyPrefix, accountID)))
		require.True(t, r.Exists(cm.keyLeaseDetails(cm.rateLimitKeyPrefix, accountID, leaseID)))
		require.True(t, r.Exists(cm.keyConstraintCheckIdempotency(cm.rateLimitKeyPrefix, accountID, leaseIdempotencyKey)))
		require.True(t, r.Exists(cm.keyOperationIdempotency(cm.rateLimitKeyPrefix, accountID, "acq", opIdempotencyKey)))
	})

	var checkHash string
	t.Run("Check", func(t *testing.T) {
		req := &CapacityCheckRequest{
			AccountID:     accountID,
			EnvID:         envID,
			FunctionID:    fnID,
			Configuration: config,
			Constraints:   constraints,
			Migration: MigrationIdentifier{
				IsRateLimit: true,
			},
		}

		_, _, hash, err := buildCheckRequestData(req, cm.rateLimitKeyPrefix)
		require.NoError(t, err)
		require.NotZero(t, hash)
		checkHash = hash

		resp, userErr, internalErr := cm.Check(ctx, req)
		require.NoError(t, userErr)
		require.NoError(t, internalErr)
		require.NotNil(t, resp)

		require.Equal(t, 11, resp.AvailableCapacity)
		require.Equal(t, ConstraintKindRateLimit, resp.LimitingConstraints[0].Kind)
		require.Equal(t, 120, resp.Usage[0].Limit)
		// TODO: Figure out why capacity calculation is buggy
		require.Equal(t, 0, resp.Usage[0].Used)
	})

	t.Run("Extend", func(t *testing.T) {
		enableDebugLogs = true

		// Simulate that 2s have passed
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		opIdempotencyKey := "extend-test"

		resp, err := cm.ExtendLease(ctx, &CapacityExtendLeaseRequest{
			IdempotencyKey: opIdempotencyKey,
			Duration:       5 * time.Second,
			AccountID:      accountID,
			LeaseID:        leaseID,
			Migration: MigrationIdentifier{
				IsRateLimit: true,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		t.Log(resp.internalDebugState.Debug)

		require.Equal(t, 4, resp.internalDebugState.Status, r.Dump())
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

		t.Log(r.Dump())

		opIdempotencyKey := "release-test"

		resp, err := cm.Release(ctx, &CapacityReleaseRequest{
			IdempotencyKey: opIdempotencyKey,
			AccountID:      accountID,
			LeaseID:        leaseID,
			Migration: MigrationIdentifier{
				IsRateLimit: true,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		t.Log(resp.internalDebugState.Debug)

		require.Equal(t, 3, resp.internalDebugState.Status, r.Dump())
		require.Equal(t, 0, resp.internalDebugState.Remaining)

		// TODO: Verify all respective keys have been updated
		// TODO: Expect 4 idempotency keys (1 constraint check + 3 operations)
		keys := r.Keys()
		require.Len(t, keys, 5, r.Dump())
		require.Contains(t, keys, cm.keyConstraintCheckIdempotency(cm.rateLimitKeyPrefix, accountID, "event1"))
		require.Contains(t, keys, cm.keyOperationIdempotency(cm.rateLimitKeyPrefix, accountID, "acq", "event1"))
		require.Contains(t, keys, cm.keyOperationIdempotency(cm.rateLimitKeyPrefix, accountID, "ext", "extend-test"))
		require.Contains(t, keys, cm.keyOperationIdempotency(cm.rateLimitKeyPrefix, accountID, "rel", "release-test"))
		require.Contains(t, keys, cm.keyOperationIdempotency(cm.rateLimitKeyPrefix, accountID, "chk", checkHash))
	})
}

func TestRedisCapacityManager_Concurrency(t *testing.T) {
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
		WithQueueShards(map[string]rueidis.Client{
			"test": rc,
		}),
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

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			AccountConcurrency:  20,
			FunctionConcurrency: 5,
		},
	}

	acctConcurrency := fmt.Sprintf("{%s}:concurrency:account:%s", cm.queueStateKeyPrefix, accountID)
	fnConcurrency := fmt.Sprintf("{%s}:concurrency:p:%s", cm.queueStateKeyPrefix, fnID)

	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeAccount,
				InProgressItemKey: acctConcurrency,
			},
		},
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeFn,
				InProgressItemKey: fnConcurrency,
			},
		},
	}

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
			Configuration:   config,
			Constraints:     constraints,
			CurrentTime:     clock.Now(),
			MaximumLifetime: time.Minute,
			Migration: MigrationIdentifier{
				QueueShard: "test",
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

		leaseID = resp.Leases[0].LeaseID

		// TODO: Verify all keys have been created as expected + TTLs set
		require.Len(t, r.Keys(), 8)
		require.False(t, r.Exists(acctConcurrency)) // we do not modify the in progress items directly
		require.True(t, r.Exists(constraints[0].Concurrency.InProgressLeasesKey(cm.queueStateKeyPrefix, accountID, envID, fnID)), r.Dump())
		require.False(t, r.Exists(fnConcurrency)) // we do not modify the in progress items directly
		require.True(t, r.Exists(constraints[1].Concurrency.InProgressLeasesKey(cm.queueStateKeyPrefix, accountID, envID, fnID)), r.Dump())

		require.True(t, r.Exists(cm.keyScavengerShard(cm.queueStateKeyPrefix, 0)))
		require.True(t, r.Exists(cm.keyAccountLeases(cm.queueStateKeyPrefix, accountID)))
		require.True(t, r.Exists(cm.keyLeaseDetails(cm.queueStateKeyPrefix, accountID, leaseID)))
		require.True(t, r.Exists(cm.keyConstraintCheckIdempotency(cm.queueStateKeyPrefix, accountID, leaseIdempotencyKey)))
		require.True(t, r.Exists(cm.keyOperationIdempotency(cm.queueStateKeyPrefix, accountID, "acq", opIdempotencyKey)))
	})

	var checkHash string
	t.Run("Check", func(t *testing.T) {
		req := &CapacityCheckRequest{
			AccountID:     accountID,
			EnvID:         envID,
			FunctionID:    fnID,
			Configuration: config,
			Constraints:   constraints,
			Migration: MigrationIdentifier{
				IsRateLimit: true,
			},
		}

		_, _, hash, err := buildCheckRequestData(req, cm.rateLimitKeyPrefix)
		require.NoError(t, err)
		require.NotZero(t, hash)
		checkHash = hash

		resp, userErr, internalErr := cm.Check(ctx, req)
		require.NoError(t, userErr)
		require.NoError(t, internalErr)
		require.NotNil(t, resp)

		require.Equal(t, 4, resp.AvailableCapacity, r.Dump())
		require.Equal(t, ConstraintKindConcurrency, resp.LimitingConstraints[0].Kind)
		require.Equal(t, enums.ConcurrencyScopeAccount, resp.LimitingConstraints[0].Concurrency.Scope)
		require.Equal(t, enums.ConcurrencyScopeFn, resp.LimitingConstraints[1].Concurrency.Scope)
		require.Equal(t, 20, resp.Usage[0].Limit)
		require.Equal(t, 1, resp.Usage[0].Used)
		require.Equal(t, 5, resp.Usage[1].Limit)
		require.Equal(t, 1, resp.Usage[2].Used)
	})

	t.Run("Extend", func(t *testing.T) {
		enableDebugLogs = true

		// Simulate that 2s have passed
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		opIdempotencyKey := "extend-test"

		resp, err := cm.ExtendLease(ctx, &CapacityExtendLeaseRequest{
			IdempotencyKey: opIdempotencyKey,
			Duration:       5 * time.Second,
			AccountID:      accountID,
			LeaseID:        leaseID,
			Migration: MigrationIdentifier{
				IsRateLimit: true,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		t.Log(resp.internalDebugState.Debug)

		require.Equal(t, 4, resp.internalDebugState.Status, r.Dump())
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

		t.Log(r.Dump())

		opIdempotencyKey := "release-test"

		resp, err := cm.Release(ctx, &CapacityReleaseRequest{
			IdempotencyKey: opIdempotencyKey,
			AccountID:      accountID,
			LeaseID:        leaseID,
			Migration: MigrationIdentifier{
				IsRateLimit: true,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		t.Log(resp.internalDebugState.Debug)

		require.Equal(t, 3, resp.internalDebugState.Status, r.Dump())
		require.Equal(t, 0, resp.internalDebugState.Remaining)

		// TODO: Verify all respective keys have been updated
		// TODO: Expect 4 idempotency keys (1 constraint check + 3 operations)
		keys := r.Keys()
		require.Len(t, keys, 5, r.Dump())
		require.Contains(t, keys, cm.keyConstraintCheckIdempotency(cm.rateLimitKeyPrefix, accountID, "event1"))
		require.Contains(t, keys, cm.keyOperationIdempotency(cm.rateLimitKeyPrefix, accountID, "acq", "event1"))
		require.Contains(t, keys, cm.keyOperationIdempotency(cm.rateLimitKeyPrefix, accountID, "ext", "extend-test"))
		require.Contains(t, keys, cm.keyOperationIdempotency(cm.rateLimitKeyPrefix, accountID, "rel", "release-test"))
		require.Contains(t, keys, cm.keyOperationIdempotency(cm.rateLimitKeyPrefix, accountID, "chk", checkHash))
	})
}
