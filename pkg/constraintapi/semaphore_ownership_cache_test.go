package constraintapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestManualSemaphoreRunRefusalDoesNotPoisonCache(t *testing.T) {
	for _, state := range []string{"active", "finalized", "shared limit full"} {
		t.Run(state, func(t *testing.T) {
			cm, redis, client, clock := newSemaphoreTestEnv(t)
			cm.manualSemaphoreOwnership = true
			cache := NewConstraintCache(WithConstraintCacheManager(cm), WithConstraintCacheClock(clock),
				WithConstraintCacheEnable(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, time.Duration, time.Duration) {
					return true, time.Hour, time.Hour
				}))
			t.Cleanup(cache.cache.Stop)
			account, env, fn, run := uuid.New(), uuid.New(), uuid.New(), ulid.Make()
			sem := SemaphoreConstraint{ID: "fnkey:cache", Weight: 1, Release: SemaphoreReleaseManual}
			capacity := 2
			if state == "shared limit full" {
				capacity = 1
			}
			require.NoError(t, redis.Set(sem.CapacityKey(account), fmt.Sprint(capacity)))
			acquire := func(manager CapacityManager, run ulid.ULID) *CapacityAcquireResponse {
				t.Helper()
				key := uuid.NewString()
				res, err := manager.Acquire(t.Context(), &CapacityAcquireRequest{
					AccountID: account, EnvID: env, FunctionID: fn, IdempotencyKey: key, Amount: 1,
					LeaseIdempotencyKeys: []string{key}, LeaseRunIDs: map[string]ulid.ULID{key: run},
					Configuration: ConstraintConfig{FunctionVersion: 1, Semaphores: []Semaphore{{ID: sem.ID, Weight: sem.Weight, Release: sem.Release}}},
					Constraints:   []ConstraintItem{{Kind: ConstraintKindSemaphore, Semaphore: &sem}},
					CurrentTime:   clock.Now(), Duration: time.Minute, MaximumLifetime: time.Hour,
					Source: LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
				})
				require.NoError(t, err)
				return res
			}
			require.Len(t, acquire(cache, run).Leases, 1)
			if state == "finalized" {
				require.NoError(t, NewRedisSemaphoreManager(client).ReleaseSemaphore(t.Context(), account, sem.ID, "", run.String(), 1))
			}
			rejectedRun := run
			if state == "shared limit full" {
				rejectedRun = ulid.Make()
			}
			rejected := acquire(cache, rejectedRun)
			require.Empty(t, rejected.Leases)
			other := acquire(cache, ulid.Make())
			if state == "shared limit full" {
				require.Empty(t, other.Leases)
				require.Len(t, rejected.ExhaustedConstraints, 1, "real shared exhaustion is still reported and cached")
				require.Equal(t, 1, cache.cache.ItemCount())
				require.Empty(t, acquire(cm, ulid.Make()).Leases, "Redis agrees that shared capacity is exhausted")
				return
			}
			require.Len(t, other.Leases, 1, "one run's refusal must not block another run with spare shared capacity")
			require.Len(t, rejected.LimitingConstraints, 1, "the refused request still reports its limiting constraint")
			require.Empty(t, rejected.ExhaustedConstraints, "a run-specific refusal is not shared exhaustion")
		})
	}
}

func TestOwnedManualSemaphoreSkipsSharedRedisCache(t *testing.T) {
	for _, tracked := range []bool{false, true} {
		t.Run(fmt.Sprint(tracked), func(t *testing.T) {
			cm, redis, _, clock := newSemaphoreTestEnv(t)
			cm.manualSemaphoreOwnership = tracked
			cm.enableAcquireCache = enableAllCache()
			cm.acquireCacheTTL = func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (time.Duration, time.Duration) {
				return time.Minute, time.Minute
			}
			account, env, fn, run := uuid.New(), uuid.New(), uuid.New(), ulid.Make()
			sem := SemaphoreConstraint{ID: "fnkey:shared-cache", Weight: 1, Release: SemaphoreReleaseManual}
			ci := ConstraintItem{Kind: ConstraintKindSemaphore, Semaphore: &sem}
			require.NoError(t, redis.Set(sem.CapacityKey(account), "1"))
			cacheKey := cm.keyConstraintCache(account, env, fn, ci)
			// Model an entry left by another instance or an older writer.
			require.NoError(t, redis.Set(cacheKey, fmt.Sprint(clock.Now().Add(time.Minute).UnixMilli())))
			redis.SetTTL(cacheKey, time.Minute)
			config := ConstraintConfig{FunctionVersion: 1, Semaphores: []Semaphore{{ID: sem.ID, Weight: 1, Release: sem.Release}}}
			constraints := []ConstraintItem{ci}
			acquire := func(run ulid.ULID) *CapacityAcquireResponse {
				key := uuid.NewString()
				req := makeAcquireRequest(account, env, fn, clock, config, constraints, key)
				req.LeaseIdempotencyKeys = []string{key}
				req.LeaseRunIDs = map[string]ulid.ULID{key: run}
				res, err := cm.Acquire(t.Context(), req)
				require.NoError(t, err)
				return res
			}
			first := acquire(run)
			if !tracked {
				require.Empty(t, first.Leases)
				require.Equal(t, 1, first.internalDebugState.CacheHit, "legacy caching is unchanged")
				return
			}
			require.Len(t, first.Leases, 1)
			require.Zero(t, first.internalDebugState.CacheHit)
			_, err := cm.Release(t.Context(), &CapacityReleaseRequest{AccountID: account, LeaseID: first.Leases[0].LeaseID,
				IdempotencyKey: uuid.NewString(), LeaseIssuedAt: clock.Now(), Source: LeaseSource{Location: CallerLocationItemLease}})
			require.NoError(t, err)
			require.Len(t, acquire(run).Leases, 1, "owned retry must reach admission checking")
			require.Empty(t, acquire(ulid.Make()).Leases, "another run still cannot take the held slot")
			worker := SemaphoreConstraint{ID: "app:shared-cache", Weight: 1, Release: SemaphoreReleaseAuto}
			workerCI := ConstraintItem{Kind: ConstraintKindSemaphore, Semaphore: &worker}
			constraints = append(constraints, workerCI)
			config.Semaphores = append(config.Semaphores, Semaphore{ID: worker.ID, Weight: 1, Release: worker.Release})
			require.NoError(t, redis.Set(cm.keyConstraintCache(account, env, fn, workerCI), fmt.Sprint(clock.Now().Add(time.Minute).UnixMilli())))
			blocked := acquire(ulid.Make())
			require.Empty(t, blocked.Leases)
			require.Equal(t, 1, blocked.internalDebugState.CacheHit, "worker cache remains authoritative")
		})
	}
}
