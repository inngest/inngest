package constraintapi

import (
	"context"
	"crypto/rand"
	mrand "math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/karlseguin/ccache/v3"
	"github.com/oklog/ulid/v2"
)

const (
	MinCacheTTL = time.Second
	MaxCacheTTL = time.Minute
)

type EnableConstraintCacheFn func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, minTTL, maxTTL time.Duration)

// ShouldCacheConstraintFn is a predicate function that determines whether a constraint should be cached.
// If nil, all constraints are cached (default behavior).
// If provided, only constraints for which this function returns true will be cached.
// The function is called for each constraint during both cache check and cache set operations.
type ShouldCacheConstraintFn func(ci ConstraintItem) bool

type constraintCache struct {
	manager CapacityManager
	clock   clockwork.Clock

	cache                                *ccache.Cache[*constraintCacheItem]
	maxSize                              int64
	itemsToPrune                         uint32
	enableHighCardinalityInstrumentation EnableHighCardinalityInstrumentation
	enableCache                          EnableConstraintCacheFn
	shouldCache                          ShouldCacheConstraintFn
}

type constraintCacheItem struct {
	constraint ConstraintItem
	retryAfter time.Time
	// addedAt records the wall-clock time at which this cache entry was created.
	// Callers can pass CapacityAcquireRequest.RequestTime to bypass entries that
	// were added after the underlying work was originally received, which prevents
	// silent drops on retries of work that predates a cached "exhausted" decision.
	addedAt time.Time
}

type ConstraintCacheOption func(c *constraintCache)

func WithConstraintCacheClock(clock clockwork.Clock) ConstraintCacheOption {
	return func(c *constraintCache) {
		c.clock = clock
	}
}

func WithConstraintCacheManager(manager CapacityManager) ConstraintCacheOption {
	return func(c *constraintCache) {
		c.manager = manager
	}
}

func WithConstraintCacheEnableHighCardinalityInstrumentation(ehci EnableHighCardinalityInstrumentation) ConstraintCacheOption {
	return func(c *constraintCache) {
		c.enableHighCardinalityInstrumentation = ehci
	}
}

func WithConstraintCacheEnable(enable EnableConstraintCacheFn) ConstraintCacheOption {
	return func(c *constraintCache) {
		c.enableCache = enable
	}
}

func WithConstraintCacheShouldCache(fn ShouldCacheConstraintFn) ConstraintCacheOption {
	return func(c *constraintCache) {
		c.shouldCache = fn
	}
}

func WithConstraintCacheMaxSize(maxSize int64) ConstraintCacheOption {
	return func(c *constraintCache) {
		c.maxSize = maxSize
	}
}

func WithConstraintCacheItemsToPrune(itemsToPrune uint32) ConstraintCacheOption {
	return func(c *constraintCache) {
		c.itemsToPrune = itemsToPrune
	}
}

// Acquire implements CapacityManager.
func (l *constraintCache) Acquire(ctx context.Context, req *CapacityAcquireRequest) (*CapacityAcquireResponse, errs.InternalError) {
	if l.enableCache == nil {
		return l.manager.Acquire(ctx, req)
	}

	enableCache, minTTL, maxTTL := l.enableCache(ctx, req.AccountID, req.EnvID, req.FunctionID)
	if !enableCache {
		return l.manager.Acquire(ctx, req)
	}

	// Report cache size gauge on ~1% of Acquire calls to avoid ItemCount() lock overhead at high volume.
	if mrand.Float64() < 0.01 {
		metrics.GaugeConstraintAPICacheSize(ctx, int64(l.cache.ItemCount()), metrics.GaugeOpt{
			PkgName: pkgName,
		})
	}

	// Check if any constraint is cached as exhausted
	recentlyLimited := make([]ConstraintItem, 0)
	var retryAfter time.Time

	// Return immediately on first cache hit since any exhausted constraint blocks the request
	for _, ci := range req.Constraints {
		// Skip constraints that don't pass the filter
		if l.shouldCache != nil && !l.shouldCache(ci) {
			continue
		}

		// Construct cache key for constraint scoped to account
		cacheKey := ci.CacheKey(req.AccountID, req.EnvID, req.FunctionID)
		if cacheKey == "" {
			continue
		}

		item := l.cache.Get(cacheKey)
		if item == nil || item.Expired() {
			// Not cached or expired
			continue
		}

		// Cache hit - this constraint is exhausted
		val := item.Value()

		// Skip the cache when the caller's request was originally received before
		// this cache entry was added. This is the silent-drop-on-retry guard: a
		// retry of work received at T0 should not be denied by a cache entry that
		// was populated at T1 > T0. We fall through to the underlying manager so
		// the actual constraint state is checked.
		if !req.RequestTime.IsZero() && val.addedAt.After(req.RequestTime) {
			tags := map[string]any{
				"op":              "skipped_stale",
				"source_location": req.Source.Location.String(),
				"source_service":  req.Source.Service.String(),
				"constraint":      ci.MetricsIdentifier(),
			}
			if l.enableHighCardinalityInstrumentation != nil && l.enableHighCardinalityInstrumentation(ctx, req.AccountID, req.EnvID, req.FunctionID) {
				tags["function_id"] = req.FunctionID
			}
			metrics.HistogramConstraintAPILimitingConstraintCacheTTL(ctx, item.TTL(), metrics.HistogramOpt{
				PkgName: pkgName,
				Tags:    tags,
			})
			continue
		}

		recentlyLimited = append(recentlyLimited, ci)
		if val.retryAfter.After(retryAfter) {
			retryAfter = val.retryAfter
		}

		tags := map[string]any{
			"op":              "hit",
			"source_location": req.Source.Location.String(),
			"source_service":  req.Source.Service.String(),
			"constraint":      ci.MetricsIdentifier(),
		}
		if l.enableHighCardinalityInstrumentation != nil && l.enableHighCardinalityInstrumentation(ctx, req.AccountID, req.EnvID, req.FunctionID) {
			tags["function_id"] = req.FunctionID
		}

		metrics.HistogramConstraintAPILimitingConstraintCacheTTL(ctx, item.TTL(), metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    tags,
		})

	}

	// If one or more requested constraints were recently limited,
	// return a synthetic response including all affected constraints.
	if len(recentlyLimited) > 0 {
		// Return immediately with synthetic response
		requestID, err := ulid.New(ulid.Timestamp(l.clock.Now()), rand.Reader)
		if err != nil {
			return nil, errs.Wrap(0, false, "could not generate request ID: %w", err)
		}

		return &CapacityAcquireResponse{
			RequestID:            requestID,
			Leases:               nil,
			ExhaustedConstraints: recentlyLimited,
			// Exhausted constraints are also limiting constraints (they reduce capacity to 0)
			LimitingConstraints: recentlyLimited,
			RetryAfter:          retryAfter,
		}, nil
	}

	// Cache miss - record metric
	tags := map[string]any{
		"op":              "miss",
		"source_location": req.Source.Location.String(),
		"source_service":  req.Source.Service.String(),
	}
	if l.enableHighCardinalityInstrumentation != nil && l.enableHighCardinalityInstrumentation(ctx, req.AccountID, req.EnvID, req.FunctionID) {
		tags["function_id"] = req.FunctionID
	}

	metrics.HistogramConstraintAPILimitingConstraintCacheTTL(ctx, 0, metrics.HistogramOpt{
		PkgName: pkgName,
		Tags:    tags,
	})

	res, err := l.manager.Acquire(ctx, req)
	if err != nil {
		return nil, err
	}

	// If we have exhausted constraints (constraints with zero remaining capacity),
	// cache each individual constraint for subsequent requests
	// for a short duration to avoid unnecessary load on Redis.
	// Exhausted constraints mean no further requests can succeed until capacity is freed.
	for _, ci := range res.ExhaustedConstraints {
		// Skip constraints that don't pass the filter
		if l.shouldCache != nil && !l.shouldCache(ci) {
			continue
		}

		cacheKey := ci.CacheKey(req.AccountID, req.EnvID, req.FunctionID)
		if cacheKey == "" {
			continue
		}

		cacheTTL := res.RetryAfter.Sub(l.clock.Now())
		if cacheTTL <= minTTL {
			cacheTTL = minTTL
		}

		// Enforce max cache ttl limit
		if cacheTTL >= maxTTL {
			cacheTTL = maxTTL
		}

		l.cache.Set(
			cacheKey,
			&constraintCacheItem{
				retryAfter: res.RetryAfter,
				constraint: ci,
				addedAt:    l.clock.Now(),
			},
			cacheTTL,
		)
		tags := map[string]any{
			"op":              "set",
			"source_location": req.Source.Location.String(),
			"source_service":  req.Source.Service.String(),
			"constraint":      ci.MetricsIdentifier(),
		}
		if l.enableHighCardinalityInstrumentation != nil && l.enableHighCardinalityInstrumentation(ctx, req.AccountID, req.EnvID, req.FunctionID) {
			tags["function_id"] = req.FunctionID
		}

		metrics.HistogramConstraintAPILimitingConstraintCacheTTL(ctx, cacheTTL, metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    tags,
		})
	}

	return res, nil
}

func (l *constraintCache) Check(ctx context.Context, req *CapacityCheckRequest) (*CapacityCheckResponse, errs.UserError, errs.InternalError) {
	return l.manager.Check(ctx, req)
}

func (l *constraintCache) ExtendLease(ctx context.Context, req *CapacityExtendLeaseRequest) (*CapacityExtendLeaseResponse, errs.InternalError) {
	return l.manager.ExtendLease(ctx, req)
}

func (l *constraintCache) Release(ctx context.Context, req *CapacityReleaseRequest) (*CapacityReleaseResponse, errs.InternalError) {
	return l.manager.Release(ctx, req)
}

func NewConstraintCache(
	options ...ConstraintCacheOption,
) *constraintCache {
	cache := &constraintCache{
		maxSize:      10_000,
		itemsToPrune: 500,
	}

	for _, opt := range options {
		opt(cache)
	}

	cache.cache = ccache.New(
		ccache.Configure[*constraintCacheItem]().
			MaxSize(cache.maxSize).
			ItemsToPrune(cache.itemsToPrune).
			OnDelete(func(item *ccache.Item[*constraintCacheItem]) {
				// Track cache evictions to detect thrashing (cache size too small).
				// OnDelete fires on the ccache worker goroutine for both LRU evictions
				// and explicit deletions. We use context.Background() since there's no
				// request context available here; OTEL SDK buffers these internally.
				expired := item.Expired()
				metrics.IncrConstraintAPICacheEvictedCounter(context.Background(), metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"expired": expired,
					},
				})
				if !expired {
					remainingTTL := item.TTL()
					if remainingTTL < 0 {
						remainingTTL = 0
					}
					metrics.HistogramConstraintAPICacheEvictedRemainingTTL(context.Background(), remainingTTL, metrics.HistogramOpt{
						PkgName: pkgName,
					})
				}
			}),
	)

	if cache.clock == nil {
		cache.clock = clockwork.NewRealClock()
	}

	return cache
}
