package devserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/coocood/freecache"
	"github.com/eko/gocache/lib/v4/cache"
	freecachestore "github.com/eko/gocache/store/freecache/v4"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/api/apiv1"
	"github.com/inngest/inngest/pkg/backoff"
	"github.com/inngest/inngest/pkg/config"
	_ "github.com/inngest/inngest/pkg/config/defaults"
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/connect"
	pubsub2 "github.com/inngest/inngest/pkg/connect/pubsub"
	connstate "github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi"
	"github.com/inngest/inngest/pkg/cqrs/sqlitecqrs"
	"github.com/inngest/inngest/pkg/deploy"
	"github.com/inngest/inngest/pkg/enums"
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
	"github.com/inngest/inngest/pkg/history_drivers/memory_reader"
	"github.com/inngest/inngest/pkg/history_drivers/memory_writer"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/inngest/inngest/pkg/run"
	"github.com/inngest/inngest/pkg/service"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/util/awsgateway"
	connectproto "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/redis/rueidis"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/sync/errgroup"
)

const (
	DefaultTick         = 150
	DefaultTickDuration = time.Millisecond * DefaultTick
	DefaultPollInterval = 5
)

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

	// SigningKey is used to decide that the server should sign requests and
	// validate responses where applicable, modelling cloud behaviour.
	SigningKey *string `json:"-"`

	// EventKey is used to authorize incoming events, ensuring they match the
	// given key.
	EventKeys []string `json:"-"`

	// RequireKeys defines whether event and signing keys are required for the
	// server to function. If this is true and signing keys are not defined,
	// the server will still boot but core actions such as syncing, runs, and
	// ingesting events will not work.
	RequireKeys bool `json:"require_keys"`
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
	db, err := sqlitecqrs.New(sqlitecqrs.SqliteCQRSOptions{InMemory: true})
	if err != nil {
		return err
	}

	if opts.Tick == 0 {
		opts.Tick = DefaultTickDuration
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

	connectRc, err := createInmemoryRedis(ctx, opts.Tick)
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

	queueShard := redis_state.QueueShard{Name: consts.DefaultQueueShardName, RedisClient: unshardedClient.Queue(), Kind: string(enums.QueueShardKindRedis)}

	shardSelector := func(ctx context.Context, _ uuid.UUID, _ *string) (redis_state.QueueShard, error) {
		return queueShard, nil
	}

	queueShards := map[string]redis_state.QueueShard{
		consts.DefaultQueueShardName: queueShard,
	}

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
		redis_state.WithRunMode(redis_state.QueueRunMode{
			Sequential: true,
			Scavenger:  true,
			Partition:  true,
		}),
		redis_state.WithIdempotencyTTL(time.Hour),
		redis_state.WithNumWorkers(100),
		redis_state.WithPollTick(opts.Tick),
		redis_state.WithCustomConcurrencyKeyLimitRefresher(func(ctx context.Context, i queue.QueueItem) []state.CustomConcurrency {
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
		redis_state.WithConcurrencyLimitGetter(func(ctx context.Context, p redis_state.QueuePartition) redis_state.PartitionConcurrencyLimits {
			// In the dev server, there are never account limits.
			limits := redis_state.PartitionConcurrencyLimits{
				AccountLimit: redis_state.NoConcurrencyLimit,
			}

			// Ensure that we return the correct concurrency values per
			// partition.
			funcs, err := dbcqrs.GetFunctions(ctx)
			if err != nil {
				return redis_state.PartitionConcurrencyLimits{
					AccountLimit:   redis_state.NoConcurrencyLimit,
					FunctionLimit:  consts.DefaultConcurrencyLimit,
					CustomKeyLimit: consts.DefaultConcurrencyLimit,
				}
			}
			for _, fun := range funcs {
				f, _ := fun.InngestFunction()
				if f.ID == uuid.Nil {
					f.ID = f.DeterministicUUID()
				}
				// Update the function's concurrency here with latest defaults
				if p.FunctionID != nil && f.ID == *p.FunctionID && f.Concurrency != nil && f.Concurrency.PartitionConcurrency() > 0 {
					limits.FunctionLimit = f.Concurrency.PartitionConcurrency()
				}
			}
			if p.EvaluatedConcurrencyKey != "" {
				limits.CustomKeyLimit = p.ConcurrencyLimit
			}

			return limits
		}),
		redis_state.WithShardSelector(shardSelector),
		redis_state.WithQueueShardClients(queueShards),
	}
	if opts.RetryInterval > 0 {
		queueOpts = append(queueOpts, redis_state.WithBackoffFunc(
			backoff.GetLinearBackoffFunc(time.Duration(opts.RetryInterval)*time.Second),
		))
	}
	rq := redis_state.NewQueue(queueShard, queueOpts...)

	rl := ratelimit.New(ctx, unshardedRc, "{ratelimit}:")

	batcher := batch.NewRedisBatchManager(shardedClient.Batch(), rq)
	debouncer := debounce.NewRedisDebouncer(unshardedClient.Debounce(), queueShard, rq)

	connectPubSubRedis := createConnectPubSubRedis()
	gatewayProxy, err := pubsub2.NewConnector(ctx, pubsub2.WithRedis(connectPubSubRedis, logger.StdlibLoggerWithCustomVarName(ctx, "CONNECT_PUBSUB_LOG_LEVEL"), true))
	if err != nil {
		return fmt.Errorf("failed to create connect pubsub connector: %w", err)
	}

	connectionManager := connstate.NewRedisConnectionStateManager(connectRc)

	// Create a new expression aggregator, using Redis to load evaluables.
	agg := expressions.NewAggregator(ctx, 100, 100, sm.(expressions.EvaluableLoader), nil)

	var drivers = []driver.Driver{}
	for _, driverConfig := range opts.Config.Execution.Drivers {
		d, err := driverConfig.NewDriver(registration.NewDriverOpts{
			ConnectForwarder: gatewayProxy,
		})
		if err != nil {
			return err
		}
		drivers = append(drivers, d)
	}
	pb, err := pubsub.NewPublisher(ctx, opts.Config.EventStream.Service)
	if err != nil {
		return fmt.Errorf("failed to create publisher: %w", err)
	}

	hmw := memory_writer.NewWriter(ctx, memory_writer.WriterOptions{DumpToFile: false})

	exec, err := executor.NewExecutor(
		executor.WithStateManager(smv2),
		executor.WithPauseManager(sm),
		executor.WithRuntimeDrivers(
			drivers...,
		),
		executor.WithExpressionAggregator(agg),
		executor.WithQueue(rq),
		executor.WithLogger(logger.From(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithLifecycleListeners(
			history.NewLifecycleListener(
				nil,
				hd,
				hmw,
			),
			Lifecycle{
				Cqrs:       dbcqrs,
				Pb:         pb,
				EventTopic: opts.Config.EventStream.Service.Concrete.TopicName(),
			},
			run.NewTraceLifecycleListener(nil),
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
		executor.WithAssignedQueueShard(queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTraceReader(dbcqrs),
	)
	if err != nil {
		return err
	}

	// Create an executor.
	executorSvc := executor.NewService(
		opts.Config,
		executor.WithExecutionManager(dbcqrs),
		executor.WithState(sm),
		executor.WithServiceQueue(rq),
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
		runner.WithRunnerQueue(rq),
		runner.WithTracker(t),
		runner.WithRateLimiter(rl),
		runner.WithBatchManager(batcher),
		runner.WithPublisher(pb),
	)

	// The devserver embeds the event API.
	ds := NewService(opts, runner, dbcqrs, pb, stepLimitOverrides, stateSizeLimitOverrides, unshardedRc, hmw, nil)
	// embed the tracker
	ds.Tracker = t
	ds.State = sm
	ds.Queue = rq
	ds.Executor = exec
	// start the API
	// Create a new API endpoint which hosts SDK-related functionality for
	// registering functions.
	devAPI := NewDevAPI(ds)

	devAPI.Route("/v1", func(r chi.Router) {
		// Add the V1 API to our dev server API.
		cache := cache.New[[]byte](freecachestore.NewFreecache(freecache.NewCache(1024 * 1024)))
		caching := apiv1.NewCacheMiddleware(cache)

		apiv1.AddRoutes(r, apiv1.Opts{
			CachingMiddleware: caching,
			EventReader:       ds.Data,
			FunctionReader:    ds.Data,
			FunctionRunReader: ds.Data,
			JobQueueReader:    ds.Queue.(queue.JobQueueReader),
			Executor:          ds.Executor,
		})
	})

	// ds.opts.Config.EventStream.Service.TopicName()

	core, err := coreapi.NewCoreApi(coreapi.Options{
		Data:          ds.Data,
		Config:        ds.Opts.Config,
		Logger:        logger.From(ctx),
		Runner:        ds.Runner,
		Tracker:       ds.Tracker,
		State:         ds.State,
		Queue:         ds.Queue,
		EventHandler:  ds.HandleEvent,
		Executor:      ds.Executor,
		HistoryReader: memory_reader.NewReader(),
	})
	if err != nil {
		return err
	}

	connGateway, connRouter, connectHandler := connect.NewConnectGatewayService(
		connect.WithConnectionStateManager(connectionManager),
		connect.WithRequestReceiver(gatewayProxy),
		connect.WithGatewayAuthHandler(func(ctx context.Context, data *connectproto.WorkerConnectRequestData) (*connect.AuthResponse, error) {
			return &connect.AuthResponse{
				AccountID: consts.DevServerAccountId,
				EnvID:     consts.DevServerEnvId,
			}, nil
		}),
		connect.WithAppLoader(dbcqrs),
		connect.WithDev(),
	)

	// Create a new data API directly in the devserver.  This allows us to inject
	// the data API into the dev server port, providing a single router for the dev
	// server UI, events, and API for loading data.
	//
	// Merge the dev server API (for handling files & registration) with the data
	// API into the event API router.
	ds.Apiservice = api.NewService(api.APIServiceOptions{
		Config: ds.Opts.Config,
		Mounts: []api.Mount{
			{At: "/", Router: devAPI},
			{At: "/v0", Router: core.Router},
			{At: "/debug", Handler: middleware.Profiler()},
			{At: "/connect", Handler: connectHandler},
		},
		LocalEventKeys: opts.EventKeys,
	})

	svcs := []service.Service{ds, runner, executorSvc, ds.Apiservice}
	svcs = append(svcs, connGateway, connRouter)
	return service.StartAll(ctx, svcs...)
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

	// If tick is lower than the default, tick every 50ms.  This lets us save
	// CPU for standard dev-server testing.
	poll := time.Second
	if tick < DefaultTickDuration {
		poll = time.Millisecond * 50
	}

	go func() {
		for range time.Tick(poll) {
			r.FastForward(poll)
		}
	}()
	return rc, nil
}

func createConnectPubSubRedis() rueidis.ClientOption {
	r := miniredis.NewMiniRedis()
	_ = r.Start()
	return rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	}
}

func getSendingEventHandler(ctx context.Context, pb pubsub.Publisher, topic string) execution.HandleSendingEvent {
	return func(ctx context.Context, evt event.Event, item queue.Item) error {
		trackedEvent := event.NewOSSTrackedEvent(evt)
		byt, err := json.Marshal(trackedEvent)
		if err != nil {
			return fmt.Errorf("error marshalling invocation event: %w", err)
		}

		carrier := itrace.NewTraceCarrier()
		itrace.UserTracer().Propagator().Inject(ctx, propagation.MapCarrier(carrier.Context))

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
