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
	var batch *Batch
	if !hit {
		cached, released, batch = m.release(nowMS, req, relKey)
	}
	mu.Unlock()

	// a release does not wait unless the request asks for it.  a lost
	// release keeps the capacity reserved until the lease expires.
	if req.CommitLevel > constraintapi.CommitLevelPrimary {
		if hit {
			batch = m.replayBatch()
		}
		if ierr := m.awaitCommit(ctx, batch, req.CommitLevel, constraintapi.CommitLevelPrimary); ierr != nil {
			return nil, false, ierr
		}
	}

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
func (m *Manager) release(nowMS int64, req *constraintapi.CapacityReleaseRequest, relKey uint64) (*constraintapi.CapacityReleaseResponse, bool, *Batch) {
	res := &constraintapi.CapacityReleaseResponse{AccountID: req.AccountID}

	expiresAtMS, seq, nonce := decodeLeaseID(req.LeaseID)
	if nonce != m.nonce {
		m.stats.foreign("release")
		return res, false, nil
	}
	sl := m.slab.get(seq)
	if sl == nil {
		return res, false, nil
	}
	if sl.expiresAtMS != expiresAtMS {
		// the seq names a live slot but the ID was not issued for it: a lease
		// from another manager that happens to share the nonce
		m.stats.foreign("release")
		return res, false, nil
	}
	rs := sl.req.Load()
	if rs == nil || rs.set.accountID != req.AccountID {
		return res, false, nil
	}
	if !sl.take() {
		return res, false, nil
	}
	set := rs.set

	// the sweeper reclaims a lease whose holder is gone, so a manual release
	// semaphore is given back too.  holding it would block every later run.
	force := req.Source.Location == constraintapi.CallerLocationLeaseScavenge

	l := m.log.Load()
	var batch *Batch
	if l != nil {
		batch = l.append(rs.route, Record{
			Kind:  RecTake,
			Seq:   seq,
			Force: force,
		})
	}

	// usage reports concurrency only, with the limit stored at acquire time
	// and the count after this release, like release.lua
	var usage []constraintapi.ConstraintUsage
	var short [constraintapi.MaxConstraints]int64
	if m.giveBack(set, force, &usage, &short) && l != nil {
		// a follower subtracts the full weight for a RecTake.  when a
		// counter was already at zero here, the difference is logged so the
		// follower ends at the same value.
		for i := range set.constraints {
			if short[i] == 0 {
				continue
			}
			key := set.constraints[i].usageKey
			batch = l.append(rs.route, Record{
				Kind:  RecDelta,
				KeyA:  key.a,
				KeyB:  key.b,
				Value: short[i],
			})
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
	return res, true, batch
}

// giveBack returns what one lease of set took from every counter.  force
// gives manual release semaphores back too.  usage gets one entry per
// concurrency constraint with the count after the return.  short[i] is how
// much constraint i could not give back because its counter was already at
// zero, and clamped is true when any entry is not zero.  takeBack is the
// same loop for a follower.
func (m *Manager) giveBack(set *constraintSet, force bool, usage *[]constraintapi.ConstraintUsage, short *[constraintapi.MaxConstraints]int64) (clamped bool) {
	for i := range set.constraints {
		rc := &set.constraints[i]
		switch rc.item.Kind {
		case constraintapi.ConstraintKindConcurrency:
			after, applied := m.giveCell(&rc.usage, rc.usageKey, 1)
			short[i] = 1 - applied
			*usage = append(*usage, constraintapi.ConstraintUsage{
				Constraint: rc.item,
				Used:       int(after),
				Limit:      rc.limits.Limit,
			})
		case constraintapi.ConstraintKindSemaphore:
			if rc.release == constraintapi.SemaphoreReleaseAuto || force {
				_, applied := m.giveCell(&rc.usage, rc.usageKey, rc.weight)
				short[i] = rc.weight - applied
			}
		}
		if short[i] != 0 {
			clamped = true
		}
	}
	return clamped
}

// takeBack is giveBack for a follower that applies a RecTake.  it subtracts
// the full weight with no clamp.  the primary logs a RecDelta for whatever
// it could not subtract.
func (m *Manager) takeBack(set *constraintSet, force bool) {
	for i := range set.constraints {
		rc := &set.constraints[i]
		switch rc.item.Kind {
		case constraintapi.ConstraintKindConcurrency:
			m.addCell(&rc.usage, rc.usageKey, -1)
		case constraintapi.ConstraintKindSemaphore:
			if rc.release == constraintapi.SemaphoreReleaseAuto || force {
				m.addCell(&rc.usage, rc.usageKey, -rc.weight)
			}
		}
	}
}
