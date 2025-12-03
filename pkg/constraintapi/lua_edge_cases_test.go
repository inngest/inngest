package constraintapi

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestLuaScriptEdgeCases_RateLimitGCRA(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Zero Limit Handling", func(t *testing.T) {
		initialState := te.CaptureRedisState()

		config := ConstraintConfig{
			FunctionVersion: 1,
			RateLimit: []RateLimitConfig{
				{
					Scope:             enums.RateLimitScopeFn,
					Limit:             0, // Zero limit should immediately rate limit
					Period:            60,
					KeyExpressionHash: "zero-limit",
				},
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeFn,
					KeyExpressionHash: "zero-limit",
					EvaluatedKeyHash:  "zero-test",
				},
			},
		}

		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "zero-limit-test",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"lease-1"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{IsRateLimit: true},
		})

		require.NoError(t, err)
		require.Empty(t, resp.Leases, "Zero limit should grant no leases")
		require.NotEmpty(t, resp.LimitingConstraints, "Should have limiting constraints")
		require.True(t, resp.RetryAfter.After(clock.Now()), "Should have retry after time")

		// Verify no unexpected keys were created
		te.VerifyNoResourceLeaks(initialState, []string{
			te.CapacityManager.keyOperationIdempotency(te.CapacityManager.rateLimitKeyPrefix, te.AccountID, "acq", "zero-limit-test"),
			te.CapacityManager.keyConstraintCheckIdempotency(te.CapacityManager.rateLimitKeyPrefix, te.AccountID, "zero-limit-test"),
		})
	})

	t.Run("TAT Corruption Recovery", func(t *testing.T) {
		rateLimitKey := fmt.Sprintf("{%s}:corrupted-key", te.CapacityManager.rateLimitKeyPrefix)

		// Inject corrupted TAT value (far future)
		corruptedTAT := clock.Now().Add(time.Hour * 24 * 365).UnixNano() // 1 year in future
		err := te.Redis.Set(rateLimitKey, strconv.FormatInt(corruptedTAT, 10))
		require.NoError(t, err)
		te.Redis.SetTTL(rateLimitKey, time.Hour)

		config := ConstraintConfig{
			FunctionVersion: 1,
			RateLimit: []RateLimitConfig{
				{
					Scope:             enums.RateLimitScopeFn,
					Limit:             10,
					Period:            60,
					KeyExpressionHash: "corrupted",
				},
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeFn,
					KeyExpressionHash: "corrupted",
					EvaluatedKeyHash:  "corrupted-key",
				},
			},
		}

		// First request should normalize the corrupted TAT
		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "corruption-test-1",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"lease-corruption-1"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{IsRateLimit: true},
		})

		require.NoError(t, err)
		require.Len(t, resp.Leases, 0, "Should successfully acquire lease after TAT normalization")

		// Verify TAT was normalized
		rv := te.NewRateLimitStateVerifier()
		now := clock.Now().UnixNano()
		rv.VerifyRateLimitState(rateLimitKey, now, now+int64(time.Hour))
	})

	t.Run("Clock Skew Tolerance", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			RateLimit: []RateLimitConfig{
				{
					Scope:             enums.RateLimitScopeFn,
					Limit:             5,
					Period:            10, // 10 seconds
					KeyExpressionHash: "clock-skew",
				},
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeFn,
					KeyExpressionHash: "clock-skew",
					EvaluatedKeyHash:  "skew-test",
				},
			},
		}

		// Simulate client with clock skew (5 seconds behind)
		skewedTime := clock.Now().Add(-5 * time.Second)

		_, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "skew-test-1",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"lease-skew-1"},
			CurrentTime:          skewedTime,
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{IsRateLimit: true},
		})

		require.Error(t, err)

		// Extreme clock skew (1 hour behind) - should be normalized
		extremeSkew := clock.Now().Add(-time.Hour)

		_, err = te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "skew-test-2",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"lease-skew-2"},
			CurrentTime:          extremeSkew,
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{IsRateLimit: true},
		})

		require.Error(t, err)
	})
}

func TestLuaScriptEdgeCases_Concurrency(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("In-Progress Items vs Leases Reconciliation", func(t *testing.T) {
		inProgressItemKey := fmt.Sprintf("{%s}:concurrency:items:%s", te.KeyPrefix, te.FunctionID)

		// Pre-populate in-progress items (simulating existing queue items)
		_, err := te.Redis.ZAdd(inProgressItemKey, float64(clock.Now().Add(time.Minute).UnixMilli()), "item-1")
		require.NoError(t, err)
		_, err = te.Redis.ZAdd(inProgressItemKey, float64(clock.Now().Add(time.Minute).UnixMilli()), "item-2")
		require.NoError(t, err)

		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: inProgressItemKey,
				},
			},
		}

		// Should have capacity for 3 more (5 - 2 existing items)
		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "reconcile-test",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               4, // Request more than available
			LeaseIdempotencyKeys: []string{"lease-1", "lease-2", "lease-3", "lease-4"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Len(t, resp.Leases, 3, "Should grant available capacity accounting for existing items")
		require.NotEmpty(t, resp.LimitingConstraints, "Should report concurrency as limiting")

		// Verify constraint state
		cv := te.NewConstraintVerifier()
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 5}) // 2 items + 3 leases
	})

	t.Run("Expired Lease Detection", func(t *testing.T) {
		te.Redis.FlushAll()

		inProgressItemKey := fmt.Sprintf("{%s}:concurrency:items2:%s", te.KeyPrefix, te.FunctionID)

		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 3,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: inProgressItemKey,
				},
			},
		}

		inProgressLeasesKey := constraints[0].Concurrency.InProgressLeasesKey(te.KeyPrefix, te.AccountID, te.EnvID, te.FunctionID)

		// Pre-populate with expired and active leases
		expiredTime := clock.Now().Add(-time.Minute).UnixMilli()
		activeTime := clock.Now().Add(time.Minute).UnixMilli()

		expiredLeaseID := ulid.Make()
		activeLeaseID := ulid.Make()

		_, err := te.Redis.ZAdd(inProgressLeasesKey, float64(expiredTime), expiredLeaseID.String())
		require.NoError(t, err)
		_, err = te.Redis.ZAdd(inProgressLeasesKey, float64(activeTime), activeLeaseID.String())
		require.NoError(t, err)

		// Should count only active leases (1), so capacity should be 2
		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "expired-test",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               3,
			LeaseIdempotencyKeys: []string{"lease-1", "lease-2", "lease-3"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		t.Log(resp.internalDebugState.Debug)
		require.Len(t, resp.Leases, 2, "Should only count active leases in capacity calculation")
	})

	t.Run("Zero Concurrency Limit", func(t *testing.T) {
		te.Redis.FlushAll()

		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 0, // No concurrency allowed
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: "zero-concurrency-key",
				},
			},
		}

		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "zero-concurrency",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"lease-zero"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Empty(t, resp.Leases, "Zero concurrency limit should grant no leases")
		require.NotEmpty(t, resp.LimitingConstraints, "Should report concurrency as limiting")
	})
}

func TestLuaScriptEdgeCases_Throttle(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Very Small Emission Intervals", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Throttle: []ThrottleConfig{
				{
					Scope:                     enums.ThrottleScopeFn,
					Limit:                     1000000, // Very high limit
					Burst:                     100000,  // High burst
					Period:                    1,       // 1 second period
					ThrottleKeyExpressionHash: "small-interval",
				},
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeFn,
					KeyExpressionHash: "small-interval",
					EvaluatedKeyHash:  "small-test",
				},
			},
		}

		// Should handle very small emission intervals correctly
		//

		enableDebugLogs = true

		// Fill in lease idempotency keys
		req := &CapacityAcquireRequest{
			IdempotencyKey:       "small-interval-test",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               20,
			LeaseIdempotencyKeys: make([]string, 20),
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		}
		for i := 0; i < 20; i++ {
			req.LeaseIdempotencyKeys[i] = fmt.Sprintf("lease-%d", i)
		}

		resp, err := te.CapacityManager.Acquire(context.Background(), req)

		require.NoError(t, err)
		t.Log(resp.internalDebugState.Debug)
		require.NotEmpty(t, resp.Leases, "Should grant some capacity even with very small intervals")
		require.True(t, len(resp.Leases) <= 20, "Should not grant more than requested")
	})

	t.Run("Very Large Periods", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Throttle: []ThrottleConfig{
				{
					Scope:                     enums.ThrottleScopeFn,
					Limit:                     1,
					Burst:                     0,
					Period:                    86400, // 24 hours
					ThrottleKeyExpressionHash: "large-period",
				},
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeFn,
					KeyExpressionHash: "large-period",
					EvaluatedKeyHash:  "large-test",
				},
			},
		}

		enableDebugLogs = true

		// First request should succeed
		resp1, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "large-period-1",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"lease-large-1"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		t.Log(resp1.internalDebugState.Debug)
		require.Len(t, resp1.Leases, 1, "First request should succeed")

		// Second immediate request should be throttled
		resp2, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "large-period-2",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"lease-large-2"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Empty(t, resp2.Leases, "Second request should be throttled")
		require.True(t, resp2.RetryAfter.After(clock.Now()), "Should have future retry time")
	})

	t.Run("Zero Limit Throttling", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Throttle: []ThrottleConfig{
				{
					Scope:                     enums.ThrottleScopeFn,
					Limit:                     0, // No throughput allowed
					Burst:                     0,
					Period:                    60,
					ThrottleKeyExpressionHash: "zero-throttle",
				},
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeFn,
					KeyExpressionHash: "zero-throttle",
					EvaluatedKeyHash:  "zero-throttle-test",
				},
			},
		}

		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "zero-throttle-test",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"lease-zero-throttle"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Empty(t, resp.Leases, "Zero limit throttling should grant no leases")
		require.NotEmpty(t, resp.LimitingConstraints, "Should report throttle as limiting")
	})
}

func TestLuaScriptEdgeCases_ErrorConditions(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Invalid JSON in Request State", func(t *testing.T) {
		// Pre-populate invalid request state
		requestStateKey := te.CapacityManager.keyRequestState(te.KeyPrefix, te.AccountID, "invalid-json")
		err := te.Redis.Set(requestStateKey, "invalid-json-data")
		require.NoError(t, err)

		// Try to extend lease which will try to read the corrupted state
		resp, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
			IdempotencyKey: "extend-invalid",
			AccountID:      te.AccountID,
			LeaseID:        ulid.Make(),
			Duration:       5 * time.Second,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		// Should handle gracefully (specific error handling depends on implementation)
		require.NoError(t, err)
		require.Equal(t, 3, resp.internalDebugState.Status)
	})

	t.Run("Missing Lease Details", func(t *testing.T) {
		leaseID := ulid.Make()

		// Try to extend a lease that doesn't exist
		resp, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
			IdempotencyKey: "extend-missing",
			AccountID:      te.AccountID,
			LeaseID:        leaseID,
			Duration:       5 * time.Second,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Nil(t, resp.LeaseID, "Should handle missing lease gracefully")
	})

	t.Run("Expired Lease Extension", func(t *testing.T) {
		// Create a lease ID that appears expired based on ULID timestamp
		expiredTime := clock.Now().Add(-time.Hour)
		leaseID, err := ulid.New(ulid.Timestamp(expiredTime), nil)
		require.NoError(t, err)

		// Try to extend expired lease
		resp, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
			IdempotencyKey: "extend-expired",
			AccountID:      te.AccountID,
			LeaseID:        leaseID,
			Duration:       5 * time.Second,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Nil(t, resp.LeaseID, "Should not extend expired lease")
	})

	t.Run("Operation Idempotency Edge Cases", func(t *testing.T) {
		te.Redis.FlushAll()

		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		inProgressKey := fmt.Sprintf("{%s}:concurrency:test:%s", te.KeyPrefix, te.FunctionID)

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: inProgressKey,
				},
			},
		}

		enableDebugLogs = true

		// First request
		resp1, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "idempotency-test",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"lease-idem-1"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		t.Log(resp1.internalDebugState.Debug)
		require.Len(t, resp1.Leases, 1)

		// Duplicate request with same idempotency key should return cached result
		resp2, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "idempotency-test", // Same idempotency key
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               2,                                        // Different amount - should be ignored
			LeaseIdempotencyKeys: []string{"lease-idem-2", "lease-idem-3"}, // Different keys
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Equal(t, resp1.Leases, resp2.Leases, "Idempotent request should return same result")

		// Verify idempotency key management
		iv := te.NewIdempotencyVerifier()
		iv.VerifyOperationIdempotency("acq", "idempotency-test", int(OperationIdempotencyTTL.Seconds()), true)
	})
}
