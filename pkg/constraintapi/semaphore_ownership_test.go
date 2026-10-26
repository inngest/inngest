package constraintapi

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// The same cases run on miniredis in CI or a disposable localhost Redis/Valkey.
func TestManualSemaphoreOwnership(t *testing.T) {
	cm, _, client, clock := newSemaphoreTestEnv(t)
	if addr := os.Getenv("SEMAPHORE_TEST_REDIS_ADDR"); addr != "" {
		host, _, err := net.SplitHostPort(addr)
		require.NoError(t, err)
		require.Equal(t, "127.0.0.1", host)
		client, err = rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{addr}, DisableCache: true})
		require.NoError(t, err)
		t.Cleanup(client.Close)
		cm.client = client
	}
	cm.manualSemaphoreOwnership = true
	sm := NewRedisSemaphoreManager(client)
	ctx := context.Background()
	account, env, fn := uuid.New(), uuid.New(), uuid.New()
	// Zero/missing legacy usage is not ownership evidence. Recovery must not
	// authorize cleanup while an untracked capacity lease could remain.
	strict := sm.(OwnedSemaphoreManager)
	for _, bucket := range []string{"legacy-missing", "legacy-zero"} {
		if bucket == "legacy-zero" {
			require.NoError(t, client.Do(ctx, client.B().Set().Key(semaphoreUsageKey(account, "legacy", bucket)).Value("0").Build()).Error())
		}
		require.ErrorIs(t, strict.ReleaseOwnedSemaphore(ctx, account, "legacy", bucket, ulid.Make().String()), ErrSemaphoreOwnershipUnknown)
	}
	sem := SemaphoreConstraint{ID: SemaphoreIDFn(fn), EvaluatedKeyHash: "app-a", Weight: 2, Release: SemaphoreReleaseManual}
	_, err := sm.SetCapacity(ctx, account, sem.ID, uuid.NewString(), 10)
	require.NoError(t, err)
	acquire := func(run ulid.ULID, c SemaphoreConstraint) *CapacityAcquireResponse {
		t.Helper()
		key := uuid.NewString()
		res, err := cm.Acquire(ctx, &CapacityAcquireRequest{
			AccountID: account, EnvID: env, FunctionID: fn, IdempotencyKey: key,
			Constraints:   []ConstraintItem{{Kind: ConstraintKindSemaphore, Semaphore: &c}},
			Configuration: ConstraintConfig{FunctionVersion: 1, Semaphores: []Semaphore{{ID: c.ID, Weight: c.Weight, Release: c.Release}}},
			Amount:        1, LeaseIdempotencyKeys: []string{key}, LeaseRunIDs: map[string]ulid.ULID{key: run},
			CurrentTime: clock.Now(), Duration: 5 * time.Second, MaximumLifetime: time.Hour,
			Source: LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease, RunProcessingMode: RunProcessingModeBackground},
		})
		require.NoError(t, err)
		return res
	}
	usage := func(want int64) {
		t.Helper()
		_, got, err := sm.GetCapacity(ctx, account, sem.ID, sem.EvaluatedKeyHash)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	owner, other := ulid.Make(), ulid.Make()
	first := acquire(owner, sem)
	require.Len(t, first.Leases, 1)
	require.Len(t, acquire(other, sem).Leases, 1)
	usage(4)

	// Never-started/legacy runs cannot spend a different run's permit.
	err = sm.ReleaseSemaphore(ctx, account, sem.ID, sem.EvaluatedKeyHash, ulid.Make().String(), 2)
	require.ErrorIs(t, err, ErrSemaphoreOwnershipUnknown)
	usage(4)

	require.NoError(t, strict.ReleaseOwnedSemaphore(ctx, account, sem.ID, sem.EvaluatedKeyHash, owner.String()))
	usage(2)
	// Erasing the old 60-second marker models a retry after its expiry. The
	// durable owner tombstone, not that marker, must prevent a second release.
	require.NoError(t, client.Do(ctx, client.B().Del().Key(semaphoreIdempotencyKey(account, "rel", owner.String())).Build()).Error())
	require.NoError(t, strict.ReleaseOwnedSemaphore(ctx, account, sem.ID, sem.EvaluatedKeyHash, owner.String()))
	usage(2)

	// Scavenge after finalization must not take the surviving holder's permit.
	_, internalErr := cm.Release(ctx, &CapacityReleaseRequest{
		AccountID: account, IdempotencyKey: uuid.NewString(), LeaseID: first.Leases[0].LeaseID,
		LeaseIssuedAt: clock.Now(), Source: LeaseSource{Location: CallerLocationLeaseScavenge},
	})
	require.NoError(t, internalErr)
	usage(2)
	// Nor may the already-finalized run reacquire the tombstoned permit.
	require.Empty(t, acquire(owner, sem).Leases)
	// After even the durable tombstone expires, ordinary callers must not
	// fall back to the unsafe legacy path and spend the surviving hold.
	require.NoError(t, client.Do(ctx, client.B().Del().Key(sem.UsageKey(account)+":owner:"+owner.String()).Build()).Error())
	err = NewRedisSemaphoreManager(client).ReleaseSemaphore(ctx, account, sem.ID, sem.EvaluatedKeyHash, owner.String(), 2)
	require.ErrorIs(t, err, ErrSemaphoreOwnershipUnknown)
	usage(2)

	// The same run can own more than one keyed permit; their releases must
	// not share the old account-wide release-idempotency marker.
	semB := sem
	semB.EvaluatedKeyHash = "app-b"
	second := acquire(other, semB)
	require.Len(t, second.Leases, 1)
	_, internalErr = cm.Release(ctx, &CapacityReleaseRequest{
		AccountID: account, IdempotencyKey: uuid.NewString(), LeaseID: second.Leases[0].LeaseID,
		LeaseIssuedAt: clock.Now(), Source: LeaseSource{Location: CallerLocationLeaseScavenge},
	})
	require.NoError(t, internalErr)
	require.NoError(t, sm.ReleaseSemaphore(ctx, account, semB.ID, semB.EvaluatedKeyHash, other.String(), 2))
	require.NoError(t, sm.ReleaseSemaphore(ctx, account, sem.ID, sem.EvaluatedKeyHash, other.String(), 2))
	usage(0)
}

func TestManualSemaphoreOwnershipRequiresRunID(t *testing.T) {
	_, _, _, clock := newSemaphoreTestEnv(t)
	_, _, _, _, err := buildRequestStateWithOwnership(&CapacityAcquireRequest{
		AccountID: uuid.New(), EnvID: uuid.New(), FunctionID: uuid.New(),
		Constraints:          []ConstraintItem{{Kind: ConstraintKindSemaphore, Semaphore: &SemaphoreConstraint{ID: "fn:test", Release: SemaphoreReleaseManual}}},
		LeaseIdempotencyKeys: []string{"no-run"}, CurrentTime: clock.Now(),
	}, true)
	require.ErrorContains(t, err, "requires a run ID")
}
