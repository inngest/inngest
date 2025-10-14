//go:generate go run github.com/dmarkham/enumer -trimprefix=RateLimitScope -type=RateLimitScope -json -text -gqlgen

package enums

type RateLimitScope int

const (
	// RateLimitScopeFn represents the default RateLimitScope 0, which means limit to the specific function
	RateLimitScopeFn RateLimitScope = 0
	// RateLimitScopeEnv limits rate limit to the given environment, forcing environment rate limits across functions
	// in the same environment.
	RateLimitScopeEnv RateLimitScope = 1
	// RateLimitScopeAccount limits rate limit to the entire account, forcing global rate limits across
	// all functions within your account.
	RateLimitScopeAccount RateLimitScope = 2
)
