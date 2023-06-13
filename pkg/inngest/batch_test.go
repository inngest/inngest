package inngest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEventBatchConfig(t *testing.T) {
	tests := []struct {
		name          string
		data          map[string]any
		expected      *EventBatchConfig
		expectedError error
	}{
		{
			name: "should return config with valid data",
			data: map[string]any{
				"maxSize": 10,
				"timeout": "2s",
			},
			expected: &EventBatchConfig{
				MaxSize: 10,
				Timeout: "2s",
			},
		},
		{
			name:     "should return nil without errors if data is empty",
			data:     nil,
			expected: nil,
		},
		{
			name: "should return error with invalid size type",
			data: map[string]any{
				"maxSize": "yolo",
			},
			expectedError: errors.New("unexpected type for MaxSize:"),
		},
		{
			name: "should return error with invalid timeout type",
			data: map[string]any{
				"maxSize": 10,
				"timeout": 10,
			},
			expectedError: errors.New("unexpected type for Timeout:"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := NewEventBatchConfig(test.data)
			require.Equal(t, test.expected, config)

			if err != nil {
				require.ErrorContains(t, err, test.expectedError.Error())
			}
		})
	}
}

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
