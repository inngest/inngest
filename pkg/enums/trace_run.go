//go:generate go run github.com/dmarkham/enumer -trimprefix=TraceRunTime -type=TraceRunTime -transform=snake -json -text
package enums

type TraceRunTime int

const (
	TraceRunTimeQueuedAt TraceRunTime = iota
	TraceRunTimeStartedAt
	TraceRunTimeEndedAt
)
