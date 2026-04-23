// Package history_reader contains the legacy history query surface while
// callers migrate to the cqrs-owned DAL contracts.
//
// Deprecated: use package cqrs history types and interfaces instead.
package history_reader

import (
	"context"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

type (
	RunTimeField                    = cqrs.RunTimeField
	CountRunOpts                    = cqrs.CountRunOpts
	GetHistoryOutputOpts            = cqrs.GetHistoryOutputOpts
	GetRunOpts                      = cqrs.GetRunOpts
	GetRunsByEventIDOpts            = cqrs.GetRunsByEventIDOpts
	GetRunsOpts                     = cqrs.GetRunsOpts
	GetUsageOpts                    = cqrs.GetUsageOpts
	Run                             = cqrs.Run
	SkippedRun                      = cqrs.SkippedRun
	RunHistory                      = cqrs.RunHistory
	RunHistoryCancel                = cqrs.RunHistoryCancel
	RunHistoryResult                = cqrs.RunHistoryResult
	RunHistorySleep                 = cqrs.RunHistorySleep
	RunHistoryWaitForEvent          = cqrs.RunHistoryWaitForEvent
	RunHistoryWaitResult            = cqrs.RunHistoryWaitResult
	RunHistoryInvokeFunction        = cqrs.RunHistoryInvokeFunction
	RunHistoryInvokeFunctionResult  = cqrs.RunHistoryInvokeFunctionResult
	ReplayRun                       = cqrs.ReplayRun
	GetReplayRunsOpts               = cqrs.GetReplayRunsOpts
	CountReplayRunsOpts             = cqrs.CountReplayRunsOpts
	ReplayRunCounts                 = cqrs.ReplayRunCounts
	GetActiveRunIDsOpts             = cqrs.GetActiveRunIDsOpts
	CountActiveRunsOpts             = cqrs.CountActiveRunsOpts
)

var (
	DefaultQueryLimit = cqrs.DefaultQueryLimit
	ErrNotFound       = cqrs.ErrNotFound
)

const (
	RunTimeFieldEndedAt   = cqrs.RunTimeFieldEndedAt
	RunTimeFieldMixed     = cqrs.RunTimeFieldMixed
	RunTimeFieldStartedAt = cqrs.RunTimeFieldStartedAt
)

func NewRunHistoryResultFromHistoryResult(hr *cqrs.Result) *RunHistoryResult {
	return cqrs.NewRunHistoryResultFromHistoryResult(hr)
}

// Reader defines the legacy history reader interface while callers migrate to
// the cqrs-owned history surfaces.
//
// Deprecated: use package cqrs history interfaces instead.
type Reader interface {
	CountRuns(ctx context.Context, opts CountRunOpts) (int, error)
	CountReplayRuns(ctx context.Context, opts CountReplayRunsOpts) (ReplayRunCounts, error)
	GetRun(ctx context.Context, runID ulid.ULID, opts GetRunOpts) (Run, error)
	GetRuns(ctx context.Context, opts GetRunsOpts) ([]Run, error)
	GetReplayRuns(ctx context.Context, opts GetReplayRunsOpts) ([]ReplayRun, error)
	GetRunHistory(ctx context.Context, runID ulid.ULID, opts GetRunOpts) ([]*RunHistory, error)
	GetRunHistoryItemOutput(ctx context.Context, historyID ulid.ULID, opts GetHistoryOutputOpts) (*string, error)
	GetRunsByEventID(ctx context.Context, eventID ulid.ULID, opts GetRunsByEventIDOpts) ([]Run, error)
	GetSkippedRunsByEventID(ctx context.Context, eventID ulid.ULID, opts GetRunsByEventIDOpts) ([]SkippedRun, error)
	GetUsage(ctx context.Context, opts GetUsageOpts) ([]cqrs.HistoryUsage, error)
	GetActiveRunIDs(ctx context.Context, opts GetActiveRunIDsOpts) ([]ulid.ULID, error)
	CountActiveRuns(ctx context.Context, opts CountActiveRunsOpts) (int, error)
	cqrs.APIV1FunctionRunReader
}
