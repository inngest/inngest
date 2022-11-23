package state

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

const (
	// PauseLeaseDuration is the lifetime that a pause's lease is valid for.
	PauseLeaseDuration = 5 * time.Second
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
	ErrPauseLeased       = fmt.Errorf("pause already leased")
	ErrIdentifierExists  = fmt.Errorf("identifier already exists")
	ErrFunctionCancelled = fmt.Errorf("function cancelled")
	ErrFunctionComplete  = fmt.Errorf("function completed")
	ErrFunctionFailed    = fmt.Errorf("function failed")
)

// Identifier represents the unique identifier for a workflow run.
type Identifier struct {
	WorkflowID      uuid.UUID `json:"workflowID"`
	WorkflowVersion int       `json:"workflowVersion"`
	RunID           ulid.ULID `json:"runID"`
	// Key represents a unique idempotency key used to deduplicate this
	// workflow run amongst other runs for the same workflow.
	Key string `json:"key"`
}

func (i Identifier) IdempotencyKey() string {
	key := i.Key
	if i.Key == "" {
		key = i.RunID.String()
	}
	return fmt.Sprintf("%s:%d:%s", i.WorkflowID, i.WorkflowVersion, key)
}

type StepNotification struct {
	ID      Identifier
	Step    string
	Attempt int
}

// Metadata must be stored for each workflow run, allowing the runner to inspect
// when the execution started, the number of steps enqueued, and the number of
// steps finalized.
//
// Pre-1.0, this is the only way to detect whether a function's execution has
// finished.  Functions may have many parallel branches with conditional execution.
// Given this, no single step can tell whether it's the last step within a function.
type Metadata struct {
	Status enums.RunStatus `json:"status"`

	// Debugger represents whether this function was started via the debugger.
	Debugger bool `json:"debugger"`

	// RunType indicates the run type for this particular flow.  This allows
	// us to store whether this is eg. a manual retry
	RunType *string `json:"runType,omitempty"`

	// OriginalRunID stores the original run ID, if this run is a retry.
	// This is some basic book-keeping.
	OriginalRunID *ulid.ULID `json:"originalRunID,omitempty"`

	// Name stores the name of the workflow as it started.
	Name string `json:"name"`

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
	Pending int `json:"pending"`
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
	Actions() map[string]any

	// Errors returns all actions that have errored.
	Errors() map[string]error

	// ActionID returns the action output or error for the given ID.
	ActionID(id string) (any, error)

	// ActionComplete returns whether the action with the given ID has finished,
	// ie. has completed with data stored in state.
	//
	// Note that if an action has errored this should return false.
	ActionComplete(id string) bool
}

// Manager represents a state manager which can both load and mutate state.
type Manager interface {
	Loader
	Mutater
	PauseManager
}

// FunctionNotifier is an optional interface that state stores can fulfil,
// invoking callbacks when functions start, complete, error, or permanently
// fail. These callbacks cannot error;  they are not retried. Callbacks are
// called after the state store commits state for functions.
//
// This exists on state stores as states manage the distributed waitgroup
// counts monitoring the number of running steps;  once this counter reaches
// zero the function completes.  Only the state store can monitor when
// functions truly complete successfully.
//
// If a state store fulfils this interface these notifications will be
// called.
type FunctionNotifier interface {
	// OnFunctionStatus adds a new callback which is invoked each time
	// a function changes status.
	OnFunctionStatus(f FunctionCallback)
}

type FunctionCallback func(context.Context, Identifier, enums.RunStatus)

// Loader allows loading of previously stored state based off of a given Identifier.
type Loader interface {
	// Load returns run state for the given identifier.
	Load(ctx context.Context, i Identifier) (State, error)

	// History loads history for the given run identifier.
	History(ctx context.Context, i Identifier) ([]History, error)

	// Runs loads all run metadata
	Runs(ctx context.Context, eventId string) ([]Metadata, error)

	// IsComplete returns whether the given identifier is complete, ie. the
	// pending count in the identifier's metadata is zero.
	IsComplete(ctx context.Context, i Identifier) (complete bool, err error)
}

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
	New(ctx context.Context, input Input) (State, error)

	// Cancel sets a function run metadata status to RunStatusCancelled, which prevents
	// future execution of steps.
	Cancel(ctx context.Context, i Identifier) error

	// scheduled increases the scheduled count for a run's metadata.
	//
	// We need to store the total number of steps enqueued to calculate when a step function
	// has finished execution.  If the state store is the same as the queuee (eg. an all-in-one
	// MySQL store) it makes sense to atomically increase this when enqueueing the step.  However,
	// we must provide compatibility for queues that exist separately to the state store (eg.
	// SQS, Celery).  In thise cases recording that a step was scheduled is a separate step.
	//
	// Attempt is zero-indexed.
	Scheduled(ctx context.Context, i Identifier, stepID string, attempt int, at *time.Time) error

	// Started is called when a step is started.
	//
	// Attempt is zero-indexed.
	Started(ctx context.Context, i Identifier, stepID string, attempt int) error

	// Finalized increases the finalized count for a run's metadata.
	//
	// This must be called after storing a response and scheduling all child steps.
	//
	// Attempt is zero-indexed.
	Finalized(ctx context.Context, i Identifier, stepID string, attempt int) error

	// SaveResponse saves the driver response for the attempt to the backing state store.
	//
	// If the response is an error, this must store the error for the specific attempt, allowing
	// visibility into each error when executing a step.
	//
	// Attempt is zero-indexed.
	SaveResponse(ctx context.Context, i Identifier, r DriverResponse, attempt int) (State, error)
}

// Input is the input for creating new state.  The required fields are Workflow,
// Identifier and Input;  the rest of the data is stored within the state store as
// metadata.
type Input struct {
	Workflow inngest.Workflow
	// Identifier represents the identifier
	Identifier Identifier

	// EventData is the input data for initializing the workflow run, eg. the
	// original event data.
	EventData map[string]any

	// Debugger represents whether this function was started via the debugger.
	Debugger bool

	// RunType indicates the run type for this particular flow.  This allows
	// us to store whether this is eg. a manual retry
	RunType *string `json:"runType,omitempty"`

	// OriginalRunID stores the original run ID, if this run is a retry.
	// This is some basic book-keeping.
	OriginalRunID *ulid.ULID `json:"originalRunID,omitempty"`

	// Steps allows users to specify pre-defined steps to run workflows from
	// arbitrary points.
	Steps map[string]any
}
