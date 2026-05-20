package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	cqrsmanager "github.com/inngest/inngest/pkg/cqrs/manager"
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

// loadDefersFailingState wraps a real statev2.RunService and fails LoadDefers
// only. All other RunService methods delegate via the embedded interface.
type loadDefersFailingState struct {
	statev2.RunService
	err error
}

func (s *loadDefersFailingState) LoadDefers(ctx context.Context, id statev2.ID) (map[string]statev2.Defer, error) {
	return nil, s.err
}

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
	queueShard    redis_state.RedisQueueShard
	shardRegistry queue.ShardRegistryController
	rq            queue.Queue
}

func newDeferTestInfra(t *testing.T) *deferTestInfra {
	t.Helper()
	ctx := logger.WithStdlib(context.Background(), logger.VoidLogger())

	db, err := dbsqlite.Open(ctx, dbsqlite.Options{Persist: false})
	require.NoError(t, err)
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

func TestDeferFinalize(t *testing.T) {
	t.Run("emits schedule events for AfterRun defers", func(t *testing.T) {
		r := require.New(t)
		infra := newDeferTestInfra(t)
		ctx := infra.ctx

		exec := infra.newExecutor(t, nil)
		var capturedEvents []event.Event
		exec.SetFinalizer(func(ctx context.Context, id statev2.ID, events []event.Event) error {
			capturedEvents = append(capturedEvents, events...)
			return nil
		})

		run := infra.scheduleRun(t, exec)

		// Nested input verifies the event carries structured JSON rather than a stringified payload.
		nestedInputJSON := `{"user":{"id":"u_123","meta":{"role":"admin","tags":["a","b"]}},"score":0.87}`
		r.NoError(infra.smv2.SaveDefer(ctx, run.ID, statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-active",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(nestedInputJSON),
		}))
		r.NoError(infra.smv2.SaveDefer(ctx, run.ID, statev2.Defer{
			FnSlug:         "onDefer-cleanup",
			HashedID:       "hash-aborted",
			ScheduleStatus: enums.DeferStatusAborted,
			Input:          json.RawMessage(`{}`),
		}))

		err := exec.Finalize(ctx, execution.FinalizeOpts{
			Metadata: *run,
			Response: execution.FinalizeResponse{
				Type:        execution.FinalizeResponseRunComplete,
				RunComplete: state.GeneratorOpcode{Op: enums.OpcodeRunComplete},
			},
			Optional: execution.FinalizeOptional{
				FnSlug: infra.fn.Slug,
			},
		})
		r.NoError(err)

		var deferredFnSlugs []string
		var activeData map[string]any
		for _, evt := range capturedEvents {
			if evt.Name != "inngest/deferred.schedule" {
				continue
			}
			raw, err := json.Marshal(evt.Data)
			r.NoError(err)
			var data map[string]any
			r.NoError(json.Unmarshal(raw, &data))
			inn := data["_inngest"].(map[string]any)
			slug := inn["fn_slug"].(string)
			deferredFnSlugs = append(deferredFnSlugs, slug)
			if slug == "onDefer-score" {
				activeData = data
			}
		}

		r.Equal([]string{"onDefer-score"}, deferredFnSlugs,
			"only the AfterRun defer should emit deferred.schedule; aborted must not")
		r.NotNil(activeData)

		inn := activeData["_inngest"].(map[string]any)
		r.Equal(infra.fn.Slug, inn["parent_fn_slug"])
		r.Equal(run.ID.RunID.String(), inn["parent_run_id"])

		user, ok := activeData["user"].(map[string]any)
		r.True(ok, "data.user should be a JSON object, got %T", activeData["user"])
		r.Equal("u_123", user["id"])
		meta, ok := user["meta"].(map[string]any)
		r.True(ok, "data.user.meta should be a JSON object, got %T", user["meta"])
		r.Equal("admin", meta["role"])
		r.Equal([]any{"a", "b"}, meta["tags"])
		r.Equal(0.87, activeData["score"])
	})

	t.Run("continues on LoadDefers error", func(t *testing.T) {
		// Better to miss deferred runs than to block the run from finalizing.
		r := require.New(t)
		infra := newDeferTestInfra(t)
		ctx := infra.ctx

		failingState := &loadDefersFailingState{
			RunService: infra.smv2,
			err:        errors.New("simulated redis outage during LoadDefers"),
		}

		exec, err := executor.NewExecutor(
			executor.WithStateManager(failingState),
			executor.WithPauseManager(infra.pauseMgr),
			executor.WithQueue(infra.rq),
			executor.WithLogger(logger.StdlibLogger(ctx)),
			executor.WithFunctionLoader(infra.loader),
			executor.WithShardRegistry(infra.shardRegistry),
			executor.WithTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond)),
		)
		r.NoError(err)

		var capturedEvents []event.Event
		exec.SetFinalizer(func(ctx context.Context, id statev2.ID, events []event.Event) error {
			capturedEvents = append(capturedEvents, events...)
			return nil
		})

		run := infra.scheduleRun(t, exec)

		err = exec.Finalize(ctx, execution.FinalizeOpts{
			Metadata: *run,
			Response: execution.FinalizeResponse{
				Type:        execution.FinalizeResponseRunComplete,
				RunComplete: state.GeneratorOpcode{Op: enums.OpcodeRunComplete},
			},
			Optional: execution.FinalizeOptional{FnSlug: infra.fn.Slug},
		})
		r.NoError(err, "Finalize must complete despite LoadDefers failure")

		var sawFnFinished, sawDeferredSchedule bool
		for _, evt := range capturedEvents {
			if evt.Name == event.FnFinishedName {
				sawFnFinished = true
			}
			if evt.Name == "inngest/deferred.schedule" {
				sawDeferredSchedule = true
			}
		}
		r.True(sawFnFinished, "function.finished must publish even when LoadDefers fails")
		r.False(sawDeferredSchedule, "no defer events should be published when LoadDefers fails")
	})

	t.Run("rejected defers do not emit schedule events", func(t *testing.T) {
		r := require.New(t)
		infra := newDeferTestInfra(t)
		ctx := infra.ctx

		exec := infra.newExecutor(t, nil)
		var capturedEvents []event.Event
		exec.SetFinalizer(func(ctx context.Context, id statev2.ID, events []event.Event) error {
			capturedEvents = append(capturedEvents, events...)
			return nil
		})

		run := infra.scheduleRun(t, exec)

		r.NoError(infra.smv2.SaveDefer(ctx, run.ID, statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-active",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`{"x":1}`),
		}))
		r.NoError(infra.smv2.SaveRejectedDefer(ctx, run.ID, "onDefer-score", "hash-rejected"))

		r.NoError(exec.Finalize(ctx, execution.FinalizeOpts{
			Metadata: *run,
			Response: execution.FinalizeResponse{
				Type:        execution.FinalizeResponseRunComplete,
				RunComplete: state.GeneratorOpcode{Op: enums.OpcodeRunComplete},
			},
			Optional: execution.FinalizeOptional{FnSlug: infra.fn.Slug},
		}))

		var deferredFnSlugs []string
		for _, evt := range capturedEvents {
			if evt.Name != "inngest/deferred.schedule" {
				continue
			}
			raw, err := json.Marshal(evt.Data)
			r.NoError(err)
			var data map[string]any
			r.NoError(json.Unmarshal(raw, &data))
			inn := data["_inngest"].(map[string]any)
			deferredFnSlugs = append(deferredFnSlugs, inn["fn_slug"].(string))
		}
		r.Equal([]string{"onDefer-score"}, deferredFnSlugs,
			"only the AfterRun defer should emit; Rejected must be skipped at finalize")
	})
}

func TestDeferAdd(t *testing.T) {
	t.Run("consistent across executor and checkpoint paths", func(t *testing.T) {
		// Originally added to catch a regression where DeferAdd worked in
		// non-checkpointing codepaths but not in checkpointing.
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
			ScheduleStatus: enums.DeferStatusAfterRun,
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
			})
		}
	})

	t.Run("with RunComplete skips discovery", func(t *testing.T) {
		// When DeferAdd is piggybacked onto RunComplete, DeferAdd does NOT
		// enqueue a discovery step. Without this gating, the discovery
		// would be orphaned because RunComplete finalizes (deletes state)
		// immediately after.
		//
		// Also asserts the defer was actually saved on this path: an
		// early-return regression that skipped SaveDefer would still
		// produce zero enqueues, but would prevent Finalize from emitting
		// a deferred.schedule event for the defer. Observing the event
		// proves SaveDefer ran before state cleanup.
		r := require.New(t)
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

		exec := infra.newExecutorWithQueue(t, countingQ, driver)

		var capturedEvents []event.Event
		exec.SetFinalizer(func(ctx context.Context, id statev2.ID, events []event.Event) error {
			capturedEvents = append(capturedEvents, events...)
			return nil
		})

		run := infra.scheduleRun(t, exec)
		countBeforeExecute := countingQ.enqueues

		_, err := exec.Execute(infra.ctx, state.Identifier{
			WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
		}, queue.Item{
			WorkspaceID: infra.wsID,
			Kind:        queue.KindStart,
			Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
			Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
		}, inngest.Edge{Incoming: "$trigger", Outgoing: stepID})
		r.NoError(err)

		enqueuesDuringExecute := countingQ.enqueues - countBeforeExecute
		r.Equal(0, enqueuesDuringExecute,
			"DeferAdd should not enqueue discovery when piggybacked onto RunComplete; got %d enqueues", enqueuesDuringExecute)

		var deferredFnSlugs []string
		for _, evt := range capturedEvents {
			if evt.Name != "inngest/deferred.schedule" {
				continue
			}
			raw, err := json.Marshal(evt.Data)
			r.NoError(err)
			var data map[string]any
			r.NoError(json.Unmarshal(raw, &data))
			inn := data["_inngest"].(map[string]any)
			deferredFnSlugs = append(deferredFnSlugs, inn["fn_slug"].(string))
		}
		r.Equal([]string{"onDefer-score"}, deferredFnSlugs,
			"piggybacked DeferAdd must persist the defer; the deferred.schedule event is the post-Finalize evidence")
	})

	t.Run("bare op enqueues discovery", func(t *testing.T) {
		// Inverse of the WithRunComplete case: a bare [DeferAdd] with no
		// host op should fall through to enqueue a discovery step so the
		// run can progress. "Shouldn't happen" in normal operation (the
		// SDK piggybacks lazy ops), but the fallback path must stay safe.
		r := require.New(t)
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

		exec := infra.newExecutorWithQueue(t, countingQ, driver)
		run := infra.scheduleRun(t, exec)
		countBeforeExecute := countingQ.enqueues

		_, err := exec.Execute(infra.ctx, state.Identifier{
			WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
		}, queue.Item{
			WorkspaceID: infra.wsID,
			Kind:        queue.KindStart,
			Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
			Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
		}, inngest.Edge{Incoming: "$trigger", Outgoing: stepID})
		r.NoError(err)

		enqueuesDuringExecute := countingQ.enqueues - countBeforeExecute
		r.Equal(1, enqueuesDuringExecute,
			"bare DeferAdd should enqueue exactly one discovery step; got %d enqueues", enqueuesDuringExecute)

		defers, err := infra.smv2.LoadDefers(infra.ctx, run.ID)
		r.NoError(err)
		r.Contains(defers, stepID, "defer should be persisted even on bare-op path")
		r.Equal(enums.DeferStatusAfterRun, defers[stepID].ScheduleStatus)
	})

	t.Run("parallel plan excludes from pending set", func(t *testing.T) {
		// Lazy ops do not decrement the pending step count, so including
		// them in the pending set wedges the run.
		r := require.New(t)
		infra := newDeferTestInfra(t)
		ctx := infra.ctx

		const (
			plannedStepID = "planned-step"
			deferStepID   = "defer-add-step"
		)

		spy := &pendingCapturingState{RunService: infra.smv2}

		driver := &mockDriverV1{
			t: t,
			response: &state.DriverResponse{
				StatusCode: 206,
				// ShouldCoalesceParallelism returns true for >= 2; required for
				// the executor to invoke SavePending.
				RequestVersion: 2,
				Generator: []*state.GeneratorOpcode{
					{Op: enums.OpcodeStepPlanned, ID: plannedStepID, Name: plannedStepID},
					{
						Op: enums.OpcodeDeferAdd,
						ID: deferStepID,
						Opts: map[string]any{
							"fn_slug": "onDefer-score",
							"input":   map[string]any{},
						},
					},
				},
			},
		}

		exec, err := executor.NewExecutor(
			executor.WithStateManager(spy),
			executor.WithPauseManager(infra.pauseMgr),
			executor.WithQueue(infra.rq),
			executor.WithLogger(logger.StdlibLogger(ctx)),
			executor.WithFunctionLoader(infra.loader),
			executor.WithShardRegistry(infra.shardRegistry),
			executor.WithTracerProvider(tracing.NewOtelTracerProvider(nil, time.Millisecond)),
			executor.WithDriverV1(driver),
		)
		r.NoError(err)

		run := infra.scheduleRun(t, exec)

		_, err = exec.Execute(ctx, state.Identifier{
			WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
		}, queue.Item{
			WorkspaceID: infra.wsID,
			Kind:        queue.KindStart,
			Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
			Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: plannedStepID}},
		}, inngest.Edge{Incoming: "$trigger", Outgoing: plannedStepID})
		r.NoError(err)

		calls := spy.calls()
		r.NotEmpty(calls,
			"this test must exercise the SavePending path; if it stops firing, the regression guard becomes vacuous and the test setup needs revisiting (check hasPlanOp + ShouldCoalesceParallelism conditions)")

		for _, ids := range calls {
			r.NotContains(ids, deferStepID,
				"lazy op IDs (DeferAdd, DeferAbort) must not enter the pending set; got %v", ids)
		}
	})

	t.Run("oversized input soft fails", func(t *testing.T) {
		// Per-defer 4MB cap: an oversized DeferAdd does NOT fail the run,
		// and a Rejected sentinel is persisted so the SDK dedupes
		// retransmits. Bare DeferAdd (no RunComplete) so the run doesn't
		// finalize during Execute, leaving state inspectable afterward.
		r := require.New(t)
		infra := newDeferTestInfra(t)
		ctx := infra.ctx

		const stepID = "step-oversized"

		oversize := make([]byte, consts.MaxDeferInputSize+1024)
		for i := range oversize {
			oversize[i] = 'x'
		}

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
							"input":   string(oversize),
						},
					},
				},
			},
		}

		exec := infra.newExecutor(t, driver)
		run := infra.scheduleRun(t, exec)

		_, err := exec.Execute(ctx, state.Identifier{
			WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
		}, queue.Item{
			WorkspaceID: infra.wsID,
			Kind:        queue.KindStart,
			Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
			Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
		}, inngest.Edge{Incoming: "$trigger", Outgoing: stepID})
		r.NoError(err, "oversized DeferAdd must NOT fail the run; soft-fail with sentinel")

		defers, err := infra.smv2.LoadDefers(ctx, run.ID)
		r.NoError(err)
		r.Len(defers, 1)
		got := defers[stepID]
		r.Equal(enums.DeferStatusRejected, got.ScheduleStatus)
		r.Empty(got.Input)
	})

	t.Run("aggregate overflow soft fails", func(t *testing.T) {
		// Aggregate cap: a defer that would overflow
		// MaxDeferInputAggregateSize is rejected via sentinel without
		// failing the run. The earlier accepted defer remains valid.
		r := require.New(t)
		infra := newDeferTestInfra(t)
		ctx := infra.ctx

		const (
			acceptedID = "step-accepted"
			rejectedID = "step-rejected"
		)

		// 3MB + 2MB > 4MB cap.
		bigInput := make([]byte, 3*1024*1024)
		for i := range bigInput {
			bigInput[i] = 'a'
		}
		overflowInput := make([]byte, 2*1024*1024)
		for i := range overflowInput {
			overflowInput[i] = 'b'
		}

		driver := &mockDriverV1{
			t: t,
			response: &state.DriverResponse{
				StatusCode: 206,
				Generator: []*state.GeneratorOpcode{
					{
						Op: enums.OpcodeDeferAdd,
						ID: acceptedID,
						Opts: map[string]any{
							"fn_slug": "onDefer-score",
							"input":   string(bigInput),
						},
					},
					{
						Op: enums.OpcodeDeferAdd,
						ID: rejectedID,
						Opts: map[string]any{
							"fn_slug": "onDefer-score",
							"input":   string(overflowInput),
						},
					},
				},
			},
		}

		exec := infra.newExecutor(t, driver)
		run := infra.scheduleRun(t, exec)

		_, err := exec.Execute(ctx, state.Identifier{
			WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
		}, queue.Item{
			WorkspaceID: infra.wsID,
			Kind:        queue.KindStart,
			Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
			Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: acceptedID}},
		}, inngest.Edge{Incoming: "$trigger", Outgoing: acceptedID})
		r.NoError(err)

		defers, err := infra.smv2.LoadDefers(ctx, run.ID)
		r.NoError(err)
		r.Len(defers, 2)

		// Both DeferAdds race within the priority group's errgroup, so we
		// can't pin which one wins. The contract is: exactly one accepted,
		// exactly one rejected, sentinel carries no input.
		var afterRun, rejected int
		for _, d := range defers {
			switch d.ScheduleStatus {
			case enums.DeferStatusAfterRun:
				afterRun++
			case enums.DeferStatusRejected:
				rejected++
				r.Empty(d.Input)
			default:
				r.Failf("unexpected status", "defer %s: status=%s", d.HashedID, d.ScheduleStatus)
			}
		}
		r.Equal(1, afterRun)
		r.Equal(1, rejected)
	})
}

func TestDeferAbort(t *testing.T) {
	t.Run("consistent across executor and checkpoint paths", func(t *testing.T) {
		// Originally added to catch a regression where DeferAbort worked
		// in non-checkpointing codepaths but not in checkpointing.
		infra := newDeferTestInfra(t)
		ctx := infra.ctx

		const (
			deferStepID = "step-defer"
			abortStepID = "step-abort"
		)
		seed := statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       deferStepID,
			Input:          json.RawMessage(`{"user_id":"u_123"}`),
			ScheduleStatus: enums.DeferStatusAfterRun,
		}
		// Aborted transition releases the Input from the aggregate budget;
		// the meta entry stays so SDK retransmits stay sticky.
		expected := seed
		expected.ScheduleStatus = enums.DeferStatusAborted
		expected.Input = nil

		abortOp := state.GeneratorOpcode{
			ID: abortStepID,
			Op: enums.OpcodeDeferAbort,
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
						response: &state.DriverResponse{StatusCode: 206, Generator: []*state.GeneratorOpcode{&abortOp}},
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
						Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: abortStepID}},
						WorkspaceID: infra.wsID,
					}, inngest.Edge{Incoming: "$trigger", Outgoing: abortStepID})
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
						Steps:     []state.GeneratorOpcode{abortOp},
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
						Steps:     []state.GeneratorOpcode{abortOp},
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
	})

	t.Run("bare op enqueues discovery", func(t *testing.T) {
		// Inverse of TestDeferAdd's bare-op case. Pre-seeds the target
		// defer so SetDeferStatus succeeds and the bare-op branch is
		// actually reached (an error there would short-circuit before the
		// OnlyHasLazyOps check).
		r := require.New(t)
		infra := newDeferTestInfra(t)
		countingQ := &enqueueCountingQueue{Queue: infra.rq}

		const (
			deferStepID = "step-defer"
			abortStepID = "step-abort"
		)

		driver := &mockDriverV1{
			t: t,
			response: &state.DriverResponse{
				StatusCode: 206,
				Generator: []*state.GeneratorOpcode{
					{
						Op: enums.OpcodeDeferAbort,
						ID: abortStepID,
						Opts: map[string]any{
							"target_hashed_id": deferStepID,
						},
					},
				},
			},
		}

		exec := infra.newExecutorWithQueue(t, countingQ, driver)
		run := infra.scheduleRun(t, exec)

		r.NoError(infra.smv2.SaveDefer(infra.ctx, run.ID, statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       deferStepID,
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`{"user_id":"u_123"}`),
		}))

		countBeforeExecute := countingQ.enqueues

		_, err := exec.Execute(infra.ctx, state.Identifier{
			WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
		}, queue.Item{
			WorkspaceID: infra.wsID,
			Kind:        queue.KindStart,
			Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
			Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: abortStepID}},
		}, inngest.Edge{Incoming: "$trigger", Outgoing: abortStepID})
		r.NoError(err)

		enqueuesDuringExecute := countingQ.enqueues - countBeforeExecute
		r.Equal(1, enqueuesDuringExecute,
			"bare DeferAbort should enqueue exactly one discovery step; got %d enqueues", enqueuesDuringExecute)

		defers, err := infra.smv2.LoadDefers(infra.ctx, run.ID)
		r.NoError(err)
		r.Equal(enums.DeferStatusAborted, defers[deferStepID].ScheduleStatus)
	})
}
