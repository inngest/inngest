package checkpoint

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/util/interval"
	"github.com/oklog/ulid/v2"
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

func TestStepRunAttrs(t *testing.T) {
	runID := ulid.Make()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	timing := interval.Interval{A: start.UnixNano(), B: int64(5 * time.Second)}
	name := "my-step"

	baseOp := func() state.GeneratorOpcode {
		return state.GeneratorOpcode{
			Op:          enums.OpcodeStepRun,
			ID:          "step-id",
			DisplayName: &name,
			Timing:      timing,
		}
	}

	t.Run("plain StepRun sets QueuedAt and StartedAt and is not paired-trailing", func(t *testing.T) {
		op := baseOp()
		attrs := stepRunAttrs(meta.NewAttrSet(), op, runID)

		_, isPaired := meta.GetBoolFlag(attrs, meta.Attrs.IsPairedTrailing)
		require.False(t, isPaired, "plain StepRun must not be flagged paired-trailing")

		qa, ok := attrs.Get(meta.Attrs.QueuedAt.Key()).(*time.Time)
		require.True(t, ok, "QueuedAt must be set for a plain StepRun")
		require.Equal(t, op.Timing.Start().UnixNano(), qa.UnixNano())

		sa, ok := attrs.Get(meta.Attrs.StartedAt.Key()).(*time.Time)
		require.True(t, ok, "StartedAt must be set for a plain StepRun")
		require.Equal(t, op.Timing.Start().UnixNano(), sa.UnixNano())

		// The common attributes are merged regardless of the paired-trailing branch.
		sn, ok := attrs.Get(meta.Attrs.StepName.Key()).(*string)
		require.True(t, ok)
		require.Equal(t, name, *sn)
	})

	t.Run("paired-trailing StepRun omits QueuedAt and StartedAt", func(t *testing.T) {
		op := baseOp()
		op.Opts = map[string]any{"_paired_trailing": true}
		attrs := stepRunAttrs(meta.NewAttrSet(), op, runID)

		val, ok := meta.GetBoolFlag(attrs, meta.Attrs.IsPairedTrailing)
		require.True(t, ok)
		require.True(t, val)

		// Omitting these is the whole point: the trailing arm must not clobber
		// the timing the leading StepPlanned already wrote to the shared span.
		require.Nil(t, attrs.Get(meta.Attrs.QueuedAt.Key()),
			"QueuedAt must be omitted so the leading arm's value survives")
		require.Nil(t, attrs.Get(meta.Attrs.StartedAt.Key()),
			"StartedAt must be omitted so the leading arm's value survives")

		// The common attributes are still merged.
		sn, ok := attrs.Get(meta.Attrs.StepName.Key()).(*string)
		require.True(t, ok)
		require.Equal(t, name, *sn)
	})
}
