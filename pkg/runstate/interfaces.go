package runstate

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type StateService interface {
	// Create creates new state in the store for the given run ID.
	Create(ctx context.Context, s State) error
	// Delete deletes state (and associated pauses for the run) from the store.
	Delete(ctx context.Context, runID ulid.ULID) error
	// Update updates configuration on the state, eg. setting the execution
	// version after communicating with the SDK.
	UpdateConfig(ctx context.Context, runID ulid.ULID, config StateConfig) error
	// SaveStep saves step output for the given run ID and step ID.
	// TODO: This should return updated stack/step state.
	SaveStep(ctx context.Context, runID ulid.ULID, stepID StepID, data []byte) error
}

type PauseService interface {
	SavePause(ctx context.Context, p any) error
	// LeasePause leases a given pause, ensuring that only the caller can
	// consume a pause.  This allows for transactionality when conuming a
	// pause, if consuming a pause, updating steps, and deleting the pause is
	// not a single transaction.
	// TODO: Remove if we make consuming atomic, etc.
	LeasePause(ctx context.Context, id ulid.ULID) error
	// ConsumePause consumes the pause with the given ID.  Each pause stores a
	// run ID and step ID;  consuming a pause must update the run's state to save
	// the step's result as the given data.
	//
	// Consuming a pause must also atomically delete the pause.
	ConsumePause(ctx context.Context, id ulid.ULID, data []byte) error
	// DeletePause deletes a pause with the given ID.
	DeletePause(ctx context.Context, id ulid.ULID) error

	// -- Loader methods

	// PausesByIDs returns pauses by their IDs.  This must return expired pauses
	// that have not yet been consumed in order to properly handle timeouts.
	//
	// This should not return consumed pauses.  Note that it is expected that
	// the number of pause IDs is minimal;  enough to return a slice of pauses without
	// an iterator.
	PausesByIDs(ctx context.Context, pauseID ...uuid.UUID) ([]any, error)

	// PausesByEvent returns all pauses for a given event, in a given workspace. If since
	// is a non-zero time, this should load only pauses stored after the given time.
	PausesByEvent(ctx context.Context, workspaceID uuid.UUID, eventName string, since time.Time) (PauseIterator, error)
}
