package constraintapi

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializedConstraintItem(t *testing.T) {
	testConfig := ConstraintConfig{
		FunctionVersion: 1,
		RateLimit: []RateLimitConfig{
			{
				Scope:             enums.RateLimitScopeAccount,
				Limit:             100,
				Period:            "1m",
				KeyExpressionHash: "test-key-hash",
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
					KeyExpressionHash: "custom-key",
				},
			},
		},
		Throttle: []ThrottleConfig{
			{
				Scope:                     enums.ThrottleScopeFn,
				Limit:                     200,
				Burst:                     300,
				Period:                    60,
				ThrottleKeyExpressionHash: "throttle-key",
			},
		},
	}

	tests := []struct {
		name     string
		input    ConstraintItem
		expected string
	}{
		{
			name: "RateLimit constraint with embedded config",
			input: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeAccount,
					KeyExpressionHash: "test-key-hash",
					EvaluatedKeyHash:  "eval-hash",
				},
			},
			expected: `{"k":1,"r":{"s":2,"h":"test-key-hash","eh":"eval-hash","l":100,"p":"1m"}}`,
		},
		{
			name: "Concurrency constraint with custom key",
			input: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeRun,
					Scope:             enums.ConcurrencyScopeEnv,
					KeyExpressionHash: "custom-key",
					EvaluatedKeyHash:  "concurrency-eval",
				},
			},
			expected: `{"k":2,"c":{"m":1,"s":1,"h":"custom-key","eh":"concurrency-eval","l":15}}`,
		},
		{
			name: "Throttle constraint with embedded config",
			input: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeFn,
					KeyExpressionHash: "throttle-key",
					EvaluatedKeyHash:  "throttle-eval",
				},
			},
			expected: `{"k":3,"t":{"h":"throttle-key","eh":"throttle-eval","l":200,"b":300,"p":60}}`,
		},
		{
			name: "Concurrency constraint with standard function step limit",
			input: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
					// KeyExpressionHash and EvaluatedKeyHash left empty for standard limit
				},
			},
			expected: `{"k":2,"c":{"l":25}}`, // Function concurrency limit embedded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized := tt.input.ToSerializedConstraintItem(testConfig)
			jsonBytes, err := json.Marshal(serialized)
			require.NoError(t, err)
			
			assert.JSONEq(t, tt.expected, string(jsonBytes))
		})
	}
}

func TestSerializedConstraintItem_SizeReduction(t *testing.T) {
	// Test that serialized version is significantly smaller
	testConfig := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			AccountConcurrency: 50,
		},
	}

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

	// Serialize optimized version with embedded config
	serialized := original.ToSerializedConstraintItem(testConfig)
	optimizedJson, err := json.Marshal(serialized)
	require.NoError(t, err)

	t.Logf("Original JSON (%d bytes): %s", len(originalJson), string(originalJson))
	t.Logf("Optimized JSON (%d bytes): %s", len(optimizedJson), string(optimizedJson))

	// The optimized version should be significantly smaller
	assert.Less(t, len(optimizedJson), len(originalJson))
}


