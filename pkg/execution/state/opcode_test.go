package state

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
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
			{enums.OpcodeGateway, enums.StepTypeFetch},
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

func TestDeferAddOpts_Validate(t *testing.T) {
	t.Run("rejects empty FnSlug", func(t *testing.T) {
		opts := DeferAddOpts{Input: json.RawMessage(`{}`)}
		require.Error(t, opts.Validate())
	})

	t.Run("rejects empty Input", func(t *testing.T) {
		opts := DeferAddOpts{FnSlug: "fn"}
		require.ErrorIs(t, opts.Validate(), ErrDeferInputInvalid)
	})

	t.Run("rejects non-object Input", func(t *testing.T) {
		// Arrays, scalars, and bare strings all fail. The SDK contract is
		// that defer Input is a JSON object so step receivers can index by
		// key.
		for _, in := range []string{`"x"`, `[1,2]`, `null`, `42`, `true`} {
			opts := DeferAddOpts{FnSlug: "fn", Input: json.RawMessage(in)}
			require.ErrorIs(t, opts.Validate(), ErrDeferInputInvalid, "Input=%q", in)
		}
	})

	t.Run("accepts valid opts", func(t *testing.T) {
		opts := DeferAddOpts{FnSlug: "fn", Input: json.RawMessage(`{"k":"v"}`)}
		require.NoError(t, opts.Validate())
	})

	t.Run("rejects Input larger than MaxDeferInputSize", func(t *testing.T) {
		// Build a valid JSON object whose total byte length is one over the
		// limit. The IsJSONObject gate runs before the size check, so the
		// payload must start with `{` — bare padding would be rejected as
		// invalid input instead of as oversized.
		oversized := buildOversizedJSONObject(consts.MaxDeferInputSize + 1)
		opts := DeferAddOpts{FnSlug: "fn", Input: json.RawMessage(oversized)}
		err := opts.Validate()
		require.ErrorIs(t, err, ErrDeferInputTooLarge,
			"oversized defer Input must be rejected at validation, not silently stored in Redis")
	})

	t.Run("accepts Input at exactly MaxDeferInputSize", func(t *testing.T) {
		// Boundary: equal-to-limit must pass; only strictly greater fails.
		atLimit := buildOversizedJSONObject(consts.MaxDeferInputSize)
		opts := DeferAddOpts{FnSlug: "fn", Input: json.RawMessage(atLimit)}
		require.NoError(t, opts.Validate())
	})
}

// buildOversizedJSONObject returns a JSON object whose serialized form is
// exactly `total` bytes. Padding is filler inside a string value so the
// payload remains a valid JSON object (IsJSONObject passes) while still
// stressing the size check.
func buildOversizedJSONObject(total int) []byte {
	const wrapper = `{"k":""}`
	pad := max(total-len(wrapper), 0)
	out := make([]byte, 0, total)
	out = append(out, []byte(`{"k":"`)...)
	out = append(out, bytes.Repeat([]byte("x"), pad)...)
	out = append(out, []byte(`"}`)...)
	return out
}
