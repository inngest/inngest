package constraintapi

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestManualSemaphoreStartRetry(t *testing.T) {
	for _, tracked := range []bool{false, true} {
		t.Run(fmt.Sprintf("ownership=%t", tracked), func(t *testing.T) {
			cm, redis, _, clock := newSemaphoreTestEnv(t)
			cm.manualSemaphoreOwnership = func(uuid.UUID) bool { return tracked }
			ctx := t.Context()
			account, env, fn, run := uuid.New(), uuid.New(), uuid.New(), ulid.Make()
			sem := SemaphoreConstraint{ID: "fnkey:retry", EvaluatedKeyHash: "app", Weight: 2, Release: SemaphoreReleaseManual}
			require.NoError(t, redis.Set(sem.CapacityKey(account), "2"))
			config := ConstraintConfig{FunctionVersion: 1, Semaphores: []Semaphore{{ID: sem.ID, Weight: 2, Release: sem.Release}}}
			acquire := func(key string, owner ulid.ULID) *CapacityAcquireResponse {
				t.Helper()
				res, err := cm.Acquire(ctx, &CapacityAcquireRequest{
					AccountID: account, EnvID: env, FunctionID: fn, IdempotencyKey: key,
					Amount: 1, LeaseIdempotencyKeys: []string{key}, LeaseRunIDs: map[string]ulid.ULID{key: owner},
					Configuration: config, Constraints: []ConstraintItem{{Kind: ConstraintKindSemaphore, Semaphore: &sem}},
					CurrentTime: clock.Now(), Duration: time.Minute, MaximumLifetime: time.Hour,
					Source: LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease, RunProcessingMode: RunProcessingModeBackground},
				})
				require.NoError(t, err)
				return res
			}
			usage := func(want string) {
				t.Helper()
				got, err := redis.Get(sem.UsageKey(account))
				require.NoError(t, err)
				require.Equal(t, want, got)
			}
			first := acquire("start-job", run)
			require.Len(t, first.Leases, 1)
			usage("2")
			// A second live attempt must not share the same run's admission.
			require.Empty(t, acquire("parallel-start", run).Leases)
			require.Empty(t, acquire("competing-run", ulid.Make()).Leases)

			// The active claim follows the request, not a renewable lease ID.
			clock.Advance(time.Second)
			redis.FastForward(time.Second)
			redis.SetTime(clock.Now())
			extended, err := cm.ExtendLease(ctx, &CapacityExtendLeaseRequest{
				AccountID: account, IdempotencyKey: "renew", LeaseID: first.Leases[0].LeaseID,
				Duration: time.Minute, LeaseIssuedAt: clock.Now(),
			})
			require.NoError(t, err)
			require.NotNil(t, extended.LeaseID)
			_, err = cm.Release(ctx, &CapacityReleaseRequest{AccountID: account, IdempotencyKey: "release-before-start",
				LeaseID: *extended.LeaseID, LeaseIssuedAt: clock.Now(), Source: LeaseSource{Location: CallerLocationItemLease}})
			require.NoError(t, err)
			usage("2")

			// Outlast the operation dedupe marker and the deleted lease's expiry.
			clock.Advance(10 * time.Minute)
			redis.FastForward(10 * time.Minute)
			redis.SetTime(clock.Now())
			scavenged, err := cm.Scavenge(ctx)
			require.NoError(t, err)
			require.Zero(t, scavenged.ReclaimedLeases)
			retry := acquire("start-job", run)
			if !tracked {
				require.Empty(t, retry.Leases, "legacy ownership remains unknown; do not guess")
				return
			}
			require.Len(t, retry.Leases, 1, "the tracked start retry reuses its own hold")
			usage("2")
			owned, getErr := redis.Get(sem.UsageKey(account) + ":owner:" + run.String())
			require.NoError(t, getErr)
			require.Equal(t, "2:"+retry.RequestID.String(), owned, "retry must not double-charge the run")
			require.False(t, redis.Exists(sem.UsageKey(account)+":owner:"+run.String()+":claim"), "no separate attempt key")
			require.Empty(t, acquire("another-live-retry", run).Leases)
			require.Empty(t, acquire("still-competing", ulid.Make()).Leases)

			// A delayed cleanup for the old lease must not revoke the retry.
			_, err = cm.Release(ctx, &CapacityReleaseRequest{AccountID: account, IdempotencyKey: "old-scavenge",
				LeaseID: *extended.LeaseID, LeaseIssuedAt: clock.Now(), Source: LeaseSource{Location: CallerLocationLeaseScavenge}})
			require.NoError(t, err)
			usage("2")
			sm := NewRedisSemaphoreManager(cm.client)
			require.NoError(t, sm.ReleaseSemaphore(ctx, account, sem.ID, sem.EvaluatedKeyHash, run.String(), 2))
			usage("0")
			require.Empty(t, acquire("finalized-retry", run).Leases, "a tombstoned run must not revive")
			other := acquire("next-run", ulid.Make())
			require.Len(t, other.Leases, 1)
			_, err = cm.Release(ctx, &CapacityReleaseRequest{AccountID: account, IdempotencyKey: "scavenge-after-finalize",
				LeaseID: retry.Leases[0].LeaseID, LeaseIssuedAt: clock.Now(), Source: LeaseSource{Location: CallerLocationLeaseScavenge}})
			require.NoError(t, err)
			usage("2")
		})
	}
}

// Run the same claim fencing on miniredis or a disposable localhost engine.
func TestManualSemaphoreStartRetryConcurrent(t *testing.T) {
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
	cm.manualSemaphoreOwnership = func(uuid.UUID) bool { return true }
	ctx := t.Context()
	account, env, fn, run := uuid.New(), uuid.New(), uuid.New(), ulid.Make()
	sem := SemaphoreConstraint{ID: SemaphoreIDFn(fn), Weight: 1, Release: SemaphoreReleaseManual}
	sm := NewRedisSemaphoreManager(client)
	_, err := sm.SetCapacity(ctx, account, sem.ID, uuid.NewString(), 1)
	require.NoError(t, err)
	request := func() *CapacityAcquireRequest {
		return &CapacityAcquireRequest{
			AccountID: account, EnvID: env, FunctionID: fn, IdempotencyKey: uuid.NewString(), Amount: 1,
			LeaseIdempotencyKeys: []string{"start-job"}, LeaseRunIDs: map[string]ulid.ULID{"start-job": run},
			Configuration: ConstraintConfig{FunctionVersion: 1, Semaphores: []Semaphore{{ID: sem.ID, Weight: sem.Weight, Release: sem.Release}}},
			Constraints:   []ConstraintItem{{Kind: ConstraintKindSemaphore, Semaphore: &sem}},
			CurrentTime:   clock.Now(), Duration: time.Minute, MaximumLifetime: time.Hour,
			Source: LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
		}
	}
	first, internalErr := cm.Acquire(ctx, request())
	require.NoError(t, internalErr)
	require.Len(t, first.Leases, 1)
	_, internalErr = cm.Release(ctx, &CapacityReleaseRequest{
		AccountID: account, IdempotencyKey: uuid.NewString(), LeaseID: first.Leases[0].LeaseID,
		LeaseIssuedAt: clock.Now(), Source: LeaseSource{Location: CallerLocationItemLease},
	})
	require.NoError(t, internalErr)

	type result struct {
		response *CapacityAcquireResponse
		err      error
	}
	const attempts = 16
	results := make(chan result, attempts)
	for range attempts {
		go func() {
			res, err := cm.Acquire(ctx, request())
			results <- result{response: res, err: err}
		}()
	}
	var retry *CapacityAcquireResponse
	for range attempts {
		res := <-results
		require.NoError(t, res.err)
		if len(res.response.Leases) > 0 {
			require.Nil(t, retry, "only one simultaneous retry may claim the held permit")
			retry = res.response
		}
	}
	require.NotNil(t, retry)
	_, usage, err := sm.GetCapacity(ctx, account, sem.ID, sem.EvaluatedKeyHash)
	require.NoError(t, err)
	require.EqualValues(t, 1, usage, "reuse must not double-charge capacity")
	// Scavenging the retry and then finalizing the run releases exactly once.
	_, internalErr = cm.Release(ctx, &CapacityReleaseRequest{
		AccountID: account, IdempotencyKey: uuid.NewString(), LeaseID: retry.Leases[0].LeaseID,
		LeaseIssuedAt: clock.Now(), Source: LeaseSource{Location: CallerLocationLeaseScavenge},
	})
	require.NoError(t, internalErr)
	require.NoError(t, sm.ReleaseSemaphore(ctx, account, sem.ID, sem.EvaluatedKeyHash, run.String(), 1))
	_, usage, err = sm.GetCapacity(ctx, account, sem.ID, sem.EvaluatedKeyHash)
	require.NoError(t, err)
	require.Zero(t, usage)
}

func TestManualSemaphoreStartRetryAfterScavenge(t *testing.T) {
	cm, redis, _, clock := newSemaphoreTestEnv(t)
	WithManualSemaphoreOwnership(func(uuid.UUID) bool { return true })(cm)
	account, env, fn, run := uuid.New(), uuid.New(), uuid.New(), ulid.Make()
	manual := SemaphoreConstraint{ID: SemaphoreIDFn(fn), Weight: 1, Release: SemaphoreReleaseManual}
	worker := SemaphoreConstraint{ID: SemaphoreIDApp(uuid.New()), Weight: 1, Release: SemaphoreReleaseAuto}
	require.NoError(t, redis.Set(manual.CapacityKey(account), "5"))
	require.NoError(t, redis.Set(worker.CapacityKey(account), "1"))
	config := ConstraintConfig{FunctionVersion: 1, Semaphores: []Semaphore{
		{ID: manual.ID, Weight: 1, Release: manual.Release}, {ID: worker.ID, Weight: 1, Release: worker.Release},
	}}
	constraints := []ConstraintItem{{Kind: ConstraintKindSemaphore, Semaphore: &manual}, {Kind: ConstraintKindSemaphore, Semaphore: &worker}}
	acquire := func(key string) *CapacityAcquireResponse {
		t.Helper()
		req := makeAcquireRequest(account, env, fn, clock, config, constraints, key)
		req.LeaseRunIDs = map[string]ulid.ULID{"item0": run}
		res, err := cm.Acquire(t.Context(), req)
		require.NoError(t, err)
		return res
	}
	usage := func(sem SemaphoreConstraint, want string) {
		t.Helper()
		got, err := redis.Get(sem.UsageKey(account))
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	first := acquire("crashed-start")
	require.Len(t, first.Leases, 1)
	clock.Advance(time.Minute)
	redis.FastForward(time.Minute)
	redis.SetTime(clock.Now())
	result, err := cm.Scavenge(t.Context())
	require.NoError(t, err)
	require.EqualValues(t, 1, result.ReclaimedLeases)
	usage(manual, "1")
	usage(worker, "0")
	ownerKey := manual.UsageKey(account) + ":owner:" + run.String()
	owner, getErr := redis.Get(ownerKey)
	require.NoError(t, getErr)
	require.Equal(t, "1:", owner, "expiry releases the attempt, not the run's admission")
	retry := acquire("retry-after-crash")
	require.Len(t, retry.Leases, 1)
	usage(manual, "1")
	usage(worker, "1")
	_, err = cm.Release(t.Context(), &CapacityReleaseRequest{AccountID: account, LeaseID: first.Leases[0].LeaseID,
		IdempotencyKey: "late-old-cleanup", LeaseIssuedAt: clock.Now(), Source: LeaseSource{Location: CallerLocationLeaseScavenge}})
	require.NoError(t, err)
	owner, getErr = redis.Get(ownerKey)
	require.NoError(t, getErr)
	require.Equal(t, "1:"+retry.RequestID.String(), owner, "old cleanup cannot revoke the retry")
	require.NoError(t, NewRedisSemaphoreManager(cm.client).ReleaseSemaphore(t.Context(), account, manual.ID, "", run.String(), 1))
	usage(manual, "0")
	require.Empty(t, acquire("after-finalization").Leases, "only finalization makes the run terminal")
}

func TestManualSemaphoreStartRetryDoesNotAdoptOldHolds(t *testing.T) {
	for _, kind := range []string{"legacy", "old-record", "active-claim", "changed-weight", "invalid-owner", "wrong-type-owner", "invalid-attempt", "finalized"} {
		t.Run(kind, func(t *testing.T) {
			cm, redis, _, clock := newSemaphoreTestEnv(t)
			cm.manualSemaphoreOwnership = func(uuid.UUID) bool { return true }
			account, env, fn, run := uuid.New(), uuid.New(), uuid.New(), ulid.Make()
			sem := SemaphoreConstraint{ID: "fnkey:old", Weight: 1, Release: SemaphoreReleaseManual}
			ownerKey := sem.UsageKey(account) + ":owner:" + run.String()
			require.NoError(t, redis.Set(sem.CapacityKey(account), "1"))
			require.NoError(t, redis.Set(sem.UsageKey(account), "1"))
			if kind != "legacy" {
				require.NoError(t, redis.Set(ownerKey, "1:"))
			}
			if kind == "old-record" {
				require.NoError(t, redis.Set(ownerKey, "1"))
			}
			if kind == "active-claim" {
				require.NoError(t, redis.Set(ownerKey, "1:"+ulid.Make().String()))
			}
			if kind == "changed-weight" {
				sem.Weight = 2
			}
			if kind == "invalid-owner" {
				require.NoError(t, redis.Set(ownerKey, "invalid"))
			}
			if kind == "wrong-type-owner" {
				redis.Del(ownerKey)
				redis.HSet(ownerKey, "invalid", "1")
			}
			if kind == "invalid-attempt" {
				require.NoError(t, redis.Set(ownerKey, "1:bad:attempt"))
			}
			if kind == "finalized" {
				require.NoError(t, redis.Set(ownerKey, "0:"))
			}
			res, err := cm.Acquire(t.Context(), &CapacityAcquireRequest{
				AccountID: account, EnvID: env, FunctionID: fn, IdempotencyKey: "retry", Amount: 1,
				LeaseIdempotencyKeys: []string{"retry"}, LeaseRunIDs: map[string]ulid.ULID{"retry": run},
				Configuration: ConstraintConfig{FunctionVersion: 1, Semaphores: []Semaphore{{ID: sem.ID, Weight: sem.Weight, Release: sem.Release}}}, Constraints: []ConstraintItem{{Kind: ConstraintKindSemaphore, Semaphore: &sem}},
				CurrentTime: clock.Now(), Duration: time.Minute, MaximumLifetime: time.Hour,
				Source: LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
			})
			if kind == "wrong-type-owner" {
				require.ErrorContains(t, err, "WRONGTYPE")
			} else if kind == "invalid-owner" || kind == "invalid-attempt" || kind == "old-record" {
				require.ErrorContains(t, err, "manual semaphore ownership is invalid")
			} else {
				require.NoError(t, err)
				require.Empty(t, res.Leases)
			}
			usage, getErr := redis.Get(sem.UsageKey(account))
			require.NoError(t, getErr)
			require.Equal(t, "1", usage)
		})
	}
}

func TestManualSemaphoreStartRetryHonorsOtherConstraints(t *testing.T) {
	cm, redis, _, clock := newSemaphoreTestEnv(t)
	cm.manualSemaphoreOwnership = func(uuid.UUID) bool { return true }
	account, env, fn, run := uuid.New(), uuid.New(), uuid.New(), ulid.Make()
	manual := SemaphoreConstraint{ID: "fnkey:retry", Weight: 1, Release: SemaphoreReleaseManual}
	auto := SemaphoreConstraint{ID: "app:worker", Weight: 1, Release: SemaphoreReleaseAuto}
	require.NoError(t, redis.Set(manual.CapacityKey(account), "1"))
	require.NoError(t, redis.Set(manual.UsageKey(account), "1"))
	ownerKey := manual.UsageKey(account) + ":owner:" + run.String()
	require.NoError(t, redis.Set(ownerKey, "1:"))
	res, err := cm.Acquire(t.Context(), &CapacityAcquireRequest{
		AccountID: account, EnvID: env, FunctionID: fn, IdempotencyKey: "retry", Amount: 1,
		LeaseIdempotencyKeys: []string{"retry"}, LeaseRunIDs: map[string]ulid.ULID{"retry": run},
		Configuration: ConstraintConfig{FunctionVersion: 1, Semaphores: []Semaphore{{ID: manual.ID, Weight: manual.Weight, Release: manual.Release}, {ID: auto.ID, Weight: auto.Weight, Release: auto.Release}}}, Constraints: []ConstraintItem{{Kind: ConstraintKindSemaphore, Semaphore: &manual}, {Kind: ConstraintKindSemaphore, Semaphore: &auto}},
		CurrentTime: clock.Now(), Duration: time.Minute, MaximumLifetime: time.Hour,
		Source: LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease},
	})
	require.NoError(t, err)
	require.Empty(t, res.Leases, "owning a manual permit cannot bypass unavailable worker capacity")
	claim, getErr := redis.Get(ownerKey)
	require.NoError(t, getErr)
	require.Equal(t, "1:", claim, "a rejected request must not claim the run")
}

func TestManualSemaphoreOwnershipRejectsDuplicateConstraints(t *testing.T) {
	run := ulid.Make()
	sem := &SemaphoreConstraint{ID: "fn:run", Release: SemaphoreReleaseManual}
	err := validateManualOwnershipRequest(&CapacityAcquireRequest{
		Amount: 1, LeaseIdempotencyKeys: []string{"a"}, LeaseRunIDs: map[string]ulid.ULID{"a": run},
		Constraints: []ConstraintItem{{Kind: ConstraintKindSemaphore, Semaphore: sem}, {Kind: ConstraintKindSemaphore, Semaphore: sem}},
	})
	require.ErrorContains(t, err, "duplicate manual semaphore")
}

func TestManualSemaphoreOwnershipPolicy(t *testing.T) {
	for _, tc := range []struct {
		name                string
		enabled, manual     bool
		amount, policyCalls int
		wantErr             bool
	}{
		{"owned single job", true, true, 1, 1, false},
		{"owned batch rejected", true, true, 2, 1, true},
		{"legacy manual batch", false, true, 2, 1, false},
		{"worker batch", true, false, 2, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cm, redis, _, clock := newSemaphoreTestEnv(t)
			account, env, fn := uuid.New(), uuid.New(), uuid.New()
			calls := 0
			WithManualSemaphoreOwnership(func(got uuid.UUID) bool {
				require.Equal(t, account, got)
				calls++
				return tc.enabled
			})(cm)
			release := SemaphoreReleaseAuto
			if tc.manual {
				release = SemaphoreReleaseManual
			}
			config := ConstraintConfig{FunctionVersion: 1}
			constraints := []ConstraintItem{}
			for _, id := range []string{"fn:policy-a", "fn:policy-b"} {
				sem := SemaphoreConstraint{ID: id, Weight: 1, Release: release}
				require.NoError(t, redis.Set(sem.CapacityKey(account), "2"))
				config.Semaphores = append(config.Semaphores, Semaphore{ID: id, Weight: 1, Release: release})
				constraints = append(constraints, ConstraintItem{Kind: ConstraintKindSemaphore, Semaphore: &sem})
			}
			req := makeAcquireRequest(account, env, fn, clock, config, constraints, "policy")
			req.Amount = tc.amount
			req.LeaseIdempotencyKeys = []string{"a", "b"}[:tc.amount]
			req.LeaseRunIDs = map[string]ulid.ULID{"a": ulid.Make(), "b": ulid.Make()}
			res, err := cm.Acquire(t.Context(), req)
			require.Equal(t, tc.policyCalls, calls, "resolve account policy once, only for manual requests")
			if tc.wantErr {
				require.ErrorContains(t, err, "single-job acquisition")
				for _, ci := range constraints {
					require.False(t, redis.Exists(ci.Semaphore.UsageKey(account)), "reject before charging capacity")
				}
				return
			}
			require.NoError(t, err)
			require.Len(t, res.Leases, tc.amount)
			stored, getErr := redis.Get(cm.keyRequestState(account, res.RequestID))
			require.NoError(t, getErr)
			if tc.enabled && tc.manual {
				require.Contains(t, stored, `"mso":true`)
			} else {
				require.NotContains(t, stored, `"mso"`)
			}
		})
	}
}
