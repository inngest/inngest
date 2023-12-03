package state

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
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
	ErrPauseLeased        = fmt.Errorf("pause already leased")
	ErrPauseAlreadyExists = fmt.Errorf("pause already exists")
	ErrIdentifierExists   = fmt.Errorf("identifier already exists")
	ErrFunctionCancelled  = fmt.Errorf("function cancelled")
	ErrFunctionComplete   = fmt.Errorf("function completed")
	ErrFunctionFailed     = fmt.Errorf("function failed")
	ErrFunctionOverflowed = fmt.Errorf("function has too many steps")
	ErrDuplicateResponse  = fmt.Errorf("duplicate response")
)

// Identifier represents the unique identifier for a workflow run.
type Identifier struct {
	RunID ulid.ULID `json:"runID"`
	// WorkflowID tracks the internal ID of the function
	WorkflowID uuid.UUID `json:"wID"`
	// WorkflowVersion tracks the version of the function that was live
	// at the time of the trigger.
	WorkflowVersion int `json:"wv"`
	// StaticVersion indicates whether the workflow is pinned to the
	// given function definition over the life of the function.  If functions
	// are deployed to their own URLs, this ensures that the endpoint we hit
	// for the function (and therefore code) stays the same.  Note:  this is only
	// important when we people use separate endpoints per function version.
	StaticVersion bool `json:"s,omitempty"`
	// EventID tracks the event ID that started the function.
	EventID ulid.ULID `json:"evtID"`
	// BatchID tracks the batch ID for the function, if the function uses batching.
	BatchID *ulid.ULID `json:"bID,omitempty"`
	// Key represents a unique user-defined key to be used as part of the
	// idempotency key.  This is appended to the workflow ID and workflow
	// version to create a full idempotency key (via the IdempotencyKey() method).
	//
	// If this is not present the RunID is used as this value.
	Key string `json:"key,omitempty"`
	// AccountID represents the account ID for this run
	AccountID uuid.UUID `json:"aID"`
	// WorkspaceID represents the ws ID for this run
	WorkspaceID uuid.UUID `json:"wsID"`
	// If this is a rerun, the original run ID is stored here.
	OriginalRunID *ulid.ULID `json:"oRunID,omitempty"`
	// ReplayID stores the ID of the replay, if this identifier belongs to a replay.
	ReplayID *uuid.UUID `json:"rID,omitempty"`
	// PriorityFactor is the overall priority factor for this particular function
	// run.  This allows individual runs to take precedence within the same queue.
	// The higher the number (up to consts.PriorityFactorMax), the higher priority
	// this run has.  All next steps will use this as the factor when scheduling
	// future edge jobs (on their first attempt).
	PriorityFactor *int64 `json:"pf,omitempty"`
	// CustomConcurrencyKeys stores custom concurrency keys for this function run.  This
	// allows us to use custom concurrency keys for each job when processing steps for
	// the function, with cached expression results.
	CustomConcurrencyKeys []CustomConcurrency `json:"cck,omitempty"`
}

type CustomConcurrency struct {
	// Key represents the actual evaluated concurrency key.
	Key string `json:"k"`
	// Hash represents the hash of the concurrency expression - unevaluated -
	// as defined in the function.  This lets us look up the latest concurrency
	// values as defined in the most recent version of the function and use
	// these concurrency values.  Without this, it's impossible to adjust concurrency
	// for in-progress functions.
	Hash string `json:"h"`
	// Limit represents the limit at the time the function started.  If the concurrency
	// key is removed from the fn definition, this pre-computed value will be used instead.
	//
	// NOTE: If the value is removed from the last deployed function we could also disregard
	// this concurrency key.
	Limit int `json:"l"`
}

// IdempotencyKey returns the unique key used to represent this single
// workflow run, across all steps.
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
	// Identifier stores the full identifier for the run, such that the only
	// thing needed to load the State run is the run ID.
	Identifier Identifier `json:"id"`

	// Status returns the function status for this run.
	Status enums.RunStatus `json:"status"`

	// Debugger represents whether this function was started via the debugger.
	Debugger bool `json:"debugger"`

	// RunType indicates the run type for this particular flow.  This allows
	// us to store whether this is eg. a manual retry
	RunType *string `json:"runType,omitempty"`

	// Name stores the name of the workflow as it started.
	//
	// DEPRECATED
	Name string `json:"name"`

	// Version represents the version of _metadata_ in particular.
	//
	// TODO: This should be removed and made specific to each particular state
	// implementation.
	Version int `json:"version"`

	// RequestVersion represents the executor request versioning/hashing style
	// used to manage state.
	//
	// TS v3, Go, Rust, Elixir, and Java all use the same hashing style (1).
	//
	// TS v1 + v2 use a unique hashing style (0) which cannot be transferred
	// to other languages.
	//
	// This lets us send the hashing style to SDKs so that we can execute in
	// the correct format with backcompat guarantees built in.
	//
	// NOTE: We can only know this the first time an SDK is responding to a step.
	RequestVersion int `json:"rv"`

	// Context allows storing any other contextual data in metadata.
	Context map[string]any `json:"ctx,omitempty"`

	// DisableImmediateExecution is used to tell the SDK whether it should
	// disallow immediate execution of steps as they are found.
	DisableImmediateExecution bool `json:"disableImmediateExecution,omitempty"`
}

type MetadataUpdate struct {
	Debugger                  bool           `json:"debugger"`
	Context                   map[string]any `json:"ctx,omitempty"`
	DisableImmediateExecution bool           `json:"disableImmediateExecution,omitempty"`
	RequestVersion            int            `json:"rv"`
}

// State represents the current state of a fn run.  It is data-structure
// agnostic;  each backing store can change the structure of the state to
// suit its implementation.
//
// It is assumed that, once initialized, state does not error when returning
// data for the given identifier.
type State interface {
	// Function returns the inngest function for the given run.
	Function() inngest.Function

	// Metadata returns the run metadata, including the started at time
	// as well as the pending count.
	Metadata() Metadata

	// Identifier returns the identifier for this functionrun.
	Identifier() Identifier

	// RunID returns the ID for the specific run.
	RunID() ulid.ULID

	// WorkflowID returns the workflow ID for the run
	WorkflowID() uuid.UUID

	// Stack returns a slice of step IDs representing the order in which
	// data is saved to the state store.  This, in effect, strongly orders
	// function steps so that we know the sequence of completed steps.
	Stack() []string

	// Event is the root data that triggers the workflow, which is typically
	// an Inngest event.
	Event() map[string]any

	// Events is the list of events that are used to trigger the workflow,
	// which is typically a list of Inngest event.
	Events() []map[string]any

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

	CronSchedule() *string
	IsCron() bool
}

// Manager represents a state manager which can both load and mutate state.
type Manager interface {
	FunctionLoader
	StateLoader
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

// StateLoader allows loading of previously stored state based off of a given Identifier.
type StateLoader interface {
	// Exists checks whether the run ID exists.
	Exists(ctx context.Context, runID ulid.ULID) (bool, error)

	// Metadata returns run metadata for the given identifier.  It may be cheaper
	// than a full load in cases where only the metadata is necessary.
	Metadata(ctx context.Context, runID ulid.ULID) (*Metadata, error)

	// Load returns run state for the given identifier.
	Load(ctx context.Context, runID ulid.ULID) (State, error)

	// IsComplete returns whether the given identifier is complete, ie. the
	// pending count in the identifier's metadata is zero.
	IsComplete(ctx context.Context, runID ulid.ULID) (complete bool, err error)

	// StackIndex returns the index for the given step ID within the function stack of
	// a given run.
	StackIndex(ctx context.Context, runID ulid.ULID, stepID string) (int, error)
}

// FunctionLoader loads function definitions based off of an identifier.
type FunctionLoader interface {
	LoadFunction(ctx context.Context, identifier Identifier) (*inngest.Function, error)
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

	UpdateMetadata(ctx context.Context, runID ulid.ULID, md MetadataUpdate) error

	// Cancel sets a function run metadata status to RunStatusCancelled, which prevents
	// future execution of steps.
	Cancel(ctx context.Context, i Identifier) error

	// SetStatus sets a status specifically.
	SetStatus(ctx context.Context, i Identifier, status enums.RunStatus) error

	// SaveResponse saves the driver response for the attempt to the backing state store.
	//
	// If the response is an error, this must store the error for the specific attempt, allowing
	// visibility into each error when executing a step. If DriverResponse is final, this must push
	// the step ID to the stack.
	//
	// Attempt is zero-indexed.
	//
	// This returns the position of this step in the stack, if the stack is modified.  For temporary
	// errors the stack position is 0, ie. unmodified.
	SaveResponse(ctx context.Context, i Identifier, r DriverResponse, attempt int) (int, error)
}

// Input is the input for creating new state.  The required fields are Workflow,
// Identifier and Input;  the rest of the data is stored within the state store as
// metadata.
type Input struct {
	// Identifier represents the identifier
	Identifier Identifier

	// EventBatchData is the input data for initializing the workflow run,
	// which is a list of EventData
	EventBatchData []map[string]any

	// Debugger represents whether this function was started via the debugger.
	Debugger bool

	// RunType indicates the run type for this particular flow.  This allows
	// us to store whether this is eg. a manual retry
	RunType *string `json:"runType,omitempty"`

	// Steps allows users to specify pre-defined steps to run workflows from
	// arbitrary points.
	Steps map[string]any

	// Context is additional context for the run stored in metadata.
	Context map[string]any
}
