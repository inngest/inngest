package constraintapi

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializedConstraintItem(t *testing.T) {
	tests := []struct {
		name     string
		input    ConstraintItem
		expected string
	}{
		{
			name: "RateLimit constraint",
			input: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeAccount,
					KeyExpressionHash: "test-key-hash",
					EvaluatedKeyHash:  "eval-hash",
				},
			},
			expected: `{"k":1,"r":{"s":2,"h":"test-key-hash","eh":"eval-hash"}}`,
		},
		{
			name: "Concurrency constraint",
			input: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeRun,
					Scope:             enums.ConcurrencyScopeEnv,
					KeyExpressionHash: "concurrency-key",
					EvaluatedKeyHash:  "concurrency-eval",
				},
			},
			expected: `{"k":2,"c":{"m":1,"s":1,"h":"concurrency-key","eh":"concurrency-eval"}}`,
		},
		{
			name: "Throttle constraint",
			input: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeFn,
					KeyExpressionHash: "throttle-key",
					EvaluatedKeyHash:  "throttle-eval",
				},
			},
			expected: `{"k":3,"t":{"h":"throttle-key","eh":"throttle-eval"}}`, // s:0 omitted due to omitempty
		},
		{
			name: "Concurrency constraint with empty fields",
			input: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
					// KeyExpressionHash and EvaluatedKeyHash left empty
				},
			},
			expected: `{"k":2,"c":{}}`, // Empty fields should be omitted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized := tt.input.ToSerializedConstraintItem()
			jsonBytes, err := json.Marshal(serialized)
			require.NoError(t, err)
			
			assert.JSONEq(t, tt.expected, string(jsonBytes))
		})
	}
}

func TestSerializedConstraintItem_SizeReduction(t *testing.T) {
	// Test that serialized version is significantly smaller
	original := ConstraintItem{
		Kind: ConstraintKindConcurrency,
		Concurrency: &ConcurrencyConstraint{
			Mode:              enums.ConcurrencyModeRun,
			Scope:             enums.ConcurrencyScopeAccount,
			KeyExpressionHash: "some-very-long-key-expression-hash-value",
			EvaluatedKeyHash:  "some-very-long-evaluated-key-hash-value",
		},
	}

	// Serialize original
	originalJson, err := json.Marshal(original)
	require.NoError(t, err)

	// Serialize optimized version
	serialized := original.ToSerializedConstraintItem()
	optimizedJson, err := json.Marshal(serialized)
	require.NoError(t, err)

	t.Logf("Original JSON (%d bytes): %s", len(originalJson), string(originalJson))
	t.Logf("Optimized JSON (%d bytes): %s", len(optimizedJson), string(optimizedJson))

	// The optimized version should be significantly smaller
	assert.Less(t, len(optimizedJson), len(originalJson))
}

func TestSerializedConstraintConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    ConstraintConfig
		expected string
	}{
		{
			name: "Complete ConstraintConfig",
			input: ConstraintConfig{
				FunctionVersion: 42,
				RateLimit: []RateLimitConfig{
					{
						Scope:             enums.RateLimitScopeAccount,
						Limit:             100,
						Period:            "1m",
						KeyExpressionHash: "rate-key-hash",
					},
				},
				Concurrency: ConcurrencyConfig{
					AccountConcurrency:     50,
					FunctionConcurrency:    25,
					AccountRunConcurrency:  10,
					FunctionRunConcurrency: 5,
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:              enums.ConcurrencyModeRun,
							Scope:             enums.ConcurrencyScopeEnv,
							Limit:             15,
							KeyExpressionHash: "custom-key-hash",
						},
					},
				},
				Throttle: []ThrottleConfig{
					{
						Scope:                     enums.ThrottleScopeAccount,
						Limit:                     200,
						Burst:                     300,
						Period:                    60,
						ThrottleKeyExpressionHash: "throttle-key-hash",
					},
				},
			},
			expected: `{"v":42,"r":[{"s":2,"l":100,"p":"1m","h":"rate-key-hash"}],"c":{"ac":50,"fc":25,"arc":10,"frc":5,"cck":[{"m":1,"s":1,"l":15,"h":"custom-key-hash"}]},"t":[{"s":2,"l":200,"b":300,"p":60,"h":"throttle-key-hash"}]}`,
		},
		{
			name: "RateLimitConfig only",
			input: ConstraintConfig{
				FunctionVersion: 1,
				RateLimit: []RateLimitConfig{
					{
						Scope:             enums.RateLimitScopeFn,
						Limit:             50,
						Period:            "30s",
						KeyExpressionHash: "fn-rate-key",
					},
				},
			},
			expected: `{"v":1,"r":[{"l":50,"p":"30s","h":"fn-rate-key"}],"c":{}}`, // s:0 omitted due to omitempty
		},
		{
			name: "ConcurrencyConfig only",
			input: ConstraintConfig{
				FunctionVersion: 2,
				Concurrency: ConcurrencyConfig{
					AccountConcurrency:  100,
					FunctionConcurrency: 20,
				},
			},
			expected: `{"v":2,"c":{"ac":100,"fc":20}}`,
		},
		{
			name: "ThrottleConfig only",
			input: ConstraintConfig{
				FunctionVersion: 3,
				Throttle: []ThrottleConfig{
					{
						Scope:  enums.ThrottleScopeFn,
						Limit:  10,
						Burst:  20,
						Period: 30,
					},
				},
			},
			expected: `{"v":3,"c":{},"t":[{"l":10,"b":20,"p":30}]}`, // s:0 and h:"" omitted due to omitempty
		},
		{
			name: "Empty ConstraintConfig",
			input: ConstraintConfig{
				FunctionVersion: 0,
			},
			expected: `{"c":{}}`, // v:0 omitted due to omitempty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized := tt.input.ToSerializedConstraintConfig()
			jsonBytes, err := json.Marshal(serialized)
			require.NoError(t, err)

			assert.JSONEq(t, tt.expected, string(jsonBytes))
		})
	}
}

func TestSerializedConstraintConfig_SizeReduction(t *testing.T) {
	// Test that serialized version is significantly smaller
	original := ConstraintConfig{
		FunctionVersion: 42,
		RateLimit: []RateLimitConfig{
			{
				Scope:             enums.RateLimitScopeAccount,
				Limit:             100,
				Period:            "1m",
				KeyExpressionHash: "some-very-long-rate-limit-key-expression-hash",
			},
			{
				Scope:             enums.RateLimitScopeEnv,
				Limit:             200,
				Period:            "5m",
				KeyExpressionHash: "another-very-long-rate-limit-key-hash",
			},
		},
		Concurrency: ConcurrencyConfig{
			AccountConcurrency:     50,
			FunctionConcurrency:    25,
			AccountRunConcurrency:  10,
			FunctionRunConcurrency: 5,
			CustomConcurrencyKeys: []CustomConcurrencyLimit{
				{
					Mode:              enums.ConcurrencyModeRun,
					Scope:             enums.ConcurrencyScopeEnv,
					Limit:             15,
					KeyExpressionHash: "very-long-custom-concurrency-key-expression-hash",
				},
				{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeAccount,
					Limit:             20,
					KeyExpressionHash: "another-very-long-custom-concurrency-key-hash",
				},
			},
		},
		Throttle: []ThrottleConfig{
			{
				Scope:                     enums.ThrottleScopeAccount,
				Limit:                     200,
				Burst:                     300,
				Period:                    60,
				ThrottleKeyExpressionHash: "very-long-throttle-key-expression-hash-value",
			},
		},
	}

	// Serialize original
	originalJson, err := json.Marshal(original)
	require.NoError(t, err)

	// Serialize optimized version
	serialized := original.ToSerializedConstraintConfig()
	optimizedJson, err := json.Marshal(serialized)
	require.NoError(t, err)

	t.Logf("Original JSON (%d bytes): %s", len(originalJson), string(originalJson))
	t.Logf("Optimized JSON (%d bytes): %s", len(optimizedJson), string(optimizedJson))

	// The optimized version should be significantly smaller
	assert.Less(t, len(optimizedJson), len(originalJson))

	// Should achieve at least 30% size reduction
	reductionPercentage := float64(len(originalJson)-len(optimizedJson)) / float64(len(originalJson)) * 100
	t.Logf("Size reduction: %.1f%%", reductionPercentage)
	assert.Greater(t, reductionPercentage, 30.0)
}