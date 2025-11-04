package constraintapi

import "github.com/inngest/inngest/pkg/enums"

type ConstraintKind string

const (
	ConstraintKindRateLimit   ConstraintKind = "rate_limit"
	ConstraintKindConcurrency ConstraintKind = "concurrency"
	ConstraintKindThrottle    ConstraintKind = "throttle"
)

func (k ConstraintKind) IsQueueConstraint() bool {
	return k == ConstraintKindConcurrency || k == ConstraintKindThrottle
}

type RateLimitConstraint struct {
	Scope enums.RateLimitScope

	KeyExpressionHash string

	EvaluatedKeyHash string
}

type ConcurrencyConstraint struct {
	// Mode specifies whether concurrency is applied to step (default) or function run level
	Mode enums.ConcurrencyMode

	// Scope specifies the concurrency scope, defaults to function
	Scope enums.ConcurrencyScope

	// KeyExpressionHash is the hashed key expression. If this is set, this refers to a custom concurrency key.
	KeyExpressionHash string
	EvaluatedKeyHash  string
}

type ThrottleConstraint struct {
	Scope enums.ThrottleScope

	KeyExpressionHash string
	EvaluatedKeyHash  string
}

type ConstraintItem struct {
	Kind ConstraintKind

	Concurrency *ConcurrencyConstraint
	Throttle    *ThrottleConstraint
	RateLimit   *RateLimitConstraint
}

type ConstraintUsage struct {
	Constraint ConstraintItem

	Used  int
	Limit int
}
