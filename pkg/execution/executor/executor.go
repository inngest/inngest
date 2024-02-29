package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/structs"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/cancellation"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/inngest/inngest/pkg/logger"
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
	ErrFunctionDebounced = fmt.Errorf("function debounced")

	ErrFunctionEnded = fmt.Errorf("function already ended")

	// ErrHandledStepError is returned when an OpcodeStepError is caught and the
	// step should be safely retried.
	ErrHandledStepError = fmt.Errorf("handled step error")

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

func WithCancellationChecker(c cancellation.Checker) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).cancellationChecker = c
		return nil
	}
}

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

// WithExpressionAggregator sets the expression aggregator singleton to use
// for matching events using our aggregate evaluator.
func WithExpressionAggregator(agg expressions.Aggregator) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).exprAggregator = agg
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

func WithFinishHandler(f execution.FinishHandler) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).finishHandler = f
		return nil
	}
}

func WithInvokeNotFoundHandler(f execution.InvokeNotFoundHandler) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).invokeNotFoundHandler = f
		return nil
	}
}

func WithSendingEventHandler(f execution.HandleSendingEvent) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).handleSendingEvent = f
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
		if limit > consts.AbsoluteMaxStepLimit {
			return fmt.Errorf("%d is greater than the absolute step limit of %d", limit, consts.AbsoluteMaxStepLimit)
		}
		e.(*executor).steplimit = limit
		return nil
	}
}

func WithDebouncer(d debounce.Debouncer) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).debouncer = d
		return nil
	}
}

func WithBatcher(b batch.BatchManager) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).batcher = b
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

	// exprAggregator is an expression aggregator used to parse and aggregate expressions
	// using trees.
	exprAggregator expressions.Aggregator

	sm                    state.Manager
	queue                 queue.Queue
	debouncer             debounce.Debouncer
	batcher               batch.BatchManager
	fl                    state.FunctionLoader
	evalFactory           func(ctx context.Context, expr string) (expressions.Evaluator, error)
	runtimeDrivers        map[string]driver.Driver
	finishHandler         execution.FinishHandler
	invokeNotFoundHandler execution.InvokeNotFoundHandler
	handleSendingEvent    execution.HandleSendingEvent
	cancellationChecker   cancellation.Checker

	lifecycles []execution.LifecycleListener

	steplimit uint
}

func (e *executor) SetFinishHandler(f execution.FinishHandler) {
	e.finishHandler = f
}

func (e *executor) SetInvokeNotFoundHandler(f execution.InvokeNotFoundHandler) {
	e.invokeNotFoundHandler = f
}

func (e *executor) InvokeNotFoundHandler(ctx context.Context, opts execution.InvokeNotFoundHandlerOpts) error {
	if e.invokeNotFoundHandler == nil {
		return nil
	}

	evt := CreateInvokeNotFoundEvent(ctx, opts)

	return e.invokeNotFoundHandler(ctx, opts, []event.Event{evt})
}

func (e *executor) AddLifecycleListener(l execution.LifecycleListener) {
	e.lifecycles = append(e.lifecycles, l)
}

// Execute loads a workflow and the current run state, then executes the
// function's step via the necessary driver.
//
// If this function has a debounce config, this will return ErrFunctionDebounced instead
// of an identifier as the function is not scheduled immediately.
func (e *executor) Schedule(ctx context.Context, req execution.ScheduleRequest) (*state.Identifier, error) {
	if req.Function.Debounce != nil && !req.PreventDebounce {
		err := e.debouncer.Debounce(ctx, debounce.DebounceItem{
			AccountID:       req.AccountID,
			WorkspaceID:     req.WorkspaceID,
			AppID:           req.AppID,
			FunctionID:      req.Function.ID,
			FunctionVersion: req.Function.FunctionVersion,
			EventID:         req.Events[0].GetInternalID(),
			Event:           req.Events[0].GetEvent(),
		}, req.Function)
		if err != nil {
			return nil, err
		}
		return nil, ErrFunctionDebounced
	}

	// Run IDs are created embedding the timestamp now, when the function is being scheduled.
	// When running a cancellation, functions are cancelled at scheduling time based off of
	// this run ID.
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

	eventIDs := []ulid.ULID{}
	for _, e := range req.Events {
		eventIDs = append(eventIDs, e.GetInternalID())
	}

	id := state.Identifier{
		WorkflowID:      req.Function.ID,
		WorkflowVersion: req.Function.FunctionVersion,
		RunID:           runID,
		BatchID:         req.BatchID,
		EventID:         req.Events[0].GetInternalID(),
		EventIDs:        eventIDs,
		Key:             key,
		AccountID:       req.AccountID,
		WorkspaceID:     req.WorkspaceID,
		AppID:           req.AppID,
		OriginalRunID:   req.OriginalRunID,
		ReplayID:        req.ReplayID,
	}

	mapped := make([]map[string]any, len(req.Events))
	for n, item := range req.Events {
		mapped[n] = item.GetEvent().Map()
	}

	if req.Function.Concurrency != nil {
		// Ensure we evaluate concurrency keys when scheduling the function.
		for _, limit := range req.Function.Concurrency.Limits {
			if !limit.IsCustomLimit() {
				continue
			}

			// Ensure we bind the limit to the correct scope.
			scopeID := req.Function.ID
			switch limit.Scope {
			case enums.ConcurrencyScopeAccount:
				scopeID = req.AccountID
			case enums.ConcurrencyScopeEnv:
				scopeID = req.WorkspaceID
			}

			// Store the concurrency limit in the function.  By copying in the raw expression hash,
			// we can update the concurrency limits for in-progress runs as new function versions
			// are stored.
			//
			// The raw keys are stored in the function state so that we don't need to re-evaluate
			// keys and input each time, as they're constant through the function run.
			id.CustomConcurrencyKeys = append(id.CustomConcurrencyKeys, state.CustomConcurrency{
				Key:   limit.Evaluate(ctx, scopeID, mapped[0]),
				Hash:  limit.Hash,
				Limit: limit.Limit,
			})
		}
	}

	// Evaluate the run priority based off of the input event data.
	factor, err := req.Function.RunPriorityFactor(ctx, mapped[0])
	if err != nil && e.log != nil {
		e.log.Warn().Err(err).Msg("run priority errored")
	}
	if factor != 0 {
		id.PriorityFactor = &factor
	}

	// Create a new function.
	s, err := e.sm.New(ctx, state.Input{
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

			// Evaluate the expression.  This lets us inspect the expression's attributes
			// so that we can store only the attrs used in the expression in the pause,
			// saving space, bandwidth, etc.
			expr := generateCancelExpression(eventIDs[0], c.If)
			eval, err := expressions.NewExpressionEvaluator(ctx, expr)
			if err != nil {
				return &id, err
			}
			ed := expressions.NewData(map[string]any{"event": req.Events[0].GetEvent().Map()})
			data := eval.FilteredAttributes(ctx, ed).Map()

			// The triggering event ID should be the first ID in the batch.
			triggeringID := req.Events[0].GetInternalID().String()

			// Remove `event` data from the expression and replace with actual event
			// data as values, now that we have the event.
			//
			// This improves performance in matching, as we can then use the values within
			// aggregate trees.
			interpolated, err := expressions.Interpolate(ctx, expr, map[string]any{
				"event": mapped[0],
			})
			if err != nil {
				logger.StdlibLogger(ctx).Warn(
					"error interpolating cancellation expression",
					"error", err,
					"expression", expr,
				)
			}

			pause := state.Pause{
				WorkspaceID:       req.WorkspaceID,
				Identifier:        id,
				ID:                pauseID,
				Expires:           state.Time(expires),
				Event:             &c.Event,
				Expression:        &interpolated,
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
		Kind:        queue.KindStart,
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
		go e.OnFunctionScheduled(context.WithoutCancel(ctx), id, item, s)
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

	// Store the metadata in context for future use.  This can be used to reduce
	// reads in the future.
	ctx = WithContextMetadata(ctx, md)

	if md.Status == enums.RunStatusCancelled {
		return nil, state.ErrFunctionCancelled
	}

	if e.steplimit != 0 && len(s.Actions()) >= int(e.steplimit) {
		// Update this function's state to overflowed, if running.
		if md.Status == enums.RunStatusRunning {
			// XXX: Update error to failed, set error message
			if err := e.sm.SetStatus(ctx, id, enums.RunStatusFailed); err != nil {
				return nil, err
			}

			// Create a new driver response to map as the function finished error.
			resp := state.DriverResponse{}
			resp.SetError(state.ErrFunctionOverflowed)
			resp.SetFinal()

			if err := e.runFinishHandler(ctx, id, s, resp); err != nil {
				logger.From(ctx).Error().Err(err).Msg("error running finish handler")
			}

			for _, e := range e.lifecycles {
				go e.OnFunctionFinished(context.WithoutCancel(ctx), id, item, resp, s)
			}
		}
		return nil, state.ErrFunctionOverflowed
	}

	// Check if the function is cancelled.
	if e.cancellationChecker != nil {
		cancel, err := e.cancellationChecker.IsCancelled(
			ctx,
			md.Identifier.WorkspaceID,
			md.Identifier.WorkflowID,
			md.Identifier.RunID,
			s.Event(),
		)
		if err != nil {
			logger.StdlibLogger(ctx).Error(
				"error checking cancellation",
				"error", err.Error(),
				"run_id", md.Identifier.RunID,
				"function_id", md.Identifier.WorkflowID,
				"workspace_id", md.Identifier.WorkspaceID,
			)
		}
		if cancel != nil {
			return nil, e.Cancel(ctx, md.Identifier.RunID, execution.CancelRequest{
				CancellationID: &cancel.ID,
			})
		}
	}

	// If this is the trigger, check if we only have one child.  If so, skip to directly executing
	// that child;  we don't need to handle the trigger individually.
	//
	// This cuts down on queue churn.
	//
	// NOTE: This is a holdover from treating functions as a *series* of DAG calls.  In that case,
	// we automatically enqueue all children of the dag from the root node.
	// This can be cleaned up.
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
		// Add retries from the step to our queue item.  Increase as retries is
		// always one less than attempts.
		retries := f.Steps[0].RetryCount() + 1
		item.MaxAttempts = &retries

		// Only just starting:  run lifecycles on first attempt.
		if item.Attempt == 0 {
			for _, e := range e.lifecycles {
				go e.OnFunctionStarted(context.WithoutCancel(ctx), id, item, s)
			}
		}
	}

	// Ensure that if users requeue steps we never re-execute.
	incoming := edge.Incoming
	if edge.IncomingGeneratorStep != "" {
		incoming = edge.IncomingGeneratorStep
	}
	if resp, _ := s.ActionID(incoming); resp != nil {
		// This has already successfully been executed.
		return &state.DriverResponse{
			Output: resp,
			Err:    nil,
		}, nil
	}

	resp, err := e.run(ctx, id, item, edge, s, stackIndex)
	if resp == nil && err != nil {
		return nil, err
	}

	err = e.HandleResponse(ctx, id, item, edge, resp)
	return resp, err
}

func init() {
	spew.Config.DisableMethods = true
}

func (e *executor) HandleResponse(ctx context.Context, id state.Identifier, item queue.Item, edge inngest.Edge, resp *state.DriverResponse) error {
	for _, e := range e.lifecycles {
		// OnStepFinished handles step success and step errors/failures.  It is
		// currently the responsibility of the lifecycle manager to handle the differing
		// step statuses when a step finishes.
		//
		// TODO (tonyhb): This should probably change, as each lifecycle listener has to
		// do the same parsing & conditional checks.
		go e.OnStepFinished(context.WithoutCancel(ctx), id, item, edge, resp.Step, *resp)
	}

	// Check for temporary failures.  The outputs of transient errors are not
	// stored in the state store;  they're tracked via executor lifecycle methods
	// for logging.
	//
	// NOTE: If the SDK was running a step (NOT function code) and quit gracefully,
	// resp.UserError will always be set, even if the step itself throws a non-retriable
	// error.
	//
	// This is purely for network errors or top-level function code errors.
	if resp.Err != nil {
		if resp.Retryable() {
			// Retries are a native aspect of the queue;  returning errors always
			// retries steps if possible.
			for _, e := range e.lifecycles {
				// Run the lifecycle method for this retry, which is baked into the queue.
				item.Attempt += 1
				go e.OnStepScheduled(context.WithoutCancel(ctx), id, item, &resp.Step.Name)
			}

			return resp
		}

		// If resp.Err != nil, we don't know whether to invoke the fn again
		// with per-step errors, as we don't know if the intent behind this queue item
		// is a step.
		//
		// In this case, for non-retryable errors, we ignore and fail the function;
		// only OpcodeStepError causes try/catch to be handled and us to continue
		// on error.
		//
		// TODO: Improve this.

		// Check if this step permanently failed.  If so, the function is a failure.
		if !resp.Retryable() {
			if serr := e.sm.SetStatus(ctx, id, enums.RunStatusFailed); serr != nil {
				return fmt.Errorf("error marking function as complete: %w", serr)
			}
			s, err := e.sm.Load(ctx, id.RunID)
			if err != nil {
				return fmt.Errorf("unable to load run: %w", err)
			}

			if err := e.runFinishHandler(ctx, id, s, *resp); err != nil {
				logger.From(ctx).Error().Err(err).Msg("error running finish handler")
			}

			for _, e := range e.lifecycles {
				go e.OnFunctionFinished(context.WithoutCancel(ctx), id, item, *resp, s)
			}
			return resp
		}
	}

	// This is a success, which means either a generator or a function result.
	if len(resp.Generator) > 0 {
		// Handle generator responses then return.
		if serr := e.HandleGeneratorResponse(ctx, resp, item); serr != nil {
			// If this is an error compiling async expressions, fail the function.
			if strings.Contains(serr.Error(), "error compiling expression") {
				resp.SetError(serr)
				resp.SetFinal()
				_ = e.sm.SaveResponse(ctx, id, resp.Step.ID, resp.Error())
				// XXX: failureHandler is legacy.
				if serr := e.sm.SetStatus(ctx, id, enums.RunStatusFailed); serr != nil {
					return fmt.Errorf("error marking function as complete: %w", serr)
				}
				s, err := e.sm.Load(ctx, id.RunID)
				if err != nil {
					return fmt.Errorf("unable to load run: %w", err)
				}
				if err := e.runFinishHandler(ctx, id, s, *resp); err != nil {
					logger.From(ctx).Error().Err(err).Msg("error running finish handler")
				}
				for _, e := range e.lifecycles {
					go e.OnFunctionFinished(context.WithoutCancel(ctx), id, item, *resp, s)
				}
				return nil
			}
			return fmt.Errorf("error handling generator response: %w", serr)
		}
		return nil
	}

	// This is the function result.

	// TODO: Use state loaded before function call instead of loading once again
	// to reduce load.  That way, we never need to call SaveResponse and Load().
	//
	// Save this in the state store (which will inevitably be GC'd), and end
	output, err := json.Marshal(resp.Output)
	if err != nil {
		return err
	}

	if serr := e.sm.SaveResponse(ctx, id, resp.Step.ID, string(output)); serr != nil {
		// Final function responses can be duplicated if multiple parallel
		// executions reach the end at the same time. Steps themselves are
		// de-duplicated in the queue.
		if serr == state.ErrDuplicateResponse {
			return resp
		}
		return fmt.Errorf("error saving function output: %w", serr)
	}
	s, err := e.sm.Load(ctx, id.RunID)
	if err != nil {
		return fmt.Errorf("unable to load run: %w", err)
	}
	// end todo

	if err := e.runFinishHandler(ctx, id, s, *resp); err != nil {
		logger.From(ctx).Error().Err(err).Msg("error running finish handler")
	}

	for _, e := range e.lifecycles {
		go e.OnFunctionFinished(context.WithoutCancel(ctx), id, item, *resp, s)
	}

	if serr := e.sm.SetStatus(ctx, id, enums.RunStatusCompleted); serr != nil {
		return fmt.Errorf("error marking function as complete: %w", serr)
	}

	return nil
}

type functionFinishedData struct {
	FunctionID          string           `json:"function_id"`
	RunID               ulid.ULID        `json:"run_id"`
	Event               map[string]any   `json:"event"`
	Events              []map[string]any `json:"events"`
	Error               any              `json:"error,omitempty"`
	Result              any              `json:"result,omitempty"`
	InvokeCorrelationID *string          `json:"correlation_id,omitempty"`
}

func (f *functionFinishedData) setResponse(r state.DriverResponse) {
	if r.Err != nil {
		f.Error = r.StandardError()
	}
	if r.UserError != nil {
		f.Error = r.UserError
	}
	if r.Output != nil {
		f.Result = r.Output
	}
}

func (f functionFinishedData) Map() map[string]any {
	s := structs.New(f)
	s.TagName = "json"
	return s.Map()
}

func (e *executor) runFinishHandler(ctx context.Context, id state.Identifier, s state.State, resp state.DriverResponse) error {
	if e.finishHandler == nil {
		return nil
	}

	// Prepare events that we must send
	now := time.Now()
	base := &functionFinishedData{
		FunctionID: s.Function().Slug,
		RunID:      id.RunID,
		Events:     s.Events(),
	}
	base.setResponse(resp)

	// We'll send many events - some for each items in the batch.  This ensures that invoke works
	// for batched functions.
	var events []event.Event
	for n, runEvt := range s.Events() {
		if name, ok := runEvt["name"].(string); ok && (name == event.FnFailedName || name == event.FnFinishedName) {
			// Don't recursively trigger internal finish handlers.
			continue
		}

		invokeID := correlationID(runEvt)
		if invokeID == nil && n > 0 {
			// We only send function finish events for either the first event in a batch or for
			// all events with a correlation ID.
			continue
		}

		// Copy the base data to set the event.
		copied := *base
		copied.Event = runEvt
		copied.InvokeCorrelationID = invokeID
		data := copied.Map()

		// Add an `inngest/function.finished` event.
		events = append(events, event.Event{
			ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
			Name:      event.FnFinishedName,
			Timestamp: now.UnixMilli(),
			Data:      data,
		})

		// Legacy - send inngest/function.failed, except for when the function has been cancelled.
		if resp.Err != nil && !strings.Contains(*resp.Err, state.ErrFunctionCancelled.Error()) {
			events = append(events, event.Event{
				ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
				Name:      event.FnFailedName,
				Timestamp: now.UnixMilli(),
				Data:      data,
			})
		}
	}

	return e.finishHandler(ctx, s, events)
}

func correlationID(event map[string]any) *string {
	dataMap, ok := event["data"].(map[string]any)
	if !ok {
		return nil
	}
	container, ok := dataMap[consts.InngestEventDataPrefix].(map[string]any)
	if !ok {
		return nil
	}
	if correlationID, ok := container[consts.InvokeCorrelationId].(string); ok {
		return &correlationID
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

	// If there's one opcode and it's of type StepError, ensure we set resp.Err to
	// a string containing the response error.
	//
	// TODO: Refactor response.Err
	if len(response.Generator) == 1 && response.Generator[0].Op == enums.OpcodeStepError {
		if !queue.ShouldRetry(nil, item.Attempt, step.RetryCount()+1) {
			response.NoRetry = true
		}
	}

	// Max attempts is encoded at the queue level from step configuration.  If we're at max attempts,
	// ensure the response's NoRetry flag is set, as we shouldn't retry any more.  This also ensures
	// that we properly handle this response as a Failure (permanent) vs an Error (transient).
	if response.Err != nil && !queue.ShouldRetry(nil, item.Attempt, step.RetryCount()+1) {
		response.NoRetry = true
	}

	return response, err
}

// HandlePauses handles pauses loaded from an incoming event.
func (e *executor) HandlePauses(ctx context.Context, iter state.PauseIterator, evt event.TrackedEvent) (execution.HandlePauseResult, error) {
	// Use the aggregator for all funciton finished events, if there are more than
	// 50 waiting.  It only takes a few milliseconds to iterate and handle less
	// than 50;  anything more runs the risk of running slow.
	if evt.GetEvent().Name == event.FnFinishedName && iter.Count() > 50 {
		aggRes, err := e.handleAggregatePauses(ctx, evt)
		if err != nil {
			log.From(ctx).Error().Err(err).Msg("error handling aggregate pauses")
		}
		return aggRes, err
	}

	res, err := e.handlePausesAllNaively(ctx, iter, evt)
	if err != nil {
		log.From(ctx).Error().Err(err).Msg("error handling aggregate pauses")
	}
	return res, nil
}

//nolint:all
func (e *executor) handlePausesAllNaively(ctx context.Context, iter state.PauseIterator, evt event.TrackedEvent) (execution.HandlePauseResult, error) {
	res := execution.HandlePauseResult{0, 0}

	if e.queue == nil || e.sm == nil {
		return res, fmt.Errorf("No queue or state manager specified")
	}

	log := e.log
	if log == nil {
		log = logger.From(ctx)
	}
	base := log.With().Str("event_id", evt.GetInternalID().String()).Logger()

	var (
		goerr error
		wg    sync.WaitGroup
	)

	evtID := evt.GetInternalID()
	evtIDStr := evtID.String()

	// Schedule up to PauseHandleConcurrency pauses at once.
	sem := semaphore.NewWeighted(int64(PauseHandleConcurrency))

	for iter.Next(ctx) {
		pause := iter.Val(ctx)

		// Block until we have capacity
		if err := sem.Acquire(ctx, 1); err != nil {
			return res, fmt.Errorf("error blocking on semaphore: %w", err)
		}

		wg.Add(1)
		go func() {
			atomic.AddInt32(&res[0], 1)

			defer wg.Done()
			// Always release one from the capacity
			defer sem.Release(1)

			if pause == nil {
				return
			}

			l := base.With().
				Str("pause_id", pause.ID.String()).
				Str("run_id", pause.Identifier.RunID.String()).
				Str("workflow_id", pause.Identifier.WorkflowID.String()).
				Str("expires", pause.Expires.String()).
				Logger()

			// NOTE: Some pauses may be nil or expired, as the iterator may take
			// time to process.  We handle that here and assume that the event
			// did not occur in time.
			if pause.Expires.Time().Before(time.Now()) {
				// Consume this pause to remove it entirely
				l.Debug().Msg("deleting expired pause")
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
					l.Error().Err(err).Msg("error compiling pause expression")
					return
				}

				val, _, err := expr.Evaluate(ctx, data)
				if err != nil {
					l.Warn().Err(err).Msg("error evaluating pause expression")
					return
				}
				result, _ := val.(bool)
				if !result {
					l.Trace().Msg("pause did not match expression")
					return
				}
			}

			// Ensure that we store the group ID for this pause, letting us properly track cancellation
			// or continuation history
			ctx = state.WithGroupID(ctx, pause.GroupID)

			// Cancelling a function can happen before a lease, as it's an atomic operation that will always happen.
			if pause.Cancel {
				err := e.Cancel(ctx, pause.Identifier.RunID, execution.CancelRequest{
					EventID:    &evtID,
					Expression: pause.Expression,
				})
				if errors.Is(err, state.ErrFunctionCancelled) ||
					errors.Is(err, state.ErrFunctionComplete) ||
					errors.Is(err, state.ErrFunctionFailed) ||
					errors.Is(err, ErrFunctionEnded) {
					// Safe to ignore.
					return
				}
				if err != nil && !strings.Contains(err.Error(), "no status stored in metadata") {
					goerr = errors.Join(goerr, fmt.Errorf("error cancelling function: %w", err))
					return
				}
				// Ensure we consume this pause, as this isn't handled by the higher-level cancel function.
				err = e.sm.ConsumePause(ctx, pause.ID, nil)
				if err == nil || err == state.ErrPauseLeased || err == state.ErrPauseNotFound {
					// Done. Add to the counter.
					atomic.AddInt32(&res[1], 1)
					return
				}
				goerr = errors.Join(goerr, fmt.Errorf("error consuming pause after cancel: %w", err))
				return
			}

			resumeData := pause.GetResumeData(evt.GetEvent())

			if e.log != nil {
				e.log.
					Debug().
					Interface("with", resumeData.With).
					Str("pause.DataKey", pause.DataKey).
					Msg("resuming pause")
			}

			err := e.Resume(ctx, *pause, execution.ResumeRequest{
				With:     resumeData.With,
				EventID:  &evtID,
				RunID:    resumeData.RunID,
				StepName: resumeData.StepName,
			})
			if err != nil {
				goerr = errors.Join(goerr, fmt.Errorf("error consuming pause after cancel: %w", err))
				return
			}
			// Add to the counter.
			atomic.AddInt32(&res[1], 1)
		}()

	}

	wg.Wait()

	if iter.Error() != context.Canceled {
		goerr = errors.Join(goerr, fmt.Errorf("pause iteration error: %w", iter.Error()))
	}

	return res, goerr
}

func (e *executor) handleAggregatePauses(ctx context.Context, evt event.TrackedEvent) (execution.HandlePauseResult, error) {
	res := execution.HandlePauseResult{0, 0}

	if e.exprAggregator == nil {
		return execution.HandlePauseResult{}, fmt.Errorf("no expression evaluator found")
	}

	base := logger.From(ctx).With().Str("event_id", evt.GetInternalID().String()).Logger()
	evtID := evt.GetInternalID()
	evtIDStr := evtID.String()

	evals, count, err := e.exprAggregator.EvaluateAsyncEvent(ctx, evt)
	if err != nil {
		return execution.HandlePauseResult{count, 0}, err
	}

	var (
		goerr error
		wg    sync.WaitGroup
	)

	base.Debug().
		Int("pause_len", len(evals)).
		Int32("matched_len", count).
		Msg("matched pauses via aggregator")

	for _, i := range evals {
		found, ok := i.(*state.Pause)
		if !ok || found == nil {
			continue
		}

		// Copy pause into function
		pause := *found
		wg.Add(1)
		go func() {
			atomic.AddInt32(&res[0], 1)

			defer wg.Done()

			l := base.With().
				Str("pause_id", pause.ID.String()).
				Str("run_id", pause.Identifier.RunID.String()).
				Str("workflow_id", pause.Identifier.WorkflowID.String()).
				Str("expires", pause.Expires.String()).
				Logger()

			// NOTE: Some pauses may be nil or expired, as the iterator may take
			// time to process.  We handle that here and assume that the event
			// did not occur in time.
			if pause.Expires.Time().Before(time.Now()) {
				// Consume this pause to remove it entirely
				l.Debug().Msg("deleting expired pause")
				_ = e.sm.DeletePause(context.Background(), pause)
				_ = e.exprAggregator.RemovePause(ctx, pause)
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
					_ = e.sm.DeletePause(context.Background(), pause)
					_ = e.exprAggregator.RemovePause(ctx, pause)
					return
				}
			}

			// Ensure that we store the group ID for this pause, letting us properly track cancellation
			// or continuation history
			ctx = state.WithGroupID(ctx, pause.GroupID)

			// Cancelling a function can happen before a lease, as it's an atomic operation that will always happen.
			if pause.Cancel {
				err := e.Cancel(ctx, pause.Identifier.RunID, execution.CancelRequest{
					EventID:    &evtID,
					Expression: pause.Expression,
				})
				if errors.Is(err, state.ErrFunctionCancelled) ||
					errors.Is(err, state.ErrFunctionComplete) ||
					errors.Is(err, state.ErrFunctionFailed) ||
					errors.Is(err, ErrFunctionEnded) {
					// Safe to ignore.
					_ = e.exprAggregator.RemovePause(ctx, pause)
					return
				}
				if err != nil && strings.Contains(err.Error(), "no status stored in metadata") {
					// Safe to ignore.
					_ = e.exprAggregator.RemovePause(ctx, pause)
					return
				}

				if err != nil {
					goerr = errors.Join(goerr, fmt.Errorf("error cancelling function: %w", err))
					return
				}
				// Ensure we consume this pause, as this isn't handled by the higher-level cancel function.
				err = e.sm.ConsumePause(ctx, pause.ID, nil)
				if err == nil || err == state.ErrPauseLeased || err == state.ErrPauseNotFound {
					// Done. Add to the counter.
					atomic.AddInt32(&res[1], 1)
					_ = e.exprAggregator.RemovePause(ctx, pause)
					return
				}
				goerr = errors.Join(goerr, fmt.Errorf("error consuming pause after cancel: %w", err))
				return
			}

			resumeData := pause.GetResumeData(evt.GetEvent())

			err := e.Resume(ctx, pause, execution.ResumeRequest{
				With:     resumeData.With,
				EventID:  &evtID,
				RunID:    resumeData.RunID,
				StepName: resumeData.StepName,
			})
			if err != nil {
				goerr = errors.Join(goerr, fmt.Errorf("error consuming pause after cancel: %w", err))
				return
			}
			// Add to the counter.
			atomic.AddInt32(&res[1], 1)
			if err := e.exprAggregator.RemovePause(ctx, pause); err != nil {
				l.Error().Err(err).Msg("error removing pause from aggregator")
			}
		}()
	}
	wg.Wait()

	return res, goerr
}

// Cancel cancels an in-progress function.
func (e *executor) Cancel(ctx context.Context, runID ulid.ULID, r execution.CancelRequest) error {
	s, err := e.sm.Load(ctx, runID)
	if err != nil {
		return fmt.Errorf("unable to load run: %w", err)
	}
	md := s.Metadata()

	switch md.Status {
	case enums.RunStatusFailed, enums.RunStatusCompleted, enums.RunStatusOverflowed:
		return ErrFunctionEnded
	case enums.RunStatusCancelled:
		return nil
	}

	if err := e.sm.Cancel(ctx, md.Identifier); err != nil {
		return fmt.Errorf("error cancelling function: %w", err)
	}

	// TODO: Load all pauses for the function and remove, once we index pauses.

	fnCancelledErr := state.ErrFunctionCancelled.Error()
	if err := e.runFinishHandler(ctx, s.Identifier(), s, state.DriverResponse{
		Err: &fnCancelledErr,
	}); err != nil {
		logger.From(ctx).Error().Err(err).Msg("error running finish handler")
	}

	for _, e := range e.lifecycles {
		go e.OnFunctionCancelled(context.WithoutCancel(ctx), md.Identifier, r, s)
	}

	return nil
}

// Resume resumes an in-progress function from the given pause.
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

	if e.log != nil {
		e.log.Debug().
			Str("pause_id", pause.ID.String()).
			Str("run_id", pause.Identifier.RunID.String()).
			Str("workflow_id", pause.Identifier.WorkflowID.String()).
			Bool("timeout", pause.OnTimeout).
			Bool("cancel", pause.Cancel).
			Msg("resuming from pause")
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

	if pause.Opcode != nil && *pause.Opcode == enums.OpcodeInvokeFunction.String() {
		for _, e := range e.lifecycles {
			go e.OnInvokeFunctionResumed(context.WithoutCancel(ctx), pause.Identifier, r, pause.GroupID)
		}
	} else {
		for _, e := range e.lifecycles {
			go e.OnWaitForEventResumed(context.WithoutCancel(ctx), pause.Identifier, r, pause.GroupID)
		}
	}

	return nil
}

func (e *executor) HandleGeneratorResponse(ctx context.Context, resp *state.DriverResponse, item queue.Item) error {
	md, err := GetFunctionRunMetadata(ctx, e.sm, item.Identifier.RunID)
	if err != nil || md == nil {
		return fmt.Errorf("error loading function metadata: %w", err)
	}

	{
		// The following code helps with parallelism and the V2 -> V3 executor changes
		var update *state.MetadataUpdate
		// NOTE: We only need to set hash versions when handling generator responses, else the
		// fn is ending and it doesn't matter.
		if md.RequestVersion == -1 {
			update = &state.MetadataUpdate{
				Context:                   md.Context,
				Debugger:                  md.Debugger,
				DisableImmediateExecution: md.DisableImmediateExecution,
				RequestVersion:            resp.RequestVersion,
			}
		}
		if len(resp.Generator) > 1 {
			if !md.DisableImmediateExecution {
				// With parallelism, we currently instruct the SDK to disable immediate execution,
				// enforcing that every step becomes pre-planned.
				if update == nil {
					update = &state.MetadataUpdate{
						Context:                   md.Context,
						Debugger:                  md.Debugger,
						DisableImmediateExecution: true,
						RequestVersion:            resp.RequestVersion,
					}
				}
				update.DisableImmediateExecution = true
			}
		}
		if update != nil {
			if err := e.sm.UpdateMetadata(ctx, item.Identifier.RunID, *update); err != nil {
				return fmt.Errorf("error updating function metadata: %w", err)
			}
		}
	}

	groups := opGroups(resp.Generator).All()
	for _, group := range groups {
		if err := e.handleGeneratorGroup(ctx, group, resp, item); err != nil {
			return err
		}
	}

	return nil
}

func (e *executor) handleGeneratorGroup(ctx context.Context, group OpcodeGroup, resp *state.DriverResponse, item queue.Item) error {
	eg := errgroup.Group{}
	for _, op := range group.Opcodes {
		if op == nil {
			// This is clearly an error.
			if e.log != nil {
				e.log.Error().Err(fmt.Errorf("nil generator returned")).Msg("error handling generator")
			}
			continue
		}
		copied := *op

		newItem := item
		if group.ShouldStartHistoryGroup {
			// Give each opcode its own group ID, since we want to track each
			// parellel step individually.
			newItem.GroupID = uuid.New().String()
		}

		eg.Go(func() error { return e.HandleGenerator(ctx, copied, newItem) })
	}
	if err := eg.Wait(); err != nil {
		if resp.NoRetry {
			return queue.NeverRetryError(err)
		}
		if resp.RetryAt != nil {
			return queue.RetryAtError(err, resp.RetryAt)
		}
		return err
	}

	return nil
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
	case enums.OpcodeStep, enums.OpcodeStepRun:
		return e.handleGeneratorStep(ctx, gen, item, edge)
	case enums.OpcodeStepError:
		return e.handleStepError(ctx, gen, item, edge)
	case enums.OpcodeStepPlanned:
		return e.handleGeneratorStepPlanned(ctx, gen, item, edge)
	case enums.OpcodeSleep:
		return e.handleGeneratorSleep(ctx, gen, item, edge)
	case enums.OpcodeWaitForEvent:
		return e.handleGeneratorWaitForEvent(ctx, gen, item, edge)
	case enums.OpcodeInvokeFunction:
		return e.handleGeneratorInvokeFunction(ctx, gen, item, edge)
	}

	return fmt.Errorf("unknown opcode: %s", gen.Op)
}

// handleGeneratorStep handles OpcodeStep and OpcodeStepRun, both indicating that a function step
// has finished
func (e *executor) handleGeneratorStep(ctx context.Context, gen state.GeneratorOpcode, item queue.Item, edge queue.PayloadEdge) error {
	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Going from the current step
		Incoming: edge.Edge.Incoming, // And re-calling the incoming function in a loop
	}

	// Save the response to the state store.
	output, err := gen.Output()
	if err != nil {
		return err
	}

	if err := e.sm.SaveResponse(ctx, item.Identifier, gen.ID, output); err != nil {
		return err
	}

	// Update the group ID in context;  we've already saved this step's success and we're now
	// running the step again, needing a new history group
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

	// Re-enqueue the exact same edge to run now.
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
	err = e.queue.Enqueue(ctx, nextItem, time.Now())
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, l := range e.lifecycles {
		// We can't specify step name here since that will result in the
		// "followup discovery step" having the same name as its predecessor.
		var stepName *string = nil
		go l.OnStepScheduled(ctx, item.Identifier, nextItem, stepName)
	}

	return err
}

func (e *executor) handleStepError(ctx context.Context, gen state.GeneratorOpcode, item queue.Item, edge queue.PayloadEdge) error {
	// With the introduction of the StepError opcode, step errors are handled graceully and we can
	// finally distinguish between application level errors (this function) and network errors/other
	// errors (as the SDK didn't return this opcode).
	//
	// Here, we need to process the error and ensure that we reschedule the job for the future.
	//
	// Things to bear in mind:
	// - Steps throwing/returning NonRetriableErrors are still OpcodeStepError
	// - We are now in charge of rescheduling the entire function

	if gen.Error == nil {
		// This should never happen.
		logger.StdlibLogger(ctx).Error("OpcodeStepError handled without user error", "gen", gen)
		return fmt.Errorf("no user error defined in OpcodeStepError")
	}

	// If this is the last attempt, store the error in the state store, with a
	// wrapping of "error".  The wrapping allows SDKs to understand whether the
	// memoized step data is an error (and they should throw/return an error) or
	// real data.
	//
	// State stored for each step MUST always be wrapped with either "error" or "data".
	retryable := true

	if gen.Error.NoRetry {
		// This is a NonRetryableError thrown in a step.
		retryable = false
	}
	if !queue.ShouldRetry(nil, item.Attempt, item.GetMaxAttempts()) {
		// This is the last attempt as per the attempt in the queue, which
		// means we've failed N times, and so it is not retryable.
		retryable = false
	}

	if retryable {
		// Return an error to trigger standard queue retries.
		for _, l := range e.lifecycles {
			item.Attempt += 1
			go l.OnStepScheduled(ctx, item.Identifier, item, &gen.Name)
		}
		return ErrHandledStepError
	}

	// This was the final step attempt and we still failed.
	//
	// First, save the error to our state store.
	//
	// Note that `onStepFinished` is called immediately after a step response is returned, so
	// the history for this error will have already been handled.
	output, err := gen.Output()
	if err != nil {
		return err
	}
	if err := e.sm.SaveResponse(ctx, item.Identifier, gen.ID, output); err != nil {
		return err
	}

	// Because this is a final step error that was handled gracefully, enqueue
	// another attempt to the function with a new edge type.
	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Going from the current step
		Incoming: edge.Edge.Incoming, // And re-calling the incoming function in a loop
	}
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)
	jobID := fmt.Sprintf("%s-%s-failure", item.Identifier.IdempotencyKey(), gen.ID)

	nextItem := queue.Item{
		JobID:       &jobID,
		WorkspaceID: item.WorkspaceID,
		GroupID:     groupID,
		Kind:        queue.KindEdgeError,
		Identifier:  item.Identifier,
		Attempt:     0,
		MaxAttempts: item.MaxAttempts,
		Payload:     queue.PayloadEdge{Edge: nextEdge},
	}
	err = e.queue.Enqueue(ctx, nextItem, time.Now())
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, l := range e.lifecycles {
		go l.OnStepScheduled(ctx, item.Identifier, nextItem, nil)
	}

	return nil
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
		go l.OnStepScheduled(ctx, item.Identifier, nextItem, &gen.Name)
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

	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Leaving sleep
		Incoming: edge.Edge.Incoming, // To re-call the SDK
	}

	// Create another group for the next item which will run.  We're enqueueing
	// the function to run again after sleep, so need a new group.
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

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

func (e *executor) handleGeneratorInvokeFunction(ctx context.Context, gen state.GeneratorOpcode, item queue.Item, edge queue.PayloadEdge) error {
	logger.From(ctx).Info().Msg("handling invoke function")
	if e.handleSendingEvent == nil {
		return fmt.Errorf("no handleSendingEvent function specified")
	}

	opts, err := gen.InvokeFunctionOpts()
	if err != nil {
		return fmt.Errorf("unable to parse invoke function opts: %w", err)
	}
	expires, err := opts.Expires()
	if err != nil {
		return fmt.Errorf("unable to parse invoke function expires: %w", err)
	}

	eventName := event.FnFinishedName
	correlationID := item.Identifier.RunID.String() + "." + gen.ID
	strExpr := fmt.Sprintf("async.data.%s == %s", consts.InvokeCorrelationId, strconv.Quote(correlationID))
	_, err = e.newExpressionEvaluator(ctx, strExpr)
	if err != nil {
		return execError{err: fmt.Errorf("failed to create expression to wait for invoked function completion: %w", err)}
	}

	logger.From(ctx).Info().Interface("opts", opts).Time("expires", expires).Str("event", eventName).Str("expr", strExpr).Msg("parsed invoke function opts")

	pauseID := uuid.NewSHA1(
		uuid.NameSpaceOID,
		[]byte(item.Identifier.RunID.String()+gen.ID),
	)

	opcode := gen.Op.String()
	err = e.sm.SavePause(ctx, state.Pause{
		ID:          pauseID,
		WorkspaceID: item.WorkspaceID,
		Identifier:  item.Identifier,
		GroupID:     item.GroupID,
		Outgoing:    gen.ID,
		Incoming:    edge.Edge.Incoming,
		StepName:    gen.UserDefinedName(),
		Opcode:      &opcode,
		Expires:     state.Time(expires),
		Event:       &eventName,
		Expression:  &strExpr,
		DataKey:     gen.ID,
	})
	if err == state.ErrPauseAlreadyExists {
		return nil
	}
	if err != nil {
		return err
	}

	// Enqueue a job that will timeout the pause.
	jobID := fmt.Sprintf("%s-%s-%s", item.Identifier.IdempotencyKey(), gen.ID, "invoke")
	err = e.queue.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
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

	// Always create an invocation event.
	evt := event.NewInvocationEvent(event.NewInvocationEventOpts{
		Event:         *opts.Payload,
		FnID:          opts.FunctionID,
		CorrelationID: &correlationID,
	})

	err = e.handleSendingEvent(ctx, evt, item)
	if err != nil {
		// TODO Cancel pause/timeout?
		return fmt.Errorf("error publishing internal invocation event: %w", err)
	}

	for _, e := range e.lifecycles {
		go e.OnInvokeFunction(context.WithoutCancel(ctx), item.Identifier, item, gen, ulid.MustParse(evt.ID), correlationID)
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
		if err := expressions.Validate(ctx, *opts.If); err != nil {
			return execError{err, true}
		}

		expr, err := e.newExpressionEvaluator(ctx, *opts.If)
		if err != nil {
			return execError{err, true}
		}

		run, err := e.sm.Load(ctx, item.Identifier.RunID)
		if err != nil {
			return execError{err: fmt.Errorf("unable to load run after execution: %w", err)}
		}

		// Take the data for expressions based off of state.
		ed := expressions.NewData(state.ExpressionData(ctx, run))
		data = expr.FilteredAttributes(ctx, ed).Map()
	}

	pauseID := uuid.NewSHA1(
		uuid.NameSpaceOID,
		[]byte(item.Identifier.RunID.String()+gen.ID),
	)

	expr := opts.If
	if expr != nil && strings.Contains(*expr, "event.") {
		// Remove `event` data from the expression and replace with actual event
		// data as values, now that we have the event.
		//
		// This improves performance in matching, as we can then use the values within
		// aggregate trees.
		if state, err := e.sm.Load(ctx, item.Identifier.RunID); err != nil {
			logger.StdlibLogger(ctx).Error(
				"error loading state to interpolate waitForEvent",
				"error", err,
				"run_id", item.Identifier.RunID,
			)
		} else {
			interpolated, err := expressions.Interpolate(ctx, *opts.If, map[string]any{
				"event": state.Event(),
			})
			if err != nil {
				logger.StdlibLogger(ctx).Warn(
					"error interpolating waitForEvent expression",
					"error", err,
					"expression", *opts.If,
				)
			}
			expr = &interpolated
		}

		// Update the generator to use the interpolated data, ensuring history is updated.
		opts.If = expr
		gen.Opts = opts
	}

	opcode := gen.Op.String()
	err = e.sm.SavePause(ctx, state.Pause{
		ID:             pauseID,
		WorkspaceID:    item.WorkspaceID,
		Identifier:     item.Identifier,
		GroupID:        item.GroupID,
		Outgoing:       gen.ID,
		Incoming:       edge.Edge.Incoming,
		StepName:       gen.UserDefinedName(),
		Opcode:         &opcode,
		Expires:        state.Time(expires),
		Event:          &opts.Event,
		Expression:     expr,
		ExpressionData: data,
		DataKey:        gen.ID,
	})
	if err == state.ErrPauseAlreadyExists {
		return nil
	}
	if err != nil {
		return err
	}

	// SDK-based event coordination is called both when an event is received
	// OR on timeout, depending on which happens first.  Both routes consume
	// the pause so this race will conclude by calling the function once, as only
	// one thread can lease and consume a pause;  the other will find that the
	// pause is no longer available and return.
	jobID := fmt.Sprintf("%s-%s-%s", item.Identifier.IdempotencyKey(), gen.ID, "wait")
	err = e.queue.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
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

func generateCancelExpression(eventID ulid.ULID, expr *string) string {
	// Ensure that we only listen to cancellation events that occur
	// after the initial event is received.
	//
	// NOTE: We don't use `event.ts` here as people can use a future-TS date
	// to schedule future runs.  Events received between now and that date should
	// still cancel the run.
	res := fmt.Sprintf("(async.ts == null || async.ts > %d)", eventID.Time())
	if expr != nil {
		res = *expr + " && " + res
	}
	return res
}
