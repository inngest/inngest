package apiv2

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/tracing/meta"
	tracemetadata "github.com/inngest/inngest/pkg/tracing/metadata"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestToFunctionRun(t *testing.T) {
	runID := ulid.MustParse("01hp1zx8m3ng9vp6qn0xk7j4cy")
	eventID := ulid.MustParse("01hp1zyb8p2nb5kvm2a6x1h9ae")
	functionID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	appID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	batchID := ulid.MustParse("01hp20067njhe1rv6s6y8007xk")
	cron := "*/5 * * * *"
	startedAt := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(1500 * time.Millisecond)

	result := toFunctionRun(&cqrs.FunctionRun{
		RunID:        runID,
		RunStartedAt: startedAt,
		FunctionID:   functionID,
		EventID:      eventID,
		BatchID:      &batchID,
		Cron:         &cron,
		Status:       enums.RunStatusCompleted,
		EndedAt:      &endedAt,
		Output:       json.RawMessage(`{"ok":true}`),
	}, inngest.DeployedFunction{
		AppID:   appID,
		AppName: "agent-app",
		Slug:    "agent-app-summary",
		Function: inngest.Function{
			Name: "Summarize",
			Slug: "summary",
		},
	}, true)

	require.Equal(t, runID.String(), result.Id)
	require.Equal(t, "summary", result.Function.Id)
	require.Equal(t, "Summarize", result.Function.Name)
	require.Equal(t, "agent-app", result.App.Id)
	require.Equal(t, apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_COMPLETED, result.Status)
	require.Equal(t, ulid.Time(runID.Time()).UTC(), result.QueuedAt.AsTime())
	require.Equal(t, startedAt, result.StartedAt.AsTime())
	require.Equal(t, endedAt, result.EndedAt.AsTime())
	require.Equal(t, uint64(1500), *result.DurationMs)
	require.Equal(t, []string{eventID.String()}, result.Trigger.EventIds)
	require.True(t, result.Trigger.IsBatch)
	require.Equal(t, batchID.String(), *result.Trigger.BatchId)
	require.Equal(t, cron, *result.Trigger.CronSchedule)
	require.True(t, result.Output.Fields["ok"].GetBoolValue())
}

func TestToFunctionRunOmitsOutputWhenNotRequested(t *testing.T) {
	runID := ulid.MustParse("01hp1zx8m3ng9vp6qn0xk7j4cy")

	result := toFunctionRun(&cqrs.FunctionRun{
		RunID:        runID,
		RunStartedAt: time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
		EventID:      runID,
		Output:       json.RawMessage(`{"ok":true}`),
	}, inngest.DeployedFunction{}, false)

	require.Nil(t, result.Output)
	require.Nil(t, result.DurationMs)
	require.False(t, result.Trigger.IsBatch)
}

func TestFunctionRefID(t *testing.T) {
	t.Run("uses function slug when present", func(t *testing.T) {
		require.Equal(t, "function-slug", functionRefID(inngest.DeployedFunction{
			Slug:    "app-function-slug",
			AppName: "app",
			Function: inngest.Function{
				Slug: "function-slug",
			},
		}))
	})

	t.Run("trims app prefix from deployed slug", func(t *testing.T) {
		require.Equal(t, "function-slug", functionRefID(inngest.DeployedFunction{
			Slug:    "app-function-slug",
			AppName: "app",
		}))
	})

	t.Run("falls back to deployed slug", func(t *testing.T) {
		require.Equal(t, "function-slug", functionRefID(inngest.DeployedFunction{
			Slug: "function-slug",
		}))
	})
}

func TestAppRefID(t *testing.T) {
	appID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	require.Equal(t, "app-name", appRefID(inngest.DeployedFunction{
		AppID:   appID,
		AppName: "app-name",
	}))
	require.Equal(t, appID.String(), appRefID(inngest.DeployedFunction{
		AppID: appID,
	}))
}

func TestToFunctionRunStatus(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status enums.RunStatus
		want   apiv2.FunctionRunStatus
	}{
		{name: "completed", status: enums.RunStatusCompleted, want: apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_COMPLETED},
		{name: "failed", status: enums.RunStatusFailed, want: apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_FAILED},
		{name: "cancelled", status: enums.RunStatusCancelled, want: apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_CANCELLED},
		{name: "running", status: enums.RunStatusRunning, want: apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_RUNNING},
		{name: "scheduled", status: enums.RunStatusScheduled, want: apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_QUEUED},
		{name: "unknown", status: enums.RunStatusUnknown, want: apiv2.FunctionRunStatus_FUNCTION_RUN_STATUS_QUEUED},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, toFunctionRunStatus(tc.status))
		})
	}
}

func TestJSONToStruct(t *testing.T) {
	t.Run("maps JSON objects", func(t *testing.T) {
		result := jsonToStruct(json.RawMessage(`{"ok":true,"message":"hello","count":2}`))

		require.NotNil(t, result)
		require.True(t, result.Fields["ok"].GetBoolValue())
		require.Equal(t, "hello", result.Fields["message"].GetStringValue())
		require.Equal(t, float64(2), result.Fields["count"].GetNumberValue())
	})

	t.Run("returns nil for empty or invalid JSON", func(t *testing.T) {
		require.Nil(t, jsonToStruct(nil))
		require.Nil(t, jsonToStruct(json.RawMessage(`not-json`)))
		require.Nil(t, jsonToStruct(json.RawMessage(`[]`)))
	})
}

func TestOptionalString(t *testing.T) {
	value := optionalString("value")

	require.NotNil(t, value)
	require.Equal(t, "value", *value)
}

func TestToFunctionTrace(t *testing.T) {
	runID := ulid.MustParse("01jpq5jcxm8qhg2x61v61bh8p0")
	queuedAt := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	completed := enums.StepStatusCompleted

	trace, err := toFunctionTrace(context.Background(), &mockFunctionTraceReader{}, &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{
			Name:      meta.SpanNameRun,
			SpanID:    "run-span",
			TraceID:   "trace-id",
			StartTime: queuedAt,
			EndTime:   queuedAt.Add(time.Second),
		},
		RunID: runID,
		Attributes: &meta.ExtractedValues{
			DynamicStatus: &completed,
			QueuedAt:      &queuedAt,
		},
	}, false)

	require.NoError(t, err)
	require.Equal(t, runID.String(), trace.RunId)
	require.Equal(t, "run-span", trace.RootSpan.Id)
	require.Equal(t, meta.SpanNameRun, trace.RootSpan.Name)
}

func TestToTraceSpan(t *testing.T) {
	startedAt := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(2 * time.Second)
	updatedAt := startedAt.Add(time.Second)
	stepOp := models.StepOpRun
	stepID := "step-1"
	duration := 123
	outputIdentifier := cqrs.SpanIdentifier{TraceID: "trace-id", SpanID: "output-span"}
	outputID, err := outputIdentifier.Encode()
	require.NoError(t, err)

	reader := &mockFunctionTraceReader{}
	reader.On("GetSpanOutput", mock.Anything, outputIdentifier).Return(&cqrs.SpanOutput{
		Input: []byte(`{"input":true}`),
		Data:  []byte(`{"output":"ok"}`),
	}, nil).Once()
	t.Cleanup(func() {
		reader.AssertExpectations(t)
	})

	result, err := toTraceSpan(context.Background(), reader, &models.RunTraceSpan{
		SpanID:    "span-1",
		Name:      "Step",
		Status:    models.RunTraceSpanStatusCompleted,
		Duration:  &duration,
		OutputID:  &outputID,
		QueuedAt:  startedAt.Add(-time.Second),
		StartedAt: &startedAt,
		EndedAt:   &endedAt,
		StepOp:    &stepOp,
		StepID:    &stepID,
		Metadata: []*models.SpanMetadata{
			nil,
			{
				Scope:     enums.MetadataScopeStep,
				Kind:      tracemetadata.Kind("userland.http"),
				Values:    tracemetadata.Values{"status": json.RawMessage(`200`)},
				UpdatedAt: updatedAt,
			},
		},
		ChildrenSpans: []*models.RunTraceSpan{
			{
				SpanID:   "child",
				Name:     "Child",
				Status:   models.RunTraceSpanStatusRunning,
				QueuedAt: startedAt,
			},
		},
	}, true)

	require.NoError(t, err)
	require.Equal(t, "span-1", result.Id)
	require.Equal(t, "Step", result.Name)
	require.Equal(t, apiv2.TraceSpanStatus_TRACE_SPAN_STATUS_COMPLETED, result.Status)
	require.Equal(t, apiv2.TraceStepOp_TRACE_STEP_OP_RUN, *result.StepOp)
	require.Equal(t, stepID, *result.StepId)
	require.Equal(t, uint64(123), *result.DurationMs)
	require.Equal(t, startedAt.Add(-time.Second), result.QueuedAt.AsTime())
	require.Equal(t, startedAt, result.StartedAt.AsTime())
	require.Equal(t, endedAt, result.EndedAt.AsTime())
	require.True(t, result.Input.Fields["input"].GetBoolValue())
	require.Equal(t, "ok", result.Output.Fields["output"].GetStringValue())
	require.Len(t, result.Metadata, 1)
	require.Equal(t, "step", result.Metadata[0].Scope)
	require.Equal(t, "userland.http", result.Metadata[0].Kind)
	require.Equal(t, map[string]string{"status": "200"}, result.Metadata[0].Values)
	require.Equal(t, updatedAt, result.Metadata[0].UpdatedAt.AsTime())
	require.Len(t, result.Children, 1)
	require.Equal(t, "child", result.Children[0].Id)
}

func TestToTraceSpanOmitsOutputWithoutOutputID(t *testing.T) {
	result, err := toTraceSpan(context.Background(), &mockFunctionTraceReader{}, &models.RunTraceSpan{
		SpanID:   "span-1",
		Name:     "Step",
		Status:   models.RunTraceSpanStatusCompleted,
		QueuedAt: time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
	}, true)

	require.NoError(t, err)
	require.Nil(t, result.Input)
	require.Nil(t, result.Output)
}

func TestToTraceSpanPropagatesOutputErrors(t *testing.T) {
	outputIdentifier := cqrs.SpanIdentifier{TraceID: "trace-id", SpanID: "output-span"}
	outputID, err := outputIdentifier.Encode()
	require.NoError(t, err)

	reader := &mockFunctionTraceReader{}
	reader.On("GetSpanOutput", mock.Anything, outputIdentifier).Return(nil, errors.New("output read failed")).Once()
	t.Cleanup(func() {
		reader.AssertExpectations(t)
	})

	result, err := toTraceSpan(context.Background(), reader, &models.RunTraceSpan{
		SpanID:   "span-1",
		Name:     "Step",
		Status:   models.RunTraceSpanStatusCompleted,
		OutputID: &outputID,
		QueuedAt: time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
	}, true)

	require.Nil(t, result)
	require.ErrorContains(t, err, "output read failed")
}

func TestToTraceStepOp(t *testing.T) {
	for _, tc := range []struct {
		name   string
		stepOp models.StepOp
		want   apiv2.TraceStepOp
	}{
		{name: "run", stepOp: models.StepOpRun, want: apiv2.TraceStepOp_TRACE_STEP_OP_RUN},
		{name: "sleep", stepOp: models.StepOpSleep, want: apiv2.TraceStepOp_TRACE_STEP_OP_SLEEP},
		{name: "wait for event", stepOp: models.StepOpWaitForEvent, want: apiv2.TraceStepOp_TRACE_STEP_OP_WAIT_FOR_EVENT},
		{name: "invoke", stepOp: models.StepOpInvoke, want: apiv2.TraceStepOp_TRACE_STEP_OP_INVOKE},
		{name: "ai gateway", stepOp: models.StepOpAiGateway, want: apiv2.TraceStepOp_TRACE_STEP_OP_AI_GATEWAY},
		{name: "wait for signal", stepOp: models.StepOpWaitForSignal, want: apiv2.TraceStepOp_TRACE_STEP_OP_WAIT_FOR_SIGNAL},
		{name: "unknown", stepOp: models.StepOp("UNKNOWN"), want: apiv2.TraceStepOp_TRACE_STEP_OP_UNSPECIFIED},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, toTraceStepOp(tc.stepOp))
		})
	}
}

func TestToTraceSpanMetadata(t *testing.T) {
	updatedAt := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)

	result := toTraceSpanMetadata([]*models.SpanMetadata{
		nil,
		{
			Scope:     enums.MetadataScopeRun,
			Kind:      tracemetadata.Kind("userland.custom"),
			Values:    tracemetadata.Values{"key": json.RawMessage(`"value"`)},
			UpdatedAt: updatedAt,
		},
	})

	require.Len(t, result, 1)
	require.Equal(t, "run", result[0].Scope)
	require.Equal(t, "userland.custom", result[0].Kind)
	require.Equal(t, map[string]string{"key": `"value"`}, result[0].Values)
	require.Equal(t, updatedAt, result[0].UpdatedAt.AsTime())
	require.Nil(t, toTraceSpanMetadata(nil))
}

func TestToTraceSpanMetadataValues(t *testing.T) {
	require.Equal(t, map[string]string{
		"number": "1",
		"string": `"value"`,
	}, toTraceSpanMetadataValues(map[string]json.RawMessage{
		"number": json.RawMessage(`1`),
		"string": json.RawMessage(`"value"`),
	}))
	require.Nil(t, toTraceSpanMetadataValues(nil))
}

func TestTraceDuration(t *testing.T) {
	startedAt := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(2500 * time.Millisecond)
	explicit := 123
	negative := -1

	require.Equal(t, uint64(123), *traceDuration(&models.RunTraceSpan{Duration: &explicit}))
	require.Equal(t, uint64(2500), *traceDuration(&models.RunTraceSpan{
		StartedAt: &startedAt,
		EndedAt:   &endedAt,
	}))
	require.Equal(t, uint64(2500), *traceDuration(&models.RunTraceSpan{
		Duration:  &negative,
		StartedAt: &startedAt,
		EndedAt:   &endedAt,
	}))
	require.Nil(t, traceDuration(&models.RunTraceSpan{
		StartedAt: &endedAt,
		EndedAt:   &startedAt,
	}))
	require.Nil(t, traceDuration(&models.RunTraceSpan{}))
}

func TestLoadTraceOutput(t *testing.T) {
	identifier := cqrs.SpanIdentifier{TraceID: "trace-id", SpanID: "output-span"}
	encodedID, err := identifier.Encode()
	require.NoError(t, err)

	reader := &mockFunctionTraceReader{}
	reader.On("GetSpanOutput", mock.Anything, identifier).Return(&cqrs.SpanOutput{
		Input: []byte(`{"input":true}`),
		Data:  []byte(`{"output":"ok"}`),
	}, nil).Once()
	t.Cleanup(func() {
		reader.AssertExpectations(t)
	})

	result, err := loadTraceOutput(context.Background(), reader, encodedID)

	require.NoError(t, err)
	require.True(t, result.input.Fields["input"].GetBoolValue())
	require.Equal(t, "ok", result.output.Fields["output"].GetStringValue())
}

func TestLoadTraceOutputReturnsDecodeError(t *testing.T) {
	result, err := loadTraceOutput(context.Background(), &mockFunctionTraceReader{}, "not-base64")

	require.Nil(t, result)
	require.Error(t, err)
}
