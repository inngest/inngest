package cluster

import (
	"context"
	"errors"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
)

// serverPipeline is how many answered frames of one client stream may wait
// for their commit.  when it is full the node stops reading the stream, and
// the transport pushes back on the client.
const serverPipeline = 64

// frameWait is one request of a frame that asked to wait for its commit.
type frameWait struct {
	idx   int
	shard int
}

// frameWork is one request frame on its way through a client stream.  the
// reader fills it and the committer waits on it and sends the answers.
// pend and ctxs are indexed by shard, so one failed shard fails only its
// own requests.  a ctxs entry points at the pend entry of the same shard
// and is reused with it.
type frameWork struct {
	resps []Response
	waits []frameWait
	used  []int
	pend  []memory.Pending
	ctxs  []context.Context
	seen  []bool
}

// ctx returns the context that collects the commit handles of one shard.
func (w *frameWork) ctx(base context.Context, shard int) context.Context {
	if !w.seen[shard] {
		w.seen[shard] = true
		w.used = append(w.used, shard)
	}
	if w.ctxs[shard] == nil {
		w.ctxs[shard] = memory.WithPending(base, &w.pend[shard])
	}
	return w.ctxs[shard]
}

// track counts a client stream handler in the node's wait group.  it is
// false once the node is closed.
func (n *Node) track() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return false
	}
	n.wg.Add(1)
	return true
}

// onTable answers one TableRequest and closes the stream.
func (n *Node) onTable(_ Message, s Stream) {
	_ = s.Send(TableReply{Table: n.table.Load()})
	_ = s.Close()
}

// onRequests serves one client stream.  this goroutine reads frames and
// runs their requests.  a second one waits for each frame's commit and
// sends the answers in order, so the reader never waits on a follower and
// frames pipeline.
func (n *Node) onRequests(first Message, s Stream) {
	if !n.track() {
		_ = s.Close()
		return
	}
	defer n.wg.Done()
	ctx, cancel := context.WithCancel(n.ctx)
	defer cancel()
	// the close ends a blocked Recv or Send when the node stops.
	stop := context.AfterFunc(ctx, func() { _ = s.Close() })
	defer stop()

	out := make(chan *frameWork, serverPipeline)
	free := make(chan *frameWork, serverPipeline)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for w := range out {
			for _, shard := range w.used {
				if err := w.pend[shard].Wait(ctx); err != nil {
					n.failWaits(w, shard, err)
				}
				w.seen[shard] = false
			}
			if err := s.Send(ResponseFrame{Resps: w.resps}); err != nil {
				// the reader sees the closed stream and ends.  the frames
				// still queued are waited on with a dead context.
				cancel()
			}
			w.resps, w.waits, w.used = nil, w.waits[:0], w.used[:0]
			select {
			case free <- w:
			default:
			}
		}
	}()

	m := first
	for {
		frame, ok := m.(RequestFrame)
		if !ok {
			break
		}
		tbl := n.table.Load()
		var w *frameWork
		select {
		case w = <-free:
		default:
			w = &frameWork{}
		}
		if len(w.pend) != len(tbl.Shards) {
			w.pend = make([]memory.Pending, len(tbl.Shards))
			w.ctxs = make([]context.Context, len(tbl.Shards))
			w.seen = make([]bool, len(tbl.Shards))
		}
		// the answers are a new slice per frame.  an in-process transport
		// passes the frame by reference, so the client still reads it after
		// the work is reused.
		w.resps = make([]Response, len(frame.Reqs))
		for i := range frame.Reqs {
			n.serve(ctx, tbl, w, i, &frame.Reqs[i])
		}
		out <- w
		var err error
		if m, err = s.Recv(); err != nil {
			break
		}
	}
	close(out)
	<-done
	_ = s.Close()
}

// serve runs one request and fills its answer.  a mutation goes to the open
// primary of the account's shard.  a read is also served by a follower.
func (n *Node) serve(base context.Context, tbl *Table, w *frameWork, idx int, q *Request) {
	p := &w.resps[idx]
	p.ID = q.ID
	account, ok := q.account()
	if !ok {
		p.Err = &ResponseError{Message: "cluster: request has no body for its op"}
		return
	}
	shard := tbl.ShardOf(account)
	mgr, err := n.Primary(shard)
	if err != nil && (q.Op == OpCheck || q.Op == OpGetCapacity) {
		mgr = n.Replica(shard)
	}
	if mgr == nil {
		p.Err = responseError(err)
		return
	}
	ctx := w.ctx(base, shard)

	waits := false
	switch q.Op {
	case OpAcquire:
		resp, ierr := mgr.Acquire(ctx, q.Acquire)
		if ierr != nil {
			p.Err = responseError(ierr)
			return
		}
		p.Acquire = resp
		waits = len(resp.Leases) > 0 && q.Acquire.CommitLevel != constraintapi.CommitLevelPrimary
	case OpExtend:
		resp, ierr := mgr.ExtendLease(ctx, q.Extend)
		if ierr != nil {
			p.Err = responseError(ierr)
			return
		}
		p.Extend = resp
		waits = resp.LeaseID != nil && q.Extend.CommitLevel != constraintapi.CommitLevelPrimary
	case OpRelease:
		resp, ierr := mgr.Release(ctx, q.Release)
		if ierr != nil {
			p.Err = responseError(ierr)
			return
		}
		p.Release = resp
		waits = q.Release.CommitLevel >= constraintapi.CommitLevelTwo
	case OpCheck:
		resp, uerr, ierr := mgr.Check(ctx, q.Check)
		switch {
		case ierr != nil:
			p.Err = responseError(ierr)
		case uerr != nil:
			p.Err = responseError(uerr)
			p.Err.User = true
		default:
			p.Check = resp
		}
	case OpSetCapacity:
		a := q.Semaphore
		res, err := mgr.SetCapacity(ctx, a.AccountID, a.Name, a.IdempotencyKey, a.Value)
		if err != nil {
			p.Err = responseError(err)
			return
		}
		p.Semaphore = &SemaphoreResult{
			Applied:  res.Applied,
			Capacity: res.Capacity,
		}
		waits = true
	case OpAdjustCapacity:
		a := q.Semaphore
		res, err := mgr.AdjustCapacity(ctx, a.AccountID, a.Name, a.IdempotencyKey, a.Value)
		if err != nil {
			p.Err = responseError(err)
			return
		}
		p.Semaphore = &SemaphoreResult{
			Applied:  res.Applied,
			Capacity: res.Capacity,
		}
		waits = true
	case OpGetCapacity:
		a := q.Semaphore
		capacity, usage, err := mgr.GetCapacity(ctx, a.AccountID, a.Name, a.EvaluatedKeyHash)
		if err != nil {
			p.Err = responseError(err)
			return
		}
		p.Semaphore = &SemaphoreResult{
			Capacity: capacity,
			Usage:    usage,
		}
	case OpReleaseSemaphore:
		a := q.Semaphore
		if err := mgr.ReleaseSemaphore(ctx, a.AccountID, a.Name, a.EvaluatedKeyHash, a.IdempotencyKey, a.Value); err != nil {
			p.Err = responseError(err)
			return
		}
		p.Semaphore = &SemaphoreResult{Applied: true}
		waits = true
	default:
		p.Err = &ResponseError{Message: "cluster: unknown op"}
	}
	if waits {
		w.waits = append(w.waits, frameWait{
			idx:   idx,
			shard: shard,
		})
	}
}

// failWaits replaces the answer of every request of the frame that waited
// on the shard.  the change is on this node and may be on a follower, so
// the error is retryable and the caller sends the request again.
func (n *Node) failWaits(w *frameWork, shard int, err error) {
	e := responseError(err)
	e.Retryable = true
	if e.NotPrimary == nil && errors.Is(err, ErrNotPrimary) {
		var np *NotPrimaryError
		if _, perr := n.Primary(shard); errors.As(perr, &np) {
			e.NotPrimary = np
		}
	}
	for _, fw := range w.waits {
		if fw.shard != shard {
			continue
		}
		w.resps[fw.idx] = Response{
			ID:  w.resps[fw.idx].ID,
			Err: e,
		}
	}
}

// responseError turns a server side error into its wire form.
func responseError(err error) *ResponseError {
	e := &ResponseError{Message: err.Error()}
	var r interface{ Retryable() bool }
	if errors.As(err, &r) {
		e.Retryable = r.Retryable()
	}
	var np *NotPrimaryError
	if errors.As(err, &np) {
		e.NotPrimary = np
		e.Retryable = true
	}
	return e
}
