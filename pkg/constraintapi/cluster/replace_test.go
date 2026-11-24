package cluster

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/stretchr/testify/require"
)

// lastError prints the newest error when a failure message is built.
type lastError struct {
	err error
}

func (e *lastError) String() string {
	return fmt.Sprint(e.err)
}

// add starts one more node.  it has no seat until a ReplaceNode gives it
// the seats of another node.
func (c *testCluster) add(t testing.TB, name string) {
	t.Helper()
	c.names = append(c.names, name)
	require.NoError(t, c.Restart(name))
	eventually(t, 5*time.Second, func() bool {
		_, ok := c.Table().Nodes[name]
		return ok
	}, "node %s did not register", name)
}

// replace proposes ReplaceNode until the table takes it.  the table refuses
// it while a primary has not yet removed the old node from its in-sync set.
func (c *testCluster) replace(t testing.TB, old, by string) {
	t.Helper()
	last := &lastError{}
	eventually(t, 15*time.Second, func() bool {
		last.err = c.co.Node("harness").Propose(context.Background(), ReplaceNode{
			Old: old,
			New: by,
		})
		return last.err == nil
	}, "node %s was not replaced by %s, last error %s, table %s", old, by, last, tableString(c))
}

// seated reports whether the node is in the in-sync set of every shard.
func (c *testCluster) seated(name string) bool {
	if !c.stable() {
		return false
	}
	for _, row := range c.Table().Shards {
		if !slices.Contains(row.InSync, name) {
			return false
		}
	}
	return true
}

// startLedger runs the ledger until the returned function is called.
func startLedger(l *ledger, rt constraintapi.CapacityManager) (stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		l.run(ctx, rt)
	}()
	return func() {
		cancel()
		wg.Wait()
	}
}

// TestReplaceDeadNodeUnderLoad kills one of three nodes, gives its seats to
// a new node while the ledger runs, and then kills a second node.  the new
// node has every answered grant, because after the second kill every shard
// has only one of the first three nodes and the new node.
func TestReplaceDeadNodeUnderLoad(t *testing.T) {
	c := newFailoverCluster(t, 3, 2, 3)
	rt := newTestClient(t, c)

	l := newLedger(ledgerConfig{
		seed:    11,
		workers: 8,
		keys:    8,
		limit:   5,
	})
	stop := startLedger(l, rt)

	time.Sleep(300 * time.Millisecond)
	dead := c.Table().Shards[0].Primary
	c.Kill(dead)
	c.add(t, "n3")
	c.replace(t, dead, "n3")

	tbl := c.Table()
	require.NotContains(t, tbl.Nodes, dead)
	for i, row := range tbl.Shards {
		require.Contains(t, row.Replicas, "n3", "shard %d", i)
		require.NotContains(t, row.Replicas, dead, "shard %d", i)
		require.Len(t, row.Replicas, 3)
	}
	eventually(t, 30*time.Second, func() bool { return c.seated("n3") },
		"the new node did not reach every in-sync set, table %s", tableString(c))

	// the new node now is one of two copies.
	second := c.Table().Shards[0].Primary
	if second == "n3" {
		second = followersOf(c.Table().Shards[0])[0]
	}
	c.Kill(second)
	eventually(t, 10*time.Second, func() bool {
		for _, row := range c.Table().Shards {
			if row.Primary == second {
				return false
			}
		}
		return c.stable()
	}, "no new primary after the second kill, table %s", tableString(c))
	time.Sleep(300 * time.Millisecond)
	stop()

	c.waitStable(t)
	check, done := context.WithTimeout(context.Background(), 20*time.Second)
	defer done()
	require.NoError(t, l.checkErrors())
	require.NoError(t, l.checkExtendable(check, rt))
	require.Greater(t, len(l.events), 100)

	c.waitCaughtUp(t)
	for shard, row := range c.Table().Shards {
		want := canonState(t, c.node(row.Primary).Replica(shard))
		for _, f := range followersOf(row) {
			if c.Running(f) {
				require.Equal(t, want, canonState(t, c.node(f).Replica(shard)), "shard %d follower %s", shard, f)
			}
		}
	}
}

// TestReplaceIsRefusedForALiveNode checks that the table keeps a node that
// is in sync.  its seats have records that a new node does not have.
func TestReplaceIsRefusedForALiveNode(t *testing.T) {
	c := newNodeCluster(t, 3, 2, 3)
	c.add(t, "n3")
	before := c.Table()
	err := c.co.Node("harness").Propose(context.Background(), ReplaceNode{
		Old: "n2",
		New: "n3",
	})
	require.ErrorIs(t, err, ErrRejected)
	require.Equal(t, before, c.Table())
}

// TestNewNodeKilledDuringItsTransfer kills the new node before it is in
// sync and starts it again.  the two old nodes answer all the time, and the
// second start ends in every in-sync set.
func TestNewNodeKilledDuringItsTransfer(t *testing.T) {
	c := newFailoverCluster(t, 3, 2, 3)
	rt := newTestClient(t, c)

	l := newLedger(ledgerConfig{
		seed:    12,
		workers: 8,
		keys:    8,
		limit:   5,
	})
	stop := startLedger(l, rt)

	time.Sleep(200 * time.Millisecond)
	c.Kill("n2")
	c.add(t, "n3")
	c.replace(t, "n2", "n3")
	// a node enters an in-sync set one idempotency TTL after its transfer,
	// so the new node is a follower outside every in-sync set now.
	eventually(t, 5*time.Second, func() bool {
		r := c.node("n3").replica(0)
		return r != nil && r.applied.Load() > 0
	}, "the new node got no state")
	for _, row := range c.Table().Shards {
		require.NotContains(t, row.InSync, "n3")
	}
	c.Kill("n3")

	time.Sleep(300 * time.Millisecond)
	c.waitStable(t)
	for _, row := range c.Table().Shards {
		require.Len(t, row.InSync, 2)
		require.NotContains(t, row.InSync, "n3")
	}

	require.NoError(t, c.Restart("n3"))
	eventually(t, 30*time.Second, func() bool { return c.seated("n3") },
		"the new node did not reach every in-sync set, table %s", tableString(c))
	stop()

	check, done := context.WithTimeout(context.Background(), 20*time.Second)
	defer done()
	require.NoError(t, l.checkErrors())
	require.NoError(t, l.checkExtendable(check, rt))
	require.Greater(t, len(l.events), 100)
}

// TestReplacedNodeThatReturnsStops cuts one node off, replaces it, and lets
// it back in.  the process that was cut off closes itself, and a new process
// with the same name cannot register.
func TestReplacedNodeThatReturnsStops(t *testing.T) {
	c := newFailoverCluster(t, 3, 2, 3)
	rt := newTestClient(t, c)

	l := newLedger(ledgerConfig{
		seed:    13,
		workers: 8,
		keys:    8,
		limit:   5,
	})
	stop := startLedger(l, rt)

	time.Sleep(200 * time.Millisecond)
	gone := c.Table().Shards[0].Primary
	old := c.node(gone)
	c.co.Cut(gone)
	c.net.Isolate(gone)
	c.add(t, "n3")
	c.replace(t, gone, "n3")
	require.False(t, old.Removed(), "the node cannot know yet")

	c.net.Rejoin(gone)
	c.co.Restore(gone)
	eventually(t, 5*time.Second, old.Removed, "the node that came back did not stop")
	eventually(t, 5*time.Second, func() bool {
		for i := range c.Table().Shards {
			if _, err := old.Primary(i); err == nil {
				return false
			}
		}
		return true
	}, "the node that came back is still a primary")

	require.NoError(t, c.Restart(gone))
	again := c.node(gone)
	eventually(t, 5*time.Second, again.Removed, "a new process with the old name did not stop")
	require.NotContains(t, c.Table().Nodes, gone)

	eventually(t, 30*time.Second, func() bool { return c.seated("n3") },
		"the new node did not reach every in-sync set, table %s", tableString(c))
	stop()

	check, done := context.WithTimeout(context.Background(), 20*time.Second)
	defer done()
	require.NoError(t, l.checkErrors())
	require.NoError(t, l.checkExtendable(check, rt))
	require.Greater(t, len(l.events), 100)
}

// TestTwoReplacementsOfOneNode proposes two new nodes for one dead node at
// the same time.  one gets every seat and the other gets none.
func TestTwoReplacementsOfOneNode(t *testing.T) {
	c := newNodeCluster(t, 3, 4, 3)
	c.Kill("n2")
	c.add(t, "n3")
	c.add(t, "n4")
	eventually(t, 10*time.Second, func() bool {
		for _, row := range c.Table().Shards {
			if slices.Contains(row.InSync, "n2") {
				return false
			}
		}
		return c.stable()
	}, "the dead node is still in sync, table %s", tableString(c))

	errs := make(chan error, 2)
	for _, by := range []string{"n3", "n4"} {
		go func() {
			errs <- c.co.Node(by).Propose(context.Background(), ReplaceNode{
				Old: "n2",
				New: by,
			})
		}()
	}
	first, second := <-errs, <-errs
	if first != nil {
		first, second = second, first
	}
	require.NoError(t, first)
	require.ErrorIs(t, second, ErrRejected)
	require.False(t, errors.Is(second, ErrRemoved))

	tbl := c.Table()
	winner := tbl.Removed["n2"]
	loser := "n3"
	if winner == "n3" {
		loser = "n4"
	}
	require.Contains(t, []string{"n3", "n4"}, winner)
	for i, row := range tbl.Shards {
		require.Contains(t, row.Replicas, winner, "shard %d", i)
		require.NotContains(t, row.Replicas, loser, "shard %d", i)
		require.NotContains(t, row.Replicas, "n2", "shard %d", i)
	}
	require.Contains(t, tbl.Nodes, loser)
	eventually(t, 30*time.Second, func() bool { return c.seated(winner) },
		"the new node did not reach every in-sync set, table %s", tableString(c))
	require.False(t, c.node(loser).Removed())
}

// TestReplaceLiveNodeAfterDrain replaces a node that runs.  a drain takes it
// out of every in-sync set with no timeout, so the table takes the command
// at once.  the node then sees that it was replaced and closes itself.
func TestReplaceLiveNodeAfterDrain(t *testing.T) {
	c := newFailoverCluster(t, 3, 2, 3)
	rt := newTestClient(t, c)

	l := newLedger(ledgerConfig{
		seed:    14,
		workers: 8,
		keys:    8,
		limit:   5,
	})
	stop := startLedger(l, rt)

	time.Sleep(200 * time.Millisecond)
	leaving := c.Table().Shards[0].Primary
	old := c.node(leaving)
	c.add(t, "n3")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	require.NoError(t, old.Drain(ctx))
	require.NoError(t, c.co.Node("harness").Propose(ctx, ReplaceNode{
		Old: leaving,
		New: "n3",
	}))
	eventually(t, 5*time.Second, old.Removed, "the node that was replaced did not stop")
	eventually(t, 30*time.Second, func() bool { return c.seated("n3") },
		"the new node did not reach every in-sync set, table %s", tableString(c))
	stop()

	require.NoError(t, l.checkErrors())
	require.NoError(t, l.checkExtendable(ctx, rt))
	require.Greater(t, len(l.events), 100)
}
