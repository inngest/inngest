//go:generate go run github.com/dmarkham/enumer -trimprefix=PartitionType -type=PartitionType -json -text

package enums

type PartitionType int

// NOTE:
// DO NOT EVER DELETE OR REUSE.
// There are Lua scripts that rely on the integer values in the state metadata.
// Deleting/reusing enum value will break things.
//
//goland:noinspection GoDeprecation
const (
	// PartitionTypeDefault indicates a regular partition for job items without
	// concurrency keys, or with only a fn level concurrency key.
	PartitionTypeDefault PartitionType = 0
	// PartitionTypenConcurrency represents a partition for a custom concurrency key
	PartitionTypeConcurrencyKey PartitionType = 1
	// PartitionTypeThrottle represents a partition for a custom throttling key.
	PartitionTypeThrottle PartitionType = 2
)
