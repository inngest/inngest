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
				require.NoError(t, NewRedisSemaphoreManager(client).(OwnedSemaphoreManager).ReleaseOwnedSemaphore(t.Context(), account, sem.ID, "", run.String()))
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
