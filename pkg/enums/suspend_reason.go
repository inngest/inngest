//go:generate go run github.com/dmarkham/enumer -trimprefix=SuspendReason -type=SuspendReason -json -text -gqlgen

package enums

type SuspendReason int

const (
	// SuspendReasonPause represents a pause/unpause operation
	SuspendReasonPause SuspendReason = iota

	// SuspendReasonMigrate indicates a migrate operation locking or unlocking the source.
	SuspendReasonMigrate
)
