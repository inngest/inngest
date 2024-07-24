package devserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/backoff"
	"github.com/inngest/inngest/pkg/config"
	_ "github.com/inngest/inngest/pkg/config/defaults"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs/sqlitecqrs"
	"github.com/inngest/inngest/pkg/deploy"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/driver/httpdriver"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/history_drivers/memory_writer"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry"
	"github.com/inngest/inngest/pkg/util/awsgateway"
	"github.com/redis/rueidis"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/sync/errgroup"
)

const defaultTick = time.Millisecond * 150

// StartOpts configures the dev server
type StartOpts struct {
	Config        config.Config `json:"-"`
	RootDir       string        `json:"dir"`
	URLs          []string      `json:"urls"`
	Autodiscover  bool          `json:"autodiscover"`
	Poll          bool          `json:"poll"`
	PollInterval  int           `json:"poll_interval"`
	Tick          time.Duration `json:"tick"`
	RetryInterval int           `json:"retry_interval"`
}

// Create and start a new dev server.  The dev server is used during (surprise surprise)
// development.
//
// It runs all available services from `inngest serve`, plus:
// - Adds development-specific APIs for communicating with the SDK.
func New(ctx context.Context, opts StartOpts) error {
	// The dev server _always_ logs output for development.
	if !opts.Config.Execution.LogOutput {
		opts.Config.Execution.LogOutput = true
	}

	// NOTE: looks deprecated?
	// Before running the development service, ensure that we change the http
	// driver in development to use our AWS Gateway http client, attempting to
	// automatically transform dev requests to lambda invocations.
	httpdriver.DefaultExecutor.Client.Transport = awsgateway.NewTransformTripper(httpdriver.DefaultExecutor.Client.Transport)
	deploy.Client.Transport = awsgateway.NewTransformTripper(deploy.Client.Transport)

	return start(ctx, opts)
}

func start(ctx context.Context, opts StartOpts) error {
	db, err := sqlitecqrs.New()
	if err != nil {
		return err
	}

	if opts.Tick == 0 {
		opts.Tick = defaultTick
	}

	// Initialize the devserver
	dbcqrs := sqlitecqrs.NewCQRS(db)
	hd := sqlitecqrs.NewHistoryDriver(db)
	loader := dbcqrs.(state.FunctionLoader)

	stepLimitOverrides := make(map[string]int)
	stateSizeLimitOverrides := make(map[string]int)

	shardedRc, err := createInmemoryRedis(ctx, opts.Tick)
	if err != nil {
		return err
	}

	unshardedRc, err := createInmemoryRedis(ctx, opts.Tick)
	if err != nil {
		return err
	}

	unshardedClient := redis_state.NewUnshardedClient(unshardedRc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	shardedClient := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient:        unshardedClient,
		FunctionRunStateClient: shardedRc,
		StateDefaultKey:        redis_state.StateDefaultKey,
		FnRunIsSharded:         redis_state.AlwaysShardOnRun,
		BatchClient:            shardedRc,
		QueueDefaultKey:        redis_state.QueueDefaultKey,
	})

	var sm state.Manager
	t := runner.NewTracker()
	sm, err = redis_state.New(
		ctx,
		redis_state.WithShardedClient(shardedClient),
		redis_state.WithUnshardedClient(unshardedClient),
	)
	if err != nil {
		return err
	}
	smv2 := redis_state.MustRunServiceV2(sm)

	queueOpts := []redis_state.QueueOpt{
		redis_state.WithIdempotencyTTL(time.Hour),
		redis_state.WithNumWorkers(100),
		redis_state.WithPollTick(opts.Tick),
		redis_state.WithCustomConcurrencyKeyGenerator(func(ctx context.Context, i redis_state.QueueItem) []state.CustomConcurrency {
			keys := i.Data.GetConcurrencyKeys()

			fn, err := dbcqrs.GetFunctionByInternalUUID(ctx, i.Data.Identifier.WorkspaceID, i.Data.Identifier.WorkflowID)
			if err != nil {
				// Use what's stored in the state store.
				return keys
			}
			f, err := fn.InngestFunction()
			if err != nil {
				return keys
			}

			if f.Concurrency != nil {
				for _, c := range f.Concurrency.Limits {
					if !c.IsCustomLimit() {
						continue
					}
					// If there's a concurrency key with the same hash, use the new function's
					// concurrency limits.
					//
					// NOTE:  This is accidentally quadratic but is okay as we bound concurrency
					// keys to a low value (2-3).
					for n, actual := range keys {
						if actual.Hash != "" && actual.Hash == c.Hash {
							actual.Limit = c.Limit
							keys[n] = actual
						}
					}
				}
			}

			return keys
		}),
		redis_state.WithConcurrencyLimitGetter(func(ctx context.Context, p redis_state.QueuePartition) (account, fn, custom int) {
			// In the dev server, there are never account limits.
			account = -1

			// Ensure that we return the correct concurrency values per
			// partition.
			funcs, err := dbcqrs.GetFunctions(ctx)
			if err != nil {
				return -1, consts.DefaultConcurrencyLimit, -1
			}
			for _, fun := range funcs {
				f, _ := fun.InngestFunction()
				if f.ID == uuid.Nil {
					f.ID = inngest.DeterministicUUID(*f)
				}
				// Update the function's concurrency here with latest defaults
				if p.FunctionID != nil && f.ID == *p.FunctionID && f.Concurrency != nil && f.Concurrency.PartitionConcurrency() > 0 {
					fn = f.Concurrency.PartitionConcurrency()
				}
			}
			if p.ConcurrencyKey != "" {
				custom = p.ConcurrencyLimit
			}

			return account, fn, custom
		}),
	}
	if opts.RetryInterval > 0 {
		queueOpts = append(queueOpts, redis_state.WithBackoffFunc(
			backoff.GetLinearBackoffFunc(time.Duration(opts.RetryInterval)*time.Second),
		))
	}
	queue := redis_state.NewQueue(unshardedClient.Queue(), queueOpts...)

	rl := ratelimit.New(ctx, unshardedRc, "{ratelimit}:")

	batcher := batch.NewRedisBatchManager(shardedClient.Batch(), queue)
	debouncer := debounce.NewRedisDebouncer(unshardedClient.Debounce(), queue)

	// Create a new expression aggregator, using Redis to load evaluables.
	agg := expressions.NewAggregator(ctx, 100, 100, sm.(expressions.EvaluableLoader), nil)

	var drivers = []driver.Driver{}
	for _, driverConfig := range opts.Config.Execution.Drivers {
		d, err := driverConfig.NewDriver()
		if err != nil {
			return err
		}
		drivers = append(drivers, d)
	}
	pb, err := pubsub.NewPublisher(ctx, opts.Config.EventStream.Service)
	if err != nil {
		return fmt.Errorf("failed to create publisher: %w", err)
	}

	exec, err := executor.NewExecutor(
		executor.WithStateManager(smv2),
		executor.WithPauseManager(sm),
		executor.WithRuntimeDrivers(
			drivers...,
		),
		executor.WithExpressionAggregator(agg),
		executor.WithQueue(queue),
		executor.WithLogger(logger.From(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithLifecycleListeners(
			history.NewLifecycleListener(
				nil,
				hd,
				memory_writer.NewWriter(),
			),
			lifecycle{
				cqrs:       dbcqrs,
				pb:         pb,
				eventTopic: opts.Config.EventStream.Service.Concrete.TopicName(),
			},
			executor.NewTraceRunLifecycleListener(
				nil,
				smv2,
			),
		),
		executor.WithStepLimits(func(id sv2.ID) int {
			if override, hasOverride := stepLimitOverrides[id.FunctionID.String()]; hasOverride {
				logger.From(ctx).Warn().Msgf("Using step limit override of %d for %q\n", override, id.FunctionID)
				return override
			}

			return consts.DefaultMaxStepLimit
		}),
		executor.WithStateSizeLimits(func(id sv2.ID) int {
			if override, hasOverride := stateSizeLimitOverrides[id.FunctionID.String()]; hasOverride {
				logger.From(ctx).Warn().Msgf("Using state size limit override of %d for %q\n", override, id.FunctionID)
				return override
			}

			return consts.DefaultMaxStateSizeLimit
		}),
		executor.WithInvokeFailHandler(getInvokeFailHandler(ctx, pb, opts.Config.EventStream.Service.Concrete.TopicName())),
		executor.WithSendingEventHandler(getSendingEventHandler(ctx, pb, opts.Config.EventStream.Service.Concrete.TopicName())),
		executor.WithDebouncer(debouncer),
		executor.WithBatcher(batcher),
	)
	if err != nil {
		return err
	}

	// Create an executor.
	executorSvc := executor.NewService(
		opts.Config,
		executor.WithExecutionManager(dbcqrs),
		executor.WithState(sm),
		executor.WithServiceQueue(queue),
		executor.WithServiceExecutor(exec),
		executor.WithServiceBatcher(batcher),
		executor.WithServiceDebouncer(debouncer),
	)

	runner := runner.NewService(
		opts.Config,
		runner.WithCQRS(dbcqrs),
		runner.WithExecutor(exec),
		runner.WithExecutionManager(dbcqrs),
		runner.WithEventManager(event.NewManager()),
		runner.WithStateManager(sm),
		runner.WithRunnerQueue(queue),
		runner.WithTracker(t),
		runner.WithRateLimiter(rl),
		runner.WithBatchManager(batcher),
		runner.WithPublisher(pb),
	)

	// The devserver embeds the event API.
	ds := newService(opts, runner, dbcqrs, pb, stepLimitOverrides, stateSizeLimitOverrides)
	// embed the tracker
	ds.tracker = t
	ds.state = sm
	ds.queue = queue
	ds.executor = exec

	return service.StartAll(ctx, ds, runner, executorSvc)
}

func createInmemoryRedis(ctx context.Context, tick time.Duration) (rueidis.Client, error) {
	r := miniredis.NewMiniRedis()
	_ = r.Start()
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	if err != nil {
		return nil, err
	}

	// If tick is lower than 250ms, tick every 100ms.  This lets us save
	// CPU for standard dev-server testing.
	poll := time.Second
	if tick < defaultTick {
		poll = time.Millisecond * 50
	}

	go func() {
		for range time.Tick(poll) {
			r.FastForward(poll)
		}
	}()
	return rc, nil
}

func getSendingEventHandler(ctx context.Context, pb pubsub.Publisher, topic string) execution.HandleSendingEvent {
	return func(ctx context.Context, evt event.Event, item queue.Item) error {
		trackedEvent := event.NewOSSTrackedEvent(evt)
		byt, err := json.Marshal(trackedEvent)
		if err != nil {
			return fmt.Errorf("error marshalling invocation event: %w", err)
		}

		carrier := telemetry.NewTraceCarrier()
		telemetry.UserTracer().Propagator().Inject(ctx, propagation.MapCarrier(carrier.Context))

		err = pb.Publish(
			ctx,
			topic,
			pubsub.Message{
				Name:      event.EventReceivedName,
				Data:      string(byt),
				Timestamp: time.Now(),
				Metadata: map[string]any{
					consts.OtelPropagationKey: carrier,
				},
			},
		)
		if err != nil {
			return fmt.Errorf("error publishing invocation event: %w", err)
		}

		return nil
	}
}

func getInvokeFailHandler(ctx context.Context, pb pubsub.Publisher, topic string) execution.InvokeFailHandler {
	return func(ctx context.Context, opts execution.InvokeFailHandlerOpts, evts []event.Event) error {
		eg := errgroup.Group{}

		for _, e := range evts {
			evt := e
			eg.Go(func() error {
				trackedEvent := event.NewOSSTrackedEvent(evt)
				byt, err := json.Marshal(trackedEvent)
				if err != nil {
					return fmt.Errorf("error marshalling function finished event: %w", err)
				}

				err = pb.Publish(
					ctx,
					topic,
					pubsub.Message{
						Name:      event.EventReceivedName,
						Data:      string(byt),
						Timestamp: trackedEvent.GetEvent().Time(),
					},
				)
				if err != nil {
					return fmt.Errorf("error publishing function finished event: %w", err)
				}

				return nil
			})
		}

		return eg.Wait()
	}
}
