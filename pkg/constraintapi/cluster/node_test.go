package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const (
	testAckTimeout   = 100 * time.Millisecond
	testFenceTimeout = 100 * time.Millisecond
)

// newNodeCluster returns a harness cluster whose nodes are real nodes with
// short timeouts.  it creates the shards, starts every node and waits until
// every shard has an open primary and every replica is in sync.
func newNodeCluster(t testing.TB, nodes, shards, rf int) *testCluster {
	t.Helper()
	return newNodeClusterAck(t, nodes, shards, rf, testAckTimeout)
}

// newNodeClusterAck is newNodeCluster with its own acknowledgement timeout.
// a test that needs a follower to stay in the in-sync set while it is
// silent uses a long one.
func newNodeClusterAck(t testing.TB, nodes, shards, rf int, ack time.Duration) *testCluster {
	t.Helper()
	return newNodeClusterCfg(t, nodes, shards, rf, func(cfg *Config) { cfg.AckTimeout = ack })
}

// testIdempotencyTTL is how long the test nodes keep an acquire or extend
// idempotency record.  a follower that joins a shard under load enters the
// in-sync set this long after its state transfer.
const testIdempotencyTTL = time.Second

// testNodeConfig is the node config of the tests.  the timeouts are short.
func testNodeConfig(name string, tr Transport, co Coordinator) Config {
	return Config{
		Name:              name,
		Transport:         tr,
		Coordinator:       co,
		AckTimeout:        testAckTimeout,
		FenceTimeout:      testFenceTimeout,
		HeartbeatInterval: testHeartbeatInterval,
		DialBackoff:       5 * time.Millisecond,
		ManagerOptions: []memory.Option{
			memory.WithOperationIdempotencyTTL(testIdempotencyTTL),
		},
	}
}

// newFailoverCluster is newNodeCluster with the default idempotency TTL.  a
// test that runs the ledger through a failover uses it.  the client sends a
// request again after a failover, and that can be later than
// testIdempotencyTTL after the first try.  the record has expired then, and
// an extend that was applied answers that the lease is gone, which the
// ledger counts as a lost lease.
func newFailoverCluster(t testing.TB, nodes, shards, rf int) *testCluster {
	t.Helper()
	return newNodeClusterCfg(t, nodes, shards, rf, func(cfg *Config) { cfg.ManagerOptions = nil })
}

// newNodeClusterCfg is newNodeCluster with a change to every node's config.
func newNodeClusterCfg(t testing.TB, nodes, shards, rf int, change func(cfg *Config)) *testCluster {
	t.Helper()
	c := newTestCluster(t, nodes, func(name string, tr Transport, co Coordinator) (testNode, error) {
		cfg := testNodeConfig(name, tr, co)
		change(&cfg)
		return NewNode(cfg)
	})
	require.NoError(t, c.Init(shards, rf))
	require.NoError(t, c.StartAll())
	c.waitStable(t)
	return c
}

// node returns the running node with this name, or nil.
func (c *testCluster) node(name string) *Node {
	c.mu.Lock()
	defer c.mu.Unlock()
	n, _ := c.nodes[name].(*Node)
	return n
}

// stable reports whether every shard has an open primary and every running
// replica of it is in the in-sync set.
func (c *testCluster) stable() bool {
	tbl := c.Table()
	if len(tbl.Shards) == 0 {
		return false
	}
	for i, row := range tbl.Shards {
		p := c.node(row.Primary)
		if p == nil {
			return false
		}
		if _, err := p.Primary(i); err != nil {
			return false
		}
		for _, name := range row.Replicas {
			if c.Running(name) && !slices.Contains(row.InSync, name) {
				return false
			}
		}
	}
	return true
}

func (c *testCluster) waitStable(t testing.TB) {
	t.Helper()
	eventually(t, 15*time.Second, c.stable, "the cluster did not become stable, table %s", tableString(c))
}

// tableDump prints the newest table when a failure message is built.
type tableDump struct {
	c *testCluster
}

func (d tableDump) String() string {
	b, _ := json.Marshal(d.c.Table())
	return string(b) + "\n" + d.c.debugString()
}

func tableString(c *testCluster) tableDump {
	return tableDump{c: c}
}

// eventually polls cond until it is true or the timeout ends.
func eventually(t testing.TB, timeout time.Duration, cond func() bool, format string, args ...any) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf(format, args...)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// testRouter is the test stand-in for the client.  it sends every call to
// the node the table names as primary of the account's shard, in process.
// when that node is not the open primary the error is retryable.
type testRouter struct {
	c *testCluster
}

var _ constraintapi.CapacityManager = (*testRouter)(nil)

func (rt *testRouter) primary(account uuid.UUID) (*memory.Manager, errs.InternalError) {
	tbl := rt.c.Table()
	shard := tbl.ShardOf(account)
	if shard < 0 {
		return nil, errs.Wrap(0, true, "the table has no shards")
	}
	n := rt.c.node(tbl.Shards[shard].Primary)
	if n == nil {
		return nil, errs.Wrap(0, true, "shard %d has no running primary", shard)
	}
	mgr, err := n.Primary(shard)
	if err != nil {
		return nil, errs.Wrap(0, true, "shard %d: %w", shard, err)
	}
	return mgr, nil
}

func (rt *testRouter) Check(ctx context.Context, req *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	mgr, err := rt.primary(req.AccountID)
	if err != nil {
		return nil, nil, err
	}
	return mgr.Check(ctx, req)
}

func (rt *testRouter) Acquire(ctx context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	mgr, err := rt.primary(req.AccountID)
	if err != nil {
		return nil, err
	}
	return mgr.Acquire(ctx, req)
}

func (rt *testRouter) ExtendLease(ctx context.Context, req *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	mgr, err := rt.primary(req.AccountID)
	if err != nil {
		return nil, err
	}
	return mgr.ExtendLease(ctx, req)
}

func (rt *testRouter) Release(ctx context.Context, req *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	mgr, err := rt.primary(req.AccountID)
	if err != nil {
		return nil, err
	}
	return mgr.Release(ctx, req)
}

// accountOn returns an account that hashes to the shard.
func accountOn(t testing.TB, c *testCluster, shard int) uuid.UUID {
	t.Helper()
	tbl := c.Table()
	for range 100000 {
		id := uuid.New()
		if tbl.ShardOf(id) == shard {
			return id
		}
	}
	t.Fatalf("found no account for shard %d", shard)
	return uuid.Nil
}

// testAccount issues single requests for one account with a concurrency
// limit, through any CapacityManager.
type testAccount struct {
	account uuid.UUID
	env     uuid.UUID
	fn      uuid.UUID
	limit   int
	n       int
}

func newTestAccount(t testing.TB, c *testCluster, shard, limit int) *testAccount {
	return &testAccount{
		account: accountOn(t, c, shard),
		env:     uuid.New(),
		fn:      uuid.New(),
		limit:   limit,
	}
}

func (a *testAccount) acquireReq(level constraintapi.CommitLevel) *constraintapi.CapacityAcquireRequest {
	a.n++
	idem := fmt.Sprintf("%s-%d", a.account, a.n)
	return &constraintapi.CapacityAcquireRequest{
		IdempotencyKey: idem + "-acquire",
		AccountID:      a.account,
		EnvID:          a.env,
		FunctionID:     a.fn,
		Configuration: constraintapi.ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: constraintapi.ConcurrencyConfig{
				FunctionConcurrency: a.limit,
			},
		},
		Constraints:          ledgerConstraints(),
		Amount:               1,
		LeaseIdempotencyKeys: []string{idem},
		LeaseRunIDs:          map[string]ulid.ULID{},
		CurrentTime:          time.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Hour,
		Source:               ledgerSource(),
		CommitLevel:          level,
	}
}

// acquire retries until the grant is answered or ctx ends.
func (a *testAccount) acquire(ctx context.Context, m constraintapi.CapacityManager, level constraintapi.CommitLevel) (ulid.ULID, error) {
	req := a.acquireReq(level)
	var id ulid.ULID
	err := retryLater(ctx, func() errs.Error {
		resp, err := m.Acquire(ctx, req)
		if err != nil {
			return err
		}
		if len(resp.Leases) == 0 {
			return errs.Wrap(0, false, "no capacity")
		}
		id = resp.Leases[0].LeaseID
		return nil
	})
	return id, err
}

// extend extends a lease at level all and returns the new ID, or nil when
// the store does not have the lease.
func (a *testAccount) extend(ctx context.Context, m constraintapi.CapacityManager, lease ulid.ULID) (*ulid.ULID, error) {
	a.n++
	req := &constraintapi.CapacityExtendLeaseRequest{
		IdempotencyKey: fmt.Sprintf("%s-extend-%d", a.account, a.n),
		AccountID:      a.account,
		LeaseID:        lease,
		Duration:       30 * time.Second,
		Source:         ledgerSource(),
		CommitLevel:    constraintapi.CommitLevelAll,
	}
	var id *ulid.ULID
	err := retryLater(ctx, func() errs.Error {
		resp, err := m.ExtendLease(ctx, req)
		if err != nil {
			return err
		}
		id = resp.LeaseID
		return nil
	})
	return id, err
}

func (a *testAccount) release(ctx context.Context, m constraintapi.CapacityManager, lease ulid.ULID) error {
	a.n++
	_, err := m.Release(ctx, &constraintapi.CapacityReleaseRequest{
		IdempotencyKey: fmt.Sprintf("%s-release-%d", a.account, a.n),
		AccountID:      a.account,
		LeaseID:        lease,
		Source:         ledgerSource(),
	})
	if err != nil {
		return err
	}
	return nil
}

// followersOf returns the replicas of a shard that are not its primary.
func followersOf(row ShardRow) []string {
	var out []string
	for _, n := range row.Replicas {
		if n != row.Primary {
			out = append(out, n)
		}
	}
	return out
}

// canonState is the state of a manager as sorted strings, in the form two
// replicas of one shard must agree on.  counters at zero are left out
// because a dropped counter reads zero.  set definitions are kept only for
// sets a live lease uses, because a follower learns of a dropped set late.
// Export emits no acquire or extend idempotency record, so they are not
// part of it.
func canonState(t testing.TB, m *memory.Manager) []string {
	t.Helper()
	var recs []memory.Record
	m.Freeze(func() {
		m.Export(func(r memory.Record) { recs = append(recs, r) })
	})
	used := map[uint64]bool{}
	for _, r := range recs {
		if r.Kind == memory.RecGrant {
			used[r.SetID] = true
		}
	}
	var out []string
	for _, r := range recs {
		switch r.Kind {
		case memory.RecSet:
			if !used[r.SetID] {
				continue
			}
			def, err := json.Marshal(r.Set)
			require.NoError(t, err)
			out = append(out, fmt.Sprintf("set %d %s", r.SetID, def))
		case memory.RecGrant:
			out = append(out, fmt.Sprintf("lease %v set %d exp %d req %d granted %d", r.Seqs, r.SetID, r.ExpiresAtMS, r.Requested, r.Granted))
		case memory.RecTAT:
			out = append(out, fmt.Sprintf("tat %x.%x %v exp %d v %d", r.KeyA, r.KeyB, r.TAT, r.ExpiresAtMS, r.Version))
		case memory.RecCell:
			if r.Value != 0 {
				out = append(out, fmt.Sprintf("cell %x.%x %d", r.KeyA, r.KeyB, r.Value))
			}
		case memory.RecIdem:
			out = append(out, fmt.Sprintf("idem %x %d exp %d", r.KeyA, r.Value, r.ExpiresAtMS))
		}
	}
	sort.Strings(out)
	return out
}

// caughtUp returns the primary's last LSN of every shard when every running
// follower has applied it, and nil otherwise.
func (c *testCluster) caughtUp() []uint64 {
	if !c.stable() {
		return nil
	}
	tbl := c.Table()
	lsns := make([]uint64, len(tbl.Shards))
	for i, row := range tbl.Shards {
		p := c.node(row.Primary)
		if p == nil || p.replica(i) == nil {
			return nil
		}
		lsns[i] = p.replica(i).lastLSN.Load()
		for _, name := range row.Replicas {
			n := c.node(name)
			if n == nil || name == row.Primary {
				continue
			}
			r := n.replica(i)
			if r == nil || r.applied.Load() != lsns[i] {
				return nil
			}
		}
	}
	return lsns
}

// waitCaughtUp waits until every running follower has applied the primary's
// last LSN and the log has been quiet for longer than the idle drain
// interval.  a record that an operation appended after the last drain is
// not numbered yet, so one look is not enough.
func (c *testCluster) waitCaughtUp(t testing.TB) {
	t.Helper()
	eventually(t, 10*time.Second, func() bool {
		before := c.caughtUp()
		if before == nil {
			return false
		}
		time.Sleep(3 * maxIdleDrain)
		return slices.Equal(before, c.caughtUp())
	}, "the followers did not catch up, table %s", tableString(c))
}

func TestNodeGrantReachesEveryReplica(t *testing.T) {
	c := newNodeCluster(t, 3, 2, 3)
	rt := &testRouter{c: c}
	ctx := context.Background()

	for shard := range 2 {
		acct := newTestAccount(t, c, shard, 5)
		lease, err := acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
		require.NoError(t, err)

		// the grant was answered at level all, so every replica has it now.
		row := c.Table().Shards[shard]
		want := canonState(t, c.node(row.Primary).Replica(shard))
		require.NotEmpty(t, want)
		for _, f := range followersOf(row) {
			eventually(t, 5*time.Second, func() bool {
				return slices.Equal(want, canonState(t, c.node(f).Replica(shard)))
			}, "follower %s of shard %d does not match the primary", f, shard)
		}

		id, err := acct.extend(ctx, rt, lease)
		require.NoError(t, err)
		require.NotNil(t, id)
	}
}

func TestNodeNotPrimaryError(t *testing.T) {
	c := newNodeCluster(t, 3, 1, 3)
	row := c.Table().Shards[0]
	f := c.node(followersOf(row)[0])

	_, err := f.Primary(0)
	require.ErrorIs(t, err, ErrNotPrimary)
	var np *NotPrimaryError
	require.True(t, errors.As(err, &np))
	require.Equal(t, row.Primary, np.Primary)
	require.Equal(t, row.Primary, np.Addr)
	require.NotNil(t, f.Replica(0))

	_, err = c.node(row.Primary).Primary(0)
	require.NoError(t, err)
}

func TestPrimaryKillLosesNoAnsweredGrant(t *testing.T) {
	c := newFailoverCluster(t, 3, 2, 3)
	rt := newTestClient(t, c)

	l := newLedger(ledgerConfig{
		seed:    1,
		workers: 8,
		keys:    8,
		limit:   5,
	})
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		l.run(ctx, rt)
	}()

	time.Sleep(300 * time.Millisecond)
	victim := c.Table().Shards[0].Primary
	c.Kill(victim)
	eventually(t, 10*time.Second, func() bool {
		p := c.Table().Shards[0].Primary
		return p != victim && c.stable()
	}, "no new primary after the kill, table %s", tableString(c))
	time.Sleep(300 * time.Millisecond)
	cancel()
	wg.Wait()

	c.waitStable(t)
	check, done := context.WithTimeout(context.Background(), 20*time.Second)
	defer done()
	require.NoError(t, l.checkErrors())
	require.NoError(t, l.checkExtendable(check, rt))
	require.Greater(t, len(l.events), 100)
}

func TestCommitLevelsWithStalledFollower(t *testing.T) {
	c := newNodeCluster(t, 3, 1, 3)
	rt := &testRouter{c: c}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	row := c.Table().Shards[0]
	stalled := followersOf(row)[1]
	acct := newTestAccount(t, c, 0, 100)
	first, err := acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
	require.NoError(t, err)

	c.net.Stall(row.Primary, stalled)

	// level two needs the primary and one follower.  the other follower
	// still answers.
	start := time.Now()
	_, err = acct.acquire(ctx, rt, constraintapi.CommitLevelTwo)
	require.NoError(t, err)
	require.Less(t, time.Since(start), testAckTimeout/2)

	// level all waits for the stalled follower until the primary has it
	// removed from the in-sync set.
	start = time.Now()
	req := acct.acquireReq(constraintapi.CommitLevelAll)
	answered := make(chan error, 1)
	go func() {
		_, err := rt.Acquire(ctx, req)
		if err != nil {
			answered <- err
			return
		}
		answered <- nil
	}()
	select {
	case err := <-answered:
		t.Fatalf("a grant at level all was answered with a follower stalled: %v", err)
	case <-time.After(testAckTimeout / 2):
	}

	// a release does not wait for anyone.
	require.NoError(t, acct.release(ctx, rt, first))
	select {
	case <-answered:
		t.Fatal("the grant at level all was answered before the release returned")
	default:
	}

	select {
	case err := <-answered:
		require.NoError(t, err)
	case <-time.After(testAckTimeout + 2*time.Second):
		t.Fatal("the grant at level all was not answered after the acknowledgement timeout")
	}
	require.GreaterOrEqual(t, time.Since(start), testAckTimeout/2)
	require.NotContains(t, c.Table().Shards[0].InSync, stalled)

	c.net.Unstall(row.Primary, stalled)
	c.waitStable(t)
	c.waitCaughtUp(t)
	require.Equal(t, canonState(t, c.node(row.Primary).Replica(0)), canonState(t, c.node(stalled).Replica(0)))
}

func TestLevelTwoSurvivesElectingTheOtherFollower(t *testing.T) {
	c := newNodeCluster(t, 3, 1, 3)
	rt := &testRouter{c: c}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	row := c.Table().Shards[0]
	has, lacks := followersOf(row)[0], followersOf(row)[1]
	acct := newTestAccount(t, c, 0, 100)

	// the grant is answered with one follower's acknowledgement.  the other
	// follower never gets it.
	c.net.Stall(row.Primary, lacks)
	lease, err := acct.acquire(ctx, rt, constraintapi.CommitLevelTwo)
	require.NoError(t, err)

	// the primary dies and the table elects the follower that lacks the
	// grant.  it must pull the grant from the other follower before it
	// opens.
	c.co.StopController()
	c.Kill(row.Primary)
	row = c.Table().Shards[0]
	require.Contains(t, row.InSync, lacks)
	require.NoError(t, c.co.Node("harness").Propose(ctx, ElectPrimary{
		Shard: 0,
		Epoch: row.Epoch,
		Node:  lacks,
	}))
	eventually(t, 10*time.Second, func() bool {
		_, err := c.node(lacks).Primary(0)
		return err == nil
	}, "the elected follower did not open")

	id, err := acct.extend(ctx, rt, lease)
	require.NoError(t, err)
	require.NotNil(t, id, "the grant answered at level two is gone")
	require.Contains(t, c.Table().Shards[0].InSync, has)

	c.co.StartController()
	require.NoError(t, c.healAll())
	c.waitStable(t)
}

func TestReplacedPrimaryCannotGrant(t *testing.T) {
	c := newNodeCluster(t, 3, 1, 3)
	rt := &testRouter{c: c}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	row := c.Table().Shards[0]
	old := row.Primary
	next, other := followersOf(row)[0], followersOf(row)[1]
	acct := newTestAccount(t, c, 0, 100)
	_, err := acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
	require.NoError(t, err)

	// the old primary cannot see the table and cannot reach the node that
	// replaces it.  it can still reach the other follower.
	c.co.StopController()
	c.co.Cut(old)
	c.net.Partition(old, next)
	require.NoError(t, c.co.Node("harness").Propose(ctx, ElectPrimary{
		Shard: 0,
		Epoch: c.Table().Shards[0].Epoch,
		Node:  next,
	}))
	eventually(t, 10*time.Second, func() bool {
		_, err := c.node(next).Primary(0)
		return err == nil
	}, "the new primary did not open")

	// the new primary answers at level all.
	_, err = acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
	require.NoError(t, err)
	require.Contains(t, c.Table().Shards[0].InSync, other)

	// the old primary still believes it has the seat.  it answers nothing
	// that needs a follower.
	mgr, err := c.node(old).Primary(0)
	require.NoError(t, err, "the old primary saw the table")
	for _, level := range []constraintapi.CommitLevel{constraintapi.CommitLevelTwo, constraintapi.CommitLevelAll} {
		short, done := context.WithTimeout(ctx, 3*testAckTimeout)
		resp, err := mgr.Acquire(short, acct.acquireReq(level))
		done()
		require.Error(t, err, "the replaced primary answered a grant at level %d: %+v", level, resp)
		require.True(t, err.Retryable())
	}

	c.co.StartController()
	require.NoError(t, c.healAll())
	c.waitStable(t)
	_, err = c.node(old).Primary(0)
	require.ErrorIs(t, err, ErrNotPrimary)
}

func TestFollowerLossDoesNotStallPastTheTimeout(t *testing.T) {
	c := newNodeCluster(t, 3, 1, 3)
	rt := &testRouter{c: c}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	row := c.Table().Shards[0]
	acct := newTestAccount(t, c, 0, 1000)
	_, err := acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
	require.NoError(t, err)

	victim := followersOf(row)[0]
	c.Kill(victim)
	var slowest time.Duration
	for range 20 {
		start := time.Now()
		_, err := acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
		require.NoError(t, err)
		slowest = max(slowest, time.Since(start))
	}
	// the stream broke, so the primary proposed the removal at once and did
	// not wait for the acknowledgement timeout.
	require.Less(t, slowest, testAckTimeout+500*time.Millisecond)
	require.NotContains(t, c.Table().Shards[0].InSync, victim)
}

func TestFastRestartLeavesTheInSyncSet(t *testing.T) {
	c := newNodeCluster(t, 3, 1, 3)
	rt := &testRouter{c: c}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	acct := newTestAccount(t, c, 0, 1000)
	var leases []ulid.ULID
	for range 10 {
		id, err := acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
		require.NoError(t, err)
		leases = append(leases, id)
	}

	restart := func(name string, peers []string) {
		t.Helper()
		boot := c.Table().Nodes[name].BootID
		// the node cannot take the state after it comes back, so it cannot
		// catch up.
		for _, p := range peers {
			c.net.Stall(name, p)
		}
		require.NoError(t, c.Restart(name))
		eventually(t, 10*time.Second, func() bool {
			return c.Table().Nodes[name].BootID != boot
		}, "%s did not register again", name)

		outside := func() {
			row := c.Table().Shards[0]
			require.NotContains(t, row.InSync, name)
			require.NotEqual(t, name, row.Primary)
		}
		outside()
		time.Sleep(2 * testAckTimeout)
		outside()

		// another node has the seat and answers at level all.
		id, err := acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
		require.NoError(t, err)
		leases = append(leases, id)
		outside()

		for _, p := range peers {
			c.net.Unstall(name, p)
		}
		c.waitStable(t)
		require.Contains(t, c.Table().Shards[0].InSync, name)
	}

	row := c.Table().Shards[0]
	follower := followersOf(row)[0]
	restart(follower, []string{row.Primary, followersOf(row)[1]})

	row = c.Table().Shards[0]
	restart(row.Primary, followersOf(row))

	for i, lease := range leases {
		id, err := acct.extend(ctx, rt, lease)
		require.NoError(t, err)
		require.NotNil(t, id, "lease %d is gone", i)
	}
}

func TestFollowersConvergeOnThePrimary(t *testing.T) {
	c := newNodeCluster(t, 3, 4, 3)
	rt := &testRouter{c: c}

	l := newLedger(ledgerConfig{
		seed:    7,
		workers: 8,
		keys:    16,
		limit:   5,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()
	l.run(ctx, rt)
	require.NoError(t, l.checkErrors())
	require.Greater(t, len(l.events), 100)

	c.waitCaughtUp(t)
	for shard, row := range c.Table().Shards {
		want := canonState(t, c.node(row.Primary).Replica(shard))
		for _, f := range followersOf(row) {
			require.Equal(t, want, canonState(t, c.node(f).Replica(shard)), "shard %d follower %s", shard, f)
		}
	}
}

func TestDrainUnderLoadLosesNothing(t *testing.T) {
	c := newNodeCluster(t, 3, 2, 3)
	rt := &testRouter{c: c}

	l := newLedger(ledgerConfig{
		seed:    3,
		workers: 8,
		keys:    8,
		limit:   5,
	})
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		l.run(ctx, rt)
	}()

	// probe measures how long one grant at level all takes, retries
	// included, while the node drains.
	probe := newTestAccount(t, c, 0, 1<<20)
	var slowest time.Duration
	probeDone := make(chan struct{})
	go func() {
		defer close(probeDone)
		for ctx.Err() == nil {
			start := time.Now()
			if _, err := probe.acquire(ctx, rt, constraintapi.CommitLevelAll); err == nil {
				slowest = max(slowest, time.Since(start))
			}
		}
	}()

	time.Sleep(200 * time.Millisecond)
	victim := c.Table().Shards[0].Primary
	start := time.Now()
	require.NoError(t, c.Drain(victim))
	t.Logf("drain took %s", time.Since(start))
	for _, row := range c.Table().Shards {
		require.NotEqual(t, victim, row.Primary)
		require.NotContains(t, row.InSync, victim)
	}
	time.Sleep(200 * time.Millisecond)
	cancel()
	wg.Wait()
	<-probeDone

	// every step of a planned move is far below a timeout.
	t.Logf("slowest grant during the drain took %s", slowest)
	require.Less(t, slowest, testAckTimeout)

	c.waitStable(t)
	check, done := context.WithTimeout(context.Background(), 20*time.Second)
	defer done()
	require.NoError(t, l.checkErrors())
	require.NoError(t, l.checkExtendable(check, rt))
}

func TestFaultScheduleKeepsAnsweredGrants(t *testing.T) {
	seeds := []int64{1}
	faults, span := 6, 2*time.Second
	if os.Getenv("CLUSTER_FAULT_LONG") != "" {
		seeds = []int64{1, 2, 3, 4, 5, 6, 7, 8}
		faults, span = 30, 15*time.Second
	}
	for _, seed := range seeds {
		t.Run(fmt.Sprintf("seed-%d", seed), func(t *testing.T) {
			c := newFailoverCluster(t, 3, 4, 3)
			rt := newTestClient(t, c)

			// every grant waits for every in-sync replica, so the limit
			// check applies and a kill of one node loses none of them.
			l := newLedger(ledgerConfig{
				seed:        seed,
				workers:     8,
				keys:        16,
				limit:       5,
				grantLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelDefault, constraintapi.CommitLevelAll},
				otherLevels: []constraintapi.CommitLevel{constraintapi.CommitLevelDefault, constraintapi.CommitLevelAll, constraintapi.CommitLevelPrimary},
				duration:    time.Minute,
			})
			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				l.run(ctx, rt)
			}()

			s := newFaultSchedule(t, c, faultConfig{
				seed:   seed,
				faults: faults,
				span:   span,
			})
			s.run(context.Background())
			require.NoError(t, c.healAll())
			c.waitStable(t)
			time.Sleep(100 * time.Millisecond)
			cancel()
			wg.Wait()

			c.waitStable(t)
			check, done := context.WithTimeout(context.Background(), 30*time.Second)
			defer done()
			require.NoError(t, l.checkErrors())
			require.NoError(t, l.checkExtendable(check, rt))
			require.NoError(t, l.checkLimits(check, rt))
			t.Logf("seed %d answered %d operations", seed, len(l.events))
		})
	}
}

// deposeFirstPrimary elects another node while the first primary cannot see
// the table, and returns the old primary, the new one and the third node.
// the old primary answers the Fence, so it stops, and it stays in the
// in-sync set because it cannot propose its own removal.
func deposeFirstPrimary(t *testing.T, c *testCluster) (old, next, other string) {
	t.Helper()
	row := c.Table().Shards[0]
	old, next, other = row.Primary, followersOf(row)[0], followersOf(row)[1]
	c.co.StopController()
	c.co.Cut(old)
	require.NoError(t, c.co.Node("harness").Propose(context.Background(), ElectPrimary{
		Shard: 0,
		Epoch: row.Epoch,
		Node:  next,
	}))
	eventually(t, 10*time.Second, func() bool {
		_, err := c.node(next).Primary(0)
		return err == nil
	}, "the new primary did not open, table %s", tableString(c))
	require.Contains(t, c.Table().Shards[0].InSync, old)
	_, err := c.node(old).Primary(0)
	require.ErrorIs(t, err, ErrNotPrimary, "the old primary did not stop on the fence")
	return old, next, other
}

func TestDeposedPrimaryOpensAgain(t *testing.T) {
	c := newNodeClusterAck(t, 3, 1, 3, 5*time.Second)
	rt := &testRouter{c: c}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	acct := newTestAccount(t, c, 0, 100)
	lease, err := acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
	require.NoError(t, err)

	// the old primary lost its seat and kept its state.  the table gives
	// the seat back before anything else was written.
	old, _, _ := deposeFirstPrimary(t, c)
	require.NoError(t, c.co.Node("harness").Propose(ctx, ElectPrimary{
		Shard: 0,
		Epoch: c.Table().Shards[0].Epoch,
		Node:  old,
	}))
	c.co.Restore(old)
	eventually(t, 10*time.Second, func() bool {
		_, err := c.node(old).Primary(0)
		return err == nil
	}, "the old primary did not open again, table %s", tableString(c))

	id, err := acct.extend(ctx, rt, lease)
	require.NoError(t, err)
	require.NotNil(t, id, "the lease from the first term is gone")
	_, err = acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
	require.NoError(t, err)

	c.co.StartController()
	c.waitStable(t)
	c.waitCaughtUp(t)
	row := c.Table().Shards[0]
	want := canonState(t, c.node(row.Primary).Replica(0))
	for _, f := range followersOf(row) {
		require.Equal(t, want, canonState(t, c.node(f).Replica(0)), "follower %s", f)
	}
}

func TestDeposedPrimaryGivesTheSeatToAReplicaThatIsAhead(t *testing.T) {
	c := newNodeClusterAck(t, 3, 1, 3, 5*time.Second)
	rt := &testRouter{c: c}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	acct := newTestAccount(t, c, 0, 100)
	_, err := acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
	require.NoError(t, err)

	// the new primary answers a grant at level two.  the old primary, still
	// in the in-sync set, never gets it.
	old, _, _ := deposeFirstPrimary(t, c)
	lease, err := acct.acquire(ctx, rt, constraintapi.CommitLevelTwo)
	require.NoError(t, err)

	// the table gives the seat back to the old primary.  its manager was a
	// primary and cannot apply records, so it must pass the seat on to a
	// replica that has the grant.
	require.NoError(t, c.co.Node("harness").Propose(ctx, ElectPrimary{
		Shard: 0,
		Epoch: c.Table().Shards[0].Epoch,
		Node:  old,
	}))
	c.co.Restore(old)
	eventually(t, 10*time.Second, func() bool {
		row := c.Table().Shards[0]
		if row.Primary == old || row.Primary == "" {
			return false
		}
		_, err := c.node(row.Primary).Primary(0)
		return err == nil
	}, "the seat did not move to a replica that is ahead, table %s", tableString(c))

	id, err := acct.extend(ctx, rt, lease)
	require.NoError(t, err)
	require.NotNil(t, id, "the grant answered at level two is gone")

	c.co.StartController()
	c.waitStable(t)
}

func TestShardStartsEmptyWhenItsLastCopyRestarts(t *testing.T) {
	c := newNodeCluster(t, 3, 1, 3)
	rt := &testRouter{c: c}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	row := c.Table().Shards[0]
	primary := row.Primary
	acct := newTestAccount(t, c, 0, 100)
	_, err := acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
	require.NoError(t, err)

	// both followers fall out of the in-sync set.  the primary is the only
	// copy with every answered write.
	for _, f := range followersOf(row) {
		c.net.Stall(primary, f)
	}
	lease, err := acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
	require.NoError(t, err)
	require.Equal(t, []string{primary}, c.Table().Shards[0].InSync)

	// the primary restarts with no state.  the table gives the shard a new
	// nonce, and every replica starts empty.
	require.NoError(t, c.Restart(primary))
	eventually(t, 10*time.Second, func() bool {
		return c.Table().Shards[0].Nonce != row.Nonce
	}, "the shard got no new nonce, table %s", tableString(c))
	for _, f := range followersOf(row) {
		c.net.Unstall(primary, f)
	}
	c.waitStable(t)

	// a lease ID from before the restart reads as foreign.
	id, err := acct.extend(ctx, rt, lease)
	require.NoError(t, err)
	require.Nil(t, id)
	_, err = acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
	require.NoError(t, err)

	c.waitCaughtUp(t)
	now := c.Table().Shards[0]
	want := canonState(t, c.node(now.Primary).Replica(0))
	require.NotEmpty(t, want)
	for _, f := range followersOf(now) {
		require.Equal(t, want, canonState(t, c.node(f).Replica(0)), "follower %s", f)
	}
}
