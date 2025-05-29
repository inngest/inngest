//go:generate go run github.com/dmarkham/enumer -trimprefix=CancellationKind -type=CancellationKind -json -text -transform=snake

package enums

type CancellationKind int

const (
	// CancellationKindRun represents a single run to be cancelled
	CancellationKindRun CancellationKind = iota
	// CancellationKindBacklog represents a bulk cancellation of runs
	CancellationKindBulkRun
	// CancellationKindBacklog represents a backlog that needs to be cancelled
	CancellationKindBacklog
)
