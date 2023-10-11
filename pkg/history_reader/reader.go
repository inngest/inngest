package history_reader

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/usage"
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

type Reader interface {
	CountRuns(ctx context.Context, opts CountRunOpts) (int, error)
	GetRun(
		ctx context.Context,
		runID ulid.ULID,
		opts GetRunOpts,
	) (Run, error)
	GetRuns(ctx context.Context, opts GetRunsOpts) ([]Run, error)
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
	GetUsage(ctx context.Context, opts GetUsageOpts) ([]usage.UsageSlot, error)
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
	zeroULID := ulid.ULID{}
	if o.RunID == zeroULID {
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
	WorkflowID  uuid.UUID
	LowerTime   time.Time
	UpperTime   time.Time
	TimeField   RunTimeField
	Limit       int
	Cursor      *ulid.ULID
	Statuses    []enums.RunStatus
}

func (c GetRunsOpts) Validate() error {
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
	StartedAt       time.Time
	Status          enums.RunStatus
	WorkflowID      uuid.UUID
	WorkspaceID     uuid.UUID
	WorkflowVersion int
}

type RunHistory struct {
	Attempt         int64                   `json:"attempt"`
	Cancel          *RunHistoryCancel       `json:"cancel"`
	CreatedAt       time.Time               `json:"createdAt"`
	FunctionVersion int64                   `json:"functionVersion"`
	GroupID         *uuid.UUID              `json:"groupID"`
	ID              ulid.ULID               `json:"id"`
	Result          *RunHistoryResult       `json:"result"`
	RunID           ulid.ULID               `json:"runID"`
	Sleep           *RunHistorySleep        `json:"sleep"`
	StepName        *string                 `json:"stepName"`
	StepType        *enums.HistoryStepType  `json:"stepType"`
	Type            enums.HistoryType       `json:"type"`
	URL             *string                 `json:"url"`
	WaitForEvent    *RunHistoryWaitForEvent `json:"waitForEvent"`
	WaitResult      *RunHistoryWaitResult   `json:"waitResult"`
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
