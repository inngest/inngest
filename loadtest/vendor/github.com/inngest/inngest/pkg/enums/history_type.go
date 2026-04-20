//go:generate go run github.com/dmarkham/enumer -trimprefix=HistoryType -type=HistoryType -json -text -gqlgen

package enums

type HistoryType int

const (
	// HistoryTypeNone represents the default HistoryType 0, which means nothing
	HistoryTypeNone HistoryType = iota

	HistoryTypeFunctionScheduled
	HistoryTypeFunctionStarted
	HistoryTypeFunctionCompleted
	HistoryTypeFunctionFailed
	HistoryTypeFunctionCancelled
	HistoryTypeFunctionStatusUpdated // TODO: Remove.  Statuses above capture everything.

	HistoryTypeStepScheduled
	HistoryTypeStepStarted
	HistoryTypeStepCompleted
	HistoryTypeStepErrored
	HistoryTypeStepFailed // Permanently failed
	HistoryTypeStepWaiting
	HistoryTypeStepSleeping
	HistoryTypeStepInvoking

	HistoryTypeFunctionSkipped // for reasons, see enums.SkipReason
)
