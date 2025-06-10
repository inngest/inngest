package executor

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
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
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
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

func TestScheduleRaceCondition(t *testing.T) {
	ctx := context.Background()
	_ = trace.UserTracer()
	work := make(chan *hookData)

	db, err := base_cqrs.New(base_cqrs.BaseCQRSOptions{InMemory: true})
	require.NoError(t, err)

	// Initialize the devserver
	dbDriver := "sqlite"
	dbcqrs := base_cqrs.NewCQRS(db, dbDriver)
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
	dbcqrs := base_cqrs.NewCQRS(db, dbDriver)
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
