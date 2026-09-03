package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func syncMapLen(m *sync.Map) int {
	n := 0
	m.Range(func(any, any) bool { n++; return true })
	return n
}

// TestHousekeepingReturnsToEmpty drives time past every TTL and lease lifetime
// and asserts the manager holds nothing but its current slab page.
func TestHousekeepingReturnsToEmpty(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(start)
	m, err := NewManager(WithShardName("hk"), WithClock(clock), WithSweepInterval(0))
	require.NoError(t, err)
	t.Cleanup(func() { _ = m.Close() })

	ctx := context.Background()
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	sem := constraintapi.SemaphoreConstraint{ID: "app:worker", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := constraintapi.ConstraintConfig{
		FunctionVersion: 1,
		Semaphores:      []constraintapi.Semaphore{{ID: sem.ID, Weight: 1}},
		Concurrency:     constraintapi.ConcurrencyConfig{FunctionConcurrency: 10},
		Throttle:        []constraintapi.ThrottleConfig{{Scope: enums.ThrottleScopeFn, KeyExpressionHash: "t", Limit: 100, Burst: 0, Period: 60}},
	}
	constraints := []constraintapi.ConstraintItem{
		{Kind: constraintapi.ConstraintKindSemaphore, Semaphore: &sem},
		{Kind: constraintapi.ConstraintKindConcurrency, Concurrency: &constraintapi.ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeFn}},
		{Kind: constraintapi.ConstraintKindThrottle, Throttle: &constraintapi.ThrottleConstraint{Scope: enums.ThrottleScopeFn, KeyExpressionHash: "t", EvaluatedKeyHash: "v"}},
	}

	_, err = m.SetCapacity(ctx, accountID, sem.ID, "setcap", 10)
	require.NoError(t, err)

	var leases []ulid.ULID
	for i := 0; i < 6; i++ {
		resp, err := m.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			AccountID: accountID, EnvID: envID, FunctionID: fnID,
			IdempotencyKey: "acq-" + string(rune('a'+i)), Constraints: constraints, Amount: 1,
			Configuration: config, LeaseIdempotencyKeys: []string{"lease-" + string(rune('a'+i))},
			CurrentTime: clock.Now(), Duration: 5 * time.Second, MaximumLifetime: time.Hour,
			Source: constraintapi.LeaseSource{Service: constraintapi.ServiceExecutor, Location: constraintapi.CallerLocationItemLease},
		})
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)
		leases = append(leases, resp.Leases[0].LeaseID)
	}

	// release half, extend one, let the rest expire
	for i := 0; i < 3; i++ {
		_, err := m.Release(ctx, &constraintapi.CapacityReleaseRequest{IdempotencyKey: "rel-" + string(rune('a'+i)), AccountID: accountID, LeaseID: leases[i]})
		require.NoError(t, err)
	}
	ext, err := m.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{IdempotencyKey: "ext", AccountID: accountID, LeaseID: leases[3], Duration: 20 * time.Second})
	require.NoError(t, err)
	require.NotNil(t, ext.LeaseID)

	require.NotZero(t, m.acqIdem.len())
	require.NotZero(t, m.relIdem.len())
	require.NotZero(t, m.extIdem.len())
	require.NotZero(t, m.checkIdem.len())
	require.NotZero(t, m.semIdem.len())
	require.NotZero(t, m.expiry.bucketCount())
	require.NotZero(t, syncMapLen(&m.sems))
	require.Equal(t, 1, syncMapLen(&m.gcra), "the throttle has a TAT cell")

	clock.Advance(constraintapi.MaximumLeaseLifetime + time.Minute)
	reclaimed, err := m.Scavenge(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, reclaimed, "the two abandoned leases and the extended one")

	_, usage, err := m.GetCapacity(ctx, accountID, sem.ID, "")
	require.NoError(t, err)
	require.Equal(t, int64(0), usage)

	// bring capacity to zero so the capacity cell can be dropped too
	_, err = m.AdjustCapacity(ctx, accountID, sem.ID, "adj", -10)
	require.NoError(t, err)
	clock.Advance(constraintapi.SemaphoreIdempotencyTTL + time.Second)

	reclaimed, err = m.Scavenge(ctx)
	require.NoError(t, err)
	require.Zero(t, reclaimed)

	require.Zero(t, m.acqIdem.len())
	require.Zero(t, m.relIdem.len())
	require.Zero(t, m.extIdem.len())
	require.Zero(t, m.checkIdem.len())
	require.Zero(t, m.semIdem.len())
	require.Zero(t, m.expiry.bucketCount())
	require.Zero(t, syncMapLen(&m.sems), "zero valued semaphore and concurrency cells are dropped")
	require.Zero(t, syncMapLen(&m.gcra), "an expired TAT cell is dropped")
	require.LessOrEqual(t, m.slab.pageCount(), slabShards, "only the shards' current allocation pages stay")

	require.Zero(t, syncMapLen(&m.sets), "manual scavenging drops an unused set")

	// a dropped cell is recreated on use
	_, err = m.SetCapacity(ctx, accountID, sem.ID, "setcap-2", 1)
	require.NoError(t, err)
	capacity, _, err := m.GetCapacity(ctx, accountID, sem.ID, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), capacity)
}

// TestBackgroundSweeper checks the goroutine reclaims leases on its own and
// that Close stops it.
func TestBackgroundSweeper(t *testing.T) {
	m, err := NewManager(WithShardName("bg"), WithSweepInterval(5*time.Millisecond))
	require.NoError(t, err)

	ctx := context.Background()
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	sem := constraintapi.SemaphoreConstraint{ID: "app:worker", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := constraintapi.ConstraintConfig{FunctionVersion: 1, Semaphores: []constraintapi.Semaphore{{ID: sem.ID, Weight: 1}}}
	constraints := []constraintapi.ConstraintItem{{Kind: constraintapi.ConstraintKindSemaphore, Semaphore: &sem}}
	_, err = m.SetCapacity(ctx, accountID, sem.ID, "setcap", 1)
	require.NoError(t, err)

	resp, err := m.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
		AccountID: accountID, EnvID: envID, FunctionID: fnID,
		IdempotencyKey: "acq", Constraints: constraints, Amount: 1,
		Configuration: config, LeaseIdempotencyKeys: []string{"lease"},
		CurrentTime: time.Now(), Duration: 2*time.Second + 50*time.Millisecond, MaximumLifetime: time.Hour,
		Source: constraintapi.LeaseSource{Service: constraintapi.ServiceExecutor, Location: constraintapi.CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Len(t, resp.Leases, 1)

	require.Eventually(t, func() bool {
		_, usage, err := m.GetCapacity(ctx, accountID, sem.ID, "")
		return err == nil && usage == 0
	}, 5*time.Second, 10*time.Millisecond, "the sweeper reclaims the expired lease")

	require.NoError(t, m.Close())
	require.NoError(t, m.Close(), "close is idempotent")
	select {
	case <-m.done:
	case <-time.After(time.Second):
		t.Fatal("the sweeper did not stop")
	}
}
