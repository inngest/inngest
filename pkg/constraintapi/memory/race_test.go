package memory_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestRaceAcquireReleaseWithSweeper runs acquire, extend and release from many
// goroutines across accounts and keys while the sweeper runs on a real clock.
// after everything is released or expired and scavenged, usage must be zero
// everywhere and no request may ever have been over granted.
func TestRaceAcquireReleaseWithSweeper(t *testing.T) {
	clock := clockwork.NewRealClock()
	m, err := memory.NewManager(
		memory.WithShardName("race"),
		memory.WithClock(clock),
		memory.WithSweepInterval(5*time.Millisecond),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = m.Close() })

	const (
		accounts   = 3
		keys       = 3
		capacity   = int64(10)
		goroutines = 24
		iterations = 150
	)

	type target struct {
		id  ids
		sem constraintapi.SemaphoreConstraint
	}
	var targets []target
	outstanding := map[string]*atomic.Int64{}
	for a := 0; a < accounts; a++ {
		id := newIDs()
		for k := 0; k < keys; k++ {
			sem := constraintapi.SemaphoreConstraint{ID: fmt.Sprintf("app:%d", k), Weight: int64(k%2 + 1), Release: constraintapi.SemaphoreReleaseAuto}
			setCapacity(t, m, id.account, sem.ID, capacity)
			targets = append(targets, target{id: id, sem: sem})
			outstanding[id.account.String()+sem.ID] = &atomic.Int64{}
		}
	}

	var overshoot, errCount atomic.Int64
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			ctx := context.Background()
			for i := 0; i < iterations; i++ {
				tg := targets[(g+i)%len(targets)]
				sem := tg.sem
				req := acquireRequest(tg.id, clock, semaphoreConfig(sem), []constraintapi.ConstraintItem{semaphoreItem(&sem)}, fmt.Sprintf("g%d-i%d", g, i))
				req.Duration = 2*time.Second + 100*time.Millisecond
				resp, err := m.Acquire(ctx, req)
				if err != nil {
					errCount.Add(1)
					continue
				}
				if len(resp.Leases) == 0 {
					continue
				}
				counter := outstanding[tg.id.account.String()+sem.ID]
				if counter.Add(sem.Weight) > capacity {
					overshoot.Add(1)
				}
				lease := resp.Leases[0].LeaseID

				switch i % 4 {
				case 0:
					ext, err := m.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{IdempotencyKey: fmt.Sprintf("e%d-%d", g, i), AccountID: tg.id.account, LeaseID: lease, Duration: 3 * time.Second})
					if err != nil {
						errCount.Add(1)
						continue
					}
					if ext.LeaseID != nil {
						lease = *ext.LeaseID
					}
					fallthrough
				case 1, 2:
					counter.Add(-sem.Weight)
					if _, err := m.Release(ctx, &constraintapi.CapacityReleaseRequest{IdempotencyKey: fmt.Sprintf("r%d-%d", g, i), AccountID: tg.id.account, LeaseID: lease}); err != nil {
						errCount.Add(1)
					}
				case 3:
					// leave it to expire and be swept
					counter.Add(-sem.Weight)
				}
			}
		}(g)
	}
	wg.Wait()

	require.Zero(t, errCount.Load())
	require.Zero(t, overshoot.Load(), "a semaphore was over granted")

	// let every abandoned lease expire, then reclaim
	time.Sleep(3200 * time.Millisecond)
	_, err = m.Scavenge(context.Background())
	require.NoError(t, err)

	for _, tg := range targets {
		require.Equal(t, int64(0), usageOf(t, m, tg.id.account, tg.sem), "account %s key %s", tg.id.account, tg.sem.ID)
	}
}

// TestRaceIdenticalIdempotentAcquires sends the same request from many
// goroutines at once.  every response carries the same leases and exactly one
// of them performed the mutation.
func TestRaceIdenticalIdempotentAcquires(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:worker", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
	setCapacity(t, m, id.account, sem.ID, 10)

	const goroutines = 20
	responses := make([]*constraintapi.CapacityAcquireResponse, goroutines)
	errs := make([]error, goroutines)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			responses[i], errs[i] = m.Acquire(context.Background(), withAmount(acquireRequest(id, clock, config, constraints, "same"), 2))
		}(i)
	}
	wg.Wait()

	misses := 0
	for i := range responses {
		require.NoError(t, errs[i])
		require.Len(t, responses[i].Leases, 2)
		require.Equal(t, responses[0].Leases, responses[i].Leases, "response %d", i)
		if !responses[i].OperationIdempotencyHit {
			misses++
		}
	}
	require.Equal(t, 1, misses, "exactly one call performed the acquire")
	require.Equal(t, int64(2), usageOf(t, m, id.account, sem))
}

// TestRaceConcurrentAcquiresNeverExceedCapacity mirrors the Redis race test
// for concurrent acquires on one constraint.
func TestRaceConcurrentAcquiresNeverExceedCapacity(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:worker", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
	setCapacity(t, m, id.account, sem.ID, 10)

	var mu sync.Mutex
	var all []constraintapi.CapacityLease
	var wg sync.WaitGroup
	for g := 0; g < 20; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				resp, err := m.Acquire(context.Background(), acquireRequest(id, clock, config, constraints, fmt.Sprintf("c-%d-%d", g, j)))
				require.NoError(t, err)
				mu.Lock()
				all = append(all, resp.Leases...)
				mu.Unlock()
			}
		}(g)
	}
	wg.Wait()

	require.Len(t, all, 10, "exactly the capacity is granted")
	seen := map[ulid.ULID]bool{}
	for _, l := range all {
		require.False(t, seen[l.LeaseID], "duplicate lease ID")
		seen[l.LeaseID] = true
	}
	require.Equal(t, int64(10), usageOf(t, m, id.account, sem))

	for _, l := range all {
		release(t, m, id.account, l.LeaseID, "cleanup-"+l.IdempotencyKey)
	}
	require.Equal(t, int64(0), usageOf(t, m, id.account, sem))
}

// TestRaceAcquireDuringScavenge ports the Redis test that acquires while the
// scavenger reclaims expired leases.
func TestRaceAcquireDuringScavenge(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:worker", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
	setCapacity(t, m, id.account, sem.ID, 5)

	for i := 0; i < 3; i++ {
		resp := acquire(t, m, acquireRequest(id, clock, config, constraints, fmt.Sprintf("expired-%d", i)))
		require.Len(t, resp.Leases, 1)
	}
	clock.Advance(10 * time.Second)

	var wg sync.WaitGroup
	var reclaimed atomic.Int64
	var mu sync.Mutex
	var granted []constraintapi.CapacityLease
	wg.Add(1)
	go func() {
		defer wg.Done()
		n, err := m.Scavenge(context.Background())
		require.NoError(t, err)
		reclaimed.Store(int64(n))
	}()
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, err := m.Acquire(context.Background(), acquireRequest(id, clock, config, constraints, fmt.Sprintf("during-%d", i)))
			require.NoError(t, err)
			mu.Lock()
			granted = append(granted, resp.Leases...)
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	require.Equal(t, int64(3), reclaimed.Load())
	require.LessOrEqual(t, len(granted), 5)
	require.Equal(t, int64(len(granted)), usageOf(t, m, id.account, sem), "usage equals the live lease count")

	require.Equal(t, uuid.Nil, uuid.Nil)
}

// TestRaceThrottleAdmitsExactlyTheCapacity hammers one throttle key from many
// goroutines with mixed amounts.  the TAT moves by CAS and a refused take
// retries with what fits now, so the total admitted must be exactly what one
// caller alone would get: limit plus burst from empty, one more per emission
// interval, and the limit again after a period.
func TestRaceThrottleAdmitsExactlyTheCapacity(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()
	config := constraintapi.ConstraintConfig{FunctionVersion: 1, Throttle: []constraintapi.ThrottleConfig{{Scope: enums.ThrottleScopeFn, KeyExpressionHash: "t", Limit: 50, Burst: 10, Period: 60}}}
	constraints := []constraintapi.ConstraintItem{throttleItem("t", "v")}

	hammer := func(round int) int {
		var admitted atomic.Int64
		var wg sync.WaitGroup
		for g := 0; g < 32; g++ {
			wg.Add(1)
			go func(g int) {
				defer wg.Done()
				for i := 0; i < 20; i++ {
					req := withAmount(acquireRequest(id, clock, config, constraints, fmt.Sprintf("r%d-g%d-i%d", round, g, i)), 1+(g+i)%3)
					resp, err := m.Acquire(context.Background(), req)
					if err != nil {
						t.Error(err)
						return
					}
					admitted.Add(int64(len(resp.Leases)))
				}
			}(g)
		}
		wg.Wait()
		return int(admitted.Load())
	}

	require.Equal(t, 60, hammer(0), "limit plus burst fit from empty")
	check := checkThrottle(t, m, id, config, constraints)
	require.Equal(t, 0, check.AvailableCapacity)
	require.Equal(t, 50, check.Usage[0].Used)

	// emission is 1.2s, so one more unit fits after one interval
	clock.Advance(1200 * time.Millisecond)
	require.Equal(t, 1, hammer(1))

	// a period later the TAT is 12s ahead, which leaves the limit
	clock.Advance(time.Minute)
	require.Equal(t, 50, hammer(2))
	require.Equal(t, 0, checkThrottle(t, m, id, config, constraints).AvailableCapacity)
}

func checkThrottle(t *testing.T, m *memory.Manager, id ids, config constraintapi.ConstraintConfig, constraints []constraintapi.ConstraintItem) *constraintapi.CapacityCheckResponse {
	t.Helper()
	resp, uerr, ierr := m.Check(context.Background(), &constraintapi.CapacityCheckRequest{AccountID: id.account, EnvID: id.env, FunctionID: id.fn, Configuration: config, Constraints: constraints})
	require.NoError(t, uerr)
	require.NoError(t, ierr)
	return resp
}

// TestRaceMixedRollsBackWhenTheThrottleShrinksTheGrant runs a semaphore, a
// rate limit and a throttle together.  commit order is counters, then rate
// limit, then throttle.  the throttle is the bottleneck, so under contention
// it refuses after the semaphore and the rate limit already took, and both
// must be rolled back to exactly what the throttle admitted.
func TestRaceMixedRollsBackWhenTheThrottleShrinksTheGrant(t *testing.T) {
	clock := clockwork.NewFakeClockAt(testStart)
	m := newMemoryManager(t, clock)
	id := newIDs()

	sem := constraintapi.SemaphoreConstraint{ID: "app:worker", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
	config := semaphoreConfig(sem)
	config.RateLimit = []constraintapi.RateLimitConfig{{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "r", Limit: 1000, Period: 60}}
	config.Throttle = []constraintapi.ThrottleConfig{{Scope: enums.ThrottleScopeFn, KeyExpressionHash: "t", Limit: 20, Burst: 0, Period: 60}}
	constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem), rateLimitItem("r", "v"), throttleItem("t", "v")}
	setCapacity(t, m, id.account, sem.ID, 30)

	var mu sync.Mutex
	var leases []ulid.ULID
	var wg sync.WaitGroup
	for g := 0; g < 32; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				resp, err := m.Acquire(context.Background(), withAmount(acquireRequest(id, clock, config, constraints, fmt.Sprintf("g%d-i%d", g, i)), 3))
				if err != nil {
					t.Error(err)
					return
				}
				mu.Lock()
				for _, l := range resp.Leases {
					leases = append(leases, l.LeaseID)
				}
				mu.Unlock()
			}
		}(g)
	}
	wg.Wait()

	require.Len(t, leases, 20, "the throttle admits exactly its limit")
	require.Equal(t, int64(20), usageOf(t, m, id.account, sem), "semaphore takes beyond the throttle were rolled back")

	// the rate limit TAT holds exactly the admitted units.  the count comes
	// from a ceiling over a sum of doubles, so one unit of slack is allowed.
	rl := checkThrottle(t, m, id, config, []constraintapi.ConstraintItem{rateLimitItem("r", "v")})
	require.InDelta(t, 20, rl.Usage[0].Used, 1, "rate limit takes beyond the throttle were rolled back")

	for i, l := range leases {
		release(t, m, id.account, l, fmt.Sprintf("rel-%d", i))
	}
	require.Equal(t, int64(0), usageOf(t, m, id.account, sem))
}
