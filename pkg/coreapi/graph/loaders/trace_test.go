package loader

import (
	"context"
	"testing"
	"time"

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

func boolPtr(b bool) *bool                               { return &b }
func strPtr(s string) *string                            { return &s }
func opcodePtr(o enums.Opcode) *enums.Opcode             { return &o }
func stepStatusPtr(s enums.StepStatus) *enums.StepStatus { return &s }
func durationPtr(d time.Duration) *time.Duration         { return &d }

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

// TestConvertRunSpanToGQL_DroppedChildrenSkipped verifies that MarkedAsDropped
// children are filtered from a parent's ChildrenSpans. This prevents a bug
// where step.sleep (and other discovery-response steps) showed a phantom
// "Attempt 0" from a leaked execution span that was exported to the DB before
// its DropSpan attribute was set via an EXTEND span.
func TestConvertRunSpanToGQL_DroppedChildrenSkipped(t *testing.T) {
	tr := &traceReader{}
	ctx := context.Background()

	// Builds the span tree that reproduces the sleep 2-attempt bug:
	// A SpanNameStepDiscovery parent with two children:
	//   - Child 0: execution span (MarkedAsDropped) - the leaked "scheduler" span
	//   - Child 1: step span with sleep info - the actual sleep
	buildSleepDiscoverySpan := func(sleepCompleted bool) *cqrs.OtelSpan {
		stepChild := &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStep},
			Attributes: &meta.ExtractedValues{
				StepOp:            opcodePtr(enums.OpcodeSleep),
				StepSleepDuration: durationPtr(5 * time.Second),
				StepName:          strPtr("wait"),
			},
		}
		if sleepCompleted {
			stepChild.Attributes.DynamicStatus = stepStatusPtr(enums.StepStatusCompleted)
			now := time.Now()
			stepChild.Attributes.EndedAt = &now
			started := now.Add(-5 * time.Second)
			stepChild.Attributes.StartedAt = &started
		}

		return &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStepDiscovery},
			Attributes:  &meta.ExtractedValues{},
			Children: []*cqrs.OtelSpan{
				{
					// The leaked execution span - exported before DropSpan
					// was set, then merged with its EXTEND to get
					// MarkedAsDropped=true.
					RawOtelSpan:     cqrs.RawOtelSpan{Name: meta.SpanNameExecution},
					Attributes:      &meta.ExtractedValues{},
					MarkedAsDropped: true,
				},
				stepChild,
			},
		}
	}

	t.Run("dropped execution child is skipped, completed sleep collapses", func(t *testing.T) {
		span := buildSleepDiscoverySpan(true)

		result, err := tr.convertRunSpanToGQL(ctx, span)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.False(t, result.Omit, "discovery span with a non-dropped child should not be omitted")
		assert.Empty(t, result.ChildrenSpans, "completed sleep should collapse to zero children")

		require.NotNil(t, result.StepOp, "StepOp should be propagated from child")
		assert.Equal(t, models.StepOpSleep, *result.StepOp)
	})

	t.Run("dropped execution child is skipped, running sleep shows 1 child", func(t *testing.T) {
		span := buildSleepDiscoverySpan(false)

		result, err := tr.convertRunSpanToGQL(ctx, span)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.False(t, result.Omit)
		require.Len(t, result.ChildrenSpans, 1, "running sleep should have exactly 1 visible child")
		assert.Equal(t, "Attempt 0", result.ChildrenSpans[0].Name)
	})

	t.Run("all children dropped causes parent to be omitted", func(t *testing.T) {
		span := &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{Name: meta.SpanNameStepDiscovery},
			Attributes:  &meta.ExtractedValues{},
			Children: []*cqrs.OtelSpan{
				{
					RawOtelSpan:     cqrs.RawOtelSpan{Name: meta.SpanNameExecution},
					Attributes:      &meta.ExtractedValues{},
					MarkedAsDropped: true,
				},
			},
		}

		result, err := tr.convertRunSpanToGQL(ctx, span)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.True(t, result.Omit, "discovery span with only dropped children should be omitted")
	})
}
