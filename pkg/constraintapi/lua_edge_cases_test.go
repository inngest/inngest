package constraintapi

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestLuaScriptEdgeCases_Concurrency(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	// NOTE: "In-Progress Items vs Leases Reconciliation" test removed
	// This test was checking backward compatibility with legacy queue-based concurrency state
	// using the old key format {q}:concurrency:items:{functionID}. Since we're doing a hard
	// migration to the new constraint API, this legacy compatibility is no longer needed.

	t.Run("Expired Lease Detection", func(t *testing.T) {
		te.Redis.FlushAll()

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
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		}

		inProgressLeasesKey := constraints[0].Concurrency.InProgressLeasesKey(te.AccountID, te.EnvID, te.FunctionID)

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
				Location: CallerLocationItemLease,
			},
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
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
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
				Location: CallerLocationItemLease,
			},
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
					Scope:             enums.ThrottleScopeFn,
					Limit:             1000000, // Very high limit
					Burst:             100000,  // High burst
					Period:            1,       // 1 second period
					KeyExpressionHash: "small-interval",
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
				Location: CallerLocationItemLease,
			},
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
					Scope:             enums.ThrottleScopeFn,
					Limit:             1,
					Burst:             0,
					Period:            86400, // 24 hours
					KeyExpressionHash: "large-period",
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
				Location: CallerLocationItemLease,
			},
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
				Location: CallerLocationItemLease,
			},
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
					Scope:             enums.ThrottleScopeFn,
					Limit:             0, // No throughput allowed
					Burst:             0,
					Period:            60,
					KeyExpressionHash: "zero-throttle",
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
				Location: CallerLocationItemLease,
			},
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
		reqID := ulid.MustNew(ulid.Timestamp(te.CapacityManager.clock.Now()), rand.Reader)
		// Pre-populate invalid request state
		requestStateKey := te.CapacityManager.keyRequestState(te.AccountID, reqID)
		err := te.Redis.Set(requestStateKey, "invalid-json-data")
		require.NoError(t, err)
		leaseID := ulid.Make()
		te.Redis.HSet(
			te.CapacityManager.keyLeaseDetails(te.AccountID, leaseID),
			"req", reqID.String(),
			"lik", util.XXHash("acquire-key"),
			"rid", "",
		)

		// Try to extend lease which will try to read the corrupted state
		_, err = te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
			IdempotencyKey: "extend-invalid",
			AccountID:      te.AccountID,
			LeaseID:        leaseID,
			Duration:       5 * time.Second,
		})
		require.ErrorContains(t, err, "requestDetails is nil after JSON decode")
		require.Error(t, err)
	})

	t.Run("Missing Lease Details", func(t *testing.T) {
		leaseID := ulid.Make()

		// Try to extend a lease that doesn't exist
		resp, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
			IdempotencyKey: "extend-missing",
			AccountID:      te.AccountID,
			LeaseID:        leaseID,
			Duration:       5 * time.Second,
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

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		}

		enableDebugLogs = true

		req := &CapacityAcquireRequest{
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
				Location: CallerLocationItemLease,
			},
		}

		_, _, _, fingerprint, err := buildRequestState(req)
		require.NoError(t, err)

		acquireIdempotencyKey := fmt.Sprintf("idempotency-test-%s", fingerprint)

		// First request
		resp1, err := te.CapacityManager.Acquire(context.Background(), req)

		require.NoError(t, err)
		t.Log(resp1.internalDebugState.Debug)
		require.Len(t, resp1.Leases, 1)

		// Duplicate request with same idempotency key should return cached result
		resp2, err := te.CapacityManager.Acquire(context.Background(), req)

		require.NoError(t, err)
		require.Equal(t, resp1.Leases, resp2.Leases, "Idempotent request should return same result")

		// Verify idempotency key management
		iv := te.NewIdempotencyVerifier()
		iv.VerifyOperationIdempotency("acq", acquireIdempotencyKey, int(OperationIdempotencyTTL.Seconds()), true)
	})
}
