package redis_state

import (
	"github.com/inngest/inngest/pkg/enums"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
)

func (q PartitionConstraintConfig) HasOutdatedThrottle(qi osqueue.QueueItem) enums.OutdatedThrottleReason {
	itemThrottle := qi.Data.Throttle
	constraintThrottle := q.Throttle

	switch {
	// Neither item nor constraint have throttle set
	case itemThrottle == nil && constraintThrottle == nil:
		return enums.OutdatedThrottleReasonNone

	// Item has throttle but constraint does not (throttle removed)
	case itemThrottle != nil && constraintThrottle == nil:
		return enums.OutdatedThrottleReasonMissingConstraint

	// Constraint has throttle but item does not (throttle added)
	case itemThrottle == nil && constraintThrottle != nil:
		return enums.OutdatedThrottleReasonMissingItemThrottle

	// Both item and constraint throttle are set but expression hash does not match
	case itemThrottle != nil && constraintThrottle != nil:
		// If item has throttle set but no key expression hash, we should re-evaluate to avoid missing throttle key updates
		if itemThrottle.KeyExpressionHash == "" {
			return enums.OutdatedThrottleReasonMissingKeyExpressionHash
		}

		// The key expression hash was set on both sides and changed: Throttle key expression was updated
		if itemThrottle.KeyExpressionHash != constraintThrottle.ThrottleKeyExpressionHash {
			return enums.OutdatedThrottleReasonKeyExpressionMismatch
		}

		return enums.OutdatedThrottleReasonNone
	default:
		return enums.OutdatedThrottleReasonNone
	}
}
