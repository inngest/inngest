package queue

import (
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
)

type PartitionConstraintConfig struct {
	FunctionVersion int `json:"fv,omitempty,omitzero"`

	Concurrency PartitionConcurrency `json:"c,omitempty,omitzero"`

	// Throttle configuration, optionally specifying key. If no key is set, the throttle value will be the function ID.
	Throttle *PartitionThrottle `json:"t,omitempty,omitzero"`
}

type CustomConcurrencyLimit struct {
	Mode                enums.ConcurrencyMode  `json:"m"`
	Scope               enums.ConcurrencyScope `json:"s"`
	HashedKeyExpression string                 `json:"k"`
	Limit               int                    `json:"l"`
}

type PartitionThrottle struct {
	// ThrottleKeyExpressionHash is the hashed throttle key expression, if set.
	ThrottleKeyExpressionHash string `json:"tkh,omitempty"`

	// Limit is the actual rate limit
	Limit int `json:"l"`
	// Burst is the busrsable capacity of the rate limit
	Burst int `json:"b"`
	// Period is the rate limit period, in seconds
	Period int `json:"p"`
}

type PartitionConcurrency struct {
	// SystemConcurrency represents the concurrency limit to apply to system queues. Unset on regular function partitions.
	SystemConcurrency int `json:"sc,omitempty"`

	// AccountConcurrency represents the global account concurrency limit. This is unset on system queues.
	AccountConcurrency int `json:"ac,omitempty"`

	// FunctionConcurrency represents the function concurrency limit.
	FunctionConcurrency int `json:"fc,omitempty"`

	// AccountRunConcurrency represents the global account run concurrency limit (how many active runs per account). This is unset on system queues.
	AccountRunConcurrency int `json:"arc,omitempty"`

	// FunctionRunConcurrency represents the function run concurrency limit (how many active runs allowed per function).
	FunctionRunConcurrency int `json:"frc,omitempty"`

	// Up to two custom concurrency keys on user-defined scopes, optionally specifying a key. The key is required
	// on env or account level scopes.
	CustomConcurrencyKeys []CustomConcurrencyLimit `json:"cck,omitempty"`
}

type BacklogRefillConstraintCheckResult struct {
	ItemsToRefill        []string
	ItemCapacityLeases   []CapacityLease
	LimitingConstraint   enums.QueueConstraint
	SkipConstraintChecks bool

	RetryAfter time.Time
}

func ConvertLimitingConstraint(
	constraints PartitionConstraintConfig,
	limitingConstraints []constraintapi.ConstraintItem,
) enums.QueueConstraint {
	constraint := enums.QueueConstraintNotLimited

	for _, c := range limitingConstraints {
		switch {
		// Account concurrency
		case
			c.Kind == constraintapi.ConstraintKindConcurrency &&
				c.Concurrency.Scope == enums.ConcurrencyScopeAccount &&
				c.Concurrency.KeyExpressionHash == "":
			constraint = enums.QueueConstraintAccountConcurrency

		// Function concurrency
		case
			c.Kind == constraintapi.ConstraintKindConcurrency &&
				c.Concurrency.Scope == enums.ConcurrencyScopeFn &&
				c.Concurrency.KeyExpressionHash == "":
			constraint = enums.QueueConstraintFunctionConcurrency

		// Custom concurrency key 1
		case
			len(constraints.Concurrency.CustomConcurrencyKeys) > 0 &&
				c.Kind == constraintapi.ConstraintKindConcurrency &&
				c.Concurrency.Mode == constraints.Concurrency.CustomConcurrencyKeys[0].Mode &&
				c.Concurrency.Scope == constraints.Concurrency.CustomConcurrencyKeys[0].Scope &&
				c.Concurrency.KeyExpressionHash == constraints.Concurrency.CustomConcurrencyKeys[0].HashedKeyExpression:
			constraint = enums.QueueConstraintCustomConcurrencyKey1

		// Custom concurrency key 2
		case
			len(constraints.Concurrency.CustomConcurrencyKeys) > 1 &&
				c.Kind == constraintapi.ConstraintKindConcurrency &&
				c.Concurrency.Mode == constraints.Concurrency.CustomConcurrencyKeys[1].Mode &&
				c.Concurrency.Scope == constraints.Concurrency.CustomConcurrencyKeys[1].Scope &&
				c.Concurrency.KeyExpressionHash == constraints.Concurrency.CustomConcurrencyKeys[1].HashedKeyExpression:
			constraint = enums.QueueConstraintCustomConcurrencyKey2

		// Throttle
		case
			c.Kind == constraintapi.ConstraintKindThrottle:
			constraint = enums.QueueConstraintThrottle
		}
	}

	return constraint
}

type ItemLeaseConstraintCheckResult struct {
	// capacityLease optionally returns a capacity lease ID which
	// must be passed to the processing function to be extended
	// while processing the item.
	CapacityLease *CapacityLease

	// limitingConstraint returns the most limiting constraint in case
	// no capacity was available.
	LimitingConstraint enums.QueueConstraint

	// skipConstraintChecks determines whether subsequent operations
	// should check and enforce constraints, and whether constraint state
	// should be updated while processing a queue item.
	//
	// When enrolled to the Constraint API and holding a valid capacity lease,
	// constraint checks _and_ updates may be skipped, as state is maintained within
	// the Constraint API.
	SkipConstraintChecks bool

	RetryAfter time.Time
}
