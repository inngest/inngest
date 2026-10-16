package enums

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpandReplayRunStatuses(t *testing.T) {
	for _, tc := range []struct {
		name      string
		selectors []ReplayRunStatus
		statuses  []RunStatus
		reasons   []SkipReason
		invalid   bool
	}{
		{name: "empty"},
		{name: "all", selectors: []ReplayRunStatus{ReplayRunStatusAll}, statuses: ReplayableFunctionRunStatuses(), reasons: ReplayableSkipReasons()},
		{name: "completed", selectors: []ReplayRunStatus{ReplayRunStatusCompleted}, statuses: []RunStatus{RunStatusCompleted}},
		{name: "failed", selectors: []ReplayRunStatus{ReplayRunStatusFailed}, statuses: []RunStatus{RunStatusFailed}},
		{name: "cancelled", selectors: []ReplayRunStatus{ReplayRunStatusCancelled}, statuses: []RunStatus{RunStatusCancelled}},
		{name: "paused", selectors: []ReplayRunStatus{ReplayRunStatusSkippedPaused}, reasons: []SkipReason{SkipReasonFunctionPaused}},
		{name: "mixed", selectors: []ReplayRunStatus{ReplayRunStatusFailed, ReplayRunStatusSkippedPaused}, statuses: []RunStatus{RunStatusFailed}, reasons: []SkipReason{SkipReasonFunctionPaused}},
		{name: "all with another selector", selectors: []ReplayRunStatus{ReplayRunStatusCompleted, ReplayRunStatusAll}, statuses: ReplayableFunctionRunStatuses(), reasons: ReplayableSkipReasons()},
		{name: "unknown", selectors: []ReplayRunStatus{ReplayRunStatus(-1)}, invalid: true},
		{name: "unknown after all", selectors: []ReplayRunStatus{ReplayRunStatusAll, ReplayRunStatus(100)}, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			statuses, reasons, err := ExpandReplayRunStatuses(tc.selectors)
			if tc.invalid {
				require.ErrorContains(t, err, "unknown replay run status")
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.statuses, statuses)
			require.Equal(t, tc.reasons, reasons)
		})
	}
}
