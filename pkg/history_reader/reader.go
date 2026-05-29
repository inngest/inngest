// Package history_reader contains the legacy history query surface while
// callers migrate to the cqrs-owned DAL contracts.
//
// Deprecated: use package cqrs history types and interfaces instead.
package history_reader

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
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

// RunHistory mirrors the legacy GraphQL-bound history item shape while callers
// migrate to cqrs-owned types.
//
// Deprecated: use cqrs.RunHistory instead.
type RunHistory struct {
	Attempt              int64                           `json:"attempt"`
	Cancel               *RunHistoryCancel               `json:"cancel"`
	CreatedAt            time.Time                       `json:"createdAt"`
	FunctionVersion      int64                           `json:"functionVersion"`
	GroupID              *uuid.UUID                      `json:"groupID"`
	ID                   ulid.ULID                       `json:"id"`
	InvokeFunction       *RunHistoryInvokeFunction       `json:"invokeFunction"`
	InvokeFunctionResult *RunHistoryInvokeFunctionResult `json:"invokeFunctionResult"`
	Result               *RunHistoryResult               `json:"result"`
	RunID                ulid.ULID                       `json:"runID"`
	Sleep                *RunHistorySleep                `json:"sleep"`
	StepName             *string                         `json:"stepName"`
	StepType             *enums.HistoryStepType          `json:"stepType"`
	Type                 enums.HistoryType               `json:"type"`
	URL                  *string                         `json:"url"`
	WaitForEvent         *RunHistoryWaitForEvent         `json:"waitForEvent"`
	WaitResult           *RunHistoryWaitResult           `json:"waitResult"`
}

// Deprecated: use cqrs.RunHistoryCancel instead.
type RunHistoryCancel struct {
	EventID    *ulid.ULID `json:"eventID"`
	Expression *string    `json:"expression"`
	UserID     *uuid.UUID `json:"userID"`
}

// Deprecated: use cqrs.RunHistoryResult instead.
type RunHistoryResult struct {
	DurationMS  int     `json:"durationMS"`
	ErrorCode   *string `json:"errorCode"`
	Framework   *string `json:"framework"`
	Platform    *string `json:"platform"`
	SDKLanguage string  `json:"sdkLanguage"`
	SDKVersion  string  `json:"sdkVersion"`
	SizeBytes   int     `json:"sizeBytes"`
}

// Deprecated: use cqrs.RunHistorySleep instead.
type RunHistorySleep struct {
	Until time.Time `json:"until"`
}

// Deprecated: use cqrs.RunHistoryWaitForEvent instead.
type RunHistoryWaitForEvent struct {
	EventName  string    `json:"eventName"`
	Expression *string   `json:"expression"`
	Timeout    time.Time `json:"timeout"`
}

// Deprecated: use cqrs.RunHistoryWaitResult instead.
type RunHistoryWaitResult struct {
	EventID *ulid.ULID `json:"eventID"`
	Timeout bool       `json:"timeout"`
}

// Deprecated: use cqrs.RunHistoryInvokeFunction instead.
type RunHistoryInvokeFunction struct {
	CorrelationID string    `json:"correlationID"`
	EventID       ulid.ULID `json:"eventID"`
	FunctionID    string    `json:"functionID"`
	Timeout       time.Time `json:"timeout"`
}

// Deprecated: use cqrs.RunHistoryInvokeFunctionResult instead.
type RunHistoryInvokeFunctionResult struct {
	EventID *ulid.ULID `json:"eventID"`
	RunID   *ulid.ULID `json:"runID"`
	Timeout bool       `json:"timeout"`
}

func NewRunHistoryResultFromHistoryResult(hr *cqrs.Result) *RunHistoryResult {
	if hr == nil {
		return nil
	}

	return &RunHistoryResult{
		DurationMS:  hr.DurationMS,
		ErrorCode:   hr.ErrorCode,
		Framework:   hr.Framework,
		Platform:    hr.Platform,
		SDKLanguage: hr.SDKLanguage,
		SDKVersion:  hr.SDKVersion,
		SizeBytes:   hr.SizeBytes,
	}
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
