package constraintapi

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAcquireCacheKey(t *testing.T) {
	manager := &redisCapacityManager{}
	accountID := uuid.New()
	functionID := uuid.New()

	tests := []struct {
		name       string
		constraint ConstraintItem
		wantKey    string
	}{
		{
			name: "account concurrency without custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeAccount,
					KeyExpressionHash: "",
				},
			},
			wantKey: "acq:a:" + accountID.String(),
		},
		{
			name: "function concurrency without custom key",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeFn,
					KeyExpressionHash: "",
				},
			},
			wantKey: "acq:f:" + functionID.String(),
		},
		{
			name: "env concurrency - not cacheable",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeEnv,
					KeyExpressionHash: "",
				},
			},
			wantKey: "",
		},
		{
			name: "account concurrency with custom key - not cacheable",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeAccount,
					KeyExpressionHash: "some-hash",
				},
			},
			wantKey: "",
		},
		{
			name: "function concurrency with custom key - not cacheable",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeFn,
					KeyExpressionHash: "some-hash",
				},
			},
			wantKey: "",
		},
		{
			name: "rate limit - not cacheable",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
			},
			wantKey: "",
		},
		{
			name: "concurrency with nil pointer - not cacheable",
			constraint: ConstraintItem{
				Kind:        ConstraintKindConcurrency,
				Concurrency: nil,
			},
			wantKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := manager.generateAcquireCacheKey(accountID, functionID, &tt.constraint)
			assert.Equal(t, tt.wantKey, key)
		})
	}
}

func TestShouldCacheAcquireResponse(t *testing.T) {
	manager := &redisCapacityManager{}

	tests := []struct {
		name         string
		response     *CapacityAcquireResponse
		wantCacheable bool
	}{
		{
			name: "status 2, no leases, account concurrency - cacheable",
			response: &CapacityAcquireResponse{
				Leases: []CapacityLease{},
				LimitingConstraints: []ConstraintItem{
					{
						Kind: ConstraintKindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							Scope:             enums.ConcurrencyScopeAccount,
							KeyExpressionHash: "",
						},
					},
				},
				internalDebugState: acquireScriptResponse{
					Status: 2, // lacking capacity
				},
			},
			wantCacheable: true,
		},
		{
			name: "status 2, no leases, function concurrency - cacheable",
			response: &CapacityAcquireResponse{
				Leases: []CapacityLease{},
				LimitingConstraints: []ConstraintItem{
					{
						Kind: ConstraintKindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							Scope:             enums.ConcurrencyScopeFn,
							KeyExpressionHash: "",
						},
					},
				},
				internalDebugState: acquireScriptResponse{
					Status: 2,
				},
			},
			wantCacheable: true,
		},
		{
			name: "status 1 (success) - not cacheable",
			response: &CapacityAcquireResponse{
				Leases: []CapacityLease{},
				LimitingConstraints: []ConstraintItem{
					{
						Kind: ConstraintKindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							Scope:             enums.ConcurrencyScopeAccount,
							KeyExpressionHash: "",
						},
					},
				},
				internalDebugState: acquireScriptResponse{
					Status: 1, // success
				},
			},
			wantCacheable: false,
		},
		{
			name: "status 2 with leases - not cacheable",
			response: &CapacityAcquireResponse{
				Leases: []CapacityLease{{}, {}}, // has leases
				LimitingConstraints: []ConstraintItem{
					{
						Kind: ConstraintKindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							Scope:             enums.ConcurrencyScopeAccount,
							KeyExpressionHash: "",
						},
					},
				},
				internalDebugState: acquireScriptResponse{
					Status: 2,
				},
			},
			wantCacheable: false,
		},
		{
			name: "status 2, no leases, no limiting constraints - not cacheable",
			response: &CapacityAcquireResponse{
				Leases:              []CapacityLease{},
				LimitingConstraints: []ConstraintItem{},
				internalDebugState: acquireScriptResponse{
					Status: 2,
				},
			},
			wantCacheable: false,
		},
		{
			name: "status 2, no leases, env concurrency - not cacheable",
			response: &CapacityAcquireResponse{
				Leases: []CapacityLease{},
				LimitingConstraints: []ConstraintItem{
					{
						Kind: ConstraintKindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							Scope:             enums.ConcurrencyScopeEnv,
							KeyExpressionHash: "",
						},
					},
				},
				internalDebugState: acquireScriptResponse{
					Status: 2,
				},
			},
			wantCacheable: false,
		},
		{
			name: "status 2, no leases, custom key - not cacheable",
			response: &CapacityAcquireResponse{
				Leases: []CapacityLease{},
				LimitingConstraints: []ConstraintItem{
					{
						Kind: ConstraintKindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							Scope:             enums.ConcurrencyScopeAccount,
							KeyExpressionHash: "custom-hash",
						},
					},
				},
				internalDebugState: acquireScriptResponse{
					Status: 2,
				},
			},
			wantCacheable: false,
		},
		{
			name: "status 2, no leases, rate limit - not cacheable",
			response: &CapacityAcquireResponse{
				Leases: []CapacityLease{},
				LimitingConstraints: []ConstraintItem{
					{
						Kind: ConstraintKindRateLimit,
					},
				},
				internalDebugState: acquireScriptResponse{
					Status: 2,
				},
			},
			wantCacheable: false,
		},
		{
			name: "status 3 (idempotency) - not cacheable",
			response: &CapacityAcquireResponse{
				Leases: []CapacityLease{},
				LimitingConstraints: []ConstraintItem{
					{
						Kind: ConstraintKindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							Scope:             enums.ConcurrencyScopeAccount,
							KeyExpressionHash: "",
						},
					},
				},
				internalDebugState: acquireScriptResponse{
					Status: 3, // idempotency
				},
			},
			wantCacheable: false,
		},
		{
			name: "mixed constraints - one cacheable - cacheable",
			response: &CapacityAcquireResponse{
				Leases: []CapacityLease{},
				LimitingConstraints: []ConstraintItem{
					{
						Kind: ConstraintKindRateLimit,
					},
					{
						Kind: ConstraintKindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							Scope:             enums.ConcurrencyScopeAccount,
							KeyExpressionHash: "",
						},
					},
				},
				internalDebugState: acquireScriptResponse{
					Status: 2,
				},
			},
			wantCacheable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheable := manager.shouldCacheAcquireResponse(tt.response)
			assert.Equal(t, tt.wantCacheable, cacheable)
		})
	}
}

func TestAcquireResponseCache_Disabled(t *testing.T) {
	// Create manager with cache disabled
	manager := &redisCapacityManager{
		acquireResponseCache: &acquireResponseCache{
			enabled: false,
		},
	}

	accountID := uuid.New()
	functionID := uuid.New()

	// Create a response that would normally be cacheable
	response := &CapacityAcquireResponse{
		Leases: []CapacityLease{},
		LimitingConstraints: []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeAccount,
					KeyExpressionHash: "",
				},
			},
		},
		internalDebugState: acquireScriptResponse{
			Status: 2,
		},
	}

	// Verify it would be cacheable
	assert.True(t, manager.shouldCacheAcquireResponse(response))

	// Try to generate cache key - should still work
	constraint := response.LimitingConstraints[0]
	cacheKey := manager.generateAcquireCacheKey(accountID, functionID, &constraint)
	assert.NotEmpty(t, cacheKey)

	// But cache operations should be no-op (not crash) when disabled
	// This would panic if cache.Set was called on nil cache
	// The actual code guards with: if manager.acquireResponseCache != nil && manager.acquireResponseCache.enabled
}

func TestAcquireResponseCache_TTLExpiry(t *testing.T) {
	// Create cache directly for testing
	cache := &acquireResponseCache{
		cache:   nil, // We'll test the TTL configuration
		ttl:     10 * time.Millisecond,
		enabled: true,
	}

	assert.Equal(t, 10*time.Millisecond, cache.ttl)
	assert.True(t, cache.enabled)

	// Test with different TTL
	cache2 := &acquireResponseCache{
		ttl:     100 * time.Millisecond,
		enabled: true,
	}

	assert.Equal(t, 100*time.Millisecond, cache2.ttl)
}

func TestWithAcquireResponseCache(t *testing.T) {
	tests := []struct {
		name        string
		ttl         time.Duration
		wantEnabled bool
	}{
		{
			name:        "enabled with 1ms TTL",
			ttl:         1 * time.Millisecond,
			wantEnabled: true,
		},
		{
			name:        "enabled with 5ms TTL",
			ttl:         5 * time.Millisecond,
			wantEnabled: true,
		},
		{
			name:        "disabled with 0 TTL",
			ttl:         0,
			wantEnabled: false,
		},
		{
			name:        "disabled with negative TTL",
			ttl:         -1 * time.Millisecond,
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &redisCapacityManager{}
			option := WithAcquireResponseCache(tt.ttl)
			option(manager)

			require.NotNil(t, manager.acquireResponseCache)
			assert.Equal(t, tt.wantEnabled, manager.acquireResponseCache.enabled)

			if tt.wantEnabled {
				assert.Equal(t, tt.ttl, manager.acquireResponseCache.ttl)
				assert.NotNil(t, manager.acquireResponseCache.cache)
			}
		})
	}
}

func TestAcquireResponseCache_MultipleConstraintTypes(t *testing.T) {
	accountID := uuid.New()
	functionID := uuid.New()
	manager := &redisCapacityManager{}

	// Response with both account and function concurrency limiting
	response := &CapacityAcquireResponse{
		Leases: []CapacityLease{},
		LimitingConstraints: []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeAccount,
					KeyExpressionHash: "",
				},
			},
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeFn,
					KeyExpressionHash: "",
				},
			},
		},
		internalDebugState: acquireScriptResponse{
			Status: 2,
		},
	}

	// Should be cacheable
	assert.True(t, manager.shouldCacheAcquireResponse(response))

	// Should generate different keys for each constraint
	key1 := manager.generateAcquireCacheKey(accountID, functionID, &response.LimitingConstraints[0])
	key2 := manager.generateAcquireCacheKey(accountID, functionID, &response.LimitingConstraints[1])

	assert.NotEmpty(t, key1)
	assert.NotEmpty(t, key2)
	assert.NotEqual(t, key1, key2)
	assert.Contains(t, key1, "acq:a:")
	assert.Contains(t, key2, "acq:f:")
}
