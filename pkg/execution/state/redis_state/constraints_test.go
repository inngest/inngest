package redis_state

import (
	"testing"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestConstraintConfigFromConstraints(t *testing.T) {
	tests := []struct {
		name        string
		constraints PartitionConstraintConfig
		expected    constraintapi.ConstraintConfig
	}{
		{
			name:        "empty constraints",
			constraints: PartitionConstraintConfig{},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 0,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:     0,
					FunctionConcurrency:    0,
					AccountRunConcurrency:  0,
					FunctionRunConcurrency: 0,
				},
			},
		},
		{
			name: "basic concurrency limits",
			constraints: PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: PartitionConcurrency{
					AccountConcurrency:     100,
					FunctionConcurrency:    10,
					AccountRunConcurrency:  50,
					FunctionRunConcurrency: 5,
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:     100,
					FunctionConcurrency:    10,
					AccountRunConcurrency:  50,
					FunctionRunConcurrency: 5,
				},
			},
		},
		{
			name: "with custom concurrency keys",
			constraints: PartitionConstraintConfig{
				FunctionVersion: 2,
				Concurrency: PartitionConcurrency{
					AccountConcurrency:  100,
					FunctionConcurrency: 10,
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               5,
							HashedKeyExpression: "key1-hash",
						},
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeFn,
							Limit:               3,
							HashedKeyExpression: "key2-hash",
						},
					},
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 2,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:  100,
					FunctionConcurrency: 10,
					CustomConcurrencyKeys: []constraintapi.CustomConcurrencyLimit{
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeAccount,
							Limit:             5,
							KeyExpressionHash: "key1-hash",
						},
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeFn,
							Limit:             3,
							KeyExpressionHash: "key2-hash",
						},
					},
				},
			},
		},
		{
			name: "with throttle",
			constraints: PartitionConstraintConfig{
				FunctionVersion: 1,
				Throttle: &PartitionThrottle{
					Limit:                     10,
					Burst:                     5,
					Period:                    60,
					ThrottleKeyExpressionHash: "throttle-hash",
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:     0,
					FunctionConcurrency:    0,
					AccountRunConcurrency:  0,
					FunctionRunConcurrency: 0,
				},
				Throttle: []constraintapi.ThrottleConfig{
					{
						Limit:                     10,
						Burst:                     5,
						Period:                    60,
						ThrottleKeyExpressionHash: "throttle-hash",
					},
				},
			},
		},
		{
			name: "complete configuration",
			constraints: PartitionConstraintConfig{
				FunctionVersion: 3,
				Concurrency: PartitionConcurrency{
					AccountConcurrency:     200,
					FunctionConcurrency:    20,
					AccountRunConcurrency:  100,
					FunctionRunConcurrency: 10,
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               15,
							HashedKeyExpression: "custom-key-hash",
						},
					},
				},
				Throttle: &PartitionThrottle{
					Limit:                     20,
					Burst:                     10,
					Period:                    30,
					ThrottleKeyExpressionHash: "complete-throttle-hash",
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 3,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:     200,
					FunctionConcurrency:    20,
					AccountRunConcurrency:  100,
					FunctionRunConcurrency: 10,
					CustomConcurrencyKeys: []constraintapi.CustomConcurrencyLimit{
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeAccount,
							Limit:             15,
							KeyExpressionHash: "custom-key-hash",
						},
					},
				},
				Throttle: []constraintapi.ThrottleConfig{
					{
						Limit:                     20,
						Burst:                     10,
						Period:                    30,
						ThrottleKeyExpressionHash: "complete-throttle-hash",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constraintConfigFromConstraints(tt.constraints)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConstraintItemsFromBacklog(t *testing.T) {
	tests := []struct {
		name     string
		backlog  *QueueBacklog
		expected []constraintapi.ConstraintItem
	}{
		{
			name:    "minimal backlog",
			backlog: &QueueBacklog{},
			expected: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeAccount,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeFn,
					},
				},
			},
		},
		{
			name: "with throttle",
			backlog: &QueueBacklog{
				Throttle: &BacklogThrottle{
					ThrottleKeyExpressionHash: "throttle-expr-hash",
					ThrottleKey:               "throttle-key-value",
				},
			},
			expected: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeAccount,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeFn,
					},
				},
				{
					Kind: constraintapi.ConstraintKindThrottle,
					Throttle: &constraintapi.ThrottleConstraint{
						KeyExpressionHash: "throttle-expr-hash",
						EvaluatedKeyHash:  "throttle-key-value",
					},
				},
			},
		},
		{
			name: "with custom concurrency keys",
			backlog: &QueueBacklog{
				ConcurrencyKeys: []BacklogConcurrencyKey{
					{
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeAccount,
						HashedKeyExpression: "custom-key-1-hash",
						HashedValue:         "custom-key-1-value",
					},
					{
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeFn,
						HashedKeyExpression: "custom-key-2-hash",
						HashedValue:         "custom-key-2-value",
					},
				},
			},
			expected: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeAccount,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeFn,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "custom-key-1-hash",
						EvaluatedKeyHash:  "custom-key-1-value",
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeFn,
						KeyExpressionHash: "custom-key-2-hash",
						EvaluatedKeyHash:  "custom-key-2-value",
					},
				},
			},
		},
		{
			name: "complete backlog with throttle and concurrency keys",
			backlog: &QueueBacklog{
				Throttle: &BacklogThrottle{
					ThrottleKeyExpressionHash: "complete-throttle-hash",
					ThrottleKey:               "complete-throttle-value",
				},
				ConcurrencyKeys: []BacklogConcurrencyKey{
					{
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeEnv,
						HashedKeyExpression: "complete-key-hash",
						HashedValue:         "complete-key-value",
					},
				},
			},
			expected: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeAccount,
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:  enums.ConcurrencyModeStep,
						Scope: enums.ConcurrencyScopeFn,
					},
				},
				{
					Kind: constraintapi.ConstraintKindThrottle,
					Throttle: &constraintapi.ThrottleConstraint{
						KeyExpressionHash: "complete-throttle-hash",
						EvaluatedKeyHash:  "complete-throttle-value",
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeEnv,
						KeyExpressionHash: "complete-key-hash",
						EvaluatedKeyHash:  "complete-key-value",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constraintItemsFromBacklog(tt.backlog)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertLimitingConstraint(t *testing.T) {
	tests := []struct {
		name                string
		constraints         PartitionConstraintConfig
		limitingConstraints []constraintapi.ConstraintItem
		expected            enums.QueueConstraint
	}{
		{
			name:                "no limiting constraints",
			constraints:         PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{},
			expected:            enums.QueueConstraintNotLimited,
		},
		{
			name:        "account concurrency constraint",
			constraints: PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: util.XXHash(""),
					},
				},
			},
			expected: enums.QueueConstraintAccountConcurrency,
		},
		{
			name:        "function concurrency constraint",
			constraints: PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeFn,
						KeyExpressionHash: util.XXHash(""),
					},
				},
			},
			expected: enums.QueueConstraintFunctionConcurrency,
		},
		{
			name: "custom concurrency key 1",
			constraints: PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							HashedKeyExpression: "custom-key-1",
						},
					},
				},
			},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "custom-key-1",
					},
				},
			},
			expected: enums.QueueConstraintCustomConcurrencyKey1,
		},
		{
			name: "custom concurrency key 2",
			constraints: PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							HashedKeyExpression: "custom-key-1",
						},
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeFn,
							HashedKeyExpression: "custom-key-2",
						},
					},
				},
			},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeFn,
						KeyExpressionHash: "custom-key-2",
					},
				},
			},
			expected: enums.QueueConstraintCustomConcurrencyKey2,
		},
		{
			name:        "throttle constraint",
			constraints: PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindThrottle,
				},
			},
			expected: enums.QueueConstraintThrottle,
		},
		{
			name:        "multiple constraints - last one wins",
			constraints: PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: util.XXHash(""),
					},
				},
				{
					Kind: constraintapi.ConstraintKindThrottle,
				},
			},
			expected: enums.QueueConstraintThrottle,
		},
		{
			name:        "unknown constraint kind",
			constraints: PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: "unknown-kind",
				},
			},
			expected: enums.QueueConstraintNotLimited,
		},
		{
			name: "custom concurrency key without matching configuration",
			constraints: PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							HashedKeyExpression: "different-key",
						},
					},
				},
			},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "non-matching-key",
					},
				},
			},
			expected: enums.QueueConstraintNotLimited,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLimitingConstraint(tt.constraints, tt.limitingConstraints)
			assert.Equal(t, tt.expected, result)
		})
	}
}
