package devserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
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
	"github.com/inngest/inngest/pkg/authn"
	"github.com/inngest/inngest/pkg/backoff"
	"github.com/inngest/inngest/pkg/config"
	_ "github.com/inngest/inngest/pkg/config/defaults"
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/connect"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/connect/lifecycles"
	connectpubsub "github.com/inngest/inngest/pkg/connect/pubsub"
	connectv0 "github.com/inngest/inngest/pkg/connect/rest/v0"
	connstate "github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
	"github.com/inngest/inngest/pkg/debugapi"
	"github.com/inngest/inngest/pkg/deploy"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/inngest/inngest/pkg/execution/realtime"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/execution/singleton"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/expressions/expragg"
	"github.com/inngest/inngest/pkg/history_drivers/memory_reader"
	"github.com/inngest/inngest/pkg/history_drivers/memory_writer"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/metrics"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/inngest/inngest/pkg/run"
	"github.com/inngest/inngest/pkg/service"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/testapi"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/pkg/util/awsgateway"
	"github.com/redis/rueidis"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/sync/errgroup"
)

const (
	DefaultTick               = 150
	DefaultTickDuration       = time.Millisecond * DefaultTick
	DefaultPollInterval       = 5
	DefaultQueueWorkers       = 100
	DefaultConnectGatewayPort = 8289
)

var defaultPartitionConstraintConfig = redis_state.PartitionConstraintConfig{
	Concurrency: redis_state.PartitionConcurrency{
		SystemConcurrency:   consts.DefaultConcurrencyLimit,
		AccountConcurrency:  consts.DefaultConcurrencyLimit,
		FunctionConcurrency: consts.DefaultConcurrencyLimit,
	},
}

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
	QueueWorkers  int           `json:"queue_workers"`

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

	ConnectGatewayPort int    `json:"connectGatewayPort"`
	ConnectGatewayHost string `json:"connectGatewayHost"`

	NoUI bool

	// InMemory controls whether to only use in-memory databases (as opposed to
	// filesystem)
	InMemory bool

	// RedisURI allows connecting to external Redis instead of in-memory Redis
	RedisURI string `json:"redis_uri"`

	// PostgresURI allows connecting to external Postgres instead of SQLite
	PostgresURI string `json:"postgres_uri"`

	// SQLiteDir specifies where SQLite files should be stored
	SQLiteDir string `json:"sqlite_dir"`
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

	return start(ctx, opts)
}

func enforceConnectLeaseExpiry(ctx context.Context, accountID uuid.UUID) bool {
	return os.Getenv("INNGEST_CONNECT_DISABLE_ENFORCE_LEASE_EXPIRY") != "true"
}

func start(ctx context.Context, opts StartOpts) error {
	l := logger.StdlibLogger(ctx)
	ctx = logger.WithStdlib(ctx, l)

	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{
		InMemory:    opts.InMemory,
		PostgresURI: opts.PostgresURI,
		Directory:   opts.SQLiteDir,
	})
	if err != nil {
		return err
	}

	if opts.Tick == 0 {
		opts.Tick = DefaultTickDuration
	}

	// Initialize the devserver
	dbDriver := "sqlite"
	if opts.PostgresURI != "" {
		dbDriver = "postgres"
	}
	dbcqrs := base_cqrs.NewCQRS(db, dbDriver)
	hd := base_cqrs.NewHistoryDriver(db, dbDriver)
	loader := dbcqrs.(state.FunctionLoader)

	stepLimitOverrides := make(map[string]int)
	stateSizeLimitOverrides := make(map[string]int)

	var shardedRc, unshardedRc, connectRc rueidis.Client
	var shardedCluster, unshardedCluster, connectCluster *miniredis.Miniredis

	if opts.RedisURI != "" {
		// Use external Redis
		// Mask Redis URI credentials before logging
		loggedURI := ""
		if u, parseErr := url.Parse(opts.RedisURI); parseErr == nil {
			loggedURI = " " + u.Redacted()
		}
		l.Info("using external redis", "url", loggedURI)

		shardedRc, err = connectToOrCreateRedis(opts.RedisURI)
		if err != nil {
			return err
		}
		unshardedRc, err = connectToOrCreateRedis(opts.RedisURI)
		if err != nil {
			return err
		}
		connectRcOpt, err := connectToOrCreateRedisOption(opts.RedisURI)
		if err != nil {
			return err
		}
		connectRc, err = rueidis.NewClient(connectRcOpt)
		if err != nil {
			return err
		}
	} else {
		// Use in-memory Redis
		shardedRc, shardedCluster, err = createInmemoryRedis(ctx, opts.Tick)
		if err != nil {
			return err
		}
		unshardedRc, unshardedCluster, err = createInmemoryRedis(ctx, opts.Tick)
		if err != nil {
			return err
		}
		connectRc, connectCluster, err = createInmemoryRedis(ctx, opts.Tick)
		if err != nil {
			return err
		}
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
	sm, err = redis_state.New(
		ctx,
		redis_state.WithShardedClient(shardedClient),
		redis_state.WithUnshardedClient(unshardedClient),
	)
	if err != nil {
		return err
	}
	smv2 := redis_state.MustRunServiceV2(sm)

	// Create a new broadcaster which lets us broadcast realtime messages.
	broadcaster := realtime.NewInProcessBroadcaster()

	runMode := redis_state.QueueRunMode{
		Sequential:    true,
		Scavenger:     true,
		Partition:     true,
		Continuations: true,
	}
	enableKeyQueues := os.Getenv("EXPERIMENTAL_KEY_QUEUES_ENABLE") == "true"

	if enableKeyQueues {
		runMode.ShadowPartition = true
		runMode.AccountShadowPartition = true
		runMode.AccountShadowPartitionWeight = 80
		runMode.ShadowContinuations = true
		runMode.ShadowContinuationSkipProbability = consts.QueueContinuationSkipProbability
		runMode.NormalizePartition = true
	}

	queueOpts := []redis_state.QueueOpt{
		redis_state.WithRunMode(runMode),
		redis_state.WithIdempotencyTTL(time.Hour),
		redis_state.WithNumWorkers(int32(opts.QueueWorkers)),
		redis_state.WithPollTick(opts.Tick),
		redis_state.WithShadowPollTick(2 * opts.Tick),
		redis_state.WithBacklogNormalizePollTick(5 * opts.Tick),

		redis_state.WithLogger(l),

		redis_state.WithShardSelector(shardSelector),
		redis_state.WithQueueShardClients(queueShards),
		//redis_state.WithKindToQueueMapping(map[string]string{
		//	queue.KindPause:           queue.KindPause,
		//	queue.KindDebounce:        queue.KindDebounce,
		//	queue.KindQueueMigrate:    queue.KindQueueMigrate,
		//	queue.KindPauseBlockFlush: queue.KindPauseBlockFlush,
		//	queue.KindScheduleBatch:   queue.KindScheduleBatch,
		//}),

		// Key queues
		redis_state.WithNormalizeRefreshItemCustomConcurrencyKeys(NormalizeConcurrencyKeys(smv2, dbcqrs)),
		redis_state.WithRefreshItemThrottle(NormalizeThrottle(smv2, dbcqrs)),
		redis_state.WithPartitionConstraintConfigGetter(PartitionConstraintConfigGetter(dbcqrs)),

		redis_state.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return enableKeyQueues
		}),
		redis_state.WithEnqueueSystemPartitionsToBacklog(false),
		redis_state.WithDisableLeaseChecksForSystemQueues(enableKeyQueues),
		redis_state.WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
			return enableKeyQueues
		}),
		redis_state.WithBacklogRefillLimit(10),
	}
	if opts.RetryInterval > 0 {
		queueOpts = append(queueOpts, redis_state.WithBackoffFunc(
			backoff.GetLinearBackoffFunc(time.Duration(opts.RetryInterval)*time.Second),
		))
	}
	rq := redis_state.NewQueue(queueShard, queueOpts...)

	rl := ratelimit.New(ctx, unshardedRc, "{ratelimit}:")

	batcher := batch.NewRedisBatchManager(shardedClient.Batch(), rq, batch.WithLogger(l))
	debouncer := debounce.NewRedisDebouncer(unshardedClient.Debounce(), queueShard, rq)

	sn := singleton.New(ctx, queueShards, shardSelector)

	conditionalTracer := itrace.NewConditionalTracer(itrace.ConnectTracer(), itrace.AlwaysTrace)

	connectPubSubRedis := createConnectPubSubRedis()
	connectPubSubLogger := logger.StdlibLoggerWithCustomVarName(ctx, "CONNECT_PUBSUB_LOG_LEVEL")

	connectionManager := connstate.NewRedisConnectionStateManager(connectRc)

	// Create a new expression aggregator, using Redis to load evaluables.
	agg := expragg.NewAggregator(ctx, 100, 100, sm.(expragg.EvaluableLoader), expressions.ExprEvaluator, nil, nil)

	executorLogger := connectPubSubLogger.With("svc", "executor")
	gatewayGRPCForwarder := connectpubsub.NewGatewayGRPCManager(ctx, connectionManager, connectpubsub.WithLogger(executorLogger))

	executorProxy, err := connectpubsub.NewConnector(ctx, connectpubsub.WithRedis(connectPubSubRedis, true, connectpubsub.RedisPubSubConnectorOpts{
		Logger:             executorLogger,
		Tracer:             conditionalTracer,
		StateManager:       connectionManager,
		EnforceLeaseExpiry: enforceConnectLeaseExpiry,
		GatewayGRPCManager: gatewayGRPCForwarder,
	}))
	if err != nil {
		return fmt.Errorf("failed to create connect pubsub connector: %w", err)
	}

	// Before running the development service, ensure that we change the http
	// driver in development to use our AWS Gateway http client, attempting to
	// automatically transform dev requests to lambda invocations.
	//
	// We also make sure to allow local requests.
	httpClient := exechttp.Client(
		exechttp.SecureDialerOpts{
			AllowHostDocker: true, // In local dev, this is OK
			AllowPrivate:    true, // In local dev, this is OK
			AllowNAT64:      true, // In local dev, this is OK
		},
		// Enable publishing of any requests made directly from the dev server.  Note that this
		// is different from the cloud.
		exechttp.WithRealtimePublishing(),
	)

	httpClient.Client.Transport = awsgateway.NewTransformTripper(httpClient.Client.Transport)
	deploy.Client.Transport = awsgateway.NewTransformTripper(deploy.Client.Transport)

	drivers := []driver.Driver{}
	for _, driverConfig := range opts.Config.Execution.Drivers {
		d, err := driverConfig.NewDriver(registration.NewDriverOpts{
			ConnectForwarder:       executorProxy,
			ConditionalTracer:      conditionalTracer,
			HTTPClient:             httpClient,
			LocalSigningKey:        opts.SigningKey,
			RequireLocalSigningKey: opts.RequireKeys,
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
		executor.WithHTTPClient(httpClient),
		executor.WithStateManager(smv2),
		executor.WithPauseManager(pauses.NewRedisOnlyManager(sm)),
		executor.WithRuntimeDrivers(
			drivers...,
		),
		executor.WithExpressionAggregator(agg),
		executor.WithQueue(rq),
		executor.WithLogger(l),
		executor.WithFunctionLoader(loader),
		executor.WithRealtimePublisher(broadcaster),
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
				l.Warn("using step limit override", "override", override, "fn_id", id.FunctionID)
				return override
			}

			return consts.DefaultMaxStepLimit
		}),
		executor.WithStateSizeLimits(func(id sv2.ID) int {
			if override, hasOverride := stateSizeLimitOverrides[id.FunctionID.String()]; hasOverride {
				l.Warn("using state size limit override", "override", override, "fn_id", id.FunctionID)
				return override
			}

			return consts.DefaultMaxStateSizeLimit
		}),
		executor.WithInvokeFailHandler(getInvokeFailHandler(ctx, pb, opts.Config.EventStream.Service.Concrete.TopicName())),
		executor.WithSendingEventHandler(getSendingEventHandler(ctx, pb, opts.Config.EventStream.Service.Concrete.TopicName())),
		executor.WithDebouncer(debouncer),
		executor.WithSingletonManager(sn),
		executor.WithBatcher(batcher),
		executor.WithAssignedQueueShard(queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTraceReader(dbcqrs),
		executor.WithRealtimeConfig(executor.ExecutorRealtimeConfig{
			Secret:     consts.DevServerRealtimeJWTSecret,
			PublishURL: fmt.Sprintf("http://%s:%d/v1/realtime/publish", opts.Config.CoreAPI.Addr, opts.Config.CoreAPI.Port),
		}),
		executor.WithTracerProvider(tracing.NewSqlcTracerProvider(base_cqrs.NewQueries(db, dbDriver))),
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
		executor.WithServiceLogger(l),
		executor.WithServiceShardSelector(shardSelector),
		executor.WithServiceEnableKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return enableKeyQueues
		}),
	)

	runner := runner.NewService(
		opts.Config,
		runner.WithCQRS(dbcqrs),
		runner.WithExecutor(exec),
		runner.WithExecutionManager(dbcqrs),
		runner.WithPauseManager(pauses.NewRedisOnlyManager(sm)),
		runner.WithStateManager(sm),
		runner.WithRunnerQueue(rq),
		runner.WithRateLimiter(rl),
		runner.WithBatchManager(batcher),
		runner.WithPublisher(pb),
		runner.WithLogger(l),
	)

	// The devserver embeds the event API.
	ds := NewService(opts, runner, dbcqrs, pb, stepLimitOverrides, stateSizeLimitOverrides, unshardedRc, hmw, nil)
	ds.State = sm
	ds.Queue = rq
	ds.Executor = exec
	// start the API
	// Create a new API endpoint which hosts SDK-related functionality for
	// registering functions.
	devAPI := NewDevAPI(ds, DevAPIOptions{AuthMiddleware: authn.SigningKeyMiddleware(opts.SigningKey), disableUI: opts.NoUI})

	devAPI.Route("/v1", func(r chi.Router) {
		// Add the V1 API to our dev server API.
		cache := cache.New[[]byte](freecachestore.NewFreecache(freecache.NewCache(1024 * 1024)))
		caching := apiv1.NewCacheMiddleware(cache)

		apiv1.AddRoutes(r, apiv1.Opts{
			AuthMiddleware:     authn.SigningKeyMiddleware(opts.SigningKey),
			CachingMiddleware:  caching,
			FunctionReader:     ds.Data,
			FunctionRunReader:  ds.Data,
			JobQueueReader:     ds.Queue.(queue.JobQueueReader),
			Executor:           ds.Executor,
			QueueShardSelector: shardSelector,
			Broadcaster:        broadcaster,
			RealtimeJWTSecret:  consts.DevServerRealtimeJWTSecret,
			TraceReader:        ds.Data,
		})
	})

	// ds.opts.Config.EventStream.Service.TopicName()
	apiConnectProxy, err := connectpubsub.NewConnector(ctx, connectpubsub.WithRedis(connectPubSubRedis, false, connectpubsub.RedisPubSubConnectorOpts{
		Logger:             connectPubSubLogger.With("svc", "api"),
		Tracer:             conditionalTracer,
		StateManager:       connectionManager,
		EnforceLeaseExpiry: enforceConnectLeaseExpiry,
	}))
	if err != nil {
		return fmt.Errorf("failed to create connect pubsub connector: %w", err)
	}

	core, err := coreapi.NewCoreApi(coreapi.Options{
		AuthMiddleware: authn.SigningKeyMiddleware(opts.SigningKey),
		Data:           ds.Data,
		Config:         ds.Opts.Config,
		Logger:         l,
		Runner:         ds.Runner,
		State:          ds.State,
		Queue:          ds.Queue,
		EventHandler:   ds.HandleEvent,
		Executor:       ds.Executor,
		HistoryReader:  memory_reader.NewReader(),
		DisableGraphQL: &opts.NoUI,
		ConnectOpts: connectv0.Opts{
			GroupManager:               connectionManager,
			ConnectManager:             connectionManager,
			ConnectResponseNotifier:    apiConnectProxy,
			ConnectRequestStateManager: connectionManager,
			Signer:                     auth.NewJWTSessionTokenSigner(consts.DevServerConnectJwtSecret),
			RequestAuther:              ds,
			ConnectGatewayRetriever:    ds,
			Dev:                        true,
			EntitlementProvider:        ds,
			ConditionalTracer:          conditionalTracer,
			ShouldUseGRPC: func(ctx context.Context, accountID uuid.UUID) bool {
				return false
			},
		},
	})
	if err != nil {
		return err
	}

	debugapi, err := debugapi.NewDebugAPI(debugapi.Opts{
		Log:           l,
		Queue:         rq,
		ShardSelector: shardSelector,
	})
	if err != nil {
		return err
	}

	connectGatewayProxy, err := connectpubsub.NewConnector(ctx, connectpubsub.WithRedis(connectPubSubRedis, false, connectpubsub.RedisPubSubConnectorOpts{
		Logger:             connectPubSubLogger.With("svc", "connect-gateway"),
		Tracer:             conditionalTracer,
		StateManager:       connectionManager,
		EnforceLeaseExpiry: enforceConnectLeaseExpiry,
	}))
	if err != nil {
		return fmt.Errorf("failed to create connect pubsub connector: %w", err)
	}

	connGateway := connect.NewConnectGatewayService(
		connect.WithConnectionStateManager(connectionManager),
		connect.WithRequestReceiver(connectGatewayProxy),
		connect.WithGatewayAuthHandler(auth.NewJWTAuthHandler(consts.DevServerConnectJwtSecret)),
		connect.WithDev(),
		connect.WithGatewayPublicPort(opts.ConnectGatewayPort),
		connect.WithApiBaseUrl(fmt.Sprintf("http://%s:%d", opts.Config.CoreAPI.Addr, opts.Config.CoreAPI.Port)),
		connect.WithLifeCycles(
			[]connect.ConnectGatewayLifecycleListener{
				lifecycles.NewHistoryLifecycle(dbcqrs),
			}),
	)

	// Initialize metrics API for Prometheus-compatible metrics endpoint.
	// This provides system queue depth metrics via /metrics endpoint.
	metricsAPI, err := metrics.NewMetricsAPI(metrics.Opts{
		AuthMiddleware: authn.SigningKeyMiddleware(opts.SigningKey),
		QueueManager:   rq,
	})
	if err != nil {
		return err
	}

	// Create a new data API directly in the devserver.  This allows us to inject
	// the data API into the dev server port, providing a single router for the dev
	// server UI, events, and API for loading data.
	//
	// Merge the dev server API (for handling files & registration) with the data
	// API into the event API router.

	mounts := []api.Mount{
		{At: "/", Router: devAPI},
		{At: "/v0", Router: core.Router},
		{At: "/debug", Handler: middleware.Profiler()},
		{At: "/dbg", Router: debugapi.Router},
		{At: "/metrics", Router: metricsAPI.Router},
	}

	if testapi.ShouldEnable() {
		mounts = append(mounts, api.Mount{At: "/test", Handler: testapi.New(testapi.Options{
			QueueShardSelector: shardSelector,
			Queue:              rq,
			Executor:           exec,
			StateManager:       smv2,
			ResetAll: func() {
				// Only flush in-memory clusters if they exist
				if shardedCluster != nil {
					shardedCluster.FlushAll()
				}
				if unshardedCluster != nil {
					unshardedCluster.FlushAll()
				}
				if connectCluster != nil {
					connectCluster.FlushAll()
				}
			},
		})})
	}

	ds.Apiservice = api.NewService(api.APIServiceOptions{
		Config:         ds.Opts.Config,
		Mounts:         mounts,
		LocalEventKeys: opts.EventKeys,
		Logger:         l,
	})

	return service.StartAll(ctx, ds, runner, executorSvc, ds.Apiservice, connGateway)
}

func createInmemoryRedis(ctx context.Context, tick time.Duration) (rueidis.Client, *miniredis.Miniredis, error) {
	r := miniredis.NewMiniRedis()
	_ = r.Start()
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	if err != nil {
		return nil, nil, err
	}

	// If tick is lower than the default, tick every 50ms.  This lets us save
	// CPU for standard dev-server testing.
	poll := DefaultTickDuration

	go func() {
		for range time.Tick(poll) {
			r.FastForward(poll)
		}
	}()
	return rc, r, nil
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
		trackedEvent := event.NewOSSTrackedEvent(evt, nil)
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
				trackedEvent := event.NewOSSTrackedEvent(evt, nil)
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

func NormalizeConcurrencyKeys(smv2 sv2.StateLoader, dbcqrs cqrs.Manager) redis_state.NormalizeRefreshItemCustomConcurrencyKeysFn {
	return func(ctx context.Context, item *queue.QueueItem, existingKeys []state.CustomConcurrency, shadowPartition *redis_state.QueueShadowPartition) ([]state.CustomConcurrency, error) {
		id := sv2.IDFromV1(item.Data.Identifier)

		workflow, err := dbcqrs.GetFunctionByInternalUUID(ctx, id.Tenant.EnvID, id.FunctionID)
		if err != nil {
			return nil, fmt.Errorf("could not find workflow: %w", err)
		}
		fn, err := workflow.InngestFunction()
		if err != nil {
			return nil, fmt.Errorf("could not convert workflow to inngest function: %w", err)
		}

		if fn.Concurrency == nil || len(fn.Concurrency.Limits) == 0 {
			return nil, nil
		}

		events, err := smv2.LoadEvents(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not load events: %w", err)
		}

		if len(events) == 0 {
			return nil, nil
		}

		var evt0 event.Event

		if err := json.Unmarshal(events[0], &evt0); err != nil {
			return nil, fmt.Errorf("could not unmarshal event: %w", err)
		}

		evtMap := evt0.Map()

		return queue.GetCustomConcurrencyKeys(ctx, id, fn.Concurrency.Limits, evtMap), nil
	}
}

func NormalizeThrottle(smv2 sv2.StateLoader, dbcqrs cqrs.Manager) redis_state.RefreshItemThrottleFn {
	return func(ctx context.Context, item *queue.QueueItem) (*queue.Throttle, error) {
		id := sv2.IDFromV1(item.Data.Identifier)

		workflow, err := dbcqrs.GetFunctionByInternalUUID(ctx, id.Tenant.EnvID, id.FunctionID)
		if err != nil {
			return nil, fmt.Errorf("could not find workflow: %w", err)
		}
		fn, err := workflow.InngestFunction()
		if err != nil {
			return nil, fmt.Errorf("could not convert workflow to inngest function: %w", err)
		}

		if fn.Throttle == nil {
			return nil, nil
		}

		events, err := smv2.LoadEvents(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not load events: %w", err)
		}

		if len(events) == 0 {
			return nil, nil
		}

		var evt0 event.Event

		if err := json.Unmarshal(events[0], &evt0); err != nil {
			return nil, fmt.Errorf("could not unmarshal event: %w", err)
		}

		evtMap := evt0.Map()

		return queue.GetThrottleConfig(ctx, id.FunctionID, fn.Throttle, evtMap), nil
	}
}

func PartitionConstraintConfigGetter(dbcqrs cqrs.Manager) redis_state.PartitionConstraintConfigGetter {
	return func(ctx context.Context, p redis_state.PartitionIdentifier) redis_state.PartitionConstraintConfig {
		if p.SystemQueueName != nil {
			return defaultPartitionConstraintConfig
		}

		workflow, err := dbcqrs.GetFunctionByInternalUUID(ctx, p.EnvID, p.FunctionID)
		if err != nil {
			return defaultPartitionConstraintConfig
		}
		fn, err := workflow.InngestFunction()
		if err != nil {
			return defaultPartitionConstraintConfig
		}

		// TODO Make this reusable in cloud, it's the same operation with different data sources
		accountLimit := consts.DefaultConcurrencyLimit

		fnLimit := fn.ConcurrencyLimit()
		if fnLimit <= 0 {
			fnLimit = accountLimit
		}

		constraints := redis_state.PartitionConstraintConfig{
			FunctionVersion: fn.FunctionVersion,

			Concurrency: redis_state.PartitionConcurrency{
				SystemConcurrency:     consts.DefaultConcurrencyLimit,
				AccountConcurrency:    accountLimit,
				FunctionConcurrency:   fnLimit,
				CustomConcurrencyKeys: nil,
			},
		}

		if fn.Concurrency != nil && len(fn.Concurrency.Limits) > 0 {
			for _, limit := range fn.Concurrency.Limits {
				if !limit.IsCustomLimit() {
					continue
				}

				constraints.Concurrency.CustomConcurrencyKeys = append(constraints.Concurrency.CustomConcurrencyKeys,
					redis_state.CustomConcurrencyLimit{
						Mode:                enums.ConcurrencyModeStep,
						Scope:               limit.Scope,
						HashedKeyExpression: limit.Hash,
						Limit:               limit.Limit,
					})
			}
		}

		if fn.Throttle != nil {
			var keyExpr string
			if fn.Throttle.Key != nil {
				keyExpr = *fn.Throttle.Key
			}

			constraints.Throttle = &redis_state.PartitionThrottle{
				ThrottleKeyExpressionHash: util.XXHash(keyExpr),
				Limit:                     int(fn.Throttle.Limit),
				Burst:                     int(fn.Throttle.Burst),
				Period:                    int(fn.Throttle.Period.Seconds()),
			}
		}

		return constraints
	}
}

func connectToOrCreateRedis(redisURI string) (rueidis.Client, error) {
	opt, err := connectToOrCreateRedisOption(redisURI)
	if err != nil {
		return nil, fmt.Errorf("could not create redis options: %w", err)
	}

	rc, err := rueidis.NewClient(opt)
	if err != nil {
		return nil, fmt.Errorf("error creating redis client: %w", err)
	}

	return rc, nil
}

func connectToOrCreateRedisOption(redisURI string) (rueidis.ClientOption, error) {
	if redisURI == "" {
		return createInmemoryRedisConnectionOpt()
	}

	opt, err := rueidis.ParseURL(redisURI)
	if err != nil {
		return rueidis.ClientOption{}, fmt.Errorf("error parsing redis uri: %w", err)
	}

	// Set default overrides
	opt.DisableCache = true
	opt.BlockingPoolSize = consts.RedisBlockingPoolSize

	return opt, nil
}

// createInmemoryRedisConnectionOpt creates the options for a new connection to the in-memory Redis
// server. If the server is not yet running, it will start one.
func createInmemoryRedisConnectionOpt() (rueidis.ClientOption, error) {
	// For devserver, we don't use a singleton like lite.go does
	r := miniredis.NewMiniRedis()
	err := r.Start()
	if err != nil {
		return rueidis.ClientOption{}, fmt.Errorf("error starting in-memory redis: %w", err)
	}

	poll := time.Second
	go func() {
		for range time.Tick(poll) {
			r.FastForward(poll)
		}
	}()

	return rueidis.ClientOption{
		InitAddress:       []string{r.Addr()},
		DisableCache:      true,
		BlockingPoolSize:  consts.RedisBlockingPoolSize,
		ForceSingleClient: true,
	}, nil
}
