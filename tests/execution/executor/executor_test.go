package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
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
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
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

	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{InMemory: true})
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

				evt := event.NewOSSTrackedEvent(event.Event{
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

	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{InMemory: true})
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

				evt := event.NewOSSTrackedEvent(event.Event{
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

	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{InMemory: true})
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
			event.NewOSSTrackedEventWithID(event.Event{
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
			event.NewOSSTrackedEventWithID(event.Event{
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
