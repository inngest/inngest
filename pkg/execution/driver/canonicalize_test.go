package driver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanonicalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			// The exact failure from the issue: a lone high surrogate escape
			// immediately followed by raw non-ASCII UTF-8 makes jcs read the
			// next byte with its ASCII-only reader ("Unexpected non-ASCII
			// character").
			name:     "lone high surrogate followed by non-ASCII",
			input:    `{"a":"\ud83dé"}`,
			expected: `{"a":"` + "�" + `é"}`,
		},
		{
			// Same family: jcs reports "Missing surrogate".
			name:     "lone high surrogate at end of string",
			input:    `{"a":"\ud83d"}`,
			expected: `{"a":"` + "�" + `"}`,
		},
		{
			name:     "lone high surrogate followed by ASCII",
			input:    `{"a":"\ud83dx"}`,
			expected: `{"a":"` + "�" + `x"}`,
		},
		{
			name:     "lone low surrogate",
			input:    `{"a":"\udc00x"}`,
			expected: `{"a":"` + "�" + `x"}`,
		},
		{
			name:     "uppercase hex lone surrogate",
			input:    `{"a":"\uD83D"}`,
			expected: `{"a":"` + "�" + `"}`,
		},
		{
			name:     "lone surrogate in an object key",
			input:    `{"\ud83d":1}`,
			expected: `{"` + "�" + `":1}`,
		},
		{
			// A valid escaped pair must be preserved, decoded to its code point.
			name:     "valid surrogate pair is untouched",
			input:    `{"a":"\ud83d\ude00"}`,
			expected: `{"a":"😀"}`,
		},
		{
			name:     "valid pair followed by a lone high surrogate",
			input:    `{"a":"\ud83d\ude00\ud83d"}`,
			expected: `{"a":"😀` + "�" + `"}`,
		},
		{
			// `\\ud83d` is an escaped backslash followed by the literal text
			// "ud83d", not a surrogate escape. It must pass through as-is.
			name:     "escaped backslash decoy",
			input:    `{"a":"\\ud83d"}`,
			expected: `{"a":"\\ud83d"}`,
		},
		{
			// Plain UTF-8 was never affected; first-try path.
			name:     "plain accented text",
			input:    `{"a":"péché"}`,
			expected: `{"a":"péché"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := canonicalize([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(out))
		})
	}
}

func TestCanonicalizeInvalidJSON(t *testing.T) {
	_, err := canonicalize([]byte(`{"a":`))
	require.Error(t, err)
}

func TestRepairUnpairedSurrogateEscapes(t *testing.T) {
	// No repairs needed returns nil so callers can skip the retry.
	assert.Nil(t, repairUnpairedSurrogateEscapes([]byte(`{"a":"😀"}`)))
	assert.Nil(t, repairUnpairedSurrogateEscapes([]byte(`{"a":"plain"}`)))
	assert.Nil(t, repairUnpairedSurrogateEscapes([]byte(`{"a":"\\ud83d"}`)))

	// Repairs emit the ASCII-safe `\ufffd` escape; jcs decodes it on the retry.
	assert.Equal(t,
		`{"a":"\ufffd"}`,
		string(repairUnpairedSurrogateEscapes([]byte(`{"a":"\ud83d"}`))),
	)
}
