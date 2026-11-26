package executor

import (
	"testing"

	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

func TestScheduleLoopRerunFromStep(t *testing.T) {
	h := newLoopHarness(t)
	original, originalID := h.request(nil)
	_, _, err := h.e.Schedule(h.ctx, original)
	require.NoError(t, err)

	req, id := h.request(nil)
	req.Events = original.Events
	req.OriginalRunID = &originalID.RunID
	req.FromStep = &execution.ScheduleRequestFromStep{
		StepID: "step",
	}

	runID, md, err := h.e.Schedule(h.ctx, req)
	require.ErrorIs(t, err, ErrLoopRerunFromStep)
	require.Nil(t, runID)
	require.Nil(t, md)
	_, err = h.e.smv2.LoadMetadata(h.ctx, id)
	require.ErrorIs(t, err, state.ErrRunNotFound)
}

func TestScheduleLoopRerun(t *testing.T) {
	h := newLoopHarness(t)
	original, originalID := h.request(nil)
	_, _, err := h.e.Schedule(h.ctx, original)
	require.NoError(t, err)

	req, id := h.request(nil)
	req.Events = original.Events
	req.OriginalRunID = &originalID.RunID

	runID, md, err := h.e.Schedule(h.ctx, req)
	require.NoError(t, err)
	require.Equal(t, id.RunID, *runID)
	require.NotNil(t, md)
	require.Equal(t, &originalID.RunID, md.Config.OriginalRunID)
	_, err = h.e.smv2.LoadMetadata(h.ctx, id)
	require.NoError(t, err)
	h.requireStartSemaphore(t, id, "user-1")
}
