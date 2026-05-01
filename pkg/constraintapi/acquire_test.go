package constraintapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSetup creates a common test setup for acquire cache tests.
func newTestSetup(t *testing.T, enableCache EnableAcquireCacheFn) (*redisCapacityManager, *miniredis.Miniredis, *clockwork.FakeClock, context.Context) {
	t.Helper()

	r := miniredis.RunT(t)
	ctx := context.Background()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(rc.Close)

	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))
	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
	ctx = logger.WithStdlib(ctx, l)

	opts := []RedisCapacityManagerOption{
		WithClient(rc),
		WithShardName("test"),
		WithClock(clock),
		WithEnableDebugLogs(true),
		WithCheckIdempotencyTTL(0),
	}
	if enableCache != nil {
		opts = append(opts, WithEnableAcquireCache(enableCache), WithAcquireCacheTTL(func(_ context.Context, _, _, _ uuid.UUID) (time.Duration, time.Duration) {
			return MinCacheTTL, MaxCacheTTL
		}))
	}

	cm, err := NewRedisCapacityManager(opts...)
	require.NoError(t, err)

	return cm, r, clock, ctx
}

// enableAllCache returns an EnableAcquireCacheFn that enables caching for all constraints.
func enableAllCache() EnableAcquireCacheFn {
	return func(_ context.Context, _, _, _ uuid.UUID, _ ConstraintItem) bool {
		return true
	}
}

func makeAcquireRequest(accountID, envID, fnID uuid.UUID, clock clockwork.Clock, config ConstraintConfig, constraints []ConstraintItem, idempotencyKey string) *CapacityAcquireRequest {
	return &CapacityAcquireRequest{
		IdempotencyKey:       idempotencyKey,
		AccountID:            accountID,
		EnvID:                envID,
		FunctionID:           fnID,
		Duration:             5 * time.Second,
		Configuration:        config,
		Constraints:          constraints,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"item0"},
		CurrentTime:          clock.Now(),
		MaximumLifetime:      time.Minute,
		Source: LeaseSource{
			Service:  ServiceAPI,
			Location: CallerLocationBacklogRefill,
		},
	}
}

func TestAcquireCachePreExistingKey(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, r, clock, ctx := newTestSetup(t, enableAllCache())

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 1,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Pre-set a cache key in Redis before calling Acquire
	cacheKey := cm.keyConstraintCache(accountID, envID, fnID, constraints[0])
	require.NotEmpty(t, cacheKey)

	retryAtMS := clock.Now().Add(10 * time.Second).UnixMilli()
	require.NoError(t, r.Set(cacheKey, fmt.Sprintf("%d", retryAtMS)))
	r.SetTTL(cacheKey, 30*time.Second)

	// Acquire should short-circuit due to cache hit
	resp, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "test-preexisting"))
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Empty(t, resp.Leases, "expected no leases on cache hit")
	assert.Len(t, resp.ExhaustedConstraints, 1, "expected one exhausted constraint")
	assert.Equal(t, 1, resp.internalDebugState.CacheHit)
}

func TestAcquireCacheKeysHaveTTL(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, r, clock, ctx := newTestSetup(t, enableAllCache())

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 1,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Fill capacity to 1
	resp1, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "fill-capacity"))
	require.NoError(t, err)
	require.Len(t, resp1.Leases, 1)

	// Second acquire should exhaust and set cache key
	resp2, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "exhaust-capacity"))
	require.NoError(t, err)
	require.Empty(t, resp2.Leases)
	assert.Len(t, resp2.ExhaustedConstraints, 1)

	// Verify cache key exists with a TTL
	cacheKey := cm.keyConstraintCache(accountID, envID, fnID, constraints[0])
	require.True(t, r.Exists(cacheKey), "cache key should exist in Redis")

	ttl := r.TTL(cacheKey)
	assert.Greater(t, ttl, time.Duration(0), "cache key must have a positive TTL")
	assert.LessOrEqual(t, ttl, MaxCacheTTL, "cache key TTL must not exceed MaxCacheTTL")
}

func TestAcquireCacheAnyConstraintShortCircuits(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, r, clock, ctx := newTestSetup(t, enableAllCache())

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 5,
			AccountConcurrency:  10,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeAccount,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Pre-cache only the second constraint as exhausted
	sortedConstraints := make([]ConstraintItem, len(constraints))
	copy(sortedConstraints, constraints)
	sortConstraints(sortedConstraints)

	// Cache the function concurrency constraint
	fnConstraint := ConstraintItem{
		Kind: ConstraintKindConcurrency,
		Concurrency: &ConcurrencyConstraint{
			Scope: enums.ConcurrencyScopeFn,
			Mode:  enums.ConcurrencyModeStep,
		},
	}
	cacheKey := cm.keyConstraintCache(accountID, envID, fnID, fnConstraint)
	require.NotEmpty(t, cacheKey)

	retryAtMS := clock.Now().Add(5 * time.Second).UnixMilli()
	require.NoError(t, r.Set(cacheKey, fmt.Sprintf("%d", retryAtMS)))
	r.SetTTL(cacheKey, 30*time.Second)

	// Acquire should short-circuit without evaluating other constraints
	resp, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "multi-constraint"))
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, 1, resp.internalDebugState.CacheHit, "expected cache hit when any constraint is cached")
	assert.Empty(t, resp.Leases)
	assert.NotEmpty(t, resp.ExhaustedConstraints)
}

func TestAcquireCacheMissFallsThrough(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, _, clock, ctx := newTestSetup(t, enableAllCache())

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 10,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// No cache keys set - should proceed normally
	resp, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "cache-miss"))
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, 0, resp.internalDebugState.CacheHit, "expected cache miss")
	assert.Len(t, resp.Leases, 1, "should have granted a lease on cache miss")
}

func TestAcquireCacheTTLExpiry(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, r, clock, ctx := newTestSetup(t, enableAllCache())

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 10,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Set cache key with 2s TTL
	cacheKey := cm.keyConstraintCache(accountID, envID, fnID, constraints[0])
	retryAtMS := clock.Now().Add(2 * time.Second).UnixMilli()
	require.NoError(t, r.Set(cacheKey, fmt.Sprintf("%d", retryAtMS)))
	r.SetTTL(cacheKey, 2*time.Second)

	// Should be cache hit now
	resp1, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "before-expiry"))
	require.NoError(t, err)
	assert.Equal(t, 1, resp1.internalDebugState.CacheHit)

	// Advance past TTL
	r.FastForward(3 * time.Second)
	clock.Advance(3 * time.Second)

	// Should be cache miss now - normal acquire succeeds
	resp2, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "after-expiry"))
	require.NoError(t, err)
	assert.Equal(t, 0, resp2.internalDebugState.CacheHit, "expected cache miss after TTL expiry")
	assert.Len(t, resp2.Leases, 1, "should grant lease after cache expires")
}

func TestAcquireCacheTTLClamping(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	// Custom min/max TTL for this test
	customMinTTL := 5 * time.Second
	customMaxTTL := 15 * time.Second

	cm, r, clock, ctx := newTestSetup(t, enableAllCache())
	cm.acquireCacheTTL = func(_ context.Context, _, _, _ uuid.UUID) (time.Duration, time.Duration) {
		return customMinTTL, customMaxTTL
	}

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 1,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Fill capacity
	_, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "fill"))
	require.NoError(t, err)

	// Exhaust - retryAfter is ~2s for concurrency, which is less than our minTTL of 5s
	_, err = cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "exhaust"))
	require.NoError(t, err)

	cacheKey := cm.keyConstraintCache(accountID, envID, fnID, constraints[0])
	require.True(t, r.Exists(cacheKey))

	ttl := r.TTL(cacheKey)
	// TTL should be clamped to at least minTTL
	assert.GreaterOrEqual(t, ttl, customMinTTL, "TTL should be >= minTTL")
	assert.LessOrEqual(t, ttl, customMaxTTL, "TTL should be <= maxTTL")
}

func TestAcquireCacheFeatureFlagDisabled(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	// No cache function - caching is disabled
	cm, r, clock, ctx := newTestSetup(t, nil)

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 1,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Fill capacity
	_, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "fill"))
	require.NoError(t, err)

	// Exhaust capacity
	resp, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "exhaust"))
	require.NoError(t, err)
	assert.Empty(t, resp.Leases)
	assert.Equal(t, 0, resp.internalDebugState.CacheHit)

	// Verify NO cache key was set
	cacheKey := cm.keyConstraintCache(accountID, envID, fnID, constraints[0])
	assert.False(t, r.Exists(cacheKey), "cache key should NOT exist when feature flag is disabled")
}

func TestAcquireCachePerConstraintGranularity(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	// Only enable caching for concurrency, not throttle
	enableCache := func(_ context.Context, _, _, _ uuid.UUID, ci ConstraintItem) bool {
		return ci.Kind == ConstraintKindConcurrency
	}

	cm, r, clock, ctx := newTestSetup(t, enableCache)

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 1,
		},
		Throttle: []ThrottleConfig{
			{
				Scope:  enums.ThrottleScopeFn,
				Limit:  1,
				Burst:  1,
				Period: 60, // 60 second period
			},
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
		{
			Kind: ConstraintKindThrottle,
			Throttle: &ThrottleConstraint{
				Scope:            enums.ThrottleScopeFn,
				EvaluatedKeyHash: "test-hash",
			},
		},
	}

	// Fill capacity (both concurrency and throttle will be at limit after this)
	_, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "fill"))
	require.NoError(t, err)

	// Exhaust - both constraints should be exhausted
	resp, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "exhaust"))
	require.NoError(t, err)
	assert.Empty(t, resp.Leases)

	// Only concurrency cache key should exist, not throttle.
	// Note: sortConstraints mutates the slice, so we look up by Kind rather than index.
	concurrencyCacheKey := cm.keyConstraintCache(accountID, envID, fnID, ConstraintItem{
		Kind: ConstraintKindConcurrency,
		Concurrency: &ConcurrencyConstraint{
			Scope: enums.ConcurrencyScopeFn,
			Mode:  enums.ConcurrencyModeStep,
		},
	})
	throttleCacheKey := cm.keyConstraintCache(accountID, envID, fnID, ConstraintItem{
		Kind: ConstraintKindThrottle,
		Throttle: &ThrottleConstraint{
			Scope:            enums.ThrottleScopeFn,
			EvaluatedKeyHash: "test-hash",
		},
	})

	assert.True(t, r.Exists(concurrencyCacheKey), "concurrency cache key should exist")
	assert.False(t, r.Exists(throttleCacheKey), "throttle cache key should NOT exist when feature flag disabled for throttle")
}

func TestAcquireCacheIsolationAcrossAccounts(t *testing.T) {
	accountA, envA, fnA := uuid.New(), uuid.New(), uuid.New()
	accountB, envB, fnB := uuid.New(), uuid.New(), uuid.New()
	cm, r, clock, ctx := newTestSetup(t, enableAllCache())

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 10,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Cache a constraint as exhausted for account A
	cacheKeyA := cm.keyConstraintCache(accountA, envA, fnA, constraints[0])
	retryAtMS := clock.Now().Add(10 * time.Second).UnixMilli()
	require.NoError(t, r.Set(cacheKeyA, fmt.Sprintf("%d", retryAtMS)))
	r.SetTTL(cacheKeyA, 30*time.Second)

	// Account A should see cache hit
	respA, err := cm.Acquire(ctx, makeAcquireRequest(accountA, envA, fnA, clock, config, constraints, "account-a"))
	require.NoError(t, err)
	assert.Equal(t, 1, respA.internalDebugState.CacheHit, "account A should see cache hit")
	assert.Empty(t, respA.Leases)

	// Account B should be unaffected
	respB, err := cm.Acquire(ctx, makeAcquireRequest(accountB, envB, fnB, clock, config, constraints, "account-b"))
	require.NoError(t, err)
	assert.Equal(t, 0, respB.internalDebugState.CacheHit, "account B should not see cache hit")
	assert.Len(t, respB.Leases, 1, "account B should get a lease")
}

func TestAcquireCacheDebugResponse(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, r, clock, ctx := newTestSetup(t, enableAllCache())

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 10,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Pre-cache constraint
	cacheKey := cm.keyConstraintCache(accountID, envID, fnID, constraints[0])
	retryAtMS := clock.Now().Add(10 * time.Second).UnixMilli()
	require.NoError(t, r.Set(cacheKey, fmt.Sprintf("%d", retryAtMS)))
	r.SetTTL(cacheKey, 30*time.Second)

	resp, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "debug-test"))
	require.NoError(t, err)

	assert.Equal(t, 1, resp.internalDebugState.CacheHit)
	assert.Equal(t, 1, resp.internalDebugState.CacheHit)

	// Cache miss case
	resp2, err := cm.Acquire(ctx, makeAcquireRequest(uuid.New(), uuid.New(), uuid.New(), clock, config, constraints, "debug-miss"))
	require.NoError(t, err)
	assert.Equal(t, 0, resp2.internalDebugState.CacheHit)
	assert.Equal(t, 0, resp2.internalDebugState.CacheHit)
}

func TestAcquireCacheStorageOnPartialGrant(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, r, clock, ctx := newTestSetup(t, enableAllCache())

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 2,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Fill 1 of 2 slots
	_, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "fill-1"))
	require.NoError(t, err)

	// Request 2 more but only 1 is available - constraint will be exhausted after granting
	req := makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "fill-2")
	req.Amount = 2
	req.LeaseIdempotencyKeys = []string{"item0", "item1"}
	resp, err := cm.Acquire(ctx, req)
	require.NoError(t, err)
	// Should grant 1 lease (capacity=2, 1 used, 1 available)
	assert.Len(t, resp.Leases, 1)
	// After granting, constraint should be exhausted (2/2 used)
	assert.NotEmpty(t, resp.ExhaustedConstraints)

	// Verify cache key was stored for the exhausted constraint
	cacheKey := cm.keyConstraintCache(accountID, envID, fnID, constraints[0])
	assert.True(t, r.Exists(cacheKey), "cache key should be set when constraint exhausted after partial grant")
	ttl := r.TTL(cacheKey)
	assert.Greater(t, ttl, time.Duration(0))
}

func TestAcquireCacheFeatureFlagReturnsFalse(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	// Feature flag function exists but always returns false
	enableCache := func(_ context.Context, _, _, _ uuid.UUID, _ ConstraintItem) bool {
		return false
	}

	cm, r, clock, ctx := newTestSetup(t, enableCache)

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 1,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Fill and exhaust
	_, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "fill"))
	require.NoError(t, err)

	resp, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "exhaust"))
	require.NoError(t, err)
	assert.Empty(t, resp.Leases)
	assert.Equal(t, 0, resp.internalDebugState.CacheHit)

	// No cache key should be set
	cacheKey := cm.keyConstraintCache(accountID, envID, fnID, constraints[0])
	assert.False(t, r.Exists(cacheKey), "cache key should NOT exist when feature flag returns false")
}

func TestAcquireCacheAccountScopedFlag(t *testing.T) {
	targetAccountID := uuid.New()
	otherAccountID := uuid.New()
	envID, fnID := uuid.New(), uuid.New()

	// Only enable for a specific account
	enableCache := func(_ context.Context, accountID, _, _ uuid.UUID, _ ConstraintItem) bool {
		return accountID == targetAccountID
	}

	cm, r, clock, ctx := newTestSetup(t, enableCache)

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 1,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Exhaust capacity for target account
	_, err := cm.Acquire(ctx, makeAcquireRequest(targetAccountID, envID, fnID, clock, config, constraints, "target-fill"))
	require.NoError(t, err)
	_, err = cm.Acquire(ctx, makeAcquireRequest(targetAccountID, envID, fnID, clock, config, constraints, "target-exhaust"))
	require.NoError(t, err)

	// Exhaust capacity for other account
	_, err = cm.Acquire(ctx, makeAcquireRequest(otherAccountID, envID, fnID, clock, config, constraints, "other-fill"))
	require.NoError(t, err)
	_, err = cm.Acquire(ctx, makeAcquireRequest(otherAccountID, envID, fnID, clock, config, constraints, "other-exhaust"))
	require.NoError(t, err)

	// Only target account should have cache key
	targetCacheKey := cm.keyConstraintCache(targetAccountID, envID, fnID, constraints[0])
	otherCacheKey := cm.keyConstraintCache(otherAccountID, envID, fnID, constraints[0])

	assert.True(t, r.Exists(targetCacheKey), "target account cache key should exist")
	assert.False(t, r.Exists(otherCacheKey), "other account cache key should NOT exist")
}

func TestAcquireCacheInvalidatedOnRelease(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, r, clock, ctx := newTestSetup(t, enableAllCache())

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 1,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Fill capacity
	resp1, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "fill"))
	require.NoError(t, err)
	require.Len(t, resp1.Leases, 1)

	// Exhaust — cache key should be set
	resp2, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "exhaust"))
	require.NoError(t, err)
	require.Empty(t, resp2.Leases)

	cacheKey := cm.keyConstraintCache(accountID, envID, fnID, constraints[0])
	require.True(t, r.Exists(cacheKey), "cache key should exist after exhaustion")

	// Release the lease
	_, err = cm.Release(ctx, &CapacityReleaseRequest{
		IdempotencyKey: "release-1",
		AccountID:      accountID,
		LeaseID:        resp1.Leases[0].LeaseID,
		Source:         LeaseSource{Service: ServiceExecutor},
	})
	require.NoError(t, err)

	// Cache key should be deleted
	assert.False(t, r.Exists(cacheKey), "cache key should be deleted after release")

	// Acquire should succeed (not blocked by stale cache)
	resp3, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "after-release"))
	require.NoError(t, err)
	assert.Len(t, resp3.Leases, 1, "should get lease after release invalidated cache")
	assert.Equal(t, 0, resp3.internalDebugState.CacheHit, "should not be a cache hit")
}

func TestAcquireCacheNotInvalidatedForThrottleOnRelease(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, r, clock, ctx := newTestSetup(t, enableAllCache())

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 1,
		},
		Throttle: []ThrottleConfig{
			{
				Scope:  enums.ThrottleScopeFn,
				Limit:  1,
				Burst:  1,
				Period: 60,
			},
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
		{
			Kind: ConstraintKindThrottle,
			Throttle: &ThrottleConstraint{
				Scope:            enums.ThrottleScopeFn,
				EvaluatedKeyHash: "test-hash",
			},
		},
	}

	// Fill capacity
	resp1, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "fill"))
	require.NoError(t, err)
	require.Len(t, resp1.Leases, 1)

	// Build cache keys by explicit constraint kind (sortConstraints reorders the slice)
	concurrencyCacheKey := cm.keyConstraintCache(accountID, envID, fnID, ConstraintItem{
		Kind:        ConstraintKindConcurrency,
		Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn, Mode: enums.ConcurrencyModeStep},
	})
	throttleCacheKey := cm.keyConstraintCache(accountID, envID, fnID, ConstraintItem{
		Kind:     ConstraintKindThrottle,
		Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn, EvaluatedKeyHash: "test-hash"},
	})

	// Manually set both cache keys (simulates prior acquires that exhausted each constraint)
	retryAtMS := clock.Now().Add(10 * time.Second).UnixMilli()
	require.NoError(t, r.Set(concurrencyCacheKey, fmt.Sprintf("%d", retryAtMS)))
	r.SetTTL(concurrencyCacheKey, 30*time.Second)
	require.NoError(t, r.Set(throttleCacheKey, fmt.Sprintf("%d", retryAtMS)))
	r.SetTTL(throttleCacheKey, 30*time.Second)

	// Release the lease
	_, err = cm.Release(ctx, &CapacityReleaseRequest{
		IdempotencyKey: "release-1",
		AccountID:      accountID,
		LeaseID:        resp1.Leases[0].LeaseID,
		Source:         LeaseSource{Service: ServiceExecutor},
	})
	require.NoError(t, err)

	// Concurrency cache key should be deleted, throttle should remain
	assert.False(t, r.Exists(concurrencyCacheKey), "concurrency cache key should be deleted after release")
	assert.True(t, r.Exists(throttleCacheKey), "throttle cache key should still exist after release")
}

func TestAcquireCacheInvalidatedWithCustomKey(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, r, clock, ctx := newTestSetup(t, enableAllCache())

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			CustomConcurrencyKeys: []CustomConcurrencyLimit{
				{
					Scope:             enums.ConcurrencyScopeFn,
					Limit:             1,
					KeyExpressionHash: "keyhash123",
				},
			},
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope:             enums.ConcurrencyScopeFn,
				Mode:              enums.ConcurrencyModeStep,
				KeyExpressionHash: "keyhash123",
				EvaluatedKeyHash:  "evalhash456",
			},
		},
	}

	// Fill capacity
	resp1, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "fill"))
	require.NoError(t, err)
	require.Len(t, resp1.Leases, 1)

	// Exhaust
	resp2, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "exhaust"))
	require.NoError(t, err)
	require.Empty(t, resp2.Leases)

	cacheKey := cm.keyConstraintCache(accountID, envID, fnID, constraints[0])
	require.True(t, r.Exists(cacheKey), "custom key cache should exist after exhaustion")

	// Release
	_, err = cm.Release(ctx, &CapacityReleaseRequest{
		IdempotencyKey: "release-1",
		AccountID:      accountID,
		LeaseID:        resp1.Leases[0].LeaseID,
		Source:         LeaseSource{Service: ServiceExecutor},
	})
	require.NoError(t, err)

	assert.False(t, r.Exists(cacheKey), "custom key cache should be deleted after release")
}

func TestAcquireCacheInvalidatedMultipleConstraints(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, r, clock, ctx := newTestSetup(t, enableAllCache())

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 1,
			AccountConcurrency:  1,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeAccount,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Fill capacity
	resp1, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "fill"))
	require.NoError(t, err)
	require.Len(t, resp1.Leases, 1)

	// Exhaust
	resp2, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "exhaust"))
	require.NoError(t, err)
	require.Empty(t, resp2.Leases)

	fnCacheKey := cm.keyConstraintCache(accountID, envID, fnID, constraints[0])
	acctCacheKey := cm.keyConstraintCache(accountID, envID, fnID, constraints[1])
	require.True(t, r.Exists(fnCacheKey), "fn concurrency cache key should exist")
	require.True(t, r.Exists(acctCacheKey), "account concurrency cache key should exist")

	// Release
	_, err = cm.Release(ctx, &CapacityReleaseRequest{
		IdempotencyKey: "release-1",
		AccountID:      accountID,
		LeaseID:        resp1.Leases[0].LeaseID,
		Source:         LeaseSource{Service: ServiceExecutor},
	})
	require.NoError(t, err)

	assert.False(t, r.Exists(fnCacheKey), "fn concurrency cache key should be deleted after release")
	assert.False(t, r.Exists(acctCacheKey), "account concurrency cache key should be deleted after release")
}

func TestAcquireCacheNotInvalidatedWhenCacheDisabled(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	// Create manager WITHOUT cache enabled
	cm, r, clock, ctx := newTestSetup(t, nil)

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 1,
		},
	}
	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	// Manually set a cache key to verify it's NOT deleted on release
	cacheKey := cm.keyConstraintCache(accountID, envID, fnID, constraints[0])
	require.NotEmpty(t, cacheKey)
	require.NoError(t, r.Set(cacheKey, "12345"))

	// Fill capacity
	resp1, err := cm.Acquire(ctx, makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "fill"))
	require.NoError(t, err)
	require.Len(t, resp1.Leases, 1)

	// Release
	_, err = cm.Release(ctx, &CapacityReleaseRequest{
		IdempotencyKey: "release-1",
		AccountID:      accountID,
		LeaseID:        resp1.Leases[0].LeaseID,
		Source:         LeaseSource{Service: ServiceExecutor},
	})
	require.NoError(t, err)

	// Cache key should still exist since cache invalidation is disabled
	assert.True(t, r.Exists(cacheKey), "cache key should NOT be deleted when cache is disabled")
}
