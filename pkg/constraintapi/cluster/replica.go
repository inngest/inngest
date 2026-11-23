package cluster

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi/memory"
)

type replicaMode int

const (
	// modeFollower has a manager that only applies records.
	modeFollower replicaMode = iota
	// modePrimary is the open primary.  its replicator runs.
	modePrimary
	// modeDeposed has a manager that was a primary and lost its seat.  it
	// keeps its state until the table removes it from the in-sync set,
	// because until then the table can elect it again.
	modeDeposed
	// modeBroken has a follower manager that no longer matches the log.  it
	// leaves the in-sync set and then takes the whole state again.
	modeBroken
)

// replica is this node's copy of one shard.  one goroutine moves it to the
// role the table gives it.  a Fence from a new primary can stop it at any
// time.
type replica struct {
	n     *Node
	shard int

	mu      sync.Mutex
	row     ShardRow
	version uint64
	rowCh   chan struct{}
	// term is the highest term seen, from the table or from a Fence.
	term          uint64
	mode          replicaMode
	nonce         uint32
	fol           *follower
	rep           *replicator
	primaryTerm   uint64
	promoteCancel context.CancelFunc
	// transferring is true between StateBegin and StateEnd.
	transferring bool
	selfShrinkAt time.Time
	closed       bool

	// applyMu is taken by whoever changes the manager, the tail and lastLSN
	// together outside the follower goroutine.
	applyMu sync.Mutex
	mgr     atomic.Pointer[memory.Manager]
	open    atomic.Bool
	// lastLSN is the last record this replica has.  a follower counts what
	// it received and a primary what it numbered.
	lastLSN atomic.Uint64
	// applied is the last record a follower has applied.  it trails lastLSN
	// by the batch in hand.
	applied atomic.Uint64
	tail    *tail

	kick chan struct{}
}

func newReplica(n *Node, shard int, version uint64, row ShardRow) (*replica, error) {
	r := &replica{
		n:       n,
		shard:   shard,
		row:     row,
		version: version,
		rowCh:   make(chan struct{}),
		nonce:   row.Nonce,
		tail:    newTail(n.cfg.TailBytes, n.cfg.TailWindow),
		kick:    make(chan struct{}, 1),
	}
	mgr, err := n.newManager(shard, row.Nonce)
	if err != nil {
		return nil, err
	}
	r.mgr.Store(mgr)
	return r, nil
}

// setRow gives the replica a newer row of its shard.
func (r *replica) setRow(version uint64, row ShardRow) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if version <= r.version {
		return
	}
	r.version = version
	r.row = row
	close(r.rowCh)
	r.rowCh = make(chan struct{})
}

func (r *replica) current() (ShardRow, <-chan struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.row, r.rowCh
}

func (r *replica) highestTerm() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.term
}

func (r *replica) modeIs(m replicaMode) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.mode == m
}

func (r *replica) inTransfer() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.transferring
}

func (r *replica) nudge() {
	select {
	case r.kick <- struct{}{}:
	default:
	}
}

// wait returns when the row changes, after one dial backoff, or when ctx
// ends.
func (r *replica) wait(ctx context.Context, ch <-chan struct{}) {
	select {
	case <-ch:
	case <-time.After(r.n.cfg.DialBackoff):
	case <-ctx.Done():
	}
}

func (r *replica) run(ctx context.Context) {
	tick := time.NewTicker(r.n.cfg.retryInterval())
	defer tick.Stop()
	for {
		row, ch := r.current()
		r.reconcile(ctx, row)
		select {
		case <-ch:
		case <-r.kick:
		case <-tick.C:
		case <-ctx.Done():
			return
		}
	}
}

// reconcile moves the replica toward the role the row gives it.  it is safe
// to call again with the same row.
func (r *replica) reconcile(ctx context.Context, row ShardRow) {
	if ctx.Err() != nil {
		return
	}
	r.mu.Lock()
	reset := row.Nonce != r.nonce
	r.mu.Unlock()
	if reset {
		r.reset(row.Nonce)
	}
	switch {
	case row.Primary == r.n.cfg.Name:
		r.lead(ctx, row)
	case row.Primary != "":
		r.follow(row)
	default:
		r.idle()
	}
}

// reset starts the shard empty with a new nonce.  the table does this when
// no copy with every answered write is left.
func (r *replica) reset(nonce uint32) {
	r.mu.Lock()
	fol, rep := r.fol, r.rep
	r.fol, r.rep = nil, nil
	r.mode = modeFollower
	r.open.Store(false)
	r.nonce = nonce
	r.transferring = false
	r.mu.Unlock()
	if fol != nil {
		fol.stop()
	}
	if rep != nil {
		rep.stop(ErrNotPrimary)
	}
	if _, err := r.replaceManager(); err != nil {
		r.n.log().Error("building a manager failed", "shard", r.shard, "err", err)
	}
}

// replaceManager swaps in a new empty follower manager at LSN 0 and closes
// the old one.
func (r *replica) replaceManager() (*memory.Manager, error) {
	r.mu.Lock()
	nonce := r.nonce
	r.mu.Unlock()
	mgr, err := r.n.newManager(r.shard, nonce)
	if err != nil {
		return nil, err
	}
	r.applyMu.Lock()
	old := r.mgr.Swap(mgr)
	r.tail.reset(0)
	r.lastLSN.Store(0)
	r.applied.Store(0)
	r.applyMu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return mgr, nil
}

// beginTransfer drops the state for a state transfer.
func (r *replica) beginTransfer() (*memory.Manager, error) {
	r.mu.Lock()
	r.transferring = true
	r.mu.Unlock()
	return r.replaceManager()
}

// endTransfer records that the state at lsn is complete.
func (r *replica) endTransfer(lsn uint64) {
	r.applyMu.Lock()
	r.tail.reset(lsn)
	r.lastLSN.Store(lsn)
	r.applied.Store(lsn)
	r.applyMu.Unlock()
	r.mu.Lock()
	r.transferring = false
	r.mu.Unlock()
}

// markBroken is called by a follower goroutine whose manager no longer
// matches the log.
func (r *replica) markBroken(f *follower) {
	r.mu.Lock()
	if r.fol == f {
		r.fol = nil
	}
	if r.mode == modeFollower {
		r.mode = modeBroken
	}
	r.mu.Unlock()
	r.nudge()
}

// follow makes the replica a follower of the row's primary.
func (r *replica) follow(row ShardRow) {
	me := r.n.cfg.Name
	r.mu.Lock()
	if row.Term < r.term || r.closed {
		r.mu.Unlock()
		return
	}
	if r.mode != modeFollower {
		rep := r.rep
		r.rep = nil
		if r.mode == modePrimary {
			r.mode = modeDeposed
			r.open.Store(false)
		}
		r.mu.Unlock()
		if rep != nil {
			rep.stop(ErrNotPrimary)
		}
		if slices.Contains(row.InSync, me) {
			// the table can still elect this node.  it keeps the state it
			// has until the table shows it out of the in-sync set.
			r.shrinkSelf(row)
			return
		}
		if _, err := r.replaceManager(); err != nil {
			r.n.log().Error("building a manager failed", "shard", r.shard, "err", err)
			return
		}
		r.mu.Lock()
		if r.mode == modePrimary {
			r.mu.Unlock()
			return
		}
		r.mode = modeFollower
		r.transferring = false
	}
	r.term = row.Term
	if r.fol != nil && r.fol.primary == row.Primary && r.fol.term == row.Term {
		r.mu.Unlock()
		return
	}
	old := r.fol
	r.fol = nil
	r.mu.Unlock()
	if old != nil {
		old.stop()
	}
	if r.n.draining.Load() {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if row.Term < r.term || r.closed || r.fol != nil || r.mode != modeFollower {
		return
	}
	r.fol = newFollower(r, row.Primary, row.Term)
	r.fol.start()
}

// shrinkSelf proposes to take this node out of the in-sync set.  any node
// may propose that.  it only makes the primary wait on fewer replicas.
func (r *replica) shrinkSelf(row ShardRow) {
	cfg := &r.n.cfg
	r.mu.Lock()
	now := time.Now()
	if now.Sub(r.selfShrinkAt) < cfg.DialBackoff {
		r.mu.Unlock()
		return
	}
	r.selfShrinkAt = now
	r.mu.Unlock()
	go func() {
		_ = r.n.propose(r.n.ctx, ShrinkInSync{
			Shard: r.shard,
			Epoch: row.Epoch,
			Node:  cfg.Name,
		})
	}()
}

// idle stops the replica's work while the row names no primary.
func (r *replica) idle() {
	r.mu.Lock()
	fol, rep := r.fol, r.rep
	r.fol, r.rep = nil, nil
	if r.mode == modePrimary {
		r.mode = modeDeposed
		r.open.Store(false)
	}
	r.mu.Unlock()
	if fol != nil {
		fol.stop()
	}
	if rep != nil {
		rep.stop(ErrNotPrimary)
	}
}

// lead makes the replica the primary of the row's term.
func (r *replica) lead(ctx context.Context, row ShardRow) {
	r.mu.Lock()
	if row.Term < r.term || r.closed {
		r.mu.Unlock()
		return
	}
	if r.mode == modePrimary && r.primaryTerm == row.Term {
		rp := r.rep
		r.mu.Unlock()
		rp.setRow(row)
		return
	}
	r.term = row.Term
	fol, rep := r.fol, r.rep
	r.fol, r.rep = nil, nil
	if r.mode == modePrimary {
		r.mode = modeDeposed
		r.open.Store(false)
	}
	pctx, cancel := context.WithCancel(ctx)
	r.promoteCancel = cancel
	r.mu.Unlock()

	if fol != nil {
		fol.stop()
	}
	if rep != nil {
		rep.stop(ErrNotPrimary)
	}
	r.promote(pctx, row.Term)

	r.mu.Lock()
	r.promoteCancel = nil
	r.mu.Unlock()
	cancel()
}

// promote is the path from a seat in the table to an open primary.  every
// other in-sync member answers a Fence or leaves the set first, and this
// replica takes any record one of them has beyond its own.  after that no
// answered write is missing here.
func (r *replica) promote(ctx context.Context, term uint64) {
	me := r.n.cfg.Name
	answered := map[string]uint64{}
	for {
		if ctx.Err() != nil {
			return
		}
		row, ch := r.current()
		if row.Primary != me || row.Term != term {
			return
		}
		if r.modeIs(modeBroken) {
			r.handoffAny(ctx, row)
			r.wait(ctx, ch)
			continue
		}

		// a peer that left the in-sync set no longer counts.  a write that
		// only it has was not answered at level all, and one answered at
		// level two is on a peer that is still in the set.
		for p := range answered {
			if !slices.Contains(row.InSync, p) {
				delete(answered, p)
			}
		}
		var missing []string
		for _, p := range row.InSync {
			if _, ok := answered[p]; !ok && p != me {
				missing = append(missing, p)
			}
		}
		if len(missing) > 0 {
			for p, lsn := range r.fenceAll(ctx, missing, term) {
				answered[p] = lsn
			}
			again := false
			for _, p := range missing {
				if _, ok := answered[p]; ok {
					continue
				}
				again = true
				_ = r.n.propose(ctx, ShrinkInSync{
					Shard: r.shard,
					Epoch: row.Epoch,
					Node:  p,
				})
			}
			if again {
				r.wait(ctx, ch)
				continue
			}
		}

		best, bestLSN := "", r.lastLSN.Load()
		for p, lsn := range answered {
			if lsn > bestLSN {
				best, bestLSN = p, lsn
			}
		}
		if best == "" {
			break
		}

		// best has records this replica lacks.  a manager that was a primary
		// cannot apply records, so the seat goes to best instead.
		if r.modeIs(modeDeposed) {
			r.handoff(ctx, row, best)
			r.wait(ctx, ch)
			continue
		}
		reply, err := r.fetchTail(ctx, best, r.lastLSN.Load()+1)
		if err != nil {
			// ask it again.  if it is gone it leaves the in-sync set.
			delete(answered, best)
			continue
		}
		if !reply.OK {
			// best no longer has the records this replica lacks.  it has
			// the state they led to, so it takes the seat.
			r.n.log().Warn("a replica that is ahead no longer has the records, it gets the seat", "shard", r.shard, "peer", best)
			r.handoff(ctx, row, best)
			r.wait(ctx, ch)
			continue
		}
		if !r.applyTail(ctx, reply) {
			return
		}
	}
	r.openPrimary(ctx, term)
}

// handoff gives the seat to an in-sync peer that is ahead of this replica.
// the command also takes this node out of the in-sync set, so it takes the
// whole state from the peer afterwards.
func (r *replica) handoff(ctx context.Context, row ShardRow, peer string) {
	_ = r.n.propose(ctx, MovePrimary{
		Shard: r.shard,
		Epoch: row.Epoch,
		Node:  peer,
	})
}

// handoffAny gives the seat to any other in-sync member.  a replica whose
// state is broken uses it.
func (r *replica) handoffAny(ctx context.Context, row ShardRow) {
	for _, p := range row.InSync {
		if p != r.n.cfg.Name {
			r.handoff(ctx, row, p)
			return
		}
	}
	r.n.log().Error("the only in-sync replica has a broken state", "shard", r.shard)
}

// fenceAll sends a Fence to every peer and returns the last LSN of each one
// that answered within the fence timeout.
func (r *replica) fenceAll(ctx context.Context, peers []string, term uint64) map[string]uint64 {
	ctx, cancel := context.WithTimeout(ctx, r.n.cfg.FenceTimeout)
	defer cancel()
	var mu sync.Mutex
	var wg sync.WaitGroup
	out := map[string]uint64{}
	for _, p := range peers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if lsn, ok := r.fenceOne(ctx, p, term); ok {
				mu.Lock()
				out[p] = lsn
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return out
}

func (r *replica) fenceOne(ctx context.Context, peer string, term uint64) (uint64, bool) {
	for ctx.Err() == nil {
		m, err := r.n.call(ctx, r.n.addrOf(peer), Fence{
			Shard: r.shard,
			Term:  term,
		})
		if err == nil {
			reply, ok := m.(FenceReply)
			return reply.LastLSN, ok && reply.OK
		}
		select {
		case <-time.After(r.n.cfg.DialBackoff):
		case <-ctx.Done():
		}
	}
	return 0, false
}

func (r *replica) fetchTail(ctx context.Context, peer string, from uint64) (TailReply, error) {
	ctx, cancel := context.WithTimeout(ctx, r.n.cfg.FenceTimeout)
	defer cancel()
	m, err := r.n.call(ctx, r.n.addrOf(peer), TailRequest{
		Shard:   r.shard,
		FromLSN: from,
	})
	if err != nil {
		return TailReply{}, err
	}
	reply, ok := m.(TailReply)
	if !ok {
		return TailReply{}, fmt.Errorf("cluster: unexpected answer %T to a tail request", m)
	}
	return reply, nil
}

// applyTail applies records pulled from a peer.  it returns false when the
// promotion must stop.
func (r *replica) applyTail(ctx context.Context, reply TailReply) bool {
	r.applyMu.Lock()
	defer r.applyMu.Unlock()
	if ctx.Err() != nil {
		return false
	}
	last := r.lastLSN.Load()
	if reply.FirstLSN > last+1 {
		return false
	}
	if reply.FirstLSN+uint64(reply.Count) <= last+1 {
		return true
	}
	recs, err := DecodeRecords(make([]memory.Record, 0, min(reply.Count, maxPrealloc)), reply.Data)
	if err == nil && len(recs) != reply.Count {
		err = fmt.Errorf("cluster: reply says %d records and has %d", reply.Count, len(recs))
	}
	if err == nil {
		recs = r.keep(recs, reply.FirstLSN, reply.Data)
		r.lastLSN.Store(last + uint64(len(recs)))
		err = r.mgr.Load().Apply(recs)
	}
	if err != nil {
		r.n.log().Error("applying pulled records failed", "shard", r.shard, "err", err)
		r.mu.Lock()
		if r.mode == modeFollower {
			r.mode = modeBroken
		}
		r.mu.Unlock()
		return false
	}
	r.applied.Store(r.lastLSN.Load())
	return true
}

// keep puts records this replica received into its tail and returns the
// ones it does not have yet.  recs is the decoding of data and starts at
// first, at or below the next LSN.  the tail takes data as it is.  when the
// replica already has the front of recs, which is a resume from inside a
// chunk, the tail gets a new encoding of the rest.
func (r *replica) keep(recs []memory.Record, first uint64, data []byte) []memory.Record {
	next := r.lastLSN.Load() + 1
	if skip := next - first; skip > 0 {
		recs = recs[skip:]
		data = AppendRecords(nil, recs)
	}
	r.tail.append(next, len(recs), data, time.Now())
	return recs
}

// openPrimary turns the manager into a primary and starts the replicator.
func (r *replica) openPrimary(ctx context.Context, term uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ctx.Err() != nil || r.closed || r.term != term || r.mode == modeBroken {
		return
	}
	if r.row.Primary != r.n.cfg.Name || r.row.Term != term {
		return
	}
	mgr := r.mgr.Load()
	log := memory.NewLog()
	if r.mode == modeDeposed {
		mgr.SetLog(log)
	} else {
		mgr.Promote(log)
	}
	r.rep = newReplicator(r, mgr, log, term, r.lastLSN.Load(), r.row)
	r.mode = modePrimary
	r.primaryTerm = term
	r.rep.start()
	r.open.Store(true)
}

// fence handles a Fence from a node elected at term.  it stops whatever
// this replica runs and returns the last LSN it has.  ok is false when the
// replica knows a higher term or has no complete state.
func (r *replica) fence(term uint64) (lastLSN uint64, ok bool) {
	r.mu.Lock()
	if term < r.term || r.closed || r.mode == modeBroken || r.transferring {
		r.mu.Unlock()
		return 0, false
	}
	if r.mode == modePrimary && r.primaryTerm == term {
		r.mu.Unlock()
		return 0, false
	}
	r.term = term
	fol, rep := r.fol, r.rep
	r.fol, r.rep = nil, nil
	cancel := r.promoteCancel
	if r.mode == modePrimary {
		r.mode = modeDeposed
		r.open.Store(false)
	}
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if fol != nil {
		fol.stop()
	}
	if rep != nil {
		rep.stop(ErrNotPrimary)
	}
	r.applyMu.Lock()
	lastLSN = r.lastLSN.Load()
	r.applyMu.Unlock()
	r.nudge()
	return lastLSN, true
}

// acceptHello gives a follower's stream to the replicator.
func (r *replica) acceptHello(h Hello, s Stream) {
	r.mu.Lock()
	rp := r.rep
	r.mu.Unlock()
	if rp == nil {
		_ = s.Close()
		return
	}
	rp.accept(h, s)
}

// movePrimary is the planned move of the seat to the in-sync follower that
// has the most.  it stops accepting requests, waits until that follower has
// every record, and proposes MovePrimary.
func (r *replica) movePrimary(ctx context.Context) error {
	cfg := &r.n.cfg
	r.mu.Lock()
	rp := r.rep
	if r.mode != modePrimary || rp == nil {
		r.mu.Unlock()
		return nil
	}
	r.open.Store(false)
	r.mu.Unlock()

	moved := false
	defer func() {
		if moved {
			return
		}
		r.mu.Lock()
		if r.mode == modePrimary && r.rep == rp {
			r.open.Store(true)
		}
		r.mu.Unlock()
	}()

	// a request that got the manager before the seat closed can still
	// append.  the move goes on once the log is quiet and the target has
	// all of it, twice in a row.
	var target string
	var seen uint64
	quiet := 0
	noFollower := time.Now().Add(2 * cfg.AckTimeout)
	for quiet < 2 {
		node, caughtUp, last := rp.best()
		switch {
		case node == "" && time.Now().After(noFollower):
			return fmt.Errorf("cluster: shard %d has no in-sync follower to take the seat", r.shard)
		case node != "" && caughtUp && node == target && last == seen:
			quiet++
		default:
			quiet = 0
		}
		target, seen = node, last
		select {
		case <-time.After(time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	for {
		row, ch := r.current()
		if row.Primary != cfg.Name {
			moved = true
			return nil
		}
		if !slices.Contains(row.InSync, target) {
			return fmt.Errorf("cluster: shard %d target %s left the in-sync set", r.shard, target)
		}
		err := r.n.propose(ctx, MovePrimary{
			Shard: r.shard,
			Epoch: row.Epoch,
			Node:  target,
		})
		if err != nil && !errors.Is(err, ErrRejected) && ctx.Err() != nil {
			return ctx.Err()
		}
		r.wait(ctx, ch)
	}
}

// leaveInSync takes this node out of the shard's in-sync set before a
// planned stop.
func (r *replica) leaveInSync(ctx context.Context) error {
	me := r.n.cfg.Name
	for {
		row, ch := r.current()
		if !slices.Contains(row.InSync, me) || len(row.InSync) == 1 {
			return nil
		}
		if row.Primary == me {
			if err := r.movePrimary(ctx); err != nil {
				return err
			}
		} else {
			_ = r.n.propose(ctx, ShrinkInSync{
				Shard: r.shard,
				Epoch: row.Epoch,
				Node:  me,
			})
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		r.wait(ctx, ch)
	}
}

// close stops the replica at once.  nothing is proposed.
func (r *replica) close() {
	r.mu.Lock()
	r.closed = true
	fol, rep := r.fol, r.rep
	r.fol, r.rep = nil, nil
	cancel := r.promoteCancel
	r.open.Store(false)
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if fol != nil {
		fol.stop()
	}
	if rep != nil {
		rep.stop(ErrNotPrimary)
	}
	if mgr := r.mgr.Load(); mgr != nil {
		_ = mgr.Close()
	}
}
