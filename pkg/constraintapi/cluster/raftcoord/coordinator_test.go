package raftcoord

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/inngest/inngest/pkg/constraintapi/cluster"
	"github.com/stretchr/testify/require"
)

const waitFor = 20 * time.Second

func fastRaft(c *raft.Config) {
	c.HeartbeatTimeout = 100 * time.Millisecond
	c.ElectionTimeout = 100 * time.Millisecond
	c.LeaderLeaseTimeout = 100 * time.Millisecond
	c.CommitTimeout = 5 * time.Millisecond
}

type testNode struct {
	id     string
	coord  *Coordinator
	trans  *raft.InmemTransport
	killed bool
	// stopReports stops the node's reports and returns when the last
	// report that got an answer was sent.
	stopReports func() time.Time
}

type testCluster struct {
	t     *testing.T
	net   *cluster.MemNetwork
	nodes []*testNode
}

// newTestCluster starts n voters with in-memory Raft stores and transports.
func newTestCluster(t *testing.T, n int, interval, timeout time.Duration) *testCluster {
	tc := &testCluster{
		t:   t,
		net: cluster.NewMemNetwork(),
	}
	var peers []Peer
	for i := range n {
		id := fmt.Sprintf("n%d", i)
		_, trans := raft.NewInmemTransport(raft.ServerAddress("raft-" + id))
		tc.nodes = append(tc.nodes, &testNode{
			id:    id,
			trans: trans,
		})
		peers = append(peers, Peer{
			ID:       id,
			RaftAddr: "raft-" + id,
			Addr:     id,
			Voter:    true,
		})
	}
	for _, a := range tc.nodes {
		for _, b := range tc.nodes {
			if a != b {
				a.trans.Connect(b.trans.LocalAddr(), b.trans)
			}
		}
	}
	for _, node := range tc.nodes {
		mux := cluster.NewMux()
		transport := tc.net.Transport(node.id)
		_, err := transport.Listen(node.id, mux.Accept)
		require.NoError(t, err)
		node.coord, err = New(Config{
			ID:                node.id,
			Addr:              node.id,
			Peers:             peers,
			Transport:         transport,
			Mux:               mux,
			HeartbeatInterval: interval,
			HeartbeatTimeout:  timeout,
			RaftTransport:     node.trans,
			InMemory:          true,
			tune:              fastRaft,
		})
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		for _, node := range tc.nodes {
			if node.stopReports != nil {
				node.stopReports()
			}
			_ = node.coord.Close()
		}
	})
	return tc
}

func (tc *testCluster) live() []*testNode {
	var out []*testNode
	for _, node := range tc.nodes {
		if !node.killed {
			out = append(out, node)
		}
	}
	return out
}

// leader waits until one live node is the Raft leader and every live node
// knows it.
func (tc *testCluster) leader() *testNode {
	tc.t.Helper()
	var lead *testNode
	eventually(tc.t, func() bool {
		lead = nil
		for _, node := range tc.live() {
			if node.coord.raft.State() == raft.Leader {
				lead = node
			}
		}
		if lead == nil {
			return false
		}
		for _, node := range tc.live() {
			if _, id := node.coord.raft.LeaderWithID(); string(id) != lead.id {
				return false
			}
		}
		return true
	})
	return lead
}

// kill stops the node's Raft, its listener and its reports.
func (tc *testCluster) kill(node *testNode) {
	if node.stopReports != nil {
		node.stopReports()
	}
	node.killed = true
	require.NoError(tc.t, node.coord.Close())
	tc.net.Kill(node.id)
	for _, other := range tc.live() {
		other.trans.Disconnect(node.trans.LocalAddr())
	}
}

// setup registers every node with a boot ID and creates one shard per node
// with every node as a replica.  shard i has node i as primary.  it returns
// when every node has that table.
func (tc *testCluster) setup() {
	tc.t.Helper()
	var ids []string
	for _, node := range tc.nodes {
		ids = append(ids, node.id)
		tc.propose(node, cluster.RegisterNode{
			ID:     node.id,
			Addr:   node.id,
			BootID: "boot-" + node.id,
		})
	}
	tc.propose(tc.nodes[0], cluster.Init{
		Shards: len(ids),
		RF:     len(ids),
		Nodes:  ids,
	})
	for _, node := range tc.nodes {
		waitTable(tc.t, node.coord, func(table *cluster.Table) bool {
			return len(table.Shards) == len(ids)
		})
	}
}

// propose retries while the coordinator is unavailable, which it is for a
// short time around a Raft election.
func (tc *testCluster) propose(node *testNode, cmd cluster.Command) {
	tc.t.Helper()
	var err error
	eventually(tc.t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err = node.coord.Propose(ctx, cmd)
		return !errors.Is(err, cluster.ErrUnavailable)
	})
	require.NoError(tc.t, err)
}

// startReports makes every node report through its own coordinator.  lsn
// gives the LSN a node reports for every shard.
func (tc *testCluster) startReports(interval time.Duration, lsn map[string]uint64) {
	for _, node := range tc.nodes {
		r := cluster.Report{
			Node:   node.id,
			BootID: "boot-" + node.id,
			LSN:    map[int]uint64{},
		}
		for shard := range tc.nodes {
			r.LSN[shard] = lsn[node.id]
		}
		stop := make(chan struct{})
		done := make(chan struct{})
		var last time.Time
		go func() {
			defer close(done)
			tick := time.NewTicker(interval)
			defer tick.Stop()
			for {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				sent := time.Now()
				if node.coord.Report(ctx, r) == nil {
					last = sent
				}
				cancel()
				select {
				case <-stop:
					return
				case <-tick.C:
				}
			}
		}()
		var once sync.Once
		node.stopReports = func() time.Time {
			once.Do(func() { close(stop) })
			<-done
			return last
		}
	}
}

// reported waits until the Raft leader has a report from every node.
func (tc *testCluster) reported() {
	tc.t.Helper()
	eventually(tc.t, func() bool {
		for _, node := range tc.live() {
			c := node.coord
			c.mu.Lock()
			ok := c.leader && len(c.reports) == len(tc.nodes)
			c.mu.Unlock()
			if ok {
				return true
			}
		}
		return false
	})
}

func eventually(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(waitFor)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// waitTable waits until the node's table passes cond.
func waitTable(t *testing.T, c *Coordinator, cond func(*cluster.Table) bool) *cluster.Table {
	t.Helper()
	deadline := time.After(waitFor)
	for {
		table, changed := c.Table()
		if cond(table) {
			return table
		}
		select {
		case <-changed:
		case <-deadline:
			t.Fatalf("table did not reach the wanted state, last %+v", table)
		}
	}
}

func TestProposeFromFollower(t *testing.T) {
	tc := newTestCluster(t, 3, 20*time.Millisecond, time.Second)
	lead := tc.leader()
	var follower *testNode
	var fired []<-chan struct{}
	for _, node := range tc.nodes {
		if node != lead {
			follower = node
		}
		_, changed := node.coord.Table()
		fired = append(fired, changed)
	}

	init := cluster.Init{
		Shards: 4,
		RF:     2,
		Nodes:  []string{"n0", "n1", "n2"},
	}
	tc.propose(follower, init)
	for i, node := range tc.nodes {
		select {
		case <-fired[i]:
		case <-time.After(waitFor):
			t.Fatalf("the table channel of %s did not fire", node.id)
		}
		table := waitTable(t, node.coord, func(table *cluster.Table) bool {
			return len(table.Shards) == 4
		})
		require.Equal(t, uint64(1), table.Version)
		require.Equal(t, "n1", table.Shards[1].Primary)
	}

	ctx, cancel := context.WithTimeout(context.Background(), waitFor)
	defer cancel()
	for _, node := range []*testNode{tc.leader(), follower} {
		err := node.coord.Propose(ctx, init)
		require.ErrorIs(t, err, cluster.ErrRejected, node.id)
		require.NotErrorIs(t, err, cluster.ErrUnavailable)
		require.Contains(t, err.Error(), "already has shards")
	}
	table, _ := follower.coord.Table()
	require.Equal(t, uint64(1), table.Version)
}

func TestControllerElectsWhenReportsStop(t *testing.T) {
	const interval, timeout = 20 * time.Millisecond, 300 * time.Millisecond
	tc := newTestCluster(t, 3, interval, timeout)
	tc.setup()
	// n2 is ahead of n1, so the controller must pick n2 for the shard of n0.
	tc.startReports(interval, map[string]uint64{
		"n1": 5,
		"n2": 9,
	})
	tc.reported()

	last := tc.nodes[0].stopReports()
	table := waitTable(t, tc.nodes[1].coord, func(table *cluster.Table) bool {
		return table.Shards[0].Primary != "n0"
	})
	require.Greater(t, time.Since(last), timeout)
	require.Equal(t, "n2", table.Shards[0].Primary)
	require.Equal(t, uint64(2), table.Shards[0].Term)
	require.Equal(t, []string{"n0", "n1", "n2"}, table.Shards[0].InSync)
	for shard := 1; shard < 3; shard++ {
		require.Equal(t, tc.nodes[shard].id, table.Shards[shard].Primary)
		require.Equal(t, uint64(1), table.Shards[shard].Term)
	}
}

func TestRaftLeaderShutdown(t *testing.T) {
	const interval, timeout = 20 * time.Millisecond, 1500 * time.Millisecond
	tc := newTestCluster(t, 3, interval, timeout)
	tc.setup()
	tc.startReports(interval, nil)
	tc.reported()

	old := tc.leader()
	var shard int
	for i, node := range tc.nodes {
		if node == old {
			shard = i
		}
	}
	killed := time.Now()
	tc.kill(old)

	lead := tc.leader()
	require.NotEqual(t, old.id, lead.id)
	for i, node := range tc.live() {
		tc.propose(node, cluster.RegisterNode{
			ID:     fmt.Sprintf("extra%d", i),
			BootID: "boot",
		})
	}

	// the new leader got its seat after the kill, so its wait cannot end
	// before one heartbeat timeout after the kill.
	table, _ := lead.coord.Table()
	if time.Since(killed) < timeout {
		require.Equal(t, old.id, table.Shards[shard].Primary)
	}
	table = waitTable(t, lead.coord, func(table *cluster.Table) bool {
		return table.Shards[shard].Primary != old.id
	})
	require.GreaterOrEqual(t, time.Since(killed), timeout)
	require.NotEmpty(t, table.Shards[shard].Primary)
	require.Equal(t, uint64(2), table.Shards[shard].Term)
	for i, node := range tc.nodes {
		if i != shard {
			require.Equal(t, node.id, table.Shards[i].Primary)
			require.Equal(t, uint64(1), table.Shards[i].Term)
		}
	}
}

func TestNoQuorum(t *testing.T) {
	for _, keepLeader := range []bool{true, false} {
		t.Run(fmt.Sprintf("leader survives %v", keepLeader), func(t *testing.T) {
			tc := newTestCluster(t, 3, 20*time.Millisecond, time.Second)
			tc.setup()
			lead := tc.leader()
			var keep *testNode
			for _, node := range tc.nodes {
				if (node == lead) == keepLeader {
					keep = node
				}
			}
			for _, node := range tc.nodes {
				if node != keep {
					tc.kill(node)
				}
			}

			const deadline = 300 * time.Millisecond
			for range 3 {
				ctx, cancel := context.WithTimeout(context.Background(), deadline)
				start := time.Now()
				err := keep.coord.Propose(ctx, cluster.RegisterNode{
					ID:     "late",
					BootID: "boot",
				})
				cancel()
				require.ErrorIs(t, err, cluster.ErrUnavailable)
				require.Less(t, time.Since(start), deadline+200*time.Millisecond)
			}
		})
	}
}

func TestRestartFromDisk(t *testing.T) {
	// the port is free when the listener closes.  Raft binds it again.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	raftAddr := l.Addr().String()
	require.NoError(t, l.Close())

	cfg := Config{
		ID:   "n0",
		Addr: "n0",
		Peers: []Peer{{
			ID:       "n0",
			RaftAddr: raftAddr,
			Addr:     "n0",
			Voter:    true,
		}},
		DataDir:           t.TempDir(),
		Transport:         cluster.NewMemNetwork().Transport("n0"),
		HeartbeatInterval: 20 * time.Millisecond,
		HeartbeatTimeout:  time.Second,
		tune:              fastRaft,
	}
	open := func() *Coordinator {
		cfg.Mux = cluster.NewMux()
		c, err := New(cfg)
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.Close() })
		return c
	}
	propose := func(c *Coordinator, cmd cluster.Command) {
		eventually(t, func() bool {
			err = c.Propose(context.Background(), cmd)
			return !errors.Is(err, cluster.ErrUnavailable)
		})
		require.NoError(t, err)
	}

	c := open()
	propose(c, cluster.RegisterNode{
		ID:     "n0",
		Addr:   "n0",
		BootID: "boot-1",
	})
	propose(c, cluster.Init{
		Shards: 2,
		RF:     1,
		Nodes:  []string{"n0"},
	})
	// the snapshot has the first two commands.  the third is only in the
	// Raft log, so the restart reads both files.
	require.NoError(t, c.raft.Snapshot().Error())
	propose(c, cluster.RegisterNode{
		ID:     "n1",
		Addr:   "n1",
		BootID: "boot-2",
	})
	want, _ := c.Table()
	require.Equal(t, uint64(3), want.Version)
	require.NoError(t, c.Close())

	c = open()
	restored, _ := c.Table()
	require.GreaterOrEqual(t, restored.Version, uint64(2))
	got := waitTable(t, c, func(table *cluster.Table) bool {
		return table.Version == want.Version
	})
	require.Equal(t, want, got)
}
