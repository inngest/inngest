package cqrs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// FunctionRun represents a currently ongoing ro past function run.
type FunctionRun struct {
	RunID           ulid.ULID
	RunStartedAt    time.Time
	FunctionID      uuid.UUID
	FunctionVersion int64
	TriggerType     string
	EventID         ulid.ULID
	BatchID         ulid.ULID
	OriginalRunID   ulid.ULID
}

// FunctionRunFinish represents the end of a function.  This may be
// completed, failed, or cancelled, depending on the function's
// status.
//
// If there is no finish entry for a function it is safe to assume
// that the function is still in progress and is part of the state
// store.
type FunctionRunFinish struct {
	RunID              ulid.ULID
	Status             string
	Output             json.RawMessage
	CompletedStepCount int64
	CreatedAt          time.Time
}

type FunctionRunManager interface {
	FunctionRunWriter
	FunctionRunReader
}

type FunctionRunWriter interface {
	InsertFunctionRun(ctx context.Context, run FunctionRun) error
}

type FunctionRunReader interface {
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
