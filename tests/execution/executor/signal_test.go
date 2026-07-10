package executor

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/require"
)

// TestSignalRoundTrip_WriteConsumeEnqueue drives a full signal lifecycle
// against the real pause manager: a run whose driver returns an
// OpcodeWaitForSignal opcode writes a discoverable pause, then ResumeSignal
// consumes it, enqueues the next edge, and plumbs the signal payload into
// saved step state. The signal path has zero existing coverage, so this pins
// current behavior rather than asserting an "ideal" contract.
func TestSignalRoundTrip_WriteConsumeEnqueue(t *testing.T) {
	infra := newDeferTestInfra(t)
	capturingQ := &capturingQueue{Queue: infra.rq}

	signalID := "signal-round-trip"
	stepID := "step-defer"

	driver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{{
				Op:   enums.OpcodeWaitForSignal,
				ID:   stepID,
				Opts: map[string]any{"signal": signalID},
			}},
		},
	}

	exec := infra.newExecutorWithQueue(t, capturingQ, driver)
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

	writtenPause, err := infra.pauseMgr.PauseBySignalID(infra.ctx, infra.wsID, signalID)
	require.NoError(t, err)
	require.NotNil(t, writtenPause, "handleGeneratorWaitForSignal must write a pause discoverable by signal ID")
	require.Equal(t, stepID, writtenPause.DataKey)

	pauseTimeoutJobs := capturingQ.itemsOfKind(queue.KindPause)
	require.Len(t, pauseTimeoutJobs, 1, "handleGeneratorWaitForSignal must enqueue a pause-timeout job")

	payload := json.RawMessage(`{"foo":"bar"}`)
	res, err := exec.ResumeSignal(infra.ctx, infra.wsID, signalID, payload)
	require.NoError(t, err)
	require.True(t, res.MatchedSignal, "ResumeSignal must report a match when the pause is consumed")
	require.NotNil(t, res.RunID)
	require.Equal(t, run.ID.RunID, *res.RunID)

	consumedPause, err := infra.pauseMgr.PauseBySignalID(infra.ctx, infra.wsID, signalID)
	require.NoError(t, err)
	require.Nil(t, consumedPause, "the pause must no longer be discoverable by signal ID once consumed")

	edgeJobs := capturingQ.itemsOfKind(queue.KindEdge)
	require.Len(t, edgeJobs, 1, "ResumeSignal must enqueue exactly one edge job to continue the run")

	steps, err := infra.smv2.LoadSteps(infra.ctx, run.ID)
	require.NoError(t, err)
	savedStep, ok := steps[stepID]
	require.True(t, ok, "the signal opcode's step output must be saved")

	var savedReturn struct {
		Data state.SignalStepReturn `json:"data"`
	}
	require.NoError(t, json.Unmarshal(savedStep, &savedReturn))
	require.Equal(t, signalID, savedReturn.Data.Signal, "the resumed signal ID must be plumbed into the saved step data")
	require.JSONEq(t, string(payload), string(savedReturn.Data.Data), "the ResumeSignal payload must be plumbed into the saved step data")
}
