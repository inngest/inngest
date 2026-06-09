package db

import (
	"strconv"

	"github.com/inngest/inngest/pkg/enums"
)

func RunStatusFromSpanStatus(statusText string) enums.RunStatus {
	if code, err := strconv.ParseInt(statusText, 10, 64); err == nil {
		if status := enums.RunCodeToStatus(code); status != enums.RunStatusUnknown {
			return status
		}
	}

	if stepStatus, err := enums.StepStatusString(statusText); err == nil && stepStatus != enums.StepStatusUnknown {
		return enums.StepStatusToRunStatus(stepStatus)
	}

	return enums.RunStatusRunning
}
