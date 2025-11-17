package constraintapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestStateConsistency_LeaseOperations(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Acquire and Release Capacity Restoration", func(t *testing.T) {
		initialState := te.CaptureRedisState()

		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
			Throttle: []ThrottleConfig{
				{
					Limit:                     10,
					Period:                    60,
					ThrottleKeyExpressionHash: "consistency-test",
				},
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:test:%s", te.KeyPrefix, te.FunctionID),
				},
			},
			{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					KeyExpressionHash: "consistency-test",
					EvaluatedKeyHash:  "test-value",
				},
			},
		}
		var err error

		// Acquire multiple leases
		acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "consistency-acquire-1",
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
		require.Len(t, acquireResp.Leases, 3, "Should acquire 3 leases")

		afterAcquireState := te.CaptureRedisState()
		te.CompareRedisState(initialState, afterAcquireState, "After Acquire")

		// Verify constraint state after acquisition
		cv := te.NewConstraintVerifier()
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 3}) // 3 leases in progress

		// Verify account leases are tracked
		leaseIDs := make([]ulid.ULID, len(acquireResp.Leases))
		for i, lease := range acquireResp.Leases {
			leaseIDs[i] = lease.LeaseID
		}
		cv.VerifyAccountLeases(leaseIDs)

		// Verify scavenger shard is updated (ULID contains expiry time)
		cv.VerifyScavengerShard(float64(ulid.Time(acquireResp.Leases[0].LeaseID.Time()).UnixMilli()), true)

		// Release all leases
		for i, lease := range acquireResp.Leases {
			_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
				IdempotencyKey: fmt.Sprintf("consistency-release-%d", i+1),
				AccountID:      te.AccountID,
				LeaseID:        lease.LeaseID,
				Migration:      MigrationIdentifier{QueueShard: "test"},
			})

			require.NoError(t, err)
		}

		finalState := te.CaptureRedisState()
		te.CompareRedisState(afterAcquireState, finalState, "After Release")

		_, err = te.Redis.Get(te.CapacityManager.keyRequestState(te.KeyPrefix, te.AccountID, "consistency-acquire-1"))
		require.Error(t, err)

		// Verify all capacity is restored - only idempotency keys should remain
		expectedRemainingKeys := []string{
			te.CapacityManager.keyOperationIdempotency(te.KeyPrefix, te.AccountID, "acq", "consistency-acquire-1"),
			te.CapacityManager.keyConstraintCheckIdempotency(te.KeyPrefix, te.AccountID, "consistency-acquire-1"),
		}
		for i := 1; i <= 3; i++ {
			expectedRemainingKeys = append(expectedRemainingKeys,
				te.CapacityManager.keyOperationIdempotency(te.KeyPrefix, te.AccountID, "rel", fmt.Sprintf("consistency-release-%d", i)))
		}

		// NOTE: Throttle keys are stored _without_ prefix
		expectedRemainingKeys = append(expectedRemainingKeys, "consistency-test")

		te.VerifyNoResourceLeaks(initialState, expectedRemainingKeys)

		// Verify no in-progress items remain
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 0})

		// Verify account leases are cleaned up
		cv.VerifyAccountLeases(nil)

		// Verify scavenger shard is cleaned up when no leases remain
		cv.VerifyScavengerShard(0, false)
	})

	t.Run("Partial Acquisition State Consistency", func(t *testing.T) {
		te.Redis.FlushAll()

		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 2, // Very low limit to force partial acquisition
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:partial:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Try to acquire more than available capacity
		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "partial-acquire",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               5, // Request more than available (2)
			LeaseIdempotencyKeys: []string{"lease-1", "lease-2", "lease-3", "lease-4", "lease-5"},
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
		require.Len(t, resp.Leases, 2, "Should only grant available capacity")
		require.NotEmpty(t, resp.LimitingConstraints, "Should report limiting constraints")

		// Verify state consistency - only granted leases should be tracked
		cv := te.NewConstraintVerifier()
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 2})

		// Verify only granted leases are in account tracking
		leaseIDs := make([]ulid.ULID, len(resp.Leases))
		for i, lease := range resp.Leases {
			leaseIDs[i] = lease.LeaseID
		}
		cv.VerifyAccountLeases(leaseIDs)

		// Clean up
		for i, lease := range resp.Leases {
			_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
				IdempotencyKey: fmt.Sprintf("partial-release-%d", i+1),
				AccountID:      te.AccountID,
				LeaseID:        lease.LeaseID,
				Migration:      MigrationIdentifier{QueueShard: "test"},
			})
			require.NoError(t, err)
		}

		// Verify complete cleanup
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 0})
		cv.VerifyAccountLeases(nil)
	})

	t.Run("Extend Lease State Consistency", func(t *testing.T) {
		te.Redis.FlushAll()

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
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:extend:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		var err error

		// Acquire a lease
		acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "extend-acquire",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"extend-lease-1"},
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
		require.Len(t, acquireResp.Leases, 1)

		originalLease := acquireResp.Leases[0]
		originalExpiry := originalLease.LeaseID.Timestamp()

		cv := te.NewConstraintVerifier()
		cv.VerifyScavengerShard(float64(originalExpiry.UnixMilli()), true)

		accountScore, err := te.Redis.ZScore(te.CapacityManager.keyScavengerShard(te.KeyPrefix, 0), te.AccountID.String())
		require.NoError(t, err)

		require.Equal(t, float64(originalExpiry.UnixMilli()), accountScore)

		// Capture state before extension
		beforeExtendState := te.CaptureRedisState()

		// Extend the lease
		extendResp, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
			IdempotencyKey: "extend-operation",
			AccountID:      te.AccountID,
			LeaseID:        originalLease.LeaseID,
			Duration:       10 * time.Second,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.NotNil(t, extendResp.LeaseID)
		require.NotEqual(t, originalLease.LeaseID, *extendResp.LeaseID)
		require.NotEqual(t, originalExpiry, extendResp.LeaseID.Timestamp())

		afterExtendState := te.CaptureRedisState()
		te.CompareRedisState(beforeExtendState, afterExtendState, "After Extend")

		// Verify lease details are updated but capacity count remains the same
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 1}) // Still 1 lease

		// Verify account leases track the new lease (with updated expiry)
		cv.VerifyAccountLeases([]ulid.ULID{*extendResp.LeaseID})

		t.Log(extendResp.LeaseID.Timestamp().UnixMilli())

		leaseScore, err := te.Redis.ZScore(te.CapacityManager.keyAccountLeases(te.KeyPrefix, te.AccountID), extendResp.LeaseID.String())
		require.NoError(t, err)
		require.Equal(t, float64(extendResp.LeaseID.Timestamp().UnixMilli()), leaseScore)

		accountScore, err = te.Redis.ZScore(te.CapacityManager.keyScavengerShard(te.KeyPrefix, 0), te.AccountID.String())
		require.NoError(t, err)
		require.Equal(t, float64(extendResp.LeaseID.Timestamp().UnixMilli()), accountScore)

		t.Log(te.Redis.Dump())

		// Verify scavenger shard score is updated with new expiry
		cv.VerifyScavengerShard(float64(extendResp.LeaseID.Timestamp().UnixMilli()), true)

		// Verify lease details contain extension information
		cv.VerifyLeaseDetails(
			*extendResp.LeaseID,
			"extend-lease-1",
			"",
			util.XXHash("extend-acquire"),
		)

		enableDebugLogs = true

		// Clean up
		resp, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "extend-release",
			AccountID:      te.AccountID,
			LeaseID:        *extendResp.LeaseID,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})
		require.NoError(t, err)

		t.Log(resp.internalDebugState.Debug)

		// Verify complete cleanup
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 0})
		cv.VerifyAccountLeases(nil)
		cv.VerifyScavengerShard(0, false)
	})

	t.Run("Idempotency Key TTL Consistency", func(t *testing.T) {
		te.Redis.FlushAll()

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
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:idempotency:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Perform acquire operation
		acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "ttl-test",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"ttl-lease-1"},
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
		require.Len(t, acquireResp.Leases, 1)

		// Verify idempotency keys are properly set with TTL
		iv := te.NewIdempotencyVerifier()
		iv.VerifyOperationIdempotency("acq", "ttl-test", int(OperationIdempotencyTTL.Seconds()), true)

		// Perform release operation
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "ttl-release",
			AccountID:      te.AccountID,
			LeaseID:        acquireResp.Leases[0].LeaseID,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)

		// Verify release idempotency key is set
		iv.VerifyOperationIdempotency("rel", "ttl-release", int(OperationIdempotencyTTL.Seconds()), true)

		// Simulate TTL expiration by advancing time and clearing Redis TTLs
		te.AdvanceTimeAndRedis(OperationIdempotencyTTL + time.Second)

		// Verify idempotency keys are cleaned up
		iv.VerifyOperationIdempotency("acq", "ttl-test", 0, false)
		iv.VerifyOperationIdempotency("rel", "ttl-release", 0, false)

		// Verify no other resource leaks exist after TTL cleanup
		finalState := te.CaptureRedisState()
		require.Equal(t, finalState.Keys,
			[]string{
				te.CapacityManager.keyConstraintCheckIdempotency(te.KeyPrefix, te.AccountID, "ttl-test"),
			},
			"No keys should remain after TTL cleanup")
	})

	t.Run("Multi-Constraint State Consistency", func(t *testing.T) {
		te.Redis.FlushAll()

		initialState := te.CaptureRedisState()

		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 3,
			},
			Throttle: []ThrottleConfig{
				{
					Limit:                     5,
					Period:                    60,
					ThrottleKeyExpressionHash: "throttle-expr",
				},
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:multi:%s", te.KeyPrefix, te.FunctionID),
				},
			},
			{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					KeyExpressionHash: "throttle-expr",
					EvaluatedKeyHash:  "multi-value",
				},
			},
		}

		// Acquire capacity affecting both constraints
		acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "multi-acquire",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               2,
			LeaseIdempotencyKeys: []string{"multi-lease-1", "multi-lease-2"},
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
		require.Len(t, acquireResp.Leases, 2)

		// Verify both constraint states are updated
		cv := te.NewConstraintVerifier()
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 2}) // 2 leases in concurrency

		// Release leases and verify both constraints are properly cleaned up
		for i, lease := range acquireResp.Leases {
			_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
				IdempotencyKey: fmt.Sprintf("multi-release-%d", i+1),
				AccountID:      te.AccountID,
				LeaseID:        lease.LeaseID,
				Migration:      MigrationIdentifier{QueueShard: "test"},
			})
			require.NoError(t, err)
		}

		// Verify concurrency constraint is cleaned up
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 0})

		// Verify only expected keys remain (rate limit state + idempotency keys)
		expectedRemainingKeys := []string{
			te.CapacityManager.keyOperationIdempotency(te.CapacityManager.queueStateKeyPrefix, te.AccountID, "acq", "multi-acquire"),
			te.CapacityManager.keyOperationIdempotency(te.CapacityManager.queueStateKeyPrefix, te.AccountID, "rel", "multi-release-1"),
			te.CapacityManager.keyOperationIdempotency(te.CapacityManager.queueStateKeyPrefix, te.AccountID, "rel", "multi-release-2"),
			te.CapacityManager.keyConstraintCheckIdempotency(te.CapacityManager.queueStateKeyPrefix, te.AccountID, "multi-acquire"),
			"multi-value", // throttle key
		}

		te.VerifyNoResourceLeaks(initialState, expectedRemainingKeys)
	})
}

func TestStateConsistency_ErrorRecovery(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Release Non-Existent Lease Cleanup", func(t *testing.T) {
		initialState := te.CaptureRedisState()

		nonExistentLeaseID := ulid.Make()

		// Attempt to release a lease that doesn't exist
		resp, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "release-nonexistent",
			AccountID:      te.AccountID,
			LeaseID:        nonExistentLeaseID,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		_ = resp // Suppress unused variable warning
		// Note: CapacityReleaseResponse doesn't return LeaseID in current implementation
		// The operation is considered successful if no error is returned

		// Verify only idempotency key is created, no other state changes
		expectedKeys := []string{
			te.CapacityManager.keyOperationIdempotency(te.KeyPrefix, te.AccountID, "rel", "release-nonexistent"),
		}

		te.VerifyNoResourceLeaks(initialState, expectedKeys)
	})

	t.Run("Extend Non-Existent Lease Cleanup", func(t *testing.T) {
		initialState := te.CaptureRedisState()

		nonExistentLeaseID := ulid.Make()

		// Attempt to extend a lease that doesn't exist
		resp, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
			IdempotencyKey: "extend-nonexistent",
			AccountID:      te.AccountID,
			LeaseID:        nonExistentLeaseID,
			Duration:       10 * time.Second,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Nil(t, resp.LeaseID, "Should return nil for non-existent lease")

		// Verify only idempotency key is created
		expectedKeys := []string{
			te.CapacityManager.keyOperationIdempotency(te.KeyPrefix, te.AccountID, "ext", "extend-nonexistent"),
		}

		te.VerifyNoResourceLeaks(initialState, expectedKeys)
	})
}
