package state

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExperimentRunOpts_WithExperimentFields(t *testing.T) {
	t.Parallel()

	opcode := GeneratorOpcode{
		Op:   enums.OpcodeStepRun,
		ID:   "step-variant-a",
		Name: "variant-a",
		Opts: map[string]any{
			"type":             "step.run",
			"experimentStepID": "step-experiment-123",
			"experimentName":   "checkout-flow",
			"variant":          "variant-a",
		},
	}

	opts, err := opcode.RunOpts()
	require.NoError(t, err)
	require.NotNil(t, opts)

	assert.Equal(t, "step.run", opts.Type)
	assert.Equal(t, "step-experiment-123", opts.ExperimentStepID)
	assert.Equal(t, "checkout-flow", opts.ExperimentName)
	assert.Equal(t, "variant-a", opts.Variant)
}

func TestExperimentRunOpts_WithoutExperimentFields(t *testing.T) {
	t.Parallel()

	// Backward compatibility: RunOpts without experiment fields
	opcode := GeneratorOpcode{
		Op:   enums.OpcodeStepRun,
		ID:   "regular-step",
		Name: "do-work",
		Opts: map[string]any{
			"type": "step.run",
		},
	}

	opts, err := opcode.RunOpts()
	require.NoError(t, err)
	require.NotNil(t, opts)

	assert.Equal(t, "step.run", opts.Type)
	assert.Empty(t, opts.ExperimentStepID, "ExperimentStepID should be empty for non-experiment steps")
	assert.Empty(t, opts.ExperimentName, "ExperimentName should be empty for non-experiment steps")
	assert.Empty(t, opts.Variant, "Variant should be empty for non-experiment steps")
}

func TestExperimentRunOpts_OmitEmptyJSON(t *testing.T) {
	t.Parallel()

	// When experiment fields are empty, they should be omitted from JSON
	opts := RunOpts{
		Type: "step.run",
	}

	byt, err := json.Marshal(opts)
	require.NoError(t, err)

	var raw map[string]any
	err = json.Unmarshal(byt, &raw)
	require.NoError(t, err)

	_, hasExperimentStepID := raw["experimentStepID"]
	_, hasExperimentName := raw["experimentName"]
	_, hasVariant := raw["variant"]

	assert.False(t, hasExperimentStepID, "experimentStepID should be omitted when empty")
	assert.False(t, hasExperimentName, "experimentName should be omitted when empty")
	assert.False(t, hasVariant, "variant should be omitted when empty")
}

func TestExperimentRunOpts_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := RunOpts{
		Type:             "step.run",
		ExperimentStepID: "exp-step-1",
		ExperimentName:   "pricing-test",
		Variant:          "high-price",
	}

	byt, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded RunOpts
	err = json.Unmarshal(byt, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.ExperimentStepID, decoded.ExperimentStepID)
	assert.Equal(t, original.ExperimentName, decoded.ExperimentName)
	assert.Equal(t, original.Variant, decoded.Variant)
}

func TestExperimentRunOpts_UnmarshalAny(t *testing.T) {
	t.Parallel()

	t.Run("from map", func(t *testing.T) {
		t.Parallel()
		opts := &RunOpts{}
		err := opts.UnmarshalAny(map[string]any{
			"type":             "step.run",
			"experimentStepID": "step-exp-1",
			"experimentName":   "ab-test",
			"variant":          "control",
		})
		require.NoError(t, err)

		assert.Equal(t, "step-exp-1", opts.ExperimentStepID)
		assert.Equal(t, "ab-test", opts.ExperimentName)
		assert.Equal(t, "control", opts.Variant)
	})

	t.Run("from bytes", func(t *testing.T) {
		t.Parallel()
		opts := &RunOpts{}
		err := opts.UnmarshalAny([]byte(`{"type":"step.run","experimentStepID":"step-exp-2","experimentName":"feature-flag","variant":"enabled"}`))
		require.NoError(t, err)

		assert.Equal(t, "step-exp-2", opts.ExperimentStepID)
		assert.Equal(t, "feature-flag", opts.ExperimentName)
		assert.Equal(t, "enabled", opts.Variant)
	})

	t.Run("without experiment fields is backward compatible", func(t *testing.T) {
		t.Parallel()
		opts := &RunOpts{}
		err := opts.UnmarshalAny([]byte(`{"type":"step.run"}`))
		require.NoError(t, err)

		assert.Equal(t, "step.run", opts.Type)
		assert.Empty(t, opts.ExperimentStepID)
		assert.Empty(t, opts.ExperimentName)
		assert.Empty(t, opts.Variant)
	})
}
