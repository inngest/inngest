//go:generate go run github.com/dmarkham/enumer -trimprefix=StepStatus -type=StepStatus -json -text -gqlgen
package enums

type StepStatus int

const (
	StepStatusUnknown StepStatus = iota
	// deprecated: use StepStatusQueued for new tracing.
	StepStatusScheduled
	StepStatusRunning
	StepStatusWaiting
	StepStatusSleeping
	StepStatusInvoking
	StepStatusCompleted
	StepStatusFailed
	StepStatusErrored
	StepStatusCancelled
	StepStatusTimedOut
	StepStatusSkipped
	// StepStatusQueued replaces StepStatusScheduled in new traces.
	StepStatusQueued
)

func (s StepStatus) IsEnded() bool {
	switch s {
	case StepStatusCompleted, StepStatusFailed, StepStatusErrored, StepStatusCancelled, StepStatusTimedOut:
		return true
	default:
		return false
	}
}
