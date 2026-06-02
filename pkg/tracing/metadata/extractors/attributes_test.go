package extractors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
)

func strAttr(key, value string) *v1.KeyValue {
	return &v1.KeyValue{
		Key:   key,
		Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: value}},
	}
}

func intAttr(key string, value int64) *v1.KeyValue {
	return &v1.KeyValue{
		Key:   key,
		Value: &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: value}},
	}
}

func strArrAttr(key string, values ...string) *v1.KeyValue {
	vals := make([]*v1.AnyValue, len(values))
	for i, v := range values {
		vals[i] = &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: v}}
	}
	return &v1.KeyValue{
		Key:   key,
		Value: &v1.AnyValue{Value: &v1.AnyValue_ArrayValue{ArrayValue: &v1.ArrayValue{Values: vals}}},
	}
}

// TestCanonicalKeysHaveFieldSetters guards against drift between
// canonicalKeyMapping and metadataFieldSetters: every canonical key referenced
// by a mapping must have a setter, otherwise the attribute is silently dropped.
func TestCanonicalKeysHaveFieldSetters(t *testing.T) {
	t.Parallel()

	for sourceKey, mapping := range keyFieldMap {
		_, ok := metadataFieldSetters[mapping.field]
		assert.Truef(t, ok,
			"field %q (mapped from %q) has no entry in metadataFieldSetters",
			mapping.field, sourceKey)
	}
}

// TestRankingPrefersProviderNameOverDeprecatedSystem verifies that when both
// the deprecated gen_ai.system and its replacement gen_ai.provider.name are
// present, the deprecated key's keyRank demotes it so the replacement wins,
// regardless of attribute order.
func TestRankingPrefersProviderNameOverDeprecatedSystem(t *testing.T) {
	t.Parallel()

	orders := [][]*v1.KeyValue{
		{strAttr("gen_ai.system", "openai"), strAttr("gen_ai.provider.name", "anthropic")},
		{strAttr("gen_ai.provider.name", "anthropic"), strAttr("gen_ai.system", "openai")},
	}

	for _, attrs := range orders {
		var md AIMetadata
		foundAny := extractAIMetadataFromAttributes(attrs, &md)
		assert.True(t, foundAny)
		assert.Equal(t, "anthropic", md.System,
			"gen_ai.provider.name should win over deprecated gen_ai.system")
	}
}

// TestExtractAIMetadataFromAttributes_RealSpans feeds the full instrumentation
// attribute set captured from real runs of test apps and asserts the
// extractor pulls the correct values for the fields it parses.
func TestExtractAIMetadataFromAttributes_RealSpans(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		attrs []*v1.KeyValue
		want  AIMetadata
	}{
		{
			// @traceloop/instrumentation-openai (gen_ai.* semconv).
			name: "semconv (traceloop)",
			attrs: []*v1.KeyValue{
				strAttr("gen_ai.input.messages", `[{"role":"user","parts":[{"type":"text","content":"Write a one-sentence bedtime story about a unicorn."}]}]`),
				strAttr("gen_ai.operation.name", "chat"),
				strAttr("gen_ai.output.messages", `[{"role":"assistant","finish_reason":"stop","parts":[{"type":"text","content":"Under a moonlit sky, a gentle unicorn tiptoed through a sleepy meadow, sprinkling stardust on every leaf until the whole forest drifted into the softest, dreamiest quiet."}]}]`),
				strAttr("gen_ai.provider.name", "openai"),
				strAttr("gen_ai.request.model", "gpt-5.4-nano"),
				strAttr("gen_ai.response.finish_reasons", ""),
				strAttr("gen_ai.response.id", "resp_0a6244b051b4439d006a1dfd4611e0819a9ff1b38c811bc714"),
				strAttr("gen_ai.response.model", "gpt-5.4-nano-2026-03-17"),
				intAttr("gen_ai.usage.input_tokens", 17),
				intAttr("gen_ai.usage.output_tokens", 44),
				intAttr("gen_ai.usage.total_tokens", 61),
			},
			want: AIMetadata{
				InputTokens:   17,
				OutputTokens:  44,
				Model:         "gpt-5.4-nano",
				System:        "openai",
				OperationName: "chat",
			},
		},
		{
			// @arizeai/openinference-instrumentation-openai (llm.*/openinference.*).
			name: "openinference (arizeai)",
			attrs: []*v1.KeyValue{
				strAttr("input.mime_type", "application/json"),
				strAttr("input.value", `{"model":"gpt-5.4-nano","input":"Write a one-sentence bedtime story about a unicorn."}`),
				strAttr("llm.input_messages.0.message.content", "Write a one-sentence bedtime story about a unicorn."),
				strAttr("llm.input_messages.0.message.role", "user"),
				strAttr("llm.invocation_parameters", `{"model":"gpt-5.4-nano"}`),
				strAttr("llm.model_name", "gpt-5.4-nano-2026-03-17"),
				strAttr("llm.output_messages.0.message.contents.0.message_content.text", "On a moonlit night, a gentle unicorn tucked the stars into the corners of the sky and drifted off to sleep with a sigh of silver dreams."),
				strAttr("llm.output_messages.0.message.contents.0.message_content.type", "output_text"),
				strAttr("llm.output_messages.0.message.role", "assistant"),
				strAttr("llm.provider", "openai"),
				strAttr("llm.system", "openai"),
				intAttr("llm.token_count.completion", 35),
				intAttr("llm.token_count.completion_details.reasoning", 0),
				intAttr("llm.token_count.prompt", 17),
				intAttr("llm.token_count.prompt_details.cache_read", 0),
				intAttr("llm.token_count.total", 52),
				strAttr("openinference.span.kind", "LLM"),
				strAttr("output.mime_type", "application/json"),
				strAttr("output.value", `{"id":"resp_0f3cf5bafdfbe25c006a1dfb297544819aa73f0d05c13d5bd2","object":"response","created_at":1780349737,"status":"completed","model":"gpt-5.4-nano-2026-03-17","output":[{"type":"message","status":"completed","content":[{"type":"output_text","text":"On a moonlit night, a gentle unicorn tucked the stars into the corners of the sky and drifted off to sleep with a sigh of silver dreams."}],"role":"assistant"}],"usage":{"input_tokens":17,"output_tokens":35,"total_tokens":52},"output_text":"On a moonlit night, a gentle unicorn tucked the stars into the corners of the sky and drifted off to sleep with a sigh of silver dreams."}`),
			},
			want: AIMetadata{
				InputTokens:  17,
				OutputTokens: 35,
				Model:        "gpt-5.4-nano-2026-03-17",
				System:       "openai",
				// No gen_ai.operation.name equivalent, so OperationName stays empty.
				OperationName: "",
			},
		},
		{
			// @traceloop/instrumentation-openai, function-calling request.
			name: "semconv tool call (traceloop)",
			attrs: []*v1.KeyValue{
				strAttr("gen_ai.input.messages", `[{"role":"user","parts":[{"type":"text","content":"What is the weather in Paris? Use the tool."}]}]`),
				strAttr("gen_ai.operation.name", "chat"),
				strAttr("gen_ai.output.messages", `[{"role":"assistant","finish_reason":"tool_call","parts":[{"type":"tool_call","id":"call_6MxqYADSZjLxOISAobIryAVW","name":"get_weather","arguments":{"city":"Paris"}}]}]`),
				strAttr("gen_ai.provider.name", "openai"),
				strAttr("gen_ai.request.model", "gpt-4.1-nano"),
				strArrAttr("gen_ai.response.finish_reasons", "tool_call"),
				strAttr("gen_ai.response.id", "chatcmpl-Dm5c6BetL4BkrR6mEAWxcGqloC8Dg"),
				strAttr("gen_ai.response.model", "gpt-4.1-nano-2025-04-14"),
				strAttr("gen_ai.tool.definitions", `[{"type":"function","function":{"name":"get_weather","description":"Get the current weather for a city","parameters":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}}}]`),
				intAttr("gen_ai.usage.input_tokens", 56),
				intAttr("gen_ai.usage.output_tokens", 30),
				intAttr("gen_ai.usage.total_tokens", 86),
			},
			want: AIMetadata{
				InputTokens:   56,
				OutputTokens:  30,
				Model:         "gpt-4.1-nano",
				System:        "openai",
				OperationName: "chat",
			},
		},
		{
			// @traceloop/instrumentation-openai, streamed chat completion.
			//
			// Even with stream_options.include_usage, this emitter omits all
			// gen_ai.usage.* tokens on streamed spans.
			name: "semconv streaming, no usage (traceloop)",
			attrs: []*v1.KeyValue{
				strAttr("gen_ai.input.messages", `[{"role":"user","parts":[{"type":"text","content":"Count to five."}]}]`),
				strAttr("gen_ai.operation.name", "chat"),
				strAttr("gen_ai.output.messages", `[{"role":"assistant","finish_reason":"stop","parts":[{"type":"text","content":"One, two, three, four, five."}]}]`),
				strAttr("gen_ai.provider.name", "openai"),
				strAttr("gen_ai.request.model", "gpt-4.1-nano"),
				strArrAttr("gen_ai.response.finish_reasons", "stop"),
				strAttr("gen_ai.response.id", "chatcmpl-Dm5c7PQERrIFXe4rcDT0Z34aebODn"),
				strAttr("gen_ai.response.model", "gpt-4.1-nano-2025-04-14"),
			},
			want: AIMetadata{
				// No usage tokens on the streamed span.
				InputTokens:   0,
				OutputTokens:  0,
				Model:         "gpt-4.1-nano",
				System:        "openai",
				OperationName: "chat",
			},
		},
		{
			// @arizeai/openinference-instrumentation-openai, function-calling
			// request.
			name: "openinference tool call (arizeai)",
			attrs: []*v1.KeyValue{
				strAttr("openinference.span.kind", "LLM"),
				strAttr("llm.model_name", "gpt-4.1-nano-2025-04-14"),
				strAttr("llm.system", "openai"),
				strAttr("llm.provider", "openai"),
				strAttr("llm.input_messages.0.message.role", "user"),
				strAttr("llm.input_messages.0.message.content", "What is the weather in Paris? Use the tool."),
				strAttr("llm.tools.0.tool.json_schema", `{"type":"function","function":{"name":"get_weather","description":"Get the current weather for a city","parameters":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}}}`),
				strAttr("llm.output_messages.0.message.role", "assistant"),
				strAttr("llm.output_messages.0.message.tool_calls.0.tool_call.id", "call_H0gsiYIf5Yqe1WszKYFflRii"),
				strAttr("llm.output_messages.0.message.tool_calls.0.tool_call.function.name", "get_weather"),
				strAttr("llm.output_messages.0.message.tool_calls.0.tool_call.function.arguments", `{"city":"Paris"}`),
				strAttr("llm.finish_reason", "tool_calls"),
				intAttr("llm.token_count.prompt", 56),
				intAttr("llm.token_count.completion", 14),
				intAttr("llm.token_count.total", 70),
				intAttr("llm.token_count.prompt_details.cache_read", 0),
				intAttr("llm.token_count.completion_details.reasoning", 0),
			},
			want: AIMetadata{
				InputTokens:   56,
				OutputTokens:  14,
				Model:         "gpt-4.1-nano-2025-04-14",
				System:        "openai",
				OperationName: "",
			},
		},
		{
			// @arizeai/openinference-instrumentation-openai, streamed chat.
			name: "openinference streaming, no usage (arizeai)",
			attrs: []*v1.KeyValue{
				strAttr("openinference.span.kind", "LLM"),
				strAttr("llm.model_name", "gpt-4.1-nano"),
				strAttr("llm.system", "openai"),
				strAttr("llm.provider", "openai"),
				strAttr("llm.input_messages.0.message.role", "user"),
				strAttr("llm.input_messages.0.message.content", "Count to five."),
				strAttr("llm.output_messages.0.message.role", "assistant"),
				strAttr("llm.output_messages.0.message.content", "One, two, three, four, five."),
				strAttr("llm.finish_reason", "stop"),
				strAttr("input.mime_type", "application/json"),
				strAttr("output.mime_type", "text/plain"),
				strAttr("output.value", "One, two, three, four, five."),
			},
			want: AIMetadata{
				InputTokens:   0,
				OutputTokens:  0,
				Model:         "gpt-4.1-nano",
				System:        "openai",
				OperationName: "",
			},
		},
		{
			// @arizeai/openinference-instrumentation-openai, embeddings.
			name: "openinference embeddings, provider only (arizeai)",
			attrs: []*v1.KeyValue{
				strAttr("openinference.span.kind", "EMBEDDING"),
				strAttr("embedding.model_name", "text-embedding-3-small"),
				strAttr("llm.system", "openai"),
				strAttr("llm.provider", "openai"),
				strAttr("embedding.embeddings.0.embedding.text", "The quick brown fox jumps over the lazy dog."),
				strAttr("input.mime_type", "text/plain"),
				strAttr("input.value", "The quick brown fox jumps over the lazy dog."),
			},
			want: AIMetadata{
				System: "openai",
			},
		},
		{
			// Official @opentelemetry/instrumentation-openai (gen_ai.* + server.*).
			name: "official otel semconv (chat)",
			attrs: []*v1.KeyValue{
				strAttr("gen_ai.operation.name", "chat"),
				strAttr("gen_ai.request.model", "gpt-4.1-nano"),
				strArrAttr("gen_ai.response.finish_reasons", "tool_calls"),
				strAttr("gen_ai.response.id", "chatcmpl-Dm5nsU4lZuCVnOIxBJMAE2hhaQ98v"),
				strAttr("gen_ai.response.model", "gpt-4.1-nano-2025-04-14"),
				strAttr("gen_ai.system", "openai"),
				intAttr("gen_ai.usage.input_tokens", 56),
				intAttr("gen_ai.usage.output_tokens", 14),
				strAttr("server.address", "api.openai.com"),
				intAttr("server.port", 443),
			},
			want: AIMetadata{
				InputTokens:   56,
				OutputTokens:  14,
				Model:         "gpt-4.1-nano",
				System:        "openai",
				OperationName: "chat",
			},
		},
		{
			// Official @opentelemetry/instrumentation-openai, embeddings.
			name: "official otel embeddings",
			attrs: []*v1.KeyValue{
				strAttr("gen_ai.operation.name", "embeddings"),
				strAttr("gen_ai.request.model", "text-embedding-3-small"),
				strAttr("gen_ai.response.model", "text-embedding-3-small"),
				strAttr("gen_ai.system", "openai"),
				intAttr("gen_ai.usage.input_tokens", 10),
				strAttr("server.address", "api.openai.com"),
				intAttr("server.port", 443),
			},
			want: AIMetadata{
				InputTokens:   10,
				OutputTokens:  0,
				Model:         "text-embedding-3-small",
				System:        "openai",
				OperationName: "embeddings",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var md AIMetadata
			foundAny := extractAIMetadataFromAttributes(tc.attrs, &md)

			assert.True(t, foundAny)
			assert.Equal(t, tc.want, md)
		})
	}
}
