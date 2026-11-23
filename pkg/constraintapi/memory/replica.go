package memory

import (
	"fmt"
	"sync/atomic"

	"github.com/inngest/inngest/pkg/constraintapi"
)

// WithLog attaches a log.  the manager then appends a Record for every
// change, and an operation waits for its commit level before it answers.
// the caller must drain the log and resolve every handle Drain returns.
func WithLog(l *Log) Option {
	return func(m *Manager) { m.log.Store(l) }
}

// SetLog replaces the log of a primary.  a primary that lost its seat and
// got it back uses it, because the log it had was ended with Fail.  nothing
// may run an operation on the manager at the same time.
func (m *Manager) SetLog(l *Log) {
	m.log.Store(l)
}

// WithNonce sets the nonce in every lease ID instead of drawing one.  every
// replica of a shard uses the same nonce, so a lease ID from one primary is
// valid on the next.
func WithNonce(n uint32) Option {
	return func(m *Manager) {
		m.nonce = n
		m.nonceSet = true
	}
}

// WithFollower builds a manager that only applies records.  it has no
// sweeper and no housekeeping until Promote.  the caller must not send it
// Acquire, ExtendLease, Release or a semaphore write before Promote.
func WithFollower() Option {
	return func(m *Manager) { m.follower = true }
}

// Apply applies records in order.  one goroutine calls it, and nothing else
// may change the manager at the same time.  an error means this manager no
// longer matches the primary and must be rebuilt from a state export.
func (m *Manager) Apply(recs []Record) error {
	for i := range recs {
		if err := m.apply(&recs[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) apply(r *Record) error {
	switch r.Kind {
	case RecSet:
		if r.Set == nil || len(r.Set.Items) != len(r.Set.Limits) || len(r.Set.Items) > constraintapi.MaxConstraints {
			return fmt.Errorf("constraint set %d has no valid definition", r.SetID)
		}
		if m.replSets == nil {
			m.replSets = map[uint64]*constraintSet{}
		}
		if _, ok := m.replSets[r.SetID]; ok {
			return nil
		}
		m.replSets[r.SetID] = m.buildSetFrom(r.Set)
		raise(&m.setSeq, r.SetID)

	case RecSetDrop:
		delete(m.replSets, r.SetID)

	case RecGrant:
		set := m.replSets[r.SetID]
		if set == nil {
			return fmt.Errorf("grant for unknown constraint set %d", r.SetID)
		}
		if len(r.Seqs) == 0 {
			return nil
		}
		m.keepAcquire(r, set)
		rs := &requestState{
			set:             set,
			requestedAmount: r.Requested,
			grantedAmount:   r.Granted,
			maximumLifetime: r.MaximumLifetime,
			source:          r.Source,
			route:           uint8(r.Seqs[0] % logBuffers),
		}
		var placed int64
		for _, seq := range r.Seqs {
			if m.slab.allocAt(seq, r.ExpiresAtMS, rs) {
				placed++
			}
		}
		rs.active.Store(placed)
		if placed == 0 {
			return nil
		}
		for i := range set.constraints {
			rc := &set.constraints[i]
			switch rc.item.Kind {
			case constraintapi.ConstraintKindConcurrency, constraintapi.ConstraintKindSemaphore:
				m.addCell(&rc.usage, rc.usageKey, rc.weight*placed)
			}
		}

	case RecTake:
		sl := m.slab.get(r.Seq)
		if sl == nil {
			return nil
		}
		rs := sl.req.Load()
		if rs == nil || !sl.take() {
			return nil
		}
		m.takeBack(rs.set, r.Force)
		rs.active.Add(-1)
		sl.req.Store(nil)

	case RecExtend:
		sl := m.slab.get(r.Seq)
		if sl == nil {
			return nil
		}
		rs := sl.req.Load()
		if rs == nil || !sl.take() {
			return nil
		}
		m.slab.allocAt(r.NewSeq, r.ExpiresAtMS, rs)
		sl.req.Store(nil)
		m.keepExtend(r, rs.set)

	case RecTAT:
		key := cellKey{a: r.KeyA, b: r.KeyB}
		for c := m.gcraCell(key); !c.apply(r.TAT, r.ExpiresAtMS, r.Version); c = m.gcraCell(key) {
		}

	case RecCell:
		key := cellKey{a: r.KeyA, b: r.KeyB}
		for c := m.sem(key); !c.set(r.Value); c = m.sem(key) {
		}

	case RecDelta:
		key := cellKey{a: r.KeyA, b: r.KeyB}
		for c := m.sem(key); !c.add(r.Value); c = m.sem(key) {
		}

	case RecIdem:
		m.semIdem.set(r.KeyA, r.Value, r.ExpiresAtMS)

	case RecSeqs:
		if len(r.Seqs) != slabShards {
			return fmt.Errorf("sequence record has %d counters, want %d", len(r.Seqs), slabShards)
		}
		for i, n := range r.Seqs {
			m.slab.raise(i, n)
		}

	default:
		return fmt.Errorf("unknown record kind %d", r.Kind)
	}
	return nil
}

// keepAcquire stores the idempotency record of a grant, so an Acquire with
// the same key replays it once this manager is the primary.  the record has
// the leases and the amounts.  it has no limiting or exhausted constraints
// and no usage, because the log does not carry what the first answer
// reported, so a replay on a new primary returns those empty.
func (m *Manager) keepAcquire(r *Record, set *constraintSet) {
	if r.Value == 0 {
		return
	}
	m.acqIdem.set(r.KeyA, &acquireRecord{
		set:         set,
		status:      3,
		requested:   r.Requested,
		granted:     r.Granted,
		expiresAtMS: r.ExpiresAtMS,
		seqs:        r.Seqs,
	}, r.Value)
}

// keepExtend stores the idempotency record of an extend, so an ExtendLease
// with the same key replays the same new lease ID once this manager is the
// primary.  the replayed answer has no usage.
func (m *Manager) keepExtend(r *Record, set *constraintSet) {
	if r.Value == 0 {
		return
	}
	id := encodeLeaseID(r.ExpiresAtMS, r.NewSeq, m.nonce)
	m.extIdem.set(r.KeyA, &constraintapi.CapacityExtendLeaseResponse{
		LeaseID:    &id,
		AccountID:  set.accountID,
		EnvID:      set.envID,
		AppID:      set.appID,
		FunctionID: set.functionID,
	}, r.Value)
}

// raise moves a counter up to at least n.
func raise(c *atomic.Uint64, n uint64) {
	for {
		cur := c.Load()
		if cur >= n || c.CompareAndSwap(cur, n) {
			return
		}
	}
}

// addCell adds delta to a counter through its pointer with no clamp,
// replacing the pointer when housekeeping dropped the counter.
func (m *Manager) addCell(p *atomic.Pointer[semaphoreCell], key cellKey, delta int64) {
	for {
		if p.Load().add(delta) {
			return
		}
		p.Store(m.sem(key))
	}
}

// FollowerHousekeepSteps is the number of FollowerHousekeepStep calls that
// free everything a follower no longer needs.
const FollowerHousekeepSteps = ttlStripes

// FollowerHousekeepStep frees a part of what a follower no longer needs.
// every call drops the expired acquire, extend and semaphore idempotency
// records of one stripe.  one call in FollowerHousekeepSteps also drops slab
// pages with no live lease, counters at zero and expired rate limit and
// throttle cells.  it keeps every constraint set, because the primary logs a
// definition once and may use it again.  call it from the goroutine that
// calls Apply.  that goroutine acknowledges records, and a sweep of every
// stripe in one call stops acknowledgements for as long as it takes.
func (m *Manager) FollowerHousekeepStep() {
	nowMS := m.nowMS()
	i := m.followerStep
	m.followerStep = (i + 1) % FollowerHousekeepSteps
	m.acqIdem.sweepStripe(i, nowMS)
	m.extIdem.sweepStripe(i, nowMS)
	m.semIdem.sweepStripe(i, nowMS)
	if i != 0 {
		return
	}
	m.slab.freeIdlePages()
	m.sems.Range(func(k, v any) bool {
		if v.(*semaphoreCell).kill() {
			m.sems.CompareAndDelete(k, v)
		}
		return true
	})
	m.gcra.Range(func(k, v any) bool {
		if v.(*gcraCell).kill(nowMS) {
			m.gcra.CompareAndDelete(k, v)
		}
		return true
	})
}

// Freeze stops every change to the manager, runs fn, and lets changes go on.
// reads are not stopped.  inside fn the log receives no record, so a Drain
// followed by an Export gives a state and the exact log position it belongs
// to.
func (m *Manager) Freeze(fn func()) {
	m.hkMu.Lock()
	for i := range m.locks {
		m.locks[i].Lock()
	}
	m.capMu.Lock()

	fn()

	m.capMu.Unlock()
	for i := range m.locks {
		m.locks[i].Unlock()
	}
	m.hkMu.Unlock()
}

// Export calls emit with every record a fresh follower needs to reach this
// state.  call it inside Freeze.  the order is fixed.  first the sequence
// counters, then every constraint set, one grant per live lease, every live
// rate limit and throttle state, every counter, and every live semaphore
// idempotency record.  the counter records come after the grants and
// overwrite what the grants added.
//
// acquire and extend idempotency records are left out.  a follower built
// from an export does not replay a request the primary answered before the
// export.  it must not become a primary before IdempotencyEnd.
func (m *Manager) Export(emit func(Record)) {
	nowMS := m.nowMS()

	seqs := make([]uint64, slabShards)
	for i := range seqs {
		seqs[i] = m.slab.shards[i].next.Load()
	}
	emit(Record{
		Kind: RecSeqs,
		Seqs: seqs,
	})

	// every set the sets map has, because a grant logs a definition once per
	// buffer and will not log it again for a follower that joins later.  a
	// set that left the map is exported when a live lease still uses it, or
	// when its RecSetDrop is not logged yet and a request can still grant
	// on it.
	seen := map[uint64]struct{}{}
	emitSet := func(s *constraintSet) {
		if _, ok := seen[s.id]; ok {
			return
		}
		seen[s.id] = struct{}{}
		emit(Record{
			Kind:  RecSet,
			SetID: s.id,
			Set:   s.def,
		})
	}
	m.sets.Range(func(_, v any) bool {
		emitSet(v.(*constraintSet))
		return true
	})
	for _, s := range m.replSets {
		emitSet(s)
	}
	m.slab.each(func(_ uint64, sl *slot) {
		if rs := sl.req.Load(); rs != nil {
			emitSet(rs.set)
		}
	})
	for _, d := range m.setDrops {
		emitSet(d.set)
	}

	// usage is every usage counter a live lease reaches.  a counter that was
	// dropped at zero is exported as zero so the grants do not leave it
	// above the primary's value.
	usage := map[cellKey]struct{}{}
	m.slab.each(func(seq uint64, sl *slot) {
		rs := sl.req.Load()
		if rs == nil {
			return
		}
		emit(Record{
			Kind:            RecGrant,
			SetID:           rs.set.id,
			Seqs:            []uint64{seq},
			ExpiresAtMS:     sl.expiresAtMS,
			Requested:       rs.requestedAmount,
			Granted:         rs.grantedAmount,
			MaximumLifetime: rs.maximumLifetime,
			Source:          rs.source,
		})
		for i := range rs.set.constraints {
			rc := &rs.set.constraints[i]
			switch rc.item.Kind {
			case constraintapi.ConstraintKindConcurrency, constraintapi.ConstraintKindSemaphore:
				usage[rc.usageKey] = struct{}{}
			}
		}
	})

	m.gcra.Range(func(k, v any) bool {
		st := v.(*gcraCell).state.Load()
		if st == nil || st == deadGCRA || st.expiresAt <= nowMS {
			return true
		}
		key := k.(cellKey)
		emit(Record{
			Kind:        RecTAT,
			KeyA:        key.a,
			KeyB:        key.b,
			TAT:         st.tat,
			ExpiresAtMS: st.expiresAt,
			Version:     st.version,
		})
		return true
	})

	m.sems.Range(func(k, v any) bool {
		val, ok := v.(*semaphoreCell).load()
		if !ok {
			return true
		}
		key := k.(cellKey)
		delete(usage, key)
		emit(Record{
			Kind:  RecCell,
			KeyA:  key.a,
			KeyB:  key.b,
			Value: val,
		})
		return true
	})
	for key := range usage {
		emit(Record{
			Kind: RecCell,
			KeyA: key.a,
			KeyB: key.b,
		})
	}

	m.semIdem.each(nowMS, func(key uint64, v int64, expiresAtMS int64) {
		emit(Record{
			Kind:        RecIdem,
			KeyA:        key,
			Value:       v,
			ExpiresAtMS: expiresAtMS,
		})
	})
}

// IdempotencyEnd returns the time, in milliseconds on this manager's clock,
// at which every acquire and extend idempotency record now in the manager
// has expired.  a follower built from an Export of the same Freeze has every
// record that is still live after that time, because the log carried them.
// call it inside Freeze.
func (m *Manager) IdempotencyEnd() int64 {
	return max(m.acqIdem.lastExpiry(), m.extIdem.lastExpiry())
}

// NowMS returns the manager's clock in milliseconds.  idempotency records
// expire on this clock.
func (m *Manager) NowMS() int64 {
	return m.nowMS()
}

// Promote turns a follower into a primary.  the caller has applied every
// record it will get and sends no more.  after Promote the manager accepts
// operations, logs to l and runs its sweeper.  l may be nil.
//
// the lease sequence counters jump past every seq the old primary can have
// handed out, so a new lease never gets the seq of an old one.  usage
// counters are recounted from the live leases, and the expiry index is
// rebuilt so the sweeper finds them.
func (m *Manager) Promote(l *Log) {
	m.hkMu.Lock()
	defer m.hkMu.Unlock()
	if !m.follower {
		return
	}

	m.slab.seal()
	m.recount()
	m.slab.each(func(seq uint64, sl *slot) {
		m.expiry.add(sl.expiresAtMS, seq)
	})

	m.replSets = nil
	m.follower = false
	m.log.Store(l)

	select {
	case <-m.stop:
		return
	default:
	}
	if m.sweepInterval > 0 {
		m.wg.Add(1)
		go m.run()
	}
}

// recount sets usage counters from the live leases.  a counter that only
// concurrency and auto release semaphore constraints reach is set to the sum
// of the weights of its live leases.  a counter that a manual release
// semaphore reaches is left alone, because its usage outlives the lease.  a
// counter no live lease reaches is left alone too.  then every counter below
// zero is set to zero.
func (m *Manager) recount() {
	sum := map[cellKey]int64{}
	manual := map[cellKey]struct{}{}
	m.slab.each(func(_ uint64, sl *slot) {
		rs := sl.req.Load()
		if rs == nil {
			return
		}
		for i := range rs.set.constraints {
			rc := &rs.set.constraints[i]
			switch rc.item.Kind {
			case constraintapi.ConstraintKindConcurrency:
				sum[rc.usageKey]++
			case constraintapi.ConstraintKindSemaphore:
				if rc.release == constraintapi.SemaphoreReleaseAuto {
					sum[rc.usageKey] += rc.weight
				} else {
					manual[rc.usageKey] = struct{}{}
				}
			}
		}
	})
	for key, v := range sum {
		if _, ok := manual[key]; ok {
			continue
		}
		for c := m.sem(key); !c.set(v); c = m.sem(key) {
		}
	}

	// a follower adds with no clamp, so a counter is below zero when the
	// records that would have raised it never arrived.  a primary's counters
	// are never negative.
	m.sems.Range(func(_, v any) bool {
		c := v.(*semaphoreCell)
		for {
			old := c.v.Load()
			if old >= 0 || isDead(old) || c.v.CompareAndSwap(old, 0) {
				return true
			}
		}
	})
}
