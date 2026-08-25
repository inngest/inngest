package expressions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapify(t *testing.T) {
	tests := []struct {
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			input: map[string]interface{}{
				"hi": "yea",
			},
			expected: map[string]interface{}{
				"hi": "yea",
			},
		},
		{
			input: map[string]interface{}{
				"hi":  "yea",
				"nil": nil,
			},
			expected: map[string]interface{}{
				"hi":  "yea",
				"nil": nil,
			},
		},
		{
			input: map[string]interface{}{
				"hi": "yea",
				"nested": map[string]interface{}{
					"struct": struct {
						Value string `json:"val"`
					}{Value: "somestr"},
				},
			},
			expected: map[string]interface{}{
				"hi": "yea",
				"nested": map[string]interface{}{
					"struct": map[string]interface{}{"val": "somestr"},
				},
			},
		},
	}

	for _, test := range tests {
		actual := mapify(test.input)
		require.EqualValues(t, test.expected, actual)
	}
}

func TestPathExists(t *testing.T) {
	tests := []struct {
		path     []string
		data     map[string]interface{}
		expected bool
	}{
		{
			path: []string{"event", "data", "issue", "fields", "tags"},
			data: map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"issue": struct {
							Fields interface{} `json:"fields"`
						}{
							Fields: map[string]interface{}{
								"tags": []string{"a", "b"},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			path: []string{"event", "data", "issue", "fields", "tags", "wut"},
			data: map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"issue": struct {
							Fields interface{} `json:"fields"`
						}{
							Fields: map[string]interface{}{
								"tags": []string{"a", "b"},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			path: []string{"event", "Map"},
			data: map[string]interface{}{
				"event": struct {
					Map map[string]interface{}
				}{},
			},
			expected: true,
		},
		{
			path: []string{"event", "Map", "foo"},
			data: map[string]interface{}{
				"event": struct {
					Map map[string]interface{}
				}{},
			},
			expected: false,
		},
		{
			path: []string{"event", "lol", "wut"},
			data: map[string]interface{}{
				"event": map[string]interface{}{},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		ok := NewData(test.data).PathExists(context.Background(), test.path)
		require.Equal(t, test.expected, ok, "path: %s", test.path)
	}
}
