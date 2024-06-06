//go:generate go run github.com/dmarkham/enumer -trimprefix=ReplayRunStatus -type=ReplayRunStatus -json -text -gqlgen

package enums

type ReplayRunStatus int

const (
	ReplayRunStatusAll ReplayRunStatus = 0

	ReplayRunStatusCompleted = ReplayRunStatus(10 + RunStatusCompleted)
	ReplayRunStatusFailed    = ReplayRunStatus(10 + RunStatusFailed)
	ReplayRunStatusCancelled = ReplayRunStatus(10 + RunStatusCancelled)

	ReplayRunStatusSkippedPaused = ReplayRunStatus(100 + SkipReasonFunctionPaused)
)
