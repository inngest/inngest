package extractors_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
	"github.com/inngest/inngest/pkg/util"
)

// TestAIMetadataExtractor_CapturedFixtures asserts AIMetadata fields against captured OTLP spans
func TestAIMetadataExtractor_CapturedFixtures(t *testing.T) {
	cases := []struct {
		fixture  string
		expected extractors.AIMetadata
	}{
		// official @opentelemetry/instrumentation-openai
		{
			fixture: "openai_otel_official/params_chat.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "gpt-4.1-nano",
				System:        "openai",
				OperationName: "chat",
				ResponseModel: "gpt-4.1-nano-2025-04-14",
				ResponseID:    "chatcmpl-DmQgLKk5KeV2yWFNlpzOVrmXErHfC",
				FinishReasons: []string{"stop"},
				InputTokens:   22,
				OutputTokens:  6,
				TotalTokens:   util.ToPtr[int64](28),
			},
		},
		{
			fixture: "openai_otel_official/tools_chat.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "gpt-4.1-nano",
				System:        "openai",
				OperationName: "chat",
				ResponseModel: "gpt-4.1-nano-2025-04-14",
				ResponseID:    "chatcmpl-DmQgMTBrf7SFrIb1VMoKSIbKkIcBf",
				FinishReasons: []string{"tool_calls"},
				InputTokens:   56,
				OutputTokens:  14,
				TotalTokens:   util.ToPtr[int64](70),
			},
		},
		{
			fixture: "openai_otel_official/stream_chat.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "gpt-4.1-nano",
				System:        "openai",
				OperationName: "chat",
				ResponseModel: "gpt-4.1-nano-2025-04-14",
				ResponseID:    "chatcmpl-DmQgMYCyOdQgzMUm4UU6jFY7NuSGB",
				FinishReasons: []string{"stop"},
				InputTokens:   11,
				OutputTokens:  10,
				TotalTokens:   util.ToPtr[int64](21),
			},
		},
		{
			fixture: "openai_otel_official/embeddings.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "text-embedding-3-small",
				System:        "openai",
				OperationName: "embeddings",
				ResponseModel: "text-embedding-3-small",
				ResponseID:    "",
				FinishReasons: nil,
				InputTokens:   10,
				OutputTokens:  0,
				TotalTokens:   util.ToPtr[int64](10),
			},
		},

		// @traceloop/instrumentation-openai
		{
			fixture: "openai_otel_traceloop/basic_responses.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "gpt-5.4-nano",
				System:        "openai",
				OperationName: "chat",
				ResponseModel: "gpt-5.4-nano-2026-03-17",
				ResponseID:    "resp_0ba2819472261845006a1f474f066c819983ffb0a8d045e3d5",
				FinishReasons: []string{"stop"},
				InputTokens:   17,
				OutputTokens:  44,
				TotalTokens:   util.ToPtr[int64](61),
			},
		},
		{
			fixture: "openai_otel_traceloop/params_chat.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "gpt-4.1-nano",
				System:        "openai",
				OperationName: "chat",
				ResponseModel: "gpt-4.1-nano-2025-04-14",
				ResponseID:    "chatcmpl-DmQhkoGWd8y8RoHHnRbkw7BHGMj8j",
				FinishReasons: []string{"stop"},
				InputTokens:   22,
				OutputTokens:  6,
				TotalTokens:   util.ToPtr[int64](28),
			},
		},
		{
			fixture: "openai_otel_traceloop/tools_chat.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "gpt-4.1-nano",
				System:        "openai",
				OperationName: "chat",
				ResponseModel: "gpt-4.1-nano-2025-04-14",
				ResponseID:    "chatcmpl-DmQhkWBiDrfqwzKv6CFoRu2DACmfB",
				FinishReasons: []string{"tool_call"},
				InputTokens:   56,
				OutputTokens:  14,
				TotalTokens:   util.ToPtr[int64](70),
			},
		},
		{
			fixture: "openai_otel_traceloop/stream_chat.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "gpt-4.1-nano",
				System:        "openai",
				OperationName: "chat",
				ResponseModel: "gpt-4.1-nano-2025-04-14",
				ResponseID:    "chatcmpl-DmQhlDTJIJh3jrQMeZ8BBMqtVzv8A",
				FinishReasons: []string{"stop"},
				InputTokens:   0,
				OutputTokens:  0,
				TotalTokens:   nil,
			},
		},
		{
			fixture: "openai_otel_traceloop/reasoning_responses.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "gpt-5.4-nano",
				System:        "openai",
				OperationName: "chat",
				ResponseModel: "gpt-5.4-nano-2026-03-17",
				ResponseID:    "resp_0947697ae2c24bfe006a1f4751ad68819aa81bfb021e3ac05e",
				FinishReasons: []string{"stop"},
				InputTokens:   17,
				OutputTokens:  13,
				TotalTokens:   util.ToPtr[int64](30),
			},
		},

		// @arizeai/openinference-instrumentation-openai
		{
			fixture: "openai_openinference_arize/basic_responses.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "gpt-5.4-nano-2026-03-17",
				System:        "openai",
				OperationName: "",
				ResponseModel: "",
				ResponseID:    "",
				FinishReasons: nil,
				InputTokens:   17,
				OutputTokens:  51,
				TotalTokens:   util.ToPtr[int64](68),
			},
		},
		{
			fixture: "openai_openinference_arize/params_chat.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "gpt-4.1-nano-2025-04-14",
				System:        "openai",
				OperationName: "",
				ResponseModel: "",
				ResponseID:    "",
				FinishReasons: []string{"stop"},
				InputTokens:   22,
				OutputTokens:  6,
				TotalTokens:   util.ToPtr[int64](28),
			},
		},
		{
			fixture: "openai_openinference_arize/tools_chat.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "gpt-4.1-nano-2025-04-14",
				System:        "openai",
				OperationName: "",
				ResponseModel: "",
				ResponseID:    "",
				FinishReasons: []string{"tool_calls"},
				InputTokens:   56,
				OutputTokens:  30,
				TotalTokens:   util.ToPtr[int64](86),
			},
		},
		{
			fixture: "openai_openinference_arize/stream_chat.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "gpt-4.1-nano",
				System:        "openai",
				OperationName: "",
				ResponseModel: "",
				ResponseID:    "",
				FinishReasons: []string{"stop"},
				InputTokens:   0,
				OutputTokens:  0,
				TotalTokens:   nil,
			},
		},
		{
			fixture: "openai_openinference_arize/reasoning_responses.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "gpt-5.4-nano-2026-03-17",
				System:        "openai",
				OperationName: "",
				ResponseModel: "",
				ResponseID:    "",
				FinishReasons: nil,
				InputTokens:   17,
				OutputTokens:  13,
				TotalTokens:   util.ToPtr[int64](30),
			},
		},
		{
			fixture: "openai_openinference_arize/embeddings.otlp.json",
			expected: extractors.AIMetadata{
				Model:         "",
				System:        "openai",
				OperationName: "",
				ResponseModel: "",
				ResponseID:    "",
				FinishReasons: nil,
				InputTokens:   0,
				OutputTokens:  0,
				TotalTokens:   nil,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			spans := loadOTLPSpans(t, tc.fixture)
			require.Len(t, spans, 1, "fixture should contain exactly one span")

			structured, err := extractors.NewAIMetadataExtractor().
				ExtractSpanMetadata(context.Background(), spans[0])
			require.NoError(t, err)
			require.Len(t, structured, 1)

			md, ok := structured[0].(extractors.AIMetadata)
			require.True(t, ok, "expected AIMetadata, got %T", structured[0])

			// Blank the derived fields: LatencyMs is span-timing dependent and
			// EstimatedCost is covered by EstimateCost's own tests. The rest is
			// what's mapped from span attributes.
			md.LatencyMs = nil
			md.EstimatedCost = nil
			require.Equal(t, tc.expected, md)
		})
	}
}
