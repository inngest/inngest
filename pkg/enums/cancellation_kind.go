//go:generate go run github.com/dmarkham/enumer -trimprefix=CancellationKind -type=CancellationKind -json -text -transform=snake

package enums

type CancellationKind int

const (
	CancellationKindRun CancellationKind = iota
	CancellationKindBacklog
)
