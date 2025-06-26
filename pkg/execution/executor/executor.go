package executor

import (
	"bytes"
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
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/cancellation"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/realtime"
	"github.com/inngest/inngest/pkg/execution/singleton"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/expressions/expragg"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/run"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/pkg/util/gateway"
	"github.com/oklog/ulid/v2"
	"github.com/xhit/go-str2duration/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

const (
	pkgName = "executor.execution.inngest"
)

var (
	ErrRuntimeRegistered          = fmt.Errorf("runtime is already registered")
	ErrNoStateManager             = fmt.Errorf("no state manager provided")
	ErrNoPauseManager             = fmt.Errorf("no pause manager provided")
	ErrNoActionLoader             = fmt.Errorf("no action loader provided")
	ErrNoRuntimeDriver            = fmt.Errorf("runtime driver for action not found")
	ErrFunctionDebounced          = fmt.Errorf("function debounced")
	ErrFunctionSkipped            = fmt.Errorf("function skipped")
	ErrFunctionSkippedIdempotency = fmt.Errorf("function skipped due to idempotency")

	ErrFunctionEnded = fmt.Errorf("function already ended")

	// ErrHandledStepError is returned when an OpcodeStepError is caught and the
	// step should be safely retried.
	ErrHandledStepError = fmt.Errorf("handled step error")

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

	if m.smv2 == nil {
		return nil, ErrNoStateManager
	}

	if m.pm == nil {
		return nil, ErrNoPauseManager
	}

	if m.httpClient == nil {
		// Default to the secure client.
		m.httpClient = exechttp.Client(exechttp.SecureDialerOpts{})
	}

	if m.tracerProvider == nil {
		m.tracerProvider = tracing.NewNoopTracerProvider()
	}

	return m, nil
}

// ExecutorOpt modifies the built-in executor on creation.
type ExecutorOpt func(m execution.Executor) error

func WithHTTPClient(c exechttp.RequestExecutor) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).httpClient = c
		return nil
	}
}

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
func WithPauseManager(pm pauses.Manager) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).pm = pm
		return nil
	}
}

// WithExpressionAggregator sets the expression aggregator singleton to use
// for matching events using our aggregate evaluator.
func WithExpressionAggregator(agg expragg.Aggregator) ExecutorOpt {
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

func WithLogger(l logger.Logger) ExecutorOpt {
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

func WithSingletonManager(sn singleton.Singleton) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).singletonMgr = sn
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

func WithAssignedQueueShard(shard redis_state.QueueShard) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).assignedQueueShard = shard
		return nil
	}
}

func WithShardSelector(selector redis_state.ShardSelector) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).shardFinder = selector
		return nil
	}
}

func WithTraceReader(m cqrs.TraceReader) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).traceReader = m
		return nil
	}
}

func WithTracerProvider(t tracing.TracerProvider) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).tracerProvider = t
		return nil
	}
}

// WithRealtimePublisher configures a new publisher in the executor.  This publishes
// directly to the backing implementaiton.
func WithRealtimePublisher(b realtime.Publisher) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).rtpub = b
		return nil
	}
}

// WithRealtimeAPIPublisher adds JWT configuration which allows publishing of data to the
// realtime API, without connecting to the backing realtime service directly.
func WithRealtimeConfig(config ExecutorRealtimeConfig) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).rtconfig = config
		return nil
	}
}

type ExecutorRealtimeConfig struct {
	Secret     []byte
	PublishURL string
}

// executor represents a built-in executor for running workflows.
type executor struct {
	log logger.Logger

	// exprAggregator is an expression aggregator used to parse and aggregate expressions
	// using trees.
	exprAggregator expragg.Aggregator

	pm   pauses.Manager
	smv2 sv2.RunService

	queue               queue.Queue
	debouncer           debounce.Debouncer
	batcher             batch.BatchManager
	singletonMgr        singleton.Singleton
	fl                  state.FunctionLoader
	evalFactory         func(ctx context.Context, expr string) (expressions.Evaluator, error)
	runtimeDrivers      map[string]driver.Driver
	finishHandler       execution.FinalizePublisher
	invokeFailHandler   execution.InvokeFailHandler
	handleSendingEvent  execution.HandleSendingEvent
	cancellationChecker cancellation.Checker
	httpClient          exechttp.RequestExecutor

	lifecycles []execution.LifecycleListener

	// rtpub represents teh realtime publisher used to broadcast notifications
	// on run execution.
	rtpub    realtime.Publisher
	rtconfig ExecutorRealtimeConfig

	// steplimit finds step limits for a given run.
	steplimit func(sv2.ID) int

	// stateSizeLimit finds state size limits for a given run
	stateSizeLimit func(sv2.ID) int

	preDeleteStateSizeReporter execution.PreDeleteStateSizeReporter

	assignedQueueShard redis_state.QueueShard
	shardFinder        redis_state.ShardSelector

	traceReader    cqrs.TraceReader
	tracerProvider tracing.TracerProvider
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
		e.log.Error("error closing lifecycle listeners", "error", err)
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
	if req.AppID == uuid.Nil {
		return nil, fmt.Errorf("app ID is required to schedule a run")
	}

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
	spanID := run.NewSpanID(ctx)

	config := *sv2.InitConfig(&sv2.Config{
		FunctionVersion: req.Function.FunctionVersion,
		SpanID:          spanID.String(),
		EventIDs:        eventIDs,
		Idempotency:     key,
		ReplayID:        req.ReplayID,
		OriginalRunID:   req.OriginalRunID,
		PriorityFactor:  &factor,
		BatchID:         req.BatchID,
		Context:         req.Context,
	})

	// Grab the cron schedule for function config.  This is necessary for fast
	// lookups, trace info, etc.
	if len(req.Events) == 1 && req.Events[0].GetEvent().Name == event.FnCronName {
		if cron, ok := req.Events[0].GetEvent().Data["cron"].(string); ok {
			config.SetCronSchedule(cron)
		}
	}

	// FunctionSlug is not stored in V1 format, so needs to be stored in Context
	config.SetFunctionSlug(req.Function.GetSlug())
	config.SetDebounceFlag(req.PreventDebounce)
	config.SetEventIDMapping(req.Events)

	carrier := itrace.NewTraceCarrier(itrace.WithTraceCarrierSpanID(&spanID))
	itrace.UserTracer().Propagator().Inject(ctx, propagation.MapCarrier(carrier.Context))
	config.SetFunctionTrace(carrier)

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
		Config: config,
	}

	// Always the root span.
	runSpanRef, err := e.tracerProvider.CreateSpan(
		meta.SpanNameRun,
		&tracing.CreateSpanOptions{
			Location: "executor.Schedule",
			Metadata: &metadata,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating run span: %w", err)
	}

	config.NewSetFunctionTrace(runSpanRef)

	// If this is paused, immediately end just before creating state.
	if skipped := req.SkipReason(); skipped != enums.SkipReasonNone {
		return e.handleFunctionSkipped(ctx, req, metadata, evts, skipped)
	}

	mapped := make([]map[string]any, len(req.Events))
	for n, item := range req.Events {
		mapped[n] = item.GetEvent().Map()
	}

	// Evaluate concurrency keys to use initially
	if req.Function.Concurrency != nil {
		metadata.Config.CustomConcurrencyKeys = queue.GetCustomConcurrencyKeys(ctx, metadata.ID, req.Function.Concurrency.Limits, evtMap)
	}

	//
	// Create throttle information prior to creating state.  This is used in the queue.
	//
	throttle := queue.GetThrottleConfig(ctx, req.Function.ID, req.Function.Throttle, evtMap)

	//
	// Create singleton information and try to handle it prior to creating state.
	//
	var singletonConfig *queue.Singleton
	data := req.Events[0].GetEvent().Map()

	if req.Function.Singleton != nil {
		singletonKey, err := singleton.SingletonKey(ctx, req.Function.ID, *req.Function.Singleton, data)
		switch {
		case err == nil:
			// Attempt to early handle function singletons when in skip mode. Function runs may still
			// fail to enqueue later when attempting to atomically acquire the function mutex.
			//
			// In cancel mode, this call releases the singleton mutex and atomically returns the
			// current run holding the lock, which will be cancelled further down. After releasing,
			// the lock becomes available to any competing run. If a faster run acquires it before
			// this one tries to, it will fail to acquire the lock and be skipped; Effectively
			// behaving as if the singleton mode were set to skip.
			singletonRunID, err := e.singletonMgr.HandleSingleton(ctx, singletonKey, *req.Function.Singleton, req.AccountID)
			if err != nil {
				return nil, err
			}

			eventID := req.Events[0].GetInternalID()

			if singletonRunID != nil {
				switch req.Function.Singleton.Mode {
				case enums.SingletonModeCancel:
					runID := sv2.ID{
						RunID:      *singletonRunID,
						FunctionID: req.Function.ID,
						Tenant: sv2.Tenant{
							AccountID: req.AccountID,
							EnvID:     req.WorkspaceID,
						},
					}
					err = e.Cancel(ctx, runID, execution.CancelRequest{
						EventID: &eventID,
					})
					logger.StdlibLogger(ctx).Error("error canceling singleton run", "error", err)
				default:
					// Immediately end before creating state
					return nil, ErrFunctionSkipped
				}
			}
			singletonConfig = &queue.Singleton{Key: singletonKey}
		case errors.Is(err, singleton.ErrEvaluatingSingletonExpression):
			// Ignore singleton expressions if we cannot evaluate them
			logger.StdlibLogger(ctx).Warn("error evaluating singleton expression", "error", err)
		case errors.Is(err, singleton.ErrNotASingleton):
			// We no-op, and we run the function normally not as a singleton
		default:
			return nil, err
		}
	}

	//
	// Create the run state.
	//

	newState := sv2.CreateState{
		Events:   evts,
		Metadata: metadata,
		Steps:    []state.MemoizedStep{},
	}

	if req.OriginalRunID != nil && req.FromStep != nil && req.FromStep.StepID != "" {
		if err := reconstruct(ctx, e.traceReader, req, &newState); err != nil {
			return nil, fmt.Errorf("error reconstructing input state: %w", err)
		}
	}

	st, err := e.smv2.Create(ctx, newState)
	switch err {
	case nil, state.ErrIdentifierExists: // no-op
	case state.ErrIdentifierTomestone:
		return nil, ErrFunctionSkippedIdempotency
	default:
		return nil, fmt.Errorf("error creating run state: %w", err)
	}
	if st == nil {
		return nil, fmt.Errorf("missing state after create: %w", err)
	}

	// NOTE: if the runID mismatches, it means there's already a state available
	// and we need to override the one we already have to make sure we're using
	// the correct metedata values
	if metadata.ID.RunID != st.Identifier().RunID {
		id := sv2.IDFromV1(st.Identifier())
		metadata, err = e.smv2.LoadMetadata(ctx, id)
		if err != nil {
			return nil, err
		}
	}

	//
	// Create cancellation pauses immediately, only if this is a non-batch event.
	//
	if req.BatchID == nil {
		for _, c := range req.Function.Cancel {
			expires := time.Now().Add(consts.CancelTimeout)
			if c.Timeout != nil {
				dur, err := str2duration.ParseDuration(*c.Timeout)
				if err != nil {
					return &metadata, fmt.Errorf("error parsing cancel duration: %w", err)
				}
				expires = time.Now().Add(dur)
			}

			// The triggering event ID should be the first ID in the batch.
			triggeringID := req.Events[0].GetInternalID().String()
			idSrc := fmt.Sprintf("%s-%s", key, c.Event)

			var expr *string
			// Evaluate the expression.  This lets us inspect the expression's attributes
			// so that we can store only the attrs used in the expression in the pause,
			// saving space, bandwidth, etc.
			if c.If != nil {

				// Remove `event` data from the expression and replace with actual event
				// data as values, now that we have the event.
				//
				// This improves performance in matching, as we can then use the values within
				// aggregate trees.
				interpolated, err := expressions.Interpolate(ctx, *c.If, map[string]any{
					"event": evtMap,
				})
				if err != nil {
					logger.StdlibLogger(ctx).Warn(
						"error interpolating cancellation expression",
						"error", err,
						"expression", expr,
					)
				}
				expr = &interpolated
				idSrc = fmt.Sprintf("%s-%s", idSrc, interpolated)
			}

			// NOTE: making this deterministic so pause creation is also idempotent
			pauseID := inngest.DeterministicSha1UUID(idSrc)
			pause := state.Pause{
				WorkspaceID:       st.Identifier().WorkspaceID,
				Identifier:        sv2.NewPauseIdentifier(metadata.ID),
				ID:                pauseID,
				Expires:           state.Time(expires),
				Event:             &c.Event,
				Expression:        expr,
				Cancel:            true,
				TriggeringEventID: &triggeringID,
			}
			_, err = e.pm.Write(ctx, pauses.Index{WorkspaceID: req.WorkspaceID, EventName: c.Event}, &pause)
			switch err {
			case nil, state.ErrPauseAlreadyExists: // no-op
			default:
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
	maxAttempts := consts.MaxRetries + 1
	item := queue.Item{
		JobID:                 &queueKey,
		GroupID:               uuid.New().String(),
		WorkspaceID:           st.Identifier().WorkspaceID,
		Kind:                  queue.KindStart,
		Identifier:            st.Identifier(),
		CustomConcurrencyKeys: metadata.Config.CustomConcurrencyKeys,
		PriorityFactor:        metadata.Config.PriorityFactor,
		Attempt:               0,
		MaxAttempts:           &maxAttempts,
		Payload: queue.PayloadEdge{
			Edge: inngest.SourceEdge,
		},
		Throttle:  throttle,
		Metadata:  map[string]any{},
		Singleton: singletonConfig,
	}

	err = e.queue.Enqueue(ctx, item, at, queue.EnqueueOpts{})

	switch err {
	case nil:
		// no-op
	case redis_state.ErrQueueItemExists:
		return nil, state.ErrIdentifierExists

	case redis_state.ErrQueueItemSingletonExists:
		_, err := e.smv2.Delete(ctx, sv2.IDFromV1(st.Identifier()))
		if err != nil {
			logger.StdlibLogger(ctx).Error(
				"error deleting function state",
				"error", err,
			)
		}
		return nil, ErrFunctionSkipped

	default:
		return nil, fmt.Errorf("error enqueueing source edge '%v': %w", queueKey, err)
	}

	for _, e := range e.lifecycles {
		go e.OnFunctionScheduled(context.WithoutCancel(ctx), metadata, item, req.Events)
	}

	return &metadata, nil
}

func (e *executor) handleFunctionSkipped(ctx context.Context, req execution.ScheduleRequest, metadata sv2.Metadata, evts []json.RawMessage, reason enums.SkipReason) (*sv2.Metadata, error) {
	for _, e := range e.lifecycles {
		go e.OnFunctionSkipped(context.WithoutCancel(ctx), metadata, execution.SkipState{
			CronSchedule: req.Events[0].GetEvent().CronSchedule(),
			Reason:       reason,
			Events:       evts,
		})
	}
	return nil, ErrFunctionSkipped
}

type runInstance struct {
	md         sv2.Metadata
	f          inngest.Function
	events     []json.RawMessage
	item       queue.Item
	edge       inngest.Edge
	resp       *state.DriverResponse
	httpClient exechttp.RequestExecutor
	stackIndex int
	// If specified, this is the span reference that represents this execution.
	execSpan *meta.SpanReference
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
	isSleepResume := item.Kind == queue.KindSleep && item.Attempt == 0
	if isSleepResume {
		err := e.tracerProvider.UpdateSpan(&tracing.UpdateSpanOptions{
			EndTime:    time.Now(),
			Location:   "executor.Execute",
			QueueItem:  &item,
			Status:     enums.StepStatusCompleted,
			TargetSpan: tracing.SpanRefFromQueueItem(&item),
		})
		if err != nil {
			return nil, fmt.Errorf("error updating sleep resume span: %w", err)
		}

		hasPendingSteps, err := e.smv2.SaveStep(ctx, sv2.ID{
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

		if hasPendingSteps {
			// Other steps are pending before we re-enter the function, so
			// we're now done with this execution.
			return nil, nil
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

	ef, err := e.fl.LoadFunction(ctx, md.ID.Tenant.EnvID, md.ID.FunctionID)
	if err != nil {
		return nil, fmt.Errorf("error loading function for run: %w", err)
	}
	if ef.Paused {
		return nil, state.ErrFunctionPaused
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

	// Validate that the run can execute.
	v := newRunValidator(e, ef.Function, md, events, item) // TODO: Load events for this.
	if err := v.validate(ctx); err != nil {
		return nil, err
	}

	//
	// record function start time using the same method as step started,
	// ensures ui timeline alignment
	start, ok := redis_state.GetItemStart(ctx)
	if !ok {
		start = time.Now()
	}

	if md.Config.StartedAt.IsZero() {
		md.Config.StartedAt = start
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

	evtIDs := make([]string, len(id.EventIDs))
	for i, eid := range id.EventIDs {
		evtIDs[i] = eid.String()
	}

	// TODO: find a way to remove this
	// set function trace context so downstream execution have the function
	// trace context set
	ctx = extractTraceCtx(ctx, md)

	isFirstExecution := edge.Incoming == inngest.TriggerName && item.Attempt == 0

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
		if len(ef.Function.Steps) > 1 {
			return nil, fmt.Errorf("DAG-based steps are no longer supported")
		}

		edge.Outgoing = inngest.TriggerName
		edge.Incoming = ef.Function.Steps[0].ID
		// Update the payload
		payload := item.Payload.(queue.PayloadEdge)
		payload.Edge = edge
		item.Payload = payload
		// Add retries from the step to our queue item.  Increase as retries is
		// always one less than attempts.
		retries := ef.Function.Steps[0].RetryCount() + 1
		item.MaxAttempts = &retries

		// Only just starting:  run lifecycles on first attempt.
		if item.Attempt == 0 {
			// Set the start time and spanID in metadata for subsequent runs
			// This should be an one time operation and is never updated after,
			// which is enforced on the Lua script.
			if err := e.smv2.UpdateMetadata(ctx, md.ID, sv2.MutableConfig{
				StartedAt:      md.Config.StartedAt,
				ForceStepPlan:  md.Config.ForceStepPlan,
				RequestVersion: md.Config.RequestVersion,
			}); err != nil {
				e.log.Error("error updating metadata on function start", "error", err)
			}

			for _, e := range e.lifecycles {
				go e.OnFunctionStarted(context.WithoutCancel(ctx), md, item, events)
			}
		}
	}

	instance := runInstance{
		md:         md,
		f:          *ef.Function,
		events:     events,
		item:       item,
		edge:       edge,
		stackIndex: stackIndex,
		httpClient: e.httpClient,
	}

	// Set the parent span for this execution.
	var execParent *meta.SpanReference
	if isFirstExecution {
		// If this is the first ever attempt, we haven't created a step yet. If
		// this is not the first attempt, the step span is created when it is
		// enqueued, so we don't need to create one here.
		execParent, err = e.tracerProvider.CreateSpan(
			meta.SpanNameStepDiscovery,
			&tracing.CreateSpanOptions{
				Location:  "executor.Execute",
				Parent:    tracing.RunSpanRefFromMetadata(&md),
				Metadata:  &md,
				QueueItem: &item,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error creating initial step span: %w", err)
		}

	} else if isSleepResume {
		// If we're resuming a sleep here, we're also starting a new discovery
		// step here.
		execParent, err = e.tracerProvider.CreateSpan(
			meta.SpanNameStepDiscovery,
			&tracing.CreateSpanOptions{
				FollowsFrom: tracing.SpanRefFromQueueItem(&item),
				Location:    "executor.Execute",
				Metadata:    &md,
				Parent:      tracing.RunSpanRefFromMetadata(&md),
				QueueItem:   &item,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error creating discovery step span after sleep resume: %w", err)
		}
	} else {
		// If we're here, we assume that the step span has already been
		// created, so add it here.
		execParent = tracing.SpanRefFromQueueItem(&item)
	}

	instance.execSpan, err = e.tracerProvider.CreateSpan(
		meta.SpanNameExecution,
		&tracing.CreateSpanOptions{
			Location:  "executor.Execute",
			Parent:    execParent,
			Metadata:  &md,
			QueueItem: &item,
			SpanOptions: []trace.SpanStartOption{
				tracing.WithFunctionAttrs(&instance.f),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating execution span: %w", err)
	}

	return util.CritT(ctx, "run step", func(ctx context.Context) (*state.DriverResponse, error) {
		resp, err := e.run(ctx, &instance)

		status := enums.StepStatusCompleted
		if err != nil || resp.Err != nil || resp.UserError != nil {
			status = enums.StepStatusFailed
		}
		_ = e.tracerProvider.UpdateSpan(
			&tracing.UpdateSpanOptions{
				EndTime:    time.Now(),
				Location:   "executor.Execute",
				Metadata:   &md,
				QueueItem:  &item,
				TargetSpan: instance.execSpan,
				SpanOptions: []trace.SpanStartOption{
					tracing.WithDriverResponseAttrs(resp, nil),
				},
				Status: status,
			},
		)

		// Now we have a response, update the run instance.  We need to do this as request
		// offloads must mutate the response directly.
		instance.resp = resp
		if resp == nil && err != nil {
			for _, e := range e.lifecycles {
				// OnStepFinished handles step success and step errors/failures.  It is
				// currently the responsibility of the lifecycle manager to handle the differing
				// step statuses when a step finishes.
				go e.OnStepFinished(context.WithoutCancel(ctx), md, item, edge, resp, err)
			}
			return nil, err
		}

		if handleErr := e.HandleResponse(ctx, &instance); handleErr != nil {
			return resp, handleErr
		}
		return resp, err
	},
		// wait up to 2h and add a short delay to allow driver implementations to
		// return a specific timeout error here
		util.WithTimeout(consts.MaxFunctionTimeout+5*time.Second),
	)
}

func (e *executor) HandleResponse(ctx context.Context, i *runInstance) error {
	l := logger.StdlibLogger(ctx).With(
		"run_id", i.md.ID.RunID.String(),
		"workflow_id", i.md.ID.FunctionID.String(),
	)

	for _, e := range e.lifecycles {
		go e.OnStepFinished(context.WithoutCancel(ctx), i.md, i.item, i.edge, i.resp, nil)
	}

	if i.resp.Err == nil && !i.resp.IsFunctionResult() {
		// Handle generator responses then return.
		if serr := e.HandleGeneratorResponse(ctx, i, i.resp); serr != nil {
			// If this is an error compiling async expressions, fail the function.
			shouldFailEarly := errors.Is(serr, &expressions.CompileError{}) || errors.Is(serr, state.ErrStateOverflowed) || errors.Is(serr, state.ErrFunctionOverflowed) || errors.Is(serr, state.ErrSignalConflict)

			if shouldFailEarly {
				var gracefulErr *state.WrappedStandardError
				if hasGracefulErr := errors.As(serr, &gracefulErr); hasGracefulErr {
					serialized := gracefulErr.Serialize(execution.StateErrorKey)
					i.resp.Output = serialized
					i.resp.Err = &gracefulErr.StandardError.Name

					// Immediately fail the function.
					i.resp.NoRetry = true

					// This is required to get old history to look correct.
					// Without it, the function run will have no output. We can
					// probably delete this when we fully remove old history.
					i.resp.Generator = []*state.GeneratorOpcode{}
				}

				metrics.IncrRunFinalizedCounter(ctx, metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"reason": "fail-early",
					},
				})

				if err := e.finalize(ctx, i.md, i.events, i.f.GetSlug(), e.assignedQueueShard, *i.resp, i.execSpan); err != nil {
					l.Error("error running finish handler", "error", err)
				}

				// Can be reached multiple times for parallel discovery steps
				for _, e := range e.lifecycles {
					go e.OnFunctionFinished(context.WithoutCancel(ctx), i.md, i.item, i.events, *i.resp)
				}

				return nil
			}

			return fmt.Errorf("error handling generator response: %w", serr)
		}
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
	if i.resp.Err != nil {
		if i.resp.Retryable() {
			// Retries are a native aspect of the queue;  returning errors always
			// retries steps if possible.
			for _, e := range e.lifecycles {
				// Run the lifecycle method for this retry, which is baked into the queue.
				i.item.Attempt += 1
				go e.OnStepScheduled(context.WithoutCancel(ctx), i.md, i.item, &i.resp.Step.Name)
			}
			return nil
		}

		// If i.resp.Err != nil, we don't know whether to invoke the fn again
		// with per-step errors, as we don't know if the intent behind this queue item
		// is a step.
		//
		// In this case, for non-retryable errors, we ignore and fail the function;
		// only OpcodeStepError causes try/catch to be handled and us to continue
		// on error.

		metrics.IncrRunFinalizedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"reason": "resp-err",
			},
		})

		// TODO: Refactor state input
		if err := e.finalize(ctx, i.md, i.events, i.f.GetSlug(), e.assignedQueueShard, *i.resp, i.execSpan); err != nil {
			l.Error("error running finish handler", "error", err)
		}

		// Can be reached multiple times for parallel discovery steps
		for _, e := range e.lifecycles {
			go e.OnFunctionFinished(context.WithoutCancel(ctx), i.md, i.item, i.events, *i.resp)
		}

		return nil
	}

	// The generator length check is necessary because parallel steps in older
	// SDK versions (e.g. 2.7.2) can result in an OpcodeNone.
	if len(i.resp.Generator) == 0 && i.resp.IsFunctionResult() {
		metrics.IncrRunFinalizedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"reason": "opcode-none",
			},
		})

		// This is the function result.
		if err := e.finalize(ctx, i.md, i.events, i.f.GetSlug(), e.assignedQueueShard, *i.resp, i.execSpan); err != nil {
			l.Error("error running finish handler", "error", err)
		}

		// Can be reached multiple times for parallel discovery steps
		for _, e := range e.lifecycles {
			go e.OnFunctionFinished(context.WithoutCancel(ctx), i.md, i.item, i.events, *i.resp)
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
func (e *executor) finalize(ctx context.Context, md sv2.Metadata, evts []json.RawMessage, fnSlug string, queueShard redis_state.QueueShard, resp state.DriverResponse, outputSpanRef *meta.SpanReference) error {
	ctx = context.WithoutCancel(ctx)

	runStatus := enums.StepStatusCompleted
	if resp.Error() != "" {
		runStatus = enums.StepStatusFailed
	}

	err := e.tracerProvider.UpdateSpan(&tracing.UpdateSpanOptions{
		EndTime:    time.Now(),
		Location:   "executor.finalize",
		Metadata:   &md,
		TargetSpan: tracing.RunSpanRefFromMetadata(&md),
		Status:     runStatus,
		SpanOptions: []trace.SpanStartOption{
			tracing.WithDriverResponseAttrs(&resp, outputSpanRef),
		},
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"error updating run span end time",
			"error", err,
			"run_id", md.ID.RunID.String(),
		)
	}

	// Parse events for the fail handler before deleting state.
	inputEvents := make([]event.Event, len(evts))
	for n, e := range evts {
		evt, err := event.NewEvent(e)
		if err != nil {
			return err
		}
		inputEvents[n] = *evt
	}

	if e.preDeleteStateSizeReporter != nil {
		e.preDeleteStateSizeReporter(ctx, md)
	}

	// Delete the function state in every case.
	_, err = e.smv2.Delete(ctx, md.ID)
	if err != nil {
		logger.StdlibLogger(ctx).Error(
			"error deleting state in finalize",
			"error", err,
			"run_id", md.ID.RunID.String(),
		)
	}

	// We may be cancelling an in-progress run.  If that's the case, we want to delete any
	// outstanding jobs from the queue, if possible.
	//
	// XXX: Remove this typecast and normalize queue interface to a single package
	q, ok := e.queue.(redis_state.QueueManager)
	if ok {
		// Find all items for the current function run.
		jobs, err := q.RunJobs(
			ctx,
			queueShard.Name,
			md.ID.Tenant.EnvID,
			md.ID.FunctionID,
			md.ID.RunID,
			1000,
			0,
		)
		if err != nil {
			logger.StdlibLogger(ctx).Error(
				"error fetching run jobs",
				"error", err,
				"run_id", md.ID.RunID.String(),
			)
		}

		for _, j := range jobs {
			qi, _ := j.Raw.(*queue.QueueItem)
			if qi == nil {
				continue
			}

			jobID := queue.JobIDFromContext(ctx)
			if jobID != "" && qi.ID == jobID {
				// Do not dequeue the current job that we're working on.
				continue
			}

			err := q.Dequeue(ctx, queueShard, *qi)
			if err != nil && !errors.Is(err, redis_state.ErrQueueItemNotFound) {
				logger.StdlibLogger(ctx).Error(
					"error dequeueing run job",
					"error", err,
					"run_id", md.ID.RunID.String(),
				)
			}
		}
	}

	if e.finishHandler == nil {
		return nil
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

		if resp.Err != nil {
			// Legacy - send inngest/function.failed, except for when the function has been cancelled.
			if !strings.Contains(*resp.Err, state.ErrFunctionCancelled.Error()) {
				freshEvents = append(freshEvents, event.Event{
					ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
					Name:      event.FnFailedName,
					Timestamp: now.UnixMilli(),
					Data:      data,
				})
			}

			// Add function cancelled event
			if *resp.Err == state.ErrFunctionCancelled.Error() {
				freshEvents = append(freshEvents, event.Event{
					ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
					Name:      event.FnCancelledName,
					Timestamp: now.UnixMilli(),
					Data:      data,
				})
			}
		}
	}

	return e.finishHandler(ctx, md.ID, freshEvents)
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
		var serr syscode.Error
		if errors.As(err, &serr) {
			gracefulErr := state.StandardError{
				Error:   fmt.Sprintf("%s: %s", serr.Code, serr.Message),
				Name:    serr.Code,
				Message: serr.Message,
			}.Serialize(execution.StateErrorKey)
			response.Output = gracefulErr
			response.Err = &serr.Code
		} else {
			// Set the response error if it wasn't set, or if Execute had an internal error.
			// This ensures that we only ever need to check resp.Err to handle errors.
			byt, e := json.Marshal(err.Error())
			if e != nil {
				response.Output = err
			} else {
				response.Output = string(byt)
			}

			errstr := err.Error()
			response.Err = &errstr
		}
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
func (e *executor) HandlePauses(ctx context.Context, evt event.TrackedEvent) (execution.HandlePauseResult, error) {
	idx := pauses.Index{
		WorkspaceID: evt.GetWorkspaceID(),
		EventName:   evt.GetEvent().Name,
	}

	aggregated, err := e.pm.Aggregated(
		ctx,
		idx,
		consts.AggregatePauseThreshold,
	)
	if err != nil {
		e.log.Error("error checking pause aggregation", "error", err)
	}

	// Use the aggregator for all funciton finished events, if there are more than
	// 50 waiting.  It only takes a few milliseconds to iterate and handle less
	// than 50;  anything more runs the risk of running slow.
	if aggregated {
		aggRes, err := e.handleAggregatePauses(ctx, evt)
		if err != nil {
			e.log.Error("error handling aggregate pauses", "error", err)
		}
		return aggRes, err
	}

	iter, err := e.pm.PausesSince(ctx, idx, time.Time{})
	if err != nil {
		return execution.HandlePauseResult{}, fmt.Errorf("error loading pause iterator: %w", err)
	}

	res, err := e.handlePausesAllNaively(ctx, iter, evt)
	if err != nil {
		e.log.Error("error handling naive pauses", "error", err)
	}
	return res, nil
}

//nolint:all
func (e *executor) handlePausesAllNaively(ctx context.Context, iter state.PauseIterator, evt event.TrackedEvent) (execution.HandlePauseResult, error) {
	res := execution.HandlePauseResult{0, 0}

	if e.queue == nil || e.smv2 == nil || e.pm == nil {
		return res, fmt.Errorf("no queue or state manager specified")
	}

	log := e.log.With("event_id", evt.GetInternalID().String())

	var (
		goerr error
		wg    sync.WaitGroup
	)

	evtID := evt.GetInternalID()

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

			l := log.With(
				"pause_id", pause.ID.String(),
				"run_id", pause.Identifier.RunID.String(),
				"workflow_id", pause.Identifier.FunctionID.String(),
				"expires", pause.Expires.String(),
				"strategy", "naive",
			)

			// If this is a cancellation, ensure that we're not handling an event that
			// was received before the run (due to eg. latency in a bad case).
			//
			// NOTE: Fast path this before handling the expression.
			if pause.Cancel && bytes.Compare(evtID[:], pause.Identifier.RunID[:]) <= 0 {
				return
			}

			// Run an expression if this exists.
			if pause.Expression != nil {
				// Precompute the expression data once, as a value (not pointer)
				data := expressions.NewData(map[string]any{
					"async": evt.GetEvent().Map(),
				})

				expr, err := expressions.NewExpressionEvaluator(ctx, *pause.Expression)
				if err != nil {
					l.Error("error compiling pause expression", "error", err)
					return
				}

				val, _, err := expr.Evaluate(ctx, data)
				if err != nil {
					l.Warn("error evaluating pause expression", "error", err)
					return
				}
				result, _ := val.(bool)
				if !result {
					return
				}
			}

			if err := e.handlePause(ctx, evt, evtID, pause, &res, l); err != nil {
				goerr = errors.Join(goerr, err)
				l.Error("error handling pause", "error", err, "pause", pause)
			}
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

	log := e.log.With(
		"event_id", evt.GetInternalID().String(),
		"workspace_id", evt.GetWorkspaceID(),
		"event", evt.GetEvent().Name,
	)

	evtID := evt.GetInternalID()
	evals, count, err := e.exprAggregator.EvaluateAsyncEvent(ctx, evt)
	if err != nil {
		log.Error("error evaluating async event", "error", err)
	}

	// We only want to return an error if we have no evaluations. Since we
	// evaluate multiple expressions, a returned error means that at least one
	// expression errored -- not that all expressions errored.
	if err != nil && len(evals) == 0 {
		return execution.HandlePauseResult{count, 0}, err
	}

	var (
		goerr error
		wg    sync.WaitGroup
	)

	for _, i := range evals {
		// Copy pause into function
		pause := *i
		wg.Add(1)
		go func() {
			atomic.AddInt32(&res[0], 1)

			defer wg.Done()

			l := log.With(
				"pause_id", pause.ID.String(),
				"run_id", pause.Identifier.RunID.String(),
				"workflow_id", pause.Identifier.FunctionID.String(),
				"expires", pause.Expires.String(),
			)

			if err := e.handlePause(ctx, evt, evtID, &pause, &res, l); err != nil {
				goerr = errors.Join(goerr, err)
				l.Error("error handling pause", "error", err, "pause", pause)
			}
		}()
	}
	wg.Wait()

	return res, goerr
}

func (e *executor) handlePause(
	ctx context.Context,
	evt event.TrackedEvent,
	evtID ulid.ULID,
	pause *state.Pause,
	res *execution.HandlePauseResult,
	l logger.Logger,
) error {
	// If this is a cancellation, ensure that we're not handling an event that
	// was received before the run (due to eg. latency in a bad case).
	if pause.Cancel && bytes.Compare(evtID[:], pause.Identifier.RunID[:]) <= 0 {
		return nil
	}

	return util.Crit(ctx, "handle pause", func(ctx context.Context) error {
		cleanup := func(ctx context.Context) {
			eg := errgroup.Group{}
			eg.Go(func() error {
				return e.pm.Delete(
					context.Background(),
					pauses.Index{WorkspaceID: pause.WorkspaceID, EventName: evt.GetEvent().Name},
					*pause,
				)
			})
			eg.Go(func() error {
				return e.exprAggregator.RemovePause(ctx, pause)
			})
			_ = eg.Wait()
		}

		// NOTE: Some pauses may be nil or expired, as the iterator may take
		// time to process.  We handle that here and assume that the event
		// did not occur in time.
		if pause.Expires.Time().Before(time.Now()) {
			l.Debug("encountered expired pause")

			shouldDelete := pause.Expires.Time().Add(consts.PauseExpiredDeletionGracePeriod).Before(time.Now())
			if shouldDelete {
				// Consume this pause to remove it entirely
				l.Debug("deleting expired pause")

				cleanup(ctx)
			}
			return nil
		}

		// NOTE: Make sure the event that created the pause isn't also the one resuming it
		if pause.TriggeringEventID != nil && *pause.TriggeringEventID == evtID.String() {
			return nil
		}

		// Ensure that we store the group ID for this pause, letting us properly track cancellation
		// or continuation history
		ctx = state.WithGroupID(ctx, pause.GroupID)

		if pause.Cancel {
			// This is a cancellation signal.  Check if the function
			// has ended, and if so remove the pause.
			//
			// NOTE: Bookkeeping must be added to individual function runs and handled on
			// completion instead of here.  This is a hot path and should only exist whilst
			// bookkeeping is not implemented.
			if exists, err := e.smv2.Exists(ctx, sv2.IDFromPause(*pause)); !exists && err == nil {
				// This function has ended.  Delete the pause and continue
				cleanup(ctx)
				return nil
			}

			// Cancelling a function can happen before a lease, as it's an atomic operation that will always happen.
			err := e.Cancel(ctx, sv2.IDFromPause(*pause), execution.CancelRequest{
				EventID:    &evtID,
				Expression: pause.Expression,
			})
			if errors.Is(err, state.ErrFunctionCancelled) ||
				errors.Is(err, state.ErrFunctionComplete) ||
				errors.Is(err, state.ErrFunctionFailed) ||
				errors.Is(err, ErrFunctionEnded) {
				// Safe to ignore.
				cleanup(ctx)
				return nil
			}
			if err != nil && strings.Contains(err.Error(), "no status stored in metadata") {
				// Safe to ignore.
				cleanup(ctx)
				return nil
			}

			if err != nil {
				return fmt.Errorf("error cancelling function: %w", err)
			}

			// Ensure we consume this pause, as this isn't handled by the higher-level cancel function.
			// NOTE: cleanup closure is ignored here since there's already another one that will be called
			_, _, err = e.pm.ConsumePause(context.Background(), *pause, state.ConsumePauseOpts{
				IdempotencyKey: evtID.String(),
				Data:           nil,
			})
			if err == nil || err == state.ErrPauseLeased || err == state.ErrPauseNotFound {
				atomic.AddInt32(&res[1], 1)
				cleanup(ctx)
				return nil
			}
			return fmt.Errorf("error consuming pause after cancel: %w", err)
		}

		resumeData := pause.GetResumeData(evt.GetEvent())

		err := e.Resume(ctx, *pause, execution.ResumeRequest{
			With:           resumeData.With,
			EventID:        &evtID,
			EventName:      evt.GetEvent().Name,
			RunID:          resumeData.RunID,
			StepName:       resumeData.StepName,
			IdempotencyKey: evtID.String(),
		})
		if errors.Is(err, state.ErrPauseLeased) ||
			errors.Is(err, state.ErrPauseNotFound) ||
			errors.Is(err, state.ErrRunNotFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("error resuming pause: %w", err)
		}

		// Add to the counter.
		atomic.AddInt32(&res[1], 1)
		if err := e.exprAggregator.RemovePause(ctx, pause); err != nil {
			l.Warn("error removing pause from aggregator", "error", err)
		}
		return nil
	})
}

func (e *executor) HandleInvokeFinish(ctx context.Context, evt event.TrackedEvent) error {
	evtID := evt.GetInternalID()
	l := e.log.With("event_id", evtID.String())

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

	var eventName string
	if pause.Event != nil {
		eventName = *pause.Event
	}

	if pause.Expires.Time().Before(time.Now()) {
		l.Debug("encountered expired pause")

		shouldDelete := pause.Expires.Time().Add(consts.PauseExpiredDeletionGracePeriod).Before(time.Now())
		if shouldDelete {
			// Consume this pause to remove it entirely
			l.Debug("deleting expired pause")
			_ = e.pm.Delete(context.Background(), pauses.Index{WorkspaceID: pause.WorkspaceID, EventName: eventName}, *pause)
		}

		return nil
	}

	if pause.Cancel {
		// This is a cancellation signal.  Check if the function
		// has ended, and if so remove the pause.
		//
		// NOTE: Bookkeeping must be added to individual function runs and handled on
		// completion instead of here.  This is a hot path and should only exist whilst
		// bookkeeping is not implemented.
		if exists, err := e.smv2.Exists(ctx, sv2.IDFromPause(*pause)); !exists && err == nil {
			// This function has ended.  Delete the pause and continue
			_ = e.pm.Delete(context.Background(), pauses.Index{WorkspaceID: pause.WorkspaceID, EventName: eventName}, *pause)
			return nil
		}
	}

	resumeData := pause.GetResumeData(evt.GetEvent())
	l.Debug("resuming pause from invoke", "pause.DataKey", pause.DataKey)

	return e.Resume(ctx, *pause, execution.ResumeRequest{
		With:           resumeData.With,
		EventID:        &evtID,
		EventName:      evt.GetEvent().Name,
		RunID:          resumeData.RunID,
		StepName:       resumeData.StepName,
		IdempotencyKey: evtID.String(),
	})
}

// Cancel cancels an in-progress function.
func (e *executor) Cancel(ctx context.Context, id sv2.ID, r execution.CancelRequest) error {
	l := e.log.With(
		"run_id", id.RunID.String(),
		"workflow_id", id.FunctionID.String(),
	)

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

	shard, err := e.shardFinder(ctx, md.ID.Tenant.AccountID, nil)
	if err != nil {
		return fmt.Errorf("could not find shard for account %q: %w", md.ID.Tenant, err)
	}

	metrics.IncrRunFinalizedCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"reason": "cancel",
		},
	})

	fnCancelledErr := state.ErrFunctionCancelled.Error()
	// TODO Is the execspan set here? Does it matter?
	if err := e.finalize(ctx, md, evts, f.Function.GetSlug(), shard, state.DriverResponse{
		Err: &fnCancelledErr,
	}, nil); err != nil {
		l.Error("error running finish handler", "error", err)
	}
	for _, e := range e.lifecycles {
		go e.OnFunctionCancelled(context.WithoutCancel(ctx), md, r, evts)
	}

	return nil
}

// ResumePauseTimeout times out a step.  This is used to reusme a pause from timeout when:
//
// - A waitForEvent step doesn't receive its event before the timeout
// - A waitForSignal step doesn't receive its signal before the timeout
// - An invoked function doesnt finish before the timeout
//
// Resume can also resume as a timeout.  This is a separate method so that we can resume
// the timeout without loading and leasing pauses, relying on state store atomicity to instead
// resume and cancel a pause.
func (e *executor) ResumePauseTimeout(ctx context.Context, pause state.Pause, r execution.ResumeRequest) error {
	// (tonyhb): this could be refactored to not require a pause, and instead only require the fields
	// necessary for timeouts.  This will save space in the queue.  This requires a refactor of the
	// trace lifecycles, whihc also require pauses.
	id := sv2.IDFromPause(pause)
	md, err := e.smv2.LoadMetadata(ctx, id)
	if err == state.ErrRunNotFound {
		return err
	}
	if err != nil {
		return fmt.Errorf("error loading metadata to resume from pause: %w", err)
	}

	data, err := json.Marshal(r.With)
	if err != nil {
		return fmt.Errorf("error marshalling timeout step data: %w", err)
	}

	e.log.Debug("resuming from timeout ", "identifier", id)

	pending, err := e.smv2.SaveStep(ctx, id, pause.DataKey, data)
	if errors.Is(err, state.ErrDuplicateResponse) {
		// cannot resume as the pause has already been resumed and consumed.
		return nil
	}
	if err != nil && !errors.Is(err, state.ErrIdempotentResponse) {
		// This is a non-idempotent error, so there was a legitimate error saving the response.
		e.log.Error("error saving timeout step", "error", err, "identifier", id)
		return err
	}

	if !pending {
		// If there are no parallel steps ongoing, we must enqueue the next SDK ping to continue on with
		// execution.
		jobID := fmt.Sprintf("%s-%s-timeout", md.IdempotencyKey(), pause.DataKey)

		err = e.queue.Enqueue(ctx, queue.Item{
			JobID: &jobID,
			// Add a new group ID for the child;  this will be a new step.
			GroupID:               uuid.New().String(),
			WorkspaceID:           id.Tenant.EnvID,
			Kind:                  queue.KindEdge,
			Identifier:            sv2.V1FromMetadata(md),
			PriorityFactor:        md.Config.PriorityFactor,
			CustomConcurrencyKeys: md.Config.CustomConcurrencyKeys,
			MaxAttempts:           pause.MaxAttempts,
			Payload: queue.PayloadEdge{
				Edge: inngest.Edge{
					Outgoing: pause.DataKey,
					Incoming: "step",
				},
			},
		}, time.Now(), queue.EnqueueOpts{})
		if err != nil && err != redis_state.ErrQueueItemExists {
			return fmt.Errorf("error enqueueing after pause: %w", err)
		}
	}

	// Only run lifecycles if we consumed the pause and enqueued next step.
	switch pause.GetOpcode() {
	case enums.OpcodeInvokeFunction:
		for _, e := range e.lifecycles {
			go e.OnInvokeFunctionResumed(context.WithoutCancel(ctx), md, pause, r)
		}
	case enums.OpcodeWaitForSignal:
		for _, e := range e.lifecycles {
			go e.OnWaitForSignalResumed(context.WithoutCancel(ctx), md, pause, r)
		}
	case enums.OpcodeWaitForEvent:
		for _, e := range e.lifecycles {
			go e.OnWaitForEventResumed(context.WithoutCancel(ctx), md, pause, r)
		}
	}

	// And delete the OG pause.
	if err := e.pm.Delete(ctx, pauses.PauseIndex(pause), pause); err != nil {
		return fmt.Errorf("deleting pause by ID: %w", err)
	}

	return nil
}

// Resume resumes an in-progress function from the given pause.
func (e *executor) Resume(ctx context.Context, pause state.Pause, r execution.ResumeRequest) error {
	if e.queue == nil || e.smv2 == nil || e.pm == nil {
		return fmt.Errorf("no queue or state manager specified")
	}

	md, err := e.smv2.LoadMetadata(ctx, sv2.ID{
		RunID:      pause.Identifier.RunID,
		FunctionID: pause.Identifier.FunctionID,
		Tenant: sv2.Tenant{
			EnvID:     pause.WorkspaceID,
			AccountID: pause.Identifier.AccountID,
			// NOTE: Pauses do not store app IDs.
		},
	})
	if err == state.ErrRunNotFound {
		return err
	}
	if err != nil {
		return fmt.Errorf("error loading metadata to resume from pause: %w", err)
	}

	err = util.Crit(ctx, "consume pause", func(ctx context.Context) error {
		if pause.OnTimeout && r.EventID != nil {
			// Delete this pause, as an event has occured which matches
			// the timeout.  We can do this prior to leasing a pause as it's the
			// only work that needs to happen
			_, cleanup, err := e.pm.ConsumePause(ctx, pause, state.ConsumePauseOpts{
				IdempotencyKey: r.IdempotencyKey,
				Data:           nil,
			})
			switch err {
			case nil, state.ErrPauseNotFound: // no-op
			default:
				return fmt.Errorf("error consuming pause via timeout: %w", err)
			}

			return cleanup()
		}

		consumeResult, cleanup, err := e.pm.ConsumePause(ctx, pause, state.ConsumePauseOpts{
			IdempotencyKey: r.IdempotencyKey,
			Data:           r.With,
		})
		if err != nil {
			return fmt.Errorf("error consuming pause via event: %w", err)
		}

		e.log.Debug("resuming from pause",
			"error", err,
			"pause_id", pause.ID.String(),
			"run_id", pause.Identifier.RunID.String(),
			"workflow_id", pause.Identifier.FunctionID.String(),
			"timeout", pause.OnTimeout,
			"cancel", pause.Cancel,
			"consumed", consumeResult,
		)

		if !consumeResult.DidConsume {
			// We don't need to do anything here.  This could be a dupe;  consuming a pause
			// is transactional / atomic, so ignore this.
			return nil
		}

		status := enums.StepStatusCompleted
		if r.IsTimeout {
			status = enums.StepStatusTimedOut
		}
		pauseSpan := tracing.SpanRefFromPause(&pause)
		_ = e.tracerProvider.UpdateSpan(&tracing.UpdateSpanOptions{
			EndTime:    time.Now(),
			Location:   "executor.Resume",
			Status:     status,
			TargetSpan: pauseSpan,
			SpanOptions: []trace.SpanStartOption{
				tracing.WithResumeAttrs(&pause, &r),
			},
		})

		if !consumeResult.HasPendingSteps {
			// Schedule an execution from the pause's entrypoint.  We do this
			// after consuming the pause to guarantee the event data is
			// stored via the pause for the next run.  If the ConsumePause
			// call comes after enqueue, the TCP conn may drop etc. and
			// running the job may occur prior to saving state data.
			//
			// NOTE: This has an "-event" prefix so that it does not conflict
			// with the timeout job ID.
			jobID := fmt.Sprintf("%s-%s-event", md.IdempotencyKey(), pause.DataKey)
			nextItem := queue.Item{
				JobID: &jobID,
				// Add a new group ID for the child;  this will be a new step.
				GroupID:               uuid.New().String(),
				WorkspaceID:           pause.WorkspaceID,
				Kind:                  queue.KindEdge,
				Identifier:            sv2.V1FromMetadata(md),
				PriorityFactor:        md.Config.PriorityFactor,
				CustomConcurrencyKeys: md.Config.CustomConcurrencyKeys,
				MaxAttempts:           pause.MaxAttempts,
				Payload: queue.PayloadEdge{
					Edge: pause.Edge(),
				},
				Metadata: make(map[string]any),
			}

			nextStepSpan, err := e.tracerProvider.CreateDroppableSpan(
				meta.SpanNameStepDiscovery,
				&tracing.CreateSpanOptions{
					Carriers:    []map[string]any{nextItem.Metadata},
					FollowsFrom: pauseSpan,
					Location:    "executor.Resume",
					Metadata:    &md,
					Parent:      tracing.RunSpanRefFromMetadata(&md),
					QueueItem:   &nextItem,
				},
			)
			if err != nil {
				return fmt.Errorf("error creating span for next step after resume: %w", err)
			}

			err = e.queue.Enqueue(ctx, nextItem, time.Now(), queue.EnqueueOpts{})
			if err != nil {
				if err == redis_state.ErrQueueItemExists {
					nextStepSpan.Drop()
				} else {
					_ = nextStepSpan.Send()
					return fmt.Errorf("error enqueueing after pause: %w", err)
				}
			}

			_ = nextStepSpan.Send()
		}

		// Only run lifecycles if we consumed the pause and enqueued next step.
		switch pause.GetOpcode() {
		case enums.OpcodeInvokeFunction:
			for _, e := range e.lifecycles {
				go e.OnInvokeFunctionResumed(context.WithoutCancel(ctx), md, pause, r)
			}
		case enums.OpcodeWaitForSignal:
			for _, e := range e.lifecycles {
				go e.OnWaitForSignalResumed(context.WithoutCancel(ctx), md, pause, r)
			}
		case enums.OpcodeWaitForEvent:
			for _, e := range e.lifecycles {
				go e.OnWaitForEventResumed(context.WithoutCancel(ctx), md, pause, r)
			}
		}

		// The timeout job is running on the queue and will Dequeue() itself. No need to continue.
		if r.IsTimeout {
			return cleanup()
		}

		// And dequeue the timeout job to remove unneeded work from the queue, etc.
		if q, ok := e.queue.(redis_state.QueueManager); ok {
			// timeout jobs are enqueued to the workflow partition (see handleGeneratorWaitForEvent)
			// this is _not_ a system partition and lives on the account shard, which we need to retrieve
			shard, err := e.shardFinder(ctx, md.ID.Tenant.AccountID, nil)
			if err != nil {
				return fmt.Errorf("could not find shard for pause timeout item for account %q: %w", md.ID.Tenant.AccountID, err)
			}

			jobID := fmt.Sprintf("%s-%s", md.IdempotencyKey(), pause.DataKey)
			err = q.Dequeue(ctx, shard, queue.QueueItem{
				ID:         queue.HashID(ctx, jobID),
				FunctionID: md.ID.FunctionID,
				Data: queue.Item{
					Kind:       queue.KindPause,
					Identifier: sv2.V1FromMetadata(md),
				},
			})
			if err != nil {
				if errors.Is(err, redis_state.ErrQueueItemNotFound) {
					logger.StdlibLogger(ctx).Warn("missing pause timeout item", "shard", shard.Name, "pause", pause)
				} else {
					logger.StdlibLogger(ctx).Error("error dequeueing consumed pause job when resuming", "error", err)
				}
			}
		}

		// clean up pause
		return cleanup()
	}, util.WithBoundaries(20*time.Second))
	if err != nil {
		return err
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
		if resp.HasAI() {
			if update == nil {
				update = &sv2.MutableConfig{
					ForceStepPlan:  i.md.Config.ForceStepPlan,
					RequestVersion: resp.RequestVersion,
					StartedAt:      i.md.Config.StartedAt,
				}
			}
			update.HasAI = true
		}
		if update != nil {
			if err := e.smv2.UpdateMetadata(ctx, i.md.ID, *update); err != nil {
				return fmt.Errorf("error updating function metadata: %w", err)
			}
		}
	}

	stepCount := len(resp.Generator)

	if stepCount > consts.DefaultMaxStepLimit {
		// Disallow parallel plans that exceed the step limit
		return state.WrapInStandardError(
			state.ErrFunctionOverflowed,
			state.InngestErrFunctionOverflowed,
			fmt.Sprintf("The function run exceeded the step limit of %d steps.", consts.DefaultMaxStepLimit),
			"",
		)
	}

	groups := opGroups(resp.Generator)

	if stepCount > 1 && i.md.ShouldCoalesceParallelism(resp) {
		if err := e.smv2.SavePending(ctx, i.md.ID, groups.IDs()); err != nil {
			return fmt.Errorf("error saving pending steps: %w", err)
		}
	}

	for _, group := range groups.All() {
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
				e.log.Error("error handling generator", "error", "nil generator returned")
			}
			continue
		}
		copied := *op
		if group.ShouldStartHistoryGroup {
			// Give each opcode its own group ID, since we want to track each
			// parellel step individually.
			i.item.GroupID = uuid.New().String()
		}
		eg.Go(func() error {
			defer func() {
				if r := recover(); r != nil {
					e.log.Error(
						"panic in handleGenerator",
						"error", r,
					)
				}
			}()
			return e.handleGenerator(ctx, i, copied)
		})
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
	case enums.OpcodeAIGateway:
		return e.handleGeneratorAIGateway(ctx, i, gen, edge)
	case enums.OpcodeGateway:
		return e.handleGeneratorGateway(ctx, i, gen, edge)
	case enums.OpcodeWaitForSignal:
		return e.handleGeneratorWaitForSignal(ctx, i, gen, edge)
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

	hasPendingSteps, err := e.smv2.SaveStep(ctx, i.md.ID, gen.ID, []byte(output))
	if err != nil {
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
		Metadata:              make(map[string]any),
	}

	if !hasPendingSteps {
		span, err := e.tracerProvider.CreateDroppableSpan(
			meta.SpanNameStepDiscovery,
			&tracing.CreateSpanOptions{
				Carriers:    []map[string]any{nextItem.Metadata},
				FollowsFrom: tracing.SpanRefFromQueueItem(&i.item),
				Location:    "executor.handleGeneratorStep",
				Metadata:    &i.md,
				Parent:      tracing.RunSpanRefFromMetadata(&i.md),
				QueueItem:   &nextItem,
			},
		)
		if err != nil {
			return fmt.Errorf("error creating span for next step after Step: %w", err)
		}

		err = e.queue.Enqueue(ctx, nextItem, now, queue.EnqueueOpts{})
		if err != nil {
			span.Drop()

			if err == redis_state.ErrQueueItemExists {
				return nil
			}

			logger.StdlibLogger(ctx).Error("error scheduling step queue item", "error", err)

			return err
		}

		_ = span.Send()
	}

	for _, l := range e.lifecycles {
		// We can't specify step name here since that will result in the
		// "followup discovery step" having the same name as its predecessor.
		var stepName *string = nil
		go l.OnStepScheduled(ctx, i.md, nextItem, stepName)
	}

	// NOTE: Default topics are not yet implemented and are a V2 realtime feature.
	//
	// if e.rtpub != nil {
	// 	e.rtpub.Publish(ctx, realtime.Message{
	// 		Kind:       streamingtypes.MessageKindStep,
	// 		Data:       gen.Data,
	// 		Topic:      gen.UserDefinedName(),
	// 		EnvID:      i.md.ID.Tenant.EnvID,
	// 		FnID:       i.md.ID.FunctionID,
	// 		FnSlug:     i.f.GetSlug(),
	// 		Channel:    i.md.ID.RunID.String(),
	// 		CreatedAt:  time.Now(),
	// 		RunID:      i.md.ID.RunID,
	// 	})
	// }

	return nil
}

func (e *executor) handleStepError(ctx context.Context, i *runInstance, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	// With the introduction of the StepError opcode, step errors are handled gracefully, and we can
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
	output, err := gen.Output()
	if err != nil {
		return err
	}

	hasPendingSteps, err := e.smv2.SaveStep(ctx, i.md.ID, gen.ID, []byte(output))
	if err != nil {
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
		Metadata:              make(map[string]any),
	}

	if !hasPendingSteps {
		span, err := e.tracerProvider.CreateDroppableSpan(
			meta.SpanNameStepDiscovery,
			&tracing.CreateSpanOptions{
				Carriers:    []map[string]any{nextItem.Metadata},
				FollowsFrom: tracing.SpanRefFromQueueItem(&i.item),
				Location:    "executor.handleStepError",
				Metadata:    &i.md,
				QueueItem:   &nextItem,
				Parent:      tracing.RunSpanRefFromMetadata(&i.md),
			},
		)
		if err != nil {
			return fmt.Errorf("error creating span for next step after StepError: %w", err)
		}

		err = e.queue.Enqueue(ctx, nextItem, now, queue.EnqueueOpts{})
		if err == redis_state.ErrQueueItemExists {
			span.Drop()
			return nil
		}

		_ = span.Send()
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
		IncomingGeneratorStep:     gen.ID,
		IncomingGeneratorStepName: gen.Name,
		Outgoing:                  edge.Edge.Outgoing,
		Incoming:                  edge.Edge.Incoming,
	}
	// prefer DisplayName if available
	if gen.DisplayName != nil {
		nextEdge.IncomingGeneratorStepName = *gen.DisplayName
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
		Metadata: make(map[string]any),
	}

	span, err := e.tracerProvider.CreateDroppableSpan(
		meta.SpanNameStep,
		&tracing.CreateSpanOptions{
			Carriers:    []map[string]any{nextItem.Metadata},
			FollowsFrom: tracing.SpanRefFromQueueItem(&i.item),
			Location:    "executor.handleGeneratorStepPlanned",
			Metadata:    &i.md,
			QueueItem:   &nextItem,
			Parent:      tracing.RunSpanRefFromMetadata(&i.md),
			SpanOptions: []trace.SpanStartOption{
				tracing.WithGeneratorAttrs(&gen),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error creating span for next step after StepPlanned: %w", err)
	}

	err = e.queue.Enqueue(ctx, nextItem, now, queue.EnqueueOpts{})
	if err == redis_state.ErrQueueItemExists {
		span.Drop()
		return nil
	}

	_ = span.Send()

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

	now := time.Now()

	startedAt := now
	until := startedAt.Add(dur)

	// Create another group for the next item which will run.  We're enqueueing
	// the function to run again after sleep, so need a new group.
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

	jobID := queue.HashID(ctx, fmt.Sprintf("%s-%s", i.md.IdempotencyKey(), gen.ID))
	nextItem := queue.Item{
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
		Metadata:              make(map[string]any),
	}

	span, err := e.tracerProvider.CreateDroppableSpan(
		meta.SpanNameStep,
		&tracing.CreateSpanOptions{
			Carriers:    []map[string]any{nextItem.Metadata},
			FollowsFrom: tracing.SpanRefFromQueueItem(&i.item),
			Location:    "executor.handleGeneratorSleep",
			Metadata:    &i.md,
			QueueItem:   &nextItem,
			Parent:      tracing.RunSpanRefFromMetadata(&i.md),
			SpanOptions: []trace.SpanStartOption{
				tracing.WithGeneratorAttrs(&gen),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error creating span for next step after Sleep: %w", err)
	}

	err = e.queue.Enqueue(ctx, nextItem, until, queue.EnqueueOpts{
		PassthroughJobId: true,
	})
	if err == redis_state.ErrQueueItemExists {
		span.Drop()
		return nil
	}

	_ = span.Send()

	for _, e := range e.lifecycles {
		go e.OnSleep(context.WithoutCancel(ctx), i.md, i.item, gen, until)
	}

	return err
}

func (e *executor) handleGeneratorGateway(ctx context.Context, i *runInstance, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	input, err := gen.GatewayOpts()
	if err != nil {
		return fmt.Errorf("error parsing gateway step: %w", err)
	}

	req, err := input.SerializableRequest()
	if err != nil {
		return fmt.Errorf("error creating gateway request: %w", err)
	}

	// If the opcode contains streaming data, we should fetch a JWT with perms
	// for us to stream then add streaming data to the serializable request.
	//
	// Without this, publishing will not work.
	e.addRequestPublishOpts(ctx, i, &req)

	var output []byte

	resp, err := i.httpClient.DoRequest(ctx, req)
	if err != nil {
		// Request failed entirely. Create an error.
		userLandErr := state.UserError{
			Name:    "GatewayError",
			Message: fmt.Sprintf("Error making gateway request: %s", err),
		}
		i.resp.UpdateOpcodeError(&gen, userLandErr)

		if queue.ShouldRetry(nil, i.item.Attempt, i.item.GetMaxAttempts()) {
			i.resp.SetError(err)

			for _, e := range e.lifecycles {
				go e.OnStepGatewayRequestFinished(context.WithoutCancel(ctx), i.md, i.item, i.edge, gen, nil, &userLandErr)
			}

			// This will retry, as it hits the queue directly.
			return fmt.Errorf("error making inference request: %w", err)
		}

		userLandErrByt, _ := json.Marshal(userLandErr)
		output, _ = json.Marshal(map[string]json.RawMessage{
			execution.StateErrorKey: userLandErrByt,
		})

		for _, e := range e.lifecycles {
			go e.OnStepGatewayRequestFinished(context.WithoutCancel(ctx), i.md, i.item, i.edge, gen, nil, &userLandErr)
		}
	} else {
		headers := make(map[string]string)
		for k, v := range resp.Header {
			headers[k] = strings.Join(v, ",")
		}

		output, err = json.Marshal(map[string]gateway.Response{
			execution.StateDataKey: {
				URL:        req.URL,
				Headers:    headers,
				Body:       string(resp.Body),
				StatusCode: resp.StatusCode,
			},
		})
		if err != nil {
			return fmt.Errorf("error wrapping gateway result in map: %w", err)
		}

		i.resp.UpdateOpcodeOutput(&gen, output)
		for _, e := range e.lifecycles {
			// OnStepFinished handles step success and step errors/failures.  It is
			// currently the responsibility of the lifecycle manager to handle the differing
			// step statuses when a step finishes.
			go e.OnStepGatewayRequestFinished(context.WithoutCancel(ctx), i.md, i.item, i.edge, gen, nil, nil)
		}
	}

	// Save the output as the step result.
	hasPendingSteps, err := e.smv2.SaveStep(ctx, i.md.ID, gen.ID, output)
	if err != nil {
		return err
	}

	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

	// Enqueue the next step
	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Going from the current step
		Incoming: edge.Edge.Incoming, // And re-calling the incoming function in a loop
	}
	jobID := fmt.Sprintf("%s-%s", i.md.IdempotencyKey(), gen.ID)
	now := time.Now()
	nextItem := queue.Item{
		JobID:                 &jobID,
		WorkspaceID:           i.md.ID.Tenant.EnvID,
		GroupID:               groupID,
		Kind:                  queue.KindEdge,
		Identifier:            i.item.Identifier,
		PriorityFactor:        i.item.PriorityFactor,
		CustomConcurrencyKeys: i.item.CustomConcurrencyKeys,
		Attempt:               0,
		MaxAttempts:           i.item.MaxAttempts,
		Payload:               queue.PayloadEdge{Edge: nextEdge},
	}

	if !hasPendingSteps {
		err = e.queue.Enqueue(ctx, nextItem, now, queue.EnqueueOpts{})
		if err == redis_state.ErrQueueItemExists {
			return nil
		}
	}

	for _, l := range e.lifecycles {
		// We can't specify step name here since that will result in the
		// "followup discovery step" having the same name as its predecessor.
		var stepName *string = nil
		go l.OnStepScheduled(ctx, i.md, nextItem, stepName)
	}

	return err
}

func (e *executor) handleGeneratorAIGateway(ctx context.Context, i *runInstance, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	input, err := gen.AIGatewayOpts()
	if err != nil {
		return fmt.Errorf("error parsing ai gateway step: %w", err)
	}

	// NOTE:  It's the responsibility of `trace_lifecycle` to parse the gateway request,
	// then generate an aigateway.ParsedInferenceRequest to store in the history store.
	// This happens automatically within trace_lifecycle.go.

	req, err := input.SerializableRequest()
	if err != nil {
		return fmt.Errorf("error creating ai gateway request: %w", err)
	}

	// If the opcode contains streaming data, we should fetch a JWT with perms
	// for us to stream then add streaming data to the serializable request.
	//
	// Without this, publishing will not work.
	e.addRequestPublishOpts(ctx, i, &req)

	resp, err := i.httpClient.DoRequest(ctx, req)
	failure := err != nil || (resp != nil && resp.StatusCode > 299)

	// Update the driver response appropriately for the trace lifecycles.
	if resp == nil {
		resp = &exechttp.Response{}
	}

	i.resp.StatusCode = resp.StatusCode

	// Handle errors individually, here.
	if failure {
		if len(resp.Body) == 0 {
			// Add some output for the response.
			resp.Body = []byte(`{"error":"Error making AI request"}`)
		}

		if err == nil {
			err = fmt.Errorf("unsuccessful status code: %d", resp.StatusCode)
		}

		// Ensure the opcode is treated as an error when calling OnStepFinish.
		userLandErr := state.UserError{
			Name:    "AIGatewayError",
			Message: fmt.Sprintf("Error making AI request: %s", err),
			Data:    resp.Body, // For golang's multiple returns.
			Stack:   string(resp.Body),
		}
		i.resp.UpdateOpcodeError(&gen, userLandErr)

		// And, finally, if this is retryable return an error which will be retried.
		// Otherwise, we enqueue the next step directly so that the SDK can throw
		// an error on output.
		if queue.ShouldRetry(nil, i.item.Attempt, i.item.GetMaxAttempts()) {
			// Set the response error, ensuring the response is retryable in the queue.
			i.resp.SetError(err)

			for _, e := range e.lifecycles {
				// OnStepFinished handles step success and step errors/failures.  It is
				// currently the responsibility of the lifecycle manager to handle the differing
				// step statuses when a step finishes.
				go e.OnStepGatewayRequestFinished(context.WithoutCancel(ctx), i.md, i.item, i.edge, gen, nil, &userLandErr)
			}

			// This will retry, as it hits the queue directly.
			return fmt.Errorf("error making inference request: %w", err)
		}

		// If we can't retry, carry on by enqueueing the next step, in the same way
		// that OpcodeStepError works.
		//
		// The actual error should be wrapped with an "error" so that it respects the
		// error wrapping of step errors.
		userLandErrByt, _ := json.Marshal(userLandErr)
		resp.Body, _ = json.Marshal(map[string]json.RawMessage{
			execution.StateErrorKey: userLandErrByt,
		})

		for _, e := range e.lifecycles {
			// OnStepFinished handles step success and step errors/failures.  It is
			// currently the responsibility of the lifecycle manager to handle the differing
			// step statuses when a step finishes.
			go e.OnStepGatewayRequestFinished(context.WithoutCancel(ctx), i.md, i.item, i.edge, gen, nil, &userLandErr)
		}
	} else {
		// The response output is actually now the result of this AI call. We need
		// to modify the opcode data so that accessing the step output is correct.
		//
		// Also note that the output is always wrapped within "data", allowing us
		// to differentiate between success and failure in the SDK in the single
		// opcode map.
		resp.Body, err = json.Marshal(map[string]json.RawMessage{
			execution.StateDataKey: resp.Body,
		})
		if err != nil {
			return fmt.Errorf("error wrapping ai result in map: %w", err)
		}

		i.resp.UpdateOpcodeOutput(&gen, resp.Body)
		for _, e := range e.lifecycles {
			// OnStepFinished handles step success and step errors/failures.  It is
			// currently the responsibility of the lifecycle manager to handle the differing
			// step statuses when a step finishes.
			go e.OnStepGatewayRequestFinished(context.WithoutCancel(ctx), i.md, i.item, i.edge, gen, nil, nil)
		}
	}

	// Save the output as the step result.
	hasPendingSteps, err := e.smv2.SaveStep(ctx, i.md.ID, gen.ID, resp.Body)
	if err != nil {
		return err
	}

	// XXX: If auto-call is supported and a tool is provided, auto-call invokes
	// before scheduling the next step.  This can only happen if the tool is an
	// invoke.  We do not support this yet.

	// XXX: Remove once deprecated from history.
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

	// Enqueue the next step
	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Going from the current step
		Incoming: edge.Edge.Incoming, // And re-calling the incoming function in a loop
	}
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

	if !hasPendingSteps {
		err = e.queue.Enqueue(ctx, nextItem, now, queue.EnqueueOpts{})
		if err == redis_state.ErrQueueItemExists {
			return nil
		}
	}

	for _, l := range e.lifecycles {
		// We can't specify step name here since that will result in the
		// "followup discovery step" having the same name as its predecessor.
		var stepName *string = nil
		go l.OnStepScheduled(ctx, i.md, nextItem, stepName)
	}

	return err
}

func (e *executor) handleGeneratorWaitForSignal(ctx context.Context, i *runInstance, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	opts, err := gen.SignalOpts()
	if err != nil {
		return fmt.Errorf("unable to parse signal opts: %w", err)
	}
	if opts.Signal == "" {
		return fmt.Errorf("signal name is empty")
	}
	expires, err := opts.Expires()
	if err != nil {
		return fmt.Errorf("unable to parse signal expires: %w", err)
	}

	pauseID := inngest.DeterministicSha1UUID(i.md.ID.RunID.String() + gen.ID)
	opcode := gen.Op.String()
	now := time.Now()

	sid := run.NewSpanID(ctx)
	carrier := itrace.NewTraceCarrier(
		itrace.WithTraceCarrierTimestamp(now),
		itrace.WithTraceCarrierSpanID(&sid),
	)
	itrace.UserTracer().Propagator().Inject(ctx, propagation.MapCarrier(carrier.Context))

	// Default to failing if there's a conflict
	shouldReplaceSignalOnConflict := false
	if opts.OnConflict == "replace" {
		shouldReplaceSignalOnConflict = true
	}

	pause := state.Pause{
		ID:                      pauseID,
		WorkspaceID:             i.md.ID.Tenant.EnvID,
		Identifier:              sv2.NewPauseIdentifier(i.md.ID),
		GroupID:                 i.item.GroupID,
		Outgoing:                gen.ID,
		Incoming:                edge.Edge.Incoming,
		StepName:                gen.UserDefinedName(),
		Opcode:                  &opcode,
		Expires:                 state.Time(expires),
		DataKey:                 gen.ID,
		SignalID:                &opts.Signal,
		ReplaceSignalOnConflict: shouldReplaceSignalOnConflict,
		MaxAttempts:             i.item.MaxAttempts,
		Metadata: map[string]any{
			consts.OtelPropagationKey: carrier,
		},
	}

	_, err = e.pm.Write(ctx, pauses.PauseIndex(pause), &pause)
	if err == state.ErrSignalConflict {
		return state.WrapInStandardError(
			err,
			"Error",
			"Signal conflict; signal wait already exists for another run",
			"",
		)
	}
	if err != nil && !errors.Is(err, state.ErrPauseAlreadyExists) {
		return fmt.Errorf("error saving pause when handling WaitForSignal opcode: %w", err)
	}

	// Enqueue a job that will timeout the pause.
	jobID := fmt.Sprintf("%s-%s", i.md.IdempotencyKey(), gen.ID)
	err = e.queue.Enqueue(ctx, queue.Item{
		JobID:                 &jobID,
		WorkspaceID:           i.md.ID.Tenant.EnvID,
		GroupID:               i.item.GroupID,
		Kind:                  queue.KindPause,
		Identifier:            i.item.Identifier,
		PriorityFactor:        i.item.PriorityFactor,
		CustomConcurrencyKeys: i.item.CustomConcurrencyKeys,
		MaxAttempts:           i.item.MaxAttempts,
		Payload: queue.PayloadPauseTimeout{
			PauseID: pauseID,
			Pause:   pause,
		},
	}, expires, queue.EnqueueOpts{})
	if err == redis_state.ErrQueueItemExists {
		return nil
	}

	for _, e := range e.lifecycles {
		go e.OnWaitForSignal(
			context.WithoutCancel(ctx),
			i.md,
			i.item,
			gen,
			pause,
		)
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

	pauseID := inngest.DeterministicSha1UUID(i.md.ID.RunID.String() + gen.ID)
	opcode := gen.Op.String()
	now := time.Now()

	sid := run.NewSpanID(ctx)
	// NOTE: the context here still contains the execSpan's traceID & spanID,
	// which is what we want because that's the parent that needs to be referenced later on
	carrier := itrace.NewTraceCarrier(
		itrace.WithTraceCarrierTimestamp(now),
		itrace.WithTraceCarrierSpanID(&sid),
	)
	itrace.UserTracer().Propagator().Inject(ctx, propagation.MapCarrier(carrier.Context))

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

	pause := state.Pause{
		ID:                  pauseID,
		WorkspaceID:         i.md.ID.Tenant.EnvID,
		Identifier:          sv2.NewPauseIdentifier(i.md.ID),
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
	}
	_, err = e.pm.Write(
		ctx,
		pauses.Index{WorkspaceID: i.md.ID.Tenant.EnvID, EventName: eventName},
		&pause,
	)
	if err == state.ErrPauseAlreadyExists {
		return nil
	}
	if err != nil {
		return err
	}

	// Enqueue a job that will timeout the pause.
	jobID := fmt.Sprintf("%s-%s", i.md.IdempotencyKey(), gen.ID)
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
			PauseID: pauseID,
			Pause:   pause,
		},
	}, expires, queue.EnqueueOpts{})
	if err == redis_state.ErrQueueItemExists {
		return nil
	} else if err != nil {
		logger.StdlibLogger(ctx).Error(
			"failed to enqueue invoke function pause timeout",
			"error", err,
			"run_id", i.md.ID.RunID,
			"workspace_id", i.md.ID.Tenant.EnvID,
		)
	}

	// Send the event.
	err = e.handleSendingEvent(ctx, evt, i.item)
	if err != nil {
		// TODO Cancel pause/timeout?
		return fmt.Errorf("error publishing internal invocation event: %w", err)
	}

	for _, e := range e.lifecycles {
		go e.OnInvokeFunction(context.WithoutCancel(ctx), i.md, i.item, gen, evt)
	}

	return err
}

func (e *executor) handleGeneratorWaitForEvent(ctx context.Context, i *runInstance, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	opts, err := gen.WaitForEventOpts()
	if err != nil {
		return fmt.Errorf("unable to parse wait for event opts: %w", err)
	}

	if opts.If != nil {
		if err = expressions.Validate(ctx, expressions.DefaultRestrictiveValidationPolicy(), *opts.If); err != nil {
			if errors.Is(err, expressions.ErrValidationFailed) {
				logger.StdlibLogger(ctx).
					With("err", err.Error()).
					With("expression", *opts.If).
					Warn("waitForEvent If expression failed validation")
				// "just log a warning right now, then we can collect stats and do our own alerting a week in" - Tony, 2025-05-07
				// intentionally not returning; continue handling this as before for now
			} else if errors.Is(err, expressions.ErrCompileFailed) {
				return state.WrapInStandardError(
					err,
					"InvalidExpression",
					"Wait for event If expression failed to compile",
					err.Error(),
				)
			} else {
				return state.WrapInStandardError(
					err,
					"InvalidExpression",
					"Wait for event If expression is invalid",
					err.Error(),
				)
			}
		}
	}

	expires, err := opts.Expires()
	if err != nil {
		return fmt.Errorf("unable to parse wait for event expires: %w", err)
	}

	pauseID := inngest.DeterministicSha1UUID(i.md.ID.RunID.String() + gen.ID)

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
	now := time.Now()

	sid := run.NewSpanID(ctx)
	// NOTE: the context here still contains the execSpan's traceID & spanID,
	// which is what we want because that's the parent that needs to be referenced later on
	carrier := itrace.NewTraceCarrier(
		itrace.WithTraceCarrierTimestamp(now),
		itrace.WithTraceCarrierSpanID(&sid),
	)
	itrace.UserTracer().Propagator().Inject(ctx, propagation.MapCarrier(carrier.Context))

	// SDK-based event coordination is called both when an event is received
	// OR on timeout, depending on which happens first.  Both routes consume
	// the pause so this race will conclude by calling the function once, as only
	// one thread can lease and consume a pause;  the other will find that the
	// pause is no longer available and return.
	pause := state.Pause{
		ID:          pauseID,
		WorkspaceID: i.md.ID.Tenant.EnvID,
		Identifier:  sv2.NewPauseIdentifier(i.md.ID),
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
	}

	// SDK-based event coordination is called both when an event is received
	// OR on timeout, depending on which happens first.  Both routes consume
	// the pause so this race will conclude by calling the function once, as only
	// one thread can lease and consume a pause;  the other will find that the
	// pause is no longer available and return.
	jobID := fmt.Sprintf("%s-%s", i.md.IdempotencyKey(), gen.ID)
	nextItem := queue.Item{
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
			PauseID: pauseID,
			Pause:   pause,
		},
		Metadata: make(map[string]any),
	}

	span, err := e.tracerProvider.CreateDroppableSpan(
		meta.SpanNameStep,
		&tracing.CreateSpanOptions{
			Carriers:    []map[string]any{pause.Metadata, nextItem.Metadata},
			FollowsFrom: tracing.SpanRefFromQueueItem(&i.item),
			Location:    "executor.handleGeneratorWaitForEvent",
			Metadata:    &i.md,
			QueueItem:   &nextItem,
			Parent:      tracing.RunSpanRefFromMetadata(&i.md),
			SpanOptions: []trace.SpanStartOption{
				tracing.WithGeneratorAttrs(&gen),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error creating span for next step after WaitForEvent: %w", err)
	}

	idx := pauses.Index{WorkspaceID: i.md.ID.Tenant.EnvID, EventName: opts.Event}
	_, err = e.pm.Write(ctx, idx, &pause)
	if err != nil {
		if err == state.ErrPauseAlreadyExists {
			span.Drop()
			return nil
		}

		return err
	}

	// TODO Is this fine to leave? No attempts.
	err = e.queue.Enqueue(ctx, nextItem, expires, queue.EnqueueOpts{})
	if err == redis_state.ErrQueueItemExists {
		span.Drop()
		return nil
	}

	_ = span.Send()

	for _, e := range e.lifecycles {
		go e.OnWaitForEvent(context.WithoutCancel(ctx), i.md, i.item, gen, pause)
	}

	return err
}

func (e *executor) newExpressionEvaluator(ctx context.Context, expr string) (expressions.Evaluator, error) {
	if e.evalFactory != nil {
		return e.evalFactory(ctx, expr)
	}
	return expressions.NewExpressionEvaluator(ctx, expr)
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

		metrics.IncrBatchScheduledCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"account_id":  bi.AccountID.String(),
				"function_id": bi.FunctionID.String(),
			},
		})
	case enums.BatchFull, enums.BatchMaxSize:
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

	// root span for scheduling a batch
	ctx, span := run.NewSpan(ctx,
		run.WithScope(consts.OtelScopeBatch),
		run.WithName(consts.OtelSpanBatch),
		run.WithNewRoot(),
		run.WithSpanAttributes(
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

	// Ensure to delete batch when Schedule worked, we already processed it, or the function was paused
	shouldDeleteBatch := err == nil ||
		err == redis_state.ErrQueueItemExists ||
		errors.Is(err, ErrFunctionSkipped) ||
		errors.Is(err, ErrFunctionSkippedIdempotency) ||
		errors.Is(err, state.ErrIdentifierExists)
	if shouldDeleteBatch {
		// TODO: check if all errors can be blindly returned
		if err := e.batcher.DeleteKeys(ctx, payload.FunctionID, payload.BatchID); err != nil {
			return err
		}
	}

	// Don't bother if it's already there
	// If function is paused, we do not schedule runs
	if err == redis_state.ErrQueueItemExists ||
		errors.Is(err, ErrFunctionSkipped) ||
		errors.Is(err, ErrFunctionSkippedIdempotency) {
		span.SetAttributes(attribute.Bool(consts.OtelSysStepDelete, true))
		return nil
	}

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.Bool(consts.OtelSysStepDelete, true))
		return err
	}

	metrics.IncrBatchProcessStartCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			// whether batch was full or started by timeout
			"batch_timeout": opts == nil,
			"account_id":    payload.AccountID.String(),
		},
	})

	if md != nil {
		span.SetAttributes(attribute.String(consts.OtelAttrSDKRunID, md.ID.RunID.String()))
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

func (e *executor) ResumeSignal(ctx context.Context, workspaceID uuid.UUID, signalID string, data json.RawMessage) (res *execution.ResumeSignalResult, err error) {
	if workspaceID == uuid.Nil {
		err = fmt.Errorf("workspace ID is empty")
		return
	}

	if signalID == "" {
		err = fmt.Errorf("signal ID is empty")
		return
	}

	sanitizedSignalID := strings.ReplaceAll(signalID, "\n", "")
	sanitizedSignalID = strings.ReplaceAll(sanitizedSignalID, "\r", "")
	l := e.log.With("signal_id", sanitizedSignalID, "workspace_id", workspaceID.String())
	defer func() {
		if err != nil {
			l.Error("error receiving signal", "error", err)
		} else {
			l.Info("signal received")
		}
	}()

	pause, err := e.pm.PauseBySignalID(ctx, workspaceID, signalID)
	if err != nil {
		err = fmt.Errorf("error getting pause by signal ID: %w", err)
		return
	}

	res = &execution.ResumeSignalResult{}

	if pause == nil {
		l.Debug("no pause found for signal")
		return
	}

	if pause.Expires.Time().Before(time.Now()) {
		l.Debug("encountered expired signal")

		shouldDelete := pause.Expires.Time().Add(consts.PauseExpiredDeletionGracePeriod).Before(time.Now())
		if shouldDelete {
			l.Debug("deleting expired pause")
			_ = e.pm.Delete(ctx, pauses.PauseIndex(*pause), *pause)
		}

		return
	}

	l.Debug("resuming pause from signal", "pause.DataKey", pause.DataKey)

	err = e.Resume(ctx, *pause, execution.ResumeRequest{
		RunID:          &pause.Identifier.RunID,
		StepName:       pause.StepName,
		IdempotencyKey: signalID,
		With: map[string]any{
			execution.StateDataKey: state.SignalStepReturn{
				Signal: signalID,
				Data:   data,
			},
		},
	})
	if err != nil {
		if errors.Is(err, state.ErrPauseLeased) ||
			errors.Is(err, state.ErrPauseNotFound) ||
			errors.Is(err, state.ErrRunNotFound) {
			// Just return that we found nothing
			err = nil
		}

		return
	}

	res.MatchedSignal = true
	res.RunID = &pause.Identifier.RunID

	return
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

// extractTraceCtx extracts the trace context from the given item, if it exists.
// If it doesn't it falls back to extracting the trace for the run overall.
// If neither exist or they are invalid, it returns the original context.
func extractTraceCtx(ctx context.Context, md sv2.Metadata) context.Context {
	fntrace := md.Config.FunctionTrace()
	if fntrace != nil {
		// NOTE:
		// this gymastics happens because the carrier stores the spanID separately.
		// it probably can be simplified
		tmp := itrace.UserTracer().Propagator().Extract(ctx, propagation.MapCarrier(fntrace.Context))
		spanID, err := md.Config.GetSpanID()
		if err != nil {
			return ctx
		}

		sctx := trace.SpanContextFromContext(tmp).WithSpanID(*spanID)
		return trace.ContextWithSpanContext(ctx, sctx)
	}

	return ctx
}

// addRequestPublishOpts generates a new JWT to publish gateway requests in realtime.
func (e *executor) addRequestPublishOpts(ctx context.Context, i *runInstance, sr *exechttp.SerializableRequest) {
	if e.rtconfig.PublishURL == "" {
		return
	}

	token, err := realtime.NewPublishJWT(
		ctx,
		e.rtconfig.Secret,
		i.item.Identifier.AccountID,
		i.item.WorkspaceID,
	)
	if err != nil {
		// XXX: We should be able to attach warnings to runs;  in this case, we couldn't create
		// a JWT to publish data.  However, the step should still execute without realtime publishing,
		// and the UI should show a warning for this run.
		return
	}

	sr.Publish.Token = token
	sr.Publish.PublishURL = e.rtconfig.PublishURL
}
