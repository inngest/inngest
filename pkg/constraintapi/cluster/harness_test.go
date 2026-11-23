package cluster

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
)

const (
	testHeartbeatInterval = 10 * time.Millisecond
	testHeartbeatTimeout  = 100 * time.Millisecond
)

// testNode is what the harness needs from a node.
type testNode interface {
	// Close stops the node at once.  nothing is moved to other nodes.
	Close() error
	// Drain moves the node's primaries away and takes it out of every
	// in-sync set, so a Close after it loses nothing.
	Drain(ctx context.Context) error
}

// testCluster runs n nodes on one MemNetwork and one MemCoordinator with a
// real clock and short timeouts.
type testCluster struct {
	t     testing.TB
	names []string
	net   *MemNetwork
	co    *MemCoordinator
	// start builds and starts one node.  every start is a new process as
	// far as the cluster can tell, so the node must draw a new boot ID.
	start func(name string, tr Transport, co Coordinator) (testNode, error)

	mu    sync.Mutex
	nodes map[string]testNode
}

// newTestCluster returns a cluster with the nodes n0, n1 and so on.  no node
// runs until StartAll.
func newTestCluster(t testing.TB, n int, start func(name string, tr Transport, co Coordinator) (testNode, error)) *testCluster {
	t.Helper()
	c := &testCluster{
		t:     t,
		net:   NewMemNetwork(),
		co:    NewMemCoordinator(testHeartbeatInterval, testHeartbeatTimeout),
		start: start,
		nodes: map[string]testNode{},
	}
	for i := range n {
		c.names = append(c.names, fmt.Sprintf("n%d", i))
	}
	t.Cleanup(c.Close)
	return c
}

// Init creates the shards with every node of the cluster as a candidate.
func (c *testCluster) Init(shards, rf int) error {
	return c.co.Node("harness").Propose(context.Background(), Init{
		Shards: shards,
		RF:     rf,
		Nodes:  c.names,
	})
}

// StartAll starts every node that is not running.
func (c *testCluster) StartAll() error {
	for _, name := range c.names {
		if c.Running(name) {
			continue
		}
		if err := c.Restart(name); err != nil {
			return err
		}
	}
	return nil
}

// Close stops every node with no drain and stops the coordinator.
func (c *testCluster) Close() {
	for _, name := range c.names {
		c.Kill(name)
	}
	_ = c.co.Close()
}

// Table returns the newest table, also when every node is cut off.
func (c *testCluster) Table() *Table {
	tbl, _ := c.co.Node("harness").Table()
	return tbl
}

func (c *testCluster) Running(name string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nodes[name] != nil
}

// Kill drops a node with no drain.  the node is cut off from the coordinator
// and from every other node before its streams break.  without that order
// the dying node sees its followers' streams break and removes them from the
// in-sync sets, which a process that crashed cannot do.  the node stays cut
// off until Restart.
func (c *testCluster) Kill(name string) {
	c.mu.Lock()
	node := c.nodes[name]
	delete(c.nodes, name)
	c.mu.Unlock()
	if node == nil {
		return
	}
	c.co.Cut(name)
	c.net.Isolate(name)
	c.net.Kill(name)
	_ = node.Close()
}

// Restart starts a node again through the factory, with no state.  a node
// that still runs is killed first.  it also heals every partition of the
// node and restores its coordinator handle.
func (c *testCluster) Restart(name string) error {
	c.Kill(name)
	c.net.Rejoin(name)
	c.co.Restore(name)
	node, err := c.start(name, c.net.Transport(name), c.co.Node(name))
	if err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}
	c.mu.Lock()
	c.nodes[name] = node
	c.mu.Unlock()
	return nil
}

// Drain is the planned stop.  it drains the node and then closes it.
func (c *testCluster) Drain(name string) error {
	c.mu.Lock()
	node := c.nodes[name]
	c.mu.Unlock()
	if node == nil {
		return fmt.Errorf("drain %s: the node does not run", name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := node.Drain(ctx); err != nil {
		return fmt.Errorf("drain %s: %w", name, err)
	}
	c.mu.Lock()
	delete(c.nodes, name)
	c.mu.Unlock()
	return node.Close()
}

// safeToKill reports whether a kill of the node loses no write that was
// answered at level all.  that is the case when every other node runs and
// every in-sync set the node is in has one more member.
func (c *testCluster) safeToKill(name string) bool {
	for _, other := range c.names {
		if other != name && !c.Running(other) {
			return false
		}
	}
	for _, row := range c.Table().Shards {
		if len(row.InSync) == 1 && row.InSync[0] == name {
			return false
		}
	}
	return true
}

// healAll removes every fault and starts every node that does not run.
func (c *testCluster) healAll() error {
	for i, a := range c.names {
		c.net.Rejoin(a)
		c.co.Restore(a)
		for _, b := range c.names[i+1:] {
			c.net.Unstall(a, b)
			c.net.SetDelay(a, b, 0)
		}
	}
	c.co.SetProposeDelay(0)
	c.co.StartController()
	return c.StartAll()
}

type faultKind int

const (
	faultPartition faultKind = iota
	faultIsolate
	faultStall
	faultDelay
	faultCut
	faultProposeDelay
	faultStopController
	faultKill
)

var allFaultKinds = []faultKind{
	faultPartition,
	faultIsolate,
	faultStall,
	faultDelay,
	faultCut,
	faultProposeDelay,
	faultStopController,
	faultKill,
}

type faultConfig struct {
	seed int64
	// faults is how many faults the schedule draws.  each one is undone
	// later, so it gives two actions.  a kill drawn while the node of the
	// kill before it is still down is dropped.
	faults int
	// span is the time from the first action to the last.
	span time.Duration
	// kinds limits the faults.  nil means every kind.
	kinds []faultKind
}

type faultAction struct {
	at   time.Duration
	name string
	do   func()
}

// faultSchedule is a sequence of timed fault actions.  the same seed and
// cluster size give the same sequence.
type faultSchedule struct {
	t       testing.TB
	c       *testCluster
	actions []faultAction
}

// newFaultSchedule draws the schedule.  nothing happens until run.  kills
// never overlap, so at most one node is down at a time.
func newFaultSchedule(t testing.TB, c *testCluster, cfg faultConfig) *faultSchedule {
	t.Helper()
	s := &faultSchedule{
		t: t,
		c: c,
	}
	if len(c.names) < 2 {
		t.Fatalf("a fault schedule needs two nodes or more, the cluster has %d", len(c.names))
	}
	kinds := cfg.kinds
	if kinds == nil {
		kinds = allFaultKinds
	}
	rng := rand.New(rand.NewSource(cfg.seed))
	node := func() string { return c.names[rng.Intn(len(c.names))] }
	pair := func() (string, string) {
		i := rng.Intn(len(c.names))
		j := (i + 1 + rng.Intn(len(c.names)-1)) % len(c.names)
		return c.names[i], c.names[j]
	}

	var killFreeAt time.Duration
	for i := range cfg.faults {
		// faults start at even steps over the first three quarters of
		// the span and last up to a quarter of it.
		at := cfg.span * 3 / 4 * time.Duration(i) / time.Duration(cfg.faults)
		end := at + cfg.span/20 + time.Duration(rng.Int63n(int64(cfg.span/5)))
		kind := kinds[rng.Intn(len(kinds))]
		if kind == faultKill && at < killFreeAt {
			// the node of the kill before this one is still down.
			continue
		}

		var name string
		var do, undo func()
		switch kind {
		case faultPartition:
			a, b := pair()
			name = fmt.Sprintf("partition %s %s", a, b)
			do = func() { c.net.Partition(a, b) }
			undo = func() { c.net.Heal(a, b) }
		case faultIsolate:
			n := node()
			name = "isolate " + n
			do = func() { c.net.Isolate(n) }
			undo = func() { c.net.Rejoin(n) }
		case faultStall:
			a, b := pair()
			name = fmt.Sprintf("stall %s %s", a, b)
			do = func() { c.net.Stall(a, b) }
			undo = func() { c.net.Unstall(a, b) }
		case faultDelay:
			a, b := pair()
			d := time.Duration(1+rng.Intn(20)) * time.Millisecond
			name = fmt.Sprintf("delay %s %s %s", a, b, d)
			do = func() { c.net.SetDelay(a, b, d) }
			undo = func() { c.net.SetDelay(a, b, 0) }
		case faultCut:
			n := node()
			name = "cut " + n
			do = func() { c.co.Cut(n) }
			undo = func() { c.co.Restore(n) }
		case faultProposeDelay:
			d := time.Duration(1+rng.Intn(50)) * time.Millisecond
			name = fmt.Sprintf("propose delay %s", d)
			do = func() { c.co.SetProposeDelay(d) }
			undo = func() { c.co.SetProposeDelay(0) }
		case faultStopController:
			name = "stop controller"
			do = c.co.StopController
			undo = c.co.StartController
		case faultKill:
			n := node()
			killFreeAt = end
			name = "kill " + n
			do = func() {
				if !c.safeToKill(n) {
					t.Logf("fault kill %s skipped, it has the last in-sync copy of a shard or another node is down", n)
					return
				}
				c.Kill(n)
			}
			undo = func() {
				if c.Running(n) {
					return
				}
				if err := c.Restart(n); err != nil {
					t.Errorf("fault restart: %v", err)
				}
			}
		}
		s.actions = append(s.actions,
			faultAction{
				at:   at,
				name: name,
				do:   do,
			},
			faultAction{
				at:   end,
				name: "end " + name,
				do:   undo,
			},
		)
	}
	sort.SliceStable(s.actions, func(i, j int) bool { return s.actions[i].at < s.actions[j].at })
	return s
}

// describe returns one line per action, in the order run applies them.
func (s *faultSchedule) describe() []string {
	out := make([]string, len(s.actions))
	for i, a := range s.actions {
		out[i] = fmt.Sprintf("+%s %s", a.at.Round(time.Millisecond), a.name)
	}
	return out
}

// run applies the actions at their times and logs each one.  when it
// returns, also after ctx is done, every fault is removed and every node
// runs.  the test must not end before run returns.
func (s *faultSchedule) run(ctx context.Context) {
	defer func() {
		s.t.Logf("fault heal all")
		if err := s.c.healAll(); err != nil {
			s.t.Errorf("fault heal all: %v", err)
		}
	}()
	lines := s.describe()
	start := time.Now()
	for i, a := range s.actions {
		select {
		case <-time.After(a.at - time.Since(start)):
		case <-ctx.Done():
			return
		}
		s.t.Logf("fault %s", lines[i])
		a.do()
	}
}

type ledgerConfig struct {
	seed    int64
	workers int
	// keys is the number of accounts.  each account has one function and
	// so one function concurrency key.
	keys int
	// limit is the concurrency limit of every key.
	limit int
	// grantLevels are the levels an acquire draws from.  nil means all
	// four.
	grantLevels []constraintapi.CommitLevel
	// otherLevels are the levels an extend or a release draws from.  nil
	// means all four.
	otherLevels []constraintapi.CommitLevel
	// duration is the lease duration of every acquire and extend.
	duration time.Duration
}

var allCommitLevels = []constraintapi.CommitLevel{
	constraintapi.CommitLevelDefault,
	constraintapi.CommitLevelPrimary,
	constraintapi.CommitLevelTwo,
	constraintapi.CommitLevelAll,
}

type ledgerKey struct {
	account uuid.UUID
	env     uuid.UUID
	fn      uuid.UUID
}

// ledgerEvent is one answer the workload got.
type ledgerEvent struct {
	// op is grant, extend or release.
	op  string
	key int
	// idem is the lease idempotency key.  it names the lease over all of
	// its lease IDs.
	idem string
	// lease is the lease ID in the answer.  for a release it is the ID
	// that was released.
	lease ulid.ULID
	level constraintapi.CommitLevel
}

type leaseState int

const (
	// leaseLive means a grant was answered and no release was sent.
	leaseLive leaseState = iota
	// leaseReleasing means a release was sent and not answered.
	leaseReleasing
	leaseReleased
	// leaseGone means an extend found no lease.
	leaseGone
)

type ledgerLease struct {
	key  int
	idem string
	// id and level come from the newest answered grant or extend.
	id    ulid.ULID
	level constraintapi.CommitLevel
	// safeUntil is the expiry from the newest answer at level all.  until
	// then every in-sync replica has the lease, so it must count toward
	// the limit on any primary.
	safeUntil time.Time
	state     leaseState
	// unsure is set when an extend failed.  the store may have applied it,
	// and then the lease has an ID the ledger does not know.
	unsure bool
}

// ledger drives acquire, extend and release against a CapacityManager and
// records every answer.  it works against any manager, so the same checks
// run on a fake, on one store and on a cluster client.
type ledger struct {
	cfg  ledgerConfig
	keys []ledgerKey
	next atomic.Uint64

	mu     sync.Mutex
	events []ledgerEvent
	leases []*ledgerLease
	// live lists, per key, the leases that are in state leaseLive.
	live []map[*ledgerLease]struct{}
	// peak is, per key, the highest count of live leases inside their
	// safeUntil that the ledger saw at the answer of a grant.
	peak []int
	// weakGrants counts answered grants below level all.
	weakGrants int
	// lost lists leases that an extend did not find before their expiry,
	// though they were answered at level two or all.
	lost []string
	// failures lists errors that were not retryable.
	failures []string
}

func newLedger(cfg ledgerConfig) *ledger {
	if cfg.grantLevels == nil {
		cfg.grantLevels = allCommitLevels
	}
	if cfg.otherLevels == nil {
		cfg.otherLevels = allCommitLevels
	}
	if cfg.duration == 0 {
		cfg.duration = 10 * time.Second
	}
	l := &ledger{
		cfg:  cfg,
		live: make([]map[*ledgerLease]struct{}, cfg.keys),
		peak: make([]int, cfg.keys),
	}
	// the IDs come from the seed, so a run with the same seed uses the
	// same shards.
	rng := rand.New(rand.NewSource(cfg.seed))
	id := func() uuid.UUID {
		var u uuid.UUID
		rng.Read(u[:])
		return u
	}
	for i := range cfg.keys {
		l.keys = append(l.keys, ledgerKey{
			account: id(),
			env:     id(),
			fn:      id(),
		})
		l.live[i] = map[*ledgerLease]struct{}{}
	}
	return l
}

// run drives the workload from cfg.workers goroutines until ctx is done.
func (l *ledger) run(ctx context.Context, m constraintapi.CapacityManager) {
	var wg sync.WaitGroup
	for w := range l.cfg.workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.work(ctx, m, rand.New(rand.NewSource(l.cfg.seed+int64(w)+1)))
		}()
	}
	wg.Wait()
}

// work is one worker.  a lease belongs to the worker that acquired it, so
// no two operations on one lease run at the same time.
func (l *ledger) work(ctx context.Context, m constraintapi.CapacityManager, rng *rand.Rand) {
	var own []*ledgerLease
	for ctx.Err() == nil {
		var ok bool
		if len(own) == 0 || rng.Intn(10) < 4 {
			var lease *ledgerLease
			lease, ok = l.acquire(ctx, m, rng)
			if lease != nil {
				own = append(own, lease)
			}
		} else {
			i := rng.Intn(len(own))
			var done bool
			if own[i].state == leaseReleasing || rng.Intn(2) == 0 {
				done, ok = l.release(ctx, m, rng, own[i])
			} else {
				done, ok = l.extend(ctx, m, rng, own[i])
			}
			if done {
				own = slices.Delete(own, i, i+1)
			}
		}
		if !ok {
			// retry later.  the store is at the limit or cannot answer now.
			select {
			case <-time.After(time.Millisecond):
			case <-ctx.Done():
			}
		}
	}
}

func ledgerSource() constraintapi.LeaseSource {
	return constraintapi.LeaseSource{
		Service:           constraintapi.ServiceExecutor,
		Location:          constraintapi.CallerLocationItemLease,
		RunProcessingMode: constraintapi.RunProcessingModeBackground,
	}
}

func (l *ledger) constraintConfig() constraintapi.ConstraintConfig {
	return constraintapi.ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: constraintapi.ConcurrencyConfig{
			FunctionConcurrency: l.cfg.limit,
		},
	}
}

func ledgerConstraints() []constraintapi.ConstraintItem {
	return []constraintapi.ConstraintItem{
		{
			Kind: constraintapi.ConstraintKindConcurrency,
			Concurrency: &constraintapi.ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
		},
	}
}

// acquire asks for one lease on a random key.  it returns the lease if one
// was answered.  ok is false when the worker must wait before the next call.
func (l *ledger) acquire(ctx context.Context, m constraintapi.CapacityManager, rng *rand.Rand) (lease *ledgerLease, ok bool) {
	key := rng.Intn(len(l.keys))
	level := l.cfg.grantLevels[rng.Intn(len(l.cfg.grantLevels))]
	idem := fmt.Sprintf("ledger-%d-%d", l.cfg.seed, l.next.Add(1))
	k := l.keys[key]
	resp, err := m.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
		IdempotencyKey:       idem + "-acquire",
		AccountID:            k.account,
		EnvID:                k.env,
		FunctionID:           k.fn,
		Configuration:        l.constraintConfig(),
		Constraints:          ledgerConstraints(),
		Amount:               1,
		LeaseIdempotencyKeys: []string{idem},
		LeaseRunIDs:          map[string]ulid.ULID{},
		CurrentTime:          time.Now(),
		Duration:             l.cfg.duration,
		MaximumLifetime:      time.Hour,
		Source:               ledgerSource(),
		CommitLevel:          level,
	})
	if err != nil {
		// the grant may exist.  no caller was answered, so it is not in
		// the ledger and the store reclaims it at its expiry.
		l.failed(ctx, "acquire", err)
		return nil, false
	}
	if len(resp.Leases) == 0 {
		return nil, false
	}

	lease = &ledgerLease{
		key:   key,
		idem:  idem,
		id:    resp.Leases[0].LeaseID,
		level: level,
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if waitsForAll(level) {
		lease.safeUntil = lease.id.Timestamp()
	} else {
		l.weakGrants++
	}
	l.leases = append(l.leases, lease)
	l.live[key][lease] = struct{}{}
	l.events = append(l.events, ledgerEvent{
		op:    "grant",
		key:   key,
		idem:  idem,
		lease: lease.id,
		level: level,
	})
	// the store took the capacity before it answered, and the ledger
	// drops a lease from live before it sends the release.  so every
	// lease counted here was live in the store at one moment.
	now := time.Now()
	count := 0
	for other := range l.live[key] {
		if now.Before(other.safeUntil) {
			count++
		}
	}
	l.peak[key] = max(l.peak[key], count)
	return lease, true
}

// extend extends a lease.  done is true when the lease is gone and the
// worker must forget it.
func (l *ledger) extend(ctx context.Context, m constraintapi.CapacityManager, rng *rand.Rand, lease *ledgerLease) (done, ok bool) {
	level := l.cfg.otherLevels[rng.Intn(len(l.cfg.otherLevels))]
	resp, err := m.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
		IdempotencyKey: fmt.Sprintf("%s-extend-%d", lease.idem, l.next.Add(1)),
		AccountID:      l.keys[lease.key].account,
		LeaseID:        lease.id,
		Duration:       l.cfg.duration,
		Source:         ledgerSource(),
		CommitLevel:    level,
	})
	if err != nil {
		l.failed(ctx, "extend", err)
		l.mu.Lock()
		lease.unsure = true
		l.mu.Unlock()
		return false, false
	}
	l.extended(lease, resp.LeaseID, level)
	return resp.LeaseID == nil, true
}

// extended records the answer of an extend.  a nil ID means the store did
// not find the lease.
func (l *ledger) extended(lease *ledgerLease, id *ulid.ULID, level constraintapi.CommitLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if id == nil {
		// the answer came after the store looked, so a lease that is
		// still inside its expiry now was inside it then.
		if survivesFailover(lease.level) && !lease.unsure && time.Now().Before(lease.id.Timestamp()) {
			l.lost = append(l.lost, fmt.Sprintf("lease %s (%s) answered at level %d", lease.id, lease.idem, lease.level))
		}
		lease.state = leaseGone
		delete(l.live[lease.key], lease)
		return
	}
	lease.id = *id
	lease.level = level
	lease.unsure = false
	if waitsForAll(level) {
		lease.safeUntil = id.Timestamp()
	}
	l.events = append(l.events, ledgerEvent{
		op:    "extend",
		key:   lease.key,
		idem:  lease.idem,
		lease: *id,
		level: level,
	})
}

// release releases a lease.  done is true when the release was answered.
func (l *ledger) release(ctx context.Context, m constraintapi.CapacityManager, rng *rand.Rand, lease *ledgerLease) (done, ok bool) {
	level := l.cfg.otherLevels[rng.Intn(len(l.cfg.otherLevels))]
	// the store frees the capacity before it answers.  the lease must
	// leave the live count first, or a grant that reuses the capacity
	// reads as one over the limit.
	l.mu.Lock()
	lease.state = leaseReleasing
	delete(l.live[lease.key], lease)
	l.mu.Unlock()

	_, err := m.Release(ctx, &constraintapi.CapacityReleaseRequest{
		IdempotencyKey: fmt.Sprintf("%s-release-%d", lease.idem, l.next.Add(1)),
		AccountID:      l.keys[lease.key].account,
		LeaseID:        lease.id,
		Source:         ledgerSource(),
		CommitLevel:    level,
	})
	if err != nil {
		l.failed(ctx, "release", err)
		return false, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	lease.state = leaseReleased
	l.events = append(l.events, ledgerEvent{
		op:    "release",
		key:   lease.key,
		idem:  lease.idem,
		lease: lease.id,
		level: level,
	})
	return true, true
}

// failed records an error that is not retryable.  a retryable error and an
// error after ctx is done mean retry later.
func (l *ledger) failed(ctx context.Context, op string, err errs.Error) {
	if ctx.Err() != nil || err.Retryable() {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures = append(l.failures, fmt.Sprintf("%s: %v", op, err))
}

// waitsForAll reports whether a grant or an extend at this level is
// answered only when every in-sync replica has it.
func waitsForAll(level constraintapi.CommitLevel) bool {
	return level == constraintapi.CommitLevelDefault || level == constraintapi.CommitLevelAll
}

// survivesFailover reports whether a grant or an extend answered at this
// level is on one in-sync follower or more.
func survivesFailover(level constraintapi.CommitLevel) bool {
	return level != constraintapi.CommitLevelPrimary
}

// retryLater calls fn until it returns no error or an error that is not
// retryable, or until ctx is done.
func retryLater(ctx context.Context, fn func() errs.Error) error {
	for {
		err := fn()
		if err == nil {
			return nil
		}
		if !err.Retryable() {
			return err
		}
		select {
		case <-time.After(2 * time.Millisecond):
		case <-ctx.Done():
			return fmt.Errorf("%w, last error: %v", ctx.Err(), err)
		}
	}
}

// checkErrors fails if the workload got an error that was not retryable.
func (l *ledger) checkErrors() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.failures) > 0 {
		return fmt.Errorf("ledger: %d errors were not retryable, first: %s", len(l.failures), l.failures[0])
	}
	return nil
}

// checkExtendable extends, through m, every lease that was answered at level
// two or all, was not released and is inside its expiry.  it fails if one of
// them is not found, now or during the run.  call it after run has returned.
func (l *ledger) checkExtendable(ctx context.Context, m constraintapi.CapacityManager) error {
	l.mu.Lock()
	leases := slices.Clone(l.leases)
	l.mu.Unlock()

	for _, lease := range leases {
		if lease.state != leaseLive || lease.unsure || !survivesFailover(lease.level) {
			continue
		}
		if !time.Now().Before(lease.id.Timestamp()) {
			continue
		}
		var resp *constraintapi.CapacityExtendLeaseResponse
		err := retryLater(ctx, func() errs.Error {
			var err errs.InternalError
			resp, err = m.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
				IdempotencyKey: fmt.Sprintf("%s-check-%d", lease.idem, l.next.Add(1)),
				AccountID:      l.keys[lease.key].account,
				LeaseID:        lease.id,
				Duration:       l.cfg.duration,
				Source:         ledgerSource(),
				CommitLevel:    constraintapi.CommitLevelAll,
			})
			if err != nil {
				// the store may have applied the extend.  a retry
				// that finds no lease proves nothing then.
				l.mu.Lock()
				lease.unsure = true
				l.mu.Unlock()
				return err
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("ledger: extend of lease %s: %w", lease.id, err)
		}
		l.extended(lease, resp.LeaseID, constraintapi.CommitLevelAll)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.lost) > 0 {
		return fmt.Errorf("ledger: %d answered leases were lost:\n%s", len(l.lost), strings.Join(l.lost, "\n"))
	}
	return nil
}

// checkLimits fails if the count of live leases of one key passed the limit
// at a moment the ledger saw, or if m now reports less usage for a key than
// the ledger has live leases.  it applies only when every grant was answered
// at level all, and fails otherwise.  call it after run has returned.
func (l *ledger) checkLimits(ctx context.Context, m constraintapi.CapacityManager) error {
	l.mu.Lock()
	weak := l.weakGrants
	peak := slices.Clone(l.peak)
	l.mu.Unlock()
	if weak > 0 {
		return fmt.Errorf("ledger: %d grants were answered below level all, the limit check does not apply", weak)
	}
	for key, n := range peak {
		if n > l.cfg.limit {
			return fmt.Errorf("ledger: key %d had %d live leases, the limit is %d", key, n, l.cfg.limit)
		}
	}

	for key, k := range l.keys {
		var resp *constraintapi.CapacityCheckResponse
		err := retryLater(ctx, func() errs.Error {
			var userErr errs.UserError
			var err errs.InternalError
			resp, userErr, err = m.Check(ctx, &constraintapi.CapacityCheckRequest{
				AccountID:     k.account,
				EnvID:         k.env,
				FunctionID:    k.fn,
				Configuration: l.constraintConfig(),
				Constraints:   ledgerConstraints(),
			})
			if userErr != nil {
				return userErr
			}
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("ledger: check of key %d: %w", key, err)
		}
		if len(resp.Usage) != 1 {
			return errors.New("ledger: the check answered no usage for the concurrency constraint")
		}
		// count after the answer.  a lease inside safeUntil now was
		// inside it when the store counted.
		now := time.Now()
		count := 0
		l.mu.Lock()
		for lease := range l.live[key] {
			if now.Before(lease.safeUntil) {
				count++
			}
		}
		l.mu.Unlock()
		if resp.Usage[0].Used < count {
			return fmt.Errorf("ledger: the store counts %d leases for key %d, the ledger has %d live", resp.Usage[0].Used, key, count)
		}
	}
	return nil
}
