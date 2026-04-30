package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/cqrs/base_cqrs"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/inngest/inngest/pkg/enums"
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
	"github.com/stretchr/testify/require"
)

func TestDeferAddSavesDefer(t *testing.T) {
	ctx := logger.WithStdlib(context.Background(), logger.VoidLogger())

	db, err := base_cqrs.New(ctx, base_cqrs.BaseCQRSOptions{Persist: false})
	require.NoError(t, err)
	adapter := dbsqlite.New(db)
	dbcqrs := base_cqrs.NewCQRS(adapter)
	loader := dbcqrs.(state.FunctionLoader)

	fnID, wsID, appID, aID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ID:              fnID,
		FunctionVersion: 1,
		Name:            "test-fn",
		Slug:            "test-fn",
		Steps: []inngest.Step{
			{
				ID:   "step-defer",
				Name: "step-defer",
				URI:  "/step-defer",
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

	pauseMgr := pauses.NewPauseStoreManager(unshardedClient)

	var sm state.Manager
	sm, err = redis_state.New(ctx,
		redis_state.WithShardedClient(shardedClient),
		redis_state.WithPauseDeleter(pauseMgr),
	)
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	queueOpts := []queue.QueueOpt{queue.WithIdempotencyTTL(time.Hour)}
	queueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), queueOpts...)
	shardSelector := func(ctx context.Context, _ uuid.UUID, _ *string) (queue.QueueShard, error) {
		return queueShard, nil
	}

	rq, err := queue.New(
		ctx,
		"test-queue",
		queueShard,
		map[string]queue.QueueShard{queueShard.Name(): queueShard},
		shardSelector,
		queueOpts...,
	)
	require.NoError(t, err)

	stepID := "step-defer"
	mockDriver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{{
				Op: enums.OpcodeDeferAdd,
				ID: stepID,
				Opts: map[string]any{
					"fn_slug": "onDefer-score",
					"input":   map[string]any{"user_id": "u_123"},
				},
			}},
		},
	}

	exec, err := executor.NewExecutor(
		executor.WithStateManager(smv2),
		executor.WithPauseManager(pauseMgr),
		executor.WithQueue(rq),
		executor.WithLogger(logger.StdlibLogger(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithAssignedQueueShard(queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond)),
		executor.WithDriverV1(mockDriver),
	)
	require.NoError(t, err)

	now := time.Now()
	evtID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	_, run, err := exec.Schedule(ctx, execution.ScheduleRequest{
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
		Payload: queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
	}, inngest.Edge{
		Incoming: "$trigger",
		Outgoing: stepID,
	})
	require.NoError(t, err)

	steps, err := smv2.LoadSteps(ctx, run.ID)
	require.NoError(t, err)
	require.Contains(t, steps, stepID, "step.defer should be memoized")
	require.Equal(t, json.RawMessage("null"), steps[stepID], "memoized data should be null")

	defers, err := smv2.LoadDefers(ctx, run.ID)
	require.NoError(t, err)
	require.Len(t, defers, 1, "expected exactly one defer")

	d := defers[stepID]
	require.Equal(t, "onDefer-score", d.FnSlug)
	require.Equal(t, statev2.ScheduleStatusAfterRun, d.ScheduleStatus)
	require.JSONEq(t, `{"user_id":"u_123"}`, string(d.Input))
}

// TestFinalizeEmitsDeferredStartEvents verifies that when a run with defers
// is finalized, the executor emits an inngest/deferred.start event for each
// defer whose ScheduleStatus is AfterRun.
func TestFinalizeEmitsDeferredStartEvents(t *testing.T) {
	ctx := logger.WithStdlib(context.Background(), logger.VoidLogger())

	db, err := base_cqrs.New(ctx, base_cqrs.BaseCQRSOptions{Persist: false})
	require.NoError(t, err)
	adapter := dbsqlite.New(db)
	dbcqrs := base_cqrs.NewCQRS(adapter)
	loader := dbcqrs.(state.FunctionLoader)

	fnID, wsID, appID, aID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	fn := inngest.Function{
		ID:              fnID,
		FunctionVersion: 1,
		Name:            "test-fn",
		Slug:            "test-fn",
		Steps: []inngest.Step{
			{ID: "step-1", Name: "step-1", URI: "/step-1"},
		},
	}

	config, err := json.Marshal(fn)
	require.NoError(t, err)

	_, err = dbcqrs.UpsertApp(ctx, cqrs.UpsertAppParams{ID: appID, Name: "test-app"})
	require.NoError(t, err)

	_, err = dbcqrs.InsertFunction(ctx, cqrs.InsertFunctionParams{
		ID: fnID, AppID: appID, Name: fn.Name, Slug: fn.Slug, Config: string(config),
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

	pauseMgr := pauses.NewPauseStoreManager(unshardedClient)

	var sm state.Manager
	sm, err = redis_state.New(ctx,
		redis_state.WithShardedClient(shardedClient),
		redis_state.WithPauseDeleter(pauseMgr),
	)
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	queueOpts := []queue.QueueOpt{queue.WithIdempotencyTTL(time.Hour)}
	queueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), queueOpts...)
	shardSelector := func(ctx context.Context, _ uuid.UUID, _ *string) (queue.QueueShard, error) {
		return queueShard, nil
	}

	rq, err := queue.New(
		ctx, "test-queue", queueShard,
		map[string]queue.QueueShard{queueShard.Name(): queueShard},
		shardSelector, queueOpts...,
	)
	require.NoError(t, err)

	exec, err := executor.NewExecutor(
		executor.WithStateManager(smv2),
		executor.WithPauseManager(pauseMgr),
		executor.WithQueue(rq),
		executor.WithLogger(logger.StdlibLogger(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithAssignedQueueShard(queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond)),
	)
	require.NoError(t, err)

	var capturedEvents []event.Event
	exec.SetFinalizer(func(ctx context.Context, id statev2.ID, events []event.Event) error {
		capturedEvents = append(capturedEvents, events...)
		return nil
	})

	now := time.Now()
	evtID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	_, run, err := exec.Schedule(ctx, execution.ScheduleRequest{
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

	// Nested input verifies the event carries structured JSON rather than a stringified payload.
	nestedInputJSON := `{"user":{"id":"u_123","meta":{"role":"admin","tags":["a","b"]}},"score":0.87}`
	activeDefer := statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       "hash-active",
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
		Input:          json.RawMessage(nestedInputJSON),
	}
	cancelledDefer := statev2.Defer{
		FnSlug:         "onDefer-cleanup",
		HashedID:       "hash-cancelled",
		ScheduleStatus: statev2.ScheduleStatusCancelled,
		Input:          json.RawMessage(`{}`),
	}

	require.NoError(t, smv2.SaveDefer(ctx, run.ID, activeDefer))
	require.NoError(t, smv2.SaveDefer(ctx, run.ID, cancelledDefer))

	err = exec.Finalize(ctx, execution.FinalizeOpts{
		Metadata: *run,
		Response: execution.FinalizeResponse{
			Type:        execution.FinalizeResponseRunComplete,
			RunComplete: state.GeneratorOpcode{Op: enums.OpcodeRunComplete},
		},
		Optional: execution.FinalizeOptional{
			FnSlug: fn.Slug,
		},
	})
	require.NoError(t, err)

	var deferredEvents []event.Event
	for _, evt := range capturedEvents {
		if evt.Name == "inngest/deferred.start" {
			deferredEvents = append(deferredEvents, evt)
		}
	}

	// Only the active defer should produce an event; the cancelled one should not.
	require.Len(t, deferredEvents, 1, "expected exactly one inngest/deferred.start event")

	evt := deferredEvents[0]
	evtData, err := json.Marshal(evt.Data)
	require.NoError(t, err)

	var data map[string]any
	require.NoError(t, json.Unmarshal(evtData, &data))

	inngestData := data["_inngest"].(map[string]any)
	require.Equal(t, "onDefer-score", inngestData["fn_slug"])
	require.Equal(t, fn.Slug, inngestData["parent_fn_slug"])
	require.Equal(t, run.ID.RunID.String(), inngestData["parent_run_id"])

	user, ok := data["user"].(map[string]any)
	require.True(t, ok, "data.user should be a JSON object, got %T", data["user"])
	require.Equal(t, "u_123", user["id"])
	meta, ok := user["meta"].(map[string]any)
	require.True(t, ok, "data.user.meta should be a JSON object, got %T", user["meta"])
	require.Equal(t, "admin", meta["role"])
	require.Equal(t, []any{"a", "b"}, meta["tags"])
	require.Equal(t, 0.87, data["score"])
}

// TestDeferCancelUpdatesDeferStatus verifies that when the executor processes
// an OpcodeDeferCancel, it flips the existing defer's ScheduleStatus to Cancelled.
func TestDeferCancelUpdatesDeferStatus(t *testing.T) {
	ctx := logger.WithStdlib(context.Background(), logger.VoidLogger())

	db, err := base_cqrs.New(ctx, base_cqrs.BaseCQRSOptions{Persist: false})
	require.NoError(t, err)
	adapter := dbsqlite.New(db)
	dbcqrs := base_cqrs.NewCQRS(adapter)
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
	_, err = dbcqrs.InsertFunction(ctx, cqrs.InsertFunctionParams{
		ID: fnID, AppID: appID, Name: fn.Name, Slug: fn.Slug, Config: string(config),
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

	pauseMgr := pauses.NewPauseStoreManager(unshardedClient)

	var sm state.Manager
	sm, err = redis_state.New(ctx,
		redis_state.WithShardedClient(shardedClient),
		redis_state.WithPauseDeleter(pauseMgr),
	)
	require.NoError(t, err)
	smv2 := redis_state.MustRunServiceV2(sm)

	queueOpts := []queue.QueueOpt{queue.WithIdempotencyTTL(time.Hour)}
	queueShard := redis_state.NewQueueShard(consts.DefaultQueueShardName, unshardedClient.Queue(), queueOpts...)
	shardSelector := func(ctx context.Context, _ uuid.UUID, _ *string) (queue.QueueShard, error) {
		return queueShard, nil
	}

	rq, err := queue.New(
		ctx, "test-queue", queueShard,
		map[string]queue.QueueShard{queueShard.Name(): queueShard},
		shardSelector, queueOpts...,
	)
	require.NoError(t, err)

	// The DeferAdd step and the DeferCancel step have DIFFERENT hashed IDs.
	// DeferCancel identifies the target defer by target_hashed_id, not by
	// the cancel step's own gen.ID.
	deferStepID := "step-defer"
	cancelStepID := "step-cancel"

	mockDriver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{{
				Op: enums.OpcodeDeferCancel,
				ID: cancelStepID,
				Opts: map[string]any{
					"target_hashed_id": deferStepID,
				},
			}},
		},
	}

	exec, err := executor.NewExecutor(
		executor.WithStateManager(smv2),
		executor.WithPauseManager(pauseMgr),
		executor.WithQueue(rq),
		executor.WithLogger(logger.StdlibLogger(ctx)),
		executor.WithFunctionLoader(loader),
		executor.WithAssignedQueueShard(queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond)),
		executor.WithDriverV1(mockDriver),
	)
	require.NoError(t, err)

	now := time.Now()
	evtID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	_, run, err := exec.Schedule(ctx, execution.ScheduleRequest{
		Function: fn, At: &now, AccountID: aID, WorkspaceID: wsID, AppID: appID,
		Events: []event.TrackedEvent{
			event.NewBaseTrackedEventWithID(event.Event{Name: "test/event"}, evtID),
		},
	})
	require.NoError(t, err)

	// Pre-seed a defer as if DeferAdd had already run.
	require.NoError(t, smv2.SaveDefer(ctx, run.ID, statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       deferStepID,
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
		Input:          json.RawMessage(`{"user_id":"u_123"}`),
	}))

	_, err = exec.Execute(ctx, state.Identifier{
		WorkflowID: fnID, RunID: run.ID.RunID, AccountID: aID,
	}, queue.Item{
		WorkspaceID: wsID,
		Kind:        queue.KindStart,
		Identifier:  state.Identifier{WorkflowID: fnID, RunID: run.ID.RunID, AccountID: aID},
		Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: cancelStepID}},
	}, inngest.Edge{Incoming: "$trigger", Outgoing: cancelStepID})
	require.NoError(t, err)

	defers, err := smv2.LoadDefers(ctx, run.ID)
	require.NoError(t, err)
	require.Len(t, defers, 1)

	d := defers[deferStepID]
	require.Equal(t, "onDefer-score", d.FnSlug, "FnSlug should be preserved")
	require.Equal(t, statev2.ScheduleStatusCancelled, d.ScheduleStatus, "status should be Cancelled")
	require.JSONEq(t, `{"user_id":"u_123"}`, string(d.Input), "Input should be preserved")
}

// deferTestInfra holds the shared state manager, queue, and loader used by the
// checkpoint-vs-executor consistency tests so each test can spin up 3 runs
// against the same backing store.
type deferTestInfra struct {
	ctx        context.Context
	fn         inngest.Function
	fnID       uuid.UUID
	wsID       uuid.UUID
	appID      uuid.UUID
	aID        uuid.UUID
	smv2       statev2.RunService
	pauseMgr   pauses.Manager
	loader     state.FunctionLoader
	dbcqrs     cqrs.Manager
	queueShard redis_state.RedisQueueShard
	rq         queue.Queue
}

func newDeferTestInfra(t *testing.T) *deferTestInfra {
	t.Helper()
	ctx := logger.WithStdlib(context.Background(), logger.VoidLogger())

	db, err := base_cqrs.New(ctx, base_cqrs.BaseCQRSOptions{Persist: false})
	require.NoError(t, err)
	adapter := dbsqlite.New(db)
	dbcqrs := base_cqrs.NewCQRS(adapter)
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
	_, err = dbcqrs.InsertFunction(ctx, cqrs.InsertFunctionParams{
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
	shardSelector := func(ctx context.Context, _ uuid.UUID, _ *string) (queue.QueueShard, error) {
		return queueShard, nil
	}

	rq, err := queue.New(
		ctx, "test-queue", queueShard,
		map[string]queue.QueueShard{queueShard.Name(): queueShard},
		shardSelector, queueOpts...,
	)
	require.NoError(t, err)

	return &deferTestInfra{
		ctx:        ctx,
		fn:         fn,
		fnID:       fnID,
		wsID:       wsID,
		appID:      appID,
		aID:        aID,
		smv2:       smv2,
		pauseMgr:   pauseMgr,
		loader:     loader,
		dbcqrs:     dbcqrs,
		queueShard: queueShard,
		rq:         rq,
	}
}

// newExecutor builds an executor wired to the shared infra. Pass a non-nil
// driver to drive Execute() calls; pass nil when only the checkpointer will
// be used.
func (i *deferTestInfra) newExecutor(t *testing.T, driver *mockDriverV1) execution.Executor {
	t.Helper()
	shardSelector := func(ctx context.Context, _ uuid.UUID, _ *string) (queue.QueueShard, error) {
		return i.queueShard, nil
	}

	opts := []executor.ExecutorOpt{
		executor.WithStateManager(i.smv2),
		executor.WithPauseManager(i.pauseMgr),
		executor.WithQueue(i.rq),
		executor.WithLogger(logger.StdlibLogger(i.ctx)),
		executor.WithFunctionLoader(i.loader),
		executor.WithAssignedQueueShard(i.queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond)),
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
		TracerProvider: tracing.NewOtelTracerProvider(nil, time.Millisecond),
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

// TestDeferAdd_ExecutorAndCheckpointProduceSameDefer drives an OpcodeDeferAdd
// through executor.Execute, CheckpointSyncSteps, and CheckpointAsyncSteps. It
// asserts each path produces the same expected Defer record.
//
// We added this test because we had a regression where Defer worked in
// non-checkpointing codepaths but not in checkpointing.
func TestDeferAdd_ExecutorAndCheckpointProduceSameDefer(t *testing.T) {
	infra := newDeferTestInfra(t)
	ctx := infra.ctx

	op := state.GeneratorOpcode{
		ID: "step-defer",
		Op: enums.OpcodeDeferAdd,
		Opts: map[string]any{
			"fn_slug": "onDefer-score",
			"input":   map[string]any{"user_id": "u_123"},
		},
	}
	expected := statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       op.ID,
		Input:          json.RawMessage(`{"user_id":"u_123"}`),
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
	}

	cases := []struct {
		name string
		run  func(t *testing.T) statev2.ID
	}{
		{
			name: "executor",
			run: func(t *testing.T) statev2.ID {
				driver := &mockDriverV1{
					response: &state.DriverResponse{StatusCode: 206, Generator: []*state.GeneratorOpcode{&op}},
					t:        t,
				}
				exec := infra.newExecutor(t, driver)
				run := infra.scheduleRun(t, exec)
				_, err := exec.Execute(ctx, state.Identifier{
					WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
				}, queue.Item{
					Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
					Kind:        queue.KindStart,
					Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: op.ID}},
					WorkspaceID: infra.wsID,
				}, inngest.Edge{Incoming: "$trigger", Outgoing: op.ID})
				require.NoError(t, err)
				return run.ID
			},
		},
		{
			name: "sync-checkpoint",
			run: func(t *testing.T) statev2.ID {
				exec := infra.newExecutor(t, nil)
				run := infra.scheduleRun(t, exec)
				cp := infra.newCheckpointer(t, exec)
				err := cp.CheckpointSyncSteps(ctx, checkpoint.SyncCheckpoint{
					AccountID: infra.aID,
					AppID:     infra.appID,
					EnvID:     infra.wsID,
					FnID:      infra.fnID,
					Metadata:  run,
					RunID:     run.ID.RunID,
					Steps:     []state.GeneratorOpcode{op},
				})
				require.NoError(t, err)
				return run.ID
			},
		},
		{
			name: "async-checkpoint",
			run: func(t *testing.T) statev2.ID {
				exec := infra.newExecutor(t, nil)
				run := infra.scheduleRun(t, exec)
				cp := infra.newCheckpointer(t, exec)
				// No QueueItemRef → async path skips the ResetAttemptsByJobID call.
				err := cp.CheckpointAsyncSteps(ctx, checkpoint.AsyncCheckpoint{
					AccountID: infra.aID,
					EnvID:     infra.wsID,
					FnID:      infra.fnID,
					RunID:     run.ID.RunID,
					Steps:     []state.GeneratorOpcode{op},
				})
				require.NoError(t, err)
				return run.ID
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := require.New(t)
			runID := c.run(t)

			defers, err := infra.smv2.LoadDefers(ctx, runID)
			r.NoError(err)
			r.Len(defers, 1)
			r.Equal(expected, defers[op.ID])

			steps, err := infra.smv2.LoadSteps(ctx, runID)
			r.NoError(err)
			r.Equal(json.RawMessage("null"), steps[op.ID],
				"DeferAdd should memoize the step with a null payload")
		})
	}
}

// TestDeferCancel_ExecutorAndCheckpointProduceSameDefer exercises DeferCancel
// via all three paths (executor, sync checkpoint, async checkpoint) against
// runs pre-seeded with a matching defer. Each path must flip ScheduleStatus
// to Cancelled while preserving every other field.
//
// We added this test because we had a regression where DeferCancel worked in
// non-checkpointing codepaths but not in checkpointing.
func TestDeferCancel_ExecutorAndCheckpointProduceSameDefer(t *testing.T) {
	infra := newDeferTestInfra(t)
	ctx := infra.ctx

	const (
		deferStepID  = "step-defer"
		cancelStepID = "step-cancel"
	)
	seed := statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       deferStepID,
		Input:          json.RawMessage(`{"user_id":"u_123"}`),
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
	}
	expected := seed
	expected.ScheduleStatus = statev2.ScheduleStatusCancelled

	cancelOp := state.GeneratorOpcode{
		ID: cancelStepID,
		Op: enums.OpcodeDeferCancel,
		Opts: map[string]any{
			"target_hashed_id": deferStepID,
		},
	}

	paths := []struct {
		name string
		run  func(t *testing.T) statev2.ID
	}{
		{
			name: "executor",
			run: func(t *testing.T) statev2.ID {
				driver := &mockDriverV1{
					response: &state.DriverResponse{StatusCode: 206, Generator: []*state.GeneratorOpcode{&cancelOp}},
					t:        t,
				}
				exec := infra.newExecutor(t, driver)
				run := infra.scheduleRun(t, exec)
				require.NoError(t, infra.smv2.SaveDefer(ctx, run.ID, seed))
				_, err := exec.Execute(ctx, state.Identifier{
					WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
				}, queue.Item{
					Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
					Kind:        queue.KindStart,
					Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: cancelStepID}},
					WorkspaceID: infra.wsID,
				}, inngest.Edge{Incoming: "$trigger", Outgoing: cancelStepID})
				require.NoError(t, err)
				return run.ID
			},
		},
		{
			name: "sync-checkpoint",
			run: func(t *testing.T) statev2.ID {
				exec := infra.newExecutor(t, nil)
				run := infra.scheduleRun(t, exec)
				require.NoError(t, infra.smv2.SaveDefer(ctx, run.ID, seed))
				cp := infra.newCheckpointer(t, exec)
				err := cp.CheckpointSyncSteps(ctx, checkpoint.SyncCheckpoint{
					AccountID: infra.aID,
					AppID:     infra.appID,
					EnvID:     infra.wsID,
					FnID:      infra.fnID,
					Metadata:  run,
					RunID:     run.ID.RunID,
					Steps:     []state.GeneratorOpcode{cancelOp},
				})
				require.NoError(t, err)
				return run.ID
			},
		},
		{
			name: "async-checkpoint",
			run: func(t *testing.T) statev2.ID {
				exec := infra.newExecutor(t, nil)
				run := infra.scheduleRun(t, exec)
				require.NoError(t, infra.smv2.SaveDefer(ctx, run.ID, seed))
				cp := infra.newCheckpointer(t, exec)
				err := cp.CheckpointAsyncSteps(ctx, checkpoint.AsyncCheckpoint{
					AccountID: infra.aID,
					EnvID:     infra.wsID,
					FnID:      infra.fnID,
					RunID:     run.ID.RunID,
					Steps:     []state.GeneratorOpcode{cancelOp},
				})
				require.NoError(t, err)
				return run.ID
			},
		},
	}

	for _, p := range paths {
		t.Run(p.name, func(t *testing.T) {
			r := require.New(t)
			runID := p.run(t)

			defers, err := infra.smv2.LoadDefers(ctx, runID)
			r.NoError(err)
			r.Len(defers, 1)
			r.Equal(expected, defers[deferStepID])
		})
	}
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

// TestDeferAdd_WithRunCompleteSkipsDiscovery asserts that when DeferAdd is
// piggybacked onto RunComplete, DeferAdd does NOT enqueue a discovery step.
// Without this gating, the discovery would be orphaned because RunComplete
// finalizes (deletes state) immediately after.
func TestDeferAdd_WithRunCompleteSkipsDiscovery(t *testing.T) {
	infra := newDeferTestInfra(t)

	countingQ := &enqueueCountingQueue{Queue: infra.rq}

	stepID := "step-defer"
	driver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{
				{
					Op: enums.OpcodeDeferAdd,
					ID: stepID,
					Opts: map[string]any{
						"fn_slug": "onDefer-score",
						"input":   map[string]any{},
					},
				},
				{
					Op:   enums.OpcodeRunComplete,
					ID:   "run-complete",
					Data: json.RawMessage(`{"data": {"status_code": 200}}`),
				},
			},
		},
	}

	shardSelector := func(ctx context.Context, _ uuid.UUID, _ *string) (queue.QueueShard, error) {
		return infra.queueShard, nil
	}

	exec, err := executor.NewExecutor(
		executor.WithStateManager(infra.smv2),
		executor.WithPauseManager(infra.pauseMgr),
		executor.WithQueue(countingQ),
		executor.WithLogger(logger.StdlibLogger(infra.ctx)),
		executor.WithFunctionLoader(infra.loader),
		executor.WithAssignedQueueShard(infra.queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond)),
		executor.WithDriverV1(driver),
	)
	require.NoError(t, err)

	run := infra.scheduleRun(t, exec)
	countBeforeExecute := countingQ.enqueues

	_, err = exec.Execute(infra.ctx, state.Identifier{
		WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
	}, queue.Item{
		WorkspaceID: infra.wsID,
		Kind:        queue.KindStart,
		Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
		Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
	}, inngest.Edge{Incoming: "$trigger", Outgoing: stepID})
	require.NoError(t, err)

	enqueuesDuringExecute := countingQ.enqueues - countBeforeExecute
	require.Equal(t, 0, enqueuesDuringExecute,
		"DeferAdd should not enqueue discovery when piggybacked onto RunComplete; got %d enqueues", enqueuesDuringExecute)
}

// TestDeferAdd_BareOpEnqueuesDiscovery is the inverse of
// TestDeferAdd_WithRunCompleteSkipsDiscovery: a bare [DeferAdd] with no host
// op should fall through to enqueue a discovery step so the run can progress.
// "Shouldn't happen" in normal operation (the SDK piggybacks lazy ops), but
// the fallback path must stay safe.
func TestDeferAdd_BareOpEnqueuesDiscovery(t *testing.T) {
	infra := newDeferTestInfra(t)

	countingQ := &enqueueCountingQueue{Queue: infra.rq}

	stepID := "step-defer"
	driver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{
				{
					Op: enums.OpcodeDeferAdd,
					ID: stepID,
					Opts: map[string]any{
						"fn_slug": "onDefer-score",
						"input":   map[string]any{},
					},
				},
			},
		},
	}

	shardSelector := func(ctx context.Context, _ uuid.UUID, _ *string) (queue.QueueShard, error) {
		return infra.queueShard, nil
	}

	exec, err := executor.NewExecutor(
		executor.WithStateManager(infra.smv2),
		executor.WithPauseManager(infra.pauseMgr),
		executor.WithQueue(countingQ),
		executor.WithLogger(logger.StdlibLogger(infra.ctx)),
		executor.WithFunctionLoader(infra.loader),
		executor.WithAssignedQueueShard(infra.queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond)),
		executor.WithDriverV1(driver),
	)
	require.NoError(t, err)

	run := infra.scheduleRun(t, exec)
	countBeforeExecute := countingQ.enqueues

	_, err = exec.Execute(infra.ctx, state.Identifier{
		WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
	}, queue.Item{
		WorkspaceID: infra.wsID,
		Kind:        queue.KindStart,
		Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
		Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
	}, inngest.Edge{Incoming: "$trigger", Outgoing: stepID})
	require.NoError(t, err)

	enqueuesDuringExecute := countingQ.enqueues - countBeforeExecute
	require.Equal(t, 1, enqueuesDuringExecute,
		"bare DeferAdd should enqueue exactly one discovery step; got %d enqueues", enqueuesDuringExecute)

	defers, err := infra.smv2.LoadDefers(infra.ctx, run.ID)
	require.NoError(t, err)
	require.Contains(t, defers, stepID, "defer should be persisted even on bare-op path")
	require.Equal(t, statev2.ScheduleStatusAfterRun, defers[stepID].ScheduleStatus)
}

// TestDeferCancel_BareOpEnqueuesDiscovery is the DeferCancel counterpart: a
// bare [DeferCancel] with no host op should fall through to enqueue a discovery
// step. Pre-seeds the target defer so SetDeferStatus succeeds and the bare-op
// branch is actually reached (an error there would short-circuit before the
// lazyOpHasHost check).
func TestDeferCancel_BareOpEnqueuesDiscovery(t *testing.T) {
	infra := newDeferTestInfra(t)

	countingQ := &enqueueCountingQueue{Queue: infra.rq}

	const (
		deferStepID  = "step-defer"
		cancelStepID = "step-cancel"
	)

	driver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{
				{
					Op: enums.OpcodeDeferCancel,
					ID: cancelStepID,
					Opts: map[string]any{
						"target_hashed_id": deferStepID,
					},
				},
			},
		},
	}

	shardSelector := func(ctx context.Context, _ uuid.UUID, _ *string) (queue.QueueShard, error) {
		return infra.queueShard, nil
	}

	exec, err := executor.NewExecutor(
		executor.WithStateManager(infra.smv2),
		executor.WithPauseManager(infra.pauseMgr),
		executor.WithQueue(countingQ),
		executor.WithLogger(logger.StdlibLogger(infra.ctx)),
		executor.WithFunctionLoader(infra.loader),
		executor.WithAssignedQueueShard(infra.queueShard),
		executor.WithShardSelector(shardSelector),
		executor.WithTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond)),
		executor.WithDriverV1(driver),
	)
	require.NoError(t, err)

	run := infra.scheduleRun(t, exec)

	// Pre-seed the target defer; otherwise SetDeferStatus would error out
	// before the bare-op fallback branch is reached.
	require.NoError(t, infra.smv2.SaveDefer(infra.ctx, run.ID, statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       deferStepID,
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
		Input:          json.RawMessage(`{"user_id":"u_123"}`),
	}))

	countBeforeExecute := countingQ.enqueues

	_, err = exec.Execute(infra.ctx, state.Identifier{
		WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
	}, queue.Item{
		WorkspaceID: infra.wsID,
		Kind:        queue.KindStart,
		Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
		Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: cancelStepID}},
	}, inngest.Edge{Incoming: "$trigger", Outgoing: cancelStepID})
	require.NoError(t, err)

	enqueuesDuringExecute := countingQ.enqueues - countBeforeExecute
	require.Equal(t, 1, enqueuesDuringExecute,
		"bare DeferCancel should enqueue exactly one discovery step; got %d enqueues", enqueuesDuringExecute)

	defers, err := infra.smv2.LoadDefers(infra.ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, statev2.ScheduleStatusCancelled, defers[deferStepID].ScheduleStatus,
		"defer status should be Cancelled even on bare-op path")
}
