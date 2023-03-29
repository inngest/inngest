//go:generate go run github.com/dmarkham/enumer -trimprefix=CancellationType -type=CancellationType -json -text

package enums

type CancellationType int

const (
	// CancellationTypeNone represents the default CancellationType 0, which does nothing
	CancellationTypeNone CancellationType = iota

	CancellationTypeEvent
	CancellationTypeManual
)
