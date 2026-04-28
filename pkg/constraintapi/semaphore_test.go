package constraintapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func newSemaphoreTestEnv(t *testing.T) (*redisCapacityManager, *miniredis.Miniredis, rueidis.Client, *clockwork.FakeClock) {
	t.Helper()
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	clock := clockwork.NewFakeClock()

	cm, err := NewRedisCapacityManager(
		WithShardName("default"),
		WithClient(rc),
		WithClock(clock),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		rc.Close()
		r.Close()
	})

	return cm, r, rc, clock
}

func acquireWithSemaphore(
	t *testing.T,
	cm *redisCapacityManager,
	clock *clockwork.FakeClock,
	accountID, envID, fnID uuid.UUID,
	config ConstraintConfig,
	constraints []ConstraintItem,
	idempotencyKey string,
) *CapacityAcquireResponse {
	t.Helper()

	resp, err := cm.Acquire(context.Background(), &CapacityAcquireRequest{
		AccountID:            accountID,
		IdempotencyKey:       idempotencyKey,
		Constraints:          constraints,
		Amount:               1,
		EnvID:                envID,
		FunctionID:           fnID,
		Configuration:        config,
		LeaseIdempotencyKeys: []string{idempotencyKey + "-lease"},
		LeaseRunIDs:          map[string]ulid.ULID{},
		CurrentTime:          clock.Now(),
		Duration:             5 * time.Second,
		MaximumLifetime:      time.Hour,
		Source: LeaseSource{
			Service:           ServiceExecutor,
			Location:          CallerLocationItemLease,
			RunProcessingMode: RunProcessingModeBackground,
		},
	})
	require.NoError(t, err)
	return resp
}

func TestSemaphoreAcquire(t *testing.T) {
	cm, r, _, clock := newSemaphoreTestEnv(t)
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	sem := SemaphoreConstraint{
		ID:      "app:" + uuid.New().String(),
		Weight:  1,
		Release: SemaphoreReleaseAuto,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{ID: sem.ID, Weight: 1, Release: SemaphoreReleaseAuto}},
	}

	constraints := []ConstraintItem{
		{Kind: ConstraintKindSemaphore, Semaphore: &sem},
	}

	// Set capacity to 2
	capKey := sem.CapacityKey(accountID)
	_ = r.Set(capKey, "2")

	// First acquire should succeed
	resp := acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, config, constraints, "acq1")
	require.Len(t, resp.Leases, 1, "should grant 1 lease when capacity is available")

	// Usage should be 1
	usageKey := sem.UsageKey(accountID)
	val, err := r.Get(usageKey)
	require.NoError(t, err)
	require.Equal(t, "1", val)

	// Second acquire should succeed (capacity=2, usage=1)
	clock.Advance(time.Second)
	resp = acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, config, constraints, "acq2")
	require.Len(t, resp.Leases, 1, "should grant 1 lease when capacity remains")

	// Usage should be 2
	val, err = r.Get(usageKey)
	require.NoError(t, err)
	require.Equal(t, "2", val)

	// Third acquire should fail (capacity=2, usage=2)
	clock.Advance(time.Second)
	resp = acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, config, constraints, "acq3")
	require.Len(t, resp.Leases, 0, "should grant 0 leases when exhausted")
	require.Len(t, resp.ExhaustedConstraints, 1)
	require.Equal(t, ConstraintKindSemaphore, resp.ExhaustedConstraints[0].Kind)
}

func TestSemaphoreAutoRelease(t *testing.T) {
	cm, r, _, clock := newSemaphoreTestEnv(t)
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	sem := SemaphoreConstraint{
		ID:      "app:" + uuid.New().String(),
		Weight:  1,
		Release: SemaphoreReleaseAuto,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{ID: sem.ID, Weight: 1, Release: SemaphoreReleaseAuto}},
	}

	constraints := []ConstraintItem{
		{Kind: ConstraintKindSemaphore, Semaphore: &sem},
	}

	// Set capacity to 1
	capKey := sem.CapacityKey(accountID)
	_ = r.Set(capKey, "1")

	// Acquire
	resp := acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, config, constraints, "acq-auto")
	require.Len(t, resp.Leases, 1)

	usageKey := sem.UsageKey(accountID)
	val, _ := r.Get(usageKey)
	require.Equal(t, "1", val, "usage should be 1 after acquire")

	// Extend the lease so we can release it
	clock.Advance(2 * time.Second)
	r.FastForward(2 * time.Second)
	r.SetTime(clock.Now())

	extResp, err := cm.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
		IdempotencyKey: "extend-auto",
		LeaseID:        resp.Leases[0].LeaseID,
		AccountID:      accountID,
		Duration:       5 * time.Second,
		LeaseIssuedAt:  clock.Now(),
	})
	require.NoError(t, err)
	require.NotNil(t, extResp.LeaseID)

	// Release — should DECRBY for auto-release
	_, err = cm.Release(context.Background(), &CapacityReleaseRequest{
		AccountID:      accountID,
		IdempotencyKey: "release-auto",
		LeaseID:        *extResp.LeaseID,
		LeaseIssuedAt:  clock.Now(),
	})
	require.NoError(t, err)

	val, _ = r.Get(usageKey)
	require.Equal(t, "0", val, "usage should be 0 after auto-release")
}

func TestSemaphoreManualRelease(t *testing.T) {
	cm, r, _, clock := newSemaphoreTestEnv(t)
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	sem := SemaphoreConstraint{
		ID:      "fn:" + fnID.String(),
		Weight:  1,
		Release: SemaphoreReleaseManual,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{ID: sem.ID, Weight: 1, Release: SemaphoreReleaseManual}},
	}

	constraints := []ConstraintItem{
		{Kind: ConstraintKindSemaphore, Semaphore: &sem},
	}

	// Set capacity to 1
	capKey := sem.CapacityKey(accountID)
	_ = r.Set(capKey, "1")

	// Acquire
	resp := acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, config, constraints, "acq-manual")
	require.Len(t, resp.Leases, 1)

	usageKey := sem.UsageKey(accountID)
	val, _ := r.Get(usageKey)
	require.Equal(t, "1", val, "usage should be 1 after acquire")

	// Extend + Release the constraint lease — should NOT decrement for manual release
	clock.Advance(2 * time.Second)
	r.FastForward(2 * time.Second)
	r.SetTime(clock.Now())

	extResp, err := cm.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
		IdempotencyKey: "extend-manual",
		LeaseID:        resp.Leases[0].LeaseID,
		AccountID:      accountID,
		Duration:       5 * time.Second,
		LeaseIssuedAt:  clock.Now(),
	})
	require.NoError(t, err)
	require.NotNil(t, extResp.LeaseID)

	_, err = cm.Release(context.Background(), &CapacityReleaseRequest{
		AccountID:      accountID,
		IdempotencyKey: "release-manual",
		LeaseID:        *extResp.LeaseID,
		LeaseIssuedAt:  clock.Now(),
	})
	require.NoError(t, err)

	// Usage should still be 1 — manual release doesn't decrement
	val, _ = r.Get(usageKey)
	require.Equal(t, "1", val, "usage should still be 1 after manual release (not decremented)")
}

func TestSemaphoreWeight(t *testing.T) {
	cm, r, _, clock := newSemaphoreTestEnv(t)
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	sem := SemaphoreConstraint{
		ID:      "app:" + uuid.New().String(),
		Weight:  3,
		Release: SemaphoreReleaseAuto,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{ID: sem.ID, Weight: 3, Release: SemaphoreReleaseAuto}},
	}

	constraints := []ConstraintItem{
		{Kind: ConstraintKindSemaphore, Semaphore: &sem},
	}

	// Set capacity to 5
	capKey := sem.CapacityKey(accountID)
	_ = r.Set(capKey, "5")

	// First acquire with weight=3 should succeed (5-0 >= 3)
	resp := acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, config, constraints, "w1")
	require.Len(t, resp.Leases, 1)

	usageKey := sem.UsageKey(accountID)
	val, _ := r.Get(usageKey)
	require.Equal(t, "3", val, "usage should be 3 (weight=3)")

	// Second acquire should fail (5-3=2 < 3)
	clock.Advance(time.Second)
	resp = acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, config, constraints, "w2")
	require.Len(t, resp.Leases, 0, "should fail when remaining < weight")
}

func TestSemaphoreZeroCapacity(t *testing.T) {
	cm, _, _, clock := newSemaphoreTestEnv(t)
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	sem := SemaphoreConstraint{
		ID:      "app:" + uuid.New().String(),
		Weight:  1,
		Release: SemaphoreReleaseAuto,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{ID: sem.ID, Weight: 1, Release: SemaphoreReleaseAuto}},
	}

	constraints := []ConstraintItem{
		{Kind: ConstraintKindSemaphore, Semaphore: &sem},
	}

	// Don't set any capacity — defaults to 0
	resp := acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, config, constraints, "zero")
	require.Len(t, resp.Leases, 0, "should grant 0 leases when capacity is 0")
	require.Len(t, resp.ExhaustedConstraints, 1)
}

func TestSemaphoreWithConcurrency(t *testing.T) {
	cm, r, _, clock := newSemaphoreTestEnv(t)
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	sem := SemaphoreConstraint{
		ID:      "app:" + uuid.New().String(),
		Weight:  1,
		Release: SemaphoreReleaseAuto,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			AccountConcurrency:  10,
			FunctionConcurrency: 5,
		},
		Semaphores: []Semaphore{{ID: sem.ID, Weight: 1, Release: SemaphoreReleaseAuto}},
	}

	constraints := []ConstraintItem{
		{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: 2, Mode: 0}, // account
		},
		{
			Kind:        ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{Scope: 0, Mode: 0}, // function
		},
		{Kind: ConstraintKindSemaphore, Semaphore: &sem},
	}

	// Semaphore capacity = 1, concurrency limits = plenty
	capKey := sem.CapacityKey(accountID)
	_ = r.Set(capKey, "1")

	// First acquire: all constraints pass
	resp := acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, config, constraints, "multi1")
	require.Len(t, resp.Leases, 1)

	// Second acquire: semaphore blocks even though concurrency has capacity
	clock.Advance(time.Second)
	resp = acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, config, constraints, "multi2")
	require.Len(t, resp.Leases, 0, "semaphore should block despite concurrency capacity")
}

func TestSemaphoreManager(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		rc.Close()
		r.Close()
	})

	sm := NewRedisSemaphoreManager(rc)
	accountID := uuid.New()
	name := fmt.Sprintf("app:%s", uuid.New())
	ctx := context.Background()

	t.Run("set and get capacity", func(t *testing.T) {
		err := sm.SetCapacity(ctx, accountID, name, "set-1", 10)
		require.NoError(t, err)

		cap, usage, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(10), cap)
		require.Equal(t, int64(0), usage)
	})

	t.Run("set capacity idempotency", func(t *testing.T) {
		err := sm.SetCapacity(ctx, accountID, name, "set-idem", 20)
		require.NoError(t, err)
		// Same idempotency key — should not change
		err = sm.SetCapacity(ctx, accountID, name, "set-idem", 99)
		require.NoError(t, err)

		cap, _, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(20), cap, "idempotent call should not change capacity")
	})

	t.Run("adjust capacity", func(t *testing.T) {
		name := fmt.Sprintf("app:%s", uuid.New())
		err := sm.SetCapacity(ctx, accountID, name, "adj-set", 5)
		require.NoError(t, err)

		err = sm.AdjustCapacity(ctx, accountID, name, "adj-1", 3)
		require.NoError(t, err)

		cap, _, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(8), cap)
	})

	t.Run("adjust capacity idempotency", func(t *testing.T) {
		name := fmt.Sprintf("app:%s", uuid.New())
		err := sm.SetCapacity(ctx, accountID, name, "adj-idem-set", 5)
		require.NoError(t, err)

		err = sm.AdjustCapacity(ctx, accountID, name, "adj-idem-1", 3)
		require.NoError(t, err)
		// Same idempotency key — should not double-add
		err = sm.AdjustCapacity(ctx, accountID, name, "adj-idem-1", 3)
		require.NoError(t, err)

		cap, _, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(8), cap, "idempotent adjust should not double-add")
	})

	t.Run("release semaphore", func(t *testing.T) {
		name := fmt.Sprintf("fn:%s", uuid.New())
		// Manually set usage
		usageKey := semaphoreUsageKey(accountID, name, "")
		_ = r.Set(usageKey, "5")

		err := sm.ReleaseSemaphore(ctx, accountID, name, "", "rel-1", 2)
		require.NoError(t, err)

		_, usage, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(3), usage)
	})

	t.Run("release semaphore clamps to zero", func(t *testing.T) {
		name := fmt.Sprintf("fn:%s", uuid.New())
		usageKey := semaphoreUsageKey(accountID, name, "")
		_ = r.Set(usageKey, "1")

		err := sm.ReleaseSemaphore(ctx, accountID, name, "", "rel-clamp", 5)
		require.NoError(t, err)

		_, usage, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(0), usage, "should clamp to 0")
	})

	t.Run("release semaphore idempotency", func(t *testing.T) {
		name := fmt.Sprintf("fn:%s", uuid.New())
		usageKey := semaphoreUsageKey(accountID, name, "")
		_ = r.Set(usageKey, "5")

		err := sm.ReleaseSemaphore(ctx, accountID, name, "", "rel-idem", 2)
		require.NoError(t, err)
		// Same idempotency key — should not double-decrement
		err = sm.ReleaseSemaphore(ctx, accountID, name, "", "rel-idem", 2)
		require.NoError(t, err)

		_, usage, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(3), usage, "idempotent release should not double-decrement")
	})
}

// TestSemaphoreScavengeManualRelease verifies that the scavenger force-releases
// manual-release semaphores when a constraint lease expires.  Without this,
// a crashed executor holding a manual-release semaphore would deadlock all
// future runs waiting on that capacity.
func TestSemaphoreScavengeManualRelease(t *testing.T) {
	cm, r, _, clock := newSemaphoreTestEnv(t)
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	sem := SemaphoreConstraint{
		ID:      "fn:" + fnID.String(),
		Weight:  1,
		Release: SemaphoreReleaseManual,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{ID: sem.ID, Weight: 1, Release: SemaphoreReleaseManual}},
	}

	constraints := []ConstraintItem{
		{Kind: ConstraintKindSemaphore, Semaphore: &sem},
	}

	// Set capacity to 1
	capKey := sem.CapacityKey(accountID)
	_ = r.Set(capKey, "1")

	// Acquire a lease with a manual-release semaphore
	resp := acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, config, constraints, "scav-manual")
	require.Len(t, resp.Leases, 1)

	usageKey := sem.UsageKey(accountID)
	val, _ := r.Get(usageKey)
	require.Equal(t, "1", val, "usage should be 1 after acquire")

	// Advance time past the lease expiry so the scavenger can find it
	clock.Advance(10 * time.Second)
	r.FastForward(10 * time.Second)
	r.SetTime(clock.Now())

	// Run scavenger — this should release the expired lease AND decrement the semaphore
	result, err := cm.Scavenge(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, result.ReclaimedLeases, "scavenger should reclaim 1 expired lease")

	// The semaphore usage MUST be decremented even though release mode is manual.
	// A scavenged lease means the executor died — holding the semaphore would deadlock.
	val, _ = r.Get(usageKey)
	require.Equal(t, "0", val, "scavenger must force-release manual semaphore to prevent deadlock")
}

// TestSemaphoreUsageValueIsolation verifies that different UsageValues get independent
// usage counters while sharing the same capacity.  This is the core of key-based fn
// concurrency: e.g., 5 concurrent runs per customer, where each customer has its own
// usage counter but all share the same capacity limit.
func TestSemaphoreUsageValueIsolation(t *testing.T) {
	cm, r, _, clock := newSemaphoreTestEnv(t)
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	semID := "fnkey:" + uuid.New().String()

	// Set shared capacity to 2
	capKey := (&SemaphoreConstraint{ID: semID}).CapacityKey(accountID)
	_ = r.Set(capKey, "2")

	// Two different usage values (e.g., two different customers)
	semA := SemaphoreConstraint{ID: semID, UsageValue: "customer-a", Weight: 1, Release: SemaphoreReleaseManual}
	semB := SemaphoreConstraint{ID: semID, UsageValue: "customer-b", Weight: 1, Release: SemaphoreReleaseManual}

	configA := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{ID: semA.ID, UsageValue: semA.UsageValue, Weight: 1, Release: SemaphoreReleaseManual}},
	}
	configB := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{ID: semB.ID, UsageValue: semB.UsageValue, Weight: 1, Release: SemaphoreReleaseManual}},
	}

	constraintsA := []ConstraintItem{{Kind: ConstraintKindSemaphore, Semaphore: &semA}}
	constraintsB := []ConstraintItem{{Kind: ConstraintKindSemaphore, Semaphore: &semB}}

	// Acquire for customer A — should succeed (cap=2, usageA=0)
	resp := acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, configA, constraintsA, "iso-a1")
	require.Len(t, resp.Leases, 1, "customer A first acquire should succeed")

	// Acquire for customer B — should succeed (cap=2, usageB=0, independent counter)
	clock.Advance(time.Second)
	resp = acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, configB, constraintsB, "iso-b1")
	require.Len(t, resp.Leases, 1, "customer B first acquire should succeed")

	// Verify independent usage counters
	usageA, _ := r.Get(semA.UsageKey(accountID))
	usageB, _ := r.Get(semB.UsageKey(accountID))
	require.Equal(t, "1", usageA, "customer A usage should be 1")
	require.Equal(t, "1", usageB, "customer B usage should be 1")

	// Acquire again for customer A — should succeed (cap=2, usageA=1)
	clock.Advance(time.Second)
	resp = acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, configA, constraintsA, "iso-a2")
	require.Len(t, resp.Leases, 1, "customer A second acquire should succeed")

	// Acquire again for customer A — should fail (cap=2, usageA=2)
	clock.Advance(time.Second)
	resp = acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, configA, constraintsA, "iso-a3")
	require.Len(t, resp.Leases, 0, "customer A third acquire should fail — capacity exhausted")

	// Customer B should still be able to acquire (cap=2, usageB=1)
	clock.Advance(time.Second)
	resp = acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, configB, constraintsB, "iso-b2")
	require.Len(t, resp.Leases, 1, "customer B second acquire should still succeed — independent counter")
}

// TestSemaphoreSameUsageValueShared verifies that two acquires with the same
// UsageValue share a single counter (same customer, same semaphore).
func TestSemaphoreSameUsageValueShared(t *testing.T) {
	cm, r, _, clock := newSemaphoreTestEnv(t)
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	semID := "fnkey:" + uuid.New().String()

	// Set shared capacity to 1
	capKey := (&SemaphoreConstraint{ID: semID}).CapacityKey(accountID)
	_ = r.Set(capKey, "1")

	sem := SemaphoreConstraint{ID: semID, UsageValue: "same-customer", Weight: 1, Release: SemaphoreReleaseManual}
	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{ID: sem.ID, UsageValue: sem.UsageValue, Weight: 1, Release: SemaphoreReleaseManual}},
	}
	constraints := []ConstraintItem{{Kind: ConstraintKindSemaphore, Semaphore: &sem}}

	// First acquire succeeds
	resp := acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, config, constraints, "same-1")
	require.Len(t, resp.Leases, 1)

	// Second acquire with same usage value fails — they share the counter
	clock.Advance(time.Second)
	resp = acquireWithSemaphore(t, cm, clock, accountID, envID, fnID, config, constraints, "same-2")
	require.Len(t, resp.Leases, 0, "same usage value should share counter and exhaust capacity")

	usageKey := sem.UsageKey(accountID)
	val, _ := r.Get(usageKey)
	require.Equal(t, "1", val, "shared counter should be 1")
}

func TestSemaphoreGetCapacity(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	sm := NewRedisSemaphoreManager(rc)
	ctx := context.Background()
	accountID := uuid.New()
	name := "app:" + uuid.New().String()

	t.Run("nonexistent returns zero", func(t *testing.T) {
		cap, usage, err := sm.GetCapacity(ctx, accountID, name, "some-value")
		require.NoError(t, err)
		require.Equal(t, int64(0), cap)
		require.Equal(t, int64(0), usage)
	})

	t.Run("returns set capacity and usage", func(t *testing.T) {
		capKey := fmt.Sprintf("{cs}:%s:sem:%s:cap", accountScope(accountID), name)
		usageKey := fmt.Sprintf("{cs}:%s:sem:%s:usage:%s", accountScope(accountID), name, "run-abc")

		require.NoError(t, r.Set(capKey, "100"))
		require.NoError(t, r.Set(usageKey, "42"))

		cap, usage, err := sm.GetCapacity(ctx, accountID, name, "run-abc")
		require.NoError(t, err)
		require.Equal(t, int64(100), cap)
		require.Equal(t, int64(42), usage)
	})
}

func TestSemaphoreAdjustCapacityClampsToZero(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	sm := NewRedisSemaphoreManager(rc)
	ctx := context.Background()
	accountID := uuid.New()
	name := "app:" + uuid.New().String()

	// Set capacity to 5
	err = sm.SetCapacity(ctx, accountID, name, "set-1", 5)
	require.NoError(t, err)

	// Adjust by -10, should clamp to 0
	err = sm.AdjustCapacity(ctx, accountID, name, "adj-1", -10)
	require.NoError(t, err)

	cap, _, err := sm.GetCapacity(ctx, accountID, name, "")
	require.NoError(t, err)
	require.Equal(t, int64(0), cap, "capacity should clamp to zero, not go negative")
}
