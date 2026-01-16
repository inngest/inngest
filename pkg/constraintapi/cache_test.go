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
	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	type deps struct {
		cm         CapacityManager
		cache      *limitingConstraintCache
		clock      clockwork.Clock
		rc         rueidis.Client
		r          *miniredis.Miniredis
		lifecycles *ConstraintApiDebugLifecycles
	}

	cases := []struct {
		name string
		run  func(ctx context.Context, t *testing.T, deps deps)
	}{
		{
			name: "should cache when limited by a single constraint",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				}

				// First request should pass
				res, err := deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 1,
						},
					},
					Constraints: []ConstraintItem{
						accountConcurrency,
					},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item1"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 1)
				require.Len(t, res.LimitingConstraints, 0)

				require.Equal(t, 0, deps.cache.limitingConstraintCache.ItemCount())
				require.Nil(t, deps.cache.limitingConstraintCache.Get(accountConcurrency.CacheKey(accountID, envID, fnID)))

				// Second request should fail and get cached
				res, err = deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq2",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 1,
						},
					},
					Constraints: []ConstraintItem{
						accountConcurrency,
					},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item2"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 0)
				require.Len(t, res.LimitingConstraints, 1)

				require.Equal(t, 1, deps.cache.limitingConstraintCache.ItemCount())
				require.NotNil(t, deps.cache.limitingConstraintCache.Get(accountConcurrency.CacheKey(accountID, envID, fnID)))

				require.Equal(t, 2, len(deps.lifecycles.AcquireCalls))

				// Third request should be cached
				res, err = deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq3",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 1,
						},
					},
					Constraints: []ConstraintItem{
						accountConcurrency,
					},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item3"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 0)
				require.Len(t, res.LimitingConstraints, 1)

				require.Equal(t, 2, len(deps.lifecycles.AcquireCalls))

				// After cache expires, request should go to API again
				deps.cache.limitingConstraintCache.Delete(accountConcurrency.CacheKey(accountID, envID, fnID))

				res, err = deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq4",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 1,
						},
					},
					Constraints: []ConstraintItem{
						accountConcurrency,
					},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item4"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 0)
				require.Len(t, res.LimitingConstraints, 1)

				require.Equal(t, 3, len(deps.lifecycles.AcquireCalls))
			},
		},
		{
			name: "should cache when limited by multiple constraints",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				}

				fnConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeFn,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:p:%s", fnID),
					},
				}

				customConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "hash",
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:custom:a:%s:hash", accountID),
					},
				}

				// First request should pass
				res, err := deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency:  2,
							FunctionConcurrency: 3,
							CustomConcurrencyKeys: []CustomConcurrencyLimit{
								{
									Scope:             enums.ConcurrencyScopeAccount,
									Limit:             1,
									KeyExpressionHash: "expr-hash",
								},
							},
						},
					},
					Constraints: []ConstraintItem{
						accountConcurrency,
						fnConcurrency,
						customConcurrency,
					},
					Amount:               3,
					LeaseIdempotencyKeys: []string{"item1", "item2", "item3"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 1)
				require.Len(t, res.LimitingConstraints, 2)

				require.Equal(t, 2, deps.cache.limitingConstraintCache.ItemCount())
				require.NotNil(t, deps.cache.limitingConstraintCache.Get(accountConcurrency.CacheKey(accountID, envID, fnID)))
				require.Nil(t, deps.cache.limitingConstraintCache.Get(fnConcurrency.CacheKey(accountID, envID, fnID)))
				require.NotNil(t, deps.cache.limitingConstraintCache.Get(customConcurrency.CacheKey(accountID, envID, fnID)))
			},
		},
		{
			name: "should cache for partial limit hit",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				}

				// First request should pass
				res, err := deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 1,
						},
					},
					Constraints: []ConstraintItem{
						accountConcurrency,
					},
					Amount:               2,
					LeaseIdempotencyKeys: []string{"item1", "item2"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 1)
				require.Len(t, res.LimitingConstraints, 1)

				require.Equal(t, 1, deps.cache.limitingConstraintCache.ItemCount())
				require.NotNil(t, deps.cache.limitingConstraintCache.Get(accountConcurrency.CacheKey(accountID, envID, fnID)))
			},
		},
		{
			name: "should cache throttle constraints when limited",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				throttle := ConstraintItem{
					Kind: ConstraintKindThrottle,
					Throttle: &ThrottleConstraint{
						Scope:             enums.ThrottleScopeAccount,
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "key-hash",
					},
				}

				// First request should pass
				res, err := deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq1",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Throttle: []ThrottleConfig{
							{
								Scope:             enums.ThrottleScopeAccount,
								Limit:             1,
								Period:            60,
								KeyExpressionHash: "expr-hash",
							},
						},
					},
					Constraints: []ConstraintItem{throttle},
					Amount:      1,
					LeaseIdempotencyKeys: []string{"item1"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 1)
				require.Len(t, res.LimitingConstraints, 0)
				require.Equal(t, 0, deps.cache.limitingConstraintCache.ItemCount())

				// Second request should fail and be cached
				res, err = deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq2",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Throttle: []ThrottleConfig{
							{
								Scope:             enums.ThrottleScopeAccount,
								Limit:             1,
								Period:            60,
								KeyExpressionHash: "expr-hash",
							},
						},
					},
					Constraints:          []ConstraintItem{throttle},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item2"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 0)
				require.Len(t, res.LimitingConstraints, 1)
				require.Equal(t, 1, deps.cache.limitingConstraintCache.ItemCount())
				require.NotNil(t, deps.cache.limitingConstraintCache.Get(throttle.CacheKey(accountID, envID, fnID)))

				// Third request should return cached response
				res, err = deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq3",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Throttle: []ThrottleConfig{
							{
								Scope:             enums.ThrottleScopeAccount,
								Limit:             1,
								Period:            60,
								KeyExpressionHash: "expr-hash",
							},
						},
					},
					Constraints:          []ConstraintItem{throttle},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item3"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 0)
				require.Len(t, res.LimitingConstraints, 1)

				// Verify manager was only called twice (not three times due to cache)
				require.Equal(t, 2, len(deps.lifecycles.AcquireCalls))
			},
		},
		{
			name: "should cache rate limit constraints when limited",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				rateLimit := ConstraintItem{
					Kind: ConstraintKindRateLimit,
					RateLimit: &RateLimitConstraint{
						Scope:             enums.RateLimitScopeAccount,
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "key-hash",
					},
				}

				// First request should pass
				res, err := deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						IsRateLimit: true,
					},
					IdempotencyKey: "acq1",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						RateLimit: []RateLimitConfig{
							{
								Scope:             enums.RateLimitScopeAccount,
								Limit:             1,
								Period:            60,
								KeyExpressionHash: "expr-hash",
							},
						},
					},
					Constraints:          []ConstraintItem{rateLimit},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item1"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 1)
				require.Len(t, res.LimitingConstraints, 0)
				require.Equal(t, 0, deps.cache.limitingConstraintCache.ItemCount())

				// Second request should fail and be cached
				res, err = deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						IsRateLimit: true,
					},
					IdempotencyKey: "acq2",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						RateLimit: []RateLimitConfig{
							{
								Scope:             enums.RateLimitScopeAccount,
								Limit:             1,
								Period:            60,
								KeyExpressionHash: "expr-hash",
							},
						},
					},
					Constraints:          []ConstraintItem{rateLimit},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item2"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 0)
				require.Len(t, res.LimitingConstraints, 1)
				require.Equal(t, 1, deps.cache.limitingConstraintCache.ItemCount())
				require.NotNil(t, deps.cache.limitingConstraintCache.Get(rateLimit.CacheKey(accountID, envID, fnID)))

				// Third request should return cached response
				res, err = deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						IsRateLimit: true,
					},
					IdempotencyKey: "acq3",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						RateLimit: []RateLimitConfig{
							{
								Scope:             enums.RateLimitScopeAccount,
								Limit:             1,
								Period:            60,
								KeyExpressionHash: "expr-hash",
							},
						},
					},
					Constraints:          []ConstraintItem{rateLimit},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item3"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 0)
				require.Len(t, res.LimitingConstraints, 1)

				// Verify manager was only called twice (not three times due to cache)
				require.Equal(t, 2, len(deps.lifecycles.AcquireCalls))
			},
		},
		{
			name: "should passthrough Check() calls without caching",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				}

				// Check should passthrough to manager
				checkResp, userErr, internalErr := deps.cache.Check(ctx, &CapacityCheckRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 10,
						},
					},
					Constraints: []ConstraintItem{accountConcurrency},
				})
				require.NoError(t, userErr)
				require.NoError(t, internalErr)
				require.NotNil(t, checkResp)
				require.Len(t, checkResp.Usage, 1)

				// Verify nothing was cached
				require.Equal(t, 0, deps.cache.limitingConstraintCache.ItemCount())
			},
		},
		{
			name: "should passthrough ExtendLease() calls",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				}

				// First acquire a lease
				acquireResp, err := deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq1",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 10,
						},
					},
					Constraints:          []ConstraintItem{accountConcurrency},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item1"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, acquireResp.Leases, 1)

				// ExtendLease should passthrough
				extendResp, err := deps.cache.ExtendLease(ctx, &CapacityExtendLeaseRequest{
					AccountID:      accountID,
					IdempotencyKey: "extend1",
					LeaseID:        acquireResp.Leases[0].LeaseID,
					Duration:       5 * time.Second,
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
				})
				require.NoError(t, err)
				require.NotNil(t, extendResp)
				require.NotNil(t, extendResp.LeaseID)
			},
		},
		{
			name: "should passthrough Release() calls",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				}

				// First acquire a lease
				acquireResp, err := deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq1",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 10,
						},
					},
					Constraints:          []ConstraintItem{accountConcurrency},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item1"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, acquireResp.Leases, 1)

				// Release should passthrough
				releaseResp, err := deps.cache.Release(ctx, &CapacityReleaseRequest{
					AccountID:      accountID,
					IdempotencyKey: "release1",
					LeaseID:        acquireResp.Leases[0].LeaseID,
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
				})
				require.NoError(t, err)
				require.NotNil(t, releaseResp)
			},
		},
		{
			name: "should isolate cache by accountID",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				accountID1 := uuid.New()
				accountID2 := uuid.New()

				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID1),
					},
				}

				// Account 1: Acquire until limited
				_, err := deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID1,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq1",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 1,
						},
					},
					Constraints:          []ConstraintItem{accountConcurrency},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item1"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)

				// Account 1: Get limited and cached
				res1, err := deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID1,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq2",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 1,
						},
					},
					Constraints:          []ConstraintItem{accountConcurrency},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item2"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res1.Leases, 0)
				require.Len(t, res1.LimitingConstraints, 1)

				// Account 2: Should not be limited (different cache key)
				accountConcurrency2 := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID2),
					},
				}

				res2, err := deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID2,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq3",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 10,
						},
					},
					Constraints:          []ConstraintItem{accountConcurrency2},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item3"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res2.Leases, 1)
				require.Len(t, res2.LimitingConstraints, 0)

				// Verify separate cache entries
				require.NotNil(t, deps.cache.limitingConstraintCache.Get(accountConcurrency.CacheKey(accountID1, envID, fnID)))
				require.Nil(t, deps.cache.limitingConstraintCache.Get(accountConcurrency2.CacheKey(accountID2, envID, fnID)))
			},
		},
		{
			name: "should expire cache entries after advancing time",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				}

				// First request - acquire
				_, err := deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq1",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 1,
						},
					},
					Constraints:          []ConstraintItem{accountConcurrency},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item1"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)

				// Second request - get limited and cached
				_, err = deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq2",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 1,
						},
					},
					Constraints:          []ConstraintItem{accountConcurrency},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item2"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Equal(t, 1, deps.cache.limitingConstraintCache.ItemCount())
				require.Equal(t, 2, len(deps.lifecycles.AcquireCalls))

				// Delete cache entry to simulate expiration
				// Note: ccache uses real time internally, not the fake clock
				deps.cache.limitingConstraintCache.Delete(accountConcurrency.CacheKey(accountID, envID, fnID))
				require.Equal(t, 0, deps.cache.limitingConstraintCache.ItemCount())

				// Third request - cache is cleared, should go to manager
				_, err = deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					Migration: MigrationIdentifier{
						QueueShard: "test",
					},
					IdempotencyKey: "acq3",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency: 1,
						},
					},
					Constraints:          []ConstraintItem{accountConcurrency},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item3"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)

				// Verify manager was called again (cache expired)
				require.Equal(t, 3, len(deps.lifecycles.AcquireCalls))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := miniredis.RunT(t)
			ctx := context.Background()

			rc, err := rueidis.NewClient(rueidis.ClientOption{
				InitAddress:  []string{r.Addr()},
				DisableCache: true,
			})
			require.NoError(t, err)
			defer rc.Close()

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))
			l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
			ctx = logger.WithStdlib(ctx, l)

			lifecycles := NewConstraintAPIDebugLifecycles()

			cm, err := NewRedisCapacityManager(
				WithRateLimitClient(rc),
				WithQueueShards(map[string]rueidis.Client{
					"test": rc,
				}),
				WithClock(clock),
				WithNumScavengerShards(1),
				WithQueueStateKeyPrefix("q:v1"),
				WithRateLimitKeyPrefix("rl"),
				WithEnableDebugLogs(true),
				// Do not cache check requests
				WithCheckIdempotencyTTL(0),
				WithLifecycles(lifecycles),
			)
			require.NoError(t, err)
			require.NotNil(t, cm)

			cache := NewLimitingConstraintCache(clock, cm)

			tc.run(ctx, t, deps{
				clock:      clock,
				cm:         cm,
				cache:      cache,
				rc:         rc,
				r:          r,
				lifecycles: lifecycles,
			})
		})
	}
}
