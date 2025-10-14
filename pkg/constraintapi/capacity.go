package constraintapi

import "github.com/inngest/inngest/pkg/enums"

type ConstraintKind string

const (
	CapacityKindRateLimit   ConstraintKind = "rate_limit"
	CapacityKindConcurrency ConstraintKind = "concurrency"
	CapacityKindThrottle    ConstraintKind = "throttle"
)

type RateLimitCapacity struct {
	Scope enums.RateLimitScope

	KeyExpressionHash string

	EvaluatedKeyHash string
}

type ConcurrencyCapacity struct {
	// Mode specifies whether concurrency is applied to step (default) or function run level
	Mode enums.ConcurrencyMode

	// Scope specifies the concurrency scope, defaults to function
	Scope enums.ConcurrencyScope

	// KeyExpressionHash is the hashed key expression. If this is set, this refers to a custom concurrency key.
	KeyExpressionHash string
	EvaluatedKeyHash  string
}

type ThrottleCapacity struct {
	Scope enums.ThrottleScope

	KeyExpressionHash string
	EvaluatedKeyHash  string
}

type ConstraintCapacityItem struct {
	Kind *ConstraintKind

	// Amount specifies the number of units for a constraint.
	//
	// Examples:
	// 3 units of step-level concurrency allows to run 3 steps (~queue items).
	// 1 unit of rate limit capacity allows to start 1 run. Rejecting causes event to be skipped.
	// 1 unit of throttle capacity allows to start 1 run. Rejecting causes queue to wait and retry.
	Amount int
}
