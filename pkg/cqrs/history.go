package cqrs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/oklog/ulid/v2"
)

type Step struct {
	ID          *string          `json:"id"`
	AccountID   uuid.UUID        `json:"account_id"`
	WorkspaceID uuid.UUID        `json:"environment_id"`
	Name        *string          `json:"name"`
	Type        string           `json:"type"`
	Status      *string          `json:"status"`
	Config      *json.RawMessage `json:"config"`
	Output      json.RawMessage  `json:"output"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	//Error	    TBD - Should use error codes struct

	// Attempt              int64
	// BatchID              *ulid.ULID
	// Cancel               *execution.CancelRequest
	// CompletedStepCount   *int64
	// CreatedAt            time.Time
	// Cron                 *string
	// EventID              ulid.ULID
	// FunctionID           uuid.UUID
	// FunctionVersion      int64
	// GroupID              *uuid.UUID
	// ID                   ulid.ULID
	// IdempotencyKey       string
	// InvokeFunction       *InvokeFunction
	// InvokeFunctionResult *InvokeFunctionResult
	// LatencyMS            *int64
	// OriginalRunID        *ulid.ULID
	// Result               *Result
	// RunID                ulid.ULID
	// Sleep                *Sleep
	// Status               *string
	// StepID               *string
	// StepName             *string
	// StepType             *enums.HistoryStepType
	// Type                 string
	// URL                  *string
	// WaitForEvent         *WaitForEvent
	// WaitResult           *WaitResult
}

type HistoryManager interface {
	HistoryWriter
	HistoryReader
}

type HistoryWriter interface {
	InsertHistory(ctx context.Context, h history.History) error
}

type HistoryReader interface {
	APIV1FunctionRunHistoryReader

	// GetFunctionRunHistory must return history for the given function run,
	// ordered from oldest to newest.
	GetFunctionRunHistory(ctx context.Context, runID ulid.ULID) ([]*history.History, error)
}

type APIV1FunctionRunHistoryReader interface {
	// GetFunctionRunLogs returns history for the given function run
	GetFunctionRunLogs(
		ctx context.Context,
		accountID uuid.UUID,
		workspaceID uuid.UUID,
		runID ulid.ULID,
	) ([]*Step, error)
}
