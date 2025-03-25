package aigateway

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseInput(t *testing.T) {
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
		out, err := ParseOutput(context.Background(), test.format, []byte(test.input))
		require.EqualValues(t, test.expected, out, test.description)
		require.EqualValues(t, test.err, err, test.description)
	}

}
