package extractors

import (
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/pkg/util/aigateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func extractGatewayAIMetadata(t *testing.T, req aigateway.Request, resp []byte, serverProcessingMs int64) *AIMetadata {
	t.Helper()

	md, err := ExtractAIGatewayMetadata(req, 200, resp, serverProcessingMs)
	require.NoError(t, err)
	require.Len(t, md, 2, "Expected AI and HTTP metadata")

	aiMd, ok := md[0].(*AIMetadata)
	require.True(t, ok, "Expected first entry to be AIMetadata")
	return aiMd
}

func TestExtractAIGatewayMetadata_OpenAIChatCompletion(t *testing.T) {
	t.Parallel()

	req := aigateway.Request{
		URL:    "https://api.openai.com/v1/chat/completions",
		Format: aigateway.FormatOpenAIChat,
		Body: json.RawMessage(`{
			"model": "gpt-4o",
			"messages": [{"role": "user", "content": "Hello"}]
		}`),
	}
	resp := []byte(`{
		"id": "chatcmpl-abc123",
		"object": "chat.completion",
		"created": 1741569952,
		"model": "gpt-4o-2024-08-06",
		"choices": [
			{
				"index": 0,
				"message": {"role": "assistant", "content": "Hi there!"},
				"finish_reason": "stop"
			}
		],
		"usage": {"prompt_tokens": 25, "completion_tokens": 50, "total_tokens": 75}
	}`)

	aiMd := extractGatewayAIMetadata(t, req, resp, 1234)

	assert.Equal(t, "gpt-4o", aiMd.RequestModel)
	assert.Equal(t, "gpt-4o-2024-08-06", aiMd.ResponseModel)
	assert.Equal(t, "chatcmpl-abc123", aiMd.ResponseID)
	assert.Equal(t, []string{"stop"}, aiMd.FinishReasons)
	assert.Equal(t, "openai-chat", aiMd.Provider)
	assert.Equal(t, int64(25), aiMd.InputTokens)
	assert.Equal(t, int64(50), aiMd.OutputTokens)
	assert.Equal(t, util.ToPtr[int64](75), aiMd.TotalTokens)
	assert.NotNil(t, aiMd.EstimatedCost)
	assert.Equal(t, util.ToPtr[int64](1234), aiMd.LatencyMs)
}

func TestExtractAIGatewayMetadata_AnthropicFormatFallback(t *testing.T) {
	t.Parallel()

	req := aigateway.Request{
		URL:    "https://llm-proxy.internal.example.com/v1/messages",
		Format: aigateway.FormatAnthropic,
		Body:   json.RawMessage(`{"model": "claude-sonnet-4-5", "max_tokens": 512}`),
	}
	resp := []byte(`{
		"id": "msg_01XYZ",
		"type": "message",
		"role": "assistant",
		"model": "claude-sonnet-4-5",
		"content": [{"type": "text", "text": "Hello"}],
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 30, "output_tokens": 60}
	}`)

	aiMd := extractGatewayAIMetadata(t, req, resp, 0)

	assert.Equal(t, "anthropic", aiMd.Provider)
	assert.Equal(t, "msg_01XYZ", aiMd.ResponseID)
	assert.Equal(t, []string{"end_turn"}, aiMd.FinishReasons)
	assert.Equal(t, int64(30), aiMd.InputTokens)
	assert.Equal(t, int64(60), aiMd.OutputTokens)
	assert.Equal(t, util.ToPtr[int64](90), aiMd.TotalTokens)
	assert.NotNil(t, aiMd.EstimatedCost)
	assert.Nil(t, aiMd.LatencyMs, "Should not report latency when server processing time is unknown")
}

func TestExtractAIGatewayMetadata_RequestParams(t *testing.T) {
	t.Parallel()

	resp := []byte(`{
		"id": "chatcmpl-1",
		"model": "gpt-4o",
		"choices": [{"index": 0, "message": {"role": "assistant", "content": "ok"}, "finish_reason": "stop"}],
		"usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
	}`)

	cases := []struct {
		name          string
		body          string
		wantTemp      *float64
		wantTopP      *float64
		wantMaxTokens *int64
		wantSeed      *int64
	}{
		{
			name:          "all params set",
			body:          `{"model": "gpt-4o", "temperature": 0.5, "top_p": 0.75, "max_tokens": 1024, "seed": 42}`,
			wantTemp:      util.ToPtr(0.5),
			wantTopP:      util.ToPtr(0.75),
			wantMaxTokens: util.ToPtr[int64](1024),
			wantSeed:      util.ToPtr[int64](42),
		},
		{
			name:          "max tokens falls back to max completion tokens",
			body:          `{"model": "gpt-4o", "max_completion_tokens": 2048}`,
			wantMaxTokens: util.ToPtr[int64](2048),
		},
		{
			name:     "seed zero is preserved",
			body:     `{"model": "gpt-4o", "seed": 0}`,
			wantSeed: util.ToPtr[int64](0),
		},
		{
			name:     "temperature and top_p zero are preserved",
			body:     `{"model": "gpt-4o", "temperature": 0, "top_p": 0}`,
			wantTemp: util.ToPtr(0.0),
			wantTopP: util.ToPtr(0.0),
		},
		{
			name: "null params stay nil",
			body: `{"model": "gpt-4o", "temperature": null, "top_p": null}`,
		},
		{
			name: "absent params stay nil",
			body: `{"model": "gpt-4o"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := aigateway.Request{
				URL:    "https://api.openai.com/v1/chat/completions",
				Format: aigateway.FormatOpenAIChat,
				Body:   json.RawMessage(tc.body),
			}

			aiMd := extractGatewayAIMetadata(t, req, resp, 0)

			assert.Equal(t, tc.wantTemp, aiMd.Temperature)
			assert.Equal(t, tc.wantTopP, aiMd.TopP)
			assert.Equal(t, tc.wantMaxTokens, aiMd.MaxTokens)
			assert.Equal(t, tc.wantSeed, aiMd.Seed)
		})
	}
}

func TestExtractAIGatewayMetadata_UnknownModelHasNoEstimatedCost(t *testing.T) {
	t.Parallel()

	req := aigateway.Request{
		URL:    "https://my-llm.internal:8080/v1/chat/completions",
		Format: aigateway.FormatOpenAIChat,
		Body:   json.RawMessage(`{"model": "local-model"}`),
	}
	resp := []byte(`{
		"id": "chatcmpl-2",
		"model": "local-model",
		"choices": [{"index": 0, "message": {"role": "assistant", "content": "ok"}, "finish_reason": "stop"}],
		"usage": {"prompt_tokens": 5, "completion_tokens": 5, "total_tokens": 10}
	}`)

	aiMd := extractGatewayAIMetadata(t, req, resp, 0)

	assert.Nil(t, aiMd.EstimatedCost, "Should not estimate cost for unknown models")
}

func TestExtractAIGatewayMetadata_NoFinishReasonsWhenStopReasonEmpty(t *testing.T) {
	t.Parallel()

	req := aigateway.Request{
		URL:    "https://api.openai.com/v1/chat/completions",
		Format: aigateway.FormatOpenAIChat,
		Body:   json.RawMessage(`{"model": "gpt-4o"}`),
	}
	resp := []byte(`{
		"id": "chatcmpl-3",
		"model": "gpt-4o",
		"choices": [{"index": 0, "message": {"role": "assistant", "content": "ok"}}],
		"usage": {"prompt_tokens": 5, "completion_tokens": 5, "total_tokens": 10}
	}`)

	aiMd := extractGatewayAIMetadata(t, req, resp, 0)

	assert.Nil(t, aiMd.FinishReasons)
}
