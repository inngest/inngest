package executor

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/require"
)

// aiGatewayDriver returns a mock driver whose generator response contains a
// single OpcodeAIGateway op with a minimal-but-valid inference request.
func aiGatewayDriver(t *testing.T, stepID string) *mockDriverV1 {
	t.Helper()
	return &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{{
				Op: enums.OpcodeAIGateway,
				ID: stepID,
				Opts: map[string]any{
					"url":    "http://ai-gateway.test/v1/chat/completions",
					"format": "openai-chat",
					"body":   map[string]any{"model": "gpt-4o-mini"},
				},
			}},
		},
	}
}

func executeAIGatewayStep(t *testing.T, infra *deferTestInfra, exec execution.Executor, run *statev2.Metadata, stepID string, attempt int) (*state.DriverResponse, error) {
	t.Helper()
	return exec.Execute(infra.ctx, state.Identifier{
		WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID,
	}, queue.Item{
		WorkspaceID: infra.wsID,
		Kind:        queue.KindStart,
		Identifier:  state.Identifier{WorkflowID: infra.fnID, RunID: run.ID.RunID, AccountID: infra.aID},
		Attempt:     attempt,
		Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "$trigger", Outgoing: stepID}},
	}, inngest.Edge{Incoming: "$trigger", Outgoing: stepID})
}

// TestAIGateway_RetryableFailure_NoSaveStepNoEnqueue pins the early retry
// return in handleGeneratorAIGateway: when the inference request fails and
// attempts remain, the handler returns an error for the queue to retry and
// must NOT save the step or enqueue a discovery step.
func TestAIGateway_RetryableFailure_NoSaveStepNoEnqueue(t *testing.T) {
	infra := newDeferTestInfra(t)
	capturingQ := &capturingQueue{Queue: infra.rq}
	capturingState := &saveStepCapturingState{RunService: infra.smv2}

	stepID := "step-defer"
	driver := aiGatewayDriver(t, stepID)
	inferenceErr := errors.New("provider unreachable")
	fakeHTTP := &scriptedHTTPExecutor{err: inferenceErr}

	exec := infra.newExecutorWithQueue(t, capturingQ, driver,
		executor.WithStateManager(capturingState),
		executor.WithHTTPClient(fakeHTTP),
	)
	run := infra.scheduleRun(t, exec)

	resp, err := executeAIGatewayStep(t, infra, exec, run, stepID, 0)

	require.Error(t, err)
	require.Contains(t, err.Error(), "error making inference request")
	require.ErrorIs(t, err, inferenceErr)

	effectiveMaxAttempts := infra.fn.Steps[0].RetryCount() + 1
	require.True(t, queue.ShouldRetry(err, 0, effectiveMaxAttempts),
		"the returned error must be retryable by the queue")

	require.NotNil(t, resp)
	require.True(t, resp.Retryable(), "the driver response must carry a retryable error")

	require.Equal(t, 1, fakeHTTP.callCount())
	require.Empty(t, capturingState.calls(), "a retryable inference failure must not save the step")
	require.Empty(t, capturingQ.itemsOfKind(queue.KindEdge), "a retryable inference failure must not enqueue a discovery step")
}

// TestAIGateway_NonRetryable_WrapsErrorSavesEnqueues pins the fall-through in
// handleGeneratorAIGateway when no attempts remain: the inference error is
// wrapped under the state error key, saved as the step output, and a discovery
// step is enqueued so the SDK can surface the error to userland.
func TestAIGateway_NonRetryable_WrapsErrorSavesEnqueues(t *testing.T) {
	infra := newDeferTestInfra(t)
	capturingQ := &capturingQueue{Queue: infra.rq}
	capturingState := &saveStepCapturingState{RunService: infra.smv2}

	stepID := "step-defer"
	driver := aiGatewayDriver(t, stepID)
	providerBody := `{"error":{"type":"server_error"}}`
	fakeHTTP := &scriptedHTTPExecutor{resp: &exechttp.Response{
		StatusCode: 500,
		Body:       []byte(providerBody),
	}}

	exec := infra.newExecutorWithQueue(t, capturingQ, driver,
		executor.WithStateManager(capturingState),
		executor.WithHTTPClient(fakeHTTP),
	)
	run := infra.scheduleRun(t, exec)

	// Execute rewrites trigger items' MaxAttempts to step retries + 1, so the
	// final attempt index comes from the function config, not the queue item.
	finalAttempt := infra.fn.Steps[0].RetryCount()
	resp, err := executeAIGatewayStep(t, infra, exec, run, stepID, finalAttempt)

	require.NoError(t, err, "a non-retryable inference failure continues the run rather than erroring")
	require.NotNil(t, resp)

	saveCalls := capturingState.calls()
	require.Len(t, saveCalls, 1, "the error-wrapped payload must be saved exactly once")
	require.Equal(t, run.ID, saveCalls[0].id)
	require.Equal(t, stepID, saveCalls[0].stepID)

	var savedPayload map[string]state.UserError
	require.NoError(t, json.Unmarshal(saveCalls[0].data, &savedPayload))
	require.Len(t, savedPayload, 1)
	savedErr, ok := savedPayload[execution.StateErrorKey]
	require.True(t, ok, "the saved step output must be keyed under the state error key")
	require.Equal(t, "AIGatewayError", savedErr.Name)
	require.Equal(t, "Error making AI request: unsuccessful status code: 500", savedErr.Message)
	require.JSONEq(t, providerBody, string(savedErr.Data))
	require.Equal(t, providerBody, savedErr.Stack)

	steps, err := infra.smv2.LoadSteps(infra.ctx, run.ID)
	require.NoError(t, err)
	require.Contains(t, steps, stepID)
	require.JSONEq(t, string(saveCalls[0].data), string(steps[stepID]))

	discoveryJobs := capturingQ.itemsOfKind(queue.KindEdge)
	require.Len(t, discoveryJobs, 1, "the shared tail must enqueue a discovery step")
	payload, ok := discoveryJobs[0].Payload.(queue.PayloadEdge)
	require.True(t, ok)
	require.Equal(t, stepID, payload.Edge.Outgoing)
}
