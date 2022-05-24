package state

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/oklog/ulid/v2"
)

var (
	ErrStepIncomplete = fmt.Errorf("step has not yet completed")
)

// Identifier represents the unique identifier for a workflow run.
type Identifier struct {
	WorkflowID uuid.UUID `json:"workflowID"`
	RunID      ulid.ULID `json:"runID"`
}

// Pause allows steps of a function to be paused until some time in the future.
// It pauses a specific workflow run via an Identifier, at a specific step in
// the function as specified by Target.
type Pause struct {
	ID uuid.UUID `json:"token"`
	// Identifier is the specific workflow run to resume.  This is required.
	Identifier Identifier `json:"identifier"`
	// Target is the client ID of the step to resume from when the pause
	// is completed.  This is required.
	Target string `json:"target"`
	// Expires is a time at which the pause can no longer be resumed.  This
	// gives each pause of a function a TTL.  This is required.
	Expires time.Time `json:"expires"`
	// Event is an optional event that can resume the pause automatically,
	// often paired with an expression.
	Event *string `json:"event"`
	// Expression is an optional expression that must match for the pause
	// to be resumed.
	Expression *string `json:"expression"`
}

// State represents the current state of a workflow.  It is data-structure
// agnostic;  each backing store can change the structure of the state to
// suit its implementation.
//
// It is assumed that, once initialized, state does not error when returning
// data for the given identifier.
type State interface {
	// Workflow returns the concrete workflow that is being executed
	// for the given run.
	Workflow() (inngest.Workflow, error)

	Identifier() Identifier

	// RunID returns the ID for the specific run.
	RunID() ulid.ULID

	// WorkflowID returns the workflow ID for the run
	WorkflowID() uuid.UUID

	// Event is the root data that triggers the workflow, which is typically
	// an Inngest event.  For scheduled workflows this is a nil map.
	Event() map[string]interface{}

	// Actions returns a map of all output from each individual action.
	Actions() map[string]map[string]interface{}

	// Errors returns all actions that have errored.
	Errors() map[string]error

	// ActionID returns the action output or error for the given ID.
	ActionID(id string) (map[string]interface{}, error)

	// ActionComplete returns whether the action with the given ID has finished,
	// ie. has completed with data stored in state or has errored.
	ActionComplete(id string) bool
}

// Loader allows loading of previously stored state based off of a given Identifier.
type Loader interface {
	Load(ctx context.Context, i Identifier) (State, error)
}

// Mutater mutates state for a given identifier, storing the state and returning
// the new state.
//
// It accepst any starting state as its input.  This is usually, and locally in dev,
// a map[string]interface{} containing event data.
type Mutater interface {
	// New creates a new state for the given run ID, using the event as the input data for the root workflow.
	New(ctx context.Context, workflow inngest.Workflow, runID ulid.ULID, input any) (State, error)

	// SaveActionOutput stores output for a single action within a workflow run.
	SaveActionOutput(ctx context.Context, i Identifier, actionID string, data map[string]interface{}) (State, error)

	// SaveActionError stores an error for a single action within a workflow run.  This is
	// considered final, as in the action will not be retried.
	SaveActionError(ctx context.Context, i Identifier, actionID string, err error) (State, error)

	// SavePause indicates that the traversal of an edge is paused until some future time.
	//
	// The runner which coordinates workflow executions is responsible for managing paused
	// DAG executions.
	//
	// Note that in production it's likely that you want to coordinate amongst pauses and only
	// let a pause be used once.  This requires syncing with distributed locks / leases.  This
	// basic interface does not provide a mechanism for "leasing" a pause temporarily whilst
	// a step is scheduled.  An implementation may choose to extend this interface with its own
	// leasing or locking mechanism.
	SavePause(ctx context.Context, p Pause) error

	// ConsumePause consumes a pause by its ID such that it won't be used again.
	ConsumePause(ctx context.Context, id uuid.UUID) error
}

// Manager represents a state manager which can both load and mutate state.
type Manager interface {
	Loader
	Mutater
}
