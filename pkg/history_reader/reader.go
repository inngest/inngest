package history_reader

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/usage"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
)

var (
	DefaultQueryLimit = 50
	ErrNotFound       = errors.New("not found")
)

type RunTimeField string

const (
	RunTimeFieldEndedAt RunTimeField = "ended_at"

	// TODO: Delete this when the UI no longer needs to filter by a mix of
	// started_at and ended_at.
	RunTimeFieldMixed RunTimeField = "mixed"

	RunTimeFieldStartedAt RunTimeField = "started_at"
)

// Reader defines the history reader interface, loading runs and run history
type Reader interface {
	CountRuns(ctx context.Context, opts CountRunOpts) (int, error)
	CountReplayRuns(ctx context.Context, opts CountReplayRunsOpts) (ReplayRunCounts, error)
	GetRun(
		ctx context.Context,
		runID ulid.ULID,
		opts GetRunOpts,
	) (Run, error)
	GetRuns(ctx context.Context, opts GetRunsOpts) ([]Run, error)
	GetReplayRuns(ctx context.Context, opts GetReplayRunsOpts) ([]ReplayRun, error)
	GetRunHistory(
		ctx context.Context,
		runID ulid.ULID,
		opts GetRunOpts,
	) ([]*RunHistory, error)
	GetRunHistoryItemOutput(
		ctx context.Context,
		historyID ulid.ULID,
		opts GetHistoryOutputOpts,
	) (*string, error)
	GetRunsByEventID(
		ctx context.Context,
		eventID ulid.ULID,
		opts GetRunsByEventIDOpts,
	) ([]Run, error)
	GetSkippedRunsByEventID(
		ctx context.Context,
		eventID ulid.ULID,
		opts GetRunsByEventIDOpts,
	) ([]SkippedRun, error)
	GetUsage(ctx context.Context, opts GetUsageOpts) ([]usage.UsageSlot, error)

	// GetActiveRunIDs returns the IDs of runs that are queued or running (i.e.
	// not ended)
	GetActiveRunIDs(
		ctx context.Context,
		opts GetActiveRunIDsOpts,
	) ([]ulid.ULID, error)

	// GetActiveRunIDs returns a count of runs that are queued or running (i.e.
	// not ended)
	CountActiveRuns(
		ctx context.Context,
		opts CountActiveRunsOpts,
	) (int, error)

	// This also embeds the V1 function reader interface.
	cqrs.APIV1FunctionRunReader
}

type CountRunOpts struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	WorkflowID  uuid.UUID
	LowerTime   time.Time
	UpperTime   time.Time
	TimeField   RunTimeField
	Statuses    []enums.RunStatus
}

func (c CountRunOpts) Validate() error {
	if c.AccountID == uuid.Nil {
		return errors.New("account ID must be provided")
	}
	if c.WorkspaceID == uuid.Nil {
		return errors.New("workspace ID must be provided")
	}
	if c.WorkflowID == uuid.Nil {
		return errors.New("workflow ID must be provided")
	}
	if c.LowerTime.IsZero() {
		return errors.New("lower time must be provided")
	}
	if c.UpperTime.IsZero() {
		return errors.New("upper time must be provided")
	}

	return nil
}

type GetHistoryOutputOpts struct {
	AccountID   uuid.UUID
	RunID       ulid.ULID
	WorkspaceID uuid.UUID
	WorkflowID  uuid.UUID
}

func (o GetHistoryOutputOpts) Validate() error {
	if o.AccountID == uuid.Nil {
		return errors.New("account ID must be provided")
	}
	if o.RunID.IsZero() {
		return errors.New("run ID must be provided")
	}
	if o.WorkspaceID == uuid.Nil {
		return errors.New("workspace ID must be provided")
	}
	if o.WorkflowID == uuid.Nil {
		return errors.New("workflow ID must be provided")
	}

	return nil
}

type GetRunOpts struct {
	AccountID   uuid.UUID
	WorkspaceID *uuid.UUID
	WorkflowID  *uuid.UUID
}

func (o GetRunOpts) Validate() error {
	if o.AccountID == uuid.Nil {
		return errors.New("account ID must be provided")
	}

	return nil
}

type GetRunsByEventIDOpts struct {
	AccountID   uuid.UUID
	WorkspaceID *uuid.UUID
}

func (o GetRunsByEventIDOpts) Validate() error {
	if o.AccountID == uuid.Nil {
		return errors.New("account ID must be provided")
	}
	if o.WorkspaceID != nil && *o.WorkspaceID == uuid.Nil {
		return errors.New("workspace ID cannot be an empty UUID")
	}

	return nil
}

type GetRunsOpts struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	// If the workflow ID is nil, all functions in an env will be queried.
	WorkflowID *uuid.UUID
	LowerTime  time.Time
	UpperTime  time.Time
	TimeField  RunTimeField
	Limit      int
	Cursor     *ulid.ULID
	Statuses   []enums.RunStatus
	// If true returns oldest first.  Defaults to descending/newest first.
	Ascending bool
}

func (c GetRunsOpts) Validate() error {
	if c.AccountID == uuid.Nil {
		return errors.New("account ID must be provided")
	}
	if c.WorkspaceID == uuid.Nil {
		return errors.New("workspace ID must be provided")
	}
	if c.WorkflowID != nil && *c.WorkflowID == uuid.Nil {
		return errors.New("workflow ID must be provided")
	}
	if c.LowerTime.IsZero() {
		return errors.New("lower time must be provided")
	}
	if c.UpperTime.IsZero() {
		return errors.New("upper time must be provided")
	}
	if c.Limit < 0 {
		return errors.New("limit must be positive")
	}

	return nil
}

type GetUsageOpts struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	WorkflowID  uuid.UUID
	LowerTime   time.Time
	UpperTime   time.Time
	Period      enums.Period // deprecated, no longer used
	Granularity time.Duration
	Statuses    []enums.RunStatus
}

func (o GetUsageOpts) Validate() error {
	if o.AccountID == uuid.Nil {
		return errors.New("account ID must be set")
	}
	if o.WorkspaceID == uuid.Nil {
		return errors.New("workspace ID must be set")
	}
	if o.WorkflowID == uuid.Nil {
		return errors.New("workflow ID must be set")
	}
	if o.LowerTime.IsZero() {
		return errors.New("lower time must be set")
	}
	if o.UpperTime.IsZero() {
		return errors.New("upper time must be set")
	}

	return nil
}

type Run struct {
	AccountID       uuid.UUID
	BatchID         *ulid.ULID
	EndedAt         *time.Time
	EventID         ulid.ULID
	ID              ulid.ULID
	OriginalRunID   *ulid.ULID
	Output          *string
	StartedAt       time.Time
	Status          enums.RunStatus
	WorkflowID      uuid.UUID
	WorkspaceID     uuid.UUID
	WorkflowVersion int
	Cron            *string
}

type SkippedRun struct {
	AccountID   uuid.UUID
	BatchID     *ulid.ULID
	EventID     ulid.ULID
	ID          ulid.ULID
	SkippedAt   time.Time
	SkipReason  enums.SkipReason
	WorkflowID  uuid.UUID
	WorkspaceID uuid.UUID
}

func (r Run) ToCQRS() *cqrs.FunctionRun {
	run := &cqrs.FunctionRun{
		RunID:           r.ID,
		RunStartedAt:    r.StartedAt,
		FunctionID:      r.WorkflowID,
		FunctionVersion: int64(r.WorkflowVersion),
		WorkspaceID:     r.WorkspaceID,
		EventID:         r.EventID,
		BatchID:         r.BatchID,
		OriginalRunID:   r.OriginalRunID,
		Status:          r.Status,
		EndedAt:         r.EndedAt,
		Cron:            r.Cron,
	}
	if r.Output != nil {
		run.Output = util.EnsureJSON(json.RawMessage(*r.Output))
	}
	return run
}

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

type RunHistoryCancel struct {
	EventID    *ulid.ULID `json:"eventID"`
	Expression *string    `json:"expression"`
	UserID     *uuid.UUID `json:"userID"`
}

type RunHistoryResult struct {
	DurationMS  int     `json:"durationMS"`
	ErrorCode   *string `json:"errorCode"`
	Framework   *string `json:"framework"`
	Platform    *string `json:"platform"`
	SDKLanguage string  `json:"sdkLanguage"`
	SDKVersion  string  `json:"sdkVersion"`
	SizeBytes   int     `json:"sizeBytes"`
}

func NewRunHistoryResultFromHistoryResult(hr *history.Result) *RunHistoryResult {
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

type RunHistorySleep struct {
	Until time.Time `json:"until"`
}

type RunHistoryWaitForEvent struct {
	EventName  string    `json:"eventName"`
	Expression *string   `json:"expression"`
	Timeout    time.Time `json:"timeout"`
}

type RunHistoryWaitResult struct {
	EventID *ulid.ULID `json:"eventID"`
	Timeout bool       `json:"timeout"`
}

type RunHistoryInvokeFunction struct {
	CorrelationID string    `json:"correlationID"`
	EventID       ulid.ULID `json:"eventID"`
	FunctionID    string    `json:"functionID"`
	Timeout       time.Time `json:"timeout"`
}

type RunHistoryInvokeFunctionResult struct {
	EventID *ulid.ULID `json:"eventID"`
	RunID   *ulid.ULID `json:"runID"`
	Timeout bool       `json:"timeout"`
}

type ReplayRun struct {
	ID         ulid.ULID  // run ID
	BatchID    *ulid.ULID // batch ID
	EventID    ulid.ULID  // event ID
	WorkflowID uuid.UUID  // workflow ID
	Cron       *string    // cron schedule, if this was a cron-triggered run
}

type GetReplayRunsOpts struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	WorkflowID  *uuid.UUID // if workflow ID is nil, all functions in the env will be queried
	LowerTime   time.Time
	UpperTime   time.Time
	Statuses    []enums.RunStatus  // if empty, no completed/failed/cancelled runs will be included
	SkipReasons []enums.SkipReason // if empty, no skipped runs will be included
	Limit       int
	Cursor      *ulid.ULID
}

func (c GetReplayRunsOpts) Validate() error {
	if c.AccountID == uuid.Nil {
		return errors.New("account ID must be provided")
	}
	if c.WorkspaceID == uuid.Nil {
		return errors.New("workspace ID must be provided")
	}
	if c.WorkflowID != nil && *c.WorkflowID == uuid.Nil {
		return errors.New("workflow ID must be provided")
	}
	if c.LowerTime.IsZero() {
		return errors.New("lower time must be provided")
	}
	if c.UpperTime.IsZero() {
		return errors.New("upper time must be provided")
	}
	if c.UpperTime.Before(c.LowerTime) {
		return errors.New("upper/end time must be after lower/start time")
	}
	if c.Limit < 0 {
		return errors.New("limit must be positive")
	}
	if len(c.Statuses) == 0 && len(c.SkipReasons) == 0 {
		return errors.New("at least one status or skip reason must be provided")
	}

	return nil
}

// CountReplayRunsOpts is used to estimate the number of runs that would match the given criteria for replay.
// See GetReplayRunsOpts for field documentation.
type CountReplayRunsOpts struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	WorkflowID  *uuid.UUID
	LowerTime   time.Time
	UpperTime   time.Time
}

type ReplayRunCounts struct {
	CompletedCount     int
	FailedCount        int
	CancelledCount     int
	SkippedPausedCount int
}

func (c CountReplayRunsOpts) Validate() error {
	gRROpts := GetReplayRunsOpts{
		AccountID:   c.AccountID,
		WorkspaceID: c.WorkspaceID,
		WorkflowID:  c.WorkflowID,
		LowerTime:   c.LowerTime,
		UpperTime:   c.UpperTime,
		Statuses:    enums.ReplayableFunctionRunStatuses(),
		SkipReasons: enums.ReplayableSkipReasons(),
		Limit:       DefaultQueryLimit,
		Cursor:      nil,
	}
	return gRROpts.Validate()
}

type GetActiveRunIDsOpts struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	WorkflowID  uuid.UUID
	LowerTime   time.Time
	UpperTime   time.Time
	Limit       int
	Cursor      *ulid.ULID
}

func (c GetActiveRunIDsOpts) Validate() error {
	if c.AccountID == uuid.Nil {
		return errors.New("account ID must be provided")
	}
	if c.WorkspaceID == uuid.Nil {
		return errors.New("workspace ID must be provided")
	}
	if c.WorkflowID == uuid.Nil {
		return errors.New("workflow ID must be provided")
	}
	if c.LowerTime.IsZero() {
		return errors.New("lower time must be provided")
	}
	if c.UpperTime.IsZero() {
		return errors.New("upper time must be provided")
	}
	if c.UpperTime.Before(c.LowerTime) {
		return errors.New("upper/end time must be after lower/start time")
	}
	if c.Limit < 0 {
		return errors.New("limit must be positive")
	}

	return nil
}

type CountActiveRunsOpts struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	WorkflowID  uuid.UUID
	LowerTime   *time.Time
	UpperTime   time.Time
}

func (c CountActiveRunsOpts) Validate() error {
	if c.AccountID == uuid.Nil {
		return errors.New("account ID must be provided")
	}
	if c.WorkspaceID == uuid.Nil {
		return errors.New("workspace ID must be provided")
	}
	if c.WorkflowID == uuid.Nil {
		return errors.New("workflow ID must be provided")
	}
	if c.LowerTime.IsZero() {
		return errors.New("lower time must be provided")
	}
	if c.UpperTime.IsZero() {
		return errors.New("upper time must be provided")
	}
	if c.LowerTime != nil && c.UpperTime.Before(*c.LowerTime) {
		return errors.New("upper/end time must be after lower/start time")
	}

	return nil
}
