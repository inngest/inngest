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
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog"
	"github.com/xhit/go-str2duration/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

var (
	ErrRuntimeRegistered = fmt.Errorf("runtime is already registered")
	ErrNoStateManager    = fmt.Errorf("no state manager provided")
	ErrNoPauseManager    = fmt.Errorf("no pause manager provided")
	ErrNoActionLoader    = fmt.Errorf("no action loader provided")
	ErrNoRuntimeDriver   = fmt.Errorf("runtime driver for action not found")
	ErrFunctionDebounced = fmt.Errorf("function debounced")
	ErrFunctionSkipped   = fmt.Errorf("function skipped")

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

	if m.smv2 == nil {
		return nil, ErrNoStateManager
	}

	if m.pm == nil {
		return nil, ErrNoPauseManager
	}

	return m, nil
}

// ExecutorOpt modifies the built-in executor on creation.
type ExecutorOpt func(m execution.Executor) error

func WithCancellationChecker(c cancellation.Checker) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).cancellationChecker = c
		return nil
	}
}

// WithStateManager sets which state manager to use when creating an executor.
func WithStateManager(sm sv2.RunService) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).smv2 = sm
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

// WithPauseManager sets which pause manager to use when creating an executor.
func WithPauseManager(pm state.PauseManager) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).pm = pm
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

func WithFinalizer(f execution.FinalizePublisher) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).SetFinalizer(f)
		return nil
	}
}

func WithInvokeFailHandler(f execution.InvokeFailHandler) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).invokeFailHandler = f
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

func WithStepLimits(limit func(id sv2.ID) int) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).steplimit = limit
		return nil
	}
}

func WithStateSizeLimits(limit func(id sv2.ID) int) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).stateSizeLimit = limit
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

func WithPreDeleteStateSizeReporter(f execution.PreDeleteStateSizeReporter) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).preDeleteStateSizeReporter = f
		return nil
	}
}

// executor represents a built-in executor for running workflows.
type executor struct {
	log *zerolog.Logger

	// exprAggregator is an expression aggregator used to parse and aggregate expressions
	// using trees.
	exprAggregator expressions.Aggregator

	pm   state.PauseManager
	smv2 sv2.RunService

	queue               queue.Queue
	debouncer           debounce.Debouncer
	batcher             batch.BatchManager
	fl                  state.FunctionLoader
	evalFactory         func(ctx context.Context, expr string) (expressions.Evaluator, error)
	runtimeDrivers      map[string]driver.Driver
	finishHandler       execution.FinalizePublisher
	invokeFailHandler   execution.InvokeFailHandler
	handleSendingEvent  execution.HandleSendingEvent
	cancellationChecker cancellation.Checker

	lifecycles []execution.LifecycleListener

	// steplimit finds step limits for a given run.
	steplimit func(sv2.ID) int

	// stateSizeLimit finds state size limits for a given run
	stateSizeLimit func(sv2.ID) int

	preDeleteStateSizeReporter execution.PreDeleteStateSizeReporter
}

func (e *executor) SetFinalizer(f execution.FinalizePublisher) {
	e.finishHandler = f
}

func (e *executor) SetInvokeFailHandler(f execution.InvokeFailHandler) {
	e.invokeFailHandler = f
}

func (e *executor) InvokeFailHandler(ctx context.Context, opts execution.InvokeFailHandlerOpts) error {
	if e.invokeFailHandler == nil {
		return nil
	}

	evt := CreateInvokeFailedEvent(ctx, opts)

	return e.invokeFailHandler(ctx, opts, []event.Event{evt})
}

func (e *executor) AddLifecycleListener(l execution.LifecycleListener) {
	e.lifecycles = append(e.lifecycles, l)
}

func (e *executor) CloseLifecycleListeners(ctx context.Context) {
	var eg errgroup.Group

	for _, l := range e.lifecycles {
		ll := l
		eg.Go(func() error {
			return ll.Close(ctx)
		})
	}

	if err := eg.Wait(); err != nil {
		log.From(ctx).Error().Err(err).Msg("error closing lifecycle listeners")
	}
}

func idempotencyKey(req execution.ScheduleRequest, runID ulid.ULID) string {
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

	// The idempotency key is always prefixed by the function ID.
	return fmt.Sprintf("%s-%s", util.XXHash(req.Function.ID.String()), util.XXHash(key))
}

// Execute loads a workflow and the current run state, then executes the
// function's step via the necessary driver.
//
// If this function has a debounce config, this will return ErrFunctionDebounced instead
// of an identifier as the function is not scheduled immediately.
func (e *executor) Schedule(ctx context.Context, req execution.ScheduleRequest) (*sv2.Metadata, error) {
	if req.Function.Debounce != nil && !req.PreventDebounce {
		err := e.debouncer.Debounce(ctx, debounce.DebounceItem{
			AccountID:        req.AccountID,
			WorkspaceID:      req.WorkspaceID,
			AppID:            req.AppID,
			FunctionID:       req.Function.ID,
			FunctionVersion:  req.Function.FunctionVersion,
			EventID:          req.Events[0].GetInternalID(),
			Event:            req.Events[0].GetEvent(),
			FunctionPausedAt: req.FunctionPausedAt,
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

	key := idempotencyKey(req, runID)

	if req.Context == nil {
		req.Context = map[string]any{}
	}

	// Normalization
	eventIDs := []ulid.ULID{}
	for _, e := range req.Events {
		id := e.GetInternalID()
		eventIDs = append(eventIDs, id)
	}

	evts := make([]json.RawMessage, len(req.Events))
	for n, item := range req.Events {
		evt := item.GetEvent()
		// serialize this data to the span at the same time
		byt, err := json.Marshal(evt)
		if err != nil {
			return nil, fmt.Errorf("error marshalling event: %w", err)
		}
		evts[n] = byt
	}
	// Evaluate the run priority based off of the input event data.
	evtMap := req.Events[0].GetEvent().Map()
	factor, _ := req.Function.RunPriorityFactor(ctx, evtMap)
	// function run spanID
	spanID := telemetry.NewSpanID(ctx)

	metadata := sv2.Metadata{
		ID: sv2.ID{
			RunID:      runID,
			FunctionID: req.Function.ID,
			Tenant: sv2.Tenant{
				AppID:     req.AppID,
				EnvID:     req.WorkspaceID,
				AccountID: req.AccountID,
			},
		},
		Config: sv2.Config{
			FunctionVersion: req.Function.FunctionVersion,
			SpanID:          spanID.String(),
			EventIDs:        eventIDs,
			Idempotency:     key,
			ReplayID:        req.ReplayID,
			OriginalRunID:   req.OriginalRunID,
			PriorityFactor:  &factor,
			BatchID:         req.BatchID,
			Context:         req.Context,
		},
	}

	// Grab the cron schedule for function config.  This is necessary for fast
	// lookups, trace info, etc.
	if len(req.Events) == 1 && req.Events[0].GetEvent().Name == event.FnCronName {
		if cron, ok := req.Events[0].GetEvent().Data["cron"].(string); ok {
			metadata.Config.SetCronSchedule(cron)
		}
	}

	// FunctionSlug is not stored in V1 format, so needs to be stored in Context
	metadata.Config.SetFunctionSlug(req.Function.GetSlug())
	metadata.Config.SetDebounceFlag(req.PreventDebounce)

	carrier := telemetry.NewTraceCarrier(telemetry.WithTraceCarrierSpanID(&spanID))
	telemetry.UserTracer().Propagator().Inject(ctx, propagation.MapCarrier(carrier.Context))
	metadata.Config.SetFunctionTrace(carrier)
	// NOTE 2024-08-01:  backward compatibility, should be removed in the future
	metadata.Config.Context[consts.OtelPropagationKey] = carrier

	// If this is paused, immediately end just before creating state.
	isPaused := req.FunctionPausedAt != nil && req.FunctionPausedAt.Before(time.Now())
	if isPaused {
		for _, e := range e.lifecycles {
			go e.OnFunctionSkipped(context.WithoutCancel(ctx), metadata, execution.SkipState{
				CronSchedule: req.Events[0].GetEvent().CronSchedule(),
				Reason:       enums.SkipReasonFunctionPaused,
			})
		}
		return nil, ErrFunctionSkipped
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
			metadata.Config.CustomConcurrencyKeys = append(
				metadata.Config.CustomConcurrencyKeys,
				sv2.CustomConcurrency{
					Key:   limit.Evaluate(ctx, scopeID, evtMap),
					Hash:  limit.Hash,
					Limit: limit.Limit,
				},
			)
		}
	}

	//
	// Create throttle information prior to creating state.  This is used in the queue.
	//
	var throttle *queue.Throttle
	if req.Function.Throttle != nil {
		throttleKey := redis_state.HashID(ctx, req.Function.ID.String())
		if req.Function.Throttle.Key != nil {
			val, _, _ := expressions.Evaluate(ctx, *req.Function.Throttle.Key, map[string]any{
				"event": evtMap,
			})
			throttleKey = throttleKey + "-" + redis_state.HashID(ctx, fmt.Sprintf("%v", val))
		}
		throttle = &queue.Throttle{
			Key:    throttleKey,
			Limit:  int(req.Function.Throttle.Limit),
			Burst:  int(req.Function.Throttle.Burst),
			Period: int(req.Function.Throttle.Period.Seconds()),
		}
	}

	//
	// Create the run state.
	//

	err := e.smv2.Create(ctx, sv2.CreateState{
		Metadata: metadata,
		Events:   evts,
	})
	if err == state.ErrIdentifierExists {
		// This function was already created.
		return nil, state.ErrIdentifierExists
	}
	if err != nil {
		return nil, fmt.Errorf("error creating run state: %w", err)
	}

	//
	// Create cancellation pauses immediately, only if this is a non-batch event.
	//
	if req.BatchID == nil {
		for _, c := range req.Function.Cancel {
			pauseID := uuid.New()
			expires := time.Now().Add(consts.CancelTimeout)
			if c.Timeout != nil {
				dur, err := str2duration.ParseDuration(*c.Timeout)
				if err != nil {
					return &metadata, fmt.Errorf("error parsing cancel duration: %w", err)
				}
				expires = time.Now().Add(dur)
			}

			// Evaluate the expression.  This lets us inspect the expression's attributes
			// so that we can store only the attrs used in the expression in the pause,
			// saving space, bandwidth, etc.
			expr := generateCancelExpression(eventIDs[0], c.If)
			eval, err := expressions.NewExpressionEvaluator(ctx, expr)
			if err != nil {
				return &metadata, err
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
				"event": evtMap,
			})
			if err != nil {
				logger.StdlibLogger(ctx).Warn(
					"error interpolating cancellation expression",
					"error", err,
					"expression", expr,
				)
			}

			pause := state.Pause{
				WorkspaceID: req.WorkspaceID,
				Identifier: state.Identifier{
					RunID:       runID,
					WorkflowID:  req.Function.ID,
					WorkspaceID: req.WorkspaceID,
					AccountID:   req.AccountID,
					AppID:       req.AppID,
					EventID:     metadata.Config.EventID(),
					EventIDs:    metadata.Config.EventIDs,
				},
				ID:                pauseID,
				Expires:           state.Time(expires),
				Event:             &c.Event,
				Expression:        &interpolated,
				ExpressionData:    data,
				Cancel:            true,
				TriggeringEventID: &triggeringID,
			}
			err = e.pm.SavePause(ctx, pause)
			if err != nil {
				return &metadata, fmt.Errorf("error saving pause: %w", err)
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
		Identifier: state.Identifier{
			RunID:       runID,
			WorkflowID:  req.Function.ID,
			WorkspaceID: req.WorkspaceID,
			AccountID:   req.AccountID,
			AppID:       req.AppID,
			EventID:     metadata.Config.EventID(),
			EventIDs:    metadata.Config.EventIDs,
		},
		CustomConcurrencyKeys: metadata.Config.CustomConcurrencyKeys,
		PriorityFactor:        metadata.Config.PriorityFactor,
		Attempt:               0,
		MaxAttempts:           &sourceEdgeRetries,
		Payload: queue.PayloadEdge{
			Edge: inngest.SourceEdge,
		},
		Throttle: throttle,
	}
	err = e.queue.Enqueue(ctx, item, at)
	if err == redis_state.ErrQueueItemExists {
		return nil, state.ErrIdentifierExists
	}
	if err != nil {
		return nil, fmt.Errorf("error enqueueing source edge '%v': %w", queueKey, err)
	}

	for _, e := range req.Events {
		go func(ctx context.Context, evt event.Event) {
			if v, ok := evt.Data[consts.InngestEventDataPrefix]; ok {
				meta := event.InngestMetadata{}
				if err := meta.Decode(v); err == nil {
					if meta.InvokeTraceCarrier != nil && meta.InvokeTraceCarrier.CanResumePause() {
						ictx := telemetry.UserTracer().Propagator().Extract(ctx, propagation.MapCarrier(meta.InvokeTraceCarrier.Context))

						sid := meta.InvokeTraceCarrier.SpanID()

						cIDs := strings.Split(meta.InvokeCorrelationId, ".")
						if len(cIDs) != 2 {
							log.From(ctx).Error().Interface("metadata", meta).Msg("invalid invoke correlation ID")
							// format is invalid
							return
						}

						var mrunID ulid.ULID
						if meta.RunID() != nil {
							mrunID = *meta.RunID()
						}

						_, ispan := telemetry.NewSpan(ictx,
							telemetry.WithScope(consts.OtelScopeStep),
							telemetry.WithName(consts.OtelSpanInvoke),
							telemetry.WithTimestamp(meta.InvokeTraceCarrier.Timestamp),
							telemetry.WithSpanID(sid),
							telemetry.WithSpanAttributes(
								attribute.Bool(consts.OtelUserTraceFilterKey, true),
								attribute.String(consts.OtelSysAccountID, req.AccountID.String()),
								attribute.String(consts.OtelSysWorkspaceID, req.WorkspaceID.String()),
								attribute.String(consts.OtelSysAppID, meta.SourceAppID),
								attribute.String(consts.OtelSysFunctionID, meta.SourceFnID),
								attribute.Int(consts.OtelSysFunctionVersion, meta.SourceFnVersion),
								attribute.String(consts.OtelAttrSDKRunID, mrunID.String()),
								attribute.Int(consts.OtelSysStepAttempt, 0),    // ?
								attribute.Int(consts.OtelSysStepMaxAttempt, 1), // ?
								attribute.String(consts.OtelSysStepGroupID, meta.InvokeGroupID),
								attribute.String(consts.OtelSysStepOpcode, enums.OpcodeInvokeFunction.String()),
								attribute.String(consts.OtelSysStepDisplayName, meta.InvokeDisplayName),

								attribute.String(consts.OtelSysStepInvokeTargetFnID, req.Function.ID.String()),
								attribute.Int64(consts.OtelSysStepInvokeExpires, meta.InvokeExpiresAt),
								attribute.String(consts.OtelSysStepInvokeTriggeringEventID, evt.ID),
								attribute.String(consts.OtelSysStepInvokeRunID, runID.String()),
								attribute.Bool(consts.OtelSysStepInvokeExpired, false),
							),
						)
						defer ispan.End()
					}
				}
			}
		}(ctx, e.GetEvent())
	}

	for _, e := range e.lifecycles {
		go e.OnFunctionScheduled(context.WithoutCancel(ctx), metadata, item, req.Events)
	}

	return &metadata, nil
}

type runInstance struct {
	md         sv2.Metadata
	f          inngest.Function
	events     []json.RawMessage
	item       queue.Item
	edge       inngest.Edge
	stackIndex int
}

// Execute loads a workflow and the current run state, then executes the
// function's step via the necessary driver.
func (e *executor) Execute(ctx context.Context, id state.Identifier, item queue.Item, edge inngest.Edge) (*state.DriverResponse, error) {
	if e.fl == nil {
		return nil, fmt.Errorf("no function loader specified running step")
	}

	// If this is of type sleep, ensure that we save "nil" within the state store
	// for the outgoing edge ID.  This ensures that we properly increase the stack
	// for `tools.sleep` within generator functions.
	if item.Kind == queue.KindSleep && item.Attempt == 0 {
		err := e.smv2.SaveStep(ctx, sv2.ID{
			RunID:      id.RunID,
			FunctionID: id.WorkflowID,
			Tenant: sv2.Tenant{
				AppID:     id.AppID,
				EnvID:     id.WorkspaceID,
				AccountID: id.AccountID,
			},
		}, edge.Outgoing, []byte("null"))
		if !errors.Is(err, state.ErrDuplicateResponse) && err != nil {
			return nil, err
		}
		// After the sleep, we start a new step.  This means we also want to start a new
		// group ID, ensuring that we correlate the next step _after_ this sleep (to be
		// scheduled in this executor run)
		ctx = state.WithGroupID(ctx, uuid.New().String())
	}

	md, err := e.smv2.LoadMetadata(ctx, sv2.ID{
		RunID:      id.RunID,
		FunctionID: id.WorkflowID,
		Tenant: sv2.Tenant{
			AppID:     id.AppID,
			EnvID:     id.WorkspaceID,
			AccountID: id.AccountID,
		},
	})
	// XXX: MetadataNotFound -> assume fn is deleted.
	if err != nil {
		return nil, fmt.Errorf("cannot load metadata to execute run: %w", err)
	}

	// Find the stack index for the incoming step.
	//
	// stackIndex represents the stack pointer at the time this step was scheduled.
	// This lets SDKs correctly evaluate parallelism by replaying generated steps in the
	// right order.
	var stackIndex int
	for n, id := range md.Stack {
		if id == edge.Outgoing {
			stackIndex = n + 1
			break
		}
	}

	events, err := e.smv2.LoadEvents(ctx, md.ID)
	if err != nil {
		return nil, fmt.Errorf("cannot load run events: %w", err)
	}

	// for recording function start time after a successful step.
	start := time.Now()
	if !md.Config.StartedAt.IsZero() {
		start = md.Config.StartedAt
	}

	f, err := e.fl.LoadFunction(ctx, md.ID.Tenant.EnvID, md.ID.FunctionID)
	if err != nil {
		return nil, fmt.Errorf("error loading function for run: %w", err)
	}

	// Validate that the run can execute.
	v := newRunValidator(e, f, md, events, item) // TODO: Load events for this.
	if err := v.validate(ctx); err != nil {
		return nil, err
	}
	if v.stopWithoutRetry {
		if e.preDeleteStateSizeReporter != nil {
			e.preDeleteStateSizeReporter(ctx, md)
		}

		// Validation prevented execution and doesn't want the executor to retry, so
		// don't return an error - assume the function finishes and delete state.
		_, err := e.smv2.Delete(ctx, md.ID)
		return nil, err
	}

	// Store the metadata in context for future use and propagate trace
	// context. This can be used to reduce reads in the future.
	ctx = e.extractTraceCtx(ctx, md, &item)

	evtIDs := make([]string, len(id.EventIDs))
	for i, eid := range id.EventIDs {
		evtIDs[i] = eid.String()
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
			// Set the start time and spanID in metadata for subsequent runs
			// This should be an one time operation and is never updated after,
			// which is enforced on the Lua script.
			if err := e.smv2.UpdateMetadata(ctx, md.ID, sv2.MutableConfig{
				StartedAt:      start,
				ForceStepPlan:  md.Config.ForceStepPlan,
				RequestVersion: md.Config.RequestVersion,
			}); err != nil {
				log.From(ctx).Error().Err(err).Msg("error updating metadata on function start")
			}

			for _, e := range e.lifecycles {
				go e.OnFunctionStarted(context.WithoutCancel(ctx), md, item, events)
			}
		}
	}

	instance := runInstance{
		md:         md,
		f:          *f,
		events:     events,
		item:       item,
		edge:       edge,
		stackIndex: stackIndex,
	}

	resp, err := e.run(ctx, &instance)
	if resp == nil && err != nil {
		for _, e := range e.lifecycles {
			// OnStepFinished handles step success and step errors/failures.  It is
			// currently the responsibility of the lifecycle manager to handle the differing
			// step statuses when a step finishes.
			go e.OnStepFinished(context.WithoutCancel(ctx), md, item, edge, resp, err)
		}
		return nil, err
	}

	err = e.HandleResponse(ctx, &instance, resp)
	return resp, err
}

func (e *executor) HandleResponse(ctx context.Context, i *runInstance, resp *state.DriverResponse) error {
	for _, e := range e.lifecycles {
		// OnStepFinished handles step success and step errors/failures.  It is
		// currently the responsibility of the lifecycle manager to handle the differing
		// step statuses when a step finishes.
		//
		// TODO (tonyhb): This should probably change, as each lifecycle listener has to
		// do the same parsing & conditional checks.
		go e.OnStepFinished(context.WithoutCancel(ctx), i.md, i.item, i.edge, resp, nil)
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
				i.item.Attempt += 1
				go e.OnStepScheduled(context.WithoutCancel(ctx), i.md, i.item, &resp.Step.Name)
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
			// TODO: Refactor state input
			if performedFinalization, err := e.finalize(ctx, i.md, i.events, i.f.GetSlug(), *resp); err != nil {
				logger.From(ctx).Error().Err(err).Msg("error running finish handler")
			} else if performedFinalization {
				for _, e := range e.lifecycles {
					go e.OnFunctionFinished(context.WithoutCancel(ctx), i.md, i.item, i.events, *resp)
				}
			}

			return resp
		}
	}

	// This is a success, which means either a generator or a function result.
	if len(resp.Generator) > 0 {
		// Handle generator responses then return.
		if serr := e.HandleGeneratorResponse(ctx, i, resp); serr != nil {

			// If this is an error compiling async expressions, fail the function.
			shouldFailEarly := errors.Is(serr, &expressions.CompileError{}) || errors.Is(serr, state.ErrStateOverflowed)
			if shouldFailEarly {
				var gracefulErr *state.WrappedStandardError
				if hasGracefulErr := errors.As(serr, &gracefulErr); hasGracefulErr {
					serialized := gracefulErr.Serialize(execution.StateErrorKey)
					resp.Output = nil
					resp.Err = &serialized
				}

				if performedFinalization, err := e.finalize(ctx, i.md, i.events, i.f.GetSlug(), *resp); err != nil {
					logger.From(ctx).Error().Err(err).Msg("error running finish handler")
				} else if performedFinalization {
					for _, e := range e.lifecycles {
						go e.OnFunctionFinished(context.WithoutCancel(ctx), i.md, i.item, i.events, *resp)
					}
				}

				return nil
			}
			return fmt.Errorf("error handling generator response: %w", serr)
		}
		return nil
	}

	// This is the function result.
	if performedFinalization, err := e.finalize(ctx, i.md, i.events, i.f.GetSlug(), *resp); err != nil {
		logger.From(ctx).Error().Err(err).Msg("error running finish handler")
	} else if performedFinalization {
		for _, e := range e.lifecycles {
			go e.OnFunctionFinished(context.WithoutCancel(ctx), i.md, i.item, i.events, *resp)
		}
	}

	return nil
}

type functionFinishedData struct {
	FunctionID          string         `json:"function_id"`
	RunID               ulid.ULID      `json:"run_id"`
	Event               map[string]any `json:"event"`
	Events              []event.Event  `json:"events"`
	Error               any            `json:"error,omitempty"`
	Result              any            `json:"result,omitempty"`
	InvokeCorrelationID *string        `json:"correlation_id,omitempty"`
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

// finalize performs run finalization, which involves sending the function
// finished/failed event and deleting state.
//
// Returns a boolean indicating whether it performed finalization. If the run
// had parallel steps then it may be false, since parallel steps cause the
// function end to be reached multiple times in a single run
func (e *executor) finalize(ctx context.Context, md sv2.Metadata, evts []json.RawMessage, fnSlug string, resp state.DriverResponse) (bool, error) {
	// Parse events for the fail handler before deleting state.
	inputEvents := make([]event.Event, len(evts))
	for n, e := range evts {
		evt, err := event.NewEvent(e)
		if err != nil {
			return false, err
		}
		inputEvents[n] = *evt
	}

	if e.preDeleteStateSizeReporter != nil {
		e.preDeleteStateSizeReporter(ctx, md)
	}

	// Delete the function state in every case.
	performedFinalization, err := e.smv2.Delete(ctx, md.ID)
	if err != nil {
		logger.StdlibLogger(ctx).Error("error deleting state in finalize", "error", err)
	}
	if err == nil && !performedFinalization {
		return performedFinalization, nil
	}

	// TODO: Load all pauses for the function and remove, once we index pauses.

	if e.finishHandler == nil {
		return performedFinalization, nil
	}

	// Prepare events that we must send
	now := time.Now()
	base := &functionFinishedData{
		FunctionID: fnSlug,
		RunID:      md.ID.RunID,
		Events:     inputEvents,
	}
	base.setResponse(resp)

	// We'll send many events - some for each items in the batch.  This ensures that invoke works
	// for batched functions.
	freshEvents := []event.Event{}
	for n, runEvt := range inputEvents {
		if runEvt.Name == event.FnFailedName || runEvt.Name == event.FnFinishedName {
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
		copied.Event = runEvt.Map()
		copied.InvokeCorrelationID = invokeID
		data := copied.Map()

		// Add an `inngest/function.finished` event.
		freshEvents = append(freshEvents, event.Event{
			ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
			Name:      event.FnFinishedName,
			Timestamp: now.UnixMilli(),
			Data:      data,
		})

		// Legacy - send inngest/function.failed, except for when the function has been cancelled.
		if resp.Err != nil && !strings.Contains(*resp.Err, state.ErrFunctionCancelled.Error()) {
			freshEvents = append(freshEvents, event.Event{
				ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
				Name:      event.FnFailedName,
				Timestamp: now.UnixMilli(),
				Data:      data,
			})
		}
	}

	return performedFinalization, e.finishHandler(ctx, md.ID, freshEvents)
}

func correlationID(event event.Event) *string {
	container, ok := event.Data[consts.InngestEventDataPrefix].(map[string]any)
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
func (e *executor) run(ctx context.Context, i *runInstance) (*state.DriverResponse, error) {
	url, _ := i.f.URI()
	for _, e := range e.lifecycles {
		go e.OnStepStarted(context.WithoutCancel(ctx), i.md, i.item, i.edge, url.String())
	}

	// Execute the actual step.
	response, err := e.executeDriverForStep(ctx, i)

	if response.Err != nil && err == nil {
		// This step errored, so always return an error.
		return response, fmt.Errorf("%s", *response.Err)
	}
	return response, err
}

// executeDriverForStep runs the enqueued step by invoking the driver.  It also inspects
// and normalizes responses (eg. max retry attempts).
func (e *executor) executeDriverForStep(ctx context.Context, i *runInstance) (*state.DriverResponse, error) {
	url, _ := i.f.URI()

	driverName := inngest.SchemeDriver(url.Scheme)
	d, ok := e.runtimeDrivers[driverName]
	if !ok {
		return nil, fmt.Errorf("%w: '%s'", ErrNoRuntimeDriver, driverName)
	}

	step := &i.f.Steps[0]

	response, err := d.Execute(ctx, e.smv2, i.md, i.item, i.edge, *step, i.stackIndex, i.item.Attempt)

	// TODO: Steps.
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
		if !queue.ShouldRetry(nil, i.item.Attempt, step.RetryCount()+1) {
			response.NoRetry = true
		}
	}

	// Max attempts is encoded at the queue level from step configuration.  If we're at max attempts,
	// ensure the response's NoRetry flag is set, as we shouldn't retry any more.  This also ensures
	// that we properly handle this response as a Failure (permanent) vs an Error (transient).
	if response.Err != nil && !queue.ShouldRetry(nil, i.item.Attempt, step.RetryCount()+1) {
		response.NoRetry = true
	}

	return response, err
}

// HandlePauses handles pauses loaded from an incoming event.
func (e *executor) HandlePauses(ctx context.Context, iter state.PauseIterator, evt event.TrackedEvent) (execution.HandlePauseResult, error) {
	// Use the aggregator for all funciton finished events, if there are more than
	// 50 waiting.  It only takes a few milliseconds to iterate and handle less
	// than 50;  anything more runs the risk of running slow.
	if iter.Count() > 10 {
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

	if e.queue == nil || e.smv2 == nil || e.pm == nil {
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
				_ = e.pm.DeletePause(context.Background(), *pause)
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
				if exists, err := e.smv2.Exists(ctx, sv2.IDFromV1(pause.Identifier)); !exists && err == nil {
					// This function has ended.  Delete the pause and continue
					_ = e.pm.DeletePause(context.Background(), *pause)
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
				err := e.Cancel(ctx, sv2.IDFromV1(pause.Identifier), execution.CancelRequest{
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
				err = e.pm.ConsumePause(ctx, pause.ID, nil)
				if err == nil || err == state.ErrPauseLeased || err == state.ErrPauseNotFound || err == state.ErrRunNotFound {
					// Done. Add to the counter.
					_ = e.pm.DeletePause(context.Background(), *pause)
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
			if errors.Is(err, state.ErrPauseLeased) ||
				errors.Is(err, state.ErrPauseNotFound) ||
				errors.Is(err, state.ErrRunNotFound) {
				return
			}
			if err != nil {
				goerr = errors.Join(goerr, fmt.Errorf("error resuming pause: %w", err))
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

	log := logger.StdlibLogger(ctx).With("event_id", evt.GetInternalID().String())
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

			l := log.With(
				"pause_id", pause.ID.String(),
				"run_id", pause.Identifier.RunID.String(),
				"workflow_id", pause.Identifier.WorkflowID.String(),
				"expires", pause.Expires.String(),
			)

			// NOTE: Some pauses may be nil or expired, as the iterator may take
			// time to process.  We handle that here and assume that the event
			// did not occur in time.
			if pause.Expires.Time().Before(time.Now()) {
				// Consume this pause to remove it entirely
				l.Debug("deleting expired pause")
				_ = e.pm.DeletePause(context.Background(), pause)
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
				if exists, err := e.smv2.Exists(ctx, sv2.IDFromV1(pause.Identifier)); !exists && err == nil {
					// This function has ended.  Delete the pause and continue
					_ = e.pm.DeletePause(context.Background(), pause)
					_ = e.exprAggregator.RemovePause(ctx, pause)
					return
				}
			}

			// Ensure that we store the group ID for this pause, letting us properly track cancellation
			// or continuation history
			ctx = state.WithGroupID(ctx, pause.GroupID)

			// Cancelling a function can happen before a lease, as it's an atomic operation that will always happen.
			if pause.Cancel {
				err := e.Cancel(ctx, sv2.IDFromV1(pause.Identifier), execution.CancelRequest{
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
				err = e.pm.ConsumePause(ctx, pause.ID, nil)
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
			if errors.Is(err, state.ErrPauseLeased) ||
				errors.Is(err, state.ErrPauseNotFound) ||
				errors.Is(err, state.ErrRunNotFound) {
				return
			}
			if err != nil {
				goerr = errors.Join(goerr, fmt.Errorf("error resuming pause: %w", err))
				return
			}
			// Add to the counter.
			atomic.AddInt32(&res[1], 1)
			if err := e.exprAggregator.RemovePause(ctx, pause); err != nil {
				l.Warn("error removing pause from aggregator", "error", err)
			}
		}()
	}
	wg.Wait()

	return res, goerr
}

func (e *executor) HandleInvokeFinish(ctx context.Context, evt event.TrackedEvent) error {
	evtID := evt.GetInternalID()

	log := e.log
	if log == nil {
		log = logger.From(ctx)
	}
	l := log.With().Str("event_id", evtID.String()).Logger()

	correlationID := evt.GetEvent().CorrelationID()
	if correlationID == "" {
		return fmt.Errorf("no correlation ID found in event when trying to handle finish")
	}

	// find the pause with correlationID
	wsID := evt.GetWorkspaceID()
	pause, err := e.pm.PauseByInvokeCorrelationID(ctx, wsID, correlationID)
	if err != nil {
		return err
	}

	if pause.Expires.Time().Before(time.Now()) {
		// Consume this pause to remove it entirely
		l.Debug().Msg("deleting expired pause")
		_ = e.pm.DeletePause(context.Background(), *pause)
		return nil
	}

	if pause.Cancel {
		// This is a cancellation signal.  Check if the function
		// has ended, and if so remove the pause.
		//
		// NOTE: Bookkeeping must be added to individual function runs and handled on
		// completion instead of here.  This is a hot path and should only exist whilst
		// bookkeeping is not implemented.
		if exists, err := e.smv2.Exists(ctx, sv2.IDFromV1(pause.Identifier)); !exists && err == nil {
			// This function has ended.  Delete the pause and continue
			_ = e.pm.DeletePause(context.Background(), *pause)
			return nil
		}
	}

	resumeData := pause.GetResumeData(evt.GetEvent())
	if e.log != nil {
		e.log.
			Debug().
			Interface("with", resumeData.With).
			Str("pause.DataKey", pause.DataKey).
			Msg("resuming pause from invoke")
	}

	return e.Resume(ctx, *pause, execution.ResumeRequest{
		With:     resumeData.With,
		EventID:  &evtID,
		RunID:    resumeData.RunID,
		StepName: resumeData.StepName,
	})
}

// Cancel cancels an in-progress function.
func (e *executor) Cancel(ctx context.Context, id sv2.ID, r execution.CancelRequest) error {
	md, err := e.smv2.LoadMetadata(ctx, id)
	if err == sv2.ErrMetadataNotFound || err == state.ErrRunNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to load run: %w", err)
	}

	// We need events to finalize the function.
	evts, err := e.smv2.LoadEvents(ctx, id)
	if err != nil {
		return fmt.Errorf("unable to load run events: %w", err)
	}

	// We need the function slug.
	f, err := e.fl.LoadFunction(ctx, md.ID.Tenant.EnvID, md.ID.FunctionID)
	if err != nil {
		return fmt.Errorf("unable to load function: %w", err)
	}

	fnCancelledErr := state.ErrFunctionCancelled.Error()
	if performedFinalization, err := e.finalize(ctx, md, evts, f.GetSlug(), state.DriverResponse{
		Err: &fnCancelledErr,
	}); err != nil {
		logger.From(ctx).Error().Err(err).Msg("error running finish handler")
	} else if performedFinalization || r.ForceLifecycleHook {
		ctx = e.extractTraceCtx(ctx, md, nil)
		for _, e := range e.lifecycles {
			go e.OnFunctionCancelled(context.WithoutCancel(ctx), md, r, evts)
		}
	}

	return nil
}

// Resume resumes an in-progress function from the given pause.
func (e *executor) Resume(ctx context.Context, pause state.Pause, r execution.ResumeRequest) error {
	if e.queue == nil || e.smv2 == nil || e.pm == nil {
		return fmt.Errorf("No queue or state manager specified")
	}

	md, err := e.smv2.LoadMetadata(ctx, sv2.ID{
		RunID:      pause.Identifier.RunID,
		FunctionID: pause.Identifier.WorkflowID,
		Tenant: sv2.Tenant{
			AppID:     pause.Identifier.AppID,
			EnvID:     pause.Identifier.WorkspaceID,
			AccountID: pause.Identifier.AccountID,
		},
	})
	if err == state.ErrRunNotFound {
		return err
	}
	if err != nil {
		return fmt.Errorf("error loading metadata to resume from pause: %w", err)
	}

	// Lease this pause so that only this thread can schedule the execution.
	//
	// If we don't do this, there's a chance that two concurrent runners
	// attempt to enqueue the next step of the workflow.
	err = e.pm.LeasePause(ctx, pause.ID)
	if err == state.ErrPauseLeased || err == state.ErrPauseNotFound {
		// Ignore;  this is being handled by another runner.
		return nil
	}

	if pause.OnTimeout && r.EventID != nil {
		// Delete this pause, as an event has occured which matches
		// the timeout.  We can do this prior to leasing a pause as it's the
		// only work that needs to happen
		err := e.pm.ConsumePause(ctx, pause.ID, nil)
		if err == nil || err == state.ErrPauseNotFound {
			return nil
		}
		return err
	}

	if err = e.pm.ConsumePause(ctx, pause.ID, r.With); err != nil {
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
			GroupID:               uuid.New().String(),
			WorkspaceID:           pause.WorkspaceID,
			Kind:                  queue.KindEdge,
			Identifier:            pause.Identifier,
			PriorityFactor:        md.Config.PriorityFactor,
			CustomConcurrencyKeys: md.Config.CustomConcurrencyKeys,
			MaxAttempts:           pause.MaxAttempts,
			Payload: queue.PayloadEdge{
				Edge: pause.Edge(),
			},
		},
		time.Now(),
	)
	if err != nil && err != redis_state.ErrQueueItemExists {
		return fmt.Errorf("error enqueueing after pause: %w", err)
	}

	if pause.IsInvoke() {
		for _, e := range e.lifecycles {
			go e.OnInvokeFunctionResumed(context.WithoutCancel(ctx), md, pause, r)
		}
	} else {
		for _, e := range e.lifecycles {
			go e.OnWaitForEventResumed(context.WithoutCancel(ctx), md, pause, r)
		}
	}

	return nil
}

func (e *executor) HandleGeneratorResponse(ctx context.Context, i *runInstance, resp *state.DriverResponse) error {
	{
		// The following code helps with parallelism and the V2 -> V3 executor changes
		var update *sv2.MutableConfig
		// NOTE: We only need to set hash versions when handling generator responses, else the
		// fn is ending and it doesn't matter.
		if i.md.Config.RequestVersion == -1 {
			update = &sv2.MutableConfig{
				ForceStepPlan:  i.md.Config.ForceStepPlan,
				RequestVersion: resp.RequestVersion,
				StartedAt:      i.md.Config.StartedAt,
			}
		}
		if len(resp.Generator) > 1 {
			if !i.md.Config.ForceStepPlan {
				// With parallelism, we currently instruct the SDK to disable immediate execution,
				// enforcing that every step becomes pre-planned.
				if update == nil {
					update = &sv2.MutableConfig{
						ForceStepPlan:  i.md.Config.ForceStepPlan,
						RequestVersion: resp.RequestVersion,
						StartedAt:      i.md.Config.StartedAt,
					}
				}
				update.ForceStepPlan = true
			}
		}
		if update != nil {
			if err := e.smv2.UpdateMetadata(ctx, i.md.ID, *update); err != nil {
				return fmt.Errorf("error updating function metadata: %w", err)
			}
		}
	}

	groups := opGroups(resp.Generator).All()
	for _, group := range groups {
		if err := e.handleGeneratorGroup(ctx, i, group, resp); err != nil {
			return err
		}
	}

	return nil
}

func (e *executor) handleGeneratorGroup(ctx context.Context, i *runInstance, group OpcodeGroup, resp *state.DriverResponse) error {
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
		newItem := i.item
		if group.ShouldStartHistoryGroup {
			// Give each opcode its own group ID, since we want to track each
			// parellel step individually.
			newItem.GroupID = uuid.New().String()
		}
		eg.Go(func() error { return e.handleGenerator(ctx, i, copied) })
	}
	if err := eg.Wait(); err != nil {
		if errors.Is(err, state.ErrStateOverflowed) {
			return err
		}
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

func (e *executor) handleGenerator(ctx context.Context, i *runInstance, gen state.GeneratorOpcode) error {
	// Grab the edge that triggered this step execution.
	edge, ok := i.item.Payload.(queue.PayloadEdge)
	if !ok {
		return fmt.Errorf("unknown queue item type handling generator: %T", i.item.Payload)
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
		return e.handleGeneratorStep(ctx, i, gen, edge)
	case enums.OpcodeStepError:
		return e.handleStepError(ctx, i, gen, edge)
	case enums.OpcodeStepPlanned:
		return e.handleGeneratorStepPlanned(ctx, i, gen, edge)
	case enums.OpcodeSleep:
		return e.handleGeneratorSleep(ctx, i, gen, edge)
	case enums.OpcodeWaitForEvent:
		return e.handleGeneratorWaitForEvent(ctx, i, gen, edge)
	case enums.OpcodeInvokeFunction:
		return e.handleGeneratorInvokeFunction(ctx, i, gen, edge)
	}

	return fmt.Errorf("unknown opcode: %s", gen.Op)
}

// handleGeneratorStep handles OpcodeStep and OpcodeStepRun, both indicating that a function step
// has finished
func (e *executor) handleGeneratorStep(ctx context.Context, i *runInstance, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Going from the current step
		Incoming: edge.Edge.Incoming, // And re-calling the incoming function in a loop
	}

	// Save the response to the state store.
	output, err := gen.Output()
	if err != nil {
		return err
	}

	if err := e.validateStateSize(len(output), i.md); err != nil {
		return err
	}

	if err := e.smv2.SaveStep(ctx, i.md.ID, gen.ID, []byte(output)); err != nil {
		return err
	}

	// Update the group ID in context;  we've already saved this step's success and we're now
	// running the step again, needing a new history group
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

	// Re-enqueue the exact same edge to run now.
	jobID := fmt.Sprintf("%s-%s", i.md.IdempotencyKey(), gen.ID)
	now := time.Now()
	nextItem := queue.Item{
		JobID:                 &jobID,
		WorkspaceID:           i.md.ID.Tenant.EnvID,
		GroupID:               groupID,
		Kind:                  queue.KindEdge,
		Identifier:            i.item.Identifier, // TODO: Refactor
		PriorityFactor:        i.item.PriorityFactor,
		CustomConcurrencyKeys: i.item.CustomConcurrencyKeys,
		Attempt:               0,
		MaxAttempts:           i.item.MaxAttempts,
		Payload:               queue.PayloadEdge{Edge: nextEdge},
	}
	err = e.queue.Enqueue(ctx, nextItem, now)
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, l := range e.lifecycles {
		// We can't specify step name here since that will result in the
		// "followup discovery step" having the same name as its predecessor.
		var stepName *string = nil
		go l.OnStepScheduled(ctx, i.md, nextItem, stepName)
	}

	return err
}

func (e *executor) handleStepError(ctx context.Context, i *runInstance, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	// With the introduction of the StepError opcode, step errors are handled graceully and we can
	// finally distinguish between application level errors (this function) and network errors/other
	// errors (as the SDK didn't return this opcode).
	//
	// Here, we need to process the error and ensure that we reschedule the job for the future.
	//
	// Things to bear in mind:
	// - Steps throwing/returning NonRetriableErrors are still OpcodeStepError
	// - We are now in charge of rescheduling the entire function
	span := trace.SpanFromContext(ctx)
	span.SetStatus(codes.Error, gen.Error.Name)

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
	if !queue.ShouldRetry(nil, i.item.Attempt, i.item.GetMaxAttempts()) {
		// This is the last attempt as per the attempt in the queue, which
		// means we've failed N times, and so it is not retryable.
		retryable = false
	}

	if retryable {
		// Return an error to trigger standard queue retries.
		for _, l := range e.lifecycles {
			i.item.Attempt += 1
			go l.OnStepScheduled(ctx, i.md, i.item, &gen.Name)
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
	if err := e.smv2.SaveStep(ctx, i.md.ID, gen.ID, []byte(output)); err != nil {
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

	// This is the discovery step to find what happens after we error
	jobID := fmt.Sprintf("%s-%s-failure", i.md.IdempotencyKey(), gen.ID)
	now := time.Now()
	nextItem := queue.Item{
		JobID:                 &jobID,
		WorkspaceID:           i.md.ID.Tenant.EnvID,
		GroupID:               groupID,
		Kind:                  queue.KindEdgeError,
		Identifier:            i.item.Identifier,
		PriorityFactor:        i.item.PriorityFactor,
		CustomConcurrencyKeys: i.item.CustomConcurrencyKeys,
		Attempt:               0,
		MaxAttempts:           i.item.MaxAttempts,
		Payload:               queue.PayloadEdge{Edge: nextEdge},
	}
	err = e.queue.Enqueue(ctx, nextItem, now)
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, l := range e.lifecycles {
		go l.OnStepScheduled(ctx, i.md, nextItem, nil)
	}

	return nil
}

func (e *executor) handleGeneratorStepPlanned(ctx context.Context, i *runInstance, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
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
	jobID := fmt.Sprintf("%s-%s", i.item.Identifier.IdempotencyKey(), gen.ID+"-plan")
	now := time.Now()
	nextItem := queue.Item{
		JobID:                 &jobID,
		GroupID:               groupID, // Ensure we correlate future jobs with this group ID, eg. started/failed.
		WorkspaceID:           i.md.ID.Tenant.EnvID,
		Kind:                  queue.KindEdge,
		Identifier:            i.item.Identifier,
		PriorityFactor:        i.item.PriorityFactor,
		CustomConcurrencyKeys: i.item.CustomConcurrencyKeys,
		Attempt:               0,
		MaxAttempts:           i.item.MaxAttempts,
		Payload: queue.PayloadEdge{
			Edge: nextEdge,
		},
	}
	err := e.queue.Enqueue(ctx, nextItem, now)
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, l := range e.lifecycles {
		go l.OnStepScheduled(ctx, i.md, nextItem, &gen.Name)
	}
	return err
}

// handleSleep handles the sleep opcode, ensuring that we enqueue the function to rerun
// at the correct time.
func (e *executor) handleGeneratorSleep(ctx context.Context, i *runInstance, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	dur, err := gen.SleepDuration()
	if err != nil {
		return err
	}

	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Leaving sleep
		Incoming: edge.Edge.Incoming, // To re-call the SDK
	}

	startedAt := time.Now()
	until := startedAt.Add(dur)

	// Create another group for the next item which will run.  We're enqueueing
	// the function to run again after sleep, so need a new group.
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

	jobID := fmt.Sprintf("%s-%s", i.md.IdempotencyKey(), gen.ID)
	// TODO Should this also include a parent step span? It will never have attempts.
	err = e.queue.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
		WorkspaceID: i.md.ID.Tenant.EnvID,
		// Sleeps re-enqueue the step so that we can mark the step as completed
		// in the executor after the sleep is complete.  This will re-call the
		// generator step, but we need the same group ID for correlation.
		GroupID:               groupID,
		Kind:                  queue.KindSleep,
		Identifier:            i.item.Identifier,
		PriorityFactor:        i.item.PriorityFactor,
		CustomConcurrencyKeys: i.item.CustomConcurrencyKeys,
		Attempt:               0,
		MaxAttempts:           i.item.MaxAttempts,
		Payload:               queue.PayloadEdge{Edge: nextEdge},
	}, until)
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, e := range e.lifecycles {
		go e.OnSleep(context.WithoutCancel(ctx), i.md, i.item, gen, until)
	}

	return err
}

func (e *executor) handleGeneratorInvokeFunction(ctx context.Context, i *runInstance, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
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
	correlationID := i.md.ID.RunID.String() + "." + gen.ID
	strExpr := fmt.Sprintf("async.data.%s == %s", consts.InvokeCorrelationId, strconv.Quote(correlationID))
	_, err = e.newExpressionEvaluator(ctx, strExpr)
	if err != nil {
		return execError{err: fmt.Errorf("failed to create expression to wait for invoked function completion: %w", err)}
	}

	pauseID := uuid.NewSHA1(
		uuid.NameSpaceOID,
		[]byte(i.md.ID.RunID.String()+gen.ID),
	)
	opcode := gen.Op.String()
	now := time.Now()

	sid := telemetry.NewSpanID(ctx)
	// NOTE: the context here still contains the execSpan's traceID & spanID,
	// which is what we want because that's the parent that needs to be referenced later on
	carrier := telemetry.NewTraceCarrier(
		telemetry.WithTraceCarrierTimestamp(now),
		telemetry.WithTraceCarrierSpanID(&sid),
	)
	telemetry.UserTracer().Propagator().Inject(ctx, propagation.MapCarrier(carrier.Context))

	// Always create an invocation event.
	evt := event.NewInvocationEvent(event.NewInvocationEventOpts{
		Event:           *opts.Payload,
		FnID:            opts.FunctionID,
		CorrelationID:   &correlationID,
		TraceCarrier:    carrier,
		ExpiresAt:       expires.UnixMilli(),
		GroupID:         i.item.GroupID,
		DisplayName:     gen.UserDefinedName(),
		SourceAppID:     i.item.Identifier.AppID.String(),
		SourceFnID:      i.item.Identifier.WorkflowID.String(),
		SourceFnVersion: i.item.Identifier.WorkflowVersion,
	})

	ctx, span := telemetry.NewSpan(ctx,
		telemetry.WithScope(consts.OtelScopeStep),
		telemetry.WithName(consts.OtelSpanInvoke),
		telemetry.WithTimestamp(now),
		telemetry.WithSpanID(sid),
		telemetry.WithSpanAttributes(
			attribute.Bool(consts.OtelUserTraceFilterKey, true),
			attribute.String(consts.OtelSysAccountID, i.item.Identifier.AccountID.String()),
			attribute.String(consts.OtelSysWorkspaceID, i.item.Identifier.WorkspaceID.String()),
			attribute.String(consts.OtelSysAppID, i.item.Identifier.AppID.String()),
			attribute.String(consts.OtelSysFunctionID, i.item.Identifier.WorkflowID.String()),
			attribute.String(consts.OtelSysFunctionSlug, i.md.Config.FunctionSlug()),
			attribute.Int(consts.OtelSysFunctionVersion, i.item.Identifier.WorkflowVersion),
			attribute.String(consts.OtelAttrSDKRunID, i.item.Identifier.RunID.String()),
			attribute.Int(consts.OtelSysStepAttempt, 0),    // ?
			attribute.Int(consts.OtelSysStepMaxAttempt, 1), // ?
			attribute.String(consts.OtelSysStepGroupID, i.item.GroupID),
			attribute.String(consts.OtelSysStepOpcode, enums.OpcodeInvokeFunction.String()),
			attribute.String(consts.OtelSysStepDisplayName, gen.UserDefinedName()),
			attribute.String(consts.OtelSysStepInvokeTargetFnID, opts.FunctionID),
			attribute.Int64(consts.OtelSysStepInvokeExpires, expires.UnixMilli()),
			attribute.String(consts.OtelSysStepInvokeTriggeringEventID, evt.ID),
		),
	)
	defer span.End()

	err = e.pm.SavePause(ctx, state.Pause{
		ID:                  pauseID,
		WorkspaceID:         i.md.ID.Tenant.EnvID,
		Identifier:          i.item.Identifier,
		GroupID:             i.item.GroupID,
		Outgoing:            gen.ID,
		Incoming:            edge.Edge.Incoming,
		StepName:            gen.UserDefinedName(),
		Opcode:              &opcode,
		Expires:             state.Time(expires),
		Event:               &eventName,
		Expression:          &strExpr,
		DataKey:             gen.ID,
		InvokeCorrelationID: &correlationID,
		TriggeringEventID:   &evt.ID,
		InvokeTargetFnID:    &opts.FunctionID,
		MaxAttempts:         i.item.MaxAttempts,
		Metadata: map[string]any{
			consts.OtelPropagationKey: carrier,
		},
	})
	if err == state.ErrPauseAlreadyExists {
		span.Cancel(ctx)
		return nil
	}
	if err != nil {
		span.Cancel(ctx)
		return err
	}

	// Enqueue a job that will timeout the pause.
	jobID := fmt.Sprintf("%s-%s-%s", i.md.IdempotencyKey(), gen.ID, "invoke")
	// TODO I think this is fine sending no metadata, as we have no attempts.
	err = e.queue.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
		WorkspaceID: i.md.ID.Tenant.EnvID,
		// Use the same group ID, allowing us to track the cancellation of
		// the step correctly.
		GroupID:               i.item.GroupID,
		Kind:                  queue.KindPause,
		Identifier:            i.item.Identifier,
		PriorityFactor:        i.item.PriorityFactor,
		CustomConcurrencyKeys: i.item.CustomConcurrencyKeys,
		MaxAttempts:           i.item.MaxAttempts,
		Payload: queue.PayloadPauseTimeout{
			PauseID:   pauseID,
			OnTimeout: true,
		},
	}, expires)
	if err == redis_state.ErrQueueItemExists {
		span.Cancel(ctx)
		return nil
	}

	// Send the event.
	err = e.handleSendingEvent(ctx, evt, i.item)
	if err != nil {
		span.Cancel(ctx)
		// TODO Cancel pause/timeout?
		return fmt.Errorf("error publishing internal invocation event: %w", err)
	}

	for _, e := range e.lifecycles {
		go e.OnInvokeFunction(context.WithoutCancel(ctx), i.md, i.item, gen, ulid.MustParse(evt.ID), correlationID)
	}

	return err
}

func (e *executor) handleGeneratorWaitForEvent(ctx context.Context, i *runInstance, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	opts, err := gen.WaitForEventOpts()
	if err != nil {
		return fmt.Errorf("unable to parse wait for event opts: %w", err)
	}
	expires, err := opts.Expires()
	if err != nil {
		return fmt.Errorf("unable to parse wait for event expires: %w", err)
	}

	pauseID := uuid.NewSHA1(
		uuid.NameSpaceOID,
		[]byte(i.md.ID.RunID.String()+gen.ID),
	)

	expr := opts.If
	if expr != nil && strings.Contains(*expr, "event.") {
		// Remove `event` data from the expression and replace with actual event
		// data as values, now that we have the event.
		//
		// This improves performance in matching, as we can then use the values within
		// aggregate trees.
		evt := event.Event{}
		if err := json.Unmarshal(i.events[0], &evt); err != nil {
			logger.StdlibLogger(ctx).Error("error unmarshalling trigger event in waitForEvent op", "error", err)
		}

		interpolated, err := expressions.Interpolate(ctx, *opts.If, map[string]any{
			"event": evt.Map(),
		})
		if err != nil {
			var compileError *expressions.CompileError
			if errors.As(err, &compileError) {
				return fmt.Errorf("error interpolating wait for event expression: %w", state.WrapInStandardError(
					compileError,
					"CompileError",
					"Could not compile expression",
					compileError.Message(),
				))
			}

			return fmt.Errorf("error interpolating wait for event expression: %w", err)
		}
		expr = &interpolated

		// Update the generator to use the interpolated data, ensuring history is updated.
		opts.If = expr
		gen.Opts = opts
	}

	opcode := gen.Op.String()
	sid := telemetry.NewSpanID(ctx)
	now := time.Now()

	// NOTE: the context here still contains the execSpan's traceID & spanID,
	// which is what we want because that's the parent that needs to be referenced later on
	carrier := telemetry.NewTraceCarrier(
		telemetry.WithTraceCarrierTimestamp(now),
		telemetry.WithTraceCarrierSpanID(&sid),
	)
	telemetry.UserTracer().Propagator().Inject(ctx, propagation.MapCarrier(carrier.Context))

	_, span := telemetry.NewSpan(ctx,
		telemetry.WithScope(consts.OtelScopeStep),
		telemetry.WithName(consts.OtelSpanWaitForEvent),
		telemetry.WithTimestamp(now),
		telemetry.WithSpanID(sid),
		telemetry.WithSpanAttributes(
			attribute.Bool(consts.OtelUserTraceFilterKey, true),
			attribute.String(consts.OtelSysStepOpcode, enums.OpcodeWaitForEvent.String()),
			attribute.String(consts.OtelSysAccountID, i.item.Identifier.AccountID.String()),
			attribute.String(consts.OtelSysWorkspaceID, i.item.Identifier.WorkspaceID.String()),
			attribute.String(consts.OtelSysAppID, i.item.Identifier.AppID.String()),
			attribute.String(consts.OtelSysFunctionID, i.item.Identifier.WorkflowID.String()),
			attribute.Int(consts.OtelSysFunctionVersion, i.item.Identifier.WorkflowVersion),
			attribute.String(consts.OtelAttrSDKRunID, i.item.Identifier.RunID.String()),
			attribute.Int(consts.OtelSysStepAttempt, 0),
			attribute.Int(consts.OtelSysStepMaxAttempt, 1),
			attribute.String(consts.OtelSysStepGroupID, i.item.GroupID),
			attribute.String(consts.OtelSysStepWaitEventName, opts.Event),
			attribute.Int64(consts.OtelSysStepWaitExpires, expires.UnixMilli()),
			attribute.String(consts.OtelSysStepDisplayName, gen.UserDefinedName()),
		),
	)
	defer span.End()

	if opts.If != nil {
		span.SetAttributes(attribute.String(consts.OtelSysStepWaitExpression, *opts.If))
	}

	err = e.pm.SavePause(ctx, state.Pause{
		ID:          pauseID,
		WorkspaceID: i.md.ID.Tenant.EnvID,
		Identifier:  i.item.Identifier,
		GroupID:     i.item.GroupID,
		Outgoing:    gen.ID,
		Incoming:    edge.Edge.Incoming,
		StepName:    gen.UserDefinedName(),
		Opcode:      &opcode,
		Expires:     state.Time(expires),
		Event:       &opts.Event,
		Expression:  expr,
		DataKey:     gen.ID,
		MaxAttempts: i.item.MaxAttempts,
		Metadata: map[string]any{
			consts.OtelPropagationKey: carrier,
		},
	})
	if err != nil {
		span.Cancel(ctx)
		if err == state.ErrPauseAlreadyExists {
			return nil
		}

		return err
	}

	// SDK-based event coordination is called both when an event is received
	// OR on timeout, depending on which happens first.  Both routes consume
	// the pause so this race will conclude by calling the function once, as only
	// one thread can lease and consume a pause;  the other will find that the
	// pause is no longer available and return.
	jobID := fmt.Sprintf("%s-%s-%s", i.md.IdempotencyKey(), gen.ID, "wait")
	// TODO Is this fine to leave? No attempts.
	err = e.queue.Enqueue(ctx, queue.Item{
		JobID:       &jobID,
		WorkspaceID: i.md.ID.Tenant.EnvID,
		// Use the same group ID, allowing us to track the cancellation of
		// the step correctly.
		GroupID:               i.item.GroupID,
		Kind:                  queue.KindPause,
		Identifier:            i.item.Identifier,
		PriorityFactor:        i.item.PriorityFactor,
		CustomConcurrencyKeys: i.item.CustomConcurrencyKeys,
		Payload: queue.PayloadPauseTimeout{
			PauseID:   pauseID,
			OnTimeout: true,
		},
	}, expires)
	if err == redis_state.ErrQueueItemExists {
		span.Cancel(ctx)
		return nil
	}

	for _, e := range e.lifecycles {
		go e.OnWaitForEvent(context.WithoutCancel(ctx), i.md, i.item, gen)
	}

	return err
}

func (e *executor) newExpressionEvaluator(ctx context.Context, expr string) (expressions.Evaluator, error) {
	if e.evalFactory != nil {
		return e.evalFactory(ctx, expr)
	}
	return expressions.NewExpressionEvaluator(ctx, expr)
}

// extractTraceCtx extracts the trace context from the given item, if it exists.
// If it doesn't it falls back to extracting the trace for the run overall.
// If neither exist or they are invalid, it returns the original context.
func (e *executor) extractTraceCtx(ctx context.Context, md sv2.Metadata, item *queue.Item) context.Context {
	fntrace := md.Config.FunctionTrace()
	if fntrace != nil {
		// NOTE:
		// this gymastics happens because the carrier stores the spanID separately.
		// it probably can be simplified
		tmp := telemetry.UserTracer().Propagator().Extract(ctx, propagation.MapCarrier(fntrace.Context))
		sctx := trace.SpanContextFromContext(tmp).WithSpanID(fntrace.SpanID())
		return trace.ContextWithSpanContext(ctx, sctx)
	}

	if item != nil {
		metadata := make(map[string]any)
		for k, v := range item.Metadata {
			metadata[k] = v
		}
		if newCtx, ok := extractTraceCtxFromMap(ctx, metadata); ok {
			return newCtx
		}
	}

	if md.Config.Context != nil {
		if newCtx, ok := extractTraceCtxFromMap(ctx, md.Config.Context); ok {
			return newCtx
		}
	}

	return ctx
}

// AppendAndScheduleBatch appends a new batch item. If a new batch is created, it will be scheduled to run
// after the batch timeout. If the item finalizes the batch, a function run is immediately scheduled.
func (e *executor) AppendAndScheduleBatch(ctx context.Context, fn inngest.Function, bi batch.BatchItem, opts *execution.BatchExecOpts) error {
	result, err := e.batcher.Append(ctx, bi, fn)
	if err != nil {
		return err
	}

	if opts == nil {
		opts = &execution.BatchExecOpts{}
	}

	switch result.Status {
	case enums.BatchAppend:
		// noop
	case enums.BatchNew:
		dur, err := time.ParseDuration(fn.EventBatch.Timeout)
		if err != nil {
			return err
		}
		at := time.Now().Add(dur)

		if err := e.batcher.ScheduleExecution(ctx, batch.ScheduleBatchOpts{
			ScheduleBatchPayload: batch.ScheduleBatchPayload{
				BatchID:         ulid.MustParse(result.BatchID),
				AccountID:       bi.AccountID,
				WorkspaceID:     bi.WorkspaceID,
				AppID:           bi.AppID,
				FunctionID:      bi.FunctionID,
				FunctionVersion: bi.FunctionVersion,
				BatchPointer:    result.BatchPointerKey,
			},
			At: at,
		}); err != nil {
			return err
		}
	case enums.BatchFull:
		// start execution immediately
		batchID := ulid.MustParse(result.BatchID)
		if err := e.RetrieveAndScheduleBatch(ctx, fn, batch.ScheduleBatchPayload{
			BatchID:         batchID,
			BatchPointer:    result.BatchPointerKey,
			AccountID:       bi.AccountID,
			WorkspaceID:     bi.WorkspaceID,
			AppID:           bi.AppID,
			FunctionID:      bi.FunctionID,
			FunctionVersion: bi.FunctionVersion,
		}, &execution.BatchExecOpts{
			FunctionPausedAt: opts.FunctionPausedAt,
		}); err != nil {
			return fmt.Errorf("could not retrieve and schedule batch items: %w", err)
		}
	default:
		return fmt.Errorf("invalid status of batch append ops: %d", result.Status)
	}

	return nil
}

// RetrieveAndScheduleBatch retrieves all items from a started batch and schedules a function run
func (e *executor) RetrieveAndScheduleBatch(ctx context.Context, fn inngest.Function, payload batch.ScheduleBatchPayload, opts *execution.BatchExecOpts) error {
	evtList, err := e.batcher.RetrieveItems(ctx, payload.FunctionID, payload.BatchID)
	if err != nil {
		return err
	}

	if opts == nil {
		opts = &execution.BatchExecOpts{}
	}

	evtIDs := make([]string, len(evtList))
	events := make([]event.TrackedEvent, len(evtList))
	for i, e := range evtList {
		events[i] = e
		evtIDs[i] = e.GetInternalID().String()
	}

	ctx, span := telemetry.NewSpan(ctx,
		telemetry.WithScope(consts.OtelScopeBatch),
		telemetry.WithName(consts.OtelSpanBatch),
		telemetry.WithNewRoot(),
		telemetry.WithSpanAttributes(
			attribute.Bool(consts.OtelUserTraceFilterKey, true),
			attribute.String(consts.OtelSysAccountID, payload.AccountID.String()),
			attribute.String(consts.OtelSysWorkspaceID, payload.WorkspaceID.String()),
			attribute.String(consts.OtelSysAppID, payload.AppID.String()),
			attribute.String(consts.OtelSysFunctionID, fn.ID.String()),
			attribute.String(consts.OtelSysBatchID, payload.BatchID.String()),
			attribute.String(consts.OtelSysEventIDs, strings.Join(evtIDs, ",")),
		))
	defer span.End()

	// still process events in case the user disables batching while a batch is still in-flight
	if fn.EventBatch != nil {
		if len(events) == fn.EventBatch.MaxSize {
			span.SetAttributes(attribute.Bool(consts.OtelSysBatchFull, true))
		} else {
			span.SetAttributes(attribute.Bool(consts.OtelSysBatchTimeout, true))
		}
	}

	key := fmt.Sprintf("%s-%s", fn.ID, payload.BatchID)
	md, err := e.Schedule(ctx, execution.ScheduleRequest{
		AccountID:        payload.AccountID,
		WorkspaceID:      payload.WorkspaceID,
		AppID:            payload.AppID,
		Function:         fn,
		Events:           events,
		BatchID:          &payload.BatchID,
		IdempotencyKey:   &key,
		FunctionPausedAt: opts.FunctionPausedAt,
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if md != nil {
		span.SetAttributes(attribute.String(consts.OtelAttrSDKRunID, md.ID.RunID.String()))
	} else {
		span.SetAttributes(attribute.Bool(consts.OtelSysStepDelete, true))
	}

	if err := e.batcher.ExpireKeys(ctx, payload.FunctionID, payload.BatchID); err != nil {
		return err
	}

	return nil
}

func (e *executor) validateStateSize(outputSize int, md sv2.Metadata) error {
	// validate state size and exit early if we're over the limit
	if e.stateSizeLimit != nil {
		stateSizeLimit := e.stateSizeLimit(md.ID)

		if stateSizeLimit == 0 {
			stateSizeLimit = consts.DefaultMaxStateSizeLimit
		}

		if outputSize+md.Metrics.StateSize > stateSizeLimit {
			return state.WrapInStandardError(
				state.ErrStateOverflowed,
				state.InngestErrStateOverflowed,
				fmt.Sprintf("The function run exceeded the state size limit of %d bytes.", stateSizeLimit),
				"",
			)
		}
	}

	return nil
}

// extractTraceCtxFromMap extracts the trace context from a map, if it exists.
// If it doesn't or it is invalid, it nil.
func extractTraceCtxFromMap(ctx context.Context, target map[string]any) (context.Context, bool) {
	if trace, ok := target[consts.OtelPropagationKey]; ok {
		carrier := telemetry.NewTraceCarrier()
		if err := carrier.Unmarshal(trace); err == nil {
			targetCtx := telemetry.UserTracer().Propagator().Extract(ctx, propagation.MapCarrier(carrier.Context))
			return targetCtx, true
		}
	}

	return ctx, false
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
