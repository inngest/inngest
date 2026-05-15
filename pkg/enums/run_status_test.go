package enums

import (
	"testing"
)

func TestStepStatusToRunStatus(t *testing.T) {
	tests := []struct {
		name       string
		stepStatus StepStatus
		want       RunStatus
	}{
		{
			name:       "StepStatusQueued should map to RunStatusScheduled",
			stepStatus: StepStatusQueued,
			want:       RunStatusScheduled,
		},
		{
			name:       "StepStatusScheduled should map to RunStatusScheduled",
			stepStatus: StepStatusScheduled,
			want:       RunStatusScheduled,
		},
		{
			name:       "StepStatusRunning should map to RunStatusRunning",
			stepStatus: StepStatusRunning,
			want:       RunStatusRunning,
		},
		{
			name:       "StepStatusWaiting should map to RunStatusRunning",
			stepStatus: StepStatusWaiting,
			want:       RunStatusRunning,
		},
		{
			name:       "StepStatusSleeping should map to RunStatusRunning",
			stepStatus: StepStatusSleeping,
			want:       RunStatusRunning,
		},
		{
			name:       "StepStatusInvoking should map to RunStatusRunning",
			stepStatus: StepStatusInvoking,
			want:       RunStatusRunning,
		},
		{
			name:       "StepStatusCompleted should map to RunStatusCompleted",
			stepStatus: StepStatusCompleted,
			want:       RunStatusCompleted,
		},
		{
			name:       "StepStatusFailed should map to RunStatusFailed",
			stepStatus: StepStatusFailed,
			want:       RunStatusFailed,
		},
		{
			name:       "StepStatusErrored should map to RunStatusFailed",
			stepStatus: StepStatusErrored,
			want:       RunStatusFailed,
		},
		{
			name:       "StepStatusCancelled should map to RunStatusCancelled",
			stepStatus: StepStatusCancelled,
			want:       RunStatusCancelled,
		},
		{
			name:       "StepStatusTimedOut should map to RunStatusCancelled",
			stepStatus: StepStatusTimedOut,
			want:       RunStatusCancelled,
		},
		{
			name:       "StepStatusSkipped should map to RunStatusSkipped",
			stepStatus: StepStatusSkipped,
			want:       RunStatusSkipped,
		},
		{
			name:       "StepStatusUnknown should map to RunStatusUnknown",
			stepStatus: StepStatusUnknown,
			want:       RunStatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StepStatusToRunStatus(tt.stepStatus)
			if got != tt.want {
				t.Errorf("StepStatusToRunStatus(%v) = %v, want %v", tt.stepStatus, got, tt.want)
			}
		})
	}
}
