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

// TestDeferAddSavesDefer verifies that when the executor processes an
// OpcodeDeferAdd generator response, it:
//  1. Memoizes the step with null data (SaveStep)
//  2. Persists a Defer record in the run's defers map (SaveDefer)
func TestDeferAddSavesDefer(t *testing.T) {
	ctx := context.Background()

	// --- boilerplate: database + function loader ---
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

	// --- boilerplate: Redis + state manager + queue ---
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

	// --- mock driver: returns a DeferAdd opcode ---
	stepID := "step-defer"
	mockDriver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{{
				Op: enums.OpcodeDeferAdd,
				ID: stepID,
				Opts: map[string]any{
					"companion_id": "score",
					"input":        map[string]any{"user_id": "u_123"},
				},
			}},
		},
	}

	// --- create executor ---
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

	// --- schedule + execute ---
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

	// --- assertions ---

	// 1. The step should be memoized with null data
	steps, err := smv2.LoadSteps(ctx, run.ID)
	require.NoError(t, err)
	require.Contains(t, steps, stepID, "step.defer should be memoized")
	require.Equal(t, json.RawMessage("null"), steps[stepID], "memoized data should be null")

	// 2. A defer record should exist in the run's defers map
	defers, err := smv2.LoadDefers(ctx, run.ID)
	require.NoError(t, err)
	require.Len(t, defers, 1, "expected exactly one defer")

	d := defers[stepID]
	require.Equal(t, "score", d.CompanionID)
	require.Equal(t, "test-fn-defer-score", d.FnSlug)
	require.Equal(t, statev2.ScheduleStatusAfterRun, d.ScheduleStatus)
	require.JSONEq(t, `{"user_id":"u_123"}`, string(d.Input))
}

// TestFinalizeEmitsDeferredStartEvents verifies that when a run with defers
// is finalized, the executor emits an inngest/deferred.start event for each
// defer whose ScheduleStatus is AfterRun.
func TestFinalizeEmitsDeferredStartEvents(t *testing.T) {
	ctx := context.Background()

	// --- boilerplate: database + function loader ---
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

	// --- boilerplate: Redis + state manager + queue ---
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

	// --- create executor (no driver needed — we won't Execute, just Finalize) ---
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

	// --- set up a spy to capture events published during finalization ---
	var capturedEvents []event.Event
	exec.SetFinalizer(func(ctx context.Context, id statev2.ID, events []event.Event) error {
		capturedEvents = append(capturedEvents, events...)
		return nil
	})

	// --- schedule a run so state exists ---
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

	// --- save two defers: one active (AfterRun) and one cancelled ---
	activeDefer := statev2.Defer{
		CompanionID:    "score",
		FnSlug:         "my-app-score",
		HashedID:       "hash-active",
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
		Input:          json.RawMessage(`{"user_id":"u_123"}`),
	}
	cancelledDefer := statev2.Defer{
		CompanionID:    "cleanup",
		FnSlug:         "my-app-cleanup",
		HashedID:       "hash-cancelled",
		ScheduleStatus: statev2.ScheduleStatusCancelled,
		Input:          json.RawMessage(`{}`),
	}

	require.NoError(t, smv2.SaveDefer(ctx, run.ID, activeDefer))
	require.NoError(t, smv2.SaveDefer(ctx, run.ID, cancelledDefer))

	// --- finalize the run ---
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

	// --- assert inngest/deferred.start events ---
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

	// Verify the event shape from the ticket
	var data map[string]any
	require.NoError(t, json.Unmarshal(evtData, &data))

	inngestData := data["_inngest"].(map[string]any)
	deferredRun := inngestData["deferred_run"].(map[string]any)
	parentRun := inngestData["parent_run"].(map[string]any)

	require.Equal(t, "score", deferredRun["companion_id"])
	require.Equal(t, fn.Slug, parentRun["fn_slug"])
	require.Equal(t, run.ID.RunID.String(), parentRun["run_id"])

	// Verify user input is forwarded
	input := data["input"].(map[string]any)
	require.Equal(t, "u_123", input["user_id"])
}

// TestDeferCancelUpdatesDeferStatus verifies that when the executor processes
// an OpcodeDeferCancel, it flips the existing defer's ScheduleStatus to Cancelled.
func TestDeferCancelUpdatesDeferStatus(t *testing.T) {
	ctx := context.Background()

	// --- boilerplate: database + function loader ---
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

	// --- boilerplate: Redis + state manager + queue ---
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

	stepID := "step-defer"

	// --- mock driver: returns DeferCancel opcode ---
	mockDriver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{{
				Op: enums.OpcodeDeferCancel,
				ID: stepID,
				Opts: map[string]any{
					"companion_id": "score",
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

	// --- schedule a run ---
	now := time.Now()
	evtID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	_, run, err := exec.Schedule(ctx, execution.ScheduleRequest{
		Function: fn, At: &now, AccountID: aID, WorkspaceID: wsID, AppID: appID,
		Events: []event.TrackedEvent{
			event.NewBaseTrackedEventWithID(event.Event{Name: "test/event"}, evtID),
		},
	})
	require.NoError(t, err)

	// --- pre-seed a defer (as if DeferAdd already ran) ---
	require.NoError(t, smv2.SaveDefer(ctx, run.ID, statev2.Defer{
		CompanionID:    "score",
		HashedID:       stepID,
		ScheduleStatus: statev2.ScheduleStatusAfterRun,
		Input:          json.RawMessage(`{"user_id":"u_123"}`),
	}))

	// --- execute the DeferCancel opcode ---
	_, err = exec.Execute(ctx, state.Identifier{
		WorkflowID: fnID, RunID: run.ID.RunID, AccountID: aID,
	}, queue.Item{
		WorkspaceID: wsID,
		Kind:        queue.KindStart,
		Identifier:  state.Identifier{WorkflowID: fnID, RunID: run.ID.RunID, AccountID: aID},
		Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
	}, inngest.Edge{Incoming: "$trigger", Outgoing: stepID})
	require.NoError(t, err)

	// --- assert the defer status flipped to Cancelled ---
	defers, err := smv2.LoadDefers(ctx, run.ID)
	require.NoError(t, err)
	require.Len(t, defers, 1)

	d := defers[stepID]
	require.Equal(t, "score", d.CompanionID, "CompanionID should be preserved")
	require.Equal(t, statev2.ScheduleStatusCancelled, d.ScheduleStatus, "status should be Cancelled")
	require.JSONEq(t, `{"user_id":"u_123"}`, string(d.Input), "Input should be preserved")
}
