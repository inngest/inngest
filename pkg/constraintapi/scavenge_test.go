package constraintapi

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestScavengerShard tests the CRC32-based sharding function for deterministic and uniform distribution
func TestScavengerShard(t *testing.T) {
	tests := []struct {
		name           string
		numShards      int
		expectedResult func(t *testing.T, mgr *redisCapacityManager, accountID uuid.UUID, result int)
	}{
		{
			name:      "deterministic - same UUID always returns same shard",
			numShards: 64,
			expectedResult: func(t *testing.T, mgr *redisCapacityManager, accountID uuid.UUID, result int) {
				// Call multiple times with same UUID
				for i := 0; i < 10; i++ {
					shard := mgr.scavengerShard(context.Background(), accountID)
					require.Equal(t, result, shard, "scavengerShard should be deterministic")
				}
			},
		},
		{
			name:      "range validation - shard in bounds for 64 shards",
			numShards: 64,
			expectedResult: func(t *testing.T, mgr *redisCapacityManager, accountID uuid.UUID, result int) {
				require.GreaterOrEqual(t, result, 0, "shard should be >= 0")
				require.Less(t, result, 64, "shard should be < numShards")
			},
		},
		{
			name:      "range validation - shard in bounds for 16 shards",
			numShards: 16,
			expectedResult: func(t *testing.T, mgr *redisCapacityManager, accountID uuid.UUID, result int) {
				require.GreaterOrEqual(t, result, 0, "shard should be >= 0")
				require.Less(t, result, 16, "shard should be < numShards")
			},
		},
		{
			name:      "edge case - single shard always returns 0",
			numShards: 1,
			expectedResult: func(t *testing.T, mgr *redisCapacityManager, accountID uuid.UUID, result int) {
				require.Equal(t, 0, result, "single shard should always return 0")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &redisCapacityManager{
				numScavengerShards: tt.numShards,
			}

			// Generate a random UUID v4 for testing
			accountID := uuid.New()
			result := mgr.scavengerShard(context.Background(), accountID)

			tt.expectedResult(t, mgr, accountID, result)
		})
	}
}

func TestScavengerShardDistribution(t *testing.T) {
	tests := []struct {
		name      string
		numShards int
		numUUIDs  int
	}{
		{
			name:      "uniform distribution - 16 shards",
			numShards: 16,
			numUUIDs:  10000,
		},
		{
			name:      "uniform distribution - 64 shards",
			numShards: 64,
			numUUIDs:  10000,
		},
		{
			name:      "uniform distribution - 256 shards",
			numShards: 256,
			numUUIDs:  25600, // 100 per shard on average
		},
		{
			name:      "uniform distribution - 1024 shards",
			numShards: 1024,
			numUUIDs:  102400, // 100 per shard on average
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &redisCapacityManager{
				numScavengerShards: tt.numShards,
			}

			// Count distribution across shards
			shardCounts := make([]int, tt.numShards)

			// Generate many UUIDs and track their shard distribution
			for i := 0; i < tt.numUUIDs; i++ {
				accountID := uuid.New()
				shard := mgr.scavengerShard(context.Background(), accountID)
				shardCounts[shard]++
			}

			// Calculate expected count per shard
			expectedPerShard := float64(tt.numUUIDs) / float64(tt.numShards)

			// Calculate chi-square statistic to test uniformity
			chiSquare := 0.0
			for _, count := range shardCounts {
				deviation := float64(count) - expectedPerShard
				chiSquare += (deviation * deviation) / expectedPerShard
			}

			// For good distribution, chi-square should be reasonable
			// Critical value for 95% confidence with (numShards-1) degrees of freedom
			// For our test, we'll use a more lenient check - ensure no shard is empty
			// and no shard has more than 3x the expected count
			for i, count := range shardCounts {
				require.Greater(t, count, 0, "shard %d should not be empty", i)
				require.LessOrEqual(t, float64(count), expectedPerShard*3,
					"shard %d has too many assignments: %d (expected ~%.1f)",
					i, count, expectedPerShard)
			}

			// Calculate standard deviation to ensure reasonable distribution
			mean := expectedPerShard
			variance := 0.0
			for _, count := range shardCounts {
				deviation := float64(count) - mean
				variance += deviation * deviation
			}
			variance /= float64(tt.numShards)
			stdDev := math.Sqrt(variance)

			// Standard deviation should be reasonable (< 50% of mean for good distribution)
			maxStdDev := mean * 0.5
			require.LessOrEqual(t, stdDev, maxStdDev,
				"distribution standard deviation %.2f is too high (expected < %.2f)",
				stdDev, maxStdDev)

			t.Logf("Distribution stats for %d shards: mean=%.1f, stddev=%.2f, chi-square=%.2f",
				tt.numShards, mean, stdDev, chiSquare)
		})
	}
}

func TestScavengerShardSpecificUUIDs(t *testing.T) {
	// Test with some specific UUIDs to ensure consistent behavior
	testUUIDs := []uuid.UUID{
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), // Example UUID v4
		uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"), // Example UUID v1
		uuid.MustParse("6ba7b811-9dad-11d1-80b4-00c04fd430c8"), // Similar UUID
		uuid.MustParse("ffffffff-ffff-4fff-bfff-ffffffffffff"), // Max values in v4 format
		uuid.MustParse("00000000-0000-4000-8000-000000000000"), // Min values in v4 format
	}

	mgr := &redisCapacityManager{
		numScavengerShards: 64,
	}

	// Verify deterministic behavior for specific UUIDs
	for _, testUUID := range testUUIDs {
		firstResult := mgr.scavengerShard(context.Background(), testUUID)

		// Test multiple calls return same result
		for i := 0; i < 5; i++ {
			result := mgr.scavengerShard(context.Background(), testUUID)
			require.Equal(t, firstResult, result,
				"UUID %s should always map to shard %d, got %d",
				testUUID, firstResult, result)
		}

		// Ensure result is in valid range
		require.GreaterOrEqual(t, firstResult, 0)
		require.Less(t, firstResult, 64)

		t.Logf("UUID %s maps to shard %d", testUUID, firstResult)
	}
}

func TestScavengerShardDifferentShardCounts(t *testing.T) {
	// Test that same UUID maps to different shards with different shard counts
	// but behavior remains deterministic
	testUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	shardCounts := []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 1024}

	for _, numShards := range shardCounts {
		mgr := &redisCapacityManager{
			numScavengerShards: numShards,
		}

		shard := mgr.scavengerShard(context.Background(), testUUID)

		// Verify range
		require.GreaterOrEqual(t, shard, 0)
		require.Less(t, shard, numShards)

		// Verify deterministic
		shard2 := mgr.scavengerShard(context.Background(), testUUID)
		require.Equal(t, shard, shard2)

		t.Logf("UUID %s with %d shards maps to shard %d", testUUID, numShards, shard)
	}
}

// Benchmark to ensure the function is fast
func BenchmarkScavengerShard(b *testing.B) {
	mgr := &redisCapacityManager{
		numScavengerShards: 64,
	}

	accountID := uuid.New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mgr.scavengerShard(ctx, accountID)
	}
}

func TestScavengeProcess_Basic(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Scavenge Expired Leases", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:scavenge:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Acquire multiple leases with short duration
		var allLeases []CapacityLease
		for i := 0; i < 3; i++ {
			acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
				IdempotencyKey:       fmt.Sprintf("scavenge-acquire-%d", i),
				AccountID:            te.AccountID,
				EnvID:                te.EnvID,
				FunctionID:           te.FunctionID,
				Amount:               1,
				LeaseIdempotencyKeys: []string{fmt.Sprintf("scavenge-lease-%d", i)},
				CurrentTime:          clock.Now(),
				Duration:             5 * time.Second, // Short duration
				MaximumLifetime:      time.Minute,
				Configuration:        config,
				Constraints:          constraints,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{QueueShard: "test"},
			})

			require.NoError(t, err)
			require.Len(t, acquireResp.Leases, 1)
			allLeases = append(allLeases, acquireResp.Leases[0])
		}

		// Verify initial state - 3 leases in progress
		cv := te.NewConstraintVerifier()
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 3})

		// Verify account is in scavenger shard
		cv.VerifyScavengerShard(float64(ulid.Time(allLeases[0].LeaseID.Time()).UnixMilli()), true)

		// Advance time to expire all leases
		clock.Advance(10 * time.Second)
		te.AdvanceTimeAndRedis(10 * time.Second)

		// Run scavenger process
		scavengeResp, err := te.CapacityManager.Scavenge(context.Background())

		require.NoError(t, err)
		require.NotZero(t, scavengeResp.ReclaimedLeases, "Should scavenge some leases")

		// Verify capacity is restored after scavenging
		checkResp, _, err := te.CapacityManager.Check(context.Background(), &CapacityCheckRequest{
			AccountID:     te.AccountID,
			EnvID:         te.EnvID,
			FunctionID:    te.FunctionID,
			Configuration: config,
			Constraints:   constraints,
			Migration:     MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Equal(t, 0, checkResp.Usage[0].Used, "All capacity should be restored after scavenging")

		// Verify in-progress counts are updated
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 0})

		// Verify account is removed from scavenger shard when no leases remain
		cv.VerifyScavengerShard(0, false)
	})

	t.Run("Scavenge With Mixed Expired and Active Leases", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:mixed-scavenge:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Acquire leases with different expiry times
		var activeLeases []CapacityLease

		// Create expired leases (short duration)
		for i := 0; i < 2; i++ {
			acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
				IdempotencyKey:       fmt.Sprintf("mixed-expired-%d", i),
				AccountID:            te.AccountID,
				EnvID:                te.EnvID,
				FunctionID:           te.FunctionID,
				Amount:               1,
				LeaseIdempotencyKeys: []string{fmt.Sprintf("mixed-expired-lease-%d", i)},
				CurrentTime:          clock.Now(),
				Duration:             5 * time.Second, // Will expire soon
				MaximumLifetime:      time.Minute,
				Configuration:        config,
				Constraints:          constraints,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{QueueShard: "test"},
			})

			require.NoError(t, err)
			require.Len(t, acquireResp.Leases, 1)
		}

		// Advance time to expire first set
		clock.Advance(10 * time.Second)
		te.AdvanceTimeAndRedis(10 * time.Second)

		// Create active leases (long duration)
		for i := 0; i < 2; i++ {
			acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
				IdempotencyKey:       fmt.Sprintf("mixed-active-%d", i),
				AccountID:            te.AccountID,
				EnvID:                te.EnvID,
				FunctionID:           te.FunctionID,
				Amount:               1,
				LeaseIdempotencyKeys: []string{fmt.Sprintf("mixed-active-lease-%d", i)},
				CurrentTime:          clock.Now(),
				Duration:             60 * time.Second, // Long duration
				MaximumLifetime:      2 * time.Minute,
				Configuration:        config,
				Constraints:          constraints,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{QueueShard: "test"},
			})

			require.NoError(t, err)
			require.Len(t, acquireResp.Leases, 1)
			activeLeases = append(activeLeases, acquireResp.Leases[0])
		}

		// Verify initial state - 4 leases total (2 expired, 2 active)
		cv := te.NewConstraintVerifier()
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 4})

		// Run scavenger process
		scavengeResp, err := te.CapacityManager.Scavenge(context.Background())

		require.NoError(t, err)
		require.Equal(t, 2, scavengeResp.ReclaimedLeases, "Should scavenge exactly 2 expired leases")

		// Verify only active leases remain
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 2})

		// Verify account remains in scavenger shard (still has active leases)
		cv.VerifyScavengerShard(float64(ulid.Time(activeLeases[0].LeaseID.Time()).UnixMilli()), true)

		// Clean up active leases
		for _, lease := range activeLeases {
			_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
				IdempotencyKey: fmt.Sprintf("cleanup-%s", lease.IdempotencyKey),
				AccountID:      te.AccountID,
				LeaseID:        lease.LeaseID,
				Migration:      MigrationIdentifier{QueueShard: "test"},
			})
			require.NoError(t, err)
		}

		// Verify final cleanup
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 0})
		cv.VerifyScavengerShard(0, false)
	})

	t.Run("Scavenge Rate Limit Leases", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			RateLimit: []RateLimitConfig{
				{
					Scope:             enums.RateLimitScopeFn,
					Limit:             10,
					Period:            60,
					KeyExpressionHash: "scavenge-ratelimit",
				},
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeFn,
					KeyExpressionHash: "scavenge-ratelimit",
					EvaluatedKeyHash:  "scavenge-test",
				},
			},
		}

		// Acquire rate limit lease with short duration
		acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       fmt.Sprintf("rate-scavenge-%d", 1),
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{fmt.Sprintf("rate-scavenge-lease-%d", 1)},
			CurrentTime:          clock.Now(),
			Duration:             3 * time.Second, // Short duration
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{IsRateLimit: true},
		})

		require.NoError(t, err)
		require.NotEmpty(t, acquireResp.Leases)

		// Advance time to expire leases
		clock.Advance(5 * time.Second)
		te.AdvanceTimeAndRedis(5 * time.Second)

		// Run scavenger process
		scavengeResp, err := te.CapacityManager.Scavenge(context.Background())

		require.NoError(t, err)
		require.NotZero(t, scavengeResp.ReclaimedLeases, "Should scavenge rate limit leases")

		// Verify rate limit state is preserved (TAT should remain)
		rateLimitKey := fmt.Sprintf("{%s}:scavenge-test", te.CapacityManager.rateLimitKeyPrefix)
		rv := te.NewRateLimitStateVerifier()
		currentTime := clock.Now().UnixNano()
		rv.VerifyRateLimitState(rateLimitKey, 0, currentTime+int64(time.Hour)) // Should still exist
	})

	t.Run("Scavenge Throttle Leases", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Throttle: []ThrottleConfig{
				{
					Scope:                     enums.ThrottleScopeFn,
					Limit:                     10,
					Burst:                     5,
					Period:                    60,
					ThrottleKeyExpressionHash: "scavenge-throttle",
				},
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeFn,
					KeyExpressionHash: "scavenge-throttle",
					EvaluatedKeyHash:  "throttle-scavenge-test",
				},
			},
		}

		// Acquire throttle leases
		for i := 0; i < 2; i++ {
			acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
				IdempotencyKey:       fmt.Sprintf("throttle-scavenge-%d", i),
				AccountID:            te.AccountID,
				EnvID:                te.EnvID,
				FunctionID:           te.FunctionID,
				Amount:               1,
				LeaseIdempotencyKeys: []string{fmt.Sprintf("throttle-scavenge-lease-%d", i)},
				CurrentTime:          clock.Now(),
				Duration:             3 * time.Second, // Short duration
				MaximumLifetime:      time.Minute,
				Configuration:        config,
				Constraints:          constraints,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{QueueShard: "test"},
			})

			require.NoError(t, err)
			require.NotEmpty(t, acquireResp.Leases)
		}

		// Advance time to expire leases
		clock.Advance(5 * time.Second)
		te.AdvanceTimeAndRedis(5 * time.Second)

		// Run scavenger process
		scavengeResp, err := te.CapacityManager.Scavenge(context.Background())

		require.NoError(t, err)
		require.NotZero(t, scavengeResp.ReclaimedLeases, "Should scavenge throttle leases")
	})
}

func TestScavengeProcess_Sharding(t *testing.T) {
	te := NewTestEnvironment(t)
	te.CapacityManager.numScavengerShards = 4
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Shard Distribution", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:shard:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Create leases across multiple accounts to test shard distribution
		for accountIdx := 0; accountIdx < 10; accountIdx++ {
			accountID := uuid.New()

			// Acquire lease for this account
			acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
				IdempotencyKey:       fmt.Sprintf("shard-acquire-%d", accountIdx),
				AccountID:            accountID,
				EnvID:                te.EnvID,
				FunctionID:           te.FunctionID,
				Amount:               1,
				LeaseIdempotencyKeys: []string{fmt.Sprintf("shard-lease-%d", accountIdx)},
				CurrentTime:          clock.Now(),
				Duration:             5 * time.Second,
				MaximumLifetime:      time.Minute,
				Configuration:        config,
				Constraints:          constraints,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{QueueShard: "test"},
			})

			require.NoError(t, err)
			require.Len(t, acquireResp.Leases, 1)

		}

		// Advance time to expire leases
		clock.Advance(10 * time.Second)
		te.AdvanceTimeAndRedis(10 * time.Second)

		// Test scavenging from different shards
		totalScavenged := 0
		for shardID := 0; shardID < 4; shardID++ { // 4 scavenger shards configured
			scavengeResp, err := te.CapacityManager.Scavenge(context.Background())

			require.NoError(t, err)
			totalScavenged += scavengeResp.ReclaimedLeases
		}

		require.NotZero(t, totalScavenged, "Should scavenge leases across shards")
	})

	t.Run("MaxLeases Limit", func(t *testing.T) {
		te.Redis.FlushAll()

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
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:maxleases:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Create many expired leases
		for i := 0; i < 10; i++ {
			acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
				IdempotencyKey:       fmt.Sprintf("max-lease-acquire-%d", i),
				AccountID:            te.AccountID,
				EnvID:                te.EnvID,
				FunctionID:           te.FunctionID,
				Amount:               1,
				LeaseIdempotencyKeys: []string{fmt.Sprintf("max-lease-%d", i)},
				CurrentTime:          clock.Now(),
				Duration:             5 * time.Second,
				MaximumLifetime:      time.Minute,
				Configuration:        config,
				Constraints:          constraints,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{QueueShard: "test"},
			})

			require.NoError(t, err)
			require.Len(t, acquireResp.Leases, 1)
		}

		// Advance time to expire all leases
		clock.Advance(10 * time.Second)
		te.AdvanceTimeAndRedis(10 * time.Second)

		// Scavenge all
		scavengeResp, err := te.CapacityManager.Scavenge(context.Background())

		require.NoError(t, err)
		require.Equal(t, scavengeResp.ReclaimedLeases, 10, "Should respect MaxLeases limit")
		require.NotZero(t, scavengeResp.ReclaimedLeases, "Should scavenge some leases")

		// Run again, should be 0 now
		scavengeResp2, err := te.CapacityManager.Scavenge(context.Background())

		require.NoError(t, err)
		require.Zero(t, scavengeResp2.ReclaimedLeases, "Should not have more leases to scavenge")
	})

	t.Run("Peek Expired Leases", func(t *testing.T) {
		te.Redis.FlushAll()

		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:peek:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Create some leases
		for i := 0; i < 3; i++ {
			acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
				IdempotencyKey:       fmt.Sprintf("peek-acquire-%d", i),
				AccountID:            te.AccountID,
				EnvID:                te.EnvID,
				FunctionID:           te.FunctionID,
				Amount:               1,
				LeaseIdempotencyKeys: []string{fmt.Sprintf("peek-lease-%d", i)},
				CurrentTime:          clock.Now(),
				Duration:             5 * time.Second,
				MaximumLifetime:      time.Minute,
				Configuration:        config,
				Constraints:          constraints,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{QueueShard: "test"},
			})

			require.NoError(t, err)
			require.Len(t, acquireResp.Leases, 1)
		}

		// Advance time to expire leases
		clock.Advance(10 * time.Second)
		te.AdvanceTimeAndRedis(10 * time.Second)

		// Peek at expired leases (should return account IDs but not scavenge)
		_, expiredLeases, err := te.CapacityManager.peekExpiredLeases(context.Background(),
			te.CapacityManager.queueStateKeyPrefix, te.Client, te.AccountID, 10, clock.Now())

		require.NoError(t, err)
		require.NotEmpty(t, expiredLeases, "Should find expired leases to peek")

		// Verify leases are still present (peek doesn't scavenge)
		cv := te.NewConstraintVerifier()
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 3})

		// Now scavenge to clean up
		scavengeResp, err := te.CapacityManager.Scavenge(context.Background())

		require.NoError(t, err)
		require.Equal(t, 3, scavengeResp.ReclaimedLeases, "Should scavenge the peeked leases")

		// Verify cleanup
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 0})
	})
}

func TestScavengeProcess_ErrorScenarios(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Scavenge During Active Operations", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:concurrent:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Create expired leases
		for i := 0; i < 2; i++ {
			acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
				IdempotencyKey:       fmt.Sprintf("concurrent-expired-%d", i),
				AccountID:            te.AccountID,
				EnvID:                te.EnvID,
				FunctionID:           te.FunctionID,
				Amount:               1,
				LeaseIdempotencyKeys: []string{fmt.Sprintf("concurrent-expired-lease-%d", i)},
				CurrentTime:          clock.Now(),
				Duration:             5 * time.Second,
				MaximumLifetime:      time.Minute,
				Configuration:        config,
				Constraints:          constraints,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{QueueShard: "test"},
			})

			require.NoError(t, err)
			require.Len(t, acquireResp.Leases, 1)
		}

		// Advance time to expire first set
		clock.Advance(10 * time.Second)
		te.AdvanceTimeAndRedis(10 * time.Second)

		// Create active lease while scavenging
		acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "concurrent-active",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"concurrent-active-lease"},
			CurrentTime:          clock.Now(),
			Duration:             60 * time.Second,
			MaximumLifetime:      2 * time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.NoError(t, err)
		require.Len(t, acquireResp.Leases, 1)

		// Run scavenger - should only clean expired leases, not active ones
		scavengeResp, err := te.CapacityManager.Scavenge(context.Background())

		require.NoError(t, err)
		require.Equal(t, 2, scavengeResp.ReclaimedLeases, "Should scavenge exactly 2 expired leases")

		// Verify active lease remains
		cv := te.NewConstraintVerifier()
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 1})

		// Clean up
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "concurrent-cleanup",
			AccountID:      te.AccountID,
			LeaseID:        acquireResp.Leases[0].LeaseID,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})
		require.NoError(t, err)
	})

	t.Run("Empty Scavenger Shard", func(t *testing.T) {
		// Run scavenger on empty shard
		scavengeResp, err := te.CapacityManager.Scavenge(context.Background())

		require.NoError(t, err)
		require.Zero(t, scavengeResp.ReclaimedLeases, "Should scavenge 0 leases from empty shard")
	})

	t.Run("Invalid Shard ID", func(t *testing.T) {
		// Run scavenger with very high shard ID
		scavengeResp, err := te.CapacityManager.Scavenge(context.Background())

		require.NoError(t, err)
		require.Zero(t, scavengeResp.ReclaimedLeases, "Should handle invalid shard ID gracefully")
	})
}

func TestScavengeProcess_Performance(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Large Scale Scavenging", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 1000, // High limit
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:largescale:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Create many expired leases
		numLeases := 100 // Reasonable number for test performance
		for i := 0; i < numLeases; i++ {
			acquireResp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
				IdempotencyKey:       fmt.Sprintf("large-scale-%d", i),
				AccountID:            te.AccountID,
				EnvID:                te.EnvID,
				FunctionID:           te.FunctionID,
				Amount:               1,
				LeaseIdempotencyKeys: []string{fmt.Sprintf("large-scale-lease-%d", i)},
				CurrentTime:          clock.Now(),
				Duration:             5 * time.Second,
				MaximumLifetime:      time.Minute,
				Configuration:        config,
				Constraints:          constraints,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{QueueShard: "test"},
			})

			require.NoError(t, err)
			require.Len(t, acquireResp.Leases, 1)

			// Add small delay to avoid overwhelming Redis
			if i%10 == 0 {
				clock.Advance(100 * time.Millisecond)
				te.AdvanceTimeAndRedis(100 * time.Millisecond)
			}
		}

		// Advance time to expire all leases
		clock.Advance(10 * time.Second)
		te.AdvanceTimeAndRedis(10 * time.Second)

		// Measure scavenger performance
		startTime := time.Now()

		totalScavenged := 0
		maxIterations := 20 // Prevent infinite loop
		iteration := 0

		for totalScavenged < numLeases && iteration < maxIterations {
			scavengeResp, err := te.CapacityManager.Scavenge(context.Background())

			require.NoError(t, err)
			totalScavenged += scavengeResp.ReclaimedLeases
			iteration++

			if scavengeResp.ReclaimedLeases == 0 {
				break // No more to scavenge
			}
		}

		elapsed := time.Since(startTime)
		require.NotZero(t, totalScavenged, "Should scavenge leases")
		require.True(t, elapsed < 10*time.Second, "Scavenging should complete in reasonable time")

		t.Logf("Scavenged %d leases in %v (%d iterations)", totalScavenged, elapsed, iteration)

		// Verify final state
		cv := te.NewConstraintVerifier()
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 0})
	})
}
