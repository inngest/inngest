package checkpoint

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

func TestIsPairedTrailingStepRun(t *testing.T) {
	tests := []struct {
		name string
		op   state.GeneratorOpcode
		want bool
	}{
		{
			name: "non-StepRun opcode is never paired-trailing",
			op: state.GeneratorOpcode{
				Op:   enums.OpcodeStepPlanned,
				Opts: map[string]any{"_paired_trailing": true},
			},
			want: false,
		},
		{
			name: "StepRun with nil opts",
			op:   state.GeneratorOpcode{Op: enums.OpcodeStepRun},
			want: false,
		},
		{
			name: "StepRun with non-map opts",
			op: state.GeneratorOpcode{
				Op:   enums.OpcodeStepRun,
				Opts: "not-a-map",
			},
			want: false,
		},
		{
			name: "StepRun with map opts missing the flag",
			op: state.GeneratorOpcode{
				Op:   enums.OpcodeStepRun,
				Opts: map[string]any{"other": true},
			},
			want: false,
		},
		{
			name: "StepRun with flag set to false",
			op: state.GeneratorOpcode{
				Op:   enums.OpcodeStepRun,
				Opts: map[string]any{"_paired_trailing": false},
			},
			want: false,
		},
		{
			name: "StepRun with flag as a string is the wrong type",
			op: state.GeneratorOpcode{
				Op:   enums.OpcodeStepRun,
				Opts: map[string]any{"_paired_trailing": "true"},
			},
			want: false,
		},
		{
			name: "StepRun with flag set to true",
			op: state.GeneratorOpcode{
				Op:   enums.OpcodeStepRun,
				Opts: map[string]any{"_paired_trailing": true},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isPairedTrailingStepRun(tc.op))
		})
	}
}

// TestIsPairedTrailingStepRun_WireShape guards the realistic path: when the
// opcode is decoded from the JSON an SDK actually sends, the opts object
// decodes to map[string]any and the flag decodes to a Go bool. If either shape
// drifts, the type assertions in isPairedTrailingStepRun silently return false
// and the flag is never honored.
func TestIsPairedTrailingStepRun_WireShape(t *testing.T) {
	var op state.GeneratorOpcode
	require.NoError(t, json.Unmarshal([]byte(`{
		"op": "StepRun",
		"id": "step-id",
		"opts": {"_paired_trailing": true}
	}`), &op))

	require.Equal(t, enums.OpcodeStepRun, op.Op)
	require.True(t, isPairedTrailingStepRun(op))
}
