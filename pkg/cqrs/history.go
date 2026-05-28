package cqrs

import (
	"context"

	exechistory "github.com/inngest/inngest/pkg/execution/history"
	"github.com/oklog/ulid/v2"
)

type HistoryManager interface {
	HistoryWriter
	HistoryReader
}

type History = exechistory.History
type CancelEvent = exechistory.CancelEvent
type CancelUser = exechistory.CancelUser
type Sleep = exechistory.Sleep
type WaitForEvent = exechistory.WaitForEvent
type WaitResult = exechistory.WaitResult
type WaitForSignal = exechistory.WaitForSignal
type WaitForSignalResult = exechistory.WaitForSignalResult
type InvokeFunction = exechistory.InvokeFunction
type InvokeFunctionResult = exechistory.InvokeFunctionResult
type Result = exechistory.Result

type HistoryWriter interface {
	InsertHistory(ctx context.Context, h History) error
}

type HistoryReader interface {
	CountRuns(ctx context.Context, opts CountRunOpts) (int, error)
	CountReplayRuns(ctx context.Context, opts CountReplayRunsOpts) (ReplayRunCounts, error)
	GetHistoryRun(ctx context.Context, runID ulid.ULID, opts GetRunOpts) (Run, error)
	GetHistoryRuns(ctx context.Context, opts GetRunsOpts) ([]Run, error)
	GetReplayRuns(ctx context.Context, opts GetReplayRunsOpts) ([]ReplayRun, error)
	GetRunHistory(ctx context.Context, runID ulid.ULID, opts GetRunOpts) ([]*RunHistory, error)
	GetRunHistoryItemOutput(ctx context.Context, historyID ulid.ULID, opts GetHistoryOutputOpts) (*string, error)
	GetRunsByEventID(ctx context.Context, eventID ulid.ULID, opts GetRunsByEventIDOpts) ([]Run, error)
	GetSkippedRunsByEventID(ctx context.Context, eventID ulid.ULID, opts GetRunsByEventIDOpts) ([]SkippedRun, error)
	GetUsage(ctx context.Context, opts GetUsageOpts) ([]HistoryUsage, error)
	GetActiveRunIDs(ctx context.Context, opts GetActiveRunIDsOpts) ([]ulid.ULID, error)
	CountActiveRuns(ctx context.Context, opts CountActiveRunsOpts) (int, error)
	APIV1FunctionRunReader

	// GetFunctionRunHistory must return history for the given function run,
	// ordered from oldest to newest.
	GetFunctionRunHistory(ctx context.Context, runID ulid.ULID) ([]*History, error)

	// GetRunDefers returns defers attached to each parent run, keyed by
	// parent run ID. Each entry's Run is the child TraceRun if one has been
	// scheduled, otherwise nil. Parents with no defers are omitted.
	GetRunDefers(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID][]RunDefer, error)

	// GetRunDeferredFrom returns the parent linkage for each deferred child
	// run, keyed by child run ID. A batched child can descend from multiple
	// parents, so each child maps to a list of parents. Runs with no linkage
	// are omitted.
	GetRunDeferredFrom(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID][]RunDeferredFrom, error)
}
