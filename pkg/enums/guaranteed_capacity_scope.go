//go:generate go run github.com/dmarkham/enumer -trimprefix=GuaranteedCapacityScope -type=GuaranteedCapacityScope -json -text

package enums

type GuaranteedCapacityScope int

// NOTE:
// DO NOT EVER DELETE OR REUSE.
// There are Lua scripts that rely on the integer values in the state metadata.
// Deleting/reusing enum value will break things.
//
//goland:noinspection GoDeprecation
const (
	// GuaranteedCapacityScopeAccount indicates guaranteed capacity on the level of an account
	GuaranteedCapacityScopeAccount GuaranteedCapacityScope = 0
)
