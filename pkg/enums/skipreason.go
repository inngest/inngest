//go:generate go run github.com/dmarkham/enumer -trimprefix=SkipReason -type=SkipReason -json -text -gqlgen

package enums

type SkipReason int

const (
	// SkipReasonNone represents the default SkipReason 0, which means nothing
	SkipReasonNone SkipReason = iota

	// SkipReasonFunctionPaused indicates that the function was paused.
	SkipReasonFunctionPaused
)
