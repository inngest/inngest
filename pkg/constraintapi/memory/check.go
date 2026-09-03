package memory

import (
	"context"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/util/errs"
)

// Check implements constraintapi.CapacityManager.  it reads every constraint
// the way check.lua does and writes nothing: the first constraint is always
// limiting, evaluation stops once capacity reaches zero, and retry after comes
// only from exhausted constraints.  check.lua has no semaphore branch and
// reports one as exhausted; here a semaphore is read like any counter.  the
// Redis check idempotency record is write only and is not kept.
func (m *Manager) Check(ctx context.Context, req *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	if err := req.Valid(); err != nil {
		return nil, nil, errs.Wrap(0, false, "invalid request: %w", err)
	}

	now := m.clock.Now()
	pc := passCtx{nowMS: now.UnixMilli(), nowNS: now.UnixNano()}

	var arr [512]byte
	b, ierr := appendSetIdentity(arr[:0], req.AccountID, req.EnvID, req.FunctionID, uuid.Nil, req.Configuration, req.Constraints)
	if ierr != nil {
		return nil, nil, ierr
	}
	set := m.set(cellKey{a: xxhash.Sum64(b), b: sumSeeded(b, cellSeed)}, func() *constraintSet {
		return m.buildSet(req.AccountID, req.EnvID, req.FunctionID, uuid.Nil, req.Configuration, req.Constraints)
	})
	item := func(sorted int) constraintapi.ConstraintItem {
		if oi := int(set.order[sorted]); oi < len(req.Constraints) {
			return req.Constraints[oi]
		}
		return set.constraints[sorted].item
	}

	var limiting, exhausted []constraintapi.ConstraintItem
	usage := make([]constraintapi.ConstraintUsage, 0, len(set.constraints))
	var retryAt int64
	available := 0
	for i := range set.constraints {
		if i > 0 && available <= 0 {
			break
		}
		rc := &set.constraints[i]
		var v view
		m.snapshot(pc, rc, &v, available)

		// check.lua reports rate limit usage raw and derives the others from
		// the remaining capacity
		used := v.used
		switch rc.item.Kind {
		case constraintapi.ConstraintKindThrottle, constraintapi.ConstraintKindConcurrency:
			used = max(min(v.limit-int64(v.units), v.limit), 0)
		}
		usage = append(usage, constraintapi.ConstraintUsage{Constraint: item(i), Used: int(used), Limit: int(v.limit)})

		if v.units <= 0 {
			exhausted = append(exhausted, item(i))
			if v.retryAt > retryAt {
				retryAt = v.retryAt
			}
		}
		if i == 0 || v.units < available {
			available = v.units
			limiting = append(limiting, item(i))
		}
	}

	retryAfter := time.UnixMilli(retryAt)
	if retryAfter.Before(now) {
		retryAfter = time.Time{}
	}

	return &constraintapi.CapacityCheckResponse{
		AvailableCapacity:    available,
		LimitingConstraints:  limiting,
		ExhaustedConstraints: exhausted,
		Usage:                usage,
		RetryAfter:           retryAfter,
	}, nil, nil
}
