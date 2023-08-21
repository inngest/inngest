package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

var (
	ErrRuntimeRegistered = fmt.Errorf("runtime is already registered")
	ErrNoStateManager    = fmt.Errorf("no state manager provided")
	ErrNoActionLoader    = fmt.Errorf("no action loader provided")
	ErrNoRuntimeDriver   = fmt.Errorf("runtime driver for action not found")

	ErrFunctionEnded = fmt.Errorf("function already ended")

	PauseHandleConcurrency = 100
)

// NewExecutor returns a new executor, responsible for running the specific step of a
// function (using the available drivers) and storing the step's output or error.
//
// Note that this only executes a single step of the function;  it returns which children
// can be directly executed next and saves a state.Pause for edges that have async conditions.
func NewExecutor(opts ...ExecutorOpt) (execution.Executor, error) {
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

	return m, nil
}

// ExecutorOpt modifies the built in executor on creation.
type ExecutorOpt func(m execution.Executor) error

// WithStateManager sets which state manager to use when creating an executor.
func WithStateManager(sm state.Manager) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).sm = sm
		return nil
	}
}

// WithQueue sets which state manager to use when creating an executor.
func WithQueue(q queue.Queue) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).queue = q
		return nil
	}
}

func WithFunctionLoader(l state.FunctionLoader) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).fl = l
		return nil
	}
}

func WithLogger(l *zerolog.Logger) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).log = l
		return nil
	}
}

func WithFailureHandler(f execution.FailureHandler) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).failureHandler = f
		return nil
	}
}

func WithLifecycleListeners(l ...execution.LifecycleListener) ExecutorOpt {
	return func(e execution.Executor) error {
		for _, item := range l {
			e.AddLifecycleListener(item)
		}
		return nil
	}
}

func WithStepLimits(limit uint) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).steplimit = limit
		return nil
	}
}

// WithEvaluatorFactory allows customizing of the expression evaluator factory function.
func WithEvaluatorFactory(f func(ctx context.Context, expr string) (expressions.Evaluator, error)) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).evalFactory = f
		return nil
	}
}

// WithRuntimeDrivers specifies the drivers available to use when executing steps
// of a function.
//
// When invoking a step in a function, we find the registered driver with the step's
// RuntimeType() and use that driver to execute the step.
func WithRuntimeDrivers(drivers ...driver.Driver) ExecutorOpt {
	return func(exec execution.Executor) error {
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
	queue          queue.Queue
	fl             state.FunctionLoader
	evalFactory    func(ctx context.Context, expr string) (expressions.Evaluator, error)
	runtimeDrivers map[string]driver.Driver
	failureHandler execution.FailureHandler

	lifecycles []execution.LifecycleListener

	steplimit uint
}

func (e *executor) SetFailureHandler(f execution.FailureHandler) {
	e.failureHandler = f
}

func (e *executor) AddLifecycleListener(l execution.LifecycleListener) {
	e.lifecycles = append(e.lifecycles, l)
}

// Execute loads a workflow and the current run state, then executes the
// workflow via an executor.
func (e *executor) Execute(ctx context.Context, id state.Identifier, item queue.Item, edge inngest.Edge, stackIndex int) (*state.DriverResponse, int, error) {
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
			Int("attempt", item.Attempt).
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

		// Get the index from the previous save.
		idx, _ := e.sm.StackIndex(ctx, id.RunID, edge.Incoming)

		// This has already successfully been executed.
		return &state.DriverResponse{
			Scheduled: false,
			Output:    resp,
			Err:       nil,
		}, idx, nil
	}

	resp, idx, err := e.run(ctx, id, item, edge, s, stackIndex)

	if resp.Final() && e.failureHandler != nil {
		if err := e.failureHandler(ctx, id, s, *resp); err != nil {
			logger.From(ctx).Error().Err(err).Msg("error handling failure in executor fail handler")
		}
	}

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
func (e *executor) run(ctx context.Context, id state.Identifier, item queue.Item, edge inngest.Edge, s state.State, stackIndex int) (*state.DriverResponse, int, error) {
	var (
		response *state.DriverResponse
		err      error
	)

	if e.fl == nil {
		return nil, 0, fmt.Errorf("no function loader specified running step")
	}

	if edge.Incoming == inngest.TriggerName {
		return nil, 0, nil
	}

	f, err := e.fl.LoadFunction(ctx, id)
	if err != nil {
		return nil, 0, fmt.Errorf("error loading function for run: %w", err)
	}

	var step *inngest.Step
	for _, s := range f.Steps {
		if s.ID == edge.Incoming {
			step = &s
			break
		}
	}
	if step == nil {
		// This isn't fixable.
		return nil, 0, newFinalError(fmt.Errorf("unknown vertex: %s", edge.Incoming))
	}

	for _, e := range e.lifecycles {
		go e.OnStepStarted(ctx, id, item, edge, *step, s)
	}

	response, idx, err := e.executeStep(ctx, id, item, step, s, edge, stackIndex)

	for _, e := range e.lifecycles {
		go e.OnStepFinished(ctx, id, item, edge, *step, *response)
	}

	if err != nil {
		return response, idx, err
	}
	if response.Err != nil {
		// This action errored.  We've stored this in our state manager already;
		// return the response error only.  We can use the same variable for both
		// the response and the error to indicate an error value.
		return response, idx, fmt.Errorf("%s", *response.Err)
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

func (e *executor) executeStep(ctx context.Context, id state.Identifier, item queue.Item, step *inngest.Step, s state.State, edge inngest.Edge, stackIndex int) (*state.DriverResponse, int, error) {
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

	if err := e.sm.Started(ctx, id, step.ID, item.Attempt); err != nil {
		return nil, 0, fmt.Errorf("error saving started state: %w", err)
	}

	response, err := d.Execute(ctx, s, edge, *step, stackIndex, item.Attempt)
	if response == nil {
		// Add an error response here.
		response = &state.DriverResponse{
			Step: *step,
		}
	}
	if err != nil && response.Err == nil {
		// Set the response error
		errstr := err.Error()
		response.Err = &errstr
	}

	// Ensure that the step is always set.  This removes the need for drivers to always
	// set this.
	if response.Step.ID == "" {
		response.Step = *step
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
				errstr := fmt.Sprintf("error unmarshalling generator step data as json: %s", err)
				response.Err = &errstr
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

	if response.Err != nil && (!response.Retryable() || !queue.ShouldRetry(nil, item.Attempt, step.RetryCount())) {
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

	idx, serr := e.sm.SaveResponse(ctx, id, *response, item.Attempt)
	if serr != nil {
		if l != nil {
			logger.From(ctx).Error().Err(serr).Msg("unable to save state")
		}
		err = multierror.Append(err, serr)
	}

	if response.Err == nil && len(response.Generator) == 0 {
		// Mark this step as finalized.
		// This must happen after everything is enqueued, else the scheduled <> finalized count
		// is out of order.
		if err := e.sm.Finalized(ctx, id, edge.Incoming, item.Attempt); err != nil {
			return response, stackIndex, fmt.Errorf("unable to finalize step: %w", err)
		}
	}

	return response, idx, err
}

// HandlePauses handles pauses loaded from an incoming event.
func (e *executor) HandlePauses(ctx context.Context, iter state.PauseIterator, evt event.TrackedEvent) error {
	if e.queue == nil || e.sm == nil {
		return fmt.Errorf("No queue or state manager specified")
	}

	var (
		goerr error
		wg    sync.WaitGroup
	)

	evtID := evt.InternalID()

	// Schedule up to PauseHandleConcurrency pauses at once.
	sem := semaphore.NewWeighted(int64(PauseHandleConcurrency))

	for iter.Next(ctx) {
		if goerr != nil {
			break
		}

		pause := iter.Val(ctx)

		// Block until we have capacity
		_ = sem.Acquire(ctx, 1)

		wg.Add(1)
		go func() {
			defer wg.Done()
			// Always release one from the capacity
			defer sem.Release(1)

			if pause.TriggeringEventID != nil && *pause.TriggeringEventID == evt.InternalID().String() {
				// Don't allow the original function trigger to trigger the
				// cancellation
				return
			}

			// NOTE: Some pauses may be nil or expired, as the iterator may take
			// time to process.  We handle that here and assume that the event
			// did not occur in time.
			if pause == nil || pause.Expires.Time().Before(time.Now()) {
				// Consume this pause to remove it entirely
				_ = e.sm.ConsumePause(context.Background(), pause.ID, nil)
				return
			}

			// Ensure that we store the group ID for this pause, letting us properly track cancellation
			// or continuation history
			ctx = state.WithGroupID(ctx, pause.GroupID)

			// Run an expression if this exists.
			if pause.Expression != nil {
				// Precompute the expression data once, as a value (not pointer)
				data := expressions.NewData(map[string]any{
					"async": evt.Event().Map(),
				})

				if len(pause.ExpressionData) > 0 {
					// If we have cached data for the expression (eg. the expression is evaluating workflow
					// state which we don't have access to here), unmarshal the data and add it to our
					// event data.
					data.Add(pause.ExpressionData)
				}

				expr, err := expressions.NewExpressionEvaluator(ctx, *pause.Expression)
				if err != nil {
					return
				}

				val, _, err := expr.Evaluate(ctx, data)
				if err != nil {
					// XXX: Track error here.
					return
				}
				result, _ := val.(bool)
				if !result {
					return
				}
			}

			// Cancelling a function can happen before a lease, as it's an atomic operation that will always happen.
			if pause.Cancel {
				err := e.Cancel(ctx, pause.Identifier, execution.CancelRequest{
					EventID:    &evtID,
					Expression: pause.Expression,
				})
				if err != nil {
					goerr = errors.Join(goerr, fmt.Errorf("error cancelling function: %w", err))
					return
				}
				// Ensure we consume this pause, as this isn't handled by the higher-level cancel function.
				err = e.sm.ConsumePause(ctx, pause.ID, nil)
				if err == nil || err == state.ErrPauseLeased || err == state.ErrPauseNotFound {
					// Done.
					return
				}
				goerr = errors.Join(goerr, fmt.Errorf("error consuming pause after cancel: %w", err))
				return
			}

			err := e.Resume(ctx, *pause, execution.ResumeRequest{
				With:    evt.Event().Map(),
				EventID: &evtID,
			})
			if err != nil {
				goerr = errors.Join(goerr, fmt.Errorf("error consuming pause after cancel: %w", err))
			}
		}()

	}

	wg.Wait()
	return goerr
}

// Cancel cancels an in-progress function.
func (e *executor) Cancel(ctx context.Context, id state.Identifier, r execution.CancelRequest) error {
	md, err := e.sm.Metadata(ctx, id.RunID)
	if err != nil {
		return err
	}

	switch md.Status {
	case enums.RunStatusFailed, enums.RunStatusCompleted, enums.RunStatusOverflowed:
		return ErrFunctionEnded
	case enums.RunStatusCancelled:
		return nil
	}

	// TODO: Load all pauses for the function and remove.

	if err := e.sm.Cancel(ctx, md.Identifier); err != nil {
		return fmt.Errorf("error cancelling function: %w", err)
	}

	// XXX: Write to history here.
	return nil
}

// Resume resumes an in-progress function from the given waitForEvent pause.
func (e *executor) Resume(ctx context.Context, pause state.Pause, r execution.ResumeRequest) error {
	if e.queue == nil || e.sm == nil {
		return fmt.Errorf("No queue or state manager specified")
	}

	// Lease this pause so that only this thread can schedule the execution.
	//
	// If we don't do this, there's a chance that two concurrent runners
	// attempt to enqueue the next step of the workflow.
	err := e.sm.LeasePause(ctx, pause.ID)
	if err == state.ErrPauseLeased || err == state.ErrPauseNotFound {
		// Ignore;  this is being handled by another runner.
		return nil
	}

	if pause.OnTimeout {
		// Delete this pause, as an event has occured which matches
		// the timeout.  We can do this prior to leasing a pause as it's the
		// only work that needs to happen
		err := e.sm.ConsumePause(ctx, pause.ID, nil)
		if err == nil || err == state.ErrPauseNotFound {
			return nil
		}
		return err
	}

	// Schedule an execution from the pause's entrypoint.  We do this after
	// consuming the pause to guarantee the event data is stored via the pause
	// for the next run.  If the ConsumePause call comes after enqueue, the TCP
	// conn may drop etc. and running the job may occur prior to saving state data.
	if err := e.queue.Enqueue(
		ctx,
		queue.Item{
			// Add a new group ID for the child;  this will be a new step.
			GroupID:     uuid.New().String(),
			WorkspaceID: pause.WorkspaceID,
			Kind:        queue.KindEdge,
			Identifier:  pause.Identifier,
			Payload: queue.PayloadEdge{
				Edge: inngest.Edge{
					Outgoing: pause.Outgoing,
					Incoming: pause.Incoming,
				},
			},
		},
		time.Now(),
	); err != nil {
		return fmt.Errorf("error enqueueing after pause: %w", err)
	}

	if err = e.sm.ConsumePause(ctx, pause.ID, r.With); err != nil {
		return fmt.Errorf("error consuming pause via event: %w", err)
	}
	return nil
}

func (e *executor) HandleGeneratorResponse(ctx context.Context, gen []*state.GeneratorOpcode, item queue.Item) error {
	// Ensure that we process waitForEvents first, as these are highest priority.
	sortOps(gen)

	eg := errgroup.Group{}
	for _, op := range gen {
		copied := *op
		eg.Go(func() error { return e.HandleGenerator(ctx, copied, item) })
	}

	return eg.Wait()
}

func (e *executor) HandleGenerator(ctx context.Context, gen state.GeneratorOpcode, item queue.Item) error {
	// Grab the edge that triggered this step execution.
	edge, ok := item.Payload.(queue.PayloadEdge)
	if !ok {
		return fmt.Errorf("unknown queue item type handling generator: %T", item.Payload)
	}

	switch gen.Op {
	case enums.OpcodeNone:
		// OpcodeNone essentially terminates this "thread" or execution path.  We don't need to do
		// anything - including scheduling future steps.
		//
		// This is necessary for parallelization:  we may fan out from 1 step -> 10 parallel steps,
		// then need to coalesce back to a single thread after all 10 have finished.  We expect
		// drivers/the SDK to return OpcodeNone for all but the last of parallel steps.
		return nil
	case enums.OpcodeStep:
		return e.handleGeneratorStep(ctx, gen, item, edge)
	case enums.OpcodeStepPlanned:
		return e.handleGeneratorStepPlanned(ctx, gen, item, edge)
	case enums.OpcodeSleep:
		return e.handleGeneratorSleep(ctx, gen, item, edge)
	case enums.OpcodeWaitForEvent:
		return e.handleGeneratorWaitForEvent(ctx, gen, item, edge)
	}

	return fmt.Errorf("unknown opcode: %s", gen.Op)
}

func (e *executor) handleGeneratorStep(ctx context.Context, gen state.GeneratorOpcode, item queue.Item, edge queue.PayloadEdge) error {
	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Going from the current step
		Incoming: edge.Edge.Incoming, // And re-calling the incoming function in a loop
	}

	// Re-enqueue the exact same edge to run now.
	if err := e.sm.Scheduled(ctx, item.Identifier, nextEdge.Incoming, 0, nil); err != nil {
		return err
	}
	// Update the group ID in context;  we've already saved this step's success and we're now
	// running the step again, needing a new history group
	groupID := uuid.New().String()
	err := e.queue.Enqueue(ctx, queue.Item{
		WorkspaceID: item.WorkspaceID,
		GroupID:     groupID,
		Kind:        queue.KindEdge,
		Identifier:  item.Identifier,
		Attempt:     0,
		MaxAttempts: item.MaxAttempts,
		Payload:     queue.PayloadEdge{Edge: nextEdge},
		SdkVersion:  item.SdkVersion,
	}, time.Now())
	if err == redis_state.ErrQueueItemExists {
		return nil
	}
	return err
}

func (e *executor) handleGeneratorStepPlanned(ctx context.Context, gen state.GeneratorOpcode, item queue.Item, edge queue.PayloadEdge) error {
	nextEdge := inngest.Edge{
		// Planned generator IDs are the same as the actual OpcodeStep IDs.
		// We can't set edge.Edge.Outgoing here because the step hasn't yet ran.
		//
		// We do, though, want to store the incomin step ID name _without_ overriding
		// the actual DAG step, though.
		// Run the same action.
		IncomingGeneratorStep: gen.ID,
		Outgoing:              edge.Edge.Outgoing,
		Incoming:              edge.Edge.Incoming,
	}

	// Re-enqueue the exact same edge to run now.
	if err := e.sm.Scheduled(ctx, item.Identifier, edge.Edge.Incoming, 0, nil); err != nil {
		return err
	}

	jobID := fmt.Sprintf("%s-%s", item.Identifier.IdempotencyKey(), gen.ID)
	// Update the group ID in context;  we're scheduling a new step, and we want
	// to start a new history group for this item.
	groupID := uuid.New().String()
	err := e.queue.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
		GroupID:     groupID, // Ensure we correlate future jobs with this group ID, eg. started/failed.
		WorkspaceID: item.WorkspaceID,
		Kind:        queue.KindEdge,
		Identifier:  item.Identifier,
		Attempt:     0,
		MaxAttempts: item.MaxAttempts,
		Payload: queue.PayloadEdge{
			Edge: nextEdge,
		},
		SdkVersion: item.SdkVersion,
	}, time.Now())
	if err == redis_state.ErrQueueItemExists {
		return nil
	}
	return err
}

// handleSleep handles the sleep opcode, ensuring that we enqueue the function to rerun
// at the correct time.
func (e *executor) handleGeneratorSleep(ctx context.Context, gen state.GeneratorOpcode, item queue.Item, edge queue.PayloadEdge) error {
	dur, err := gen.SleepDuration()
	if err != nil {
		return err
	}
	at := time.Now().Add(dur)

	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Leaving sleep
		Incoming: edge.Edge.Incoming, // To re-call the SDK
	}

	// XXX: Remove this after we create queues for function runs.
	if err := e.sm.Scheduled(ctx, item.Identifier, nextEdge.Incoming, 0, &at); err != nil {
		return err
	}

	// Create another group for the next item which will run.  We're enqueueing
	// the function to run again after sleep, so need a new group.
	groupID := uuid.New().String()
	err = e.queue.Enqueue(ctx, queue.Item{
		WorkspaceID: item.WorkspaceID,
		// Sleeps re-enqueue the step so that we can mark the step as completed
		// in the executor after the sleep is complete.  This will re-call the
		// generator step, but we need the same group ID for correlation.
		GroupID:     groupID,
		Kind:        queue.KindSleep,
		Identifier:  item.Identifier,
		Attempt:     0,
		MaxAttempts: item.MaxAttempts,
		Payload:     queue.PayloadEdge{Edge: nextEdge},
		SdkVersion:  item.SdkVersion,
	}, time.Now().Add(dur))
	if err == redis_state.ErrQueueItemExists {
		// Safely ignore this error.
		return nil
	}
	return err
}

func (e *executor) handleGeneratorWaitForEvent(ctx context.Context, gen state.GeneratorOpcode, item queue.Item, edge queue.PayloadEdge) error {
	opts, err := gen.WaitForEventOpts()
	if err != nil {
		return fmt.Errorf("unable to parse wait for event opts: %w", err)
	}
	expires, err := opts.Expires()
	if err != nil {
		return fmt.Errorf("unable to parse wait for event expires: %w", err)
	}

	// Filter the expression data such that it contains only the variables used
	// in the expression.
	data := map[string]any{}
	if opts.If != nil {
		expr, err := e.newExpressionEvaluator(ctx, *opts.If)
		if err != nil {
			return execError{err, true}
		}

		run, err := e.sm.Load(ctx, item.Identifier.RunID)
		if err != nil {
			return execError{err: fmt.Errorf("unable to load run after execution: %w", err)}
		}

		// Take the data for expressions based off of state.
		ed := expressions.NewData(state.EdgeExpressionData(ctx, run, ""))
		data = expr.FilteredAttributes(ctx, ed).Map()
	}

	pauseID := uuid.New()
	err = e.sm.SavePause(ctx, state.Pause{
		ID:             pauseID,
		WorkspaceID:    item.WorkspaceID,
		Identifier:     item.Identifier,
		GroupID:        item.GroupID,
		Outgoing:       gen.ID,
		Incoming:       edge.Edge.Incoming,
		StepName:       gen.Name,
		Expires:        state.Time(expires),
		Event:          &opts.Event,
		Expression:     opts.If,
		ExpressionData: data,
		DataKey:        gen.ID,
	})
	if err != nil {
		return err
	}

	// This should also increase the waitgroup count, as we have an
	// edge that is outstanding.
	//
	// TODO: Remove with function run specific queues
	if err := e.sm.Scheduled(ctx, item.Identifier, edge.Edge.IncomingGeneratorStep, 0, nil); err != nil {
		return fmt.Errorf("unable to schedule wait for event: %w", err)
	}

	// SDK-based event coordination is called both when an event is received
	// OR on timeout, depending on which happens first.  Both routes consume
	// the pause so this race will conclude by calling the function once, as only
	// one thread can lease and consume a pause;  the other will find that the
	// pause is no longer available and return.
	err = e.queue.Enqueue(ctx, queue.Item{
		WorkspaceID: item.WorkspaceID,
		// Use the same group ID, allowing us to track the cancellation of
		// the step correctly.
		GroupID:    item.GroupID,
		Kind:       queue.KindPause,
		Identifier: item.Identifier,
		Payload: queue.PayloadPauseTimeout{
			PauseID:   pauseID,
			OnTimeout: true,
		},
		SdkVersion: item.SdkVersion,
	}, expires)
	if err == redis_state.ErrQueueItemExists {
		return nil
	}
	return err
}

func (e *executor) newExpressionEvaluator(ctx context.Context, expr string) (expressions.Evaluator, error) {
	if e.evalFactory != nil {
		return e.evalFactory(ctx, expr)
	}
	return expressions.NewExpressionEvaluator(ctx, expr)
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
