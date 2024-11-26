package aigateway

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseVercelGenerateText(t *testing.T) {
	inputs := []struct {
		input    json.RawMessage
		expected ParsedInferenceRequest
		err      error
	}{
		{
			// most basic demo
			input: json.RawMessage(`[{"model":{"config":{"compatibility":"strict","provider":"openai.chat"},"modelId":"gpt-4-turbo","settings":{},"specificationVersion":"v1"},"prompt":"What is love?"}]`),
			expected: ParsedInferenceRequest{
				Model: "gpt-4-turbo",
			},
		},
		{
			// system prompt
			input: json.RawMessage(`[{"model":{"config":{"compatibility":"strict","provider":"openai.chat"},"modelId":"gpt-4-turbo","settings":{},"specificationVersion":"v1"},"prompt":"What is love?"}]`),
			expected: ParsedInferenceRequest{
				Model: "gpt-4-turbo",
			},
		},
		{
			// settings
			input: json.RawMessage(`[{"maxTokens":512,"model":{"config":{"compatibility":"strict","provider":"openai.chat"},"modelId":"gpt-4-turbo","settings":{},"specificationVersion":"v1"},"prompt":"What is love?","seed":123,"stopSequences":["\u003cstop /\u003e"],"system":"you are helpful and terse.","temperature":0.3,"topK":50,"topP":0.75}]`),
			expected: ParsedInferenceRequest{
				Model:         "gpt-4-turbo",
				Seed:          ptr(123),
				Temprature:    0.3,
				TopP:          0.75,
				TopK:          50,
				MaxTokens:     512,
				StopSequences: []string{"<stop />"},
			},
		},
	}

	for _, test := range inputs {
		actual, err := parseVercelGenerateTextArgs(test.input)
		require.EqualValues(t, test.err, err)
		require.EqualValues(t, test.expected, actual)
	}
}

func ptr[T any](in T) *T {
	return &in
}
