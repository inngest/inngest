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
}
