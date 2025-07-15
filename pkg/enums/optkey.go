//go:generate go run github.com/dmarkham/enumer -trimprefix=OptKey -type=OptKey -json -text

package enums

// OptKey represents a key in the "opts" map reported by the SDK
type OptKey int

const (
	// OptKeyNone represents the default opt key 0, which does nothing
	OptKeyNone OptKey = iota

	// OptKeyOptPar represents the "optimize parallel" key
	OptKeyOptPar
)
