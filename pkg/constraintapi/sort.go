package constraintapi

import (
	"sort"

	"github.com/inngest/inngest/pkg/enums"
)

// sortConstraints applies a stable in-place sort to the given constraint slice.
// Sorting order:
// 1. By constraint kind: Rate limit < Throttle < Concurrency 
// 2. By scope: Account < Environment < Function
// 3. By key expression hash: empty hash comes first
func sortConstraints(constraints []ConstraintItem) {
	sort.SliceStable(constraints, func(i, j int) bool {
		a, b := constraints[i], constraints[j]
		
		// Primary sort: by constraint kind priority
		kindPriorityA := getConstraintKindPriority(a.Kind)
		kindPriorityB := getConstraintKindPriority(b.Kind)
		if kindPriorityA != kindPriorityB {
			return kindPriorityA < kindPriorityB
		}
		
		// Secondary sort: by scope priority
		scopePriorityA := getConstraintScopePriority(a)
		scopePriorityB := getConstraintScopePriority(b)
		if scopePriorityA != scopePriorityB {
			return scopePriorityA < scopePriorityB
		}
		
		// Tertiary sort: by key expression hash (empty comes first)
		hashA := getConstraintKeyExpressionHash(a)
		hashB := getConstraintKeyExpressionHash(b)
		
		// Empty hash comes first
		if hashA == "" && hashB != "" {
			return true
		}
		if hashA != "" && hashB == "" {
			return false
		}
		
		// Both are empty or both are non-empty, compare lexicographically
		return hashA < hashB
	})
}

// getConstraintKindPriority returns the priority for sorting constraint kinds.
// Lower values have higher priority: Rate limit (1) < Throttle (2) < Concurrency (3)
func getConstraintKindPriority(kind ConstraintKind) int {
	switch kind {
	case ConstraintKindRateLimit:
		return 1
	case ConstraintKindThrottle:
		return 2
	case ConstraintKindConcurrency:
		return 3
	default:
		return 4 // Unknown constraints come last
	}
}

// getConstraintScopePriority returns the priority for sorting constraint scopes.
// Lower values have higher priority: Account (1) < Environment (2) < Function (3)
func getConstraintScopePriority(constraint ConstraintItem) int {
	switch constraint.Kind {
	case ConstraintKindRateLimit:
		if constraint.RateLimit != nil {
			switch constraint.RateLimit.Scope {
			case enums.RateLimitScopeAccount:
				return 1
			case enums.RateLimitScopeEnv:
				return 2
			case enums.RateLimitScopeFn:
				return 3
			default:
				return 4
			}
		}
	case ConstraintKindThrottle:
		if constraint.Throttle != nil {
			switch constraint.Throttle.Scope {
			case enums.ThrottleScopeAccount:
				return 1
			case enums.ThrottleScopeEnv:
				return 2
			case enums.ThrottleScopeFn:
				return 3
			default:
				return 4
			}
		}
	case ConstraintKindConcurrency:
		if constraint.Concurrency != nil {
			switch constraint.Concurrency.Scope {
			case enums.ConcurrencyScopeAccount:
				return 1
			case enums.ConcurrencyScopeEnv:
				return 2
			case enums.ConcurrencyScopeFn:
				return 3
			default:
				return 4
			}
		}
	}
	return 4 // Default case for invalid/nil constraints
}

// getConstraintKeyExpressionHash returns the key expression hash for the constraint.
func getConstraintKeyExpressionHash(constraint ConstraintItem) string {
	switch constraint.Kind {
	case ConstraintKindRateLimit:
		if constraint.RateLimit != nil {
			return constraint.RateLimit.KeyExpressionHash
		}
	case ConstraintKindThrottle:
		if constraint.Throttle != nil {
			return constraint.Throttle.KeyExpressionHash
		}
	case ConstraintKindConcurrency:
		if constraint.Concurrency != nil {
			return constraint.Concurrency.KeyExpressionHash
		}
	}
	return ""
}