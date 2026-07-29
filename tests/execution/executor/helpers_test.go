package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	cqrsmanager "github.com/inngest/inngest/pkg/cqrs/manager"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// execTestInfra holds the shared state manager, queue, and loader used by
// tests in this package so each test can spin up runs against the same
// backing store without re-wiring cqrs/redis/queue by hand.
type execTestInfra struct {
	ctx           context.Context
	fn            inngest.Function
	fnID          uuid.UUID
	wsID          uuid.UUID
	appID         uuid.UUID
	aID           uuid.UUID
	smv2          statev2.RunService
	pauseMgr      pauses.Manager
	loader        state.FunctionLoader
	dbcqrs        cqrs.Manager
	adapter       *dbsqlite.Adapter
	queueShard    redis_state.RedisQueueShard
	shardRegistry queue.ShardRegistryController
	rq            queue.Queue
}

// newExecTestInfra builds the shared executor test infra. stepID is
// parameterized so callers can pin their function's single step to whatever
// ID their opcodes/fixtures target.
func newExecTestInfra(t *testing.T, stepID string) *execTestInfra {
	t.Helper()
	ctx := logger.WithStdlib(context.Background(), logger.VoidLogger())

	db, err := dbsqlite.Open(ctx, dbsqlite.Options{Persist: false, ForTest: true})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	adapter := dbsqlite.New(db)
	dbcqrs := cqrsmanager.New(adapter)
	loader := dbcqrs.(state.FunctionLoader)

	fnID, wsID, appID, aID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ID:              fnID,
		FunctionVersion: 1,
		Name:            "test-fn",
		Slug:            "test-fn",
		Steps: []inngest.Step{
			{ID: stepID, Name: stepID, URI: "/" + stepID},
		},
	}

	config, err := json.Marshal(fn)
	require.NoError(t, err)

	_, err = dbcqrs.UpsertApp(ctx, cqrs.UpsertAppParams{ID: appID, Name: "test-app"})
	require.NoError(t, err)
	_, err = dbcqrs.UpsertFunction(ctx, cqrs.UpsertFunctionParams{
		ID: fnID, AppID: appID, Name: fn.Name, Slug: fn.Slug, Config: string(config),
	})
	require.NoError(t, err)

	_, shardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	t.Cleanup(func() { shardedRc.Close() })

	_, unshardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	t.Cleanup(func() { unshardedRc.Close() })

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
	sm, err := redis_state.New(ctx,
		redis_state.WithShardedClient(shardedClient),
		redis_state.WithPauseDeleter(pauseMgr),
	)
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	queueOpts := []queue.QueueOpt{queue.WithIdempotencyTTL(time.Hour)}
	queueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), queueOpts...)

	shardRegistry, err := queue.NewSingleShardRegistry(queueShard)
	require.NoError(t, err)

	rq, err := queue.New(ctx, "test-queue", shardRegistry, queueOpts...)
	require.NoError(t, err)

	return &execTestInfra{
		ctx:           ctx,
		fn:            fn,
		fnID:          fnID,
		wsID:          wsID,
		appID:         appID,
		aID:           aID,
		smv2:          smv2,
		pauseMgr:      pauseMgr,
		loader:        loader,
		dbcqrs:        dbcqrs,
		adapter:       adapter,
		queueShard:    queueShard,
		shardRegistry: shardRegistry,
		rq:            rq,
	}
}

// newExecutor builds an executor wired to the shared infra's default queue.
// Base opts (state, pause manager, queue, logger, loader, shard registry,
// tracer provider) are set first; extra is appended last so scalar setters
// (e.g. WithTracerProvider, WithInvokeEventHandler) can override the
// defaults, and callers can add WithLifecycleListeners (additive) or
// WithDriverV1. The base opts intentionally never register a driver:
// WithDriverV1 errors on a duplicate driver name, so the driver must always
// come from the caller's extra opts.
func (i *execTestInfra) newExecutor(t *testing.T, extra ...executor.ExecutorOpt) execution.Executor {
	t.Helper()
	return i.newExecutorWithQueue(t, i.rq, extra...)
}

// newExecutorWithQueue is newExecutor with an overridable queue, used by the
// discovery-enqueue tests that wrap the shared queue in enqueueCountingQueue.
func (i *execTestInfra) newExecutorWithQueue(t *testing.T, q queue.Queue, extra ...executor.ExecutorOpt) execution.Executor {
	t.Helper()

	opts := []executor.ExecutorOpt{
		executor.WithStateManager(i.smv2),
		executor.WithPauseManager(i.pauseMgr),
		executor.WithQueue(q),
		executor.WithLogger(logger.StdlibLogger(i.ctx)),
		executor.WithFunctionLoader(i.loader),
		executor.WithShardRegistry(i.shardRegistry),
		executor.WithTracerProvider(tracing.NewSqlcTracerProvider(i.adapter.Q())),
	}
	opts = append(opts, extra...)

	exec, err := executor.NewExecutor(opts...)
	require.NoError(t, err)
	return exec
}

// scheduleRun kicks off a fresh run and returns its metadata.
func (i *execTestInfra) scheduleRun(t *testing.T, exec execution.Executor) *statev2.Metadata {
	t.Helper()
	now := time.Now()
	evtID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	_, run, err := exec.Schedule(i.ctx, execution.ScheduleRequest{
		Function: i.fn, At: &now, AccountID: i.aID, WorkspaceID: i.wsID, AppID: i.appID,
		Events: []event.TrackedEvent{
			event.NewBaseTrackedEventWithID(event.Event{Name: "test/event"}, evtID),
		},
	})
	require.NoError(t, err)
	return run
}
