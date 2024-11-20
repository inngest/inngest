package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

	StateManager() state.Manager
	InitializeCrons(ctx context.Context) error
	Runs(ctx context.Context, accountId uuid.UUID, eventId ulid.ULID) ([]state.State, error)
	Events(ctx context.Context, eventId string) ([]event.Event, error)
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

func WithEventManager(e event.Manager) func(s *svc) {
	return func(s *svc) {
		s.em = &e
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

// WithTracker is used in the dev server to track runs.
func WithTracker(t *Tracker) func(s *svc) {
	// XXX: Replace with sqlite
	return func(s *svc) {
		s.tracker = t
	}
}

func WithPublisher(p pubsub.Publisher) func(s *svc) {
	return func(s *svc) {
		s.publisher = p
	}
}

func NewService(c config.Config, opts ...Opt) Runner {
	svc := &svc{config: c}
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
	// queue allows the scheduling of new functions.
	queue queue.Queue
	// batcher handles batch operations
	batcher batch.BatchManager
	// rl rate-limits functions.
	rl ratelimit.RateLimiter
	// cronmanager allows the creation of new scheduled functions.
	cronmanager *cron.Cron
	em          *event.Manager

	tracker *Tracker
}

func (s svc) Name() string {
	return "runner"
}

func (s *svc) Pre(ctx context.Context) error {
	var err error

	logger.From(ctx).Info().Str("backend", s.config.Queue.Service.Backend).Msg("starting event stream")
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

	l := logger.From(ctx)
	l.Info().
		Str("topic", s.config.EventStream.Service.TopicName()).
		Msg("subscribing to events")
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
				})

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
						logger.From(ctx).Error().Err(err).Msg("error publishing cron event")
					}
				} else {
					logger.From(ctx).Error().Err(err).Msg("error marshaling cron event")
				}

				err = s.initialize(ctx, fn, trackedEvent)
				if err != nil {
					logger.From(ctx).Error().Err(err).Msg("error initializing scheduled function")
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

func (s *svc) Runs(ctx context.Context, accountId uuid.UUID, eventID ulid.ULID) ([]state.State, error) {
	items, _ := s.tracker.Runs(ctx, eventID)
	result := make([]state.State, len(items))
	for n, i := range items {
		state, err := s.state.Load(ctx, accountId, i)
		if err != nil {
			return nil, err
		}
		result[n] = state
	}
	return result, nil
}

func (s *svc) StateManager() state.Manager {
	return s.state
}

func (s *svc) Events(ctx context.Context, eventId string) ([]event.Event, error) {
	if eventId != "" {
		evt := s.em.EventById(eventId)
		if evt != nil {
			return []event.Event{evt.GetEvent()}, nil
		}

		return []event.Event{}, nil
	}

	trackedEvents := s.em.Events()
	evts := make([]event.Event, len(trackedEvents))
	for i, evt := range trackedEvents {
		evts[i] = evt.GetEvent()
	}

	return evts, nil
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

	var tracked event.TrackedEvent
	var err error

	if s.em == nil {
		tracked, err = event.NewOSSTrackedEventFromString(m.Data)
	} else {
		tracked, err = s.em.NewEvent(m.Data)
	}
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

	l := logger.From(ctx).With().
		Str("event", tracked.GetEvent().Name).
		Str("id", tracked.GetEvent().ID).
		Str("internal_id", tracked.GetInternalID().String()).
		Logger()
	ctx = logger.With(ctx, l)

	l.Info().Msg("received message")

	var errs error
	wg := &sync.WaitGroup{}

	// Trigger both new functions and pauses.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.functions(ctx, tracked); err != nil {
			l.Error().Err(err).Msg("error scheduling functions")
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
					l.Warn().Err(err).Msg("can't find paused function to resume after invoke")
					return
				}

				l.Error().Err(err).Msg("error resuming function after invoke")
				errs = multierror.Append(errs, err)
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.pauses(ctx, tracked); err != nil {
			l.Error().Err(err).Msg("error consuming pauses")
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
				logger.From(ctx).Error().
					Err(err).
					Str("function", fn.Name).
					Msg("error invoking fn")
				errs = multierror.Append(errs, err)
			}
		}
	}()

	fns, err := s.data.FunctionsByTrigger(ctx, evt.Name)
	if err != nil {
		return fmt.Errorf("error loading functions by trigger: %w", err)
	}
	if len(fns) == 0 {
		return nil
	}

	logger.From(ctx).Debug().Int("len", len(fns)).Msg("scheduling functions")

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
			for _, t := range copied.Triggers {
				if t.EventTrigger == nil || t.Event != evt.Name {
					// This isn't triggered by an event, so we skip this trigger entirely.
					continue
				}

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
					logger.From(ctx).Error().
						Err(err).
						Str("function", copied.Name).
						Msg("error initializing fn")
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
	l := logger.From(ctx).With().
		Str("event", evt.GetEvent().Name).
		Str("id", evt.GetEvent().ID).
		Str("internal_id", evt.GetInternalID().String()).
		Logger()

	l.Trace().Msg("querying for invoke pauses")

	return s.executor.HandleInvokeFinish(ctx, evt)
}

// pauses searches for and triggers all pauses from this event.
func (s *svc) pauses(ctx context.Context, evt event.TrackedEvent) error {
	l := logger.From(ctx).With().
		Str("event", evt.GetEvent().Name).
		Str("id", evt.GetEvent().ID).
		Str("internal_id", evt.GetInternalID().String()).
		Logger()

	l.Trace().Msg("querying for pauses")

	wsID := evt.GetWorkspaceID()
	if ok, err := s.state.EventHasPauses(ctx, wsID, evt.GetEvent().Name); err == nil && !ok {
		return nil
	}

	l.Trace().Msg("pauses found; handling")

	iter, err := s.state.PausesByEvent(ctx, wsID, evt.GetEvent().Name)
	if err != nil {
		return fmt.Errorf("error finding event pauses: %w", err)
	}

	_, err = s.executor.HandlePauses(ctx, iter, evt)
	return err
}

func (s *svc) initialize(ctx context.Context, fn inngest.Function, evt event.TrackedEvent) error {
	l := logger.From(ctx).With().
		Str("function", fn.Name).
		Str("function_id", fn.ID.String()).Logger()

	var appID uuid.UUID
	wsID := evt.GetWorkspaceID()
	{
		fn, err := s.cqrs.GetFunctionByInternalUUID(ctx, wsID, fn.ID)
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
			AccountID:       consts.DevServerAccountId,
		}

		if err := s.executor.AppendAndScheduleBatch(ctx, fn, bi, nil); err != nil {
			return fmt.Errorf("could not append and schedule batch item: %w", err)
		}

		return nil
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
						l.Error().Err(err).Msg("error handling invoke rate limit")
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

	l.Info().Msg("initializing fn")
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

	// If this is a debounced function, run this through a debouncer.
	md, err := opts.exec.Schedule(ctx, execution.ScheduleRequest{
		WorkspaceID:    wsID,
		AppID:          opts.appID,
		Function:       fn,
		Events:         []event.TrackedEvent{tracked},
		IdempotencyKey: &idempotencyKey,
		AccountID:      consts.DevServerAccountId,
	})

	switch err {
	case executor.ErrFunctionDebounced,
		executor.ErrFunctionSkipped,
		state.ErrIdentifierExists:
		return nil, nil
	}

	if err != nil {
		logger.StdlibLogger(ctx).Error("error scheduling function", "error", err)
	}
	return md, err
}

// NewTracker returns a crappy in-memory tracker used for registering function runs.
func NewTracker() (t *Tracker) {
	return &Tracker{
		l:      &sync.RWMutex{},
		evtIDs: map[string][]ulid.ULID{},
	}
}

type Tracker struct {
	l      *sync.RWMutex
	evtIDs map[string][]ulid.ULID
}

func (t *Tracker) Add(evtID string, id state.Identifier) {
	if t.l == nil {
		return
	}

	t.l.Lock()
	defer t.l.Unlock()
	if _, ok := t.evtIDs[evtID]; !ok {
		t.evtIDs[evtID] = []ulid.ULID{id.RunID}
		return
	}
	t.evtIDs[evtID] = append(t.evtIDs[evtID], id.RunID)
}

func (t *Tracker) Runs(ctx context.Context, eventId ulid.ULID) ([]ulid.ULID, error) {
	if t.l == nil {
		return nil, nil
	}
	t.l.RLock()
	defer t.l.RUnlock()
	return t.evtIDs[eventId.String()], nil
}
