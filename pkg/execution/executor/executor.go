package executor

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/structs"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/apiresult"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/cancellation"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/executor/queueref"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/inngest/inngest/pkg/execution/realtime"
	"github.com/inngest/inngest/pkg/execution/singleton"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/expressions/expragg"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/run"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/pkg/util/gateway"
	"github.com/inngest/inngestgo"
	"github.com/jonboulle/clockwork"
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
	ErrFunctionRateLimited        = fmt.Errorf("function rate-limited")
	ErrFunctionSkipped            = fmt.Errorf("function skipped")
	ErrFunctionSkippedIdempotency = fmt.Errorf("function skipped due to idempotency")

	ErrFunctionEnded   = fmt.Errorf("function already ended")
	ErrNoCorrelationID = fmt.Errorf("no correlation ID found in event when trying to resume invoke parent")

	// ErrHandledStepError is returned when an OpcodeStepError is caught and the
	// step should be safely retried.
	ErrHandledStepError = fmt.Errorf("handled step error")

	PauseHandleConcurrency = 100
)

const (
	RateLimitIdempotencyTTL = 30 * time.Minute
)

// ScheduleStatus returns a string status category for a Schedule error.
// This is useful for metrics and observability to categorize schedule attempts.
func ScheduleStatus(err error) string {
	switch {
	case err == nil:
		return "success"
	case errors.Is(err, ErrFunctionRateLimited):
		return "rate_limited"
	case errors.Is(err, ErrFunctionDebounced):
		return "debounced"
	case errors.Is(err, ErrFunctionSkipped):
		return "skipped"
	case errors.Is(err, queue.ErrQueueItemExists), errors.Is(err, ErrFunctionSkippedIdempotency), errors.Is(err, state.ErrIdentifierExists):
		return "idempotency"
	case err != nil:
		return "error"
	default:
		// should be unreachable
		return "unknown"
	}
}

// NewExecutor returns a new executor, responsible for running the specific step of a
// function (using the available drivers) and storing the step's output or error.
//
// Note that this only executes a single step of the function;  it returns which children
// can be directly executed next and saves a state.Pause for edges that have async conditions.
func NewExecutor(opts ...ExecutorOpt) (execution.Executor, error) {
	m := &executor{
		driverv1: map[string]driver.DriverV1{},
		driverv2: map[string]driver.DriverV2{},
		clock:    clockwork.NewRealClock(),
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

func WithInvokeEventHandler(f execution.HandleInvokeEvent) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).handleInvokeEvent = f
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

func WithRateLimiter(rl ratelimit.RateLimiter) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).rateLimiter = rl
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

func WithCapacityManager(cm constraintapi.CapacityManager) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).capacityManager = cm
		return nil
	}
}

func WithUseConstraintAPI(uca constraintapi.UseConstraintAPIFn) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).useConstraintAPI = uca
		return nil
	}
}

func WithEnableBatchingInstrumentation(ebi func(ctx context.Context, accountID, envID uuid.UUID) (enable bool)) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).enableBatchingInstrumentation = ebi
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

func WithSigningKeyLoader(f func(ctx context.Context, envID uuid.UUID) ([]byte, error)) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).signingKeyLoader = f
		return nil
	}
}

// WithDriverV1 specifies the drivers available to use when executing steps
// of a function.
//
// When invoking a step in a function, we find the registered driver with the step's URI
// and use that driver to execute the step.
func WithDriverV1(drivers ...driver.DriverV1) ExecutorOpt {
	return func(exec execution.Executor) error {
		e := exec.(*executor)
		for _, d := range drivers {
			if _, ok := e.driverv1[d.Name()]; ok {
				return ErrRuntimeRegistered
			}
			e.driverv1[d.Name()] = d

		}
		return nil
	}
}

func WithDriverV2(drivers ...driver.DriverV2) ExecutorOpt {
	return func(exec execution.Executor) error {
		e := exec.(*executor)
		for _, d := range drivers {
			if _, ok := e.driverv2[d.Name()]; ok {
				return ErrRuntimeRegistered
			}
			e.driverv2[d.Name()] = d

		}
		return nil
	}
}

func WithAssignedQueueShard(shard queue.QueueShard) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).assignedQueueShard = shard
		return nil
	}
}

func WithShardSelector(selector queue.ShardSelector) ExecutorOpt {
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

func WithClock(clock clockwork.Clock) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).clock = clock
		return nil
	}
}

type ExecutorRealtimeConfig struct {
	Secret     []byte
	PublishURL string
}

// AllowStepMetadata determines if step metadata should be enabled for the account
type AllowStepMetadata func(ctx context.Context, acctID uuid.UUID) bool

func (am AllowStepMetadata) Enabled(ctx context.Context, acctID uuid.UUID) bool {
	if am == nil {
		return false
	}

	return am(ctx, acctID)
}

func WithAllowStepMetadata(md AllowStepMetadata) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).allowStepMetadata = md
		return nil
	}
}

func WithFunctionBacklogSizeLimit(fbsl BacklogSizeLimitFn) ExecutorOpt {
	return func(e execution.Executor) error {
		e.(*executor).functionBacklogSizeLimit = fbsl
		return nil
	}
}

type BacklogSizeLimit struct {
	Limit   int
	Enforce bool
}

type BacklogSizeLimitFn func(ctx context.Context, accountID, envID, fnID uuid.UUID) BacklogSizeLimit

// executor represents a built-in executor for running workflows.
type executor struct {
	log logger.Logger

	// exprAggregator is an expression aggregator used to parse and aggregate expressions
	// using trees.
	exprAggregator expragg.Aggregator

	pm   pauses.Manager
	smv2 sv2.RunService

	rateLimiter  ratelimit.RateLimiter
	queue        queue.Queue
	debouncer    debounce.Debouncer
	batcher      batch.BatchManager
	singletonMgr singleton.Singleton

	capacityManager               constraintapi.CapacityManager
	useConstraintAPI              constraintapi.UseConstraintAPIFn
	enableBatchingInstrumentation func(ctx context.Context, accountID, envID uuid.UUID) (enable bool)

	fl                  state.FunctionLoader
	evalFactory         func(ctx context.Context, expr string) (expressions.Evaluator, error)
	finishHandler       execution.FinalizePublisher
	invokeFailHandler   execution.InvokeFailHandler
	handleInvokeEvent   execution.HandleInvokeEvent
	cancellationChecker cancellation.Checker
	httpClient          exechttp.RequestExecutor
	// signingKeyLoader is used to load signing keys for an env.  This is required for the
	// HTTPv2 driver.
	signingKeyLoader func(ctx context.Context, envID uuid.UUID) ([]byte, error)

	driverv1 map[string]driver.DriverV1
	driverv2 map[string]driver.DriverV2

	lifecycles []execution.LifecycleListener

	// rtpub represents teh realtime publisher used to broadcast notifications
	// on run execution.
	rtpub    realtime.Publisher
	rtconfig ExecutorRealtimeConfig

	// steplimit finds step limits for a given run.
	steplimit func(sv2.ID) int

	// stateSizeLimit finds state size limits for a given run
	stateSizeLimit func(sv2.ID) int

	functionBacklogSizeLimit BacklogSizeLimitFn

	assignedQueueShard queue.QueueShard
	shardFinder        queue.ShardSelector

	traceReader    cqrs.TraceReader
	tracerProvider tracing.TracerProvider

	allowStepMetadata AllowStepMetadata
	clock             clockwork.Clock
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

func (e *executor) createCancellationPauses(ctx context.Context, l logger.Logger, idempontenceKey string, evtMap map[string]any, id sv2.ID, req execution.ScheduleRequest) error {
	for _, c := range req.Function.Cancel {
		expires := e.now().Add(consts.CancelTimeout)
		if c.Timeout != nil {
			dur, err := str2duration.ParseDuration(*c.Timeout)
			if err != nil {
				return fmt.Errorf("error parsing cancel duration: %w", err)
			}
			expires = e.now().Add(dur)
		}

		// The triggering event ID should be the first ID in the batch.
		triggeringID := req.Events[0].GetInternalID().String()
		idSrc := fmt.Sprintf("%s-%s", idempontenceKey, c.Event)

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
				l.Warn(
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
			WorkspaceID:       id.Tenant.EnvID,
			Identifier:        sv2.NewPauseIdentifier(id),
			ID:                pauseID,
			Expires:           state.Time(expires),
			Event:             &c.Event,
			Expression:        expr,
			Cancel:            true,
			TriggeringEventID: &triggeringID,
		}
		_, err := e.pm.Write(ctx, pauses.Index{WorkspaceID: req.WorkspaceID, EventName: c.Event}, &pause)
		if err != nil && err != state.ErrPauseAlreadyExists {
			return err
		}
	}
	return nil
}

// enqueue a system job in the future for eager cancellation of timed out jobs.
func (e *executor) createEagerCancellationForTimeout(ctx context.Context, since time.Time, timeout *time.Duration, cancellationKind enums.CancellationKind, id state.Identifier) error {
	l := logger.StdlibLogger(context.Background()).With("run_id", id.RunID, "kind", cancellationKind)

	// no timeout or invalid timeout, nothing to do
	if timeout == nil || *timeout <= 0 {
		l.Warn("attempting to create eager cancellation system jobs with empty or invalid timeout")
		return nil
	}

	var systemJobPrefix string
	switch cancellationKind {
	case enums.CancellationKindFinishTimeout:
		systemJobPrefix = "eager-cancel-finish-timeout"
	case enums.CancellationKindStartTimeout:
		systemJobPrefix = "eager-cancel-start-timeout"
	default:
		return fmt.Errorf("invalid cancellation kind: %s", cancellationKind)
	}

	// enqueue a system job for the finish timeout to eagerly cancel this run and all pending queue items for this function that are delayed beyond the timeout.
	enqueueAt := since.Add(*timeout)
	eagerCancelJobID := fmt.Sprintf("%s-%s:%s", systemJobPrefix, id.WorkflowID, id.IdempotencyKey())
	queueName := queue.KindCancel
	maxAttempts := consts.MaxRetries + 1

	l = l.With("systemJobId", eagerCancelJobID, "enqueueAt", enqueueAt)

	// Schedule for async functons (the default)
	c := cqrs.Cancellation{
		ID:          ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:   id.AccountID,
		WorkspaceID: id.WorkspaceID,
		FunctionID:  id.WorkflowID,
		Kind:        cancellationKind,
		Type:        enums.CancellationTypeEvent,
		TargetID:    id.RunID.String(),
	}
	err := e.queue.Enqueue(ctx, queue.Item{
		JobID:       &eagerCancelJobID,
		GroupID:     uuid.New().String(),
		WorkspaceID: id.WorkspaceID,
		Kind:        queue.KindCancel,
		Identifier: state.Identifier{
			AccountID:   id.AccountID,
			WorkspaceID: id.WorkspaceID,
			AppID:       id.AppID,
			WorkflowID:  id.WorkflowID,
			Key:         eagerCancelJobID,
		},
		MaxAttempts: &maxAttempts,
		Payload:     c,
		QueueName:   &queueName,
	}, enqueueAt, queue.EnqueueOpts{})

	if err != nil && err != queue.ErrQueueItemExists {
		l.Trace("Error enqueueing system job", "error", err.Error())
		return err
	}
	l.Trace("Enqueued system job for eager cancellation of timed out job")

	return nil
}

func (e *executor) skipped(ctx context.Context, req execution.ScheduleRequest) enums.SkipReason {
	l := logger.StdlibLogger(ctx)

	// Check if function is paused, draining
	skipReason := req.SkipReason()
	if skipReason != enums.SkipReasonNone {
		return skipReason
	}

	// Check if backlog size limit was hit
	res, err := e.checkBacklogSizeLimit(ctx, req)
	if err != nil {
		l.ReportError(err, "error checking backlog size limit")
		return enums.SkipReasonNone
	}

	return res
}

func (e *executor) checkBacklogSizeLimit(ctx context.Context, req execution.ScheduleRequest) (enums.SkipReason, error) {
	if e.functionBacklogSizeLimit == nil {
		return enums.SkipReasonNone, nil
	}

	backlogSizeLimit := e.functionBacklogSizeLimit(ctx, req.AccountID, req.WorkspaceID, req.Function.ID)
	if backlogSizeLimit.Limit <= 0 {
		return enums.SkipReasonNone, nil
	}

	scheduledSteps, err := e.queue.StatusCount(ctx, req.Function.ID, "start")
	if err != nil {
		return enums.SkipReasonNone, fmt.Errorf("could not get scheduled step count: %w", err)
	}

	if int(scheduledSteps) < backlogSizeLimit.Limit {
		return enums.SkipReasonNone, nil
	}

	// The backlog size exceeds the limit

	id := sv2.ID{
		FunctionID: req.Function.ID,
		Tenant: sv2.Tenant{
			AccountID: req.AccountID,
			EnvID:     req.WorkspaceID,
			AppID:     req.AppID,
		},
	}

	for _, ll := range e.lifecycles {
		service.Go(func() {
			ll.OnFunctionBacklogSizeLimitReached(ctx, id)
		})
	}

	if !backlogSizeLimit.Enforce {
		return enums.SkipReasonNone, nil
	}

	return enums.SkipReasonFunctionBacklogSizeLimitHit, nil
}

func (e *executor) Schedule(ctx context.Context, req execution.ScheduleRequest) (*sv2.Metadata, error) {
	// Run IDs are created embedding the timestamp now, when the function is being scheduled.
	// When running a cancellation, functions are cancelled at scheduling time based off of
	// this run ID.
	var runID ulid.ULID

	if req.RunID == nil {
		runID = ulid.MustNew(ulid.Now(), rand.Reader)
	} else {
		runID = *req.RunID
	}

	key := idempotencyKey(req, runID)

	// Check constraints and acquire lease
	return WithConstraints(
		ctx,
		e.now(),
		e.capacityManager,
		e.useConstraintAPI,
		req,
		key,
		func(ctx context.Context, performChecks bool) (*sv2.Metadata, error) {
			return util.CritT(ctx, "schedule", func(ctx context.Context) (*sv2.Metadata, error) {
				return e.schedule(ctx, req, runID, key, performChecks)
			}, util.WithBoundaries(2*time.Second))
		})
}

func (e *executor) now() time.Time {
	if e.clock != nil {
		return e.clock.Now()
	}
	return time.Now()
}

// Execute loads a workflow and the current run state, then executes the
// function's step via the necessary driver.
//
// If this function has a debounce config, this will return ErrFunctionDebounced instead
// of an identifier as the function is not scheduled immediately.
func (e *executor) schedule(
	ctx context.Context,
	req execution.ScheduleRequest,
	runID ulid.ULID,
	// key is the idempotency key
	key string,
	// performChecks determines whether constraint checks must be performed
	// This may be false when the Constraint API was used to enforce constraints.
	performChecks bool,
) (*sv2.Metadata, error) {
	if req.AppID == uuid.Nil {
		return nil, fmt.Errorf("app ID is required to schedule a run")
	}

	l := e.log.With(
		"account_id", req.AccountID,
		"env_id", req.WorkspaceID,
		"app_id", req.AppID,
		"fn_id", req.Function.ID,
		"fn_v", req.Function.FunctionVersion,
		"evt_id", req.Events[0].GetInternalID(),
	)

	if performChecks {
		// Attempt to rate-limit the incoming function.
		if e.rateLimiter != nil && req.Function.RateLimit != nil && !req.PreventRateLimit {
			evtMap := req.Events[0].GetEvent().Map()
			rateLimitKey, err := ratelimit.RateLimitKey(ctx, req.Function.ID, *req.Function.RateLimit, evtMap)
			switch err {
			case nil:
				res, err := e.rateLimiter.RateLimit(
					logger.WithStdlib(ctx, l),
					rateLimitKey,
					*req.Function.RateLimit,
					ratelimit.WithNow(e.now()),
					ratelimit.WithIdempotency(key, RateLimitIdempotencyTTL),
				)
				if err != nil {
					metrics.IncrRateLimitUsage(ctx, metrics.CounterOpt{
						PkgName: pkgName,
						Tags: map[string]any{
							"impl":   "lua",
							"status": "error",
						},
					})
					return nil, fmt.Errorf("could not check rate limit: %w", err)
				}

				if res.Limited {
					// Do nothing.
					metrics.IncrRateLimitUsage(ctx, metrics.CounterOpt{
						PkgName: pkgName,
						Tags: map[string]any{
							"impl":   "lua",
							"status": "limited",
						},
					})
					metrics.IncrScheduleConstraintsHitCounter(ctx, "rate_limit", metrics.CounterOpt{
						PkgName: pkgName,
						Tags: map[string]any{
							"constraint_api": false,
						},
					})
					return nil, ErrFunctionRateLimited
				}

				status := "allowed"
				if res.IdempotencyHit {
					status = "idempotent"
				}

				metrics.IncrRateLimitUsage(ctx, metrics.CounterOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"impl":   "lua",
						"status": status,
					},
				})
			case ratelimit.ErrNotRateLimited:
				// no-op: proceed with function run as usual
			default:
				return nil, fmt.Errorf("could not evaluate rate limit: %w", err)
			}
		}
	}

	// NOTE: From this point, we are guaranteed to operate within user constraints.

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

	if req.Context == nil {
		req.Context = map[string]any{}
	}

	// Normalization
	eventIDs := []ulid.ULID{}
	for _, e := range req.Events {
		id := e.GetInternalID()
		eventIDs = append(eventIDs, id)
	}

	var eventName *string

	evts := make([]json.RawMessage, len(req.Events))
	for n, item := range req.Events {
		evt := item.GetEvent()
		if eventName == nil {
			name := evt.Name
			eventName = &name
		}

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

	cfg := sv2.Config{
		FunctionVersion: req.Function.FunctionVersion,
		SpanID:          spanID.String(),
		EventIDs:        eventIDs,
		Idempotency:     key,
		ReplayID:        req.ReplayID,
		OriginalRunID:   req.OriginalRunID,
		PriorityFactor:  &factor,
		BatchID:         req.BatchID,
		Context:         req.Context,
		RequestVersion:  consts.RequestVersionUnknown,
	}
	if req.RequestVersion != nil {
		cfg.RequestVersion = *req.RequestVersion
	}

	config := *sv2.InitConfig(&cfg)

	// If we have a specifc URL to hit for this run, add it to context.
	if req.URL != "" {
		config.Context["url"] = req.URL
	}

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

	if req.DebugSessionID != nil {
		config.SetDebugSessionID(*req.DebugSessionID)
	}
	if req.DebugRunID != nil {
		config.SetDebugRunID(*req.DebugRunID)
	}

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

	bytEvts, err := json.Marshal(evts)
	if err != nil {
		return nil, fmt.Errorf("error marshalling events: %w", err)
	}

	strEvts := string(bytEvts)

	var (
		runSpanRef       *tracing.DroppableSpan
		discoverySpanRef *tracing.DroppableSpan
	)

	// Send spans to the history store (ClickHouse). If not called, we'll drop
	// the spans and not send them. There's a variety of scenarios where the run
	// ends up not scheduling so we don't want to add it to the history store.
	// Some scenarios are happy path (e.g.  queue idempotency) and some are sad
	// path (e.g. Executor borked)
	sendSpans := func() {
		if runSpanRef != nil {
			err := runSpanRef.Send()
			if err != nil {
				l.Error(
					"error sending run span",
					"error", err,
					"run_id", runID,
				)
			}
		}

		if discoverySpanRef != nil {
			err := discoverySpanRef.Send()
			if err != nil {
				l.Error(
					"error sending discovery span",
					"error", err,
					"run_id", runID,
				)
			}
		}
	}

	// Handle span dropping. The drops will be noops if the spans were sent
	defer func() {
		if runSpanRef != nil {
			runSpanRef.Drop()
		}

		if discoverySpanRef != nil {
			discoverySpanRef.Drop()
		}
	}()

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

	// Track skip reason and context for span attributes
	var skipReason enums.SkipReason
	var singletonSkipRunID *ulid.ULID

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
					if err != nil {
						l.ReportError(err, "error canceling singleton run")
					}
				default:
					// Mark as singleton skip - will be handled after span creation
					skipReason = enums.SkipReasonSingleton
					singletonSkipRunID = singletonRunID
				}
			}
			singletonConfig = &queue.Singleton{Key: singletonKey}
		case errors.Is(err, singleton.ErrEvaluatingSingletonExpression):
			// Ignore singleton expressions if we cannot evaluate them
			l.Warn("error evaluating singleton expression", "error", err)
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

	stv1ID := sv2.V1FromMetadata(metadata)

	// Check if the function should be skipped (paused, draining, backlog limit)
	// Only check if not already marked as skipped (e.g., by singleton)
	if skipReason == enums.SkipReasonNone {
		skipReason = e.skipped(ctx, req)
	}

	// Create run state if not skipped
	if skipReason == enums.SkipReasonNone {
		st, err := e.smv2.Create(ctx, newState)
		switch {
		case err == nil: // no-op
		case errors.Is(err, state.ErrIdentifierExists): // no-op
		case errors.Is(err, state.ErrIdentifierTombstone):
			return nil, ErrFunctionSkippedIdempotency
		default:
			return nil, fmt.Errorf("error creating run state: %w", err)
		}

		// Override existing identifier in case we changed the run ID due to idempotency
		stv1ID = sv2.V1FromMetadata(st.Metadata)

		// NOTE: if the runID mismatches, it means there's already a state available
		// and we need to override the one we already have to make sure we're using
		// the correct metedata values
		if metadata.ID.RunID != stv1ID.RunID {
			id := sv2.IDFromV1(stv1ID)
			metadata, err = e.smv2.LoadMetadata(ctx, id)
			if err != nil {
				return nil, err
			}
		}
	}

	runTimestamp := runID.Timestamp()
	runSpanOpts := &tracing.CreateSpanOptions{
		Debug:    &tracing.SpanDebugData{Location: "executor.Schedule"},
		Metadata: &metadata,
		Attributes: meta.NewAttrSet(
			meta.Attr(meta.Attrs.DebugSessionID, req.DebugSessionID),
			meta.Attr(meta.Attrs.DebugRunID, req.DebugRunID),
			meta.Attr(meta.Attrs.EventsInput, &strEvts),
			meta.Attr(meta.Attrs.TriggeringEventName, eventName),
			meta.Attr(meta.Attrs.QueuedAt, &runTimestamp),
		),
		Seed: []byte(metadata.ID.RunID[:]),
	}
	if req.RunMode == enums.RunModeSync {
		// XXX: If this is a sync run, always add the start time to the span. We do this
		// because sync runs have already started by the time we call Schedule; they're
		// in-process, and Schedule gets called via an API endpoint when the run starts.
		time := runID.Timestamp()
		runSpanOpts.StartTime = time
		meta.AddAttr(runSpanOpts.Attributes, meta.Attrs.StartedAt, &time)

		// Mark this as a Durable Endpoint run
		isDurableEndpointRun := true
		meta.AddAttr(runSpanOpts.Attributes, meta.Attrs.IsDurableEndpointRun, &isDurableEndpointRun)
	}

	status := enums.StepStatusQueued
	if skipReason != enums.SkipReasonNone {
		status = enums.StepStatusSkipped
	} else if req.RunMode == enums.RunModeSync {
		// Sync runs are already executing by the time Schedule is called, so
		// mark as Running instead of Queued.
		status = enums.StepStatusRunning
	}

	// Always add either queued or skipped as a status.
	meta.AddAttr(
		runSpanOpts.Attributes,
		meta.Attrs.DynamicStatus,
		&status,
	)

	if skipReason != enums.SkipReasonNone {
		meta.AddAttr(runSpanOpts.Attributes, meta.Attrs.SkipReason, &skipReason)
		if singletonSkipRunID != nil {
			existingRunID := singletonSkipRunID.String()
			meta.AddAttr(runSpanOpts.Attributes, meta.Attrs.SkipExistingRunID, &existingRunID)
		}
	}

	// Always the root span.
	runSpanRef, err = e.tracerProvider.CreateDroppableSpan(
		ctx,
		meta.SpanNameRun,
		runSpanOpts,
	)
	if err != nil {
		// return nil, fmt.Errorf("error creating run span: %w", err)
		l.Debug("error creating run span", "error", err)
	}

	// If the function is being skipped, send spans and handle skip.
	if skipReason != enums.SkipReasonNone {
		sendSpans()
		return e.handleFunctionSkipped(ctx, req, metadata, evts, skipReason)
	}

	if req.BatchID == nil {

		// Create cancellation pauses immediately, only if this is a non-batch event.
		if len(req.Function.Cancel) > 0 {
			if err := e.createCancellationPauses(ctx, l, key, evtMap, metadata.ID, req); err != nil {
				return &metadata, err
			}
		}

		// Add a system job to eager-cancel this function run on timeouts, only if this is a non-batch event.
		if req.Function.Timeouts != nil && req.Function.Timeouts.Start != nil {
			enqueuedAt := ulid.Time(runID.Time())
			if err := e.createEagerCancellationForTimeout(ctx, enqueuedAt, req.Function.Timeouts.StartDuration(), enums.CancellationKindStartTimeout, stv1ID); err != nil {
				return &metadata, err
			}
		}
	}

	at := e.now()
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
		WorkspaceID:           stv1ID.WorkspaceID,
		Kind:                  queue.KindStart,
		Identifier:            stv1ID,
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

	if runSpanRef != nil {
		// We also create the first discovery step right now, as then every single
		// queue item has a span to reference.
		//
		// Initially, this helps combat a situation whereby erroring calls within
		// the very first discovery step of a function are difficult to attribute
		// to the same step span across retries.
		//
		// In the future, this also means that we can remove some magic around
		// where to find the latest span and just always fetch it from the queue
		// item.
		discoverySpanRef, err = e.tracerProvider.CreateDroppableSpan(
			ctx,
			meta.SpanNameStepDiscovery,
			&tracing.CreateSpanOptions{
				Debug:     &tracing.SpanDebugData{Location: "executor.Schedule"},
				Parent:    runSpanRef.Ref,
				Metadata:  &metadata,
				QueueItem: &item,
				Carriers:  []map[string]any{item.Metadata},
				Attributes: meta.NewAttrSet(
					meta.Attr(meta.Attrs.QueuedAt, &runTimestamp),
				),
			},
		)
		if err != nil {
			l.Debug("error creating initial step span", "error", err)
		}
	}

	// If this is run mode sync, we do NOT need to create a queue item, as the
	// Inngest SDK is checkpointing and the execution is happening in a single
	// external API request.
	if req.RunMode == enums.RunModeSync {
		sendSpans()
		for _, e := range e.lifecycles {
			go e.OnFunctionScheduled(context.WithoutCancel(ctx), metadata, item, req.Events)
		}
		return &metadata, nil
	}

	// Schedule for async functons (the default)
	err = e.queue.Enqueue(ctx, item, at, queue.EnqueueOpts{})

	switch err {
	case nil:
		// no-op
	case queue.ErrQueueItemExists:
		// If the item already exists in the queue, we can safely ignore this
		// entire schedule request; it's basically a retry and we should not
		// persist this for the user.
		return nil, state.ErrIdentifierExists

	case queue.ErrQueueItemSingletonExists:
		err := e.smv2.Delete(ctx, sv2.IDFromV1(stv1ID))
		if err != nil {
			l.ReportError(err, "error deleting function state")
		}
		return nil, ErrFunctionSkipped

	default:
		return nil, fmt.Errorf("error enqueueing source edge '%v': %w", queueKey, err)
	}

	sendSpans()
	for _, e := range e.lifecycles {
		go e.OnFunctionScheduled(context.WithoutCancel(ctx), metadata, item, req.Events)
	}

	return &metadata, nil
}

func (e *executor) handleFunctionSkipped(ctx context.Context, req execution.ScheduleRequest, metadata sv2.Metadata, evts []json.RawMessage, reason enums.SkipReason) (*sv2.Metadata, error) {
	for _, e := range e.lifecycles {
		service.Go(
			func() {
				e.OnFunctionSkipped(context.WithoutCancel(ctx), metadata, execution.SkipState{
					CronSchedule: req.Events[0].GetEvent().CronSchedule(),
					Reason:       reason,
					Events:       evts,
				})
			})
	}
	return nil, ErrFunctionSkipped
}

// Execute loads a workflow and the current run state, then executes the
// function's step via the necessary driver.
func (e *executor) Execute(ctx context.Context, id state.Identifier, item queue.Item, edge inngest.Edge) (*state.DriverResponse, error) {
	// Immediately store execution context for tracing.
	ctx = tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
		Identifier:  sv2.IDFromV1(id),
		Attempt:     item.Attempt,
		MaxAttempts: item.MaxAttempts,
		QueueKind:   item.Kind,
	})

	if e.fl == nil {
		return nil, fmt.Errorf("no function loader specified running step")
	}

	l := e.log.With(
		"account_id", item.Identifier.AccountID,
		"env_id", item.WorkspaceID,
		"app_id", item.Identifier.AppID,
		"fn_id", item.Identifier.WorkflowID,
		"run_id", id.RunID,
	)
	ctx = logger.WithStdlib(ctx, l)

	// If this is of type sleep, ensure that we save "nil" within the state store
	// for the outgoing edge ID.  This ensures that we properly increase the stack
	// for `tools.sleep` within generator functions.
	//
	// This also marks the sleep item as completed.
	isSleepResume := item.Kind == queue.KindSleep && item.Attempt == 0
	if isSleepResume {
		err := e.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
			EndTime:    e.now(),
			Debug:      &tracing.SpanDebugData{Location: "executor.SleepResume"},
			QueueItem:  &item,
			Status:     enums.StepStatusCompleted,
			TargetSpan: tracing.SpanRefFromQueueItem(&item),
		})
		if err != nil {
			l.Debug("error updating sleep resume span", "error", err)
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
		if !shouldEnqueueDiscovery(hasPendingSteps, item.ParallelMode) {
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

	// Start cancellation check
	cancelled, err := e.checkCancellation(ctx, md, events)
	if err != nil {
		return nil, fmt.Errorf("could not check cancellation: %w", err)
	}

	//
	// record function start time using the same method as step started,
	// ensures ui timeline alignment
	start, ok := queue.GetItemStart(ctx)
	if !ok {
		start = e.now()
	}

	if md.Config.StartedAt.IsZero() {
		md.Config.StartedAt = start

		// Add a system job to eager-cancel this function run on timeouts
		if ef.Function.Timeouts != nil && ef.Function.Timeouts.Finish != nil {
			if err := e.createEagerCancellationForTimeout(ctx, start, ef.Function.Timeouts.FinishDuration(), enums.CancellationKindFinishTimeout, id); err != nil {
				return nil, err
			}
		}
	}

	if cancelled || v.stopWithoutRetry {
		// Validation prevented execution and doesn't want the executor to retry, so
		// don't return an error - assume the function finishes and delete state.
		err := e.smv2.Delete(ctx, md.ID)
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
	runSpanRef := tracing.RunSpanRefFromMetadata(&md)
	parentRef := e.getParentSpan(ctx, item, md)

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

		if md.Config.RequestVersion == 0 {
			// The intent of this is to ensure that the 1st request received by
			// the SDK does not have a request version of 0. This fixes an issue
			// caused by a zero value when initializing state.
			//
			// If the SDK receives a request version of 0 in the 1st request
			// then it'll be "stuck" on 0 for the life of the run.
			//
			// Don't put this override within the `item.Attempt == 0` check,
			// just in case we both fail to update metadata and the attempt
			// errors
			md.Config.RequestVersion = consts.RequestVersionUnknown
		}

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
				l.ReportError(err, "error updating metadata on function start")
			}

			// Set some run span details to be explicit that this has been
			// kicked off
			if err := e.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
				Debug:      &tracing.SpanDebugData{Location: "executor.ExecuteTrigger"},
				QueueItem:  &item,
				Metadata:   &md,
				Status:     enums.StepStatusRunning,
				TargetSpan: runSpanRef,
				Attributes: meta.NewAttrSet(
					meta.Attr(meta.Attrs.StartedAt, &md.Config.StartedAt),
				),
			}); err != nil {
				l.ReportError(err, "error updating run span on function start")
			}

			for _, e := range e.lifecycles {
				go e.OnFunctionStarted(context.WithoutCancel(ctx), md, item, events)
			}
		}
	}

	// Organize the run instance.
	instance := runInstance{
		md:         md,
		f:          *ef.Function,
		events:     events,
		item:       item,
		edge:       edge,
		stackIndex: stackIndex,
		httpClient: e.httpClient,
		parentSpan: parentRef,
		c:          e.clock,
		start:      start,
	}

	// This span will be updated with output as soon as execution finishes.
	instance.execSpan, err = e.tracerProvider.CreateSpan(
		ctx,
		meta.SpanNameExecution,
		&tracing.CreateSpanOptions{
			Debug:      &tracing.SpanDebugData{Location: "executor.ExecutePre"},
			Parent:     parentRef,
			Metadata:   &md,
			QueueItem:  &item,
			Attributes: tracing.FunctionAttrs(&instance.f),
			StartTime:  e.now(),
		},
	)
	if err != nil {
		// return nil, fmt.Errorf("error creating execution span: %w", err)
		l.Debug("error creating execution span", "error", err)
	}

	return util.CritT(ctx, "run step", func(ctx context.Context) (*state.DriverResponse, error) {
		// Track how long it took us from the queue item job starting -> calling run.
		instance.trackLatencyHistogram(ctx, "queue_to_run_start", nil)
		resp, err := e.run(ctx, &instance)
		instance.trackLatencyHistogram(ctx, "run_start_to_request_end", nil)

		defer func() {
			// track how long it takes to finish accounting after running.
			instance.trackLatencyHistogram(ctx, "request_end_to_finalize", map[string]any{
				"error": err == nil,
			})
		}()

		// XX: This is going to drop any sleep requests, because DriverResponseAttrs
		// forces the drop field if resp.IsDiscoveryResponse() is true.
		updateOpts := &tracing.UpdateSpanOptions{
			Debug:      &tracing.SpanDebugData{Location: "executor.ExecutePost"},
			Metadata:   &md,
			QueueItem:  &item,
			TargetSpan: instance.execSpan,
			Attributes: tracing.DriverResponseAttrs(resp, nil),
		}

		// For most executions, we now set the status of the execution span.
		// For some responses, however, the execution as the user sees it is
		// still ongoing. Account for that here.
		if !resp.IsGatewayRequest() {
			updateOpts.EndTime = e.now()

			status := enums.StepStatusCompleted
			if err != nil || resp.Err != nil || resp.UserError != nil {
				status = enums.StepStatusFailed
			}
			updateOpts.Status = status
		}

		_ = e.tracerProvider.UpdateSpan(ctx, updateOpts)

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

		if e.allowStepMetadata.Enabled(ctx, instance.Metadata().ID.Tenant.AccountID) {
			l := l.With("step_metadata", true)
			for _, opcode := range resp.Generator {
				for _, md := range opcode.Metadata {
					if err := md.Validate(); err != nil {
						l.Warn("invalid metadata in driver response", "error", err)
						continue
					}

					// TODO: validate that specific kinds are allowed to be set by the user and check account-level metadata
					// limits.
					_, err := e.createMetadataSpan(
						ctx,
						&instance,
						"executor.ExecutePostMetadata",
						md,
						md.Scope,
					)
					if err != nil {
						l.Warn("error creating metadata span", "error", err)
					}
				}
			}

			// Extract HTTP timing metadata from httpstat if available.
			// This captures the detailed connection timing breakdown (DNS, TCP, TLS, TTFB, transfer)
			// from the HTTP request to the user's SDK function.
			if resp.HTTPStat != nil {
				httpTimingMd := extractors.ExtractHTTPTimingMetadata(resp.HTTPStat)
				_, err := e.createMetadataSpan(
					ctx,
					&instance,
					"executor.httpTiming",
					httpTimingMd,
					enums.MetadataScopeStepAttempt,
				)
				if err != nil {
					l.Warn("error creating HTTP timing metadata span", "error", err)
				}
			}
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

	if i.resp.Err == nil && i.resp.IsOpResponse() {
		// Handle generator op responses then return.
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

				if err := e.Finalize(ctx, execution.FinalizeOpts{
					Metadata: i.md,
					// Always, when called from the executor, as this handles async
					// finalization.
					Response: execution.FinalizeResponse{
						Type:           execution.FinalizeResponseDriver,
						DriverResponse: *i.resp,
					},
					Optional: execution.FinalizeOptional{
						FnSlug:        i.f.GetSlug(),
						InputEvents:   i.events,
						OutputSpanRef: i.execSpan,
						Reason:        "fail-early",
					},
				}); err != nil {
					l.ReportError(err, "error running finish handler")
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

		if err := e.Finalize(ctx, execution.FinalizeOpts{
			Metadata: i.md,
			// Always, when called from the executor, as this handles async
			// finalization.
			Response: execution.FinalizeResponse{
				Type:           execution.FinalizeResponseDriver,
				DriverResponse: *i.resp,
			},
			Optional: execution.FinalizeOptional{
				FnSlug:        i.f.GetSlug(),
				InputEvents:   i.events,
				OutputSpanRef: i.execSpan,
				Reason:        "resp-err",
			},
		}); err != nil {
			l.ReportError(err, "error running finish handler")
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
		// This is the function result.
		if err := e.Finalize(ctx, execution.FinalizeOpts{
			Metadata: i.md,
			// Always, when called from the executor, as this handles async
			// finalization.
			Response: execution.FinalizeResponse{
				Type:           execution.FinalizeResponseDriver,
				DriverResponse: *i.resp,
			},
			Optional: execution.FinalizeOptional{
				FnSlug:        i.f.GetSlug(),
				InputEvents:   i.events,
				OutputSpanRef: i.execSpan,
				Reason:        "opcode-none",
			},
		}); err != nil {
			l.ReportError(err, "error running finish handler")
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

func (f *functionFinishedData) setResponse(resp execution.FinalizeResponse) {
	switch resp.Type {

	case execution.FinalizeResponseRunComplete:
		// NOTE: This should never be wrapped with a `{"data":T}` field because
		// run complete is always raw data.
		f.Result = resp.RunComplete.Data

	case execution.FinalizeResponseAPI:
		f.Result = resp.APIResponse

	case execution.FinalizeResponseDriver:
		r := resp.DriverResponse
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
}

func (f functionFinishedData) Map() map[string]any {
	s := structs.New(f)
	s.TagName = "json"
	return s.Map()
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

func (e *executor) checkCancellation(ctx context.Context, md sv2.Metadata, evts []json.RawMessage) (bool, error) {
	// If no cancellation checker was provided, assume run should not be cancelled
	if e.cancellationChecker == nil {
		return false, nil
	}

	start := time.Now()

	l := logger.StdlibLogger(ctx).With(
		"run_id", md.ID.RunID,
		"function_id", md.ID.FunctionID,
		"workspace_id", md.ID.Tenant.EnvID,
	)
	evt := event.Event{}
	if err := json.Unmarshal(evts[0], &evt); err != nil {
		return false, fmt.Errorf("error decoding input event in cancellation checker: %w", err)
	}

	// Wait for result to be available within deadline and return, or continue processing asynchronously
	deadline := 100 * time.Millisecond

	// Create buffered channel to allow sending even without receiver
	// but block receive until message is ready
	done := make(chan bool, 1)

	// Ensure this completes before we shut down the service
	service.Go(func() {
		defer func() {
			metrics.HistogramCancellationCheckDuration(ctx, time.Since(start), metrics.HistogramOpt{
				PkgName: pkgName,
			})
		}()

		cancel, err := e.cancellationChecker.IsCancelled(
			ctx,
			md.ID.Tenant.EnvID,
			md.ID.FunctionID,
			md.ID.RunID,
			evt.Map(),
		)
		if err != nil {
			if errors.Is(err, &expressions.CompileError{}) {
				l.Warn("invalid cancellation expression", "error", err.Error())
			} else {
				l.Error("error checking cancellation", "error", err.Error())
			}
		}
		if cancel != nil {
			err = e.Cancel(ctx, md.ID, execution.CancelRequest{
				CancellationID: &cancel.ID,
			})
			if err != nil {
				l.ReportError(err, "failed to cancel run after checking cancellation")
			}

			done <- true
			return
		}

		done <- false
	})

	select {
	// Wait for result to be available
	case cancelled := <-done:
		return cancelled, nil
		// Or continue processing after hitting deadline
	case <-e.clock.After(deadline):
		l.Debug("continuing cancellation check in background")
		metrics.IncrAsyncCancellationCheckCounter(ctx, 1, metrics.CounterOpt{
			PkgName: pkgName,
		})
		return false, nil
	}
}

// run executes the step with the given step ID.
//
// A nil response with an error indicates that an internal error occurred and the step
// did not run.
func (e *executor) run(ctx context.Context, i *runInstance) (*state.DriverResponse, error) {
	endpoint := i.f.URI()

	// XXX: If we have a URI in the run metadata, use it.
	//
	// This allows us to override URIs on a per-run bases, useful if the URI has
	// identifiers in it which change between runs.
	//
	// This is only used for V2 based drivers, specifically for sync REST-based endpoints.
	if uri, ok := i.md.Config.Context["url"].(string); ok && len(uri) > 0 {
		if parsed, _ := url.Parse(uri); parsed != nil {
			endpoint = parsed
		}
	}

	for _, e := range e.lifecycles {
		go e.OnStepStarted(context.WithoutCancel(ctx), i.md, i.item, i.edge, endpoint.String())
	}

	switch d := e.fnDriver(ctx, i.f).(type) {
	case driver.DriverV2:
		return e.executeDriverV2(ctx, i, d, endpoint.String())
	case driver.DriverV1:
		{
			// Execute the actual step using V1 drivers.  The V1 driver embeds errors in driver
			// response and has generally difficult error management.
			response, err := e.executeDriverV1(ctx, i)
			if response.Err != nil && err == nil {
				// This step errored, so always return an error.
				return response, fmt.Errorf("%s", *response.Err)
			}
			return response, err
		}
	default:
		return nil, fmt.Errorf("%w: '%s'", ErrNoRuntimeDriver, inngest.Driver(i.f))
	}
}

func (e *executor) executeDriverV2(ctx context.Context, run *runInstance, d driver.DriverV2, url string) (*state.DriverResponse, error) {
	var sk []byte

	if e.signingKeyLoader != nil {
		var err error
		if sk, err = e.signingKeyLoader(ctx, run.Metadata().ID.Tenant.EnvID); err != nil {
			return nil, fmt.Errorf("error loading environment from ID: %w", err)
		}
	}

	// Use IncomingGeneratorStep if set, otherwise fall back to Incoming
	stepID := run.edge.IncomingGeneratorStep
	if stepID == "" {
		stepID = run.edge.Incoming
	}

	resp, uerr, ierr := d.Do(ctx, e.smv2, driver.V2RequestOpts{
		Metadata:   *run.Metadata(),
		Fn:         run.f,
		SigningKey: sk,
		Attempt:    run.AttemptCount(),
		Index:      run.stackIndex,
		StepID:     &stepID,
		QueueRef:   queueref.StringFromCtx(ctx),
		URL:        url,
	})

	// For now, the executor expects V1 style errors directly in state.DriverResponse.
	// We move all UserErrors into state.DriverResponse, and always return a response...
	// until we refactor the executor to handle (Option<Response>, UserError, InternalError).
	if resp == nil {
		resp = &state.DriverResponse{}
	}

	if uerr != nil {
		resp.Output = uerr.Raw()
		resp.OutputSize = len(resp.Output.([]byte))
	}

	if ierr != nil {
		str := ierr.Error()
		resp.Err = &str
	}

	return resp, ierr
}

// executeDriverV1 runs the enqueued step by invoking the driver.  It also inspects
// and normalizes responses (eg. max retry attempts).
func (e *executor) executeDriverV1(ctx context.Context, i *runInstance) (*state.DriverResponse, error) {
	driverName := inngest.Driver(i.f)

	d, ok := e.driverv1[driverName]
	if !ok {
		return nil, fmt.Errorf("%w: '%s'", ErrNoRuntimeDriver, driverName)
	}

	step := &i.f.Steps[0]

	if i.execSpan != nil {
		// Allow deep driver code to grab the execution span from context
		ctx = i.execSpan.SetToCtx(ctx)
	}

	response, err := d.Execute(ctx, e.smv2, i.md, i.item, i.edge, *step, i.stackIndex, i.item.Attempt)

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
			}

			// check for connect worker capacity errors after updating the UI response
			if state.IsConnectWorkerAtCapacityCode(serr.Code) {
				err = queue.AlwaysRetryError(state.ErrConnectWorkerCapacity)
				gracefulErr.Message = "All workers are at capacity"
				gracefulErr.Stack = fmt.Sprintf("%s\n%s\n%s", serr.Message, "This is a retryable error", "The executor will retry again")
			}

			// serialize error
			gracefulErrSerialized := gracefulErr.Serialize(execution.StateErrorKey)
			response.Output = gracefulErrSerialized
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
	l := e.log.With(
		"workspace_id", evt.GetWorkspaceID(),
		"event_id", evt.GetInternalID(),
	)

	idx := pauses.Index{
		WorkspaceID: evt.GetWorkspaceID(),
		EventName:   evt.GetEvent().Name,
	}

	if bufferCount, _ := e.pm.BufferLen(ctx, idx); bufferCount > 0 {
		// Log the total number of items in the buffer at any point.
		l = l.With("buffer_count", bufferCount)
	}

	aggregated, err := e.pm.Aggregated(
		ctx,
		idx,
		consts.AggregatePauseThreshold,
	)
	if err != nil {
		l.ReportError(err, "error checking pause aggregation")
	}

	// Use the aggregator for all funciton finished events, if there are more than
	// 50 waiting.  It only takes a few milliseconds to iterate and handle less
	// than 50;  anything more runs the risk of running slow.
	if aggregated {
		aggRes, err := e.handleAggregatePauses(ctx, evt)
		if err != nil {
			l.ReportError(err, "error handling aggregate pauses")
		}
		return aggRes, err
	}

	iter, err := e.pm.PausesSince(ctx, idx, time.Time{})
	if err != nil {
		return execution.HandlePauseResult{}, fmt.Errorf("error loading pause iterator: %w", err)
	}

	res, err := e.handlePausesAllNaively(ctx, iter, evt)
	if err != nil {
		l.ReportError(err, "error handling naive pauses")
	}
	return res, err
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
					l.Warn("error compiling pause expression", "error", err)
					return
				}

				val, err := expr.Evaluate(ctx, data)
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
		if pause.Expires.Time().Before(e.now()) {
			l.Debug("encountered expired pause")

			shouldDelete := pause.Expires.Time().Add(consts.PauseExpiredDeletionGracePeriod).Before(e.now())
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
				errors.Is(err, state.ErrEventNotFound) ||
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
			_, _, err = e.pm.ConsumePause(context.Background(), e.smv2, *pause, state.ConsumePauseOpts{
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
	correlationID := evt.GetEvent().CorrelationID()
	if correlationID == "" {
		return ErrNoCorrelationID
	}

	var (
		evtID = evt.GetInternalID()
		wsID  = evt.GetWorkspaceID()
		l     = e.log.With("event_id", evtID.String())

		eventName string
	)

	// find the pause with correlationID
	pause, err := e.pm.PauseByInvokeCorrelationID(ctx, wsID, correlationID)
	if err != nil {
		return err
	}
	if pause.Event != nil {
		eventName = *pause.Event
	}

	if pause.Expires.Time().Before(e.now()) {
		l.Debug("expired pause resuming invoke")

		shouldDelete := pause.Expires.Time().Add(consts.PauseExpiredDeletionGracePeriod).Before(e.now())
		if shouldDelete {
			// Consume this pause to remove it entirely
			l.Debug("deleting expired pause")
			_ = e.pm.Delete(context.Background(), pauses.Index{WorkspaceID: pause.WorkspaceID, EventName: eventName}, *pause)
		}
		return nil
	}

	l.DebugSample(10, "resuming pause from invoke", "pause.DataKey", pause.DataKey)

	resumeData := pause.GetResumeData(evt.GetEvent())
	return e.Resume(ctx, *pause, execution.ResumeRequest{
		With:           resumeData.With,
		EventID:        &evtID,
		EventName:      evt.GetEvent().Name,
		RunID:          resumeData.RunID,
		StepName:       resumeData.StepName,
		IdempotencyKey: correlationID,
	})
}

// Cancel cancels an in-progress function.
func (e *executor) Cancel(ctx context.Context, id sv2.ID, r execution.CancelRequest) error {
	l := e.log.With(
		"run_id", id.RunID.String(),
		"workflow_id", id.FunctionID.String(),
	)

	md, err := e.smv2.LoadMetadata(ctx, id)
	if err == sv2.ErrMetadataNotFound || errors.Is(err, state.ErrRunNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to load run: %w", err)
	}

	ctx = tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
		Identifier: md.ID,
		Attempt:    0,
	})

	// We need events to finalize the function.
	evts, err := e.smv2.LoadEvents(ctx, id)
	if errors.Is(err, state.ErrEventNotFound) {
		// If the event has gone, another thread cancelled the function.
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to load run events: %w", err)
	}

	// We need the function slug.
	f, err := e.fl.LoadFunction(ctx, md.ID.Tenant.EnvID, md.ID.FunctionID)
	if err != nil {
		return fmt.Errorf("unable to load function: %w", err)
	}

	if err := e.Finalize(ctx, execution.FinalizeOpts{
		Metadata: md,
		// Always, when called from the executor, as this handles async
		// finalization.
		Response: execution.FinalizeResponse{
			Type:           execution.FinalizeResponseDriver,
			DriverResponse: state.DriverResponse{}, // empty.
		},
		Optional: execution.FinalizeOptional{
			FnSlug:      f.Function.GetSlug(),
			InputEvents: evts,
			Cancel:      true,
			Reason:      "cancel",
		},
	}); err != nil {
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

	// Immediately store execution context for tracing.
	ctx = tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
		Identifier:  md.ID,
		Attempt:     0,
		MaxAttempts: pause.MaxAttempts,
	})

	data, err := json.Marshal(r.With)
	if err != nil {
		return fmt.Errorf("error marshalling timeout step data: %w", err)
	}

	e.log.Debug("resuming from timeout ", "identifier", id)

	hasPendingSteps, err := e.smv2.SaveStep(ctx, id, pause.DataKey, data)
	if errors.Is(err, state.ErrDuplicateResponse) {
		// cannot resume as the pause has already been resumed and consumed.
		return nil
	}
	if err != nil && !errors.Is(err, state.ErrIdempotentResponse) {
		// This is a non-idempotent error, so there was a legitimate error saving the response.
		e.log.Error("error saving timeout step", "error", err, "identifier", id)
		return err
	}

	pauseSpan := tracing.SpanRefFromPause(&pause)
	_ = e.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
		EndTime:    e.now(),
		Debug:      &tracing.SpanDebugData{Location: "executor.ResumePauseTimeout"},
		Status:     enums.StepStatusTimedOut,
		TargetSpan: pauseSpan,
		Attributes: tracing.ResumeAttrs(&pause, &r),
	})

	if shouldEnqueueDiscovery(hasPendingSteps, pause.ParallelMode) {
		// If there are no parallel steps ongoing, we must enqueue the next SDK ping to continue on with
		// execution.
		jobID := fmt.Sprintf("%s-%s-timeout", md.IdempotencyKey(), pause.DataKey)
		nextItem := queue.Item{
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
			Metadata: make(map[string]any),
		}

		nextStepSpan, err := e.tracerProvider.CreateDroppableSpan(
			ctx,
			meta.SpanNameStepDiscovery,
			&tracing.CreateSpanOptions{
				Carriers:    []map[string]any{nextItem.Metadata},
				FollowsFrom: pauseSpan,
				Debug:       &tracing.SpanDebugData{Location: "executor.ResumePauseTimeout"},
				Metadata:    &md,
				Parent:      tracing.RunSpanRefFromMetadata(&md),
				QueueItem:   &nextItem,
			},
		)
		if err != nil {
			// return fmt.Errorf("error creating span for next step after
			// resume timeout: %w", err)
			e.log.Error("error creating span for next step after resume timeout", "error", err)
		}

		err = e.queue.Enqueue(ctx, nextItem, e.now(), queue.EnqueueOpts{})
		if err != nil {
			if errors.Is(err, queue.ErrQueueItemExists) {
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

	sv2id := sv2.ID{
		RunID:      pause.Identifier.RunID,
		FunctionID: pause.Identifier.FunctionID,
		Tenant: sv2.Tenant{
			EnvID:     pause.WorkspaceID,
			AccountID: pause.Identifier.AccountID,
			// NOTE: Pauses do not store app IDs.
		},
	}

	// Immediately store execution context for tracing.
	ctx = tracing.WithExecutionContext(ctx, tracing.ExecutionContext{
		Identifier:  sv2id,
		Attempt:     0,
		MaxAttempts: pause.MaxAttempts,
	})

	md, err := e.smv2.LoadMetadata(ctx, sv2id)
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
			_, cleanup, err := e.pm.ConsumePause(ctx, e.smv2, pause, state.ConsumePauseOpts{
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

		consumeResult, cleanup, err := e.pm.ConsumePause(ctx, e.smv2, pause, state.ConsumePauseOpts{
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
		_ = e.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
			EndTime:    e.now(),
			Debug:      &tracing.SpanDebugData{Location: "executor.Resume"},
			Status:     status,
			TargetSpan: pauseSpan,
			Attributes: tracing.ResumeAttrs(&pause, &r),
		})

		if shouldEnqueueDiscovery(consumeResult.HasPendingSteps, pause.ParallelMode) {
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
				ctx,
				meta.SpanNameStepDiscovery,
				&tracing.CreateSpanOptions{
					Carriers:    []map[string]any{nextItem.Metadata},
					FollowsFrom: pauseSpan,
					Debug:       &tracing.SpanDebugData{Location: "executor.Resume"},
					Metadata:    &md,
					Parent:      tracing.RunSpanRefFromMetadata(&md),
					QueueItem:   &nextItem,
				},
			)
			if err != nil {
				// return fmt.Errorf("error creating span for next step
				// after resume: %w", err)
				e.log.Debug("error creating span for next step after resume", "error", err)
			}

			err = e.queue.Enqueue(ctx, nextItem, e.now(), queue.EnqueueOpts{})
			if err != nil {
				if err == queue.ErrQueueItemExists {
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
		if q, ok := e.queue.(queue.QueueManager); ok {
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
				if errors.Is(err, queue.ErrQueueItemNotFound) {
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

	// NOTE: Before checkpointing, we could never have a slice of opcodes with len(1)
	// which contained a step.run.  However, with checkpointing we can batch step.run
	// outputs into one single HTTP response.
	//
	// When this happens, we ALWAYS need to create a trace for each step.
	//
	// We pass this down in context, unfortunately.
	if len(resp.Generator) > 1 {
		for _, op := range resp.Generator {
			if op.Op == enums.OpcodeStepRun {
				ctx = setEmitCheckpointTraces(ctx)
				break
			}
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
			// parallel step individually.
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
			return e.HandleGenerator(ctx, i, copied)
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

func (e *executor) HandleGenerator(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode) error {
	// Grab the edge that triggered this step execution.
	lifecycleItem := runCtx.LifecycleItem()
	edge, ok := lifecycleItem.Payload.(queue.PayloadEdge)
	if !ok {
		return fmt.Errorf("unknown queue item type handling generator: %T", lifecycleItem.Payload)
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
		return e.handleGeneratorStep(ctx, runCtx, gen, edge)
	case enums.OpcodeStepError:
		return e.handleStepError(ctx, runCtx, gen, edge)
	case enums.OpcodeStepPlanned:
		return e.handleGeneratorStepPlanned(ctx, runCtx, gen, edge)
	case enums.OpcodeSleep:
		return e.handleGeneratorSleep(ctx, runCtx, gen, edge)
	case enums.OpcodeWaitForEvent:
		return e.handleGeneratorWaitForEvent(ctx, runCtx, gen, edge)
	case enums.OpcodeInvokeFunction:
		return e.handleGeneratorInvokeFunction(ctx, runCtx, gen, edge)
	case enums.OpcodeAIGateway:
		return e.handleGeneratorAIGateway(ctx, runCtx, gen, edge)
	case enums.OpcodeGateway:
		return e.handleGeneratorGateway(ctx, runCtx, gen, edge)
	case enums.OpcodeWaitForSignal:
		return e.handleGeneratorWaitForSignal(ctx, runCtx, gen, edge)
	case enums.OpcodeRunComplete:
		return e.handleGeneratorFunctionFinished(ctx, runCtx, gen, edge)
	case enums.OpcodeSyncRunComplete:
		// This is an API-based function executed synchronously that had
		// an async conversion.  The result must always be in the shape of
		// apiresult.APIResult
		return e.handleGeneratorSyncFunctionFinished(ctx, runCtx, gen, edge)
	case enums.OpcodeStepFailed:
		return e.handleStepFailed(ctx, runCtx, gen, edge)
	case enums.OpcodeDiscoveryRequest:
		return e.handleGeneratorDiscoveryRequest(ctx, runCtx, gen, edge)
	}

	return fmt.Errorf("unknown opcode: %s", gen.Op)
}

func (e *executor) maybeEnqueueDiscoveryStep(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge, groupID string, hasPendingSteps bool) error {
	// Enqueue the next discovery step to continue execution.
	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Going from the current step
		Incoming: edge.Edge.Incoming, // And re-calling the incoming function in a loop
	}

	now := e.now()
	jobID := fmt.Sprintf("%s-%s", runCtx.Metadata().IdempotencyKey(), gen.ID)

	nextItem := queue.Item{
		JobID:                 &jobID,
		WorkspaceID:           runCtx.Metadata().ID.Tenant.EnvID,
		GroupID:               groupID,
		Kind:                  queue.KindEdge,
		Identifier:            sv2.V1FromMetadata(*runCtx.Metadata()), // Convert from v2 metadata
		PriorityFactor:        runCtx.PriorityFactor(),
		CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
		Attempt:               0,
		MaxAttempts:           runCtx.MaxAttempts(),
		Payload:               queue.PayloadEdge{Edge: nextEdge},
		Metadata:              make(map[string]any),
		ParallelMode:          gen.ParallelMode(),
	}

	if shouldEnqueueDiscovery(hasPendingSteps, gen.ParallelMode()) {

		lifecycleItem := runCtx.LifecycleItem()
		metadata := runCtx.Metadata()
		span, err := e.tracerProvider.CreateDroppableSpan(
			ctx,
			meta.SpanNameStepDiscovery,
			&tracing.CreateSpanOptions{
				Carriers:    []map[string]any{nextItem.Metadata},
				FollowsFrom: tracing.SpanRefFromQueueItem(&lifecycleItem),
				Debug:       &tracing.SpanDebugData{Location: "executor.maybeEnqueueDiscoveryStep"},
				Metadata:    metadata,
				Parent:      tracing.RunSpanRefFromMetadata(metadata),
				QueueItem:   &nextItem,
			},
		)
		if err != nil {
			// return fmt.Errorf("error creating span for next step after
			// Step: %w", err)
			e.log.Debug("error creating span for next step after Step", "error", err)
		}

		err = e.queue.Enqueue(ctx, nextItem, now, queue.EnqueueOpts{})
		if err != nil {
			span.Drop()

			if err == queue.ErrQueueItemExists {
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
		go l.OnStepScheduled(ctx, *runCtx.Metadata(), nextItem, stepName)
	}

	return nil
}

// handleGeneratorDiscoveryRequest handles OpcodeDiscoveryRequest, which
// indicates that the SDK is requesting new work to be scheduled, typically
// after checkpointing or in an effort to recover from non-determinism.
func (e *executor) handleGeneratorDiscoveryRequest(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	// Currently we always enqueue based off of this request, but in the future
	// we should fetch `hasPendingSteps` without saving state and use that to
	// decide whether to enqueue, as that takes in to account execution
	// versions and parallel steps.
	groupID := uuid.New().String()

	return e.maybeEnqueueDiscoveryStep(
		state.WithGroupID(ctx, groupID),
		runCtx,
		gen,
		edge,
		groupID,
		false,
	)
}

// handleGeneratorStep handles OpcodeStep and OpcodeStepRun, both indicating that a function step
// has finished
func (e *executor) handleGeneratorStep(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	// Save the response to the state store.
	output, err := gen.Output()
	if err != nil {
		return err
	}

	if err := e.validateStateSize(len(output), *runCtx.Metadata()); err != nil {
		return err
	}

	hasPendingSteps, err := e.smv2.SaveStep(ctx, runCtx.Metadata().ID, gen.ID, []byte(output))
	if errors.Is(err, state.ErrDuplicateResponse) || errors.Is(err, state.ErrIdempotentResponse) {
		// This is fine.
		// XXX: we should totally attach a warning to the function run here.
		return nil
	}

	if err != nil {
		return err
	}

	// Steps can be batched with checkpointing!  Imagine an SDK that opts into checkpointing,
	// then returned as an async response because the checkpooint batch time was greater than
	// the run execution.  In this case, all opcodes are returned to the executor via the async
	// response, and we have to retroactively save traces for each step.
	//
	// Again, we ONLY create new traces if the steps were batched, otherwise the standard
	// trace -> exec -> cleanup flow handles individual steps.
	//
	// In this case, we MUST retroactively record spans for each past step.
	//
	// XXX: (feat: checkpoint) We also only want to enqueue one discovery step per request,
	// if this isn't in parallelism.
	if emitCheckpointTraces(ctx) {
		attrs := tracing.GeneratorAttrs(&gen)
		tracing.AddMetadataTenantAttrs(attrs, runCtx.Metadata().ID)
		_, err := e.tracerProvider.CreateSpan(
			ctx,
			meta.SpanNameStep,
			&tracing.CreateSpanOptions{
				Seed:      []byte(gen.ID + gen.Timing.String()),
				Parent:    tracing.RunSpanRefFromMetadata(runCtx.Metadata()),
				StartTime: gen.Timing.Start(),
				EndTime:   gen.Timing.End(),
				Attributes: attrs.Merge(
					meta.NewAttrSet(
						meta.Attr(meta.Attrs.StepName, inngestgo.Ptr(gen.UserDefinedName())),
						meta.Attr(meta.Attrs.RunID, &runCtx.Metadata().ID.RunID),
						meta.Attr(meta.Attrs.QueuedAt, inngestgo.Ptr(gen.Timing.Start())),
						meta.Attr(meta.Attrs.StartedAt, inngestgo.Ptr(gen.Timing.Start())),
						meta.Attr(meta.Attrs.EndedAt, inngestgo.Ptr(gen.Timing.End())),
						meta.Attr(meta.Attrs.DynamicStatus, inngestgo.Ptr(enums.StepStatusCompleted)),
					),
				),
			},
		)
		if err != nil {
			// We should never hit a blocker creating a span.  If so, warn loudly.
			logger.StdlibLogger(ctx).Error("error saving span for checkpoint op", "error", err)
		}
	}

	// Update the group ID in context;  we've already saved this step's success and we're now
	// running the step again, needing a new history group
	groupID := uuid.New().String()

	// Re-enqueue the exact same edge to run now.
	return e.maybeEnqueueDiscoveryStep(
		state.WithGroupID(ctx, groupID),
		runCtx,
		gen,
		edge,
		groupID,
		hasPendingSteps,
	)

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
	// 		CreatedAt:  e.now(),
	// 		RunID:      i.md.ID.RunID,
	// 	})
	// }
}

func (e *executor) handleStepError(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
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
	if !runCtx.ShouldRetry() {
		// This is the last attempt as per the attempt in the queue, which
		// means we've failed N times, and so it is not retryable.
		retryable = false
	}

	if retryable {
		// Return an error to trigger standard queue retries.
		runCtx.IncrementAttempt()
		for _, l := range e.lifecycles {
			lifecycleItem := runCtx.LifecycleItem()
			go l.OnStepScheduled(ctx, *runCtx.Metadata(), lifecycleItem, &gen.Name)
		}
		return ErrHandledStepError
	}

	// This was the final step attempt and we still failed, so we convert the Error to Failed
	// and use that handler.
	gen.Op = enums.OpcodeStepFailed
	return e.handleStepFailed(ctx, runCtx, gen, edge)
}

func (e *executor) handleStepFailed(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	// First, save the error to our state store.
	output, err := gen.Output()
	if err != nil {
		return err
	}

	hasPendingSteps, err := e.smv2.SaveStep(ctx, runCtx.Metadata().ID, gen.ID, []byte(output))
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
	jobID := fmt.Sprintf("%s-%s-failure", runCtx.Metadata().IdempotencyKey(), gen.ID)
	now := e.now()
	nextItem := queue.Item{
		JobID:                 &jobID,
		WorkspaceID:           runCtx.Metadata().ID.Tenant.EnvID,
		GroupID:               groupID,
		Kind:                  queue.KindEdgeError,
		Identifier:            sv2.V1FromMetadata(*runCtx.Metadata()),
		PriorityFactor:        runCtx.PriorityFactor(),
		CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
		Attempt:               0,
		MaxAttempts:           runCtx.MaxAttempts(),
		Payload:               queue.PayloadEdge{Edge: nextEdge},
		Metadata:              make(map[string]any),
		ParallelMode:          gen.ParallelMode(),
	}

	if shouldEnqueueDiscovery(hasPendingSteps, runCtx.ParallelMode()) {
		lifecycleItem := runCtx.LifecycleItem()
		metadata := runCtx.Metadata()
		span, err := e.tracerProvider.CreateDroppableSpan(
			ctx,
			meta.SpanNameStepDiscovery,
			&tracing.CreateSpanOptions{
				Carriers:    []map[string]any{nextItem.Metadata},
				FollowsFrom: tracing.SpanRefFromQueueItem(&lifecycleItem),
				Debug:       &tracing.SpanDebugData{Location: "executor.handleStepFailed"},
				Metadata:    runCtx.Metadata(),
				QueueItem:   &nextItem,
				Parent:      tracing.RunSpanRefFromMetadata(metadata),
			},
		)
		if err != nil {
			// return fmt.Errorf("error creating span for next step after
			// StepError: %w", err)
			e.log.Debug("error creating span for next step after StepFailed", "error", err)
		}

		err = e.queue.Enqueue(ctx, nextItem, now, queue.EnqueueOpts{})
		if err == queue.ErrQueueItemExists {
			span.Drop()
			return nil
		}

		_ = span.Send()
	}

	for _, l := range e.lifecycles {
		go l.OnStepScheduled(ctx, *runCtx.Metadata(), nextItem, nil)
	}

	return nil
}

func (e *executor) handleGeneratorFunctionFinished(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	// In this case, we've reported that the function has finished.  It's an async
	// function.  In this case, we always want to update the span ourselves to mark
	// the function as finished, and add the output here.
	md := runCtx.Metadata()
	evts := runCtx.Events()
	resp := runCtx.DriverResponse()

	err := e.Finalize(ctx, execution.FinalizeOpts{
		Metadata: *md,
		Response: execution.FinalizeResponse{
			Type:        execution.FinalizeResponseRunComplete,
			RunComplete: gen,
		},
		Optional: execution.FinalizeOptional{
			InputEvents: evts,
		},
	})

	if resp != nil {
		for _, e := range e.lifecycles {
			go e.OnFunctionFinished(
				context.WithoutCancel(ctx),
				*md,
				runCtx.LifecycleItem(),
				evts,
				*resp,
			)
		}
	}

	return err
}

func (e *executor) handleGeneratorSyncFunctionFinished(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	// An API-based function went async and finished.  This must always be a apiresult.APIResult.
	// Both opcodes in a sync fn cehckpoint should always return this shape of data.
	result := struct {
		Data apiresult.APIResult `json:"data"`
	}{}
	if err := json.Unmarshal(gen.Data, &result); err != nil {
		// This should never happen with well-formed SDKs.  The SDK should always send the sync run complete
		// opcode with well-formed data.
		logger.StdlibLogger(ctx).Error("error unmarshalling api result from sync RunComplete op", "error", err)
		return err
	}

	md := runCtx.Metadata()
	evts := runCtx.Events()
	resp := runCtx.DriverResponse()

	err := e.Finalize(ctx, execution.FinalizeOpts{
		Metadata: *md,
		Response: execution.FinalizeResponse{
			Type:        execution.FinalizeResponseAPI,
			APIResponse: result.Data,
		},
		Optional: execution.FinalizeOptional{
			InputEvents: evts,
		},
	})

	if resp != nil {
		for _, e := range e.lifecycles {
			go e.OnFunctionFinished(
				context.WithoutCancel(ctx),
				*md,
				runCtx.LifecycleItem(),
				evts,
				*resp,
			)
		}
	}

	return err
}

func (e *executor) handleGeneratorStepPlanned(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
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
	jobID := fmt.Sprintf("%s-%s", runCtx.Metadata().IdempotencyKey(), gen.ID+"-plan")
	now := e.now()
	nextItem := queue.Item{
		JobID:                 &jobID,
		GroupID:               groupID, // Ensure we correlate future jobs with this group ID, eg. started/failed.
		WorkspaceID:           runCtx.Metadata().ID.Tenant.EnvID,
		Kind:                  queue.KindEdge,
		Identifier:            sv2.V1FromMetadata(*runCtx.Metadata()),
		PriorityFactor:        runCtx.PriorityFactor(),
		CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
		Attempt:               0,
		MaxAttempts:           runCtx.MaxAttempts(),
		Payload: queue.PayloadEdge{
			Edge: nextEdge,
		},
		Metadata:     make(map[string]any),
		ParallelMode: gen.ParallelMode(),
	}

	lifecycleItem := runCtx.LifecycleItem()
	span, err := e.tracerProvider.CreateDroppableSpan(
		ctx,
		meta.SpanNameStep,
		&tracing.CreateSpanOptions{
			Carriers:    []map[string]any{nextItem.Metadata},
			FollowsFrom: tracing.SpanRefFromQueueItem(&lifecycleItem),
			Debug:       &tracing.SpanDebugData{Location: "executor.handleGeneratorStepPlanned"},
			Metadata:    runCtx.Metadata(),
			QueueItem:   &nextItem,
			Parent:      tracing.RunSpanRefFromMetadata(runCtx.Metadata()),
			Attributes:  tracing.GeneratorAttrs(&gen),
		},
	)
	if err != nil {
		// return fmt.Errorf("error creating span for next step after
		// StepPlanned: %w", err)
		e.log.Debug("error creating span for next step after StepPlanned", "error", err)
	}

	err = e.queue.Enqueue(ctx, nextItem, now, queue.EnqueueOpts{})
	if err == queue.ErrQueueItemExists {
		span.Drop()
		return nil
	}

	_ = span.Send()

	for _, l := range e.lifecycles {
		go l.OnStepScheduled(ctx, *runCtx.Metadata(), nextItem, &gen.Name)
	}
	return err
}

// handleSleep handles the sleep opcode, ensuring that we enqueue the function to rerun
// at the correct time.
func (e *executor) handleGeneratorSleep(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	dur, err := gen.SleepDuration()
	if err != nil {
		return err
	}

	nextEdge := inngest.Edge{
		Outgoing: gen.ID,             // Leaving sleep
		Incoming: edge.Edge.Incoming, // To re-call the SDK
	}

	until := e.now().Add(dur)

	// Create another group for the next item which will run.  We're enqueueing
	// the function to run again after sleep, so need a new group.
	groupID := uuid.New().String()
	ctx = state.WithGroupID(ctx, groupID)

	jobID := queue.HashID(ctx, fmt.Sprintf("%s-%s", runCtx.Metadata().IdempotencyKey(), gen.ID))
	nextItem := queue.Item{
		JobID:       &jobID,
		WorkspaceID: runCtx.Metadata().ID.Tenant.EnvID,
		// Sleeps re-enqueue the step so that we can mark the step as completed
		// in the executor after the sleep is complete.  This will re-call the
		// generator step, but we need the same group ID for correlation.
		GroupID:               groupID,
		Kind:                  queue.KindSleep,
		Identifier:            sv2.V1FromMetadata(*runCtx.Metadata()),
		PriorityFactor:        runCtx.PriorityFactor(),
		CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
		Attempt:               0,
		MaxAttempts:           runCtx.MaxAttempts(),
		Payload:               queue.PayloadEdge{Edge: nextEdge},
		Metadata:              make(map[string]any),
		ParallelMode:          gen.ParallelMode(),
	}

	lifecycleItem := runCtx.LifecycleItem()
	metadata := runCtx.Metadata()

	// Create a new span that we'll use to record the sleep as complete.
	// This is going to be attached to the same parent (the discovery step that started this sleep).
	span, err := e.tracerProvider.CreateDroppableSpan(
		ctx,
		meta.SpanNameStep,
		&tracing.CreateSpanOptions{
			Carriers:    []map[string]any{nextItem.Metadata},
			FollowsFrom: tracing.SpanRefFromQueueItem(&lifecycleItem),
			Debug:       &tracing.SpanDebugData{Location: "executor.handleGeneratorSleep"},
			Metadata:    metadata,
			QueueItem:   &nextItem,
			Parent:      runCtx.ParentSpan(),
			Attributes:  tracing.GeneratorAttrs(&gen),
		},
	)
	if err != nil {
		e.log.Debug("error creating span for next step after Sleep", "error", err)
	}

	// And, annoyingly, we need to schedule the next discovery step span _now_.  We must do that
	// because when the sleep resumes, the next step may fail;  if we create a discovery step
	// when we resume the sleep there'll be a new discovery group per retry.  Not ideal.
	//
	// Doing that here allows us to make this deterministic.  In the future, if we had deterministic
	// span IDs we could remove this.
	{
		discoveryRef, err := e.tracerProvider.CreateSpan(
			ctx,
			meta.SpanNameStepDiscovery,
			&tracing.CreateSpanOptions{
				Debug:       &tracing.SpanDebugData{Location: "executor.sleepDiscovery"},
				Metadata:    metadata,
				FollowsFrom: span.Ref,
				// Always from the root span.
				Parent:    tracing.RunSpanRefFromMetadata(metadata),
				QueueItem: &nextItem,
				StartTime: until,
			},
		)
		if err != nil {
			e.log.Debug("error creating span discovery step after sleep", "error", err)
		}
		// Plumb this into the queue item manually, unfortunately.
		byt, _ := json.Marshal(discoveryRef)
		nextItem.Metadata["discovery"] = string(byt)
	}

	err = e.queue.Enqueue(ctx, nextItem, until, queue.EnqueueOpts{
		PassthroughJobId: true,
	})
	if err == queue.ErrQueueItemExists {
		span.Drop()
		return nil
	}

	_ = span.Send()

	for _, e := range e.lifecycles {
		go e.OnSleep(context.WithoutCancel(ctx), *runCtx.Metadata(), lifecycleItem, gen, until)
	}

	return err
}

func (e *executor) handleGeneratorGateway(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
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
	lifecycleItem := runCtx.LifecycleItem()
	e.addRequestPublishOpts(ctx, lifecycleItem, &req)
	metadata := runCtx.Metadata()
	execSpan := runCtx.ExecutionSpan()

	var output []byte

	resp, err := runCtx.HTTPClient().DoRequest(ctx, req)
	if err != nil {
		// Request failed entirely. Create an error.
		userLandErr := state.UserError{
			Name:    "GatewayError",
			Message: fmt.Sprintf("Error making gateway request: %s", err),
		}
		runCtx.UpdateOpcodeError(&gen, userLandErr)

		if spanErr := e.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
			Attributes: tracing.GatewayResponseAttrs(resp, &userLandErr, gen, nil),
			Debug:      &tracing.SpanDebugData{Location: "executor.handleGeneratorGateway"},
			Metadata:   metadata,
			QueueItem:  &lifecycleItem,
			TargetSpan: execSpan,
		}); spanErr != nil {
			e.log.Debug("error updating span for erroring gateway request during handleGeneratorGateway", "error", spanErr)
		}

		if runCtx.ShouldRetry() {
			runCtx.SetError(err)

			lifecycleItem := runCtx.LifecycleItem()
			for _, e := range e.lifecycles {
				go e.OnStepGatewayRequestFinished(context.WithoutCancel(ctx), *runCtx.Metadata(), lifecycleItem, edge.Edge, gen, nil, &userLandErr)
			}

			// This will retry, as it hits the queue directly.
			return fmt.Errorf("error making inference request: %w", err)
		}

		userLandErrByt, _ := json.Marshal(userLandErr)
		output, _ = json.Marshal(map[string]json.RawMessage{
			execution.StateErrorKey: userLandErrByt,
		})

		lifecycleItem := runCtx.LifecycleItem()
		for _, e := range e.lifecycles {
			go e.OnStepGatewayRequestFinished(context.WithoutCancel(ctx), *runCtx.Metadata(), lifecycleItem, edge.Edge, gen, nil, &userLandErr)
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

		runCtx.UpdateOpcodeOutput(&gen, output)
		lifecycleItem := runCtx.LifecycleItem()

		if spanErr := e.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
			Attributes: tracing.GatewayResponseAttrs(resp, nil, gen, nil),
			Debug:      &tracing.SpanDebugData{Location: "executor.handleGeneratorGateway"},
			Metadata:   metadata,
			QueueItem:  &lifecycleItem,
			TargetSpan: execSpan,
		}); spanErr != nil {
			e.log.Debug("error updating span for successful gateway request during handleGeneratorGateway", "error", spanErr)
		}

		for _, e := range e.lifecycles {
			// OnStepFinished handles step success and step errors/failures.  It is
			// currently the responsibility of the lifecycle manager to handle the differing
			// step statuses when a step finishes.
			go e.OnStepGatewayRequestFinished(context.WithoutCancel(ctx), *runCtx.Metadata(), lifecycleItem, edge.Edge, gen, nil, nil)
		}
	}

	// Save the output as the step result.
	hasPendingSteps, err := e.smv2.SaveStep(ctx, runCtx.Metadata().ID, gen.ID, output)
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
	jobID := fmt.Sprintf("%s-%s", runCtx.Metadata().IdempotencyKey(), gen.ID)
	now := e.now()
	nextItem := queue.Item{
		JobID:                 &jobID,
		WorkspaceID:           runCtx.Metadata().ID.Tenant.EnvID,
		GroupID:               groupID,
		Kind:                  queue.KindEdge,
		Identifier:            sv2.V1FromMetadata(*runCtx.Metadata()),
		PriorityFactor:        runCtx.PriorityFactor(),
		CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
		Attempt:               0,
		MaxAttempts:           runCtx.MaxAttempts(),
		Payload:               queue.PayloadEdge{Edge: nextEdge},
		Metadata:              make(map[string]any),
		ParallelMode:          gen.ParallelMode(),
	}

	if shouldEnqueueDiscovery(hasPendingSteps, gen.ParallelMode()) {
		lifecycleItem := runCtx.LifecycleItem()
		metadata := runCtx.Metadata()
		span, err := e.tracerProvider.CreateDroppableSpan(
			ctx,
			meta.SpanNameStepDiscovery,
			&tracing.CreateSpanOptions{
				Carriers:    []map[string]any{nextItem.Metadata},
				FollowsFrom: tracing.SpanRefFromQueueItem(&lifecycleItem),
				Debug:       &tracing.SpanDebugData{Location: "executor.handleGeneratorGateway"},
				Metadata:    metadata,
				Parent:      tracing.RunSpanRefFromMetadata(metadata),
				QueueItem:   &nextItem,
			},
		)
		if err != nil {
			e.log.Debug("error creating span for next step after Gateway", "error", err)
		}

		err = e.queue.Enqueue(ctx, nextItem, now, queue.EnqueueOpts{})
		if err != nil {
			if span != nil {
				span.Drop()
			}

			if err == queue.ErrQueueItemExists {
				return nil
			}

			logger.StdlibLogger(ctx).Error("error scheduling Gateway step queue item", "error", err)
			return err
		}

		if span != nil {
			_ = span.Send()
		}
	}

	for _, l := range e.lifecycles {
		// We can't specify step name here since that will result in the
		// "followup discovery step" having the same name as its predecessor.
		var stepName *string = nil
		go l.OnStepScheduled(ctx, *runCtx.Metadata(), nextItem, stepName)
	}

	return err
}

func (e *executor) handleGeneratorAIGateway(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
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

	lifecycleItem := runCtx.LifecycleItem()
	runMetadata := runCtx.Metadata()

	// If the opcode contains streaming data, we should fetch a JWT with perms
	// for us to stream then add streaming data to the serializable request.
	//
	// Without this, publishing will not work.
	e.addRequestPublishOpts(ctx, lifecycleItem, &req)

	resp, err := runCtx.HTTPClient().DoRequest(ctx, req)
	failure := err != nil || (resp != nil && resp.StatusCode > 299)

	// Update the driver response appropriately for the trace lifecycles.
	if resp == nil {
		resp = &exechttp.Response{}
	}

	runCtx.SetStatusCode(resp.StatusCode)

	if e.allowStepMetadata.Enabled(ctx, runMetadata.ID.Tenant.AccountID) {
		md := metadata.WithWarnings(extractors.ExtractAIGatewayMetadata(
			input,
			resp.StatusCode,
			resp.Body,
		))
		for _, m := range md {
			_, err := e.createMetadataSpan(
				ctx,
				runCtx,
				"executor.handleGeneratorAIGateway",
				m,
				enums.MetadataScopeStepAttempt,
			)
			if err != nil {
				e.log.Warn("error creating metadata span", "error", err)
			}
		}
	}

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
		runCtx.UpdateOpcodeError(&gen, userLandErr)

		if spanErr := e.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
			Attributes: tracing.GatewayResponseAttrs(resp, &userLandErr, gen, nil),
			Debug:      &tracing.SpanDebugData{Location: "executor.handleGeneratorAIGateway"},
			Metadata:   runMetadata,
			QueueItem:  &lifecycleItem,
			TargetSpan: runCtx.ExecutionSpan(),
		}); spanErr != nil {
			e.log.Debug("error updating span for successful gateway request during handleGeneratorAIGateway", "error", spanErr)
		}

		// And, finally, if this is retryable return an error which will be retried.
		// Otherwise, we enqueue the next step directly so that the SDK can throw
		// an error on output.
		if runCtx.ShouldRetry() {
			// Set the response error, ensuring the response is retryable in the queue.
			runCtx.SetError(err)

			lifecycleItem := runCtx.LifecycleItem()
			for _, e := range e.lifecycles {
				// OnStepFinished handles step success and step errors/failures.  It is
				// currently the responsibility of the lifecycle manager to handle the differing
				// step statuses when a step finishes.
				go e.OnStepGatewayRequestFinished(context.WithoutCancel(ctx), *runCtx.Metadata(), lifecycleItem, edge.Edge, gen, nil, &userLandErr)
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

		lifecycleItem := runCtx.LifecycleItem()
		for _, e := range e.lifecycles {
			// OnStepFinished handles step success and step errors/failures.  It is
			// currently the responsibility of the lifecycle manager to handle the differing
			// step statuses when a step finishes.
			go e.OnStepGatewayRequestFinished(context.WithoutCancel(ctx), *runCtx.Metadata(), lifecycleItem, edge.Edge, gen, nil, &userLandErr)
		}
	} else {
		rawBody := resp.Body

		// The response output is actually now the result of this AI call. We need
		// to modify the opcode data so that accessing the step output is correct.
		//
		// Also note that the output is always wrapped within "data", allowing us
		// to differentiate between success and failure in the SDK in the single
		// opcode map.
		resp.Body, err = json.Marshal(map[string]json.RawMessage{
			execution.StateDataKey: rawBody,
		})
		if err != nil {
			return fmt.Errorf("error wrapping ai result in map: %w", err)
		}

		runCtx.UpdateOpcodeOutput(&gen, resp.Body)
		lifecycleItem := runCtx.LifecycleItem()

		if spanErr := e.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
			Attributes: tracing.GatewayResponseAttrs(resp, nil, gen, rawBody),
			Debug:      &tracing.SpanDebugData{Location: "executor.handleGeneratorAIGateway"},
			Metadata:   runMetadata,
			QueueItem:  &lifecycleItem,
			TargetSpan: runCtx.ExecutionSpan(),
		}); spanErr != nil {
			e.log.Debug("error updating span for successful gateway request during handleGeneratorAIGateway", "error", spanErr)
		}

		for _, e := range e.lifecycles {
			// OnStepFinished handles step success and step errors/failures.  It is
			// currently the responsibility of the lifecycle manager to handle the differing
			// step statuses when a step finishes.
			go e.OnStepGatewayRequestFinished(context.WithoutCancel(ctx), *runCtx.Metadata(), lifecycleItem, edge.Edge, gen, nil, nil)
		}
	}

	// Save the output as the step result.
	hasPendingSteps, err := e.smv2.SaveStep(ctx, runCtx.Metadata().ID, gen.ID, resp.Body)
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
	jobID := fmt.Sprintf("%s-%s", runCtx.Metadata().IdempotencyKey(), gen.ID)
	now := e.now()
	nextItem := queue.Item{
		JobID:                 &jobID,
		WorkspaceID:           runCtx.Metadata().ID.Tenant.EnvID,
		GroupID:               groupID,
		Kind:                  queue.KindEdge,
		Identifier:            sv2.V1FromMetadata(*runCtx.Metadata()), // Convert from v2 metadata
		PriorityFactor:        runCtx.PriorityFactor(),
		CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
		Attempt:               0,
		MaxAttempts:           runCtx.MaxAttempts(),
		Payload:               queue.PayloadEdge{Edge: nextEdge},
		Metadata:              make(map[string]any),
		ParallelMode:          gen.ParallelMode(),
	}

	if shouldEnqueueDiscovery(hasPendingSteps, runCtx.ParallelMode()) {
		lifecycleItem := runCtx.LifecycleItem()
		metadata := runCtx.Metadata()
		span, err := e.tracerProvider.CreateDroppableSpan(
			ctx,
			meta.SpanNameStepDiscovery,
			&tracing.CreateSpanOptions{
				Carriers:    []map[string]any{nextItem.Metadata},
				FollowsFrom: tracing.SpanRefFromQueueItem(&lifecycleItem),
				Debug:       &tracing.SpanDebugData{Location: "executor.handleGeneratorAIGateway"},
				Metadata:    metadata,
				Parent:      tracing.RunSpanRefFromMetadata(metadata),
				QueueItem:   &nextItem,
			},
		)
		if err != nil {
			e.log.Debug("error creating span for next step after AI Gateway", "error", err)
		}

		err = e.queue.Enqueue(ctx, nextItem, now, queue.EnqueueOpts{})
		if err != nil {
			if span != nil {
				span.Drop()
			}

			if err == queue.ErrQueueItemExists {
				return nil
			}

			logger.StdlibLogger(ctx).Error("error scheduling AI Gateway step queue item", "error", err)
			return err
		}

		if span != nil {
			_ = span.Send()
		}
	}

	for _, l := range e.lifecycles {
		// We can't specify step name here since that will result in the
		// "followup discovery step" having the same name as its predecessor.
		var stepName *string = nil
		go l.OnStepScheduled(ctx, *runCtx.Metadata(), nextItem, stepName)
	}

	return err
}

func (e *executor) handleGeneratorWaitForSignal(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
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

	pauseID := inngest.DeterministicSha1UUID(runCtx.Metadata().ID.RunID.String() + gen.ID)
	opcode := gen.Op.String()
	now := e.now()

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
		WorkspaceID:             runCtx.Metadata().ID.Tenant.EnvID,
		Identifier:              sv2.NewPauseIdentifier(runCtx.Metadata().ID),
		GroupID:                 runCtx.GroupID(),
		Outgoing:                gen.ID,
		Incoming:                edge.Edge.Incoming,
		StepName:                gen.UserDefinedName(),
		Opcode:                  &opcode,
		Expires:                 state.Time(expires),
		DataKey:                 gen.ID,
		SignalID:                &opts.Signal,
		ReplaceSignalOnConflict: shouldReplaceSignalOnConflict,
		MaxAttempts:             runCtx.MaxAttempts(),
		Metadata: map[string]any{
			consts.OtelPropagationKey: carrier,
		},
		ParallelMode: gen.ParallelMode(),
		CreatedAt:    now,
	}

	// Enqueue a job that will timeout the pause.
	jobID := fmt.Sprintf("%s-%s", runCtx.Metadata().IdempotencyKey(), gen.ID)
	nextItem := queue.Item{
		JobID:                 &jobID,
		WorkspaceID:           runCtx.Metadata().ID.Tenant.EnvID,
		GroupID:               runCtx.GroupID(),
		Kind:                  queue.KindPause,
		Identifier:            sv2.V1FromMetadata(*runCtx.Metadata()),
		PriorityFactor:        runCtx.PriorityFactor(),
		CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
		MaxAttempts:           runCtx.MaxAttempts(),
		Payload: queue.PayloadPauseTimeout{
			PauseID: pauseID,
			Pause:   pause,
		},
		Metadata:     make(map[string]any),
		ParallelMode: gen.ParallelMode(),
	}

	lifecycleItem := runCtx.LifecycleItem()
	span, err := e.tracerProvider.CreateDroppableSpan(
		ctx,
		meta.SpanNameStep,
		&tracing.CreateSpanOptions{
			Carriers:    []map[string]any{pause.Metadata, nextItem.Metadata},
			FollowsFrom: tracing.SpanRefFromQueueItem(&lifecycleItem),
			Debug:       &tracing.SpanDebugData{Location: "executor.handleGeneratorWaitForSignal"},
			Metadata:    runCtx.Metadata(),
			QueueItem:   &nextItem,
			Parent:      tracing.RunSpanRefFromMetadata(runCtx.Metadata()),
			Attributes:  tracing.GeneratorAttrs(&gen),
			StartTime:   now,
		},
	)
	if err != nil {
		// return fmt.Errorf("error creating span for next step after
		// WaitForSignal: %w", err)
		e.log.Debug("error creating span for next step after WaitForSignal", "error", err)
	}

	_, err = e.pm.Write(ctx, pauses.PauseIndex(pause), &pause)
	if err == state.ErrSignalConflict {
		stdErr := state.WrapInStandardError(
			err,
			"Error",
			"Signal conflict; signal wait already exists for another run",
			"",
		)

		if span != nil {
			// Write and update the span with the failure
			_ = span.Send()

			attrs := meta.NewAttrSet()

			byt, marshalErr := json.Marshal(stdErr)
			if marshalErr != nil {
				attrs.AddErr(fmt.Errorf("error marshalling standard error: %w", marshalErr))
			} else {
				output := string(byt)
				hasOutput := true

				meta.AddAttr(attrs, meta.Attrs.StepOutput, &output)
				meta.AddAttr(attrs, meta.Attrs.StepHasOutput, &hasOutput)
			}

			if updateSpanErr := e.tracerProvider.UpdateSpan(ctx, &tracing.UpdateSpanOptions{
				EndTime:    e.now(),
				Debug:      &tracing.SpanDebugData{Location: "executor.handleGeneratorWaitForSignal"},
				Status:     enums.StepStatusFailed,
				TargetSpan: span.Ref,
				Attributes: attrs,
				Metadata:   runCtx.Metadata(),
				QueueItem:  &nextItem,
			}); updateSpanErr != nil {
				e.log.Debug("error updating span for conflicting WaitForSignal during handleGeneratorWaitForSignal", "error", updateSpanErr)
			}
		}

		return stdErr
	}
	if err != nil {
		if errors.Is(err, state.ErrPauseAlreadyExists) {
			if span != nil {
				span.Drop()
			}
		} else {
			return fmt.Errorf("error saving pause when handling WaitForSignal opcode: %w", err)
		}
	}

	err = e.queue.Enqueue(ctx, nextItem, expires, queue.EnqueueOpts{})
	if err == queue.ErrQueueItemExists {
		if span != nil {
			span.Drop()
		}

		return nil
	}

	if span != nil {
		_ = span.Send()
	}

	for _, e := range e.lifecycles {
		go e.OnWaitForSignal(
			context.WithoutCancel(ctx),
			*runCtx.Metadata(),
			lifecycleItem,
			gen,
			pause,
		)
	}

	return err
}

func (e *executor) handleGeneratorInvokeFunction(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
	if e.handleInvokeEvent == nil {
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
	correlationID := runCtx.Metadata().ID.RunID.String() + "." + gen.ID
	strExpr := fmt.Sprintf("async.data.%s == %s", consts.InvokeCorrelationId, strconv.Quote(correlationID))
	_, err = e.newExpressionEvaluator(ctx, strExpr)
	if err != nil {
		return execError{err: fmt.Errorf("failed to create expression to wait for invoked function completion: %w", err)}
	}

	pauseID := inngest.DeterministicSha1UUID(runCtx.Metadata().ID.RunID.String() + gen.ID)
	opcode := gen.Op.String()
	now := e.now()

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
		AccountID:       runCtx.Metadata().ID.Tenant.AccountID,
		EnvID:           runCtx.Metadata().ID.Tenant.EnvID,
		Event:           *opts.Payload,
		FnID:            opts.FunctionID,
		CorrelationID:   &correlationID,
		TraceCarrier:    carrier,
		ExpiresAt:       expires.UnixMilli(),
		GroupID:         runCtx.GroupID(),
		DisplayName:     gen.UserDefinedName(),
		SourceAppID:     runCtx.Metadata().ID.Tenant.AppID.String(),
		SourceFnID:      runCtx.Metadata().ID.FunctionID.String(),
		SourceFnVersion: runCtx.Metadata().Config.FunctionVersion,
	})

	pause := state.Pause{
		ID:                  pauseID,
		WorkspaceID:         runCtx.Metadata().ID.Tenant.EnvID,
		Identifier:          sv2.NewPauseIdentifier(runCtx.Metadata().ID),
		GroupID:             runCtx.GroupID(),
		Outgoing:            gen.ID,
		Incoming:            edge.Edge.Incoming,
		StepName:            gen.UserDefinedName(),
		Opcode:              &opcode,
		Expires:             state.Time(expires),
		Event:               &eventName,
		Expression:          &strExpr,
		DataKey:             gen.ID,
		InvokeCorrelationID: &correlationID,
		TriggeringEventID:   &evt.Event.ID,
		InvokeTargetFnID:    &opts.FunctionID,
		MaxAttempts:         runCtx.MaxAttempts(),
		Metadata: map[string]any{
			consts.OtelPropagationKey: carrier,
		},
		ParallelMode: gen.ParallelMode(),
		CreatedAt:    now,
	}

	// Enqueue a job that will timeout the pause.
	jobID := fmt.Sprintf("%s-%s", runCtx.Metadata().IdempotencyKey(), gen.ID)
	nextItem := queue.Item{
		JobID:       &jobID,
		WorkspaceID: runCtx.Metadata().ID.Tenant.EnvID,
		// Use the same group ID, allowing us to track the cancellation of
		// the step correctly.
		GroupID:               runCtx.GroupID(),
		Kind:                  queue.KindPause,
		Identifier:            sv2.V1FromMetadata(*runCtx.Metadata()),
		PriorityFactor:        runCtx.PriorityFactor(),
		CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
		MaxAttempts:           runCtx.MaxAttempts(),
		Payload: queue.PayloadPauseTimeout{
			PauseID: pauseID,
			Pause:   pause,
		},
		Metadata:     make(map[string]any),
		ParallelMode: gen.ParallelMode(),
	}

	lifecycleItem := runCtx.LifecycleItem()
	span, err := e.tracerProvider.CreateDroppableSpan(
		ctx,
		meta.SpanNameStep,
		&tracing.CreateSpanOptions{
			Carriers:    []map[string]any{pause.Metadata, nextItem.Metadata},
			StartTime:   now,
			FollowsFrom: tracing.SpanRefFromQueueItem(&lifecycleItem),
			Debug:       &tracing.SpanDebugData{Location: "executor.handleGeneratorInvokeFunction"},
			Metadata:    runCtx.Metadata(),
			QueueItem:   &nextItem,
			Parent:      tracing.RunSpanRefFromMetadata(runCtx.Metadata()),
			Attributes: tracing.GeneratorAttrs(&gen).Merge(
				// Always correlate the triggering event ID with the invoked step.
				meta.NewAttrSet(meta.Attr(meta.Attrs.StepInvokeTriggerEventID, &evt.ID)),
			),
		},
	)
	if err != nil {
		// return fmt.Errorf("error creating span for next step after
		// InvokeFunction: %w", err)
		e.log.Debug("error creating span for next step after InvokeFunction", "error", err)
	}

	idx := pauses.Index{WorkspaceID: runCtx.Metadata().ID.Tenant.EnvID, EventName: eventName}

	// We really don't want this to fail, the invoke can be retried fine in an idempotent way but
	// workflows with 0 retries setup will just hang forever if pause creation fails.
	_, err = util.WithRetry(ctx, "pause.handleGeneratorInvokeFunction", func(ctx context.Context) (int, error) {
		return e.pm.Write(ctx, idx, &pause)
	}, util.NewRetryConf(util.WithRetryConfRetryableErrors(pauses.WritePauseRetryableError)))
	// A pause may already exist if the write succeeded but we timed out before
	// returning (MDB i/o timeouts). In that case, we ignore the
	// ErrPauseAlreadyExists error and continue. We rely on the pause timeout enqueuing
	// to avoid duplicate invokes instead.
	if err != nil {
		if errors.Is(err, state.ErrPauseAlreadyExists) {
			if span != nil {
				span.Drop()
			}
		} else {
			return err
		}
	}

	err = e.queue.Enqueue(ctx, nextItem, expires, queue.EnqueueOpts{})
	if err == queue.ErrQueueItemExists {
		if span != nil {
			span.Drop()
		}

		return nil
	} else if err != nil {
		logger.StdlibLogger(ctx).Error(
			"failed to enqueue invoke function pause timeout",
			"error", err,
			"run_id", runCtx.Metadata().ID.RunID,
			"workspace_id", runCtx.Metadata().ID.Tenant.EnvID,
		)
	}

	if span != nil {
		_ = span.Send()
	}

	// Send the event.
	err = e.handleInvokeEvent(ctx, evt)
	if err != nil {
		// TODO Cancel pause/timeout?
		return fmt.Errorf("error publishing internal invocation event: %w", err)
	}

	for _, e := range e.lifecycles {
		go e.OnInvokeFunction(context.WithoutCancel(ctx), *runCtx.Metadata(), lifecycleItem, gen, evt.Event)
	}

	return err
}

func (e *executor) handleGeneratorWaitForEvent(ctx context.Context, runCtx execution.RunContext, gen state.GeneratorOpcode, edge queue.PayloadEdge) error {
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

	pauseID := inngest.DeterministicSha1UUID(runCtx.Metadata().ID.RunID.String() + gen.ID)

	expr := opts.If
	if expr != nil && strings.Contains(*expr, "event.") {
		// Remove `event` data from the expression and replace with actual event
		// data as values, now that we have the event.
		//
		// This improves performance in matching, as we can then use the values within
		// aggregate trees.
		evt := event.Event{}
		if err := json.Unmarshal(runCtx.Events()[0], &evt); err != nil {
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
	now := e.now()

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
		WorkspaceID: runCtx.Metadata().ID.Tenant.EnvID,
		Identifier:  sv2.NewPauseIdentifier(runCtx.Metadata().ID),
		GroupID:     runCtx.GroupID(),
		Outgoing:    gen.ID,
		Incoming:    edge.Edge.Incoming,
		StepName:    gen.UserDefinedName(),
		Opcode:      &opcode,
		Expires:     state.Time(expires),
		Event:       &opts.Event,
		Expression:  expr,
		DataKey:     gen.ID,
		MaxAttempts: runCtx.MaxAttempts(),
		Metadata: map[string]any{
			consts.OtelPropagationKey: carrier,
		},
		ParallelMode: gen.ParallelMode(),
		CreatedAt:    now,
	}

	// SDK-based event coordination is called both when an event is received
	// OR on timeout, depending on which happens first.  Both routes consume
	// the pause so this race will conclude by calling the function once, as only
	// one thread can lease and consume a pause;  the other will find that the
	// pause is no longer available and return.
	jobID := fmt.Sprintf("%s-%s", runCtx.Metadata().IdempotencyKey(), gen.ID)
	nextItem := queue.Item{
		JobID:       &jobID,
		WorkspaceID: runCtx.Metadata().ID.Tenant.EnvID,
		// Use the same group ID, allowing us to track the cancellation of
		// the step correctly.
		GroupID:               runCtx.GroupID(),
		Kind:                  queue.KindPause,
		Identifier:            sv2.V1FromMetadata(*runCtx.Metadata()),
		PriorityFactor:        runCtx.PriorityFactor(),
		CustomConcurrencyKeys: runCtx.ConcurrencyKeys(),
		Payload: queue.PayloadPauseTimeout{
			PauseID: pauseID,
			Pause:   pause,
		},
		Metadata:     make(map[string]any),
		ParallelMode: gen.ParallelMode(),
	}

	lifecycleItem := runCtx.LifecycleItem()
	span, err := e.tracerProvider.CreateDroppableSpan(
		ctx,
		meta.SpanNameStep,
		&tracing.CreateSpanOptions{
			Carriers:    []map[string]any{pause.Metadata, nextItem.Metadata},
			FollowsFrom: tracing.SpanRefFromQueueItem(&lifecycleItem),
			Debug:       &tracing.SpanDebugData{Location: "executor.handleGeneratorWaitForEvent"},
			Metadata:    runCtx.Metadata(),
			QueueItem:   &nextItem,
			Parent:      tracing.RunSpanRefFromMetadata(runCtx.Metadata()),
			Attributes:  tracing.GeneratorAttrs(&gen),
		},
	)
	if err != nil {
		// return fmt.Errorf("error creating span for next step after
		// WaitForEvent: %w", err)
		e.log.Debug("error creating span for next step after WaitForEvent", "error", err)
	}

	idx := pauses.Index{WorkspaceID: runCtx.Metadata().ID.Tenant.EnvID, EventName: opts.Event}

	// We really don't want this to fail, this can be retried in an idempotent way but
	// workflows with 0 retries setup will just hang forever if pause creation fails.
	_, err = util.WithRetry(ctx, "pause.handleGeneratorWaitForEvent", func(ctx context.Context) (int, error) {
		return e.pm.Write(ctx, idx, &pause)
	}, util.NewRetryConf(util.WithRetryConfRetryableErrors(pauses.WritePauseRetryableError)))
	// A pause may already exist if the write succeeded but we timed out before
	// returning (MDB i/o timeouts). In that case, we ignore the
	// ErrPauseAlreadyExists error and continue.
	// Instead we rely on the pause timeout queue item for idempotency.
	if err != nil {
		if err != state.ErrPauseAlreadyExists {
			return err
		}
		// Allow pause already existing to be idempotent, and continue on with enqueueing.
		span.Drop()
	}

	// TODO Is this fine to leave? No attempts.
	err = e.queue.Enqueue(ctx, nextItem, expires, queue.EnqueueOpts{})
	if err == queue.ErrQueueItemExists {
		span.Drop()
		return nil
	}

	_ = span.Send()

	for _, e := range e.lifecycles {
		go e.OnWaitForEvent(context.WithoutCancel(ctx), *runCtx.Metadata(), lifecycleItem, gen, pause)
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
	enableInstrumentation := e.enableBatchingInstrumentation != nil && e.enableBatchingInstrumentation(ctx, bi.AccountID, bi.WorkspaceID)
	l := logger.StdlibLogger(ctx).With("eventID", bi.EventID)
	result, err := e.batcher.Append(ctx, bi, fn)
	if enableInstrumentation {
		l.Debug("Appending to batch", "err", err, "result", result)
	}
	if err != nil {
		return err
	}

	if opts == nil {
		opts = &execution.BatchExecOpts{}
	}

	switch result.Status {
	case enums.BatchAppend, enums.BatchItemExists:
		// noop
	case enums.BatchNew:
		dur, err := time.ParseDuration(fn.EventBatch.Timeout)
		if err != nil {
			return err
		}
		at := e.now().Add(dur)

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
	enableInstrumentation := e.enableBatchingInstrumentation != nil && e.enableBatchingInstrumentation(ctx, payload.AccountID, payload.WorkspaceID)
	evtList, err := e.batcher.RetrieveItems(ctx, payload.FunctionID, payload.BatchID)

	l := logger.StdlibLogger(ctx).With("accountID", payload.AccountID, "workspace_id", payload.WorkspaceID, "batchID", payload.BatchID)
	if enableInstrumentation {
		l.Debug("retrieved batch items", "events", len(evtList), "err", err)
	}
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
		// Batching does not work with rate limiting
		PreventRateLimit: true,
	})

	if enableInstrumentation {
		l.Debug("attempted to schedule batch", "err", err)
	}

	metrics.IncrExecutorScheduleCount(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"type":   "batch",
			"status": ScheduleStatus(err),
		},
	})

	// Ensure to delete batch when Schedule worked, we already processed it, or the function was paused
	shouldDeleteBatch := err == nil ||
		err == queue.ErrQueueItemExists ||
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
	if err == queue.ErrQueueItemExists ||
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

func (e *executor) GetEvent(ctx context.Context, id ulid.ULID, accountID uuid.UUID, workspaceID uuid.UUID) (any, error) {
	return e.traceReader.GetEvent(ctx, id, accountID, workspaceID)
}

func (e *executor) fnDriver(ctx context.Context, fn inngest.Function) any {
	name := inngest.Driver(fn)
	if d, ok := e.driverv1[name]; ok {
		return d
	}
	if d, ok := e.driverv2[name]; ok {
		return d
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
		return res, err
	}

	if signalID == "" {
		err = fmt.Errorf("signal ID is empty")
		return res, err
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
		return res, err
	}

	res = &execution.ResumeSignalResult{}

	if pause == nil {
		l.Debug("no pause found for signal")
		return res, err
	}

	if pause.Expires.Time().Before(e.now()) {
		l.Debug("encountered expired signal")

		shouldDelete := pause.Expires.Time().Add(consts.PauseExpiredDeletionGracePeriod).Before(e.now())
		if shouldDelete {
			l.Debug("deleting expired pause")
			_ = e.pm.Delete(ctx, pauses.PauseIndex(*pause), *pause)
		}

		return res, err
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

		return res, err
	}

	res.MatchedSignal = true
	res.RunID = &pause.Identifier.RunID

	return res, err
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
func (e *executor) addRequestPublishOpts(ctx context.Context, item queue.Item, sr *exechttp.SerializableRequest) {
	if e.rtconfig.PublishURL == "" {
		return
	}

	token, err := realtime.NewPublishJWT(
		ctx,
		e.rtconfig.Secret,
		item.Identifier.AccountID,
		item.WorkspaceID,
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

// shouldEnqueueDiscovery returns true if the ended step should have a discovery
// step enqueued
func shouldEnqueueDiscovery(hasPendingSteps bool, mode enums.ParallelMode) bool {
	return !hasPendingSteps || mode == enums.ParallelModeRace
}

func (e *executor) getParentSpan(ctx context.Context, item queue.Item, md sv2.Metadata) *meta.SpanReference {
	if item.Kind != queue.KindSleep {
		return tracing.SpanRefFromQueueItem(&item)
	}

	// Grab the discovery step from the queue item, if it exists.  This is created when
	// handling the generator item.
	//
	// This makes sure that the discovery span ID is *stable* after a queue item is ran
	// across all retries.
	if data, ok := item.Metadata["discovery"].(string); ok {
		ref := &meta.SpanReference{}
		if err := json.Unmarshal([]byte(data), ref); err == nil {
			return ref
		}
	}

	// The embedded discovery span might not've existed for old sleeps, so create a new
	// one and deal with it being unstable across each sleep resume attempt...
	parentRef, err := e.tracerProvider.CreateSpan(
		ctx,
		meta.SpanNameStepDiscovery,
		&tracing.CreateSpanOptions{
			FollowsFrom: tracing.SpanRefFromQueueItem(&item),
			Debug:       &tracing.SpanDebugData{Location: "executor.PostSleepDiscovery"},
			Metadata:    &md,
			// Always from the root span.
			Parent:    tracing.RunSpanRefFromMetadata(&md),
			QueueItem: &item,
			StartTime: e.now(),
		},
	)
	if err != nil {
		logger.StdlibLogger(ctx).Warn("error creating discovery step span after sleep resume", "error", err)
		// fallback. this literally should NEVER happen
		parentRef = tracing.SpanRefFromQueueItem(&item)
	}

	return parentRef
}

// Checkpoint traces configures hwehter we should emit traces after recording steps.

type traceStepsValT struct{}

var traceStepsVal = traceStepsValT{}

func setEmitCheckpointTraces(ctx context.Context) context.Context {
	return context.WithValue(ctx, traceStepsVal, true)
}

func emitCheckpointTraces(ctx context.Context) bool {
	ok, _ := ctx.Value(traceStepsVal).(bool)
	return ok
}

func (e *executor) createMetadataSpan(ctx context.Context, runCtx execution.RunContext, location string, md metadata.Structured, scope metadata.Scope) (*meta.SpanReference, error) {
	var parent *meta.SpanReference

	switch scope {
	case enums.MetadataScopeRun:
		parent = tracing.RunSpanRefFromMetadata(runCtx.Metadata())
	case enums.MetadataScopeStep:
		parent = runCtx.ParentSpan()
	case enums.MetadataScopeStepAttempt:
		parent = runCtx.ExecutionSpan()
	default:
		return nil, fmt.Errorf("unknown metadata scope: %s", scope)
	}

	return tracing.CreateMetadataSpan(
		ctx,
		e.tracerProvider,
		parent,
		location,
		pkgName,
		runCtx.Metadata(),
		md,
		scope,
	)
}
