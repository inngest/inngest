//go:generate go run github.com/dmarkham/enumer -trimprefix=ReplayRunStatus -type=ReplayRunStatus -json -text -gqlgen

package enums

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
