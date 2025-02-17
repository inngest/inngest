//go:generate go run github.com/dmarkham/enumer -trimprefix=ConcurrencyScope -type=ConcurrencyScope -json -text -gqlgen

package enums

type ConcurrencyScope int

const (
	// ConcurrencyScopeFn represents the default ConcurrencyScope 0, which means limit to the specific function
	ConcurrencyScopeFn ConcurrencyScope = 0
	// ConcurrencyScopeEnv limits concurrency to the given environment, forcing environment limits across functions
	// in the same environment.
	ConcurrencyScopeEnv ConcurrencyScope = 1
	// ConcurrencyScopeAccount limits concurrency to the entire account, forcing global concurrency limits across
	// all functions within your account.
	ConcurrencyScopeAccount ConcurrencyScope = 2
)
