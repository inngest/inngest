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
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/authn"
	"github.com/inngest/inngest/pkg/backoff"
	"github.com/inngest/inngest/pkg/config"
	connectConfig "github.com/inngest/inngest/pkg/config/connect"
	_ "github.com/inngest/inngest/pkg/config/defaults"
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/connect"
	"github.com/inngest/inngest/pkg/connect/auth"
	connectgrpc "github.com/inngest/inngest/pkg/connect/grpc"
	"github.com/inngest/inngest/pkg/connect/lifecycles"
	connectv0 "github.com/inngest/inngest/pkg/connect/rest/v0"
	connstate "github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
	sqlc_postgres "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres"
	"github.com/inngest/inngest/pkg/debugapi"
	"github.com/inngest/inngest/pkg/deploy"
	"github.com/inngest/inngest/pkg/devserver/devutil"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/cron"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/driver/httpv2"
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
	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/pkg/util/awsgateway"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/sync/errgroup"
)

const (
	DefaultTick         = 150
	DefaultTickDuration = time.Millisecond * DefaultTick
	DefaultPollInterval = 5
	DefaultQueueWorkers = 100

	DefaultConnectGatewayPort      = 8289
	DefaultConnectGatewayGRPCPort  = 50052
	DefaultConnectExecutorGRPCPort = 50053

	DefaultDebugAPIPort = 7777
)

var defaultPartitionConstraintConfig = queue.PartitionConstraintConfig{
	Concurrency: queue.PartitionConcurrency{
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

	ConnectGatewayPort int                             `json:"connectGatewayPort"`
	ConnectGatewayHost string                          `json:"connectGatewayHost"`
	ConnectGRPCConfig  connectConfig.ConnectGRPCConfig `json:"connectGRPCConfig"`

	NoUI bool

	// Persist controls whether to persist data in between restarts.  If false,
	// the dev server will use in-memory databases.
	Persist bool `json:"persist"`

	// RedisURI allows connecting to external Redis instead of in-memory Redis
	RedisURI string `json:"redis_uri"`

	// PostgresURI allows connecting to external Postgres instead of SQLite
	PostgresURI             string `json:"postgres_uri"`
	PostgresMaxIdleConns    int    `json:"postgres-max-idle-conns"`
	PostgresMaxOpenConns    int    `json:"postgres-max-open-conns"`
	PostgresConnMaxIdleTime int    `json:"postgres-conn-max-idle-time"`
	PostgresConnMaxLifetime int    `json:"postgres-conn-max-lifetime"`

	// SQLiteDir specifies where SQLite files should be stored
	SQLiteDir string `json:"sqlite_dir"`

	// Debug API
	DebugAPIPort int `json:"debugAPIPort"`
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

	services := []service.Service{}

	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{
		Persist:     opts.Persist,
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
	dbcqrs := base_cqrs.NewCQRS(db, dbDriver, sqlc_postgres.NewNormalizedOpts{
		MaxIdleConns:    opts.PostgresMaxIdleConns,
		MaxOpenConns:    opts.PostgresMaxOpenConns,
		ConnMaxIdle:     opts.PostgresConnMaxIdleTime,
		ConnMaxLifetime: opts.PostgresConnMaxLifetime,
	})
	hd := base_cqrs.NewHistoryDriver(db, dbDriver, sqlc_postgres.NewNormalizedOpts{
		MaxIdleConns:    opts.PostgresMaxIdleConns,
		MaxOpenConns:    opts.PostgresMaxOpenConns,
		ConnMaxIdle:     opts.PostgresConnMaxIdleTime,
		ConnMaxLifetime: opts.PostgresConnMaxLifetime,
	})
	loader := dbcqrs.(state.FunctionLoader)

	stepLimitOverrides := make(map[string]int)
	stateSizeLimitOverrides := make(map[string]int)
	pauseOverrides := make(map[uuid.UUID]bool)

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

	pauseMgr := pauses.NewPauseStoreManager(unshardedClient)

	var sm state.Manager
	sm, err = redis_state.New(ctx, redis_state.WithShardedClient(shardedClient), redis_state.WithPauseDeleter(pauseMgr))
	if err != nil {
		return err
	}
	smv2 := redis_state.MustRunServiceV2(sm)

	// Create a new broadcaster which lets us broadcast realtime messages.
	broadcaster := realtime.NewInProcessBroadcaster()

	runMode := queue.QueueRunMode{
		Sequential:    true,
		Scavenger:     true,
		Partition:     true,
		Continuations: true,
	}
	enableKeyQueues := os.Getenv("EXPERIMENTAL_KEY_QUEUES_ENABLE") == "true"
	enableStepMetadata := os.Getenv("EXPERIMENTAL_STEP_METADATA") == "true"

	if enableKeyQueues {
		runMode.ShadowPartition = true
		runMode.AccountShadowPartition = true
		runMode.AccountShadowPartitionWeight = 80
		runMode.ShadowContinuations = true
		runMode.ShadowContinuationSkipProbability = consts.QueueContinuationSkipProbability
		runMode.NormalizePartition = true
	}

	conditionalQueueTracer := itrace.NewConditionalTracer(itrace.QueueTracer(), itrace.AlwaysTrace)

	queueOpts := []queue.QueueOpt{
		queue.WithRunMode(runMode),
		queue.WithIdempotencyTTL(time.Hour),
		queue.WithNumWorkers(int32(opts.QueueWorkers)),
		queue.WithPollTick(opts.Tick),
		queue.WithShadowPollTick(2 * opts.Tick),
		queue.WithBacklogNormalizePollTick(5 * opts.Tick),

		queue.WithLogger(l),

		// Key queues
		queue.WithNormalizeRefreshItemCustomConcurrencyKeys(NormalizeConcurrencyKeys(smv2, dbcqrs)),
		queue.WithRefreshItemThrottle(NormalizeThrottle(smv2, dbcqrs)),
		queue.WithPartitionConstraintConfigGetter(PartitionConstraintConfigGetter(dbcqrs)),

		queue.WithAllowKeyQueues(func(ctx context.Context, acctID, functionID uuid.UUID) bool {
			return enableKeyQueues
		}),
		queue.WithBacklogRefillLimit(10),
		queue.WithPartitionPausedGetter(func(ctx context.Context, functionID uuid.UUID) queue.PartitionPausedInfo {
			if paused, ok := pauseOverrides[functionID]; ok && paused {
				return queue.PartitionPausedInfo{
					Stale:  false,
					Paused: true,
				}
			}
			return queue.PartitionPausedInfo{}
		}),
		queue.WithConditionalTracer(conditionalQueueTracer),
	}

	const rateLimitPrefix = "ratelimit"
	var capacityManager constraintapi.RolloutManager
	enableConstraintAPI := os.Getenv("ENABLE_CONSTRAINT_API") == "true"
	if enableConstraintAPI {
		shards := map[string]rueidis.Client{
			consts.DefaultQueueShardName: unshardedClient.Queue().Client(),
		}

		cm, err := constraintapi.NewRedisCapacityManager(
			constraintapi.WithClock(clockwork.NewRealClock()),
			constraintapi.WithNumScavengerShards(1),
			constraintapi.WithQueueShards(shards),
			constraintapi.WithRateLimitClient(unshardedRc),
			constraintapi.WithQueueStateKeyPrefix(redis_state.QueueDefaultKey),
			constraintapi.WithRateLimitKeyPrefix(rateLimitPrefix),
			constraintapi.WithEnableHighCardinalityInstrumentation(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return false
			}),
		)
		if err != nil {
			return fmt.Errorf("could not create contraint API: %w", err)
		}

		queueOpts = append(queueOpts, queue.WithCapacityManager(cm))
		queueOpts = append(queueOpts, queue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
			return true, true
		}))

		services = append(services, constraintapi.NewLeaseScavengerService(cm, consts.ConstraintAPIScavengerTick))

		capacityManager = cm

		l.Warn("EXPERIMENTAL: Enabling Constraint API")
	}

	if opts.RetryInterval > 0 {
		queueOpts = append(queueOpts, queue.WithBackoffFunc(
			backoff.GetLinearBackoffFunc(time.Duration(opts.RetryInterval)*time.Second),
		))
	}

	queueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), queueOpts...)

	shardSelector := func(ctx context.Context, _ uuid.UUID, _ *string) (queue.QueueShard, error) {
		return queueShard, nil
	}

	queueShards := map[string]queue.QueueShard{
		consts.DefaultQueueShardName: queueShard,
	}

	rq, err := queue.New(
		ctx,
		"queue",
		queueShard,
		queueShards,
		shardSelector,
		queueOpts...,
	)
	if err != nil {
		return fmt.Errorf("could not create queue: %w", err)
	}

	rl := ratelimit.New(ctx, unshardedRc, fmt.Sprintf("{%s}:", rateLimitPrefix))

	batcher := batch.NewRedisBatchManager(shardedClient.Batch(), rq, batch.WithLogger(l))
	debouncer := debounce.NewRedisDebouncer(unshardedClient.Debounce(), queueShard, rq)
	croner := cron.NewRedisCronManager(queueShard, rq, l)

	sn := singleton.New(ctx, map[string]*redis_state.QueueClient{
		consts.DefaultQueueShardName: unshardedClient.Queue(),
	}, shardSelector)

	conditionalConnectTracer := itrace.NewConditionalTracer(itrace.ConnectTracer(), itrace.AlwaysTrace)

	connectPubSubLogger := logger.StdlibLoggerWithCustomVarName(ctx, "CONNECT_PUBSUB_LOG_LEVEL")

	connectionManager := connstate.NewRedisConnectionStateManager(connectRc)

	// Create a new expression aggregator, using the Redis only pause manager to load evaluables.
	agg := expragg.NewAggregator(ctx, 100, 100, pauseMgr, expressions.ExprEvaluator, nil, nil)

	executorLogger := connectPubSubLogger.With("svc", "executor")

	executorProxy := connectgrpc.NewConnector(ctx, connectgrpc.GRPCConnectorOpts{
		Tracer:             conditionalConnectTracer,
		StateManager:       connectionManager,
		EnforceLeaseExpiry: enforceConnectLeaseExpiry,
		GRPCConfig:         opts.ConnectGRPCConfig,
	}, connectgrpc.WithConnectorLogger(executorLogger))

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

	drivers := []driver.DriverV1{}
	for _, driverConfig := range opts.Config.Execution.Drivers {
		d, err := driverConfig.NewDriver(registration.NewDriverOpts{
			ConnectForwarder:       executorProxy,
			ConditionalTracer:      conditionalConnectTracer,
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

	tp := tracing.NewSqlcTracerProvider(base_cqrs.NewQueries(db, dbDriver, sqlc_postgres.NewNormalizedOpts{
		MaxIdleConns:    opts.PostgresMaxIdleConns,
		MaxOpenConns:    opts.PostgresMaxOpenConns,
		ConnMaxIdle:     opts.PostgresConnMaxIdleTime,
		ConnMaxLifetime: opts.PostgresConnMaxLifetime,
	}))

	url := opts.Config.CoreAPI.Addr
	if url == "0.0.0.0" {
		url = "127.0.0.1"
	}

	executorOpts := []executor.ExecutorOpt{
		executor.WithHTTPClient(httpClient),
		executor.WithStateManager(smv2),
		executor.WithPauseManager(pauseMgr),
		executor.WithDriverV1(drivers...),
		executor.WithDriverV2(httpv2.NewDriver(httpClient)),
		executor.WithExpressionAggregator(agg),
		executor.WithQueue(rq),
		executor.WithRateLimiter(rl),
		executor.WithLogger(l),
		executor.WithFunctionLoader(loader),
		executor.WithRealtimePublisher(broadcaster),
		executor.WithSigningKeyLoader(func(ctx context.Context, envID uuid.UUID) ([]byte, error) {
			// for httpv2, ensuring we sign sync and async requests using the new driver.
			if opts.SigningKey == nil {
				return nil, nil
			}
			return []byte(*opts.SigningKey), nil
		}),
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
		executor.WithInvokeEventHandler(getInvokeEventHandler(ctx, pb, opts.Config.EventStream.Service.Concrete.TopicName())),
		executor.WithDebouncer(debouncer),
		executor.WithSingletonManager(sn),
		executor.WithBatcher(batcher),
		executor.WithAssignedQueueShard(queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTraceReader(dbcqrs),
		executor.WithRealtimeConfig(executor.ExecutorRealtimeConfig{
			Secret:     consts.DevServerRealtimeJWTSecret,
			PublishURL: fmt.Sprintf("http://%s:%d/v1/realtime/publish", url, opts.Config.CoreAPI.Port),
		}),
		executor.WithTracerProvider(tp),

		executor.WithAllowStepMetadata(func(ctx context.Context, acctID uuid.UUID) bool {
			return enableStepMetadata
		}),
	}

	if capacityManager != nil {
		executorOpts = append(executorOpts, executor.WithCapacityManager(capacityManager))
		executorOpts = append(executorOpts, executor.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
			return true, true
		}))
	}

	exec, err := executor.NewExecutor(executorOpts...)
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
		executor.WithServiceCroner(croner),
		executor.WithServiceLogger(l),
		executor.WithServiceShardSelector(shardSelector),
		executor.WithServicePublisher(pb),
		executor.WithServiceEnableKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return enableKeyQueues
		}),
	)

	runner := runner.NewService(
		opts.Config,
		runner.WithCQRS(dbcqrs),
		runner.WithExecutor(exec),
		runner.WithExecutionManager(dbcqrs),
		runner.WithPauseManager(pauseMgr),
		runner.WithStateManager(sm),
		runner.WithRunnerQueue(rq),
		runner.WithBatchManager(batcher),
		runner.WithCronManager(croner),
		runner.WithPublisher(pb),
		runner.WithLogger(l),
	)

	// The devserver embeds the event API.
	ds := NewService(opts, runner, dbcqrs, pb, stepLimitOverrides, stateSizeLimitOverrides, unshardedRc, hmw, nil)
	ds.State = sm
	ds.Queue = rq
	ds.Executor = exec
	ds.CronSyncer = croner
	// start the API
	// Create a new API endpoint which hosts SDK-related functionality for
	// registering functions.
	devAPI := NewDevAPI(ds, DevAPIOptions{AuthMiddleware: authn.SigningKeyMiddleware(opts.SigningKey), disableUI: opts.NoUI})

	// Add MCP server route
	AddMCPRoute(devAPI, ds.HandleEvent, ds.Data, opts.Tick)
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
			ConnectRequestStateManager: connectionManager,
			Signer:                     auth.NewJWTSessionTokenSigner(consts.DevServerConnectJwtSecret),
			RequestAuther:              ds,
			ConnectGatewayRetriever:    ds,
			Dev:                        true,
			EntitlementProvider:        ds,
			ConditionalTracer:          conditionalConnectTracer,
			ConnectGRPCConfig:          opts.ConnectGRPCConfig,
		},
	})
	if err != nil {
		return err
	}

	devAPI.Route("/v1", func(r chi.Router) {
		// Add the V1 API to our dev server API.
		cache := cache.New[[]byte](freecachestore.NewFreecache(freecache.NewCache(1024 * 1024)))
		caching := apiv1.NewCacheMiddleware(cache)

		apiv1.AddRoutes(r, apiv1.Opts{
			AuthMiddleware:     authn.SigningKeyMiddleware(opts.SigningKey),
			CachingMiddleware:  caching,
			FunctionReader:     ds.Data,
			JobQueueReader:     ds.Queue.(queue.JobQueueReader),
			Executor:           ds.Executor,
			Queue:              rq,
			QueueShardSelector: shardSelector,
			Broadcaster:        broadcaster,
			TraceReader:        ds.Data,

			AppCreator:        dbcqrs,
			FunctionCreator:   dbcqrs,
			EventPublisher:    runner,
			TracerProvider:    tp,
			State:             smv2,
			RealtimeJWTSecret: consts.DevServerRealtimeJWTSecret,

			CheckpointOpts: apiv1.CheckpointAPIOpts{
				RunOutputReader: devutil.NewLocalOutputReader(core.Resolver(), ds.Data, ds.Data),
				RunJWTSecret:    consts.DevServerRunJWTSecret,
			},

			MetadataOpts: apiv1.MetadataOpts{
				Flag: func(ctx context.Context, accountID uuid.UUID) bool {
					return enableStepMetadata
				},
				SpanExtractor: extractors.DefaultSpanExtractors,
			},
		})
	})

	connGateway := connect.NewConnectGatewayService(
		connect.WithConnectionStateManager(connectionManager),
		connect.WithGatewayAuthHandler(auth.NewJWTAuthHandler(consts.DevServerConnectJwtSecret)),
		connect.WithDev(),
		connect.WithGatewayPublicPort(opts.ConnectGatewayPort),
		connect.WithGRPCConfig(opts.ConnectGRPCConfig),
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
		QueueManager:   queueShard,
	})
	if err != nil {
		return err
	}

	// Create the API v2 service handler
	serviceOpts := apiv2.ServiceOptions{
		SigningKeysProvider: apiv2.NewSigningKeysProvider(opts.SigningKey),
		EventKeysProvider:   apiv2.NewEventKeysProvider(opts.EventKeys),
	}

	apiv2Base := apiv2base.NewBase()
	apiv2Handler, err := apiv2.NewHTTPHandler(ctx, serviceOpts, apiv2.HTTPHandlerOptions{
		AuthnMiddleware: authn.SigningKeyMiddleware(opts.SigningKey),
	}, apiv2Base)
	if err != nil {
		return fmt.Errorf("failed to create v2 handler: %w", err)
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
		{At: "/api/v2", Handler: apiv2Handler},
		{At: "/debug", Handler: middleware.Profiler()},
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
			PauseFunction: func(id uuid.UUID) {
				pauseOverrides[id] = true
			},
			UnpauseFunction: func(id uuid.UUID) {
				delete(pauseOverrides, id)
			},
		})})
	}

	ds.Apiservice = api.NewService(api.APIServiceOptions{
		Config:         ds.Opts.Config,
		Mounts:         mounts,
		LocalEventKeys: opts.EventKeys,
		Logger:         l,
	})

	services = append(services, ds, runner, executorSvc, ds.Apiservice, connGateway)

	if os.Getenv("DEBUG") != "" {
		services = append(services, debugapi.NewDebugAPI(debugapi.Opts{
			Log:             l,
			DB:              ds.Data,
			Queue:           rq,
			State:           ds.State,
			Cron:            croner,
			ShardSelector:   shardSelector,
			Port:            ds.Opts.DebugAPIPort,
			PauseManager:    pauseMgr,
			CapacityManager: capacityManager,
		}))
	}

	return service.StartAll(ctx, services...)
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

func getInvokeEventHandler(ctx context.Context, pb pubsub.Publisher, topic string) execution.HandleInvokeEvent {
	return func(ctx context.Context, evt event.TrackedEvent) error {
		byt, err := json.Marshal(evt)
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
				trackedEvent := event.NewBaseTrackedEvent(evt, nil)
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

func NormalizeConcurrencyKeys(smv2 sv2.StateLoader, dbcqrs cqrs.Manager) queue.NormalizeRefreshItemCustomConcurrencyKeysFn {
	return func(ctx context.Context, item *queue.QueueItem, existingKeys []state.CustomConcurrency, shadowPartition *queue.QueueShadowPartition) ([]state.CustomConcurrency, error) {
		id := sv2.IDFromV1(item.Data.Identifier)

		workflow, err := dbcqrs.GetFunctionByInternalUUID(ctx, id.FunctionID)
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

func NormalizeThrottle(smv2 sv2.StateLoader, dbcqrs cqrs.Manager) queue.RefreshItemThrottleFn {
	return func(ctx context.Context, item *queue.QueueItem) (*queue.Throttle, error) {
		id := sv2.IDFromV1(item.Data.Identifier)

		workflow, err := dbcqrs.GetFunctionByInternalUUID(ctx, id.FunctionID)
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

func PartitionConstraintConfigGetter(dbcqrs cqrs.Manager) queue.PartitionConstraintConfigGetter {
	return func(ctx context.Context, p queue.PartitionIdentifier) queue.PartitionConstraintConfig {
		if p.SystemQueueName != nil {
			return defaultPartitionConstraintConfig
		}

		workflow, err := dbcqrs.GetFunctionByInternalUUID(ctx, p.FunctionID)
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

		fnVersion := fn.FunctionVersion
		if fnVersion <= 0 {
			fnVersion = 1
		}

		constraints := queue.PartitionConstraintConfig{
			FunctionVersion: fnVersion,

			Concurrency: queue.PartitionConcurrency{
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
					queue.CustomConcurrencyLimit{
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

			constraints.Throttle = &queue.PartitionThrottle{
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
		return rueidis.ClientOption{}, fmt.Errorf("error parsing redis uri: invalid format")
	}

	// Fix for Redis Sentinel authentication: rueidis.ParseURL correctly identifies
	// Sentinel configurations but fails to populate Sentinel credentials.
	// When a master_set is configured but Sentinel credentials are empty,
	// copy the main credentials to Sentinel authentication.
	if opt.Sentinel.MasterSet != "" && opt.Sentinel.Username == "" && opt.Username != "" {
		opt.Sentinel.Username = opt.Username
		opt.Sentinel.Password = opt.Password
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
