package executor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/coredata"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/rs/zerolog"
)

var (
	ErrRuntimeRegistered = fmt.Errorf("runtime is already registered")
	ErrNoStateManager    = fmt.Errorf("no state manager provided")
	ErrNoActionLoader    = fmt.Errorf("no action loader provided")
	ErrNoRuntimeDriver   = fmt.Errorf("runtime driver for action not found")
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
		// edge represents the edge to run.  This executes the step defined within
		// Incoming, optionally using the StepPlanned field to execute a substep if
		// the step is a generator.
		edge inngest.Edge,
		// attempt represents the attempt number for this step.
		attempt int,
		// stackIndex represents the stack pointer at the time this step was scheduled.
		// This lets SDKs correctly evaluate parallelism by replaying generated steps in the
		// right order.
		stackIndex int,
	) (*state.DriverResponse, int, error)
}

// FunctionLoader returns an inngest.Function for a given function run.
type FunctionLoader func(ctx context.Context, id state.Identifier) (inngest.Function, error)

// FailureHandler is a function that handles failures in the executor.
type FailureHandler func(context.Context, state.Identifier, state.State, state.DriverResponse) error

// NewExecutor returns a new executor, responsible for running the specific step of a
// function (using the available drivers) and storing the step's output or error.
//
// Note that this only executes a single step of the function;  it returns which children
// can be directly executed next and saves a state.Pause for edges that have async conditions.
func NewExecutor(opts ...ExecutorOpt) (Executor, error) {
	m := &executor{
		runtimeDrivers: map[string]driver.Driver{},
	}

	for _, o := range opts {
		if err := o(m); err != nil {
			return nil, err
		}
	}

	if m.sm == nil {
		return nil, ErrNoStateManager
	}

	if m.al == nil {
		return nil, ErrNoActionLoader
	}

	return m, nil
}

// ExecutorOpt modifies the built in executor on creation.
type ExecutorOpt func(m Executor) error

// WithActionLoader sets the action loader to use when retrieving function definitions
// in a workflow.
func WithActionLoader(al coredata.ExecutionActionLoader) ExecutorOpt {
	return func(e Executor) error {
		e.(*executor).al = al
		return nil
	}
}

// WithStateManager sets which state manager to use when creating an executor.
func WithStateManager(sm state.Manager) ExecutorOpt {
	return func(e Executor) error {
		e.(*executor).sm = sm
		return nil
	}
}

func WithFunctionLoader(l FunctionLoader) ExecutorOpt {
	return func(e Executor) error {
		e.(*executor).fl = l
		return nil
	}
}

func WithLogger(l *zerolog.Logger) ExecutorOpt {
	return func(e Executor) error {
		e.(*executor).log = l
		return nil
	}
}

func WithFailureHandler(f FailureHandler) ExecutorOpt {
	return func(e Executor) error {
		e.(*executor).failureHandler = f
		return nil
	}
}

func WithStepLimits(limit uint) ExecutorOpt {
	return func(e Executor) error {
		e.(*executor).steplimit = limit
		return nil
	}
}

// WithRuntimeDrivers specifies the drivers available to use when executing steps
// of a function.
//
// When invoking a step in a function, we find the registered driver with the step's
// RuntimeType() and use that driver to execute the step.
func WithRuntimeDrivers(drivers ...driver.Driver) ExecutorOpt {
	return func(exec Executor) error {
		e := exec.(*executor)
		for _, d := range drivers {
			if _, ok := e.runtimeDrivers[d.RuntimeType()]; ok {
				return ErrRuntimeRegistered
			}
			e.runtimeDrivers[d.RuntimeType()] = d

		}
		return nil
	}
}

// executor represents a built-in executor for running workflows.
type executor struct {
	log *zerolog.Logger

	sm             state.Manager
	al             coredata.ExecutionActionLoader
	fl             FunctionLoader
	runtimeDrivers map[string]driver.Driver
	failureHandler FailureHandler

	steplimit uint
}

// Execute loads a workflow and the current run state, then executes the
// workflow via an executor.
func (e *executor) Execute(ctx context.Context, id state.Identifier, edge inngest.Edge, attempt int, stackIndex int) (*state.DriverResponse, int, error) {
	var log *zerolog.Logger

	s, err := e.sm.Load(ctx, id.RunID)
	if err != nil {
		return nil, 0, err
	}

	md := s.Metadata()

	if md.Status == enums.RunStatusCancelled {
		return nil, 0, state.ErrFunctionCancelled
	}

	if e.steplimit != 0 && len(s.Actions()) >= int(e.steplimit) {
		// Update this function's state to overflowed, if running.
		if md.Status == enums.RunStatusRunning {
			if err := e.sm.SetStatus(ctx, id, enums.RunStatusOverflowed); err != nil {
				return nil, 0, err
			}
		}
		return nil, 0, state.ErrFunctionOverflowed
	}

	if e.log != nil {
		l := e.log.With().
			Str("run_id", id.RunID.String()).
			Interface("edge", edge).
			Str("step", edge.Incoming).
			Str("fn_name", s.Function().Name).
			Str("fn_id", s.Function().ID.String()).
			Int("attempt", attempt).
			Logger()
		log = &l
		log.Debug().Msg("executing step")
	}

	// This could have been retried due to a state load error after
	// the particular step's code has ran; we need to load state after
	// each action to properly evaluate the next set of edges.
	//
	// To fix this particular consistency issue, always check to see
	// if there's output stored for this action ID.
	if resp, _ := s.ActionID(edge.Incoming); resp != nil {
		if log != nil {
			log.Warn().Msg("step already executed")
		}
		// TODO: Index ???
		// This has already successfully been executed.
		return &state.DriverResponse{
			Scheduled: false,
			Output:    resp,
			Err:       nil,
		}, 0, nil
	}

	resp, idx, err := e.run(ctx, id, edge, s, attempt, stackIndex)
	if log != nil {
		var (
			l   *zerolog.Event
			msg string
		)
		// We want a different log level depending on whether this step
		// errored.
		if err == nil {
			l = log.Debug()
			msg = "executed step"
		} else {
			retryable := true
			if resp != nil {
				retryable = resp.Retryable()
			}
			l = log.Warn().Err(err).Bool("retryable", retryable)
			msg = "error executing step"
		}
		l.Str("run_id", id.RunID.String()).Str("step", edge.Incoming).Msg(msg)

		if resp != nil {
			/*
				// Log the output separately, highlighting it with
				// a different caller.  This lets users scan for
				// output easily, and if we build a TUI to filter on
				// caller to show only step outputs, etc.
				log.Info().
					Str("caller", "output").
					Interface("generator", resp.Generator).
					Interface("output", resp.Output).
					Str("run_id", id.RunID.String()).
					Str("step", edge.Incoming).
					Msg("step output")
			*/
		}
	}

	if err != nil {
		// This is likely a state.DriverResponse, which itself includes
		// whether the action can be retried based off of the output.
		//
		// The runner is responsible for scheduling jobs and will check
		// whether the action can be retried.
		return resp, idx, err
	}

	return resp, idx, nil
}

// run executes the step with the given step ID.
func (e *executor) run(ctx context.Context, id state.Identifier, edge inngest.Edge, s state.State, attempt int, stackIndex int) (*state.DriverResponse, int, error) {
	var (
		response *state.DriverResponse
		err      error
	)

	if edge.Incoming == inngest.TriggerName {
		return nil, 0, nil
	}

	f, err := e.fl(ctx, id)
	if err != nil {
		return nil, 0, fmt.Errorf("error loading function for run: %w", err)
	}

	var step *inngest.Step
	for _, s := range f.Steps {
		if s.ID == edge.Incoming {
			// TODO
			step = &s
			break
		}
	}
	if step == nil {
		// This isn't fixable.
		return nil, 0, newFinalError(fmt.Errorf("unknown vertex: %s", edge.Incoming))
	}
	response, idx, err := e.executeStep(ctx, id, step, s, edge, attempt, stackIndex)
	if err != nil {
		return response, idx, err
	}
	if response.Err != nil {
		if response.Final() && e.failureHandler != nil {
			if err := e.failureHandler(ctx, id, s, *response); err != nil {
				logger.From(ctx).Error().Err(err).Msg("error handling failure in executor fail handler")
			}
		}

		// This action errored.  We've stored this in our state manager already;
		// return the response error only.  We can use the same variable for both
		// the response and the error to indicate an error value.
		return response, idx, response
	}
	if response.Scheduled {
		// This action is not yet complete, so we can't traverse
		// its children.  We assume that the driver is responsible for
		// retrying and coordinating async state here;  the executor's
		// job is to execute the action only.
		return response, idx, nil
	}

	return response, idx, err
}

func (e *executor) executeStep(ctx context.Context, id state.Identifier, step *inngest.Step, s state.State, edge inngest.Edge, attempt, stackIndex int) (*state.DriverResponse, int, error) {
	var l *zerolog.Logger
	if e.log != nil {
		log := e.log.With().
			Str("run_id", id.RunID.String()).
			Logger()
		l = &log
	}

	d, ok := e.runtimeDrivers[step.Driver()]
	if !ok {
		return nil, 0, fmt.Errorf("%w: '%s'", ErrNoRuntimeDriver, step.Driver())
	}

	if l != nil {
		l.Info().Str("uri", step.URI).Msg("executing action")
	}

	if err := e.sm.Started(ctx, id, step.ID, attempt); err != nil {
		return nil, 0, fmt.Errorf("error saving started state: %w", err)
	}

	response, err := d.Execute(ctx, s, edge, *step, stackIndex)
	if response == nil {
		// Add an error response here.
		response = &state.DriverResponse{
			Step: *step,
			Err:  err,
		}
	}
	if err != nil && response.Err == nil {
		// Set the response error
		response.Err = err
	}

	// Ensure that the step is always set.  This removes the need for drivers to always
	// set this.
	response.Step = *step

	// NOTE: We must ensure that the Step ID is overwritten for Generator steps.  Generator
	// steps are executed many times until they stop yielding;  each execution returns a
	// new step ID and data that must be stored independently of the parent generator ID.
	//
	// By updating the step ID, we ensure that the data will be saved to the generator's ID.
	//
	// We only do this for individual generator steps, as these denote that a single step ran.
	if len(response.Generator) == 1 {
		response.Step.ID = response.Generator[0].ID
		response.Step.Name = response.Generator[0].Name
		// Unmarshal the generator data into the step.
		if response.Generator[0].Data != nil {
			err = json.Unmarshal(response.Generator[0].Data, &response.Output)
			if err != nil {
				response.Err = fmt.Errorf("error unmarshalling generator step data as json: %w", err)
			}
		}

		// If this is a plan step or a noop, we _dont_ want to save the response.  That's because the
		// step in question didn't actually run.
		if response.Generator[0].Op != enums.OpcodeStep {
			return response, stackIndex, err
		}
	}
	if len(response.Generator) > 1 {
		return response, stackIndex, err
	}

	// This action may have executed _asynchronously_.  That is, it may have been
	// scheduled for execution but the result is pending.  In this case we cannot
	// traverse this node's children as we don't have the response yet.
	//
	// This happens when eg. a docker image takes a long time to run, and/or running
	// the container (via a scheduler) isn't a blocking operation.
	//
	// XXX: We can add a state interface to indicate that the step is pending.
	if response.Scheduled {
		return response, 0, nil
	}

	if response.Err != nil && !queue.ShouldRetry(err, attempt, step.RetryCount()) {
		// We need to detect whether this error is 'final' here, depending on whether
		// we've hit the retry count or this error is deemed non-retryable.
		//
		// When this error is final, we need to store the error and update the step
		// as finalized within the state store.
		response.SetFinal()
	}

	if l != nil {
		logger.From(ctx).Trace().Msg("saving response to state")
	}

	idx, serr := e.sm.SaveResponse(ctx, id, *response, attempt)
	if serr != nil {
		if l != nil {
			logger.From(ctx).Error().Err(serr).Msg("unable to save state")
		}
		err = multierror.Append(err, serr)
	}

	return response, idx, err
}

type execError struct {
	err   error
	final bool
}

func (e execError) Unwrap() error {
	return e.err
}

func (e execError) Error() string {
	return e.err.Error()
}

func (e execError) Retryable() bool {
	return !e.final
}

func newFinalError(err error) error {
	return execError{err: err, final: true}
}
