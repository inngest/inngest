package executor

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/service"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type capSkipLifecycle struct {
	execution.NoopLifecyceListener

	mu      sync.Mutex
	skipped []execution.SkipState
}

func (l *capSkipLifecycle) OnFunctionSkipped(_ context.Context, _ statev2.Metadata, s execution.SkipState) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.skipped = append(l.skipped, s)
}

func TestExecutorScheduleAccountExecutionCap(t *testing.T) {
	infra := newExecTestInfra(t, "step")
	lc := &capSkipLifecycle{}

	exec := infra.newExecutor(t,
		executor.WithLifecycleListeners(lc),
		executor.WithAccountExecutionCap(func(context.Context, uuid.UUID) executor.ExecutionCapLimitDecision {
			return executor.ExecutionCapLimitDecision{Exceeded: true, Enforce: true}
		}),
	)

	now := time.Now()
	runID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)
	evtID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	_, md, err := exec.Schedule(infra.ctx, execution.ScheduleRequest{
		RunID:       &runID,
		Function:    infra.fn,
		At:          &now,
		AccountID:   infra.aID,
		WorkspaceID: infra.wsID,
		AppID:       infra.appID,
		Events: []event.TrackedEvent{
			event.NewBaseTrackedEventWithID(event.Event{Name: "test/event"}, evtID),
		},
	})
	require.ErrorIs(t, err, executor.ErrFunctionSkipped)
	require.Nil(t, md)

	var skipped executor.SkippedError
	require.True(t, errors.As(err, &skipped))
	require.Equal(t, enums.SkipReasonAccountExecutionCapHit, skipped.Reason)

	id := statev2.ID{
		RunID:      runID,
		FunctionID: infra.fnID,
		Tenant:     statev2.Tenant{AccountID: infra.aID, EnvID: infra.wsID, AppID: infra.appID},
	}
	_, err = infra.smv2.LoadState(infra.ctx, id)
	require.Error(t, err)

	jobs, err := infra.rq.RunJobs(infra.ctx, infra.queueShard.Name(), queue.Scope{
		AccountID:  infra.aID,
		EnvID:      infra.wsID,
		FunctionID: infra.fnID,
	}, runID, 1000, 0)
	require.NoError(t, err)
	require.Empty(t, jobs)

	service.Wait()

	lc.mu.Lock()
	defer lc.mu.Unlock()
	require.Len(t, lc.skipped, 1)
	require.Equal(t, enums.SkipReasonAccountExecutionCapHit, lc.skipped[0].Reason)
}

func TestExecutorInvokeAfterAccountExecutionCap(t *testing.T) {
	infra := newExecTestInfra(t, "step")
	var capped atomic.Bool
	invocations := make(chan event.TrackedEvent, 2)
	driver := &mockDriverV1{response: &state.DriverResponse{
		StatusCode: 206,
		Generator: []*state.GeneratorOpcode{{
			Op: enums.OpcodeInvokeFunction,
			ID: "call-child",
			Opts: map[string]any{
				"function_id": infra.fn.Slug,
				"payload":     map[string]any{"data": map[string]any{"input": "child"}},
			},
		}},
	}}
	exec := infra.newExecutor(t,
		executor.WithDriverV1(driver),
		executor.WithAccountExecutionCap(func(context.Context, uuid.UUID) executor.ExecutionCapLimitDecision {
			return executor.ExecutionCapLimitDecision{Exceeded: capped.Load(), Enforce: true}
		}),
		executor.WithInvokeEventHandler(func(_ context.Context, evt event.TrackedEvent) error {
			invocations <- evt
			return nil
		}),
	)
	run := infra.scheduleRun(t, exec)
	capped.Store(true)
	scope := queue.Scope{AccountID: infra.aID, EnvID: infra.wsID, FunctionID: infra.fnID}
	for _, generation := range []string{"child", "grandchild"} {
		t.Run(generation, func(t *testing.T) {
			jobs, err := infra.rq.RunJobs(infra.ctx, infra.queueShard.Name(), scope, run.ID.RunID, 100, 0)
			require.NoError(t, err)
			require.Len(t, jobs, 1)
			job := jobs[0].Raw.(*queue.QueueItem)
			_, err = exec.Execute(infra.ctx, statev2.V1FromMetadata(*run), job.Data, inngest.SourceEdge)
			require.NoError(t, err)
			var invoke event.TrackedEvent
			select {
			case invoke = <-invocations:
			case <-time.After(time.Second):
				t.Fatal("running parent did not publish step.invoke")
			}
			_, next, err := exec.Schedule(infra.ctx, execution.ScheduleRequest{
				Function: infra.fn, AccountID: infra.aID, WorkspaceID: infra.wsID, AppID: infra.appID,
				Events: []event.TrackedEvent{invoke},
			})
			require.NoError(t, err)
			require.NotNil(t, next)
			require.NotEqual(t, run.ID.RunID, next.ID.RunID)
			childJobs, err := infra.rq.RunJobs(infra.ctx, infra.queueShard.Name(), scope, next.ID.RunID, 100, 0)
			require.NoError(t, err)
			require.Len(t, childJobs, 1)
			run = next
		})
	}
}
