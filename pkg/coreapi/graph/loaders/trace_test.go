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

func boolPtr(b bool) *bool     { return &b }
func strPtr(s string) *string  { return &s }

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
