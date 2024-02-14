package executor

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
		StaticVersion:   req.StaticVersion,
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
	state, err := e.sm.Load(ctx, id.RunID)
	if err != nil {
		return nil, err
	}
	i := instance{
		executor: e,
		state:    state,
	}
	return i.execute(ctx, id, item, edge, stackIndex)
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
