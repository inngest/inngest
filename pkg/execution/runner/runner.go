package runner

import (
	"bytes"
	"context"
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
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/inngest/inngest/pkg/service"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/oklog/ulid/v2"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	CancelTimeout = (24 * time.Hour) * 365
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

func WithRateLimiter(rl ratelimit.RateLimiter) func(s *svc) {
	return func(s *svc) {
		s.rl = rl
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
	// rl rate-limits functions.
	rl ratelimit.RateLimiter
	// cronmanager allows the creation of new scheduled functions.
	cronmanager *cron.Cron

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
	// Each runner service is responsible for initializing cron-based executions.
	// As the runners are shared-nothing, there is contention when running multiple
	// services;  each individual service will attempt to create a new cron execution
	// simultaneously.  We currently rely on idempotency within the state store to
	// ensure that only one run can execute.
	//
	// In the future, we may want to add distributed locking and/or a limit on the
	// number of concurrent services that can schedule crons.  We don't really want
	// to rely on a single executor to 'claim' ownership:  we'd have to implement
	// more complex logic to check for the last heartbeat and valid cron scheduled,
	// then backtrack to re-execute in the case of node downtime.  This is simple.
	if err := s.InitializeCrons(ctx); err != nil {
		return err
	}

	s.log.Info("subscribing to events", "topic", s.config.EventStream.Service.TopicName())
	err := s.pubsub.Subscribe(ctx, s.config.EventStream.Service.TopicName(), s.handleMessage)
	if err != nil {
		return err
	}
	return nil
}

func (s *svc) Stop(ctx context.Context) error {
	if s.cronmanager != nil {
		cronCtx := s.cronmanager.Stop()
		select {
		case <-cronCtx.Done():
		case <-ctx.Done():
			return fmt.Errorf("error waiting for scheduled executions to finish")
		}
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

func (s *svc) InitializeCrons(ctx context.Context) error {
	// If a previous cron manager exists, cancel it.
	if s.cronmanager != nil {
		s.cronmanager.Stop()
	}

	s.cronmanager = cron.New(
		cron.WithParser(
			cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
		),
	)

	// Set the functions within the engine, then iterate through each function's
	// triggers so that we can easily invoke them.  We also need to immediately
	// set up cron timers to invoke functions on a schedule.
	fns, err := s.data.FunctionsScheduled(ctx)
	if err != nil {
		return err
	}

	for _, f := range fns {
		fn := f
		// Set up a cron schedule for the current function.
		for _, t := range f.Triggers {
			if t.CronTrigger == nil {
				continue
			}
			cron := t.CronTrigger.Cron
			_, err := s.cronmanager.AddFunc(cron, func() {
				// Create a new context to avoid "context canceled" errors. This
				// callback is run as a non-blocking goroutine in Cron.Start, so
				// contexts from outside its scope will likely be cancelled
				// before the function is run
				ctx := context.Background()

				ctx, span := itrace.UserTracer().Provider().
					Tracer(consts.OtelScopeCron).
					Start(ctx, "cron", trace.WithAttributes(
						attribute.String(consts.OtelSysFunctionID, fn.ID.String()),
						attribute.Int(consts.OtelSysFunctionVersion, fn.FunctionVersion),
					))
				defer span.End()

				trackedEvent := event.NewOSSTrackedEvent(event.Event{
					Data: map[string]any{
						"cron": cron,
					},
					ID:        time.Now().UTC().Format(time.RFC3339),
					Name:      event.FnCronName,
					Timestamp: time.Now().UnixMilli(),
				}, nil)

				byt, err := json.Marshal(trackedEvent)
				if err == nil {
					err := s.publisher.Publish(
						ctx,
						s.config.EventStream.Service.TopicName(),
						pubsub.Message{
							Name:      event.EventReceivedName,
							Data:      string(byt),
							Timestamp: time.Now(),
						},
					)
					if err != nil {
						s.log.Error("error publishing cron event", "error", err)
					}
				} else {
					s.log.Error("error marshaling cron event", "error", err)
				}

				err = s.initialize(ctx, fn, trackedEvent)
				if err != nil {
					s.log.Error("error initializing scheduled function", "error", err)
				}
			})
			if err != nil {
				return err
			}
		}
	}
	s.cronmanager.Start()
	return nil
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

	tracked, err := event.NewOSSTrackedEventFromString(m.Data)
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

				// Evaluate all expressions for matching triggers
				if t.Expression != nil {
					// Execute expressions here, ensuring that each function is only triggered
					// under the correct conditions.
					ok, _, evalerr := expressions.EvaluateBoolean(ctx, *t.Expression, map[string]interface{}{
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
			ok, _, evalerr := expressions.EvaluateBoolean(ctx, *batchCondition, map[string]interface{}{
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

	// Attempt to rate-limit the incoming function.
	if s.rl != nil && fn.RateLimit != nil {
		key, err := ratelimit.RateLimitKey(ctx, fn.ID, *fn.RateLimit, evt.GetEvent().Map())
		switch err {
		case nil:
			limited, _, err := s.rl.RateLimit(ctx, key, *fn.RateLimit)
			if err != nil {
				return err
			}
			if limited {
				if evt.GetEvent().IsInvokeEvent() {
					// This function was invoked by another function, so we need to
					// ensure that the invoker fails. If we don't do this, it'll
					// hang forever
					if err := s.executor.InvokeFailHandler(ctx, execution.InvokeFailHandlerOpts{
						OriginalEvent: evt,
						Err: map[string]any{
							"name":    "Error",
							"message": "invoked function is rate limited",
						},
					}); err != nil {
						l.Error("error handling invoke rate limit", "error", err)
					}
				}
				// Do nothing.
				return nil
			}
		case ratelimit.ErrNotRateLimited:
			// no-op: proceed with function run as usual
		default:
			return err
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

	switch err {
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
