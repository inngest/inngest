package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const replNonce = 0xC0FFEE

var replStart = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func newReplPrimary(t testing.TB, clock clockwork.Clock) (*Manager, *Log) {
	t.Helper()
	l := NewLog()
	m, err := NewManager(
		WithShardName("primary"),
		WithClock(clock),
		WithSweepInterval(0),
		WithNonce(replNonce),
		WithLog(l),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = m.Close() })
	return m, l
}

func newReplFollower(t testing.TB, clock clockwork.Clock) *Manager {
	t.Helper()
	m, err := NewManager(
		WithShardName("follower"),
		WithClock(clock),
		WithSweepInterval(0),
		WithNonce(replNonce),
		WithFollower(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = m.Close() })
	return m
}

// replPump is the one goroutine that drains a log.  it gives every drained
// record to each sink in order and then acknowledges the handle, the way a
// primary with followers that never fail does.  do runs fn on the pump
// goroutine between two drains, which is where a follower joins.
type replPump struct {
	t     testing.TB
	l     *Log
	sinks []*Manager
	jobs  chan func()
	stop  chan struct{}
	done  chan struct{}
	buf   []Record
	n     int
}

func startReplPump(t testing.TB, l *Log, sinks ...*Manager) *replPump {
	p := &replPump{
		t:     t,
		l:     l,
		sinks: sinks,
		jobs:  make(chan func()),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	go p.run()
	return p
}

func (p *replPump) run() {
	defer close(p.done)
	tick := time.NewTicker(200 * time.Microsecond)
	defer tick.Stop()
	for {
		select {
		case <-p.l.Wake():
		case <-tick.C:
		case fn := <-p.jobs:
			fn()
			continue
		case <-p.stop:
			p.cycle()
			return
		}
		p.cycle()
	}
}

func (p *replPump) cycle() {
	recs, b := p.l.Drain(p.buf[:0])
	for _, s := range p.sinks {
		if err := s.Apply(recs); err != nil {
			p.t.Errorf("apply failed: %v", err)
		}
	}
	p.buf = recs
	b.AckAll()

	p.n++
	if p.n%64 == 0 {
		for _, s := range p.sinks {
			followerHousekeep(s)
		}
	}
}

// followerHousekeep runs every step of the follower housekeeping once.
func followerHousekeep(m *Manager) {
	for range FollowerHousekeepSteps {
		m.FollowerHousekeepStep()
	}
}

func (p *replPump) do(fn func()) {
	done := make(chan struct{})
	p.jobs <- func() {
		fn()
		close(done)
	}
	<-done
}

// close drains one last time and stops the pump.  nothing may append after.
func (p *replPump) close() {
	close(p.stop)
	<-p.done
}

// canon is the state of a manager as sorted strings, in the form two
// replicas of one shard must agree on when the second got every record from
// the log.  it is canonState and the live acquire and extend idempotency
// records.
func canon(t testing.TB, m *Manager) []string {
	t.Helper()
	out := canonState(t, m)
	nowMS := m.nowMS()
	m.acqIdem.each(nowMS, func(key uint64, rec *acquireRecord, expiresAtMS int64) {
		if len(rec.seqs) == 0 {
			return
		}
		out = append(out, fmt.Sprintf("acquire %x leases %v set %d exp %d req %d granted %d until %d", key, rec.seqs, rec.set.id, rec.expiresAtMS, rec.requested, rec.granted, expiresAtMS))
	})
	m.extIdem.each(nowMS, func(key uint64, res *constraintapi.CapacityExtendLeaseResponse, expiresAtMS int64) {
		if res.LeaseID == nil {
			return
		}
		out = append(out, fmt.Sprintf("extend %x lease %s until %d", key, res.LeaseID, expiresAtMS))
	})
	sort.Strings(out)
	return out
}

// canonState is what Export gives, as sorted strings.  counters at zero are
// left out because a dropped counter reads zero.  set definitions are kept
// only for sets a live lease uses, because a follower learns of a dropped
// set late.
func canonState(t testing.TB, m *Manager) []string {
	t.Helper()
	var recs []Record
	m.Freeze(func() {
		m.Export(func(r Record) { recs = append(recs, r) })
	})

	used := map[uint64]bool{}
	for _, r := range recs {
		if r.Kind == RecGrant {
			used[r.SetID] = true
		}
	}
	var out []string
	for _, r := range recs {
		switch r.Kind {
		case RecSeqs:
		case RecSet:
			if !used[r.SetID] {
				continue
			}
			def, err := json.Marshal(r.Set)
			require.NoError(t, err)
			out = append(out, fmt.Sprintf("set %d %s", r.SetID, def))
		case RecGrant:
			require.False(t, r.Force)
			require.Zero(t, r.Value, "export emits no acquire idempotency record")
			require.Len(t, r.Seqs, 1)
			out = append(out, fmt.Sprintf("lease %d set %d exp %d req %d granted %d max %s src %+v", r.Seqs[0], r.SetID, r.ExpiresAtMS, r.Requested, r.Granted, r.MaximumLifetime, r.Source))
		case RecTAT:
			out = append(out, fmt.Sprintf("tat %x.%x %v exp %d v %d", r.KeyA, r.KeyB, r.TAT, r.ExpiresAtMS, r.Version))
		case RecCell:
			if r.Value == 0 {
				continue
			}
			out = append(out, fmt.Sprintf("cell %x.%x %d", r.KeyA, r.KeyB, r.Value))
		case RecIdem:
			out = append(out, fmt.Sprintf("idem %x %d exp %d", r.KeyA, r.Value, r.ExpiresAtMS))
		default:
			t.Fatalf("export emitted record kind %d", r.Kind)
		}
	}
	sort.Strings(out)
	return out
}

// replUsageKey is the usage counter key of the first concurrency constraint
// with the given scope in the set id of a follower.
func replUsageKey(t testing.TB, m *Manager, id uint64, scope enums.ConcurrencyScope) cellKey {
	t.Helper()
	set := m.replSets[id]
	require.NotNil(t, set)
	for i := range set.constraints {
		rc := &set.constraints[i]
		if rc.item.Kind == constraintapi.ConstraintKindConcurrency && rc.item.Concurrency.Scope == scope {
			return rc.usageKey
		}
	}
	t.Fatalf("set %d has no concurrency constraint with scope %v", id, scope)
	return cellKey{}
}

// replShape is one request shape of the load.  it is an account, its config
// and a constraint list.
type replShape struct {
	account, env, fn, app uuid.UUID
	config                constraintapi.ConstraintConfig
	constraints           []constraintapi.ConstraintItem
}

var (
	replAutoSem   = constraintapi.SemaphoreConstraint{ID: "app:auto", Weight: 2, Release: constraintapi.SemaphoreReleaseAuto}
	replManualSem = constraintapi.SemaphoreConstraint{ID: "fn:manual", Weight: 1, Release: constraintapi.SemaphoreReleaseManual}
)

// replShapes builds the request shapes for n accounts.  between them they
// touch every constraint kind.
func replShapes(n int) []replShape {
	concurrency := func(scope enums.ConcurrencyScope) constraintapi.ConstraintItem {
		return constraintapi.ConstraintItem{
			Kind: constraintapi.ConstraintKindConcurrency,
			Concurrency: &constraintapi.ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: scope,
			},
		}
	}
	semaphore := func(s constraintapi.SemaphoreConstraint) constraintapi.ConstraintItem {
		return constraintapi.ConstraintItem{
			Kind:      constraintapi.ConstraintKindSemaphore,
			Semaphore: &s,
		}
	}
	throttle := constraintapi.ConstraintItem{
		Kind: constraintapi.ConstraintKindThrottle,
		Throttle: &constraintapi.ThrottleConstraint{
			Scope:             enums.ThrottleScopeFn,
			KeyExpressionHash: "t",
			EvaluatedKeyHash:  "v",
		},
	}
	rateLimit := constraintapi.ConstraintItem{
		Kind: constraintapi.ConstraintKindRateLimit,
		RateLimit: &constraintapi.RateLimitConstraint{
			Scope:             enums.RateLimitScopeFn,
			KeyExpressionHash: "r",
			EvaluatedKeyHash:  "v",
		},
	}

	config := constraintapi.ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: constraintapi.ConcurrencyConfig{
			AccountConcurrency:  40,
			FunctionConcurrency: 12,
		},
		Throttle: []constraintapi.ThrottleConfig{{
			Scope:             enums.ThrottleScopeFn,
			KeyExpressionHash: "t",
			Limit:             200,
			Burst:             5,
			Period:            60,
		}},
		RateLimit: []constraintapi.RateLimitConfig{{
			Scope:             enums.RateLimitScopeFn,
			KeyExpressionHash: "r",
			Limit:             300,
			Period:            60,
		}},
		Semaphores: []constraintapi.Semaphore{
			constraintapi.Semaphore(replAutoSem),
			constraintapi.Semaphore(replManualSem),
		},
	}

	var shapes []replShape
	for i := 0; i < n; i++ {
		base := replShape{
			account: uuid.New(),
			env:     uuid.New(),
			fn:      uuid.New(),
			app:     uuid.New(),
			config:  config,
		}
		for _, constraints := range [][]constraintapi.ConstraintItem{
			{concurrency(enums.ConcurrencyScopeAccount), concurrency(enums.ConcurrencyScopeFn)},
			{concurrency(enums.ConcurrencyScopeFn), throttle},
			{rateLimit},
			{semaphore(replAutoSem)},
			{semaphore(replManualSem), concurrency(enums.ConcurrencyScopeFn)},
			{semaphore(replAutoSem), throttle, rateLimit},
		} {
			s := base
			s.constraints = constraints
			shapes = append(shapes, s)
		}
	}
	return shapes
}

func (s replShape) acquire(clock clockwork.Clock, key string, amount int) *constraintapi.CapacityAcquireRequest {
	keys := make([]string, amount)
	for i := range keys {
		keys[i] = fmt.Sprintf("%s-lease-%d", key, i)
	}
	return &constraintapi.CapacityAcquireRequest{
		IdempotencyKey:       key,
		AccountID:            s.account,
		EnvID:                s.env,
		FunctionID:           s.fn,
		AppID:                s.app,
		Configuration:        s.config,
		Constraints:          s.constraints,
		Amount:               amount,
		LeaseIdempotencyKeys: keys,
		LeaseRunIDs:          map[string]ulid.ULID{},
		CurrentTime:          clock.Now(),
		Duration:             5 * time.Second,
		MaximumLifetime:      time.Hour,
		Source: constraintapi.LeaseSource{
			Service:           constraintapi.ServiceExecutor,
			Location:          constraintapi.CallerLocationItemLease,
			RunProcessingMode: constraintapi.RunProcessingModeBackground,
		},
	}
}

var replSource = constraintapi.LeaseSource{
	Service:  constraintapi.ServiceExecutor,
	Location: constraintapi.CallerLocationItemLease,
}

type replLease struct {
	account uuid.UUID
	id      ulid.ULID
}

// replLoad runs a mixed load against m from several goroutines until stop is
// closed.  the load is grants, extends, releases and the three semaphore
// writes.  a
// second goroutine moves the clock and sweeps, so leases expire and are
// reclaimed while the load runs.
func replLoad(t testing.TB, m *Manager, clock *clockwork.FakeClock, shapes []replShape, workers int, stop <-chan struct{}) *sync.WaitGroup {
	ctx := context.Background()
	for _, s := range shapes {
		_, err := m.SetCapacity(ctx, s.account, replAutoSem.ID, "init-auto-"+s.account.String(), 30)
		require.NoError(t, err)
		_, err = m.SetCapacity(ctx, s.account, replManualSem.ID, "init-manual-"+s.account.String(), 30)
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	var n atomic.Int64
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(uint64(w), 42))
			var leases []replLease
			for {
				select {
				case <-stop:
					return
				default:
				}
				key := fmt.Sprintf("w%d-%d", w, n.Add(1))
				s := shapes[rng.IntN(len(shapes))]

				switch p := rng.IntN(100); {
				case p < 45:
					resp, err := m.Acquire(ctx, s.acquire(clock, key, 1+rng.IntN(3)))
					if err != nil {
						t.Errorf("acquire failed: %v", err)
						return
					}
					for _, l := range resp.Leases {
						leases = append(leases, replLease{
							account: s.account,
							id:      l.LeaseID,
						})
					}
				case p < 65 && len(leases) > 0:
					i := rng.IntN(len(leases))
					resp, err := m.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
						IdempotencyKey: key,
						AccountID:      leases[i].account,
						LeaseID:        leases[i].id,
						Duration:       5 * time.Second,
						Source:         replSource,
					})
					if err != nil {
						t.Errorf("extend failed: %v", err)
						return
					}
					if resp.LeaseID != nil {
						leases[i].id = *resp.LeaseID
					} else {
						leases = append(leases[:i], leases[i+1:]...)
					}
				case p < 90 && len(leases) > 0:
					i := rng.IntN(len(leases))
					_, err := m.Release(ctx, &constraintapi.CapacityReleaseRequest{
						IdempotencyKey: key,
						AccountID:      leases[i].account,
						LeaseID:        leases[i].id,
						Source:         replSource,
					})
					if err != nil {
						t.Errorf("release failed: %v", err)
						return
					}
					leases = append(leases[:i], leases[i+1:]...)
				case p < 93:
					if _, err := m.SetCapacity(ctx, s.account, replAutoSem.ID, key, int64(20+rng.IntN(20))); err != nil {
						t.Errorf("set capacity failed: %v", err)
						return
					}
				case p < 96:
					if _, err := m.AdjustCapacity(ctx, s.account, replManualSem.ID, key, int64(rng.IntN(7)-3)); err != nil {
						t.Errorf("adjust capacity failed: %v", err)
						return
					}
				default:
					// more than the usage sometimes, so the counter is clamped
					if err := m.ReleaseSemaphore(ctx, s.account, replManualSem.ID, "", key, int64(1+rng.IntN(3))); err != nil {
						t.Errorf("release semaphore failed: %v", err)
						return
					}
				}
			}
		}(w)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			clock.Advance(100 * time.Millisecond)
			if _, err := m.Scavenge(ctx); err != nil {
				t.Errorf("scavenge failed: %v", err)
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()
	return &wg
}

// TestReplicaEquivalence runs a concurrent mixed load on a primary and
// applies its log to a follower.  when the load stops and the log is drained
// the two export the same state.
func TestReplicaEquivalence(t *testing.T) {
	clock := clockwork.NewFakeClockAt(replStart)
	primary, l := newReplPrimary(t, clock)
	follower := newReplFollower(t, clock)
	pump := startReplPump(t, l, follower)

	stop := make(chan struct{})
	wg := replLoad(t, primary, clock, replShapes(3), 8, stop)
	time.Sleep(400 * time.Millisecond)
	close(stop)
	wg.Wait()
	pump.close()

	want := canon(t, primary)
	require.NotEmpty(t, want)
	require.Equal(t, want, canon(t, follower))
	t.Logf("compared %d state lines after %d drain cycles", len(want), pump.n)

	kinds := map[string]bool{}
	for _, line := range want {
		var kind string
		_, _ = fmt.Sscan(line, &kind)
		kinds[kind] = true
	}
	for _, kind := range []string{"set", "lease", "tat", "cell", "idem"} {
		require.True(t, kinds[kind], "the load left no %s to compare", kind)
	}
}

// TestReplicaStateTransfer joins a second follower in the middle of the
// load.  it gets an export taken inside Freeze and then the live log, and
// ends with the same state as the primary and the first follower.
func TestReplicaStateTransfer(t *testing.T) {
	clock := clockwork.NewFakeClockAt(replStart)
	primary, l := newReplPrimary(t, clock)
	first := newReplFollower(t, clock)
	second := newReplFollower(t, clock)
	pump := startReplPump(t, l, first)

	stop := make(chan struct{})
	wg := replLoad(t, primary, clock, replShapes(3), 8, stop)
	time.Sleep(200 * time.Millisecond)

	var exported int
	pump.do(func() {
		primary.Freeze(func() {
			// what the log still has belongs before the export
			pump.cycle()
			primary.Export(func(r Record) {
				exported++
				if err := second.Apply([]Record{r}); err != nil {
					t.Errorf("apply of the export failed: %v", err)
				}
			})
			pump.sinks = append(pump.sinks, second)
		})
	})
	require.NotZero(t, exported)

	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()
	pump.close()

	want := canon(t, primary)
	require.NotEmpty(t, want)
	require.Equal(t, want, canon(t, first))
	// the export has no acquire or extend idempotency record
	require.Equal(t, canonState(t, primary), canonState(t, second))
}

// TestReplicaPromote promotes a follower after its primary is gone.  leases
// granted before keep their IDs, new leases get new seqs, the recount fixes a
// wrong counter, and the sweeper reclaims what expires.
func TestReplicaPromote(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClockAt(replStart)
	primary, l := newReplPrimary(t, clock)
	follower := newReplFollower(t, clock)
	pump := startReplPump(t, l, follower)

	shapes := replShapes(1)
	fnAndAccount, autoSem := shapes[0], shapes[3]
	_, err := primary.SetCapacity(ctx, autoSem.account, replAutoSem.ID, "cap", 30)
	require.NoError(t, err)

	var old []replLease
	var maxSeq [slabShards]uint64
	for i := 0; i < 6; i++ {
		s := fnAndAccount
		if i%2 == 1 {
			s = autoSem
		}
		resp, err := primary.Acquire(ctx, s.acquire(clock, fmt.Sprintf("old-%d", i), 1))
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)
		old = append(old, replLease{
			account: s.account,
			id:      resp.Leases[0].LeaseID,
		})
		_, seq, _ := decodeLeaseID(resp.Leases[0].LeaseID)
		shard := seq >> shardShift
		maxSeq[shard] = max(maxSeq[shard], seq&seqMask)
	}
	pump.close()
	require.NoError(t, primary.Close())
	require.Equal(t, canon(t, primary), canon(t, follower))

	// three leases use the function concurrency counter and three the
	// semaphore, at weight two
	fnKey := replUsageKey(t, follower, 1, enums.ConcurrencyScopeFn)
	semKey, _ := semaphoreCells(autoSem.account, replAutoSem.ID, "")
	require.EqualValues(t, 3, follower.peekSem(fnKey))
	require.EqualValues(t, 6, follower.peekSem(semKey))

	// a counter that drifted, and one left below zero by records that never
	// arrived
	follower.sem(fnKey).set(99)
	stray := cellKeyOf('u', uuid.New(), "stray", "")
	require.True(t, follower.sem(stray).add(-5))

	l2 := NewLog()
	follower.Promote(l2)
	pump2 := startReplPump(t, l2)
	defer pump2.close()

	require.EqualValues(t, 3, follower.peekSem(fnKey), "recount")
	require.EqualValues(t, 6, follower.peekSem(semKey))
	require.EqualValues(t, 0, follower.peekSem(stray), "a negative counter is set to zero")

	// a lease from before the promotion extends and releases under its ID
	ext, err := follower.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
		IdempotencyKey: "ext",
		AccountID:      old[0].account,
		LeaseID:        old[0].id,
		Duration:       5 * time.Second,
		Source:         replSource,
	})
	require.NoError(t, err)
	require.NotNil(t, ext.LeaseID)
	require.Equal(t, fnAndAccount.fn, ext.FunctionID)

	rel, err := follower.Release(ctx, &constraintapi.CapacityReleaseRequest{
		IdempotencyKey: "rel",
		AccountID:      old[1].account,
		LeaseID:        old[1].id,
		Source:         replSource,
	})
	require.NoError(t, err)
	require.Equal(t, autoSem.fn, rel.FunctionID, "the release found the lease")
	require.EqualValues(t, 4, follower.peekSem(semKey))

	// new seqs start past every seq the old primary can have used
	for _, id := range []ulid.ULID{*ext.LeaseID} {
		_, seq, nonce := decodeLeaseID(id)
		require.EqualValues(t, replNonce, nonce)
		require.GreaterOrEqual(t, seq&seqMask, maxSeq[seq>>shardShift]+seqJump)
	}
	seen := map[uint64]bool{}
	for i := 0; i < 200; i++ {
		resp, err := follower.Acquire(ctx, fnAndAccount.acquire(clock, fmt.Sprintf("new-%d", i), 1))
		require.NoError(t, err)
		if len(resp.Leases) == 0 {
			break
		}
		_, seq, _ := decodeLeaseID(resp.Leases[0].LeaseID)
		require.False(t, seen[seq])
		seen[seq] = true
		require.GreaterOrEqual(t, seq&seqMask, maxSeq[seq>>shardShift]+seqJump)
	}
	// the function limit is 12 and three old leases still count
	require.Len(t, seen, 9)
	require.EqualValues(t, 12, follower.peekSem(fnKey))

	// everything expires and the sweeper of the promoted manager reclaims it
	clock.Advance(10 * time.Second)
	reclaimed, err := follower.Scavenge(ctx)
	require.NoError(t, err)
	require.Equal(t, 3+2+9, reclaimed)
	require.EqualValues(t, 0, follower.peekSem(fnKey))
	require.EqualValues(t, 0, follower.peekSem(semKey))
}

// TestReplicaPromoteStartsSweeper checks a promoted manager runs its own
// sweeper when it has a sweep interval.
func TestReplicaPromoteStartsSweeper(t *testing.T) {
	ctx := context.Background()
	primary, l := newReplPrimary(t, clockwork.NewRealClock())
	follower, err := NewManager(
		WithShardName("follower"),
		WithSweepInterval(5*time.Millisecond),
		WithNonce(replNonce),
		WithFollower(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = follower.Close() })
	pump := startReplPump(t, l, follower)

	s := replShapes(1)[0]
	req := s.acquire(clockwork.NewRealClock(), "a", 1)
	req.Duration = 2100 * time.Millisecond
	req.MaximumLifetime = time.Minute
	resp, err := primary.Acquire(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Leases, 1)
	pump.close()

	fnKey := replUsageKey(t, follower, 1, enums.ConcurrencyScopeFn)
	follower.Promote(nil)
	require.EqualValues(t, 1, follower.peekSem(fnKey))
	require.Eventually(t, func() bool {
		return follower.peekSem(fnKey) == 0
	}, 5*time.Second, 10*time.Millisecond)
}

// replWaitWake blocks until an operation has signalled the log.
func replWaitWake(t *testing.T, l *Log) {
	t.Helper()
	select {
	case <-l.Wake():
	case <-time.After(5 * time.Second):
		t.Fatal("no operation signalled the log")
	}
}

func replBlocked(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
		t.Fatal("the operation answered before its commit level was reached")
	case <-time.After(30 * time.Millisecond):
	}
}

func replAnswered(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the operation did not answer")
	}
}

func TestCommitWait(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClockAt(replStart)
	s := replShapes(1)[0]

	type result struct {
		resp *constraintapi.CapacityAcquireResponse
		err  error
	}
	start := func(m *Manager, ctx context.Context, req *constraintapi.CapacityAcquireRequest) (<-chan struct{}, *result) {
		done := make(chan struct{})
		res := &result{}
		go func() {
			defer close(done)
			resp, err := m.Acquire(ctx, req)
			res.resp = resp
			if err != nil {
				res.err = err
			}
		}()
		return done, res
	}

	t.Run("acquire waits for every replica", func(t *testing.T) {
		m, l := newReplPrimary(t, clock)
		done, res := start(m, ctx, s.acquire(clock, "a", 1))
		replWaitWake(t, l)
		recs, b := l.Drain(nil)
		require.NotEmpty(t, recs)
		replBlocked(t, done)
		b.AckTwo()
		replBlocked(t, done)
		b.AckAll()
		replAnswered(t, done)
		require.NoError(t, res.err)
		require.Len(t, res.resp.Leases, 1)
	})

	t.Run("level two answers on the first follower", func(t *testing.T) {
		m, l := newReplPrimary(t, clock)
		req := s.acquire(clock, "a", 1)
		req.CommitLevel = constraintapi.CommitLevelTwo
		done, res := start(m, ctx, req)
		replWaitWake(t, l)
		_, b := l.Drain(nil)
		replBlocked(t, done)
		b.AckTwo()
		replAnswered(t, done)
		require.NoError(t, res.err)
		require.Len(t, res.resp.Leases, 1)
	})

	t.Run("level primary does not wait", func(t *testing.T) {
		m, _ := newReplPrimary(t, clock)
		req := s.acquire(clock, "a", 1)
		req.CommitLevel = constraintapi.CommitLevelPrimary
		resp, err := m.Acquire(ctx, req)
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)
	})

	t.Run("a denied acquire does not wait", func(t *testing.T) {
		m, _ := newReplPrimary(t, clock)
		denied := replShapes(1)[3]
		resp, err := m.Acquire(ctx, denied.acquire(clock, "a", 1))
		require.NoError(t, err)
		require.Empty(t, resp.Leases, "the semaphore has no capacity")
	})

	t.Run("fail gives a retryable error", func(t *testing.T) {
		m, l := newReplPrimary(t, clock)
		lost := errors.New("not primary")
		done, res := start(m, ctx, s.acquire(clock, "a", 1))
		replWaitWake(t, l)
		_, b := l.Drain(nil)
		b.Fail(lost)
		replAnswered(t, done)
		require.ErrorIs(t, res.err, lost)
		var ierr interface{ Retryable() bool }
		require.ErrorAs(t, res.err, &ierr)
		require.True(t, ierr.Retryable())
	})

	t.Run("fail after an acknowledgement keeps the acknowledgement", func(t *testing.T) {
		l := NewLog()
		_, b := l.Drain(nil)
		b.AckTwo()
		b.Fail(errors.New("not primary"))
		require.NoError(t, b.Wait(ctx, constraintapi.CommitLevelTwo))
		require.Error(t, b.Wait(ctx, constraintapi.CommitLevelAll))
	})

	t.Run("a cancelled context stops the wait", func(t *testing.T) {
		m, l := newReplPrimary(t, clock)
		cctx, cancel := context.WithCancel(ctx)
		done, res := start(m, cctx, s.acquire(clock, "a", 1))
		replWaitWake(t, l)
		cancel()
		replAnswered(t, done)
		require.ErrorIs(t, res.err, context.Canceled)
	})

	t.Run("pending defers the wait", func(t *testing.T) {
		m, l := newReplPrimary(t, clock)
		var p Pending
		pctx := WithPending(ctx, &p)
		for i := 0; i < 3; i++ {
			resp, err := m.Acquire(pctx, s.acquire(clock, fmt.Sprintf("a-%d", i), 1))
			require.NoError(t, err)
			require.Len(t, resp.Leases, 1)
		}
		require.NoError(t, m.ReleaseSemaphore(pctx, s.account, replManualSem.ID, "", "rel", 1))
		require.Len(t, p.entries, 1, "one log, one entry")

		done := make(chan struct{})
		var werr error
		go func() {
			defer close(done)
			werr = p.Wait(ctx)
		}()
		replBlocked(t, done)
		_, b := l.Drain(nil)
		b.AckAll()
		replAnswered(t, done)
		require.NoError(t, werr)
		require.NoError(t, p.Wait(ctx), "an empty pending does not wait")
	})

	t.Run("a replay waits", func(t *testing.T) {
		m, l := newReplPrimary(t, clock)
		done, _ := start(m, ctx, s.acquire(clock, "a", 1))
		replWaitWake(t, l)
		_, b := l.Drain(nil)
		b.AckAll()
		replAnswered(t, done)

		done, res := start(m, ctx, s.acquire(clock, "a", 1))
		replWaitWake(t, l)
		replBlocked(t, done)
		recs, b := l.Drain(nil)
		require.Empty(t, recs, "a replay logs nothing")
		b.AckAll()
		replAnswered(t, done)
		require.NoError(t, res.err)
		require.True(t, res.resp.OperationIdempotencyHit)
	})

	t.Run("extend and semaphore writes wait, release does not", func(t *testing.T) {
		m, l := newReplPrimary(t, clock)
		req := s.acquire(clock, "a", 2)
		req.CommitLevel = constraintapi.CommitLevelPrimary
		resp, err := m.Acquire(ctx, req)
		require.NoError(t, err)
		require.Len(t, resp.Leases, 2)

		_, err = m.Release(ctx, &constraintapi.CapacityReleaseRequest{
			IdempotencyKey: "rel",
			AccountID:      s.account,
			LeaseID:        resp.Leases[0].LeaseID,
			Source:         replSource,
		})
		require.NoError(t, err)

		for name, op := range map[string]func() error{
			"extend": func() error {
				_, err := m.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
					IdempotencyKey: "ext",
					AccountID:      s.account,
					LeaseID:        resp.Leases[1].LeaseID,
					Duration:       5 * time.Second,
					Source:         replSource,
				})
				if err != nil {
					return err
				}
				return nil
			},
			"set capacity": func() error {
				_, err := m.SetCapacity(ctx, s.account, replAutoSem.ID, "set", 5)
				return err
			},
			"adjust capacity": func() error {
				_, err := m.AdjustCapacity(ctx, s.account, replAutoSem.ID, "adj", 1)
				return err
			},
			"release semaphore": func() error {
				return m.ReleaseSemaphore(ctx, s.account, replManualSem.ID, "", "relsem", 1)
			},
			"release at level all": func() error {
				_, err := m.Release(ctx, &constraintapi.CapacityReleaseRequest{
					IdempotencyKey: "rel",
					AccountID:      s.account,
					LeaseID:        resp.Leases[0].LeaseID,
					Source:         replSource,
					CommitLevel:    constraintapi.CommitLevelAll,
				})
				if err != nil {
					return err
				}
				return nil
			},
		} {
			for len(l.Wake()) > 0 {
				<-l.Wake()
			}
			done := make(chan struct{})
			var opErr error
			go func() {
				defer close(done)
				opErr = op()
			}()
			replWaitWake(t, l)
			replBlocked(t, done)
			_, b := l.Drain(nil)
			b.AckAll()
			replAnswered(t, done)
			require.NoError(t, opErr, name)
		}
	})

	t.Run("no log means no wait and no records", func(t *testing.T) {
		m, err := NewManager(WithShardName("plain"), WithClock(clock), WithSweepInterval(0))
		require.NoError(t, err)
		t.Cleanup(func() { _ = m.Close() })
		resp, err := m.Acquire(ctx, s.acquire(clock, "a", 1))
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)
		require.NoError(t, m.ReleaseSemaphore(ctx, s.account, replManualSem.ID, "", "rel", 1))
	})
}

// TestLogTATVersion checks a rate limit state written after its cell was
// dropped has a higher version than the state the dropped cell had, so a
// follower that still has the old state takes the new one.
func TestLogTATVersion(t *testing.T) {
	var c gcraCell
	write := func(c *gcraCell, nowMS int64) *gcraState {
		st, ok := c.update(nowMS, func(float64, bool) (float64, bool, int64) { return 1, true, 1 })
		require.True(t, ok)
		return st
	}
	a := write(&c, 1_000)
	b := write(&c, 1_000)
	require.Greater(t, b.version, a.version)
	back := write(&c, 900)
	require.Greater(t, back.version, b.version, "a clock that went back")

	require.True(t, c.kill(5_000))
	var fresh gcraCell
	again := write(&fresh, 5_000)
	require.Greater(t, again.version, back.version)

	var follower gcraCell
	require.True(t, follower.apply(2, 9_000, b.version))
	require.True(t, follower.apply(3, 9_000, a.version))
	tat, ok := follower.load(0)
	require.True(t, ok)
	require.EqualValues(t, 2, tat, "an older version does not replace a newer one")
}

// TestExportHasSetWaitingForDrop checks the export has a set that left the
// sets map while its RecSetDrop is not logged.  a request that loaded the
// set before it left can still log a grant for it, and a follower built from
// the export must know the set then.
func TestExportHasSetWaitingForDrop(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClockAt(replStart)
	primary, l := newReplPrimary(t, clock)
	pump := startReplPump(t, l)
	defer pump.close()

	s := replShapes(1)[0]
	resp, err := primary.Acquire(ctx, s.acquire(clock, "a", 1))
	require.NoError(t, err)
	require.Len(t, resp.Leases, 1)
	_, err = primary.Release(ctx, &constraintapi.CapacityReleaseRequest{
		IdempotencyKey: "release-a",
		AccountID:      s.account,
		LeaseID:        resp.Leases[0].LeaseID,
		Source:         replSource,
	})
	require.NoError(t, err)

	exported := func() bool {
		found := false
		pump.do(func() {
			primary.Freeze(func() {
				primary.Export(func(r Record) {
					if r.Kind == RecSet && r.SetID == 1 {
						found = true
					}
				})
			})
		})
		return found
	}
	inMap := func() bool {
		found := false
		primary.sets.Range(func(_, _ any) bool {
			found = true
			return false
		})
		return found
	}
	require.True(t, exported())

	// two rounds with no use drop the set from the primary's map
	_, _ = primary.Scavenge(ctx)
	_, _ = primary.Scavenge(ctx)
	require.False(t, inMap())
	require.True(t, exported(), "the drop is not logged yet")

	clock.Advance(setDropDelay)
	_, _ = primary.Scavenge(ctx)
	require.False(t, exported())
}

// TestLogSetDrop checks a follower forgets a set only after the delay, and
// that a set built again under the same key has a new ID.
func TestLogSetDrop(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClockAt(replStart)
	primary, l := newReplPrimary(t, clock)
	follower := newReplFollower(t, clock)
	pump := startReplPump(t, l, follower)

	s := replShapes(1)[0]
	_, err := primary.Acquire(ctx, s.acquire(clock, "a", 1))
	require.NoError(t, err)
	// knows drains the log and says whether the follower has the set
	knows := func(id uint64) bool {
		var ok bool
		pump.do(func() {
			pump.cycle()
			_, ok = follower.replSets[id]
		})
		return ok
	}
	require.True(t, knows(1))

	// two rounds with no use drop the set from the primary's map
	_, _ = primary.Scavenge(ctx)
	_, _ = primary.Scavenge(ctx)
	require.True(t, knows(1), "the drop is logged late")

	clock.Advance(setDropDelay)
	_, _ = primary.Scavenge(ctx)
	require.False(t, knows(1))

	_, err = primary.Acquire(ctx, s.acquire(clock, "b", 1))
	require.NoError(t, err)
	require.True(t, knows(2))
	pump.close()
	require.Equal(t, canon(t, primary), canon(t, follower))
}

// TestReplicaClampedGiveBack covers a counter the primary clamped at zero.  a
// manual release semaphore is released by hand with more than its usage, and
// then the sweeper reclaims the lease and gives the same unit back again.  the
// follower subtracts in full both times, and the corrections the primary logs
// bring it to the same value.
func TestReplicaClampedGiveBack(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClockAt(replStart)
	primary, l := newReplPrimary(t, clock)
	follower := newReplFollower(t, clock)
	pump := startReplPump(t, l, follower)

	s := replShapes(1)[4]
	_, err := primary.SetCapacity(ctx, s.account, replManualSem.ID, "cap", 5)
	require.NoError(t, err)
	resp, err := primary.Acquire(ctx, s.acquire(clock, "a", 1))
	require.NoError(t, err)
	require.Len(t, resp.Leases, 1)

	usageKey, _ := semaphoreCells(s.account, replManualSem.ID, "")
	require.EqualValues(t, 1, primary.peekSem(usageKey))
	require.NoError(t, primary.ReleaseSemaphore(ctx, s.account, replManualSem.ID, "", "by-hand", 5))
	require.EqualValues(t, 0, primary.peekSem(usageKey))

	clock.Advance(10 * time.Second)
	reclaimed, err := primary.Scavenge(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, reclaimed)
	require.EqualValues(t, 0, primary.peekSem(usageKey))

	pump.close()
	require.EqualValues(t, 0, follower.peekSem(usageKey))
	require.Equal(t, canon(t, primary), canon(t, follower))
}

// TestReplicaReplaysAfterPromote checks a promoted follower answers a retry
// of an acquire and of an extend made on the old primary with the same lease
// IDs.  the follower got the records from the log.
func TestReplicaReplaysAfterPromote(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClockAt(replStart)
	primary, l := newReplPrimary(t, clock)
	streamed := newReplFollower(t, clock)
	pump := startReplPump(t, l, streamed)

	shape := replShapes(1)[0]
	acq := func() *constraintapi.CapacityAcquireRequest {
		return shape.acquire(clock, "retry-me", 2)
	}
	first, err := primary.Acquire(ctx, acq())
	require.NoError(t, err)
	require.Len(t, first.Leases, 2)
	require.False(t, first.OperationIdempotencyHit)

	ext := func() *constraintapi.CapacityExtendLeaseRequest {
		return &constraintapi.CapacityExtendLeaseRequest{
			IdempotencyKey: "extend-me",
			AccountID:      shape.account,
			LeaseID:        first.Leases[0].LeaseID,
			Duration:       5 * time.Second,
			Source:         replSource,
		}
	}
	extended, err := primary.ExtendLease(ctx, ext())
	require.NoError(t, err)
	require.NotNil(t, extended.LeaseID)

	pump.close()
	require.NoError(t, primary.Close())

	want := canon(t, primary)
	require.Contains(t, strings.Join(want, "\n"), "acquire ")
	require.Contains(t, strings.Join(want, "\n"), "extend ")
	require.Equal(t, want, canon(t, streamed))

	m := streamed
	l2 := NewLog()
	m.Promote(l2)
	pump2 := startReplPump(t, l2)
	defer pump2.close()

	again, err := m.Acquire(ctx, acq())
	require.NoError(t, err)
	require.True(t, again.OperationIdempotencyHit)
	require.Equal(t, first.Leases, again.Leases)

	againExt, err := m.ExtendLease(ctx, ext())
	require.NoError(t, err)
	require.True(t, againExt.OperationIdempotencyHit)
	require.NotNil(t, againExt.LeaseID)
	require.Equal(t, *extended.LeaseID, *againExt.LeaseID)
	require.Equal(t, shape.fn, againExt.FunctionID)

	// the replayed leases are the live ones.  the extended lease releases
	// under its new ID and the other under its first ID.
	for _, id := range []ulid.ULID{*againExt.LeaseID, again.Leases[1].LeaseID} {
		rel, err := m.Release(ctx, &constraintapi.CapacityReleaseRequest{
			IdempotencyKey: "rel-" + id.String(),
			AccountID:      shape.account,
			LeaseID:        id,
			Source:         replSource,
		})
		require.NoError(t, err)
		require.Equal(t, shape.fn, rel.FunctionID, "the release found the lease")
	}
}

// TestReplicaExportLeavesOutIdempotencyRecords checks what a follower built
// from an export has.  it lacks the acquire record from before the export,
// and it has every record made after.  at IdempotencyEnd the primary's
// record from before the export has expired, so both have the same records.
func TestReplicaExportLeavesOutIdempotencyRecords(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClockAt(replStart)
	primary, l := newReplPrimary(t, clock)
	pump := startReplPump(t, l)

	shape := replShapes(1)[0]
	before, err := primary.Acquire(ctx, shape.acquire(clock, "before", 1))
	require.NoError(t, err)
	require.Len(t, before.Leases, 1)
	ttlMS := constraintapi.OperationIdempotencyTTL.Milliseconds()
	beforeEnd := clock.Now().UnixMilli() + ttlMS

	clock.Advance(2 * time.Second)
	joined := newReplFollower(t, clock)
	var end int64
	pump.do(func() {
		primary.Freeze(func() {
			pump.cycle()
			end = primary.IdempotencyEnd()
			primary.Export(func(r Record) {
				require.NoError(t, joined.Apply([]Record{r}))
			})
			pump.sinks = append(pump.sinks, joined)
		})
	})
	require.Equal(t, beforeEnd, end)
	require.Equal(t, canonState(t, primary), canonState(t, joined))
	require.Equal(t, 1, primary.acqIdem.len())
	require.Zero(t, joined.acqIdem.len())

	after, err := primary.Acquire(ctx, shape.acquire(clock, "after", 1))
	require.NoError(t, err)
	require.Len(t, after.Leases, 1)
	pump.close()
	require.NoError(t, primary.Close())
	require.NotEqual(t, canon(t, primary), canon(t, joined))

	clock.Advance(time.UnixMilli(end).Sub(clock.Now()))
	require.GreaterOrEqual(t, primary.NowMS(), end)
	want := canon(t, primary)
	require.Contains(t, strings.Join(want, "\n"), "acquire ")
	require.Equal(t, want, canon(t, joined))

	joined.Promote(nil)
	again, err := joined.Acquire(ctx, shape.acquire(clock, "after", 1))
	require.NoError(t, err)
	require.True(t, again.OperationIdempotencyHit)
	require.Equal(t, after.Leases, again.Leases)
}

// TestReplicaReplayExpires checks a replicated acquire record ends at the
// same time as the one on the primary, so a late retry is a new request.
func TestReplicaReplayExpires(t *testing.T) {
	ctx := context.Background()
	clock := clockwork.NewFakeClockAt(replStart)
	primary, l := newReplPrimary(t, clock)
	follower := newReplFollower(t, clock)
	pump := startReplPump(t, l, follower)

	shape := replShapes(1)[0]
	first, err := primary.Acquire(ctx, shape.acquire(clock, "late", 1))
	require.NoError(t, err)
	require.Len(t, first.Leases, 1)
	_, err = primary.Acquire(ctx, shape.acquire(clock, "swept", 1))
	require.NoError(t, err)
	pump.close()
	require.NoError(t, primary.Close())

	follower.Promote(nil)
	clock.Advance(constraintapi.OperationIdempotencyTTL)
	again, err := follower.Acquire(ctx, shape.acquire(clock, "late", 1))
	require.NoError(t, err)
	require.False(t, again.OperationIdempotencyHit)
	if len(again.Leases) == 1 {
		require.NotEqual(t, first.Leases[0].LeaseID, again.Leases[0].LeaseID)
	}

	// the retry left a new record.  the one nobody retried is swept.
	require.Equal(t, 2, follower.acqIdem.len())
	followerHousekeep(follower)
	require.Equal(t, 1, follower.acqIdem.len(), "the sweep removed the expired record")
}

// TestFollowerHousekeepStepSweepsOneStripe checks that one step drops the
// expired records of one stripe, and that FollowerHousekeepSteps steps drop
// all of them from any start.
func TestFollowerHousekeepStepSweepsOneStripe(t *testing.T) {
	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
	follower := newReplFollower(t, clock)
	expired := clock.Now().UnixMilli()
	live := expired + 1

	const perStripe = 3
	fill := func() {
		for k := range uint64(perStripe * ttlStripes) {
			follower.acqIdem.set(k, &acquireRecord{}, expired)
			follower.extIdem.set(k, &constraintapi.CapacityExtendLeaseResponse{}, expired)
			follower.semIdem.set(k, 1, expired)
		}
		follower.acqIdem.set(perStripe*ttlStripes, &acquireRecord{}, live)
	}
	total := func() int {
		return follower.acqIdem.len() + follower.extIdem.len() + follower.semIdem.len()
	}

	fill()
	// stop in the middle of a pass.  the next pass starts there.
	for step := 1; step <= 100; step++ {
		follower.FollowerHousekeepStep()
		require.Equal(t, 3*perStripe*(ttlStripes-step)+1, total(), "after step %d", step)
	}
	fill()
	followerHousekeep(follower)
	require.Equal(t, 1, total(), "a full pass keeps only the record that is not expired")
}
