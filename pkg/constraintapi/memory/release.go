package memory

import (
	"context"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/util/errs"
)

type flightRelease struct {
	res      *constraintapi.CapacityReleaseResponse
	replayed bool
	// released is true when this flight took the slot and gave its counters
	// back.  a replay or a no-op leaves it false.
	released bool
}

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
	relKey := hashKey(req.AccountID, "rel", req.IdempotencyKey)

	executed := false
	v, _, _ := m.flight.Do(flightKey('r', relKey), func() (any, error) {
		executed = true
		if r, ok := m.relIdem.get(nowMS, relKey); ok {
			return flightRelease{res: r, replayed: true}, nil
		}
		res, released := m.release(nowMS, req, relKey)
		return flightRelease{res: res, released: released}, nil
	})
	fr := v.(flightRelease)

	res := *fr.res
	res.OperationIdempotencyHit = fr.replayed || !executed

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

	return &res, fr.released && executed, nil
}

// release is the body of one Release.  it runs once per idempotency key.
// the caller that takes the slot gives the counters back.  a lease from
// another manager, a freed page, or a slot already taken is a no-op.
func (m *Manager) release(nowMS int64, req *constraintapi.CapacityReleaseRequest, relKey uint64) (*constraintapi.CapacityReleaseResponse, bool) {
	res := &constraintapi.CapacityReleaseResponse{AccountID: req.AccountID}

	_, seq, nonce := decodeLeaseID(req.LeaseID)
	if nonce != m.nonce {
		return res, false
	}
	sl := m.slab.get(seq)
	if sl == nil || !sl.take() {
		return res, false
	}
	rs := sl.req

	// the sweeper reclaims a lease whose holder is gone, so a manual release
	// semaphore is given back too.  holding it would block every later run.
	force := req.Source.Location == constraintapi.CallerLocationLeaseScavenge

	var usage []constraintapi.ConstraintUsage
	for _, rc := range rs.constraints {
		switch rc.item.Kind {
		case constraintapi.ConstraintKindSemaphore:
			if rc.release == constraintapi.SemaphoreReleaseAuto || force {
				m.giveCell(&rc.usage, rc.usageKey, rc.weight)
			}
		}
	}

	rs.active.Add(-1)
	if p := m.slab.page(seq); p != nil {
		p.live.Add(-1)
	}

	res.EnvID = rs.envID
	res.FunctionID = rs.functionID
	res.AppID = rs.appID
	res.CreationSource = rs.source
	res.Usage = usage

	m.relIdem.set(relKey, res, idemExpiry(nowMS, m.operationIdempotencyTTL))
	return res, true
}
