package memory

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/oklog/ulid/v2"
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
	constraints []*resolvedConstraint

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
	usageKey string
	usage    atomic.Pointer[semaphoreCell]

	// capacity is the semaphore capacity counter.  unset for other kinds.
	capacityKey string
	capacity    atomic.Pointer[semaphoreCell]

	// weight is the semaphore weight, at least 1.
	weight int64
	// release is the semaphore release mode.
	release constraintapi.SemaphoreReleaseMode
}

// acquireResult is the immutable outcome one Acquire computed.  every caller
// sharing the flight or replaying it builds its own response from this.
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

type flightAcquire struct {
	res      *acquireResult
	replayed bool
}

// fingerprint is hashed into the acquire idempotency key so a retry with a
// different configuration or constraint set is a new request.
type fingerprint struct {
	IdempotencyKey string                                   `json:"k"`
	EnvID          uuid.UUID                                `json:"e"`
	FunctionID     uuid.UUID                                `json:"f"`
	AppID          uuid.UUID                                `json:"ai"`
	Constraints    []constraintapi.SerializedConstraintItem `json:"s"`
	ConfigVersion  int                                      `json:"cv"`
	Requested      int                                      `json:"r"`
	MaxLifetimeMS  int64                                    `json:"l"`
	LeaseKeys      []string                                 `json:"lik"`
	RunIDs         map[string]ulid.ULID                     `json:"lri,omitempty"`
	Source         constraintapi.LeaseSource                `json:"m"`
}

// resolve sorts the request's constraints in place, looks up their counters
// and limits, and hashes the request fingerprint.
func (m *Manager) resolve(req *constraintapi.CapacityAcquireRequest) (*requestState, string, errs.InternalError) {
	constraintapi.SortConstraints(req.Constraints)

	rs := &requestState{
		accountID:       req.AccountID,
		envID:           req.EnvID,
		functionID:      req.FunctionID,
		appID:           req.AppID,
		constraints:     make([]*resolvedConstraint, 0, len(req.Constraints)),
		configVersion:   req.Configuration.FunctionVersion,
		requestedAmount: req.Amount,
		maximumLifetime: req.MaximumLifetime,
		source:          req.Source,
	}
	fp := fingerprint{
		IdempotencyKey: req.IdempotencyKey,
		EnvID:          req.EnvID,
		FunctionID:     req.FunctionID,
		AppID:          req.AppID,
		Constraints:    make([]constraintapi.SerializedConstraintItem, 0, len(req.Constraints)),
		ConfigVersion:  req.Configuration.FunctionVersion,
		Requested:      req.Amount,
		MaxLifetimeMS:  req.MaximumLifetime.Milliseconds(),
		LeaseKeys:      req.LeaseIdempotencyKeys,
		RunIDs:         req.LeaseRunIDs,
		Source:         req.Source,
	}

	for _, ci := range req.Constraints {
		rc := &resolvedConstraint{item: ci, limits: ci.ResolveLimits(req.Configuration)}
		switch ci.Kind {
		case constraintapi.ConstraintKindSemaphore:
			rc.usageKey = ci.Semaphore.UsageKey(req.AccountID)
			rc.capacityKey = ci.Semaphore.CapacityKey(req.AccountID)
			rc.weight = ci.Semaphore.Weight
			if rc.weight <= 0 {
				rc.weight = 1
			}
			rc.release = ci.Semaphore.Release
			rc.usage.Store(m.sem(rc.usageKey))
			rc.capacity.Store(m.sem(rc.capacityKey))
		default:
			return nil, "", errs.Wrap(0, false, "constraint kind %q is not implemented in the memory store", ci.Kind)
		}
		rs.constraints = append(rs.constraints, rc)
		fp.Constraints = append(fp.Constraints, ci.ToSerializedConstraintItem(req.Configuration, req.AccountID, req.EnvID, req.FunctionID))
	}

	b, err := json.Marshal(fp)
	if err != nil {
		return nil, "", errs.Wrap(0, false, "could not fingerprint request: %w", err)
	}
	return rs, strconv.FormatUint(xxhash.Sum64(b), 16), nil
}

// Acquire implements constraintapi.CapacityManager.
func (m *Manager) Acquire(ctx context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	requestID, err := ulid.New(ulid.Timestamp(m.clock.Now()), rand.Reader)
	if err != nil {
		return nil, errs.Wrap(0, false, "could not generate request ID: %w", err)
	}
	if err := req.Valid(); err != nil {
		return nil, errs.Wrap(0, false, "invalid request: %w", err)
	}

	now := m.clock.Now()
	nowMS := now.UnixMilli()

	metrics.HistogramConstraintAPIRequestLatency(ctx, now.Sub(req.CurrentTime), metrics.HistogramOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"operation": "acquire",
			"attempt":   req.RequestAttempt,
			"shard":     m.shardName,
		},
	})

	highCardinality := m.enableHighCardinalityInstrumentation != nil && m.enableHighCardinalityInstrumentation(ctx, req.AccountID, req.EnvID, req.FunctionID)
	leaseExpiryMS := now.Add(req.Duration).UnixMilli()

	rs, fp, ierr := m.resolve(req)
	if ierr != nil {
		return nil, ierr
	}
	acqKey := hashKey(req.AccountID, "acq", req.IdempotencyKey, fp)

	// only the caller whose closure runs performed the operation.  every
	// other caller in the flight, and every replay, reports an idempotency hit
	// the way a second Lua call does.
	executed := false
	v, _, _ := m.flight.Do(flightKey('a', acqKey), func() (any, error) {
		executed = true
		if r, ok := m.acqIdem.get(nowMS, acqKey); ok {
			return flightAcquire{res: r, replayed: true}, nil
		}
		return flightAcquire{res: m.acquire(nowMS, leaseExpiryMS, req, rs, acqKey)}, nil
	})
	fa := v.(flightAcquire)
	res := fa.res
	hit := fa.replayed || !executed

	leases := make([]constraintapi.CapacityLease, len(res.leases))
	copy(leases, res.leases)

	retryAfter := time.UnixMilli(res.retryAtMS)
	if retryAfter.Before(now) {
		retryAfter = time.Time{}
	}

	metrics.HistogramConstraintAPIRetryAfterDuration(ctx, max(retryAfter.Sub(now), 0), metrics.HistogramOpt{
		PkgName: pkgName,
		Tags:    m.sourceTags(req.Source),
	})

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

	tags := map[string]any{"shard": m.shardName}
	if highCardinality {
		tags["function_id"] = req.FunctionID
	}
	metrics.IncrConstraintAPILeasesRequestedCounter(ctx, int64(res.requested), metrics.CounterOpt{PkgName: "constraintapi", Tags: tags})
	metrics.IncrConstraintAPILeasesGrantedCounter(ctx, int64(res.granted), metrics.CounterOpt{PkgName: "constraintapi", Tags: tags})
	for _, constraint := range res.limitingConstraints {
		tags["limiting_constraint"] = constraint.MetricsIdentifier()
		metrics.IncrConstraintAPILimitingConstraintsCounter(ctx, metrics.CounterOpt{PkgName: "constraintapi", Tags: tags})
	}
	delete(tags, "limiting_constraint")
	for _, constraint := range res.exhaustedConstraints {
		tags["constraint"] = constraint.MetricsIdentifier()
		metrics.IncrConstraintAPIExhaustedConstraintsCounter(ctx, metrics.CounterOpt{PkgName: "constraintapi", Tags: tags})
	}
	delete(tags, "constraint")

	if res.status == 3 {
		metrics.IncrConstraintAPIIssuedLeaseCounter(ctx, int64(len(leases)), metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    m.sourceTags(req.Source),
		})
	}

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

func (m *Manager) sourceTags(source constraintapi.LeaseSource) map[string]any {
	return map[string]any{
		"location":            source.Location.String(),
		"service":             source.Service.String(),
		"run_processing_mode": source.RunProcessingMode.String(),
		"shard":               m.shardName,
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
	requested := req.Amount

	var limiting, exhausted []int
	limitingSet := make([]bool, n)
	exhaustedSet := make([]bool, n)
	var retryAt int64
	usage := make([]constraintapi.ConstraintUsage, 0, n)

	markExhausted := func(i int, at int64) {
		if !exhaustedSet[i] {
			exhaustedSet[i] = true
			exhausted = append(exhausted, i)
		}
		if at > retryAt {
			retryAt = at
		}
	}
	markLimiting := func(i int) {
		if !limitingSet[i] {
			limitingSet[i] = true
			limiting = append(limiting, i)
		}
	}
	report := func() {
		retryAt = 0
		usage = usage[:0]
		for i, rc := range rs.constraints {
			capacity, at, u := m.snapshot(rc)
			usage = append(usage, u)
			if capacity <= 0 {
				markExhausted(i, at)
			}
		}
	}
	result := func(status, granted int, leases []constraintapi.CapacityLease) *acquireResult {
		return &acquireResult{
			status:               status,
			requested:            requested,
			granted:              granted,
			leases:               leases,
			limitingConstraints:  m.items(rs, limiting),
			exhaustedConstraints: m.items(rs, exhausted),
			usage:                usage,
			retryAtMS:            retryAt,
		}
	}

	available := requested
	for i, rc := range rs.constraints {
		capacity, at, u := m.snapshot(rc)
		usage = append(usage, u)
		if capacity <= 0 {
			markExhausted(i, at)
		}
		if capacity < available {
			available = capacity
			markLimiting(i)
		}
	}
	if available <= 0 {
		return result(2, 0, nil)
	}

	granted := available
	taken := make([]int, n)
	for i, rc := range rs.constraints {
		fit := m.commit(rc, granted)
		taken[i] = fit
		if fit < granted {
			for j := 0; j < i; j++ {
				m.rollback(rs.constraints[j], taken[j]-fit)
				taken[j] = fit
			}
			granted = fit
			markLimiting(i)
		}
	}
	if granted == 0 {
		report()
		return result(2, 0, nil)
	}

	report()

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
		m.checkIdem.set(hashKey(req.AccountID, "cc", key), struct{}{}, ccExpiry)
	}
	m.checkIdem.set(hashKey(req.AccountID, "cc", req.IdempotencyKey), struct{}{}, ccExpiry)
	rs.grantedAmount = granted
	rs.active.Store(int64(granted))

	res := result(3, granted, leases)
	m.acqIdem.set(acqKey, res, idemExpiry(nowMS, min(m.operationIdempotencyTTL, req.Duration)))
	return res
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
func (m *Manager) loadCell(p *atomic.Pointer[semaphoreCell], key string) int64 {
	for {
		if v, ok := p.Load().load(); ok {
			return v
		}
		p.Store(m.sem(key))
	}
}

// giveCell subtracts w from a counter through its pointer, replacing the
// pointer when housekeeping dropped the counter.
func (m *Manager) giveCell(p *atomic.Pointer[semaphoreCell], key string, w int64) int64 {
	for {
		if v, ok := p.Load().give(w); ok {
			return v
		}
		p.Store(m.sem(key))
	}
}
