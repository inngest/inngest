package loader

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/enums"
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

func TestFilterSleepSchedulingAttempts(t *testing.T) {
	sleepOp := models.StepOpSleep
	now := time.Now().UTC()

	t.Run("filters transient 206 scheduling attempt when follow-up sleep attempt exists", func(t *testing.T) {
		step := &models.RunTraceSpan{
			StepOp: &sleepOp,
			ChildrenSpans: []*models.RunTraceSpan{
				{
					StepOp:    &sleepOp,
					StartedAt: &now,
					EndedAt:   &now,
					Response: &models.RunTraceSpanResponseInfo{
						StatusCode: 206,
					},
				},
				{
					StepOp:    &sleepOp,
					StartedAt: &now,
					EndedAt:   ptrTime(now.Add(time.Second)),
				},
			},
		}

		filterSleepSchedulingAttempts(step)
		require.Len(t, step.ChildrenSpans, 1)
		assert.Nil(t, step.ChildrenSpans[0].Response)
	})

	t.Run("keeps single sleep attempt with 206 response", func(t *testing.T) {
		step := &models.RunTraceSpan{
			StepOp: &sleepOp,
			ChildrenSpans: []*models.RunTraceSpan{
				{
					StepOp:    &sleepOp,
					StartedAt: &now,
					EndedAt:   &now,
					Response: &models.RunTraceSpanResponseInfo{
						StatusCode: 206,
					},
				},
			},
		}

		filterSleepSchedulingAttempts(step)
		require.Len(t, step.ChildrenSpans, 1)
		assert.Equal(t, 206, step.ChildrenSpans[0].Response.StatusCode)
	})
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
