//go:generate go run github.com/dmarkham/enumer -trimprefix=SemaphoreScope -type=SemaphoreScope -json -text -gqlgen

package enums

type SemaphoreScope int

const (
	// SemaphoreScopeFn represents the default SemaphoreScope 0, which means limit to the specific function
	SemaphoreScopeFn SemaphoreScope = 0
	// SemaphoreScopeEnv limits the semaphore to the given environment
	SemaphoreScopeEnv SemaphoreScope = 1
	// SemaphoreScopeAccount limits the semaphore to the entire account
	SemaphoreScopeAccount SemaphoreScope = 2
)
