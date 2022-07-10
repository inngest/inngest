package executor

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/coredata"
	"github.com/inngest/inngest-cli/pkg/execution/driver"
	"github.com/inngest/inngest-cli/pkg/execution/state"
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
	Execute(ctx context.Context, id state.Identifier, from string, attempt int) (*state.DriverResponse, error)
}

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

func WithLogger(l *zerolog.Logger) ExecutorOpt {
	return func(e Executor) error {
		e.(*executor).log = l
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
	runtimeDrivers map[string]driver.Driver
}

// Execute loads a workflow and the current run state, then executes the
// workflow via an executor.  This returns all available steps we can run from
// the workflow after the step has been executed.
func (e *executor) Execute(ctx context.Context, id state.Identifier, from string, attempt int) (*state.DriverResponse, error) {
	if e.log != nil {
		e.log.Info().
			Str("run_id", id.RunID.String()).
			Str("step", from).
			Int("attempt", attempt).
			Msg("executing step")
	}

	s, err := e.sm.Load(ctx, id)
	if err != nil {
		return nil, err
	}

	w := s.Workflow()

	// This could have been retried due to a state load error after
	// the particular step's code has ran; we need to load state after
	// each action to properly evaluate the next set of edges.
	//
	// To fix this particular consistency issue, always check to see
	// if there's output stored for this action ID.
	if resp, _ := s.ActionID(from); resp != nil {
		if e.log != nil {
			e.log.Warn().
				Str("run_id", id.RunID.String()).
				Str("step", from).
				Int("attempt", attempt).
				Msg("step already executed")
		}
		// This has already successfully been executed.
		return &state.DriverResponse{
			Scheduled: false,
			Output:    resp,
			Err:       nil,
			// TODO: This data isn't necessarily available in the state
			// store.  Should we mandate that this is saved?
			//
			// We're only short-circuiting execution here, which means
			// everything has been recorded and all logs should be up
			// to date;  this *shouldn't* be an issue (but... we need
			// to check).
			ActionVersion: nil,
		}, nil
	}

	resp, err := e.run(ctx, w, id, from, s, attempt)

	if e.log != nil {
		if err == nil {
			e.log.Info().
				Str("run_id", id.RunID.String()).
				Str("step", from).
				Bool("scheduled", resp.Scheduled).
				Msg("executed step")
		} else {
			retryable := false
			if resp != nil {
				retryable = resp.Retryable()
			}

			e.log.Info().
				Str("run_id", id.RunID.String()).
				Str("step", from).
				Err(err).
				Interface("response", resp).
				Bool("retryable", retryable).
				Msg("error executing step")
		}
	}

	if err != nil {
		// This is likely a state.DriverResponse, which itself includes
		// whether the action can be retried based off of the output.
		//
		// The runner is responsible for scheduling jobs and will check
		// whether the action can be retried.
		return resp, err
	}

	return resp, nil
}

// run executes the step with the given step ID.
func (e *executor) run(ctx context.Context, w inngest.Workflow, id state.Identifier, stepID string, s state.State, attempt int) (*state.DriverResponse, error) {
	var (
		response *state.DriverResponse
		err      error
	)

	if stepID != inngest.TriggerName {
		var step *inngest.Step
		for _, s := range w.Steps {
			if s.ID == stepID {
				step = &s
				break
			}
		}
		if step == nil {
			return nil, fmt.Errorf("unknown vertex: %s", stepID)
		}
		response, err = e.executeAction(ctx, id, step, s, attempt)
		if err != nil {
			return nil, err
		}
		if response.Err != nil {
			// This action errored.  We've stored this in our state manager already;
			// return the response error only.  We can use the same variable for both
			// the response and the error to indicate an error value.
			return response, response
		}
		if response.Scheduled {
			// This action is not yet complete, so we can't traverse
			// its children.  We assume that the driver is responsible for
			// retrying and coordinating async state here;  the executor's
			// job is to execute the action only.
			return response, nil
		}
	}

	return response, err
}

func (e *executor) executeAction(ctx context.Context, id state.Identifier, action *inngest.Step, s state.State, attempt int) (*state.DriverResponse, error) {
	definition, err := e.al.Action(ctx, action.DSN, action.Version)
	if err != nil {
		return nil, fmt.Errorf("error loading action: %w", err)
	}
	if definition == nil {
		return nil, fmt.Errorf("no action returned: %s", action.DSN)
	}

	d, ok := e.runtimeDrivers[definition.Runtime.RuntimeType()]
	if !ok {
		return nil, fmt.Errorf("%w: '%s'", ErrNoRuntimeDriver, definition.Runtime.RuntimeType())
	}

	if e.log != nil {
		e.log.Debug().
			Str("dsn", definition.DSN).
			Interface("version", definition.Version).
			Interface("scopes", definition.Scopes).
			Str("run_id", id.RunID.String()).
			Str("step", action.ID).
			Msg("executing action")
	}

	response, err := d.Execute(ctx, s, *definition, *action)
	if err != nil || response == nil {
		return nil, fmt.Errorf("error executing action: %w", err)
	}

	// Ensure that the step is always set.  This removes the need for drivers to always
	// set this.
	response.Step = *action

	if response.ActionVersion == nil {
		// Set the ActionVersion automatically from the executor, where
		// provided from the definition.
		response.ActionVersion = definition.Version
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
		return response, nil
	}

	if response.Err != nil && (!response.Retryable() || attempt >= action.RetryCount()-1) {
		// We need to detect whether this error is 'final' here, depending on whether
		// we've hit the retry count or this error is deemed non-retryable.
		//
		// When this error is final, we need to store the error and update the step
		// as finalized within the state store.
		response.SetFinal()
	}

	if _, serr := e.sm.SaveResponse(ctx, id, *response, attempt); serr != nil {
		err = multierror.Append(err, serr)
	}

	return response, err
}
