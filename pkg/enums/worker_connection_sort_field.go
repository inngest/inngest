//go:generate go run github.com/dmarkham/enumer -trimprefix=WorkerConnectionTimeField -type=WorkerConnectionTimeField -transform=snake -json -text
package enums

type WorkerConnectionTimeField int

const (
	WorkerConnectionTimeFieldConnectedAt WorkerConnectionTimeField = iota
	WorkerConnectionTimeFieldDisconnectedAt
	WorkerConnectionTimeFieldLastHeartbeatAt
)
