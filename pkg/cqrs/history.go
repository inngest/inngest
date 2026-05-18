package cqrs

import (
	"context"

	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/oklog/ulid/v2"
)

type HistoryManager interface {
	HistoryWriter
	HistoryReader
}

type HistoryWriter interface {
	InsertHistory(ctx context.Context, h history.History) error
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
	GetFunctionRunHistory(ctx context.Context, runID ulid.ULID) ([]*history.History, error)

	// GetRunDefers returns defers attached to each parent run, keyed by
	// parent run ID. Each entry's Run is the child TraceRun if one has been
	// scheduled, otherwise nil. Parents with no defers are omitted.
	GetRunDefers(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID][]RunDefer, error)

	// GetRunDeferredFrom returns the parent linkage for each deferred child
	// run, keyed by child run ID. Runs with no linkage are omitted.
	GetRunDeferredFrom(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID]*RunDeferredFrom, error)

	// GetRunInvokedFrom returns the parent linkage for each child run that
	// was triggered by a parent's `step.invoke`, keyed by child run ID. Runs
	// with no invoke linkage are omitted.
	GetRunInvokedFrom(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID]*RunInvokedFrom, error)
}
