package db

import (
	"github.com/inngest/inngest/pkg/enums"
)

// span statuses are StepStatus strings stamped onto run-root extension rows;
// an absent or unknown status means the run is still in flight, matching the
// GraphQL runs list behavior
func RunStatusFromSpanStatus(statusText string) enums.RunStatus {
	if stepStatus, err := enums.StepStatusString(statusText); err == nil && stepStatus != enums.StepStatusUnknown {
		return enums.StepStatusToRunStatus(stepStatus)
	}

	return enums.RunStatusRunning
}
