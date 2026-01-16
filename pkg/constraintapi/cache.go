package constraintapi

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/karlseguin/ccache/v3"
	"github.com/oklog/ulid/v2"
)

type limitingConstraintCache struct {
	manager CapacityManager
	clock   clockwork.Clock

	limitingConstraintCache        *ccache.Cache[*limitingConstraintCacheItem]
	limitingConstraintCacheTTLFunc func(c ConstraintItem) time.Duration
}

type limitingConstraintCacheItem struct {
	constraint ConstraintItem
	retryAfter time.Time
}

func DefaultLimitingConstraintCacheTTLFunc(c ConstraintItem) time.Duration {
	switch c.Kind {
	case ConstraintKindConcurrency:
		return time.Second
	case ConstraintKindThrottle:
		return time.Second
	case ConstraintKindRateLimit:
		return time.Second
	default:
		return time.Second
	}
}

// Acquire implements CapacityManager.
func (l *limitingConstraintCache) Acquire(ctx context.Context, req *CapacityAcquireRequest) (*CapacityAcquireResponse, errs.InternalError) {
	// Check if we previously got limited
	recentlyLimited := make([]ConstraintItem, 0)
	var retryAfter time.Time
	for _, ci := range req.Constraints {
		// Construct cache key for constraint scoped to account
		cacheKey := ci.CacheKey(req.AccountID, req.EnvID, req.FunctionID)

		item := l.limitingConstraintCache.Get(cacheKey)
		if item == nil || item.Expired() {
			// Not limited previously
			continue
		}

		// This constraint was recently limited
		val := item.Value()

		recentlyLimited = append(recentlyLimited, ci)
		if val.retryAfter.After(retryAfter) {
			retryAfter = val.retryAfter
		}
	}

	// If one or more requested constraints were recently limited,
	// return a synthetic response including all affected constraints.
	if len(recentlyLimited) > 0 {
		requestID, err := ulid.New(ulid.Timestamp(l.clock.Now()), rand.Reader)
		if err != nil {
			return nil, errs.Wrap(0, false, "could not generate request ID: %w", err)
		}

		return &CapacityAcquireResponse{
			RequestID:           requestID,
			Leases:              nil,
			LimitingConstraints: recentlyLimited,
			RetryAfter:          retryAfter,
		}, nil
	}

	res, err := l.manager.Acquire(ctx, req)
	if err != nil {
		return nil, err
	}

	// If we are limited by constraints,
	// cache each individual constraint for subsequent requests
	// for a short duration to avoid unnecessary load on Redis
	if len(res.Leases) == 0 && len(res.LimitingConstraints) > 0 {
		for _, ci := range res.LimitingConstraints {
			cacheKey := ci.CacheKey(req.AccountID, req.EnvID, req.FunctionID)

			retryDelay := retryAfter.Sub(l.clock.Now())
			if retryDelay <= 0 {
				retryDelay = time.Second
			}

			l.limitingConstraintCache.Set(
				cacheKey,
				&limitingConstraintCacheItem{
					retryAfter: retryAfter,
					constraint: ci,
				},
				retryDelay,
			)
		}
	}

	return res, nil
}

func (l *limitingConstraintCache) Check(ctx context.Context, req *CapacityCheckRequest) (*CapacityCheckResponse, errs.UserError, errs.InternalError) {
	return l.manager.Check(ctx, req)
}

func (l *limitingConstraintCache) ExtendLease(ctx context.Context, req *CapacityExtendLeaseRequest) (*CapacityExtendLeaseResponse, errs.InternalError) {
	return l.manager.ExtendLease(ctx, req)
}

func (l *limitingConstraintCache) Release(ctx context.Context, req *CapacityReleaseRequest) (*CapacityReleaseResponse, errs.InternalError) {
	return l.manager.Release(ctx, req)
}

func NewLimitingConstraintCache(
	clock clockwork.Clock,
	manager CapacityManager,
	ttlFunc func(c ConstraintItem) time.Duration,
) CapacityManager {
	return &limitingConstraintCache{
		manager: manager,

		limitingConstraintCache: ccache.New(
			ccache.Configure[*limitingConstraintCacheItem]().
				MaxSize(10_000).
				ItemsToPrune(500),
		),
		limitingConstraintCacheTTLFunc: ttlFunc,
	}
}
