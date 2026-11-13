package devserver

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/config/registration"
	connectgrpc "github.com/inngest/inngest/pkg/connect/grpc"
	connstate "github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/deploy"
	"github.com/inngest/inngest/pkg/duckdb/query"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/cancellation"
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
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/expressions/expragg"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/metrics"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/inngest/inngest/pkg/run"
	"github.com/inngest/inngest/pkg/service/servicefx"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/util/awsgateway"
	"go.uber.org/fx"
)

// dualWriteStopTimeout bounds the flush of the DuckDB dual-write at stop.
// without it, a flush to an unresponsive subprocess blocks the stop of the
// server.
const dualWriteStopTimeout = 30 * time.Second

type connectResult struct {
	fx.Out

	State connstate.StateManager
	// Tracer records connect spans.
	Tracer itrace.ConditionalTracer
	// Proxy sends executor requests to connect workers.
	Proxy connectgrpc.Connector
}

func newConnect(ctx context.Context, opts StartOpts, clients *redisClients) connectResult {
	tracer := itrace.NewConditionalTracer(itrace.ConnectTracer(), itrace.AlwaysTrace)
	stateManager := connstate.NewRedisConnectionStateManager(clients.connect)
	log := logger.StdlibLoggerWithCustomVarName(ctx, "CONNECT_PUBSUB_LOG_LEVEL").With("svc", "executor")

	return connectResult{
		State:  stateManager,
		Tracer: tracer,
		Proxy: connectgrpc.NewConnector(ctx, connectgrpc.GRPCConnectorOpts{
			Tracer:             tracer,
			StateManager:       stateManager,
			EnforceLeaseExpiry: enforceConnectLeaseExpiry,
			GRPCConfig:         opts.ConnectGRPCConfig,
		}, connectgrpc.WithConnectorLogger(log)),
	}
}

// realtimePublishURL is the URL that requests made by the server publish
// realtime messages to.
func realtimePublishURL(opts StartOpts) string {
	host := opts.Config.CoreAPI.Addr
	if host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d/v1/realtime/publish", host, opts.Config.CoreAPI.Port)
}

// newHTTPClient creates the client that calls SDKs.  it allows local
// addresses, and turns requests to AWS API Gateway URLs into Lambda
// invocations.  the transport of deploy.Client gets the same Lambda
// transform, and deploy.Client is process-wide.
func newHTTPClient(opts StartOpts) exechttp.ExtendedClient {
	httpClient := exechttp.Client(
		exechttp.SecureDialerOpts{
			AllowHostDocker: true, // In local dev, this is OK
			AllowPrivate:    true, // In local dev, this is OK
			AllowNAT64:      true, // In local dev, this is OK
		},
		// Enable publishing of any requests made directly from the dev server.  Note that this
		// is different from the cloud.
		exechttp.WithDevServerRealtimePublishing(realtimePublishURL(opts)),
	)

	httpClient.Client.Transport = awsgateway.NewTransformTripper(httpClient.Client.Transport)
	deploy.Client.Transport = awsgateway.NewTransformTripper(deploy.Client.Transport)
	return httpClient
}

func newPublisher(ctx context.Context, opts StartOpts) (pubsub.Publisher, error) {
	pb, err := pubsub.NewPublisher(ctx, opts.Config.EventStream.Service)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}
	return pb, nil
}

// dualWrite is the opt-in DuckDB dual-write.  it runs only with
// StartOpts.EnableDuckDB, and only when the duckdb subprocess starts.  on any
// failure, such as a missing binary or a failed migration, the server starts
// without it.
type dualWrite struct {
	// listeners has the dual-write listener when dual-write runs.
	listeners []execution.SyncLifecycleListener

	// readDB reads from the duckdb subprocess, on a pool apart from the pool
	// that dual-write writes through, so reads cannot starve flushes.  it is
	// nil unless dual-write runs.  insights runs on it, also without
	// StartOpts.EnableDuckDBReads.
	readDB *sql.DB

	// data serves GQL, and the trace and run reads of the REST APIs.  it
	// also reads runs, events, and run traces from DuckDB when dual-write
	// runs with StartOpts.EnableDuckDBReads.  writes, and the reads of apps
	// and functions, use the primary cqrs.Manager, because DuckDB has no
	// apps or functions tables.
	data cqrs.Manager
}

// newDualWrite starts the DuckDB dual-write.  when the app stops, it flushes
// dual-write and stops the subprocess.  this hook runs after every service
// stops, so no write comes after the flush.
func newDualWrite(
	ctx context.Context,
	opts StartOpts,
	exp experiments,
	primary cqrs.Manager,
	log logger.Logger,
	lifecycle fx.Lifecycle,
) *dualWrite {
	dw := &dualWrite{
		data: primary,
	}

	// --persist alone does not write an on-disk DuckLake catalog.  that also
	// needs EXPERIMENTAL_DUCKDB_PERSISTENT_DUALWRITE, so --duckdb without it
	// dual-writes to an in-memory catalog.  a persisted catalog uses the
	// database of StartOpts.PostgresURI as its catalog store when it is set
	// (see duckLakeOptionsFor).
	listener, _, readDB := setupDualWrite(ctx, opts.EnableDuckDB, opts.Persist && exp.duckDBPersistence, "", opts.SQLiteDir, opts.PostgresURI)
	if listener == nil {
		return dw
	}

	log.Info("dual-write enabled: syncing executions to DuckDB subprocess")
	dw.listeners = []execution.SyncLifecycleListener{listener}
	dw.readDB = readDB
	if opts.EnableDuckDBReads {
		log.Info("dual-write reads enabled: serving GQL/REST trace and run data from DuckDB")
		dw.data = query.Wrap(primary, readDB)
	}

	lifecycle.Append(fx.StopHook(dw.close))
	return dw
}

// close flushes dual-write and stops the duckdb subprocess.  the subprocess
// does not stop when the context of New is cancelled, so it keeps running
// until close runs.
func (dw *dualWrite) close(ctx context.Context) {
	if dw.readDB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, dualWriteStopTimeout)
	defer cancel()
	// Before stopDualWrite, which closes the write handle and with it
	// the subprocess readDB's connections run against.
	_ = dw.readDB.Close()
	for _, l := range dw.listeners {
		stopDualWrite(ctx, l)
	}
}

// limitOverrides are the step and state size limits of single functions.  the
// dev server API sets them, and the executor reads them for every run.
type limitOverrides struct {
	steps     map[string]int
	stateSize map[string]int
}

func newLimitOverrides() *limitOverrides {
	return &limitOverrides{
		steps:     map[string]int{},
		stateSize: map[string]int{},
	}
}

type executorParams struct {
	fx.In

	Context        context.Context
	Opts           StartOpts
	Log            logger.Logger
	CQRS           cqrs.Manager
	History        history.Driver
	Clients        *redisClients
	RunService     sv2.RunService
	Pauses         pauses.Manager
	Broadcaster    realtime.Broadcaster
	Queue          queueProcessor
	Registry       queue.ShardRegistryController
	RateLimiter    ratelimit.RateLimiter
	Batcher        batch.BatchManager
	Debouncer      debounce.Debouncer
	Singleton      singleton.Singleton
	HTTPClient     exechttp.ExtendedClient
	ConnectProxy   connectgrpc.Connector
	ConnectTracer  itrace.ConditionalTracer
	Publisher      pubsub.Publisher
	TracerProvider tracing.TracerProvider
	DualWrite      *dualWrite
	Capacity       capacityManager
	Semaphores     constraintapi.SemaphoreManager
	Limits         *limitOverrides
}

func newExecutor(p executorParams) (execution.Executor, error) {
	drivers := []driver.DriverV1{}
	for _, driverConfig := range p.Opts.Config.Execution.Drivers {
		d, err := driverConfig.NewDriver(registration.NewDriverOpts{
			ConnectForwarder:       p.ConnectProxy,
			ConditionalTracer:      p.ConnectTracer,
			HTTPClient:             p.HTTPClient,
			LocalSigningKey:        p.Opts.SigningKey,
			RequireLocalSigningKey: p.Opts.RequireKeys,
		})
		if err != nil {
			return nil, err
		}
		drivers = append(drivers, d)
	}

	// Create a new expression aggregator, using the Redis only pause manager to load evaluables.
	agg := expragg.NewAggregator(p.Context, 100, 100, p.Pauses, expressions.ExprEvaluator, nil, nil)
	topic := p.Opts.Config.EventStream.Service.Concrete.TopicName()

	return executor.NewExecutor(
		executor.WithHTTPClient(p.HTTPClient),
		executor.WithStateManager(p.RunService),
		executor.WithPauseManager(p.Pauses),
		executor.WithDriverV1(drivers...),
		executor.WithDriverV2(httpv2.NewDriver(p.HTTPClient)),
		executor.WithExpressionAggregator(agg),
		executor.WithQueue(p.Queue),
		// records never expire; each step reads all of a function's cancellations
		executor.WithCancellationChecker(cancellation.NewChecker(cancellation.NewRedisReader(p.Clients.unsharded, ""))),
		executor.WithRateLimiter(p.RateLimiter),
		executor.WithLogger(p.Log),
		executor.WithFunctionLoader(p.CQRS.(state.FunctionLoader)),
		executor.WithRealtimePublisher(p.Broadcaster),
		executor.WithSigningKeyLoader(func(ctx context.Context, envID uuid.UUID) ([]byte, error) {
			// for httpv2, ensuring we sign sync and async requests using the new driver.
			if p.Opts.SigningKey == nil {
				return nil, nil
			}
			return []byte(*p.Opts.SigningKey), nil
		}),
		executor.WithLifecycleListeners(
			append([]execution.LifecycleListener{
				history.NewLifecycleListener(
					nil,
					p.History,
				),
				Lifecycle{
					Cqrs:       p.CQRS,
					Pb:         p.Publisher,
					EventTopic: topic,
				},
				run.NewTraceLifecycleListener(nil),
			}, metrics.NewLifecycleListeners()...)...,
		),
		executor.WithEventLifecycleListeners(execution.NoopEventLifecycleListener{}),
		executor.WithSyncLifecycleListeners(p.DualWrite.listeners...),
		executor.WithStepLimits(func(id sv2.ID) int {
			if override, hasOverride := p.Limits.steps[id.FunctionID.String()]; hasOverride {
				p.Log.Warn("using step limit override", "override", override, "fn_id", id.FunctionID)
				return override
			}

			return consts.DefaultMaxStepLimit
		}),
		executor.WithStateSizeLimits(func(id sv2.ID) int {
			if override, hasOverride := p.Limits.stateSize[id.FunctionID.String()]; hasOverride {
				p.Log.Warn("using state size limit override", "override", override, "fn_id", id.FunctionID)
				return override
			}

			return consts.DefaultMaxStateSizeLimit
		}),
		executor.WithInvokeFailHandler(getInvokeFailHandler(p.Context, p.Publisher, topic)),
		executor.WithInvokeEventHandler(getInvokeEventHandler(p.Context, p.Publisher, topic)),
		executor.WithDebouncer(p.Debouncer),
		executor.WithSingletonManager(p.Singleton),
		executor.WithBatcher(p.Batcher),
		executor.WithShardRegistry(p.Registry),
		executor.WithTraceReader(p.CQRS),
		executor.WithRealtimeConfig(executor.ExecutorRealtimeConfig{
			Secret:     consts.DevServerRealtimeJWTSecret,
			PublishURL: realtimePublishURL(p.Opts),
		}),
		executor.WithTracerProvider(p.TracerProvider),

		executor.WithCapacityManager(p.Capacity),
		executor.WithSemaphoreManager(p.Semaphores),
		executor.WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool) {
			return true
		}),
		executor.WithEnableBatchingInstrumentation(func(ctx context.Context, accountID, envID uuid.UUID) (enable bool) {
			return false
		}),
	)
}

type executorServiceParams struct {
	fx.In

	Opts      StartOpts
	Log       logger.Logger
	CQRS      cqrs.Manager
	State     state.Manager
	Queue     queueProcessor
	Registry  queue.ShardRegistryController
	Executor  execution.Executor
	Batcher   batch.BatchManager
	Debouncer debounce.Debouncer
	Croner    cron.CronManager
	Publisher pubsub.Publisher
}

// newExecutorTask runs the jobs on the queue, such as steps and crons.
func newExecutorTask(p executorServiceParams) (servicefx.Task, error) {
	svc, err := executor.NewService(
		p.Opts.Config,
		executor.WithExecutionManager(p.CQRS),
		executor.WithState(p.State),
		executor.WithServiceQueueProcessor(p.Queue),
		executor.WithServiceExecutor(p.Executor),
		executor.WithServiceBatcher(p.Batcher),
		executor.WithServiceDebouncer(p.Debouncer),
		executor.WithServiceCroner(p.Croner),
		executor.WithServiceLogger(p.Log),
		executor.WithServiceShardRegistry(p.Registry),
		executor.WithServicePublisher(p.Publisher),
	)
	if err != nil {
		return servicefx.Task{}, err
	}
	return servicefx.Task{
		Name: "executor",
		Run:  svc.Run,
		Stop: svc.Stop,
	}, nil
}

type runnerParams struct {
	fx.In

	Context   context.Context
	Opts      StartOpts
	Log       logger.Logger
	CQRS      cqrs.Manager
	Executor  execution.Executor
	Pauses    pauses.Manager
	State     state.Manager
	Batcher   batch.BatchManager
	Croner    cron.CronManager
	Publisher pubsub.Publisher
	DualWrite *dualWrite
}

// newRunner creates the runner, which starts the function runs that events
// trigger.
func newRunner(p runnerParams) (runner.Runner, error) {
	return runner.NewService(
		p.Context,
		p.Opts.Config,
		runner.WithCQRS(p.CQRS),
		runner.WithExecutor(p.Executor),
		runner.WithExecutionManager(p.CQRS),
		runner.WithPauseManager(p.Pauses),
		runner.WithStateManager(p.State),
		runner.WithBatchManager(p.Batcher),
		runner.WithCronManager(p.Croner),
		runner.WithPublisher(p.Publisher),
		runner.WithLogger(p.Log),
		runner.WithSyncLifecycleListeners(p.DualWrite.listeners...),
	)
}

// newRunnerTask handles the events on the event stream.
func newRunnerTask(r runner.Runner) servicefx.Task {
	return servicefx.Task{
		Name: "runner",
		Run:  r.Run,
		Stop: r.Stop,
	}
}
