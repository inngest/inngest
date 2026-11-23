package aigateway

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseInput(t *testing.T) {
	weatherSchema := `{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}`

	tests := []struct {
		description string
		format      string
		body        string
		expected    ParsedInferenceRequest
	}{
		{
			description: "openai plain request without tools",
			format:      FormatOpenAIChat,
			body:        `{"model":"gpt-4o","max_tokens":100,"temperature":0.5,"top_p":0.9,"stop":["END"],"messages":[{"role":"user","content":"hi"}]}`,
			expected: ParsedInferenceRequest{
				URL:           "https://api.openai.com/v1/chat/completions",
				Model:         "gpt-4o",
				MaxTokens:     100,
				Temprature:    0.5,
				TopP:          0.9,
				StopSequences: []string{"END"},
				Tools:         []ToolUseRequest{},
			},
		},
		{
			description: "openai string tool choice",
			format:      FormatOpenAIChat,
			body:        `{"model":"gpt-4o","tool_choice":"auto","messages":[]}`,
			expected: ParsedInferenceRequest{
				URL:        "https://api.openai.com/v1/chat/completions",
				Model:      "gpt-4o",
				ToolChoice: "auto",
				Tools:      []ToolUseRequest{},
			},
		},
		{
			description: "openai forced tool choice records the function name and tool schema",
			format:      FormatOpenAIChat,
			body: `{"model":"gpt-4o","messages":[],
				"tool_choice":{"type":"function","function":{"name":"get_weather"}},
				"tools":[{"type":"function","function":{"name":"get_weather","description":"Get the weather","parameters":` + weatherSchema + `}}]}`,
			expected: ParsedInferenceRequest{
				URL:        "https://api.openai.com/v1/chat/completions",
				Model:      "gpt-4o",
				ToolChoice: "get_weather",
				Tools: []ToolUseRequest{
					{
						Name:        "get_weather",
						Description: "Get the weather",
						Parameters:  json.RawMessage(weatherSchema),
					},
				},
			},
		},
		{
			description: "openai tools without a function are skipped rather than recorded blank",
			format:      FormatOpenAIChat,
			body:        `{"model":"gpt-4o","messages":[],"tools":[{"type":"code_interpreter"},{"type":"function","function":{"name":"a","parameters":{}}}]}`,
			expected: ParsedInferenceRequest{
				URL:   "https://api.openai.com/v1/chat/completions",
				Model: "gpt-4o",
				Tools: []ToolUseRequest{
					{Name: "a", Parameters: json.RawMessage(`{}`)},
				},
			},
		},
		{
			description: "anthropic request with a forced tool",
			format:      FormatAnthropic,
			body: `{"model":"claude-sonnet-4-5","max_tokens":1024,"temperature":0.2,"stop_sequences":["STOP"],"messages":[],
				"tool_choice":{"type":"tool","name":"get_weather"},
				"tools":[{"name":"get_weather","description":"Get the weather","input_schema":` + weatherSchema + `}]}`,
			expected: ParsedInferenceRequest{
				URL:           "https://api.anthropic.com/v1/messages",
				Model:         "claude-sonnet-4-5",
				MaxTokens:     1024,
				Temprature:    0.2,
				StopSequences: []string{"STOP"},
				ToolChoice:    "get_weather",
				Tools: []ToolUseRequest{
					{
						Name:        "get_weather",
						Description: "Get the weather",
						Parameters:  json.RawMessage(weatherSchema),
					},
				},
			},
		},
		{
			description: "anthropic request with an automatic tool choice",
			format:      FormatAnthropic,
			body:        `{"model":"claude-sonnet-4-5","messages":[],"tool_choice":{"type":"auto"}}`,
			expected: ParsedInferenceRequest{
				URL:        "https://api.anthropic.com/v1/messages",
				Model:      "claude-sonnet-4-5",
				ToolChoice: "auto",
				Tools:      []ToolUseRequest{},
			},
		},
	}

	urls := map[string]string{
		FormatOpenAIChat: "https://api.openai.com/v1/chat/completions",
		FormatAnthropic:  "https://api.anthropic.com/v1/messages",
	}

	for _, test := range tests {
		out, err := ParseInput(Request{
			URL:    urls[test.format],
			Format: test.format,
			Body:   json.RawMessage(test.body),
		})
		require.NoError(t, err, test.description)
		require.Len(t, out.Tools, len(test.expected.Tools), test.description)
		for i, tool := range test.expected.Tools {
			require.Equal(t, tool.Name, out.Tools[i].Name, test.description)
			require.Equal(t, tool.Description, out.Tools[i].Description, test.description)
			// Compare schemas as JSON, as map keys are re-marshalled in sorted order.
			require.JSONEq(t, string(tool.Parameters), string(out.Tools[i].Parameters), test.description)
		}

		// Tool schemas were compared above, so don't compare their bytes again.
		out.Tools, test.expected.Tools = nil, nil
		require.EqualValues(t, test.expected, out, test.description)
	}

	t.Run("tool schema numbers are not rounded", func(t *testing.T) {
		// 2^53 + 1 cannot be represented as a float64.
		out, err := ParseInput(Request{
			Format: FormatOpenAIChat,
			Body:   json.RawMessage(`{"model":"gpt-4o","messages":[],"tools":[{"type":"function","function":{"name":"a","parameters":{"enum":[9007199254740993]}}}]}`),
		})
		require.NoError(t, err)
		require.Len(t, out.Tools, 1)
		require.Equal(t, `{"enum":[9007199254740993]}`, string(out.Tools[0].Parameters))
	})

	t.Run("invalid json returns an error", func(t *testing.T) {
		for _, format := range []string{FormatOpenAIChat, FormatAnthropic} {
			_, err := ParseInput(Request{Format: format, Body: json.RawMessage(`{not json`)})
			require.Error(t, err, format)
		}
	})
}

func TestParseOutput(t *testing.T) {

	tests := []struct {
		description string
		format      string
		input       string
		expected    ParsedInferenceResponse
		err         error
	}{
		{
			description: "anthropic msg response with content and tool",
			format:      FormatAnthropic,
			input:       `{"id":"msg_01AWkeVsTqhLbsS2MDmjGJxj","type":"message","role":"assistant","model":"claude-3-5-haiku-20241022","content":[{"type":"text","text":"I see 'pvlib/pvsystem.py' looks relevant. Let's examine the exact code for the PVSystem class:"},{"type":"tool_use","id":"toolu_014WnC1qhZZRtKf4kENoVEnx","name":"read_file","input":{"filename":"pvlib/pvsystem.py"}}],"stop_reason":"tool_use","stop_sequence":null,"usage":{"input_tokens":7947,"output_tokens":90}} `,
			expected: ParsedInferenceResponse{
				ID:         "msg_01AWkeVsTqhLbsS2MDmjGJxj",
				Model:      "claude-3-5-haiku-20241022",
				TokensIn:   7947,
				TokensOut:  90,
				StopReason: "tool_use",
				Tools: []ToolUseResponse{
					{
						ID:        "toolu_014WnC1qhZZRtKf4kENoVEnx",
						Name:      "read_file",
						Arguments: `{"filename":"pvlib/pvsystem.py"}`,
					},
				},
			},
		},
		{
			description: "anthropic rate limit error",
			format:      FormatAnthropic,
			input:       `{"type":"error","error":{"type":"rate_limit_error","message":"This request would exceed your organization’s rate limit of 50,000 input tokens per minute. For details, refer to: https://docs.anthropic.com/en/api/rate-limits; see the response headers for current usage. Please reduce the prompt length or the maximum tokens requested, or try again later. You may also contact sales at https://www.anthropic.com/contact-sales to discuss your options for a rate limit increase."}}`,
			expected: ParsedInferenceResponse{
				Error: "rate_limit_error",
			},
			err: fmt.Errorf("anthropic api error: rate_limit_error"),
		},
	}

	for _, test := range tests {
		out, err := ParseOutput(test.format, []byte(test.input))
		require.EqualValues(t, test.expected, out, test.description)
		require.EqualValues(t, test.err, err, test.description)
	}

}
