//go:generate go run github.com/dmarkham/enumer -trimprefix=MetadataScope -type=MetadataScope -json -text -transform=snake -gqlgen
package enums

type MetadataScope int

const (
	MetadataScopeUnknown MetadataScope = iota
	MetadataScopeRun
	MetadataScopeStep
	// NOTE: StepAttempt scope is legacy now and should be treated as step scope.
	// We keep it here for backward compatibility with old metadata that was written with this scope.
	MetadataScopeStepAttempt
	MetadataScopeExtendedTrace
)
