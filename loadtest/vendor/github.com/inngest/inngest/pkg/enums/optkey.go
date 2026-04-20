//go:generate go run github.com/dmarkham/enumer -trimprefix=OptKey -type=OptKey -json -text -transform=title-lower

package enums

type OptKey int

const (
	// OptKeyNone represents the default opt key 0, which does nothing
	OptKeyNone OptKey = iota

	// OptKeyParallelMode represents the "parallelMode" key
	OptKeyParallelMode
)
