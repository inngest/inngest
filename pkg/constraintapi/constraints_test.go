package constraintapi

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/inngest/inngest/pkg/enums"
)

func TestConcurrencyConstraint_InProgressLeasesKey(t *testing.T) {
	// Test UUIDs
	accountID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	envID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440002")
	functionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440003")
	prefix := "test-prefix"

	tests := []struct {
		name        string
		constraint  ConcurrencyConstraint
		prefix      string
		accountID   uuid.UUID
		envID       uuid.UUID
		functionID  uuid.UUID
		expected    string
		description string
	}{
		// Basic Scope Testing
		{
			name: "account scope",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeAccount,
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:a:550e8400-e29b-41d4-a716-446655440001",
			description: "should use account scope ID 'a' and accountID as entityID",
		},
		{
			name: "environment scope",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeEnv,
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:e:550e8400-e29b-41d4-a716-446655440002",
			description: "should use environment scope ID 'e' and envID as entityID",
		},
		{
			name: "function scope",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003",
			description: "should use function scope ID 'f' and functionID as entityID",
		},

		// Mode Testing (should not affect key generation)
		{
			name: "step mode with function scope",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003",
			description: "step mode should generate same key format as other modes",
		},
		{
			name: "run mode with function scope",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeRun,
				Scope: enums.ConcurrencyScopeFn,
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003",
			description: "run mode should generate same key format as other modes",
		},

		// Key Expression Hash Testing
		{
			name: "no custom key hash",
			constraint: ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeFn,
				KeyExpressionHash: "",
				EvaluatedKeyHash:  "",
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003",
			description: "empty KeyExpressionHash should not append keyID suffix",
		},
		{
			name: "with custom key hash",
			constraint: ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeFn,
				KeyExpressionHash: "expr_hash_123",
				EvaluatedKeyHash:  "eval_hash_456",
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003<expr_hash_123:eval_hash_456>",
			description: "non-empty KeyExpressionHash should append keyID suffix with format <hash:evaluated>",
		},
		{
			name: "expression hash without evaluated hash",
			constraint: ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeFn,
				KeyExpressionHash: "expr_hash_789",
				EvaluatedKeyHash:  "",
			},
			prefix:      prefix,
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{test-prefix}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003<expr_hash_789:>",
			description: "KeyExpressionHash with empty EvaluatedKeyHash should still include format",
		},

		// Parameter Validation Testing
		{
			name: "empty prefix",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
			prefix:      "",
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003",
			description: "empty prefix should still generate valid key format",
		},
		{
			name: "different prefix",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
			prefix:      "production",
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{production}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:f:550e8400-e29b-41d4-a716-446655440003",
			description: "different prefix should be reflected in generated key",
		},
		{
			name: "zero UUIDs",
			constraint: ConcurrencyConstraint{
				Mode:  enums.ConcurrencyModeStep,
				Scope: enums.ConcurrencyScopeFn,
			},
			prefix:      prefix,
			accountID:   uuid.Nil,
			envID:       uuid.Nil,
			functionID:  uuid.Nil,
			expected:    "{test-prefix}:00000000-0000-0000-0000-000000000000:state:concurrency:f:00000000-0000-0000-0000-000000000000",
			description: "nil UUIDs should be formatted as zero UUIDs",
		},

		// Integration Testing - Complex Combinations
		{
			name: "account scope with custom key and run mode",
			constraint: ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeRun,
				Scope:             enums.ConcurrencyScopeAccount,
				KeyExpressionHash: "account_key",
				EvaluatedKeyHash:  "account_eval",
			},
			prefix:      "prod-redis",
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{prod-redis}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:a:550e8400-e29b-41d4-a716-446655440001<account_key:account_eval>",
			description: "complex combination should work correctly with all parameters",
		},
		{
			name: "environment scope with custom key",
			constraint: ConcurrencyConstraint{
				Mode:              enums.ConcurrencyModeStep,
				Scope:             enums.ConcurrencyScopeEnv,
				KeyExpressionHash: "env_custom",
				EvaluatedKeyHash:  "env_value",
			},
			prefix:      "staging",
			accountID:   accountID,
			envID:       envID,
			functionID:  functionID,
			expected:    "{staging}:550e8400-e29b-41d4-a716-446655440001:state:concurrency:e:550e8400-e29b-41d4-a716-446655440002<env_custom:env_value>",
			description: "environment scope with custom keys should generate correct format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.constraint.InProgressLeasesKey(tt.prefix, tt.accountID, tt.envID, tt.functionID)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestConcurrencyConstraint_InProgressLeasesKey_KeyFormat(t *testing.T) {
	// Additional tests to verify key format consistency
	constraint := ConcurrencyConstraint{
		Mode:  enums.ConcurrencyModeStep,
		Scope: enums.ConcurrencyScopeFn,
	}

	accountID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	envID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	functionID := uuid.MustParse("ffffffff-1111-2222-3333-444444444444")

	t.Run("key format validation", func(t *testing.T) {
		key := constraint.InProgressLeasesKey("prefix", accountID, envID, functionID)

		// Verify the key follows expected pattern: prefix:accountID:state:concurrency:scopeID:entityID[keyID]
		assert.Contains(t, key, "{prefix}:")
		assert.Contains(t, key, ":11111111-2222-3333-4444-555555555555:")
		assert.Contains(t, key, ":state:concurrency:")
		assert.Contains(t, key, ":f:")
		assert.Contains(t, key, ":ffffffff-1111-2222-3333-444444444444")
	})

	t.Run("key uniqueness", func(t *testing.T) {
		// Different scopes should produce different keys
		constraintAccount := ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeAccount}
		constraintEnv := ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeEnv}
		constraintFn := ConcurrencyConstraint{Mode: enums.ConcurrencyModeStep, Scope: enums.ConcurrencyScopeFn}

		keyAccount := constraintAccount.InProgressLeasesKey("test", accountID, envID, functionID)
		keyEnv := constraintEnv.InProgressLeasesKey("test", accountID, envID, functionID)
		keyFn := constraintFn.InProgressLeasesKey("test", accountID, envID, functionID)

		assert.NotEqual(t, keyAccount, keyEnv, "account and environment scoped keys should be different")
		assert.NotEqual(t, keyEnv, keyFn, "environment and function scoped keys should be different")
		assert.NotEqual(t, keyAccount, keyFn, "account and function scoped keys should be different")
	})
}

