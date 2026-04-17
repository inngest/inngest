package state

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratorOpcode_StepType(t *testing.T) {
	t.Run("RunType takes precedence over Op", func(t *testing.T) {
		// OpcodeStepRun with a RunType should return the RunType-derived step type,
		// not StepTypeRun.
		op := GeneratorOpcode{
			Op:   enums.OpcodeStepRun,
			Opts: map[string]any{"type": "step.sendEvent"},
		}
		assert.Equal(t, enums.StepTypeSendEvent, op.StepType())
	})

	t.Run("by RunType", func(t *testing.T) {
		cases := []struct {
			runType  string
			expected enums.StepType
		}{
			{"step.sendEvent", enums.StepTypeSendEvent},
			{"step.sendSignal", enums.StepTypeSendSignal},
			{"step.ai.wrap", enums.StepTypeAiWrap},
			{"step.ai.infer", enums.StepTypeAiInfer},
			{"step.fetch", enums.StepTypeFetch},
			{"step.realtime.publish", enums.StepTypeRealtimePublish},
			{"group.experiment", enums.StepTypeGroupExperiment},
		}
		for _, tc := range cases {
			t.Run(tc.runType, func(t *testing.T) {
				op := GeneratorOpcode{
					Op:   enums.OpcodeStepRun,
					Opts: map[string]any{"type": tc.runType},
				}
				assert.Equal(t, tc.expected, op.StepType())
			})
		}
	})

	t.Run("by Op when no RunType", func(t *testing.T) {
		cases := []struct {
			op       enums.Opcode
			expected enums.StepType
		}{
			{enums.OpcodeStepRun, enums.StepTypeRun},
			{enums.OpcodeStepError, enums.StepTypeRun},
			{enums.OpcodeStepFailed, enums.StepTypeRun},
			{enums.OpcodeSleep, enums.StepTypeSleep},
			{enums.OpcodeWaitForEvent, enums.StepTypeWaitForEvent},
			{enums.OpcodeWaitForSignal, enums.StepTypeWaitForSignal},
			{enums.OpcodeInvokeFunction, enums.StepTypeInvoke},
			{enums.OpcodeAIGateway, enums.StepTypeAiInfer},
		}
		for _, tc := range cases {
			t.Run(tc.op.String(), func(t *testing.T) {
				op := GeneratorOpcode{Op: tc.op}
				assert.Equal(t, tc.expected, op.StepType())
			})
		}
	})

	t.Run("unknown for unhandled opcodes", func(t *testing.T) {
		op := GeneratorOpcode{Op: enums.OpcodeStep}
		assert.Equal(t, enums.StepTypeUnknown, op.StepType())
	})
}

func TestWaitForEventOpts_Expires(t *testing.T) {
	t.Run("accepts duration", func(t *testing.T) {
		opts := WaitForEventOpts{Timeout: "3d"}
		now := time.Now()
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.WithinDuration(t, now.Add(3*24*time.Hour), got, time.Second)
	})

	t.Run("empty timeout returns now", func(t *testing.T) {
		opts := WaitForEventOpts{Timeout: ""}
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now(), got, time.Second)
	})
}

func TestSignalOpts_Expires(t *testing.T) {
	t.Run("accepts duration", func(t *testing.T) {
		opts := SignalOpts{Timeout: "3d"}
		now := time.Now()
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.WithinDuration(t, now.Add(3*24*time.Hour), got, time.Second)
	})

	t.Run("accepts RFC 3339", func(t *testing.T) {
		opts := SignalOpts{Timeout: "2030-01-02T03:04:05Z"}
		want, err := time.Parse(time.RFC3339, "2030-01-02T03:04:05Z")
		require.NoError(t, err)
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("empty defaults to ~1 year", func(t *testing.T) {
		opts := SignalOpts{}
		now := time.Now()
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.WithinDuration(t, now.AddDate(1, 0, 0), got, time.Second)
	})
}

func TestInvokeFunctionOpts_Expires(t *testing.T) {
	t.Run("accepts duration", func(t *testing.T) {
		opts := InvokeFunctionOpts{Timeout: "3d"}
		now := time.Now()
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.WithinDuration(t, now.Add(3*24*time.Hour), got, time.Second)
	})

	t.Run("accepts RFC 3339", func(t *testing.T) {
		opts := InvokeFunctionOpts{Timeout: "2030-01-02T03:04:05Z"}
		want, err := time.Parse(time.RFC3339, "2030-01-02T03:04:05Z")
		require.NoError(t, err)
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("empty defaults to ~1 year", func(t *testing.T) {
		opts := InvokeFunctionOpts{}
		now := time.Now()
		got, err := opts.Expires()
		require.NoError(t, err)
		assert.WithinDuration(t, now.AddDate(1, 0, 0), got, time.Second)
	})
}

// TestDeferAddOpts pins down the first real behavior of OpcodeDeferAdd:
// the SDK emits a GeneratorOpcode whose Opts field carries the target
// onDefer function slug and the user input passed to `step.defer(...)`,
// and the executor needs a typed accessor to read them.
//
// This mirrors the pattern used by every other opcode with options —
// see InvokeFunctionOpts(), WaitForEventOpts(), SignalOpts() in opcode.go.
//
// To make this pass:
//  1. Add a DeferAddOpts struct with `fn_slug` and `input` JSON fields.
//  2. Add an UnmarshalAny method on it (copy the shape of InvokeFunctionOpts).
//  3. Add a DeferAddOpts() method on GeneratorOpcode that parses g.Opts.
func TestDeferAddOpts(t *testing.T) {
	g := GeneratorOpcode{
		Op: enums.OpcodeDeferAdd,
		ID: "deferred-step",
		Opts: map[string]any{
			"fn_slug": "onDefer-score",
			"input":   map[string]any{"user_id": "u_123"},
		},
	}

	opts, err := g.DeferAddOpts()
	require.NoError(t, err)
	require.Equal(t, "onDefer-score", opts.FnSlug)
	require.JSONEq(t, `{"user_id":"u_123"}`, string(opts.Input))
}

func TestDeferCancelOpts(t *testing.T) {
	g := GeneratorOpcode{
		Op: enums.OpcodeDeferCancel,
		ID: "deferred-step",
		Opts: map[string]any{
			"fn_slug":          "onDefer-score",
			"target_hashed_id": "1bbc125d9bcd5b2a07d7d2ea2f0bb42cc721268b",
		},
	}

	opts, err := g.DeferCancelOpts()
	require.NoError(t, err)
	require.Equal(t, "onDefer-score", opts.FnSlug)
	require.Equal(t, "1bbc125d9bcd5b2a07d7d2ea2f0bb42cc721268b", opts.TargetHashedID)
}
