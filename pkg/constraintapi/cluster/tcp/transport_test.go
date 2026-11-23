package tcp

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"slices"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/cluster"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// listen starts a listener on a free loopback port and returns its address
// and the streams it accepts.
func listen(t testing.TB) (*Listener, <-chan cluster.Stream) {
	t.Helper()
	accepted := make(chan cluster.Stream, 16)
	ln, err := New().Listen("127.0.0.1:0", func(s cluster.Stream) { accepted <- s })
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	return ln.(*Listener), accepted
}

// connect returns both ends of one new stream.
func connect(t testing.TB) (client, server cluster.Stream) {
	t.Helper()
	ln, accepted := listen(t)
	client, err := New().Dial(context.Background(), ln.Addr().String())
	require.NoError(t, err)
	server = soon(t, accepted)
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})
	return client, server
}

func soon[T any](t testing.TB, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(10 * time.Second):
		t.Fatal("got no value")
	}
	panic("unreachable")
}

type recvResult struct {
	m   cluster.Message
	err error
}

func recvAsync(s cluster.Stream) <-chan recvResult {
	out := make(chan recvResult, 1)
	go func() {
		m, err := s.Recv()
		out <- recvResult{
			m:   m,
			err: err,
		}
	}()
	return out
}

func TestStreamOrderUnderConcurrentSenders(t *testing.T) {
	client, server := connect(t)

	// every sender counts up in the high bits of its own LSNs.  the receiver
	// must see each sender's messages whole and in order.
	const senders, each = 8, 2000
	var wg sync.WaitGroup
	for id := range senders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range each {
				m := cluster.Message(cluster.Ack{LSN: uint64(id)<<32 | uint64(i)})
				if i%100 == 0 {
					m = cluster.Hello{
						Node:    fmt.Sprintf("sender %d", id),
						Term:    uint64(id),
						LastLSN: uint64(i),
					}
				}
				if err := client.Send(m); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	next := make([]uint64, senders)
	for range senders * each {
		m, err := server.Recv()
		require.NoError(t, err)
		var id, i uint64
		switch m := m.(type) {
		case cluster.Ack:
			id, i = m.LSN>>32, m.LSN&(1<<32-1)
		case cluster.Hello:
			id, i = m.Term, m.LastLSN
			require.Equal(t, fmt.Sprintf("sender %d", id), m.Node)
		default:
			t.Fatalf("got %T", m)
		}
		require.Equal(t, next[id], i)
		next[id]++
	}
	wg.Wait()

	// the other direction works on the same stream.
	require.NoError(t, server.Send(cluster.Resume{Term: 9}))
	m, err := client.Recv()
	require.NoError(t, err)
	require.Equal(t, cluster.Resume{Term: 9}, m)
}

func TestStreamLargeFrame(t *testing.T) {
	client, server := connect(t)
	rng := rand.New(rand.NewSource(7))
	big := cluster.Batch{
		Term:     1,
		FirstLSN: 2,
		Count:    200_000,
		Data:     cluster.AppendRecords(nil, mixedBatch(rng, 200_000)),
	}
	small := cluster.Batch{
		Term:     1,
		FirstLSN: 200_002,
		Count:    10,
		Data:     cluster.AppendRecords(nil, mixedBatch(rng, 10)),
	}
	go func() {
		_ = client.Send(big)
		_ = client.Send(small)
	}()
	for _, want := range []cluster.Batch{big, small} {
		m, err := server.Recv()
		require.NoError(t, err)
		require.Equal(t, want, m)
	}
}

func TestStreamRefusesFrameAboveLimit(t *testing.T) {
	ln, accepted := listen(t)
	conn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	defer conn.Close()
	server := soon(t, accepted)
	defer server.Close()

	_, err = conn.Write(binary.BigEndian.AppendUint32(nil, maxFrame+1))
	require.NoError(t, err)
	_, err = server.Recv()
	require.ErrorIs(t, err, cluster.ErrStreamBroken)
	require.ErrorContains(t, err, "above the limit")
	require.Error(t, server.Send(cluster.Ack{}))
}

func TestStreamBadMessageBreaksStream(t *testing.T) {
	client, server := connect(t)

	// a type with no codec ends the stream on the sending end.
	require.Error(t, client.Send("no codec"))
	require.Error(t, client.Send(cluster.Ack{}))
	_, err := server.Recv()
	require.ErrorIs(t, err, cluster.ErrStreamClosed)
}

func TestStreamCloseUnblocksRecvAndSend(t *testing.T) {
	client, server := connect(t)

	got := recvAsync(client)
	select {
	case <-got:
		t.Fatal("Recv returned with nothing sent")
	case <-time.After(50 * time.Millisecond):
	}
	require.NoError(t, client.Close())
	require.ErrorIs(t, soon(t, got).err, cluster.ErrStreamClosed)
	require.ErrorIs(t, client.Send(cluster.Ack{}), cluster.ErrStreamClosed)
	_, err := client.Recv()
	require.ErrorIs(t, err, cluster.ErrStreamClosed)
	require.NoError(t, client.Close())

	// the peer does not read, so Send fills the socket buffers and blocks.
	_, server = connect(t)
	batch := cluster.Batch{
		Count: benchBatch,
		Data:  cluster.AppendRecords(nil, mixedBatch(rand.New(rand.NewSource(8)), benchBatch)),
	}
	sent := make(chan int, 1024)
	sendErr := make(chan error, 1)
	go func() {
		for i := 0; ; i++ {
			if err := server.Send(batch); err != nil {
				sendErr <- err
				return
			}
			select {
			case sent <- i:
			default:
			}
		}
	}()
	for blocked := false; !blocked; {
		select {
		case <-sent:
		case err := <-sendErr:
			t.Fatalf("Send failed before the Close, %v", err)
		case <-time.After(200 * time.Millisecond):
			blocked = true
		}
	}
	require.NoError(t, server.Close())
	require.ErrorIs(t, soon(t, sendErr), cluster.ErrStreamClosed)
}

func TestStreamPeerCloseDeliversThenErrors(t *testing.T) {
	client, server := connect(t)
	const n = 500
	for i := range n {
		require.NoError(t, client.Send(cluster.Ack{LSN: uint64(i)}))
	}
	require.NoError(t, client.Close())

	for i := range n {
		m, err := server.Recv()
		require.NoError(t, err)
		require.Equal(t, cluster.Ack{LSN: uint64(i)}, m)
	}
	_, err := server.Recv()
	require.ErrorIs(t, err, cluster.ErrStreamClosed)
	_, err = server.Recv()
	require.ErrorIs(t, err, cluster.ErrStreamClosed)
	require.ErrorIs(t, server.Send(cluster.Ack{}), cluster.ErrStreamClosed)
}

func TestListenDialAndListenAgain(t *testing.T) {
	tr := New()
	ln, accepted := listen(t)
	addr := ln.Addr().String()

	open, err := tr.Dial(context.Background(), addr)
	require.NoError(t, err)
	defer open.Close()
	server := soon(t, accepted)
	defer server.Close()

	_, err = tr.Listen(addr, func(cluster.Stream) {})
	require.Error(t, err, "the address is in use")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = tr.Dial(ctx, addr)
	require.ErrorIs(t, err, context.Canceled)

	// a closed listener refuses dials and leaves open streams open.
	require.NoError(t, ln.Close())
	_, err = tr.Dial(context.Background(), addr)
	require.ErrorIs(t, err, cluster.ErrDialRefused)
	require.NoError(t, open.Send(cluster.Ack{LSN: 1}))
	m, err := server.Recv()
	require.NoError(t, err)
	require.Equal(t, cluster.Ack{LSN: 1}, m)

	again := make(chan cluster.Stream, 1)
	ln2, err := tr.Listen(addr, func(s cluster.Stream) { again <- s })
	require.NoError(t, err)
	defer ln2.Close()
	client, err := tr.Dial(context.Background(), addr)
	require.NoError(t, err)
	defer client.Close()
	server2 := soon(t, again)
	defer server2.Close()
	require.NoError(t, client.Send(cluster.Ack{LSN: 2}))
	m, err = server2.Recv()
	require.NoError(t, err)
	require.Equal(t, cluster.Ack{LSN: 2}, m)
}

// canonState is the state of a manager as sorted strings, in the form two
// replicas of one shard must agree on.  Export emits no acquire or extend
// idempotency record, so they are not part of it.
func canonState(t testing.TB, m *memory.Manager) []string {
	t.Helper()
	if m == nil {
		return nil
	}
	var recs []memory.Record
	m.Freeze(func() {
		m.Export(func(r memory.Record) { recs = append(recs, r) })
	})
	var out []string
	for _, r := range recs {
		switch r.Kind {
		case memory.RecSet:
			def, err := json.Marshal(r.Set)
			require.NoError(t, err)
			out = append(out, fmt.Sprintf("set %d %s", r.SetID, def))
		case memory.RecGrant:
			out = append(out, fmt.Sprintf("lease %v set %d exp %d req %d granted %d", r.Seqs, r.SetID, r.ExpiresAtMS, r.Requested, r.Granted))
		case memory.RecCell:
			if r.Value != 0 {
				out = append(out, fmt.Sprintf("cell %x.%x %d", r.KeyA, r.KeyB, r.Value))
			}
		}
	}
	sort.Strings(out)
	return out
}

func TestNodesReplicateOverTCP(t *testing.T) {
	const ackTimeout = 2 * time.Second
	names := []string{"a", "b", "c"}
	co := cluster.NewMemCoordinator(20*time.Millisecond, ackTimeout)
	t.Cleanup(func() { _ = co.Close() })
	tr := New()
	nodes := map[string]*cluster.Node{}
	for _, name := range names {
		// the listener starts first because the node needs the real port for
		// the address it registers.
		mux := cluster.NewMux()
		ln, err := tr.Listen("127.0.0.1:0", mux.Accept)
		require.NoError(t, err)
		node, err := cluster.NewNode(cluster.Config{
			Name:              name,
			Addr:              ln.(*Listener).Addr().String(),
			Transport:         tr,
			Coordinator:       co.Node(name),
			Mux:               mux,
			AckTimeout:        ackTimeout,
			HeartbeatInterval: 20 * time.Millisecond,
			DialBackoff:       5 * time.Millisecond,
		})
		require.NoError(t, err)
		nodes[name] = node
		t.Cleanup(func() {
			_ = node.Close()
			_ = ln.Close()
		})
	}
	ctx := context.Background()
	require.NoError(t, co.Node("test").Propose(ctx, cluster.Init{
		Shards: 1,
		RF:     3,
		Nodes:  names,
	}))

	// the shard is ready when its primary is open and both followers are in
	// the in-sync set.
	var primary *memory.Manager
	var row cluster.ShardRow
	require.Eventually(t, func() bool {
		tbl, _ := co.Node("test").Table()
		if len(tbl.Shards) != 1 || len(tbl.Shards[0].InSync) != 3 {
			return false
		}
		row = tbl.Shards[0]
		mgr, err := nodes[row.Primary].Primary(0)
		primary = mgr
		return err == nil
	}, 15*time.Second, 5*time.Millisecond)

	account, env, fn := uuid.New(), uuid.New(), uuid.New()
	for i := range 20 {
		idem := fmt.Sprintf("grant-%d", i)
		resp, err := primary.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			IdempotencyKey: idem + "-acquire",
			AccountID:      account,
			EnvID:          env,
			FunctionID:     fn,
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
			LeaseIdempotencyKeys: []string{idem},
			LeaseRunIDs:          map[string]ulid.ULID{},
			CurrentTime:          time.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Hour,
			Source: constraintapi.LeaseSource{
				Service:  constraintapi.ServiceExecutor,
				Location: constraintapi.CallerLocationItemLease,
			},
			CommitLevel: constraintapi.CommitLevelAll,
		})
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)
	}

	want := canonState(t, nodes[row.Primary].Replica(0))
	require.Len(t, want, 22, "one set, twenty leases and one usage counter")
	for _, name := range row.Replicas {
		if name == row.Primary {
			continue
		}
		require.Eventually(t, func() bool {
			return slices.Equal(want, canonState(t, nodes[name].Replica(0)))
		}, 5*time.Second, 5*time.Millisecond, "follower %s does not match the primary", name)
	}
}

// BenchmarkStreamBatches sends batches of records over loopback the way a
// primary and a follower do.  the sender encodes each batch once and the
// receiver decodes it into a buffer it uses again.
func BenchmarkStreamBatches(b *testing.B) {
	client, server := connect(b)
	recs := mixedBatch(rand.New(rand.NewSource(6)), benchBatch)
	done := make(chan error, 1)
	b.ReportAllocs()
	b.ResetTimer()
	go func() {
		var buf []memory.Record
		for range b.N {
			m, err := server.Recv()
			if err == nil {
				buf, err = cluster.DecodeRecords(buf[:0], m.(cluster.Batch).Data)
			}
			if err == nil && len(buf) != benchBatch {
				err = fmt.Errorf("got a batch of %d records", len(buf))
			}
			if err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	var scratch []byte
	for range b.N {
		scratch = cluster.AppendRecords(scratch[:0], recs)
		err := client.Send(cluster.Batch{
			Term:     1,
			FirstLSN: 1,
			Count:    benchBatch,
			Data:     append([]byte(nil), scratch...),
		})
		if err != nil {
			b.Fatal(err)
		}
	}
	if err := <-done; err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(b.N*benchBatch)/b.Elapsed().Seconds(), "records/s")
}

const benchBatch = 1000

// mixedBatch is n records in the mix one acquire and one release produce.
// each lease has a grant, a take and two TAT states.
func mixedBatch(rng *rand.Rand, n int) []memory.Record {
	now := time.Now().UnixMilli()
	recs := make([]memory.Record, 0, n)
	for i := 0; len(recs) < n; i++ {
		seq := uint64(i%16)<<44 | uint64(1_000_000+i)
		tat := memory.Record{
			Kind:        memory.RecTAT,
			KeyA:        rng.Uint64(),
			KeyB:        rng.Uint64(),
			TAT:         float64(now) * 1e6,
			ExpiresAtMS: now + 60_000,
			Version:     uint64(now) << 20,
		}
		recs = append(recs, tat, memory.Record{
			Kind:            memory.RecGrant,
			SetID:           uint64(rng.Intn(5000)),
			Seqs:            []uint64{seq},
			ExpiresAtMS:     now + 30_000,
			Requested:       1,
			Granted:         1,
			MaximumLifetime: time.Hour,
			Source: constraintapi.LeaseSource{
				Service:  constraintapi.ServiceExecutor,
				Location: constraintapi.CallerLocationItemLease,
			},
			KeyA:  rng.Uint64(),
			Value: now + 5_000,
		}, tat, memory.Record{
			Kind: memory.RecTake,
			Seq:  seq,
		})
	}
	return recs[:n]
}
