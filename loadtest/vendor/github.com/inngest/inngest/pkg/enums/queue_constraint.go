//go:generate go run github.com/dmarkham/enumer -trimprefix=QueueConstraint -type=QueueConstraint -transform=snake -json -text

package enums

type QueueConstraint int

// NOTE:
// DO NOT EVER DELETE OR REUSE.
// There are Lua scripts that rely on the integer values in the state metadata.
// Deleting/reusing enum value will break things.
//
//goland:noinspection GoDeprecation
const (
	QueueConstraintNotLimited            QueueConstraint = 0
	QueueConstraintAccountConcurrency    QueueConstraint = 1
	QueueConstraintFunctionConcurrency   QueueConstraint = 2
	QueueConstraintCustomConcurrencyKey1 QueueConstraint = 3
	QueueConstraintCustomConcurrencyKey2 QueueConstraint = 4
	QueueConstraintThrottle              QueueConstraint = 5
)
