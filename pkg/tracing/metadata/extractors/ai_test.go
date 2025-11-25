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

func TestAIMetadataExtractor_OpenAISpan(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	span := &tracev1.Span{
		SpanId: []byte("test-span-id"),
		Name:   "chat gpt-4",
		Attributes: []*commonv1.KeyValue{
			{
				Key: "gen_ai.usage.input_tokens",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_IntValue{IntValue: 147},
				},
			},
			{
				Key: "gen_ai.usage.output_tokens",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_IntValue{IntValue: 97},
				},
			},
			{
				Key: "gen_ai.request.model",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "gpt-4"},
				},
			},
			{
				Key: "gen_ai.operation.name",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "chat"},
				},
			},
			{
				Key: "gen_ai.system",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "openai"},
				},
			},
			{
				Key: "http.method",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "POST"},
				},
			},
		},
	}

	extractor := NewAIMetadataExtractor()
	md, err := extractor.ExtractSpanMetadata(ctx, span)

	require.NoError(t, err)

	require.NotNil(t, md, "Expected metadata for OpenAI span")
	require.Len(t, md, 1, "Expected exactly one metadata item")

	assert.Equal(t, metadata.Kind("inngest.ai"), md[0].Kind())
	assert.Equal(t, enums.MetadataOpcodeMerge, md[0].Op())

	// Verify the extracted data content
	raw, err := md[0].Serialize()
	require.NoError(t, err)

	var data map[string]any
	if dataBytes, ok := raw["data"]; ok {
		err = json.Unmarshal(dataBytes, &data)
		require.NoError(t, err)
	} else {
		// Or the data might be directly in the raw map
		data = make(map[string]any)
		for k, v := range raw {
			var value any
			if err := json.Unmarshal(v, &value); err == nil {
				data[k] = value
			}
		}
	}

	// Verify token data was extracted correctly
	assert.Equal(t, 147.0, data["input_tokens"], "Should extract input tokens")
	assert.Equal(t, 97.0, data["output_tokens"], "Should extract output tokens")

	// Verify model and operation data
	assert.Equal(t, "gpt-4", data["model"], "Should extract request model")
	assert.Equal(t, "chat", data["operation_name"], "Should extract operation name")
	assert.Equal(t, "openai", data["system"], "Should extract AI system")
}

func TestAIMetadataExtractor_NonAISpan(t *testing.T) {
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
			{
				Key: "http.path",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "/api/users"},
				},
			},
		},
	}

	extractor := NewAIMetadataExtractor()
	metadata, err := extractor.ExtractSpanMetadata(ctx, span)

	require.NoError(t, err)

	assert.Nil(t, metadata, "Non-AI span should not produce metadata")
}
