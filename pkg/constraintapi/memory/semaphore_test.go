package memory_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

// these are the semaphore tests from constraintapi/semaphore_test.go run
// against the memory manager.  usage is read through GetCapacity instead of
// the Redis key, and capacity is set through SetCapacity.

func TestSemaphoreAcquire(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:" + uuid.NewString(), Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}

	setCapacity(t, m, id.account, sem.ID, 2)

	resp := acquire(t, m, acquireRequest(id, clock, config, constraints, "acq1"))
	require.Len(t, resp.Leases, 1, "should grant 1 lease when capacity is available")
	require.Equal(t, "acq1-lease", resp.Leases[0].IdempotencyKey)
	require.Equal(t, clock.Now().Add(5*time.Second).UnixMilli(), int64(resp.Leases[0].LeaseID.Time()), "the lease ID timestamp is the expiry")
	require.Equal(t, int64(1), usageOf(t, m, id.account, sem))
	require.Len(t, resp.Usage, 1)
	require.Equal(t, 1, resp.Usage[0].Used)
	require.Equal(t, 2, resp.Usage[0].Limit)

	clock.Advance(time.Second)
	resp = acquire(t, m, acquireRequest(id, clock, config, constraints, "acq2"))
	require.Len(t, resp.Leases, 1, "should grant 1 lease when capacity remains")
	require.Equal(t, int64(2), usageOf(t, m, id.account, sem))
	require.Len(t, resp.ExhaustedConstraints, 1, "the semaphore is exhausted after this grant")

	clock.Advance(time.Second)
	resp = acquire(t, m, acquireRequest(id, clock, config, constraints, "acq3"))
	require.Len(t, resp.Leases, 0, "should grant 0 leases when exhausted")
	require.Len(t, resp.ExhaustedConstraints, 1)
	require.Equal(t, constraintapi.ConstraintKindSemaphore, resp.ExhaustedConstraints[0].Kind)
	require.Len(t, resp.LimitingConstraints, 1)
	require.True(t, resp.RetryAfter.IsZero(), "semaphore exhaustion has no retry after")
	require.Equal(t, int64(2), usageOf(t, m, id.account, sem), "a rejected acquire changes nothing")
}

func TestSemaphoreAutoRelease(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:" + uuid.NewString(), Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
	setCapacity(t, m, id.account, sem.ID, 1)

	resp := acquire(t, m, acquireRequest(id, clock, config, constraints, "acq-auto"))
	require.Len(t, resp.Leases, 1)
	require.Equal(t, int64(1), usageOf(t, m, id.account, sem))

	clock.Advance(2 * time.Second)
	ext := extend(t, m, id.account, resp.Leases[0].LeaseID, "extend-auto", 5*time.Second)
	require.NotNil(t, ext.LeaseID)
	require.Equal(t, clock.Now().Add(5*time.Second).UnixMilli(), int64(ext.LeaseID.Time()))
	require.Equal(t, id.account, ext.AccountID)
	require.Equal(t, id.env, ext.EnvID)
	require.Equal(t, id.fn, ext.FunctionID)
	require.Equal(t, id.app, ext.AppID)

	rel := release(t, m, id.account, *ext.LeaseID, "release-auto")
	require.Equal(t, id.env, rel.EnvID)
	require.Equal(t, constraintapi.CallerLocationItemLease, rel.CreationSource.Location)
	require.Equal(t, int64(0), usageOf(t, m, id.account, sem), "usage should be 0 after auto-release")
}

func TestSemaphoreManualRelease(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "fn:" + id.fn.String(), Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
	setCapacity(t, m, id.account, sem.ID, 1)

	resp := acquire(t, m, acquireRequest(id, clock, config, constraints, "acq-manual"))
	require.Len(t, resp.Leases, 1)
	require.Equal(t, int64(1), usageOf(t, m, id.account, sem))

	clock.Advance(2 * time.Second)
	ext := extend(t, m, id.account, resp.Leases[0].LeaseID, "extend-manual", 5*time.Second)
	require.NotNil(t, ext.LeaseID)

	release(t, m, id.account, *ext.LeaseID, "release-manual")
	require.Equal(t, int64(1), usageOf(t, m, id.account, sem), "usage should still be 1 after manual release (not decremented)")

	require.NoError(t, m.ReleaseSemaphore(context.Background(), id.account, sem.ID, "", "run-1", 1))
	require.Equal(t, int64(0), usageOf(t, m, id.account, sem))
}

func TestSemaphoreWeight(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:" + uuid.NewString(), Weight: 3, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
	setCapacity(t, m, id.account, sem.ID, 5)

	resp := acquire(t, m, acquireRequest(id, clock, config, constraints, "w1"))
	require.Len(t, resp.Leases, 1)
	require.Equal(t, int64(3), usageOf(t, m, id.account, sem), "usage should be 3 (weight=3)")

	clock.Advance(time.Second)
	resp = acquire(t, m, acquireRequest(id, clock, config, constraints, "w2"))
	require.Len(t, resp.Leases, 0, "should fail when remaining < weight")
	require.Equal(t, int64(3), usageOf(t, m, id.account, sem))

	release(t, m, id.account, acquire(t, m, acquireRequest(id, clock, config, constraints, "w1")).Leases[0].LeaseID, "rel-w1")
	require.Equal(t, int64(0), usageOf(t, m, id.account, sem), "release gives back the full weight")
}

func TestSemaphoreZeroCapacity(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:" + uuid.NewString(), Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}

	resp := acquire(t, m, acquireRequest(id, clock, config, constraints, "zero"))
	require.Len(t, resp.Leases, 0, "should grant 0 leases when capacity is 0")
	require.Len(t, resp.ExhaustedConstraints, 1)
	require.Len(t, resp.Usage, 1)
	require.Equal(t, 0, resp.Usage[0].Limit)
}

func TestSemaphoreWithConcurrency(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:" + uuid.NewString(), Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	config.Concurrency = constraintapi.ConcurrencyConfig{AccountConcurrency: 10, FunctionConcurrency: 5}

	constraints := []constraintapi.ConstraintItem{
		{Kind: constraintapi.ConstraintKindConcurrency, Concurrency: &constraintapi.ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount, Mode: enums.ConcurrencyModeStep}},
		{Kind: constraintapi.ConstraintKindConcurrency, Concurrency: &constraintapi.ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn, Mode: enums.ConcurrencyModeStep}},
		semaphoreItem(&sem),
	}
	setCapacity(t, m, id.account, sem.ID, 1)

	resp := acquire(t, m, acquireRequest(id, clock, config, constraints, "multi1"))
	require.Len(t, resp.Leases, 1)

	clock.Advance(time.Second)
	resp = acquire(t, m, acquireRequest(id, clock, config, constraints, "multi2"))
	require.Len(t, resp.Leases, 0, "semaphore should block despite concurrency capacity")
	require.Len(t, resp.ExhaustedConstraints, 1)
	require.Equal(t, constraintapi.ConstraintKindSemaphore, resp.ExhaustedConstraints[0].Kind)
	require.Len(t, resp.Usage, 3)
	require.Equal(t, 1, resp.Usage[0].Used, "the rejected request holds nothing on account concurrency")
}

func TestSemaphoreManager(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	var sm constraintapi.SemaphoreManager = m
	accountID := uuid.New()
	name := fmt.Sprintf("app:%s", uuid.New())
	ctx := context.Background()

	// buildUsage acquires one lease of the given weight so a usage counter
	// exists, the way the Redis test writes the key directly.
	buildUsage := func(t *testing.T, name string, weight int64) {
		id := newIDs()
		id.account = accountID
		sem := constraintapi.SemaphoreConstraint{ID: name, Weight: weight, Release: constraintapi.SemaphoreReleaseManual}
		setCapacity(t, sm, accountID, name, weight)
		resp := acquire(t, m, acquireRequest(id, clock, semaphoreConfig(sem), []constraintapi.ConstraintItem{semaphoreItem(&sem)}, "usage-"+uuid.NewString()))
		require.Len(t, resp.Leases, 1)
	}

	t.Run("set and get capacity", func(t *testing.T) {
		res, err := sm.SetCapacity(ctx, accountID, name, "set-1", 10)
		require.NoError(t, err)
		require.True(t, res.Applied)
		require.Equal(t, int64(10), res.Capacity)

		cap, usage, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(10), cap)
		require.Equal(t, int64(0), usage)
	})

	t.Run("set capacity idempotency", func(t *testing.T) {
		res, err := sm.SetCapacity(ctx, accountID, name, "set-idem", 20)
		require.NoError(t, err)
		require.True(t, res.Applied)
		require.Equal(t, int64(20), res.Capacity)

		res, err = sm.SetCapacity(ctx, accountID, name, "set-idem", 99)
		require.NoError(t, err)
		require.False(t, res.Applied, "replay should not be applied")
		require.Equal(t, int64(20), res.Capacity, "replay returns the cached capacity")

		cap, _, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(20), cap, "idempotent call should not change capacity")
	})

	t.Run("set capacity idempotency expires", func(t *testing.T) {
		name := fmt.Sprintf("app:%s", uuid.New())
		_, err := sm.SetCapacity(ctx, accountID, name, "set-ttl", 1)
		require.NoError(t, err)
		clock.Advance(constraintapi.SemaphoreIdempotencyTTL)
		res, err := sm.SetCapacity(ctx, accountID, name, "set-ttl", 2)
		require.NoError(t, err)
		require.True(t, res.Applied, "the key is forgotten after the TTL")
		require.Equal(t, int64(2), res.Capacity)
	})

	t.Run("adjust capacity", func(t *testing.T) {
		name := fmt.Sprintf("app:%s", uuid.New())
		_, err := sm.SetCapacity(ctx, accountID, name, "adj-set", 5)
		require.NoError(t, err)

		res, err := sm.AdjustCapacity(ctx, accountID, name, "adj-1", 3)
		require.NoError(t, err)
		require.True(t, res.Applied)
		require.Equal(t, int64(8), res.Capacity)

		cap, _, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(8), cap)
	})

	t.Run("adjust capacity upserts", func(t *testing.T) {
		name := fmt.Sprintf("app:%s", uuid.New())
		res, err := sm.AdjustCapacity(ctx, accountID, name, "adj-new", 4)
		require.NoError(t, err)
		require.True(t, res.Applied)
		require.Equal(t, int64(4), res.Capacity)
	})

	t.Run("adjust capacity idempotency", func(t *testing.T) {
		name := fmt.Sprintf("app:%s", uuid.New())
		_, err := sm.SetCapacity(ctx, accountID, name, "adj-idem-set", 5)
		require.NoError(t, err)

		res, err := sm.AdjustCapacity(ctx, accountID, name, "adj-idem-1", 3)
		require.NoError(t, err)
		require.True(t, res.Applied)
		require.Equal(t, int64(8), res.Capacity)

		res, err = sm.AdjustCapacity(ctx, accountID, name, "adj-idem-1", 3)
		require.NoError(t, err)
		require.False(t, res.Applied, "replay should not be applied")
		require.Equal(t, int64(8), res.Capacity, "replay returns the cached capacity")

		cap, _, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(8), cap, "idempotent adjust should not double-add")
	})

	t.Run("release semaphore", func(t *testing.T) {
		name := fmt.Sprintf("fn:%s", uuid.New())
		buildUsage(t, name, 5)

		require.NoError(t, sm.ReleaseSemaphore(ctx, accountID, name, "", "rel-1", 2))

		_, usage, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(3), usage)
	})

	t.Run("release semaphore clamps to zero", func(t *testing.T) {
		name := fmt.Sprintf("fn:%s", uuid.New())
		buildUsage(t, name, 1)

		require.NoError(t, sm.ReleaseSemaphore(ctx, accountID, name, "", "rel-clamp", 5))

		_, usage, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(0), usage, "should clamp to 0")
	})

	t.Run("release semaphore idempotency", func(t *testing.T) {
		name := fmt.Sprintf("fn:%s", uuid.New())
		buildUsage(t, name, 5)

		require.NoError(t, sm.ReleaseSemaphore(ctx, accountID, name, "", "rel-idem", 2))
		require.NoError(t, sm.ReleaseSemaphore(ctx, accountID, name, "", "rel-idem", 2))

		_, usage, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(3), usage, "idempotent release should not double-decrement")
	})

	t.Run("release unknown semaphore is a no-op", func(t *testing.T) {
		name := fmt.Sprintf("fn:%s", uuid.New())
		require.NoError(t, sm.ReleaseSemaphore(ctx, accountID, name, "", "rel-unknown", 2))
		_, usage, err := sm.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(0), usage)
	})

	t.Run("idempotency keys are per account", func(t *testing.T) {
		other := uuid.New()
		res, err := sm.SetCapacity(ctx, other, name, "set-1", 7)
		require.NoError(t, err)
		require.True(t, res.Applied, "another account's key does not replay")
		require.Equal(t, int64(7), res.Capacity)
	})
}

// TestSemaphoreScavengeManualRelease verifies that the scavenger force
// releases manual release semaphores when a constraint lease expires.  a
// crashed executor holding a manual release semaphore would otherwise block
// every future run waiting on that capacity.
func TestSemaphoreScavengeManualRelease(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	hooks := constraintapi.NewConstraintAPIDebugLifecycles()
	m := newMemoryManager(t, clock, memory.WithLifecycles(hooks))
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "fn:" + id.fn.String(), Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
	setCapacity(t, m, id.account, sem.ID, 1)

	resp := acquire(t, m, acquireRequest(id, clock, config, constraints, "scav-manual"))
	require.Len(t, resp.Leases, 1)
	require.Equal(t, int64(1), usageOf(t, m, id.account, sem))

	clock.Advance(3 * time.Second)
	reclaimed, err := m.Scavenge(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, reclaimed, "the lease has not expired yet")
	require.Equal(t, int64(1), usageOf(t, m, id.account, sem))

	clock.Advance(7 * time.Second)
	reclaimed, err = m.Scavenge(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, reclaimed, "scavenger should reclaim 1 expired lease")
	require.Equal(t, int64(0), usageOf(t, m, id.account, sem), "scavenger must force-release manual semaphore to prevent deadlock")

	require.Len(t, hooks.ReleaseCalls, 1, "the sweeper releases through the normal path")
	require.Equal(t, resp.Leases[0].LeaseID, hooks.ReleaseCalls[0].LeaseID)
	require.Equal(t, id.fn, hooks.ReleaseCalls[0].FunctionID)

	reclaimed, err = m.Scavenge(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, reclaimed, "a lease is reclaimed once")

	rel := release(t, m, id.account, resp.Leases[0].LeaseID, "late-release")
	require.Equal(t, uuid.Nil, rel.EnvID, "a late release finds no lease")
	require.Equal(t, int64(0), usageOf(t, m, id.account, sem))
}

// TestSemaphoreEvaluatedKeyHashIsolation verifies that different evaluated key
// hashes get independent usage counters while sharing the same capacity.
func TestSemaphoreEvaluatedKeyHashIsolation(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	semID := "fnkey:" + uuid.NewString()
	setCapacity(t, m, id.account, semID, 2)

	semA := constraintapi.SemaphoreConstraint{ID: semID, EvaluatedKeyHash: "customer-a", Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
	semB := constraintapi.SemaphoreConstraint{ID: semID, EvaluatedKeyHash: "customer-b", Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
	configA, configB := semaphoreConfig(semA), semaphoreConfig(semB)
	constraintsA := []constraintapi.ConstraintItem{semaphoreItem(&semA)}
	constraintsB := []constraintapi.ConstraintItem{semaphoreItem(&semB)}

	resp := acquire(t, m, acquireRequest(id, clock, configA, constraintsA, "iso-a1"))
	require.Len(t, resp.Leases, 1, "customer A first acquire should succeed")

	clock.Advance(time.Second)
	resp = acquire(t, m, acquireRequest(id, clock, configB, constraintsB, "iso-b1"))
	require.Len(t, resp.Leases, 1, "customer B first acquire should succeed")

	require.Equal(t, int64(1), usageOf(t, m, id.account, semA))
	require.Equal(t, int64(1), usageOf(t, m, id.account, semB))

	clock.Advance(time.Second)
	resp = acquire(t, m, acquireRequest(id, clock, configA, constraintsA, "iso-a2"))
	require.Len(t, resp.Leases, 1, "customer A second acquire should succeed")

	clock.Advance(time.Second)
	resp = acquire(t, m, acquireRequest(id, clock, configA, constraintsA, "iso-a3"))
	require.Len(t, resp.Leases, 0, "customer A third acquire should fail — capacity exhausted")

	clock.Advance(time.Second)
	resp = acquire(t, m, acquireRequest(id, clock, configB, constraintsB, "iso-b2"))
	require.Len(t, resp.Leases, 1, "customer B second acquire should still succeed — independent counter")
}

// TestSemaphoreSameEvaluatedKeyHashShared verifies that two acquires with the
// same evaluated key hash share a single counter.
func TestSemaphoreSameEvaluatedKeyHashShared(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	semID := "fnkey:" + uuid.NewString()
	setCapacity(t, m, id.account, semID, 1)

	sem := constraintapi.SemaphoreConstraint{ID: semID, EvaluatedKeyHash: "same-customer", Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}

	resp := acquire(t, m, acquireRequest(id, clock, config, constraints, "same-1"))
	require.Len(t, resp.Leases, 1)

	clock.Advance(time.Second)
	resp = acquire(t, m, acquireRequest(id, clock, config, constraints, "same-2"))
	require.Len(t, resp.Leases, 0, "same evaluated key hash should share counter and exhaust capacity")
	require.Equal(t, int64(1), usageOf(t, m, id.account, sem), "shared counter should be 1")
}

func TestSemaphoreGetCapacity(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	ctx := context.Background()
	accountID := uuid.New()
	name := "app:" + uuid.NewString()

	t.Run("nonexistent returns zero", func(t *testing.T) {
		cap, usage, err := m.GetCapacity(ctx, accountID, name, "some-value")
		require.NoError(t, err)
		require.Equal(t, int64(0), cap)
		require.Equal(t, int64(0), usage)
	})

	t.Run("returns set capacity and usage", func(t *testing.T) {
		id := newIDs()
		id.account = accountID
		sem := constraintapi.SemaphoreConstraint{ID: name, EvaluatedKeyHash: "run-abc", Weight: 42, Release: constraintapi.SemaphoreReleaseManual}
		setCapacity(t, m, accountID, name, 100)
		resp := acquire(t, m, acquireRequest(id, clock, semaphoreConfig(sem), []constraintapi.ConstraintItem{semaphoreItem(&sem)}, "getcap"))
		require.Len(t, resp.Leases, 1)

		cap, usage, err := m.GetCapacity(ctx, accountID, name, "run-abc")
		require.NoError(t, err)
		require.Equal(t, int64(100), cap)
		require.Equal(t, int64(42), usage)

		cap, usage, err = m.GetCapacity(ctx, accountID, name, "")
		require.NoError(t, err)
		require.Equal(t, int64(100), cap, "capacity is shared across evaluated keys")
		require.Equal(t, int64(0), usage, "usage is per evaluated key")
	})
}

func TestSemaphoreAdjustCapacityClampsToZero(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	ctx := context.Background()
	accountID := uuid.New()
	name := "app:" + uuid.NewString()

	_, err := m.SetCapacity(ctx, accountID, name, "set-1", 5)
	require.NoError(t, err)

	res, err := m.AdjustCapacity(ctx, accountID, name, "adj-1", -10)
	require.NoError(t, err)
	require.True(t, res.Applied)
	require.Equal(t, int64(0), res.Capacity, "clamped capacity should be reported in the result")

	cap, _, err := m.GetCapacity(ctx, accountID, name, "")
	require.NoError(t, err)
	require.Equal(t, int64(0), cap, "capacity should clamp to zero, not go negative")
}

func TestSemaphoreAcquireIdempotentReplay(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:" + uuid.NewString(), Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
	setCapacity(t, m, id.account, sem.ID, 5)

	first := acquire(t, m, withAmount(acquireRequest(id, clock, config, constraints, "replay"), 2))
	require.Len(t, first.Leases, 2)
	require.False(t, first.OperationIdempotencyHit)

	second := acquire(t, m, withAmount(acquireRequest(id, clock, config, constraints, "replay"), 2))
	require.True(t, second.OperationIdempotencyHit)
	require.Equal(t, first.Leases, second.Leases, "the replay returns the same leases")
	require.NotEqual(t, first.RequestID, second.RequestID, "the request ID is fresh on replay")
	require.Equal(t, int64(2), usageOf(t, m, id.account, sem), "the replay holds no extra capacity")

	// the fingerprint covers the configuration, so a changed config is a new request
	changed := semaphoreConfig(sem)
	changed.FunctionVersion = 2
	third := acquire(t, m, withAmount(acquireRequest(id, clock, changed, constraints, "replay"), 2))
	require.False(t, third.OperationIdempotencyHit)
	require.Len(t, third.Leases, 2)
	require.Equal(t, int64(4), usageOf(t, m, id.account, sem))

	// past the idempotency window the same request acquires again
	clock.Advance(constraintapi.OperationIdempotencyTTL)
	fourth := acquire(t, m, withAmount(acquireRequest(id, clock, config, constraints, "replay"), 2))
	require.False(t, fourth.OperationIdempotencyHit)
	require.Len(t, fourth.Leases, 1, "only one unit is left")
	require.Len(t, fourth.LimitingConstraints, 1)
}

func TestSemaphoreExhaustedAcquireIsNotCached(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:" + uuid.NewString(), Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}

	resp := acquire(t, m, acquireRequest(id, clock, config, constraints, "uncached"))
	require.Empty(t, resp.Leases)
	resp = acquire(t, m, acquireRequest(id, clock, config, constraints, "uncached"))
	require.Empty(t, resp.Leases)
	require.False(t, resp.OperationIdempotencyHit, "a rejected acquire is never replayed")

	setCapacity(t, m, id.account, sem.ID, 1)
	resp = acquire(t, m, acquireRequest(id, clock, config, constraints, "uncached"))
	require.Len(t, resp.Leases, 1, "the same key acquires once capacity exists")
	require.False(t, resp.OperationIdempotencyHit)
}

func TestSemaphoreReleaseOldLeaseAfterExtend(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:" + uuid.NewString(), Weight: 2, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
	setCapacity(t, m, id.account, sem.ID, 4)

	resp := acquire(t, m, acquireRequest(id, clock, config, constraints, "ext"))
	old := resp.Leases[0].LeaseID

	ext := extend(t, m, id.account, old, "ext-1", 10*time.Second)
	require.NotNil(t, ext.LeaseID)
	require.NotEqual(t, old, *ext.LeaseID)

	again := extend(t, m, id.account, old, "ext-2", 10*time.Second)
	require.Nil(t, again.LeaseID, "the old lease ID is gone after extend")

	rel := release(t, m, id.account, old, "rel-old")
	require.False(t, rel.OperationIdempotencyHit)
	require.Equal(t, int64(2), usageOf(t, m, id.account, sem), "releasing the old ID decrements nothing")

	rel = release(t, m, id.account, old, "rel-old")
	require.False(t, rel.OperationIdempotencyHit, "a no-op release is not cached")

	release(t, m, id.account, *ext.LeaseID, "rel-new")
	require.Equal(t, int64(0), usageOf(t, m, id.account, sem))
	rel = release(t, m, id.account, *ext.LeaseID, "rel-new")
	require.True(t, rel.OperationIdempotencyHit, "a successful release replays")
	require.Equal(t, int64(0), usageOf(t, m, id.account, sem), "the replay decrements nothing")

	rel = release(t, m, id.account, *ext.LeaseID, "rel-new-2")
	require.False(t, rel.OperationIdempotencyHit)
	require.Equal(t, int64(0), usageOf(t, m, id.account, sem), "a second release of a gone lease is a no-op")
}

func TestSemaphoreExtendExpiredLease(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:" + uuid.NewString(), Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
	setCapacity(t, m, id.account, sem.ID, 1)

	resp := acquire(t, m, acquireRequest(id, clock, config, constraints, "expired"))
	lease := resp.Leases[0].LeaseID

	clock.Advance(5*time.Second + time.Millisecond)
	ext := extend(t, m, id.account, lease, "ext-expired", 5*time.Second)
	require.Nil(t, ext.LeaseID, "an expired lease cannot be extended")
	require.Equal(t, id.account, ext.AccountID)
	require.Equal(t, uuid.Nil, ext.EnvID)

	// the lease still holds usage until the sweeper reclaims it
	require.Equal(t, int64(1), usageOf(t, m, id.account, sem))
	reclaimed, err := m.Scavenge(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, reclaimed)
	require.Equal(t, int64(0), usageOf(t, m, id.account, sem))
}

func TestSemaphoreForeignLeaseID(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	other := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:" + uuid.NewString(), Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
	setCapacity(t, m, id.account, sem.ID, 1)

	resp := acquire(t, m, acquireRequest(id, clock, config, constraints, "foreign"))
	lease := resp.Leases[0].LeaseID

	// flip the nonce bytes so the ID belongs to no manager
	foreign := lease
	foreign[14] ^= 0xFF
	foreign[15] ^= 0xFF

	ext := extend(t, m, id.account, foreign, "ext-foreign", 5*time.Second)
	require.Nil(t, ext.LeaseID)
	rel := release(t, m, id.account, foreign, "rel-foreign")
	require.Equal(t, uuid.Nil, rel.EnvID)
	require.Equal(t, int64(1), usageOf(t, m, id.account, sem), "a foreign ID touches nothing")

	rel = release(t, other, id.account, lease, "rel-other")
	require.Equal(t, uuid.Nil, rel.EnvID, "another manager does not know this lease")
	require.Equal(t, int64(1), usageOf(t, m, id.account, sem))
}

func TestLeaseAccountIsolation(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()
	otherAccountID := uuid.New()

	sem := constraintapi.SemaphoreConstraint{ID: "app:" + uuid.NewString(), Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
	setCapacity(t, m, id.account, sem.ID, 2)

	first := acquire(t, m, acquireRequest(id, clock, config, constraints, "account-extend"))
	second := acquire(t, m, acquireRequest(id, clock, config, constraints, "account-release"))
	require.Equal(t, int64(2), usageOf(t, m, id.account, sem))

	ext := extend(t, m, otherAccountID, first.Leases[0].LeaseID, "wrong-account-extend", 5*time.Second)
	require.Nil(t, ext.LeaseID)
	require.Equal(t, uuid.Nil, ext.EnvID)
	rel := release(t, m, otherAccountID, second.Leases[0].LeaseID, "wrong-account-release")
	require.Equal(t, uuid.Nil, rel.EnvID)
	require.Equal(t, int64(2), usageOf(t, m, id.account, sem))

	release(t, m, id.account, first.Leases[0].LeaseID, "right-account-release-first")
	release(t, m, id.account, second.Leases[0].LeaseID, "right-account-release-second")
	require.Equal(t, int64(0), usageOf(t, m, id.account, sem))
}

// TestLeaseIDWithWrongExpiryIsUnknown covers a lease from another manager
// that shares this manager's nonce and names a live seq here.  the expiry in
// the ID does not match the slot's, so it is refused and nothing is released.
func TestLeaseIDWithWrongExpiryIsUnknown(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:" + uuid.NewString(), Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
	setCapacity(t, m, id.account, sem.ID, 1)

	resp := acquire(t, m, acquireRequest(id, clock, config, constraints, "own"))
	lease := resp.Leases[0].LeaseID

	// same nonce, same seq, expiry one millisecond later
	wrong := lease
	require.NoError(t, wrong.SetTime(lease.Time()+1))

	rel := release(t, m, id.account, wrong, "rel-wrong")
	require.Equal(t, uuid.Nil, rel.EnvID, "not this manager's lease")
	require.Equal(t, int64(1), usageOf(t, m, id.account, sem), "nothing was released")
	ext := extend(t, m, id.account, wrong, "ext-wrong", 5*time.Second)
	require.Nil(t, ext.LeaseID)

	rel = release(t, m, id.account, lease, "rel-right")
	require.Equal(t, id.env, rel.EnvID)
	require.Equal(t, int64(0), usageOf(t, m, id.account, sem))
}
