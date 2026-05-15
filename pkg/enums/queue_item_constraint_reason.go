//go:generate go run github.com/dmarkham/enumer -trimprefix=QueueItemConstraintReason -type=QueueItemConstraintReason -transform=snake -json -text

package enums

type QueueItemConstraintReason int

const (
	QueueItemConstraintReasonConstraintAPIUninitialized QueueItemConstraintReason = iota
	QueueItemConstraintReasonIdNil
	QueueItemConstraintReasonFeatureFlagDisabled
	QueueItemConstraintReasonConstraintAPIError
)
