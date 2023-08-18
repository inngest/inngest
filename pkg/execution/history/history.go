package history

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type Driver interface {
	Close() error
	Write(context.Context, History) error
}

// Represents a row in the workflow_run_history table
type History struct {
	ID              ulid.ULID
	CreatedAt       time.Time
	AccountID       uuid.UUID
	WorkspaceID     uuid.UUID
	FunctionID      uuid.UUID
	FunctionVersion int64
	EventID         ulid.ULID
	BatchID         *ulid.ULID
	RunID           ulid.ULID
	OriginalRunID   *ulid.ULID
	StepID          *string
	StepName        *string
	GroupID         *uuid.UUID
	IdempotencyKey  string
	Status          *string

	// HistoryType enum
	Type string

	Attempt            int64
	CompletedStepCount *int64
	URL                *string
	CancelEvent        *CancelEvent
	CancelUser         *CancelUser
	Sleep              *Sleep
	WaitForEvent       *WaitForEvent
	Result             *Result
}

type CancelEvent struct {
	EventID    ulid.ULID `json:"event_id"`
	Expression *string   `json:"expression"`
}

type CancelUser struct {
	UserID uuid.UUID `json:"user_id"`
}

type Sleep struct {
	Until time.Time `json:"until"`
}

type WaitForEvent struct {
	EventName  *string `json:"event_name"`
	Expression *string `json:"expression"`
	Timeout    int     `json:"timeout"`
}

type Result struct {
	ErrorCode   *string             `json:"error_code"`
	Framework   *string             `json:"framework"`
	Output      any                 `json:"output"`
	Platform    *string             `json:"platform"`
	DurationMS  int                 `json:"response_duration_ms"`
	Headers     map[string][]string `json:"response_headers"`
	SizeBytes   int                 `json:"response_size_bytes"`
	SDKLanguage string              `json:"sdk_language"`
	SDKVersion  string              `json:"sdk_version"`
	Stack       []map[string]any    `json:"stack"`
}
