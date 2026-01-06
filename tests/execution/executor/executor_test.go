package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
	sqlc_postgres "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

type hookData struct {
	runID ulid.ULID
}

func newFakeLifecycle(c chan *hookData) execution.LifecycleListener {
	return &fakeLifecycle{
		work: c,
	}
}

type fakeLifecycle struct {
	execution.NoopLifecyceListener

	work chan *hookData
}

func (f *fakeLifecycle) OnFunctionScheduled(
	ctx context.Context,
	md statev2.Metadata,
	qi queue.Item,
	evt []event.TrackedEvent,
) {
	f.work <- &hookData{runID: md.ID.RunID}
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

type fakeQueue struct {
	queue.Queue

	lock              sync.Mutex
	dequeuedCalled    map[string]int64
	dequeuedCalledRun map[ulid.ULID]int64
}

func (fq *fakeQueue) reset() {
	fq.lock.Lock()
	fq.dequeuedCalled = make(map[string]int64)
	fq.dequeuedCalledRun = make(map[ulid.ULID]int64)
	fq.lock.Unlock()
}

func (fq *fakeQueue) RemoveQueueItem(ctx context.Context, shard string, partitionKey string, itemID string) error {
	qm := fq.Queue.(redis_state.QueueManager)
	return qm.RemoveQueueItem(ctx, shard, partitionKey, itemID)
}

func (fq *fakeQueue) Requeue(ctx context.Context, queueShard redis_state.QueueShard, i queue.QueueItem, at time.Time) error {
	qm := fq.Queue.(redis_state.QueueManager)
	return qm.Requeue(ctx, queueShard, i, at)
}

func (fq *fakeQueue) RequeueByJobID(ctx context.Context, queueShard redis_state.QueueShard, jobID string, at time.Time) error {
	qm := fq.Queue.(redis_state.QueueManager)
	return qm.RequeueByJobID(ctx, queueShard, jobID, at)
}

func (fq *fakeQueue) Dequeue(ctx context.Context, queueShard redis_state.QueueShard, i queue.QueueItem) error {
	qm := fq.Queue.(redis_state.QueueManager)

	err := qm.Dequeue(ctx, queueShard, i)

	fq.lock.Lock()
	logger.StdlibLogger(ctx).Info("called fakeQueue.Dequeue()", "i", i, "err", err)
	fq.dequeuedCalled[i.ID]++
	fq.dequeuedCalledRun[i.Data.Identifier.RunID]++
	fq.lock.Unlock()

	return err
}

func newFakeQueue(q queue.Queue) *fakeQueue {
	fq := &fakeQueue{
		Queue:             q,
		lock:              sync.Mutex{},
		dequeuedCalled:    make(map[string]int64),
		dequeuedCalledRun: make(map[ulid.ULID]int64),
	}

	return fq
}

func TestScheduleRaceCondition(t *testing.T) {
	ctx := context.Background()
	_ = trace.UserTracer()
	work := make(chan *hookData)

	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{Persist: false})
	require.NoError(t, err)

	// Initialize the devserver
	dbDriver := "sqlite"
	dbcqrs := base_cqrs.NewCQRS(db, dbDriver, sqlc_postgres.NewNormalizedOpts{})
	loader := dbcqrs.(state.FunctionLoader)

	_, shardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer shardedRc.Close()

	_, unshardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer unshardedRc.Close()

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
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	queueOpts := []redis_state.QueueOpt{
		redis_state.WithIdempotencyTTL(time.Hour),
		redis_state.WithShardSelector(shardSelector),
		redis_state.WithQueueShardClients(queueShards),
	}

	rq := redis_state.NewQueue(queueShard, queueOpts...)

	exec, err := executor.NewExecutor(
		executor.WithStateManager(smv2),
		executor.WithPauseManager(pauses.NewRedisOnlyManager(sm)),
		executor.WithQueue(rq),
		executor.WithLogger(logger.StdlibLogger(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithLifecycleListeners(newFakeLifecycle(work)),
		executor.WithAssignedQueueShard(queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTraceReader(dbcqrs),
		executor.WithTracerProvider(tracing.NewSqlcTracerProvider(base_cqrs.NewQueries(db, dbDriver, sqlc_postgres.NewNormalizedOpts{}))),
	)
	require.NoError(t, err)

	fnID, accountID, wsID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ConfigVersion:   0,
		ID:              fnID,
		FunctionVersion: 0,
		Name:            "",
		Slug:            "",
	}

	now := time.Now()

	at := now

	key := "same-idempotency-key"

	wg := sync.WaitGroup{}

	var (
		runnerLock             sync.Mutex
		successCount, errCount int64
		successMetaIDs         []ulid.ULID
		successEventIDs        []ulid.ULID

		// variables for capturing lifecycle hook
		scheduledCnt int
		hookRunIDs   []ulid.ULID
	)

	iterations := 100
	go func() {
		for range iterations {
			wg.Add(1)

			go func() {
				defer wg.Done()

				evtID := ulid.MustNew(ulid.Timestamp(at), rand.Reader)

				evt := event.NewBaseTrackedEvent(event.Event{
					Name: "cron-resumed",
					ID:   evtID.String(),
				}, event.SeededIDFromString("", 0))
				md, err := exec.Schedule(ctx, execution.ScheduleRequest{
					Function:       fn,
					At:             &at,
					AccountID:      accountID,
					WorkspaceID:    wsID,
					AppID:          appID,
					Events:         []event.TrackedEvent{evt},
					IdempotencyKey: &key,
				})

				runnerLock.Lock()
				defer runnerLock.Unlock()

				if err != nil {
					errCount++
					return
				}

				successCount++
				successMetaIDs = append(successMetaIDs, md.ID.RunID)
				successEventIDs = append(successEventIDs, evtID)
			}()
		}

		wg.Wait()

		// NOTE: close the channel after a bit of time
		// so there's enough room for the lifecycle hooks to execute
		// otherwise there'll be a data race
		<-time.After(2 * time.Second)
		close(work)
	}()

	for hook := range work {
		scheduledCnt++
		hookRunIDs = append(hookRunIDs, hook.runID)
	}

	require.Equal(t, 1, int(successCount))
	require.Equal(t, iterations-1, int(errCount))

	require.Len(t, successMetaIDs, 1)

	require.Equal(t, 1, scheduledCnt)
	require.Len(t, hookRunIDs, 1)

	// This is expected: One could reach the state creation earlier, BUT: the run ID must diverge
	// require.Equal(t, testLifecycle.evtIDs[0], successEventIDs[0])

	require.Equal(t, hookRunIDs[0], successMetaIDs[0])
}

func TestScheduleRaceConditionWithExistingIdempotencyKey(t *testing.T) {
	_ = trace.UserTracer()
	ctx := context.Background()

	work := make(chan *hookData)

	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{Persist: false})
	require.NoError(t, err)

	// Initialize the devserver
	dbDriver := "sqlite"
	dbcqrs := base_cqrs.NewCQRS(db, dbDriver, sqlc_postgres.NewNormalizedOpts{})
	loader := dbcqrs.(state.FunctionLoader)

	stateRedis, shardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer shardedRc.Close()

	_, unshardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer unshardedRc.Close()

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
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	queueOpts := []redis_state.QueueOpt{
		redis_state.WithIdempotencyTTL(time.Hour),
		redis_state.WithShardSelector(shardSelector),
		redis_state.WithQueueShardClients(queueShards),
	}

	rq := redis_state.NewQueue(queueShard, queueOpts...)

	exec, err := executor.NewExecutor(
		executor.WithStateManager(smv2),
		executor.WithPauseManager(pauses.NewRedisOnlyManager(sm)),
		executor.WithQueue(rq),
		executor.WithLogger(logger.StdlibLogger(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithLifecycleListeners(newFakeLifecycle(work)),
		executor.WithAssignedQueueShard(queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTraceReader(dbcqrs),
	)
	require.NoError(t, err)

	accountID, wsID, appID, fnID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ConfigVersion:   0,
		ID:              fnID,
		FunctionVersion: 0,
		Name:            "",
		Slug:            "",
	}

	now := time.Now()
	at := now

	key := "same-idempotency-key"

	var (
		runnerLock             sync.Mutex
		successCount, errCount int64
		successMetaIDs         []ulid.ULID
		successEventIDs        []ulid.ULID

		// variables for capturing lifecycle hook
		scheduledCnt int
		hookRunIDs   []ulid.ULID
	)

	wg := sync.WaitGroup{}
	fakeRunID := ulid.MustNew(ulid.Timestamp(now.Add(-16*time.Hour)), rand.Reader)

	// Simulate an existing request
	require.NoError(t, stateRedis.Set(shardedClient.FunctionRunState().KeyGenerator().Idempotency(ctx, true, state.Identifier{
		WorkflowID: fnID,
		Key:        fmt.Sprintf("%s-%s", util.XXHash(fnID.String()), util.XXHash(key)),
		AccountID:  accountID,
	}), fakeRunID.String()))

	iterations := 10
	go func() {
		for range iterations {
			wg.Add(1)

			go func() {
				defer wg.Done()

				evtID := ulid.MustNew(ulid.Timestamp(at), rand.Reader)

				evt := event.NewBaseTrackedEvent(event.Event{
					Name: "cron-resumed",
					ID:   evtID.String(),
				}, event.SeededIDFromString("", 0))

				md, err := exec.Schedule(ctx, execution.ScheduleRequest{
					Function:       fn,
					At:             &at,
					AccountID:      accountID,
					WorkspaceID:    wsID,
					AppID:          appID,
					Events:         []event.TrackedEvent{evt},
					IdempotencyKey: &key,
				})

				runnerLock.Lock()
				defer runnerLock.Unlock()

				if err != nil {
					errCount++
					return
				}

				successCount++
				successMetaIDs = append(successMetaIDs, md.ID.RunID)
				successEventIDs = append(successEventIDs, evtID)
			}()
		}

		wg.Wait()

		// NOTE: close the channel after a bit of time
		// so there's enough room for the lifecycle hooks to execute
		// otherwise there'll be a data race
		<-time.After(2 * time.Second)
		close(work)
	}()

	for hook := range work {
		scheduledCnt++
		hookRunIDs = append(hookRunIDs, hook.runID)
	}

	// NOTE: event IDs being different is expected

	require.Equal(t, 1, int(successCount))
	require.Equal(t, iterations-1, int(errCount))
	require.Len(t, successMetaIDs, 1)

	require.Equal(t, 1, scheduledCnt)
	require.Len(t, hookRunIDs, 1)
	require.Equal(t, hookRunIDs[0], successMetaIDs[0])
	require.Equal(t, fakeRunID, hookRunIDs[0])
}

func TestFinalize(t *testing.T) {
	t.Skip("this is flaky but helpful to understand finalize behavior")

	ctx := context.Background()
	_ = trace.UserTracer()
	work := make(chan *hookData)

	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{Persist: false})
	require.NoError(t, err)

	// Initialize the devserver
	dbDriver := "sqlite"
	dbcqrs := base_cqrs.NewCQRS(db, dbDriver, sqlc_postgres.NewNormalizedOpts{})
	loader := dbcqrs.(state.FunctionLoader)

	fnID, accountID, wsID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ID:              fnID,
		FunctionVersion: 1,
		Name:            "test-fn",
		Slug:            "test-fn",
	}

	config, err := json.Marshal(fn)
	require.NoError(t, err)

	_, err = dbcqrs.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:   appID,
		Name: "test-app",
	})
	require.NoError(t, err)

	_, err = dbcqrs.InsertFunction(ctx, cqrs.InsertFunctionParams{
		ID:     fnID,
		AppID:  appID,
		Name:   fn.Name,
		Slug:   fn.Slug,
		Config: string(config),
	})
	require.NoError(t, err)

	_, shardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer shardedRc.Close()

	unshardedCluster, unshardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer unshardedRc.Close()

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
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	queueOpts := []redis_state.QueueOpt{
		redis_state.WithIdempotencyTTL(time.Hour),
		redis_state.WithShardSelector(shardSelector),
		redis_state.WithQueueShardClients(queueShards),
	}

	rq := redis_state.NewQueue(queueShard, queueOpts...)

	testQueue := newFakeQueue(rq)

	exec, err := executor.NewExecutor(
		executor.WithStateManager(smv2),
		executor.WithPauseManager(pauses.NewRedisOnlyManager(sm)),
		executor.WithQueue(testQueue),
		executor.WithLogger(logger.StdlibLogger(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithLifecycleListeners(newFakeLifecycle(work)),
		executor.WithAssignedQueueShard(queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTraceReader(dbcqrs),
	)
	require.NoError(t, err)

	now := time.Now()
	evtID1, evtID2 := ulid.MustNew(ulid.Timestamp(now), rand.Reader), ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	//
	// Schedule two runs
	//

	run1, err := exec.Schedule(ctx, execution.ScheduleRequest{
		Function:    fn,
		At:          &now,
		AccountID:   accountID,
		WorkspaceID: wsID,
		AppID:       appID,
		Events: []event.TrackedEvent{
			event.NewBaseTrackedEventWithID(event.Event{
				Name: "test/event",
			}, evtID1),
		},
	})
	require.NoError(t, err)

	kg := unshardedClient.Queue().KeyGenerator()

	// Validate first run

	require.True(t, unshardedCluster.Exists(kg.RunIndex(run1.ID.RunID)))

	state1, err := smv2.LoadState(ctx, run1.ID)
	require.NoError(t, err)

	require.Equal(t, 1, len(state1.Metadata.Config.EventIDs))

	jobs1, err := rq.RunJobs(
		ctx,
		queueShard.Name,
		run1.ID.Tenant.EnvID,
		run1.ID.FunctionID,
		run1.ID.RunID,
		1000,
		0,
	)
	require.NoError(t, err)

	require.Len(t, jobs1, 1)
	require.Equal(t, queue.KindStart, jobs1[0].Kind)

	run2, err := exec.Schedule(ctx, execution.ScheduleRequest{
		Function:    fn,
		At:          &now,
		AccountID:   accountID,
		WorkspaceID: wsID,
		AppID:       appID,
		Events: []event.TrackedEvent{
			event.NewBaseTrackedEventWithID(event.Event{
				Name: "test/event",
			}, evtID2),
		},
	})
	require.NoError(t, err)

	// Validate second run
	require.True(t, unshardedCluster.Exists(kg.RunIndex(run2.ID.RunID)))

	state2, err := smv2.LoadState(ctx, run2.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(state2.Metadata.Config.EventIDs))

	jobs2, err := rq.RunJobs(
		ctx,
		queueShard.Name,
		run2.ID.Tenant.EnvID,
		run2.ID.FunctionID,
		run2.ID.RunID,
		1000,
		0,
	)
	require.NoError(t, err)

	require.Len(t, jobs2, 1)

	var item2 queue.QueueItem
	require.NoError(t, json.Unmarshal([]byte(unshardedCluster.HGet(kg.QueueItem(), jobs2[0].JobID)), &item2))

	t.Run("racing finalize will dequeue each other", func(t *testing.T) {
		//
		// Cancel both runs concurrently (we really only care about finalize)
		//

		eg := errgroup.Group{}
		eg.Go(func() error {
			return exec.Cancel(ctx, run1.ID, execution.CancelRequest{})
		})
		eg.Go(func() error {
			return exec.Cancel(ctx, run1.ID, execution.CancelRequest{})
		})

		require.NoError(t, eg.Wait())

		testQueue.lock.Lock()
		require.Greater(t, len(testQueue.dequeuedCalled), 0)
		require.Equal(t, int64(2), testQueue.dequeuedCalled[jobs1[0].JobID])
		require.Greater(t, len(testQueue.dequeuedCalledRun), 0)
		require.Equal(t, int64(2), testQueue.dequeuedCalledRun[run1.ID.RunID])
		testQueue.lock.Unlock()
	})

	t.Run("finalize run from a queue item will Dequeue itself", func(t *testing.T) {
		//
		// Cancel both runs concurrently (we really only care about finalize)
		//

		testQueue.reset()

		err := exec.Cancel(ctx, run2.ID, execution.CancelRequest{})
		require.NoError(t, err)

		testQueue.lock.Lock()
		require.Greater(t, len(testQueue.dequeuedCalled), 0)
		require.Equal(t, int64(1), testQueue.dequeuedCalled[jobs2[0].JobID])
		require.Greater(t, len(testQueue.dequeuedCalledRun), 0)
		require.Equal(t, int64(1), testQueue.dequeuedCalledRun[run2.ID.RunID])
		testQueue.lock.Unlock()

		err = rq.Dequeue(ctx, queueShard, item2)
		require.ErrorIs(t, err, redis_state.ErrQueueItemNotFound)

		err = rq.Requeue(ctx, queueShard, item2, time.Now())
		require.ErrorIs(t, err, redis_state.ErrQueueItemNotFound)
	})
}

// mockDriverV1 implements driver.DriverV1 for testing
type mockDriverV1 struct {
	response *state.DriverResponse
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
	return m.response, nil
}

// This tests a scenario where a run used to hang when retrying an invoke and the invoke's pause was already created.
func TestInvokeRetrySucceedsIfPauseAlreadyCreated(t *testing.T) {
	ctx := context.Background()

	// Set up database and function loader
	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{Persist: false})
	require.NoError(t, err)

	dbDriver := "sqlite"
	dbcqrs := base_cqrs.NewCQRS(db, dbDriver, sqlc_postgres.NewNormalizedOpts{})
	loader := dbcqrs.(state.FunctionLoader)

	fnID, wsID, appID, aID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	targetFnID := uuid.New()

	// Create the invoking function
	fn := inngest.Function{
		ID:              fnID,
		FunctionVersion: 1,
		Name:            "test-fn",
		Slug:            "test-fn",
		Steps: []inngest.Step{
			{
				ID:   "invoke-step",
				Name: "invoke-step",
				URI:  "/invoke-step",
			},
		},
	}

	// Create the target function being invoked
	targetFn := inngest.Function{
		ID:              targetFnID,
		FunctionVersion: 1,
		Name:            "target-fn",
		Slug:            "target-fn",
	}

	config, err := json.Marshal(fn)
	require.NoError(t, err)
	targetConfig, err := json.Marshal(targetFn)
	require.NoError(t, err)

	_, err = dbcqrs.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:   appID,
		Name: "test-app",
	})
	require.NoError(t, err)

	_, err = dbcqrs.InsertFunction(ctx, cqrs.InsertFunctionParams{
		ID:     fnID,
		AppID:  appID,
		Name:   fn.Name,
		Slug:   fn.Slug,
		Config: string(config),
	})
	require.NoError(t, err)

	_, err = dbcqrs.InsertFunction(ctx, cqrs.InsertFunctionParams{
		ID:     targetFnID,
		AppID:  appID,
		Name:   targetFn.Name,
		Slug:   targetFn.Slug,
		Config: string(targetConfig),
	})
	require.NoError(t, err)

	// Set up Redis
	_, shardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer shardedRc.Close()

	_, unshardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer unshardedRc.Close()

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
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	queueOpts := []redis_state.QueueOpt{
		redis_state.WithIdempotencyTTL(time.Hour),
		redis_state.WithShardSelector(shardSelector),
		redis_state.WithQueueShardClients(queueShards),
	}

	rq := redis_state.NewQueue(queueShard, queueOpts...)

	genID := "invoke-step"

	mockDriver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{{
				Op: enums.OpcodeInvokeFunction,
				ID: genID,
				Opts: map[string]any{
					"function_id": uuid.New().String(),
					"payload":     map[string]any{"data": map[string]any{"test": "value"}},
				},
			}},
		},
	}

	pm := pauses.NewRedisOnlyManager(sm)

	eventCaptured := false

	exec, err := executor.NewExecutor(
		executor.WithStateManager(smv2),
		executor.WithPauseManager(pm),
		executor.WithQueue(rq),
		executor.WithLogger(logger.StdlibLogger(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithAssignedQueueShard(queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond)),
		executor.WithSendingEventHandler(func(ctx context.Context, evt event.Event, item queue.Item) error {
			if evt.Name == "inngest/function.invoked" {
				eventCaptured = true
			}
			return nil
		}),
		executor.WithDriverV1(mockDriver),
	)
	require.NoError(t, err)

	now := time.Now()
	evtID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	run, err := exec.Schedule(ctx, execution.ScheduleRequest{
		Function:    fn,
		At:          &now,
		AccountID:   aID,
		WorkspaceID: wsID,
		AppID:       appID,
		Events: []event.TrackedEvent{
			event.NewBaseTrackedEventWithID(event.Event{
				Name: "test/event",
			}, evtID),
		},
	})
	require.NoError(t, err)

	pauseID := inngest.DeterministicSha1UUID(run.ID.RunID.String() + genID)

	pause := state.Pause{
		ID:          pauseID,
		WorkspaceID: wsID,
		Identifier: state.PauseIdentifier{
			RunID:      run.ID.RunID,
			FunctionID: fnID,
			AccountID:  aID,
		},
		GroupID:  "test-group",
		Incoming: "$trigger",
		Expires:  state.Time(time.Now().Add(time.Hour)),
	}

	_, err = pm.Write(ctx, pauses.Index{WorkspaceID: wsID, EventName: "test/event"}, &pause)
	require.NoError(t, err, "First pause write should succeed")

	_, err = exec.Execute(ctx, state.Identifier{
		WorkflowID: fnID,
		RunID:      run.ID.RunID,
		AccountID:  aID,
	}, queue.Item{
		WorkspaceID: wsID,
		Kind:        queue.KindStart,
		Identifier: state.Identifier{
			WorkflowID: fnID,
			RunID:      run.ID.RunID,
			AccountID:  aID,
		},
		Payload: queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: "invoke-step"}},
	}, inngest.Edge{
		Incoming: "$trigger",
		Outgoing: "invoke-step",
	})

	require.NoError(t, err)
	require.True(t, eventCaptured)
}

func TestExecutorReturnsResponseWhenNonRetriableError(t *testing.T) {
	ctx := context.Background()

	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{Persist: false})
	require.NoError(t, err)

	dbDriver := "sqlite"
	dbcqrs := base_cqrs.NewCQRS(db, dbDriver, sqlc_postgres.NewNormalizedOpts{})
	loader := dbcqrs.(state.FunctionLoader)

	fnID, wsID, appID, aID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ID:              fnID,
		FunctionVersion: 1,
		Name:            "test-fn",
		Slug:            "test-fn",
		Steps: []inngest.Step{
			{
				ID:   "step",
				Name: "step",
				URI:  "/step",
			},
		},
	}

	config, err := json.Marshal(fn)
	require.NoError(t, err)

	_, err = dbcqrs.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:   appID,
		Name: "test-app",
	})
	require.NoError(t, err)

	_, err = dbcqrs.InsertFunction(ctx, cqrs.InsertFunctionParams{
		ID:     fnID,
		AppID:  appID,
		Name:   fn.Name,
		Slug:   fn.Slug,
		Config: string(config),
	})
	require.NoError(t, err)

	_, shardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer shardedRc.Close()

	_, unshardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer unshardedRc.Close()

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
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	queueOpts := []redis_state.QueueOpt{
		redis_state.WithIdempotencyTTL(time.Hour),
		redis_state.WithShardSelector(shardSelector),
		redis_state.WithQueueShardClients(queueShards),
	}

	rq := redis_state.NewQueue(queueShard, queueOpts...)

	nonRetriableDriver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 400,
			Err:        func() *string { s := "NonRetriableError"; return &s }(),
			NoRetry:    true,
			Header: map[string][]string{
				"Content-Type":       {"application/json; charset=utf-8"},
				"X-Inngest-No-Retry": {"true"},
			},
		},
	}

	exec, err := executor.NewExecutor(
		executor.WithStateManager(smv2),
		executor.WithPauseManager(pauses.NewRedisOnlyManager(sm)),
		executor.WithQueue(rq),
		executor.WithLogger(logger.StdlibLogger(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithAssignedQueueShard(queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond)),
		executor.WithDriverV1(nonRetriableDriver),
	)
	require.NoError(t, err)

	now := time.Now()
	evtID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	run, err := exec.Schedule(ctx, execution.ScheduleRequest{
		Function:    fn,
		At:          &now,
		AccountID:   aID,
		WorkspaceID: wsID,
		AppID:       appID,
		Events: []event.TrackedEvent{
			event.NewBaseTrackedEventWithID(event.Event{
				Name: "test/event",
			}, evtID),
		},
	})
	require.NoError(t, err)

	// Job should have been scheduled
	jobsAfterSchedule, err := rq.RunJobs(
		ctx,
		queueShard.Name,
		run.ID.Tenant.EnvID,
		run.ID.FunctionID,
		run.ID.RunID,
		1000,
		0,
	)
	require.NoError(t, err)
	require.NotEmpty(t, jobsAfterSchedule)

	stateBefore, err := smv2.LoadState(ctx, run.ID)
	require.NoError(t, err)
	require.NotNil(t, stateBefore)

	// Add the job ID to the queue context. This is to avoid that the
	// finalize call remove the current queue item that it's currently
	// executing.
	jobCtx := queue.WithJobID(ctx, jobsAfterSchedule[0].JobID)

	resp, err := exec.Execute(jobCtx, state.Identifier{
		WorkflowID: fnID,
		RunID:      run.ID.RunID,
		AccountID:  aID,
	}, queue.Item{
		WorkspaceID: wsID,
		Kind:        queue.KindStart,
		Identifier: state.Identifier{
			WorkflowID: fnID,
			RunID:      run.ID.RunID,
			AccountID:  aID,
		},
		Payload: queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: "step"}},
	}, inngest.Edge{
		Incoming: "$trigger",
		Outgoing: "step",
	})

	require.Contains(t, err.Error(), "NonRetriableError")
	// Verifies Execute returns a non-nil response for non-retriable errors, allowing callers to check retryability
	require.NotNil(t, resp)
	require.False(t, resp.Retryable())

	// State should have been deleted when finalizing the run
	_, err = smv2.LoadState(ctx, run.ID)
	require.ErrorContains(t, err, state.ErrRunNotFound.Error())
}

func TestExecutorScheduleRateLimit(t *testing.T) {
	ctx := context.Background()

	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{Persist: false})
	require.NoError(t, err)

	dbDriver := "sqlite"
	dbcqrs := base_cqrs.NewCQRS(db, dbDriver, sqlc_postgres.NewNormalizedOpts{})
	loader := dbcqrs.(state.FunctionLoader)

	fnID, wsID, appID, aID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	rateLimitKey := "event.data.userID"
	rateLimit := &inngest.RateLimit{
		Limit:  1,
		Period: "24h",
		Key:    &rateLimitKey,
	}

	fn := inngest.Function{
		ID:              fnID,
		FunctionVersion: 1,
		Name:            "test-fn",
		Slug:            "test-fn",
		RateLimit:       rateLimit,
		Steps: []inngest.Step{
			{
				ID:   "step",
				Name: "step",
				URI:  "/step",
			},
		},
	}

	config, err := json.Marshal(fn)
	require.NoError(t, err)

	_, err = dbcqrs.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:   appID,
		Name: "test-app",
	})
	require.NoError(t, err)

	_, err = dbcqrs.InsertFunction(ctx, cqrs.InsertFunctionParams{
		ID:     fnID,
		AppID:  appID,
		Name:   fn.Name,
		Slug:   fn.Slug,
		Config: string(config),
	})
	require.NoError(t, err)

	_, shardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer shardedRc.Close()

	_, unshardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer unshardedRc.Close()

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
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	queueOpts := []redis_state.QueueOpt{
		redis_state.WithIdempotencyTTL(time.Hour),
		redis_state.WithShardSelector(shardSelector),
		redis_state.WithQueueShardClients(queueShards),
	}

	rq := redis_state.NewQueue(queueShard, queueOpts...)

	rl := ratelimit.New(ctx, unshardedClient.Global().Client(), "{ratelimit}:")

	exec, err := executor.NewExecutor(
		executor.WithStateManager(smv2),
		executor.WithPauseManager(pauses.NewRedisOnlyManager(sm)),
		executor.WithQueue(rq),
		executor.WithLogger(logger.StdlibLogger(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithAssignedQueueShard(queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond)),
		executor.WithRateLimiter(rl),
	)
	require.NoError(t, err)

	//
	// First event: Success
	//

	now := time.Now()
	evtID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	evt := event.NewBaseTrackedEventWithID(event.Event{
		Name: "test/event",
		Data: map[string]any{
			"userID": "inngest",
		},
	}, evtID)

	run, err := exec.Schedule(ctx, execution.ScheduleRequest{
		Function:    fn,
		At:          &now,
		AccountID:   aID,
		WorkspaceID: wsID,
		AppID:       appID,
		Events: []event.TrackedEvent{
			evt,
		},
	})
	require.NoError(t, err)

	// Job should have been scheduled
	jobsAfterSchedule, err := rq.RunJobs(
		ctx,
		queueShard.Name,
		run.ID.Tenant.EnvID,
		run.ID.FunctionID,
		run.ID.RunID,
		1000,
		0,
	)
	require.NoError(t, err)
	require.NotEmpty(t, jobsAfterSchedule)

	stateBefore, err := smv2.LoadState(ctx, run.ID)
	require.NoError(t, err)
	require.NotNil(t, stateBefore)

	//
	// Second event: Rate limited
	//

	now = time.Now()
	evtID = ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	evt2 := event.NewBaseTrackedEventWithID(event.Event{
		Name: "test/event",
		Data: map[string]any{
			"userID": "inngest",
		},
	}, evtID)

	_, err = exec.Schedule(ctx, execution.ScheduleRequest{
		Function:    fn,
		At:          &now,
		AccountID:   aID,
		WorkspaceID: wsID,
		AppID:       appID,
		Events: []event.TrackedEvent{
			evt2,
		},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, executor.ErrFunctionRateLimited)
}

type fakeLimitLifecycle struct {
	execution.NoopLifecyceListener

	skippedCount      int64
	limitReachedCount int64
}

func (fll *fakeLimitLifecycle) OnFunctionSkipped(
	context.Context,
	statev2.Metadata,
	execution.SkipState,
) {
	atomic.AddInt64(&fll.skippedCount, 1)
}

// OnFunctionBacklogSizeLimitReached is called when a function backlog size limit is hit
func (fll *fakeLimitLifecycle) OnFunctionBacklogSizeLimitReached(context.Context, statev2.ID) {
	atomic.AddInt64(&fll.limitReachedCount, 1)
}

func TestExecutorScheduleBacklogSizeLimit(t *testing.T) {
	ctx := context.Background()

	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{Persist: false})
	require.NoError(t, err)

	dbDriver := "sqlite"
	dbcqrs := base_cqrs.NewCQRS(db, dbDriver, sqlc_postgres.NewNormalizedOpts{})
	loader := dbcqrs.(state.FunctionLoader)

	fnID, wsID, appID, aID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ID:              fnID,
		FunctionVersion: 1,
		Name:            "test-fn",
		Slug:            "test-fn",
		Steps: []inngest.Step{
			{
				ID:   "step",
				Name: "step",
				URI:  "/step",
			},
		},
	}

	config, err := json.Marshal(fn)
	require.NoError(t, err)

	_, err = dbcqrs.UpsertApp(ctx, cqrs.UpsertAppParams{
		ID:   appID,
		Name: "test-app",
	})
	require.NoError(t, err)

	_, err = dbcqrs.InsertFunction(ctx, cqrs.InsertFunctionParams{
		ID:     fnID,
		AppID:  appID,
		Name:   fn.Name,
		Slug:   fn.Slug,
		Config: string(config),
	})
	require.NoError(t, err)

	_, shardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer shardedRc.Close()

	_, unshardedRc, err := createInmemoryRedis(t)
	require.NoError(t, err)
	defer unshardedRc.Close()

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
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	queueOpts := []redis_state.QueueOpt{
		redis_state.WithIdempotencyTTL(time.Hour),
		redis_state.WithShardSelector(shardSelector),
		redis_state.WithQueueShardClients(queueShards),
	}

	rq := redis_state.NewQueue(queueShard, queueOpts...)

	fll := &fakeLimitLifecycle{}

	exec, err := executor.NewExecutor(
		executor.WithStateManager(smv2),
		executor.WithPauseManager(pauses.NewRedisOnlyManager(sm)),
		executor.WithQueue(rq),
		executor.WithLogger(logger.StdlibLogger(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithAssignedQueueShard(queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond)),

		executor.WithLifecycleListeners(fll),
		executor.WithFunctionBacklogSizeLimit(func(ctx context.Context, accountID, envID, fnID uuid.UUID) executor.BacklogSizeLimit {
			return executor.BacklogSizeLimit{
				Limit:   1,
				Enforce: true,
			}
		}),
	)
	require.NoError(t, err)

	//
	// First event: Success
	//

	now := time.Now()
	evtID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	evt := event.NewBaseTrackedEventWithID(event.Event{
		Name: "test/event",
		Data: map[string]any{
			"userID": "inngest",
		},
	}, evtID)

	run, err := exec.Schedule(ctx, execution.ScheduleRequest{
		Function:    fn,
		At:          &now,
		AccountID:   aID,
		WorkspaceID: wsID,
		AppID:       appID,
		Events: []event.TrackedEvent{
			evt,
		},
	})
	require.NoError(t, err)

	// Job should have been scheduled
	jobsAfterSchedule, err := rq.RunJobs(
		ctx,
		queueShard.Name,
		run.ID.Tenant.EnvID,
		run.ID.FunctionID,
		run.ID.RunID,
		1000,
		0,
	)
	require.NoError(t, err)
	require.NotEmpty(t, jobsAfterSchedule)

	stateBefore, err := smv2.LoadState(ctx, run.ID)
	require.NoError(t, err)
	require.NotNil(t, stateBefore)

	require.Equal(t, 0, int(fll.limitReachedCount))
	require.Equal(t, 0, int(fll.skippedCount))

	//
	// Second event: Backlog size limit exceeded
	//

	now = time.Now()
	evtID = ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	evt2 := event.NewBaseTrackedEventWithID(event.Event{
		Name: "test/event",
		Data: map[string]any{
			"userID": "inngest",
		},
	}, evtID)

	_, err = exec.Schedule(ctx, execution.ScheduleRequest{
		Function:    fn,
		At:          &now,
		AccountID:   aID,
		WorkspaceID: wsID,
		AppID:       appID,
		Events: []event.TrackedEvent{
			evt2,
		},
	})
	require.ErrorIs(t, err, executor.ErrFunctionSkipped)

	service.Wait()

	require.Equal(t, 1, int(atomic.LoadInt64(&fll.limitReachedCount)))
	require.Equal(t, 1, int(atomic.LoadInt64(&fll.skippedCount)))
}
