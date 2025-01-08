//go:generate go run github.com/dmarkham/enumer -trimprefix=TraceRunTime -type=TraceRunTime -transform=snake -json -text
package enums

type WorkerConnectionTimeField int

const (
	WorkerConnectionTimeFieldConnectedAt WorkerConnectionTimeField = iota
	WorkerConnectionTimeFieldLastHeartbeatAt
)
