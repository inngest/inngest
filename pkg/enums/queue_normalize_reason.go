//go:generate go run github.com/dmarkham/enumer -trimprefix=QueueNormalizeReason -type=QueueNormalizeReason -transform=snake -json -text

package enums

type QueueNormalizeReason int

// NOTE:
// DO NOT EVER DELETE OR REUSE.
// There are Lua scripts that rely on the integer values in the state metadata.
// Deleting/reusing enum value will break things.
//
//goland:noinspection GoDeprecation
const (
	QueueNormalizeReasonUnchanged QueueNormalizeReason = iota
	QueueNormalizeReasonThrottleRemoved
	QueueNormalizeReasonThrottleKeyChanged
	QueueNormalizeReasonCustomConcurrencyKeyCountMismatch
	QueueNormalizeReasonCustomConcurrencyKeyNotFoundOnShadowPartition
)
