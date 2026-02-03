package runner

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/event_trigger_patterns"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/cron"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/propagation"
)

const (
	CancelTimeout = (24 * time.Hour) * 365
	pkgName       = "execution.runner"
)

type Opt func(s *svc)

// Runner is the interface for the runner, which provides a standard Service
// and the ability to re-initialize crons.
type Runner interface {
	service.Service

	// This allows publishing of events to local CQRS for development.
	event.Publisher

	StateManager() state.Manager
	InitializeCrons(ctx context.Context) error
}

func WithCQRS(data cqrs.Manager) func(s *svc) {
	return func(s *svc) {
		s.cqrs = data
	}
}

func WithExecutor(e execution.Executor) func(s *svc) {
	return func(s *svc) {
		s.executor = e
	}
}

func WithExecutionManager(l cqrs.Manager) func(s *svc) {
	return func(s *svc) {
		s.data = l
	}
}

func WithPauseManager(pm pauses.Manager) func(s *svc) {
	return func(s *svc) {
		s.pm = pm
	}
}

func WithStateManager(sm state.Manager) func(s *svc) {
	return func(s *svc) {
		s.state = sm
	}
}

func WithRunnerQueue(q queue.Queue) func(s *svc) {
	return func(s *svc) {
		s.queue = q
	}
}

func WithBatchManager(b batch.BatchManager) func(s *svc) {
	return func(s *svc) {
		s.batcher = b
	}
}

func WithCronManager(c cron.CronManager) func(s *svc) {
	return func(s *svc) {
		s.croner = c
	}
}

func WithPublisher(p pubsub.Publisher) func(s *svc) {
	return func(s *svc) {
		s.publisher = p
	}
}

func WithLogger(l logger.Logger) func(s *svc) {
	return func(s *svc) {
		s.log = l
	}
}

func NewService(c config.Config, opts ...Opt) Runner {
	svc := &svc{config: c, log: logger.StdlibLogger(context.Background())}
	for _, o := range opts {
		o(svc)
	}
	return svc
}

type svc struct {
	config config.Config
	cqrs   cqrs.Manager
	// pubsub allows us to subscribe to new events, and re-publish events
	// if there are errors.
	pubsub    pubsub.PublishSubscriber
	publisher pubsub.Publisher
	// executor handles execution of functions.
	executor execution.Executor
	// data provides the required loading capabilities to trigger functions
	// from events.
	data cqrs.Manager
	// state allows the creation of new function runs.
	state state.Manager
	// pauses allows management of pauses, used to resume function runs on matching events.
	pm pauses.Manager
	// queue allows the scheduling of new functions.
	queue queue.Queue
	// batcher handles batch operations
	batcher batch.BatchManager
	// croner handles cron operations
	croner cron.CronManager

	log logger.Logger
}

func (s svc) Name() string {
	return "runner"
}

func (s *svc) Pre(ctx context.Context) error {
	var err error

	s.log.Info("starting event stream", "backend", s.config.Queue.Service.Backend)
	s.pubsub, err = pubsub.NewPublishSubscriber(ctx, s.config.EventStream.Service)
	if err != nil {
		return err
	}

	if s.state == nil {
		s.state, err = s.config.State.Service.Concrete.SingleClusterManager(ctx)
		if err != nil {
			return err
		}
	}

	if s.queue == nil {
		s.queue, err = s.config.Queue.Service.Concrete.Queue()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *svc) Run(ctx context.Context) error {
	// initialize crons from data store.
	//
	// this is more relevant for persisted environment like lite, where there's an external data store
	// persisting function configuration.
	if err := s.InitializeCrons(ctx); err != nil {
		return err
	}

	s.log.Info("subscribing to events", "topic", s.config.EventStream.Service.TopicName())
	// Use SubscribeN with concurrency to allow parallel event handling.
	// This is necessary for batch buffering - when Append() blocks waiting for
	// the buffer to fill/flush, we need other events to be processed concurrently
	// to fill the buffer.
	err := s.pubsub.SubscribeN(ctx, s.config.EventStream.Service.TopicName(), s.handleMessage, 1000)
	if err != nil {
		return err
	}
	return nil
}

func (s *svc) Stop(ctx context.Context) error {
	if s.batcher != nil {
		return s.batcher.Close()
	}
	return nil
}

// Publish fulfils the event.Publisher interface for local development.
func (s *svc) Publish(ctx context.Context, evt event.TrackedEvent) error {
	byt, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("error marshalling event: %w", err)
	}
	return s.publisher.Publish(
		ctx,
		s.config.EventStream.Service.TopicName(),
		pubsub.Message{
			Name:      event.EventReceivedName,
			Data:      string(byt),
			Timestamp: time.Now(),
		},
	)
}

// InitializeCrons initializes cron schedules for all scheduled functions in the system.
// This method is called during service startup to ensure that all functions with
// cron triggers are properly scheduled and ready to execute.
//
// The initialization process:
// 1. Retrieves all functions that have scheduled triggers from the data store
// 2. For each scheduled function, creates a CronItem with CronInit operation
// 3. Enqueues the CronItem as a sync job to initialize the cron schedule
//
// The CronInit operation ensures that:
// - If no schedule exists for the function, a new one is created
// - If a schedule already exists, no changes are made (idempotent)
//
// This approach allows for safe restarts and prevents duplicate schedules while
// ensuring all scheduled functions are properly initialized.
func (s *svc) InitializeCrons(ctx context.Context) error {
	l := s.log.With("action", "executor.InitializeCrons")

	// Retrieve all functions that have scheduled triggers from the data store.
	// This includes functions with cron expressions that need to be executed
	// on a periodic basis.
	fns, err := s.data.FunctionsScheduled(ctx)
	if err != nil {
		return err
	}

	// Process each function to initialize its cron schedule
	for _, f := range fns {
		fn := f

		cqrsFn, err := s.cqrs.GetFunctionByInternalUUID(ctx, fn.ID)
		if err != nil {
			return fmt.Errorf("error fetching appID during cron initialization for fn: %s, err: %w", fn.ID, err)
		}
		appID := cqrsFn.AppID

		cronExprs := f.ScheduleExpressions()
		for _, cronExpr := range cronExprs {
			// Launch each cron initialization in a separate goroutine to avoid
			// blocking the startup process. This allows multiple functions to be
			// initialized concurrently.
			go func(ctx context.Context, fn inngest.Function) {
				// Configure queue item parameters for the cron sync job
				//
				// This will trigger the cron manager's UpdateSchedule method with the
				// CronInit operation to initialize the schedule if needed.
				if err := s.croner.Sync(ctx, cron.CronItem{
					ID:              ulid.MustNew(ulid.Now(), rand.Reader),
					AccountID:       consts.DevServerAccountID,
					WorkspaceID:     consts.DevServerEnvID,
					FunctionID:      fn.ID,
					AppID:           appID,
					FunctionVersion: fn.FunctionVersion,
					Expression:      cronExpr,
					Op:              enums.CronInit, // Initialize operation
				}); err != nil {
					l.Error("error initializing cron sync job", "error", err)
				}
			}(ctx, fn)
		}
	}

	// Start health check for crons.
	return s.croner.EnqueueNextHealthCheck(ctx)
}

func (s *svc) StateManager() state.Manager {
	return s.state
}

func (s *svc) handleMessage(ctx context.Context, m pubsub.Message) error {
	if m.Name != event.EventReceivedName {
		return fmt.Errorf("unknown event type: %s", m.Name)
	}

	if m.Metadata != nil {
		if trace, ok := m.Metadata[consts.OtelPropagationKey]; ok {
			carrier := itrace.NewTraceCarrier()
			if err := carrier.Unmarshal(trace); err == nil {
				ctx = itrace.UserTracer().Propagator().Extract(ctx, propagation.MapCarrier(carrier.Context))
			}
		}
	}

	tracked, err := event.NewBaseTrackedEventFromString(m.Data)
	if err != nil {
		return fmt.Errorf("error creating event: %w", err)
	}

	// Write the event to our CQRS manager for long-term storage.
	err = s.cqrs.InsertEvent(
		ctx,
		cqrs.ConvertFromEvent(tracked.GetInternalID(), tracked.GetEvent()),
	)
	if err != nil {
		return err
	}

	l := s.log.With(
		"event", tracked.GetEvent().Name,
		"event_id", tracked.GetEvent().ID,
		"internal_id", tracked.GetInternalID().String(),
	)

	ctx = logger.WithStdlib(ctx, l)

	l.Info("received event")

	var errs error
	wg := &sync.WaitGroup{}

	// Trigger both new functions and pauses.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.functions(ctx, tracked); err != nil {
			l.Error("error scheduling functions", "error", err)
			errs = multierror.Append(errs, err)
		}
	}()

	// check if this is an "inngest/function.finished" event
	// triggered by invoke
	corrId := tracked.GetEvent().CorrelationID()
	if tracked.GetEvent().IsFinishedEvent() && corrId != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.invokes(ctx, tracked); err != nil {
				if err == state.ErrInvokePauseNotFound || err == state.ErrPauseNotFound {
					l.Warn("can't find paused function to resume after invoke", "error", err)
					return
				}

				l.Error("error resuming function after invoke", "error", err)
				errs = multierror.Append(errs, err)
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.pauses(ctx, tracked); err != nil {
			l.Error("error consuming pauses", "error", err)
			errs = multierror.Append(errs, err)
		}
	}()

	wg.Wait()
	return errs
}

// FindInvokedFunction is a helper method which loads all available functions, checks
// the incoming event and returns the function to be invoked via the RPC invoke event,
// or nil if a function is not being invoked.
func FindInvokedFunction(ctx context.Context, tracked event.TrackedEvent, fl cqrs.ExecutionLoader) (*inngest.Function, error) {
	evt := tracked.GetEvent()

	if evt.Name != event.InvokeFnName {
		return nil, nil
	}

	fns, err := fl.Functions(ctx)
	if err != nil {
		return nil, err
	}

	fnID := ""
	metadata, err := evt.InngestMetadata()
	if err != nil {
		return nil, err
	}
	if metadata != nil && metadata.InvokeFnID != "" {
		fnID = metadata.InvokeFnID
	}
	if fnID == "" {
		return nil, fmt.Errorf("could not extract function ID from event")
	}

	for _, fn := range fns {
		if fn.GetSlug() == fnID {
			return &fn, nil
		}
	}

	return nil, fmt.Errorf("could not find function with ID: %s", fnID)
}

// functions triggers all functions from the given event.
func (s *svc) functions(ctx context.Context, tracked event.TrackedEvent) error {
	evt := tracked.GetEvent()

	// Don't use an errgroup here as we want all errors together, vs the first
	// non-nil error.
	var errs error
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		// Invoke functions by RPC-like calling
		defer wg.Done()
		// Find any invoke functions specified.
		fn, err := FindInvokedFunction(ctx, tracked, s.data)
		if err != nil {
			errs = multierror.Append(errs, err)

			// If this errored, then we were supposed to find a function to
			// invoke. In this case, emit a completion event with the error.
			perr := s.executor.InvokeFailHandler(ctx, execution.InvokeFailHandlerOpts{
				OriginalEvent: tracked,
				FunctionID:    "",
				RunID:         "",
				// TODO unify
				Err: map[string]any{
					"name":    "Error",
					"message": err.Error(),
				},
			})
			if perr != nil {
				errs = multierror.Append(errs, perr)
			}
		}
		if fn != nil {
			// Initialize this function for this event only once;  we don't
			// want multiple matching triggers to run the function more than once.
			err := s.initialize(ctx, *fn, tracked)
			if err != nil {
				s.log.Error("error invoking fn",
					"error", err,
					"function", fn.Name,
				)
				errs = multierror.Append(errs, err)
			}
		}
	}()

	// Look up all functions have a trigger that matches the event name, including wildcards.
	fns, err := s.data.FunctionsByTrigger(ctx, evt.Name)
	if err != nil {
		return fmt.Errorf("error loading functions by trigger: %w", err)
	}
	if len(fns) == 0 {
		return nil
	}

	s.log.Debug("scheduling functions", "len", len(fns))

	// Do this once instead of many times when evaluating expressions.
	evtMap := evt.Map()

	matchingPatterns := event_trigger_patterns.GenerateMatchingPatterns(evt.Name)

	for _, fn := range fns {
		// We want to initialize each function concurrently;  some of these
		// may have expressions that take ~tens of milliseconds to run, and
		// each function should have as little latency as possible.
		copied := fn
		wg.Add(1)
		go func() {
			defer wg.Done()

			defer func() {
				if r := recover(); r != nil {
					s.log.Error("panic initializing function",
						"error", fmt.Sprintf("%v", r),
						"functions", copied.Name,
						"stack", string(debug.Stack()),
					)
				}
			}()

			for _, t := range copied.Triggers {
				if t.EventTrigger == nil {
					continue
				}

				// Only process triggers that match the current event
				if !t.EventTrigger.MatchesAnyPattern(matchingPatterns) {
					continue
				}

				// Evaluate all expressions for matching triggers
				if t.Expression != nil {
					// Execute expressions here, ensuring that each function is only triggered
					// under the correct conditions.
					ok, evalerr := expressions.EvaluateBoolean(ctx, *t.Expression, map[string]interface{}{
						"event": evtMap,
					})
					if evalerr != nil {
						errs = multierror.Append(errs, evalerr)
						continue
					}
					if !ok {
						// Skip this trigger.
						continue
					}
				}

				// Initialize this function for this event only once;  we don't
				// want multiple matching triggers to run the function more than once.
				err := s.initialize(ctx, copied, tracked)
				if err != nil {
					s.log.Error("error initializing fn",
						"error", err,
						"function", copied.Name,
					)
					errs = multierror.Append(errs, err)
				}
				return
			}
		}()
	}

	wg.Wait()
	return errs
}

// invokes looks for a pause with the same correlation ID and triggers it
func (s *svc) invokes(ctx context.Context, evt event.TrackedEvent) error {
	l := logger.StdlibLogger(ctx).With(
		"event", evt.GetEvent().Name,
		"event_id", evt.GetEvent().ID,
		"internal_id", evt.GetInternalID().String(),
	)

	l.Trace("querying for invoke pauses")

	return s.executor.HandleInvokeFinish(ctx, evt)
}

// pauses searches for and triggers all pauses from this event.
func (s *svc) pauses(ctx context.Context, evt event.TrackedEvent) error {
	l := logger.StdlibLogger(ctx).With(
		"event", evt.GetEvent().Name,
		"event_id", evt.GetEvent().ID,
		"internal_id", evt.GetInternalID().String(),
	)

	l.Trace("querying for pauses")

	wsID := evt.GetWorkspaceID()
	idx := pauses.Index{WorkspaceID: wsID, EventName: evt.GetEvent().Name}

	if ok, err := s.pm.IndexExists(ctx, idx); err == nil && !ok {
		l.Debug("no pauses found for event")
		return nil
	}

	l.Debug("handling found pauses for event")

	_, err := s.executor.HandlePauses(ctx, evt)
	if err != nil {
		l.Error("error handling pauses", "error", err)
	}
	return err
}

func (s *svc) initialize(ctx context.Context, fn inngest.Function, evt event.TrackedEvent) error {
	l := logger.StdlibLogger(ctx).With(
		"function", fn.Name,
		"function_id", fn.ID.String(),
	)

	var appID uuid.UUID
	wsID := evt.GetWorkspaceID()
	{
		fn, err := s.cqrs.GetFunctionByInternalUUID(ctx, fn.ID)
		if err != nil {
			return err
		}
		appID = fn.AppID
	}

	if fn.IsBatchEnabled() {
		bi := batch.BatchItem{
			WorkspaceID:     wsID,
			AppID:           appID,
			FunctionID:      fn.ID,
			FunctionVersion: fn.FunctionVersion,
			EventID:         evt.GetInternalID(),
			Event:           evt.GetEvent(),
			AccountID:       consts.DevServerAccountID,
		}

		// When conditional batching is requested based on `EventBatch.If`, batching is enabled only for events that successfully evaluate to true.
		// If the conditional expression evaluation fails or the expression evaluates to false, then the event is scheduled for immediate execution without waiting for a batch to fill up.
		eligibleForBatching := true
		if batchCondition := fn.EventBatch.If; batchCondition != nil {
			ok, evalerr := expressions.EvaluateBoolean(ctx, *batchCondition, map[string]interface{}{
				"event": evt.GetEvent().Map(),
			})
			if evalerr != nil || !ok {
				eligibleForBatching = false
			}
		}

		if eligibleForBatching {
			if err := s.executor.AppendAndScheduleBatch(ctx, fn, bi, nil); err != nil {
				return fmt.Errorf("could not append and schedule batch item: %w", err)
			}
			return nil
		}
	}

	l.Info("initializing fn")
	_, err := Initialize(ctx, InitOpts{
		appID: appID,
		fn:    fn,
		evt:   evt,
		exec:  s.executor,
	})
	if err == state.ErrIdentifierExists {
		// This run exists;  do not attempt to recreate it.
		return nil
	}
	if err == executor.ErrFunctionDebounced {
		return nil
	}
	return err
}

type InitOpts struct {
	appID uuid.UUID
	fn    inngest.Function
	evt   event.TrackedEvent
	exec  execution.Executor
}

// Initialize creates a new funciton run identifier for the given workflow and
// event, stores this in our state store, then enqueues a new function run
// within the given queue for execution.
//
// This is a separate, exported function so that it can be used from this service
// and also from eg. the run command.
func Initialize(ctx context.Context, opts InitOpts) (*sv2.Metadata, error) {
	l := logger.StdlibLogger(ctx)
	zero := uuid.UUID{}
	tracked := opts.evt
	wsID := tracked.GetWorkspaceID()
	fn := opts.fn

	if bytes.Equal(fn.ID[:], zero[:]) {
		// Locally, we want to ensure that each function has its own deterministic
		// UUID for managing state.
		//
		// Using a remote API, this UUID may be a surrogate primary key.
		fn.ID = fn.DeterministicUUID()
	}

	// Use the custom event ID (a.k.a. event idempotency key) if it exists, else
	// use the internal event ID
	idempotencyKey := tracked.GetEvent().ID

	var debugSessionID, debugRunID *ulid.ULID
	if evt := tracked.GetEvent(); evt.IsInvokeEvent() {
		if metadata, err := evt.InngestMetadata(); err == nil {
			debugSessionID = metadata.DebugSessionID
			debugRunID = metadata.DebugRunID
		}
	}

	// If this is a debounced function, run this through a debouncer.
	md, err := opts.exec.Schedule(ctx, execution.ScheduleRequest{
		WorkspaceID:    wsID,
		AppID:          opts.appID,
		Function:       fn,
		Events:         []event.TrackedEvent{tracked},
		IdempotencyKey: &idempotencyKey,
		AccountID:      consts.DevServerAccountID,
		DebugSessionID: debugSessionID,
		DebugRunID:     debugRunID,
	})

	metrics.IncrExecutorScheduleCount(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"type":   "event",
			"status": executor.ScheduleStatus(err),
		},
	})

	switch err {
	case executor.ErrFunctionRateLimited:
		if opts.evt.GetEvent().IsInvokeEvent() {
			// This function was invoked by another function, so we need to
			// ensure that the invoker fails. If we don't do this, it'll
			// hang forever
			if err := opts.exec.InvokeFailHandler(ctx, execution.InvokeFailHandlerOpts{
				OriginalEvent: opts.evt,
				Err: map[string]any{
					"name":    "Error",
					"message": "invoked function is rate limited",
				},
			}); err != nil {
				l.Error("error handling invoke rate limit", "error", err)
			}
		}

		return nil, nil
	case executor.ErrFunctionDebounced,
		executor.ErrFunctionSkipped,
		executor.ErrFunctionSkippedIdempotency,
		state.ErrIdentifierExists:
		return nil, nil
	}

	if err != nil {
		logger.StdlibLogger(ctx).Error("error scheduling function", "error", err)
	}
	return md, err
}
