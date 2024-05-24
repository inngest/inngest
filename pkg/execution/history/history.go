package history

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/oklog/ulid/v2"
)

type Driver interface {
	Close() error
	Write(context.Context, History) error
}

// History represents a row in:
// - the workflow_run_history table and associated start/end tables
// - or the workflow_run_skips table
type History struct {
	AccountID            uuid.UUID
	Attempt              int64 // ignored for skips
	BatchID              *ulid.ULID
	Cancel               *execution.CancelRequest
	CompletedStepCount   *int64
	CreatedAt            time.Time // for skips, means "run skipped at"
	Cron                 *string   // cron schedule for the run, if this was triggered by cron
	EventID              ulid.ULID
	FunctionID           uuid.UUID
	FunctionVersion      int64 // ignored for skips
	GroupID              *uuid.UUID
	ID                   ulid.ULID // ignored for skips
	IdempotencyKey       string    // ignored for skips
	InvokeFunction       *InvokeFunction
	InvokeFunctionResult *InvokeFunctionResult
	LatencyMS            *int64
	OriginalRunID        *ulid.ULID
	Result               *Result
	RunID                ulid.ULID
	SkipReason           *enums.SkipReason // valid iff Type == enums.HistoryTypeFunctionSkipped
	Sleep                *Sleep
	Status               *string
	StepID               *string
	StepName             *string
	StepType             *enums.HistoryStepType
	Type                 string // see enums.HistoryType
	URL                  *string
	WaitForEvent         *WaitForEvent
	WaitResult           *WaitResult
	WorkspaceID          uuid.UUID
}

type CancelEvent struct {
	EventID    *ulid.ULID `json:"event_id"`
	Expression *string    `json:"expression"`
}

type CancelUser struct {
	UserID uuid.UUID `json:"user_id"`
}

type Sleep struct {
	Until time.Time `json:"until"`
}

type WaitForEvent struct {
	EventName  string    `json:"event_name"`
	Expression *string   `json:"expression"`
	Timeout    time.Time `json:"timeout"`
}

type WaitResult struct {
	EventID *ulid.ULID `json:"event_id"`
	Timeout bool       `json:"timeout"`
}

type InvokeFunction struct {
	CorrelationID string    `json:"correlation_id"`
	EventID       ulid.ULID `json:"event_id"`
	FunctionID    string    `json:"function_id"`
	Timeout       time.Time `json:"timeout"`
}

type InvokeFunctionResult struct {
	EventID *ulid.ULID `json:"event_id"`
	RunID   *ulid.ULID `json:"run_id"`
	Timeout bool       `json:"timeout"`
}

type Result struct {
	DurationMS int                 `json:"response_duration_ms"`
	ErrorCode  *string             `json:"error_code"`
	Framework  *string             `json:"framework"`
	Headers    map[string][]string `json:"response_headers"`
	// Output is either the data or error, depending on the
	// history type.
	Output      string           `json:"output"`
	RawOutput   any              `json:"raw_output"`
	Platform    *string          `json:"platform"`
	SDKLanguage string           `json:"sdk_language"`
	SDKVersion  string           `json:"sdk_version"`
	SizeBytes   int              `json:"response_size_bytes"`
	Stack       []map[string]any `json:"stack"`
}
