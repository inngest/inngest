package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// newTestCoordinator returns a coordinator with the nodes a, b and c
// registered and one shard at RF 3 with a as primary.
func newTestCoordinator(t *testing.T, interval, timeout time.Duration) *MemCoordinator {
	t.Helper()
	co := NewMemCoordinator(interval, timeout)
	t.Cleanup(func() { _ = co.Close() })
	admin := co.Node("admin")
	for _, n := range []string{"a", "b", "c"} {
		require.NoError(t, admin.Propose(context.Background(), RegisterNode{
			ID:     n,
			Addr:   n,
			BootID: n + "-boot-1",
		}))
	}
	require.NoError(t, admin.Propose(context.Background(), Init{
		Shards: 1,
		RF:     3,
		Nodes:  []string{"a", "b", "c"},
	}))
	return co
}

// waitTable blocks until the handle's table passes cond and returns it.
func waitTable(t *testing.T, h Coordinator, cond func(*Table) bool) *Table {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		tbl, changed := h.Table()
		if cond(tbl) {
			return tbl
		}
		select {
		case <-changed:
		case <-deadline:
			t.Fatalf("table did not reach the wanted state, last version %d", tbl.Version)
		}
	}
}

// reportEvery sends a report for the node every interval until the returned
// function is called.
func reportEvery(co *MemCoordinator, interval time.Duration, node, bootID string, lsn uint64) (stop func()) {
	quit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		h := co.Node(node)
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			_ = h.Report(context.Background(), Report{
				Node:   node,
				BootID: bootID,
				LSN: map[int]uint64{
					0: lsn,
				},
			})
			select {
			case <-quit:
				return
			case <-tick.C:
			}
		}
	}()
	return func() {
		close(quit)
		<-done
	}
}

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestMemCoordinatorTableAndPropose(t *testing.T) {
	co := newTestCoordinator(t, time.Hour, time.Hour)
	a := co.Node("a")

	tbl, changed := a.Table()
	require.Equal(t, uint64(4), tbl.Version)
	require.False(t, isClosed(changed))

	// a rejected command changes nothing and wakes nobody.
	err := a.Propose(context.Background(), ShrinkInSync{
		Shard: 0,
		Epoch: 9,
		Node:  "b",
	})
	require.ErrorIs(t, err, ErrRejected)
	require.False(t, isClosed(changed))

	require.NoError(t, a.Propose(context.Background(), ShrinkInSync{
		Shard: 0,
		Epoch: 1,
		Node:  "b",
	}))
	require.True(t, isClosed(changed))

	// the table from before is a snapshot.  it does not change.
	require.Equal(t, uint64(4), tbl.Version)
	require.Equal(t, []string{"a", "b", "c"}, tbl.Shards[0].InSync)

	next, changed := co.Node("b").Table()
	require.Equal(t, uint64(5), next.Version)
	require.Equal(t, []string{"a", "c"}, next.Shards[0].InSync)
	require.False(t, isClosed(changed))
}

func TestMemCoordinatorCutAndRestore(t *testing.T) {
	co := newTestCoordinator(t, time.Hour, time.Hour)
	a, b := co.Node("a"), co.Node("b")
	ctx := context.Background()

	before, _ := a.Table()
	co.Cut("a")
	co.Cut("a")

	require.ErrorIs(t, a.Propose(ctx, ShrinkInSync{
		Shard: 0,
		Epoch: 1,
		Node:  "b",
	}), ErrUnavailable)
	require.ErrorIs(t, a.Report(ctx, Report{
		Node:   "a",
		BootID: "a-boot-1",
	}), ErrUnavailable)

	// the rest of the cluster moves on.  the cut node does not see it.
	require.NoError(t, b.Propose(ctx, ShrinkInSync{
		Shard: 0,
		Epoch: 1,
		Node:  "c",
	}))
	stale, changed := a.Table()
	require.Equal(t, before.Version, stale.Version)
	require.False(t, isClosed(changed))
	fresh, _ := b.Table()
	require.Equal(t, before.Version+1, fresh.Version)

	co.Restore("a")
	require.True(t, isClosed(changed))
	now, changed := a.Table()
	require.Equal(t, fresh.Version, now.Version)
	require.False(t, isClosed(changed))
	require.NoError(t, a.Report(ctx, Report{
		Node:   "a",
		BootID: "a-boot-1",
	}))
	require.NoError(t, a.Propose(ctx, ExpandInSync{
		Shard: 0,
		Epoch: 2,
		Node:  "c",
	}))
	require.True(t, isClosed(changed))
}

func TestMemCoordinatorCutDuringDelayedPropose(t *testing.T) {
	co := newTestCoordinator(t, time.Hour, time.Hour)
	co.SetProposeDelay(50 * time.Millisecond)
	errc := make(chan error, 1)
	go func() {
		errc <- co.Node("a").Propose(context.Background(), ShrinkInSync{
			Shard: 0,
			Epoch: 1,
			Node:  "b",
		})
	}()
	co.Cut("a")
	require.ErrorIs(t, <-errc, ErrUnavailable)
	tbl, _ := co.Node("b").Table()
	require.Equal(t, []string{"a", "b", "c"}, tbl.Shards[0].InSync)
}

func TestMemCoordinatorProposeDelay(t *testing.T) {
	co := newTestCoordinator(t, time.Hour, time.Hour)
	a := co.Node("a")
	const delay = 60 * time.Millisecond
	co.SetProposeDelay(delay)

	start := time.Now()
	require.NoError(t, a.Propose(context.Background(), ShrinkInSync{
		Shard: 0,
		Epoch: 1,
		Node:  "b",
	}))
	require.GreaterOrEqual(t, time.Since(start), delay)

	// a caller that gives up during the delay applies nothing.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	err := a.Propose(ctx, ShrinkInSync{
		Shard: 0,
		Epoch: 2,
		Node:  "c",
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	tbl, _ := a.Table()
	require.Equal(t, []string{"a", "c"}, tbl.Shards[0].InSync)

	co.SetProposeDelay(0)
	start = time.Now()
	require.NoError(t, a.Propose(context.Background(), ShrinkInSync{
		Shard: 0,
		Epoch: 2,
		Node:  "c",
	}))
	require.Less(t, time.Since(start), delay)
}

func TestMemCoordinatorControllerGrace(t *testing.T) {
	const timeout = 150 * time.Millisecond
	start := time.Now()
	co := newTestCoordinator(t, 4*time.Millisecond, timeout)
	// the primary never reports.  the controller must still wait one
	// timeout before it replaces it.
	defer reportEvery(co, 4*time.Millisecond, "b", "b-boot-1", 1)()
	defer reportEvery(co, 4*time.Millisecond, "c", "c-boot-1", 2)()

	tbl := waitTable(t, co.Node("b"), func(tbl *Table) bool {
		return tbl.Shards[0].Primary != "a"
	})
	require.GreaterOrEqual(t, time.Since(start), timeout)
	require.Equal(t, "c", tbl.Shards[0].Primary)
	require.Equal(t, uint64(2), tbl.Shards[0].Term)
	require.Equal(t, []string{"a", "b", "c"}, tbl.Shards[0].InSync)
}

func TestMemCoordinatorElectsAfterReportsStop(t *testing.T) {
	const timeout = 250 * time.Millisecond
	co := newTestCoordinator(t, 4*time.Millisecond, timeout)
	stopA := reportEvery(co, 4*time.Millisecond, "a", "a-boot-1", 9)
	defer reportEvery(co, 4*time.Millisecond, "b", "b-boot-1", 5)()
	defer reportEvery(co, 4*time.Millisecond, "c", "c-boot-1", 4)()

	// past the grace, a primary that reports keeps its seat.
	time.Sleep(timeout + timeout/2)
	tbl, _ := co.Node("b").Table()
	require.Equal(t, "a", tbl.Shards[0].Primary)

	stopA()
	stopped := time.Now()
	tbl = waitTable(t, co.Node("b"), func(tbl *Table) bool {
		return tbl.Shards[0].Primary != "a"
	})
	require.GreaterOrEqual(t, time.Since(stopped), timeout/2)
	require.Equal(t, "b", tbl.Shards[0].Primary)
}

func TestMemCoordinatorIgnoresReportWithOldBootID(t *testing.T) {
	co := newTestCoordinator(t, 4*time.Millisecond, 100*time.Millisecond)
	// this process of a was replaced.  its reports must not keep a live.
	defer reportEvery(co, 4*time.Millisecond, "a", "a-boot-0", 9)()
	defer reportEvery(co, 4*time.Millisecond, "b", "b-boot-1", 5)()

	tbl := waitTable(t, co.Node("b"), func(tbl *Table) bool {
		return tbl.Shards[0].Primary != "a"
	})
	require.Equal(t, "b", tbl.Shards[0].Primary)
}

func TestMemCoordinatorStopAndStartController(t *testing.T) {
	const timeout = 100 * time.Millisecond
	co := newTestCoordinator(t, 4*time.Millisecond, timeout)
	defer reportEvery(co, 4*time.Millisecond, "b", "b-boot-1", 1)()

	co.StopController()
	co.StopController()
	// with no controller a dead primary keeps its seat.
	time.Sleep(3 * timeout)
	tbl, _ := co.Node("b").Table()
	require.Equal(t, "a", tbl.Shards[0].Primary)

	started := time.Now()
	co.StartController()
	co.StartController()
	tbl = waitTable(t, co.Node("b"), func(tbl *Table) bool {
		return tbl.Shards[0].Primary != "a"
	})
	// the start applies the grace again.
	require.GreaterOrEqual(t, time.Since(started), timeout)
	require.Equal(t, "b", tbl.Shards[0].Primary)

	require.NoError(t, co.Close())
	require.NoError(t, co.Close())
}
