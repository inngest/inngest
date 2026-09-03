package memory

import (
	"context"
	"sort"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/util/errs"
)

// requestState is shared by every lease one Acquire granted.  slots point at
// it, so there is no request ID lookup.  it is garbage collected when the
// last slot that references it is freed.
type requestState struct {
	accountID  uuid.UUID
	envID      uuid.UUID
	functionID uuid.UUID
	appID      uuid.UUID

	// constraints is the request's constraint list in evaluation order, with
	// the counter for each already resolved.
	constraints []resolvedConstraint

	configVersion   int
	requestedAmount int
	grantedAmount   int

	// active counts leases not yet released.
	active atomic.Int64

	maximumLifetime time.Duration
	source          constraintapi.LeaseSource
}

// resolvedConstraint is one constraint with its counters and configured
// limits looked up once at acquire time.  counter pointers are replaced when
// housekeeping dropped the counter they point at.
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

// acquireResult is the immutable outcome one Acquire computed.  every caller
// replaying it builds its own response from this.
type acquireResult struct {
	status               int
	requested            int
	granted              int
	leases               []constraintapi.CapacityLease
	limitingConstraints  []constraintapi.ConstraintItem
	exhaustedConstraints []constraintapi.ConstraintItem
	usage                []constraintapi.ConstraintUsage
	retryAtMS            int64
}

// resolve sorts the request's constraints in place, looks up their counters
// and limits, and returns the acquire idempotency key.  the key covers the
// request fingerprint, so a retry with a different configuration or
// constraint set is a new request.
func (m *Manager) resolve(req *constraintapi.CapacityAcquireRequest) (*requestState, uint64, errs.InternalError) {
	constraintapi.SortConstraints(req.Constraints)

	rs := &requestState{
		accountID:       req.AccountID,
		envID:           req.EnvID,
		functionID:      req.FunctionID,
		appID:           req.AppID,
		constraints:     make([]resolvedConstraint, len(req.Constraints)),
		configVersion:   req.Configuration.FunctionVersion,
		requestedAmount: req.Amount,
		maximumLifetime: req.MaximumLifetime,
		source:          req.Source,
	}

	var h hasher
	h.reset()
	h.uuid(req.AccountID)
	h.str("acq")
	h.str(req.IdempotencyKey)
	h.uuid(req.EnvID)
	h.uuid(req.FunctionID)
	h.uuid(req.AppID)
	h.int(req.Configuration.FunctionVersion)
	h.int(req.Amount)
	h.u64(uint64(req.MaximumLifetime.Milliseconds()))
	h.int(int(req.Source.Service))
	h.int(int(req.Source.Location))
	h.int(int(req.Source.RunProcessingMode))
	h.int(len(req.LeaseIdempotencyKeys))
	for _, k := range req.LeaseIdempotencyKeys {
		h.str(k)
	}
	h.int(len(req.LeaseRunIDs))
	switch len(req.LeaseRunIDs) {
	case 0:
	case 1:
		for k, v := range req.LeaseRunIDs {
			h.str(k)
			h.bytes(v[:])
		}
	default:
		keys := make([]string, 0, len(req.LeaseRunIDs))
		for k := range req.LeaseRunIDs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := req.LeaseRunIDs[k]
			h.str(k)
			h.bytes(v[:])
		}
	}

	for i, ci := range req.Constraints {
		rc := &rs.constraints[i]
		rc.item = ci
		rc.limits = ci.ResolveLimits(req.Configuration)
		h.str(string(ci.Kind))
		h.int(rc.limits.Limit)
		h.int(rc.limits.Burst)
		h.int(rc.limits.Period)

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
			h.str(ci.Semaphore.ID)
			h.str(ci.Semaphore.EvaluatedKeyHash)
			h.u64(uint64(rc.weight))
			h.int(int(rc.release))
		default:
			return nil, 0, errs.Wrap(0, false, "constraint kind %q is not implemented in the memory store", ci.Kind)
		}
	}

	return rs, h.sum(), nil
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

	rs, acqKey, ierr := m.resolve(req)
	if ierr != nil {
		return nil, ierr
	}

	// the lock serializes callers with the same idempotency key.  the first
	// one performs the acquire and every later one replays its result, the
	// way a second Lua call finds the first call's record.
	mu := m.lock(acqKey)
	res, hit := m.acqIdem.get(nowMS, acqKey)
	if !hit {
		res = m.acquire(nowMS, leaseExpiryMS, req, rs, acqKey)
	}
	mu.Unlock()

	leases := make([]constraintapi.CapacityLease, len(res.leases))
	copy(leases, res.leases)

	retryAfter := time.UnixMilli(res.retryAtMS)
	if retryAfter.Before(now) {
		retryAfter = time.Time{}
	}
	m.stats.record(metricEvent{kind: histRetryAfter, value: max(retryAfter.Sub(now), 0), source: req.Source})

	for _, hook := range m.lifecycles {
		err := hook.OnCapacityLeaseAcquired(ctx, constraintapi.OnCapacityLeaseAcquiredData{
			AccountID:               req.AccountID,
			EnvID:                   req.EnvID,
			AppID:                   req.AppID,
			FunctionID:              req.FunctionID,
			Configuration:           req.Configuration,
			Constraints:             req.Constraints,
			LimitingConstraints:     res.limitingConstraints,
			ExhaustedConstraints:    res.exhaustedConstraints,
			RetryAfter:              retryAfter,
			RequestedAmount:         req.Amount,
			Duration:                req.Duration,
			Source:                  req.Source,
			GrantedLeases:           leases,
			Usage:                   res.usage,
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
	m.stats.acquired(fn, res, req.Source)

	return &constraintapi.CapacityAcquireResponse{
		RequestID:               requestID,
		Leases:                  leases,
		LimitingConstraints:     res.limitingConstraints,
		ExhaustedConstraints:    res.exhaustedConstraints,
		Usage:                   res.usage,
		RetryAfter:              retryAfter,
		OperationIdempotencyHit: hit,
	}, nil
}

// acquireState collects the limiting and exhausted indices of one acquire.
// requests hold at most constraintapi.MaxConstraints constraints, which
// Valid enforces, so fixed arrays are enough.
type acquireState struct {
	limiting     [constraintapi.MaxConstraints]int
	exhausted    [constraintapi.MaxConstraints]int
	nl, ne       int
	limitingSet  [constraintapi.MaxConstraints]bool
	exhaustedSet [constraintapi.MaxConstraints]bool
	retryAt      int64
}

func (st *acquireState) limit(i int) {
	if !st.limitingSet[i] {
		st.limitingSet[i] = true
		st.limiting[st.nl] = i
		st.nl++
	}
}

func (st *acquireState) exhaust(i int, retryAt int64) {
	if !st.exhaustedSet[i] {
		st.exhaustedSet[i] = true
		st.exhausted[st.ne] = i
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
// another request got there first, rolling earlier counters back.  pass
// three reads the counters again for the response.  under no contention the
// three passes see the same values and the result matches the Lua.
func (m *Manager) acquire(nowMS, leaseExpiryMS int64, req *constraintapi.CapacityAcquireRequest, rs *requestState, acqKey uint64) *acquireResult {
	n := len(rs.constraints)
	var st acquireState
	usage := make([]constraintapi.ConstraintUsage, n)

	available := req.Amount
	for i := range rs.constraints {
		capacity, at, u := m.snapshot(&rs.constraints[i])
		usage[i] = u
		if capacity <= 0 {
			st.exhaust(i, at)
		}
		if capacity < available {
			available = capacity
			st.limit(i)
		}
	}
	if available <= 0 {
		return m.result(rs, &st, req.Amount, 0, nil, usage)
	}

	granted := available
	var taken [constraintapi.MaxConstraints]int
	for i := range rs.constraints {
		fit := m.commit(&rs.constraints[i], granted)
		taken[i] = fit
		if fit < granted {
			for j := 0; j < i; j++ {
				m.rollback(&rs.constraints[j], taken[j]-fit)
				taken[j] = fit
			}
			granted = fit
			st.limit(i)
		}
	}

	st.retryAt = 0
	for i := range rs.constraints {
		capacity, at, u := m.snapshot(&rs.constraints[i])
		usage[i] = u
		if capacity <= 0 {
			st.exhaust(i, at)
		}
	}
	if granted == 0 {
		return m.result(rs, &st, req.Amount, 0, nil, usage)
	}

	leases := make([]constraintapi.CapacityLease, granted)
	ccExpiry := idemExpiry(nowMS, m.constraintCheckIdempotencyTTL)
	for i := 0; i < granted; i++ {
		seq, _ := m.slab.alloc(nowMS, leaseExpiryMS, rs)
		m.expiry.add(leaseExpiryMS, seq)
		key := req.LeaseIdempotencyKeys[i]
		leases[i] = constraintapi.CapacityLease{
			LeaseID:        encodeLeaseID(leaseExpiryMS, seq, m.nonce),
			IdempotencyKey: key,
		}
		m.checkIdem.set(opKey(req.AccountID, "cc", key), struct{}{}, ccExpiry)
	}
	m.checkIdem.set(opKey(req.AccountID, "cc", req.IdempotencyKey), struct{}{}, ccExpiry)
	rs.grantedAmount = granted
	rs.active.Store(int64(granted))

	res := m.result(rs, &st, req.Amount, granted, leases, usage)
	res.status = 3
	m.acqIdem.set(acqKey, res, idemExpiry(nowMS, min(m.operationIdempotencyTTL, req.Duration)))
	return res
}

// result builds a status 2 result.  the caller raises the status on a grant.
func (m *Manager) result(rs *requestState, st *acquireState, requested, granted int, leases []constraintapi.CapacityLease, usage []constraintapi.ConstraintUsage) *acquireResult {
	return &acquireResult{
		status:               2,
		requested:            requested,
		granted:              granted,
		leases:               leases,
		limitingConstraints:  m.items(rs, st.limiting[:st.nl]),
		exhaustedConstraints: m.items(rs, st.exhausted[:st.ne]),
		usage:                usage,
		retryAtMS:            st.retryAt,
	}
}

// items maps sorted constraint indices to the constraints.  nil when empty,
// the way the Redis manager leaves them.
func (m *Manager) items(rs *requestState, idx []int) []constraintapi.ConstraintItem {
	if len(idx) == 0 {
		return nil
	}
	out := make([]constraintapi.ConstraintItem, len(idx))
	for i, k := range idx {
		out[i] = rs.constraints[k].item
	}
	return out
}

// snapshot reads one constraint and returns how many units fit, its retry
// time, and its usage entry.  nothing is written.
func (m *Manager) snapshot(rc *resolvedConstraint) (capacity int, retryAtMS int64, usage constraintapi.ConstraintUsage) {
	switch rc.item.Kind {
	case constraintapi.ConstraintKindSemaphore:
		u := m.loadCell(&rc.usage, rc.usageKey)
		c := m.loadCell(&rc.capacity, rc.capacityKey)
		if remaining := c - u; remaining >= rc.weight {
			capacity = int(remaining / rc.weight)
		}
		return capacity, 0, constraintapi.ConstraintUsage{Constraint: rc.item, Limit: int(c), Used: int(max(u, 0))}
	}
	return 0, 0, constraintapi.ConstraintUsage{Constraint: rc.item}
}

// commit takes q units from one constraint and returns how many fit.
func (m *Manager) commit(rc *resolvedConstraint, q int) int {
	switch rc.item.Kind {
	case constraintapi.ConstraintKindSemaphore:
		for {
			c := m.loadCell(&rc.capacity, rc.capacityKey)
			if fit, ok := rc.usage.Load().take(c, rc.weight, q); ok {
				return fit
			}
			rc.usage.Store(m.sem(rc.usageKey))
		}
	}
	return 0
}

// rollback gives units back to one constraint after a later constraint
// shrank the grant.
func (m *Manager) rollback(rc *resolvedConstraint, units int) {
	if units <= 0 {
		return
	}
	switch rc.item.Kind {
	case constraintapi.ConstraintKindSemaphore:
		m.giveCell(&rc.usage, rc.usageKey, rc.weight*int64(units))
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
