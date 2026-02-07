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
			input:    ConstraintKindRateLimit,
			expected: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT,
		},
		{
			name:     "concurrency",
			input:    ConstraintKindConcurrency,
			expected: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY,
		},
		{
			name:     "throttle",
			input:    ConstraintKindThrottle,
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
			name:     "durable endpoint mode",
			input:    RunProcessingModeDurableEndpoint,
			expected: pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_DURABLE_ENDPOINT,
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

func TestCallerLocationConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    CallerLocation
		expected pb.ConstraintApiCallerLocation
	}{
		{
			name:     "unknown location",
			input:    CallerLocationUnknown,
			expected: pb.ConstraintApiCallerLocation_CONSTRAINT_API_CALLER_LOCATION_UNSPECIFIED,
		},
		{
			name:     "schedule run",
			input:    CallerLocationSchedule,
			expected: pb.ConstraintApiCallerLocation_CONSTRAINT_API_CALLER_LOCATION_SCHEDULE,
		},
		{
			name:     "backlog refill",
			input:    CallerLocationBacklogRefill,
			expected: pb.ConstraintApiCallerLocation_CONSTRAINT_API_CALLER_LOCATION_BACKLOG_REFILL,
		},
		{
			name:     "item lease",
			input:    CallerLocationItemLease,
			expected: pb.ConstraintApiCallerLocation_CONSTRAINT_API_CALLER_LOCATION_ITEM_LEASE,
		},
		{
			name:     "checkpoint",
			input:    CallerLocationCheckpoint,
			expected: pb.ConstraintApiCallerLocation_CONSTRAINT_API_CALLER_LOCATION_CHECKPOINT,
		},
		{
			name:     "invalid location (fallback to unspecified)",
			input:    CallerLocation(999),
			expected: pb.ConstraintApiCallerLocation_CONSTRAINT_API_CALLER_LOCATION_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CallerLocationToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := LeaseLocationFromProto(result)
			if tt.input != CallerLocation(999) { // Skip round trip for invalid input
				assert.Equal(t, tt.input, backConverted)
			}
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
			name:     "unknown service",
			input:    ServiceUnknown,
			expected: pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_UNSPECIFIED,
		},
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
		{
			name:     "invalid service (fallback to unspecified)",
			input:    LeaseService(999),
			expected: pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LeaseServiceToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := LeaseServiceFromProto(result)
			if tt.input != LeaseService(999) { // Skip round trip for invalid input
				assert.Equal(t, tt.input, backConverted)
			}
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
				Period:            60,
				KeyExpressionHash: "hash123",
			},
			expected: &pb.RateLimitConfig{
				Scope:             pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_ACCOUNT,
				Limit:             100,
				Period:            60,
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
				Period:            0,
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
				AccountConcurrency:     100,
				FunctionConcurrency:    50,
				AccountRunConcurrency:  20,
				FunctionRunConcurrency: 10,
				CustomConcurrencyKeys:  customKeys,
			},
			expected: &pb.ConcurrencyConfig{
				AccountConcurrency:     100,
				FunctionConcurrency:    50,
				AccountRunConcurrency:  20,
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
				AccountConcurrency:     0,
				FunctionConcurrency:    0,
				AccountRunConcurrency:  0,
				FunctionRunConcurrency: 0,
				CustomConcurrencyKeys:  []*pb.CustomConcurrencyLimit{},
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
				Scope:             enums.ThrottleScopeEnv,
				KeyExpressionHash: "throttle_hash",
				Limit:             1000,
				Burst:             100,
				Period:            60,
			},
			expected: &pb.ThrottleConfig{
				Scope:             pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_ENV,
				KeyExpressionHash: "throttle_hash",
				Limit:             1000,
				Burst:             100,
				Period:            60,
			},
		},
		{
			name: "minimal config",
			input: ThrottleConfig{
				Scope: enums.ThrottleScopeFn,
			},
			expected: &pb.ThrottleConfig{
				Scope:             pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_FUNCTION,
				KeyExpressionHash: "",
				Limit:             0,
				Burst:             0,
				Period:            0,
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
						Period:            3600,
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
						Scope:             enums.ThrottleScopeEnv,
						KeyExpressionHash: "throttle_hash",
						Limit:             1000,
						Burst:             200,
						Period:            300,
					},
				},
			},
			expected: &pb.ConstraintConfig{
				FunctionVersion: 42,
				RateLimit: []*pb.RateLimitConfig{
					{
						Scope:             pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_ACCOUNT,
						Limit:             100,
						Period:            3600,
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
						Scope:             pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_ENV,
						KeyExpressionHash: "throttle_hash",
						Limit:             1000,
						Burst:             200,
						Period:            300,
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

func TestConstraintItemConversion(t *testing.T) {
	kindRateLimit := ConstraintKindRateLimit
	kindConcurrency := ConstraintKindConcurrency
	kindThrottle := ConstraintKindThrottle

	tests := []struct {
		name     string
		input    ConstraintItem
		expected *pb.ConstraintItem
	}{
		{
			name: "rate limit constraint",
			input: ConstraintItem{
				Kind: kindRateLimit,
			},
			expected: &pb.ConstraintItem{
				Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT,
			},
		},
		{
			name: "concurrency constraint",
			input: ConstraintItem{
				Kind: kindConcurrency,
			},
			expected: &pb.ConstraintItem{
				Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY,
			},
		},
		{
			name: "throttle constraint",
			input: ConstraintItem{
				Kind: kindThrottle,
			},
			expected: &pb.ConstraintItem{
				Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_THROTTLE,
			},
		},
		{
			name: "rate limit constraint with details",
			input: ConstraintItem{
				Kind: kindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeFn,
					KeyExpressionHash: "hash123",
					EvaluatedKeyHash:  "eval456",
				},
			},
			expected: &pb.ConstraintItem{
				Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT,
				RateLimit: &pb.RateLimitConstraint{
					Scope:             pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_FUNCTION,
					KeyExpressionHash: "hash123",
					EvaluatedKeyHash:  "eval456",
				},
			},
		},
		{
			name: "concurrency constraint with details",
			input: ConstraintItem{
				Kind: kindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeEnv,
					KeyExpressionHash: "concurrency_hash",
					EvaluatedKeyHash:  "eval_concurrency",
				},
			},
			expected: &pb.ConstraintItem{
				Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY,
				Concurrency: &pb.ConcurrencyConstraint{
					Mode:              pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_STEP,
					Scope:             pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_ENV,
					KeyExpressionHash: "concurrency_hash",
					EvaluatedKeyHash:  "eval_concurrency",
				},
			},
		},
		{
			name: "throttle constraint with details",
			input: ConstraintItem{
				Kind: kindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeAccount,
					KeyExpressionHash: "throttle_hash",
					EvaluatedKeyHash:  "eval_throttle",
				},
			},
			expected: &pb.ConstraintItem{
				Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_THROTTLE,
				Throttle: &pb.ThrottleConstraint{
					Scope:             pb.ConstraintApiThrottleScope_CONSTRAINT_API_THROTTLE_SCOPE_ACCOUNT,
					KeyExpressionHash: "throttle_hash",
					EvaluatedKeyHash:  "eval_throttle",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConstraintItemToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := ConstraintItemFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}

	// Test nil protobuf handling
	t.Run("nil protobuf", func(t *testing.T) {
		result := ConstraintItemFromProto(nil)
		assert.Equal(t, ConstraintItem{}, result)
	})

	// Test unspecified kind handling
	t.Run("unspecified kind from protobuf", func(t *testing.T) {
		pbItem := &pb.ConstraintItem{
			Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_UNSPECIFIED,
		}
		result := ConstraintItemFromProto(pbItem)
		expected := ConstraintItem{
			Kind: ConstraintKind(""),
		}
		assert.Equal(t, expected, result)
	})
}

func TestConstraintUsageConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    ConstraintUsage
		expected *pb.ConstraintUsage
	}{
		{
			name: "complete usage",
			input: ConstraintUsage{
				Constraint: ConstraintItem{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeFn,
						KeyExpressionHash: "hash123",
						EvaluatedKeyHash:  "eval456",
					},
				},
				Used:  5,
				Limit: 10,
			},
			expected: &pb.ConstraintUsage{
				Constraint: &pb.ConstraintItem{
					Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY,
					Concurrency: &pb.ConcurrencyConstraint{
						Mode:              pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_STEP,
						Scope:             pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_FUNCTION,
						KeyExpressionHash: "hash123",
						EvaluatedKeyHash:  "eval456",
					},
				},
				Used:  5,
				Limit: 10,
			},
		},
		{
			name: "minimal usage",
			input: ConstraintUsage{
				Constraint: ConstraintItem{
					Kind: ConstraintKindRateLimit,
				},
				Used:  0,
				Limit: 100,
			},
			expected: &pb.ConstraintUsage{
				Constraint: &pb.ConstraintItem{
					Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT,
				},
				Used:  0,
				Limit: 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConstraintUsageToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := ConstraintUsageFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}

	// Test nil handling
	t.Run("nil protobuf", func(t *testing.T) {
		result := ConstraintUsageFromProto(nil)
		assert.Equal(t, ConstraintUsage{}, result)
	})
}

func TestCapacityLeaseConversion(t *testing.T) {
	leaseID := ulid.Make()

	tests := []struct {
		name     string
		input    CapacityLease
		expected *pb.CapacityLease
	}{
		{
			name: "complete lease",
			input: CapacityLease{
				LeaseID:        leaseID,
				IdempotencyKey: "test-key-123",
			},
			expected: &pb.CapacityLease{
				LeaseId:        leaseID.String(),
				IdempotencyKey: "test-key-123",
			},
		},
		{
			name: "minimal lease",
			input: CapacityLease{
				LeaseID:        leaseID,
				IdempotencyKey: "",
			},
			expected: &pb.CapacityLease{
				LeaseId:        leaseID.String(),
				IdempotencyKey: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapacityLeaseToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted, err := CapacityLeaseFromProto(result)
			require.NoError(t, err)
			assert.Equal(t, tt.input, backConverted)
		})
	}

	// Test invalid ULID
	t.Run("invalid lease ID", func(t *testing.T) {
		pbLease := &pb.CapacityLease{
			LeaseId:        "invalid-ulid",
			IdempotencyKey: "test-key",
		}
		_, err := CapacityLeaseFromProto(pbLease)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid lease ID")
	})

	// Test nil protobuf
	t.Run("nil protobuf", func(t *testing.T) {
		result, err := CapacityLeaseFromProto(nil)
		require.NoError(t, err)
		assert.Equal(t, CapacityLease{}, result)
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
				Location:          CallerLocationSchedule,
				RunProcessingMode: RunProcessingModeBackground,
			},
			expected: &pb.LeaseSource{
				Service:           pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_NEW_RUNS,
				Location:          pb.ConstraintApiCallerLocation_CONSTRAINT_API_CALLER_LOCATION_SCHEDULE,
				RunProcessingMode: pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_BACKGROUND,
			},
		},
		{
			name: "executor durable endpoint item lease",
			input: LeaseSource{
				Service:           ServiceExecutor,
				Location:          CallerLocationItemLease,
				RunProcessingMode: RunProcessingModeDurableEndpoint,
			},
			expected: &pb.LeaseSource{
				Service:           pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_EXECUTOR,
				Location:          pb.ConstraintApiCallerLocation_CONSTRAINT_API_CALLER_LOCATION_ITEM_LEASE,
				RunProcessingMode: pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_DURABLE_ENDPOINT,
			},
		},
		{
			name: "api backlog refill",
			input: LeaseSource{
				Service:           ServiceAPI,
				Location:          CallerLocationBacklogRefill,
				RunProcessingMode: RunProcessingModeBackground,
			},
			expected: &pb.LeaseSource{
				Service:           pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_API,
				Location:          pb.ConstraintApiCallerLocation_CONSTRAINT_API_CALLER_LOCATION_BACKLOG_REFILL,
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
	envID := uuid.New()
	functionID := uuid.New()

	tests := []struct {
		name     string
		input    *CapacityCheckRequest
		expected *pb.CapacityCheckRequest
	}{
		{
			name: "complete request",
			input: &CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
					RateLimit:       []RateLimitConfig{},
					Concurrency: ConcurrencyConfig{
						CustomConcurrencyKeys: []CustomConcurrencyLimit{},
					},
					Throttle: []ThrottleConfig{},
				},
				Constraints: []ConstraintItem{
					{
						Kind: ConstraintKindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeFn,
							KeyExpressionHash: "hash123",
							EvaluatedKeyHash:  "eval456",
						},
					},
				},
			},
			expected: &pb.CapacityCheckRequest{
				AccountId:  accountID.String(),
				EnvId:      envID.String(),
				FunctionId: functionID.String(),
				Configuration: &pb.ConstraintConfig{
					FunctionVersion: 1,
					RateLimit:       []*pb.RateLimitConfig{},
					Concurrency: &pb.ConcurrencyConfig{
						CustomConcurrencyKeys: []*pb.CustomConcurrencyLimit{},
					},
					Throttle: []*pb.ThrottleConfig{},
				},
				Constraints: []*pb.ConstraintItem{
					{
						Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY,
						Concurrency: &pb.ConcurrencyConstraint{
							Mode:              pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_STEP,
							Scope:             pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_FUNCTION,
							KeyExpressionHash: "hash123",
							EvaluatedKeyHash:  "eval456",
						},
					},
				},
			},
		},
		{
			name: "minimal request",
			input: &CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					RateLimit: []RateLimitConfig{},
					Concurrency: ConcurrencyConfig{
						CustomConcurrencyKeys: []CustomConcurrencyLimit{},
					},
					Throttle: []ThrottleConfig{},
				},
				Constraints: []ConstraintItem{},
			},
			expected: &pb.CapacityCheckRequest{
				AccountId:  accountID.String(),
				EnvId:      envID.String(),
				FunctionId: functionID.String(),
				Configuration: &pb.ConstraintConfig{
					FunctionVersion: 0,
					RateLimit:       []*pb.RateLimitConfig{},
					Concurrency: &pb.ConcurrencyConfig{
						CustomConcurrencyKeys: []*pb.CustomConcurrencyLimit{},
					},
					Throttle: []*pb.ThrottleConfig{},
				},
				Constraints: []*pb.ConstraintItem{},
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
	t.Run("invalid account ID", func(t *testing.T) {
		pbReq := &pb.CapacityCheckRequest{
			AccountId:  "invalid-uuid",
			EnvId:      envID.String(),
			FunctionId: functionID.String(),
		}
		_, err := CapacityCheckRequestFromProto(pbReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid account ID")
	})

	t.Run("invalid env ID", func(t *testing.T) {
		pbReq := &pb.CapacityCheckRequest{
			AccountId:  accountID.String(),
			EnvId:      "invalid-uuid",
			FunctionId: functionID.String(),
		}
		_, err := CapacityCheckRequestFromProto(pbReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid env ID")
	})

	t.Run("invalid function ID", func(t *testing.T) {
		pbReq := &pb.CapacityCheckRequest{
			AccountId:  accountID.String(),
			EnvId:      envID.String(),
			FunctionId: "invalid-uuid",
		}
		_, err := CapacityCheckRequestFromProto(pbReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid function ID")
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
			name: "complete response",
			input: &CapacityCheckResponse{
				AvailableCapacity: 50,
				LimitingConstraints: []ConstraintItem{
					{
						Kind: ConstraintKindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeFn,
							KeyExpressionHash: "limiting_hash",
							EvaluatedKeyHash:  "limiting_eval",
						},
					},
				},
				ExhaustedConstraints: []ConstraintItem{},
				Usage: []ConstraintUsage{
					{
						Constraint: ConstraintItem{
							Kind: ConstraintKindRateLimit,
							RateLimit: &RateLimitConstraint{
								Scope:             enums.RateLimitScopeFn,
								KeyExpressionHash: "usage_hash",
								EvaluatedKeyHash:  "usage_eval",
							},
						},
						Used:  25,
						Limit: 100,
					},
				},
			},
			expected: &pb.CapacityCheckResponse{
				AvailableCapacity: 50,
				LimitingConstraints: []*pb.ConstraintItem{
					{
						Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY,
						Concurrency: &pb.ConcurrencyConstraint{
							Mode:              pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_STEP,
							Scope:             pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_FUNCTION,
							KeyExpressionHash: "limiting_hash",
							EvaluatedKeyHash:  "limiting_eval",
						},
					},
				},
				Usage: []*pb.ConstraintUsage{
					{
						Constraint: &pb.ConstraintItem{
							Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT,
							RateLimit: &pb.RateLimitConstraint{
								Scope:             pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_FUNCTION,
								KeyExpressionHash: "usage_hash",
								EvaluatedKeyHash:  "usage_eval",
							},
						},
						Used:  25,
						Limit: 100,
					},
				},
				ExhaustedConstraints: []*pb.ConstraintItem{},
			},
		},
		{
			name: "empty response",
			input: &CapacityCheckResponse{
				AvailableCapacity:    0,
				LimitingConstraints:  []ConstraintItem{},
				ExhaustedConstraints: []ConstraintItem{},
				Usage:                []ConstraintUsage{},
				RetryAfter:           time.Time{},
			},
			expected: &pb.CapacityCheckResponse{
				AvailableCapacity:    0,
				LimitingConstraints:  []*pb.ConstraintItem{},
				ExhaustedConstraints: []*pb.ConstraintItem{},
				Usage:                []*pb.ConstraintUsage{},
				RetryAfter:           nil,
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
			result := CapacityCheckResponseToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := CapacityCheckResponseFromProto(result)
			assert.Equal(t, tt.input, backConverted)
		})
	}
}

func TestCapacityAcquireRequestConversion(t *testing.T) {
	accountID := uuid.New()
	envID := uuid.New()
	functionID := uuid.New()
	currentTime := time.Date(2023, 10, 15, 12, 30, 45, 0, time.UTC)
	kindConcurrency := ConstraintKindConcurrency

	tests := []struct {
		name     string
		input    *CapacityAcquireRequest
		expected *pb.CapacityAcquireRequest
	}{
		{
			name: "complete request",
			input: &CapacityAcquireRequest{
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
							Period:            60,
							KeyExpressionHash: "hash1",
						},
					},
					Concurrency: ConcurrencyConfig{
						CustomConcurrencyKeys: []CustomConcurrencyLimit{},
					},
					Throttle: []ThrottleConfig{},
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount:               3,
				LeaseIdempotencyKeys: []string{"lease-key-1", "lease-key-2", "lease-key-3"},
				LeaseRunIDs:          map[string]ulid.ULID{},
				CurrentTime:          currentTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				BlockingThreshold:    10 * time.Second,
				Source: LeaseSource{
					Service:           ServiceExecutor,
					Location:          CallerLocationItemLease,
					RunProcessingMode: RunProcessingModeBackground,
				},
			},
			expected: &pb.CapacityAcquireRequest{
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
							Period:            60,
							KeyExpressionHash: "hash1",
						},
					},
					Concurrency: &pb.ConcurrencyConfig{
						CustomConcurrencyKeys: []*pb.CustomConcurrencyLimit{},
					},
					Throttle: []*pb.ThrottleConfig{},
				},
				Constraints: []*pb.ConstraintItem{
					{
						Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY,
					},
				},
				Amount:               3,
				LeaseIdempotencyKeys: []string{"lease-key-1", "lease-key-2", "lease-key-3"},
				LeaseRunIds:          map[string]string{},
				CurrentTime:          timestamppb.New(currentTime),
				Duration:             durationpb.New(5 * time.Minute),
				MaximumLifetime:      durationpb.New(30 * time.Minute),
				BlockingThreshold:    durationpb.New(10 * time.Second),
				Source: &pb.LeaseSource{
					Service:           pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_EXECUTOR,
					Location:          pb.ConstraintApiCallerLocation_CONSTRAINT_API_CALLER_LOCATION_ITEM_LEASE,
					RunProcessingMode: pb.ConstraintApiRunProcessingMode_CONSTRAINT_API_RUN_PROCESSING_MODE_BACKGROUND,
				},
			},
		},
		{
			name: "minimal request",
			input: &CapacityAcquireRequest{
				IdempotencyKey: "minimal",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					RateLimit: []RateLimitConfig{},
					Concurrency: ConcurrencyConfig{
						CustomConcurrencyKeys: []CustomConcurrencyLimit{},
					},
					Throttle: []ThrottleConfig{},
				},
				Constraints: []ConstraintItem{},
				Amount:      0,
				LeaseRunIDs: map[string]ulid.ULID{},
			},
			expected: &pb.CapacityAcquireRequest{
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
				Constraints:       []*pb.ConstraintItem{},
				Amount:            0,
				LeaseRunIds:       map[string]string{},
				CurrentTime:       timestamppb.New(time.Time{}),
				Duration:          durationpb.New(0),
				MaximumLifetime:   durationpb.New(0),
				BlockingThreshold: durationpb.New(0),
				Source: &pb.LeaseSource{
					Service:           pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_UNSPECIFIED,
					Location:          pb.ConstraintApiCallerLocation_CONSTRAINT_API_CALLER_LOCATION_UNSPECIFIED,
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
			result := CapacityAcquireRequestToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip (but skip minimal test due to enum zero values vs Go defaults)
			if tt.input != nil && tt.name != "minimal request" {
				backConverted, err := CapacityAcquireRequestFromProto(result)
				require.NoError(t, err)
				assert.Equal(t, tt.input, backConverted)
			}
		})
	}

	// Test invalid UUIDs
	t.Run("invalid account ID", func(t *testing.T) {
		pbReq := &pb.CapacityAcquireRequest{
			AccountId:  "invalid-uuid",
			EnvId:      envID.String(),
			FunctionId: functionID.String(),
		}
		_, err := CapacityAcquireRequestFromProto(pbReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid account ID")
	})

	t.Run("invalid env ID", func(t *testing.T) {
		pbReq := &pb.CapacityAcquireRequest{
			AccountId:  accountID.String(),
			EnvId:      "invalid-uuid",
			FunctionId: functionID.String(),
		}
		_, err := CapacityAcquireRequestFromProto(pbReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid env ID")
	})

	t.Run("invalid function ID", func(t *testing.T) {
		pbReq := &pb.CapacityAcquireRequest{
			AccountId:  accountID.String(),
			EnvId:      envID.String(),
			FunctionId: "invalid-uuid",
		}
		_, err := CapacityAcquireRequestFromProto(pbReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid function ID")
	})

	// Test nil protobuf
	t.Run("nil protobuf", func(t *testing.T) {
		result, err := CapacityAcquireRequestFromProto(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestCapacityAcquireResponseConversion(t *testing.T) {
	leaseID1 := ulid.Make()
	leaseID2 := ulid.Make()
	retryAfter := time.Date(2023, 10, 15, 13, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    *CapacityAcquireResponse
		expected *pb.CapacityAcquireResponse
	}{
		{
			name: "complete response with leases",
			input: &CapacityAcquireResponse{
				Leases: []CapacityLease{
					{
						LeaseID:        leaseID1,
						IdempotencyKey: "lease-key-1",
					},
					{
						LeaseID:        leaseID2,
						IdempotencyKey: "lease-key-2",
					},
				},
				LimitingConstraints: []ConstraintItem{
					{
						Kind: ConstraintKindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeFn,
							KeyExpressionHash: "limiting_hash",
							EvaluatedKeyHash:  "limiting_eval",
						},
					},
				},
				ExhaustedConstraints: []ConstraintItem{
					{
						Kind: ConstraintKindRateLimit,
						RateLimit: &RateLimitConstraint{
							Scope:             enums.RateLimitScopeFn,
							KeyExpressionHash: "rate-limit-key",
							EvaluatedKeyHash:  "eval-hash",
						},
					},
				},
				RetryAfter: retryAfter,
			},
			expected: &pb.CapacityAcquireResponse{
				Leases: []*pb.CapacityLease{
					{
						LeaseId:        leaseID1.String(),
						IdempotencyKey: "lease-key-1",
					},
					{
						LeaseId:        leaseID2.String(),
						IdempotencyKey: "lease-key-2",
					},
				},
				LimitingConstraints: []*pb.ConstraintItem{
					{
						Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_CONCURRENCY,
						Concurrency: &pb.ConcurrencyConstraint{
							Mode:              pb.ConstraintApiConcurrencyMode_CONSTRAINT_API_CONCURRENCY_MODE_STEP,
							Scope:             pb.ConstraintApiConcurrencyScope_CONSTRAINT_API_CONCURRENCY_SCOPE_FUNCTION,
							KeyExpressionHash: "limiting_hash",
							EvaluatedKeyHash:  "limiting_eval",
						},
					},
				},
				ExhaustedConstraints: []*pb.ConstraintItem{
					{
						Kind: pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_RATE_LIMIT,
						RateLimit: &pb.RateLimitConstraint{
							Scope:             pb.ConstraintApiRateLimitScope_CONSTRAINT_API_RATE_LIMIT_SCOPE_FUNCTION,
							KeyExpressionHash: "rate-limit-key",
							EvaluatedKeyHash:  "eval-hash",
						},
					},
				},
				RetryAfter: timestamppb.New(retryAfter),
			},
		},
		{
			name: "response without leases",
			input: &CapacityAcquireResponse{
				Leases:               []CapacityLease{},
				LimitingConstraints:  []ConstraintItem{},
				ExhaustedConstraints: []ConstraintItem{},
				RetryAfter:           time.Time{},
			},
			expected: &pb.CapacityAcquireResponse{
				Leases:               []*pb.CapacityLease{},
				LimitingConstraints:  []*pb.ConstraintItem{},
				ExhaustedConstraints: []*pb.ConstraintItem{},
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
			result := CapacityAcquireResponseToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			if tt.input != nil {
				backConverted, err := CapacityAcquireResponseFromProto(result)
				require.NoError(t, err)
				assert.Equal(t, tt.input, backConverted)
			}
		})
	}

	// Test invalid ULID in lease
	t.Run("invalid lease ID in lease", func(t *testing.T) {
		pbResp := &pb.CapacityAcquireResponse{
			Leases: []*pb.CapacityLease{
				{
					LeaseId:        "invalid-ulid",
					IdempotencyKey: "test-key",
				},
			},
		}
		_, err := CapacityAcquireResponseFromProto(pbResp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid lease at index 0")
	})

	// Test nil protobuf
	t.Run("nil protobuf", func(t *testing.T) {
		result, err := CapacityAcquireResponseFromProto(nil)
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
				Source: LeaseSource{
					Service:           ServiceAPI,
					Location:          CallerLocationItemLease,
					RunProcessingMode: RunProcessingModeBackground,
				},
			},
			expected: &pb.CapacityExtendLeaseRequest{
				IdempotencyKey: "extend-key",
				AccountId:      accountID.String(),
				LeaseId:        leaseID.String(),
				Duration:       durationpb.New(15 * time.Minute),
				Source: &pb.LeaseSource{
					Service:           pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_API,
					Location:          pb.ConstraintApiCallerLocation_CONSTRAINT_API_CALLER_LOCATION_ITEM_LEASE,
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

func TestCapacityReleaseRequestConversion(t *testing.T) {
	accountID := uuid.New()
	leaseID := ulid.Make()

	tests := []struct {
		name     string
		input    *CapacityReleaseRequest
		expected *pb.CapacityReleaseRequest
	}{
		{
			name: "valid request",
			input: &CapacityReleaseRequest{
				IdempotencyKey: "commit-key",
				AccountID:      accountID,
				LeaseID:        leaseID,
				Source: LeaseSource{
					Service:           ServiceAPI,
					Location:          CallerLocationItemLease,
					RunProcessingMode: RunProcessingModeBackground,
				},
			},
			expected: &pb.CapacityReleaseRequest{
				IdempotencyKey: "commit-key",
				AccountId:      accountID.String(),
				LeaseId:        leaseID.String(),
				Source: &pb.LeaseSource{
					Service:           pb.ConstraintApiLeaseService_CONSTRAINT_API_LEASE_SERVICE_API,
					Location:          pb.ConstraintApiCallerLocation_CONSTRAINT_API_CALLER_LOCATION_ITEM_LEASE,
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
			result := CapacityReleaseRequestToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			if tt.input != nil {
				backConverted, err := CapacityReleaseRequestFromProto(result)
				require.NoError(t, err)
				assert.Equal(t, tt.input, backConverted)
			}
		})
	}
}

func TestCapacityReleaseResponseConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    *CapacityReleaseResponse
		expected *pb.CapacityReleaseResponse
	}{
		{
			name:     "valid response",
			input:    &CapacityReleaseResponse{},
			expected: &pb.CapacityReleaseResponse{},
		},
		{
			name:     "nil response",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapacityReleaseResponseToProto(tt.input)
			assert.Equal(t, tt.expected, result)

			// Test round trip
			backConverted := CapacityReleaseResponseFromProto(result)
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
					Period:            3600,
					KeyExpressionHash: "hash123",
				},
			},
			Concurrency: ConcurrencyConfig{
				AccountConcurrency:     50,
				FunctionConcurrency:    25,
				AccountRunConcurrency:  10,
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
					Scope:             2, // ThrottleScopeAccount
					KeyExpressionHash: "throttle_hash",
					Limit:             1000,
					Burst:             200,
					Period:            300,
				},
			},
		}

		// Convert to protobuf and back
		pbConfig := ConstraintConfigToProto(original)
		result := ConstraintConfigFromProto(pbConfig)

		assert.Equal(t, original, result)
	})

	t.Run("CapacityAcquireRequest round trip", func(t *testing.T) {
		accountID := uuid.New()
		envID := uuid.New()
		functionID := uuid.New()
		currentTime := time.Date(2023, 10, 15, 12, 30, 45, 123456789, time.UTC)
		kindConcurrency := ConstraintKindConcurrency

		original := &CapacityAcquireRequest{
			IdempotencyKey: "test-key-123",
			AccountID:      accountID,
			EnvID:          envID,
			FunctionID:     functionID,
			Configuration: ConstraintConfig{
				FunctionVersion: 1,
				RateLimit:       []RateLimitConfig{},
				Concurrency: ConcurrencyConfig{
					CustomConcurrencyKeys: []CustomConcurrencyLimit{},
				},
				Throttle: []ThrottleConfig{},
			},
			Constraints: []ConstraintItem{
				{
					Kind: kindConcurrency,
				},
			},
			Amount:            3,
			CurrentTime:       currentTime,
			LeaseRunIDs:       map[string]ulid.ULID{},
			Duration:          5 * time.Minute,
			MaximumLifetime:   30 * time.Minute,
			BlockingThreshold: 10 * time.Second,
			Source: LeaseSource{
				Service:           ServiceExecutor,
				Location:          CallerLocationItemLease,
				RunProcessingMode: RunProcessingModeBackground,
			},
		}

		// Convert to protobuf and back
		pbRequest := CapacityAcquireRequestToProto(original)
		result, err := CapacityAcquireRequestFromProto(pbRequest)
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
		item := ConstraintItem{
			Kind: ConstraintKind(""),
		}

		pbItem := ConstraintItemToProto(item)
		assert.Equal(t, pb.ConstraintApiConstraintKind_CONSTRAINT_API_CONSTRAINT_KIND_UNSPECIFIED, pbItem.Kind)

		// From protobuf unspecified becomes empty
		result := ConstraintItemFromProto(pbItem)
		assert.Empty(t, result.Kind)
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
