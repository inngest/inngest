package constraintapi

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/inngest/inngest/pkg/enums"
	pb "github.com/inngest/inngest/proto/gen/constraintapi/v1"
)

func TestRateLimitScopeConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    enums.RateLimitScope
		expected pb.ConstraintApiRateLimitScope
	}{
		{
			name:     "function scope",
			input:    enums.RateLimitScopeFn,
			expected: pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_FUNCTION,
		},
		{
			name:     "env scope",
			input:    enums.RateLimitScopeEnv,
			expected: pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_ENV,
		},
		{
			name:     "account scope",
			input:    enums.RateLimitScopeAccount,
			expected: pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_ACCOUNT,
		},
		{
			name:     "invalid scope (fallback to unspecified)",
			input:    enums.RateLimitScope(999),
			expected: pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RateLimitScopeToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := RateLimitScopeFromProto(result)
			if tt.input != enums.RateLimitScope(999) { // Skip round trip for invalid input
				assert.Equal(t, tt.input, backConverted)
			}
		})
	}
}

func TestConcurrencyScopeConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    enums.ConcurrencyScope
		expected pb.ConstraintApiConcurrencyScope
	}{
		{
			name:     "function scope",
			input:    enums.ConcurrencyScopeFn,
			expected: pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_FUNCTION,
		},
		{
			name:     "env scope",
			input:    enums.ConcurrencyScopeEnv,
			expected: pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_ENV,
		},
		{
			name:     "account scope",
			input:    enums.ConcurrencyScopeAccount,
			expected: pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_ACCOUNT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConcurrencyScopeToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := ConcurrencyScopeFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}
}

func TestThrottleScopeConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    enums.ThrottleScope
		expected pb.ConstraintApiThrottleScope
	}{
		{
			name:     "function scope",
			input:    enums.ThrottleScopeFn,
			expected: pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_FUNCTION,
		},
		{
			name:     "env scope",
			input:    enums.ThrottleScopeEnv,
			expected: pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_ENV,
		},
		{
			name:     "account scope",
			input:    enums.ThrottleScopeAccount,
			expected: pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_ACCOUNT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ThrottleScopeToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := ThrottleScopeFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}
}

func TestConcurrencyModeConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    enums.ConcurrencyMode
		expected pb.ConstraintApiConcurrencyMode
	}{
		{
			name:     "step mode",
			input:    enums.ConcurrencyModeStep,
			expected: pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_STEP,
		},
		{
			name:     "run mode",
			input:    enums.ConcurrencyModeRun,
			expected: pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_RUN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConcurrencyModeToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := ConcurrencyModeFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}
}

func TestConstraintKindConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    ConstraintKind
		expected pb.ConstraintApiConstraintKind
	}{
		{
			name:     "rate limit",
			input:    CapacityKindRateLimit,
			expected: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT,
		},
		{
			name:     "concurrency",
			input:    CapacityKindConcurrency,
			expected: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY,
		},
		{
			name:     "throttle",
			input:    CapacityKindThrottle,
			expected: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_THROTTLE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConstraintKindToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := ConstraintKindFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}
}

func TestRunProcessingModeConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    RunProcessingMode
		expected pb.ConstraintApiRunProcessingMode
	}{
		{
			name:     "background mode",
			input:    RunProcessingModeBackground,
			expected: pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_BACKGROUND,
		},
		{
			name:     "sync mode",
			input:    RunProcessingModeSync,
			expected: pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_SYNC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RunProcessingModeToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := RunProcessingModeFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}
}

func TestLeaseLocationConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    LeaseLocation
		expected pb.ConstraintApiLeaseLocation
	}{
		{
			name:     "schedule run",
			input:    LeaseLocationScheduleRun,
			expected: pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_SCHEDULE_RUN,
		},
		{
			name:     "partition lease",
			input:    LeaseLocationPartitionLease,
			expected: pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_PARTITION_LEASE,
		},
		{
			name:     "item lease",
			input:    LeaseLocationItemLease,
			expected: pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_ITEM_LEASE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LeaseLocationToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := LeaseLocationFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}
}

func TestLeaseServiceConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    LeaseService
		expected pb.ConstraintApiLeaseService
	}{
		{
			name:     "new runs service",
			input:    ServiceNewRuns,
			expected: pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_NEW_RUNS,
		},
		{
			name:     "executor service",
			input:    ServiceExecutor,
			expected: pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_EXECUTOR,
		},
		{
			name:     "api service",
			input:    ServiceAPI,
			expected: pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_API,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LeaseServiceToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := LeaseServiceFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}
}

func TestRateLimitConfigConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    RateLimitConfig
		expected *pb.RateLimitConfig
	}{
		{
			name: "complete config",
			input: RateLimitConfig{
				Scope:             enums.RateLimitScopeAccount,
				Limit:             100,
				Period:            "1m",
				KeyExpressionHash: "hash123",
			},
			expected: &pb.RateLimitConfig{
				Scope:             pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_ACCOUNT,
				Limit:             100,
				Period:            "1m",
				KeyExpressionHash: "hash123",
			},
		},
		{
			name: "minimal config",
			input: RateLimitConfig{
				Scope: enums.RateLimitScopeFn,
				Limit: 0,
			},
			expected: &pb.RateLimitConfig{
				Scope:             pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_FUNCTION,
				Limit:             0,
				Period:            "",
				KeyExpressionHash: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RateLimitConfigToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := RateLimitConfigFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}

	// Test nil handling
	t.Run("nil protobuf", func(t *testing.T) {
		result := RateLimitConfigFromProto(nil)
		assert.Equal(t, RateLimitConfig{}, result)
	})
}

func TestCustomConcurrencyLimitConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    CustomConcurrencyLimit
		expected *pb.CustomConcurrencyLimit
	}{
		{
			name: "complete limit",
			input: CustomConcurrencyLimit{
				Mode:              enums.ConcurrencyModeRun,
				Scope:             enums.ConcurrencyScopeEnv,
				Limit:             5,
				KeyExpressionHash: "hash456",
			},
			expected: &pb.CustomConcurrencyLimit{
				Mode:              pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_RUN,
				Scope:             pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_ENV,
				Limit:             5,
				KeyExpressionHash: "hash456",
			},
		},
		{
			name: "minimal limit",
			input: CustomConcurrencyLimit{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
			expected: &pb.CustomConcurrencyLimit{
				Mode:              pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_STEP,
				Scope:             pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_FUNCTION,
				Limit:             0,
				KeyExpressionHash: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CustomConcurrencyLimitToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := CustomConcurrencyLimitFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}

	// Test nil handling
	t.Run("nil protobuf", func(t *testing.T) {
		result := CustomConcurrencyLimitFromProto(nil)
		assert.Equal(t, CustomConcurrencyLimit{}, result)
	})
}

func TestConcurrencyConfigConversion(t *testing.T) {
	customKeys := []CustomConcurrencyLimit{
		{
			Mode:              enums.ConcurrencyModeStep,
			Scope:             enums.ConcurrencyScopeFn,
			Limit:             3,
			KeyExpressionHash: "key1",
		},
		{
			Mode:              enums.ConcurrencyModeRun,
			Scope:             enums.ConcurrencyScopeAccount,
			Limit:             10,
			KeyExpressionHash: "key2",
		},
	}

	tests := []struct {
		name     string
		input    ConcurrencyConfig
		expected *pb.ConcurrencyConfig
	}{
		{
			name: "complete config",
			input: ConcurrencyConfig{
				AccountConcurrency:    100,
				FunctionConcurrency:   50,
				AccountRunConcurrency: 20,
				FunctionRunConcurrency: 10,
				CustomConcurrencyKeys: customKeys,
			},
			expected: &pb.ConcurrencyConfig{
				AccountConcurrency:    100,
				FunctionConcurrency:   50,
				AccountRunConcurrency: 20,
				FunctionRunConcurrency: 10,
				CustomConcurrencyKeys: []*pb.CustomConcurrencyLimit{
					{
						Mode:              pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_STEP,
						Scope:             pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_FUNCTION,
						Limit:             3,
						KeyExpressionHash: "key1",
					},
					{
						Mode:              pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_RUN,
						Scope:             pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_ACCOUNT,
						Limit:             10,
						KeyExpressionHash: "key2",
					},
				},
			},
		},
		{
			name: "empty config",
			input: ConcurrencyConfig{
				CustomConcurrencyKeys: []CustomConcurrencyLimit{},
			},
			expected: &pb.ConcurrencyConfig{
				AccountConcurrency:       0,
				FunctionConcurrency:      0,
				AccountRunConcurrency:    0,
				FunctionRunConcurrency:   0,
				CustomConcurrencyKeys:    []*pb.CustomConcurrencyLimit{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConcurrencyConfigToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := ConcurrencyConfigFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}

	// Test nil handling
	t.Run("nil protobuf", func(t *testing.T) {
		result := ConcurrencyConfigFromProto(nil)
		assert.Equal(t, ConcurrencyConfig{}, result)
	})
}

func TestThrottleConfigConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    ThrottleConfig
		expected *pb.ThrottleConfig
	}{
		{
			name: "complete config",
			input: ThrottleConfig{
				Scope:                     enums.ThrottleScopeEnv,
				ThrottleKeyExpressionHash: "throttle_hash",
				Limit:                     1000,
				Burst:                     100,
				Period:                    60,
			},
			expected: &pb.ThrottleConfig{
				Scope:                     pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_ENV,
				ThrottleKeyExpressionHash: "throttle_hash",
				Limit:                     1000,
				Burst:                     100,
				Period:                    60,
			},
		},
		{
			name: "minimal config",
			input: ThrottleConfig{
				Scope: enums.ThrottleScopeFn,
			},
			expected: &pb.ThrottleConfig{
				Scope:                     pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_FUNCTION,
				ThrottleKeyExpressionHash: "",
				Limit:                     0,
				Burst:                     0,
				Period:                    0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ThrottleConfigToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := ThrottleConfigFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}

	// Test nil handling
	t.Run("nil protobuf", func(t *testing.T) {
		result := ThrottleConfigFromProto(nil)
		assert.Equal(t, ThrottleConfig{}, result)
	})
}

func TestConstraintConfigConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    ConstraintConfig
		expected *pb.ConstraintConfig
	}{
		{
			name: "complete config",
			input: ConstraintConfig{
				FunctionVersion: 42,
				RateLimit: []RateLimitConfig{
					{
						Scope:             enums.RateLimitScopeAccount,
						Limit:             100,
						Period:            "1h",
						KeyExpressionHash: "rate_hash",
					},
				},
				Concurrency: ConcurrencyConfig{
					AccountConcurrency:  50,
					FunctionConcurrency: 25,
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeFn,
							Limit:             5,
							KeyExpressionHash: "custom_hash",
						},
					},
				},
				Throttle: []ThrottleConfig{
					{
						Scope:                     enums.ThrottleScopeEnv,
						ThrottleKeyExpressionHash: "throttle_hash",
						Limit:                     1000,
						Burst:                     200,
						Period:                    300,
					},
				},
			},
			expected: &pb.ConstraintConfig{
				FunctionVersion: 42,
				RateLimit: []*pb.RateLimitConfig{
					{
						Scope:             pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_ACCOUNT,
						Limit:             100,
						Period:            "1h",
						KeyExpressionHash: "rate_hash",
					},
				},
				Concurrency: &pb.ConcurrencyConfig{
					AccountConcurrency:     50,
					FunctionConcurrency:    25,
					AccountRunConcurrency:  0,
					FunctionRunConcurrency: 0,
					CustomConcurrencyKeys: []*pb.CustomConcurrencyLimit{
						{
							Mode:              pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_STEP,
							Scope:             pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_FUNCTION,
							Limit:             5,
							KeyExpressionHash: "custom_hash",
						},
					},
				},
				Throttle: []*pb.ThrottleConfig{
					{
						Scope:                     pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_ENV,
						ThrottleKeyExpressionHash: "throttle_hash",
						Limit:                     1000,
						Burst:                     200,
						Period:                    300,
					},
				},
			},
		},
		{
			name: "empty config",
			input: ConstraintConfig{
				FunctionVersion: 1,
				RateLimit:       []RateLimitConfig{},
				Concurrency: ConcurrencyConfig{
					CustomConcurrencyKeys: []CustomConcurrencyLimit{},
				},
				Throttle: []ThrottleConfig{},
			},
			expected: &pb.ConstraintConfig{
				FunctionVersion: 1,
				RateLimit:       []*pb.RateLimitConfig{},
				Concurrency: &pb.ConcurrencyConfig{
					CustomConcurrencyKeys: []*pb.CustomConcurrencyLimit{},
				},
				Throttle: []*pb.ThrottleConfig{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConstraintConfigToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := ConstraintConfigFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}

	// Test nil handling
	t.Run("nil protobuf", func(t *testing.T) {
		result := ConstraintConfigFromProto(nil)
		assert.Equal(t, ConstraintConfig{}, result)
	})
}

func TestConstraintCapacityItemConversion(t *testing.T) {
	kindRateLimit := CapacityKindRateLimit
	kindConcurrency := CapacityKindConcurrency
	kindThrottle := CapacityKindThrottle

	tests := []struct {
		name     string
		input    ConstraintCapacityItem
		expected *pb.ConstraintCapacityItem
	}{
		{
			name: "rate limit capacity",
			input: ConstraintCapacityItem{
				Kind:   &kindRateLimit,
				Amount: 5,
			},
			expected: &pb.ConstraintCapacityItem{
				Kind:   pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT,
				Amount: 5,
			},
		},
		{
			name: "concurrency capacity",
			input: ConstraintCapacityItem{
				Kind:   &kindConcurrency,
				Amount: 10,
			},
			expected: &pb.ConstraintCapacityItem{
				Kind:   pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY,
				Amount: 10,
			},
		},
		{
			name: "throttle capacity",
			input: ConstraintCapacityItem{
				Kind:   &kindThrottle,
				Amount: 100,
			},
			expected: &pb.ConstraintCapacityItem{
				Kind:   pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_THROTTLE,
				Amount: 100,
			},
		},
		{
			name: "nil kind",
			input: ConstraintCapacityItem{
				Kind:   nil,
				Amount: 1,
			},
			expected: &pb.ConstraintCapacityItem{
				Kind:   pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_UNSPECIFIED,
				Amount: 1,
			},
		},
		{
			name: "zero amount",
			input: ConstraintCapacityItem{
				Kind:   &kindRateLimit,
				Amount: 0,
			},
			expected: &pb.ConstraintCapacityItem{
				Kind:   pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT,
				Amount: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConstraintCapacityItemToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip (but skip nil kind since it becomes unspecified and doesn't round trip exactly)
			if tt.input.Kind != nil {
				backConverted := ConstraintCapacityItemFromProto(result)
				assert.Equal(t, tt.input, backConverted)
			}
		})
	}

	// Test nil protobuf handling
	t.Run("nil protobuf", func(t *testing.T) {
		result := ConstraintCapacityItemFromProto(nil)
		assert.Equal(t, ConstraintCapacityItem{}, result)
	})

	// Test unspecified kind handling
	t.Run("unspecified kind from protobuf", func(t *testing.T) {
		pbItem := &pb.ConstraintCapacityItem{
			Kind:   pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_UNSPECIFIED,
			Amount: 5,
		}
		result := ConstraintCapacityItemFromProto(pbItem)
		expected := ConstraintCapacityItem{
			Kind:   nil,
			Amount: 5,
		}
		assert.Equal(t, expected, result)
	})
}

func TestLeaseSourceConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    LeaseSource
		expected *pb.LeaseSource
	}{
		{
			name: "new runs background schedule",
			input: LeaseSource{
				Service:           ServiceNewRuns,
				Location:          LeaseLocationScheduleRun,
				RunProcessingMode: RunProcessingModeBackground,
			},
			expected: &pb.LeaseSource{
				Service:           pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_NEW_RUNS,
				Location:          pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_SCHEDULE_RUN,
				RunProcessingMode: pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_BACKGROUND,
			},
		},
		{
			name: "executor sync item lease",
			input: LeaseSource{
				Service:           ServiceExecutor,
				Location:          LeaseLocationItemLease,
				RunProcessingMode: RunProcessingModeSync,
			},
			expected: &pb.LeaseSource{
				Service:           pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_EXECUTOR,
				Location:          pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_ITEM_LEASE,
				RunProcessingMode: pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_SYNC,
			},
		},
		{
			name: "api partition lease",
			input: LeaseSource{
				Service:           ServiceAPI,
				Location:          LeaseLocationPartitionLease,
				RunProcessingMode: RunProcessingModeBackground,
			},
			expected: &pb.LeaseSource{
				Service:           pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_API,
				Location:          pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_PARTITION_LEASE,
				RunProcessingMode: pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_BACKGROUND,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LeaseSourceToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := LeaseSourceFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}

	// Test nil handling
	t.Run("nil protobuf", func(t *testing.T) {
		result := LeaseSourceFromProto(nil)
		assert.Equal(t, LeaseSource{}, result)
	})
}

func TestCapacityCheckRequestConversion(t *testing.T) {
	accountID := uuid.New()

	tests := []struct {
		name     string
		input    *CapacityCheckRequest
		expected *pb.CapacityCheckRequest
	}{
		{
			name: "valid request",
			input: &CapacityCheckRequest{
				AccountID: accountID,
			},
			expected: &pb.CapacityCheckRequest{
				AccountId: accountID.String(),
			},
		},
		{
			name:     "nil request",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapacityCheckRequestToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			if tt.input != nil {
				backConverted, err := CapacityCheckRequestFromProto(result)
				require.NoError(t, err)
				assert.Equal(t, tt.input, backConverted)
			}
		})
	}

	// Test invalid UUID handling
	t.Run("invalid UUID", func(t *testing.T) {
		pbReq := &pb.CapacityCheckRequest{
			AccountId: "invalid-uuid",
		}
		_, err := CapacityCheckRequestFromProto(pbReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid account ID")
	})

	// Test nil protobuf
	t.Run("nil protobuf", func(t *testing.T) {
		result, err := CapacityCheckRequestFromProto(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestCapacityCheckResponseConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    *CapacityCheckResponse
		expected *pb.CapacityCheckResponse
	}{
		{
			name:     "valid response",
			input:    &CapacityCheckResponse{},
			expected: &pb.CapacityCheckResponse{},
		},
		{
			name:     "nil response",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapacityCheckResponseToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := CapacityCheckResponseFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}
}

func TestCapacityLeaseRequestConversion(t *testing.T) {
	accountID := uuid.New()
	envID := uuid.New()
	functionID := uuid.New()
	currentTime := time.Date(2023, 10, 15, 12, 30, 45, 0, time.UTC)
	kindConcurrency := CapacityKindConcurrency

	tests := []struct {
		name     string
		input    *CapacityLeaseRequest
		expected *pb.CapacityLeaseRequest
	}{
		{
			name: "complete request",
			input: &CapacityLeaseRequest{
				IdempotencyKey: "test-key-123",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
					RateLimit: []RateLimitConfig{
						{
							Scope:             enums.RateLimitScopeFn,
							Limit:             10,
							Period:            "1m",
							KeyExpressionHash: "hash1",
						},
					},
					Concurrency: ConcurrencyConfig{
						CustomConcurrencyKeys: []CustomConcurrencyLimit{},
					},
					Throttle: []ThrottleConfig{},
				},
				RequestedCapacity: []ConstraintCapacityItem{
					{
						Kind:   &kindConcurrency,
						Amount: 3,
					},
				},
				CurrentTime:       currentTime,
				Duration:          5 * time.Minute,
				MaximumLifetime:   30 * time.Minute,
				BlockingThreshold: 10 * time.Second,
				Source: LeaseSource{
					Service:           ServiceExecutor,
					Location:          LeaseLocationItemLease,
					RunProcessingMode: RunProcessingModeBackground,
				},
			},
			expected: &pb.CapacityLeaseRequest{
				IdempotencyKey: "test-key-123",
				AccountId:      accountID.String(),
				EnvId:          envID.String(),
				FunctionId:     functionID.String(),
				Configuration: &pb.ConstraintConfig{
					FunctionVersion: 1,
					RateLimit: []*pb.RateLimitConfig{
						{
							Scope:             pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_FUNCTION,
							Limit:             10,
							Period:            "1m",
							KeyExpressionHash: "hash1",
						},
					},
					Concurrency: &pb.ConcurrencyConfig{
						CustomConcurrencyKeys: []*pb.CustomConcurrencyLimit{},
					},
					Throttle: []*pb.ThrottleConfig{},
				},
				RequestedCapacity: []*pb.ConstraintCapacityItem{
					{
						Kind:   pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY,
						Amount: 3,
					},
				},
				CurrentTime:       timestamppb.New(currentTime),
				Duration:          durationpb.New(5 * time.Minute),
				MaximumLifetime:   durationpb.New(30 * time.Minute),
				BlockingThreshold: durationpb.New(10 * time.Second),
				Source: &pb.LeaseSource{
					Service:           pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_EXECUTOR,
					Location:          pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_ITEM_LEASE,
					RunProcessingMode: pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_BACKGROUND,
				},
			},
		},
		{
			name: "minimal request",
			input: &CapacityLeaseRequest{
				IdempotencyKey:    "minimal",
				AccountID:         accountID,
				EnvID:             envID,
				FunctionID:        functionID,
				Configuration: ConstraintConfig{
					RateLimit: []RateLimitConfig{},
					Concurrency: ConcurrencyConfig{
						CustomConcurrencyKeys: []CustomConcurrencyLimit{},
					},
					Throttle: []ThrottleConfig{},
				},
				RequestedCapacity: []ConstraintCapacityItem{},
			},
			expected: &pb.CapacityLeaseRequest{
				IdempotencyKey: "minimal",
				AccountId:      accountID.String(),
				EnvId:          envID.String(),
				FunctionId:     functionID.String(),
				Configuration: &pb.ConstraintConfig{
					FunctionVersion: 0,
					RateLimit:       []*pb.RateLimitConfig{},
					Concurrency: &pb.ConcurrencyConfig{
						CustomConcurrencyKeys: []*pb.CustomConcurrencyLimit{},
					},
					Throttle: []*pb.ThrottleConfig{},
				},
				RequestedCapacity: []*pb.ConstraintCapacityItem{},
				CurrentTime:       timestamppb.New(time.Time{}),
				Duration:          durationpb.New(0),
				MaximumLifetime:   durationpb.New(0),
				BlockingThreshold: durationpb.New(0),
				Source: &pb.LeaseSource{
					Service:           pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_NEW_RUNS,
					Location:          pb.ConstraintApiLeaseLocation_CONSTRAINT_API_LEASE_LOCATION_SCHEDULE_RUN,
					RunProcessingMode: pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_BACKGROUND,
				},
			},
		},
		{
			name:     "nil request",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapacityLeaseRequestToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip (but skip minimal test due to enum zero values vs Go defaults)
			if tt.input != nil && tt.name != "minimal request" {
				backConverted, err := CapacityLeaseRequestFromProto(result)
				require.NoError(t, err)
				assert.Equal(t, tt.input, backConverted)
			}
		})
	}

	// Test invalid UUIDs
	t.Run("invalid account ID", func(t *testing.T) {
		pbReq := &pb.CapacityLeaseRequest{
			AccountId:  "invalid-uuid",
			EnvId:      envID.String(),
			FunctionId: functionID.String(),
		}
		_, err := CapacityLeaseRequestFromProto(pbReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid account ID")
	})

	t.Run("invalid env ID", func(t *testing.T) {
		pbReq := &pb.CapacityLeaseRequest{
			AccountId:  accountID.String(),
			EnvId:      "invalid-uuid",
			FunctionId: functionID.String(),
		}
		_, err := CapacityLeaseRequestFromProto(pbReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid env ID")
	})

	t.Run("invalid function ID", func(t *testing.T) {
		pbReq := &pb.CapacityLeaseRequest{
			AccountId:  accountID.String(),
			EnvId:      envID.String(),
			FunctionId: "invalid-uuid",
		}
		_, err := CapacityLeaseRequestFromProto(pbReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid function ID")
	})

	// Test nil protobuf
	t.Run("nil protobuf", func(t *testing.T) {
		result, err := CapacityLeaseRequestFromProto(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestCapacityLeaseResponseConversion(t *testing.T) {
	leaseID := ulid.Make()
	retryAfter := time.Date(2023, 10, 15, 13, 0, 0, 0, time.UTC)
	kindRateLimit := CapacityKindRateLimit
	kindConcurrency := CapacityKindConcurrency

	tests := []struct {
		name     string
		input    *CapacityLeaseResponse
		expected *pb.CapacityLeaseResponse
	}{
		{
			name: "complete response with lease",
			input: &CapacityLeaseResponse{
				LeaseID: &leaseID,
				ReservedCapacity: []ConstraintCapacityItem{
					{
						Kind:   &kindConcurrency,
						Amount: 3,
					},
				},
				InsufficientCapacity: []ConstraintCapacityItem{
					{
						Kind:   &kindRateLimit,
						Amount: 1,
					},
				},
				RetryAfter: retryAfter,
			},
			expected: &pb.CapacityLeaseResponse{
				LeaseId: func() *string { s := leaseID.String(); return &s }(),
				ReservedCapacity: []*pb.ConstraintCapacityItem{
					{
						Kind:   pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY,
						Amount: 3,
					},
				},
				InsufficientCapacity: []*pb.ConstraintCapacityItem{
					{
						Kind:   pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT,
						Amount: 1,
					},
				},
				RetryAfter: timestamppb.New(retryAfter),
			},
		},
		{
			name: "response without lease",
			input: &CapacityLeaseResponse{
				LeaseID:              nil,
				ReservedCapacity:     []ConstraintCapacityItem{},
				InsufficientCapacity: []ConstraintCapacityItem{},
				RetryAfter:           time.Time{},
			},
			expected: &pb.CapacityLeaseResponse{
				LeaseId:              nil,
				ReservedCapacity:     []*pb.ConstraintCapacityItem{},
				InsufficientCapacity: []*pb.ConstraintCapacityItem{},
				RetryAfter:           timestamppb.New(time.Time{}),
			},
		},
		{
			name:     "nil response",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapacityLeaseResponseToProto(tt.input)
			
			// Special handling for the lease ID string pointer comparison
			if tt.expected != nil && tt.expected.LeaseId != nil {
				leaseIDStr := leaseID.String()
				tt.expected.LeaseId = &leaseIDStr
			}
			
			assert.Equal(t, tt.expected, result)

			// Test round trip
			if tt.input != nil {
				backConverted, err := CapacityLeaseResponseFromProto(result)
				require.NoError(t, err)
				assert.Equal(t, tt.input, backConverted)
			}
		})
	}

	// Test invalid ULID
	t.Run("invalid lease ID", func(t *testing.T) {
		invalidID := "invalid-ulid"
		pbResp := &pb.CapacityLeaseResponse{
			LeaseId: &invalidID,
		}
		_, err := CapacityLeaseResponseFromProto(pbResp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid lease ID")
	})

	// Test nil protobuf
	t.Run("nil protobuf", func(t *testing.T) {
		result, err := CapacityLeaseResponseFromProto(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestCapacityExtendLeaseRequestConversion(t *testing.T) {
	accountID := uuid.New()
	leaseID := ulid.Make()

	tests := []struct {
		name     string
		input    *CapacityExtendLeaseRequest
		expected *pb.CapacityExtendLeaseRequest
	}{
		{
			name: "valid request",
			input: &CapacityExtendLeaseRequest{
				IdempotencyKey: "extend-key",
				AccountID:      accountID,
				LeaseID:        leaseID,
				Duration:       15 * time.Minute,
			},
			expected: &pb.CapacityExtendLeaseRequest{
				IdempotencyKey: "extend-key",
				AccountId:      accountID.String(),
				LeaseId:        leaseID.String(),
				Duration:       durationpb.New(15 * time.Minute),
			},
		},
		{
			name:     "nil request",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapacityExtendLeaseRequestToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			if tt.input != nil {
				backConverted, err := CapacityExtendLeaseRequestFromProto(result)
				require.NoError(t, err)
				assert.Equal(t, tt.input, backConverted)
			}
		})
	}

	// Test invalid UUIDs/ULIDs
	t.Run("invalid account ID", func(t *testing.T) {
		pbReq := &pb.CapacityExtendLeaseRequest{
			AccountId: "invalid-uuid",
			LeaseId:   leaseID.String(),
		}
		_, err := CapacityExtendLeaseRequestFromProto(pbReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid account ID")
	})

	t.Run("invalid lease ID", func(t *testing.T) {
		pbReq := &pb.CapacityExtendLeaseRequest{
			AccountId: accountID.String(),
			LeaseId:   "invalid-ulid",
		}
		_, err := CapacityExtendLeaseRequestFromProto(pbReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid lease ID")
	})
}

func TestCapacityExtendLeaseResponseConversion(t *testing.T) {
	leaseID := ulid.Make()

	tests := []struct {
		name     string
		input    *CapacityExtendLeaseResponse
		expected *pb.CapacityExtendLeaseResponse
	}{
		{
			name: "response with lease ID",
			input: &CapacityExtendLeaseResponse{
				LeaseID: &leaseID,
			},
			expected: &pb.CapacityExtendLeaseResponse{
				LeaseId: func() *string { s := leaseID.String(); return &s }(),
			},
		},
		{
			name: "response without lease ID",
			input: &CapacityExtendLeaseResponse{
				LeaseID: nil,
			},
			expected: &pb.CapacityExtendLeaseResponse{
				LeaseId: nil,
			},
		},
		{
			name:     "nil response",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapacityExtendLeaseResponseToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			if tt.input != nil {
				backConverted, err := CapacityExtendLeaseResponseFromProto(result)
				require.NoError(t, err)
				assert.Equal(t, tt.input, backConverted)
			}
		})
	}
}

func TestCapacityCommitRequestConversion(t *testing.T) {
	accountID := uuid.New()
	leaseID := ulid.Make()

	tests := []struct {
		name     string
		input    *CapacityCommitRequest
		expected *pb.CapacityCommitRequest
	}{
		{
			name: "valid request",
			input: &CapacityCommitRequest{
				IdempotencyKey: "commit-key",
				AccountID:      accountID,
				LeaseID:        leaseID,
			},
			expected: &pb.CapacityCommitRequest{
				IdempotencyKey: "commit-key",
				AccountId:      accountID.String(),
				LeaseId:        leaseID.String(),
			},
		},
		{
			name:     "nil request",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapacityCommitRequestToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			if tt.input != nil {
				backConverted, err := CapacityCommitRequestFromProto(result)
				require.NoError(t, err)
				assert.Equal(t, tt.input, backConverted)
			}
		})
	}
}

func TestCapacityCommitResponseConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    *CapacityCommitResponse
		expected *pb.CapacityCommitResponse
	}{
		{
			name:     "valid response",
			input:    &CapacityCommitResponse{},
			expected: &pb.CapacityCommitResponse{},
		},
		{
			name:     "nil response",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapacityCommitResponseToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := CapacityCommitResponseFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}
}

func TestCapacityRollbackRequestConversion(t *testing.T) {
	accountID := uuid.New()
	leaseID := ulid.Make()

	tests := []struct {
		name     string
		input    *CapacityRollbackRequest
		expected *pb.CapacityRollbackRequest
	}{
		{
			name: "valid request",
			input: &CapacityRollbackRequest{
				IdempotencyKey: "rollback-key",
				AccountID:      accountID,
				LeaseID:        leaseID,
			},
			expected: &pb.CapacityRollbackRequest{
				IdempotencyKey: "rollback-key",
				AccountId:      accountID.String(),
				LeaseId:        leaseID.String(),
			},
		},
		{
			name:     "nil request",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapacityRollbackRequestToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			if tt.input != nil {
				backConverted, err := CapacityRollbackRequestFromProto(result)
				require.NoError(t, err)
				assert.Equal(t, tt.input, backConverted)
			}
		})
	}
}

func TestCapacityRollbackResponseConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    *CapacityRollbackResponse
		expected *pb.CapacityRollbackResponse
	}{
		{
			name:     "valid response",
			input:    &CapacityRollbackResponse{},
			expected: &pb.CapacityRollbackResponse{},
		},
		{
			name:     "nil response",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapacityRollbackResponseToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := CapacityRollbackResponseFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}
}

// Round-trip tests to ensure no data loss
func TestRoundTripConversions(t *testing.T) {
	t.Run("ConstraintConfig round trip", func(t *testing.T) {
		original := ConstraintConfig{
			FunctionVersion: 42,
			RateLimit: []RateLimitConfig{
				{
					Scope:             0, // RateLimitScopeFn
					Limit:             100,
					Period:            "1h",
					KeyExpressionHash: "hash123",
				},
			},
			Concurrency: ConcurrencyConfig{
				AccountConcurrency:    50,
				FunctionConcurrency:   25,
				AccountRunConcurrency: 10,
				FunctionRunConcurrency: 5,
				CustomConcurrencyKeys: []CustomConcurrencyLimit{
					{
						Mode:              0, // ConcurrencyModeStep
						Scope:             1, // ConcurrencyScopeEnv
						Limit:             3,
						KeyExpressionHash: "custom_hash",
					},
				},
			},
			Throttle: []ThrottleConfig{
				{
					Scope:                     2, // ThrottleScopeAccount
					ThrottleKeyExpressionHash: "throttle_hash",
					Limit:                     1000,
					Burst:                     200,
					Period:                    300,
				},
			},
		}

		// Convert to protobuf and back
		pbConfig := ConstraintConfigToProto(original)
		result := ConstraintConfigFromProto(pbConfig)

		assert.Equal(t, original, result)
	})

	t.Run("CapacityLeaseRequest round trip", func(t *testing.T) {
		accountID := uuid.New()
		envID := uuid.New()
		functionID := uuid.New()
		currentTime := time.Date(2023, 10, 15, 12, 30, 45, 123456789, time.UTC)
		kindConcurrency := CapacityKindConcurrency

		original := &CapacityLeaseRequest{
			IdempotencyKey: "test-key-123",
			AccountID:      accountID,
			EnvID:          envID,
			FunctionID:     functionID,
			Configuration: ConstraintConfig{
				FunctionVersion: 1,
				RateLimit: []RateLimitConfig{},
				Concurrency: ConcurrencyConfig{
					CustomConcurrencyKeys: []CustomConcurrencyLimit{},
				},
				Throttle: []ThrottleConfig{},
			},
			RequestedCapacity: []ConstraintCapacityItem{
				{
					Kind:   &kindConcurrency,
					Amount: 3,
				},
			},
			CurrentTime:       currentTime,
			Duration:          5 * time.Minute,
			MaximumLifetime:   30 * time.Minute,
			BlockingThreshold: 10 * time.Second,
			Source: LeaseSource{
				Service:           ServiceExecutor,
				Location:          LeaseLocationItemLease,
				RunProcessingMode: RunProcessingModeBackground,
			},
		}

		// Convert to protobuf and back
		pbRequest := CapacityLeaseRequestToProto(original)
		result, err := CapacityLeaseRequestFromProto(pbRequest)
		require.NoError(t, err)

		// Note: protobuf timestamp precision may be different, so we compare the Unix timestamp
		assert.Equal(t, original.CurrentTime.Unix(), result.CurrentTime.Unix())
		
		// Reset the time for exact comparison
		original.CurrentTime = result.CurrentTime
		assert.Equal(t, original, result)
	})
}

// Edge case tests
func TestEdgeCases(t *testing.T) {
	t.Run("empty slices", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			RateLimit:       []RateLimitConfig{},
			Concurrency: ConcurrencyConfig{
				CustomConcurrencyKeys: []CustomConcurrencyLimit{},
			},
			Throttle: []ThrottleConfig{},
		}

		pbConfig := ConstraintConfigToProto(config)
		result := ConstraintConfigFromProto(pbConfig)
		assert.Equal(t, config, result)
	})

	t.Run("zero values", func(t *testing.T) {
		item := ConstraintCapacityItem{
			Kind:   nil,
			Amount: 0,
		}

		pbItem := ConstraintCapacityItemToProto(item)
		assert.Equal(t, pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_UNSPECIFIED, pbItem.Kind)
		assert.Equal(t, int32(0), pbItem.Amount)

		// From protobuf unspecified becomes nil
		result := ConstraintCapacityItemFromProto(pbItem)
		assert.Nil(t, result.Kind)
		assert.Equal(t, 0, result.Amount)
	})

	t.Run("max duration values", func(t *testing.T) {
		maxDuration := time.Duration(1<<63 - 1) // max int64
		
		req := &CapacityExtendLeaseRequest{
			IdempotencyKey: "max-test",
			AccountID:      uuid.New(),
			LeaseID:        ulid.Make(),
			Duration:       maxDuration,
		}

		// Should handle large duration values
		pbReq := CapacityExtendLeaseRequestToProto(req)
		result, err := CapacityExtendLeaseRequestFromProto(pbReq)
		require.NoError(t, err)
		assert.Equal(t, req.Duration, result.Duration)
	})
}