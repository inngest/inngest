package loader

import (
	"context"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepStatusToGQL(t *testing.T) {
	tr := &traceReader{}

	tests := []struct {
		name     string
		input    enums.StepStatus
		expected models.RunTraceSpanStatus
	}{
		{"Running", enums.StepStatusRunning, models.RunTraceSpanStatusRunning},
		{"Invoking", enums.StepStatusInvoking, models.RunTraceSpanStatusRunning},
		{"Completed", enums.StepStatusCompleted, models.RunTraceSpanStatusCompleted},
		{"TimedOut", enums.StepStatusTimedOut, models.RunTraceSpanStatusCompleted},
		{"Failed", enums.StepStatusFailed, models.RunTraceSpanStatusFailed},
		{"Errored", enums.StepStatusErrored, models.RunTraceSpanStatusFailed},
		{"Cancelled", enums.StepStatusCancelled, models.RunTraceSpanStatusCancelled},
		{"Scheduled", enums.StepStatusScheduled, models.RunTraceSpanStatusQueued},
		{"Queued", enums.StepStatusQueued, models.RunTraceSpanStatusQueued},
		{"Sleeping", enums.StepStatusSleeping, models.RunTraceSpanStatusWaiting},
		{"Waiting", enums.StepStatusWaiting, models.RunTraceSpanStatusWaiting},
		{"Skipped", enums.StepStatusSkipped, models.RunTraceSpanStatusSkipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tt.input
			result := tr.stepStatusToGQL(&status)
			require.NotNil(t, result, "stepStatusToGQL should not return nil for %s", tt.name)
			assert.Equal(t, tt.expected, *result)
		})
	}

	t.Run("nil input", func(t *testing.T) {
		result := tr.stepStatusToGQL(nil)
		assert.Nil(t, result)
	})
}

func TestRunTraceEnded(t *testing.T) {
	terminal := []models.RunTraceSpanStatus{
		models.RunTraceSpanStatusCompleted,
		models.RunTraceSpanStatusCancelled,
		models.RunTraceSpanStatusFailed,
		models.RunTraceSpanStatusSkipped,
	}
	for _, s := range terminal {
		assert.True(t, models.RunTraceEnded(s), "%s should be terminal", s)
	}

	nonTerminal := []models.RunTraceSpanStatus{
		models.RunTraceSpanStatusRunning,
		models.RunTraceSpanStatusQueued,
		models.RunTraceSpanStatusWaiting,
	}
	for _, s := range nonTerminal {
		assert.False(t, models.RunTraceEnded(s), "%s should not be terminal", s)
	}
}

func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

func TestConvertRunSpanToGQL_UserlandCollapse(t *testing.T) {
	tr := &traceReader{}
	ctx := context.Background()

	t.Run("leaf userland span is preserved", func(t *testing.T) {
		span := &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameExecution},
			Attributes:  &meta.ExtractedValues{},
			Children: []*cqrs.OtelSpan{
				{
					RawOtelSpan: cqrs.RawOtelSpan{Name: "GET"},
					Attributes: &meta.ExtractedValues{
						IsUserland:   boolPtr(true),
						UserlandName: strPtr("GET"),
					},
				},
			},
		}

		result, err := tr.convertRunSpanToGQL(ctx, span)
		require.NoError(t, err)
		require.Len(t, result.ChildrenSpans, 1, "leaf userland span should not be dropped")
		assert.True(t, result.ChildrenSpans[0].IsUserland)
		assert.Equal(t, "GET", result.ChildrenSpans[0].Name)
	})

	t.Run("SDK execution wrapper with children is collapsed", func(t *testing.T) {
		span := &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameExecution},
			Attributes:  &meta.ExtractedValues{},
			Children: []*cqrs.OtelSpan{
				{
					RawOtelSpan: cqrs.RawOtelSpan{Name: SDKExecutionSpanName},
					Attributes: &meta.ExtractedValues{
						IsUserland:   boolPtr(true),
						UserlandName: strPtr(SDKExecutionSpanName),
					},
					Children: []*cqrs.OtelSpan{
						{
							RawOtelSpan: cqrs.RawOtelSpan{Name: "GET"},
							Attributes: &meta.ExtractedValues{
								IsUserland:   boolPtr(true),
								UserlandName: strPtr("GET"),
							},
						},
					},
				},
			},
		}

		result, err := tr.convertRunSpanToGQL(ctx, span)
		require.NoError(t, err)
		require.Len(t, result.ChildrenSpans, 1, "should collapse to grandchild")
		assert.Equal(t, "GET", result.ChildrenSpans[0].Name)
	})

	t.Run("non-wrapper userland span with children is preserved", func(t *testing.T) {
		span := &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameExecution},
			Attributes:  &meta.ExtractedValues{},
			Children: []*cqrs.OtelSpan{
				{
					RawOtelSpan: cqrs.RawOtelSpan{Name: "my-span"},
					Attributes: &meta.ExtractedValues{
						IsUserland:   boolPtr(true),
						UserlandName: strPtr("my-span"),
					},
					Children: []*cqrs.OtelSpan{
						{
							RawOtelSpan: cqrs.RawOtelSpan{Name: "child-span"},
							Attributes: &meta.ExtractedValues{
								IsUserland:   boolPtr(true),
								UserlandName: strPtr("child-span"),
							},
						},
					},
				},
			},
		}

		result, err := tr.convertRunSpanToGQL(ctx, span)
		require.NoError(t, err)
		require.Len(t, result.ChildrenSpans, 1, "should keep the userland span")
		assert.Equal(t, "my-span", result.ChildrenSpans[0].Name)
		assert.True(t, result.ChildrenSpans[0].IsUserland)
		require.Len(t, result.ChildrenSpans[0].ChildrenSpans, 1, "should preserve children")
		assert.Equal(t, "child-span", result.ChildrenSpans[0].ChildrenSpans[0].Name)
	})
}

func TestConvertRunSpan(t *testing.T) {
	status := enums.StepStatusCompleted
	queuedAt := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)

	result, err := ConvertRunSpan(context.Background(), &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{
			Name:      meta.SpanNameRun,
			SpanID:    "run-span",
			TraceID:   "trace-id",
			StartTime: queuedAt,
			EndTime:   queuedAt.Add(time.Second),
		},
		Attributes: &meta.ExtractedValues{
			DynamicStatus: &status,
			QueuedAt:      &queuedAt,
		},
	})

	require.NoError(t, err)
	require.Equal(t, "run-span", result.SpanID)
	require.Equal(t, models.RunTraceSpanStatusCompleted, result.Status)
}

func TestConvertRunSpanToGQL_DroppedDiscoveryExecutionDoesNotCreateSleepAttempts(t *testing.T) {
	tr := &traceReader{}
	ctx := context.Background()

	status := enums.StepStatusCompleted
	stepOp := enums.OpcodeSleep
	sleepDuration := time.Second
	queuedAt := time.Date(2026, 5, 8, 15, 34, 39, 0, time.UTC)
	endedAt := queuedAt.Add(sleepDuration)
	stepID := "b101b7cef051778f3c4378e92fad02d5f90a527d"
	stepName := "wait-a-moment"

	result, err := tr.convertRunSpanToGQL(ctx, &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{
			Name:      meta.SpanNameRun,
			SpanID:    "run",
			TraceID:   "trace",
			StartTime: queuedAt,
			EndTime:   endedAt,
		},
		Attributes: &meta.ExtractedValues{
			DynamicStatus: &status,
			QueuedAt:      &queuedAt,
			StartedAt:     &queuedAt,
			EndedAt:       &endedAt,
		},
		Children: []*cqrs.OtelSpan{
			{
				RawOtelSpan: cqrs.RawOtelSpan{
					Name:      meta.SpanNameStepDiscovery,
					SpanID:    "discovery",
					TraceID:   "trace",
					StartTime: queuedAt,
					EndTime:   endedAt,
				},
				Attributes: &meta.ExtractedValues{
					QueuedAt:  &queuedAt,
					StartedAt: &queuedAt,
					EndedAt:   &endedAt,
				},
				Children: []*cqrs.OtelSpan{
					{
						RawOtelSpan: cqrs.RawOtelSpan{
							Name:      meta.SpanNameExecution,
							SpanID:    "discovery-execution",
							TraceID:   "trace",
							StartTime: queuedAt,
							EndTime:   queuedAt,
						},
						Attributes: &meta.ExtractedValues{
							DynamicStatus:     &status,
							StepOp:            &stepOp,
							StepID:            &stepID,
							StepName:          &stepName,
							StepSleepDuration: &sleepDuration,
							QueuedAt:          &queuedAt,
							StartedAt:         &queuedAt,
							EndedAt:           &queuedAt,
						},
						MarkedAsDropped: true,
					},
					{
						RawOtelSpan: cqrs.RawOtelSpan{
							Name:      meta.SpanNameStep,
							SpanID:    "sleep",
							TraceID:   "trace",
							StartTime: queuedAt,
							EndTime:   endedAt,
						},
						Attributes: &meta.ExtractedValues{
							DynamicStatus:     &status,
							StepOp:            &stepOp,
							StepID:            &stepID,
							StepName:          &stepName,
							StepSleepDuration: &sleepDuration,
							QueuedAt:          &queuedAt,
							StartedAt:         &queuedAt,
							EndedAt:           &endedAt,
						},
					},
				},
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.ChildrenSpans, 1)

	sleep := result.ChildrenSpans[0]
	assert.Equal(t, stepName, sleep.Name)
	assert.Equal(t, models.RunTraceSpanStatusCompleted, sleep.Status)
	assert.Empty(t, sleep.ChildrenSpans)
	require.NotNil(t, sleep.StepOp)
	assert.Equal(t, models.StepOpSleep, *sleep.StepOp)
	require.NotNil(t, sleep.StepID)
	assert.Equal(t, stepID, *sleep.StepID)

	info, ok := sleep.StepInfo.(*models.SleepStepInfo)
	require.True(t, ok)
	assert.Equal(t, endedAt, info.SleepUntil)
}

func TestConvertRunSpanToGQL_MetadataPromotion(t *testing.T) {
	tr := &traceReader{}
	ctx := context.Background()
	now := time.Now()

	mdA := &cqrs.SpanMetadata{
		Scope: enums.MetadataScopeStep,
		Kind:  metadata.Kind("inngest.timing"),
		Values: metadata.Values{
			"step_a": []byte(`"data_a"`),
		},
		UpdatedAt: now,
	}
	mdB := &cqrs.SpanMetadata{
		Scope: enums.MetadataScopeStep,
		Kind:  metadata.Kind("inngest.timing"),
		Values: metadata.Values{
			"step_b": []byte(`"data_b"`),
		},
		UpdatedAt: now,
	}

	completedStatus := enums.StepStatusCompleted
	stepOpRun := enums.OpcodeStepRun

	t.Run("single omitted discovery promotes metadata to following step", func(t *testing.T) {
		span := &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameRun},
			Attributes:  &meta.ExtractedValues{},
			Children: []*cqrs.OtelSpan{
				{
					// Omitted step discovery with metadata
					RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStepDiscovery},
					Attributes:  &meta.ExtractedValues{},
					Metadata:    []*cqrs.SpanMetadata{mdA},
				},
				{
					// Visible step
					RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStep},
					Attributes: &meta.ExtractedValues{
						DynamicStatus: &completedStatus,
						StepOp:        &stepOpRun,
						StepID:        strPtr("step-a"),
					},
				},
			},
		}

		result, err := tr.convertRunSpanToGQL(ctx, span)
		require.NoError(t, err)
		require.Len(t, result.ChildrenSpans, 1, "only the visible step child")
		assert.Len(t, result.ChildrenSpans[0].Metadata, 1, "step should have promoted metadata")
		assert.Equal(t, metadata.Kind("inngest.timing"), result.ChildrenSpans[0].Metadata[0].Kind)
	})

	t.Run("multi-step: each step gets only its own discovery metadata", func(t *testing.T) {
		span := &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameRun},
			Attributes:  &meta.ExtractedValues{},
			Children: []*cqrs.OtelSpan{
				{
					// Omitted step discovery for step A
					RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStepDiscovery},
					Attributes:  &meta.ExtractedValues{},
					Metadata:    []*cqrs.SpanMetadata{mdA},
				},
				{
					// Visible step A
					RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStep},
					Attributes: &meta.ExtractedValues{
						DynamicStatus: &completedStatus,
						StepOp:        &stepOpRun,
						StepID:        strPtr("step-a"),
					},
				},
				{
					// Omitted step discovery for step B
					RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStepDiscovery},
					Attributes:  &meta.ExtractedValues{},
					Metadata:    []*cqrs.SpanMetadata{mdB},
				},
				{
					// Visible step B
					RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStep},
					Attributes: &meta.ExtractedValues{
						DynamicStatus: &completedStatus,
						StepOp:        &stepOpRun,
						StepID:        strPtr("step-b"),
					},
				},
			},
		}

		result, err := tr.convertRunSpanToGQL(ctx, span)
		require.NoError(t, err)
		require.Len(t, result.ChildrenSpans, 2, "two visible step children")

		// Step A should only have mdA
		stepA := result.ChildrenSpans[0]
		require.Len(t, stepA.Metadata, 1, "step A should have exactly one metadata entry")
		assert.Contains(t, stepA.Metadata[0].Values, "step_a")
		assert.NotContains(t, stepA.Metadata[0].Values, "step_b")

		// Step B should only have mdB
		stepB := result.ChildrenSpans[1]
		require.Len(t, stepB.Metadata, 1, "step B should have exactly one metadata entry")
		assert.Contains(t, stepB.Metadata[0].Values, "step_b")
		assert.NotContains(t, stepB.Metadata[0].Values, "step_a")
	})

	t.Run("visible discovery span does not absorb omitted discovery metadata", func(t *testing.T) {
		// A StepDiscovery span becomes visible when it has non-dropped
		// children. If it appears between omitted discoveries, it must
		// NOT receive their promoted metadata — only StepSpans should.
		executionStatus := enums.StepStatusCompleted
		executionOp := enums.OpcodeStepRun
		span := &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameRun},
			Attributes:  &meta.ExtractedValues{},
			Children: []*cqrs.OtelSpan{
				{
					// Omitted discovery with metadata
					RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStepDiscovery},
					Attributes:  &meta.ExtractedValues{},
					Metadata:    []*cqrs.SpanMetadata{mdA},
				},
				{
					// Visible discovery (has a non-dropped child execution)
					RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStepDiscovery},
					Attributes:  &meta.ExtractedValues{},
					Children: []*cqrs.OtelSpan{
						{
							RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameExecution},
							Attributes: &meta.ExtractedValues{
								DynamicStatus: &executionStatus,
								StepOp:        &executionOp,
								StepID:        strPtr("visible-discovery"),
							},
						},
					},
				},
				{
					// Visible step that should receive mdA
					RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStep},
					Attributes: &meta.ExtractedValues{
						DynamicStatus: &completedStatus,
						StepOp:        &stepOpRun,
						StepID:        strPtr("step-a"),
					},
				},
			},
		}

		result, err := tr.convertRunSpanToGQL(ctx, span)
		require.NoError(t, err)

		// Find the visible step span (not the discovery)
		var stepSpan *models.RunTraceSpan
		var discoverySpan *models.RunTraceSpan
		for _, child := range result.ChildrenSpans {
			if child.SpanTypeName == meta.SpanNameStep {
				stepSpan = child
			}
			if child.SpanTypeName == meta.SpanNameStepDiscovery {
				discoverySpan = child
			}
		}

		require.NotNil(t, stepSpan, "should have a visible step span")
		assert.Len(t, stepSpan.Metadata, 1, "step should receive promoted metadata from omitted discovery")
		assert.Contains(t, stepSpan.Metadata[0].Values, "step_a")

		if discoverySpan != nil {
			assert.Empty(t, discoverySpan.Metadata, "visible discovery span must not absorb omitted discovery metadata")
		}
	})

	t.Run("trailing omitted discovery with no following step discards metadata", func(t *testing.T) {
		span := &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameRun},
			Attributes:  &meta.ExtractedValues{},
			Children: []*cqrs.OtelSpan{
				{
					// Visible step
					RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStep},
					Attributes: &meta.ExtractedValues{
						DynamicStatus: &completedStatus,
						StepOp:        &stepOpRun,
						StepID:        strPtr("step-a"),
					},
				},
				{
					// Trailing omitted step discovery (no following step)
					RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStepDiscovery},
					Attributes:  &meta.ExtractedValues{},
					Metadata:    []*cqrs.SpanMetadata{mdB},
				},
			},
		}

		result, err := tr.convertRunSpanToGQL(ctx, span)
		require.NoError(t, err)
		require.Len(t, result.ChildrenSpans, 1, "only the visible step")
		assert.Empty(t, result.ChildrenSpans[0].Metadata, "step should not have trailing discovery metadata")
	})
}
