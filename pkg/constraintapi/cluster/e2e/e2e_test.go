// Package e2e_test runs the Raft coordinator, the TCP transport, replication
// and the client together.  no other test does.
package e2e_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/raft"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/cluster"
	"github.com/inngest/inngest/pkg/constraintapi/cluster/raftcoord"
	"github.com/inngest/inngest/pkg/constraintapi/cluster/tcp"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	e2eShards = 4
	e2eRF     = 3
	// e2eAccountsPerShard is how many accounts the test uses on every shard.
	e2eAccountsPerShard = 3

	e2eHeartbeatInterval = 50 * time.Millisecond
	// e2eHeartbeatTimeout is the age of a node's last report at which the
	// controller elects another primary.
	e2eHeartbeatTimeout = time.Second
	e2eAckTimeout       = time.Second
	// e2eWait bounds every wait of the test.  a failover takes one Raft
	// election, one heartbeat timeout and one fence timeout.
	e2eWait = 60 * time.Second
)

// process is what one store process runs.  the coordinator and the node
// share the listener through one Mux.
type process struct {
	name     string
	addr     string
	mux      *cluster.Mux
	listener io.Closer
	raftNet  *raft.InmemTransport
	coord    *raftcoord.Coordinator
	node     *cluster.Node
	stopped  bool
}

// stop ends the process with no drain.  the node goes first, because a node
// with a closed coordinator still reaches the Raft leader through its
// transport.
func (p *process) stop() {
	if p.stopped {
		return
	}
	p.stopped = true
	if p.node != nil {
		_ = p.node.Close()
	}
	if p.coord != nil {
		_ = p.coord.Close()
	}
	_ = p.listener.Close()
}

// grant is one lease the client was given.
type grant struct {
	shard   int
	account *account
	lease   ulid.ULID
}

// account is one account with one function and a concurrency limit.
type account struct {
	id    uuid.UUID
	env   uuid.UUID
	fn    uuid.UUID
	shard int
	n     int
}

func source() constraintapi.LeaseSource {
	return constraintapi.LeaseSource{
		Service:           constraintapi.ServiceExecutor,
		Location:          constraintapi.CallerLocationItemLease,
		RunProcessingMode: constraintapi.RunProcessingModeBackground,
	}
}

// acquire asks for one lease at level all through the client.
func (a *account) acquire(ctx context.Context, cl *cluster.Client) (ulid.ULID, error) {
	a.n++
	key := fmt.Sprintf("%s-%d", a.id, a.n)
	resp, err := cl.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
		IdempotencyKey: key + "-acquire",
		AccountID:      a.id,
		EnvID:          a.env,
		FunctionID:     a.fn,
		Configuration: constraintapi.ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: constraintapi.ConcurrencyConfig{
				FunctionConcurrency: 100,
			},
		},
		Constraints: []constraintapi.ConstraintItem{
			{
				Kind: constraintapi.ConstraintKindConcurrency,
				Concurrency: &constraintapi.ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		},
		Amount:               1,
		LeaseIdempotencyKeys: []string{key},
		LeaseRunIDs:          map[string]ulid.ULID{},
		CurrentTime:          time.Now(),
		Duration:             constraintapi.MaximumDuration,
		MaximumLifetime:      time.Hour,
		Source:               source(),
		CommitLevel:          constraintapi.CommitLevelAll,
	})
	if err != nil {
		return ulid.ULID{}, err
	}
	if len(resp.Leases) != 1 {
		return ulid.ULID{}, fmt.Errorf("the acquire was denied, got %d leases", len(resp.Leases))
	}
	return resp.Leases[0].LeaseID, nil
}

// extend extends a lease at level all through the client.  the error says so
// when the store no longer has the lease.
func (a *account) extend(ctx context.Context, cl *cluster.Client, lease ulid.ULID) (ulid.ULID, error) {
	a.n++
	resp, err := cl.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
		IdempotencyKey: fmt.Sprintf("%s-%d-extend", a.id, a.n),
		AccountID:      a.id,
		LeaseID:        lease,
		Duration:       constraintapi.MaximumDuration,
		Source:         source(),
		CommitLevel:    constraintapi.CommitLevelAll,
	})
	if err != nil {
		return ulid.ULID{}, err
	}
	if resp.LeaseID == nil {
		return ulid.ULID{}, errors.New("the store no longer has the lease")
	}
	return *resp.LeaseID, nil
}

// e2eCluster is the three processes of the test.
type e2eCluster struct {
	t     *testing.T
	tr    cluster.Transport
	procs []*process
}

func (c *e2eCluster) live() []*process {
	var out []*process
	for _, p := range c.procs {
		if !p.stopped {
			out = append(out, p)
		}
	}
	return out
}

func (c *e2eCluster) proc(name string) *process {
	for _, p := range c.procs {
		if p.name == name {
			return p
		}
	}
	return nil
}

// table returns the table of the first live process.
func (c *e2eCluster) table() *cluster.Table {
	tbl, _ := c.live()[0].coord.Table()
	return tbl
}

func (c *e2eCluster) tableString() string {
	out, _ := json.Marshal(c.table())
	return string(out)
}

// unready returns why the cluster is not ready, or "" when it is.  it is
// ready when every live process has a table in which every shard has a live
// primary that is open and wantInSync in-sync members.
func (c *e2eCluster) unready(wantInSync int) string {
	for _, p := range c.live() {
		tbl, _ := p.coord.Table()
		if len(tbl.Shards) != e2eShards {
			return fmt.Sprintf("the table of %s has %d shards", p.name, len(tbl.Shards))
		}
		for shard, row := range tbl.Shards {
			primary := c.proc(row.Primary)
			if primary == nil || primary.stopped {
				return fmt.Sprintf("shard %d has the primary %q in the table of %s, and that node does not run", shard, row.Primary, p.name)
			}
			if _, err := primary.node.Primary(shard); err != nil {
				return fmt.Sprintf("shard %d is not open on %s, %v", shard, row.Primary, err)
			}
			if len(row.InSync) != wantInSync {
				return fmt.Sprintf("shard %d has the in-sync set %v in the table of %s, want %d members", shard, row.InSync, p.name, wantInSync)
			}
		}
	}
	return ""
}

// waitReady fails the test when the cluster is not ready in time.
func (c *e2eCluster) waitReady(step string, wantInSync int) {
	c.t.Helper()
	deadline := time.Now().Add(e2eWait)
	for {
		why := c.unready(wantInSync)
		if why == "" {
			return
		}
		if time.Now().After(deadline) {
			c.t.Fatalf("step %q: %s after %s, table %s", step, why, e2eWait, c.tableString())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// startCluster starts three processes.  each one has a TCP listener, a Raft
// coordinator and a node.  Raft runs on its in-memory transport with
// in-memory stores.  everything else uses TCP on loopback.
func startCluster(t *testing.T) *e2eCluster {
	t.Helper()
	tr := tcp.New()
	c := &e2eCluster{
		t:  t,
		tr: tr,
	}
	t.Cleanup(func() {
		for _, p := range c.procs {
			p.stop()
		}
	})

	// every listener starts first.  the Raft peer list and the node need the
	// real ports.
	var peers []raftcoord.Peer
	for i := range 3 {
		name := fmt.Sprintf("n%d", i)
		mux := cluster.NewMux()
		ln, err := tr.Listen("127.0.0.1:0", mux.Accept)
		require.NoError(t, err, "step \"listen\": node %s", name)
		raftAddr := raft.ServerAddress("raft-" + name)
		_, raftNet := raft.NewInmemTransport(raftAddr)
		p := &process{
			name:     name,
			addr:     ln.(*tcp.Listener).Addr().String(),
			mux:      mux,
			listener: ln,
			raftNet:  raftNet,
		}
		c.procs = append(c.procs, p)
		peers = append(peers, raftcoord.Peer{
			ID:       name,
			RaftAddr: string(raftAddr),
			Addr:     p.addr,
			Voter:    true,
		})
	}
	for _, a := range c.procs {
		for _, b := range c.procs {
			if a != b {
				a.raftNet.Connect(b.raftNet.LocalAddr(), b.raftNet)
			}
		}
	}
	for _, p := range c.procs {
		var err error
		p.coord, err = raftcoord.New(raftcoord.Config{
			ID:                p.name,
			Addr:              p.addr,
			Peers:             peers,
			Transport:         tr,
			Mux:               p.mux,
			HeartbeatInterval: e2eHeartbeatInterval,
			HeartbeatTimeout:  e2eHeartbeatTimeout,
			RaftTransport:     p.raftNet,
			InMemory:          true,
		})
		require.NoError(t, err, "step \"start coordinator\": node %s", p.name)
	}
	for _, p := range c.procs {
		c.startNode(p)
	}
	return c
}

func (c *e2eCluster) startNode(p *process) {
	c.t.Helper()
	var err error
	p.node, err = cluster.NewNode(cluster.Config{
		Name:              p.name,
		Addr:              p.addr,
		Transport:         c.tr,
		Coordinator:       p.coord,
		Mux:               p.mux,
		AckTimeout:        e2eAckTimeout,
		HeartbeatInterval: e2eHeartbeatInterval,
		DialBackoff:       10 * time.Millisecond,
	})
	require.NoError(c.t, err, "step \"start node\": node %s", p.name)
}

// join starts one more process with an empty Raft store.  its coordinator
// asks the processes that run to add it, and its node registers.
func (c *e2eCluster) join(name string) *process {
	c.t.Helper()
	mux := cluster.NewMux()
	ln, err := c.tr.Listen("127.0.0.1:0", mux.Accept)
	require.NoError(c.t, err, "step \"listen\": node %s", name)
	raftAddr := raft.ServerAddress("raft-" + name)
	_, raftNet := raft.NewInmemTransport(raftAddr)
	p := &process{
		name:     name,
		addr:     ln.(*tcp.Listener).Addr().String(),
		mux:      mux,
		listener: ln,
		raftNet:  raftNet,
	}
	var seeds []string
	for _, other := range c.live() {
		seeds = append(seeds, other.addr)
		other.raftNet.Connect(raftAddr, raftNet)
		raftNet.Connect(other.raftNet.LocalAddr(), other.raftNet)
	}
	c.procs = append(c.procs, p)
	p.coord, err = raftcoord.New(raftcoord.Config{
		ID:   name,
		Addr: p.addr,
		Peers: []raftcoord.Peer{{
			ID:       name,
			RaftAddr: string(raftAddr),
			Addr:     p.addr,
		}},
		Join:              seeds,
		Transport:         c.tr,
		Mux:               mux,
		HeartbeatInterval: e2eHeartbeatInterval,
		HeartbeatTimeout:  e2eHeartbeatTimeout,
		RaftTransport:     raftNet,
		InMemory:          true,
	})
	require.NoError(c.t, err, "step \"start coordinator\": node %s", name)
	c.startNode(p)
	return p
}

// kill stops a process with no drain and cuts its Raft transport off from
// the others, as when the process is gone.
func (c *e2eCluster) kill(p *process) {
	p.stop()
	for _, other := range c.live() {
		other.raftNet.Disconnect(p.raftNet.LocalAddr())
	}
}

// open creates the shards, connects a client and takes two leases for every
// account.  the accounts cover every shard.
func (c *e2eCluster) open(ctx context.Context) (*cluster.Client, []*account, []grant) {
	t := c.t
	t.Helper()

	// every node registers through its own coordinator once Raft has a
	// leader.
	step := "register nodes"
	require.Eventually(t, func() bool {
		for _, p := range c.procs {
			if tbl, _ := p.coord.Table(); len(tbl.Nodes) != len(c.procs) {
				return false
			}
		}
		return true
	}, e2eWait, 5*time.Millisecond, "step %q: not every table lists the three nodes, table %s", step, c.tableString())

	step = "init"
	var names []string
	for _, p := range c.procs {
		names = append(names, p.name)
	}
	var err error
	require.Eventually(t, func() bool {
		pctx, done := context.WithTimeout(ctx, 2*time.Second)
		defer done()
		err = c.procs[1].coord.Propose(pctx, cluster.Init{
			Shards: e2eShards,
			RF:     e2eRF,
			Nodes:  names,
		})
		return !errors.Is(err, cluster.ErrUnavailable)
	}, e2eWait, 10*time.Millisecond, "step %q: the coordinator of %s stayed unavailable", step, c.procs[1].name)
	require.NoError(t, err, "step %q: the proposal through %s failed", step, c.procs[1].name)
	c.waitReady(step, e2eRF)

	var addrs []string
	for _, p := range c.procs {
		addrs = append(addrs, p.addr)
	}
	cl, err := cluster.NewClient(cluster.ClientConfig{
		Transport: tcp.New(),
		Seeds:     addrs,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cl.Close() })

	// the accounts cover every shard.
	step = "grant before the shutdown"
	tbl := c.table()
	perShard := make([]int, e2eShards)
	var accounts []*account
	for len(accounts) < e2eShards*e2eAccountsPerShard {
		id := uuid.New()
		shard := tbl.ShardOf(id)
		if perShard[shard] == e2eAccountsPerShard {
			continue
		}
		perShard[shard]++
		accounts = append(accounts, &account{
			id:    id,
			env:   uuid.New(),
			fn:    uuid.New(),
			shard: shard,
		})
	}
	var grants []grant
	for _, a := range accounts {
		for range 2 {
			lease, err := a.acquire(ctx, cl)
			require.NoError(t, err, "step %q: shard %d, account %s, table %s", step, a.shard, a.id, c.tableString())
			grants = append(grants, grant{
				shard:   a.shard,
				account: a,
				lease:   lease,
			})
		}
	}

	return cl, accounts, grants
}

func TestFailoverThroughRaftTCPAndClient(t *testing.T) {
	c := startCluster(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*e2eWait)
	defer cancel()
	cl, accounts, grants := c.open(ctx)

	// the node that leads the most shards stops with every in-sync set
	// full, so every shard it leads has two other copies.
	step := "shutdown"
	c.waitReady(step, e2eRF)
	tbl := c.table()
	leads := map[string][]int{}
	for shard, row := range tbl.Shards {
		leads[row.Primary] = append(leads[row.Primary], shard)
	}
	var victim *process
	for _, p := range c.procs {
		if victim == nil || len(leads[p.name]) > len(leads[victim.name]) {
			victim = p
		}
	}
	require.NotEmpty(t, leads[victim.name], "step %q: no node leads a shard, table %s", step, c.tableString())
	t.Logf("stopping %s, the primary of shards %v", victim.name, leads[victim.name])
	stopped := time.Now()
	c.kill(victim)

	// every shard gets an open primary on a node that runs.  the node that
	// stopped leaves every in-sync set.
	step = "wait for primaries after the shutdown"
	c.waitReady(step, e2eRF-1)
	t.Logf("every shard had an open primary %s after the shutdown", time.Since(stopped).Round(time.Millisecond))
	for shard, row := range c.table().Shards {
		require.NotEqual(t, victim.name, row.Primary, "step %q: shard %d", step, shard)
		require.NotContains(t, row.InSync, victim.name, "step %q: shard %d", step, shard)
	}

	step = "extend after the shutdown"
	for _, g := range grants {
		led := "a node that still runs"
		if tbl.Shards[g.shard].Primary == victim.name {
			led = "the node that stopped"
		}
		_, err := g.account.extend(ctx, cl, g.lease)
		require.NoError(t, err, "step %q: shard %d, led before by %s, account %s, lease %s, table %s", step, g.shard, led, g.account.id, g.lease, c.tableString())
	}

	step = "grant after the shutdown"
	for _, a := range accounts {
		_, err := a.acquire(ctx, cl)
		require.NoError(t, err, "step %q: shard %d, account %s, table %s", step, a.shard, a.id, c.tableString())
	}
}

// TestReplaceNodeThroughRaftTCPAndClient stops one of three processes and
// gives its seats and its Raft vote to a fourth one.  then it stops a second
// one of the first three.  the two that are left have a Raft leader only
// with the vote of the new process, and every lease from before the first
// stop is still there.
func TestReplaceNodeThroughRaftTCPAndClient(t *testing.T) {
	c := startCluster(t)
	ctx, cancel := context.WithTimeout(context.Background(), 4*e2eWait)
	defer cancel()
	cl, accounts, grants := c.open(ctx)

	step := "first shutdown"
	c.waitReady(step, e2eRF)
	dead := c.procs[2]
	c.kill(dead)
	c.waitReady(step, e2eRF-1)

	step = "join"
	joined := c.join("n3")
	require.Eventually(t, func() bool {
		_, ok := c.table().Nodes[joined.name]
		return ok
	}, e2eWait, 5*time.Millisecond, "step %q: the new node did not register, table %s", step, c.tableString())

	step = "replace"
	var err error
	require.Eventually(t, func() bool {
		rctx, done := context.WithTimeout(ctx, 2*time.Second)
		defer done()
		err = c.procs[0].coord.ReplaceNode(rctx, dead.name, joined.name)
		return err == nil
	}, e2eWait, 10*time.Millisecond, "step %q: the table did not take the command, table %s", step, c.tableString())
	started := time.Now()
	c.waitReady(step, e2eRF)
	t.Logf("the new node was in every in-sync set %s after the command", time.Since(started).Round(time.Millisecond))
	for shard, row := range c.table().Shards {
		require.Contains(t, row.InSync, joined.name, "step %q: shard %d", step, shard)
		require.NotContains(t, row.Replicas, dead.name, "step %q: shard %d", step, shard)
	}
	want := []raftcoord.Member{
		{
			ID:    c.procs[0].name,
			Voter: true,
		},
		{
			ID:    c.procs[1].name,
			Voter: true,
		},
		{
			ID:    joined.name,
			Voter: true,
		},
	}
	for _, p := range c.live() {
		require.Eventually(t, func() bool {
			got, err := p.coord.Members()
			return err == nil && assert.ObjectsAreEqual(want, got)
		}, e2eWait, 5*time.Millisecond, "step %q: the Raft membership of %s is wrong", step, p.name)
	}

	// the second process to stop leads at least one shard when it can.
	step = "second shutdown"
	second := c.procs[0]
	if c.table().Shards[0].Primary == c.procs[1].name {
		second = c.procs[1]
	}
	c.kill(second)
	c.waitReady(step, e2eRF-1)

	step = "extend after the second shutdown"
	for _, g := range grants {
		_, err := g.account.extend(ctx, cl, g.lease)
		require.NoError(t, err, "step %q: shard %d, account %s, lease %s, table %s", step, g.shard, g.account.id, g.lease, c.tableString())
	}
	step = "grant after the second shutdown"
	for _, a := range accounts {
		_, err := a.acquire(ctx, cl)
		require.NoError(t, err, "step %q: shard %d, account %s, table %s", step, a.shard, a.id, c.tableString())
	}
}
