package constraintapi

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestInputValidation_ResourceLimits(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Maximum Lease Count", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 10000, // Very high limit
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:maxcount:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Create lease idempotency keys for maximum amount
		maxAmount := 1000 // Large but reasonable for testing
		leaseKeys := make([]string, maxAmount)
		for i := 0; i < maxAmount; i++ {
			leaseKeys[i] = fmt.Sprintf("max-lease-%d", i)
		}

		// Test acquiring maximum amount
		_, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "max-count-test",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               maxAmount,
			LeaseIdempotencyKeys: leaseKeys,
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.Error(t, err)
		require.ErrorContains(t, err, "must request no more than 20 leases")
	})

	t.Run("Extreme Duration Values", func(t *testing.T) {
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
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:duration:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Test very small duration
		_, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "small-duration",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"small-duration-lease"},
			CurrentTime:          clock.Now(),
			Duration:             time.Nanosecond, // Extremely small
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.Error(t, err)
		require.ErrorContains(t, err, "duration smaller than minimum of 2s")

		// Test very large duration (but within lifetime)
		_, err = te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "large-duration",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"large-duration-lease"},
			CurrentTime:          clock.Now(),
			Duration:             23 * time.Hour, // Very large
			MaximumLifetime:      24 * time.Hour, // Larger lifetime
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.Error(t, err)
		require.ErrorContains(t, err, "duration exceeds max value of 1m0s")
	})

	t.Run("Complex Constraint Configurations", func(t *testing.T) {
		// Test maximum complexity constraint configuration
		config := ConstraintConfig{
			FunctionVersion: 999999,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 1000000, // Very high
			},
			RateLimit: make([]RateLimitConfig, 100), // Many rate limits
			Throttle:  make([]ThrottleConfig, 100),  // Many throttles
		}

		// Fill with complex configurations
		for i := 0; i < 100; i++ {
			config.RateLimit[i] = RateLimitConfig{
				Scope:             enums.RateLimitScopeFn,
				Limit:             1000000,
				Period:            3600, // 1 hour
				KeyExpressionHash: fmt.Sprintf("complex-rate-limit-%d", i),
			}
			config.Throttle[i] = ThrottleConfig{
				Scope:                     enums.ThrottleScopeFn,
				Limit:                     1000000,
				Burst:                     100000,
				Period:                    3600,
				ThrottleKeyExpressionHash: fmt.Sprintf("complex-throttle-%d", i),
			}
		}

		// Single constraint for testing
		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:complex:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Test with complex configuration
		_, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "complex-config",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"complex-lease"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "exceeded maximum of 1 rate limits")
		require.ErrorContains(t, err, "exceeded maximum of 1 throttles")
	})

	t.Run("Large Idempotency Keys and Hashes", func(t *testing.T) {
		// Create very long strings
		longIdempotencyKey := strings.Repeat("a", 1000)
		longLeaseKey := strings.Repeat("b", 1000)
		longInProgressKey := fmt.Sprintf("{%s}:concurrency:%s:%s", te.KeyPrefix, strings.Repeat("d", 500), te.FunctionID)

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
					InProgressItemKey: longInProgressKey,
				},
			},
		}

		_, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       longIdempotencyKey,
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{longLeaseKey},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "idempotency key longer than 128 chars")
		require.ErrorContains(t, err, "idempotency key 0 longer than 128 chars")
	})
}

func TestInputValidation_MalformedData(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Invalid UUID Formats", func(t *testing.T) {
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
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:invalid-uuid:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Test with zero UUIDs (should be caught by validation)
		_, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "invalid-uuid-test",
			AccountID:            uuid.Nil, // Invalid
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"invalid-uuid-lease"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.Error(t, err, "Should reject nil UUID")
		require.Contains(t, err.Error(), "missing accountID")

		// Test with nil env ID
		_, err = te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "invalid-uuid-test-2",
			AccountID:            te.AccountID,
			EnvID:                uuid.Nil, // Invalid
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"invalid-uuid-lease-2"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.Error(t, err, "Should reject nil EnvID")
		require.Contains(t, err.Error(), "missing envID")
	})

	t.Run("Invalid ULID Formats", func(t *testing.T) {
		// Test extend with invalid ULID
		_, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
			IdempotencyKey: "invalid-ulid-test",
			AccountID:      te.AccountID,
			LeaseID:        ulid.ULID{}, // Zero ULID
			Duration:       30 * time.Second,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.Error(t, err, "Should reject zero ULID")

		// Test release with invalid ULID
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "invalid-ulid-release",
			AccountID:      te.AccountID,
			LeaseID:        ulid.ULID{}, // Zero ULID
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})

		require.Error(t, err, "Should reject zero ULID")
	})

	t.Run("Out of Bounds Enum Values", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		// Test with invalid concurrency mode
		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              999, // Invalid mode
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:invalid-enum:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Should be handled gracefully (may not error but should not break)
		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "invalid-enum-test",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"invalid-enum-lease"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		// Implementation may handle this gracefully or return error
		if err == nil {
			require.NotNil(t, resp, "Should handle invalid enum gracefully")
			// Clean up if successful
			for _, lease := range resp.Leases {
				_, _ = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
					IdempotencyKey: "cleanup-enum",
					AccountID:      te.AccountID,
					LeaseID:        lease.LeaseID,
					Migration:      MigrationIdentifier{QueueShard: "test"},
				})
			}
		}
	})

	t.Run("Missing Required Fields", func(t *testing.T) {
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
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:missing:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Test missing idempotency key
		_, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "", // Empty
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"missing-field-lease"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.Error(t, err, "Should reject empty idempotency key")

		// Test missing lease idempotency keys
		_, err = te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "missing-lease-keys",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{}, // Empty
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.Error(t, err, "Should reject empty lease idempotency keys")

		// Test missing constraints
		_, err = te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "missing-constraints",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"missing-constraints-lease"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          []ConstraintItem{}, // Empty
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.Error(t, err, "Should reject empty constraints")
	})

	t.Run("Zero and Negative Values", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 0, // Zero version
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
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:zero-values:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Test zero version
		_, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "zero-version",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"zero-version-lease"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.Error(t, err, "Should reject zero function version")

		// Test negative duration
		_, err = te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "negative-duration",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"negative-duration-lease"},
			CurrentTime:          clock.Now(),
			Duration:             -30 * time.Second, // Negative
			MaximumLifetime:      time.Minute,
			Configuration:        ConstraintConfig{FunctionVersion: 1, Concurrency: ConcurrencyConfig{FunctionConcurrency: 5}},
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.Error(t, err, "Should reject negative duration")

		// Test zero amount
		_, err = te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "zero-amount",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               0, // Zero amount
			LeaseIdempotencyKeys: []string{"zero-amount-lease"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        ConstraintConfig{FunctionVersion: 1, Concurrency: ConcurrencyConfig{FunctionConcurrency: 5}},
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})
		require.Error(t, err, "Should reject zero amount")
	})
}

func TestInputValidation_BoundaryConditions(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Maximum Integer Values", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 2147483647, // Max int32
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 2147483647, // Max int32
			},
			RateLimit: []RateLimitConfig{
				{
					Scope:             enums.RateLimitScopeFn,
					Limit:             2147483647, // Max int32
					Period:            2147483647, // Max int32
					KeyExpressionHash: "max-int-test",
				},
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:maxint:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "max-int-test",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               2147483647, // Max int32
			LeaseIdempotencyKeys: []string{"max-int-lease"},
			CurrentTime:          clock.Now(),
			Duration:             time.Duration(9223372036854775807), // Max int64 nanoseconds
			MaximumLifetime:      time.Duration(9223372036854775807), // Max int64 nanoseconds
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		// Should handle large values gracefully (may cap or reject)
		if err == nil {
			require.NotNil(t, resp)
			// Clean up if successful
			for _, lease := range resp.Leases {
				_, _ = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
					IdempotencyKey: "cleanup-maxint",
					AccountID:      te.AccountID,
					LeaseID:        lease.LeaseID,
					Migration:      MigrationIdentifier{QueueShard: "test"},
				})
			}
		}
	})

	t.Run("Empty Collections and Arrays", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
			RateLimit: []RateLimitConfig{}, // Empty
			Throttle:  []ThrottleConfig{},  // Empty
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:empty:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "empty-collections",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"empty-collections-lease"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
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
		require.Len(t, resp.Leases, 1, "Should handle empty collections gracefully")

		// Clean up
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "cleanup-empty",
			AccountID:      te.AccountID,
			LeaseID:        resp.Leases[0].LeaseID,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})
		require.NoError(t, err)
	})

	t.Run("Unicode and Special Characters", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		// Use unicode characters in keys
		unicodeKey := "🔥测试-key-αβγ-🚀"
		unicodeInProgressKey := fmt.Sprintf("{%s}:concurrency:unicode:%s:🌟", te.KeyPrefix, te.FunctionID)

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: unicodeInProgressKey,
				},
			},
		}

		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       unicodeKey,
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"unicode-lease-🎯"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
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
		require.Len(t, resp.Leases, 1, "Should handle unicode characters gracefully")

		// Clean up
		_, err = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
			IdempotencyKey: "cleanup-unicode",
			AccountID:      te.AccountID,
			LeaseID:        resp.Leases[0].LeaseID,
			Migration:      MigrationIdentifier{QueueShard: "test"},
		})
		require.NoError(t, err)
	})

	t.Run("Time Edge Cases", func(t *testing.T) {
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
					InProgressItemKey: fmt.Sprintf("{%s}:concurrency:time:%s", te.KeyPrefix, te.FunctionID),
				},
			},
		}

		// Test with zero time
		_, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "zero-time",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"zero-time-lease"},
			CurrentTime:          time.Time{}, // Zero time
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		require.Error(t, err, "Should reject zero current time")

		// Test with time far in the past
		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "past-time",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"past-time-lease"},
			CurrentTime:          time.Unix(0, 0), // Epoch
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		// May be rejected or handled gracefully depending on implementation
		if err == nil {
			require.NotNil(t, resp)
			// Clean up if successful
			for _, lease := range resp.Leases {
				_, _ = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
					IdempotencyKey: "cleanup-past",
					AccountID:      te.AccountID,
					LeaseID:        lease.LeaseID,
					Migration:      MigrationIdentifier{QueueShard: "test"},
				})
			}
		}

		// Test with time far in the future
		resp2, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "future-time",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"future-time-lease"},
			CurrentTime:          time.Now().Add(100 * 365 * 24 * time.Hour), // 100 years in future
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: LeaseLocationItemLease,
			},
			Migration: MigrationIdentifier{QueueShard: "test"},
		})

		// May be rejected due to clock skew validation
		if err == nil {
			require.NotNil(t, resp2)
			// Clean up if successful
			for _, lease := range resp2.Leases {
				_, _ = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
					IdempotencyKey: "cleanup-future",
					AccountID:      te.AccountID,
					LeaseID:        lease.LeaseID,
					Migration:      MigrationIdentifier{QueueShard: "test"},
				})
			}
		}
	})
}
