package cqrs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// FunctionRun represents a currently ongoing or past function run.
type FunctionRun struct {
	RunID           ulid.ULID `json:"run_id"`
	RunStartedAt    time.Time `json:"run_started_at"`
	FunctionID      uuid.UUID `json:"function_id"`
	FunctionVersion int64     `json:"function_version"`
	WorkspaceID     uuid.UUID `json:"workspace_id"`
	TriggerType     string    `json:"trigger_type"`
	EventID         ulid.ULID `json:"event_id"`
	BatchID         ulid.ULID `json:"batch_id"`
	OriginalRunID   ulid.ULID `json:"original_run_id"`
	Cron            *string   `json:"cron,omitempty"`

	Result *FunctionRunFinish `json:"result"`
}

// FunctionRunFinish represents the end of a function.  This may be
// completed, failed, or cancelled, depending on the function's
// status.
//
// If there is no finish entry for a function it is safe to assume
// that the function is still in progress and is part of the state
// store.
type FunctionRunFinish struct {
	RunID              ulid.ULID       `json:"-"`
	Status             string          `json:"status"`
	Output             json.RawMessage `json:"output"`
	CompletedStepCount int64           `json:"-"`
	CreatedAt          time.Time       `json:"finished_at"`
}

type FunctionRunManager interface {
	FunctionRunWriter
	FunctionRunReader
}

type FunctionRunWriter interface {
	InsertFunctionRun(ctx context.Context, run FunctionRun) error
}

type FunctionRunReader interface {
	GetFunctionRun(ctx context.Context, workspaceID uuid.UUID, id ulid.ULID) (*FunctionRun, error)
	GetFunctionRunsFromEvents(ctx context.Context, eventIDs []ulid.ULID) ([]*FunctionRun, error)
	GetFunctionRunsTimebound(ctx context.Context, t Timebound, limit int) ([]*FunctionRun, error)
	// GetFunctionRunFinishesByrunIDs loads all function finishes for the given run IDs.  Note that
	// the function run IDs specified may not have finished resulting in no data for those runs.
	//
	// This means that we never guarantee that the length of the returned slice equals the length
	// of the given run IDs.
	//
	// Function finishes are inserted via history lifecycles and are not directly written by
	// the CQRS layer.
	GetFunctionRunFinishesByRunIDs(ctx context.Context, runIDs []ulid.ULID) ([]*FunctionRunFinish, error)
}
