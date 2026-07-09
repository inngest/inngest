package constraintapi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

func TestCapacityOperationUsageObservations(t *testing.T) {
	accountID, envID, fnID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	cm, _, clock, ctx := newTestSetup(t, nil)

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			AccountConcurrency:  5,
			FunctionConcurrency: 3,
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

	acquireReq := makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "usage-acquire")
	acquireReq.AppID = appID
	acquireResp, err := cm.Acquire(ctx, acquireReq)
	require.NoError(t, err)
	require.Len(t, acquireResp.Leases, 1)
	requireConcurrencyUsage(t, acquireResp.Usage, enums.ConcurrencyScopeAccount, 1, 5)
	requireConcurrencyUsage(t, acquireResp.Usage, enums.ConcurrencyScopeFn, 1, 3)

	extendResp, err := cm.ExtendLease(ctx, &CapacityExtendLeaseRequest{
		IdempotencyKey: "usage-extend",
		AccountID:      accountID,
		LeaseID:        acquireResp.Leases[0].LeaseID,
		Duration:       5 * time.Second,
	})
	require.NoError(t, err)
	require.NotNil(t, extendResp.LeaseID)
	require.Equal(t, accountID, extendResp.AccountID)
	require.Equal(t, envID, extendResp.EnvID)
	require.Equal(t, appID, extendResp.AppID)
	require.Equal(t, fnID, extendResp.FunctionID)
	requireConcurrencyUsage(t, extendResp.Usage, enums.ConcurrencyScopeAccount, 1, 5)
	requireConcurrencyUsage(t, extendResp.Usage, enums.ConcurrencyScopeFn, 1, 3)

	releaseResp, err := cm.Release(ctx, &CapacityReleaseRequest{
		IdempotencyKey: "usage-release",
		AccountID:      accountID,
		LeaseID:        *extendResp.LeaseID,
	})
	require.NoError(t, err)
	require.Equal(t, accountID, releaseResp.AccountID)
	require.Equal(t, envID, releaseResp.EnvID)
	require.Equal(t, appID, releaseResp.AppID)
	require.Equal(t, fnID, releaseResp.FunctionID)
	requireConcurrencyUsage(t, releaseResp.Usage, enums.ConcurrencyScopeAccount, 0, 5)
	requireConcurrencyUsage(t, releaseResp.Usage, enums.ConcurrencyScopeFn, 0, 3)
}

func TestAcquireConcurrencyUsageCanExceedCurrentLimit(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, _, clock, ctx := newTestSetup(t, nil)

	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
				Mode:  enums.ConcurrencyModeStep,
			},
		},
	}

	initialConfig := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 3,
		},
	}
	initialReq := makeAcquireRequest(accountID, envID, fnID, clock, initialConfig, constraints, "usage-over-limit-fill")
	initialReq.Amount = 3
	initialReq.LeaseIdempotencyKeys = []string{"item0", "item1", "item2"}
	initialResp, err := cm.Acquire(ctx, initialReq)
	require.NoError(t, err)
	require.Len(t, initialResp.Leases, 3)
	requireConcurrencyUsage(t, initialResp.Usage, enums.ConcurrencyScopeFn, 3, 3)

	loweredConfig := ConstraintConfig{
		FunctionVersion: 2,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 2,
		},
	}
	rejectedReq := makeAcquireRequest(accountID, envID, fnID, clock, loweredConfig, constraints, "usage-over-limit-reject")
	rejectedResp, err := cm.Acquire(ctx, rejectedReq)
	require.NoError(t, err)
	require.Empty(t, rejectedResp.Leases)
	requireConcurrencyUsage(t, rejectedResp.Usage, enums.ConcurrencyScopeFn, 3, 2)
}

func TestCapacityOperationIdempotencyReplaysAreMarked(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, _, clock, ctx := newTestSetup(t, nil)

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 3,
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

	acquireReq := makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "replay-acquire")
	acquireResp, err := cm.Acquire(ctx, acquireReq)
	require.NoError(t, err)
	require.False(t, acquireResp.OperationIdempotencyHit)

	acquireReplay, err := cm.Acquire(ctx, acquireReq)
	require.NoError(t, err)
	require.True(t, acquireReplay.OperationIdempotencyHit)

	extendReq := &CapacityExtendLeaseRequest{
		IdempotencyKey: "replay-extend",
		AccountID:      accountID,
		LeaseID:        acquireResp.Leases[0].LeaseID,
		Duration:       5 * time.Second,
	}
	extendResp, err := cm.ExtendLease(ctx, extendReq)
	require.NoError(t, err)
	require.False(t, extendResp.OperationIdempotencyHit)

	extendReplay, err := cm.ExtendLease(ctx, extendReq)
	require.NoError(t, err)
	require.True(t, extendReplay.OperationIdempotencyHit)

	releaseReq := &CapacityReleaseRequest{
		IdempotencyKey: "replay-release",
		AccountID:      accountID,
		LeaseID:        *extendResp.LeaseID,
	}
	releaseResp, err := cm.Release(ctx, releaseReq)
	require.NoError(t, err)
	require.False(t, releaseResp.OperationIdempotencyHit)

	releaseReplay, err := cm.Release(ctx, releaseReq)
	require.NoError(t, err)
	require.True(t, releaseReplay.OperationIdempotencyHit)
}

func TestCapacityOperationUsageCachesUseStoredConstraints(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	cm, r, clock, ctx := newTestSetup(t, nil)
	previousDebugLogs := enableDebugLogs
	enableDebugLogs = false
	t.Cleanup(func() {
		enableDebugLogs = previousDebugLogs
	})
	cm.enableDebugLogs = false

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 3,
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

	acquireReq := makeAcquireRequest(accountID, envID, fnID, clock, config, constraints, "cache-compact-acquire")
	acquireResp, err := cm.Acquire(ctx, acquireReq)
	require.NoError(t, err)
	require.Len(t, acquireResp.Leases, 1)

	extendReq := &CapacityExtendLeaseRequest{
		IdempotencyKey: "cache-compact-extend",
		AccountID:      accountID,
		LeaseID:        acquireResp.Leases[0].LeaseID,
		Duration:       5 * time.Second,
	}
	extendResp, err := cm.ExtendLease(ctx, extendReq)
	require.NoError(t, err)
	require.NotNil(t, extendResp.LeaseID)

	extendRaw, redisErr := r.Get(cm.keyOperationIdempotency(accountID, "ext", extendReq.IdempotencyKey))
	require.NoError(t, redisErr)
	requireUsageCacheUsesStoredConstraints(t, extendRaw)

	extendReplay, err := cm.ExtendLease(ctx, extendReq)
	require.NoError(t, err)
	require.True(t, extendReplay.OperationIdempotencyHit)

	releaseReq := &CapacityReleaseRequest{
		IdempotencyKey: "cache-compact-release",
		AccountID:      accountID,
		LeaseID:        *extendResp.LeaseID,
	}
	releaseResp, err := cm.Release(ctx, releaseReq)
	require.NoError(t, err)
	requireConcurrencyUsage(t, releaseResp.Usage, enums.ConcurrencyScopeFn, 0, 3)

	releaseRaw, redisErr := r.Get(cm.keyOperationIdempotency(accountID, "rel", releaseReq.IdempotencyKey))
	require.NoError(t, redisErr)
	requireUsageCacheUsesStoredConstraints(t, releaseRaw)

	releaseReplay, err := cm.Release(ctx, releaseReq)
	require.NoError(t, err)
	require.True(t, releaseReplay.OperationIdempotencyHit)
}

func requireConcurrencyUsage(
	t *testing.T,
	usage []ConstraintUsage,
	scope enums.ConcurrencyScope,
	used int,
	limit int,
) {
	t.Helper()

	for _, u := range usage {
		if u.Constraint.Kind != ConstraintKindConcurrency || u.Constraint.Concurrency == nil {
			continue
		}
		if u.Constraint.Concurrency.Scope != scope {
			continue
		}
		require.Equal(t, used, u.Used)
		require.Equal(t, limit, u.Limit)
		return
	}

	require.Failf(t, "missing concurrency usage", "scope=%s usage=%+v", scope, usage)
}

func requireUsageCacheUsesStoredConstraints(t *testing.T, raw string) {
	t.Helper()

	var cached struct {
		ConstraintUsage   []scriptConstraintUsage    `json:"cu"`
		StoredConstraints []SerializedConstraintItem `json:"sc"`
	}
	require.NoError(t, json.Unmarshal([]byte(raw), &cached))
	require.NotEmpty(t, cached.ConstraintUsage)
	require.NotEmpty(t, cached.StoredConstraints)

	for _, usage := range cached.ConstraintUsage {
		require.Positive(t, usage.Index)
		require.Zero(t, usage.Constraint)
	}
}
