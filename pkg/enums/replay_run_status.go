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
