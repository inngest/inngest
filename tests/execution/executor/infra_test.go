package executor

// Shared test infrastructure for the executor package's test suite: fixture
// builders, fakes, and mocks used across multiple _test.go files.

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	cqrsmanager "github.com/inngest/inngest/pkg/cqrs/manager"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/checkpoint"
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
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// deferTestInfra holds the shared state manager, queue, and loader used by the
// checkpoint-vs-executor consistency tests so each test can spin up 3 runs
// against the same backing store.
type deferTestInfra struct {
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

func newDeferTestInfra(t *testing.T) *deferTestInfra {
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
			{ID: "step-defer", Name: "step-defer", URI: "/step-defer"},
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

	return &deferTestInfra{
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

// newExecutor builds an executor wired to the shared infra. Pass a non-nil
// driver to drive Execute() calls; pass nil when only the checkpointer will
// be used.
func (i *deferTestInfra) newExecutor(t *testing.T, driver *mockDriverV1) execution.Executor {
	t.Helper()
	return i.newExecutorWithQueue(t, i.rq, driver)
}

// newExecutorWithQueue is newExecutor with an overridable queue, used by the
// discovery-enqueue tests that wrap the shared queue in enqueueCountingQueue.
func (i *deferTestInfra) newExecutorWithQueue(t *testing.T, q queue.Queue, driver *mockDriverV1) execution.Executor {
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
	if driver != nil {
		opts = append(opts, executor.WithDriverV1(driver))
	}

	exec, err := executor.NewExecutor(opts...)
	require.NoError(t, err)
	return exec
}

// newCheckpointer builds a Checkpointer using the shared infra. The Executor
// is passed in so the checkpointer can reuse the same handler for non-Defer
// async opcodes; for Defer-only tests, any executor works.
func (i *deferTestInfra) newCheckpointer(t *testing.T, exec execution.Executor) checkpoint.Checkpointer {
	t.Helper()
	return checkpoint.New(checkpoint.Opts{
		State:          i.smv2,
		FnReader:       i.dbcqrs,
		Executor:       exec,
		TracerProvider: tracing.NewSqlcTracerProvider(i.adapter.Q()),
		Queue:          i.rq,
	})
}

// scheduleRun kicks off a fresh run and returns its metadata.
func (i *deferTestInfra) scheduleRun(t *testing.T, exec execution.Executor) *statev2.Metadata {
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

// enqueueCountingQueue wraps a queue.Queue and counts Enqueue calls. Reads
// happen post-Execute (after eg.Wait), so the field can be read without
// locking; the mutex protects the increment side from concurrent op handlers.
type enqueueCountingQueue struct {
	queue.Queue
	mu       sync.Mutex
	enqueues int
}

func (q *enqueueCountingQueue) Enqueue(ctx context.Context, item queue.Item, at time.Time, opts queue.EnqueueOpts) error {
	q.mu.Lock()
	q.enqueues++
	q.mu.Unlock()
	return q.Queue.Enqueue(ctx, item, at, opts)
}

// pendingCapturingState wraps a real RunService and captures every SavePending
// call so tests can assert on what the executor handed off to the state layer.
// All other methods pass through.
type pendingCapturingState struct {
	statev2.RunService
	mu       sync.Mutex
	captured [][]string
}

func (s *pendingCapturingState) SavePending(ctx context.Context, id statev2.ID, pending []string) error {
	s.mu.Lock()
	s.captured = append(s.captured, append([]string(nil), pending...))
	s.mu.Unlock()
	return s.RunService.SavePending(ctx, id, pending)
}

func (s *pendingCapturingState) calls() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]string, len(s.captured))
	for i, c := range s.captured {
		out[i] = append([]string(nil), c...)
	}
	return out
}

func createInmemoryRedis(t *testing.T) (*miniredis.Miniredis, rueidis.Client, error) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	if err != nil {
		return nil, nil, err
	}

	// If tick is lower than the default, tick every 50ms.  This lets us save
	// CPU for standard dev-server testing.
	poll := 150 * time.Millisecond

	go func() {
		for range time.Tick(poll) {
			r.FastForward(poll)
		}
	}()
	return r, rc, nil
}

// mockDriverV1 implements driver.DriverV1 for testing
type mockDriverV1 struct {
	response *state.DriverResponse
	err      error
	t        *testing.T
}

func (m *mockDriverV1) Name() string { return "http" }

func (m *mockDriverV1) Execute(
	ctx context.Context,
	sl statev2.StateLoader,
	md statev2.Metadata,
	item queue.Item,
	edge inngest.Edge,
	step inngest.Step,
	stackIndex int,
	attempt int,
) (*state.DriverResponse, error) {
	return m.response, m.err
}
