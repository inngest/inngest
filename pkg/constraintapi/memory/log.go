package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
)

// RecordKind names what one Record changes.
type RecordKind uint8

const (
	// RecSet defines a constraint set.  Set carries the definition.
	RecSet RecordKind = iota + 1
	// RecSetDrop forgets the set SetID.
	RecSetDrop
	// RecGrant creates one live lease per entry of Seqs, all for the set
	// SetID and all expiring at ExpiresAtMS.  KeyA is the hash of the acquire
	// idempotency key and Value is when that idempotency record expires, so a
	// follower can replay the grant after it is promoted.
	RecGrant
	// RecTake ends the lease Seq.  Force gives manual release semaphores
	// back too.
	RecTake
	// RecExtend ends the lease Seq and creates NewSeq for the same request,
	// expiring at ExpiresAtMS.  KeyA is the hash of the extend idempotency key
	// and Value is when that idempotency record expires.
	RecExtend
	// RecTAT is the state of the rate limit or throttle cell KeyA, KeyB.
	RecTAT
	// RecCell sets the counter KeyA, KeyB to Value.
	RecCell
	// RecDelta adds Value to the counter KeyA, KeyB.  the result never goes
	// below zero.
	RecDelta
	// RecIdem is a semaphore idempotency record.  KeyA is the key hash.
	RecIdem
	// RecSeqs carries the 16 lease sequence counters in Seqs.
	RecSeqs
)

// Record is one physical change to a Manager.  the primary of a shard emits
// records and a follower applies them in the same order with Apply.  which
// fields are set depends on Kind.
type Record struct {
	Kind RecordKind

	// Force is set on RecTake when the sweeper reclaimed the lease.
	Force bool

	// SetID is the constraint set of RecSet, RecSetDrop and RecGrant.
	SetID uint64

	// Seq is the lease of RecTake and the old lease of RecExtend.  NewSeq is
	// the new lease of RecExtend.
	Seq    uint64
	NewSeq uint64

	// ExpiresAtMS is the lease expiry of RecGrant and RecExtend, and the
	// record expiry of RecTAT and RecIdem.
	ExpiresAtMS int64

	// KeyA and KeyB are the cell key of RecTAT, RecCell and RecDelta.  KeyA
	// is the key hash of RecIdem, and the idempotency key hash of RecGrant and
	// RecExtend.
	KeyA uint64
	KeyB uint64

	// Value is the counter of RecCell, the delta of RecDelta and the stored
	// result of RecIdem.  on RecGrant and RecExtend it is the unix millisecond
	// at which the idempotency record expires, and zero when there is none.
	Value int64

	// TAT and Version are the state of RecTAT.  a follower stores the state
	// only when Version is greater than the one it has.
	TAT     float64
	Version uint64

	// Seqs is the leases of RecGrant and the 16 counters of RecSeqs.  the
	// slice is shared and must not be changed.
	Seqs []uint64

	// Requested, Granted, MaximumLifetime and Source describe the acquire
	// behind a RecGrant.
	Requested       int32
	Granted         int32
	MaximumLifetime time.Duration
	Source          constraintapi.LeaseSource

	// Set is the definition of RecSet.  it is shared and must not be
	// changed.
	Set *SetDef
}

// SetDef is everything a follower needs to build a constraint set.  Items
// are in the order of the request and Limits[i] is the resolved limit of
// Items[i].
type SetDef struct {
	ID            uint64
	AccountID     uuid.UUID
	EnvID         uuid.UUID
	FunctionID    uuid.UUID
	AppID         uuid.UUID
	ConfigVersion int
	Items         []constraintapi.ConstraintItem
	Limits        []constraintapi.ConstraintLimits
}

// logBuffers is the number of record buffers.  the top bits of a route pick
// the buffer, so cores that log for different requests rarely share a lock.
const logBuffers = 64

// logBuffer is one padded record buffer.  batch is the handle of the drain
// cycle that takes the records now in buf.
type logBuffer struct {
	mu    sync.Mutex
	buf   []Record
	batch *Batch
	_     [24]byte
}

// Log collects the records one Manager emits.  operations append inside
// their locked section.  one goroutine calls Drain and ships what it gets.
// nothing here limits the buffers, so a Log that is attached must be
// drained.
type Log struct {
	bufs [logBuffers]logBuffer

	// cur is the handle now in the buffers and n is its cycle number.  only
	// Drain and Fail touch them.
	cur *Batch
	n   uint64

	// failed is set by Fail.  Drain discards from then on.
	failed error

	wake chan struct{}
}

// NewLog returns an empty log.
func NewLog() *Log {
	l := &Log{wake: make(chan struct{}, 1)}
	l.cur = newBatch(l, 0)
	for i := range l.bufs {
		l.bufs[i].batch = l.cur
	}
	return l
}

// Drain appends every buffered record to dst and returns the handle that
// covers them and every record drained before.  one goroutine calls it.  the
// handle is returned even when no record was taken, because a replayed
// operation can wait on it.  the caller resolves each handle it gets, in
// order, with AckTwo, AckAll or Fail.
func (l *Log) Drain(dst []Record) ([]Record, *Batch) {
	if l.failed != nil {
		for i := range l.bufs {
			b := &l.bufs[i]
			b.mu.Lock()
			clear(b.buf)
			b.buf = b.buf[:0]
			b.mu.Unlock()
		}
		return dst, l.cur
	}
	old := l.cur
	l.n++
	next := newBatch(l, l.n)
	for i := range l.bufs {
		b := &l.bufs[i]
		b.mu.Lock()
		dst = append(dst, b.buf...)
		clear(b.buf)
		b.buf = b.buf[:0]
		b.batch = next
		b.mu.Unlock()
	}
	l.cur = next
	return dst, old
}

// Fail ends the log.  the handle now in the buffers fails with err, so every
// operation that appends from now on gets err from its wait.  Drain returns
// no record after it.  the goroutine that calls Drain calls it, after its
// last Drain, when the primary loses its seat.  an operation still running
// on a manager that lost its seat then does not wait forever.
func (l *Log) Fail(err error) {
	l.failed = err
	l.cur.Fail(err)
}

// Wake is signalled when an operation is about to wait on a handle.  the
// drain goroutine selects on it so a waiting caller is not left until the
// next timer tick.
func (l *Log) Wake() <-chan struct{} {
	return l.wake
}

func (l *Log) signal() {
	select {
	case l.wake <- struct{}{}:
	default:
	}
}

// buffer returns the buffer for a route.
func (l *Log) buffer(route uint8) *logBuffer {
	return &l.bufs[route%logBuffers]
}

// newest returns a handle that covers every record appended before the
// call.  Drain swaps buffer 0 first, so its handle is never older than the
// handle of any other buffer.
func (l *Log) newest() *Batch {
	b := &l.bufs[0]
	b.mu.Lock()
	h := b.batch
	b.mu.Unlock()
	return h
}

// append adds records to the buffer of route and returns the handle of the
// drain cycle that takes them.
func (l *Log) append(route uint8, recs ...Record) *Batch {
	b := l.buffer(route)
	b.mu.Lock()
	b.buf = append(b.buf, recs...)
	h := b.batch
	b.mu.Unlock()
	return h
}

// Batch is the handle of one drain cycle.  a caller that must not answer
// before its records are on other replicas waits on it.  handles are
// resolved in cycle order, so a resolved handle means every earlier record
// is resolved too.
type Batch struct {
	log *Log
	n   uint64

	two chan struct{}
	all chan struct{}

	mu      sync.Mutex
	twoDone bool
	allDone bool
	errTwo  error
	errAll  error
}

func newBatch(l *Log, n uint64) *Batch {
	return &Batch{
		log: l,
		n:   n,
		two: make(chan struct{}),
		all: make(chan struct{}),
	}
}

// AckTwo says the primary and one in-sync follower have every record up to
// this handle.
func (b *Batch) AckTwo() {
	b.mu.Lock()
	b.resolveTwo(nil)
	b.mu.Unlock()
}

// AckAll says every in-sync replica has every record up to this handle.
func (b *Batch) AckAll() {
	b.mu.Lock()
	b.resolveTwo(nil)
	b.resolveAll(nil)
	b.mu.Unlock()
}

// Fail wakes every waiter that is not yet satisfied and gives it err.  the
// primary calls it when it loses its seat.
func (b *Batch) Fail(err error) {
	b.mu.Lock()
	b.resolveTwo(err)
	b.resolveAll(err)
	b.mu.Unlock()
}

func (b *Batch) resolveTwo(err error) {
	if !b.twoDone {
		b.twoDone = true
		b.errTwo = err
		close(b.two)
	}
}

func (b *Batch) resolveAll(err error) {
	if !b.allDone {
		b.allDone = true
		b.errAll = err
		close(b.all)
	}
}

// Wait blocks until the handle is resolved at level, the handle fails, or
// ctx ends.  CommitLevelDefault and CommitLevelPrimary do not wait.
func (b *Batch) Wait(ctx context.Context, level constraintapi.CommitLevel) error {
	switch {
	case level >= constraintapi.CommitLevelAll:
		select {
		case <-b.all:
			return b.errAll
		case <-ctx.Done():
			return ctx.Err()
		}
	case level == constraintapi.CommitLevelTwo:
		select {
		case <-b.two:
			return b.errTwo
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// Pending collects the handles of many operations so the caller can wait
// once.  put it in the context with WithPending, run the operations, then
// call Wait.  it is safe for concurrent use.
type Pending struct {
	mu      sync.Mutex
	entries []pendingEntry
}

// pendingEntry is the newest handle of one log and the strongest level any
// operation asked of that log.
type pendingEntry struct {
	batch *Batch
	level constraintapi.CommitLevel
}

func (p *Pending) add(b *Batch, level constraintapi.CommitLevel) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range p.entries {
		e := &p.entries[i]
		if e.batch.log != b.log {
			continue
		}
		if b.n > e.batch.n {
			e.batch = b
		}
		if level > e.level {
			e.level = level
		}
		return
	}
	p.entries = append(p.entries, pendingEntry{
		batch: b,
		level: level,
	})
}

// Wait blocks until every collected handle is resolved at its level.  it
// returns the first error.  the collected handles are cleared, so a Pending
// can be used again.
func (p *Pending) Wait(ctx context.Context) error {
	p.mu.Lock()
	entries := p.entries
	p.entries = nil
	p.mu.Unlock()

	var first error
	for _, e := range entries {
		if err := e.batch.Wait(ctx, e.level); err != nil && first == nil {
			first = err
		}
	}
	return first
}

type pendingKey struct{}

// WithPending returns a context that makes every operation add its handle to
// p and return without waiting.
func WithPending(ctx context.Context, p *Pending) context.Context {
	return context.WithValue(ctx, pendingKey{}, p)
}

func pendingFrom(ctx context.Context) *Pending {
	p, _ := ctx.Value(pendingKey{}).(*Pending)
	return p
}
