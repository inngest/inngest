package extractors

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

func TestExperimentMetadataExtractor_FullSpan(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	span := &tracev1.Span{
		SpanId: []byte("experiment-span-id"),
		Name:   "step.experiment",
		Attributes: []*commonv1.KeyValue{
			{
				Key: "inngest.experiment.name",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "checkout-flow"},
				},
			},
			{
				Key: "inngest.experiment.variant_selected",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "variant-b"},
				},
			},
			{
				Key: "inngest.experiment.selection_strategy",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "weighted"},
				},
			},
			{
				Key: "inngest.experiment.available_variants",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_ArrayValue{
						ArrayValue: &commonv1.ArrayValue{
							Values: []*commonv1.AnyValue{
								{Value: &commonv1.AnyValue_StringValue{StringValue: "control"}},
								{Value: &commonv1.AnyValue_StringValue{StringValue: "variant-a"}},
								{Value: &commonv1.AnyValue_StringValue{StringValue: "variant-b"}},
							},
						},
					},
				},
			},
			{
				Key: "inngest.experiment.variant_weights",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: `{"control":50,"variant-a":25,"variant-b":25}`},
				},
			},
		},
	}

	extractor := NewExperimentMetadataExtractor()
	md, err := extractor.ExtractSpanMetadata(ctx, span)

	require.NoError(t, err)
	require.NotNil(t, md, "Expected metadata for experiment span")
	require.Len(t, md, 1, "Expected exactly one metadata item")

	assert.Equal(t, metadata.Kind("inngest.experiment"), md[0].Kind())
	assert.Equal(t, enums.MetadataOpcodeMerge, md[0].Op())

	// Verify the extracted data content via Serialize
	raw, err := md[0].Serialize()
	require.NoError(t, err)

	data := make(map[string]any)
	for k, v := range raw {
		var value any
		if err := json.Unmarshal(v, &value); err == nil {
			data[k] = value
		}
	}

	assert.Equal(t, "checkout-flow", data["experiment_name"])
	assert.Equal(t, "variant-b", data["variant_selected"])
	assert.Equal(t, "weighted", data["selection_strategy"])

	variants, ok := data["available_variants"].([]any)
	require.True(t, ok, "available_variants should be a slice")
	require.Len(t, variants, 3)
	assert.Equal(t, "control", variants[0])
	assert.Equal(t, "variant-a", variants[1])
	assert.Equal(t, "variant-b", variants[2])

	weights, ok := data["variant_weights"].(map[string]any)
	require.True(t, ok, "variant_weights should be a map")
	assert.Equal(t, 50.0, weights["control"])
	assert.Equal(t, 25.0, weights["variant-a"])
	assert.Equal(t, 25.0, weights["variant-b"])
}

func TestExperimentMetadataExtractor_NonExperimentSpan(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	span := &tracev1.Span{
		SpanId: []byte("http-span-id"),
		Name:   "GET /api/users",
		Attributes: []*commonv1.KeyValue{
			{
				Key: "http.method",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "GET"},
				},
			},
			{
				Key: "http.status_code",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_IntValue{IntValue: 200},
				},
			},
		},
	}

	extractor := NewExperimentMetadataExtractor()
	md, err := extractor.ExtractSpanMetadata(ctx, span)

	require.NoError(t, err)
	assert.Nil(t, md, "Non-experiment span should not produce metadata")
}

func TestExperimentMetadataExtractor_PartialAttributes(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Span with only experiment name — should still be detected and extracted
	span := &tracev1.Span{
		SpanId: []byte("partial-experiment-span"),
		Name:   "step.experiment",
		Attributes: []*commonv1.KeyValue{
			{
				Key: "inngest.experiment.name",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "ab-test"},
				},
			},
		},
	}

	extractor := NewExperimentMetadataExtractor()
	md, err := extractor.ExtractSpanMetadata(ctx, span)

	require.NoError(t, err)
	require.NotNil(t, md, "Span with experiment attribute should produce metadata")
	require.Len(t, md, 1)

	raw, err := md[0].Serialize()
	require.NoError(t, err)

	data := make(map[string]any)
	for k, v := range raw {
		var value any
		if err := json.Unmarshal(v, &value); err == nil {
			data[k] = value
		}
	}

	assert.Equal(t, "ab-test", data["experiment_name"])
	assert.Equal(t, "", data["variant_selected"])
	assert.Equal(t, "", data["selection_strategy"])
}

func TestExperimentMetadataExtractor_NoWeights(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	span := &tracev1.Span{
		SpanId: []byte("no-weights-span"),
		Attributes: []*commonv1.KeyValue{
			{
				Key: "inngest.experiment.name",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "feature-flag"},
				},
			},
			{
				Key: "inngest.experiment.variant_selected",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "enabled"},
				},
			},
		},
	}

	extractor := NewExperimentMetadataExtractor()
	md, err := extractor.ExtractSpanMetadata(ctx, span)

	require.NoError(t, err)
	require.NotNil(t, md)
	require.Len(t, md, 1)

	raw, err := md[0].Serialize()
	require.NoError(t, err)

	// variant_weights should not appear in serialized output (omitempty)
	_, hasWeights := raw["variant_weights"]
	assert.False(t, hasWeights, "variant_weights should be omitted when nil")
}

func TestExperimentMetadata_Serialize(t *testing.T) {
	t.Parallel()

	md := ExperimentMetadata{
		ExperimentName:    "pricing-test",
		VariantSelected:   "high-price",
		SelectionStrategy: "random",
		AvailableVariants: []string{"low-price", "high-price"},
		VariantWeights:    map[string]int{"low-price": 50, "high-price": 50},
	}

	raw, err := md.Serialize()
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	// Round-trip: re-marshal the Values map into JSON, then deserialize back
	intermediate, err := json.Marshal(raw)
	require.NoError(t, err)

	var roundTripped ExperimentMetadata
	err = json.Unmarshal(intermediate, &roundTripped)
	require.NoError(t, err)

	assert.Equal(t, md.ExperimentName, roundTripped.ExperimentName)
	assert.Equal(t, md.VariantSelected, roundTripped.VariantSelected)
	assert.Equal(t, md.SelectionStrategy, roundTripped.SelectionStrategy)
	assert.Equal(t, md.AvailableVariants, roundTripped.AvailableVariants)
	assert.Equal(t, md.VariantWeights, roundTripped.VariantWeights)
}
