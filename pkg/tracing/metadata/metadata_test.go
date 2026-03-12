package metadata

import (
	"encoding/json"
	"testing"
)

func TestValuesSize(t *testing.T) {
	tests := []struct {
		name     string
		values   Values
		expected int
	}{
		{
			name:     "empty values",
			values:   Values{},
			expected: 0,
		},
		{
			name: "single key-value",
			values: Values{
				"key": json.RawMessage(`"value"`),
			},
			expected: len("key") + len(`"value"`),
		},
		{
			name: "multiple key-values",
			values: Values{
				"alpha": json.RawMessage(`"one"`),
				"beta":  json.RawMessage(`{"nested":true}`),
			},
			expected: len("alpha") + len(`"one"`) + len("beta") + len(`{"nested":true}`),
		},
		{
			name: "nil json value",
			values: Values{
				"key": json.RawMessage(nil),
			},
			expected: len("key"),
		},
		{
			name: "empty json value",
			values: Values{
				"key": json.RawMessage{},
			},
			expected: len("key"),
		},
		{
			name: "realistic metadata payload",
			values: Values{
				"model":        json.RawMessage(`"gpt-4"`),
				"prompt":       json.RawMessage(`"Tell me about Go programming"`),
				"completion":   json.RawMessage(`"Go is a statically typed language designed at Google."`),
				"tokens_used":  json.RawMessage(`150`),
				"latency_ms":   json.RawMessage(`432`),
			},
			expected: len("model") + len(`"gpt-4"`) +
				len("prompt") + len(`"Tell me about Go programming"`) +
				len("completion") + len(`"Go is a statically typed language designed at Google."`) +
				len("tokens_used") + len(`150`) +
				len("latency_ms") + len(`432`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.values.Size()
			if got != tt.expected {
				t.Errorf("Values.Size() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestValuesSizeNilMap(t *testing.T) {
	var v Values
	if got := v.Size(); got != 0 {
		t.Errorf("nil Values.Size() = %d, want 0", got)
	}
}
