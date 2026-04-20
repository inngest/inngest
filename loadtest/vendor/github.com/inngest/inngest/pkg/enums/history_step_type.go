//go:generate go run github.com/dmarkham/enumer -trimprefix=HistoryStepType -type=HistoryStepType -json -text -gqlgen

package enums

type HistoryStepType int

const (
	HistoryStepTypeRun HistoryStepType = iota
	HistoryStepTypeSend
	HistoryStepTypeSleep
	HistoryStepTypeWait
)
