package util

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid JSON is unchanged",
			input:    `{"ok":true}`,
			expected: `{"ok":true}`,
		},
		{
			name:     "empty input",
			input:    "",
			expected: `""`,
		},
		{
			name:     "plain text",
			input:    "hello world",
			expected: `"hello world"`,
		},
		{
			name:     "ANSI color codes",
			input:    "\x1b[31mred\x1b[0m",
			expected: `"\u001b[31mred\u001b[0m"`,
		},
		{
			name:     "control characters without a JSON short escape",
			input:    "a\ab\vc",
			expected: `"a\u0007b\u000bc"`,
		},
		{
			name:     "invalid UTF-8",
			input:    "bad \xff byte",
			expected: `"bad � byte"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := EnsureJSON(json.RawMessage(tt.input))
			require.JSONEq(t, tt.expected, string(actual))

			// Callers embed the result in API responses, which fails to
			// marshal if the raw message isn't valid JSON.
			_, err := json.Marshal(struct {
				Output json.RawMessage `json:"output"`
			}{Output: actual})
			require.NoError(t, err)
		})
	}
}
