package memory

import (
	"context"
	"math"
	"sort"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/gcra"
	"github.com/inngest/inngest/pkg/enums"
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

	// usage is the in progress counter for concurrency and semaphores.  a
	// concurrency constraint is a semaphore of weight 1 whose capacity is the
	// request's configured limit instead of a counter.
	usageKey cellKey
	usage    atomic.Pointer[semaphoreCell]

	// capacity is the semaphore capacity counter.  unset for other kinds.
	capacityKey cellKey
	capacity    atomic.Pointer[semaphoreCell]

	// weight is the semaphore weight, at least 1.  1 for concurrency.
	weight int64
	// release is the semaphore release mode.  auto for concurrency.
	release constraintapi.SemaphoreReleaseMode

	// gcra is the TAT cell for rate limits and throttles.  period is in the
	// helper's unit, nanoseconds for rate limits and milliseconds for
	// throttles.  burst is what the helper takes: limit/10 for rate limits and
	// limit + burst - 1 for throttles, so one request can take every unit.
	stateKey cellKey
	gcra     atomic.Pointer[gcraCell]
	period   float64
	burst    int
}

func (rc *resolvedConstraint) isGCRA() bool {
	return rc.item.Kind == constraintapi.ConstraintKindRateLimit || rc.item.Kind == constraintapi.ConstraintKindThrottle
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
	// usage is one entry per constraint in evaluation order.  a constraint
	// the request skipped has no usage, like the Lua leaves it out.
	usage []usagePair
}

type usagePair struct {
	used  int64
	limit int64
	has   bool
}

// appendSetIdentity appends what makes a constraint set: the IDs, the config
// version, and every constraint's identity and resolved limits in request
// order.
func appendSetIdentity(b []byte, accountID, envID, functionID, appID uuid.UUID, config constraintapi.ConstraintConfig, constraints []constraintapi.ConstraintItem) ([]byte, errs.InternalError) {
	b = appendUUID(b, accountID)
	b = appendUUID(b, envID)
	b = appendUUID(b, functionID)
	b = appendUUID(b, appID)
	b = appendInt(b, config.FunctionVersion)
	b = appendInt(b, len(constraints))
	for _, ci := range constraints {
		limits := ci.ResolveLimits(config)
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
		case constraintapi.ConstraintKindRateLimit:
			b = appendInt(b, int(ci.RateLimit.Scope))
			b = appendStr(b, ci.RateLimit.KeyExpressionHash)
			b = appendStr(b, ci.RateLimit.EvaluatedKeyHash)
		case constraintapi.ConstraintKindThrottle:
			b = appendInt(b, int(ci.Throttle.Scope))
			b = appendStr(b, ci.Throttle.KeyExpressionHash)
			b = appendStr(b, ci.Throttle.EvaluatedKeyHash)
		case constraintapi.ConstraintKindConcurrency:
			b = appendInt(b, int(ci.Concurrency.Mode))
			b = appendInt(b, int(ci.Concurrency.Scope))
			b = appendStr(b, ci.Concurrency.KeyExpressionHash)
			b = appendStr(b, ci.Concurrency.EvaluatedKeyHash)
		default:
			return nil, errs.Wrap(0, false, "constraint kind %q is not implemented in the memory store", ci.Kind)
		}
	}
	return b, nil
}

// resolve hashes the request into its set key and its acquire idempotency
// key, and returns the set, building it on first use.  the acquire key adds
// the request fields to the set identity, so a retry with anything changed is
// a new request.
func (m *Manager) resolve(req *constraintapi.CapacityAcquireRequest) (*constraintSet, uint64, errs.InternalError) {
	var arr [512]byte
	b, ierr := appendSetIdentity(arr[:0], req.AccountID, req.EnvID, req.FunctionID, req.AppID, req.Configuration, req.Constraints)
	if ierr != nil {
		return nil, 0, ierr
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

	set := m.set(setKey, func() *constraintSet {
		return m.buildSet(req.AccountID, req.EnvID, req.FunctionID, req.AppID, req.Configuration, req.Constraints)
	})
	return set, acqKey, nil
}

// set returns the interned set for key, building it on first use.
func (m *Manager) set(key cellKey, build func() *constraintSet) *constraintSet {
	if v, ok := m.sets.Load(key); ok {
		s := v.(*constraintSet)
		if !s.used.Load() {
			s.used.Store(true)
		}
		return s
	}
	s := build()
	s.used.Store(true)
	v, _ := m.sets.LoadOrStore(key, s)
	return v.(*constraintSet)
}

func (m *Manager) buildSet(accountID, envID, functionID, appID uuid.UUID, config constraintapi.ConstraintConfig, constraints []constraintapi.ConstraintItem) *constraintSet {
	n := len(constraints)
	order := make([]uint8, n)
	for i := range order {
		order[i] = uint8(i)
	}
	sort.SliceStable(order, func(i, j int) bool {
		return constraintapi.ConstraintLess(constraints[order[i]], constraints[order[j]])
	})

	s := &constraintSet{
		accountID:     accountID,
		envID:         envID,
		functionID:    functionID,
		appID:         appID,
		configVersion: config.FunctionVersion,
		constraints:   make([]resolvedConstraint, n),
		order:         order,
	}
	for i, oi := range order {
		ci := constraints[oi]
		rc := &s.constraints[i]
		rc.item = ci
		rc.limits = ci.ResolveLimits(config)
		switch ci.Kind {
		case constraintapi.ConstraintKindSemaphore:
			rc.usageKey, rc.capacityKey = semaphoreCells(accountID, ci.Semaphore.ID, ci.Semaphore.EvaluatedKeyHash)
			rc.weight = ci.Semaphore.Weight
			if rc.weight <= 0 {
				rc.weight = 1
			}
			rc.release = ci.Semaphore.Release
			rc.usage.Store(m.sem(rc.usageKey))
			rc.capacity.Store(m.sem(rc.capacityKey))
		case constraintapi.ConstraintKindConcurrency:
			var entity uuid.UUID
			switch ci.Concurrency.Scope {
			case enums.ConcurrencyScopeAccount:
				entity = accountID
			case enums.ConcurrencyScopeEnv:
				entity = envID
			case enums.ConcurrencyScopeFn:
				entity = functionID
			}
			rc.usageKey = scopedKey('n', accountID, int(ci.Concurrency.Scope), entity, ci.Concurrency.KeyExpressionHash, ci.Concurrency.EvaluatedKeyHash)
			rc.weight = 1
			rc.release = constraintapi.SemaphoreReleaseAuto
			rc.usage.Store(m.sem(rc.usageKey))
		case constraintapi.ConstraintKindRateLimit:
			var entity uuid.UUID
			switch ci.RateLimit.Scope {
			case enums.RateLimitScopeAccount:
				entity = accountID
			case enums.RateLimitScopeEnv:
				entity = envID
			case enums.RateLimitScopeFn:
				entity = functionID
			}
			rc.stateKey = scopedKey('r', accountID, int(ci.RateLimit.Scope), entity, ci.RateLimit.KeyExpressionHash, ci.RateLimit.EvaluatedKeyHash)
			rc.period = float64(rc.limits.Period)
			rc.burst = rc.limits.Burst
			rc.gcra.Store(m.gcraCell(rc.stateKey))
		case constraintapi.ConstraintKindThrottle:
			var entity uuid.UUID
			switch ci.Throttle.Scope {
			case enums.ThrottleScopeAccount:
				entity = accountID
			case enums.ThrottleScopeEnv:
				entity = envID
			case enums.ThrottleScopeFn:
				entity = functionID
			}
			rc.stateKey = scopedKey('t', accountID, int(ci.Throttle.Scope), entity, ci.Throttle.KeyExpressionHash, ci.Throttle.EvaluatedKeyHash)
			rc.period = float64(rc.limits.Period)
			rc.burst = rc.limits.Limit + rc.limits.Burst - 1
			rc.gcra.Store(m.gcraCell(rc.stateKey))
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
	requestID := m.newRequestID(now)

	m.stats.record(metricEvent{kind: histRequestLatency, value: now.Sub(req.CurrentTime), attempt: req.RequestAttempt})

	leaseExpiryMS := now.Add(req.Duration).UnixMilli()

	set, acqKey, ierr := m.resolve(req)
	if ierr != nil {
		return nil, ierr
	}

	pc := passCtx{nowMS: now.UnixMilli(), nowNS: now.UnixNano()}

	// the lock serializes callers with the same idempotency key.  the first
	// one performs the acquire and every later one replays its record, the
	// way a second Lua call finds the first call's record.
	mu := m.lock(acqKey)
	rec, hit := m.acqIdem.get(pc.nowMS, acqKey)
	if !hit {
		rec = m.acquire(pc, leaseExpiryMS, req, set, acqKey)
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

	usage := make([]constraintapi.ConstraintUsage, 0, len(rec.usage))
	for i, u := range rec.usage {
		if !u.has {
			continue
		}
		usage = append(usage, constraintapi.ConstraintUsage{Constraint: item(uint8(i)), Used: int(u.used), Limit: int(u.limit)})
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

// passCtx is what every pass of one acquire or check shares.
type passCtx struct {
	nowMS int64
	nowNS int64
	// skipGCRA is set when the request's idempotency key was seen before, so
	// rate limits and throttles are not charged again for a retry.
	skipGCRA bool
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
// loads.  pass two updates used with the value the counter update returned,
// so the report needs no second load of a counter other cores are writing.
// has is false when the constraint was skipped and reports no usage.
type view struct {
	units   int
	retryAt int64
	used    int64
	limit   int64
	has     bool
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
// pass one reads every constraint and computes the grant the way acquire.lua
// does.  pass two takes the grant from each constraint and shrinks it when
// another request got there first, rolling the ones already taken back.
// counters are taken first, exactly, then the rate limit and throttle, so a
// shrink almost never has to roll a TAT back.  the report recomputes each
// constraint from the values pass two left, which is what acquire.lua reads
// back after its own increments.  under no contention the result matches the
// Lua.
func (m *Manager) acquire(pc passCtx, leaseExpiryMS int64, req *constraintapi.CapacityAcquireRequest, set *constraintSet, acqKey uint64) *acquireRecord {
	var st acquireState
	pc.skipGCRA = m.hasCheckIdem(pc.nowMS, req.AccountID, req.IdempotencyKey)

	available := req.Amount
	for i := range set.constraints {
		rc, v := &set.constraints[i], &st.views[i]
		m.snapshot(pc, rc, v, available)
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

	// commit order: counters first, GCRA last
	var order [constraintapi.MaxConstraints]int
	n := 0
	for i := range set.constraints {
		if !set.constraints[i].isGCRA() {
			order[n] = i
			n++
		}
	}
	for i := range set.constraints {
		if set.constraints[i].isGCRA() {
			order[n] = i
			n++
		}
	}

	granted := available
	var taken [constraintapi.MaxConstraints]int
	for k := 0; k < n; k++ {
		i := order[k]
		fit := m.commit(pc, &set.constraints[i], granted, &st.views[i])
		taken[i] = fit
		if fit < granted {
			for j := 0; j < k; j++ {
				oi := order[j]
				m.rollback(pc, &set.constraints[oi], taken[oi]-fit, &st.views[oi])
				taken[oi] = fit
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
	ccExpiry := idemExpiry(pc.nowMS, m.constraintCheckIdempotencyTTL)
	for i := 0; i < granted; i++ {
		seq, _ := m.slab.alloc(pc.nowMS, leaseExpiryMS, rs)
		m.expiry.add(leaseExpiryMS, seq)
		seqs[i] = seq
		m.checkIdem.set(opKey(req.AccountID, "cc", req.LeaseIdempotencyKeys[i]), struct{}{}, ccExpiry)
	}
	m.checkIdem.set(opKey(req.AccountID, "cc", req.IdempotencyKey), struct{}{}, ccExpiry)

	rec := m.record(req, set, &st, granted, leaseExpiryMS, seqs)
	rec.status = 3
	m.acqIdem.set(acqKey, rec, idemExpiry(pc.nowMS, min(m.operationIdempotencyTTL, req.Duration)))
	return rec
}

// hasCheckIdem reports whether an acquire with this idempotency key, or a
// lease with it as lease key, was granted within the check idempotency TTL.
func (m *Manager) hasCheckIdem(nowMS int64, accountID uuid.UUID, key string) bool {
	_, ok := m.checkIdem.get(nowMS, opKey(accountID, "cc", key))
	return ok
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
		rec.usage[i] = usagePair{used: max(v.used, 0), limit: v.limit, has: v.has}
	}
	return rec
}

// toInteger rounds the way the Lua toInteger does.
func toInteger(v float64) int64 {
	return int64(math.Floor(v + 0.5))
}

// recompute derives units from used and limit for a counter constraint.
// a concurrency constraint keeps the Lua's limit - usage, which goes negative
// when a limit was lowered below the usage and Check reports it that way; a
// semaphore floors at zero and divides by its weight.  retryAt for
// concurrency is set once by snapshot.  GCRA views hold the helper's result
// and are left alone.
func recompute(rc *resolvedConstraint, v *view) {
	switch rc.item.Kind {
	case constraintapi.ConstraintKindConcurrency:
		v.units = int(v.limit - v.used)
	case constraintapi.ConstraintKindSemaphore:
		v.units = 0
		v.retryAt = 0
		if remaining := v.limit - v.used; remaining >= rc.weight {
			v.units = int(remaining / rc.weight)
		}
	}
}

// gcraStep runs the helper for rc with quantity q on the given TAT.
func (rc *resolvedConstraint) gcraStep(pc passCtx, tat float64, present bool, q int) (gcra.Result, float64, int64, bool) {
	if rc.item.Kind == constraintapi.ConstraintKindRateLimit {
		return gcra.RateLimit(tat, present, float64(pc.nowNS), rc.period, rc.limits.Limit, rc.burst, q)
	}
	return gcra.Throttle(tat, present, float64(pc.nowMS), rc.period, rc.limits.Limit, rc.burst, q)
}

// gcraRetryAt converts the helper's retry_at to unix milliseconds.
func (rc *resolvedConstraint) gcraRetryAt(res gcra.Result) int64 {
	if rc.item.Kind == constraintapi.ConstraintKindRateLimit {
		return toInteger(res.RetryAt / 1_000_000)
	}
	return toInteger(res.RetryAt)
}

// gcraView fills v from a helper result.
func (rc *resolvedConstraint) gcraView(res gcra.Result, v *view) {
	v.units = res.Remaining
	v.retryAt = rc.gcraRetryAt(res)
	v.used = res.Usage
	v.limit = int64(rc.limits.Limit)
	v.has = true
}

// snapshot reads one constraint into v.  nothing is written.  available is
// the grant so far, which a skipped GCRA constraint passes through.
func (m *Manager) snapshot(pc passCtx, rc *resolvedConstraint, v *view, available int) {
	switch rc.item.Kind {
	case constraintapi.ConstraintKindConcurrency:
		v.used = m.loadCell(&rc.usage, rc.usageKey)
		v.limit = int64(rc.limits.Limit)
		v.retryAt = pc.nowMS + constraintapi.ConcurrencyLimitRetryAfter.Milliseconds()
		v.has = true
		recompute(rc, v)
	case constraintapi.ConstraintKindSemaphore:
		v.used = m.loadCell(&rc.usage, rc.usageKey)
		v.limit = m.loadCell(&rc.capacity, rc.capacityKey)
		v.has = true
		recompute(rc, v)
	case constraintapi.ConstraintKindRateLimit, constraintapi.ConstraintKindThrottle:
		if pc.skipGCRA {
			*v = view{units: max(available, 1)}
			return
		}
		tat, present := m.loadGCRA(&rc.gcra, rc.stateKey, pc.nowMS)
		res, _, _, _ := rc.gcraStep(pc, tat, present, 0)
		rc.gcraView(res, v)
	}
}

// commit takes q units from one constraint and returns how many fit.  v is
// left as the constraint reads after the take.
func (m *Manager) commit(pc passCtx, rc *resolvedConstraint, q int, v *view) int {
	switch rc.item.Kind {
	case constraintapi.ConstraintKindSemaphore, constraintapi.ConstraintKindConcurrency:
		for {
			if fit, after, ok := rc.usage.Load().take(v.limit, rc.weight, q); ok {
				v.used = after
				return fit
			}
			rc.usage.Store(m.sem(rc.usageKey))
		}
	case constraintapi.ConstraintKindRateLimit, constraintapi.ConstraintKindThrottle:
		if pc.skipGCRA {
			*v = view{units: 1}
			return q
		}
		return m.commitGCRA(pc, rc, q, v)
	}
	return 0
}

// commitGCRA advances the TAT by q units.  when the helper refuses, the
// refusal is one of two things.  if the TAT is in the past, pass one saw an
// inflated remaining and acquire.lua grants anyway without writing; that is
// kept.  otherwise another request moved the TAT between the passes, and the
// take is retried with what fits now, down to nothing.
func (m *Manager) commitGCRA(pc passCtx, rc *resolvedConstraint, q int, v *view) int {
	for {
		var res gcra.Result
		ok := rc.gcra.Load().update(pc.nowMS, func(tat float64, present bool) (float64, bool, int64) {
			var newTAT float64
			var ttl int64
			var store bool
			res, newTAT, ttl, store = rc.gcraStep(pc, tat, present, q)
			return newTAT, store, ttl
		})
		if !ok {
			rc.gcra.Store(m.gcraCell(rc.stateKey))
			continue
		}
		rc.gcraView(res, v)
		if !res.Limited {
			return q
		}
		fitNow := 0
		if res.Next > -res.EmissionInterval {
			fitNow = int(math.Floor(res.Next / res.EmissionInterval))
		}
		if fitNow >= q || fitNow <= 0 {
			if fitNow >= q {
				return q
			}
			return 0
		}
		q = fitNow
	}
}

// rollback gives units back to one constraint after a later constraint
// shrank the grant.
func (m *Manager) rollback(pc passCtx, rc *resolvedConstraint, units int, v *view) {
	if units <= 0 {
		return
	}
	switch rc.item.Kind {
	case constraintapi.ConstraintKindSemaphore, constraintapi.ConstraintKindConcurrency:
		v.used = m.giveCell(&rc.usage, rc.usageKey, rc.weight*int64(units))
	case constraintapi.ConstraintKindRateLimit, constraintapi.ConstraintKindThrottle:
		if pc.skipGCRA {
			return
		}
		m.rollbackGCRA(pc, rc, units, v)
	}
}

// rollbackGCRA moves the TAT back by units emission intervals.  exact when
// nothing else wrote in between, otherwise off by what the other writer did.
func (m *Manager) rollbackGCRA(pc passCtx, rc *resolvedConstraint, units int, v *view) {
	now := float64(pc.nowMS)
	perSecond := 1_000.0
	if rc.item.Kind == constraintapi.ConstraintKindRateLimit {
		now = float64(pc.nowNS)
		perSecond = 1_000_000_000
	}
	var after float64
	for {
		ok := rc.gcra.Load().update(pc.nowMS, func(tat float64, present bool) (float64, bool, int64) {
			if !present {
				after = now
				return 0, false, 0
			}
			emission := rc.period / float64(max(rc.limits.Limit, 1))
			after = tat - float64(units)*emission
			return after, true, int64(math.Max((after-now)/perSecond, 1))
		})
		if ok {
			break
		}
		rc.gcra.Store(m.gcraCell(rc.stateKey))
	}
	res, _, _, _ := rc.gcraStep(pc, after, true, 0)
	rc.gcraView(res, v)
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

// loadGCRA reads a TAT cell through its pointer, replacing the pointer when
// housekeeping dropped the cell.
func (m *Manager) loadGCRA(p *atomic.Pointer[gcraCell], key cellKey, nowMS int64) (float64, bool) {
	for {
		c := p.Load()
		if c.alive() {
			return c.load(nowMS)
		}
		p.Store(m.gcraCell(key))
	}
}
