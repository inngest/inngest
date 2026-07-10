package executor

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// createWaitForEventPause schedules a run whose driver returns an
// OpcodeWaitForEvent opcode, drives it through Execute so the real pause
// manager writes a discoverable pause and the queue holds its timeout job,
// then returns the run metadata and the written pause.
func createWaitForEventPause(t *testing.T, infra *deferTestInfra, exec execution.Executor, stepID, eventName string) (*statev2.Metadata, *state.Pause) {
	t.Helper()

	run := infra.scheduleRun(t, exec)

	_, err := exec.Execute(infra.ctx, state.Identifier{
		WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
	}, queue.Item{
		WorkspaceID: infra.wsID,
		Kind:        queue.KindStart,
		Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
		Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
	}, inngest.Edge{Incoming: "$trigger", Outgoing: stepID})
	require.NoError(t, err)

	pauseID := inngest.DeterministicSha1UUID(run.ID.RunID.String() + stepID)
	pause, err := infra.pauseMgr.PauseByID(infra.ctx, pauses.Index{WorkspaceID: infra.wsID, EventName: eventName}, pauseID)
	require.NoError(t, err)
	require.NotNil(t, pause, "handleGeneratorWaitForEvent must write a pause discoverable by ID")

	return run, pause
}

// jobsOfKind lists the queue jobs currently scheduled for a run and filters
// them down to a single kind, so tests can observe enqueue/dequeue outcomes
// against the real queue rather than a mock.
func jobsOfKind(t *testing.T, infra *deferTestInfra, run *statev2.Metadata, kind string) []queue.JobResponse {
	t.Helper()

	jobs, err := infra.rq.RunJobs(infra.ctx, infra.queueShard.Name(), queue.Scope{
		AccountID:  run.ID.Tenant.AccountID,
		EnvID:      run.ID.Tenant.EnvID,
		FunctionID: run.ID.FunctionID,
	}, run.ID.RunID, 100, 0)
	require.NoError(t, err)

	var out []queue.JobResponse
	for _, job := range jobs {
		if job.Kind == kind {
			out = append(out, job)
		}
	}
	return out
}

// TestResume_TimeoutVsEvent_DequeueDivergence pins the one point where
// Resume's event-driven and timeout-driven paths diverge: consuming the
// pause, enqueueing the next edge, and firing OnWaitForEventResumed all
// happen either way, but only the event-driven path (r.IsTimeout == false)
// dequeues the now-stale pause timeout job from the real queue.
func TestResume_TimeoutVsEvent_DequeueDivergence(t *testing.T) {
	infra := newDeferTestInfra(t)
	recorder := newLifecycleRecorder()
	eventName := "resume/waitforevent"
	stepID := "step-defer"

	driver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{{
				Op:   enums.OpcodeWaitForEvent,
				ID:   stepID,
				Opts: map[string]any{"event": eventName, "timeout": "1h"},
			}},
		},
	}

	exec := infra.newExecutorWithQueue(t, infra.rq, driver, executor.WithLifecycleListeners(recorder))

	t.Run("event resume dequeues the stale timeout job", func(t *testing.T) {
		run, pause := createWaitForEventPause(t, infra, exec, stepID, eventName)
		require.Len(t, jobsOfKind(t, infra, run, queue.KindPause), 1, "waitForEvent must enqueue a pause timeout job")

		evtID := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)
		err := exec.Resume(infra.ctx, *pause, execution.ResumeRequest{
			With:      map[string]any{"result": "matched-by-event"},
			EventID:   &evtID,
			EventName: eventName,
			IsTimeout: false,
		})
		require.NoError(t, err)

		require.Empty(t, jobsOfKind(t, infra, run, queue.KindPause), "event-driven resume must dequeue the now-stale timeout job")
		require.Len(t, jobsOfKind(t, infra, run, queue.KindEdge), 1, "event-driven resume must enqueue the next edge")

		resumed := recorder.drainWaitForEventResumed(t, 1, 2*time.Second)
		require.False(t, resumed[0].req.IsTimeout, "the lifecycle hook must carry the event-driven request")
		require.Equal(t, run.ID.RunID, resumed[0].md.ID.RunID)

		_, err = infra.pauseMgr.PauseByID(infra.ctx, pauses.Index{WorkspaceID: infra.wsID, EventName: eventName}, pause.ID)
		require.ErrorIs(t, err, state.ErrPauseNotFound, "the consumed pause must be deleted")
	})

	t.Run("timeout resume leaves the timeout job queued", func(t *testing.T) {
		run, pause := createWaitForEventPause(t, infra, exec, stepID, eventName)
		require.Len(t, jobsOfKind(t, infra, run, queue.KindPause), 1, "waitForEvent must enqueue a pause timeout job")

		err := exec.Resume(infra.ctx, *pause, execution.ResumeRequest{
			With:      map[string]any{"result": "timed-out"},
			IsTimeout: true,
		})
		require.NoError(t, err)

		require.Len(t, jobsOfKind(t, infra, run, queue.KindPause), 1, "timeout-driven resume must skip dequeuing its own timeout job")
		require.Len(t, jobsOfKind(t, infra, run, queue.KindEdge), 1, "timeout-driven resume must still enqueue the next edge")

		resumed := recorder.drainWaitForEventResumed(t, 1, 2*time.Second)
		require.True(t, resumed[0].req.IsTimeout, "the lifecycle hook must carry the timeout-driven request")
		require.Equal(t, run.ID.RunID, resumed[0].md.ID.RunID)

		_, err = infra.pauseMgr.PauseByID(infra.ctx, pauses.Index{WorkspaceID: infra.wsID, EventName: eventName}, pause.ID)
		require.ErrorIs(t, err, state.ErrPauseNotFound, "the pause is still consumed and deleted on the timeout path")
	})
}

// TestResumePauseTimeout_Duplicate_LeavesPause drives the real race
// ResumePauseTimeout guards against: an event resumes the pause first
// (saving its own data for the step and enqueueing the next edge), then the
// pause's timeout job fires and calls ResumePauseTimeout with different
// data for the same step. SaveStep returns state.ErrDuplicateResponse, which
// must short-circuit before the next-edge enqueue and before the pause is
// touched again.
func TestResumePauseTimeout_Duplicate_LeavesPause(t *testing.T) {
	infra := newDeferTestInfra(t)
	eventName := "resume/timeout-duplicate"
	stepID := "step-defer"

	driver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{{
				Op:   enums.OpcodeWaitForEvent,
				ID:   stepID,
				Opts: map[string]any{"event": eventName, "timeout": "1h"},
			}},
		},
	}

	exec := infra.newExecutorWithQueue(t, infra.rq, driver)
	run, pause := createWaitForEventPause(t, infra, exec, stepID, eventName)

	evtID := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)
	err := exec.Resume(infra.ctx, *pause, execution.ResumeRequest{
		With:      map[string]any{"result": "matched-by-event"},
		EventID:   &evtID,
		EventName: eventName,
		IsTimeout: false,
	})
	require.NoError(t, err)
	require.Len(t, jobsOfKind(t, infra, run, queue.KindEdge), 1, "the winning event resume enqueues the next edge")

	_, err = infra.pauseMgr.PauseByID(infra.ctx, pauses.Index{WorkspaceID: infra.wsID, EventName: eventName}, pause.ID)
	require.ErrorIs(t, err, state.ErrPauseNotFound, "the event resume already deleted the pause")

	err = exec.ResumePauseTimeout(infra.ctx, *pause, execution.ResumeRequest{
		With:      map[string]any{"result": "timed-out"},
		IsTimeout: true,
	})
	require.NoError(t, err, "a duplicate resume must be swallowed rather than surfaced as an error")

	require.Len(t, jobsOfKind(t, infra, run, queue.KindEdge), 1, "the losing, duplicate timeout resume must not enqueue a second edge")

	_, err = infra.pauseMgr.PauseByID(infra.ctx, pauses.Index{WorkspaceID: infra.wsID, EventName: eventName}, pause.ID)
	require.ErrorIs(t, err, state.ErrPauseNotFound, "ResumePauseTimeout must not touch the already-deleted pause")
}
