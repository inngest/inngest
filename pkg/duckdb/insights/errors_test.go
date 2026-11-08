package insights

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserFacingExecutionMessage(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{
			name: "truncated macro args in LINE context",
			in:   "executing query: duckdb: statement failed: Could not convert string 'r1' to INT32 when casting from source column run_id\n\nLINE 1: SELECT run_id::INTEGER FROM inngest.insights_runs('00000000-0000-4000...\n                     ^^^^^^^^^",
			want: "executing query: duckdb: statement failed: Could not convert string 'r1' to INT32 when casting from source column run_id",
		},
		{
			name: "LINE context starting mid-args",
			in:   "Could not convert string 'Running' to INT32 when casting from source column status\n\nLINE 1: ...', '00000000-0000-4000-b000-000000000000') AS runs WHERE status = 1 LIMIT 1000\n                                                                    ^^^^^^",
			want: "Could not convert string 'Running' to INT32 when casting from source column status",
		},
		{
			name: "macro w/ full args in body",
			in:   `Table "inngest.insights_step_attempts('a', 'b')" has no column "x"; see inngest.insights_steps(?, ?) and insights_extended_trace_spans`,
			want: `Table "step_attempts" has no column "x"; see steps and extended_trace_spans`,
		},
		{
			name: "truncated macro in body",
			in:   "near inngest.insights_events('00000000-0000...",
			want: "near events",
		},
		{
			name: "candidate list survives",
			in:   "No function matches 'length(TIMESTAMP WITH TIME ZONE)'.\n\tCandidate functions:\n\tlength(col0 VARCHAR) -> BIGINT\n\n\nLINE 1: SELECT length(queued_at) FROM inngest.insights_runs('00000000-0000...\n               ^^^^^^^^^^^^^^^^^",
			want: "No function matches 'length(TIMESTAMP WITH TIME ZONE)'.\n\tCandidate functions:\n\tlength(col0 VARCHAR) -> BIGINT",
		},
		{
			name: "unrelated text untouched",
			in:   `column "insights_runs_count" must appear in the GROUP BY clause`,
			want: `column "insights_runs_count" must appear in the GROUP BY clause`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, UserFacingExecutionMessage(tc.in))
		})
	}
}
