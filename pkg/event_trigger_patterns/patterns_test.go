package event_trigger_patterns

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateMatchingPatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "app/user.created example",
			input:    "app/user.created",
			expected: []string{"app/user.created", "app/*", "app/user.*"},
		},
		{
			name:     "api/v1/users example",
			input:    "api/v1/users",
			expected: []string{"api/v1/users", "api/*", "api/v1/*"},
		},
		{
			name:     "user.updated example",
			input:    "user.updated",
			expected: []string{"user.updated", "user.*"},
		},
		{
			name:     "simple event name without delimiters",
			input:    "simple",
			expected: []string{"simple"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateMatchingPatterns(tt.input)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

