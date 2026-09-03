package memory

import (
	"context"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/util/errs"
)

// ExtendLease implements constraintapi.CapacityManager.  the old slot is
// taken and a new one allocated, so a release with the old ID finds nothing.
func (m *Manager) ExtendLease(ctx context.Context, req *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	if err := req.Valid(); err != nil {
		return nil, errs.Wrap(0, false, "invalid request: %w", err)
	}

	now := m.clock.Now()
	nowMS := now.UnixMilli()
	leaseExpiryMS := now.Add(req.Duration).UnixMilli()
	extKey := opKey(req.AccountID, "ext", req.IdempotencyKey)

	mu := m.lock(extKey)
	cached, hit := m.extIdem.get(nowMS, extKey)
	if !hit {
		cached = m.extend(nowMS, leaseExpiryMS, req, extKey)
	}
	mu.Unlock()

	res := *cached
	res.OperationIdempotencyHit = hit

	if res.LeaseID != nil {
		for _, hook := range m.lifecycles {
			err := hook.OnCapacityLeaseExtended(ctx, constraintapi.OnCapacityLeaseExtendedData{
				AccountID:               req.AccountID,
				EnvID:                   res.EnvID,
				AppID:                   res.AppID,
				FunctionID:              res.FunctionID,
				Duration:                req.Duration,
				OldLeaseID:              req.LeaseID,
				NewLeaseID:              *res.LeaseID,
				Usage:                   res.Usage,
				OperationIdempotencyHit: res.OperationIdempotencyHit,
			})
			if err != nil {
				return nil, errs.Wrap(0, false, "extend lifecycle failed: %w", err)
			}
		}
	}

	return &res, nil
}

// extend is the body of one ExtendLease.  it runs once per idempotency key.
// an expired lease ID is rejected before any lookup, like extend.lua.
func (m *Manager) extend(nowMS, leaseExpiryMS int64, req *constraintapi.CapacityExtendLeaseRequest, extKey uint64) *constraintapi.CapacityExtendLeaseResponse {
	res := &constraintapi.CapacityExtendLeaseResponse{AccountID: req.AccountID}

	expiresAtMS, seq, nonce := decodeLeaseID(req.LeaseID)
	if expiresAtMS < nowMS {
		return res
	}
	if nonce != m.nonce {
		m.stats.foreign("extend")
		return res
	}
	sl := m.slab.get(seq)
	if sl == nil {
		return res
	}
	if sl.expiresAtMS != expiresAtMS {
		m.stats.foreign("extend")
		return res
	}
	rs := sl.req.Load()
	if rs == nil || rs.set.accountID != req.AccountID {
		return res
	}
	if !sl.take() {
		return res
	}

	newSeq, _ := m.slab.alloc(nowMS, leaseExpiryMS, rs)
	m.expiry.add(leaseExpiryMS, newSeq)
	sl.req.Store(nil)

	newID := encodeLeaseID(leaseExpiryMS, newSeq, m.nonce)
	res.LeaseID = &newID
	res.EnvID = rs.set.envID
	res.FunctionID = rs.set.functionID
	res.AppID = rs.set.appID

	// usage reports concurrency only, with the limit stored at acquire time,
	// like extend.lua.  the count does not change on extend.
	for i := range rs.set.constraints {
		rc := &rs.set.constraints[i]
		if rc.item.Kind == constraintapi.ConstraintKindConcurrency {
			res.Usage = append(res.Usage, constraintapi.ConstraintUsage{
				Constraint: rc.item,
				Used:       int(m.loadCell(&rc.usage, rc.usageKey)),
				Limit:      rc.limits.Limit,
			})
		}
	}

	m.extIdem.set(extKey, res, idemExpiry(nowMS, m.operationIdempotencyTTL))
	return res
}
