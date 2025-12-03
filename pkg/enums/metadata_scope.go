//go:generate go run github.com/dmarkham/enumer -trimprefix=MetadataScope -type=MetadataScope -json -text -transform=snake -gqlgen
package enums

type MetadataScope int

const (
	MetadataScopeUnknown MetadataScope = iota
	MetadataScopeRun
	MetadataScopeStep
	MetadataScopeStepAttempt
	MetadataScopeExtendedTrace
)
