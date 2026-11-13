package devserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
	"github.com/inngest/inngest/pkg/connect"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/connect/lifecycles"
	connectv0 "github.com/inngest/inngest/pkg/connect/rest/v0"
	connstate "github.com/inngest/inngest/pkg/connect/state"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi"
	"github.com/inngest/inngest/pkg/cqrs"
	cqrsmanager "github.com/inngest/inngest/pkg/cqrs/manager"
	"github.com/inngest/inngest/pkg/debugapi"
	"github.com/inngest/inngest/pkg/devserver/devutil"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/cancellation"
	"github.com/inngest/inngest/pkg/execution/cron"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/realtime"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/metrics"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/inngest/inngest/pkg/service"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/testapi"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
	"go.uber.org/fx"
)

type devserverParams struct {
	fx.In

	Opts       StartOpts
	Runner     runner.Runner
	CQRS       cqrs.Manager
	Publisher  pubsub.Publisher
	Limits     *limitOverrides
	Clients    *redisClients
	History    history.Driver
	DualWrite  *dualWrite
	State      state.Manager
	Executor   execution.Executor
	Semaphores constraintapi.SemaphoreManager
	Croner     cron.CronManager
}

// newDevserver creates the service that syncs apps and handles events.  the
// API service serves its HTTP routes.
func newDevserver(p devserverParams) *devserver {
	ds := NewService(p.Opts, p.Runner, p.CQRS, p.Publisher, p.Limits.steps, p.Limits.stateSize, p.Clients.unsharded, p.History, nil)
	// devapi.Info reports the "duckdb-insights" feature flag from it.  nil
	// means insights would return an error.
	ds.DuckDB = p.DualWrite.readDB
	ds.State = p.State
	ds.Executor = p.Executor
	ds.SemaphoreManager = p.Semaphores
	ds.CronSyncer = p.Croner
	return ds
}

func devserverService(ds *devserver) service.Service {
	return ds
}

type apiParams struct {
	fx.In

	Context        context.Context
	Opts           StartOpts
	Experiments    experiments
	Log            logger.Logger
	Devserver      *devserver
	Database       database
	CQRS           cqrs.Manager
	DualWrite      *dualWrite
	Clients        *redisClients
	RunService     sv2.RunService
	Broadcaster    realtime.Broadcaster
	Shard          redis_state.RedisQueueShard
	Registry       queue.ShardRegistryController
	Queue          queueProcessor
	Retry          retryPolicy
	Functions      *cqrsFunctionProvider
	Executor       execution.Executor
	Runner         runner.Runner
	TracerProvider tracing.TracerProvider
	ConnectState   connstate.StateManager
	ConnectTracer  itrace.ConditionalTracer
}

// newAPIService serves the dev server UI, the event API, and the REST,
// GraphQL, and MCP APIs on one port.  this keeps the CLI flags and the SDK
// config to one URL.
func newAPIService(p apiParams) (service.Service, error) {
	authMiddleware := authn.SigningKeyMiddleware(p.Opts.SigningKey)

	// Create a new API endpoint which hosts SDK-related functionality for
	// registering functions.
	devAPI := NewDevAPI(p.Devserver, DevAPIOptions{
		AuthMiddleware: authMiddleware,
		disableUI:      p.Opts.NoUI,
	})

	core, err := coreapi.NewCoreApi(coreapi.Options{
		AuthMiddleware: authMiddleware,
		Data:           p.DualWrite.data,
		DuckDB:         p.DualWrite.readDB,
		Config:         p.Opts.Config,
		Logger:         p.Log,
		Runner:         p.Runner,
		State:          p.Devserver.State,
		QueueReader:    p.Queue,
		EventHandler:   p.Devserver.HandleEvent,
		Executor:       p.Executor,
		HistoryReader:  cqrsmanager.NewHistoryReader(p.Database),
		DisableGraphQL: &p.Opts.NoUI,
		ConnectOpts: connectv0.Opts{
			GroupManager:               p.ConnectState,
			ConnectManager:             p.ConnectState,
			ConnectRequestStateManager: p.ConnectState,
			Signer:                     auth.NewJWTSessionTokenSigner(consts.DevServerConnectJwtSecret),
			RequestAuther:              p.Devserver,
			ConnectGatewayRetriever:    p.Devserver,
			Dev:                        true,
			EntitlementProvider:        p.Devserver,
			ConditionalTracer:          p.ConnectTracer,
			ConnectGRPCConfig:          p.Opts.ConnectGRPCConfig,
		},
	})
	if err != nil {
		return nil, err
	}

	devAPI.Route("/v1", func(r chi.Router) {
		p.addV1Routes(r, core, authMiddleware)
	})

	// Initialize metrics API for Prometheus-compatible metrics endpoint.
	// This provides system queue depth metrics via /metrics endpoint.
	metricsAPI, err := metrics.NewMetricsAPI(metrics.Opts{
		QueueManager: p.Shard,
	})
	if err != nil {
		return nil, err
	}

	apiv2Handler, err := apiv2.NewHTTPHandler(p.Context, p.v2ServiceOptions(), apiv2.HTTPHandlerOptions{
		AuthnMiddleware: authMiddleware,
	}, apiv2base.NewBase())
	if err != nil {
		return nil, fmt.Errorf("failed to create v2 handler: %w", err)
	}

	mcpHandler := newMCPHandler(p.Devserver.HandleEvent, p.CQRS, p.Opts.Tick, apiv2Handler)
	addMCPRoute(devAPI, mcpHandler)

	mounts := []api.Mount{
		{At: "/", Router: devAPI},
		{At: "/v0", Router: core.Router},
		{At: "/api/v2/operations", Handler: http.HandlerFunc(mcpHandler.Operations)},
		{At: "/v2/operations", Handler: http.HandlerFunc(mcpHandler.Operations)},
		{At: "/api/v2", Handler: apiv2Handler},
		{At: "/v2", Handler: apiv2Handler},
		{At: "/debug", Handler: middleware.Profiler()},
		{At: "/metrics", Router: metricsAPI.Router},
	}

	if p.Opts.CloudSandboxes != nil {
		for _, path := range []string{"/dev/cloud", "/v2/sandboxes", "/v2/snapshots", "/api/v2/sandboxes", "/api/v2/snapshots"} {
			mounts = append(mounts, api.Mount{At: path, Handler: p.Opts.CloudSandboxes})
		}
	}

	if testapi.ShouldEnable() {
		mounts = append(mounts, api.Mount{At: "/test", Handler: testapi.New(testapi.Options{
			QueueShards:  p.Registry,
			Executor:     p.Executor,
			StateManager: p.RunService,
			ResetAll:     p.Clients.flush,
			PauseFunction: func(id uuid.UUID) {
				p.Functions.pauses.pause(id)
			},
		})})
	}

	return api.NewService(api.APIServiceOptions{
		Config:         p.Opts.Config,
		Mounts:         mounts,
		LocalEventKeys: p.Opts.EventKeys,
		Logger:         p.Log,
	}), nil
}

// addV1Routes adds the REST API v1 to r.
func (p apiParams) addV1Routes(r chi.Router, core *coreapi.CoreAPI, authMiddleware func(http.Handler) http.Handler) {
	cache := cache.New[[]byte](freecachestore.NewFreecache(freecache.NewCache(1024 * 1024)))

	apiv1.AddRoutes(r, apiv1.Opts{
		AuthMiddleware:    authMiddleware,
		CachingMiddleware: apiv1.NewCacheMiddleware(cache),
		FunctionReader:    p.CQRS,
		Executor:          p.Executor,
		QueueProducer:     p.Queue,
		RunQueueReader:    p.Queue,
		QueueItemReader:   p.Queue,
		AttemptResetter:   p.Queue,
		QueueShards:       p.Registry,
		Broadcaster:       p.Broadcaster,
		TraceReader:       p.DualWrite.data,

		AppCreator:             p.CQRS,
		FunctionCreator:        p.CQRS,
		EventPublisher:         p.Runner,
		TracerProvider:         p.TracerProvider,
		SyncLifecycleListeners: p.DualWrite.listeners,
		State:                  p.RunService,
		RealtimeJWTSecret:      consts.DevServerRealtimeJWTSecret,

		CheckpointOpts: apiv1.CheckpointAPIOpts{
			RunOutputReader: devutil.NewLocalOutputReader(core.Resolver(), p.DualWrite.data, p.DualWrite.data),
			RunJWTSecret:    consts.DevServerRunJWTSecret,
			BackoffFunc:     p.Retry.backoff,
			AllowAsyncDispatchValidation: func(ctx context.Context, acctID uuid.UUID) bool {
				return p.Experiments.asyncDispatchValidation
			},
		},

		MetadataOpts: apiv1.MetadataOpts{
			Flag: func(ctx context.Context, accountID uuid.UUID) bool {
				return p.Experiments.stepMetadata
			},
			SpanExtractor: extractors.DefaultSpanExtractors,
		},
	})
}

// v2ServiceOptions are the providers of the REST API v2.
func (p apiParams) v2ServiceOptions() apiv2.ServiceOptions {
	return apiv2.ServiceOptions{
		SigningKeysProvider: apiv2.NewSigningKeysProvider(p.Opts.SigningKey),
		EventKeysProvider:   apiv2.NewEventKeysProvider(p.Opts.EventKeys),
		Apps:                NewAppProvider(p.CQRS),
		Functions:           p.Functions,
		FunctionPauser:      p.Functions,
		Cancellations:       NewCancellationProvider(cancellation.NewRedisWriter(p.Clients.unsharded, p.Queue, "")),
		Runs:                NewRunProvider(p.DualWrite.data, p.Executor),
		Events:              NewEventProvider(p.DualWrite.data),
		Replays: NewReplayProvider(p.CQRS, p.Functions, p.Executor, func(id uuid.UUID) bool {
			return !p.Functions.pauses.get(id).IsZero()
		}),
		FunctionTraces: NewFunctionTraceReader(p.DualWrite.data),
		Executor:       p.Executor,
		EventPublisher: p.Runner,
		// nil means QueryInsights returns an error instead of an empty result.
		DuckDB: p.DualWrite.readDB,
		EventSender: func(ctx context.Context, evt *event.Event) (string, error) {
			return p.Devserver.HandleEvent(ctx, evt, nil)
		},
		Scores: apiv2.NewStateScoreProvider(apiv2.StateScoreProviderOptions{
			State:          p.RunService,
			TracerProvider: p.TracerProvider,
			Auth: func(ctx context.Context) (uuid.UUID, uuid.UUID, error) {
				return consts.DevServerAccountID, consts.DevServerEnvID, nil
			},
			Enabled: func(ctx context.Context, accountID uuid.UUID) bool {
				return p.Experiments.stepMetadata
			},
			MissingStateLoader: scoreMetadataLoader(p.CQRS),
		}),
	}
}

// newConnectGateway serves the connect gateway, which connect workers open
// a WebSocket to.
func newConnectGateway(
	opts StartOpts,
	connectState connstate.StateManager,
	data cqrs.Manager,
	semaphores constraintapi.SemaphoreManager,
) service.Service {
	return connect.NewConnectGatewayService(
		connect.WithConnectionStateManager(connectState),
		connect.WithGatewayAuthHandler(auth.NewJWTAuthHandler(consts.DevServerConnectJwtSecret)),
		connect.WithDev(),
		connect.WithGatewayPublicPort(opts.ConnectGatewayPort),
		connect.WithGRPCConfig(opts.ConnectGRPCConfig),
		connect.WithApiBaseUrl(fmt.Sprintf("http://%s:%d", opts.Config.CoreAPI.Addr, opts.Config.CoreAPI.Port)),
		connect.WithLifeCycles(
			[]connect.ConnectGatewayLifecycleListener{
				lifecycles.NewHistoryLifecycle(data),
				lifecycles.NewSemaphoreLifecycleListener(semaphores, time.Time{}),
			}),
	)
}

// newLeaseScavenger releases the Constraint API leases that expire.
func newLeaseScavenger(capacity capacityManager) service.Service {
	return constraintapi.NewLeaseScavengerService(capacity, consts.ConstraintAPIScavengerTick)
}

type debugAPIParams struct {
	fx.In

	Opts       StartOpts
	Log        logger.Logger
	CQRS       cqrs.Manager
	Queue      queueProcessor
	State      state.Manager
	Croner     cron.CronManager
	Registry   queue.ShardRegistryController
	Pauses     pauses.Manager
	Capacity   capacityManager
	Semaphores constraintapi.SemaphoreManager
	Batcher    batch.BatchManager
	Debouncer  debounce.Debouncer
}

// newDebugAPI serves the debug API on StartOpts.DebugAPIPort.  options adds
// it only when the DEBUG environment variable is set.
func newDebugAPI(p debugAPIParams) service.Service {
	return debugapi.NewDebugAPI(debugapi.Opts{
		Log:              p.Log,
		DB:               p.CQRS,
		RunReader:        p.Queue,
		ItemReader:       p.Queue,
		PartitionReader:  p.Queue,
		BacklogReader:    p.Queue,
		State:            p.State,
		Cron:             p.Croner,
		ShardRegistry:    p.Registry,
		Port:             p.Opts.DebugAPIPort,
		PauseManager:     p.Pauses,
		CapacityManager:  p.Capacity,
		SemaphoreManager: p.Semaphores,
		// Dependencies for batching and debounce insights
		BatchManager: p.Batcher,
		Debouncer:    p.Debouncer,
	})
}
