package constraintapi

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/karlseguin/ccache/v3"
	"github.com/oklog/ulid/v2"
)

const (
	LimitingConstraintCacheTTLConcurrency = 5 * time.Second
	LimitingConstraintCacheTTLThrottle    = time.Second
	MaxCacheTTL                           = time.Minute
)

type limitingConstraintCache struct {
	manager CapacityManager
	clock   clockwork.Clock

	limitingConstraintCache *ccache.Cache[*limitingConstraintCacheItem]
}

type limitingConstraintCacheItem struct {
	constraint ConstraintItem
	retryAfter time.Time
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

		metrics.IncrConstraintAPILimitingConstraintCacheCounter(ctx, 1, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"op": "hit",
			},
		})
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
	for _, ci := range res.LimitingConstraints {
		cacheKey := ci.CacheKey(req.AccountID, req.EnvID, req.FunctionID)

		retryDelay := retryAfter.Sub(l.clock.Now())
		if retryDelay <= 0 {
			retryDelay = time.Second
		}

		// Enforce max cache ttl limit
		if retryDelay >= MaxCacheTTL {
			retryDelay = MaxCacheTTL
		}

		l.limitingConstraintCache.Set(
			cacheKey,
			&limitingConstraintCacheItem{
				retryAfter: retryAfter,
				constraint: ci,
			},
			retryDelay,
		)
		metrics.IncrConstraintAPILimitingConstraintCacheCounter(ctx, 1, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"op": "set",
			},
		})
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
) *limitingConstraintCache {
	return &limitingConstraintCache{
		manager: manager,
		clock:   clock,
		limitingConstraintCache: ccache.New(
			ccache.Configure[*limitingConstraintCacheItem]().
				MaxSize(10_000).
				ItemsToPrune(500),
		),
	}
}
