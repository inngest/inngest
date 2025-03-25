package execution

import (
	"context"
	"encoding/json"
	"time"

	"github.com/inngest/inngest/pkg/execution/batch"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
)

const (
	StateErrorKey         = "error"
	StateDataKey          = "data"
	SdkInvokeTimeoutError = "InngestInvokeTimeoutError"
)

// Executor manages executing actions.  It interfaces over a state store to save
// action and workflow data once an action finishes or fails.  Once a function
// finishes, its children become available to execute.  This is not handled
// immediately;  instead, the executor returns the children which can be executed.
// The owner of the executor is responsible for managing and calling the next
// child functions.
//
// # Atomicity
//
// Functions in the executor should be considered atomic.  If the context has closed
// because the process is terminating whilst we are executing, completing, or failing
// an action we must wait for the executor to finish processing before quitting. If
// we fail to wait for the executor, workflows may finish prematurely as future
// actions may not be scheduled.
//
// # Running functions
//
// The executor schedules function execution over drivers.  A driver is a runtime-specific
// implementation which runs functions, eg. a docker driver for running contianers,
// or a webassembly driver for wasm runtimes.
//
// Runtimes can be asynchronous.  A docker container may take minutes to run, and
// the connection to docker may be interrupted.  The executor provides functionality
// for storing the outcome of an action via Resume and Fail at any point after an
// action has started.
type Executor interface {
	// Schedule is called to schedule a given function with the given event.  This
	// creates a new function run by initializing blank function state and placing
	// the run in the queue.
	//
	// Note that the executor does *not* handle rate limiting, debouncing, batching,
	// expressions, etc.  Any Schedule request will immediately be scheduled for the
	// given time. Filtering of events in any way must be handled prior scheduling.
	Schedule(ctx context.Context, r ScheduleRequest) (*sv2.Metadata, error)

	// Execute runs the given function via the execution drivers.  If the
	// from ID is "$trigger" this is treated as a new workflow invocation from the
	// trigger, and all functions that are direct children of the trigger will be
	// scheduled for execution.
	//
	// Attempt is the zero-index attempt number for this execution.  The executor
	// needs knowledge of the attempt number to store the error for each attempt,
	// and to figure out whether this is the final retry for determining whether
	// the next error is "finalized".
	//
	// It is important for this function to be atomic;  if the function was scheduled
	// and the context terminates, we must store the output or async data in workflow
	// state then schedule the child functions else the workflow will terminate early.
	//
	// Execution will fail with no response and state.ErrFunctionCancelled if this function
	// run has been cancelled by an external event or process.
	//
	// This returns the step's response and any error.
	Execute(
		ctx context.Context,
		id state.Identifier,
		// item is the queue item which scheduled the execution of this step.
		// all steps are scheduled by a queue item.
		item queue.Item,
		// edge represents the edge to run.  This executes the step defined within
		// Incoming, optionally using the StepPlanned field to execute a substep if
		// the step is a generator.
		edge inngest.Edge,
	) (*state.DriverResponse, error)

	// HandlePauses handles pauses loaded from an incoming event.  This delegates to Cancel and
	// Resume where necessary, depending on pauses that have been loaded and matched.
	HandlePauses(ctx context.Context, iter state.PauseIterator, event event.TrackedEvent) (HandlePauseResult, error)
	// HandleInvokeFinish handles the invoke pauses from an incoming event. This delegates to Cancel and
	// Resume where necessary
	HandleInvokeFinish(ctx context.Context, event event.TrackedEvent) error
	// Cancel cancels an in-progress function run, preventing any enqueued or future steps from running.
	Cancel(ctx context.Context, id sv2.ID, r CancelRequest) error
	// Resume resumes an in-progress function run from the given waitForEvent pause.
	Resume(ctx context.Context, p state.Pause, r ResumeRequest) error

	// AddLifecycleListener adds a lifecycle listener to run on hooks.  This must
	// always add to a list of listeners vs replace listeners.
	AddLifecycleListener(l LifecycleListener)

	CloseLifecycleListeners(ctx context.Context)

	// SetFinalizer sets the function which publishes finalization events on
	// run completion
	SetFinalizer(f FinalizePublisher)

	// InvokeFailHandler invokes the invoke fail handler.
	InvokeFailHandler(context.Context, InvokeFailHandlerOpts) error

	AppendAndScheduleBatch(ctx context.Context, fn inngest.Function, bi batch.BatchItem, opts *BatchExecOpts) error

	RetrieveAndScheduleBatch(ctx context.Context, fn inngest.Function, payload batch.ScheduleBatchPayload, opts *BatchExecOpts) error
}

// PublishFinishedEventOpts represents the options for publishing a finished event.
type InvokeFailHandlerOpts struct {
	OriginalEvent event.TrackedEvent
	FunctionID    string
	RunID         string
	Err           map[string]any
	Result        any
}

// BatchExecOpts communicates state and options that are relevant only when scheduling a batch
// to be worked on *imminently* (i.e. ~now, not at some future time).
type BatchExecOpts struct {
	FunctionPausedAt *time.Time
}

// FinalizePublisher is a function that handles functions finishing in the executor.
// It should be used to send the given events.
type FinalizePublisher func(context.Context, sv2.ID, []event.Event) error

// InvokeFailHandler is a function that handles invocations failing due to the
// function failing to run (not found, rate-limited). It is passed a list of
// events to send.
type InvokeFailHandler func(context.Context, InvokeFailHandlerOpts, []event.Event) error

// HandleSendingEvent handles sending an event given an event and the queue
// item.
type HandleSendingEvent func(context.Context, event.Event, queue.Item) error

// PreDeleteStateSizeReporter reports the state size before deleting state
type PreDeleteStateSizeReporter func(context.Context, sv2.Metadata)

// ScheduleRequest represents all data necessary to schedule a new function.
type ScheduleRequest struct {
	Function inngest.Function
	// At allows functions to be scheduled in the future.
	At *time.Time
	// AccountID is the account that the request belongs to.
	AccountID uuid.UUID
	// WorkspaceID is the workspace that this request belongs to.
	WorkspaceID uuid.UUID
	// AppID is the app that this request belongs to.
	AppID uuid.UUID

	// OriginalRunID is the ID of the ID of the original run, if this a replay.
	OriginalRunID *ulid.ULID
	// ReplayID is the ID of the ID of the replay, if this a replay.
	ReplayID *uuid.UUID
	// FromStep is the step that this function is being scheduled from.
	FromStep *ScheduleRequestFromStep

	// Events represent one or more events that the function is being triggered with.
	Events []event.TrackedEvent
	// BatchID refers to the batch ID, if this function is started as a batch.
	BatchID *ulid.ULID
	// IdempotencyKey represents an optional idempotency key for the function.
	IdempotencyKey *string
	// Context represents additional context used when initializing function runs.
	Context map[string]any
	// PreventDebounce prevents debouncing this function and immediately schedules
	// execution.  This is used after the debounce has finished to force execution
	// of the function, instead of debouncing again.
	PreventDebounce bool
	// FunctionPausedAt indicates whether the function is paused.
	FunctionPausedAt *time.Time
}

type ScheduleRequestFromStep struct {
	// StepID is the ID of the step that this function is being scheduled from.
	StepID string

	// Input is the input data for the step. Can be partial JSON, in which case
	// an SDK will merge this with the existing input data.
	Input json.RawMessage
}

// CancelRequest stores information about the incoming cancellation request within
// history.
type CancelRequest struct {
	EventID        *ulid.ULID
	Expression     *string
	UserID         *uuid.UUID
	CancellationID *ulid.ULID

	// ForceLifecycleHook is used to force the OnFunctionCancelled lifecycle
	// hook to run even if the function is already finalized. This is useful
	// when a user wants to cancel a "false stuck" function run (i.e. it isn't
	// in the state store but the history store thinks it's running)
	ForceLifecycleHook bool
}

type ResumeRequest struct {
	With    any
	EventID *ulid.ULID
	// RunID is the ID of the run that causes this resume, used for invoking
	// functions directly.
	RunID     *ulid.ULID
	StepName  string
	IsTimeout bool
}

func (r *ResumeRequest) Error() string {
	return r.withKey(StateErrorKey)
}

func (r *ResumeRequest) HasError() bool {
	return r.Error() != ""
}

// Set `r.With` to `error` given a `name` and `message`
func (r *ResumeRequest) SetError(name string, message string) {
	r.With = map[string]any{
		StateErrorKey: state.StandardError{
			Name:    name,
			Message: message,
			Error:   name + ": " + message,
		},
	}
}

// Set `r.With` to an invoke timeout `error`
func (r *ResumeRequest) SetInvokeTimeoutError() {
	r.SetError(
		SdkInvokeTimeoutError,
		"Timed out waiting for invoked function to complete",
	)
}

func (r *ResumeRequest) Data() string {
	return r.withKey(StateDataKey)
}

func (r *ResumeRequest) HasData() bool {
	return r.Data() != ""
}

// Set `r.With` to `data` given any data to be set
func (r *ResumeRequest) SetData(data any) {
	r.With = map[string]any{
		StateDataKey: data,
	}
}

func (r *ResumeRequest) withKey(key string) string {
	if r.With != nil {
		if withData, ok := r.With.(map[string]any)[key]; ok {
			byt, err := json.Marshal(withData)
			if err == nil {
				return string(byt)
			}
			return ""
		}
	}

	return ""
}

// HandlePauseResult returns status information about pause handling.
type HandlePauseResult [2]int32

// Processed returns the number of pauses processed.
func (h HandlePauseResult) Processed() int32 {
	return h[0]
}

// Processed returns the number of pauses handled, eg. pauses that matched
// and successfully impacted runs (either by cancellation or continuing).
func (h HandlePauseResult) Handled() int32 {
	return h[1]
}
