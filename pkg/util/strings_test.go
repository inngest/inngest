package util

import "testing"

func TestSanitizeLogField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes carriage returns",
			input:    "hello\rworld",
			expected: "helloworld",
		},
		{
			name:     "removes newlines",
			input:    "hello\nworld",
			expected: "helloworld",
		},
		{
			name:     "removes both carriage returns and newlines",
			input:    "hello\r\nworld",
			expected: "helloworld",
		},
		{
			name:     "preserves other special characters",
			input:    "user@domain.com/path-name_file[0]%20",
			expected: "user@domain.com/path-name_file[0]%20",
		},
		{
			name:     "string without special characters",
			input:    "hello world",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeLogField(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeLogField(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}