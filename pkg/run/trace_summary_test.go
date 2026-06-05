package run

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

func TestOtelSpanStatusPrefersDynamicStatus(t *testing.T) {
	dynamic := enums.StepStatusCompleted

	status := OtelSpanStatus(&cqrs.OtelSpan{
		Status: enums.StepStatusRunning,
		Attributes: &meta.ExtractedValues{
			DynamicStatus: &dynamic,
		},
	})

	require.NotNil(t, status)
	assert.Equal(t, enums.StepStatusCompleted, *status)
}

func TestShouldHydrateRunSummary(t *testing.T) {
	endedAt := time.Now()

	tests := []struct {
		name          string
		state         RunSummaryState
		includeOutput bool
		want          bool
	}{
		{
			name:          "missing requested output",
			state:         RunSummaryState{Status: enums.RunStatusCompleted, EndedAt: &endedAt},
			includeOutput: true,
			want:          true,
		},
		{
			name:  "missing ended at",
			state: RunSummaryState{Status: enums.RunStatusCompleted},
			want:  true,
		},
		{
			name:  "incomplete status",
			state: RunSummaryState{Status: enums.RunStatusScheduled, EndedAt: &endedAt, HasOutput: true},
			want:  true,
		},
		{
			name:          "complete state",
			state:         RunSummaryState{Status: enums.RunStatusCompleted, EndedAt: &endedAt, HasOutput: true},
			includeOutput: true,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShouldHydrateRunSummary(tt.state, tt.includeOutput))
		})
	}
}

func TestSpanSummaryApplyTo(t *testing.T) {
	startedAt := time.Date(2026, 6, 5, 13, 47, 41, 688000000, time.UTC)
	endedAt := startedAt.Add(time.Second)
	status := enums.RunStatusCompleted

	state := (&SpanSummary{
		Status:    &status,
		StartedAt: &startedAt,
		EndedAt:   &endedAt,
	}).ApplyTo(RunSummaryState{
		Status:    enums.RunStatusScheduled,
		StartedAt: startedAt.Add(-time.Second),
	})

	assert.Equal(t, enums.RunStatusCompleted, state.Status)
	assert.Equal(t, startedAt, state.StartedAt)
	require.NotNil(t, state.EndedAt)
	assert.Equal(t, endedAt, *state.EndedAt)
}

func TestSummarizeSpanTreeUsesFunctionOutputSpan(t *testing.T) {
	startedAt := time.Date(2026, 6, 5, 13, 47, 41, 688000000, time.UTC)
	endedAt := startedAt.Add(1044 * time.Millisecond)
	outputID := "encoded-output-id"

	summary, err := SummarizeSpanTree(&cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{
			Name:      meta.SpanNameRun,
			StartTime: startedAt.Add(-time.Millisecond),
		},
		Status: enums.StepStatusQueued,
		Children: []*cqrs.OtelSpan{
			{
				RawOtelSpan: cqrs.RawOtelSpan{
					Name:      meta.SpanNameExecution,
					StartTime: startedAt,
					EndTime:   endedAt,
				},
				Status:   enums.StepStatusCompleted,
				OutputID: &outputID,
				Attributes: &meta.ExtractedValues{
					IsFunctionOutput: boolPtr(true),
				},
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, summary)
	require.NotNil(t, summary.Status)
	assert.Equal(t, enums.RunStatusCompleted, *summary.Status)
	require.NotNil(t, summary.StartedAt)
	assert.Equal(t, startedAt, *summary.StartedAt)
	require.NotNil(t, summary.EndedAt)
	assert.Equal(t, endedAt, *summary.EndedAt)
	require.NotNil(t, summary.DurationMs)
	assert.Equal(t, uint64(1044), *summary.DurationMs)
	require.NotNil(t, summary.OutputID)
	assert.Equal(t, outputID, *summary.OutputID)
}
