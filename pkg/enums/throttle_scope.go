//go:generate go run github.com/dmarkham/enumer -trimprefix=ThrottleScope -type=ThrottleScope -json -text -gqlgen

package enums

type ThrottleScope int

const (
	// ThrottleScopeFn represents the default ThrottleScope 0, which means limit to the specific function
	ThrottleScopeFn ThrottleScope = 0
	// ThrottleScopeEnv limits throttle to the given environment, forcing environment throttle limits across functions
	// in the same environment.
	ThrottleScopeEnv ThrottleScope = 1
	// ThrottleScopeAccount limits throttle to the entire account, forcing global throttle limits across
	// all functions within your account.
	ThrottleScopeAccount ThrottleScope = 2
)
