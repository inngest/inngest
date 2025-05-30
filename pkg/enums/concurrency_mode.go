//go:generate go run github.com/dmarkham/enumer -trimprefix=ConcurrencyMode -type=ConcurrencyMode -json -text -gqlgen

package enums

type ConcurrencyMode int

const (
	// ConcurrencyModeStep represents concurrency applied on steps (e.g. Function X may only have n active steps).
	ConcurrencyModeStep ConcurrencyMode = 0

	// ConcurrencyModeRun represents concurrency applied on runs (e.g. Function X may only have n active runs).
	ConcurrencyModeRun ConcurrencyMode = 1
)
