package cluster

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi/memory"
)

const (
	// maxChunkRecords is the most records one chunk of the tail or of a
	// state transfer carries.
	maxChunkRecords = 4096

	// maxBatchBytes is how many bytes of chunks the sender joins into one
	// Batch for a follower that is behind.  one chunk larger than this is
	// sent alone.
	maxBatchBytes = 256 << 10

	// maxTailReplyBytes bounds one TailReply.  it is far below the frame
	// limit of the transport.
	maxTailReplyBytes = 8 << 20

	// maxDrainBuffer is the largest record buffer the drain loop keeps for
	// the next cycle.
	maxDrainBuffer = 1 << 16

	// expandLag is how far behind the log a follower outside the in-sync set
	// may be when the primary starts to wait on it.  a larger value makes
	// every waiting operation stall for longer while the follower catches up.
	expandLag = 4096

	// maxIdleDrain is the longest the drain loop sleeps when the log is idle.
	maxIdleDrain = 16 * time.Millisecond

	// slowCommitAfter is how long records may wait for the acknowledgement
	// of a follower before the primary logs a warning.  the primary removes
	// a follower only after AckTimeout.  until then a request at commit
	// level all waits, and this line is the only sign of which follower it
	// waits for.
	slowCommitAfter = time.Second

	// slowWarnEvery is the least time between two warnings about the same
	// follower.
	slowWarnEvery = time.Second
)

// pendingHandle is one drain cycle's handle, the last LSN it covers and
// when the cycle ran.
type pendingHandle struct {
	lsn uint64
	b   *memory.Batch
	at  time.Time
}

// followerConn is one follower's stream on the primary.
type followerConn struct {
	node string
	s    Stream
	// next is the LSN the sender sends next.
	next uint64
	// state is what a state transfer sends before the first batch, encoded,
	// and stateLSN is the log position it belongs to.
	transfer bool
	state    []stateChunk
	stateLSN uint64
	// idemEndMS is when the acquire and extend idempotency records the
	// state transfer left out have all expired, on the manager's clock.  the
	// follower cannot replay those requests, so it stays out of the in-sync
	// set until then.
	idemEndMS int64
	// ready is true once the follower has the state.  a resumed follower is
	// ready from the start.
	ready bool
	// seenInSync is true once the table listed the node as in sync while
	// this stream was open.
	seenInSync bool

	wake chan struct{}
	done chan struct{}
	once sync.Once
}

func (fc *followerConn) nudge() {
	select {
	case fc.wake <- struct{}{}:
	default:
	}
}

func (fc *followerConn) close() {
	fc.once.Do(func() {
		close(fc.done)
		_ = fc.s.Close()
	})
}

// stateChunk is one encoded part of a state transfer.
type stateChunk struct {
	count int
	data  []byte
}

// expansion is a follower outside the in-sync set that the primary already
// waits on.  target is the last LSN the primary had when it started to wait.
// the primary proposes ExpandInSync once the follower has target.
type expansion struct {
	target   uint64
	proposed bool
	epoch    uint64
}

// joinRequest is a follower that needs the whole state.
type joinRequest struct {
	node string
	s    Stream
}

// replicator is the primary side of one shard.  it drains the manager's log,
// numbers the records, sends them to the followers and resolves the commit
// handles from their acknowledgements.
type replicator struct {
	r    *replica
	mgr  *memory.Manager
	log  *memory.Log
	term uint64

	mu      sync.Mutex
	row     ShardRow
	rowCh   chan struct{}
	lastLSN uint64
	pending []pendingHandle
	// twoDone counts the handles at the front of pending that are resolved
	// at level two.
	twoDone int
	conns   map[string]*followerConn
	// acks is the last LSN each follower acknowledged on its newest stream.
	acks map[string]uint64
	// progress is when a follower last acknowledged or last had everything.
	progress map[string]time.Time
	// broken lists in-sync followers whose stream broke.
	broken    map[string]bool
	expanding map[string]*expansion
	shrinkTry map[string]time.Time
	// slowWarn is when the primary last logged that it waits for a
	// follower.
	slowWarn map[string]time.Time
	started  time.Time
	stopped  bool

	// drainBuf and scratch belong to the drain goroutine.  the records of a
	// cycle are encoded at once and not kept, so both are used again.
	drainBuf []memory.Record
	scratch  []byte

	joins  chan joinRequest
	stopCh chan struct{}
	done   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
}

func newReplicator(r *replica, mgr *memory.Manager, log *memory.Log, term, lastLSN uint64, row ShardRow) *replicator {
	ctx, cancel := context.WithCancel(context.Background())
	return &replicator{
		r:         r,
		mgr:       mgr,
		log:       log,
		term:      term,
		row:       row,
		rowCh:     make(chan struct{}),
		lastLSN:   lastLSN,
		conns:     map[string]*followerConn{},
		acks:      map[string]uint64{},
		progress:  map[string]time.Time{},
		broken:    map[string]bool{},
		expanding: map[string]*expansion{},
		shrinkTry: map[string]time.Time{},
		slowWarn:  map[string]time.Time{},
		started:   time.Now(),
		joins:     make(chan joinRequest, 16),
		stopCh:    make(chan struct{}),
		done:      make(chan struct{}),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (rp *replicator) start() {
	go rp.run()
}

// run is the only goroutine that drains the log.  a state transfer runs here
// too, because its drain inside Freeze must not race another drain.
func (rp *replicator) run() {
	defer close(rp.done)
	idle := time.Millisecond
	timer := time.NewTimer(idle)
	defer timer.Stop()
	lastMaintain := time.Now()
	for {
		select {
		case <-rp.stopCh:
			return
		case j := <-rp.joins:
			rp.transfer(j)
			continue
		case <-rp.log.Wake():
			rp.cycle()
		case <-timer.C:
			// a release does not signal Wake.  the timer picks it up, and
			// backs off while the log is idle.
			if rp.cycle() > 0 {
				idle = time.Millisecond
			} else {
				idle = min(idle*2, maxIdleDrain)
			}
			timer.Reset(idle)
		}
		if now := time.Now(); now.Sub(lastMaintain) > 5*time.Millisecond {
			lastMaintain = now
			rp.mu.Lock()
			rp.maintainLocked(now)
			rp.mu.Unlock()
		}
	}
}

// cycle drains the log once and returns how many records it took.
func (rp *replicator) cycle() int {
	recs, h := rp.drain()
	rp.mu.Lock()
	rp.appendLocked(recs, h)
	rp.mu.Unlock()
	return len(recs)
}

// drain takes what the log has into the buffer of the drain goroutine.  the
// result is valid until the next call.
func (rp *replicator) drain() ([]memory.Record, *memory.Batch) {
	if cap(rp.drainBuf) > maxDrainBuffer {
		rp.drainBuf = nil
	}
	recs, h := rp.log.Drain(rp.drainBuf[:0])
	rp.drainBuf = recs
	return recs, h
}

// encode returns the encoding of recs in a slice of its own.
func (rp *replicator) encode(recs []memory.Record) []byte {
	rp.scratch = AppendRecords(rp.scratch[:0], recs)
	return append([]byte(nil), rp.scratch...)
}

// appendLocked gives the records their LSNs, keeps them encoded in the tail
// and queues the handle.  it does not keep recs.
func (rp *replicator) appendLocked(recs []memory.Record, h *memory.Batch) {
	now := time.Now()
	if len(recs) > 0 {
		// a follower that had everything starts its silence now.
		for node, acked := range rp.acks {
			if acked == rp.lastLSN {
				rp.progress[node] = now
			}
		}
		for len(recs) > 0 {
			n := min(len(recs), maxChunkRecords)
			rp.r.tail.append(rp.lastLSN+1, n, rp.encode(recs[:n]), now)
			rp.lastLSN += uint64(n)
			recs = recs[n:]
		}
		rp.r.lastLSN.Store(rp.lastLSN)
		for _, fc := range rp.conns {
			fc.nudge()
		}
	}
	rp.pending = append(rp.pending, pendingHandle{
		lsn: rp.lastLSN,
		b:   h,
		at:  now,
	})
	rp.resolveLocked()
}

// marksLocked returns how far the log is on one in-sync follower and on
// every replica the primary waits on.  with nobody to wait on both are the
// last LSN.  a follower the primary waits on before the table lists it
// counts for all, and not for two, because it cannot be elected yet.
func (rp *replicator) marksLocked() (two, all uint64) {
	me := rp.r.n.cfg.Name
	all = rp.lastLSN
	n := 0
	for _, node := range rp.row.InSync {
		if node == me {
			continue
		}
		a := rp.acks[node]
		two = max(two, a)
		all = min(all, a)
		n++
	}
	if n == 0 {
		two = rp.lastLSN
	}
	for node := range rp.expanding {
		if !slices.Contains(rp.row.InSync, node) {
			all = min(all, rp.acks[node])
		}
	}
	return two, all
}

// resolveLocked resolves every handle the acknowledgements cover, in order.
func (rp *replicator) resolveLocked() {
	two, all := rp.marksLocked()
	i := 0
	for i < len(rp.pending) && rp.pending[i].lsn <= all {
		rp.pending[i].b.AckAll()
		rp.pending[i].b = nil
		i++
	}
	if i == len(rp.pending) {
		rp.pending = rp.pending[:0]
	} else {
		rp.pending = rp.pending[i:]
	}
	rp.twoDone = max(rp.twoDone-i, 0)
	for rp.twoDone < len(rp.pending) && rp.pending[rp.twoDone].lsn <= two {
		rp.pending[rp.twoDone].b.AckTwo()
		rp.twoDone++
	}
}

// setRow gives the replicator the newest row of its shard.
func (rp *replicator) setRow(row ShardRow) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if rp.stopped || row.Epoch <= rp.row.Epoch {
		return
	}
	rp.row = row
	close(rp.rowCh)
	rp.rowCh = make(chan struct{})
	for node := range rp.broken {
		if !slices.Contains(row.InSync, node) {
			delete(rp.broken, node)
			delete(rp.shrinkTry, node)
		}
	}
	rp.maintainLocked(time.Now())
	rp.resolveLocked()
}

// maintainLocked proposes the in-sync changes the current state asks for.
// it runs on every row, every acknowledgement, every broken stream and every
// few milliseconds.
func (rp *replicator) maintainLocked(now time.Time) {
	if rp.stopped {
		return
	}
	cfg := &rp.r.n.cfg
	me := cfg.Name

	if len(rp.pending) > 0 {
		if waited := now.Sub(rp.pending[0].at); waited > cfg.slowCommit() {
			rp.warnSlowLocked(now, waited)
		}
	}

	for _, node := range rp.row.InSync {
		if node == me {
			continue
		}
		fc := rp.conns[node]
		if fc != nil {
			fc.seenInSync = true
		}
		switch {
		case fc == nil && rp.broken[node]:
			rp.shrinkLocked(node, now)
		case fc == nil && now.Sub(rp.started) > cfg.AckTimeout:
			rp.shrinkLocked(node, now)
		case fc != nil && rp.acks[node] < rp.lastLSN && now.Sub(rp.progress[node]) > cfg.AckTimeout:
			rp.shrinkLocked(node, now)
		}
	}

	for node, fc := range rp.conns {
		if slices.Contains(rp.row.InSync, node) {
			continue
		}
		if fc.seenInSync {
			// the table removed it while it was connected.  it dials
			// again and takes the whole state.
			rp.dropLocked(fc)
			continue
		}
		ex := rp.expanding[node]
		switch {
		case ex == nil:
			if fc.ready && rp.lastLSN-rp.acks[node] <= expandLag && rp.mgr.NowMS() >= fc.idemEndMS {
				rp.expanding[node] = &expansion{target: rp.lastLSN}
				rp.progress[node] = now
			}
		case !ex.proposed && rp.acks[node] >= ex.target:
			ex.proposed = true
			ex.epoch = rp.row.Epoch
			go rp.expand(node, ex)
		case !ex.proposed && now.Sub(rp.progress[node]) > cfg.AckTimeout:
			// it stopped while the primary waited on it.  stop waiting.
			delete(rp.expanding, node)
			rp.dropLocked(fc)
		}
	}
}

// warnSlowLocked logs every follower whose acknowledgement the oldest
// records wait for.  waited is how long they have waited.  one follower gets
// one line in slowWarnEvery at most.
func (rp *replicator) warnSlowLocked(now time.Time, waited time.Duration) {
	lsn := rp.pending[0].lsn
	warn := func(node string) {
		if node == rp.r.n.cfg.Name || rp.acks[node] >= lsn || now.Sub(rp.slowWarn[node]) < slowWarnEvery {
			return
		}
		rp.slowWarn[node] = now
		rp.r.n.log().Warn("records wait for the acknowledgement of a follower",
			"shard", rp.r.shard,
			"follower", node,
			"waited", waited,
			"acked", rp.acks[node],
			"last", rp.lastLSN,
		)
	}
	for _, node := range rp.row.InSync {
		warn(node)
	}
	for node := range rp.expanding {
		if !slices.Contains(rp.row.InSync, node) {
			warn(node)
		}
	}
}

// shrinkLocked proposes to remove a follower from the in-sync set.  the
// primary keeps waiting on the follower until a row shows the removal.
func (rp *replicator) shrinkLocked(node string, now time.Time) {
	cfg := &rp.r.n.cfg
	if t, ok := rp.shrinkTry[node]; ok && now.Sub(t) < cfg.DialBackoff {
		return
	}
	rp.shrinkTry[node] = now
	cmd := ShrinkInSync{
		Shard: rp.r.shard,
		Epoch: rp.row.Epoch,
		Node:  node,
	}
	go func() {
		ctx, cancel := context.WithTimeout(rp.ctx, cfg.proposeTimeout())
		defer cancel()
		_ = cfg.Coordinator.Propose(ctx, cmd)
	}()
}

// expand proposes ExpandInSync until a row with a higher epoch arrives.
// that row decides.  if it lists the node the command was applied, and if
// not it was refused.  the primary waits on the node the whole time, so an
// answer it gives while the outcome is open is on the node too.
func (rp *replicator) expand(node string, ex *expansion) {
	cfg := &rp.r.n.cfg
	cmd := ExpandInSync{
		Shard: rp.r.shard,
		Epoch: ex.epoch,
		Node:  node,
	}
	for {
		ctx, cancel := context.WithTimeout(rp.ctx, cfg.proposeTimeout())
		_ = cfg.Coordinator.Propose(ctx, cmd)
		cancel()

		deadline := time.NewTimer(cfg.proposeTimeout())
		for {
			rp.mu.Lock()
			decided := rp.stopped || rp.row.Epoch > ex.epoch || rp.expanding[node] != ex
			ch := rp.rowCh
			if decided {
				if rp.expanding[node] == ex {
					delete(rp.expanding, node)
				}
				now := time.Now()
				rp.maintainLocked(now)
				rp.resolveLocked()
				rp.mu.Unlock()
				deadline.Stop()
				return
			}
			rp.mu.Unlock()
			select {
			case <-ch:
				continue
			case <-rp.stopCh:
				continue
			case <-deadline.C:
			}
			break
		}
	}
}

func (rp *replicator) dropLocked(fc *followerConn) {
	if rp.conns[fc.node] == fc {
		delete(rp.conns, fc.node)
	}
	fc.close()
}

// onBreak is called when a follower's stream fails.
func (rp *replicator) onBreak(fc *followerConn) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if rp.conns[fc.node] != fc {
		fc.close()
		return
	}
	rp.dropLocked(fc)
	if ex := rp.expanding[fc.node]; ex != nil && !ex.proposed {
		delete(rp.expanding, fc.node)
	}
	rp.broken[fc.node] = true
	rp.maintainLocked(time.Now())
	rp.resolveLocked()
}

func (rp *replicator) onAck(fc *followerConn, lsn uint64) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if rp.conns[fc.node] != fc {
		return
	}
	// the first acknowledgement after a state transfer names the LSN of the
	// state.  that is 0 for an empty shard, so it does not have to move the
	// position to count.
	if lsn <= rp.acks[fc.node] && (fc.ready || lsn < fc.stateLSN) {
		return
	}
	now := time.Now()
	rp.acks[fc.node] = max(lsn, rp.acks[fc.node])
	rp.progress[fc.node] = now
	if lsn >= fc.stateLSN {
		fc.ready = true
	}
	rp.maintainLocked(now)
	rp.resolveLocked()
}

// accept handles the Hello of a follower.  it runs on the stream's own
// goroutine.
func (rp *replicator) accept(h Hello, s Stream) {
	cfg := &rp.r.n.cfg
	for {
		rp.mu.Lock()
		if rp.stopped || h.Term > rp.term || !slices.Contains(rp.row.Replicas, h.Node) {
			rp.mu.Unlock()
			_ = s.Close()
			return
		}
		inSync := slices.Contains(rp.row.InSync, h.Node)
		// a follower that has every record is covered also when the tail is
		// empty, which is a shard that was idle for longer than the window.
		covered := rp.r.tail.covers(h.LastLSN + 1)
		if inSync && covered && h.LastLSN <= rp.lastLSN {
			if old := rp.conns[h.Node]; old != nil {
				rp.dropLocked(old)
			}
			fc := &followerConn{
				node:       h.Node,
				s:          s,
				next:       h.LastLSN + 1,
				ready:      true,
				seenInSync: true,
				wake:       make(chan struct{}, 1),
				done:       make(chan struct{}),
			}
			rp.conns[h.Node] = fc
			rp.acks[h.Node] = h.LastLSN
			rp.progress[h.Node] = time.Now()
			delete(rp.broken, h.Node)
			rp.resolveLocked()
			rp.mu.Unlock()
			go rp.send(fc)
			go rp.recv(fc)
			return
		}
		if !inSync {
			rp.mu.Unlock()
			break
		}
		// the follower is in the in-sync set and cannot go on from where it
		// is.  it leaves the set first, so nobody elects it while it has
		// half a state.
		if old := rp.conns[h.Node]; old != nil {
			rp.dropLocked(old)
		}
		rp.acks[h.Node] = 0
		rp.shrinkLocked(h.Node, time.Now())
		ch := rp.rowCh
		rp.mu.Unlock()
		select {
		case <-ch:
		case <-time.After(cfg.DialBackoff):
		case <-rp.stopCh:
		}
	}

	select {
	case rp.joins <- joinRequest{
		node: h.Node,
		s:    s,
	}:
	case <-rp.stopCh:
		_ = s.Close()
	}
}

// transfer registers a follower at the current end of the log and exports
// the state that belongs to that position.  it runs on the drain goroutine.
func (rp *replicator) transfer(j joinRequest) {
	fc := &followerConn{
		node:     j.node,
		s:        j.s,
		transfer: true,
		wake:     make(chan struct{}, 1),
		done:     make(chan struct{}),
	}
	registered := false
	records, size := 0, 0
	rp.mgr.Freeze(func() {
		recs, h := rp.drain()
		rp.mu.Lock()
		rp.appendLocked(recs, h)
		// a node whose ExpandInSync is still open may enter the in-sync set
		// at any moment.  it must not start from an empty state then.
		ex := rp.expanding[j.node]
		if rp.stopped || slices.Contains(rp.row.InSync, j.node) || (ex != nil && ex.proposed) {
			rp.mu.Unlock()
			return
		}
		if old := rp.conns[j.node]; old != nil {
			rp.dropLocked(old)
		}
		delete(rp.expanding, j.node)
		fc.stateLSN = rp.lastLSN
		fc.idemEndMS = rp.mgr.IdempotencyEnd()
		fc.next = rp.lastLSN + 1
		rp.conns[j.node] = fc
		rp.acks[j.node] = 0
		rp.progress[j.node] = time.Now()
		delete(rp.broken, j.node)
		registered = true
		rp.mu.Unlock()
		// the state is encoded as it is exported, so the primary keeps the
		// bytes and never the records.
		buf := rp.drainBuf[:0]
		flush := func() {
			if len(buf) == 0 {
				return
			}
			data := rp.encode(buf)
			fc.state = append(fc.state, stateChunk{
				count: len(buf),
				data:  data,
			})
			records += len(buf)
			size += len(data)
			buf = buf[:0]
		}
		rp.mgr.Export(func(r memory.Record) {
			buf = append(buf, r)
			if len(buf) == maxChunkRecords {
				flush()
			}
		})
		flush()
		rp.drainBuf = buf
	})
	if !registered {
		_ = j.s.Close()
		return
	}
	st := &rp.r.n.stats
	st.transfers.Add(1)
	st.stateRecords.Add(uint64(records))
	st.stateBytes.Add(uint64(size))
	go rp.send(fc)
	go rp.recv(fc)
}

// send is the sender goroutine of one follower.  it sends whole chunks of
// the tail and joins the chunks a follower that is behind lacks into one
// batch, so a slow follower gets fewer and larger messages.
func (rp *replicator) send(fc *followerConn) {
	if !rp.sendStart(fc) {
		rp.onBreak(fc)
		return
	}
	for {
		rp.mu.Lock()
		if rp.conns[fc.node] != fc {
			rp.mu.Unlock()
			return
		}
		next := fc.next
		rp.mu.Unlock()

		first, count, data, ok := rp.r.tail.read(next, maxBatchBytes)
		if !ok {
			// the follower fell out of the tail.
			rp.onBreak(fc)
			return
		}
		if count == 0 {
			select {
			case <-fc.wake:
			case <-fc.done:
				return
			}
			continue
		}
		rp.mu.Lock()
		fc.next = first + uint64(count)
		rp.mu.Unlock()

		err := fc.s.Send(Batch{
			Term:     rp.term,
			FirstLSN: first,
			Count:    count,
			Data:     data,
		})
		if err != nil {
			rp.onBreak(fc)
			return
		}
	}
}

// sendStart sends the answer to the Hello, and the state for a transfer.
func (rp *replicator) sendStart(fc *followerConn) bool {
	if !fc.transfer {
		return fc.s.Send(Resume{Term: rp.term}) == nil
	}
	if fc.s.Send(StateBegin{Term: rp.term}) != nil {
		return false
	}
	state := fc.state
	fc.state = nil
	for i, c := range state {
		state[i] = stateChunk{}
		err := fc.s.Send(StateRecords{
			Count: c.count,
			Data:  c.data,
		})
		if err != nil {
			return false
		}
	}
	return fc.s.Send(StateEnd{LSN: fc.stateLSN}) == nil
}

// recv reads one follower's acknowledgements.
func (rp *replicator) recv(fc *followerConn) {
	for {
		m, err := fc.s.Recv()
		if err != nil {
			rp.onBreak(fc)
			return
		}
		if ack, ok := m.(Ack); ok {
			rp.onAck(fc, ack.LSN)
		}
	}
}

// best returns the connected in-sync follower with the highest
// acknowledgement, whether it has every record, and the last LSN.
func (rp *replicator) best() (node string, caughtUp bool, lastLSN uint64) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	me := rp.r.n.cfg.Name
	var bestAck uint64
	for _, n := range rp.row.InSync {
		if n == me || rp.conns[n] == nil {
			continue
		}
		if a := rp.acks[n]; node == "" || a > bestAck {
			node, bestAck = n, a
		}
	}
	return node, node != "" && bestAck == rp.lastLSN, rp.lastLSN
}

// stop ends the replicator.  every open handle and every later one fails
// with err, and every follower stream is closed.
func (rp *replicator) stop(err error) {
	rp.once.Do(func() {
		rp.mu.Lock()
		rp.stopped = true
		rp.mu.Unlock()
		rp.cancel()
		close(rp.stopCh)
		<-rp.done

		rp.mu.Lock()
		for _, p := range rp.pending {
			p.b.Fail(err)
		}
		rp.pending = nil
		for _, fc := range rp.conns {
			fc.close()
		}
		rp.conns = map[string]*followerConn{}
		rp.mu.Unlock()
		rp.log.Fail(err)

		for {
			select {
			case j := <-rp.joins:
				_ = j.s.Close()
				continue
			default:
			}
			break
		}
	})
}
