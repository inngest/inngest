package state

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/inngest"
	"github.com/oklog/ulid/v2"
)

var (
	// ErrStepIncomplete is returned when requesting output for a step that
	// has not yet completed.
	ErrStepIncomplete = fmt.Errorf("step has not yet completed")
	// ErrPauseNotFound is returned when attempting to lease or consume a pause
	// that doesn't exist within the backing state store.
	ErrPauseNotFound = fmt.Errorf("pause not found")
	// ErrPauseLeased is returned when attempting to lease a pause that is
	// already leased by another event.
	ErrPauseLeased      = fmt.Errorf("pause already leased")
	ErrIdentifierExists = fmt.Errorf("identifier already exists")
)

const (
	// PauseLeaseDuration is the lifetime that a pause's lease is valid for.
	PauseLeaseDuration = 5 * time.Second
)

// Identifier represents the unique identifier for a workflow run.
type Identifier struct {
	WorkflowID uuid.UUID `json:"workflowID"`
	RunID      ulid.ULID `json:"runID"`
	// Key represents a unique idempotency key used to deduplicate this
	// workflow run amongst other runs for the same workflow.
	Key string `json:"key"`
}

func (i Identifier) IdempotencyKey() string {
	key := i.Key
	if i.Key == "" {
		key = i.RunID.String()
	}
	return fmt.Sprintf("%s:%s", i.WorkflowID, key)
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
	//
	// NOTE: the pause should remain within the backing state store for
	// some perioud after the expiry time for checking timeout branches:
	//
	// If this pause has its OnTimeout flag set to true, we only traverse
	// the edge if the event *has not* been received.  In order to check
	// this, we enqueue a job that executes on the pause timeout:  if the
	// pause has not yet been consumed we can safely assume the event was
	// not received.  Therefore, we must be able to load the pause for some
	// time after timeout.
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

func (p Pause) Edge() inngest.Edge {
	return inngest.Edge{
		Outgoing: p.Outgoing,
		Incoming: p.Incoming,
	}
}

// Metadata must be stored for each workflow run, allowing the runner to inspect
// when the execution started, the number of steps enqueued, and the number of
// steps finalized.
//
// Pre-1.0, this is the only way to detect whether a function's execution has
// finished.  Functions may have many parallel branches with conditional execution.
// Given this, no single step can tell whether it's the last step within a function.
type Metadata struct {
	StartedAt time.Time `json:"startedAt"`

	// Pending is the number of steps that have been enqueued but have
	// not yet finalized.
	//
	// Finalized refers to:
	// - A step that has errored out and cannot be retried
	// - A step that has retried a maximum number of times and will not
	//   further be retried.
	// - A step that has completed, and has its next steps (children in
	//   the dag) enqueued. Note that the step must have its children
	//   enqueued to be considered finalized.
	Pending int
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

	// Metadata returns the run metadata, including the started at time
	// as well as the pending count.
	Metadata() Metadata

	// Identifier returns the identifier for this particular run, which
	// returns the RunID and WorkflowID within a state.Identifier struct.
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
	// Load returns run state for the given identifier.
	Load(ctx context.Context, i Identifier) (State, error)

	// IsComplete returns whether the given identifier is complete, ie. the
	// pending count in the identifier's metadata is zero.
	IsComplete(ctx context.Context, i Identifier) (complete bool, err error)
}

/*
// CompleteSubscriber allows users to subscribe to a particular identifier and block
// until the identifier's pending count reaches zero.
//
// This is an optional interface which a state can implement using internal mechanisms
// to watch keys.  The runner will check to see if the state store implements this interface;
// if so, it will subscribe to be notified when the function completes.
type CompleteSubscriber interface {
	BlockUntilComplete(ctx context.Context, i Identifier) (complete bool, err error)
}
*/

// Mutater mutates state for a given identifier, storing the state and returning
// the new state.
//
// It accepst any starting state as its input.  This is usually, and locally in dev,
// a map[string]interface{} containing event data.
type Mutater interface {
	// New creates a new state for the given run ID, using the event as the input data for the root workflow.
	//
	// If the IdempotencyKey within Identifier already exists, the state implementation should return
	// ErrIdentifierExists.
	New(ctx context.Context, workflow inngest.Workflow, i Identifier, input map[string]any) (State, error)

	// scheduled increases the scheduled count for a run's metadata.
	//
	// We need to store the total number of steps enqueued to calculate when a step function
	// has finished execution.  If the state store is the same as the queuee (eg. an all-in-one
	// MySQL store) it makes sense to atomically increase this when enqueueing the step.  However,
	// we must provide compatibility for queues that exist separately to the state store (eg.
	// SQS, Celery).  In thise cases recording that a step was scheduled is a separate step.
	Scheduled(ctx context.Context, i Identifier, stepID string) error

	// Finalized increases the finalized count for a run's metadata.
	//
	// This must be called after storing a response and scheduling all child steps.
	Finalized(ctx context.Context, i Identifier, stepID string) error

	// SaveResponse saves the driver response for the attempt to the backing state store.
	//
	// If the response is an error, this must store the error for the specific attempt, allowing
	// visibility into each error when executing a step.
	SaveResponse(ctx context.Context, i Identifier, r DriverResponse, attempt int) (State, error)
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
	// See https://github.com/inngest/inngest/issues/123 for more info
	LeasePause(ctx context.Context, id uuid.UUID) error

	// ConsumePause consumes a pause by its ID such that it can't be used again and
	// will not be returned from any query.
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

	// PauseByID returns a given pause by pause ID.  This must return expired pauses
	// that have not yet been consumed in order to properly handle timeouts.
	//
	// This should not return consumed pauses.
	PauseByID(ctx context.Context, pauseID uuid.UUID) (*Pause, error)
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
