package executor

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestReconstructUsesExecutorStepSpans(t *testing.T) {
	runID := ulid.Make()
	traceID := "trace-id"
	stepID := "step-1"
	fromStepID := "step-2"
	outputSpanID := "output-span-id"
	outputID := encodedOutputID(t, traceID, outputSpanID)

	root := &cqrs.OtelSpan{
		Children: []*cqrs.OtelSpan{
			executorStepSpan(stepID, time.UnixMilli(1), &outputID, nil),
			executorStepSpan(fromStepID, time.UnixMilli(3), nil, nil),
		},
	}

	newState := &sv2.CreateState{}
	_, err := reconstruct(context.Background(), fakeReconstructTraceReader{
		root: root,
		outputs: map[string]*cqrs.SpanOutput{
			outputSpanID: {Data: []byte(`{"ok":true}`)},
		},
	}, execution.ScheduleRequest{
		OriginalRunID: &runID,
		FromStep: &execution.ScheduleRequestFromStep{
			StepID: fromStepID,
		},
	}, newState)

	require.NoError(t, err)
	require.Equal(t, []state.MemoizedStep{
		{
			ID: stepID,
			Data: map[string]any{
				"data": map[string]any{"ok": true},
			},
		},
	}, newState.Steps)
}

func TestReconstructUsesHighestAttemptStepSpanForDuplicateStepIDs(t *testing.T) {
	runID := ulid.Make()
	traceID := "trace-id"
	stepID := "step-1"
	fromStepID := "step-2"
	attempt0OutputSpanID := "attempt-0-output-span-id"
	attempt1OutputSpanID := "attempt-1-output-span-id"
	attempt0OutputID := encodedOutputID(t, traceID, attempt0OutputSpanID)
	attempt1OutputID := encodedOutputID(t, traceID, attempt1OutputSpanID)

	root := &cqrs.OtelSpan{
		Children: []*cqrs.OtelSpan{
			executorStepSpan(stepID, time.UnixMilli(2), &attempt0OutputID, nil, withAttempt(0)),
			executorStepSpan(stepID, time.UnixMilli(1), &attempt1OutputID, nil, withAttempt(1)),
			executorStepSpan(fromStepID, time.UnixMilli(3), nil, nil),
		},
	}

	newState := &sv2.CreateState{}
	_, err := reconstruct(context.Background(), fakeReconstructTraceReader{
		root: root,
		outputs: map[string]*cqrs.SpanOutput{
			attempt0OutputSpanID: {Data: []byte(`"attempt 0"`)},
			attempt1OutputSpanID: {Data: []byte(`"attempt 1"`)},
		},
	}, execution.ScheduleRequest{
		OriginalRunID: &runID,
		FromStep: &execution.ScheduleRequestFromStep{
			StepID: fromStepID,
		},
	}, newState)

	require.NoError(t, err)
	require.Len(t, newState.Steps, 1)
	require.Equal(t, stepID, newState.Steps[0].ID)
	require.Equal(t, map[string]any{"data": "attempt 1"}, newState.Steps[0].Data)
}

func TestReconstructOrdersStepsDeterministicallyWhenStartTimesMatch(t *testing.T) {
	runID := ulid.Make()
	traceID := "trace-id"
	firstStepID := "step-a"
	secondStepID := "step-b"
	fromStepID := "step-c"
	firstOutputSpanID := "first-output-span-id"
	secondOutputSpanID := "second-output-span-id"
	firstOutputID := encodedOutputID(t, traceID, firstOutputSpanID)
	secondOutputID := encodedOutputID(t, traceID, secondOutputSpanID)
	at := time.UnixMilli(1)

	root := &cqrs.OtelSpan{
		Children: []*cqrs.OtelSpan{
			executorStepSpan(secondStepID, at, &secondOutputID, nil, withAttempt(1), withSpanID("span-a")),
			executorStepSpan(firstStepID, at, &firstOutputID, nil, withAttempt(0), withSpanID("span-b")),
			executorStepSpan(fromStepID, time.UnixMilli(2), nil, nil, withSpanID("span-c")),
		},
	}

	newState := &sv2.CreateState{}
	_, err := reconstruct(context.Background(), fakeReconstructTraceReader{
		root: root,
		outputs: map[string]*cqrs.SpanOutput{
			firstOutputSpanID:  {Data: []byte(`"first"`)},
			secondOutputSpanID: {Data: []byte(`"second"`)},
		},
	}, execution.ScheduleRequest{
		OriginalRunID: &runID,
		FromStep: &execution.ScheduleRequestFromStep{
			StepID: fromStepID,
		},
	}, newState)

	require.NoError(t, err)
	require.Len(t, newState.Steps, 2)
	require.Equal(t, firstStepID, newState.Steps[0].ID)
	require.Equal(t, secondStepID, newState.Steps[1].ID)
}

func TestReconstructMemoizesSleepWithoutOutput(t *testing.T) {
	runID := ulid.Make()
	sleepStepID := "sleep-step"
	fromStepID := "from-step"
	stepOp := enums.OpcodeSleep

	root := &cqrs.OtelSpan{
		Children: []*cqrs.OtelSpan{
			executorStepSpan(sleepStepID, time.UnixMilli(1), nil, nil, func(span *cqrs.OtelSpan) {
				span.Attributes.StepOp = &stepOp
			}),
			executorStepSpan(fromStepID, time.UnixMilli(2), nil, nil),
		},
	}

	newState := &sv2.CreateState{}
	_, err := reconstruct(context.Background(), fakeReconstructTraceReader{root: root}, execution.ScheduleRequest{
		OriginalRunID: &runID,
		FromStep: &execution.ScheduleRequestFromStep{
			StepID: fromStepID,
		},
	}, newState)

	require.NoError(t, err)
	require.Equal(t, []state.MemoizedStep{{ID: sleepStepID, Data: nil}}, newState.Steps)
}

func TestReconstructMemoizesTimedOutWaitWithoutOutput(t *testing.T) {
	runID := ulid.Make()
	waitStepID := "wait-step"
	fromStepID := "from-step"
	stepOp := enums.OpcodeWaitForEvent
	expired := true

	root := &cqrs.OtelSpan{
		Children: []*cqrs.OtelSpan{
			executorStepSpan(waitStepID, time.UnixMilli(1), nil, nil, func(span *cqrs.OtelSpan) {
				span.Attributes.StepOp = &stepOp
				span.Attributes.StepWaitExpired = &expired
			}),
			executorStepSpan(fromStepID, time.UnixMilli(2), nil, nil),
		},
	}

	newState := &sv2.CreateState{}
	_, err := reconstruct(context.Background(), fakeReconstructTraceReader{root: root}, execution.ScheduleRequest{
		OriginalRunID: &runID,
		FromStep: &execution.ScheduleRequestFromStep{
			StepID: fromStepID,
		},
	}, newState)

	require.NoError(t, err)
	require.Equal(t, []state.MemoizedStep{{ID: waitStepID, Data: nil}}, newState.Steps)
}

func TestReconstructPreservesFromStepInput(t *testing.T) {
	runID := ulid.Make()
	fromStepID := "from-step"

	root := &cqrs.OtelSpan{
		Children: []*cqrs.OtelSpan{
			executorStepSpan(fromStepID, time.UnixMilli(1), nil, nil),
		},
	}

	newState := &sv2.CreateState{}
	_, err := reconstruct(context.Background(), fakeReconstructTraceReader{root: root}, execution.ScheduleRequest{
		OriginalRunID: &runID,
		FromStep: &execution.ScheduleRequestFromStep{
			StepID: fromStepID,
			Input:  json.RawMessage(`[6,false]`),
		},
	}, newState)

	require.NoError(t, err)
	require.Len(t, newState.StepInputs, 1)
	require.Equal(t, fromStepID, newState.StepInputs[0].ID)
	require.JSONEq(t, `[6,false]`, string(newState.StepInputs[0].Data.(json.RawMessage)))
}

func TestReconstructResolvesStepName(t *testing.T) {
	runID := ulid.Make()
	fromStepID := "internal-step-id"
	fromStepName := "step 2"
	root := &cqrs.OtelSpan{
		Children: []*cqrs.OtelSpan{
			executorStepSpan(fromStepID, time.UnixMilli(1), nil, nil, withStepName(fromStepName)),
		},
	}

	newState := &sv2.CreateState{}
	result, err := reconstruct(context.Background(), fakeReconstructTraceReader{root: root}, execution.ScheduleRequest{
		OriginalRunID: &runID,
		FromStep: &execution.ScheduleRequestFromStep{
			StepID: fromStepName,
			Input:  json.RawMessage(`[{"foo":"bar"}]`),
		},
	}, newState)

	require.NoError(t, err)
	require.Equal(t, fromStepID, result.fromStepID)
	require.Equal(t, fromStepID, newState.StepInputs[0].ID)
}

func TestReconstructRejectsAmbiguousStepName(t *testing.T) {
	runID := ulid.Make()
	root := &cqrs.OtelSpan{
		Children: []*cqrs.OtelSpan{
			executorStepSpan("internal-step-1", time.UnixMilli(1), nil, nil, withStepName("duplicate")),
			executorStepSpan("internal-step-2", time.UnixMilli(2), nil, nil, withStepName("duplicate")),
		},
	}

	_, err := reconstruct(context.Background(), fakeReconstructTraceReader{root: root}, execution.ScheduleRequest{
		OriginalRunID: &runID,
		FromStep: &execution.ScheduleRequestFromStep{
			StepID: "duplicate",
		},
	}, &sv2.CreateState{})

	require.ErrorContains(t, err, "step name matches multiple steps")
}

func TestRerunFromStepEdgeTargetsRunnableStep(t *testing.T) {
	stepOp := enums.OpcodeStepRun
	edge := rerunFromStepEdge(execution.ScheduleRequest{
		FromStep: &execution.ScheduleRequestFromStep{
			StepID: "step-4",
		},
	}, []state.MemoizedStep{
		{ID: "step-1"},
		{ID: "step-2"},
		{ID: "step-3"},
	}, &reconstructResult{
		fromStepID: "step-4",
		fromStepOp: &stepOp,
	})

	require.Equal(t, "step-3", edge.Outgoing)
	require.Equal(t, "$trigger", edge.Incoming)
	require.Equal(t, "step-4", edge.IncomingGeneratorStep)
}

func TestRerunFromStepEdgeDoesNotTargetPlannedStep(t *testing.T) {
	stepOp := enums.OpcodeSleep
	edge := rerunFromStepEdge(execution.ScheduleRequest{
		FromStep: &execution.ScheduleRequestFromStep{
			StepID: "sleep-step",
		},
	}, []state.MemoizedStep{
		{ID: "step-1"},
	}, &reconstructResult{
		fromStepOp: &stepOp,
	})

	require.Equal(t, "step-1", edge.Outgoing)
	require.Equal(t, "$trigger", edge.Incoming)
	require.Empty(t, edge.IncomingGeneratorStep)
}

func TestRerunFromStepEdgeTargetsFirstStep(t *testing.T) {
	edge := rerunFromStepEdge(execution.ScheduleRequest{
		FromStep: &execution.ScheduleRequestFromStep{
			StepID: "step-1",
		},
	}, nil, nil)

	require.Empty(t, edge.Outgoing)
	require.Equal(t, "$trigger", edge.Incoming)
	require.Equal(t, "step-1", edge.IncomingGeneratorStep)
}

func TestRerunFromStepEdgeDefaultsToSource(t *testing.T) {
	require.Equal(t, inngest.SourceEdge, rerunFromStepEdge(execution.ScheduleRequest{}, nil, nil))
}

type stepSpanOption func(*cqrs.OtelSpan)

func executorStepSpan(stepID string, at time.Time, outputID *string, children []*cqrs.OtelSpan, opts ...stepSpanOption) *cqrs.OtelSpan {
	span := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{
			Name:      meta.SpanNameStep,
			StartTime: at,
		},
		Attributes: &meta.ExtractedValues{
			StepID: &stepID,
		},
		OutputID: outputID,
		Children: children,
	}

	for _, opt := range opts {
		opt(span)
	}

	return span
}

func withSpanID(spanID string) stepSpanOption {
	return func(span *cqrs.OtelSpan) {
		span.SpanID = spanID
	}
}

func withAttempt(attempt int) stepSpanOption {
	return func(span *cqrs.OtelSpan) {
		span.Attributes.StepAttempt = &attempt
	}
}

func withStepName(stepName string) stepSpanOption {
	return func(span *cqrs.OtelSpan) {
		span.Attributes.StepName = &stepName
	}
}

func encodedOutputID(t *testing.T, traceID string, spanID string) string {
	t.Helper()

	preview := true
	identifier := cqrs.SpanIdentifier{
		TraceID: traceID,
		SpanID:  spanID,
		Preview: &preview,
	}

	encoded, err := identifier.Encode()
	require.NoError(t, err)
	return encoded
}

type fakeReconstructTraceReader struct {
	cqrs.TraceReader
	root    *cqrs.OtelSpan
	outputs map[string]*cqrs.SpanOutput
}

func (f fakeReconstructTraceReader) GetSpansByRunID(context.Context, ulid.ULID) (*cqrs.OtelSpan, error) {
	return f.root, nil
}

func (f fakeReconstructTraceReader) GetSpanOutput(_ context.Context, id cqrs.SpanIdentifier) (*cqrs.SpanOutput, error) {
	return f.outputs[id.SpanID], nil
}
