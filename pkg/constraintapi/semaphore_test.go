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

func newSemaphoreTestEnv(t *testing.T) (*redisCapacityManager, *miniredis.Miniredis, rueidis.Client, clockwork.FakeClock) {
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
	clock clockwork.FakeClock,
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
		Name:    "app:" + uuid.New().String(),
		Weight:  1,
		Release: SemaphoreReleaseAuto,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{Name: sem.Name, Weight: 1, Release: SemaphoreReleaseAuto}},
	}

	constraints := []ConstraintItem{
		{Kind: ConstraintKindSemaphore, Semaphore: &sem},
	}

	// Set capacity to 2
	capKey := sem.CapacityKey(accountID)
	r.Set(capKey, "2")

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
		Name:    "app:" + uuid.New().String(),
		Weight:  1,
		Release: SemaphoreReleaseAuto,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{Name: sem.Name, Weight: 1, Release: SemaphoreReleaseAuto}},
	}

	constraints := []ConstraintItem{
		{Kind: ConstraintKindSemaphore, Semaphore: &sem},
	}

	// Set capacity to 1
	capKey := sem.CapacityKey(accountID)
	r.Set(capKey, "1")

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
		Name:    "fn:" + fnID.String(),
		Weight:  1,
		Release: SemaphoreReleaseManual,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{Name: sem.Name, Weight: 1, Release: SemaphoreReleaseManual}},
	}

	constraints := []ConstraintItem{
		{Kind: ConstraintKindSemaphore, Semaphore: &sem},
	}

	// Set capacity to 1
	capKey := sem.CapacityKey(accountID)
	r.Set(capKey, "1")

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
		Name:    "app:" + uuid.New().String(),
		Weight:  3,
		Release: SemaphoreReleaseAuto,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{Name: sem.Name, Weight: 3, Release: SemaphoreReleaseAuto}},
	}

	constraints := []ConstraintItem{
		{Kind: ConstraintKindSemaphore, Semaphore: &sem},
	}

	// Set capacity to 5
	capKey := sem.CapacityKey(accountID)
	r.Set(capKey, "5")

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
		Name:    "app:" + uuid.New().String(),
		Weight:  1,
		Release: SemaphoreReleaseAuto,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{Name: sem.Name, Weight: 1, Release: SemaphoreReleaseAuto}},
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
		Name:    "app:" + uuid.New().String(),
		Weight:  1,
		Release: SemaphoreReleaseAuto,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			AccountConcurrency:  10,
			FunctionConcurrency: 5,
		},
		Semaphores: []Semaphore{{Name: sem.Name, Weight: 1, Release: SemaphoreReleaseAuto}},
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
	r.Set(capKey, "1")

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

		cap, usage, err := sm.GetCapacity(ctx, accountID, name)
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

		cap, _, err := sm.GetCapacity(ctx, accountID, name)
		require.NoError(t, err)
		require.Equal(t, int64(20), cap, "idempotent call should not change capacity")
	})

	t.Run("adjust capacity", func(t *testing.T) {
		name := fmt.Sprintf("app:%s", uuid.New())
		err := sm.SetCapacity(ctx, accountID, name, "adj-set", 5)
		require.NoError(t, err)

		err = sm.AdjustCapacity(ctx, accountID, name, "adj-1", 3)
		require.NoError(t, err)

		cap, _, err := sm.GetCapacity(ctx, accountID, name)
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

		cap, _, err := sm.GetCapacity(ctx, accountID, name)
		require.NoError(t, err)
		require.Equal(t, int64(8), cap, "idempotent adjust should not double-add")
	})

	t.Run("release semaphore", func(t *testing.T) {
		name := fmt.Sprintf("fn:%s", uuid.New())
		// Manually set usage
		usageKey := semaphoreUsageKey(accountID, name)
		r.Set(usageKey, "5")

		err := sm.ReleaseSemaphore(ctx, accountID, name, "rel-1", 2)
		require.NoError(t, err)

		_, usage, err := sm.GetCapacity(ctx, accountID, name)
		require.NoError(t, err)
		require.Equal(t, int64(3), usage)
	})

	t.Run("release semaphore clamps to zero", func(t *testing.T) {
		name := fmt.Sprintf("fn:%s", uuid.New())
		usageKey := semaphoreUsageKey(accountID, name)
		r.Set(usageKey, "1")

		err := sm.ReleaseSemaphore(ctx, accountID, name, "rel-clamp", 5)
		require.NoError(t, err)

		_, usage, err := sm.GetCapacity(ctx, accountID, name)
		require.NoError(t, err)
		require.Equal(t, int64(0), usage, "should clamp to 0")
	})

	t.Run("release semaphore idempotency", func(t *testing.T) {
		name := fmt.Sprintf("fn:%s", uuid.New())
		usageKey := semaphoreUsageKey(accountID, name)
		r.Set(usageKey, "5")

		err := sm.ReleaseSemaphore(ctx, accountID, name, "rel-idem", 2)
		require.NoError(t, err)
		// Same idempotency key — should not double-decrement
		err = sm.ReleaseSemaphore(ctx, accountID, name, "rel-idem", 2)
		require.NoError(t, err)

		_, usage, err := sm.GetCapacity(ctx, accountID, name)
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
		Name:    "fn:" + fnID.String(),
		Weight:  1,
		Release: SemaphoreReleaseManual,
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []Semaphore{{Name: sem.Name, Weight: 1, Release: SemaphoreReleaseManual}},
	}

	constraints := []ConstraintItem{
		{Kind: ConstraintKindSemaphore, Semaphore: &sem},
	}

	// Set capacity to 1
	capKey := sem.CapacityKey(accountID)
	r.Set(capKey, "1")

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
