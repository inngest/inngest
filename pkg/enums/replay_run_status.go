//go:generate go run github.com/dmarkham/enumer -trimprefix=ReplayRunStatus -type=ReplayRunStatus -json -text -gqlgen

package enums

import "fmt"

type ReplayRunStatus int

const (
	ReplayRunStatusAll ReplayRunStatus = 0

	ReplayRunStatusCompleted = ReplayRunStatus(RunStatusCompleted)
	ReplayRunStatusFailed    = ReplayRunStatus(RunStatusFailed)
	ReplayRunStatusCancelled = ReplayRunStatus(RunStatusCancelled)

	ReplayRunStatusSkippedPaused = ReplayRunStatus(10 + SkipReasonFunctionPaused)
)

// ReplayableFunctionRunStatuses returns the function run statuses that would be replayed
// if ReplayRunStatusAll is selected.
func ReplayableFunctionRunStatuses() []RunStatus {
	return []RunStatus{
		RunStatusCompleted,
		RunStatusFailed,
		RunStatusCancelled,
	}
}

// ReplayableSkipReasons returns the function skip reasons that would be replayed
// if ReplayRunStatusAll is selected.
func ReplayableSkipReasons() []SkipReason {
	return []SkipReason{
		SkipReasonFunctionPaused,
	}
}

// ExpandReplayRunStatuses expands replay selectors into run statuses and skip reasons.
func ExpandReplayRunStatuses(selectors []ReplayRunStatus) ([]RunStatus, []SkipReason, error) {
	var statuses []RunStatus
	var skipReasons []SkipReason
	var all bool
	for _, selector := range selectors {
		switch selector {
		case ReplayRunStatusAll:
			all = true
		case ReplayRunStatusCompleted:
			statuses = append(statuses, RunStatusCompleted)
		case ReplayRunStatusFailed:
			statuses = append(statuses, RunStatusFailed)
		case ReplayRunStatusCancelled:
			statuses = append(statuses, RunStatusCancelled)
		case ReplayRunStatusSkippedPaused:
			skipReasons = append(skipReasons, SkipReasonFunctionPaused)
		default:
			return nil, nil, fmt.Errorf("unknown replay run status: %d", selector)
		}
	}
	if all {
		return ReplayableFunctionRunStatuses(), ReplayableSkipReasons(), nil
	}
	return statuses, skipReasons, nil
}
