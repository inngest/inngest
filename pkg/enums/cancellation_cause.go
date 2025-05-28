//go:generate go run github.com/dmarkham/enumer -trimprefix=CancellationCause -type=CancellationCause -json -text -transform=snake

package enums

type CancellationCause int

const (
	// CancellationCauseNone represents the default CancellationCause 0, which does nothing
	CancellationCauseNone CancellationCause = iota
	CancellationCauseEvent
	CancellationCauseManual
	CancellationCauseAPI
)
