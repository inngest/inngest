package state

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
)

type CreateState struct {
	Metadata Metadata
	// Events contains a slice of JSON-encoded events.
	Events []json.RawMessage
	// Steps allows users to specify pre-defined steps to run workflows from
	// arbitrary points.
	Steps []state.MemoizedStep
	// StepInputs allows users to specify pre-defined step inputs to run
	// workflows from arbitrary points.
	StepInputs []state.MemoizedStep
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
	StateLoader

	// Create creates new state in the store for the given run ID.
	Create(ctx context.Context, s CreateState) error
	// Delete deletes state, metadata, and - when pauses are included - associated pauses
	// for the run from the store.  Nothing referencing the run should exist in the state
	// store after.
	Delete(ctx context.Context, id ID) (bool, error)
	// Exists checks whether a run exists given an ID
	Exists(ctx context.Context, id ID) (bool, error)
	// Update updates configuration on the state, eg. setting the execution
	// version after communicating with the SDK.
	UpdateMetadata(ctx context.Context, id ID, config MutableConfig) error
	// SaveStep saves step output for the given run ID and step ID.
	SaveStep(ctx context.Context, id ID, stepID string, data []byte) error
}

// Staeloader defines an interface for loading the entire run state from the state store.
type StateLoader interface {
	// Metadata returns metadata for a given run
	LoadMetadata(ctx context.Context, id ID) (Metadata, error)
	// LoadEvents loads the triggering events for the given run.
	LoadEvents(ctx context.Context, id ID) ([]json.RawMessage, error)
	// LoadState returns all steps for a run.
	LoadSteps(ctx context.Context, id ID) (map[string]json.RawMessage, error)

	// LoadState returns all state for a run, including steps, events, and metadata.
	LoadState(ctx context.Context, id ID) (State, error)

	// StreamState returns all state without loading in-memory
	// StreamState(ctx context.Context, id ID) (io.Reader, error)
}
