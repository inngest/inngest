package extractors

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractExperimentOptsMetadata_FullFields(t *testing.T) {
	t.Parallel()

	// Simulates the SDK spreading experiment context fields into op.opts.
	opts := map[string]any{
		"type":              "step",
		"input":             []any{},
		"experimentName":    "checkout-flow",
		"variant":           "variant-b",
		"selectionStrategy": "weighted",
	}

	md, err := ExtractExperimentOptsMetadata(opts)

	require.NoError(t, err)
	require.NotNil(t, md, "expected experiment metadata to be produced")
	assert.Equal(t, KindInngestExperiment, md.Kind())

	raw, err := md.Serialize()
	require.NoError(t, err)

	decoded := decodeValues(t, raw)
	assert.Equal(t, "checkout-flow", decoded["experiment_name"])
	assert.Equal(t, "variant-b", decoded["variant"])
	assert.Equal(t, "weighted", decoded["selection_strategy"])
}

func TestExtractExperimentOptsMetadata_NoExperiment(t *testing.T) {
	t.Parallel()

	opts := map[string]any{
		"type":  "step",
		"input": []any{},
	}

	md, err := ExtractExperimentOptsMetadata(opts)

	require.NoError(t, err)
	assert.Nil(t, md, "expected nil when opts carry no experiment fields")
}

func TestExtractExperimentOptsMetadata_NameOnly(t *testing.T) {
	t.Parallel()

	// Older SDKs may send experimentName + variant without selectionStrategy.
	opts := map[string]any{
		"experimentName": "ab-test",
		"variant":        "alpha",
	}

	md, err := ExtractExperimentOptsMetadata(opts)

	require.NoError(t, err)
	require.NotNil(t, md)

	raw, err := md.Serialize()
	require.NoError(t, err)

	decoded := decodeValues(t, raw)
	assert.Equal(t, "ab-test", decoded["experiment_name"])
	assert.Equal(t, "alpha", decoded["variant"])
	assert.Equal(t, "", decoded["selection_strategy"])
}

func TestExtractExperimentOptsMetadata_NilOpts(t *testing.T) {
	t.Parallel()

	md, err := ExtractExperimentOptsMetadata(nil)

	require.NoError(t, err)
	assert.Nil(t, md)
}

func TestExtractExperimentOptsMetadata_RawJSONBytes(t *testing.T) {
	t.Parallel()

	// GeneratorOpcode.Opts arrives unmarshaled-on-demand; depending on the
	// driver path it may be a map[string]any or a raw JSON byte slice. The
	// extractor must tolerate both shapes.
	raw := []byte(`{"experimentName":"raw","variant":"v1","selectionStrategy":"fixed"}`)

	md, err := ExtractExperimentOptsMetadata(raw)

	require.NoError(t, err)
	require.NotNil(t, md)

	serialized, err := md.Serialize()
	require.NoError(t, err)

	decoded := decodeValues(t, serialized)
	assert.Equal(t, "raw", decoded["experiment_name"])
	assert.Equal(t, "v1", decoded["variant"])
	assert.Equal(t, "fixed", decoded["selection_strategy"])
}

func TestExtractExperimentOptsMetadata_EmptyExperimentName(t *testing.T) {
	t.Parallel()

	// An empty experiment name is treated the same as absent — no metadata.
	opts := map[string]any{
		"experimentName": "",
		"variant":        "alpha",
	}

	md, err := ExtractExperimentOptsMetadata(opts)

	require.NoError(t, err)
	assert.Nil(t, md)
}

// decodeValues converts a metadata.Values map (string -> raw JSON) into a
// plain map[string]any for assertions.
func decodeValues(t *testing.T, raw map[string]json.RawMessage) map[string]any {
	t.Helper()
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		var value any
		require.NoError(t, json.Unmarshal(v, &value))
		out[k] = value
	}
	return out
}
