package constraintapi

import "github.com/inngest/inngest/pkg/enums"

type ConstraintConfig struct {
	// FunctionVersion specifies the latest known function version.
	// If the version on the manager is newer, it will be used.
	// If the version on the manager is outdated (e.g. stale cache), the latest version will be fetched.
	FunctionVersion int

	RateLimit []RateLimitConfig

	// Concurrency represents all concurrency constraints
	Concurrency ConcurrencyConfig

	// Throttle represents 0-n throttle constraints
	Throttle []ThrottleConfig
}

type RateLimitConfig struct {
	Scope enums.RateLimitScope

	// Limit is how often the function can be called within the specified period
	Limit int

	// Period represents the time period for throttling the function
	Period string

	KeyExpressionHash string
}

type CustomConcurrencyLimit struct {
	// Mode specifies whether concurrency is applied to step (default) or function run level
	Mode enums.ConcurrencyMode

	// Scope specifies the concurrency scope, defaults to function
	Scope enums.ConcurrencyScope

	Limit int

	KeyExpressionHash string
}

type ThrottleConfig struct {
	Scope enums.ThrottleScope

	// ThrottleKeyExpressionHash is the hashed throttle key expression, if set.
	ThrottleKeyExpressionHash string `json:"tkh,omitempty"`

	// Limit is the actual rate limit
	Limit int `json:"l"`
	// Burst is the busrsable capacity of the rate limit
	Burst int `json:"b"`
	// Period is the rate limit period, in seconds
	Period int `json:"p"`
}

type ConcurrencyConfig struct {
	// AccountConcurrency represents the global account concurrency limit.
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
