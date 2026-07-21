package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/checkpoint"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
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

// newCheckpointer builds a Checkpointer using the shared infra. The Executor
// is passed in so the checkpointer can reuse the same handler for non-Defer
// async opcodes; for Defer-only tests, any executor works.
func (i *execTestInfra) newCheckpointer(t *testing.T, exec execution.Executor) checkpoint.Checkpointer {
	t.Helper()
	return checkpoint.New(checkpoint.Opts{
		State:          i.smv2,
		FnReader:       i.dbcqrs,
		Executor:       exec,
		TracerProvider: tracing.NewSqlcTracerProvider(i.adapter.Q()),
		Queue:          i.rq,
	})
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
		infra := newExecTestInfra(t, "step-defer")
		ctx := infra.ctx

		exec := infra.newExecutor(t)
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
		r.Equal(infra.appID.String(), inn["parent_app_id"])
		r.Equal(infra.fnID.String(), inn["parent_fn_id"])
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
		infra := newExecTestInfra(t, "step-defer")
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
			executor.WithTracerProvider(tracing.NewSqlcTracerProvider(infra.adapter.Q())),
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
		infra := newExecTestInfra(t, "step-defer")
		ctx := infra.ctx

		exec := infra.newExecutor(t)
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
		r.NoError(infra.smv2.SaveDefer(ctx, run.ID, statev2.Defer{
			FnSlug:         "onDefer-score",
			HashedID:       "hash-rejected",
			ScheduleStatus: enums.DeferStatusRejected,
		}))

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
		infra := newExecTestInfra(t, "step-defer")
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
					exec := infra.newExecutor(t, executor.WithDriverV1(driver))
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
					exec := infra.newExecutor(t)
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
					exec := infra.newExecutor(t)
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
		infra := newExecTestInfra(t, "step-defer")
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

		exec := infra.newExecutorWithQueue(t, countingQ, executor.WithDriverV1(driver))

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
		infra := newExecTestInfra(t, "step-defer")
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

		exec := infra.newExecutorWithQueue(t, countingQ, executor.WithDriverV1(driver))
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
		infra := newExecTestInfra(t, "step-defer")
		ctx := infra.ctx

		const (
			plannedStepID  = "planned-step"
			plannedStepID2 = "planned-step-2"
			deferStepID    = "defer-add-step"
		)

		spy := &pendingCapturingState{RunService: infra.smv2}

		driver := &mockDriverV1{
			t: t,
			response: &state.DriverResponse{
				StatusCode: 206,
				// Two non-lazy ops are required for len(nonLazyIDs) > 1, which
				// is the condition that triggers SavePending.
				RequestVersion: 2,
				Generator: []*state.GeneratorOpcode{
					{Op: enums.OpcodeStepPlanned, ID: plannedStepID, Name: plannedStepID},
					{Op: enums.OpcodeStepPlanned, ID: plannedStepID2, Name: plannedStepID2},
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
		infra := newExecTestInfra(t, "step-defer")
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

		exec := infra.newExecutor(t, executor.WithDriverV1(driver))
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
		infra := newExecTestInfra(t, "step-defer")
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
							"input":   map[string]any{"msg": string(bigInput)},
						},
					},
					{
						Op: enums.OpcodeDeferAdd,
						ID: rejectedID,
						Opts: map[string]any{
							"fn_slug": "onDefer-score",
							"input":   map[string]any{"msg": string(overflowInput)},
						},
					},
				},
			},
		}

		exec := infra.newExecutor(t, executor.WithDriverV1(driver))
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
		infra := newExecTestInfra(t, "step-defer")
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
					exec := infra.newExecutor(t, executor.WithDriverV1(driver))
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
					exec := infra.newExecutor(t)
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
					exec := infra.newExecutor(t)
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
		infra := newExecTestInfra(t, "step-defer")
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

		exec := infra.newExecutorWithQueue(t, countingQ, executor.WithDriverV1(driver))
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

// runParentDefer schedules a parent run, drives Execute with the given
// DeferAdd hashedIDs piggybacked on RunComplete, and returns the parent run
// ID and the deferred.schedule events emitted by Finalize. Each hashedID gets
// its own DeferAdd op targeting the same child fn_slug.
func (i *execTestInfra) runParentDefer(t *testing.T, hashedIDs ...string) (ulid.ULID, []event.Event) {
	t.Helper()
	r := require.New(t)
	r.NotEmpty(hashedIDs)

	// Build ops (DeferAdd and RunComplete)
	ops := make([]*state.GeneratorOpcode, 0, len(hashedIDs)+1)
	for _, hashedID := range hashedIDs {
		ops = append(ops, &state.GeneratorOpcode{
			Op:   enums.OpcodeDeferAdd,
			ID:   hashedID,
			Opts: map[string]any{"fn_slug": i.fn.Slug, "input": map[string]any{}},
		})
	}
	ops = append(ops, &state.GeneratorOpcode{
		Op:   enums.OpcodeRunComplete,
		ID:   "run-complete",
		Data: json.RawMessage(`{"data": {}}`),
	})

	// Mock response to have the ops
	driver := &mockDriverV1{
		t:        t,
		response: &state.DriverResponse{StatusCode: 206, Generator: ops},
	}
	exec := i.newExecutor(t, executor.WithDriverV1(driver))

	// Capture finalization events
	var finalizationEvents []event.Event
	exec.SetFinalizer(func(_ context.Context, _ statev2.ID, events []event.Event) error {
		finalizationEvents = append(finalizationEvents, events...)
		return nil
	})

	parentRun := i.scheduleRun(t, exec)
	firstOp := hashedIDs[0]
	_, err := exec.Execute(i.ctx, state.Identifier{
		WorkflowID: i.fnID, RunID: parentRun.ID.RunID, AccountID: i.aID,
	}, queue.Item{
		Identifier:  state.Identifier{WorkflowID: i.fnID, RunID: parentRun.ID.RunID, AccountID: i.aID},
		Kind:        queue.KindStart,
		Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: firstOp}},
		WorkspaceID: i.wsID,
	}, inngest.Edge{Incoming: "$trigger", Outgoing: firstOp})
	r.NoError(err)

	// Filter defer events out of finalization events
	var deferEvents []event.Event
	for _, e := range finalizationEvents {
		if e.Name == consts.FnDeferScheduleName {
			deferEvents = append(deferEvents, e)
		}
	}
	r.Len(deferEvents, len(hashedIDs))
	return parentRun.ID.RunID, deferEvents
}

// TestDeferPropagatesSessions drives the full defer session-propagation path:
// a DeferAdd op carrying both a manual meta.sessions layer and a
// meta.propagatedSessions layer rides through SaveFromOp and the persisted
// Defer to Finalize, where buildDeferEvents resolves the two layers onto the
// emitted deferred.schedule event. ResolveSessions folds propagated into
// manual: manual wins on a key collision (tenant), manual-only keys pass
// through (user), and propagated-only keys fill free slots (org). The
// propagated layer is consumed in the process.
func TestDeferPropagatesSessions(t *testing.T) {
	r := require.New(t)
	infra := newExecTestInfra(t, "step-defer")

	const hashedID = "hash-session"

	ops := []*state.GeneratorOpcode{
		{
			Op: enums.OpcodeDeferAdd,
			ID: hashedID,
			Opts: map[string]any{
				"fn_slug": infra.fn.Slug,
				"input":   map[string]any{},
				// The SDK stamps the inherited session layer here at defer
				// call-time.
				"meta": map[string]any{
					"sessions":           map[string]any{"tenant": "manual-wins", "user": "u_1"},
					"propagatedSessions": map[string]any{"tenant": "acme", "org": "o_9"},
				},
			},
		},
		{
			Op:   enums.OpcodeRunComplete,
			ID:   "run-complete",
			Data: json.RawMessage(`{"data": {}}`),
		},
	}

	driver := &mockDriverV1{
		t:        t,
		response: &state.DriverResponse{StatusCode: 206, Generator: ops},
	}
	exec := infra.newExecutor(t, executor.WithDriverV1(driver))

	var finalizationEvents []event.Event
	exec.SetFinalizer(func(_ context.Context, _ statev2.ID, events []event.Event) error {
		finalizationEvents = append(finalizationEvents, events...)
		return nil
	})

	parentRun := infra.scheduleRun(t, exec)
	_, err := exec.Execute(infra.ctx, state.Identifier{
		WorkflowID: infra.fnID, RunID: parentRun.ID.RunID, AccountID: infra.aID,
	}, queue.Item{
		Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: parentRun.ID.RunID, AccountID: infra.aID},
		Kind:        queue.KindStart,
		Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: hashedID}},
		WorkspaceID: infra.wsID,
	}, inngest.Edge{Incoming: "$trigger", Outgoing: hashedID})
	r.NoError(err)

	evt := findDeferEvent(t, finalizationEvents, parentRun.ID.RunID, hashedID)
	r.Equal("manual-wins", evt.Meta.Sessions["tenant"], "manual layer wins on key collision")
	r.Equal("u_1", evt.Meta.Sessions["user"], "manual-only key survives")
	r.Equal("o_9", evt.Meta.Sessions["org"], "propagated-only key fills a free slot")
	r.Len(evt.Meta.Sessions, 3)
	r.Empty(evt.Meta.PropagatedSessions, "propagated layer consumed by ResolveSessions")
}

// findDeferEvent picks the deferred.schedule event whose ParentDeferSpan
// matches (parentRunID, hashedID). buildDeferEvents emits in
// non-deterministic order, so we can't rely on slice position.
func findDeferEvent(t *testing.T, events []event.Event, parentRunID ulid.ULID, hashedID string) event.Event {
	t.Helper()
	wantSpanID := tracing.DeferSpanRef(parentRunID, hashedID).DynamicSpanID
	for _, e := range events {
		md, err := e.DeferredScheduleMetadata()
		if err != nil || md.ParentDeferSpan == nil {
			continue
		}
		if md.ParentDeferSpan.DynamicSpanID == wantSpanID {
			return e
		}
	}
	t.Fatalf("no deferred.schedule event found for parent=%s hashedID=%s", parentRunID, hashedID)
	return event.Event{}
}

// scheduleChildRun schedules a child run with the given deferred.schedule
// events as its triggering batch.
func (i *execTestInfra) scheduleChildRun(t *testing.T, deferSchedules ...event.Event) ulid.ULID {
	t.Helper()
	require.NotEmpty(t, deferSchedules)

	exec := i.newExecutor(t)
	now := time.Now()
	events := make([]event.TrackedEvent, len(deferSchedules))
	for k, s := range deferSchedules {
		events[k] = event.NewBaseTrackedEventWithID(s, ulid.MustNew(ulid.Timestamp(now), rand.Reader))
	}
	_, childRun, err := exec.Schedule(i.ctx, execution.ScheduleRequest{
		Function:    i.fn,
		At:          &now,
		AccountID:   i.aID,
		WorkspaceID: i.wsID,
		AppID:       i.appID,
		Events:      events,
	})
	require.NoError(t, err)
	return childRun.ID.RunID
}

// deferRecord collapses a cqrs.RunDefer to the fields under test so subtests
// can compare expected vs. actual with a single Equal call. Empty ChildRunID
// covers the "child not yet scheduled" case by mismatch.
type deferRecord struct {
	HashedDeferID string
	FnSlug        string
	Status        enums.DeferStatus
	ChildRunID    string
}

func toDeferRecords(got map[ulid.ULID][]cqrs.RunDefer) map[ulid.ULID][]deferRecord {
	out := map[ulid.ULID][]deferRecord{}
	for k, defers := range got {
		for _, d := range defers {
			rec := deferRecord{HashedDeferID: d.HashedDeferID, FnSlug: d.FnSlug, Status: d.Status}
			if d.RunID != nil {
				rec.ChildRunID = d.RunID.String()
			}
			out[k] = append(out[k], rec)
		}
	}
	return out
}

func toParentRunIDs(got map[ulid.ULID][]cqrs.RunDeferredFrom) map[ulid.ULID][]ulid.ULID {
	out := map[ulid.ULID][]ulid.ULID{}
	for k, parents := range got {
		for _, p := range parents {
			out[k] = append(out[k], p.RunID)
		}
	}
	return out
}

// Assert that parents and children are properly linked via real exec.Schedule
// and exec.Execute calls.
func TestDeferLinkage(t *testing.T) {
	t.Run("1 parent to 1 child", func(t *testing.T) {
		r := require.New(t)
		infra := newExecTestInfra(t, "step-defer")

		parentRunID, evts := infra.runParentDefer(t, "hash-1")
		childRunID := infra.scheduleChildRun(t,
			findDeferEvent(t, evts, parentRunID, "hash-1"),
		)

		// Parent linked to the child
		defers, err := infra.dbcqrs.GetRunDefers(infra.ctx,
			[]ulid.ULID{parentRunID},
		)
		r.NoError(err)
		r.Equal(map[ulid.ULID][]deferRecord{
			parentRunID: {{
				ChildRunID:    childRunID.String(),
				FnSlug:        infra.fn.Slug,
				HashedDeferID: "hash-1",
				Status:        enums.DeferStatusAfterRun,
			}},
		}, toDeferRecords(defers))

		// Child linked to the parent
		parents, err := infra.dbcqrs.GetRunDeferredFrom(infra.ctx,
			[]ulid.ULID{childRunID},
		)
		r.NoError(err)
		r.Equal(map[ulid.ULID][]ulid.ULID{
			childRunID: {parentRunID},
		}, toParentRunIDs(parents))
		r.Equal(infra.fn.Slug, parents[childRunID][0].FnSlug)
	})

	// 1 parent run calls defer() twice, triggering 2 child runs.
	t.Run("1 parent to 2 children", func(t *testing.T) {
		r := require.New(t)
		infra := newExecTestInfra(t, "step-defer")

		parentRunID, evts := infra.runParentDefer(t, "hash-a", "hash-b")
		child1ID := infra.scheduleChildRun(t,
			findDeferEvent(t, evts, parentRunID, "hash-a"),
		)
		child2ID := infra.scheduleChildRun(t,
			findDeferEvent(t, evts, parentRunID, "hash-b"),
		)

		// Parent linked to the children
		defers, err := infra.dbcqrs.GetRunDefers(infra.ctx,
			[]ulid.ULID{parentRunID},
		)
		r.NoError(err)
		r.Equal(map[ulid.ULID][]deferRecord{
			parentRunID: {
				{
					ChildRunID:    child1ID.String(),
					FnSlug:        infra.fn.Slug,
					HashedDeferID: "hash-a",
					Status:        enums.DeferStatusAfterRun,
				},
				{
					ChildRunID:    child2ID.String(),
					FnSlug:        infra.fn.Slug,
					HashedDeferID: "hash-b",
					Status:        enums.DeferStatusAfterRun,
				},
			},
		}, toDeferRecords(defers))

		// Children linked to the parent. Each has its own link to the parent
		parents, err := infra.dbcqrs.GetRunDeferredFrom(infra.ctx,
			[]ulid.ULID{child1ID, child2ID},
		)
		r.NoError(err)
		r.Equal(map[ulid.ULID][]ulid.ULID{
			child1ID: {parentRunID},
			child2ID: {parentRunID},
		}, toParentRunIDs(parents))
	})

	// 2 parent runs call defer() and both events batch into 1 child run.
	t.Run("2 parents to 1 child", func(t *testing.T) {
		r := require.New(t)
		infra := newExecTestInfra(t, "step-defer")

		parent1ID, evts1 := infra.runParentDefer(t, "hash-a")
		parent2ID, evts2 := infra.runParentDefer(t, "hash-b")

		childID := infra.scheduleChildRun(t,
			findDeferEvent(t, evts1, parent1ID, "hash-a"),
			findDeferEvent(t, evts2, parent2ID, "hash-b"),
		)

		// Parents linked to the child. Each has its own link to the child
		defers, err := infra.dbcqrs.GetRunDefers(infra.ctx,
			[]ulid.ULID{parent1ID, parent2ID},
		)
		r.NoError(err)
		r.Equal(map[ulid.ULID][]deferRecord{
			parent1ID: {{
				ChildRunID:    childID.String(),
				FnSlug:        infra.fn.Slug,
				HashedDeferID: "hash-a",
				Status:        enums.DeferStatusAfterRun,
			}},
			parent2ID: {{
				ChildRunID:    childID.String(),
				FnSlug:        infra.fn.Slug,
				HashedDeferID: "hash-b",
				Status:        enums.DeferStatusAfterRun,
			}},
		}, toDeferRecords(defers))

		// Child linked to the parents
		parents, err := infra.dbcqrs.GetRunDeferredFrom(infra.ctx,
			[]ulid.ULID{childID},
		)
		r.NoError(err)
		r.Equal(map[ulid.ULID][]ulid.ULID{
			childID: {parent1ID, parent2ID},
		}, toParentRunIDs(parents))
	})

	// 1 parent run calls defer() twice and both events batch into 1 child run.
	t.Run("1 parent batches 2 defers to 1 child", func(t *testing.T) {
		r := require.New(t)
		infra := newExecTestInfra(t, "step-defer")

		parentRunID, evts := infra.runParentDefer(t, "hash-a", "hash-b")
		childID := infra.scheduleChildRun(t,
			findDeferEvent(t, evts, parentRunID, "hash-a"),
			findDeferEvent(t, evts, parentRunID, "hash-b"),
		)

		// Parent linked to the child. The 1 parent has 2 links to the child,
		// since there were 2 defers
		defers, err := infra.dbcqrs.GetRunDefers(infra.ctx,
			[]ulid.ULID{parentRunID},
		)
		r.NoError(err)
		r.Equal(map[ulid.ULID][]deferRecord{
			parentRunID: {
				{
					ChildRunID:    childID.String(),
					FnSlug:        infra.fn.Slug,
					HashedDeferID: "hash-a",
					Status:        enums.DeferStatusAfterRun,
				},
				{
					ChildRunID:    childID.String(),
					FnSlug:        infra.fn.Slug,
					HashedDeferID: "hash-b",
					Status:        enums.DeferStatusAfterRun,
				},
			},
		}, toDeferRecords(defers))

		// Child linked to the parent twice (once per defer event). We may want
		// to dedupe this, but right now there are dupes in the slice
		parents, err := infra.dbcqrs.GetRunDeferredFrom(infra.ctx,
			[]ulid.ULID{childID},
		)
		r.NoError(err)
		r.Equal(map[ulid.ULID][]ulid.ULID{
			childID: {parentRunID, parentRunID},
		}, toParentRunIDs(parents))
	})
}
