//go:generate go run github.com/dmarkham/enumer -trimprefix=StepStatus -type=StepStatus -json -text -gqlgen
package enums

type StepStatus int

const (
	StepStatusUnknown StepStatus = iota
	StepStatusScheduled
	StepStatusRunning
	StepStatusWaiting
	StepStatusSleeping
	StepStatusInvoking
	StepStatusCompleted
	StepStatusFailed
	StepStatusErrored
	StepStatusCancelled
)
