package queue

import "github.com/inngest/inngest/pkg/enums"

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
