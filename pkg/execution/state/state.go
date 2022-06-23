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
	ErrPauseNotFound  = fmt.Errorf("pause not found")
	ErrPauseLeased    = fmt.Errorf("pause already leased")
)

const (
	// PauseLeaseDuration is the lifetime that a pause's lease is valid for.
	PauseLeaseDuration = 5 * time.Second
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
	ID uuid.UUID `json:"id"`
	// Identifier is the specific workflow run to resume.  This is required.
	Identifier Identifier `json:"identifier"`
	// Outgoing is the parent step for the pause.
	Outgoing string `json:"outgoing"`
	// Incoming is the step to run after the pause completes.
	Incoming string `json:"incoming"`
	// Expires is a time at which the pause can no longer be resumed.  This
	// gives each pause of a function a TTL.  This is required.
	Expires time.Time `json:"expires"`
	// Event is an optional event that can resume the pause automatically,
	// often paired with an expression.
	Event *string `json:"event"`
	// Expression is an optional expression that must match for the pause
	// to be resumed.
	Expression *string `json:"expression"`
	// OnTimeout indicates that this incoming edge should only be ran
	// when the pause times out, if set to true.
	OnTimeout bool `json:"onTimeout"`
	// LeasedUntil represents the time that this pause is leased until. If
	// nil, this pause is not leased.
	//
	// A lease allows a single worker to claim a pause while enqueueing the
	// pause's next step.  After enqueueing, the worker can consume the pause
	// entirely.
	LeasedUntil *time.Time `json:"leasedUntil,omitempty"`
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
	Workflow() inngest.Workflow

	Identifier() Identifier

	// RunID returns the ID for the specific run.
	RunID() ulid.ULID

	// WorkflowID returns the workflow ID for the run
	WorkflowID() uuid.UUID

	// Event is the root data that triggers the workflow, which is typically
	// an Inngest event.
	Event() map[string]interface{}

	// Actions returns a map of all output from each individual action.
	Actions() map[string]map[string]interface{}

	// Errors returns all actions that have errored.
	Errors() map[string]error

	// ActionID returns the action output or error for the given ID.
	ActionID(id string) (map[string]interface{}, error)

	// ActionComplete returns whether the action with the given ID has finished,
	// ie. has completed with data stored in state.
	//
	// Note that if an action has errored this should return false.
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
	New(ctx context.Context, workflow inngest.Workflow, runID ulid.ULID, input map[string]any) (State, error)

	// SaveActionOutput stores output for a single action within a workflow run.
	//
	// This should clear any error that exists for the current action, indicating that
	// the step is a success.
	SaveActionOutput(ctx context.Context, i Identifier, actionID string, data map[string]interface{}) (State, error)

	// SaveActionError stores an error for a single action within a workflow run.
	//
	// XXX: It might be sensible to store a record of each error that occurred for
	// every attempt, whilst still being able to distinguish between an eventual success
	// and a persistent error.  See: https://github.com/inngest/inngest-cli/issues/125
	// for more info.
	SaveActionError(ctx context.Context, i Identifier, actionID string, err error) (State, error)
}

// PauseMutater manages creating, leasing, and consuming pauses from a backend implementation.
type PauseMutater interface {
	// SavePause indicates that the traversal of an edge is paused until some future time.
	//
	// The runner which coordinates workflow executions is responsible for managing paused
	// DAG executions.
	SavePause(ctx context.Context, p Pause) error

	// LeasePause allows us to lease the pause until the next step is enqueued, at which point
	// we can 'consume' the pause to remove it.
	//
	// This prevents a failure mode in which we consume the pause but enqueueing the next
	// action fails (eg. due to power loss).
	//
	// If the given pause has been leased within LeasePauseDuration, this should return an
	// ErrPauseLeased error.
	//
	// See https://github.com/inngest/inngest-cli/issues/123 for more info
	LeasePause(ctx context.Context, id uuid.UUID) error

	// ConsumePause consumes a pause by its ID such that it can't be used again.
	ConsumePause(ctx context.Context, id uuid.UUID) error
}

// PauseGetter allows a runner to return all existing pauses by event or by outgoing ID.  This
// is required to fetch pauses to automatically continue workflows.
type PauseGetter interface {
	// PausesByEvent returns all pauses for a given event.
	PausesByEvent(ctx context.Context, eventName string) (PauseIterator, error)

	// PauseByStep returns a specific pause for a given workflow run, from a given step.
	//
	// This is required when continuing a step function from an async step, ie. one that
	// has deferred results which must be continued by resuming the specific pause set
	// up for the given step ID.
	PauseByStep(ctx context.Context, i Identifier, actionID string) (*Pause, error)
}

// PauseIterator allows the runner to iterate over all pauses returned by a PauseGetter.  This
// ensures that, at scale, all pauses do not need to be loaded into memory.
type PauseIterator interface {
	// Next advances the iterator and returns whether the next call to Val will
	// return a non-nil pause.
	//
	// Next should be called prior to any call to the iterator's Val method, after
	// the iterator has been created.
	//
	// The order of the iterator is unspecified.
	Next(ctx context.Context) bool

	// Val returns the current Pause from the iterator.
	Val(context.Context) *Pause
}

// PauseManager manages mutating and fetching pauses from a backend implementation.
type PauseManager interface {
	PauseMutater
	PauseGetter
}

// Manager represents a state manager which can both load and mutate state.
type Manager interface {
	Loader
	Mutater
	PauseManager
}
