package executor

// Characterization tests for executor.schedule's queue-duplicate cleanup and
// sync run-mode fork (docs/plans/006-executor-readability-refactor.md, Tier B).

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestSchedule_QueueDuplicate_DeletesUnownedState pins the keepState gate in
// schedule's queue.ErrQueueItemExists branch (executor.go ~1601-1618). Both
// sides of the gate are exercised: a duplicate owned by a different run
// deletes the just-created state to avoid a leak, while a duplicate owned by
// this same run leaves it alone to avoid losing data.
func TestSchedule_QueueDuplicate_DeletesUnownedState(t *testing.T) {
	t.Run("duplicate owned by a different run deletes the fresh state", func(t *testing.T) {
		infra := newDeferTestInfra(t)
		capturingQ := &capturingQueue{Queue: infra.rq}
		exec := infra.newExecutorWithQueue(t, capturingQ, nil)

		otherRunID := ulid.MustNew(ulid.Now(), rand.Reader)
		capturingQ.enqueueOverride = func(item queue.Item) error {
			return queue.QueueItemExists(*item.JobID, &otherRunID)
		}

		runID, md, err := scheduleTestRun(t, infra, exec, execution.ScheduleRequest{})
		require.ErrorIs(t, err, state.ErrIdentifierExists)
		require.Nil(t, md)
		require.NotNil(t, runID)

		id := stateIDFor(infra, *runID)
		exists, err := infra.smv2.Exists(infra.ctx, id)
		require.NoError(t, err)
		require.False(t, exists, "state owned by a different run must be deleted, not leaked")

		_, err = infra.smv2.LoadMetadata(infra.ctx, id)
		require.ErrorIs(t, err, state.ErrRunNotFound)
	})

	t.Run("duplicate owned by this run keeps the fresh state", func(t *testing.T) {
		infra := newDeferTestInfra(t)
		capturingQ := &capturingQueue{Queue: infra.rq}
		exec := infra.newExecutorWithQueue(t, capturingQ, nil)

		capturingQ.enqueueOverride = func(item queue.Item) error {
			return queue.QueueItemExists(*item.JobID, &item.Identifier.RunID)
		}

		runID, md, err := scheduleTestRun(t, infra, exec, execution.ScheduleRequest{})
		require.ErrorIs(t, err, state.ErrIdentifierExists)
		require.Nil(t, md)
		require.NotNil(t, runID)

		id := stateIDFor(infra, *runID)
		exists, err := infra.smv2.Exists(infra.ctx, id)
		require.NoError(t, err)
		require.True(t, exists, "state owned by this run must not be deleted")

		loaded, err := infra.smv2.LoadMetadata(infra.ctx, id)
		require.NoError(t, err)
		require.Equal(t, *runID, loaded.ID.RunID)
	})
}

// TestSchedule_SyncRunMode_CreatesStateWithoutEnqueue pins schedule's sync
// run-mode fork (executor.go ~1571-1577): with RunMode set to
// enums.RunModeSync, schedule creates run state and fires
// OnFunctionScheduled, but returns before ever calling queue.Enqueue, since
// sync runs are already executing in-process by the time Schedule is called.
func TestSchedule_SyncRunMode_CreatesStateWithoutEnqueue(t *testing.T) {
	infra := newDeferTestInfra(t)
	capturingQ := &capturingQueue{Queue: infra.rq}
	recorder := newLifecycleRecorder()
	exec := infra.newExecutorWithQueue(t, capturingQ, nil, executor.WithLifecycleListeners(recorder))

	runID, md, err := scheduleTestRun(t, infra, exec, execution.ScheduleRequest{RunMode: enums.RunModeSync})
	require.NoError(t, err)
	require.NotNil(t, md)
	require.NotNil(t, runID)

	require.Empty(t, capturingQ.itemsOfKind(queue.KindStart), "sync runs must not enqueue a start item")

	loaded, err := infra.smv2.LoadMetadata(infra.ctx, stateIDFor(infra, *runID))
	require.NoError(t, err, "sync run state must still be created")
	require.Equal(t, *runID, loaded.ID.RunID)

	scheduled := recorder.drainFunctionScheduled(t, 1, 2*time.Second)
	require.Equal(t, *runID, scheduled[0].md.ID.RunID)
}

// scheduleTestRun issues a Schedule call against infra's function using a
// fresh event, overlaying any caller-supplied fields (eg. RunMode) onto the
// base request.
func scheduleTestRun(t *testing.T, infra *deferTestInfra, exec execution.Executor, req execution.ScheduleRequest) (*ulid.ULID, *statev2.Metadata, error) {
	t.Helper()
	now := time.Now()
	evtID := ulid.MustNew(ulid.Timestamp(now), rand.Reader)

	req.Function = infra.fn
	req.At = &now
	req.AccountID = infra.aID
	req.WorkspaceID = infra.wsID
	req.AppID = infra.appID
	req.Events = []event.TrackedEvent{
		event.NewBaseTrackedEventWithID(event.Event{Name: "test/event"}, evtID),
	}

	return exec.Schedule(infra.ctx, req)
}

// stateIDFor builds the statev2.ID for a run scheduled against infra's
// function/tenant, for use with smv2.Exists/LoadMetadata.
func stateIDFor(infra *deferTestInfra, runID ulid.ULID) statev2.ID {
	return statev2.ID{
		RunID:      runID,
		FunctionID: infra.fnID,
		Tenant: statev2.Tenant{
			AccountID: infra.aID,
			EnvID:     infra.wsID,
			AppID:     infra.appID,
		},
	}
}
