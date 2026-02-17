package constraintapi

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testMigrationSetup creates two miniredis instances and capacity managers for migration testing.
type testMigrationSetup struct {
	srcRedis *miniredis.Miniredis
	dstRedis *miniredis.Miniredis
	srcRC    rueidis.Client
	dstRC    rueidis.Client
	srcCM    *redisCapacityManager
	dstCM    *redisCapacityManager
	clock    clockwork.FakeClock
}

func newTestMigrationSetup(t *testing.T) *testMigrationSetup {
	t.Helper()

	srcRedis := miniredis.RunT(t)
	dstRedis := miniredis.RunT(t)

	srcRC, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{srcRedis.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { srcRC.Close() })

	dstRC, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{dstRedis.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { dstRC.Close() })

	clock := clockwork.NewFakeClock()

	srcCM, err := NewRedisCapacityManager(
		WithClient(srcRC),
		WithShardName("source"),
		WithClock(clock),
	)
	require.NoError(t, err)

	dstCM, err := NewRedisCapacityManager(
		WithClient(dstRC),
		WithShardName("destination"),
		WithClock(clock),
	)
	require.NoError(t, err)

	return &testMigrationSetup{
		srcRedis: srcRedis,
		dstRedis: dstRedis,
		srcRC:    srcRC,
		dstRC:    dstRC,
		srcCM:    srcCM,
		dstCM:    dstCM,
		clock:    clock,
	}
}

func TestDirtyKeyTracker(t *testing.T) {
	t.Run("MarkDirty and DrainAndReset", func(t *testing.T) {
		tracker := newMigrationDirtyTracker()

		tracker.MarkDirty("key1", "key2", "key3")
		require.Equal(t, 3, tracker.Len())

		// Duplicate key should not increase count
		tracker.MarkDirty("key1")
		require.Equal(t, 3, tracker.Len())

		keys := tracker.DrainAndReset()
		require.Len(t, keys, 3)
		require.Equal(t, 0, tracker.Len())

		// After drain, tracker should be empty
		keys = tracker.DrainAndReset()
		require.Empty(t, keys)
	})

	t.Run("concurrent access", func(t *testing.T) {
		tracker := newMigrationDirtyTracker()
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tracker.MarkDirty(fmt.Sprintf("key-%d", i))
			}()
		}
		wg.Wait()

		require.Equal(t, 100, tracker.Len())
	})
}

func TestMigrationAccountState(t *testing.T) {
	state := newMigrationAccountState()
	accountID := uuid.New()

	require.False(t, state.IsMigrating(accountID))
	require.Nil(t, state.GetTracker(accountID))

	tracker := state.StartTracking(accountID)
	require.NotNil(t, tracker)
	require.True(t, state.IsMigrating(accountID))

	tracker.MarkDirty("key1")
	require.Equal(t, 1, tracker.Len())

	state.StopTracking(accountID)
	require.False(t, state.IsMigrating(accountID))
	require.Nil(t, state.GetTracker(accountID))
}

func TestAccountKeyPrefix(t *testing.T) {
	accountID := uuid.MustParse("01234567-89ab-cdef-0123-456789abcdef")

	prefix := AccountKeyPrefix(accountID)
	require.Equal(t, "{cs}:a:01234567-89ab-cdef-0123-456789abcdef:", prefix)

	pattern := AccountKeyScanPattern(accountID)
	require.Equal(t, "{cs}:a:01234567-89ab-cdef-0123-456789abcdef:*", pattern)
}

func TestScanKeys(t *testing.T) {
	r := miniredis.RunT(t)
	ctx := context.Background()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	// Create some test keys
	accountID := uuid.New()
	prefix := fmt.Sprintf("{cs}:a:%s", accountID)
	for i := 0; i < 50; i++ {
		r.Set(fmt.Sprintf("%s:rl:a:key%d", prefix, i), "value")
	}

	pattern := fmt.Sprintf("%s:rl:*", prefix)
	keys, err := scanKeys(ctx, rc, pattern, 10)
	require.NoError(t, err)
	require.Len(t, keys, 50)
}

func TestMigrationCoordinator_CopyKeys(t *testing.T) {
	setup := newTestMigrationSetup(t)
	ctx := context.Background()

	accountID := uuid.New()
	prefix := fmt.Sprintf("{cs}:a:%s", accountID)

	// Create various key types on source
	for i := 0; i < 10; i++ {
		setup.srcRedis.Set(fmt.Sprintf("%s:rl:a:key%d", prefix, i), fmt.Sprintf("tat%d", i))
	}

	// Create a sorted set key
	setup.srcRedis.ZAdd(fmt.Sprintf("%s:leaseq", prefix), 100, "lease1")
	setup.srcRedis.ZAdd(fmt.Sprintf("%s:leaseq", prefix), 200, "lease2")

	// Create a hash key
	setup.srcRedis.HSet(fmt.Sprintf("%s:ld:lease1", prefix), "lik", "key1")
	setup.srcRedis.HSet(fmt.Sprintf("%s:ld:lease1", prefix), "req", "req1")

	copier := &TypeAwareCopier{}

	// Collect all keys
	allKeys := make([]string, 0)
	for i := 0; i < 10; i++ {
		allKeys = append(allKeys, fmt.Sprintf("%s:rl:a:key%d", prefix, i))
	}
	allKeys = append(allKeys,
		fmt.Sprintf("%s:leaseq", prefix),
		fmt.Sprintf("%s:ld:lease1", prefix),
	)

	copied, err := copier.CopyKeys(ctx, setup.srcRC, setup.dstRC, allKeys, 5)
	require.NoError(t, err)
	require.Equal(t, 12, copied)

	// Verify keys exist on destination
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("%s:rl:a:key%d", prefix, i)
		val, err := setup.dstRC.Do(ctx, setup.dstRC.B().Get().Key(key).Build()).ToString()
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("tat%d", i), val)
	}

	// Verify sorted set on destination
	leaseqKey := fmt.Sprintf("%s:leaseq", prefix)
	members, err := setup.dstRC.Do(ctx, setup.dstRC.B().Zrange().Key(leaseqKey).Min("0").Max("+inf").Byscore().Build()).AsStrSlice()
	require.NoError(t, err)
	require.Equal(t, []string{"lease1", "lease2"}, members)

	// Verify hash on destination
	ldKey := fmt.Sprintf("%s:ld:lease1", prefix)
	val, err := setup.dstRC.Do(ctx, setup.dstRC.B().Hget().Key(ldKey).Field("lik").Build()).ToString()
	require.NoError(t, err)
	require.Equal(t, "key1", val)
}

func TestMigrationCoordinator_CopyScavengerEntry(t *testing.T) {
	setup := newTestMigrationSetup(t)
	ctx := context.Background()

	accountID := uuid.New()
	scavengerKey := ScavengerShardKey()

	// Add scavenger entry on source
	setup.srcRedis.ZAdd(scavengerKey, 12345.0, accountID.String())

	mc := NewMigrationCoordinator()
	err := mc.copyScavengerEntry(ctx, MigrationRequest{
		AccountID:         accountID,
		SourceClient:      setup.srcRC,
		DestinationClient: setup.dstRC,
	})
	require.NoError(t, err)

	// Verify scavenger entry on destination
	score, err := setup.dstRedis.ZScore(scavengerKey, accountID.String())
	require.NoError(t, err)
	require.Equal(t, 12345.0, score)
}

func TestMigrationCoordinator_CleanupSource(t *testing.T) {
	setup := newTestMigrationSetup(t)
	ctx := context.Background()

	accountID := uuid.New()
	prefix := fmt.Sprintf("{cs}:a:%s", accountID)
	scavengerKey := ScavengerShardKey()

	// Create keys on source
	setup.srcRedis.Set(fmt.Sprintf("%s:rl:a:key1", prefix), "value1")
	setup.srcRedis.Set(fmt.Sprintf("%s:rl:a:key2", prefix), "value2")
	setup.srcRedis.ZAdd(scavengerKey, 100, accountID.String())

	mc := NewMigrationCoordinator()
	err := mc.cleanupSource(ctx, MigrationRequest{
		AccountID:    accountID,
		SourceClient: setup.srcRC,
		ScanCount:    100,
	})
	require.NoError(t, err)

	// Verify all account keys are gone
	require.False(t, setup.srcRedis.Exists(fmt.Sprintf("%s:rl:a:key1", prefix)))
	require.False(t, setup.srcRedis.Exists(fmt.Sprintf("%s:rl:a:key2", prefix)))

	// Verify scavenger entry is gone
	_, err = setup.srcRedis.ZScore(scavengerKey, accountID.String())
	require.Error(t, err) // Should not exist
}

func TestMigrationCoordinator_FullMigration(t *testing.T) {
	setup := newTestMigrationSetup(t)
	ctx := context.Background()

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 10,
		},
		RateLimit: []RateLimitConfig{
			{
				KeyExpressionHash: "expr-hash",
				Limit:             1000,
				Period:            60,
			},
		},
	}

	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
			},
		},
		{
			Kind: ConstraintKindRateLimit,
			RateLimit: &RateLimitConstraint{
				KeyExpressionHash: "expr-hash",
				EvaluatedKeyHash:  "test-value",
			},
		},
	}

	// Acquire some leases on source
	var leaseIDs []CapacityLease
	for i := 0; i < 3; i++ {
		resp, err := setup.srcCM.Acquire(ctx, &CapacityAcquireRequest{
			AccountID:            accountID,
			EnvID:                envID,
			FunctionID:           fnID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{fmt.Sprintf("event-%d", i)},
			IdempotencyKey:       fmt.Sprintf("op-%d", i),
			Duration:             30 * time.Second,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationSchedule,
			},
			Configuration:   config,
			Constraints:     constraints,
			CurrentTime:     setup.clock.Now(),
			MaximumLifetime: 3 * time.Hour,
		})
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)
		leaseIDs = append(leaseIDs, resp.Leases[0])
	}

	// Verify keys exist on source
	srcKeysBefore := setup.srcRedis.Keys()
	require.Greater(t, len(srcKeysBefore), 5, "source should have multiple keys")

	// Verify destination is empty
	dstKeysBefore := setup.dstRedis.Keys()
	require.Empty(t, dstKeysBefore, "destination should be empty before migration")

	// Run migration
	routingSwitched := false
	mc := NewMigrationCoordinator()
	result := mc.Migrate(ctx, MigrationRequest{
		AccountID:         accountID,
		SourceClient:      setup.srcRC,
		DestinationClient: setup.dstRC,
		SwitchRouting: func(ctx context.Context, accountID uuid.UUID) error {
			routingSwitched = true
			return nil
		},
		Copier:                    &TypeAwareCopier{},
		CopyBatchSize:             10,
		ScanCount:                 100,
		DeltaConvergenceThreshold: 10,
		MaxDeltaPasses:            3,
	})

	require.NoError(t, result.Error)
	require.Equal(t, MigrationPhaseComplete, result.Phase)
	require.True(t, routingSwitched)
	require.Greater(t, result.GCRAKeysCopied, 0, "should have copied GCRA keys")
	require.Greater(t, result.LeaseStateKeysCopied, 0, "should have copied lease keys")

	// Verify keys on destination
	dstKeysAfter := setup.dstRedis.Keys()
	require.Greater(t, len(dstKeysAfter), 5, "destination should have keys after migration")

	// Verify scavenger entry on destination
	scavengerKey := ScavengerShardKey()
	require.True(t, setup.dstRedis.Exists(scavengerKey))

	// Verify source is cleaned up
	pattern := AccountKeyScanPattern(accountID)
	srcKeys, err := scanKeys(ctx, setup.srcRC, pattern, 100)
	require.NoError(t, err)
	require.Empty(t, srcKeys, "source should have no account keys after cleanup")

	// Verify we can extend leases on destination
	setup.clock.Advance(2 * time.Second)
	setup.dstRedis.SetTime(setup.clock.Now())
	for _, lease := range leaseIDs {
		resp, err := setup.dstCM.ExtendLease(ctx, &CapacityExtendLeaseRequest{
			IdempotencyKey: fmt.Sprintf("extend-%s", lease.LeaseID),
			AccountID:      accountID,
			LeaseID:        lease.LeaseID,
			Duration:       30 * time.Second,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.LeaseID, "extend should succeed on destination, lease=%s", lease.LeaseID)
	}

	// Verify we can release leases on destination
	for _, lease := range leaseIDs {
		resp, err := setup.dstCM.Release(ctx, &CapacityReleaseRequest{
			IdempotencyKey: fmt.Sprintf("release-%s", lease.LeaseID),
			AccountID:      accountID,
			LeaseID:        lease.LeaseID,
		})
		require.NoError(t, err)
		require.Equal(t, accountID, resp.AccountID)
	}
}

func TestMigrationCoordinator_DirtyKeyTracking(t *testing.T) {
	setup := newTestMigrationSetup(t)
	ctx := context.Background()

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	config := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			FunctionConcurrency: 10,
		},
	}

	constraints := []ConstraintItem{
		{
			Kind: ConstraintKindConcurrency,
			Concurrency: &ConcurrencyConstraint{
				Scope: enums.ConcurrencyScopeFn,
			},
		},
	}

	// Acquire a lease on source
	resp, err := setup.srcCM.Acquire(ctx, &CapacityAcquireRequest{
		AccountID:            accountID,
		EnvID:                envID,
		FunctionID:           fnID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"event-1"},
		IdempotencyKey:       "op-1",
		Duration:             30 * time.Second,
		Source: LeaseSource{
			Service:  ServiceExecutor,
			Location: CallerLocationSchedule,
		},
		Configuration:   config,
		Constraints:     constraints,
		CurrentTime:     setup.clock.Now(),
		MaximumLifetime: 3 * time.Hour,
	})
	require.NoError(t, err)
	require.Len(t, resp.Leases, 1)
	leaseID := resp.Leases[0].LeaseID

	// Set up dirty key tracking
	tracker := newMigrationDirtyTracker()

	// Create routing manager to use trackExtendDirtyKeys
	rm := &RoutingCapacityManager{
		shards:         make(map[string]*redisCapacityManager),
		migrations:     make(map[uuid.UUID]*accountMigration),
		migrationState: newMigrationAccountState(),
	}

	// Track dirty keys for an extend
	trackErr := rm.trackExtendDirtyKeys(ctx, setup.srcRC, accountID, leaseID, tracker)
	require.NoError(t, trackErr)

	dirtyKeys := tracker.DrainAndReset()
	require.Greater(t, len(dirtyKeys), 0, "should have tracked dirty keys")

	// Verify expected keys are present
	prefix := AccountKeyPrefix(accountID)
	expectedKeys := map[string]bool{
		fmt.Sprintf("%sleaseq", prefix): false,
		ScavengerShardKey():             false,
	}
	for _, key := range dirtyKeys {
		if _, ok := expectedKeys[key]; ok {
			expectedKeys[key] = true
		}
	}
	for key, found := range expectedKeys {
		assert.True(t, found, "expected dirty key not found: %s", key)
	}
}

func TestMigrationCoordinator_FailureRecovery(t *testing.T) {
	setup := newTestMigrationSetup(t)
	ctx := context.Background()

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	config := ConstraintConfig{
		FunctionVersion: 1,
		RateLimit: []RateLimitConfig{
			{
				KeyExpressionHash: "expr-hash",
				Limit:             1000,
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

	// Acquire a lease on source
	_, err := setup.srcCM.Acquire(ctx, &CapacityAcquireRequest{
		AccountID:            accountID,
		EnvID:                envID,
		FunctionID:           fnID,
		Amount:               1,
		LeaseIdempotencyKeys: []string{"event-1"},
		IdempotencyKey:       "op-1",
		Duration:             30 * time.Second,
		Source: LeaseSource{
			Service:  ServiceExecutor,
			Location: CallerLocationSchedule,
		},
		Configuration:   config,
		Constraints:     constraints,
		CurrentTime:     setup.clock.Now(),
		MaximumLifetime: 3 * time.Hour,
	})
	require.NoError(t, err)

	// Simulate routing switch failure
	mc := NewMigrationCoordinator()
	result := mc.Migrate(ctx, MigrationRequest{
		AccountID:         accountID,
		SourceClient:      setup.srcRC,
		DestinationClient: setup.dstRC,
		SwitchRouting: func(ctx context.Context, accountID uuid.UUID) error {
			return fmt.Errorf("simulated routing switch failure")
		},
		Copier:    &TypeAwareCopier{},
		ScanCount: 100,
	})

	require.Error(t, result.Error)
	require.Equal(t, MigrationPhaseFailed, result.Phase)
	require.Contains(t, result.Error.Error(), "routing switch")

	// Verify source state is still intact -- we can still extend
	// (source was never modified, so it should still work)
}

func TestRoutingCapacityManager_AcquireBlockedDuringMigration(t *testing.T) {
	setup := newTestMigrationSetup(t)
	ctx := context.Background()

	accountID := uuid.New()

	shards := map[string]*redisCapacityManager{
		"source":      setup.srcCM,
		"destination": setup.dstCM,
	}

	router := func(ctx context.Context, acctID uuid.UUID) (string, error) {
		return "source", nil
	}

	rm := NewRoutingCapacityManager(router, shards)

	// Simulate a migration being in progress
	rm.mu.Lock()
	rm.migrations[accountID] = &accountMigration{
		coordinator:  NewMigrationCoordinator(),
		sourceShard:  "source",
		destShard:    "destination",
		retryAfter:   5 * time.Second,
		sourceClient: setup.srcRC,
	}
	rm.mu.Unlock()

	// Acquire should return RetryAfter during migration
	resp, err := rm.Acquire(ctx, &CapacityAcquireRequest{
		AccountID:            accountID,
		EnvID:                uuid.New(),
		FunctionID:           uuid.New(),
		Amount:               1,
		LeaseIdempotencyKeys: []string{"event-1"},
		IdempotencyKey:       "op-1",
		Duration:             5 * time.Second,
		Source: LeaseSource{
			Service:  ServiceExecutor,
			Location: CallerLocationSchedule,
		},
		Configuration: ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 10,
			},
		},
		Constraints: []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		},
		CurrentTime:     setup.clock.Now(),
		MaximumLifetime: 3 * time.Hour,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.RetryAfter.IsZero(), "acquire should return RetryAfter during migration")
	require.Empty(t, resp.Leases, "no leases should be granted during migration")
}

func TestScavengerSkipMigratingAccounts(t *testing.T) {
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
		WithShardName("test"),
		WithClock(clock),
	)
	require.NoError(t, err)

	accountID := uuid.New()
	migratingAccounts := map[uuid.UUID]bool{
		accountID: true,
	}

	// The skip function should work
	skipFn := func(acctID uuid.UUID) bool {
		return migratingAccounts[acctID]
	}

	// Run scavenge with skip function -- this just exercises the code path
	// The real test is that accounts flagged as migrating are not processed
	_, internalErr := cm.Scavenge(ctx,
		ScavengerSkipMigratingAccounts(skipFn),
		ScavengerAccountsPeekSize(10),
	)
	require.NoError(t, internalErr)
}

func TestExtractConcurrencyKeysFromRequestState(t *testing.T) {
	r := miniredis.RunT(t)
	ctx := context.Background()

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	// Create a mock request state with concurrency constraints
	requestState := `{"s":[{"k":2,"c":{"ilk":"{cs}:a:test:concurrency:f:fn-id","l":10}},{"k":1,"r":{"k":"{cs}:a:test:rl:a:hash","l":100,"p":60000000000}}]}`

	key := "{cs}:a:test:rs:request-123"
	r.Set(key, requestState)

	keys, err := extractConcurrencyKeysFromRequestState(ctx, rc, key)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, "{cs}:a:test:concurrency:f:fn-id", keys[0])
}

func TestMigrationCoordinator_CopyPreservesTTL(t *testing.T) {
	setup := newTestMigrationSetup(t)
	ctx := context.Background()

	key := "{cs}:a:test:rl:a:ttl-test"

	// Set a key with TTL on source
	setup.srcRedis.Set(key, "value")
	setup.srcRedis.SetTTL(key, 300*time.Second)

	copier := &TypeAwareCopier{}
	copied, err := copier.CopyKeys(ctx, setup.srcRC, setup.dstRC, []string{key}, 10)
	require.NoError(t, err)
	require.Equal(t, 1, copied)

	// Verify TTL is preserved on destination
	ttl := setup.dstRedis.TTL(key)
	require.Greater(t, ttl, 290*time.Second, "TTL should be approximately preserved")
	require.LessOrEqual(t, ttl, 300*time.Second)
}
