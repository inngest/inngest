package extractors_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
	"github.com/inngest/inngest/pkg/util"
)

// TestAIMetadataExtractor_CapturedFixtures asserts AIMetadata fields against captured OTLP spans
func TestAIMetadataExtractor_CapturedFixtures(t *testing.T) {
	cases := []struct {
		fixture string
		// spanName selects a span by name from a multi-span fixture (the Vercel
		// AI SDK emits a parent ai.<op> + child ai.<op>.do<Op> per call). When
		// empty, the fixture must contain exactly one span.
		spanName string
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

		// Vercel AI SDK
		//
		// Each call emits a 2-span tree; we assert the top-level ai.<op> parent span,
		// which carries only ai.* attributes.
		//
		// Documented gaps for the parent:
		// - OperationName is empty (no gen_ai.operation.name; ai.operationId is not mapped),
		// - ResponseModel/ResponseID are empty (only the child .doGenerate span carries them).
		// - System is kept faithful as the provider+surface ("openai.responses"/
		//   "openai.chat"/"openai.embedding").
		{
			fixture:  "openai_vercel_aisdk/basic_responses.otlp.json",
			spanName: "ai.generateText",
			expected: extractors.AIMetadata{
				Model:         "gpt-5.4-nano",
				System:        "openai.responses",
				OperationName: "",
				ResponseModel: "",
				ResponseID:    "",
				FinishReasons: []string{"stop"},
				InputTokens:   17,
				OutputTokens:  40,
				TotalTokens:   util.ToPtr[int64](57),
			},
		},
		{
			fixture:  "openai_vercel_aisdk/params_chat.otlp.json",
			spanName: "ai.generateText",
			expected: extractors.AIMetadata{
				Model:         "gpt-4.1-nano",
				System:        "openai.chat",
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
			fixture:  "openai_vercel_aisdk/tools_chat.otlp.json",
			spanName: "ai.generateText",
			expected: extractors.AIMetadata{
				Model:         "gpt-4.1-nano",
				System:        "openai.chat",
				OperationName: "",
				ResponseModel: "",
				ResponseID:    "",
				// Vercel emits the finish reason as "tool-calls" (hyphen); stored raw.
				FinishReasons: []string{"tool-calls"},
				InputTokens:   56,
				OutputTokens:  14,
				TotalTokens:   util.ToPtr[int64](70),
			},
		},
		{
			fixture:  "openai_vercel_aisdk/stream_chat.otlp.json",
			spanName: "ai.streamText",
			expected: extractors.AIMetadata{
				Model:         "gpt-4.1-nano",
				System:        "openai.chat",
				OperationName: "",
				ResponseModel: "",
				ResponseID:    "",
				FinishReasons: []string{"stop"},
				// Unlike the traceloop/openinference stream cases (0/0/nil), the AI
				// SDK keeps usage on the streaming span.
				InputTokens:  11,
				OutputTokens: 10,
				TotalTokens:  util.ToPtr[int64](21),
			},
		},
		{
			fixture:  "openai_vercel_aisdk/reasoning_responses.otlp.json",
			spanName: "ai.generateText",
			expected: extractors.AIMetadata{
				Model:         "gpt-5.4-nano",
				System:        "openai.responses",
				OperationName: "",
				ResponseModel: "",
				ResponseID:    "",
				FinishReasons: []string{"stop"},
				InputTokens:   42,
				OutputTokens:  195,
				TotalTokens:   util.ToPtr[int64](237),
			},
		},
		{
			fixture:  "openai_vercel_aisdk/embeddings.otlp.json",
			spanName: "ai.embed",
			expected: extractors.AIMetadata{
				Model:         "text-embedding-3-small",
				System:        "openai.embedding",
				OperationName: "",
				ResponseModel: "",
				ResponseID:    "",
				FinishReasons: nil,
				// Embeddings emit a single ai.usage.tokens count -> inputTokens;
				// TotalTokens derives to the same value.
				InputTokens:  10,
				OutputTokens: 0,
				TotalTokens:  util.ToPtr[int64](10),
			},
		},
		// One child span (ai.generateText.doGenerate) to lock dual-convention
		// coexistence: gen_ai.* wins the shared fields (values agree), and
		// TotalTokens comes from ai.usage.totalTokens because gen_ai.* omits a
		// total. ResponseModel/ResponseID are present on the child.
		{
			fixture:  "openai_vercel_aisdk/basic_responses.otlp.json",
			spanName: "ai.generateText.doGenerate",
			expected: extractors.AIMetadata{
				Model:         "gpt-5.4-nano",
				System:        "openai.responses",
				OperationName: "",
				ResponseModel: "gpt-5.4-nano-2026-03-17",
				ResponseID:    "resp_0e56d5c95850276a006a1f57da3d60819a8974cbbffaedd001",
				FinishReasons: []string{"stop"},
				InputTokens:   17,
				OutputTokens:  40,
				TotalTokens:   util.ToPtr[int64](57),
			},
		},
	}

	for _, tc := range cases {
		name := tc.fixture
		if tc.spanName != "" {
			name += "#" + tc.spanName
		}
		t.Run(name, func(t *testing.T) {
			spans := loadOTLPSpans(t, tc.fixture)

			var span *tracev1.Span
			if tc.spanName == "" {
				require.Len(t, spans, 1, "fixture should contain exactly one span")
				span = spans[0]
			} else {
				var matches []*tracev1.Span
				for _, s := range spans {
					if s.Name == tc.spanName {
						matches = append(matches, s)
					}
				}
				require.Len(t, matches, 1, "expected exactly one %q span in %s", tc.spanName, tc.fixture)
				span = matches[0]
			}

			structured, err := extractors.NewAIMetadataExtractor().
				ExtractSpanMetadata(context.Background(), span)
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
