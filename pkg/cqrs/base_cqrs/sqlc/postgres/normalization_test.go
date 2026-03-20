package sqlc

import (
	"encoding/json"
	"testing"
)

func TestRawJSONToNullRawMessage(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		wantValid bool
		wantJSON  string
	}{
		{
			name:      "valid JSON object string",
			input:     `{"key":"value","nested":{"a":1}}`,
			wantValid: true,
			wantJSON:  `{"key":"value","nested":{"a":1}}`,
		},
		{
			name:      "valid JSON array string",
			input:     `["a","b","c"]`,
			wantValid: true,
			wantJSON:  `["a","b","c"]`,
		},
		{
			name:      "valid JSON bytes",
			input:     []byte(`{"key":"value"}`),
			wantValid: true,
			wantJSON:  `{"key":"value"}`,
		},
		{
			name:      "empty string",
			input:     "",
			wantValid: false,
		},
		{
			name:      "empty bytes",
			input:     []byte{},
			wantValid: false,
		},
		{
			name:      "nil input",
			input:     nil,
			wantValid: false,
		},
		{
			name:      "invalid JSON string",
			input:     `{not valid json`,
			wantValid: false,
		},
		{
			name:      "invalid JSON bytes",
			input:     []byte(`{not valid`),
			wantValid: false,
		},
		{
			name:      "fallback to marshal for map",
			input:     map[string]any{"key": "value"},
			wantValid: true,
			wantJSON:  `{"key":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rawJSONToNullRawMessage(tt.input)
			if result.Valid != tt.wantValid {
				t.Errorf("rawJSONToNullRawMessage() valid = %v, want %v", result.Valid, tt.wantValid)
			}
			if tt.wantValid {
				if !json.Valid(result.RawMessage) {
					t.Errorf("rawJSONToNullRawMessage() produced invalid JSON: %s", string(result.RawMessage))
				}
				if string(result.RawMessage) != tt.wantJSON {
					t.Errorf("rawJSONToNullRawMessage() = %s, want %s", string(result.RawMessage), tt.wantJSON)
				}
			}
		})
	}
}

func TestRawJSONToNullRawMessage_NoDoubleEncoding(t *testing.T) {
	// Simulate the exact flow: json.Marshal(attrs) -> string(byt) -> rawJSONToNullRawMessage
	attrs := map[string]any{"key": "value", "count": 42}
	byt, err := json.Marshal(attrs)
	if err != nil {
		t.Fatal(err)
	}

	result := rawJSONToNullRawMessage(string(byt))
	if !result.Valid {
		t.Fatal("expected valid result")
	}

	// The result should NOT start with a quote (which would indicate double encoding)
	if result.RawMessage[0] == '"' {
		t.Errorf("double encoding detected: result starts with quote: %s", string(result.RawMessage))
	}

	// Should be parseable as an object, not a string
	var parsed map[string]any
	if err := json.Unmarshal(result.RawMessage, &parsed); err != nil {
		t.Errorf("result is not a valid JSON object: %v (raw: %s)", err, string(result.RawMessage))
	}
}

func TestToNullRawMessage_StillWorksForGoValues(t *testing.T) {
	// Verify toNullRawMessage still works for Output/Input path (raw Go values)
	result := toNullRawMessage(map[string]any{"output": true})
	if !result.Valid {
		t.Fatal("expected valid result")
	}
	if !json.Valid(result.RawMessage) {
		t.Errorf("produced invalid JSON: %s", string(result.RawMessage))
	}

	var parsed map[string]any
	if err := json.Unmarshal(result.RawMessage, &parsed); err != nil {
		t.Errorf("result is not valid JSON object: %v", err)
	}
}
