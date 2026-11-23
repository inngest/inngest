package cluster

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// fakeNode does what the harness and the coordinator see of a real node.  it
// listens, registers with a new boot ID and reports.  it has no shards.
type fakeNode struct {
	name    string
	bootID  string
	ln      io.Closer
	quit    chan struct{}
	done    chan struct{}
	drained atomic.Bool
	closed  atomic.Bool
}

func startFakeNode(name string, tr Transport, co Coordinator) (*fakeNode, error) {
	// the listener answers one message with the same message.
	ln, err := tr.Listen(name, func(s Stream) {
		defer s.Close()
		if m, err := s.Recv(); err == nil {
			_ = s.Send(m)
			// the stream stays open until the dialer closes it, so
			// the answer is not cut short.
			_, _ = s.Recv()
		}
	})
	if err != nil {
		return nil, err
	}
	n := &fakeNode{
		name:   name,
		bootID: uuid.NewString(),
		ln:     ln,
		quit:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	// a fault can cut the node off while it starts.  it tries until the
	// coordinator answers, as a real node does.
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := co.Propose(context.Background(), RegisterNode{
			ID:     name,
			Addr:   name,
			BootID: n.bootID,
		})
		if err == nil {
			break
		}
		if !errors.Is(err, ErrUnavailable) || time.Now().After(deadline) {
			_ = ln.Close()
			return nil, err
		}
		time.Sleep(time.Millisecond)
	}
	go func() {
		defer close(n.done)
		tick := time.NewTicker(testHeartbeatInterval)
		defer tick.Stop()
		for {
			_ = co.Report(context.Background(), Report{
				Node:   name,
				BootID: n.bootID,
			})
			select {
			case <-n.quit:
				return
			case <-tick.C:
			}
		}
	}()
	return n, nil
}

func (n *fakeNode) Drain(context.Context) error {
	n.drained.Store(true)
	return nil
}

func (n *fakeNode) Close() error {
	if n.closed.Swap(true) {
		return nil
	}
	close(n.quit)
	<-n.done
	return n.ln.Close()
}

// fakeCluster returns a started cluster of fake nodes and a function that
// returns every fake that was started for a name, oldest first.
func fakeCluster(t *testing.T, n, shards, rf int) (*testCluster, func(name string) []*fakeNode) {
	t.Helper()
	var mu sync.Mutex
	started := map[string][]*fakeNode{}
	c := newTestCluster(t, n, func(name string, tr Transport, co Coordinator) (testNode, error) {
		node, err := startFakeNode(name, tr, co)
		if err != nil {
			return nil, err
		}
		mu.Lock()
		defer mu.Unlock()
		started[name] = append(started[name], node)
		return node, nil
	})
	require.NoError(t, c.Init(shards, rf))
	require.NoError(t, c.StartAll())
	return c, func(name string) []*fakeNode {
		mu.Lock()
		defer mu.Unlock()
		return append([]*fakeNode(nil), started[name]...)
	}
}

// requireEcho dials from one node to another and checks that a message
// comes back.
func requireEcho(t *testing.T, c *testCluster, from, to string) {
	t.Helper()
	s, err := c.net.Transport(from).Dial(context.Background(), to)
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.Send("ping"))
	got := requireSoon(t, recvAsync(s))
	require.NoError(t, got.err)
	require.Equal(t, "ping", got.m)
}

func TestHarnessKillRestartDrain(t *testing.T) {
	c, fakes := fakeCluster(t, 3, 4, 3)
	harness := c.co.Node("harness")
	require.Len(t, c.Table().Nodes, 3)
	require.True(t, c.safeToKill("n0"))
	requireEcho(t, c, "n1", "n0")

	c.Kill("n0")
	require.False(t, c.Running("n0"))
	old := fakes("n0")[0]
	require.True(t, old.closed.Load())
	require.False(t, old.drained.Load())
	_, err := c.net.Transport("n1").Dial(context.Background(), "n0")
	require.ErrorIs(t, err, ErrDialRefused)
	require.ErrorIs(t, c.co.Node("n0").Propose(context.Background(), RegisterNode{
		ID:     "n0",
		BootID: "x",
	}), ErrUnavailable)
	// one node is down, so a second kill can lose the last copy.
	require.False(t, c.safeToKill("n1"))

	// the controller sees the reports stop and moves the primaries.
	tbl := waitTable(t, harness, func(tbl *Table) bool {
		for _, row := range tbl.Shards {
			if row.Primary == "n0" {
				return false
			}
		}
		return true
	})
	require.Contains(t, tbl.Shards[0].InSync, "n0")

	require.NoError(t, c.Restart("n0"))
	require.True(t, c.Running("n0"))
	require.Len(t, fakes("n0"), 2)
	tbl = c.Table()
	require.Equal(t, fakes("n0")[1].bootID, tbl.Nodes["n0"].BootID)
	require.NotEqual(t, old.bootID, tbl.Nodes["n0"].BootID)
	for _, row := range tbl.Shards {
		require.NotContains(t, row.InSync, "n0")
	}
	requireEcho(t, c, "n1", "n0")
	requireEcho(t, c, "n0", "n2")
	require.True(t, c.safeToKill("n1"))

	// a restart of a node that runs kills it first.
	require.NoError(t, c.Restart("n2"))
	require.Len(t, fakes("n2"), 2)
	require.True(t, fakes("n2")[0].closed.Load())
	require.False(t, fakes("n2")[1].closed.Load())

	require.NoError(t, c.Drain("n1"))
	require.False(t, c.Running("n1"))
	require.True(t, fakes("n1")[0].drained.Load())
	require.True(t, fakes("n1")[0].closed.Load())
	require.Error(t, c.Drain("n1"))
	require.NoError(t, c.StartAll())
	require.True(t, c.Running("n1"))
}

func TestHarnessSafeToKillLastInSyncCopy(t *testing.T) {
	c, _ := fakeCluster(t, 2, 1, 2)
	require.True(t, c.safeToKill("n0"))
	require.NoError(t, c.co.Node("harness").Propose(context.Background(), ShrinkInSync{
		Shard: 0,
		Epoch: 1,
		Node:  "n1",
	}))
	require.False(t, c.safeToKill("n0"))
	require.True(t, c.safeToKill("n1"))
}

func TestFaultScheduleIsReproducible(t *testing.T) {
	c, _ := fakeCluster(t, 3, 4, 3)
	cfg := faultConfig{
		seed:   42,
		faults: 40,
		span:   time.Second,
	}
	first := newFaultSchedule(t, c, cfg).describe()
	require.Equal(t, first, newFaultSchedule(t, c, cfg).describe())
	t.Logf("schedule for seed 42:\n%s", strings.Join(first, "\n"))

	cfg.seed = 43
	require.NotEqual(t, first, newFaultSchedule(t, c, cfg).describe())

	// every kind is drawn, every fault has its end, and the times rise.
	all := strings.Join(first, "\n")
	for _, word := range []string{"partition", "isolate", "stall", "delay n", "cut", "propose delay", "stop controller", "kill"} {
		require.Contains(t, all, " "+word)
	}
	ends := 0
	for _, line := range first {
		if strings.Contains(line, " end ") {
			ends++
		}
	}
	require.Equal(t, len(first), 2*ends)
	s := newFaultSchedule(t, c, cfg)
	for i := 1; i < len(s.actions); i++ {
		require.LessOrEqual(t, s.actions[i-1].at, s.actions[i].at)
	}
	require.LessOrEqual(t, s.actions[len(s.actions)-1].at, cfg.span)

	// kills never overlap.
	down := ""
	for _, a := range newFaultSchedule(t, c, faultConfig{
		seed:   7,
		faults: 30,
		span:   time.Second,
		kinds:  []faultKind{faultKill},
	}).actions {
		if node, ok := strings.CutPrefix(a.name, "kill "); ok {
			require.Empty(t, down)
			down = node
		} else {
			require.Equal(t, "end kill "+down, a.name)
			down = ""
		}
	}
}

// requireHealed checks that no fault is left on the cluster.
func requireHealed(t *testing.T, c *testCluster) {
	t.Helper()
	for _, a := range c.names {
		require.True(t, c.Running(a), "node %s does not run", a)
		require.NoError(t, c.co.Node(a).Report(context.Background(), Report{
			Node: a,
		}))
		for _, b := range c.names {
			if a != b {
				requireEcho(t, c, a, b)
			}
		}
	}
	start := time.Now()
	require.ErrorIs(t, c.co.Node("n0").Propose(context.Background(), Init{}), ErrRejected)
	require.Less(t, time.Since(start), time.Second)
}

func TestFaultScheduleRunHealsEverything(t *testing.T) {
	c, _ := fakeCluster(t, 3, 4, 3)
	s := newFaultSchedule(t, c, faultConfig{
		seed:   1,
		faults: 30,
		span:   600 * time.Millisecond,
	})
	start := time.Now()
	s.run(context.Background())
	require.GreaterOrEqual(t, time.Since(start), s.actions[len(s.actions)-1].at)
	requireHealed(t, c)
}

func TestHarnessHealAll(t *testing.T) {
	c, _ := fakeCluster(t, 3, 4, 3)
	c.net.Partition("n0", "n1")
	c.net.Isolate("n2")
	c.net.Stall("n0", "n1")
	c.net.Stall("n1", "n2")
	c.net.SetDelay("n0", "n2", time.Minute)
	c.co.Cut("n0")
	c.co.SetProposeDelay(time.Minute)
	c.co.StopController()
	c.Kill("n1")

	require.NoError(t, c.healAll())
	requireHealed(t, c)

	// the controller runs again.  it replaces the primary of a dead node.
	c.Kill("n0")
	waitTable(t, c.co.Node("harness"), func(tbl *Table) bool {
		for _, row := range tbl.Shards {
			if row.Primary == "n0" {
				return false
			}
		}
		return true
	})
}

func TestFaultScheduleRunHealsWhenStoppedEarly(t *testing.T) {
	c, _ := fakeCluster(t, 3, 4, 3)
	s := newFaultSchedule(t, c, faultConfig{
		seed:   3,
		faults: 200,
		span:   20 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	start := time.Now()
	s.run(ctx)
	require.Less(t, time.Since(start), 10*time.Second)
	requireHealed(t, c)
}

// fakeManager is a CapacityManager with one function concurrency limit and
// switches that break it.
type fakeManager struct {
	mu    sync.Mutex
	limit int
	// leases maps a live lease to its function.
	leases map[ulid.ULID]uuid.UUID
	used   map[uuid.UUID]int
	seq    uint64
	calls  int
	grants int
	// levels counts the acquire, extend and release requests by level.
	levels map[constraintapi.CommitLevel]int

	// retryEvery makes every nth call fail with a retryable error before
	// it does anything.
	retryEvery int
	// fatalEvery does the same with an error that is not retryable.
	fatalEvery int
	// loseEvery makes the manager answer every nth grant and forget it.
	loseEvery int
	// ignoreLimit makes the manager grant with no limit.
	ignoreLimit bool
	// failAppliedExtend makes an extend apply and then fail with a
	// retryable error.
	failAppliedExtend bool
	// hideUsage is subtracted from the usage that Check reports.
	hideUsage int
}

func newFakeManager(limit int) *fakeManager {
	return &fakeManager{
		limit:  limit,
		leases: map[ulid.ULID]uuid.UUID{},
		used:   map[uuid.UUID]int{},
		levels: map[constraintapi.CommitLevel]int{},
	}
}

// call counts one request and returns the injected error, if one is due.
func (f *fakeManager) call(level constraintapi.CommitLevel) errs.InternalError {
	f.calls++
	f.levels[level]++
	if f.retryEvery > 0 && f.calls%f.retryEvery == 0 {
		return errs.Wrap(0, true, "fake manager: try again")
	}
	if f.fatalEvery > 0 && f.calls%f.fatalEvery == 0 {
		return errs.Wrap(0, false, "fake manager: broken")
	}
	return nil
}

// newID returns a lease ID with the expiry in its time part, as the real
// managers do.
func (f *fakeManager) newID(d time.Duration) ulid.ULID {
	f.seq++
	var id ulid.ULID
	_ = id.SetTime(ulid.Timestamp(time.Now().Add(d)))
	var entropy [10]byte
	binary.BigEndian.PutUint64(entropy[2:], f.seq)
	_ = id.SetEntropy(entropy[:])
	return id
}

func (f *fakeManager) Check(ctx context.Context, req *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	f.mu.Lock()
	defer f.mu.Unlock()
	used := f.used[req.FunctionID] - f.hideUsage
	return &constraintapi.CapacityCheckResponse{
		AvailableCapacity: f.limit - used,
		Usage: []constraintapi.ConstraintUsage{
			{
				Constraint: req.Constraints[0],
				Used:       used,
				Limit:      f.limit,
			},
		},
	}, nil, nil
}

func (f *fakeManager) Acquire(ctx context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.call(req.CommitLevel); err != nil {
		return nil, err
	}
	resp := &constraintapi.CapacityAcquireResponse{}
	if !f.ignoreLimit && f.used[req.FunctionID] >= f.limit {
		return resp, nil
	}
	id := f.newID(req.Duration)
	f.grants++
	if f.loseEvery == 0 || f.grants%f.loseEvery != 0 {
		f.leases[id] = req.FunctionID
		f.used[req.FunctionID]++
	}
	resp.Leases = []constraintapi.CapacityLease{
		{
			LeaseID:        id,
			IdempotencyKey: req.LeaseIdempotencyKeys[0],
		},
	}
	return resp, nil
}

func (f *fakeManager) ExtendLease(ctx context.Context, req *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.call(req.CommitLevel); err != nil {
		return nil, err
	}
	fn, ok := f.leases[req.LeaseID]
	if !ok {
		return &constraintapi.CapacityExtendLeaseResponse{}, nil
	}
	id := f.newID(req.Duration)
	delete(f.leases, req.LeaseID)
	f.leases[id] = fn
	if f.failAppliedExtend {
		return nil, errs.Wrap(0, true, "fake manager: the answer was lost")
	}
	return &constraintapi.CapacityExtendLeaseResponse{
		LeaseID: &id,
	}, nil
}

func (f *fakeManager) Release(ctx context.Context, req *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.call(req.CommitLevel); err != nil {
		return nil, err
	}
	if fn, ok := f.leases[req.LeaseID]; ok {
		delete(f.leases, req.LeaseID)
		f.used[fn]--
	}
	return &constraintapi.CapacityReleaseResponse{}, nil
}

// runLedger drives the workload for a short time.
func runLedger(l *ledger, m constraintapi.CapacityManager) {
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	l.run(ctx, m)
}

func TestLedgerMixedLevels(t *testing.T) {
	m := newFakeManager(3)
	m.retryEvery = 7
	l := newLedger(ledgerConfig{
		seed:    1,
		workers: 8,
		keys:    4,
		limit:   3,
	})
	runLedger(l, m)
	ctx := context.Background()

	require.NoError(t, l.checkErrors())
	require.NoError(t, l.checkExtendable(ctx, m))
	// some grants were below level all, so the limit check must refuse.
	require.ErrorContains(t, l.checkLimits(ctx, m), "does not apply")

	// every answer is in the ledger with its level and lease ID, and the
	// level went out on the request.
	ops := map[string]int{}
	levels := map[constraintapi.CommitLevel]int{}
	for _, e := range l.events {
		ops[e.op]++
		levels[e.level]++
		require.NotEqual(t, ulid.ULID{}, e.lease)
		require.NotEmpty(t, e.idem)
	}
	require.Equal(t, m.grants, ops["grant"])
	require.Positive(t, ops["extend"])
	require.Positive(t, ops["release"])
	require.Len(t, levels, 4)
	require.Len(t, m.levels, 4)
	require.Len(t, l.leases, m.grants)
}

func TestLedgerLimitCheckPasses(t *testing.T) {
	m := newFakeManager(3)
	m.retryEvery = 5
	l := newLedger(ledgerConfig{
		seed:        2,
		workers:     8,
		keys:        2,
		limit:       3,
		grantLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelAll, constraintapi.CommitLevelDefault},
	})
	runLedger(l, m)
	ctx := context.Background()

	require.NoError(t, l.checkErrors())
	require.NoError(t, l.checkExtendable(ctx, m))
	require.NoError(t, l.checkLimits(ctx, m))
	for _, peak := range l.peak {
		require.Positive(t, peak)
		require.LessOrEqual(t, peak, 3)
	}
	// the checks can run again.  the first run extended the leases.
	require.NoError(t, l.checkExtendable(ctx, m))
	require.NoError(t, l.checkLimits(ctx, m))
}

func TestLedgerDetectsLostLease(t *testing.T) {
	ctx := context.Background()
	rng := rand.New(rand.NewSource(1))
	cfg := ledgerConfig{
		keys:        1,
		limit:       10,
		grantLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelTwo},
		otherLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelAll},
	}

	t.Run("found by the check", func(t *testing.T) {
		m := newFakeManager(10)
		m.loseEvery = 2
		l := newLedger(cfg)
		kept, _ := l.acquire(ctx, m, rng)
		lost, _ := l.acquire(ctx, m, rng)
		require.NotNil(t, kept)
		require.NotNil(t, lost)
		keptID := kept.id

		err := l.checkExtendable(ctx, m)
		require.ErrorContains(t, err, "1 answered leases were lost")
		require.ErrorContains(t, err, lost.idem)
		require.NotContains(t, err.Error(), kept.idem)
		require.NotEqual(t, keptID, kept.id)
		require.Equal(t, leaseGone, lost.state)
	})

	t.Run("found by an extend of the workload", func(t *testing.T) {
		m := newFakeManager(10)
		m.loseEvery = 1
		l := newLedger(cfg)
		lease, _ := l.acquire(ctx, m, rng)
		done, ok := l.extend(ctx, m, rng, lease)
		require.True(t, done)
		require.True(t, ok)
		require.ErrorContains(t, l.checkExtendable(ctx, m), lease.idem)
	})

	t.Run("found under load", func(t *testing.T) {
		m := newFakeManager(3)
		m.loseEvery = 5
		l := newLedger(ledgerConfig{
			seed:        3,
			workers:     8,
			keys:        2,
			limit:       3,
			grantLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelAll},
			otherLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelAll},
		})
		runLedger(l, m)
		require.ErrorContains(t, l.checkExtendable(ctx, m), "answered leases were lost")
	})
}

func TestLedgerDoesNotBlameTheStore(t *testing.T) {
	ctx := context.Background()
	rng := rand.New(rand.NewSource(1))

	t.Run("a grant at level primary may be lost", func(t *testing.T) {
		m := newFakeManager(10)
		m.loseEvery = 1
		l := newLedger(ledgerConfig{
			keys:        1,
			limit:       10,
			grantLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelPrimary},
		})
		lease, _ := l.acquire(ctx, m, rng)
		require.NotNil(t, lease)
		require.NoError(t, l.checkExtendable(ctx, m))
		done, _ := l.extend(ctx, m, rng, lease)
		require.True(t, done)
		require.NoError(t, l.checkExtendable(ctx, m))
	})

	t.Run("an extend at level primary may be lost", func(t *testing.T) {
		m := newFakeManager(10)
		l := newLedger(ledgerConfig{
			keys:        1,
			limit:       10,
			grantLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelAll},
			otherLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelPrimary},
		})
		lease, _ := l.acquire(ctx, m, rng)
		_, ok := l.extend(ctx, m, rng, lease)
		require.True(t, ok)
		// the new primary has the lease under its first ID only.
		m.mu.Lock()
		clear(m.leases)
		m.mu.Unlock()
		require.NoError(t, l.checkExtendable(ctx, m))
	})

	t.Run("an extend with no answer leaves the lease ID unknown", func(t *testing.T) {
		m := newFakeManager(10)
		l := newLedger(ledgerConfig{
			keys:        1,
			limit:       10,
			grantLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelAll},
			otherLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelAll},
		})
		lease, _ := l.acquire(ctx, m, rng)
		m.failAppliedExtend = true
		done, ok := l.extend(ctx, m, rng, lease)
		require.False(t, done)
		require.False(t, ok)
		require.True(t, lease.unsure)
		m.failAppliedExtend = false

		// the check skips it, and the retry of the workload finds no
		// lease under the old ID.  neither is a loss.
		require.NoError(t, l.checkExtendable(ctx, m))
		done, ok = l.extend(ctx, m, rng, lease)
		require.True(t, done)
		require.True(t, ok)
		require.NoError(t, l.checkExtendable(ctx, m))
		require.NoError(t, l.checkErrors())
	})

	t.Run("an expired lease may be gone", func(t *testing.T) {
		m := newFakeManager(10)
		l := newLedger(ledgerConfig{
			keys:        1,
			limit:       10,
			grantLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelAll},
			duration:    20 * time.Millisecond,
		})
		lease, _ := l.acquire(ctx, m, rng)
		require.NotNil(t, lease)
		time.Sleep(30 * time.Millisecond)
		m.mu.Lock()
		clear(m.leases)
		clear(m.used)
		m.mu.Unlock()
		require.NoError(t, l.checkExtendable(ctx, m))
		require.NoError(t, l.checkLimits(ctx, m))
	})

	t.Run("a released lease is not checked", func(t *testing.T) {
		m := newFakeManager(10)
		l := newLedger(ledgerConfig{
			keys:        1,
			limit:       10,
			grantLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelAll},
		})
		lease, _ := l.acquire(ctx, m, rng)
		done, ok := l.release(ctx, m, rng, lease)
		require.True(t, done)
		require.True(t, ok)
		require.Equal(t, leaseReleased, lease.state)
		require.NoError(t, l.checkExtendable(ctx, m))
		require.NoError(t, l.checkLimits(ctx, m))
		require.Equal(t, "release", l.events[len(l.events)-1].op)
	})
}

func TestLedgerDetectsLimitBreach(t *testing.T) {
	ctx := context.Background()
	rng := rand.New(rand.NewSource(1))
	cfg := ledgerConfig{
		keys:        1,
		limit:       2,
		grantLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelAll},
	}

	t.Run("more live leases than the limit", func(t *testing.T) {
		m := newFakeManager(2)
		m.ignoreLimit = true
		l := newLedger(cfg)
		for range 3 {
			lease, ok := l.acquire(ctx, m, rng)
			require.NotNil(t, lease)
			require.True(t, ok)
		}
		require.ErrorContains(t, l.checkLimits(ctx, m), "had 3 live leases, the limit is 2")
	})

	t.Run("a release makes room", func(t *testing.T) {
		m := newFakeManager(2)
		l := newLedger(cfg)
		first, _ := l.acquire(ctx, m, rng)
		_, _ = l.acquire(ctx, m, rng)
		denied, ok := l.acquire(ctx, m, rng)
		require.Nil(t, denied)
		require.False(t, ok)
		_, _ = l.release(ctx, m, rng, first)
		again, _ := l.acquire(ctx, m, rng)
		require.NotNil(t, again)
		require.NoError(t, l.checkLimits(ctx, m))
		require.Equal(t, []int{2}, l.peak)
	})

	t.Run("the store counts fewer leases than are live", func(t *testing.T) {
		m := newFakeManager(2)
		l := newLedger(cfg)
		_, _ = l.acquire(ctx, m, rng)
		_, _ = l.acquire(ctx, m, rng)
		require.NoError(t, l.checkLimits(ctx, m))
		m.hideUsage = 1
		require.ErrorContains(t, l.checkLimits(ctx, m), "the store counts 1 leases for key 0, the ledger has 2 live")
	})
}

func TestLedgerErrors(t *testing.T) {
	ctx := context.Background()
	rng := rand.New(rand.NewSource(1))

	// a retryable error is not a failure.
	m := newFakeManager(10)
	m.retryEvery = 1
	l := newLedger(ledgerConfig{
		keys:  1,
		limit: 10,
	})
	lease, ok := l.acquire(ctx, m, rng)
	require.Nil(t, lease)
	require.False(t, ok)
	require.NoError(t, l.checkErrors())
	require.Empty(t, l.events)

	// an error that is not retryable is.
	m.retryEvery = 0
	m.fatalEvery = 1
	_, _ = l.acquire(ctx, m, rng)
	require.ErrorContains(t, l.checkErrors(), "acquire: fake manager: broken")

	// the check gives up on a store that never answers.
	m.fatalEvery = 0
	lease, _ = l.acquire(ctx, m, rng)
	require.NotNil(t, lease)
	l.mu.Lock()
	lease.level = constraintapi.CommitLevelAll
	l.mu.Unlock()
	m.retryEvery = 1
	short, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, l.checkExtendable(short, m), context.DeadlineExceeded)
}
