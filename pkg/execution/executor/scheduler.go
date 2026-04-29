package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/inngest/inngest/pkg/execution/singleton"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/run"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
)

// SchedulerOpt modifies the scheduler on creation.
type SchedulerOpt func(s *scheduler) error

func NewScheduler(opts ...SchedulerOpt) (execution.Scheduler, error) {
	s := &scheduler{
		conditionalTracer: itrace.NoopConditionalTracer(),
	}

	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func WithSchedulerLogger(l logger.Logger) SchedulerOpt {
	return func(s *scheduler) error {
		s.log = l
		return nil
	}
}

func WithSchedulerConditionalTracer(tracer itrace.ConditionalTracer) SchedulerOpt {
	return func(s *scheduler) error {
		s.conditionalTracer = tracer
		return nil
	}
}

func WithSchedulerStateManager(sm sv2.RunService) SchedulerOpt {
	return func(s *scheduler) error {
		s.smv2 = sm
		return nil
	}
}

func WithSchedulerPauseManager(pm pauses.Manager) SchedulerOpt {
	return func(s *scheduler) error {
		s.pm = pm
		return nil
	}
}

func WithSchedulerProducer(p queue.Producer) SchedulerOpt {
	return func(s *scheduler) error {
		s.producer = p
		return nil
	}
}

func WithSchedulerJobQueueReader(jqr queue.JobQueueReader) SchedulerOpt {
	return func(s *scheduler) error {
		s.jobQueueReader = jqr
		return nil
	}
}

func WithSchedulerTraceReader(tr cqrs.TraceReader) SchedulerOpt {
	return func(s *scheduler) error {
		s.traceReader = tr
		return nil
	}
}

func WithSchedulerRateLimiter(rl ratelimit.RateLimiter) SchedulerOpt {
	return func(s *scheduler) error {
		s.rateLimiter = rl
		return nil
	}
}

func WithSchedulerDebouncer(d debounce.Debouncer) SchedulerOpt {
	return func(s *scheduler) error {
		s.debouncer = d
		return nil
	}
}

func WithSchedulerSingletonManager(sn singleton.Singleton) SchedulerOpt {
	return func(s *scheduler) error {
		s.singletonMgr = sn
		return nil
	}
}

func WithSchedulerCapacityManager(cm constraintapi.CapacityManager) SchedulerOpt {
	return func(s *scheduler) error {
		s.capacityManager = cm
		return nil
	}
}

func WithSchedulerUseConstraintAPI(uca constraintapi.UseConstraintAPIFn) SchedulerOpt {
	return func(s *scheduler) error {
		s.useConstraintAPI = uca
		return nil
	}
}

func WithSchedulerTracerProvider(tp tracing.TracerProvider) SchedulerOpt {
	return func(s *scheduler) error {
		s.tracerProvider = tp
		return nil
	}
}

func WithSchedulerFunctionBacklogSizeLimit(fbsl BacklogSizeLimitFn) SchedulerOpt {
	return func(s *scheduler) error {
		s.functionBacklogSizeLimit = fbsl
		return nil
	}
}

func WithSchedulerClock(c clockwork.Clock) SchedulerOpt {
	return func(s *scheduler) error {
		s.clock = c
		return nil
	}
}

func WithSchedulerLifecycleListeners(ls ...execution.LifecycleListener) SchedulerOpt {
	return func(s *scheduler) error {
		s.lifecycles = append(s.lifecycles, ls...)
		return nil
	}
}

func WithSchedulerBatcher(b batch.BatchManager) SchedulerOpt {
	return func(s *scheduler) error {
		s.batcher = b
		return nil
	}
}

func WithSchedulerEnableBatchingInstrumentation(ebi func(ctx context.Context, accountID, envID uuid.UUID) (enable bool)) SchedulerOpt {
	return func(s *scheduler) error {
		s.enableBatchingInstrumentation = ebi
		return nil
	}
}

func WithSchedulerFunctionLoader(fl state.FunctionLoader) SchedulerOpt {
	return func(s *scheduler) error {
		s.fl = fl
		return nil
	}
}

func WithSchedulerFinishHandler(f execution.FinalizePublisher) SchedulerOpt {
	return func(s *scheduler) error {
		s.finishHandler = f
		return nil
	}
}

func WithSchedulerSemaphoreManager(sm constraintapi.SemaphoreManager) SchedulerOpt {
	return func(s *scheduler) error {
		s.semaphoreManager = sm
		return nil
	}
}

func WithSchedulerShardSelector(sel queue.ShardSelector) SchedulerOpt {
	return func(s *scheduler) error {
		s.shardFinder = sel
		return nil
	}
}

type scheduler struct {
	log               logger.Logger
	conditionalTracer itrace.ConditionalTracer
	smv2              sv2.RunService
	pm                pauses.Manager
	producer          queue.Producer
	jobQueueReader    queue.JobQueueReader
	traceReader       cqrs.TraceReader
	tracerProvider    tracing.TracerProvider

	rateLimiter              ratelimit.RateLimiter
	debouncer                debounce.Debouncer
	singletonMgr             singleton.Singleton
	capacityManager          constraintapi.CapacityManager
	useConstraintAPI         constraintapi.UseConstraintAPIFn
	functionBacklogSizeLimit BacklogSizeLimitFn

	batcher                       batch.BatchManager
	enableBatchingInstrumentation func(ctx context.Context, accountID, envID uuid.UUID) (enable bool)

	clock      clockwork.Clock
	lifecycles []execution.LifecycleListener

	// fields below support Cancel and Finalize.
	fl               state.FunctionLoader
	finishHandler    execution.FinalizePublisher
	semaphoreManager constraintapi.SemaphoreManager
	shardFinder      queue.ShardSelector

	// invokeFinishHandler optionally provides a fast-resume path for invoke
	// pauses when an executor is paired with this scheduler.  When nil, parent
	// invokes are still resumed via the regular event handler pub/sub flow.
	invokeFinishHandler func(ctx context.Context, evt event.TrackedEvent) error
}

// SetInvokeFinishHandler wires a fast-resume callback used by Finalize to
// resume parent invokes immediately rather than waiting for the pub/sub event
// handler.  Typically called by the executor after construction so the
// scheduler can delegate back into executor-internal logic.
func (s *scheduler) SetInvokeFinishHandler(fn func(ctx context.Context, evt event.TrackedEvent) error) {
	s.invokeFinishHandler = fn
}

// SetFinalizer sets the finish handler used to publish finalization events.
func (s *scheduler) SetFinalizer(f execution.FinalizePublisher) {
	s.finishHandler = f
}

// AddLifecycleListener appends a lifecycle listener.
func (s *scheduler) AddLifecycleListener(l execution.LifecycleListener) {
	s.lifecycles = append(s.lifecycles, l)
}

func (s *scheduler) now() time.Time {
	if s.clock != nil {
		return s.clock.Now()
	}
	return time.Now()
}

// Schedule initializes a new function run, ensuring that the function will be
// executed via our async execution engine as quickly as possible.
//
// This returns a run ID, metadata for the run, and any errors scheduling.
//
// If the run was impacted by flow control (idempotency, rate limiting, debounce, etc.),
// metadata will be nil.  This will return the original run ID if runs were skipped due
// to idemptoency.
func (s *scheduler) Schedule(ctx context.Context, req execution.ScheduleRequest) (*ulid.ULID, *sv2.Metadata, error) {
	ctx, span := s.conditionalTracer.NewSpan(ctx, "executor.Schedule", req.AccountID, req.WorkspaceID, req.Function.ID)
	defer span.End()

	// Run IDs are created embedding the timestamp now, when the function is being scheduled.
	// When running a cancellation, functions are cancelled at scheduling time based off of
	// this run ID.
	var runID *ulid.ULID

	if req.RunID == nil {
		id := ulid.MustNew(ulid.Now(), rand.Reader)
		runID = &id
	} else {
		runID = req.RunID
	}

	key := idempotencyKey(req, *runID)

	if len(req.Events) == 0 {
		return nil, nil, fmt.Errorf("no events provided in schedule request")
	}

	l := s.log.With(
		"account_id", req.AccountID,
		"env_id", req.WorkspaceID,
		"app_id", req.AppID,
		"fn_id", req.Function.ID,
		"fn_v", req.Function.FunctionVersion,
		"evt_id", req.Events[0].GetInternalID(),
		"run_id", runID,
		"schedule_req", req,
	)

	span.SetAttributes(attribute.String("event_id", req.Events[0].GetInternalID().String()))
	span.SetAttributes(attribute.String("run_id", runID.String()))

	l.Optional(req.AccountID, "schedule").Debug("hitting constraint API")

	// Check constraints and acquire lease
	md, err := WithConstraints(
		ctx,
		s.now(),
		s.capacityManager,
		s.useConstraintAPI,
		req,
		s.conditionalTracer,
		key,
		func(ctx context.Context, performChecks bool) (*sv2.Metadata, error) {
			return util.CritT(ctx, "schedule", func(ctx context.Context) (*sv2.Metadata, error) {
				var (
					md  *sv2.Metadata
					err error
				)
				runID, md, err = s.schedule(ctx, req, *runID, key, performChecks)
				return md, err
			}, util.WithBoundaries(2*time.Second))
		})

	return runID, md, err
}

func (s *scheduler) schedule(
	ctx context.Context,
	req execution.ScheduleRequest,
	runID ulid.ULID,
	// key is the idempotency key
	key string,
	// performChecks determines whether constraint checks must be performed
	// This may be false when the Constraint API was used to enforce constraints.
	performChecks bool,
) (*ulid.ULID, *sv2.Metadata, error) {
	if req.AppID == uuid.Nil {
		return nil, nil, fmt.Errorf("app ID is required to schedule a run")
	}

	ctx, span := s.conditionalTracer.NewSpan(ctx, "executor.schedule", req.AccountID, req.WorkspaceID, req.Function.ID)
	defer span.End()

	l := s.log.With(
		"account_id", req.AccountID,
		"env_id", req.WorkspaceID,
		"app_id", req.AppID,
		"fn_id", req.Function.ID,
		"fn_v", req.Function.FunctionVersion,
		"evt_id", req.Events[0].GetInternalID(),
	)

	if performChecks {
		// Attempt to rate-limit the incoming function.
		if s.rateLimiter != nil && req.Function.RateLimit != nil && !req.PreventRateLimit {
			evtMap := req.Events[0].GetEvent().Map()
			rateLimitKey, err := ratelimit.RateLimitKey(ctx, req.Function.ID, *req.Function.RateLimit, evtMap)

			l.Optional(req.AccountID, "schedule-ratelimit").Debug("ratelimiting schedule", "key", rateLimitKey, "error", err)

			switch err {
			case nil:
				res, err := s.rateLimiter.RateLimit(
					logger.WithStdlib(ctx, l),
					rateLimitKey,
					*req.Function.RateLimit,
					ratelimit.WithNow(s.now()),
					ratelimit.WithIdempotency(key, RateLimitIdempotencyTTL),
				)

				l.Optional(req.AccountID, "schedule-ratelimit").Debug("ratelimiting schedule", "result", res)

				if err != nil {
					metrics.IncrRateLimitUsage(ctx, metrics.CounterOpt{
						PkgName: pkgName,
						Tags: map[string]any{
							"impl":   "lua",
							"status": "error",
						},
					})
					return nil, nil, fmt.Errorf("could not check rate limit: %w", err)
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
					return nil, nil, ErrFunctionRateLimited
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
				return nil, nil, fmt.Errorf("could not evaluate rate limit: %w", err)
			}
		}
	}

	// NOTE: From this point, we are guaranteed to operate within user constraints.

	if req.Function.Debounce != nil && !req.PreventDebounce {
		ctx, span := s.conditionalTracer.NewSpan(ctx, "executor.Debounce", req.AccountID, req.WorkspaceID, req.Function.ID)
		err := s.debouncer.Debounce(ctx, debounce.DebounceItem{
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
			span.RecordError(err)
			span.End()
			return nil, nil, err
		}
		span.End()
		return nil, nil, ErrFunctionDebounced
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
			return nil, nil, fmt.Errorf("error marshalling event: %w", err)
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
		return nil, nil, fmt.Errorf("error marshalling events: %w", err)
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
		metadata.Config.Semaphores = evaluateFnConcurrency(ctx, req.AccountID, req.Function.ID, req.Function.Concurrency.Fn, evtMap)
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
			singletonRunID, err := s.singletonMgr.HandleSingleton(ctx, singletonKey, *req.Function.Singleton, req.AccountID)
			if err != nil {
				return nil, nil, err
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
					err = s.Cancel(ctx, runID, execution.CancelRequest{
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
			return nil, nil, err
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
		if err := reconstruct(ctx, s.traceReader, req, &newState); err != nil {
			return nil, nil, fmt.Errorf("error reconstructing input state: %w", err)
		}
	}

	stv1ID := sv2.V1FromMetadata(metadata)

	// Check if the function should be skipped (paused, draining, backlog limit)
	// Only check if not already marked as skipped (e.g., by singleton)
	if skipReason == enums.SkipReasonNone {
		skipReason = s.skipped(ctx, req)
	}

	// Create run state if not skipped
	if skipReason == enums.SkipReasonNone {
		ctx, span := s.conditionalTracer.NewSpan(ctx, "executor.CreateState", req.AccountID, req.WorkspaceID, req.Function.ID)
		st, err := s.smv2.Create(ctx, newState)
		span.End()

		switch {
		case err == nil: // no-op
		case errors.Is(err, state.ErrIdentifierExists): // no-op
		case errors.Is(err, state.ErrIdentifierTombstone):
			tombstoneRunID := st.Metadata.ID.RunID
			return &tombstoneRunID, nil, ErrFunctionSkippedIdempotency
		default:
			return nil, nil, fmt.Errorf("error creating run state: %w", err)
		}

		// Override existing identifier in case we changed the run ID due to idempotency
		stv1ID = sv2.V1FromMetadata(st.Metadata)

		// NOTE: if the runID mismatches, it means there's already a state available
		// and we need to override the one we already have to make sure we're using
		// the correct metedata values
		if metadata.ID.RunID != stv1ID.RunID {
			id := sv2.IDFromV1(stv1ID)
			metadata, err = s.smv2.LoadMetadata(ctx, id)
			// The run was already completed and GC'd, or was deleted.
			// The idempotency key was used, so skip this run.
			if err != nil && errors.Is(err, state.ErrRunNotFound) {
				// Log with delta to help identify short deltas (like 5ms)
				originalRunCreatedAt := time.UnixMilli(int64(id.RunID.Time()))
				deltaMs := time.Since(originalRunCreatedAt).Milliseconds()
				// This sanitization is not needed but CodeQL complains about it
				sanitizedRunID := util.SanitizeLogField(id.RunID.String())
				l.Warn("idempotency key exists but run state not found",
					"original_run_id", sanitizedRunID,
					"original_run_created_at", originalRunCreatedAt,
					"delta_ms", deltaMs,
				)
				return &stv1ID.RunID, nil, ErrFunctionSkippedIdempotency
			}
			// usually other failures (logged by caller)
			if err != nil {
				return nil, nil, err
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
	runSpanRef, err = s.tracerProvider.CreateDroppableSpan(
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
		return s.handleFunctionSkipped(ctx, req, metadata, evts, skipReason)
	}

	if req.BatchID == nil {

		// Create cancellation pauses immediately, only if this is a non-batch event.
		if len(req.Function.Cancel) > 0 {
			if err := createCancellationPauses(ctx, s.pm, l, s.now(), key, evtMap, metadata.ID, req); err != nil {
				return &metadata.ID.RunID, &metadata, err
			}
		}

		// Add a system job to eager-cancel this function run on timeouts, only if this is a non-batch event.
		if req.Function.Timeouts != nil && req.Function.Timeouts.Start != nil {
			enqueuedAt := ulid.Time(runID.Time())
			if err := createEagerCancellationForTimeout(ctx, s.producer, enqueuedAt, req.Function.Timeouts.StartDuration(), enums.CancellationKindStartTimeout, stv1ID); err != nil {
				return &metadata.ID.RunID, &metadata, err
			}
		}
	}

	at := s.now()
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
		Semaphores:            metadata.Config.Semaphores,
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
		discoverySpanRef, err = s.tracerProvider.CreateDroppableSpan(
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
		for _, e := range s.lifecycles {
			go e.OnFunctionScheduled(context.WithoutCancel(ctx), metadata, item, req.Events)
		}
		return &metadata.ID.RunID, &metadata, nil
	}

	// Schedule for async functons (the default)
	err = s.producer.Enqueue(ctx, item, at, queue.EnqueueOpts{})

	switch err {
	case nil:
		// no-op
	case queue.ErrQueueItemExists:
		// If the item already exists in the queue, we can safely ignore this
		// entire schedule request; it's basically a retry and we should not
		// persist this for the user.
		return &metadata.ID.RunID, nil, state.ErrIdentifierExists

	case queue.ErrQueueItemSingletonExists:
		err := s.smv2.Delete(ctx, sv2.IDFromV1(stv1ID))
		if err != nil {
			l.ReportError(err, "error deleting function state")
		}
		return nil, nil, ErrFunctionSkipped

	default:
		return nil, nil, fmt.Errorf("error enqueueing source edge '%v': %w", queueKey, err)
	}

	sendSpans()
	for _, e := range s.lifecycles {
		go e.OnFunctionScheduled(context.WithoutCancel(ctx), metadata, item, req.Events)
	}

	return &metadata.ID.RunID, &metadata, nil
}

func (s *scheduler) skipped(ctx context.Context, req execution.ScheduleRequest) enums.SkipReason {
	l := logger.StdlibLogger(ctx)

	// Check if function is paused, draining
	skipReason := req.SkipReason()
	if skipReason != enums.SkipReasonNone {
		return skipReason
	}

	// Check if backlog size limit was hit
	res, err := s.checkBacklogSizeLimit(ctx, req)
	if err != nil {
		l.ReportError(err, "error checking backlog size limit")
		return enums.SkipReasonNone
	}

	return res
}

func (s *scheduler) checkBacklogSizeLimit(ctx context.Context, req execution.ScheduleRequest) (enums.SkipReason, error) {
	if s.functionBacklogSizeLimit == nil {
		return enums.SkipReasonNone, nil
	}

	backlogSizeLimit := s.functionBacklogSizeLimit(ctx, req.AccountID, req.WorkspaceID, req.Function.ID)
	if backlogSizeLimit.Limit <= 0 {
		return enums.SkipReasonNone, nil
	}

	scheduledSteps, err := s.jobQueueReader.StatusCount(ctx, req.Function.ID, "start")
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

	for _, ll := range s.lifecycles {
		service.Go(func() {
			ll.OnFunctionBacklogSizeLimitReached(ctx, id)
		})
	}

	if !backlogSizeLimit.Enforce {
		return enums.SkipReasonNone, nil
	}

	return enums.SkipReasonFunctionBacklogSizeLimitHit, nil
}

func (s *scheduler) handleFunctionSkipped(ctx context.Context, req execution.ScheduleRequest, metadata sv2.Metadata, evts []json.RawMessage, reason enums.SkipReason) (*ulid.ULID, *sv2.Metadata, error) {
	for _, e := range s.lifecycles {
		service.Go(
			func() {
				e.OnFunctionSkipped(context.WithoutCancel(ctx), metadata, execution.SkipState{
					CronSchedule: req.Events[0].GetEvent().CronSchedule(),
					Reason:       reason,
					Events:       evts,
				})
			})
	}
	return nil, nil, ErrFunctionSkipped
}

func (s *scheduler) AppendAndScheduleBatch(ctx context.Context, fn inngest.Function, bi batch.BatchItem, opts *execution.BatchExecOpts) error {
	enableInstrumentation := s.enableBatchingInstrumentation != nil && s.enableBatchingInstrumentation(ctx, bi.AccountID, bi.WorkspaceID)
	l := logger.StdlibLogger(ctx).With("eventID", bi.EventID)
	result, err := s.batcher.Append(ctx, bi, fn)
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
		at := s.now().Add(dur)

		if err := s.batcher.ScheduleExecution(ctx, batch.ScheduleBatchOpts{
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
		if err := s.RetrieveAndScheduleBatch(ctx, fn, batch.ScheduleBatchPayload{
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
func (s *scheduler) RetrieveAndScheduleBatch(ctx context.Context, fn inngest.Function, payload batch.ScheduleBatchPayload, opts *execution.BatchExecOpts) error {
	enableInstrumentation := s.enableBatchingInstrumentation != nil && s.enableBatchingInstrumentation(ctx, payload.AccountID, payload.WorkspaceID)
	evtList, err := s.batcher.RetrieveItems(ctx, payload.FunctionID, payload.BatchID)

	l := logger.StdlibLogger(ctx).With("accountID", payload.AccountID, "workspace_id", payload.WorkspaceID, "batchID", payload.BatchID)
	if enableInstrumentation {
		l.Debug("retrieved batch items", "events", len(evtList), "err", err)
	}
	if err != nil {
		return err
	}

	if len(evtList) == 0 {
		l.Warn("batch has no events, skipping schedule", "function_id", payload.FunctionID, "batch_id", payload.BatchID)
		return nil
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
	_, md, err := s.Schedule(ctx, execution.ScheduleRequest{
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
		if err := s.batcher.DeleteKeys(ctx, payload.FunctionID, payload.BatchID); err != nil {
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
