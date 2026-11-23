package cluster

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

var testClientSeq atomic.Int64

// newTestClient returns a client on the cluster's network.  it has its own
// node name there, so a fault that isolates a node also cuts the client off
// from it.
func newTestClient(t testing.TB, c *testCluster) *Client {
	t.Helper()
	cl, err := NewClient(ClientConfig{
		Transport:    c.net.Transport(fmt.Sprintf("client-%d", testClientSeq.Add(1))),
		Seeds:        c.names,
		RetryBackoff: 5 * time.Millisecond,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cl.Close() })
	return cl
}

// reencode runs a message through the wire codec and returns the copy and
// both encodings.  equal encodings show the copy carries everything the
// wire carries, with no regard to nil against empty slices.
func reencode(t *testing.T, m Message) (Message, []byte, []byte) {
	t.Helper()
	first, err := EncodeMessage(nil, m)
	require.NoError(t, err)
	out, err := DecodeMessage(first)
	require.NoError(t, err)
	second, err := EncodeMessage(nil, out)
	require.NoError(t, err)
	return out, first, second
}

func TestClientCodecRoundTrip(t *testing.T) {
	at := time.Unix(1_700_000_000, 123_000_000).UTC()
	account, env, fn, app := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	lease := ulid.Make()
	items := ledgerConstraints()
	config := constraintapi.ConstraintConfig{
		FunctionVersion: 3,
		Concurrency: constraintapi.ConcurrencyConfig{
			FunctionConcurrency: 7,
		},
	}
	sem := &SemaphoreArgs{
		AccountID:        account,
		Name:             "app:x",
		EvaluatedKeyHash: "kh",
		IdempotencyKey:   "idem",
		Value:            -42,
	}

	reqs := RequestFrame{Reqs: []Request{
		{
			ID: 1,
			Op: OpAcquire,
			Acquire: &constraintapi.CapacityAcquireRequest{
				IdempotencyKey:       "acq",
				AccountID:            account,
				EnvID:                env,
				FunctionID:           fn,
				AppID:                app,
				Configuration:        config,
				Constraints:          items,
				Amount:               2,
				LeaseIdempotencyKeys: []string{"a", "b"},
				LeaseRunIDs:          map[string]ulid.ULID{"a": lease},
				CurrentTime:          at,
				Duration:             5 * time.Second,
				MaximumLifetime:      time.Hour,
				Source:               ledgerSource(),
				RequestAttempt:       2,
				CommitLevel:          constraintapi.CommitLevelTwo,
			},
		},
		{
			ID: 2,
			Op: OpExtend,
			Extend: &constraintapi.CapacityExtendLeaseRequest{
				IdempotencyKey: "ext",
				AccountID:      account,
				LeaseID:        lease,
				Duration:       3 * time.Second,
				Source:         ledgerSource(),
				LeaseIssuedAt:  at,
				CommitLevel:    constraintapi.CommitLevelAll,
			},
		},
		{
			ID: 3,
			Op: OpRelease,
			Release: &constraintapi.CapacityReleaseRequest{
				IdempotencyKey: "rel",
				AccountID:      account,
				LeaseID:        lease,
				Source:         ledgerSource(),
				CommitLevel:    constraintapi.CommitLevelPrimary,
			},
		},
		{
			ID: 4,
			Op: OpCheck,
			Check: &constraintapi.CapacityCheckRequest{
				AccountID:     account,
				EnvID:         env,
				FunctionID:    fn,
				Configuration: config,
				Constraints:   items,
			},
		},
		{
			ID:        5,
			Op:        OpSetCapacity,
			Semaphore: sem,
		},
		{
			ID:        6,
			Op:        OpAdjustCapacity,
			Semaphore: sem,
		},
		{
			ID:        7,
			Op:        OpGetCapacity,
			Semaphore: sem,
		},
		{
			ID:        1 << 40,
			Op:        OpReleaseSemaphore,
			Semaphore: sem,
		},
	}}
	out, first, second := reencode(t, reqs)
	require.Equal(t, first, second)
	got := out.(RequestFrame)
	require.Len(t, got.Reqs, len(reqs.Reqs))
	for i, q := range got.Reqs {
		require.Equal(t, reqs.Reqs[i].ID, q.ID)
		require.Equal(t, reqs.Reqs[i].Op, q.Op)
	}
	acq := got.Reqs[0].Acquire
	require.Equal(t, constraintapi.CommitLevelTwo, acq.CommitLevel)
	require.Equal(t, account, acq.AccountID)
	require.Equal(t, app, acq.AppID)
	require.Equal(t, []string{"a", "b"}, acq.LeaseIdempotencyKeys)
	require.Equal(t, lease, acq.LeaseRunIDs["a"])
	require.True(t, at.Equal(acq.CurrentTime))
	require.Equal(t, 5*time.Second, acq.Duration)
	require.Equal(t, 7, acq.Configuration.Concurrency.FunctionConcurrency)
	require.Equal(t, items, acq.Constraints)
	require.Equal(t, constraintapi.CommitLevelAll, got.Reqs[1].Extend.CommitLevel)
	require.Equal(t, lease, got.Reqs[1].Extend.LeaseID)
	require.Equal(t, constraintapi.CommitLevelPrimary, got.Reqs[2].Release.CommitLevel)
	require.Equal(t, lease, got.Reqs[2].Release.LeaseID)
	require.Equal(t, fn, got.Reqs[3].Check.FunctionID)
	for _, q := range got.Reqs[4:] {
		require.Equal(t, sem, q.Semaphore)
	}

	usage := []constraintapi.ConstraintUsage{{
		Constraint: items[0],
		Used:       2,
		Limit:      7,
	}}
	resps := ResponseFrame{Resps: []Response{
		{
			ID: 1,
			Acquire: &constraintapi.CapacityAcquireResponse{
				RequestID: lease,
				Leases: []constraintapi.CapacityLease{{
					LeaseID:        lease,
					IdempotencyKey: "a",
				}},
				LimitingConstraints:     items,
				ExhaustedConstraints:    items,
				Usage:                   usage,
				RetryAfter:              at,
				OperationIdempotencyHit: true,
			},
		},
		{
			ID: 2,
			Extend: &constraintapi.CapacityExtendLeaseResponse{
				LeaseID:    &lease,
				AccountID:  account,
				EnvID:      env,
				AppID:      app,
				FunctionID: fn,
				Usage:      usage,
			},
		},
		{
			ID: 3,
			Release: &constraintapi.CapacityReleaseResponse{
				AccountID:      account,
				EnvID:          env,
				AppID:          app,
				FunctionID:     fn,
				CreationSource: ledgerSource(),
				Usage:          usage,
			},
		},
		{
			ID: 4,
			Check: &constraintapi.CapacityCheckResponse{
				AvailableCapacity:    5,
				LimitingConstraints:  items,
				ExhaustedConstraints: items,
				Usage:                usage,
				RetryAfter:           at,
			},
		},
		{
			ID: 5,
			Semaphore: &SemaphoreResult{
				Applied:  true,
				Capacity: 9,
				Usage:    -3,
			},
		},
		{
			ID: 6,
			Err: &ResponseError{
				Message:   "busy",
				Retryable: true,
			},
		},
		{
			ID: 7,
			Err: &ResponseError{
				Message: "bad input",
				User:    true,
			},
		},
		{
			ID: 8,
			Err: &ResponseError{
				Message:   "not the primary",
				Retryable: true,
				NotPrimary: &NotPrimaryError{
					Shard:   3,
					Primary: "n1",
					Addr:    "n1:9000",
					Version: 77,
				},
			},
		},
	}}
	out, first, second = reencode(t, resps)
	require.Equal(t, first, second)
	back := out.(ResponseFrame)
	require.Len(t, back.Resps, len(resps.Resps))
	require.Equal(t, lease, back.Resps[0].Acquire.RequestID)
	require.Equal(t, resps.Resps[0].Acquire.Leases, back.Resps[0].Acquire.Leases)
	require.Equal(t, usage, back.Resps[0].Acquire.Usage)
	require.True(t, back.Resps[0].Acquire.OperationIdempotencyHit)
	require.True(t, at.Equal(back.Resps[0].Acquire.RetryAfter))
	require.Equal(t, lease, *back.Resps[1].Extend.LeaseID)
	require.Equal(t, fn, back.Resps[1].Extend.FunctionID)
	require.Equal(t, ledgerSource(), back.Resps[2].Release.CreationSource)
	require.Equal(t, usage, back.Resps[2].Release.Usage)
	require.Equal(t, 5, back.Resps[3].Check.AvailableCapacity)
	require.Equal(t, resps.Resps[4].Semaphore, back.Resps[4].Semaphore)
	for i := 5; i < 8; i++ {
		require.Equal(t, resps.Resps[i], back.Resps[i])
	}

	out, _, _ = reencode(t, TableRequest{})
	require.Equal(t, TableRequest{}, out)

	tbl := &Table{}
	require.NoError(t, tbl.Apply(Init{
		Shards: 4,
		RF:     2,
		Nodes:  []string{"a", "b", "c"},
	}))
	out, _, _ = reencode(t, TableReply{Table: tbl})
	require.Equal(t, tbl, out.(TableReply).Table)

	_, err := EncodeMessage(nil, RequestFrame{Reqs: []Request{{
		ID: 1,
		Op: OpAcquire,
	}}})
	require.Error(t, err)
	_, err = DecodeMessage(first[:len(first)/2])
	require.Error(t, err)
}

func TestClientRoutesAccountsToTheirShards(t *testing.T) {
	c := newNodeCluster(t, 3, 6, 3)
	cl := newTestClient(t, c)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	tbl := c.Table()
	seen := map[int]bool{}
	for i := 0; i < 60; i++ {
		a := &testAccount{
			account: uuid.New(),
			env:     uuid.New(),
			fn:      uuid.New(),
			limit:   3,
		}
		shard := tbl.ShardOf(a.account)
		seen[shard] = true
		req := a.acquireReq(constraintapi.CommitLevelAll)
		resp, err := cl.Acquire(ctx, req)
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)

		// the lease is on the primary of the account's shard, and the
		// primary of another shard has never seen the account.
		check := &constraintapi.CapacityCheckRequest{
			AccountID:     req.AccountID,
			EnvID:         req.EnvID,
			FunctionID:    req.FunctionID,
			Configuration: req.Configuration,
			Constraints:   req.Constraints,
		}
		for s := range tbl.Shards {
			mgr, perr := c.node(c.Table().Shards[s].Primary).Primary(s)
			require.NoError(t, perr)
			got, uerr, ierr := mgr.Check(ctx, check)
			require.NoError(t, uerr)
			require.NoError(t, ierr)
			want := 0
			if s == shard {
				want = 1
			}
			require.Equal(t, want, got.Usage[0].Used, "shard %d for an account of shard %d", s, shard)
		}

		// a check through the client reads the same shard.
		got, uerr, ierr := cl.Check(ctx, check)
		require.NoError(t, uerr)
		require.NoError(t, ierr)
		require.Equal(t, 1, got.Usage[0].Used)
	}
	require.Len(t, seen, 6, "60 accounts did not reach every shard")
}

func TestClientMatchesAnswersUnderPipelining(t *testing.T) {
	c := newNodeCluster(t, 3, 4, 3)
	cl := newTestClient(t, c)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const workers, perWorker = 64, 150
	var wg sync.WaitGroup
	errc := make(chan error, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a := &testAccount{
				account: uuid.New(),
				env:     uuid.New(),
				fn:      uuid.New(),
				limit:   perWorker,
			}
			for i := 0; i < perWorker; i++ {
				// grants at level primary answer at once, so a stream has
				// more than one frame in flight.
				level := constraintapi.CommitLevelPrimary
				if i%8 == 0 {
					level = constraintapi.CommitLevelAll
				}
				req := a.acquireReq(level)
				resp, err := cl.Acquire(ctx, req)
				if err != nil {
					errc <- err
					return
				}
				if len(resp.Leases) != 1 || resp.Leases[0].IdempotencyKey != req.LeaseIdempotencyKeys[0] {
					errc <- fmt.Errorf("request %s got the answer %+v", req.IdempotencyKey, resp.Leases)
					return
				}
				ext, err := cl.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
					IdempotencyKey: req.IdempotencyKey + "-extend",
					AccountID:      a.account,
					LeaseID:        resp.Leases[0].LeaseID,
					Duration:       30 * time.Second,
					Source:         ledgerSource(),
					CommitLevel:    constraintapi.CommitLevelPrimary,
				})
				if err != nil {
					errc <- err
					return
				}
				if ext.LeaseID == nil || ext.AccountID != a.account {
					errc <- fmt.Errorf("extend of %s got the answer %+v", req.IdempotencyKey, ext)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errc)
	for err := range errc {
		require.NoError(t, err)
	}
}

func TestClientRetriesAcrossAFailover(t *testing.T) {
	c := newNodeCluster(t, 3, 1, 3)
	cl := newTestClient(t, c)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	a := newTestAccount(t, c, 0, 10)
	first, err := cl.Acquire(ctx, a.acquireReq(constraintapi.CommitLevelAll))
	require.NoError(t, err)
	require.Len(t, first.Leases, 1)

	// the client still has the table that names the dead node.  the call
	// returns no error, the client finds the new primary itself.
	victim := c.Table().Shards[0].Primary
	c.Kill(victim)
	second, err := cl.Acquire(ctx, a.acquireReq(constraintapi.CommitLevelAll))
	require.NoError(t, err)
	require.Len(t, second.Leases, 1)
	require.NotEqual(t, victim, c.Table().Shards[0].Primary)

	// the lease answered before the kill is on the new primary.
	ext, err := cl.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
		IdempotencyKey: "extend-after-failover",
		AccountID:      a.account,
		LeaseID:        first.Leases[0].LeaseID,
		Duration:       30 * time.Second,
		Source:         ledgerSource(),
	})
	require.NoError(t, err)
	require.NotNil(t, ext.LeaseID)
}

func TestClientSemaphoreCalls(t *testing.T) {
	c := newNodeCluster(t, 3, 2, 3)
	cl := newTestClient(t, c)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	account := accountOn(t, c, 1)
	const name = "app:workers"

	set, err := cl.SetCapacity(ctx, account, name, "set-1", 3)
	require.NoError(t, err)
	require.Equal(t, constraintapi.SetResult{
		Applied:  true,
		Capacity: 3,
	}, set)
	set, err = cl.SetCapacity(ctx, account, name, "set-1", 3)
	require.NoError(t, err)
	require.False(t, set.Applied, "a replay applies nothing")

	adj, err := cl.AdjustCapacity(ctx, account, name, "adj-1", 2)
	require.NoError(t, err)
	require.Equal(t, constraintapi.AdjustResult{
		Applied:  true,
		Capacity: 5,
	}, adj)

	sem := constraintapi.SemaphoreConstraint{
		ID:      name,
		Weight:  2,
		Release: constraintapi.SemaphoreReleaseManual,
	}
	resp, err := cl.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
		IdempotencyKey: "sem-acquire",
		AccountID:      account,
		EnvID:          uuid.New(),
		FunctionID:     uuid.New(),
		Configuration: constraintapi.ConstraintConfig{
			FunctionVersion: 1,
			Semaphores:      []constraintapi.Semaphore{constraintapi.Semaphore(sem)},
		},
		Constraints: []constraintapi.ConstraintItem{{
			Kind:      constraintapi.ConstraintKindSemaphore,
			Semaphore: &sem,
		}},
		Amount:               1,
		LeaseIdempotencyKeys: []string{"sem-lease"},
		LeaseRunIDs:          map[string]ulid.ULID{},
		CurrentTime:          time.Now(),
		Duration:             30 * time.Second,
		MaximumLifetime:      time.Hour,
		Source:               ledgerSource(),
	})
	require.NoError(t, err)
	require.Len(t, resp.Leases, 1)

	capacity, usage, err := cl.GetCapacity(ctx, account, name, "")
	require.NoError(t, err)
	require.Equal(t, int64(5), capacity)
	require.Equal(t, int64(2), usage)

	require.NoError(t, cl.ReleaseSemaphore(ctx, account, name, "", "rel-1", 2))
	require.NoError(t, cl.ReleaseSemaphore(ctx, account, name, "", "rel-1", 2))
	_, usage, err = cl.GetCapacity(ctx, account, name, "")
	require.NoError(t, err)
	require.Equal(t, int64(0), usage)

	// the semaphore writes waited for every replica, so each one has them.
	c.waitCaughtUp(t)
	for _, replica := range c.Table().Shards[1].Replicas {
		got, _, gerr := c.node(replica).Replica(1).GetCapacity(ctx, account, name, "")
		require.NoError(t, gerr)
		require.Equal(t, int64(5), got, "capacity on %s", replica)
	}
}

// frameCall sends one frame on a new stream to a node and returns the
// answer frame.
func frameCall(t *testing.T, tr Transport, addr string, f RequestFrame) ResponseFrame {
	t.Helper()
	s, err := tr.Dial(context.Background(), addr)
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.Send(f))
	m, err := s.Recv()
	require.NoError(t, err)
	return m.(ResponseFrame)
}

func TestServerAnswersAMixedFrame(t *testing.T) {
	c := newNodeCluster(t, 3, 3, 3)
	tbl := c.Table()
	node := tbl.Shards[0].Primary
	other := -1
	for s, row := range tbl.Shards {
		if row.Primary != node {
			other = s
		}
	}
	require.GreaterOrEqual(t, other, 0, "one node leads every shard")

	mine := newTestAccount(t, c, 0, 5)
	theirs := newTestAccount(t, c, other, 5)
	check := func(a *testAccount) *constraintapi.CapacityCheckRequest {
		req := a.acquireReq(constraintapi.CommitLevelAll)
		return &constraintapi.CapacityCheckRequest{
			AccountID:     req.AccountID,
			EnvID:         req.EnvID,
			FunctionID:    req.FunctionID,
			Configuration: req.Configuration,
			Constraints:   req.Constraints,
		}
	}
	got := frameCall(t, c.net.Transport("frame-client"), node, RequestFrame{Reqs: []Request{
		{
			ID:      10,
			Op:      OpAcquire,
			Acquire: theirs.acquireReq(constraintapi.CommitLevelAll),
		},
		{
			ID:      11,
			Op:      OpAcquire,
			Acquire: mine.acquireReq(constraintapi.CommitLevelAll),
		},
		{
			ID:    12,
			Op:    OpCheck,
			Check: check(theirs),
		},
		{
			ID: 13,
			Op: OpAcquire,
		},
	}})
	require.Len(t, got.Resps, 4)

	// the shard this node does not lead is refused with the primary named.
	require.Equal(t, uint64(10), got.Resps[0].ID)
	require.NotNil(t, got.Resps[0].Err)
	require.True(t, got.Resps[0].Err.Retryable)
	np := got.Resps[0].Err.NotPrimary
	require.NotNil(t, np)
	require.Equal(t, other, np.Shard)
	require.Equal(t, tbl.Shards[other].Primary, np.Primary)
	require.Equal(t, tbl.Nodes[np.Primary].Addr, np.Addr)

	// the shard it leads is served in the same frame.
	require.Equal(t, uint64(11), got.Resps[1].ID)
	require.Nil(t, got.Resps[1].Err)
	require.Len(t, got.Resps[1].Acquire.Leases, 1)

	// a read of the other shard is served from this node's follower copy.
	require.Equal(t, uint64(12), got.Resps[2].ID)
	require.Nil(t, got.Resps[2].Err)
	require.NotNil(t, got.Resps[2].Check)

	// a request with no body fails alone.
	require.Equal(t, uint64(13), got.Resps[3].ID)
	require.NotNil(t, got.Resps[3].Err)
	require.False(t, got.Resps[3].Err.Retryable)
}

// stalledPrimary returns a one shard cluster whose followers have stopped
// acknowledging and stay in the in-sync set, and a stream to its primary.
// a request that waits for a follower is not answered there.
func stalledPrimary(t *testing.T) (*testCluster, ShardRow, *testAccount, Stream) {
	t.Helper()
	c := newNodeClusterAck(t, 3, 1, 3, 10*time.Second)
	row := c.Table().Shards[0]
	a := newTestAccount(t, c, 0, 5)
	for _, f := range followersOf(row) {
		c.net.Stall(row.Primary, f)
	}
	s, err := c.net.Transport("frame-client").Dial(context.Background(), row.Primary)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return c, row, a, s
}

// answersOf receives every message of a stream on a channel.
func answersOf(s Stream) <-chan Message {
	ch := make(chan Message, 16)
	go func() {
		defer close(ch)
		for {
			m, err := s.Recv()
			if err != nil {
				return
			}
			ch <- m
		}
	}()
	return ch
}

func TestServerRunsLaterFramesWhileOneWaits(t *testing.T) {
	c, row, a, s := stalledPrimary(t)
	waits := a.acquireReq(constraintapi.CommitLevelAll)
	require.NoError(t, s.Send(RequestFrame{Reqs: []Request{{
		ID:      1,
		Op:      OpAcquire,
		Acquire: waits,
	}}}))
	require.NoError(t, s.Send(RequestFrame{Reqs: []Request{{
		ID:      2,
		Op:      OpAcquire,
		Acquire: a.acquireReq(constraintapi.CommitLevelPrimary),
	}}}))

	// the second frame is read and run while the first one waits.
	mgr, perr := c.node(row.Primary).Primary(0)
	require.NoError(t, perr)
	eventually(t, 5*time.Second, func() bool {
		got, uerr, ierr := mgr.Check(context.Background(), &constraintapi.CapacityCheckRequest{
			AccountID:     waits.AccountID,
			EnvID:         waits.EnvID,
			FunctionID:    waits.FunctionID,
			Configuration: waits.Configuration,
			Constraints:   waits.Constraints,
		})
		return uerr == nil && ierr == nil && got.Usage[0].Used == 2
	}, "the second frame was not run while the first waited")

	// neither is answered.  answers keep the order of the frames, so the
	// second one waits behind the first.
	answers := answersOf(s)
	select {
	case m := <-answers:
		t.Fatalf("answered %+v with no follower acknowledgement", m)
	case <-time.After(100 * time.Millisecond):
	}

	for _, f := range followersOf(row) {
		c.net.Unstall(row.Primary, f)
	}
	for id := uint64(1); id <= 2; id++ {
		m, ok := <-answers
		require.True(t, ok)
		got := m.(ResponseFrame)
		require.Len(t, got.Resps, 1)
		require.Equal(t, id, got.Resps[0].ID)
		require.Nil(t, got.Resps[0].Err)
		require.Len(t, got.Resps[0].Acquire.Leases, 1)
	}
}

func TestServerFailsAWaitWhenThePrimaryLosesItsSeat(t *testing.T) {
	c, row, a, s := stalledPrimary(t)
	require.NoError(t, s.Send(RequestFrame{Reqs: []Request{
		{
			ID:      1,
			Op:      OpAcquire,
			Acquire: a.acquireReq(constraintapi.CommitLevelAll),
		},
		{
			ID:      2,
			Op:      OpAcquire,
			Acquire: a.acquireReq(constraintapi.CommitLevelPrimary),
		},
	}}))
	answers := answersOf(s)
	select {
	case m := <-answers:
		t.Fatalf("answered %+v with no follower acknowledgement", m)
	case <-time.After(100 * time.Millisecond):
	}

	// the table gives the seat to a follower.  the old primary can never
	// get the acknowledgement now, so it fails the request that waited.
	next := followersOf(row)[0]
	require.NoError(t, c.co.Node(row.Primary).Propose(context.Background(), ElectPrimary{
		Shard: 0,
		Epoch: c.Table().Shards[0].Epoch,
		Node:  next,
	}))
	var got ResponseFrame
	select {
	case m, ok := <-answers:
		require.True(t, ok)
		got = m.(ResponseFrame)
	case <-time.After(10 * time.Second):
		t.Fatal("the waiting frame was never answered")
	}
	require.Len(t, got.Resps, 2)
	require.NotNil(t, got.Resps[0].Err)
	require.True(t, got.Resps[0].Err.Retryable)
	require.NotNil(t, got.Resps[0].Err.NotPrimary)
	require.Equal(t, next, got.Resps[0].Err.NotPrimary.Primary)

	// the request of the same frame that did not ask to wait keeps its
	// answer.
	require.Nil(t, got.Resps[1].Err)
	require.Len(t, got.Resps[1].Acquire.Leases, 1)
}

func TestClientErrorMapping(t *testing.T) {
	c := newNodeCluster(t, 3, 1, 3)
	cl := newTestClient(t, c)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// the store refuses an invalid request.  that is not retryable.
	a := newTestAccount(t, c, 0, 5)
	bad := a.acquireReq(constraintapi.CommitLevelAll)
	bad.Amount = 0
	_, err := cl.Acquire(ctx, bad)
	require.Error(t, err)
	require.False(t, err.Retryable())
	require.Contains(t, err.Error(), "invalid request")

	_, uerr, ierr := cl.Check(ctx, &constraintapi.CapacityCheckRequest{AccountID: a.account})
	require.NoError(t, uerr)
	require.Error(t, ierr)
	require.False(t, ierr.Retryable())

	// no node answers.  the call ends with its context and is retryable.
	for _, name := range c.names {
		c.Kill(name)
	}
	short, done := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer done()
	_, err = cl.Acquire(short, a.acquireReq(constraintapi.CommitLevelAll))
	require.Error(t, err)
	require.True(t, err.Retryable())
	require.ErrorIs(t, err, context.DeadlineExceeded)

	// a closed client is retryable too, so a caller with a new client can
	// send the request again.
	require.NoError(t, cl.Close())
	_, err = cl.Acquire(ctx, a.acquireReq(constraintapi.CommitLevelAll))
	require.Error(t, err)
	require.True(t, err.Retryable())
}

// countingTransport counts the request frames, and the requests in them,
// that go out on the streams it dials.
type countingTransport struct {
	Transport
	frames atomic.Int64
	reqs   atomic.Int64
}

func (ct *countingTransport) Dial(ctx context.Context, addr string) (Stream, error) {
	s, err := ct.Transport.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &countingStream{
		Stream: s,
		ct:     ct,
	}, nil
}

type countingStream struct {
	Stream
	ct *countingTransport
}

func (cs *countingStream) Send(m Message) error {
	if f, ok := m.(RequestFrame); ok {
		cs.ct.frames.Add(1)
		cs.ct.reqs.Add(int64(len(f.Reqs)))
	}
	return cs.Stream.Send(m)
}

// startAnswerNode listens at name on the network.  it answers a table
// request with a table of one shard whose primary is name, and every request
// frame with one empty acquire answer per request.  with a gate it answers no
// request frame before the gate is closed.  it has no store, no timer and no
// other goroutine, so a test can run it on the clock of synctest.
func startAnswerNode(t *testing.T, net *MemNetwork, name string, gate chan struct{}) {
	t.Helper()
	tbl := &Table{}
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     name,
		Addr:   name,
		BootID: "boot",
	}))
	require.NoError(t, tbl.Apply(Init{
		Shards: 1,
		RF:     1,
		Nodes:  []string{name},
	}))
	ln, err := net.Transport(name).Listen(name, func(s Stream) {
		defer s.Close()
		for {
			m, err := s.Recv()
			if err != nil {
				return
			}
			switch f := m.(type) {
			case TableRequest:
				_ = s.Send(TableReply{Table: tbl})
				return
			case RequestFrame:
				if gate != nil {
					<-gate
				}
				resps := make([]Response, len(f.Reqs))
				for i := range f.Reqs {
					resps[i] = Response{
						ID:      f.Reqs[i].ID,
						Acquire: &constraintapi.CapacityAcquireResponse{},
					}
				}
				if err := s.Send(ResponseFrame{Resps: resps}); err != nil {
					return
				}
			}
		}
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
}

func TestClientFramesCarryManyRequestsUnderLoad(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const streams, inFlight, callers = 2, 2, 200
		net := NewMemNetwork()
		gate := make(chan struct{})
		startAnswerNode(t, net, "n0", gate)
		tr := &countingTransport{Transport: net.Transport("client")}
		cl, err := NewClient(ClientConfig{
			Transport:      tr,
			Seeds:          []string{"n0"},
			StreamsPerNode: streams,
			MaxInFlight:    inFlight,
		})
		require.NoError(t, err)
		defer cl.Close()

		var wg sync.WaitGroup
		errc := make(chan error, callers)
		for range callers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := cl.Acquire(context.Background(), &constraintapi.CapacityAcquireRequest{AccountID: uuid.New()}); err != nil {
					errc <- err
				}
			}()
		}

		// Wait returns when every caller is queued and waits for its answer.
		// the node has answered nothing.  the checks run after the gate is
		// open, so a failed check leaves no goroutine blocked.
		synctest.Wait()
		unanswered := tr.frames.Load()
		close(gate)
		wg.Wait()
		close(errc)
		for err := range errc {
			require.NoError(t, err)
		}

		// no stream sent more frames than it may have in flight.
		require.LessOrEqual(t, unanswered, int64(streams*inFlight))
		// the first answer on a stream lets its writer send one more frame.
		// that frame carries every request that queued in the meantime.
		frames, reqs := tr.frames.Load(), tr.reqs.Load()
		require.Equal(t, int64(callers), reqs)
		require.LessOrEqual(t, frames, int64(streams*(inFlight+1)))
		require.Greater(t, float64(reqs)/float64(frames), 1.0)
	})
}

func TestClientAnswersALoneCallerAfterOneRoundTrip(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const calls = 100
		net := NewMemNetwork()
		startAnswerNode(t, net, "n0", nil)
		tr := &countingTransport{Transport: net.Transport("client")}
		cl, err := NewClient(ClientConfig{
			Transport: tr,
			Seeds:     []string{"n0"},
		})
		require.NoError(t, err)
		defer cl.Close()

		ctx := context.Background()
		req := &constraintapi.CapacityAcquireRequest{AccountID: uuid.New()}
		// the first call fetches the table and dials a stream.
		_, err = cl.Acquire(ctx, req)
		require.NoError(t, err)

		frames, start := tr.frames.Load(), time.Now()
		for range calls {
			_, err = cl.Acquire(ctx, req)
			require.NoError(t, err)
		}
		// the clock of synctest moves only while every goroutine waits.  a
		// call that waited on a timer moves it.  a round trip on MemNetwork
		// does not.
		require.Zero(t, time.Since(start))
		// every request left at once, alone in its frame.
		require.Equal(t, int64(calls), tr.frames.Load()-frames)
		require.Equal(t, int64(calls+1), tr.reqs.Load())
	})
}

func TestClientConfigNeedsASeed(t *testing.T) {
	_, err := NewClient(ClientConfig{Transport: NewMemNetwork().Transport("x")})
	require.Error(t, err)
	_, err = NewClient(ClientConfig{Seeds: []string{"n0"}})
	require.Error(t, err)
}
