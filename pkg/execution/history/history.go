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
	StepID          string
	HistoryGroupID  *uuid.UUID
	IdempotencyKey  string
	Status          *string

	// HistoryType enum
	Type string

	Attempt            int64
	CompletedStepCount *int64
	StepName           *string
	URL                *string
	CancelEvent        *CancelEventHistory
	CancelUser         *CancelUserHistory
	Sleep              *SleepHistory
	WaitForEvent       *WaitForEventHistory
	Result             *ResultHistory
}

type CancelEventHistory struct {
	EventID    ulid.ULID `json:"event_id"`
	Expression *string   `json:"expression"`
}

type CancelUserHistory struct {
	UserID uuid.UUID `json:"user_id"`
}

type SleepHistory struct {
	Datetime string `json:"datetime"`
}

type WaitForEventHistory struct {
	EventName  *string `json:"event_name"`
	Expression *string `json:"expression"`
	Timeout    int     `json:"timeout"`
}

type ResultHistory struct {
	ErrorCode          *string             `json:"error_code"`
	Framework          *string             `json:"framework"`
	Output             any                 `json:"output"`
	Platform           *string             `json:"platform"`
	ResponseDurationMS int                 `json:"response_duration_ms"`
	ResponseHeaders    map[string][]string `json:"response_headers"`
	ResponseSizeBytes  int                 `json:"response_size_bytes"`
	SDKLanguage        string              `json:"sdk_language"`
	SDKVersion         string              `json:"sdk_version"`
	Stack              []map[string]any    `json:"stack"`
}
