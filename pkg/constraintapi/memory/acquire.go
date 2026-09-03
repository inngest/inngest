package memory

import (
	"context"
	"sort"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
)

// constraintSet is one request shape: the account, env, function and app it
// runs for, and its constraints in evaluation order with their configured
// limits and counters resolved.  every request with the same shape shares one
// set, so a request does one hash and one lookup instead of sorting,
// resolving limits and looking up counters.  the key covers the resolved
// limits, so a request carrying different limits under the same function
// version gets its own set, the way the Redis manager stores limits per
// request.  housekeeping drops a set no request used since its last round.
type constraintSet struct {
	accountID     uuid.UUID
	envID         uuid.UUID
	functionID    uuid.UUID
	appID         uuid.UUID
	configVersion int

	// constraints is in evaluation order.  order[i] is where constraints[i]
	// sits in the request's own list.  the request order is part of the key,
	// so every request sharing the set has the same order.
	constraints []resolvedConstraint
	order       []uint8

	used atomic.Bool
}

// requestState is shared by every lease one Acquire granted.  slots point at
// it, so there is no request ID lookup.  it is garbage collected when the
// last slot that references it is taken.
type requestState struct {
	set *constraintSet

	requestedAmount int32
	grantedAmount   int32

	// active counts leases not yet released.
	active atomic.Int64

	maximumLifetime time.Duration
	source          constraintapi.LeaseSource
}

// resolvedConstraint is one constraint with its counters and configured
// limits looked up once when the set was built.  counter pointers are
// replaced when housekeeping dropped the counter they point at.
type resolvedConstraint struct {
	item   constraintapi.ConstraintItem
	limits constraintapi.ConstraintLimits

	// usage is the in progress counter for concurrency and semaphores.
	usageKey cellKey
	usage    atomic.Pointer[semaphoreCell]

	// capacity is the semaphore capacity counter.  unset for other kinds.
	capacityKey cellKey
	capacity    atomic.Pointer[semaphoreCell]

	// weight is the semaphore weight, at least 1.
	weight int64
	// release is the semaphore release mode.
	release constraintapi.SemaphoreReleaseMode
}

// acquireRecord is what one Acquire leaves for replays: seqs and indices, not
// the response.  the response is rebuilt from the record and the replaying
// request, which carries the same keys and constraints by fingerprint.
type acquireRecord struct {
	set *constraintSet

	status    uint8
	nl, ne    uint8
	limiting  [constraintapi.MaxConstraints]uint8
	exhausted [constraintapi.MaxConstraints]uint8

	requested   int32
	granted     int32
	expiresAtMS int64
	retryAtMS   int64

	// seqs are the granted leases in lease key order.
	seqs []uint64
	// usage is one entry per constraint in evaluation order.
	usage []usagePair
}

type usagePair struct {
	used  int64
	limit int64
}

// resolve hashes the request into its set key and its acquire idempotency
// key, and returns the set, building it on first use.  the set key covers
// the IDs, the constraints in request order and their resolved limits.  the
// acquire key adds the request fields, so a retry with anything changed is a
// new request.
func (m *Manager) resolve(req *constraintapi.CapacityAcquireRequest) (*constraintSet, uint64, errs.InternalError) {
	var arr [512]byte
	b := arr[:0]
	b = appendUUID(b, req.AccountID)
	b = appendUUID(b, req.EnvID)
	b = appendUUID(b, req.FunctionID)
	b = appendUUID(b, req.AppID)
	b = appendInt(b, req.Configuration.FunctionVersion)
	b = appendInt(b, len(req.Constraints))
	for _, ci := range req.Constraints {
		limits := ci.ResolveLimits(req.Configuration)
		b = appendStr(b, string(ci.Kind))
		b = appendInt(b, limits.Limit)
		b = appendInt(b, limits.Burst)
		b = appendInt(b, limits.Period)
		switch ci.Kind {
		case constraintapi.ConstraintKindSemaphore:
			b = appendStr(b, ci.Semaphore.ID)
			b = appendStr(b, ci.Semaphore.EvaluatedKeyHash)
			b = appendU64(b, uint64(ci.Semaphore.Weight))
			b = appendInt(b, int(ci.Semaphore.Release))
		default:
			return nil, 0, errs.Wrap(0, false, "constraint kind %q is not implemented in the memory store", ci.Kind)
		}
	}
	setKey := cellKey{a: xxhash.Sum64(b), b: sumSeeded(b, cellSeed)}

	b = appendStr(b, "acq")
	b = appendStr(b, req.IdempotencyKey)
	b = appendInt(b, req.Amount)
	b = appendU64(b, uint64(req.MaximumLifetime.Milliseconds()))
	b = appendInt(b, int(req.Source.Service))
	b = appendInt(b, int(req.Source.Location))
	b = appendInt(b, int(req.Source.RunProcessingMode))
	b = appendInt(b, len(req.LeaseIdempotencyKeys))
	for _, k := range req.LeaseIdempotencyKeys {
		b = appendStr(b, k)
	}
	b = appendInt(b, len(req.LeaseRunIDs))
	switch len(req.LeaseRunIDs) {
	case 0:
	case 1:
		for k, v := range req.LeaseRunIDs {
			b = appendStr(b, k)
			b = appendBytes(b, v[:])
		}
	default:
		keys := make([]string, 0, len(req.LeaseRunIDs))
		for k := range req.LeaseRunIDs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := req.LeaseRunIDs[k]
			b = appendStr(b, k)
			b = appendBytes(b, v[:])
		}
	}
	acqKey := xxhash.Sum64(b)

	return m.set(setKey, req), acqKey, nil
}

// set returns the interned set for key, building it from req on first use.
func (m *Manager) set(key cellKey, req *constraintapi.CapacityAcquireRequest) *constraintSet {
	if v, ok := m.sets.Load(key); ok {
		s := v.(*constraintSet)
		if !s.used.Load() {
			s.used.Store(true)
		}
		return s
	}
	s := m.buildSet(req)
	s.used.Store(true)
	v, _ := m.sets.LoadOrStore(key, s)
	return v.(*constraintSet)
}

func (m *Manager) buildSet(req *constraintapi.CapacityAcquireRequest) *constraintSet {
	n := len(req.Constraints)
	order := make([]uint8, n)
	for i := range order {
		order[i] = uint8(i)
	}
	sort.SliceStable(order, func(i, j int) bool {
		return constraintapi.ConstraintLess(req.Constraints[order[i]], req.Constraints[order[j]])
	})

	s := &constraintSet{
		accountID:     req.AccountID,
		envID:         req.EnvID,
		functionID:    req.FunctionID,
		appID:         req.AppID,
		configVersion: req.Configuration.FunctionVersion,
		constraints:   make([]resolvedConstraint, n),
		order:         order,
	}
	for i, oi := range order {
		ci := req.Constraints[oi]
		rc := &s.constraints[i]
		rc.item = ci
		rc.limits = ci.ResolveLimits(req.Configuration)
		switch ci.Kind {
		case constraintapi.ConstraintKindSemaphore:
			rc.usageKey, rc.capacityKey = semaphoreCells(req.AccountID, ci.Semaphore.ID, ci.Semaphore.EvaluatedKeyHash)
			rc.weight = ci.Semaphore.Weight
			if rc.weight <= 0 {
				rc.weight = 1
			}
			rc.release = ci.Semaphore.Release
			rc.usage.Store(m.sem(rc.usageKey))
			rc.capacity.Store(m.sem(rc.capacityKey))
		}
	}
	return s
}

// Acquire implements constraintapi.CapacityManager.
func (m *Manager) Acquire(ctx context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	if err := req.Valid(); err != nil {
		return nil, errs.Wrap(0, false, "invalid request: %w", err)
	}

	now := m.clock.Now()
	nowMS := now.UnixMilli()
	requestID := m.newRequestID(now)

	m.stats.record(metricEvent{kind: histRequestLatency, value: now.Sub(req.CurrentTime), attempt: req.RequestAttempt})

	leaseExpiryMS := now.Add(req.Duration).UnixMilli()

	set, acqKey, ierr := m.resolve(req)
	if ierr != nil {
		return nil, ierr
	}

	// the lock serializes callers with the same idempotency key.  the first
	// one performs the acquire and every later one replays its record, the
	// way a second Lua call finds the first call's record.
	mu := m.lock(acqKey)
	rec, hit := m.acqIdem.get(nowMS, acqKey)
	if !hit {
		rec = m.acquire(nowMS, leaseExpiryMS, req, set, acqKey)
	}
	mu.Unlock()

	resp := m.response(req, rec, requestID, hit, now)

	m.stats.record(metricEvent{kind: histRetryAfter, value: max(resp.RetryAfter.Sub(now), 0), source: req.Source})

	for _, hook := range m.lifecycles {
		err := hook.OnCapacityLeaseAcquired(ctx, constraintapi.OnCapacityLeaseAcquiredData{
			AccountID:               req.AccountID,
			EnvID:                   req.EnvID,
			AppID:                   req.AppID,
			FunctionID:              req.FunctionID,
			Configuration:           req.Configuration,
			Constraints:             req.Constraints,
			LimitingConstraints:     resp.LimitingConstraints,
			ExhaustedConstraints:    resp.ExhaustedConstraints,
			RetryAfter:              resp.RetryAfter,
			RequestedAmount:         req.Amount,
			Duration:                req.Duration,
			Source:                  req.Source,
			GrantedLeases:           resp.Leases,
			Usage:                   resp.Usage,
			OperationIdempotencyHit: hit,
		})
		if err != nil {
			return nil, errs.Wrap(0, false, "acquire lifecycle failed: %w", err)
		}
	}

	fn := uuid.Nil
	if m.enableHighCardinalityInstrumentation != nil && m.enableHighCardinalityInstrumentation(ctx, req.AccountID, req.EnvID, req.FunctionID) {
		fn = req.FunctionID
	}
	m.stats.acquired(fn, int(rec.status), int(rec.requested), int(rec.granted), resp.LimitingConstraints, resp.ExhaustedConstraints, len(resp.Leases), req.Source)

	return resp, nil
}

// response builds the caller's response from a record and the request.  the
// constraints come from the request's own list so a caller sees the values it
// sent, positioned by the set's evaluation order.
func (m *Manager) response(req *constraintapi.CapacityAcquireRequest, rec *acquireRecord, requestID ulid.ULID, hit bool, now time.Time) *constraintapi.CapacityAcquireResponse {
	set := rec.set
	item := func(sorted uint8) constraintapi.ConstraintItem {
		if oi := int(set.order[sorted]); oi < len(req.Constraints) {
			return req.Constraints[oi]
		}
		return set.constraints[sorted].item
	}

	leases := make([]constraintapi.CapacityLease, len(rec.seqs))
	for i, seq := range rec.seqs {
		leases[i] = constraintapi.CapacityLease{
			LeaseID:        encodeLeaseID(rec.expiresAtMS, seq, m.nonce),
			IdempotencyKey: req.LeaseIdempotencyKeys[i],
		}
	}

	var limiting, exhausted []constraintapi.ConstraintItem
	if rec.nl > 0 {
		limiting = make([]constraintapi.ConstraintItem, rec.nl)
		for i, idx := range rec.limiting[:rec.nl] {
			limiting[i] = item(idx)
		}
	}
	if rec.ne > 0 {
		exhausted = make([]constraintapi.ConstraintItem, rec.ne)
		for i, idx := range rec.exhausted[:rec.ne] {
			exhausted[i] = item(idx)
		}
	}

	usage := make([]constraintapi.ConstraintUsage, len(rec.usage))
	for i, u := range rec.usage {
		usage[i] = constraintapi.ConstraintUsage{Constraint: item(uint8(i)), Used: int(u.used), Limit: int(u.limit)}
	}

	retryAfter := time.UnixMilli(rec.retryAtMS)
	if retryAfter.Before(now) {
		retryAfter = time.Time{}
	}

	return &constraintapi.CapacityAcquireResponse{
		RequestID:               requestID,
		Leases:                  leases,
		LimitingConstraints:     limiting,
		ExhaustedConstraints:    exhausted,
		Usage:                   usage,
		RetryAfter:              retryAfter,
		OperationIdempotencyHit: hit,
	}
}

// acquireState collects the limiting and exhausted indices of one acquire and
// what each pass saw of every constraint.  requests hold at most
// constraintapi.MaxConstraints constraints, which Valid enforces, so fixed
// arrays are enough.
type acquireState struct {
	limiting     [constraintapi.MaxConstraints]uint8
	exhausted    [constraintapi.MaxConstraints]uint8
	nl, ne       uint8
	limitingSet  [constraintapi.MaxConstraints]bool
	exhaustedSet [constraintapi.MaxConstraints]bool
	retryAt      int64
	views        [constraintapi.MaxConstraints]view
}

// view is what the passes saw of one constraint.  pass one fills it from
// loads.  pass two updates used with the value take returned, so the report
// needs no second load of a counter other cores are writing.
type view struct {
	units   int
	retryAt int64
	used    int64
	limit   int64
}

func (st *acquireState) limit(i int) {
	if !st.limitingSet[i] {
		st.limitingSet[i] = true
		st.limiting[st.nl] = uint8(i)
		st.nl++
	}
}

func (st *acquireState) exhaust(i int, retryAt int64) {
	if !st.exhaustedSet[i] {
		st.exhaustedSet[i] = true
		st.exhausted[st.ne] = uint8(i)
		st.ne++
	}
	if retryAt > st.retryAt {
		st.retryAt = retryAt
	}
}

// acquire is the body of one Acquire.  it runs once per idempotency key.
//
// pass one reads every counter and computes the grant the way acquire.lua
// does.  pass two takes the grant from each counter and shrinks it when
// another request got there first, rolling earlier counters back.  the
// report recomputes each constraint from the values pass two left, which is
// what acquire.lua reads back after its own increments.  under no contention
// the result matches the Lua.
func (m *Manager) acquire(nowMS, leaseExpiryMS int64, req *constraintapi.CapacityAcquireRequest, set *constraintSet, acqKey uint64) *acquireRecord {
	var st acquireState

	available := req.Amount
	for i := range set.constraints {
		rc, v := &set.constraints[i], &st.views[i]
		m.snapshot(rc, v)
		if v.units <= 0 {
			st.exhaust(i, v.retryAt)
		}
		if v.units < available {
			available = v.units
			st.limit(i)
		}
	}
	if available <= 0 {
		return m.record(req, set, &st, 0, leaseExpiryMS, nil)
	}

	granted := available
	var taken [constraintapi.MaxConstraints]int
	for i := range set.constraints {
		fit := m.commit(&set.constraints[i], granted, &st.views[i])
		taken[i] = fit
		if fit < granted {
			for j := 0; j < i; j++ {
				m.rollback(&set.constraints[j], taken[j]-fit, &st.views[j])
				taken[j] = fit
			}
			granted = fit
			st.limit(i)
		}
	}

	st.retryAt = 0
	for i := range set.constraints {
		rc, v := &set.constraints[i], &st.views[i]
		recompute(rc, v)
		if v.units <= 0 {
			st.exhaust(i, v.retryAt)
		}
	}
	if granted == 0 {
		return m.record(req, set, &st, 0, leaseExpiryMS, nil)
	}

	rs := &requestState{
		set:             set,
		requestedAmount: int32(req.Amount),
		grantedAmount:   int32(granted),
		maximumLifetime: req.MaximumLifetime,
		source:          req.Source,
	}
	rs.active.Store(int64(granted))

	seqs := make([]uint64, granted)
	ccExpiry := idemExpiry(nowMS, m.constraintCheckIdempotencyTTL)
	for i := 0; i < granted; i++ {
		seq, _ := m.slab.alloc(nowMS, leaseExpiryMS, rs)
		m.expiry.add(leaseExpiryMS, seq)
		seqs[i] = seq
		m.checkIdem.set(opKey(req.AccountID, "cc", req.LeaseIdempotencyKeys[i]), struct{}{}, ccExpiry)
	}
	m.checkIdem.set(opKey(req.AccountID, "cc", req.IdempotencyKey), struct{}{}, ccExpiry)

	rec := m.record(req, set, &st, granted, leaseExpiryMS, seqs)
	rec.status = 3
	m.acqIdem.set(acqKey, rec, idemExpiry(nowMS, min(m.operationIdempotencyTTL, req.Duration)))
	return rec
}

// record builds a status 2 record from what the passes saw.  the caller
// raises the status on a grant.
func (m *Manager) record(req *constraintapi.CapacityAcquireRequest, set *constraintSet, st *acquireState, granted int, expiresAtMS int64, seqs []uint64) *acquireRecord {
	rec := &acquireRecord{
		set:         set,
		status:      2,
		nl:          st.nl,
		ne:          st.ne,
		limiting:    st.limiting,
		exhausted:   st.exhausted,
		requested:   int32(req.Amount),
		granted:     int32(granted),
		expiresAtMS: expiresAtMS,
		retryAtMS:   st.retryAt,
		seqs:        seqs,
		usage:       make([]usagePair, len(set.constraints)),
	}
	for i := range set.constraints {
		v := &st.views[i]
		rec.usage[i] = usagePair{used: max(v.used, 0), limit: v.limit}
	}
	return rec
}

// recompute derives units and retryAt from used and limit, the acquire.lua
// formula for the kind.
func recompute(rc *resolvedConstraint, v *view) {
	v.units = 0
	v.retryAt = 0
	switch rc.item.Kind {
	case constraintapi.ConstraintKindSemaphore:
		if remaining := v.limit - v.used; remaining >= rc.weight {
			v.units = int(remaining / rc.weight)
		}
	}
}

// snapshot reads one constraint into v.  nothing is written.
func (m *Manager) snapshot(rc *resolvedConstraint, v *view) {
	switch rc.item.Kind {
	case constraintapi.ConstraintKindSemaphore:
		v.used = m.loadCell(&rc.usage, rc.usageKey)
		v.limit = m.loadCell(&rc.capacity, rc.capacityKey)
	}
	recompute(rc, v)
}

// commit takes q units from one constraint and returns how many fit.  v.used
// becomes the counter value with this take accounted.
func (m *Manager) commit(rc *resolvedConstraint, q int, v *view) int {
	switch rc.item.Kind {
	case constraintapi.ConstraintKindSemaphore:
		for {
			if fit, after, ok := rc.usage.Load().take(v.limit, rc.weight, q); ok {
				v.used = after
				return fit
			}
			rc.usage.Store(m.sem(rc.usageKey))
		}
	}
	return 0
}

// rollback gives units back to one constraint after a later constraint
// shrank the grant.
func (m *Manager) rollback(rc *resolvedConstraint, units int, v *view) {
	if units <= 0 {
		return
	}
	switch rc.item.Kind {
	case constraintapi.ConstraintKindSemaphore:
		v.used = m.giveCell(&rc.usage, rc.usageKey, rc.weight*int64(units))
	}
}

// loadCell reads a counter through its pointer, replacing the pointer when
// housekeeping dropped the counter.
func (m *Manager) loadCell(p *atomic.Pointer[semaphoreCell], key cellKey) int64 {
	for {
		if v, ok := p.Load().load(); ok {
			return v
		}
		p.Store(m.sem(key))
	}
}

// giveCell subtracts w from a counter through its pointer, replacing the
// pointer when housekeeping dropped the counter.
func (m *Manager) giveCell(p *atomic.Pointer[semaphoreCell], key cellKey, w int64) int64 {
	for {
		if v, ok := p.Load().give(w); ok {
			return v
		}
		p.Store(m.sem(key))
	}
}
