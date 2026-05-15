//go:generate go run github.com/dmarkham/enumer -trimprefix=OutdatedThrottleReason -type=OutdatedThrottleReason -transform=snake -json -text

package enums

type OutdatedThrottleReason int

// NOTE:
// DO NOT EVER DELETE OR REUSE.
// There are Lua scripts that rely on the integer values in the state metadata.
// Deleting/reusing enum value will break things.
//
//goland:noinspection GoDeprecation
const (
	// Item is not outdated
	OutdatedThrottleReasonNone OutdatedThrottleReason = 0

	// Constraints have throttle set but item is missing throttle key (throttle added).
	OutdatedThrottleReasonMissingItemThrottle OutdatedThrottleReason = 1

	// Item has throttle key but constraints are missing throttle (throttle removed).
	OutdatedThrottleReasonMissingConstraint OutdatedThrottleReason = 2

	// Both item and constraints have throttle configuration but the key expression hash is different (throttle key updated).
	OutdatedThrottleReasonKeyExpressionMismatch OutdatedThrottleReason = 3

	// Item is missing key expression hash (this applies to pre-key-queue items)
	OutdatedThrottleReasonMissingKeyExpressionHash OutdatedThrottleReason = 4
)
