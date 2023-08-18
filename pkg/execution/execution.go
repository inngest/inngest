package execution

import (
	"context"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
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
	// AddLifecycleListener adds a lifecycle listener to run on hooks.  This must
	// always add to a list of listeners vs replace listeners.
	AddLifecycleListener(l LifecycleListener)

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
	// This returns the step's response, the current stack pointer index, and any error.
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
		// stackIndex represents the stack pointer at the time this step was scheduled.
		// This lets SDKs correctly evaluate parallelism by replaying generated steps in the
		// right order.
		stackIndex int,
	) (*state.DriverResponse, int, error)

	// HandleGeneratorResponse handles all generator responses.
	HandleGeneratorResponse(ctx context.Context, gen []*state.GeneratorOpcode, item queue.Item) error
	// HandleGenerator handles an individual generator response returned from the SDK.
	HandleGenerator(ctx context.Context, gen state.GeneratorOpcode, item queue.Item) error

	// HandlePauses handles pauses loaded from an incoming event.  This delegates to Cancel and
	// Resume where necessary, depending on pauses that have been loaded and matched.
	HandlePauses(ctx context.Context, iter state.PauseIterator, event event.TrackedEvent) error
	// Cancel cancels an in-progress function run, preventing any enqueued or future steps from running.
	Cancel(ctx context.Context, id state.Identifier, r CancelRequest) error
	// Resume resumes an in-progress function run from the given waitForEvent pause.
	Resume(ctx context.Context, p state.Pause, r ResumeRequest) error
	// SetFailureHandler sets the failure handler, called when a function run permanently fails.
	SetFailureHandler(f FailureHandler)
}

// FailureHandler is a function that handles failures in the executor.
type FailureHandler func(context.Context, state.Identifier, state.State, state.DriverResponse) error

// CancelRequest stores information about the incoming cancellation request within
// history.
type CancelRequest struct {
	EventID    *ulid.ULID
	Expression *string
	UserID     *uuid.UUID
}

type ResumeRequest struct {
	With    any
	EventID *ulid.ULID
}

// LifecycleListener listens to lifecycle events on the executor.
type LifecycleListener interface {
	// Close closes the listener and flushes any pending writes.
	Close() error

	OnStepStarted(
		context.Context,
		state.Identifier,
		queue.Item,
		inngest.Edge,
		inngest.Step,
		state.State,
	)

	OnStepFinished(
		context.Context,
		state.Identifier,
		queue.Item,
		inngest.Edge,
		inngest.Step,
		state.DriverResponse,
	)

	OnWaitForEvent(
		context.Context,
		state.Identifier,
		queue.Item,
		state.GeneratorOpcode,
	)
}

type noopLifecyceListener struct{}

func (noopLifecyceListener) OnStepStarted(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	edge inngest.Edge,
	step inngest.Step,
	state state.State,
) {
}

func (noopLifecyceListener) OnStepFinished(
	context.Context,
	state.Identifier,
	queue.Item,
	inngest.Step,
	*state.DriverResponse,
	error,
) {
}

func (noopLifecyceListener) OnWaitForEvent(
	context.Context,
	state.Identifier,
	queue.Item,
	state.GeneratorOpcode,
) {
}
