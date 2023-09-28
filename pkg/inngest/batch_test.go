package inngest

import (
	"errors"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
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
			name: "should use default batch size if provided value is <= 0",
			data: map[string]any{
				"maxSize": -1,
				"timeout": "2s",
			},
			expected: &EventBatchConfig{
				MaxSize: consts.DefaultBatchSize,
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
			expectedError: errors.New("json: cannot unmarshal string into Go struct field EventBatchConfig.maxSize of type int"),
		},
		{
			name: "should return error with invalid timeout type",
			data: map[string]any{
				"maxSize": 10,
				"timeout": 10,
			},
			expectedError: errors.New("json: cannot unmarshal number into Go struct field EventBatchConfig.timeout of type string"),
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
		{
			name:     "should return false for batch of 1",
			config:   &EventBatchConfig{MaxSize: 1, Timeout: "2s"},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.config.IsEnabled())
		})
	}
}

func TestEventBatchConfigIsValid(t *testing.T) {
	tests := []struct {
		name     string
		config   *EventBatchConfig
		expected error
	}{
		{
			name: "should return no error if valid",
			config: &EventBatchConfig{
				MaxSize: 10,
				Timeout: "10s",
			},
		},
		{
			name: "should return error if MaxSize is less than 2",
			config: &EventBatchConfig{
				MaxSize: 1,
			},
			expected: errors.New("batch size cannot be smaller than 2"),
		},
		{
			name: "should return error if timeout is invalid duration string",
			config: &EventBatchConfig{
				MaxSize: 10,
				Timeout: "10ss", // simulating typos
			},
			expected: errors.New("invalid timeout string"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.config.IsValid(); err != nil {
				require.ErrorContains(t, err, test.expected.Error())
			}
		})
	}
}
