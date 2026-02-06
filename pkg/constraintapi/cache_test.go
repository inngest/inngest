package constraintapi

import (
	"context"
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
		cache      *constraintCache
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
				// After successfully acquiring the last available lease, the constraint is now exhausted
				require.Len(t, res.ExhaustedConstraints, 1)

				// The exhausted constraint should be cached
				require.Equal(t, 1, deps.cache.cache.ItemCount())
				require.NotNil(t, deps.cache.cache.Get(accountConcurrency.CacheKey(accountID, envID, fnID)))

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
				require.Len(t, res.ExhaustedConstraints, 1)

				require.Equal(t, 1, deps.cache.cache.ItemCount())
				require.NotNil(t, deps.cache.cache.Get(accountConcurrency.CacheKey(accountID, envID, fnID)))

				// Only 1 call - second request was served from cache after first exhausted the constraint
				require.Equal(t, 1, len(deps.lifecycles.AcquireCalls))

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
				require.Len(t, res.ExhaustedConstraints, 1)

				// Still only 1 call - third request also served from cache
				require.Equal(t, 1, len(deps.lifecycles.AcquireCalls))

				// After cache expires, request should go to API again
				deps.cache.cache.Delete(accountConcurrency.CacheKey(accountID, envID, fnID))

				res, err = deps.cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
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
				require.Len(t, res.ExhaustedConstraints, 1)

				// Total 2 calls - first request and this one after cache expiry
				require.Equal(t, 2, len(deps.lifecycles.AcquireCalls))
			},
		},
		{
			name: "should cache when limited by multiple constraints",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						},
				}

				fnConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeFn,
						},
				}

				customConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "hash",
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
				// Only the custom concurrency constraint (limit=1) is exhausted after granting 1 lease
				require.Len(t, res.ExhaustedConstraints, 1)

				// Only exhausted constraints are cached (custom concurrency)
				require.Equal(t, 1, deps.cache.cache.ItemCount())
				require.Nil(t, deps.cache.cache.Get(accountConcurrency.CacheKey(accountID, envID, fnID)))
				require.Nil(t, deps.cache.cache.Get(fnConcurrency.CacheKey(accountID, envID, fnID)))
				require.NotNil(t, deps.cache.cache.Get(customConcurrency.CacheKey(accountID, envID, fnID)))
			},
		},
		{
			name: "should cache for partial limit hit",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
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
				require.Len(t, res.ExhaustedConstraints, 1)

				require.Equal(t, 1, deps.cache.cache.ItemCount())
				require.NotNil(t, deps.cache.cache.Get(accountConcurrency.CacheKey(accountID, envID, fnID)))
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
					Constraints:          []ConstraintItem{throttle},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item1"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 1)
				require.Len(t, res.LimitingConstraints, 0)
				// After using the throttle token, the constraint is exhausted
				require.Len(t, res.ExhaustedConstraints, 1)
				// The exhausted constraint should be cached
				require.Equal(t, 1, deps.cache.cache.ItemCount())

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
				require.Len(t, res.ExhaustedConstraints, 1)
				require.Equal(t, 1, deps.cache.cache.ItemCount())
				require.NotNil(t, deps.cache.cache.Get(throttle.CacheKey(accountID, envID, fnID)))

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
				require.Len(t, res.ExhaustedConstraints, 1)

				// Verify manager was only called once - second request served from cache
				require.Equal(t, 1, len(deps.lifecycles.AcquireCalls))
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
				// After using the rate limit token, the constraint is exhausted
				require.Len(t, res.ExhaustedConstraints, 1)
				// The exhausted constraint should be cached
				require.Equal(t, 1, deps.cache.cache.ItemCount())

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
				require.Len(t, res.ExhaustedConstraints, 1)
				require.Equal(t, 1, deps.cache.cache.ItemCount())
				require.NotNil(t, deps.cache.cache.Get(rateLimit.CacheKey(accountID, envID, fnID)))

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
				require.Len(t, res.ExhaustedConstraints, 1)

				// Verify manager was only called once - second request served from cache
				require.Equal(t, 1, len(deps.lifecycles.AcquireCalls))
			},
		},
		{
			name: "should passthrough Check() calls without caching",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						},
				}

				// Check should passthrough to manager
				checkResp, userErr, internalErr := deps.cache.Check(ctx, &CapacityCheckRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
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
				require.Equal(t, 0, deps.cache.cache.ItemCount())
			},
		},
		{
			name: "should passthrough ExtendLease() calls",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
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
				require.Len(t, res1.ExhaustedConstraints, 1)

				// Account 2: Should not be limited (different cache key)
				accountConcurrency2 := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
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
				require.NotNil(t, deps.cache.cache.Get(accountConcurrency.CacheKey(accountID1, envID, fnID)))
				require.Nil(t, deps.cache.cache.Get(accountConcurrency2.CacheKey(accountID2, envID, fnID)))
			},
		},
		{
			name: "should expire cache entries after advancing time",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
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
				require.Equal(t, 1, deps.cache.cache.ItemCount())
				// Only 1 call to manager - second request was served from cache after first request exhausted the constraint
				require.Equal(t, 1, len(deps.lifecycles.AcquireCalls))

				// Delete cache entry to simulate expiration
				// Note: ccache uses real time internally, not the fake clock
				deps.cache.cache.Delete(accountConcurrency.CacheKey(accountID, envID, fnID))
				require.Equal(t, 0, deps.cache.cache.ItemCount())

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

				// Verify manager was called again (cache expired) - total 2 calls
				require.Equal(t, 2, len(deps.lifecycles.AcquireCalls))
			},
		},
		{
			name: "should filter cached constraints with shouldCache predicate",
			run: func(ctx context.Context, t *testing.T, deps deps) {
				// Create different types of constraints
				accountConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						},
				}

				fnConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeFn,
						},
				}

				customConcurrency := ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "key-hash",
						},
				}

				throttle := ConstraintItem{
					Kind: ConstraintKindThrottle,
					Throttle: &ThrottleConstraint{
						Scope:             enums.ThrottleScopeAccount,
						KeyExpressionHash: "throttle-expr",
						EvaluatedKeyHash:  "throttle-key",
					},
				}

				// Create a cache with a filter that ONLY caches:
				// - Account-level concurrency WITHOUT custom keys
				// This means fnConcurrency, customConcurrency, and throttle should NOT be cached
				cache := NewConstraintCache(
					WithConstraintCacheClock(deps.clock),
					WithConstraintCacheManager(deps.cm),
					WithConstraintCacheEnable(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, minTTL, maxTTL time.Duration) {
						return true, MinCacheTTL, MaxCacheTTL
					}),
					WithConstraintCacheShouldCache(func(ci ConstraintItem) bool {
						// Only cache account-level concurrency without custom keys
						return ci.Kind == ConstraintKindConcurrency &&
							ci.Concurrency != nil &&
							ci.Concurrency.Scope == enums.ConcurrencyScopeAccount &&
							!ci.Concurrency.IsCustomKey()
					}),
				)

				// First request that exhausts all constraints
				res, err := cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					IdempotencyKey: "acq1",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency:  1, // Will be exhausted after 1 lease
							FunctionConcurrency: 1, // Will be exhausted after 1 lease
							CustomConcurrencyKeys: []CustomConcurrencyLimit{
								{
									Scope:             enums.ConcurrencyScopeAccount,
									Limit:             1, // Will be exhausted after 1 lease
									KeyExpressionHash: "expr-hash",
								},
							},
						},
						Throttle: []ThrottleConfig{
							{
								Scope:             enums.ThrottleScopeAccount,
								Limit:             1, // Will be exhausted after 1 lease
								Period:            60,
								KeyExpressionHash: "throttle-expr",
							},
						},
					},
					Constraints: []ConstraintItem{
						accountConcurrency,
						fnConcurrency,
						customConcurrency,
						throttle,
					},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item1"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 1)
				// All constraints should be exhausted after acquiring 1 lease
				require.Len(t, res.ExhaustedConstraints, 4)

				// Verify only account concurrency (without custom key) is cached
				require.Equal(t, 1, cache.cache.ItemCount(), "Should only cache 1 constraint")
				require.NotNil(t, cache.cache.Get(accountConcurrency.CacheKey(accountID, envID, fnID)), "Account concurrency should be cached")
				require.Nil(t, cache.cache.Get(fnConcurrency.CacheKey(accountID, envID, fnID)), "Function concurrency should NOT be cached")
				require.Nil(t, cache.cache.Get(customConcurrency.CacheKey(accountID, envID, fnID)), "Custom concurrency should NOT be cached")
				require.Nil(t, cache.cache.Get(throttle.CacheKey(accountID, envID, fnID)), "Throttle should NOT be cached")

				// Clear the debug lifecycles to count fresh calls
				deps.lifecycles.AcquireCalls = nil

				// Second request - should hit cache for account concurrency only
				res, err = cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					IdempotencyKey: "acq2",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							AccountConcurrency:  1,
							FunctionConcurrency: 1,
							CustomConcurrencyKeys: []CustomConcurrencyLimit{
								{
									Scope:             enums.ConcurrencyScopeAccount,
									Limit:             1,
									KeyExpressionHash: "expr-hash",
								},
							},
						},
						Throttle: []ThrottleConfig{
							{
								Scope:             enums.ThrottleScopeAccount,
								Limit:             1,
								Period:            60,
								KeyExpressionHash: "throttle-expr",
							},
						},
					},
					Constraints: []ConstraintItem{
						accountConcurrency,
						fnConcurrency,
						customConcurrency,
						throttle,
					},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item2"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 0)
				// Should return the cached account concurrency constraint
				require.Len(t, res.ExhaustedConstraints, 1)
				require.Equal(t, accountConcurrency.Kind, res.ExhaustedConstraints[0].Kind)

				// No additional calls to manager - served from cache
				require.Equal(t, 0, len(deps.lifecycles.AcquireCalls), "Second request should be served from cache")

				// Third request with ONLY the non-cached constraints
				// This should go to the manager since these constraints aren't cached
				deps.lifecycles.AcquireCalls = nil
				res, err = cache.Acquire(ctx, &CapacityAcquireRequest{
					AccountID:  accountID,
					EnvID:      envID,
					FunctionID: fnID,
					Source: LeaseSource{
						Service:           ServiceAPI,
						Location:          CallerLocationItemLease,
						RunProcessingMode: RunProcessingModeBackground,
					},
					IdempotencyKey: "acq3",
					Configuration: ConstraintConfig{
						FunctionVersion: 1,
						Concurrency: ConcurrencyConfig{
							FunctionConcurrency: 1,
							CustomConcurrencyKeys: []CustomConcurrencyLimit{
								{
									Scope:             enums.ConcurrencyScopeAccount,
									Limit:             1,
									KeyExpressionHash: "expr-hash",
								},
							},
						},
						Throttle: []ThrottleConfig{
							{
								Scope:             enums.ThrottleScopeAccount,
								Limit:             1,
								Period:            60,
								KeyExpressionHash: "throttle-expr",
							},
						},
					},
					Constraints: []ConstraintItem{
						fnConcurrency,
						customConcurrency,
						throttle,
					},
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item3"},
					CurrentTime:          deps.clock.Now(),
					Duration:             3 * time.Second,
					MaximumLifetime:      time.Minute,
				})
				require.NoError(t, err)
				require.Len(t, res.Leases, 0)
				require.Len(t, res.ExhaustedConstraints, 3)

				// Should have called the manager since non-cached constraints were requested
				require.Equal(t, 1, len(deps.lifecycles.AcquireCalls), "Request with non-cached constraints should hit manager")

				// Verify still only 1 item cached (account concurrency)
				// The non-cached constraints should not have been added to cache
				require.Equal(t, 1, cache.cache.ItemCount(), "Should still only have 1 cached constraint")
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
				WithClient(rc),
				WithShardName("default"),
				WithClock(clock),
				WithEnableDebugLogs(true),
				// Do not cache check requests
				WithCheckIdempotencyTTL(0),
				WithLifecycles(lifecycles),
			)
			require.NoError(t, err)
			require.NotNil(t, cm)

			cache := NewConstraintCache(
				WithConstraintCacheClock(clock),
				WithConstraintCacheManager(cm),
				WithConstraintCacheEnable(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, minTTL, maxTTL time.Duration) {
					return true, MinCacheTTL, MaxCacheTTL
				}),
			)

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
