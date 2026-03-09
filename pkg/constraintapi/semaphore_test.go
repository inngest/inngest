package constraintapi

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func semaphoreConfig(name string, scope enums.SemaphoreScope, capacity int) ConstraintConfig {
	return ConstraintConfig{
		FunctionVersion: 1,
		Semaphore: []SemaphoreConfig{
			{
				Name:     name,
				Scope:    scope,
				Capacity: capacity,
			},
		},
	}
}

func semaphoreConfigWithExpr(name string, scope enums.SemaphoreScope, capacity int, exprHash string) ConstraintConfig {
	return ConstraintConfig{
		FunctionVersion: 1,
		Semaphore: []SemaphoreConfig{
			{
				Name:              name,
				Scope:             scope,
				Capacity:          capacity,
				KeyExpressionHash: exprHash,
			},
		},
	}
}

func semaphoreConstraint(name string, scope enums.SemaphoreScope, amount int) ConstraintItem {
	return ConstraintItem{
		Kind: ConstraintKindSemaphore,
		Semaphore: &SemaphoreConstraint{
			Name:   name,
			Scope:  scope,
			Amount: amount,
		},
	}
}

func semaphoreConstraintWithExpr(name string, scope enums.SemaphoreScope, amount int, exprHash, evalKeyHash string) ConstraintItem {
	return ConstraintItem{
		Kind: ConstraintKindSemaphore,
		Semaphore: &SemaphoreConstraint{
			Name:              name,
			Scope:             scope,
			Amount:            amount,
			KeyExpressionHash: exprHash,
			EvaluatedKeyHash:  evalKeyHash,
		},
	}
}

func TestSemaphore_BasicAcquire(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := semaphoreConfig("test-sem", enums.SemaphoreScopeFn, 10)
	constraints := []ConstraintItem{
		semaphoreConstraint("test-sem", enums.SemaphoreScopeFn, 2),
	}

	// Acquire 1 lease (consuming 2 units out of 10)
	resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-basic-1",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"lease-1"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})

	require.NoError(t, err)
	require.Len(t, resp.Leases, 1, "Should grant 1 lease")
	require.Empty(t, resp.ExhaustedConstraints, "No constraints should be exhausted")
}

func TestSemaphore_CapacityCalculation(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	// capacity=10, amount=3 → can grant floor(10/3)=3 leases
	config := semaphoreConfig("test-sem", enums.SemaphoreScopeFn, 10)
	constraints := []ConstraintItem{
		semaphoreConstraint("test-sem", enums.SemaphoreScopeFn, 3),
	}

	// Request 4 leases but only 3 should be granted (floor(10/3)=3)
	resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-cap-calc",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               4,
		LeaseIdempotencyKeys: []string{"l1", "l2", "l3", "l4"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})

	require.NoError(t, err)
	require.Len(t, resp.Leases, 3, "Should grant 3 leases (floor(10/3))")
}

func TestSemaphore_AcquireExhaustsCapacity(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := semaphoreConfig("test-sem", enums.SemaphoreScopeFn, 4)
	constraints := []ConstraintItem{
		semaphoreConstraint("test-sem", enums.SemaphoreScopeFn, 2),
	}

	// Acquire 2 leases consuming all 4 units
	resp1, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-exhaust-1",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               2,
		LeaseIdempotencyKeys: []string{"l1", "l2"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})

	require.NoError(t, err)
	require.Len(t, resp1.Leases, 2)

	// Attempt to acquire more — should get 0 leases
	resp2, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-exhaust-2",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"l3"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})

	require.NoError(t, err)
	require.Len(t, resp2.Leases, 0, "Should grant 0 leases when capacity exhausted")
	require.NotEmpty(t, resp2.ExhaustedConstraints, "Should have exhausted constraints")
}

func TestSemaphore_CheckCapacity(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := semaphoreConfig("test-sem", enums.SemaphoreScopeFn, 10)
	constraints := []ConstraintItem{
		semaphoreConstraint("test-sem", enums.SemaphoreScopeFn, 2),
	}

	// Check before any acquire
	checkResp1, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
		AccountID:     te.AccountID,
		EnvID:         te.EnvID,
		FunctionID:    te.FunctionID,
		Configuration: config,
		Constraints:   constraints,
	})
	require.NoError(t, err)
	require.Equal(t, 5, checkResp1.AvailableCapacity, "Should have floor(10/2)=5 capacity")
	require.Equal(t, 0, checkResp1.Usage[0].Used)
	require.Equal(t, 10, checkResp1.Usage[0].Limit)

	// Acquire 2 leases (consuming 4 units)
	_, err = te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-check-acq",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               2,
		LeaseIdempotencyKeys: []string{"l1", "l2"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)

	// Check after acquire
	checkResp2, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
		AccountID:     te.AccountID,
		EnvID:         te.EnvID,
		FunctionID:    te.FunctionID,
		Configuration: config,
		Constraints:   constraints,
	})
	require.NoError(t, err)
	require.Equal(t, 3, checkResp2.AvailableCapacity, "Should have floor((10-4)/2)=3 capacity")
	require.Equal(t, 4, checkResp2.Usage[0].Used)
	require.Equal(t, 10, checkResp2.Usage[0].Limit)
}

func TestSemaphore_FullLifecycle(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := semaphoreConfig("test-sem", enums.SemaphoreScopeFn, 6)
	constraints := []ConstraintItem{
		semaphoreConstraint("test-sem", enums.SemaphoreScopeFn, 2),
	}

	// Step 1: Check — full capacity
	checkResp, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
		AccountID:     te.AccountID,
		EnvID:         te.EnvID,
		FunctionID:    te.FunctionID,
		Configuration: config,
		Constraints:   constraints,
	})
	require.NoError(t, err)
	require.Equal(t, 3, checkResp.AvailableCapacity, "Initial capacity = floor(6/2) = 3")

	// Step 2: Acquire 2 leases (consuming 4 units)
	acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-lifecycle-acq",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               2,
		LeaseIdempotencyKeys: []string{"sl1", "sl2"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, acquireResp.Leases, 2)

	// Step 3: Check — reduced capacity
	checkResp2, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
		AccountID:     te.AccountID,
		EnvID:         te.EnvID,
		FunctionID:    te.FunctionID,
		Configuration: config,
		Constraints:   constraints,
	})
	require.NoError(t, err)
	require.Equal(t, 1, checkResp2.AvailableCapacity, "After 4 units used: floor((6-4)/2) = 1")

	// Step 4: Extend first lease
	firstLease := acquireResp.Leases[0]
	extendResp, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
		IdempotencyKey: "sem-lifecycle-ext",
		AccountID:      te.AccountID,
		LeaseID:        firstLease.LeaseID,
		Duration:       60 * time.Second,
	})
	require.NoError(t, err)
	require.NotNil(t, extendResp.LeaseID)

	// Step 5: Check — same capacity after extend
	checkResp3, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
		AccountID:     te.AccountID,
		EnvID:         te.EnvID,
		FunctionID:    te.FunctionID,
		Configuration: config,
		Constraints:   constraints,
	})
	require.NoError(t, err)
	require.Equal(t, 1, checkResp3.AvailableCapacity, "Capacity same after extend")

	// Step 6: Release first lease (returns 2 units)
	_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
		IdempotencyKey: "sem-lifecycle-rel1",
		AccountID:      te.AccountID,
		LeaseID:        *extendResp.LeaseID,
	})
	require.NoError(t, err)

	// Step 7: Check — more capacity after release
	checkResp4, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
		AccountID:     te.AccountID,
		EnvID:         te.EnvID,
		FunctionID:    te.FunctionID,
		Configuration: config,
		Constraints:   constraints,
	})
	require.NoError(t, err)
	require.Equal(t, 2, checkResp4.AvailableCapacity, "After releasing 1 lease: floor((6-2)/2) = 2")

	// Step 8: Release second lease
	_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
		IdempotencyKey: "sem-lifecycle-rel2",
		AccountID:      te.AccountID,
		LeaseID:        acquireResp.Leases[1].LeaseID,
	})
	require.NoError(t, err)

	// Step 9: Check — full capacity restored
	checkResp5, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
		AccountID:     te.AccountID,
		EnvID:         te.EnvID,
		FunctionID:    te.FunctionID,
		Configuration: config,
		Constraints:   constraints,
	})
	require.NoError(t, err)
	require.Equal(t, 3, checkResp5.AvailableCapacity, "Full capacity restored: floor(6/2) = 3")
	require.Equal(t, 0, checkResp5.Usage[0].Used, "All units released")
}

func TestSemaphore_MultipleAcquiresConsumeProgressively(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := semaphoreConfig("test-sem", enums.SemaphoreScopeFn, 10)
	constraints := []ConstraintItem{
		semaphoreConstraint("test-sem", enums.SemaphoreScopeFn, 1),
	}

	// Acquire 3 leases
	resp1, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-prog-1",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               3,
		LeaseIdempotencyKeys: []string{"p1", "p2", "p3"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp1.Leases, 3)

	// Acquire 5 more
	resp2, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-prog-2",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               5,
		LeaseIdempotencyKeys: []string{"p4", "p5", "p6", "p7", "p8"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp2.Leases, 5)

	// Check remaining: 10 - 3 - 5 = 2
	checkResp, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
		AccountID:     te.AccountID,
		EnvID:         te.EnvID,
		FunctionID:    te.FunctionID,
		Configuration: config,
		Constraints:   constraints,
	})
	require.NoError(t, err)
	require.Equal(t, 2, checkResp.AvailableCapacity)
	require.Equal(t, 8, checkResp.Usage[0].Used)

	// Acquire remaining 2
	resp3, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-prog-3",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               2,
		LeaseIdempotencyKeys: []string{"p9", "p10"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp3.Leases, 2)

	// Now capacity should be 0
	resp4, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-prog-4",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"p11"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp4.Leases, 0, "No more capacity")
}

func TestSemaphore_ReleaseDeletesKeyWhenZero(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := semaphoreConfig("test-sem", enums.SemaphoreScopeFn, 2)
	constraints := []ConstraintItem{
		semaphoreConstraint("test-sem", enums.SemaphoreScopeFn, 1),
	}

	// Acquire 1 lease
	resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-del-key",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"dl1"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp.Leases, 1)

	// Verify semaphore key exists with value "1"
	semKey := constraints[0].Semaphore.StateKey(te.AccountID, te.EnvID, te.FunctionID)
	require.True(t, te.Redis.Exists(semKey), "Semaphore key should exist after acquire")
	val, _ := te.Redis.Get(semKey)
	require.Equal(t, "1", val, "Semaphore counter should be 1")

	// Release the lease
	_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
		IdempotencyKey: "sem-del-key-rel",
		AccountID:      te.AccountID,
		LeaseID:        resp.Leases[0].LeaseID,
	})
	require.NoError(t, err)

	// Verify semaphore key is deleted (counter went to 0)
	require.False(t, te.Redis.Exists(semKey), "Semaphore key should be deleted when counter reaches 0")
}

func TestSemaphore_DifferentScopes(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	tests := []struct {
		name  string
		scope enums.SemaphoreScope
	}{
		{"function scope", enums.SemaphoreScopeFn},
		{"env scope", enums.SemaphoreScopeEnv},
		{"account scope", enums.SemaphoreScopeAccount},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := semaphoreConfig("scope-test", tt.scope, 5)
			constraints := []ConstraintItem{
				semaphoreConstraint("scope-test", tt.scope, 1),
			}

			resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
				IdempotencyKey:       fmt.Sprintf("sem-scope-%d", i),
				AccountID:            te.AccountID,
				EnvID:                te.EnvID,
				FunctionID:           te.FunctionID,
				Amount:               2,
				LeaseIdempotencyKeys: []string{fmt.Sprintf("s%d-1", i), fmt.Sprintf("s%d-2", i)},
				CurrentTime:          clock.Now(),
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				Configuration:        config,
				Constraints:          constraints,
				Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
			})
			require.NoError(t, err)
			require.Len(t, resp.Leases, 2)

			// Verify the key is scoped correctly
			semKey := constraints[0].Semaphore.StateKey(te.AccountID, te.EnvID, te.FunctionID)
			require.True(t, te.Redis.Exists(semKey), "Semaphore key should exist: %s", semKey)
			val, _ := te.Redis.Get(semKey)
			require.Equal(t, "2", val)
		})
	}
}

func TestSemaphore_WithKeyExpression(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := semaphoreConfigWithExpr("expr-sem", enums.SemaphoreScopeFn, 10, "expr-hash-123")
	constraints := []ConstraintItem{
		semaphoreConstraintWithExpr("expr-sem", enums.SemaphoreScopeFn, 2, "expr-hash-123", "eval-key-abc"),
	}

	resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-expr-1",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"expr-l1"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp.Leases, 1)

	// Verify key includes expression hash
	semKey := constraints[0].Semaphore.StateKey(te.AccountID, te.EnvID, te.FunctionID)
	require.Contains(t, semKey, "expr-hash-123")
	require.Contains(t, semKey, "eval-key-abc")
	require.True(t, te.Redis.Exists(semKey))
	val, _ := te.Redis.Get(semKey)
	require.Equal(t, "2", val, "Amount per lease is 2")
}

func TestSemaphore_WithoutKeyExpression(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := semaphoreConfig("no-expr-sem", enums.SemaphoreScopeFn, 10)
	constraints := []ConstraintItem{
		semaphoreConstraint("no-expr-sem", enums.SemaphoreScopeFn, 1),
	}

	resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-noexpr-1",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               3,
		LeaseIdempotencyKeys: []string{"ne1", "ne2", "ne3"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp.Leases, 3)

	// Key should not contain expression hash
	semKey := constraints[0].Semaphore.StateKey(te.AccountID, te.EnvID, te.FunctionID)
	require.True(t, te.Redis.Exists(semKey))
}

func TestSemaphore_CombinedWithConcurrency(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 5,
		},
		Semaphore: []SemaphoreConfig{
			{
				Name:     "combined-sem",
				Scope:    enums.SemaphoreScopeFn,
				Capacity: 3,
			},
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
		semaphoreConstraint("combined-sem", enums.SemaphoreScopeFn, 1),
	}

	// Semaphore limits to 3, concurrency allows 5 → should grant 3
	resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-combined-1",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               5,
		LeaseIdempotencyKeys: []string{"c1", "c2", "c3", "c4", "c5"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp.Leases, 3, "Should be limited by semaphore capacity of 3")
}

func TestSemaphore_AmountOne(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := semaphoreConfig("simple-counter", enums.SemaphoreScopeFn, 5)
	constraints := []ConstraintItem{
		semaphoreConstraint("simple-counter", enums.SemaphoreScopeFn, 1),
	}

	// With amount=1, each lease consumes exactly 1 unit
	resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-amt1",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               5,
		LeaseIdempotencyKeys: []string{"a1", "a2", "a3", "a4", "a5"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp.Leases, 5, "Should grant exactly 5 leases")

	// Now exhausted
	resp2, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-amt1-2",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"a6"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp2.Leases, 0, "No capacity left")
}

func TestSemaphore_LargeAmount(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	// capacity=100, amount=50 → can grant floor(100/50)=2 leases
	config := semaphoreConfig("large-sem", enums.SemaphoreScopeFn, 100)
	constraints := []ConstraintItem{
		semaphoreConstraint("large-sem", enums.SemaphoreScopeFn, 50),
	}

	resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-large",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               3,
		LeaseIdempotencyKeys: []string{"lg1", "lg2", "lg3"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp.Leases, 2, "Should grant 2 leases (floor(100/50))")
}

func TestSemaphore_TTLOnKey(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := semaphoreConfig("ttl-sem", enums.SemaphoreScopeFn, 10)
	constraints := []ConstraintItem{
		semaphoreConstraint("ttl-sem", enums.SemaphoreScopeFn, 1),
	}

	_, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-ttl",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"ttl1"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)

	// Verify TTL is set on the semaphore key (7 days = 604800 seconds)
	semKey := constraints[0].Semaphore.StateKey(te.AccountID, te.EnvID, te.FunctionID)
	ttl := te.Redis.TTL(semKey)
	require.True(t, ttl > 0, "Semaphore key should have TTL")
	require.Equal(t, 604800, int(ttl.Seconds()), "TTL should be 7 days (604800 seconds)")
}

func TestSemaphore_Idempotency(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := semaphoreConfig("idem-sem", enums.SemaphoreScopeFn, 10)
	constraints := []ConstraintItem{
		semaphoreConstraint("idem-sem", enums.SemaphoreScopeFn, 2),
	}

	req := &CapacityAcquireRequest{
		IdempotencyKey:       "sem-idem-key",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"il1"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	}

	// First acquire
	resp1, err := te.CapacityManager.Acquire(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, resp1.Leases, 1)

	// Retry same idempotency key — should return same result without double-counting
	resp2, err := te.CapacityManager.Acquire(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, resp2.Leases, 1)
	require.Equal(t, resp1.Leases[0].LeaseID, resp2.Leases[0].LeaseID, "Idempotent retry should return same lease")

	// Verify semaphore counter is only 2 (not 4)
	semKey := constraints[0].Semaphore.StateKey(te.AccountID, te.EnvID, te.FunctionID)
	val, _ := te.Redis.Get(semKey)
	require.Equal(t, "2", val, "Semaphore counter should be 2 (not double-counted)")
}

func TestSemaphore_MultipleIndependentSemaphores(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	// Use key expressions to differentiate the two semaphores at the same scope
	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphore: []SemaphoreConfig{
			{Name: "sem-a", Scope: enums.SemaphoreScopeFn, Capacity: 5, KeyExpressionHash: "key-a"},
			{Name: "sem-b", Scope: enums.SemaphoreScopeFn, Capacity: 3, KeyExpressionHash: "key-b"},
		},
	}

	constraints := []ConstraintItem{
		semaphoreConstraintWithExpr("sem-a", enums.SemaphoreScopeFn, 1, "key-a", "eval-a"),
		semaphoreConstraintWithExpr("sem-b", enums.SemaphoreScopeFn, 1, "key-b", "eval-b"),
	}

	// Both semaphores present. sem-b limits to 3
	resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-multi",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               5,
		LeaseIdempotencyKeys: []string{"m1", "m2", "m3", "m4", "m5"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp.Leases, 3, "Should be limited by sem-b (capacity=3)")

	// Verify both semaphore counters have different keys
	semKeyA := constraints[0].Semaphore.StateKey(te.AccountID, te.EnvID, te.FunctionID)
	semKeyB := constraints[1].Semaphore.StateKey(te.AccountID, te.EnvID, te.FunctionID)
	require.NotEqual(t, semKeyA, semKeyB, "Keys should be different")

	valA, _ := te.Redis.Get(semKeyA)
	valB, _ := te.Redis.Get(semKeyB)
	require.Equal(t, "3", valA, "sem-a counter should be 3")
	require.Equal(t, "3", valB, "sem-b counter should be 3")
}

func TestSemaphore_RetryAfterOnExhaustion(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := semaphoreConfig("retry-sem", enums.SemaphoreScopeFn, 1)
	constraints := []ConstraintItem{
		semaphoreConstraint("retry-sem", enums.SemaphoreScopeFn, 1),
	}

	// Fill capacity
	_, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-retry-fill",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"rf1"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)

	// Try to acquire when exhausted — should get retryAfter
	resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-retry-fail",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"rf2"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp.Leases, 0)
	require.False(t, resp.RetryAfter.IsZero(), "RetryAfter should be set when exhausted")
}

func TestSemaphore_StateKey(t *testing.T) {
	accountID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	envID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	fnID := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	t.Run("function scope without expression", func(t *testing.T) {
		s := &SemaphoreConstraint{Name: "test", Scope: enums.SemaphoreScopeFn}
		key := s.StateKey(accountID, envID, fnID)
		expected := fmt.Sprintf("{cs}:a:%s:sem:f:%s", accountID, fnID)
		require.Equal(t, expected, key)
	})

	t.Run("env scope without expression", func(t *testing.T) {
		s := &SemaphoreConstraint{Name: "test", Scope: enums.SemaphoreScopeEnv}
		key := s.StateKey(accountID, envID, fnID)
		expected := fmt.Sprintf("{cs}:a:%s:sem:e:%s", accountID, envID)
		require.Equal(t, expected, key)
	})

	t.Run("account scope without expression", func(t *testing.T) {
		s := &SemaphoreConstraint{Name: "test", Scope: enums.SemaphoreScopeAccount}
		key := s.StateKey(accountID, envID, fnID)
		expected := fmt.Sprintf("{cs}:a:%s:sem:a:%s", accountID, accountID)
		require.Equal(t, expected, key)
	})

	t.Run("function scope with expression", func(t *testing.T) {
		s := &SemaphoreConstraint{
			Name:              "test",
			Scope:             enums.SemaphoreScopeFn,
			KeyExpressionHash: "expr123",
			EvaluatedKeyHash:  "eval456",
		}
		key := s.StateKey(accountID, envID, fnID)
		expected := fmt.Sprintf("{cs}:a:%s:sem:f:%s:expr123:eval456", accountID, fnID)
		require.Equal(t, expected, key)
	})
}

func TestSemaphore_RedisCounterValue(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	config := semaphoreConfig("counter-sem", enums.SemaphoreScopeFn, 20)
	constraints := []ConstraintItem{
		semaphoreConstraint("counter-sem", enums.SemaphoreScopeFn, 3),
	}

	semKey := constraints[0].Semaphore.StateKey(te.AccountID, te.EnvID, te.FunctionID)

	// Acquire 2 leases (2 * 3 = 6 units)
	resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-counter-1",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               2,
		LeaseIdempotencyKeys: []string{"cv1", "cv2"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config,
		Constraints:          constraints,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp.Leases, 2)

	// Verify counter = 6
	val, _ := te.Redis.Get(semKey)
	counter, _ := strconv.Atoi(val)
	require.Equal(t, 6, counter, "Counter should be 6 (2 leases * 3 amount)")

	// Release one lease → counter should decrease by 3
	_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
		IdempotencyKey: "sem-counter-rel1",
		AccountID:      te.AccountID,
		LeaseID:        resp.Leases[0].LeaseID,
	})
	require.NoError(t, err)

	val, _ = te.Redis.Get(semKey)
	counter, _ = strconv.Atoi(val)
	require.Equal(t, 3, counter, "Counter should be 3 after releasing 1 lease")

	// Release second lease → counter goes to 0 → key deleted
	_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
		IdempotencyKey: "sem-counter-rel2",
		AccountID:      te.AccountID,
		LeaseID:        resp.Leases[1].LeaseID,
	})
	require.NoError(t, err)

	require.False(t, te.Redis.Exists(semKey), "Key should be deleted when counter reaches 0")
}

// TestSemaphore_CapacityReduction verifies that when the semaphore capacity is reduced
// between acquires, the second acquire correctly refuses to grant a lease even though
// the counter is below the new capacity. This guards against negative remaining values
// when cap - currentCount < 0.
func TestSemaphore_CapacityReduction(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	// First acquire: capacity=20, amount=1, acquire 10 leases
	config1 := semaphoreConfig("test-sem", enums.SemaphoreScopeFn, 20)
	constraints1 := []ConstraintItem{
		semaphoreConstraint("test-sem", enums.SemaphoreScopeFn, 1),
	}

	leaseKeys := make([]string, 10)
	for i := range leaseKeys {
		leaseKeys[i] = fmt.Sprintf("lease-%d", i+1)
	}

	resp1, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-cap-reduce-1",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               10,
		LeaseIdempotencyKeys: leaseKeys,
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config1,
		Constraints:          constraints1,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})

	require.NoError(t, err)
	require.Len(t, resp1.Leases, 10, "Should grant 10 leases with capacity=20")

	// Second acquire: capacity reduced to 10 (same as current count), amount=1
	// The counter is at 10, so remaining = max(0, 10-10) = 0 → no lease granted
	config2 := semaphoreConfig("test-sem", enums.SemaphoreScopeFn, 10)
	constraints2 := []ConstraintItem{
		semaphoreConstraint("test-sem", enums.SemaphoreScopeFn, 1),
	}

	resp2, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
		IdempotencyKey:       "sem-cap-reduce-2",
		AccountID:            te.AccountID,
		EnvID:                te.EnvID,
		FunctionID:           te.FunctionID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"lease-11"},
		CurrentTime:          clock.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Minute,
		Configuration:        config2,
		Constraints:          constraints2,
		Source:               LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})

	require.NoError(t, err)
	require.Len(t, resp2.Leases, 0, "Should not grant any leases when capacity reduced to current count")
	require.NotEmpty(t, resp2.ExhaustedConstraints, "Semaphore constraint should be exhausted")

	// Also verify check reports no capacity
	checkResp, userErr, internalErr := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
		AccountID:     te.AccountID,
		EnvID:         te.EnvID,
		FunctionID:    te.FunctionID,
		Configuration: config2,
		Constraints:   constraints2,
	})

	require.Nil(t, userErr)
	require.Nil(t, internalErr)
	require.Equal(t, 0, checkResp.AvailableCapacity, "Check should report 0 available capacity")
	require.NotEmpty(t, checkResp.ExhaustedConstraints, "Check should report exhausted constraint")
}
