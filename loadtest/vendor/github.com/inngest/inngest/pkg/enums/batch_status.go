//go:generate go run github.com/dmarkham/enumer -trimprefix=BatchStatus -type=BatchStatus -json -text

package enums

type BatchStatus int

const (
	// BatchStatusPending represents a batch that has not started yet
	BatchStatusPending BatchStatus = iota
	BatchStatusReady
	BatchStatusStarted
	BatchStatusAbsent
)
