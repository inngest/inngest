package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog"
	"github.com/xhit/go-str2duration/v2"
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

var (
	// SourceEdgeRetries represents the number of times we'll retry running a source edge.
	// Each edge gets their own set of retries in our execution engine, embedded directly
	// in the job.  The retry count is taken from function config for every step _but_
	// initialization.
	sourceEdgeRetries = 20
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
// function's step via the necessary driver.
func (e *executor) Schedule(ctx context.Context, req execution.ScheduleRequest) (*state.Identifier, error) {
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	var key string
	if req.IdempotencyKey != nil {
		// Use the given idempotency key
		key = *req.IdempotencyKey
	}
	if req.OriginalRunID != nil {
		// If this is a rerun then we want to use the run ID as the key. If we
		// used the event or batch ID as the key then we wouldn't be able to
		// rerun multiple times.
		key = runID.String()
	}
	if key == "" && len(req.Events) == 1 {
		// If not provided, use the incoming event ID if there's not a batch.
		key = req.Events[0].GetInternalID().String()
	}
	if key == "" && req.BatchID != nil {
		// Finally, if there is a batch use the batch ID as the idempotency key.
		key = req.BatchID.String()
	}

	id := state.Identifier{
		WorkflowID:      req.Function.ID,
		WorkflowVersion: req.Function.FunctionVersion,
		StaticVersion:   req.StaticVersion,
		RunID:           runID,
		BatchID:         req.BatchID,
		EventID:         req.Events[0].GetInternalID(),
		Key:             key,
		AccountID:       req.AccountID,
		WorkspaceID:     req.WorkspaceID,
		OriginalRunID:   req.OriginalRunID,
	}

	mapped := make([]map[string]any, len(req.Events))
	for n, item := range req.Events {
		mapped[n] = item.GetEvent().Map()
	}

	_, err := e.sm.New(ctx, state.Input{
		Identifier:     id,
		EventBatchData: mapped,
		Context:        req.Context,
	})
	if err == state.ErrIdentifierExists {
		// This function was already created.
		return nil, state.ErrIdentifierExists
	}

	if err != nil {
		return nil, fmt.Errorf("error creating run state: %w", err)
	}

	// Create cancellation pauses immediately, only if this is a non-batch event.
	if req.BatchID == nil {
		for _, c := range req.Function.Cancel {
			pauseID := uuid.New()
			expires := time.Now().Add(consts.CancelTimeout)
			if c.Timeout != nil {
				dur, err := str2duration.ParseDuration(*c.Timeout)
				if err != nil {
					return &id, fmt.Errorf("error parsing cancel duration: %w", err)
				}
				expires = time.Now().Add(dur)
			}

			// Ensure that we only listen to cancellation events that occur
			// after the initial event is received.
			expr := "(async.ts == null || async.ts > event.ts)"
			if c.If != nil {
				expr = expr + " && " + *c.If
			}

			// Evaluate the expression.  This lets us inspect the expression's attributes
			// so that we can store only the attrs used in the expression in the pause,
			// saving space, bandwidth, etc.
			eval, err := expressions.NewExpressionEvaluator(ctx, expr)
			if err != nil {
				return &id, err
			}
			ed := expressions.NewData(map[string]any{"event": req.Events[0].GetEvent().Map()})
			data := eval.FilteredAttributes(ctx, ed).Map()

			// The triggering event ID should be the first ID in the batch.
			triggeringID := req.Events[0].GetInternalID().String()

			pause := state.Pause{
				WorkspaceID:       req.WorkspaceID,
				Identifier:        id,
				ID:                pauseID,
				Expires:           state.Time(expires),
				Event:             &c.Event,
				Expression:        c.If,
				ExpressionData:    data,
				Cancel:            true,
				TriggeringEventID: &triggeringID,
			}
			err = e.sm.SavePause(ctx, pause)
			if err != nil {
				return &id, fmt.Errorf("error saving pause: %w", err)
			}
		}
	}

	at := time.Now()
	if req.BatchID == nil {
		evtTs := time.UnixMilli(req.Events[0].GetEvent().Timestamp)
		if evtTs.After(at) {
			// Schedule functions in the future if there's a future
			// event `ts` field.
			at = evtTs
		}
	}
	if req.At != nil {
		at = *req.At
	}

	// Prefix the workflow to the job ID so that no invocation can accidentally
	// cause idempotency issues across users/functions.
	//
	// This enures that we only ever enqueue the start job for this function once.
	queueKey := fmt.Sprintf("%s:%s", req.Function.ID, key)
	item := queue.Item{
		JobID:       &queueKey,
		GroupID:     uuid.New().String(),
		WorkspaceID: req.WorkspaceID,
		Kind:        queue.KindEdge,
		Identifier:  id,
		Attempt:     0,
		MaxAttempts: &sourceEdgeRetries,
		Payload: queue.PayloadEdge{
			Edge: inngest.SourceEdge,
		},
	}
	err = e.queue.Enqueue(ctx, item, at)
	if err == redis_state.ErrQueueItemExists {
		return nil, state.ErrIdentifierExists
	}
	if err != nil {
		return nil, fmt.Errorf("error enqueueing source edge '%v': %w", queueKey, err)
	}

	for _, e := range e.lifecycles {
		go e.OnFunctionScheduled(context.WithoutCancel(ctx), id, item)
	}

	return &id, nil
}

// Execute loads a workflow and the current run state, then executes the
// function's step via the necessary driver.
func (e *executor) Execute(ctx context.Context, id state.Identifier, item queue.Item, edge inngest.Edge, stackIndex int) (*state.DriverResponse, error) {
	if e.fl == nil {
		return nil, fmt.Errorf("no function loader specified running step")
	}

	s, err := e.sm.Load(ctx, id.RunID)
	if err != nil {
		return nil, err
	}

	md := s.Metadata()

	if md.Status == enums.RunStatusCancelled {
		return nil, state.ErrFunctionCancelled
	}

	if e.steplimit != 0 && len(s.Actions()) >= int(e.steplimit) {
		// Update this function's state to overflowed, if running.
		if md.Status == enums.RunStatusRunning {
			// XXX: Update error to failed, set error message
			if err := e.sm.SetStatus(ctx, id, enums.RunStatusOverflowed); err != nil {
				return nil, err
			}
		}
		return nil, state.ErrFunctionOverflowed
	}

	// If this is the trigger, check if we only have one child.  If so, skip to directly executing
	// that child;  we don't need to handle the trigger individually.
	//
	// This cuts down on queue churn.
	if edge.Incoming == inngest.TriggerName {
		f, err := e.fl.LoadFunction(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("error loading function for run: %w", err)
		}
		// We only support functions with a single step, as we've removed the DAG based approach.
		// This means that we always execute the first step.
		if len(f.Steps) > 1 {
			return nil, fmt.Errorf("DAG-based steps are no longer supported")
		}
		edge.Outgoing = inngest.TriggerName
		edge.Incoming = f.Steps[0].ID
		// Update the payload
		payload := item.Payload.(queue.PayloadEdge)
		payload.Edge = edge
		item.Payload = payload
		// Add retries from the step to our queue item
		retries := f.Steps[0].RetryCount()
		item.MaxAttempts = &retries

		// Only just starting:  run lifecycles on first attempt.
		if item.Attempt == 0 {
			for _, e := range e.lifecycles {
				go e.OnFunctionStarted(context.WithoutCancel(ctx), id, item)
			}
		}
	}

	// This could have been retried due to a state load error after
	// the particular step's code has ran; we need to load state after
	// each action to properly evaluate the next set of edges.
	//
	// To fix this particular consistency issue, always check to see
	// if there's output stored for this action ID.
	if resp, _ := s.ActionID(edge.Incoming); resp != nil {
		// This has already successfully been executed.
		return &state.DriverResponse{
			Scheduled: false,
			Output:    resp,
			Err:       nil,
		}, nil
	}

	resp, err := e.run(ctx, id, item, edge, s, stackIndex)
	if resp == nil && err != nil {
		return nil, err
	}

	if resp.Scheduled {
		return resp, nil
	}

	err = e.HandleResponse(ctx, id, item, edge, resp)
	return resp, err
}

func (e *executor) HandleResponse(ctx context.Context, id state.Identifier, item queue.Item, edge inngest.Edge, resp *state.DriverResponse) error {
	for _, e := range e.lifecycles {
		go e.OnStepFinished(context.WithoutCancel(ctx), id, item, edge, resp.Step, *resp)
	}

	// Check for temporary failures.  The outputs of transient errors are not
	// stored in the state store;  they're tracked via executor lifecycle methods
	// for logging.
	if resp.Err != nil && resp.Retryable() {
		// Retries are a native aspect of the queue;  returning errors always
		// retries steps if possible.

		// TODO: Remove this save error call;  it's unnecessary.
		if _, serr := e.sm.SaveResponse(ctx, id, *resp, item.Attempt); serr != nil {
			return fmt.Errorf("error saving function output: %w", serr)
		}
		return resp
	}

	// Check if this step permanently failed.  If so, the function is a failure.
	if resp.Err != nil && !resp.Retryable() {
		_ = e.sm.Finalized(ctx, id, edge.Incoming, item.Attempt, enums.RunStatusFailed)
		if serr := e.sm.SetStatus(ctx, id, enums.RunStatusFailed); serr != nil {
			return fmt.Errorf("error marking function as complete: %w", serr)
		}
		s, _ := e.sm.Load(ctx, id.RunID)
		if ferr := e.failureHandler(ctx, id, s, *resp); ferr != nil {
			// XXX: log
			_ = ferr
		}
		for _, e := range e.lifecycles {
			go e.OnFunctionFinished(context.WithoutCancel(ctx), id, item, *resp)
		}
		return resp
	}

	// This is a success, which means either a generator or a function result.
	if len(resp.Generator) > 0 {
		// Handle generator responses then return.
		if serr := e.HandleGeneratorResponse(ctx, resp.Generator, item); serr != nil {
			// If this is an error compiling async expressions, fail the function.
			if strings.Contains(serr.Error(), "error compiling expression") {
				_, _ = e.sm.SaveResponse(ctx, id, *resp, item.Attempt)
				// XXX: failureHandler is legacy.
				if serr := e.sm.SetStatus(ctx, id, enums.RunStatusFailed); serr != nil {
					return fmt.Errorf("error marking function as complete: %w", serr)
				}
				s, _ := e.sm.Load(ctx, id.RunID)
				_ = e.failureHandler(ctx, id, s, *resp)
				for _, e := range e.lifecycles {
					go e.OnFunctionFinished(context.WithoutCancel(ctx), id, item, *resp)
				}
				return nil
			}

			return fmt.Errorf("error handling generator response: %w", serr)
		}
		_ = e.sm.Finalized(ctx, id, edge.Incoming, item.Attempt)
		return nil
	}

	// This is the function result.  Save this in the state store (which will inevitably
	// be GC'd), and end.
	if _, serr := e.sm.SaveResponse(ctx, id, *resp, item.Attempt); serr != nil {
		return fmt.Errorf("error saving function output: %w", serr)
	}

	_ = e.sm.Finalized(ctx, id, edge.Incoming, item.Attempt)

	for _, e := range e.lifecycles {
		go e.OnFunctionFinished(context.WithoutCancel(ctx), id, item, *resp)
	}

	if serr := e.sm.SetStatus(ctx, id, enums.RunStatusCompleted); serr != nil {
		return fmt.Errorf("error marking function as complete: %w", serr)
	}

	return nil
}

// run executes the step with the given step ID.
//
// A nil response with an error indicates that an internal error occurred and the step
// did not run.
func (e *executor) run(ctx context.Context, id state.Identifier, item queue.Item, edge inngest.Edge, s state.State, stackIndex int) (*state.DriverResponse, error) {
	f, err := e.fl.LoadFunction(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error loading function for run: %w", err)
	}

	var step *inngest.Step
	for _, s := range f.Steps {
		if s.ID == edge.Incoming {
			step = &s
			break
		}
	}
	if step == nil {
		// Sanity check we've enqueued the right step.
		return nil, newFinalError(fmt.Errorf("unknown vertex: %s", edge.Incoming))
	}

	for _, e := range e.lifecycles {
		go e.OnStepStarted(context.WithoutCancel(ctx), id, item, edge, *step, s)
	}

	// Execute the actual step.
	response, err := e.executeDriverForStep(ctx, id, item, step, s, edge, stackIndex)

	if response != nil && response.Scheduled {
		return response, err
	}

	if response.Err != nil && err == nil {
		// This step errored, so always return an error.
		return response, fmt.Errorf("%s", *response.Err)
	}
	return response, err
}

// executeDriverForStep runs the enqueued step by invoking the driver.  It also inspects
// and normalizes responses (eg. max retry attempts).
func (e *executor) executeDriverForStep(ctx context.Context, id state.Identifier, item queue.Item, step *inngest.Step, s state.State, edge inngest.Edge, stackIndex int) (*state.DriverResponse, error) {
	d, ok := e.runtimeDrivers[step.Driver()]
	if !ok {
		return nil, fmt.Errorf("%w: '%s'", ErrNoRuntimeDriver, step.Driver())
	}
	// TODO: Remove.  This is unnecessary.
	if err := e.sm.Started(ctx, id, step.ID, item.Attempt); err != nil {
		return nil, fmt.Errorf("error saving started state: %w", err)
	}

	response, err := d.Execute(ctx, s, item, edge, *step, stackIndex, item.Attempt)
	if response == nil {
		response = &state.DriverResponse{
			Step: *step,
		}
	}
	if err != nil && response.Err == nil {
		// Set the response error if it wasn't set, or if Execute had an internal error.
		// This ensures that we only ever need to check resp.Err to handle errors.
		errstr := err.Error()
		response.Err = &errstr
	}
	// Ensure that the step is always set.  This removes the need for drivers to always
	// set this.
	if response.Step.ID == "" {
		response.Step = *step
	}
	// Max attempts is encoded at the queue level from step configuration.  If we're at max attempts,
	// ensure the response's NoRetry flag is set, as we shouldn't retry any more.  This also ensures
	// that we properly handle this response as a Failure (permanent) vs an Error (transient).
	if response.Err != nil && !queue.ShouldRetry(nil, item.Attempt, step.RetryCount()) {
		response.NoRetry = true
	}
	return response, err
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

	evtID := evt.GetInternalID()
	evtIDStr := evtID.String()

	// Schedule up to PauseHandleConcurrency pauses at once.
	sem := semaphore.NewWeighted(int64(PauseHandleConcurrency))

	for iter.Next(ctx) {
		if goerr != nil {
			break
		}

		pause := iter.Val(ctx)

		// Block until we have capacity
		if err := sem.Acquire(ctx, 1); err != nil {
			return fmt.Errorf("error blocking on semaphore: %w", err)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			// Always release one from the capacity
			defer sem.Release(1)

			if pause == nil {
				return
			}

			// NOTE: Some pauses may be nil or expired, as the iterator may take
			// time to process.  We handle that here and assume that the event
			// did not occur in time.
			if pause.Expires.Time().Before(time.Now()) {
				// Consume this pause to remove it entirely
				_ = e.sm.DeletePause(context.Background(), *pause)
				return
			}

			if pause.TriggeringEventID != nil && *pause.TriggeringEventID == evtIDStr {
				return
			}

			if pause.Cancel {
				// This is a cancellation signal.  Check if the function
				// has ended, and if so remove the pause.
				//
				// NOTE: Bookkeeping must be added to individual function runs and handled on
				// completion instead of here.  This is a hot path and should only exist whilst
				// bookkeeping is not implemented.
				if exists, err := e.sm.Exists(ctx, pause.Identifier.RunID); !exists && err == nil {
					// This function has ended.  Delete the pause and continue
					_ = e.sm.DeletePause(context.Background(), *pause)
					return
				}
			}

			// Ensure that we store the group ID for this pause, letting us properly track cancellation
			// or continuation history
			ctx = state.WithGroupID(ctx, pause.GroupID)

			// Run an expression if this exists.
			if pause.Expression != nil {
				// Precompute the expression data once, as a value (not pointer)
				data := expressions.NewData(map[string]any{
					"async": evt.GetEvent().Map(),
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
				if err != nil && err != ErrFunctionEnded && !strings.Contains(err.Error(), "no status stored in metadata") {
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
				With:    evt.GetEvent().Map(),
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

	for _, e := range e.lifecycles {
		go e.OnFunctionCancelled(context.WithoutCancel(ctx), id, r)
	}

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

	if pause.OnTimeout && r.EventID != nil {
		// Delete this pause, as an event has occured which matches
		// the timeout.  We can do this prior to leasing a pause as it's the
		// only work that needs to happen
		err := e.sm.ConsumePause(ctx, pause.ID, nil)
		if err == nil || err == state.ErrPauseNotFound {
			return nil
		}
		return err
	}

	if err = e.sm.ConsumePause(ctx, pause.ID, r.With); err != nil {
		return fmt.Errorf("error consuming pause via event: %w", err)
	}

	// Schedule an execution from the pause's entrypoint.  We do this after
	// consuming the pause to guarantee the event data is stored via the pause
	// for the next run.  If the ConsumePause call comes after enqueue, the TCP
	// conn may drop etc. and running the job may occur prior to saving state data.
	// jobID := fmt.Sprintf("%s-%s", pause.Identifier.IdempotencyKey(), pause.DataKey+"-pause")
	jobID := fmt.Sprintf("%s-%s", pause.Identifier.IdempotencyKey(), pause.DataKey)
	err = e.queue.Enqueue(
		ctx,
		queue.Item{
			JobID: &jobID,
			// Add a new group ID for the child;  this will be a new step.
			GroupID:     uuid.New().String(),
			WorkspaceID: pause.WorkspaceID,
			Kind:        queue.KindEdge,
			Identifier:  pause.Identifier,
			Payload: queue.PayloadEdge{
				Edge: pause.Edge(),
			},
		},
		time.Now(),
	)
	if err != nil && err != redis_state.ErrQueueItemExists {
		return fmt.Errorf("error enqueueing after pause: %w", err)
	}

	for _, e := range e.lifecycles {
		go e.OnWaitForEventResumed(context.WithoutCancel(ctx), pause.Identifier, r)
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

	resp := state.DriverResponse{
		Step: inngest.Step{
			ID:   gen.ID,
			Name: gen.Name,
		},
	}
	if gen.Data != nil {
		if err := json.Unmarshal(gen.Data, &resp.Output); err != nil {
			resp.Output = gen.Data
		}
	}

	// Save the response to the state store.
	if _, err := e.sm.SaveResponse(ctx, item.Identifier, resp, item.Attempt); err != nil {
		return err
	}

	// Update the group ID in context;  we've already saved this step's success and we're now
	// running the step again, needing a new history group
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

	// Re-enqueue the exact same edge to run now.
	if err := e.sm.Scheduled(ctx, item.Identifier, nextEdge.Incoming, 0, nil); err != nil {
		return err
	}

	jobID := fmt.Sprintf("%s-%s", item.Identifier.IdempotencyKey(), gen.ID)
	nextItem := queue.Item{
		JobID:       &jobID,
		WorkspaceID: item.WorkspaceID,
		GroupID:     groupID,
		Kind:        queue.KindEdge,
		Identifier:  item.Identifier,
		Attempt:     0,
		MaxAttempts: item.MaxAttempts,
		Payload:     queue.PayloadEdge{Edge: nextEdge},
	}
	err := e.queue.Enqueue(ctx, nextItem, time.Now())
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, l := range e.lifecycles {
		go l.OnStepScheduled(ctx, item.Identifier, nextItem)
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

	// Update the group ID in context;  we're scheduling a step, and we want
	// to start a new history group for this item.
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

	// Re-enqueue the exact same edge to run now.
	if err := e.sm.Scheduled(ctx, item.Identifier, edge.Edge.Incoming, 0, nil); err != nil {
		return err
	}

	jobID := fmt.Sprintf("%s-%s", item.Identifier.IdempotencyKey(), gen.ID+"-plan")
	nextItem := queue.Item{
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
	}
	err := e.queue.Enqueue(ctx, nextItem, time.Now())
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, l := range e.lifecycles {
		go l.OnStepScheduled(ctx, item.Identifier, nextItem)
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

	// Create another group for the next item which will run.  We're enqueueing
	// the function to run again after sleep, so need a new group.
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

	// XXX: Remove this after we create queues for function runs.
	if err := e.sm.Scheduled(ctx, item.Identifier, nextEdge.Incoming, 0, &at); err != nil {
		return err
	}

	until := time.Now().Add(dur)

	jobID := fmt.Sprintf("%s-%s", item.Identifier.IdempotencyKey(), gen.ID)
	err = e.queue.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
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
	}, until)
	if err == redis_state.ErrQueueItemExists {
		// Safely ignore this error.
		return nil
	}

	for _, e := range e.lifecycles {
		go e.OnSleep(context.WithoutCancel(ctx), item.Identifier, item, gen, until)
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
	}, expires)
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, e := range e.lifecycles {
		go e.OnWaitForEvent(context.WithoutCancel(ctx), item.Identifier, item, gen)
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
