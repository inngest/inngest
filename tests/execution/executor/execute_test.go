package executor

// Characterization tests for executor.Execute and executor.HandleResponse
// (docs/plans/006-executor-readability-refactor.md, Tier B).

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/stretchr/testify/require"
)

// Pins HandleResponse's first early return: an all-empty-OpcodeNone
// generator response re-drives discovery via restartDiscovery instead of
// falling through to the completion/finalization branches below it.
func TestHandleResponse_EmptyNoneOps_RedrivesDiscovery(t *testing.T) {
	infra := newDeferTestInfra(t)

	driver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			Generator: []*state.GeneratorOpcode{
				{Op: enums.OpcodeNone, ID: ""},
			},
		},
	}
	capturingQ := &capturingQueue{Queue: infra.rq}
	exec := infra.newExecutorWithQueue(t, capturingQ, driver)

	run := infra.scheduleRun(t, exec)

	jobs, err := infra.rq.RunJobs(infra.ctx, infra.queueShard.Name(), queue.Scope{
		AccountID:  run.ID.Tenant.AccountID,
		EnvID:      run.ID.Tenant.EnvID,
		FunctionID: run.ID.FunctionID,
	}, run.ID.RunID, 1000, 0)
	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	jobCtx := queue.WithJobID(infra.ctx, jobs[0].JobID)

	stepID := infra.fn.Steps[0].ID
	resp, err := exec.Execute(jobCtx, state.Identifier{
		WorkflowID: infra.fnID,
		RunID:      run.ID.RunID,
		AccountID:  infra.aID,
	}, queue.Item{
		WorkspaceID: infra.wsID,
		Kind:        queue.KindStart,
		Identifier: state.Identifier{
			WorkflowID: infra.fnID,
			RunID:      run.ID.RunID,
			AccountID:  infra.aID,
		},
		Payload: queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
	}, inngest.Edge{
		Incoming: "$trigger",
		Outgoing: stepID,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	redriven := capturingQ.itemsOfKind(queue.KindEdge)
	require.Len(t, redriven, 1, "expected exactly one redrive discovery enqueue")
	edge, ok := redriven[0].Payload.(queue.PayloadEdge)
	require.True(t, ok)
	require.Equal(t, stepID, edge.Edge.Incoming)

	md, err := infra.smv2.LoadMetadata(infra.ctx, run.ID)
	require.NoError(t, err, "run must not be finalized/deleted by a redrive response")
	require.Equal(t, run.ID.RunID, md.ID.RunID)
}

// Pins run()'s ErrNoRuntimeDriver path: reached here via a step URI whose
// scheme (ws://) resolves to a driver name distinct from the registered
// "http" one. run() returns (nil, err) before invoking any driver.
//
// Surprising current behavior, pinned as-is: Execute never surfaces this
// error. It unconditionally calls tracing.DriverResponseAttrs(resp, nil)
// with resp == nil before its own `resp == nil && err != nil` guard, which
// panics on the nil dereference. The "run step" CritT call passes
// util.WithTimeout, so CritT runs the closure in a goroutine with a
// recover(); that recover only wraps the panic into err if its local err
// var was already non-nil, which it isn't here (the panic happens mid
// `res, err = f(ctx)`, before either is assigned), so the panic is silently
// swallowed and Execute returns (nil, nil).
func TestExecute_InternalDriverError_SkipsHandleResponse(t *testing.T) {
	infra := newUnregisteredDriverInfra(t, "ws://example.com/step")

	httpDriver := &mockDriverV1{t: t}
	capturingQ := &capturingQueue{Queue: infra.rq}
	recorder := newLifecycleRecorder()
	exec := infra.newExecutorWithQueue(t, capturingQ, httpDriver, executor.WithLifecycleListeners(recorder))

	run := infra.scheduleRun(t, exec)

	jobs, err := infra.rq.RunJobs(infra.ctx, infra.queueShard.Name(), queue.Scope{
		AccountID:  run.ID.Tenant.AccountID,
		EnvID:      run.ID.Tenant.EnvID,
		FunctionID: run.ID.FunctionID,
	}, run.ID.RunID, 1000, 0)
	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	jobCtx := queue.WithJobID(infra.ctx, jobs[0].JobID)

	stepID := infra.fn.Steps[0].ID
	resp, err := exec.Execute(jobCtx, state.Identifier{
		WorkflowID: infra.fnID,
		RunID:      run.ID.RunID,
		AccountID:  infra.aID,
	}, queue.Item{
		WorkspaceID: infra.wsID,
		Kind:        queue.KindStart,
		Identifier: state.Identifier{
			WorkflowID: infra.fnID,
			RunID:      run.ID.RunID,
			AccountID:  infra.aID,
		},
		Payload: queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
	}, inngest.Edge{
		Incoming: "$trigger",
		Outgoing: stepID,
	})

	require.Nil(t, resp)
	require.NoError(t, err, "current behavior swallows the internal driver error; see comment above")
	require.Equal(t, 0, httpDriver.callCount(), "driver must never be invoked when no runtime driver resolves")

	require.Empty(t, capturingQ.itemsOfKind(queue.KindEdge), "HandleResponse's redrive/generator path must not run")
	recorder.requireNoFunctionFinished(t, 200*time.Millisecond)

	md, err := infra.smv2.LoadMetadata(infra.ctx, run.ID)
	require.NoError(t, err, "run must not be finalized/deleted when HandleResponse is skipped")
	require.Equal(t, run.ID.RunID, md.ID.RunID)
}

// Pins the item.Attempt == 0 gate in Execute's trigger-edge branch:
// OnFunctionStarted fires, and StartedAt/RequestVersion are persisted, only
// on the first attempt.
func TestExecute_FirstAttemptOnly_FiresFunctionStartedOnce(t *testing.T) {
	infra := newDeferTestInfra(t)

	// Always-retryable and non-finalizing, so the same run can be executed
	// across two attempts.
	driver := &mockDriverV1{
		t: t,
		err: syscode.Error{
			Code:    syscode.CodeConnectAllWorkersAtCapacity,
			Message: "All workers are at capacity",
		},
	}
	recorder := newLifecycleRecorder()
	exec := infra.newExecutorWithQueue(t, infra.rq, driver, executor.WithLifecycleListeners(recorder))

	run := infra.scheduleRun(t, exec)

	jobs, err := infra.rq.RunJobs(infra.ctx, infra.queueShard.Name(), queue.Scope{
		AccountID:  run.ID.Tenant.AccountID,
		EnvID:      run.ID.Tenant.EnvID,
		FunctionID: run.ID.FunctionID,
	}, run.ID.RunID, 1000, 0)
	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	jobCtx := queue.WithJobID(infra.ctx, jobs[0].JobID)
	stepID := infra.fn.Steps[0].ID

	buildItem := func(attempt int) queue.Item {
		return queue.Item{
			WorkspaceID: infra.wsID,
			Kind:        queue.KindStart,
			Identifier: state.Identifier{
				WorkflowID: infra.fnID,
				RunID:      run.ID.RunID,
				AccountID:  infra.aID,
			},
			Attempt:     attempt,
			MaxAttempts: new(5),
			Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
		}
	}

	id := state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID}
	edge := inngest.Edge{Incoming: "$trigger", Outgoing: stepID}

	_, err = exec.Execute(jobCtx, id, buildItem(0), edge)
	require.Error(t, err, "capacity errors are always retryable")

	recorder.drainFunctionStarted(t, 1, 2*time.Second)

	mdAfterFirst, err := infra.smv2.LoadMetadata(infra.ctx, run.ID)
	require.NoError(t, err)
	require.False(t, mdAfterFirst.Config.StartedAt.IsZero(), "StartedAt must be persisted on the first attempt")

	_, err = exec.Execute(jobCtx, id, buildItem(1), edge)
	require.Error(t, err)

	recorder.requireNoFunctionStarted(t, 300*time.Millisecond)

	mdAfterSecond, err := infra.smv2.LoadMetadata(infra.ctx, run.ID)
	require.NoError(t, err)
	require.True(t, mdAfterFirst.Config.StartedAt.Equal(mdAfterSecond.Config.StartedAt),
		"StartedAt must not be re-persisted on a retry attempt")
	require.Equal(t, mdAfterFirst.Config.RequestVersion, mdAfterSecond.Config.RequestVersion,
		"RequestVersion must not change on a retry attempt")
}

// newUnregisteredDriverInfra builds a deferTestInfra whose function's single
// step resolves (via pkg/inngest.Driver) to a driver name distinct from
// "http", so a driver registered under WithDriverV1 (always keyed by
// mockDriverV1.Name() == "http") is never selected by executor.fnDriver.
func newUnregisteredDriverInfra(t *testing.T, stepURI string) *deferTestInfra {
	t.Helper()
	infra := newDeferTestInfra(t)

	infra.fn.Steps[0].URI = stepURI
	config, err := json.Marshal(infra.fn)
	require.NoError(t, err)

	_, err = infra.dbcqrs.UpsertFunction(infra.ctx, cqrs.UpsertFunctionParams{
		ID:     infra.fnID,
		AppID:  infra.appID,
		Name:   infra.fn.Name,
		Slug:   infra.fn.Slug,
		Config: string(config),
	})
	require.NoError(t, err)

	return infra
}
