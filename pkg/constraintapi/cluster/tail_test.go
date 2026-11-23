package cluster

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// lsnChunk is the encoding of count records for the LSNs from first on.
// each record is a take whose seq is its own LSN, so a test can tell which
// records a read returned.
func lsnChunk(first uint64, count int) []byte {
	recs := make([]memory.Record, count)
	for i := range recs {
		recs[i] = memory.Record{
			Kind: memory.RecTake,
			Seq:  first + uint64(i),
		}
	}
	return AppendRecords(nil, recs)
}

// requireRun checks that data decodes to count takes numbered from first.
func requireRun(t *testing.T, data []byte, first uint64, count int) {
	t.Helper()
	recs, err := DecodeRecords(nil, data)
	require.NoError(t, err)
	require.Len(t, recs, count)
	for i, r := range recs {
		require.Equal(t, first+uint64(i), r.Seq, "record %d", i)
	}
}

func TestTailEvictsByBytes(t *testing.T) {
	now := time.Now()
	per := len(lsnChunk(1, 10)) + chunkOverhead
	tl := newTail(3*per, time.Hour)
	for i := range 5 {
		first := uint64(1 + 10*i)
		tl.append(first, 10, lsnChunk(first, 10), now)
	}

	// five chunks went in and three fit.  the oldest two are gone.
	bytes, records := tl.size()
	require.Equal(t, 3*per, bytes)
	require.Equal(t, 30, records)
	require.False(t, tl.covers(20))
	require.True(t, tl.covers(21))
	require.True(t, tl.covers(51), "the LSN after the last record is covered")
	require.False(t, tl.covers(52))
	_, _, _, ok := tl.read(20, 0)
	require.False(t, ok)

	// one chunk larger than the limit leaves the tail empty.  it still
	// covers the end of the log.
	big := lsnChunk(51, 1000)
	tl.append(51, 1000, big, now)
	bytes, records = tl.size()
	require.Zero(t, bytes)
	require.Zero(t, records)
	require.False(t, tl.covers(1050))
	require.True(t, tl.covers(1051))
}

func TestTailEvictsByAge(t *testing.T) {
	t0 := time.Now()
	tl := newTail(1<<20, 10*time.Second)
	tl.append(1, 10, lsnChunk(1, 10), t0)
	tl.append(11, 10, lsnChunk(11, 10), t0.Add(6*time.Second))
	require.True(t, tl.covers(1))

	// the append at 12 s finds the first chunk older than the window.
	tl.append(21, 10, lsnChunk(21, 10), t0.Add(12*time.Second))
	require.False(t, tl.covers(1))
	require.True(t, tl.covers(11))

	// an idle shard appends nothing.  evict drops what aged out.
	tl.evict(t0.Add(17 * time.Second))
	require.False(t, tl.covers(11))
	require.True(t, tl.covers(21))

	tl.evict(t0.Add(time.Minute))
	bytes, records := tl.size()
	require.Zero(t, bytes)
	require.Zero(t, records)
	require.False(t, tl.covers(30))
	require.True(t, tl.covers(31))
	first, count, data, ok := tl.read(31, 0)
	require.True(t, ok)
	require.Equal(t, uint64(31), first)
	require.Zero(t, count)
	require.Nil(t, data)

	// the log goes on from where it was.
	tl.append(31, 5, lsnChunk(31, 5), t0.Add(time.Minute))
	first, count, data, ok = tl.read(31, 0)
	require.True(t, ok)
	require.Equal(t, uint64(31), first)
	requireRun(t, data, 31, count)

	// records that do not follow the last one replace the tail.
	tl.append(100, 5, lsnChunk(100, 5), t0.Add(time.Minute))
	require.False(t, tl.covers(36))
	require.True(t, tl.covers(100))
}

func TestTailReadReturnsWholeChunks(t *testing.T) {
	now := time.Now()
	tl := newTail(1<<20, time.Hour)
	chunks := [][]byte{lsnChunk(1, 10), lsnChunk(11, 10), lsnChunk(21, 10)}
	for i, c := range chunks {
		tl.append(uint64(1+10*i), 10, c, now)
	}
	size := len(chunks[1])

	// a read from inside a chunk starts at the start of that chunk.
	first, count, data, ok := tl.read(15, 0)
	require.True(t, ok)
	require.Equal(t, uint64(11), first)
	require.Equal(t, 20, count)
	requireRun(t, data, 11, 20)

	// one chunk is returned as it is, with no copy.
	first, count, data, ok = tl.read(15, size)
	require.True(t, ok)
	require.Equal(t, uint64(11), first)
	require.Equal(t, 10, count)
	require.Same(t, &chunks[1][0], &data[0])

	// chunks are joined while they fit in the limit.
	first, count, data, ok = tl.read(1, 2*size)
	require.True(t, ok)
	require.Equal(t, uint64(1), first)
	require.Equal(t, 20, count)
	requireRun(t, data, 1, 20)

	// a limit below one chunk still returns one chunk.
	first, count, data, ok = tl.read(30, 1)
	require.True(t, ok)
	require.Equal(t, uint64(21), first)
	require.Equal(t, 10, count)
	requireRun(t, data, 21, 10)

	_, _, _, ok = tl.read(32, 0)
	require.False(t, ok, "the tail has never reached 32")
}

// cellValue returns the counter a manager has for the cell key a.b.
func cellValue(m *memory.Manager, a, b uint64) (v int64) {
	m.Freeze(func() {
		m.Export(func(r memory.Record) {
			if r.Kind == memory.RecCell && r.KeyA == a && r.KeyB == b {
				v = r.Value
			}
		})
	})
	return v
}

// TestFollowerResumesInTheMiddleOfAChunk plays the primary by hand.  the
// follower has records 1 to 3, and the batch it gets after it dials again
// starts at record 2, because a primary sends whole chunks of its tail.
func TestFollowerResumesInTheMiddleOfAChunk(t *testing.T) {
	c := newTestCluster(t, 2, func(name string, tr Transport, co Coordinator) (testNode, error) {
		return NewNode(testNodeConfig(name, tr, co))
	})
	// n0 never reports.  the controller must not give its seat to n1.
	c.co.StopController()
	require.NoError(t, c.Init(1, 2))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, c.co.Node("n0").Propose(ctx, RegisterNode{
		ID:     "n0",
		Addr:   "n0",
		BootID: "by hand",
	}))

	streams := make(chan Stream, 4)
	ln, err := c.net.Transport("n0").Listen("n0", func(s Stream) { streams <- s })
	require.NoError(t, err)
	defer ln.Close()
	require.NoError(t, c.Restart("n1"))

	// every record adds one to the same counter, so the counter says how
	// many records the follower applied.
	deltas := func(n int) []byte {
		recs := make([]memory.Record, n)
		for i := range recs {
			recs[i] = memory.Record{
				Kind:  memory.RecDelta,
				KeyA:  1,
				KeyB:  2,
				Value: 1,
			}
		}
		return AppendRecords(nil, recs)
	}
	next := func(wantLSN uint64) Stream {
		t.Helper()
		for {
			var s Stream
			select {
			case s = <-streams:
			case <-ctx.Done():
				t.Fatal("the follower did not dial")
			}
			m, err := s.Recv()
			require.NoError(t, err)
			h, ok := m.(Hello)
			require.True(t, ok, "first message is %T", m)
			require.Equal(t, "n1", h.Node)
			require.Equal(t, wantLSN, h.LastLSN)
			return s
		}
	}
	ack := func(s Stream) uint64 {
		t.Helper()
		m, err := s.Recv()
		require.NoError(t, err)
		return m.(Ack).LSN
	}

	s := next(0)
	require.NoError(t, s.Send(Resume{Term: 1}))
	require.NoError(t, s.Send(Batch{
		Term:     1,
		FirstLSN: 1,
		Count:    3,
		Data:     deltas(3),
	}))
	require.Equal(t, uint64(3), ack(s))
	require.NoError(t, s.Close())

	s = next(3)
	require.NoError(t, s.Send(Resume{Term: 1}))
	// a batch the follower has all of is skipped with no answer.
	require.NoError(t, s.Send(Batch{
		Term:     1,
		FirstLSN: 1,
		Count:    3,
		Data:     deltas(3),
	}))
	require.NoError(t, s.Send(Batch{
		Term:     1,
		FirstLSN: 2,
		Count:    4,
		Data:     deltas(4),
	}))
	require.Equal(t, uint64(5), ack(s))

	r := c.node("n1").replica(0)
	eventually(t, 5*time.Second, func() bool { return r.applied.Load() == 5 }, "the follower applied up to %d", r.applied.Load())
	require.Equal(t, int64(5), cellValue(c.node("n1").Replica(0), 1, 2), "records 2 and 3 were applied once")

	// the follower's own tail has records 1 to 5 once each.  the part it
	// kept of the second batch starts at record 4.
	first, count, data, ok := r.tail.read(1, 0)
	require.True(t, ok)
	require.Equal(t, uint64(1), first)
	require.Equal(t, 5, count)
	recs, err := DecodeRecords(nil, data)
	require.NoError(t, err)
	require.Len(t, recs, 5)
	first, count, _, ok = r.tail.read(4, 0)
	require.True(t, ok)
	require.Equal(t, uint64(4), first)
	require.Equal(t, 2, count)

	// a batch that says more records than it has is refused.  the follower
	// drops the stream and asks to leave the in-sync set.
	require.NoError(t, s.Send(Batch{
		Term:     1,
		FirstLSN: 6,
		Count:    3,
		Data:     deltas(2),
	}))
	_, err = s.Recv()
	require.Error(t, err)
	require.Equal(t, uint64(5), r.lastLSN.Load())
}

// transfersOf sums the state transfers the named nodes started.
func transfersOf(c *testCluster, names ...string) (n uint64) {
	for _, name := range names {
		if node := c.node(name); node != nil {
			n += node.Stats().Transfers
		}
	}
	return n
}

// TestIdleFailoverResumesWithoutATransfer lets a shard sit idle until every
// tail is empty, then kills the primary.  the follower that is not elected
// has every record, so the new primary resumes it with no state transfer.
func TestIdleFailoverResumesWithoutATransfer(t *testing.T) {
	window := 40 * time.Millisecond
	c := newNodeClusterCfg(t, 3, 1, 3, func(cfg *Config) { cfg.TailWindow = window })
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
	c.waitCaughtUp(t)

	row := c.Table().Shards[0]
	survivors := followersOf(row)
	eventually(t, 5*time.Second, func() bool {
		for _, name := range row.Replicas {
			if bytes, _ := c.node(name).replica(0).tail.size(); bytes != 0 {
				return false
			}
		}
		return true
	}, "the tails did not age out")
	before := transfersOf(c, survivors...)

	c.Kill(row.Primary)
	c.waitStable(t)
	next := c.Table().Shards[0]
	require.NotEqual(t, row.Primary, next.Primary)
	require.ElementsMatch(t, survivors, next.InSync)

	// a grant at level all needs the other follower's acknowledgement.
	_, err := acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
	require.NoError(t, err)
	for i, l := range leases {
		id, err := acct.extend(ctx, rt, l)
		require.NoError(t, err)
		require.NotNil(t, id, "lease %d is gone", i)
	}
	require.Equal(t, before, transfersOf(c, survivors...), "the follower took the state again, table %s", tableString(c))
}

// TestJoinUnderLoadTakesOneTransfer restarts a follower empty while the
// shard is under steady load.  it takes the state once, reaches the end of
// the log from the tail, and ends equal to the primary.
func TestJoinUnderLoadTakesOneTransfer(t *testing.T) {
	c := newNodeClusterAck(t, 3, 1, 3, 2*time.Second)
	rt := &testRouter{c: c}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var ops atomic.Int64
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for range 8 {
		acct := newTestAccount(t, c, 0, 1<<20)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				id, err := acct.acquire(ctx, rt, constraintapi.CommitLevelAll)
				if err != nil {
					return
				}
				// every fourth lease stays, so the state the follower
				// takes has live leases in it.
				if ops.Add(1)%4 != 0 {
					_ = acct.release(ctx, rt, id)
				}
			}
		}()
	}
	defer func() {
		select {
		case <-stop:
		default:
			close(stop)
		}
		wg.Wait()
	}()

	row := c.Table().Shards[0]
	follower := followersOf(row)[0]
	eventually(t, 10*time.Second, func() bool { return ops.Load() > 500 }, "the load did not start")
	before := transfersOf(c, row.Primary)
	startOps := ops.Load()

	c.Kill(follower)
	eventually(t, 10*time.Second, func() bool {
		return !slices.Contains(c.Table().Shards[0].InSync, follower)
	}, "%s did not leave the in-sync set", follower)
	require.NoError(t, c.Restart(follower))
	c.waitStable(t)
	require.Contains(t, c.Table().Shards[0].InSync, follower)
	require.Equal(t, row.Primary, c.Table().Shards[0].Primary)
	require.Greater(t, ops.Load(), startOps, "the load ran during the join")

	// let the load run on against the joined follower, then stop it.
	at := ops.Load()
	eventually(t, 10*time.Second, func() bool { return ops.Load() > at+200 }, "the load stopped after the join")
	close(stop)
	wg.Wait()

	require.Equal(t, before+1, transfersOf(c, row.Primary), "table %s", tableString(c))
	st := c.node(row.Primary).Stats()
	require.NotZero(t, st.StateRecords)
	require.NotZero(t, st.StateBytes)

	c.waitCaughtUp(t)
	want := canonState(t, c.node(row.Primary).Replica(0))
	require.NotEmpty(t, want)
	for _, name := range followersOf(row) {
		require.Equal(t, want, canonState(t, c.node(name).Replica(0)), "follower %s", name)
	}
}
