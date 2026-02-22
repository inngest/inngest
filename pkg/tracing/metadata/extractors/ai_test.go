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

func TestExtractAIOutputMetadata_VercelAISDK(t *testing.T) {
	t.Parallel()

	// Simulated Vercel AI SDK response from step.ai.wrap
	vercelResponse := map[string]any{
		"data": map[string]any{
			"totalUsage": map[string]any{
				"inputTokens":  11,
				"outputTokens": 429,
				"totalTokens":  440,
			},
			"steps": []map[string]any{
				{
					"usage": map[string]any{
						"inputTokens":  11,
						"outputTokens": 429,
						"totalTokens":  440,
					},
					"response": map[string]any{
						"modelId": "gpt-4-turbo-2024-04-09",
						"headers": map[string]any{
							"openai-processing-ms": "24314",
						},
					},
					"request": map[string]any{
						"body": map[string]any{
							"model": "gpt-4-turbo",
						},
					},
				},
			},
		},
	}

	output, err := json.Marshal(vercelResponse)
	require.NoError(t, err)

	stepDurationMs := int64(25000) // 25 seconds

	md, err := ExtractAIOutputMetadata(output, stepDurationMs)
	require.NoError(t, err)
	require.NotNil(t, md, "Expected metadata for Vercel AI SDK response")
	require.Len(t, md, 1, "Expected exactly one metadata item")

	assert.Equal(t, metadata.Kind("inngest.ai"), md[0].Kind())
	assert.Equal(t, enums.MetadataOpcodeMerge, md[0].Op())

	// Serialize and verify the content
	raw, err := md[0].Serialize()
	require.NoError(t, err)

	data := make(map[string]any)
	for k, v := range raw {
		var value any
		if err := json.Unmarshal(v, &value); err == nil {
			data[k] = value
		}
	}

	// Verify token data
	assert.Equal(t, 11.0, data["input_tokens"], "Should extract input tokens")
	assert.Equal(t, 429.0, data["output_tokens"], "Should extract output tokens")
	assert.Equal(t, 440.0, data["total_tokens"], "Should extract total tokens")

	// Verify model
	assert.Equal(t, "gpt-4-turbo-2024-04-09", data["model"], "Should extract model from response.modelId")

	// Verify system
	assert.Equal(t, "vercel-ai", data["system"], "Should set system to vercel-ai")

	// Verify latency (from openai-processing-ms header)
	assert.Equal(t, 24314.0, data["latency_ms"], "Should extract latency from OpenAI header")

	// Verify cost estimation
	assert.NotNil(t, data["estimated_cost"], "Should estimate cost")
}

func TestExtractAIOutputMetadata_FallbackLatency(t *testing.T) {
	t.Parallel()

	// Response without provider headers, should fall back to step duration
	vercelResponse := map[string]any{
		"data": map[string]any{
			"totalUsage": map[string]any{
				"inputTokens":  100,
				"outputTokens": 200,
				"totalTokens":  300,
			},
			"steps": []map[string]any{
				{
					"response": map[string]any{
						"modelId": "gpt-4o",
						// No headers, latency should fall back to step duration
					},
				},
			},
		},
	}

	output, err := json.Marshal(vercelResponse)
	require.NoError(t, err)

	stepDurationMs := int64(5000)

	md, err := ExtractAIOutputMetadata(output, stepDurationMs)
	require.NoError(t, err)
	require.NotNil(t, md)

	raw, err := md[0].Serialize()
	require.NoError(t, err)

	data := make(map[string]any)
	for k, v := range raw {
		var value any
		if err := json.Unmarshal(v, &value); err == nil {
			data[k] = value
		}
	}

	// Latency should fall back to step duration
	assert.Equal(t, 5000.0, data["latency_ms"], "Should fall back to step duration for latency")
}

func TestExtractAIOutputMetadata_NonVercelFormat(t *testing.T) {
	t.Parallel()

	// Non-Vercel format should silently skip
	nonVercelResponse := map[string]any{
		"data": map[string]any{
			"text": "Hello, world!",
			// No totalUsage or steps
		},
	}

	output, err := json.Marshal(nonVercelResponse)
	require.NoError(t, err)

	md, err := ExtractAIOutputMetadata(output, 1000)
	require.NoError(t, err)
	assert.Nil(t, md, "Non-Vercel format should return nil")
}

func TestExtractAIOutputMetadata_InvalidJSON(t *testing.T) {
	t.Parallel()

	md, err := ExtractAIOutputMetadata([]byte("not valid json"), 1000)
	require.NoError(t, err)
	assert.Nil(t, md, "Invalid JSON should return nil")
}

func TestExtractAIOutputMetadata_TypicalStepOutput(t *testing.T) {
	t.Parallel()

	// Typical non-AI step.run output - should be skipped by bytes.Contains pre-check
	// without ever reaching json.Unmarshal
	typicalOutputs := [][]byte{
		[]byte(`{"data":"hello world"}`),
		[]byte(`{"data":{"user":"john","email":"john@example.com"}}`),
		[]byte(`{"data":42}`),
		[]byte(`{"data":null}`),
		[]byte(`{"data":["item1","item2","item3"]}`),
		[]byte(`{"data":{"results":[1,2,3],"count":3}}`),
	}

	for _, output := range typicalOutputs {
		md, err := ExtractAIOutputMetadata(output, 1000)
		require.NoError(t, err)
		assert.Nil(t, md, "Non-AI step output should return nil: %s", string(output))
	}
}
