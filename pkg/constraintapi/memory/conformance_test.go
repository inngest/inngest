package memory_test

import (
	"context"
	"fmt"
	"strings"
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

// the conformance suite runs one script against the Redis manager on
// miniredis and against the memory manager, then compares what each step
// recorded.  Redis is the oracle.  scripts avoid the documented divergences,
// for example they scavenge before reading usage that expired leases hold.

type backend struct {
	name  string
	cm    constraintapi.CapacityManager
	sm    constraintapi.SemaphoreManager
	clock *clockwork.FakeClock
	hooks *constraintapi.ConstraintApiDebugLifecycles

	// advance moves every clock the backend has, including miniredis TTLs.
	advance func(d time.Duration)
	// scavenge reclaims expired leases and returns how many.
	scavenge func(t *testing.T) int
}

func newRedisBackend(t *testing.T) *backend {
	t.Helper()
	r, rc := newMiniredisClient(t)
	clock := clockwork.NewFakeClockAt(testStart)
	r.SetTime(clock.Now())
	hooks := constraintapi.NewConstraintAPIDebugLifecycles()

	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithShardName("redis"),
		constraintapi.WithClient(rc),
		constraintapi.WithClock(clock),
		constraintapi.WithLifecycles(hooks),
	)
	require.NoError(t, err)

	return &backend{
		name:  "redis",
		cm:    cm,
		sm:    constraintapi.NewRedisSemaphoreManager(rc),
		clock: clock,
		hooks: hooks,
		advance: func(d time.Duration) {
			clock.Advance(d)
			r.FastForward(d)
			r.SetTime(clock.Now())
		},
		scavenge: func(t *testing.T) int {
			res, err := cm.Scavenge(context.Background())
			require.NoError(t, err)
			return res.ReclaimedLeases
		},
	}
}

func newMemoryBackend(t *testing.T) *backend {
	t.Helper()
	clock := clockwork.NewFakeClockAt(testStart)
	hooks := constraintapi.NewConstraintAPIDebugLifecycles()
	m := newMemoryManager(t, clock, memory.WithLifecycles(hooks))

	return &backend{
		name:    "memory",
		cm:      m,
		sm:      m,
		clock:   clock,
		hooks:   hooks,
		advance: clock.Advance,
		scavenge: func(t *testing.T) int {
			n, err := m.Scavenge(context.Background())
			require.NoError(t, err)
			return n
		},
	}
}

type step struct {
	label string
	value any
}

// recorder collects the observations a script makes so the harness can diff
// them between backends.
type recorder struct {
	steps []step
}

func (r *recorder) rec(label string, v any) {
	r.steps = append(r.steps, step{label: label, value: normalize(v)})
}

type normLease struct {
	Key      string
	ExpiryMS int64
}

type normUsage struct {
	Constraint string
	Used       int
	Limit      int
}

type normAcquire struct {
	Leases       []normLease
	Limiting     []string
	Exhausted    []string
	Usage        []normUsage
	RetryAfterMS int64
	Hit          bool
}

type normExtend struct {
	HasLease bool
	ExpiryMS int64
	Env, Fn  uuid.UUID
	App      uuid.UUID
	Usage    []normUsage
	Hit      bool
}

type normRelease struct {
	Env, Fn, App uuid.UUID
	Source       constraintapi.LeaseSource
	Usage        []normUsage
	Hit          bool
}

type normCheck struct {
	Available    int
	Limiting     []string
	Exhausted    []string
	Usage        []normUsage
	RetryAfterMS int64
}

func constraintName(ci constraintapi.ConstraintItem) string {
	return fmt.Sprintf("%s{%s}", ci.Kind, ci.PrettyString())
}

func constraintNames(items []constraintapi.ConstraintItem) []string {
	out := make([]string, 0, len(items))
	for _, ci := range items {
		out = append(out, constraintName(ci))
	}
	return out
}

func normUsages(items []constraintapi.ConstraintUsage) []normUsage {
	out := make([]normUsage, 0, len(items))
	for _, u := range items {
		out = append(out, normUsage{Constraint: constraintName(u.Constraint), Used: u.Used, Limit: u.Limit})
	}
	return out
}

func retryMS(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func normalize(v any) any {
	switch r := v.(type) {
	case *constraintapi.CapacityAcquireResponse:
		n := normAcquire{
			Limiting:     constraintNames(r.LimitingConstraints),
			Exhausted:    constraintNames(r.ExhaustedConstraints),
			Usage:        normUsages(r.Usage),
			RetryAfterMS: retryMS(r.RetryAfter),
			Hit:          r.OperationIdempotencyHit,
		}
		for _, l := range r.Leases {
			n.Leases = append(n.Leases, normLease{Key: l.IdempotencyKey, ExpiryMS: int64(l.LeaseID.Time())})
		}
		return n
	case *constraintapi.CapacityExtendLeaseResponse:
		n := normExtend{Env: r.EnvID, Fn: r.FunctionID, App: r.AppID, Usage: normUsages(r.Usage), Hit: r.OperationIdempotencyHit}
		if r.LeaseID != nil {
			n.HasLease = true
			n.ExpiryMS = int64(r.LeaseID.Time())
		}
		return n
	case *constraintapi.CapacityReleaseResponse:
		return normRelease{Env: r.EnvID, Fn: r.FunctionID, App: r.AppID, Source: r.CreationSource, Usage: normUsages(r.Usage), Hit: r.OperationIdempotencyHit}
	case *constraintapi.CapacityCheckResponse:
		return normCheck{
			Available:    r.AvailableCapacity,
			Limiting:     constraintNames(r.LimitingConstraints),
			Exhausted:    constraintNames(r.ExhaustedConstraints),
			Usage:        normUsages(r.Usage),
			RetryAfterMS: retryMS(r.RetryAfter),
		}
	default:
		return v
	}
}

// runOnBoth runs script against each backend and diffs the recorded steps.
// the diff is skipped when either run failed on its own assertions.
func runOnBoth(t *testing.T, script func(t *testing.T, b *backend, rec *recorder)) {
	t.Helper()
	var recs []*recorder
	failed := false
	for _, mk := range []func(*testing.T) *backend{newRedisBackend, newMemoryBackend} {
		b := mk(t)
		rec := &recorder{}
		ok := t.Run(b.name, func(t *testing.T) { script(t, b, rec) })
		failed = failed || !ok
		recs = append(recs, rec)
	}
	if failed {
		return
	}
	compareSteps(t, recs[0].steps, recs[1].steps)
}

func compareSteps(t *testing.T, redis, mem []step) {
	t.Helper()
	require.Len(t, mem, len(redis), "the memory run recorded a different number of steps")
	for i := range redis {
		require.Equal(t, redis[i].label, mem[i].label, "step %d label", i)
		want, got := redis[i].value, mem[i].value
		switch w := want.(type) {
		case normAcquire:
			g, ok := got.(normAcquire)
			require.True(t, ok, "step %q type", redis[i].label)
			require.InDelta(t, w.RetryAfterMS, g.RetryAfterMS, 1, "step %q retry after", redis[i].label)
			w.RetryAfterMS, g.RetryAfterMS = 0, 0
			require.Equal(t, w, g, "step %q", redis[i].label)
		case normCheck:
			g, ok := got.(normCheck)
			require.True(t, ok, "step %q type", redis[i].label)
			require.InDelta(t, w.RetryAfterMS, g.RetryAfterMS, 1, "step %q retry after", redis[i].label)
			w.RetryAfterMS, g.RetryAfterMS = 0, 0
			require.Equal(t, w, g, "step %q", redis[i].label)
		default:
			require.Equal(t, want, got, "step %q", redis[i].label)
		}
	}
}

// scriptIDs derives the account, env, function and app IDs from the script
// name, minus the backend suffix, so both backends see identical IDs and
// identical fn: semaphore names.
func scriptIDs(t *testing.T) ids {
	name := t.Name()
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[:i]
	}
	mk := func(part string) uuid.UUID { return uuid.NewSHA1(uuid.NameSpaceOID, []byte(name+":"+part)) }
	return ids{account: mk("account"), env: mk("env"), fn: mk("fn"), app: mk("app")}
}

func capacityOf(t *testing.T, b *backend, accountID uuid.UUID, sem constraintapi.SemaphoreConstraint) [2]int64 {
	t.Helper()
	c, u, err := b.sm.GetCapacity(context.Background(), accountID, sem.ID, sem.EvaluatedKeyHash)
	require.NoError(t, err)
	return [2]int64{c, u}
}

func concurrencyItem(scope enums.ConcurrencyScope) constraintapi.ConstraintItem {
	return constraintapi.ConstraintItem{Kind: constraintapi.ConstraintKindConcurrency, Concurrency: &constraintapi.ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: scope}}
}

func customConcurrencyItem(scope enums.ConcurrencyScope, exprHash, evaluated string) constraintapi.ConstraintItem {
	return constraintapi.ConstraintItem{Kind: constraintapi.ConstraintKindConcurrency, Concurrency: &constraintapi.ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: scope, KeyExpressionHash: exprHash, EvaluatedKeyHash: evaluated}}
}

func throttleItem(exprHash, evaluated string) constraintapi.ConstraintItem {
	return constraintapi.ConstraintItem{Kind: constraintapi.ConstraintKindThrottle, Throttle: &constraintapi.ThrottleConstraint{Scope: enums.ThrottleScopeFn, KeyExpressionHash: exprHash, EvaluatedKeyHash: evaluated}}
}

func rateLimitItem(exprHash, evaluated string) constraintapi.ConstraintItem {
	return constraintapi.ConstraintItem{Kind: constraintapi.ConstraintKindRateLimit, RateLimit: &constraintapi.RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: exprHash, EvaluatedKeyHash: evaluated}}
}

func TestConformanceSemaphore(t *testing.T) {
	t.Run("auto release", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			sem := constraintapi.SemaphoreConstraint{ID: "app:worker", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
			config := semaphoreConfig(sem)
			constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
			setCapacity(t, b.sm, id.account, sem.ID, 2)

			var leases []ulid.ULID
			for i := 0; i < 3; i++ {
				resp := acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, fmt.Sprintf("acq-%d", i)))
				rec.rec(fmt.Sprintf("acquire %d", i), resp)
				for _, l := range resp.Leases {
					leases = append(leases, l.LeaseID)
				}
				b.advance(time.Second)
			}
			require.Len(t, leases, 2)
			rec.rec("capacity after acquires", capacityOf(t, b, id.account, sem))

			rec.rec("release first", release(t, b.cm, id.account, leases[0], "rel-0"))
			rec.rec("capacity after release", capacityOf(t, b, id.account, sem))
			rec.rec("acquire after release", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq-3")))
			rec.rec("release hook count", len(b.hooks.ReleaseCalls))
			rec.rec("acquire hook count", len(b.hooks.AcquireCalls))
		})
	})

	t.Run("manual release", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			sem := constraintapi.SemaphoreConstraint{ID: "fn:" + id.fn.String(), Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
			config := semaphoreConfig(sem)
			constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
			setCapacity(t, b.sm, id.account, sem.ID, 1)

			resp := acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq"))
			rec.rec("acquire", resp)
			rec.rec("release lease", release(t, b.cm, id.account, resp.Leases[0].LeaseID, "rel"))
			rec.rec("capacity after lease release", capacityOf(t, b, id.account, sem))
			require.NoError(t, b.sm.ReleaseSemaphore(context.Background(), id.account, sem.ID, "", "run-1", 1))
			rec.rec("capacity after manual release", capacityOf(t, b, id.account, sem))
			require.NoError(t, b.sm.ReleaseSemaphore(context.Background(), id.account, sem.ID, "", "run-1", 1))
			rec.rec("capacity after replayed manual release", capacityOf(t, b, id.account, sem))
		})
	})

	t.Run("weight and partial grant", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			sem := constraintapi.SemaphoreConstraint{ID: "app:worker", Weight: 2, Release: constraintapi.SemaphoreReleaseAuto}
			config := semaphoreConfig(sem)
			constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
			setCapacity(t, b.sm, id.account, sem.ID, 7)

			resp := acquire(t, b.cm, withAmount(acquireRequest(id, b.clock, config, constraints, "acq"), 5))
			rec.rec("partial grant", resp)
			require.Len(t, resp.Leases, 3, "three units of weight two fit under seven")
			require.Len(t, resp.LimitingConstraints, 1)
			require.Len(t, resp.ExhaustedConstraints, 1, "one unit left is below the weight")
			require.True(t, resp.RetryAfter.IsZero(), "semaphore exhaustion has no retry after")
			rec.rec("capacity", capacityOf(t, b, id.account, sem))

			rec.rec("exhausted", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq-2")))
		})
	})

	t.Run("evaluated key isolation", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			semID := "fnkey:shared"
			setCapacity(t, b.sm, id.account, semID, 2)
			semA := constraintapi.SemaphoreConstraint{ID: semID, EvaluatedKeyHash: "a", Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
			semB := constraintapi.SemaphoreConstraint{ID: semID, EvaluatedKeyHash: "b", Weight: 1, Release: constraintapi.SemaphoreReleaseManual}

			for i, sem := range []constraintapi.SemaphoreConstraint{semA, semB, semA, semA, semB} {
				resp := acquire(t, b.cm, acquireRequest(id, b.clock, semaphoreConfig(sem), []constraintapi.ConstraintItem{semaphoreItem(&sem)}, fmt.Sprintf("acq-%d", i)))
				rec.rec(fmt.Sprintf("acquire %d", i), resp)
			}
			rec.rec("capacity a", capacityOf(t, b, id.account, semA))
			rec.rec("capacity b", capacityOf(t, b, id.account, semB))
		})
	})

	t.Run("idempotent replay and rejected not cached", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			sem := constraintapi.SemaphoreConstraint{ID: "app:worker", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
			config := semaphoreConfig(sem)
			constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}

			rec.rec("rejected", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "key")))
			rec.rec("rejected again", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "key")))

			setCapacity(t, b.sm, id.account, sem.ID, 3)
			first := acquire(t, b.cm, withAmount(acquireRequest(id, b.clock, config, constraints, "key"), 2))
			rec.rec("granted", first)
			second := acquire(t, b.cm, withAmount(acquireRequest(id, b.clock, config, constraints, "key"), 2))
			rec.rec("replayed", second)
			require.Equal(t, first.Leases, second.Leases)
			rec.rec("capacity", capacityOf(t, b, id.account, sem))

			b.advance(constraintapi.OperationIdempotencyTTL)
			rec.rec("after ttl", acquire(t, b.cm, withAmount(acquireRequest(id, b.clock, config, constraints, "key"), 2)))
		})
	})

	t.Run("scavenge force releases manual semaphore", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			sem := constraintapi.SemaphoreConstraint{ID: "fn:" + id.fn.String(), Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
			config := semaphoreConfig(sem)
			constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
			setCapacity(t, b.sm, id.account, sem.ID, 1)

			resp := acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq"))
			rec.rec("acquire", resp)
			b.advance(10 * time.Second)
			rec.rec("reclaimed", b.scavenge(t))
			rec.rec("capacity", capacityOf(t, b, id.account, sem))
			rec.rec("release hooks", len(b.hooks.ReleaseCalls))
			rec.rec("late release", release(t, b.cm, id.account, resp.Leases[0].LeaseID, "late"))
		})
	})

	t.Run("extend then release", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			sem := constraintapi.SemaphoreConstraint{ID: "app:worker", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
			config := semaphoreConfig(sem)
			constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
			setCapacity(t, b.sm, id.account, sem.ID, 1)

			resp := acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq"))
			old := resp.Leases[0].LeaseID
			b.advance(2 * time.Second)

			ext := extend(t, b.cm, id.account, old, "ext", 10*time.Second)
			rec.rec("extend", ext)
			rec.rec("extend replay", extend(t, b.cm, id.account, old, "ext", 10*time.Second))
			rec.rec("extend old id", extend(t, b.cm, id.account, old, "ext-2", 10*time.Second))
			rec.rec("release old id", release(t, b.cm, id.account, old, "rel-old"))
			rec.rec("release old id again", release(t, b.cm, id.account, old, "rel-old"))
			rec.rec("capacity held", capacityOf(t, b, id.account, sem))

			b.advance(6 * time.Second)
			rec.rec("reclaimed before new expiry", b.scavenge(t))
			rec.rec("release new id", release(t, b.cm, id.account, *ext.LeaseID, "rel-new"))
			rec.rec("release new id replay", release(t, b.cm, id.account, *ext.LeaseID, "rel-new"))
			rec.rec("capacity released", capacityOf(t, b, id.account, sem))

			foreign := *ext.LeaseID
			foreign[14] ^= 0xFF
			foreign[15] ^= 0xFF
			rec.rec("foreign extend", extend(t, b.cm, id.account, foreign, "ext-f", 10*time.Second))
			rec.rec("foreign release", release(t, b.cm, id.account, foreign, "rel-f"))
		})
	})

	t.Run("extend expired", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			sem := constraintapi.SemaphoreConstraint{ID: "app:worker", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
			config := semaphoreConfig(sem)
			constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem)}
			setCapacity(t, b.sm, id.account, sem.ID, 1)

			resp := acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq"))
			b.advance(5*time.Second + time.Millisecond)
			rec.rec("extend expired", extend(t, b.cm, id.account, resp.Leases[0].LeaseID, "ext", 5*time.Second))
			rec.rec("reclaimed", b.scavenge(t))
			rec.rec("capacity", capacityOf(t, b, id.account, sem))
		})
	})
}

func TestConformanceConcurrency(t *testing.T) {
	t.Run("account and function", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			config := constraintapi.ConstraintConfig{FunctionVersion: 1, Concurrency: constraintapi.ConcurrencyConfig{AccountConcurrency: 3, FunctionConcurrency: 2}}
			constraints := []constraintapi.ConstraintItem{concurrencyItem(enums.ConcurrencyScopeFn), concurrencyItem(enums.ConcurrencyScopeAccount)}

			var leases []ulid.ULID
			for i := 0; i < 3; i++ {
				resp := acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, fmt.Sprintf("acq-%d", i)))
				rec.rec(fmt.Sprintf("acquire %d", i), resp)
				for _, l := range resp.Leases {
					leases = append(leases, l.LeaseID)
				}
			}
			require.Len(t, leases, 2)
			last := acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq-x"))
			require.Equal(t, b.clock.Now().Add(constraintapi.ConcurrencyLimitRetryAfter).UnixMilli(), last.RetryAfter.UnixMilli())

			b.advance(time.Second)
			ext := extend(t, b.cm, id.account, leases[0], "ext", 10*time.Second)
			rec.rec("extend usage is concurrency only with the stored limit", ext)
			rec.rec("release", release(t, b.cm, id.account, leases[1], "rel-1"))
			rec.rec("acquire after release", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq-3")))

			b.advance(11 * time.Second)
			rec.rec("reclaimed", b.scavenge(t))
			rec.rec("acquire after scavenge", acquire(t, b.cm, withAmount(acquireRequest(id, b.clock, config, constraints, "acq-4"), 3)))
			rec.rec("release hooks", len(b.hooks.ReleaseCalls))
		})
	})

	t.Run("custom key", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			config := constraintapi.ConstraintConfig{FunctionVersion: 1, Concurrency: constraintapi.ConcurrencyConfig{
				FunctionConcurrency:   10,
				CustomConcurrencyKeys: []constraintapi.CustomConcurrencyLimit{{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeFn, Limit: 1, KeyExpressionHash: "expr"}},
			}}
			a := []constraintapi.ConstraintItem{concurrencyItem(enums.ConcurrencyScopeFn), customConcurrencyItem(enums.ConcurrencyScopeFn, "expr", "user-a")}
			bb := []constraintapi.ConstraintItem{concurrencyItem(enums.ConcurrencyScopeFn), customConcurrencyItem(enums.ConcurrencyScopeFn, "expr", "user-b")}

			rec.rec("a1", acquire(t, b.cm, acquireRequest(id, b.clock, config, a, "a1")))
			rec.rec("a2", acquire(t, b.cm, acquireRequest(id, b.clock, config, a, "a2")))
			rec.rec("b1", acquire(t, b.cm, acquireRequest(id, b.clock, config, bb, "b1")))
		})
	})

	t.Run("check", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			config := constraintapi.ConstraintConfig{FunctionVersion: 1, Concurrency: constraintapi.ConcurrencyConfig{AccountConcurrency: 5, FunctionConcurrency: 2}}
			constraints := []constraintapi.ConstraintItem{concurrencyItem(enums.ConcurrencyScopeFn), concurrencyItem(enums.ConcurrencyScopeAccount)}
			check := func(label string) {
				resp, uerr, ierr := b.cm.Check(context.Background(), &constraintapi.CapacityCheckRequest{AccountID: id.account, EnvID: id.env, FunctionID: id.fn, Configuration: config, Constraints: constraints})
				require.NoError(t, uerr)
				require.NoError(t, ierr)
				rec.rec(label, resp)
			}
			check("empty")
			acquire(t, b.cm, withAmount(acquireRequest(id, b.clock, config, constraints, "acq"), 2))
			check("exhausted")
		})
	})
}

func TestConformanceGCRA(t *testing.T) {
	t.Run("throttle", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			config := constraintapi.ConstraintConfig{FunctionVersion: 1, Throttle: []constraintapi.ThrottleConfig{{Scope: enums.ThrottleScopeFn, KeyExpressionHash: "t", Limit: 2, Burst: 1, Period: 60}}}
			constraints := []constraintapi.ConstraintItem{throttleItem("t", "v")}

			for i := 0; i < 4; i++ {
				rec.rec(fmt.Sprintf("acquire %d", i), acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, fmt.Sprintf("acq-%d", i))))
			}
			b.advance(30 * time.Second)
			rec.rec("after half period", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq-5")))
			rec.rec("multi", acquire(t, b.cm, withAmount(acquireRequest(id, b.clock, config, constraints, "acq-6"), 3)))

			resp, uerr, ierr := b.cm.Check(context.Background(), &constraintapi.CapacityCheckRequest{AccountID: id.account, EnvID: id.env, FunctionID: id.fn, Configuration: config, Constraints: constraints})
			require.NoError(t, uerr)
			require.NoError(t, ierr)
			rec.rec("check", resp)
		})
	})

	t.Run("rate limit", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			config := constraintapi.ConstraintConfig{FunctionVersion: 1, RateLimit: []constraintapi.RateLimitConfig{{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "r", Limit: 10, Period: 60}}}
			constraints := []constraintapi.ConstraintItem{rateLimitItem("r", "v")}

			for i := 0; i < 3; i++ {
				rec.rec(fmt.Sprintf("acquire %d", i), acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, fmt.Sprintf("acq-%d", i))))
			}
			b.advance(7 * time.Second)
			rec.rec("after emission", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq-3")))

			resp, uerr, ierr := b.cm.Check(context.Background(), &constraintapi.CapacityCheckRequest{AccountID: id.account, EnvID: id.env, FunctionID: id.fn, Configuration: config, Constraints: constraints})
			require.NoError(t, uerr)
			require.NoError(t, ierr)
			rec.rec("check", resp)
		})
	})

	t.Run("non divisible limit", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			config := constraintapi.ConstraintConfig{FunctionVersion: 1, Throttle: []constraintapi.ThrottleConfig{{Scope: enums.ThrottleScopeFn, KeyExpressionHash: "t", Limit: 7, Burst: 0, Period: 60}}}
			constraints := []constraintapi.ConstraintItem{throttleItem("t", "v")}

			rec.rec("first", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq-0")))
			b.advance(8*time.Second + 571*time.Millisecond)
			rec.rec("just before emission", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq-1")))
			b.advance(time.Millisecond)
			rec.rec("at emission", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq-2")))
		})
	})

	t.Run("skip gcra for a seen idempotency key", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			config := constraintapi.ConstraintConfig{FunctionVersion: 1, Throttle: []constraintapi.ThrottleConfig{{Scope: enums.ThrottleScopeFn, KeyExpressionHash: "t", Limit: 1, Burst: 0, Period: 60}}}
			constraints := []constraintapi.ConstraintItem{throttleItem("t", "v")}

			first := acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "k1"))
			rec.rec("first", first)
			rec.rec("throttled", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "k2")))
			// the lease idempotency key of the first request was recorded, so a
			// later acquire that uses it as its request key skips the throttle
			rec.rec("skipped", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, first.Leases[0].IdempotencyKey)))
			rec.rec("request key skipped", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "k1")))

			b.advance(constraintapi.ConstraintCheckIdempotencyTTL)
			rec.rec("forgotten", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "k2-lease")))
		})
	})

	t.Run("mixed", func(t *testing.T) {
		runOnBoth(t, func(t *testing.T, b *backend, rec *recorder) {
			id := scriptIDs(t)
			sem := constraintapi.SemaphoreConstraint{ID: "app:worker", Weight: 1, Release: constraintapi.SemaphoreReleaseAuto}
			config := semaphoreConfig(sem)
			config.Concurrency = constraintapi.ConcurrencyConfig{FunctionConcurrency: 2}
			config.Throttle = []constraintapi.ThrottleConfig{{Scope: enums.ThrottleScopeFn, KeyExpressionHash: "t", Limit: 5, Burst: 0, Period: 60}}
			constraints := []constraintapi.ConstraintItem{semaphoreItem(&sem), concurrencyItem(enums.ConcurrencyScopeFn), throttleItem("t", "v")}
			setCapacity(t, b.sm, id.account, sem.ID, 3)

			rec.rec("partial", acquire(t, b.cm, withAmount(acquireRequest(id, b.clock, config, constraints, "acq"), 4)))
			rec.rec("exhausted", acquire(t, b.cm, acquireRequest(id, b.clock, config, constraints, "acq-2")))
			rec.rec("capacity", capacityOf(t, b, id.account, sem))
		})
	})
}
