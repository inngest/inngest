package constraintapi

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestRedisCapacityManager_Scavenge_RateLimit(t *testing.T) {
	r := miniredis.RunT(t)
	ctx := context.Background()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	cm, err := NewRedisCapacityManager(
		WithClient(rc),
		WithShardName("test-shard"),
		WithClock(clock),
		WithEnableDebugLogs(true),
	)
	require.NoError(t, err)
	require.NotNil(t, cm)

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	var leaseID ulid.ULID
	leaseIdempotencyKey := "event1"

	config := ConstraintConfig{
		FunctionVersion: 1,
		RateLimit: []RateLimitConfig{
			{
				KeyExpressionHash: "expr-hash",
				Limit:             120,
				Period:            60,
			},
		},
	}

	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindRateLimit,
			RateLimit: &RateLimitConstraint{
				KeyExpressionHash: "expr-hash",
				EvaluatedKeyHash:  "test-value",
			},
		},
	}

	acquireReq := &CapacityAcquireRequest{
		AccountID:            accountID,
		EnvID:                envID,
		FunctionID:           fnID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{leaseIdempotencyKey},
		IdempotencyKey:       "event1",
		LeaseRunIDs:          nil,
		Duration:             5 * time.Second,
		Source: LeaseSource{
			Service:           ServiceExecutor,
			Location:          CallerLocationSchedule,
			RunProcessingMode: RunProcessingModeBackground,
		},
		Configuration:   config,
		Constraints:     constraints,
		CurrentTime:     clock.Now(),
		MaximumLifetime: time.Minute,
	}

	t.Run("Acquire Lease", func(t *testing.T) {
		enableDebugLogs = true

		var err error
		resp, err := cm.Acquire(ctx, acquireReq)
		require.NoError(t, err)
		require.NotNil(t, resp)

		t.Log(resp.internalDebugState.Debug)
		require.Equal(t, 3, resp.internalDebugState.Status)
		require.Equal(t, 1, resp.internalDebugState.Granted)

		// One lease should have been granted
		require.Len(t, resp.Leases, 1)
		require.Equal(t, leaseIdempotencyKey, resp.Leases[0].IdempotencyKey)

		leaseID = resp.Leases[0].LeaseID

		// Verify scavenger shard contains accountID with correct score (expiry time)
		require.True(t, r.Exists(cm.keyScavengerShard()))
		score, err := r.ZScore(cm.keyScavengerShard(), accountID.String())
		require.NoError(t, err)
		require.Equal(t, float64(clock.Now().Add(5*time.Second).UnixMilli()), score)

		// Verify account leases sorted set contains the lease
		require.True(t, r.Exists(cm.keyAccountLeases(accountID)))
		members, err := r.ZMembers(cm.keyAccountLeases(accountID))
		require.NoError(t, err)
		require.Len(t, members, 1)
		require.Contains(t, members, leaseID.String())

		// Verify lease details exist
		require.True(t, r.Exists(cm.keyLeaseDetails(accountID, leaseID)))
	})

	t.Run("Scavenge - No Expired Leases", func(t *testing.T) {
		enableDebugLogs = true

		// Advance clock by 2 seconds (not enough to expire the 5 second lease)
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		var err error
		res, err := cm.Scavenge(ctx)
		require.NoError(t, err)
		require.NotNil(t, res)

		// Verify ScavengeResult shows 0 expired leases
		require.Equal(t, 0, res.TotalExpiredLeasesCount, "No leases should be expired yet")
		require.Equal(t, 0, res.ReclaimedLeases, "No leases should be reclaimed")
		require.Equal(t, 0, res.TotalExpiredAccountsCount, "No accounts should have expired leases")
		require.Equal(t, 0, res.ScannedAccounts, "No accounts should be scanned")

		// Verify lease still exists
		members, err := r.ZMembers(cm.keyAccountLeases(accountID))
		require.NoError(t, err)
		require.Len(t, members, 1, "Lease should still exist")
		require.Contains(t, members, leaseID.String())

		// Verify lease details still exist
		require.True(t, r.Exists(cm.keyLeaseDetails(accountID, leaseID)))
	})

	t.Run("Scavenge - With Expired Lease", func(t *testing.T) {
		enableDebugLogs = true

		// Advance clock by 4 more seconds (total 6 seconds, past 5 second expiry)
		clock.Advance(4 * time.Second)
		r.FastForward(4 * time.Second)
		r.SetTime(clock.Now())

		t.Logf("Current time: %v", clock.Now())
		t.Logf("Lease expiry: %v", leaseID.Timestamp().Add(5*time.Second))

		var err error
		res, err := cm.Scavenge(ctx)
		require.NoError(t, err)
		require.NotNil(t, res)

		// Verify ScavengeResult
		require.Equal(t, 1, res.TotalExpiredLeasesCount, "Should have 1 expired lease")
		require.Equal(t, 1, res.ReclaimedLeases, "Should have reclaimed 1 lease")
		require.Equal(t, 1, res.TotalExpiredAccountsCount, "Should have 1 account with expired leases")
		require.Equal(t, 1, res.ScannedAccounts, "Should have scanned 1 account")

		// Verify lease is removed from account leases sorted set
		accountLeasesKey := cm.keyAccountLeases(accountID)
		if r.Exists(accountLeasesKey) {
			members, err := r.ZMembers(accountLeasesKey)
			require.NoError(t, err)
			require.Len(t, members, 0, "Expired lease should be removed from account leases")
		} else {
			// Key doesn't exist, which means all leases were removed (expected behavior)
			require.True(t, true, "Account leases key removed as expected")
		}

		// Verify lease details hash is deleted
		require.False(t, r.Exists(cm.keyLeaseDetails(accountID, leaseID)), "Lease details should be deleted")

		// Verify scavenger shard is updated (account removed if no more leases)
		exists := r.Exists(cm.keyScavengerShard())
		if exists {
			_, err := r.ZScore(cm.keyScavengerShard(), accountID.String())
			require.Error(t, err, "Account should be removed from scavenger shard when all leases are expired")
		}
	})
}

func TestRedisCapacityManager_Scavenge_Concurrency(t *testing.T) {
	r := miniredis.RunT(t)
	ctx := context.Background()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	cm, err := NewRedisCapacityManager(
		WithClient(rc),
		WithShardName("test-shard"),
		WithClock(clock),
		WithEnableDebugLogs(true),
	)
	require.NoError(t, err)
	require.NotNil(t, cm)

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	var leaseID ulid.ULID
	leaseIdempotencyKey := "event1"

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			AccountConcurrency:  20,
			FunctionConcurrency: 5,
		},
	}

	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeAccount,
			},
		},
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
		},
	}

	acquireReq := &CapacityAcquireRequest{
		AccountID:            accountID,
		EnvID:                envID,
		FunctionID:           fnID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{leaseIdempotencyKey},
		IdempotencyKey:       "event1",
		LeaseRunIDs:          nil,
		Duration:             5 * time.Second,
		Source: LeaseSource{
			Service:           ServiceExecutor,
			Location:          CallerLocationSchedule,
			RunProcessingMode: RunProcessingModeBackground,
		},
		Configuration:   config,
		Constraints:     constraints,
		CurrentTime:     clock.Now(),
		MaximumLifetime: time.Minute,
	}

	t.Run("Acquire Lease", func(t *testing.T) {
		enableDebugLogs = true

		var err error
		resp, err := cm.Acquire(ctx, acquireReq)
		require.NoError(t, err)
		require.NotNil(t, resp)

		t.Log(resp.internalDebugState.Debug)
		require.Equal(t, 3, resp.internalDebugState.Status)
		require.Equal(t, 1, resp.internalDebugState.Granted)

		// One lease should have been granted
		require.Len(t, resp.Leases, 1)

		leaseID = resp.Leases[0].LeaseID

		// Verify scavenger shard contains accountID with correct score (expiry time)
		require.True(t, r.Exists(cm.keyScavengerShard()))
		score, err := r.ZScore(cm.keyScavengerShard(), accountID.String())
		require.NoError(t, err)
		require.Equal(t, float64(clock.Now().Add(5*time.Second).UnixMilli()), score)

		// Verify account leases sorted set contains the lease
		require.True(t, r.Exists(cm.keyAccountLeases(accountID)))
		members, err := r.ZMembers(cm.keyAccountLeases(accountID))
		require.NoError(t, err)
		require.Len(t, members, 1)
		require.Contains(t, members, leaseID.String())

		// Verify lease details exist
		require.True(t, r.Exists(cm.keyLeaseDetails(accountID, leaseID)))
	})

	t.Run("Scavenge - No Expired Leases", func(t *testing.T) {
		enableDebugLogs = true

		// Advance clock by 2 seconds (not enough to expire the 5 second lease)
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		var err error
		res, err := cm.Scavenge(ctx)
		require.NoError(t, err)
		require.NotNil(t, res)

		// Verify ScavengeResult shows 0 expired leases
		require.Equal(t, 0, res.TotalExpiredLeasesCount, "No leases should be expired yet")
		require.Equal(t, 0, res.ReclaimedLeases, "No leases should be reclaimed")
		require.Equal(t, 0, res.TotalExpiredAccountsCount, "No accounts should have expired leases")
		require.Equal(t, 0, res.ScannedAccounts, "No accounts should be scanned")

		// Verify lease still exists
		members, err := r.ZMembers(cm.keyAccountLeases(accountID))
		require.NoError(t, err)
		require.Len(t, members, 1, "Lease should still exist")
		require.Contains(t, members, leaseID.String())

		// Verify lease details still exist
		require.True(t, r.Exists(cm.keyLeaseDetails(accountID, leaseID)))
	})

	t.Run("Scavenge - With Expired Lease", func(t *testing.T) {
		enableDebugLogs = true

		// Advance clock by 4 more seconds (total 6 seconds, past 5 second expiry)
		clock.Advance(4 * time.Second)
		r.FastForward(4 * time.Second)
		r.SetTime(clock.Now())

		t.Logf("Current time: %v", clock.Now())
		t.Logf("Lease expiry: %v", leaseID.Timestamp().Add(5*time.Second))

		var err error
		res, err := cm.Scavenge(ctx)
		require.NoError(t, err)
		require.NotNil(t, res)

		// Verify ScavengeResult
		require.Equal(t, 1, res.TotalExpiredLeasesCount, "Should have 1 expired lease")
		require.Equal(t, 1, res.ReclaimedLeases, "Should have reclaimed 1 lease")
		require.Equal(t, 1, res.TotalExpiredAccountsCount, "Should have 1 account with expired leases")
		require.Equal(t, 1, res.ScannedAccounts, "Should have scanned 1 account")

		// Verify lease is removed from account leases sorted set
		accountLeasesKey := cm.keyAccountLeases(accountID)
		if r.Exists(accountLeasesKey) {
			members, err := r.ZMembers(accountLeasesKey)
			require.NoError(t, err)
			require.Len(t, members, 0, "Expired lease should be removed from account leases")
		} else {
			// Key doesn't exist, which means all leases were removed (expected behavior)
			require.True(t, true, "Account leases key removed as expected")
		}

		// Verify lease details hash is deleted
		require.False(t, r.Exists(cm.keyLeaseDetails(accountID, leaseID)), "Lease details should be deleted")

		// Verify scavenger shard is updated (account removed if no more leases)
		exists := r.Exists(cm.keyScavengerShard())
		if exists {
			_, err := r.ZScore(cm.keyScavengerShard(), accountID.String())
			require.Error(t, err, "Account should be removed from scavenger shard when all leases are expired")
		}
	})
}
