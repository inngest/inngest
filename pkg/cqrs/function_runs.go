package cqrs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

// FunctionRun represents a currently ongoing or past function run.
type FunctionRun struct {
	RunID           ulid.ULID       `json:"run_id"`
	RunStartedAt    time.Time       `json:"run_started_at"`
	FunctionID      uuid.UUID       `json:"function_id"`
	FunctionVersion int64           `json:"function_version"`
	WorkspaceID     uuid.UUID       `json:"environment_id"`
	EventID         ulid.ULID       `json:"event_id"`
	BatchID         *ulid.ULID      `json:"batch_id,omitempty"`
	OriginalRunID   *ulid.ULID      `json:"original_run_id,omitempty"`
	Cron            *string         `json:"cron,omitempty"`
	Status          enums.RunStatus `json:"status"`
	EndedAt         *time.Time      `json:"ended_at"`
	Output          json.RawMessage `json:"output,omitempty"`
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
	Status             enums.RunStatus `json:"status"`
	Output             json.RawMessage `json:"output"`
	CreatedAt          time.Time       `json:"finished_at"`
	CompletedStepCount int64           `json:"-"`
}

type FunctionRunManager interface {
	FunctionRunWriter
	FunctionRunReader
}

type FunctionRunWriter interface {
	InsertFunctionRun(ctx context.Context, run FunctionRun) error
}

type FunctionRunReader interface {
	APIV1FunctionRunReader

	GetFunctionRunsTimebound(ctx context.Context, t Timebound, limit int) ([]*FunctionRun, error)
}

type APIV1FunctionRunReader interface {
	// GetFunctionRunsFromEvents returns all function runs invoked by the given event IDs.
	GetFunctionRunsFromEvents(
		ctx context.Context,
		accountID uuid.UUID,
		workspaceID uuid.UUID,
		eventIDs []ulid.ULID,
	) ([]*FunctionRun, error)

	// GetFunctionRun returns a single function run for a given function.
	GetFunctionRun(
		ctx context.Context,
		accountID uuid.UUID,
		workspaceID uuid.UUID,
		runID ulid.ULID,
	) (*FunctionRun, error)
}
