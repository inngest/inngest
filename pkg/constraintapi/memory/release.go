package memory

import (
	"context"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/util/errs"
)

// Release implements constraintapi.CapacityManager.  it is idempotent by
// lease ID and never errors for an unknown lease.
func (m *Manager) Release(ctx context.Context, req *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	res, _, err := m.doRelease(ctx, req)
	return res, err
}

// doRelease is Release plus whether this call reclaimed the lease, which the
// sweeper counts.
func (m *Manager) doRelease(ctx context.Context, req *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, bool, errs.InternalError) {
	if err := req.Valid(); err != nil {
		return nil, false, errs.Wrap(0, false, "invalid request: %w", err)
	}

	nowMS := m.nowMS()
	relKey := opKey(req.AccountID, "rel", req.IdempotencyKey)

	mu := m.lock(relKey)
	cached, hit := m.relIdem.get(nowMS, relKey)
	var released bool
	if !hit {
		cached, released = m.release(nowMS, req, relKey)
	}
	mu.Unlock()

	res := *cached
	res.OperationIdempotencyHit = hit

	for _, hook := range m.lifecycles {
		err := hook.OnCapacityLeaseReleased(ctx, constraintapi.OnCapacityLeaseReleasedData{
			AccountID:               req.AccountID,
			EnvID:                   res.EnvID,
			AppID:                   res.AppID,
			FunctionID:              res.FunctionID,
			LeaseID:                 req.LeaseID,
			Usage:                   res.Usage,
			OperationIdempotencyHit: res.OperationIdempotencyHit,
		})
		if err != nil {
			return nil, false, errs.Wrap(0, false, "release lifecycle failed: %w", err)
		}
	}

	return &res, released, nil
}

// release is the body of one Release.  it runs once per idempotency key.
// the caller that takes the slot gives the counters back.  a lease from
// another manager, a freed page, or a slot already taken is a no-op.
func (m *Manager) release(nowMS int64, req *constraintapi.CapacityReleaseRequest, relKey uint64) (*constraintapi.CapacityReleaseResponse, bool) {
	res := &constraintapi.CapacityReleaseResponse{AccountID: req.AccountID}

	expiresAtMS, seq, nonce := decodeLeaseID(req.LeaseID)
	if nonce != m.nonce {
		m.stats.foreign("release")
		return res, false
	}
	sl := m.slab.get(seq)
	if sl == nil {
		return res, false
	}
	if sl.expiresAtMS != expiresAtMS {
		// the seq names a live slot but the ID was not issued for it: a lease
		// from another manager that happens to share the nonce
		m.stats.foreign("release")
		return res, false
	}
	rs := sl.req.Load()
	if rs == nil || rs.set.accountID != req.AccountID {
		return res, false
	}
	if !sl.take() {
		return res, false
	}
	set := rs.set

	// the sweeper reclaims a lease whose holder is gone, so a manual release
	// semaphore is given back too.  holding it would block every later run.
	force := req.Source.Location == constraintapi.CallerLocationLeaseScavenge

	// usage reports concurrency only, with the limit stored at acquire time
	// and the count after this release, like release.lua
	var usage []constraintapi.ConstraintUsage
	for i := range set.constraints {
		rc := &set.constraints[i]
		switch rc.item.Kind {
		case constraintapi.ConstraintKindConcurrency:
			after := m.giveCell(&rc.usage, rc.usageKey, 1)
			usage = append(usage, constraintapi.ConstraintUsage{Constraint: rc.item, Used: int(after), Limit: rc.limits.Limit})
		case constraintapi.ConstraintKindSemaphore:
			if rc.release == constraintapi.SemaphoreReleaseAuto || force {
				m.giveCell(&rc.usage, rc.usageKey, rc.weight)
			}
		}
	}

	rs.active.Add(-1)
	sl.req.Store(nil)

	res.EnvID = set.envID
	res.FunctionID = set.functionID
	res.AppID = set.appID
	res.CreationSource = rs.source
	res.Usage = usage

	m.relIdem.set(relKey, res, idemExpiry(nowMS, m.operationIdempotencyTTL))
	return res, true
}
