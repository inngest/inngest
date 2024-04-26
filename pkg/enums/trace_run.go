//go:generate go run github.com/dmarkham/enumer -trimprefix=TraceRunTime -type=TraceRunTime -json -text
package enums

type TraceRunTime int

const (
	TraceRunTimeQueuedAt TraceRunTime = iota
	TraceRunTimeStartedAt
	TraceRunTimeEndedAt
)
