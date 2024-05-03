package state

import (
	"context"
	"io"

	"github.com/google/uuid"
)

type CreateState struct {
	Metadata Metadata
	Events   [][]byte
	// XXX: You cannot start state with pre-existing steps yet.
}

type StateService interface {
	RunService

	// FunctionMetrics returns state metrics for a given function.
	FunctionMetrics(ctx context.Context, fnID uuid.UUID) (RunMetrics, error)
	// EnvMetrics returns state metrics grouped by environment.
	EnvMetrics(ctx context.Context, envID uuid.UUID) (RunMetrics, error)
	// AccountMetrics returns state metrics grouped by account.
	AccountMetrics(ctx context.Context, accountID uuid.UUID) (RunMetrics, error)
}

type RunService interface {
	// Create creates new state in the store for the given run ID.
	Create(ctx context.Context, s CreateState) error
	// Delete deletes state, metadata, and - when pauses are included - associated pauses
	// for the run from the store.  Nothing referencing the run should exist in the state
	// store after.
	Delete(ctx context.Context, id ID) error
	// LoadState returns all state for a run.
	LoadState(ctx context.Context, id ID) (State, error)
	// StreamState returns all state without loading in-memory
	StreamState(ctx context.Context, id ID) (io.Reader, error)
	// Metadata returns metadata for a given run
	LoadMetadata(ctx context.Context, id ID) (Metadata, error)
	// Update updates configuration on the state, eg. setting the execution
	// version after communicating with the SDK.
	UpdateMetadata(ctx context.Context, id ID, config MutableConfig) error
	// SaveStep saves step output for the given run ID and step ID.
	SaveStep(ctx context.Context, id ID, stepID StepID, data []byte) error
}
