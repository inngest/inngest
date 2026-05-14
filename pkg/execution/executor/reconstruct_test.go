package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestReconstructStackUsesDescendantOutputSpan(t *testing.T) {
	stepID := "step-1"
	fromStepID := "step-2"
	outputID := "output-id"
	responseSteps := meta.ResponseOps{
		{ID: stepID},
		{ID: fromStepID},
	}

	root := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{
			StartTime: time.UnixMilli(1),
		},
		Attributes: &meta.ExtractedValues{
			ResponseSteps: &responseSteps,
		},
		Children: []*cqrs.OtelSpan{
			{
				Attributes: &meta.ExtractedValues{
					StepID: &stepID,
				},
				Children: []*cqrs.OtelSpan{
					{
						OutputID: &outputID,
					},
				},
			},
		},
	}

	stack, stepSpans, found := reconstructStack(root, fromStepID)

	require.True(t, found)
	require.Equal(t, []string{stepID, fromStepID}, stack)
	require.Same(t, root.Children[0].Children[0], stepSpans[stepID])
}

func TestFindOutputSpanUsesLatestDescendantOutput(t *testing.T) {
	firstOutputID := "first-output-id"
	latestOutputID := "latest-output-id"
	span := &cqrs.OtelSpan{
		Children: []*cqrs.OtelSpan{
			{OutputID: &firstOutputID},
			{OutputID: &latestOutputID},
		},
	}

	require.Same(t, span.Children[1], findOutputSpan(span))
}

func TestReconstructMemoizesSleepWithoutOutput(t *testing.T) {
	sleepStepID := "sleep-step"
	fromStepID := "from-step"
	runID := ulid.Make()
	stepOp := enums.OpcodeSleep
	responseSteps := meta.ResponseOps{
		{ID: sleepStepID},
		{ID: fromStepID},
	}

	root := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{
			StartTime: time.UnixMilli(1),
		},
		Attributes: &meta.ExtractedValues{
			ResponseSteps: &responseSteps,
		},
		Children: []*cqrs.OtelSpan{
			{
				Attributes: &meta.ExtractedValues{
					StepID: &sleepStepID,
					StepOp: &stepOp,
				},
			},
		},
	}

	newState := &sv2.CreateState{}
	err := reconstruct(context.Background(), fakeReconstructTraceReader{root: root}, execution.ScheduleRequest{
		OriginalRunID: &runID,
		FromStep: &execution.ScheduleRequestFromStep{
			StepID: fromStepID,
		},
	}, newState)

	require.NoError(t, err)
	require.Len(t, newState.Steps, 1)
	require.Equal(t, sleepStepID, newState.Steps[0].ID)
	require.Nil(t, newState.Steps[0].Data)
}

type fakeReconstructTraceReader struct {
	root *cqrs.OtelSpan
}

func (f fakeReconstructTraceReader) GetTraceRuns(context.Context, cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
	return nil, errors.New("unexpected GetTraceRuns")
}

func (f fakeReconstructTraceReader) GetTraceRunsCount(context.Context, cqrs.GetTraceRunOpt) (int, error) {
	return 0, errors.New("unexpected GetTraceRunsCount")
}

func (f fakeReconstructTraceReader) GetTraceRun(context.Context, cqrs.TraceRunIdentifier) (*cqrs.TraceRun, error) {
	return nil, errors.New("unexpected GetTraceRun")
}

func (f fakeReconstructTraceReader) GetTraceSpansByRun(context.Context, cqrs.TraceRunIdentifier) ([]*cqrs.Span, error) {
	return nil, errors.New("unexpected GetTraceSpansByRun")
}

func (f fakeReconstructTraceReader) LegacyGetSpanOutput(context.Context, cqrs.SpanIdentifier) (*cqrs.SpanOutput, error) {
	return nil, errors.New("unexpected LegacyGetSpanOutput")
}

func (f fakeReconstructTraceReader) GetSpanStack(context.Context, cqrs.SpanIdentifier) ([]string, error) {
	return nil, errors.New("unexpected GetSpanStack")
}

func (f fakeReconstructTraceReader) GetSpansByRunID(context.Context, ulid.ULID) (*cqrs.OtelSpan, error) {
	return f.root, nil
}

func (f fakeReconstructTraceReader) GetSpansByDebugRunID(context.Context, ulid.ULID) ([]*cqrs.OtelSpan, error) {
	return nil, errors.New("unexpected GetSpansByDebugRunID")
}

func (f fakeReconstructTraceReader) GetSpansByDebugSessionID(context.Context, ulid.ULID) ([][]*cqrs.OtelSpan, error) {
	return nil, errors.New("unexpected GetSpansByDebugSessionID")
}

func (f fakeReconstructTraceReader) GetSpanOutput(context.Context, cqrs.SpanIdentifier) (*cqrs.SpanOutput, error) {
	return nil, errors.New("unexpected GetSpanOutput")
}

func (f fakeReconstructTraceReader) GetRunSpanByRunID(context.Context, ulid.ULID, uuid.UUID, uuid.UUID) (*cqrs.OtelSpan, error) {
	return nil, errors.New("unexpected GetRunSpanByRunID")
}

func (f fakeReconstructTraceReader) GetStepSpanByStepID(context.Context, ulid.ULID, string, uuid.UUID, uuid.UUID) (*cqrs.OtelSpan, error) {
	return nil, errors.New("unexpected GetStepSpanByStepID")
}

func (f fakeReconstructTraceReader) GetExecutionSpanByStepIDAndAttempt(context.Context, ulid.ULID, string, int, uuid.UUID, uuid.UUID) (*cqrs.OtelSpan, error) {
	return nil, errors.New("unexpected GetExecutionSpanByStepIDAndAttempt")
}

func (f fakeReconstructTraceReader) GetLatestExecutionSpanByStepID(context.Context, ulid.ULID, string, uuid.UUID, uuid.UUID) (*cqrs.OtelSpan, error) {
	return nil, errors.New("unexpected GetLatestExecutionSpanByStepID")
}

func (f fakeReconstructTraceReader) GetSpanBySpanID(context.Context, ulid.ULID, string, uuid.UUID, uuid.UUID) (*cqrs.OtelSpan, error) {
	return nil, errors.New("unexpected GetSpanBySpanID")
}

func (f fakeReconstructTraceReader) OtelTracesEnabled(context.Context, uuid.UUID) (bool, error) {
	return false, errors.New("unexpected OtelTracesEnabled")
}

func (f fakeReconstructTraceReader) GetEventRuns(context.Context, ulid.ULID, uuid.UUID, uuid.UUID) ([]*cqrs.FunctionRun, error) {
	return nil, errors.New("unexpected GetEventRuns")
}

func (f fakeReconstructTraceReader) GetRun(context.Context, ulid.ULID, uuid.UUID, uuid.UUID) (*cqrs.FunctionRun, error) {
	return nil, errors.New("unexpected GetRun")
}

func (f fakeReconstructTraceReader) GetEvent(context.Context, ulid.ULID, uuid.UUID, uuid.UUID) (*cqrs.Event, error) {
	return nil, errors.New("unexpected GetEvent")
}

func (f fakeReconstructTraceReader) GetEvents(context.Context, uuid.UUID, uuid.UUID, *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
	return nil, errors.New("unexpected GetEvents")
}
