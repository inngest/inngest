package raftcoord

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/inngest/inngest/pkg/constraintapi/cluster"
	"github.com/stretchr/testify/require"
)

// join starts one more node with an empty store.  it asks the nodes in
// seeds to add it to Raft.
func (tc *testCluster) join(id string, seeds ...*testNode) *testNode {
	tc.t.Helper()
	_, trans := raft.NewInmemTransport(raft.ServerAddress("raft-" + id))
	node := &testNode{
		id:    id,
		trans: trans,
	}
	for _, other := range tc.live() {
		other.trans.Connect(trans.LocalAddr(), trans)
		trans.Connect(other.trans.LocalAddr(), other.trans)
	}
	var addrs []string
	for _, seed := range seeds {
		addrs = append(addrs, seed.id)
	}
	mux := cluster.NewMux()
	transport := tc.net.Transport(id)
	_, err := transport.Listen(id, mux.Accept)
	require.NoError(tc.t, err)
	node.coord, err = New(Config{
		ID:   id,
		Addr: id,
		Peers: []Peer{{
			ID:       id,
			RaftAddr: "raft-" + id,
			Addr:     id,
		}},
		Join:              addrs,
		Transport:         transport,
		Mux:               mux,
		HeartbeatInterval: tc.nodes[0].coord.cfg.HeartbeatInterval,
		HeartbeatTimeout:  tc.nodes[0].coord.cfg.HeartbeatTimeout,
		RaftTransport:     trans,
		InMemory:          true,
		tune:              fastRaft,
	})
	require.NoError(tc.t, err)
	tc.nodes = append(tc.nodes, node)
	return node
}

// members returns the Raft membership a node has as a map from ID to vote.
func members(t *testing.T, node *testNode) map[string]bool {
	t.Helper()
	list, err := node.coord.Members()
	if err != nil {
		return nil
	}
	out := map[string]bool{}
	for _, m := range list {
		out[m.ID] = m.Voter
	}
	return out
}

// waitMembers waits until every live node has this Raft membership.
func (tc *testCluster) waitMembers(want map[string]bool) {
	tc.t.Helper()
	var got map[string]bool
	deadline := time.Now().Add(waitFor)
	for {
		same := true
		for _, node := range tc.live() {
			if got = members(tc.t, node); !mapsEqual(got, want) {
				same = false
				break
			}
		}
		if same {
			return
		}
		if time.Now().After(deadline) {
			tc.t.Fatalf("membership is %v, want %v", got, want)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func mapsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || v != w {
			return false
		}
	}
	return true
}

// leaveInSync takes a node out of every in-sync set, as the primaries do
// after a node died.  a shard that has the node as its primary gets the
// node move as its primary first.
func (tc *testCluster) leaveInSync(via *testNode, node, move string) {
	tc.t.Helper()
	table, _ := via.coord.Table()
	for i, row := range table.Shards {
		if row.Primary == node {
			tc.propose(via, cluster.MovePrimary{
				Shard: i,
				Epoch: row.Epoch,
				Node:  move,
			})
			continue
		}
		tc.propose(via, cluster.ShrinkInSync{
			Shard: i,
			Epoch: row.Epoch,
			Node:  node,
		})
	}
}

// register registers a node through its own coordinator.
func (tc *testCluster) register(node *testNode) {
	tc.t.Helper()
	tc.propose(node, cluster.RegisterNode{
		ID:     node.id,
		Addr:   node.id,
		BootID: "boot-" + node.id,
	})
}

// replace calls ReplaceNode through a node until Raft takes it.
func (tc *testCluster) replace(via *testNode, old, by string) {
	tc.t.Helper()
	var err error
	eventually(tc.t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err = via.coord.ReplaceNode(ctx, old, by)
		return err == nil
	})
}

// other returns a live node that is not one of the given nodes.
func (tc *testCluster) other(not ...*testNode) *testNode {
	tc.t.Helper()
	for _, node := range tc.live() {
		skip := false
		for _, n := range not {
			skip = skip || n == node
		}
		if !skip {
			return node
		}
	}
	tc.t.Fatal("no other live node")
	return nil
}

// TestJoin adds a node to a cluster that runs.  it gets the table, it has no
// vote, and it can propose before it knows the leader.
func TestJoin(t *testing.T) {
	tc := newTestCluster(t, 3, 10*time.Millisecond, time.Hour)
	tc.setup()
	lead := tc.leader()
	follower := tc.other(lead)

	// the seed is not the leader, so it sends the request on.
	joined := tc.join("n3", follower)
	tc.register(joined)
	tc.waitMembers(map[string]bool{
		"n0": true,
		"n1": true,
		"n2": true,
		"n3": false,
	})
	table := waitTable(t, joined.coord, func(table *cluster.Table) bool {
		_, ok := table.Nodes["n3"]
		return ok && len(table.Shards) == 3
	})
	for _, row := range table.Shards {
		require.NotContains(t, row.Replicas, "n3")
	}

	// the leader's address is not in the new node's peer list.  the node
	// finds it in the table.
	tc.propose(joined, cluster.ShrinkInSync{
		Shard: 0,
		Epoch: table.Shards[0].Epoch,
		Node:  followersOf(table.Shards[0])[0],
	})
}

func followersOf(row cluster.ShardRow) []string {
	var out []string
	for _, n := range row.InSync {
		if n != row.Primary {
			out = append(out, n)
		}
	}
	return out
}

// TestJoinRefusedForAVoter checks that a voter cannot join again.  a voter
// that asks has an empty store and no record of its votes.
func TestJoinRefusedForAVoter(t *testing.T) {
	tc := newTestCluster(t, 3, 10*time.Millisecond, time.Hour)
	tc.setup()
	lead := tc.leader()
	follower := tc.other(lead)

	ctx, cancel := context.WithTimeout(context.Background(), waitFor)
	defer cancel()
	err := follower.coord.send(ctx, lead.id, JoinRequest{
		ID:       follower.id,
		RaftAddr: "raft-" + follower.id,
	})
	require.ErrorIs(t, err, cluster.ErrRejected)
	tc.waitMembers(map[string]bool{
		"n0": true,
		"n1": true,
		"n2": true,
	})
}

// TestJoinWhenTheLeaderDies kills the Raft leader right after a node asked
// to join.  the node asks again and the next leader adds it.
func TestJoinWhenTheLeaderDies(t *testing.T) {
	tc := newTestCluster(t, 3, 10*time.Millisecond, time.Hour)
	tc.setup()
	lead := tc.leader()
	follower := tc.other(lead)

	joined := tc.join("n3", lead, follower)
	tc.kill(lead)
	want := map[string]bool{
		"n0": true,
		"n1": true,
		"n2": true,
		"n3": false,
	}
	tc.waitMembers(want)
	tc.register(joined)
	waitTable(t, tc.leader().coord, func(table *cluster.Table) bool {
		_, ok := table.Nodes["n3"]
		return ok
	})
}

// TestReplaceNodeMovesTheVote replaces a dead voter in a cluster of three.
// the new node gets the seats and the vote, and the dead node has none.
// then one more of the first three dies, and the two that are left still
// have a leader, which needs the vote of the new node.
func TestReplaceNodeMovesTheVote(t *testing.T) {
	tc := newTestCluster(t, 3, 10*time.Millisecond, time.Hour)
	tc.setup()
	lead := tc.leader()
	dead := tc.other(lead)
	keep := tc.other(lead, dead)
	tc.kill(dead)
	tc.leaveInSync(lead, dead.id, lead.id)

	joined := tc.join("n3", keep)
	tc.register(joined)
	// the call goes through a node that is not the leader.
	tc.replace(keep, dead.id, "n3")

	tc.waitMembers(map[string]bool{
		lead.id: true,
		keep.id: true,
		"n3":    true,
	})
	table := waitTable(t, joined.coord, func(table *cluster.Table) bool {
		return table.Removed[dead.id] == "n3"
	})
	require.NotContains(t, table.Nodes, dead.id)
	require.True(t, table.Nodes["n3"].Voter)
	for i, row := range table.Shards {
		require.Contains(t, row.Replicas, "n3", "shard %d", i)
		require.NotContains(t, row.Replicas, dead.id, "shard %d", i)
		require.NotContains(t, row.InSync, "n3", "shard %d", i)
	}

	// the same call again is not an error.
	ctx, cancel := context.WithTimeout(context.Background(), waitFor)
	defer cancel()
	require.NoError(t, keep.coord.ReplaceNode(ctx, dead.id, "n3"))

	tc.kill(lead)
	next := tc.leader()
	tc.propose(next, cluster.ExpandInSync{
		Shard: 0,
		Epoch: table.Shards[0].Epoch,
		Node:  "n3",
	})
}

// TestReplaceNodeNeedsAJoinedNode checks that the table does not change
// when the new node is not in Raft.
func TestReplaceNodeNeedsAJoinedNode(t *testing.T) {
	tc := newTestCluster(t, 3, 10*time.Millisecond, time.Hour)
	tc.setup()
	lead := tc.leader()
	dead := tc.other(lead)
	tc.kill(dead)
	tc.leaveInSync(lead, dead.id, lead.id)
	tc.propose(lead, cluster.RegisterNode{
		ID:     "n3",
		Addr:   "n3",
		BootID: "boot-n3",
	})
	before, _ := lead.coord.Table()

	ctx, cancel := context.WithTimeout(context.Background(), waitFor)
	defer cancel()
	err := lead.coord.ReplaceNode(ctx, dead.id, "n3")
	require.ErrorIs(t, err, cluster.ErrRejected)
	after, _ := lead.coord.Table()
	require.Equal(t, before, after)
}

// TestReplaceNodeIsFinishedByTheNextLeader stops the Raft leader between
// the table command and the change of the Raft membership, and kills it.
// the next leader removes the dead node and gives the new node its vote.
// the cluster has five voters, so it has a leader with two of them dead.
func TestReplaceNodeIsFinishedByTheNextLeader(t *testing.T) {
	tc := newTestCluster(t, 5, 10*time.Millisecond, time.Hour)
	tc.setup()
	lead := tc.leader()
	dead := tc.other(lead)
	keep := tc.other(lead, dead)
	tc.kill(dead)
	tc.leaveInSync(lead, dead.id, lead.id)

	joined := tc.join("n5", keep)
	tc.register(joined)
	lead.coord.holdMembership.Store(true)
	tc.replace(lead, dead.id, "n5")
	got := members(t, lead)
	require.True(t, got[dead.id], "the leader has not changed Raft")
	require.False(t, got["n5"])

	// the dead leader keeps its vote.  nothing replaced it.
	tc.kill(lead)
	want := map[string]bool{"n5": true}
	for _, node := range tc.nodes {
		if node != dead && node != joined {
			want[node.id] = true
		}
	}
	tc.waitMembers(want)
}

// TestRemovedNodeIsTold cuts a voter off, replaces it and lets it back in.
// Raft sends it nothing any more, so it knows no leader.  its report and its
// registration go through the other members, and both come back with
// cluster.ErrRemoved.
func TestRemovedNodeIsTold(t *testing.T) {
	tc := newTestCluster(t, 3, 10*time.Millisecond, time.Hour)
	tc.setup()
	lead := tc.leader()
	gone := tc.other(lead)
	keep := tc.other(lead, gone)

	tc.net.Isolate(gone.id)
	gone.trans.DisconnectAll()
	for _, node := range tc.live() {
		if node != gone {
			node.trans.Disconnect(gone.trans.LocalAddr())
		}
	}
	tc.leaveInSync(lead, gone.id, lead.id)
	joined := tc.join("n3", keep)
	// the new node cannot reach the node that is cut off.
	joined.trans.Disconnect(gone.trans.LocalAddr())
	tc.register(joined)
	tc.replace(lead, gone.id, "n3")
	want := map[string]bool{
		lead.id: true,
		keep.id: true,
		"n3":    true,
	}
	eventually(t, func() bool { return mapsEqual(members(t, lead), want) })

	tc.net.Rejoin(gone.id)
	for _, node := range tc.live() {
		if node != gone {
			node.trans.Connect(gone.trans.LocalAddr(), gone.trans)
			gone.trans.Connect(node.trans.LocalAddr(), node.trans)
		}
	}
	report := cluster.Report{
		Node:   gone.id,
		BootID: "boot-" + gone.id,
	}
	eventually(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		return errors.Is(gone.coord.Report(ctx, report), cluster.ErrRemoved)
	})
	eventually(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err := gone.coord.Propose(ctx, cluster.RegisterNode{
			ID:     gone.id,
			Addr:   gone.id,
			BootID: "boot-2-" + gone.id,
		})
		return errors.Is(err, cluster.ErrRemoved)
	})

	// the node that came back asks for votes and gets none.  the helper
	// that finds the leader must not wait for it to know one.
	time.Sleep(500 * time.Millisecond)
	gone.killed = true
	require.True(t, mapsEqual(members(t, tc.leader()), want))
	tc.propose(keep, cluster.ExpandInSync{
		Shard: 0,
		Epoch: func() uint64 { table, _ := keep.coord.Table(); return table.Shards[0].Epoch }(),
		Node:  "n3",
	})
}
