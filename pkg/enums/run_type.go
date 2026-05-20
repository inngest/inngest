//go:generate go run github.com/dmarkham/enumer -trimprefix=RunType -type=RunType -json -text -transform=snake

package enums

// RunType identifies whether a run is a top-level (primary) execution or a
// deferred child run scheduled from a parent.
type RunType int

const (
	RunTypeUnknown RunType = iota
	RunTypePrimary
	RunTypeDefer
)
