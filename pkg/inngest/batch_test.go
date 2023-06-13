package inngest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventBatchIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *EventBatchConfig
		expected bool
	}{
		{
			name:     "should return true for non zero config",
			config:   &EventBatchConfig{MaxSize: 10, Timeout: "2s"},
			expected: true,
		},
		{
			name:     "should return false for empty config",
			config:   &EventBatchConfig{},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.config.IsEnabled())
		})
	}
}
