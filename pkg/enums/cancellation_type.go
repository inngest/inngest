//go:generate go run github.com/dmarkham/enumer -trimprefix=CancellationType -type=CancellationType -json -text -transform=snake

package enums

type CancellationType int

const (
	CancellationTypeRun CancellationType = iota
	CancellationTypeBacklog
)
