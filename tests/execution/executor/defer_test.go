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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
// through executor.Execute, CheckpointSyncSteps, and CheckpointAsyncSteps and
// asserts the resulting Defer record is identical across all three paths.
func TestDeferAdd_ExecutorAndCheckpointProduceSameDefer(t *testing.T) {
	infra := newDeferTestInfra(t)
	ctx := infra.ctx

	op := state.GeneratorOpcode{
		Op: enums.OpcodeDeferAdd,
		ID: "step-defer",
		Opts: map[string]any{
			"fn_slug": "onDefer-score",
			"input":   map[string]any{"user_id": "u_123"},
		},
	}

	driver := &mockDriverV1{
		t:        t,
		response: &state.DriverResponse{StatusCode: 206, Generator: []*state.GeneratorOpcode{&op}},
	}
	execA := infra.newExecutor(t, driver)
	runA := infra.scheduleRun(t, execA)
	_, err := execA.Execute(ctx, state.Identifier{
		WorkflowID: infra.fnID, RunID: runA.ID.RunID, AccountID: infra.aID,
	}, queue.Item{
		WorkspaceID: infra.wsID,
		Kind:        queue.KindStart,
		Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: runA.ID.RunID, AccountID: infra.aID},
		Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: op.ID}},
	}, inngest.Edge{Incoming: "$trigger", Outgoing: op.ID})
	require.NoError(t, err)

	execB := infra.newExecutor(t, nil)
	runB := infra.scheduleRun(t, execB)
	cp := infra.newCheckpointer(t, execB)
	err = cp.CheckpointSyncSteps(ctx, checkpoint.SyncCheckpoint{
		RunID:     runB.ID.RunID,
		FnID:      infra.fnID,
		AppID:     infra.appID,
		AccountID: infra.aID,
		EnvID:     infra.wsID,
		Metadata:  runB,
		Steps:     []state.GeneratorOpcode{op},
	})
	require.NoError(t, err)

	execC := infra.newExecutor(t, nil)
	runC := infra.scheduleRun(t, execC)
	cp = infra.newCheckpointer(t, execC)
	err = cp.CheckpointAsyncSteps(ctx, checkpoint.AsyncCheckpoint{
		RunID:     runC.ID.RunID,
		FnID:      infra.fnID,
		AccountID: infra.aID,
		EnvID:     infra.wsID,
		Steps:     []state.GeneratorOpcode{op},
		// No QueueItemRef → async path skips the ResetAttemptsByJobID call.
	})
	require.NoError(t, err)

	defersA, err := infra.smv2.LoadDefers(ctx, runA.ID)
	require.NoError(t, err)
	defersB, err := infra.smv2.LoadDefers(ctx, runB.ID)
	require.NoError(t, err)
	defersC, err := infra.smv2.LoadDefers(ctx, runC.ID)
	require.NoError(t, err)

	require.Len(t, defersA, 1)
	require.Len(t, defersB, 1)
	require.Len(t, defersC, 1)

	// Every path should produce the same Defer record. The defers map is keyed
	// by the hashed step ID, which is identical across runs (`step-defer`).
	require.Equal(t, defersA[op.ID], defersB[op.ID],
		"executor path and sync-checkpoint path must produce identical Defer records")
	require.Equal(t, defersA[op.ID], defersC[op.ID],
		"executor path and async-checkpoint path must produce identical Defer records")

	// Step memoization should also be consistent (null payload in all paths).
	stepsA, err := infra.smv2.LoadSteps(ctx, runA.ID)
	require.NoError(t, err)
	stepsB, err := infra.smv2.LoadSteps(ctx, runB.ID)
	require.NoError(t, err)
	stepsC, err := infra.smv2.LoadSteps(ctx, runC.ID)
	require.NoError(t, err)
	require.Equal(t, json.RawMessage("null"), stepsA[op.ID])
	require.Equal(t, json.RawMessage("null"), stepsB[op.ID])
	require.Equal(t, json.RawMessage("null"), stepsC[op.ID])
}

// TestDeferCancel_ExecutorAndCheckpointProduceSameDefer exercises DeferCancel
// via all three paths (executor, sync checkpoint, async checkpoint) against
// runs that have been pre-seeded with a matching defer. All three paths must
// flip ScheduleStatus to Cancelled while preserving every other field.
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
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
		Input:          json.RawMessage(`{"user_id":"u_123"}`),
	}

	cancelOp := state.GeneratorOpcode{
		Op: enums.OpcodeDeferCancel,
		ID: cancelStepID,
		Opts: map[string]any{
			"target_hashed_id": deferStepID,
		},
	}

	driver := &mockDriverV1{
		t:        t,
		response: &state.DriverResponse{StatusCode: 206, Generator: []*state.GeneratorOpcode{&cancelOp}},
	}
	execA := infra.newExecutor(t, driver)
	runA := infra.scheduleRun(t, execA)
	require.NoError(t, infra.smv2.SaveDefer(ctx, runA.ID, seed))
	_, err := execA.Execute(ctx, state.Identifier{
		WorkflowID: infra.fnID, RunID: runA.ID.RunID, AccountID: infra.aID,
	}, queue.Item{
		WorkspaceID: infra.wsID,
		Kind:        queue.KindStart,
		Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: runA.ID.RunID, AccountID: infra.aID},
		Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: cancelStepID}},
	}, inngest.Edge{Incoming: "$trigger", Outgoing: cancelStepID})
	require.NoError(t, err)

	execB := infra.newExecutor(t, nil)
	runB := infra.scheduleRun(t, execB)
	require.NoError(t, infra.smv2.SaveDefer(ctx, runB.ID, seed))
	cp := infra.newCheckpointer(t, execB)
	err = cp.CheckpointSyncSteps(ctx, checkpoint.SyncCheckpoint{
		RunID:     runB.ID.RunID,
		FnID:      infra.fnID,
		AppID:     infra.appID,
		AccountID: infra.aID,
		EnvID:     infra.wsID,
		Metadata:  runB,
		Steps:     []state.GeneratorOpcode{cancelOp},
	})
	require.NoError(t, err)

	execC := infra.newExecutor(t, nil)
	runC := infra.scheduleRun(t, execC)
	require.NoError(t, infra.smv2.SaveDefer(ctx, runC.ID, seed))
	cp = infra.newCheckpointer(t, execC)
	err = cp.CheckpointAsyncSteps(ctx, checkpoint.AsyncCheckpoint{
		RunID:     runC.ID.RunID,
		FnID:      infra.fnID,
		AccountID: infra.aID,
		EnvID:     infra.wsID,
		Steps:     []state.GeneratorOpcode{cancelOp},
	})
	require.NoError(t, err)

	for name, runID := range map[string]statev2.ID{
		"executor":    runA.ID,
		"sync-ckpt":   runB.ID,
		"async-ckpt":  runC.ID,
	} {
		defers, err := infra.smv2.LoadDefers(ctx, runID)
		require.NoError(t, err, name)
		require.Len(t, defers, 1, name)
		d := defers[deferStepID]
		require.Equal(t, statev2.ScheduleStatusCancelled, d.ScheduleStatus,
			"%s: status should be Cancelled", name)
		require.Equal(t, seed.FnSlug, d.FnSlug, "%s: FnSlug must be preserved", name)
		require.Equal(t, seed.HashedID, d.HashedID, "%s: HashedID must be preserved", name)
		require.JSONEq(t, string(seed.Input), string(d.Input),
			"%s: Input must be preserved across status update", name)
	}
}

// TestDeferInputEmptyObjectSurvivesStatusUpdate guards against cjson's
// `{}` → `[]` corruption in setDeferStatus.lua by verifying that an
// empty-object Input survives a status flip.
func TestDeferInputEmptyObjectSurvivesStatusUpdate(t *testing.T) {
	infra := newDeferTestInfra(t)
	ctx := infra.ctx

	exec := infra.newExecutor(t, nil)
	md := infra.scheduleRun(t, exec)

	const hashedID = "step-defer"
	// Simulate an SDK that emits step.defer with no input (the SDK's default
	// serialization for Go-side `nil` / JS `undefined` can be `{}`).
	require.NoError(t, infra.smv2.SaveDefer(ctx, md.ID, statev2.Defer{
		FnSlug:         "onDefer-score",
		HashedID:       hashedID,
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
		Input:          json.RawMessage(`{}`),
	}))

	// Flip status — this executes setDeferStatus.lua, which round-trips the
	// full Defer JSON through cjson. Without normalization, `{}` would come
	// back as `[]`.
	require.NoError(t, infra.smv2.SetDeferStatus(ctx, md.ID, hashedID, statev2.ScheduleStatusCancelled))

	defers, err := infra.smv2.LoadDefers(ctx, md.ID)
	require.NoError(t, err)
	d := defers[hashedID]

	require.Equal(t, statev2.ScheduleStatusCancelled, d.ScheduleStatus)
	// Input must not have been corrupted into `[]`. Accept either nil or
	// a literal empty JSON object, since normalization picks nil.
	if len(d.Input) > 0 {
		require.JSONEq(t, `null`, string(d.Input),
			"empty-object Input should normalize to null, got %s", string(d.Input))
	}
}
