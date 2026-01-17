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
	MaxCacheTTL = time.Minute
)

type limitingConstraintCache struct {
	manager CapacityManager
	clock   clockwork.Clock

	limitingConstraintCache              *ccache.Cache[*limitingConstraintCacheItem]
	enableHighCardinalityInstrumentation EnableHighCardinalityInstrumentation
}

type limitingConstraintCacheItem struct {
	constraint ConstraintItem
	retryAfter time.Time
}

type LimitingConstraintCacheOption func(c *limitingConstraintCache)

func WithLimitingCacheClock(clock clockwork.Clock) LimitingConstraintCacheOption {
	return func(c *limitingConstraintCache) {
		c.clock = clock
	}
}

func WithLimitingCacheManager(manager CapacityManager) LimitingConstraintCacheOption {
	return func(c *limitingConstraintCache) {
		c.manager = manager
	}
}

func WithLimitingCacheEnableHighCardinalityInstrumentation(ehci EnableHighCardinalityInstrumentation) LimitingConstraintCacheOption {
	return func(c *limitingConstraintCache) {
		c.enableHighCardinalityInstrumentation = ehci
	}
}

// Acquire implements CapacityManager.
func (l *limitingConstraintCache) Acquire(ctx context.Context, req *CapacityAcquireRequest) (*CapacityAcquireResponse, errs.InternalError) {
	// Check if we previously got limited
	{
		recentlyLimited := make([]ConstraintItem, 0)
		var retryAfter time.Time
		for _, ci := range req.Constraints {
			// Construct cache key for constraint scoped to account
			cacheKey := ci.CacheKey(req.AccountID, req.EnvID, req.FunctionID)
			if cacheKey == "" {
				continue
			}

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

			tags := map[string]any{
				"op": "hit",
			}
			if l.enableHighCardinalityInstrumentation != nil && l.enableHighCardinalityInstrumentation(ctx, req.AccountID, req.EnvID, req.FunctionID) {
				tags["function_id"] = req.FunctionID
			}

			metrics.IncrConstraintAPILimitingConstraintCacheCounter(ctx, 1, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    tags,
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
		if cacheKey == "" {
			continue
		}

		cacheTTL := res.RetryAfter.Sub(l.clock.Now())
		if cacheTTL <= 0 {
			cacheTTL = time.Second
		}

		// Enforce max cache ttl limit
		if cacheTTL >= MaxCacheTTL {
			cacheTTL = MaxCacheTTL
		}

		l.limitingConstraintCache.Set(
			cacheKey,
			&limitingConstraintCacheItem{
				retryAfter: res.RetryAfter,
				constraint: ci,
			},
			cacheTTL,
		)
		tags := map[string]any{
			"op": "set",
		}
		if l.enableHighCardinalityInstrumentation != nil && l.enableHighCardinalityInstrumentation(ctx, req.AccountID, req.EnvID, req.FunctionID) {
			tags["function_id"] = req.FunctionID
		}

		metrics.IncrConstraintAPILimitingConstraintCacheCounter(ctx, 1, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    tags,
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
	options ...LimitingConstraintCacheOption,
) *limitingConstraintCache {
	cache := &limitingConstraintCache{
		limitingConstraintCache: ccache.New(
			ccache.Configure[*limitingConstraintCacheItem]().
				MaxSize(10_000).
				ItemsToPrune(500),
		),
	}

	for _, opt := range options {
		opt(cache)
	}

	if cache.clock == nil {
		cache.clock = clockwork.NewRealClock()
	}

	return cache
}
