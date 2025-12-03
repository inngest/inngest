package constraintapi

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
)

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

// StateKey returns the fully-qualified Redis key pointing to the rate limit GCRA state
func (r *RateLimitConstraint) StateKey(keyPrefix string, accountID uuid.UUID, envID uuid.UUID) string {
	switch r.Scope {
	case enums.RateLimitScopeAccount:
		return fmt.Sprintf("{%s}:rl:a:%s:%s", keyPrefix, accountID, r.EvaluatedKeyHash)
	case enums.RateLimitScopeEnv:
		return fmt.Sprintf("{%s}:rl:e:%s:%s", keyPrefix, envID, r.EvaluatedKeyHash)
	// Function rate limit key is compatible with the queue implementation
	default:
		// NOTE: Rate limit state is prefixed with the rate limit key prefix. This is important for compatibility.
		// See ratelimit/ratelimit_lua.go for the rate limit implementation.
		return fmt.Sprintf("{%s}:%s", keyPrefix, r.EvaluatedKeyHash)
	}
}

type ConcurrencyConstraint struct {
	// Mode specifies whether concurrency is applied to step (default) or function run level
	Mode enums.ConcurrencyMode

	// Scope specifies the concurrency scope, defaults to function
	Scope enums.ConcurrencyScope

	// KeyExpressionHash is the hashed key expression. If this is set, this refers to a custom concurrency key.
	KeyExpressionHash string
	EvaluatedKeyHash  string

	// InProgressItemKey is the fully-qualified Redis key storing the in-progress (concurrency) ZSET for this constraint
	// This is included for consistency purposes and will be phased out once all constraint state is moved to the new data store
	InProgressItemKey string
}

func (c ConcurrencyConstraint) InProgressLeasesKey(prefix string, accountID, envID, functionID uuid.UUID) string {
	switch c.Mode {
	case enums.ConcurrencyModeStep:
	case enums.ConcurrencyModeRun:
		// TODO: How are we going to enforce run level concurrency?
	}

	var scopeID string
	var entityID uuid.UUID
	switch c.Scope {
	case enums.ConcurrencyScopeAccount:
		scopeID = "a"
		entityID = accountID
	case enums.ConcurrencyScopeEnv:
		scopeID = "e"
		entityID = envID
	case enums.ConcurrencyScopeFn:
		scopeID = "f"
		entityID = functionID
	}

	var keyID string
	if c.KeyExpressionHash != "" {
		keyID = fmt.Sprintf("<%s:%s>", c.KeyExpressionHash, c.EvaluatedKeyHash)
	}

	return fmt.Sprintf("{%s}:%s:state:concurrency:%s:%s%s", prefix, accountID, scopeID, entityID, keyID)
}

type ThrottleConstraint struct {
	Scope enums.ThrottleScope

	KeyExpressionHash string
	EvaluatedKeyHash  string
}

// StateKey returns the fully-qualified Redis key pointing to the throttle GCRA state
func (t *ThrottleConstraint) StateKey(keyPrefix string, accountID uuid.UUID, envID uuid.UUID) string {
	switch t.Scope {
	case enums.ThrottleScopeAccount:
		return fmt.Sprintf("{%s}:throttle:a:%s:%s", keyPrefix, accountID, t.EvaluatedKeyHash)
	case enums.ThrottleScopeEnv:
		return fmt.Sprintf("{%s}:throttle:e:%s:%s", keyPrefix, envID, t.EvaluatedKeyHash)
	// Function throttle state key is compatible with the queue implementation
	default:
		return fmt.Sprintf("{%s}:throttle:%s", keyPrefix, t.EvaluatedKeyHash)
	}
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
