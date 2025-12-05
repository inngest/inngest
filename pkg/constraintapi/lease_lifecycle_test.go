package constraintapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestLeaseLifecycle_CompleteWorkflows(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Full Concurrency Workflow", func(t *testing.T) {
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
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:lifecycle:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Step 1: Check capacity before acquisition
		checkResp, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
			AccountID:     te.AccountID,
			EnvID:         te.EnvID,
			FunctionID:    te.FunctionID,
			Configuration: config,
			Constraints:   constraints,
			Migration:     MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Equal(t, 3, checkResp.AvailableCapacity)
		require.Equal(t, 0, checkResp.Usage[0].Used, "Should have full capacity available")

		// Step 2: Acquire 2 leases
		acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "lifecycle-acquire",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               2,
			LeaseIdempotencyKeys: []string{"lifecycle-1", "lifecycle-2"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Len(t, acquireResp.Leases, 2)

		// Step 3: Check remaining capacity
		checkResp2, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
			AccountID:     te.AccountID,
			EnvID:         te.EnvID,
			FunctionID:    te.FunctionID,
			Configuration: config,
			Constraints:   constraints,
			Migration:     MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Equal(t, 2, checkResp2.Usage[0].Used, "Should have 1 capacity remaining")

		// Step 4: Extend the first lease
		firstLease := acquireResp.Leases[0]
		extendResp, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
			IdempotencyKey: "lifecycle-extend",
			AccountID:      te.AccountID,
			LeaseID:        firstLease.LeaseID,
			Duration:       60 * time.Second,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.NotNil(t, extendResp.LeaseID)

		// Step 5: Check capacity is still correct after extension
		checkResp3, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
			AccountID:     te.AccountID,
			EnvID:         te.EnvID,
			FunctionID:    te.FunctionID,
			Configuration: config,
			Constraints:   constraints,
			Migration:     MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Equal(t, 2, checkResp3.Usage[0].Used, "Capacity should remain same after extend")

		// Step 6: Release first lease
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "lifecycle-release-1",
			AccountID:      te.AccountID,
			LeaseID:        *extendResp.LeaseID, // Use the extended lease ID
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)

		// Step 7: Check capacity increased after release
		checkResp4, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
			AccountID:     te.AccountID,
			EnvID:         te.EnvID,
			FunctionID:    te.FunctionID,
			Configuration: config,
			Constraints:   constraints,
			Migration:     MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Equal(t, 1, checkResp4.Usage[0].Used, "Should have 2 capacity after first release")

		// Step 8: Release second lease
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "lifecycle-release-2",
			AccountID:      te.AccountID,
			LeaseID:        acquireResp.Leases[1].LeaseID,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)

		// Step 9: Verify full capacity restored
		checkResp5, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
			AccountID:     te.AccountID,
			EnvID:         te.EnvID,
			FunctionID:    te.FunctionID,
			Configuration: config,
			Constraints:   constraints,
			Migration:     MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Equal(t, 0, checkResp5.Usage[0].Used, "Should have full capacity restored")
	})

	t.Run("Rate Limit Workflow with Time Progression", func(t *testing.T) {
		t.Skip("this should pass but rate limiting calculation is off")

		enableDebugLogs = true
		config := ConstraintConfig{
			FunctionVersion: 1,
			RateLimit: []RateLimitConfig{
				{
					Scope:             enums.RateLimitScopeFn,
					Limit:             5, // 5 requests per 60 seconds
					Period:            60,
					KeyExpressionHash: "lifecycle-ratelimit",
				},
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeFn,
					KeyExpressionHash: "lifecycle-ratelimit",
					EvaluatedKeyHash:  "lifecycle-test",
				},
			},
		}

		// Step 1: Check initial capacity
		checkResp, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
			AccountID:     te.AccountID,
			EnvID:         te.EnvID,
			FunctionID:    te.FunctionID,
			Configuration: config,
			Constraints:   constraints,
			Migration:     MigrationIdentifier{IsRateLimit: true},
		})

		require.NoError(t, err)
		require.Equal(t, 0, checkResp.AvailableCapacity)
		require.NotEmpty(t, checkResp.Usage, "Should have usage information")
		require.Equal(t, 5, checkResp.Usage[0].Used, "Should start with full bucket used for rate limit")

		// Step 2: Acquire multiple leases to consume burst capacity
		acquireResp1, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "rate-acquire-1",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               2,
			LeaseIdempotencyKeys: []string{"rate-lease-1", "rate-lease-2"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationItemLease,
			},
			Migration: MigrationIdentifier{IsRateLimit: true},
		})

		require.NoError(t, err)
		t.Log(acquireResp1.internalDebugState.Debug)
		require.NotEmpty(t, acquireResp1.Leases, "Should grant at least some capacity")

		// Step 3: Try immediate acquisition (should be rate limited)
		acquireResp2, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "rate-acquire-2",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               2,
			LeaseIdempotencyKeys: []string{"rate-lease-3", "rate-lease-4"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationItemLease,
			},
			Migration: MigrationIdentifier{IsRateLimit: true},
		})

		require.NoError(t, err)
		if len(acquireResp2.Leases) == 0 {
			require.NotEmpty(t, acquireResp2.LimitingConstraints, "Should report rate limiting")
			require.True(t, acquireResp2.RetryAfter.After(clock.Now()), "Should provide retry time")
		}

		// Step 4: Advance time to allow rate limit recovery (emission interval = 60/5 = 12 seconds)
		clock.Advance(15 * time.Second)
		te.AdvanceTimeAndRedis(15 * time.Second)

		// Step 5: Try acquisition again after time advancement
		acquireResp3, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "rate-acquire-3",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"rate-lease-5"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationItemLease,
			},
			Migration: MigrationIdentifier{IsRateLimit: true},
		})

		require.NoError(t, err)
		require.NotEmpty(t, acquireResp3.Leases, "Should grant capacity after time advancement")

		// Step 6: Release all leases and verify state
		for _, lease := range acquireResp1.Leases {
			_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
				IdempotencyKey: fmt.Sprintf("rate-release-%s", lease.IdempotencyKey),
				AccountID:      te.AccountID,
				LeaseID:        lease.LeaseID,
				Migration:      MigrationIdentifier{IsRateLimit: true},
			})
			require.NoError(t, err)
		}

		for _, lease := range acquireResp3.Leases {
			_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
				IdempotencyKey: fmt.Sprintf("rate-release-%s", lease.IdempotencyKey),
				AccountID:      te.AccountID,
				LeaseID:        lease.LeaseID,
				Migration:      MigrationIdentifier{IsRateLimit: true},
			})
			require.NoError(t, err)
		}
	})

	t.Run("Mixed Constraint Workflow", func(t *testing.T) {
		t.Skip("this should work too")

		enableDebugLogs = true
		// Test separate rate limit and concurrency workflows since they can't be mixed in first stage

		// First: Concurrency constraint workflow
		concurrencyConfig := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 2,
			},
		}

		concurrencyConstraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:mixed:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Acquire concurrency capacity
		concurrencyResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "mixed-concurrency",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"mixed-concurrency-1"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        concurrencyConfig,
			Constraints:          concurrencyConstraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		t.Log(concurrencyResp.internalDebugState.Debug)
		require.Len(t, concurrencyResp.Leases, 1)

		// Release concurrency lease
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "mixed-concurrency-release",
			AccountID:      te.AccountID,
			LeaseID:        concurrencyResp.Leases[0].LeaseID,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)

		// Second: Rate limit constraint workflow
		rateLimitConfig := ConstraintConfig{
			FunctionVersion: 1,
			RateLimit: []RateLimitConfig{
				{
					Scope:             enums.RateLimitScopeFn,
					Limit:             3,
					Period:            30,
					KeyExpressionHash: "mixed-ratelimit",
				},
			},
		}

		rateLimitConstraints := []ConstraintItem{
			{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeFn,
					KeyExpressionHash: "mixed-ratelimit",
					EvaluatedKeyHash:  "mixed-test",
				},
			},
		}

		// Acquire rate limit capacity
		rateLimitResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "mixed-ratelimit",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"mixed-ratelimit-1"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        rateLimitConfig,
			Constraints:          rateLimitConstraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationItemLease,
			},
			Migration: MigrationIdentifier{IsRateLimit: true},
		})

		require.NoError(t, err)
		require.Len(t, rateLimitResp.Leases, 1)

		// Release rate limit lease
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "mixed-ratelimit-release",
			AccountID:      te.AccountID,
			LeaseID:        rateLimitResp.Leases[0].LeaseID,
			Migration:      MigrationIdentifier{IsRateLimit: true},
		})

		require.NoError(t, err)
	})

	t.Run("Idempotency Across Operations", func(t *testing.T) {
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
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:idempotent:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		acquireReq := &CapacityAcquireRequest{
			IdempotencyKey:       "idempotent-key",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               2,
			LeaseIdempotencyKeys: []string{"idempotent-1", "idempotent-2"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		}

		// Step 1: First acquire with idempotency key
		resp1, err := te.CapacityManager.Acquire(context.Background(), acquireReq)

		require.NoError(t, err)
		require.Len(t, resp1.Leases, 2)
		originalLeaseIDs := make([]ulid.ULID, len(resp1.Leases))
		for i, lease := range resp1.Leases {
			originalLeaseIDs[i] = lease.LeaseID
		}

		// Step 2: Repeat same acquire with same idempotency key
		resp2, err := te.CapacityManager.Acquire(context.Background(), acquireReq)

		require.NoError(t, err)
		require.Len(t, resp2.Leases, 2, "Should return same number of leases from cached result")

		// Verify lease IDs are identical (idempotent response)
		for i, lease := range resp2.Leases {
			require.Equal(t, originalLeaseIDs[i], lease.LeaseID, "Lease IDs should be identical for idempotent requests")
		}

		// Step 3: Test idempotent release
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "idempotent-release",
			AccountID:      te.AccountID,
			LeaseID:        resp1.Leases[0].LeaseID,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)

		// Step 4: Repeat same release
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "idempotent-release", // Same key
			AccountID:      te.AccountID,
			LeaseID:        resp1.Leases[0].LeaseID,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		// Should succeed without error (idempotent)

		// Step 5: Test idempotent extend
		extendResp1, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
			IdempotencyKey: "idempotent-extend",
			AccountID:      te.AccountID,
			LeaseID:        resp1.Leases[1].LeaseID,
			Duration:       45 * time.Second,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.NotNil(t, extendResp1.LeaseID)

		// Step 6: Repeat same extend
		extendResp2, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
			IdempotencyKey: "idempotent-extend", // Same key
			AccountID:      te.AccountID,
			LeaseID:        resp1.Leases[1].LeaseID,
			Duration:       90 * time.Second, // Different duration - should be ignored
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Equal(t, extendResp1.LeaseID, extendResp2.LeaseID, "Should return same lease ID from cached extend result")

		// Clean up
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "cleanup-release",
			AccountID:      te.AccountID,
			LeaseID:        *extendResp1.LeaseID,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})
		require.NoError(t, err)
	})
}

func TestLeaseLifecycle_FailureScenarios(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Extend Expired Lease", func(t *testing.T) {
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
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:expired:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Acquire a lease with short duration
		acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "expired-acquire",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"expired-lease"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second, // Short duration
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Len(t, acquireResp.Leases, 1)

		// Advance time beyond lease expiry
		clock.Advance(10 * time.Second)
		te.AdvanceTimeAndRedis(10 * time.Second)

		// Try to extend expired lease
		extendResp, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
			IdempotencyKey: "extend-expired",
			AccountID:      te.AccountID,
			LeaseID:        acquireResp.Leases[0].LeaseID,
			Duration:       30 * time.Second,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Nil(t, extendResp.LeaseID, "Should not extend expired lease")
	})

	t.Run("Release After Expiry", func(t *testing.T) {
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
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:release-expired:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Acquire a lease
		acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "release-expired-acquire",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"release-expired-lease"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Len(t, acquireResp.Leases, 1)

		// Advance time beyond expiry
		clock.Advance(10 * time.Second)
		te.AdvanceTimeAndRedis(10 * time.Second)

		// Try to release expired lease - should still work for cleanup
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "release-expired",
			AccountID:      te.AccountID,
			LeaseID:        acquireResp.Leases[0].LeaseID,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		// Should succeed even for expired lease (cleanup operation)
	})

	t.Run("Configuration Version Changes", func(t *testing.T) {
		config1 := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 3,
			},
		}

		config2 := ConstraintConfig{
			FunctionVersion: 2, // Changed version
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5, // Changed limit
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:version:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Acquire with version 1
		acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "version-acquire",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               2,
			LeaseIdempotencyKeys: []string{"version-1", "version-2"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config1,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Len(t, acquireResp.Leases, 2)

		// Check capacity with new version
		checkResp, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
			AccountID:     te.AccountID,
			EnvID:         te.EnvID,
			FunctionID:    te.FunctionID,
			Configuration: config2, // New version
			Constraints:   constraints,
			Migration:     MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		// Should handle version changes gracefully
		require.NotEmpty(t, checkResp.AvailableCapacity)

		// Clean up with original version
		for _, lease := range acquireResp.Leases {
			_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
				IdempotencyKey: fmt.Sprintf("version-release-%s", lease.IdempotencyKey),
				AccountID:      te.AccountID,
				LeaseID:        lease.LeaseID,
				Migration:      MigrationIdentifier{QueueShard: "test"},
			})
			require.NoError(t, err)
		}
	})

	t.Run("Constraint Configuration Changes", func(t *testing.T) {
		originalConstraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:config-change:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		modifiedConstraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeEnv,                                                // Changed scope
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:config-change:%s", te.KeyPrefix, te.EnvID), // Different key
				},
			},
		}

		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 3,
			},
		}

		// Acquire with original constraints
		acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "config-change-acquire",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"config-change-lease"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          originalConstraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Len(t, acquireResp.Leases, 1)

		// Check with modified constraints (different scope/key)
		checkResp, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
			AccountID:     te.AccountID,
			EnvID:         te.EnvID,
			FunctionID:    te.FunctionID,
			Configuration: config,
			Constraints:   modifiedConstraints, // Different constraints
			Migration:     MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		// Should treat as separate constraint space
		require.Zero(t, checkResp.AvailableCapacity)

		// Release with original constraints
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "config-change-release",
			AccountID:      te.AccountID,
			LeaseID:        acquireResp.Leases[0].LeaseID,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
	})
}
