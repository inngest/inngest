package constraintapi

import (
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

func TestSortConstraints(t *testing.T) {
	t.Run("Constraint Kind Priority", func(t *testing.T) {
		tests := []struct {
			name     string
			input    []ConstraintItem
			expected []ConstraintKind
		}{
			{
				name: "Rate limit comes before throttle",
				input: []ConstraintItem{
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn}},
				},
				expected: []ConstraintKind{ConstraintKindRateLimit, ConstraintKindThrottle},
			},
			{
				name: "Rate limit comes before concurrency",
				input: []ConstraintItem{
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn}},
				},
				expected: []ConstraintKind{ConstraintKindRateLimit, ConstraintKindConcurrency},
			},
			{
				name: "Throttle comes before concurrency",
				input: []ConstraintItem{
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn}},
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn}},
				},
				expected: []ConstraintKind{ConstraintKindThrottle, ConstraintKindConcurrency},
			},
			{
				name: "All three kinds in reverse order",
				input: []ConstraintItem{
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn}},
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn}},
				},
				expected: []ConstraintKind{ConstraintKindRateLimit, ConstraintKindThrottle, ConstraintKindConcurrency},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				constraints := make([]ConstraintItem, len(tt.input))
				copy(constraints, tt.input)
				
				sortConstraints(constraints)
				
				require.Len(t, constraints, len(tt.expected))
				for i, expected := range tt.expected {
					require.Equal(t, expected, constraints[i].Kind, "position %d", i)
				}
			})
		}
	})

	t.Run("Rate Limit Scope Priority", func(t *testing.T) {
		tests := []struct {
			name     string
			input    []ConstraintItem
			expected []enums.RateLimitScope
		}{
			{
				name: "Account comes before environment",
				input: []ConstraintItem{
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeEnv}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeAccount}},
				},
				expected: []enums.RateLimitScope{enums.RateLimitScopeAccount, enums.RateLimitScopeEnv},
			},
			{
				name: "Account comes before function",
				input: []ConstraintItem{
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeAccount}},
				},
				expected: []enums.RateLimitScope{enums.RateLimitScopeAccount, enums.RateLimitScopeFn},
			},
			{
				name: "Environment comes before function",
				input: []ConstraintItem{
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeEnv}},
				},
				expected: []enums.RateLimitScope{enums.RateLimitScopeEnv, enums.RateLimitScopeFn},
			},
			{
				name: "All scopes in reverse order",
				input: []ConstraintItem{
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeEnv}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeAccount}},
				},
				expected: []enums.RateLimitScope{enums.RateLimitScopeAccount, enums.RateLimitScopeEnv, enums.RateLimitScopeFn},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				constraints := make([]ConstraintItem, len(tt.input))
				copy(constraints, tt.input)
				
				sortConstraints(constraints)
				
				require.Len(t, constraints, len(tt.expected))
				for i, expected := range tt.expected {
					require.Equal(t, expected, constraints[i].RateLimit.Scope, "position %d", i)
				}
			})
		}
	})

	t.Run("Throttle Scope Priority", func(t *testing.T) {
		tests := []struct {
			name     string
			input    []ConstraintItem
			expected []enums.ThrottleScope
		}{
			{
				name: "Account comes before environment",
				input: []ConstraintItem{
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeEnv}},
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeAccount}},
				},
				expected: []enums.ThrottleScope{enums.ThrottleScopeAccount, enums.ThrottleScopeEnv},
			},
			{
				name: "Account comes before function",
				input: []ConstraintItem{
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn}},
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeAccount}},
				},
				expected: []enums.ThrottleScope{enums.ThrottleScopeAccount, enums.ThrottleScopeFn},
			},
			{
				name: "Environment comes before function",
				input: []ConstraintItem{
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn}},
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeEnv}},
				},
				expected: []enums.ThrottleScope{enums.ThrottleScopeEnv, enums.ThrottleScopeFn},
			},
			{
				name: "All scopes in reverse order",
				input: []ConstraintItem{
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn}},
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeEnv}},
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeAccount}},
				},
				expected: []enums.ThrottleScope{enums.ThrottleScopeAccount, enums.ThrottleScopeEnv, enums.ThrottleScopeFn},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				constraints := make([]ConstraintItem, len(tt.input))
				copy(constraints, tt.input)
				
				sortConstraints(constraints)
				
				require.Len(t, constraints, len(tt.expected))
				for i, expected := range tt.expected {
					require.Equal(t, expected, constraints[i].Throttle.Scope, "position %d", i)
				}
			})
		}
	})

	t.Run("Concurrency Scope Priority", func(t *testing.T) {
		tests := []struct {
			name     string
			input    []ConstraintItem
			expected []enums.ConcurrencyScope
		}{
			{
				name: "Account comes before environment",
				input: []ConstraintItem{
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeEnv}},
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount}},
				},
				expected: []enums.ConcurrencyScope{enums.ConcurrencyScopeAccount, enums.ConcurrencyScopeEnv},
			},
			{
				name: "Account comes before function",
				input: []ConstraintItem{
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn}},
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount}},
				},
				expected: []enums.ConcurrencyScope{enums.ConcurrencyScopeAccount, enums.ConcurrencyScopeFn},
			},
			{
				name: "Environment comes before function",
				input: []ConstraintItem{
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn}},
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeEnv}},
				},
				expected: []enums.ConcurrencyScope{enums.ConcurrencyScopeEnv, enums.ConcurrencyScopeFn},
			},
			{
				name: "All scopes in reverse order",
				input: []ConstraintItem{
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn}},
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeEnv}},
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount}},
				},
				expected: []enums.ConcurrencyScope{enums.ConcurrencyScopeAccount, enums.ConcurrencyScopeEnv, enums.ConcurrencyScopeFn},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				constraints := make([]ConstraintItem, len(tt.input))
				copy(constraints, tt.input)
				
				sortConstraints(constraints)
				
				require.Len(t, constraints, len(tt.expected))
				for i, expected := range tt.expected {
					require.Equal(t, expected, constraints[i].Concurrency.Scope, "position %d", i)
				}
			})
		}
	})

	t.Run("Key Expression Hash Priority", func(t *testing.T) {
		tests := []struct {
			name     string
			input    []ConstraintItem
			expected []string
		}{
			{
				name: "Empty hash comes before non-empty hash (rate limit)",
				input: []ConstraintItem{
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "hash1"}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: ""}},
				},
				expected: []string{"", "hash1"},
			},
			{
				name: "Empty hash comes before non-empty hash (throttle)",
				input: []ConstraintItem{
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn, KeyExpressionHash: "hash1"}},
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn, KeyExpressionHash: ""}},
				},
				expected: []string{"", "hash1"},
			},
			{
				name: "Empty hash comes before non-empty hash (concurrency)",
				input: []ConstraintItem{
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn, KeyExpressionHash: "hash1"}},
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn, KeyExpressionHash: ""}},
				},
				expected: []string{"", "hash1"},
			},
			{
				name: "Lexicographic ordering of non-empty hashes",
				input: []ConstraintItem{
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "hash3"}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "hash1"}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "hash2"}},
				},
				expected: []string{"hash1", "hash2", "hash3"},
			},
			{
				name: "Mixed empty and non-empty hashes",
				input: []ConstraintItem{
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "hash2"}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: ""}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "hash1"}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: ""}},
				},
				expected: []string{"", "", "hash1", "hash2"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				constraints := make([]ConstraintItem, len(tt.input))
				copy(constraints, tt.input)
				
				sortConstraints(constraints)
				
				require.Len(t, constraints, len(tt.expected))
				for i, expected := range tt.expected {
					actual := getConstraintKeyExpressionHash(constraints[i])
					require.Equal(t, expected, actual, "position %d", i)
				}
			})
		}
	})

	t.Run("Complex Mixed Scenarios", func(t *testing.T) {
		tests := []struct {
			name     string
			input    []ConstraintItem
			expected []string // Description of expected order
		}{
			{
				name: "Mixed kinds with different scopes",
				input: []ConstraintItem{
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn}},
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeEnv}},
				},
				expected: []string{"RateLimit-Fn-", "Throttle-Env-", "Concurrency-Account-"},
			},
			{
				name: "Same kind, different scopes and hashes",
				input: []ConstraintItem{
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "hash1"}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeAccount, KeyExpressionHash: "hash2"}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeEnv, KeyExpressionHash: ""}},
				},
				expected: []string{"RateLimit-Account-hash2", "RateLimit-Env-", "RateLimit-Fn-hash1"},
			},
			{
				name: "All three constraint types with all three scopes",
				input: []ConstraintItem{
					// Start with reverse order to test comprehensive sorting
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn}},
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeEnv}},
					{Kind: ConstraintKindConcurrency, Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount}},
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn}},
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeEnv}},
					{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeAccount}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeEnv}},
					{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeAccount}},
				},
				expected: []string{
					"RateLimit-Account-", "RateLimit-Env-", "RateLimit-Fn-",
					"Throttle-Account-", "Throttle-Env-", "Throttle-Fn-",
					"Concurrency-Account-", "Concurrency-Env-", "Concurrency-Fn-",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				constraints := make([]ConstraintItem, len(tt.input))
				copy(constraints, tt.input)
				
				sortConstraints(constraints)
				
				require.Len(t, constraints, len(tt.expected))
				for i, expectedDesc := range tt.expected {
					constraint := constraints[i]
					actualDesc := getConstraintDescription(constraint)
					require.Equal(t, expectedDesc, actualDesc, "position %d", i)
				}
			})
		}
	})

	t.Run("Edge Cases", func(t *testing.T) {
		t.Run("Empty slice", func(t *testing.T) {
			var constraints []ConstraintItem
			sortConstraints(constraints)
			require.Empty(t, constraints)
		})

		t.Run("Single item", func(t *testing.T) {
			constraints := []ConstraintItem{
				{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn}},
			}
			sortConstraints(constraints)
			require.Len(t, constraints, 1)
			require.Equal(t, ConstraintKindRateLimit, constraints[0].Kind)
		})

		t.Run("Stable sort behavior - identical constraints", func(t *testing.T) {
			constraints := []ConstraintItem{
				{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "same"}},
				{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "same"}},
				{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn, KeyExpressionHash: "same"}},
			}
			
			// Store original memory addresses to verify stable sort
			originalAddresses := make([]*RateLimitConstraint, len(constraints))
			for i := range constraints {
				originalAddresses[i] = constraints[i].RateLimit
			}
			
			sortConstraints(constraints)
			
			require.Len(t, constraints, 3)
			// Verify that the order remains the same (stable sort)
			for i := range constraints {
				require.Equal(t, originalAddresses[i], constraints[i].RateLimit, "stable sort failed at position %d", i)
			}
		})

		t.Run("Nil constraint pointers", func(t *testing.T) {
			constraints := []ConstraintItem{
				{Kind: ConstraintKindRateLimit, RateLimit: nil}, // Should get default priority
				{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn}},
			}
			
			// Should not panic
			require.NotPanics(t, func() {
				sortConstraints(constraints)
			})
			
			require.Len(t, constraints, 2)
		})

		t.Run("Unknown constraint kinds", func(t *testing.T) {
			constraints := []ConstraintItem{
				{Kind: ConstraintKind("unknown")},
				{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn}},
				{Kind: ConstraintKind("another_unknown")},
			}
			
			sortConstraints(constraints)
			
			require.Len(t, constraints, 3)
			// Known constraints should come before unknown ones
			require.Equal(t, ConstraintKindRateLimit, constraints[0].Kind)
			require.Equal(t, ConstraintKind("unknown"), constraints[1].Kind)
			require.Equal(t, ConstraintKind("another_unknown"), constraints[2].Kind)
		})
	})
}

func TestGetConstraintKindPriority(t *testing.T) {
	tests := []struct {
		kind     ConstraintKind
		expected int
	}{
		{ConstraintKindRateLimit, 1},
		{ConstraintKindThrottle, 2},
		{ConstraintKindConcurrency, 3},
		{ConstraintKind("unknown"), 4},
	}

	for _, tt := range tests {
		actual := getConstraintKindPriority(tt.kind)
		require.Equal(t, tt.expected, actual, "kind %s", tt.kind)
	}
}

func TestGetConstraintScopePriority(t *testing.T) {
	tests := []struct {
		name     string
		constraint ConstraintItem
		expected int
	}{
		{
			name: "RateLimit Account",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeAccount},
			},
			expected: 1,
		},
		{
			name: "RateLimit Environment",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeEnv},
			},
			expected: 2,
		},
		{
			name: "RateLimit Function",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{Scope: enums.RateLimitScopeFn},
			},
			expected: 3,
		},
		{
			name: "Throttle Account",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeAccount},
			},
			expected: 1,
		},
		{
			name: "Throttle Environment",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeEnv},
			},
			expected: 2,
		},
		{
			name: "Throttle Function",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{Scope: enums.ThrottleScopeFn},
			},
			expected: 3,
		},
		{
			name: "Concurrency Account",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount},
			},
			expected: 1,
		},
		{
			name: "Concurrency Environment",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeEnv},
			},
			expected: 2,
		},
		{
			name: "Concurrency Function",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn},
			},
			expected: 3,
		},
		{
			name: "Nil RateLimit",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: nil,
			},
			expected: 4,
		},
		{
			name: "Unknown Kind",
			constraint: ConstraintItem{
				Kind: ConstraintKind("unknown"),
			},
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := getConstraintScopePriority(tt.constraint)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetConstraintKeyExpressionHash(t *testing.T) {
	tests := []struct {
		name     string
		constraint ConstraintItem
		expected string
	}{
		{
			name: "RateLimit with hash",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{KeyExpressionHash: "hash123"},
			},
			expected: "hash123",
		},
		{
			name: "RateLimit empty hash",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{KeyExpressionHash: ""},
			},
			expected: "",
		},
		{
			name: "Throttle with hash",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{KeyExpressionHash: "hash456"},
			},
			expected: "hash456",
		},
		{
			name: "Concurrency with hash",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{KeyExpressionHash: "hash789"},
			},
			expected: "hash789",
		},
		{
			name: "Nil constraint",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: nil,
			},
			expected: "",
		},
		{
			name: "Unknown kind",
			constraint: ConstraintItem{
				Kind: ConstraintKind("unknown"),
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := getConstraintKeyExpressionHash(tt.constraint)
			require.Equal(t, tt.expected, actual)
		})
	}
}

// Helper function for testing - creates a descriptive string for a constraint
func getConstraintDescription(constraint ConstraintItem) string {
	var scope string
	hash := getConstraintKeyExpressionHash(constraint)
	
	switch constraint.Kind {
	case ConstraintKindRateLimit:
		if constraint.RateLimit != nil {
			switch constraint.RateLimit.Scope {
			case enums.RateLimitScopeAccount:
				scope = "Account"
			case enums.RateLimitScopeEnv:
				scope = "Env"
			case enums.RateLimitScopeFn:
				scope = "Fn"
			}
		}
		return "RateLimit-" + scope + "-" + hash
	case ConstraintKindThrottle:
		if constraint.Throttle != nil {
			switch constraint.Throttle.Scope {
			case enums.ThrottleScopeAccount:
				scope = "Account"
			case enums.ThrottleScopeEnv:
				scope = "Env"
			case enums.ThrottleScopeFn:
				scope = "Fn"
			}
		}
		return "Throttle-" + scope + "-" + hash
	case ConstraintKindConcurrency:
		if constraint.Concurrency != nil {
			switch constraint.Concurrency.Scope {
			case enums.ConcurrencyScopeAccount:
				scope = "Account"
			case enums.ConcurrencyScopeEnv:
				scope = "Env"
			case enums.ConcurrencyScopeFn:
				scope = "Fn"
			}
		}
		return "Concurrency-" + scope + "-" + hash
	}
	return string(constraint.Kind)
}