package loader

import (
	"context"
	"testing"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
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

	t.Run("userland span with children is collapsed", func(t *testing.T) {
		span := &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameExecution},
			Attributes:  &meta.ExtractedValues{},
			Children: []*cqrs.OtelSpan{
				{
					RawOtelSpan: cqrs.RawOtelSpan{Name: "inngest.execution"},
					Attributes: &meta.ExtractedValues{
						IsUserland:   boolPtr(true),
						UserlandName: strPtr("inngest.execution"),
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
}
