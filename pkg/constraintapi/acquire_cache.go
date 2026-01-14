package constraintapi

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/karlseguin/ccache/v3"
)

// acquireResponseCache caches lacking capacity Acquire responses for a short TTL
// to reduce Redis load from burst duplicate requests.
type acquireResponseCache struct {
	cache   *ccache.Cache[*CapacityAcquireResponse]
	ttl     time.Duration
	enabled bool
}

// generateAcquireCacheKey generates a cache key for the given constraint.
// Returns empty string if the constraint is not cacheable.
func (r *redisCapacityManager) generateAcquireCacheKey(
	accountID uuid.UUID,
	functionID uuid.UUID,
	constraint *ConstraintItem,
) string {
	if constraint.Kind != ConstraintKindConcurrency || constraint.Concurrency == nil {
		return ""
	}

	cc := constraint.Concurrency

	// Only cache for account/function concurrency WITHOUT custom keys
	if cc.KeyExpressionHash != "" {
		return ""
	}

	switch cc.Scope {
	case enums.ConcurrencyScopeAccount:
		return fmt.Sprintf("acq:a:%s", accountID)
	case enums.ConcurrencyScopeFn:
		return fmt.Sprintf("acq:f:%s", functionID)
	default:
		return ""
	}
}

// shouldCacheAcquireResponse determines if a response should be cached.
func (r *redisCapacityManager) shouldCacheAcquireResponse(
	resp *CapacityAcquireResponse,
) bool {
	// Condition 1: Status must be 2 (lacking capacity)
	if resp.internalDebugState.Status != 2 {
		return false
	}

	// Condition 2: No leases generated
	if len(resp.Leases) != 0 {
		return false
	}

	// Condition 3: At least one limiting constraint
	if len(resp.LimitingConstraints) == 0 {
		return false
	}

	// Condition 4: Check if any limiting constraint is cacheable
	for _, constraint := range resp.LimitingConstraints {
		if constraint.Kind != ConstraintKindConcurrency {
			continue
		}

		if constraint.Concurrency == nil {
			continue
		}

		cc := constraint.Concurrency

		// Must be account or function scope without custom key
		if cc.KeyExpressionHash != "" {
			continue
		}

		if cc.Scope == enums.ConcurrencyScopeAccount ||
			cc.Scope == enums.ConcurrencyScopeFn {
			return true
		}
	}

	return false
}

// WithAcquireResponseCache enables caching of lacking capacity Acquire responses.
// The cache reduces Redis load for near-simultaneous duplicate requests when
// account or function concurrency limits (without custom keys) are reached.
//
// A TTL of 0 disables caching (default).
// Recommended TTL: 1ms (catches burst duplicates without stale data risk).
//
// The cache stores responses under account-level keys for account concurrency
// and function-level keys for function concurrency constraints.
func WithAcquireResponseCache(ttl time.Duration) RedisCapacityManagerOption {
	return func(m *redisCapacityManager) {
		if ttl > 0 {
			m.acquireResponseCache = &acquireResponseCache{
				cache: ccache.New(ccache.Configure[*CapacityAcquireResponse]().
					MaxSize(10_000).
					ItemsToPrune(500)),
				ttl:     ttl,
				enabled: true,
			}
		} else {
			m.acquireResponseCache = &acquireResponseCache{
				enabled: false,
			}
		}
	}
}
