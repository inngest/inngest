package constraintapi

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestInspectFunctionLeases(t *testing.T) {
	for _, name := range []string{"positive legacy usage", "zero usage", "missing usage", "live lease", "expired lease", "missing lease details", "missing request", "malformed request", "other function lease", "unindexed lease", "lease bound"} {
		t.Run(name, func(t *testing.T) {
			cm, r, client, _ := newSemaphoreTestEnv(t)
			ctx := context.Background()
			account, fn := uuid.New(), uuid.New()
			expression, bucket := "event.data.app_id", "app-a"
			key := semaphoreUsageKey(account, SemaphoreIDFnKey(fn, expression), bucket)
			other := semaphoreUsageKey(account, SemaphoreIDFnKey(uuid.New(), expression), bucket)
			require.NoError(t, r.Set(key, "1"))
			require.NoError(t, r.Set(other, "7"))
			wantErr := false
			switch name {
			case "zero usage":
				require.NoError(t, r.Set(key, "0"))
			case "missing usage":
				r.Del(key)
			case "positive legacy usage":
			default:
				wantErr = true
				lease := ulid.Make()
				if name == "expired lease" {
					lease = ulid.MustNew(ulid.Timestamp(time.Now().Add(-time.Hour)), nil)
				}
				_, err := r.ZAdd(cm.keyAccountLeases(account), float64(lease.Time()), lease.String())
				require.NoError(t, err)
				requestID := ulid.Make()
				r.HSet(cm.keyLeaseDetails(account, lease), "req", requestID.String())
				request := map[string]any{"f": fn.String(), "s": []any{}}
				if name == "other function lease" {
					request["f"] = uuid.New().String()
					wantErr = false
				}
				if name == "unindexed lease" {
					_, err := r.ZRem(cm.keyAccountLeases(account), lease.String())
					require.NoError(t, err)
					wantErr = false // Index inspection alone cannot discover this holder.
				}
				encoded, err := json.Marshal(request)
				require.NoError(t, err)
				require.NoError(t, r.Set(cm.keyRequestState(account, requestID), string(encoded)))
				if name == "missing lease details" {
					r.Del(cm.keyLeaseDetails(account, lease))
				}
				if name == "missing request" {
					r.Del(cm.keyRequestState(account, requestID))
				}
				if name == "malformed request" {
					require.NoError(t, r.Set(cm.keyRequestState(account, requestID), "garbage"))
				}
				if name == "lease bound" {
					for i := 0; i < 1001; i++ {
						_, err := r.ZAdd(cm.keyAccountLeases(account), 0, ulid.Make().String())
						require.NoError(t, err)
					}
				}
			}
			err := CheckFunctionLeasesDrained(ctx, client, account, fn)
			if wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NoError(t, CheckFunctionLeasesDrained(ctx, client, account, fn))
			}
			if name == "missing usage" {
				require.False(t, r.Exists(key))
			} else {
				value, err := r.Get(key)
				require.NoError(t, err)
				want := "1"
				if name == "zero usage" {
					want = "0"
				}
				require.Equal(t, want, value, "inspection never changes usage counters")
			}
			value, err := r.Get(other)
			require.NoError(t, err)
			require.Equal(t, "7", value)
		})
	}
}

func TestRetirementRefusesAnExpiredLeaseUntilReleased(t *testing.T) {
	cm, r, client, clock := newSemaphoreTestEnv(t)
	ctx := context.Background()
	account, env, fn := uuid.New(), uuid.New(), uuid.New()
	expression, bucket := "event.data.app_id", "app-a"
	sem := SemaphoreConstraint{ID: SemaphoreIDFnKey(fn, expression), EvaluatedKeyHash: bucket, Weight: 1, Release: SemaphoreReleaseManual}
	manager := NewRedisSemaphoreManager(client)
	_, err := manager.SetCapacity(ctx, account, sem.ID, "set-capacity", 1)
	require.NoError(t, err)
	res := acquireWithSemaphore(t, cm, clock, account, env, fn, ConstraintConfig{FunctionVersion: 1, Semaphores: []Semaphore{{ID: sem.ID, Weight: 1, Release: SemaphoreReleaseManual}}}, []ConstraintItem{{Kind: ConstraintKindSemaphore, Semaphore: &sem}}, "acquire-retirement")
	require.Len(t, res.Leases, 1)
	clock.Advance(time.Minute)
	require.Error(t, CheckFunctionLeasesDrained(ctx, client, account, fn), "expired is still releasable")
	key := semaphoreUsageKey(account, sem.ID, bucket)
	value, err := r.Get(key)
	require.NoError(t, err)
	require.Equal(t, "1", value)
	lease := res.Leases[0].LeaseID
	_, releaseErr := cm.Release(ctx, &CapacityReleaseRequest{IdempotencyKey: "normal-release", AccountID: account, LeaseID: lease, Source: LeaseSource{Service: ServiceExecutor, Location: CallerLocationItemLease}})
	require.NoError(t, releaseErr)
	value, err = r.Get(key)
	require.NoError(t, err)
	require.Equal(t, "1", value, "normal lease release preserves slots reserved for the function run")
	require.NoError(t, CheckFunctionLeasesDrained(ctx, client, account, fn))
	_, releaseErr = cm.Release(ctx, &CapacityReleaseRequest{IdempotencyKey: "late-scavenger", AccountID: account, LeaseID: lease, Source: LeaseSource{Service: ServiceConstraintScavenger, Location: CallerLocationLeaseScavenge}})
	require.NoError(t, releaseErr)
	value, err = r.Get(key)
	require.NoError(t, err)
	require.Equal(t, "1", value, "inspection preserves the old slot reserved for the function run")
}
