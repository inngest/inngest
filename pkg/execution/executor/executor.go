package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/execution/actionloader"
	"github.com/inngest/inngest-cli/pkg/execution/driver"
	"github.com/inngest/inngest-cli/pkg/execution/state"
	"github.com/inngest/inngest-cli/pkg/expressions"
	"github.com/xhit/go-str2duration/v2"
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
	// from ID is 0 this is treated as a new workflow invocation from the trigger,
	// and all functions that are direct children of the trigger will be scheduled
	// for execution.
	//
	// It is important for this function to be atomic;  if the function was scheduled
	// and the context terminates, we must store the output or async data in workflow
	// state then schedule the child functions else the workflow will terminate early.
	Execute(ctx context.Context, id state.Identifier, from string) ([]inngest.Edge, error)

	ExpressionData(ctx context.Context, id state.Identifier) (map[string]interface{}, error)
}

func NewExecutor(opts ...ExecutorOpt) (Executor, error) {
	m := &executor{
		runtimeDrivers: map[string]driver.Driver{},
		exprDataGen:    ExpressionData,
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

// ExpressionDataGenerator is a function which is used to generate data for expressions within a workflow.
type ExpressionDataGenerator func(ctx context.Context, s state.State, e inngest.GraphEdge) map[string]interface{}

// WithActionLoader sets the action loader to use when retrieving function definitions
// in a workflow.
func WithActionLoader(al actionloader.ActionLoader) ExecutorOpt {
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

// WithStateManager sets which state manager to use when creating an executor.
func WithExpressionDataGenerator(datagen ExpressionDataGenerator) ExecutorOpt {
	return func(e Executor) error {
		e.(*executor).exprDataGen = datagen
		return nil
	}
}

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
	sm             state.Manager
	al             actionloader.ActionLoader
	runtimeDrivers map[string]driver.Driver
	exprDataGen    ExpressionDataGenerator
}

// Execute loads a workflow and the current run state, then executes the
// workflow via an executor.  This returns all available steps we can run from
// the workflow after the step has been executed.
func (e *executor) Execute(ctx context.Context, id state.Identifier, from string) ([]inngest.Edge, error) {
	state, err := e.sm.Load(ctx, id)
	if err != nil {
		return nil, err
	}

	w, err := state.Workflow()
	if err != nil {
		return nil, err
	}

	next, err := e.run(ctx, w, id, from, state)
	if err != nil {
		// This is likely a driver.Response, which itself includes
		// whether the action can be retried based off of the output.
		//
		// The runner is responsible for scheduling jobs and will check
		// whether the action can be retried.
		return nil, err
	}

	return next, nil
}

func (e *executor) ExpressionData(ctx context.Context, id state.Identifier) (map[string]interface{}, error) {
	state, err := e.sm.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	return e.exprDataGen(ctx, state, inngest.GraphEdge{}), nil
}

// run executes the action with the given client ID.
func (e *executor) run(ctx context.Context, w inngest.Workflow, id state.Identifier, clientID string, s state.State) ([]inngest.Edge, error) {
	if clientID != inngest.TriggerName {
		var step *inngest.Step
		for _, s := range w.Steps {
			if s.ClientID == clientID {
				step = &s
				break
			}
		}
		if step == nil {
			return nil, fmt.Errorf("unknown vertex: %s", clientID)
		}
		response, err := e.executeAction(ctx, id, step)
		if err != nil {
			return nil, err
		}
		if response.Err != nil {
			// This action errored.  We've stored this in our state manager already;
			// return the response error only.
			return nil, response
		}
		if response.Scheduled {
			// This action is not yet complete, so we can't traverse
			// its children.  We assume that the driver is responsible for
			// retrying and coordinating async state here;  the executor's
			// job is to execute the action only.
			return nil, nil
		}
	}
	return e.availabileChildren(ctx, w, id, clientID, s)
}

func (e *executor) executeAction(ctx context.Context, id state.Identifier, action *inngest.Step) (*driver.Response, error) {
	definition, err := e.al.Load(ctx, action.DSN, action.Version)
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

	state, err := e.sm.Load(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error loading state: %w", err)
	}

	response, err := d.Execute(ctx, state, *definition, *action)
	if err != nil {
		return nil, fmt.Errorf("error executing action: %w", err)
	}

	// This action may have executed _asynchronously_.  That is, it may have been
	// scheduled for execution but the result is pending.  In this case we cannot
	// traverse this node's children as we don't have the response yet.  We must
	// save workflow state indicating that the result is pending (TODO).
	//
	// This happens when eg. a docker image takes a long time to run, and/or running
	// the container (via a scheduler) isn't a blocking operation.
	if response.Scheduled {
		return response, nil
	}

	if _, serr := e.sm.SaveActionOutput(ctx, id, action.ClientID, response.Output); serr != nil {
		err = multierror.Append(err, serr)
	}

	// Store the output or the error.
	if response.Err != nil {
		if _, serr := e.sm.SaveActionError(ctx, id, action.ClientID, response.Err); serr != nil {
			err = multierror.Append(err, serr)
		}
	}

	return response, err
}

// availabileChildren iterates through all children of the given client ID, determining which
// children can be executed based off of the current workflow state.  Some children may not
// be executed due to conditional expressions etc.
func (e *executor) availabileChildren(ctx context.Context, w inngest.Workflow, id state.Identifier, clientID string, s state.State) ([]inngest.Edge, error) {
	g, err := inngest.NewGraph(w)
	if err != nil {
		return nil, err
	}

	// Handle the outgoing edges from this particular node.
	edges := g.From(clientID)
	if len(edges) == 0 {
		return nil, nil
	}

	state, err := e.sm.Load(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error loading state: %w", err)
	}

	future := []inngest.Edge{}
	for _, edge := range edges {
		// TODO: Is this an async edge?
		ok, err := e.canTraverseEdge(ctx, state, edge)
		if err != nil {
			return nil, err
		}

		if !ok {
			continue
		}

		// We can traverse this edge.  Schedule a new execution from this node.
		// Scheduling executions needs to be done regardless of whether
		// the context has cancelled.
		future = append(future, edge.WorkflowEdge)
	}

	return future, nil
}

// canTraverseEdge determines whether the edge can be traversed immediately.  Edges come
// in three flavours:  plain graph edges which link functions in a DAG;  edges with
// expressions which are traversed conditionally based off of workflow state;  and
// asynchronous edges which wait for an event mathing a condition to be traversed (at some
// point in the future, with a TTL).
func (e *executor) canTraverseEdge(ctx context.Context, s state.State, edge inngest.GraphEdge) (bool, error) {
	if edge.Outgoing.ID() != inngest.TriggerName && !s.ActionComplete(edge.Outgoing.ID()) {
		return false, nil
	}

	exprdata := e.exprDataGen(ctx, s, edge)

	if edge.WorkflowEdge.Metadata.If != "" {
		ok, _, err := expressions.Evaluate(ctx, edge.WorkflowEdge.Metadata.If, exprdata)
		if err != nil || !ok {
			return ok, err
		}
	}

	// We want to wait for another event to come in to traverse this edge within the DAG.
	//
	// Create a new "pause", which informs the state manager that we're pausing the traversal
	// of this edge until later.
	//
	// The runner should load all pauses and automatically resume the traversal when a
	// matching event is received.
	if edge.WorkflowEdge.Metadata.AsyncEdgeMetadata != nil {
		am := edge.WorkflowEdge.Metadata.AsyncEdgeMetadata

		if am.Event == "" {
			return false, fmt.Errorf("no async edge event specified")
		}
		dur, err := str2duration.ParseDuration(am.TTL)
		if err != nil {
			return false, fmt.Errorf("error parsing async edge ttl '%s': %w", am.TTL, err)
		}

		err = e.sm.SavePause(ctx, state.Pause{
			ID:         uuid.New(),
			Identifier: s.Identifier(),
			Target:     edge.Incoming.ID(),
			Expires:    time.Now().Add(dur),
			Event:      &am.Event,
			Expression: am.Match,
		})
		if err != nil {
			return false, fmt.Errorf("error saving edge pause: %w", err)
		}
		return false, nil
	}

	return true, nil
}

func ExpressionData(ctx context.Context, s state.State, e inngest.GraphEdge) map[string]interface{} {
	// Add the outgoing edge's data as a "response" field for predefined edges.
	var response map[string]interface{}
	if e.Outgoing.Step != nil {
		response, _ = s.ActionID(e.Outgoing.ID())
	}
	data := map[string]interface{}{
		"event":    s.Event(),
		"steps":    s.Actions(),
		"response": response,
	}
	return data
}
