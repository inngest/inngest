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

func TestMerge(t *testing.T) {
	tests := []struct {
		from     map[string]interface{}
		to       map[string]interface{}
		expected map[string]interface{}
	}{
		// keeps data
		{
			from:     map[string]interface{}{},
			to:       map[string]interface{}{"lol": "indeed"},
			expected: map[string]interface{}{"lol": "indeed"},
		},
		// adds data
		{
			from: map[string]interface{}{
				"test": "bigfoot scampers with a torn umberella",
			},
			to: map[string]interface{}{},
			expected: map[string]interface{}{
				"test": "bigfoot scampers with a torn umberella",
			},
		},
		// overwrites
		{
			from: map[string]interface{}{
				"test": "bigfoot scampers with a torn umberella",
			},
			to: map[string]interface{}{
				"test": "buckets are always leaky",
			},
			expected: map[string]interface{}{
				"test": "bigfoot scampers with a torn umberella",
			},
		},
		// nested, plain
		{
			from: map[string]interface{}{
				"test": "bigfoot scampers with a torn umberella",
				"nested": map[string]interface{}{
					"good ol merge": 1,
				},
			},
			to: map[string]interface{}{},
			expected: map[string]interface{}{
				"test": "bigfoot scampers with a torn umberella",
				"nested": map[string]interface{}{
					"good ol merge": 1,
				},
			},
		},
		// nested, overwriting
		{
			from: map[string]interface{}{
				"test": "bigfoot scampers with a torn umberella",
				"nested": map[string]interface{}{
					"overwritten": 1,
				},
			},
			to: map[string]interface{}{
				"nested": map[string]interface{}{
					"overwritten": "lol",
				},
			},
			expected: map[string]interface{}{
				"test": "bigfoot scampers with a torn umberella",
				"nested": map[string]interface{}{
					"overwritten": 1,
				},
			},
		},
		// nested, overwriting non-maps
		{
			from: map[string]interface{}{
				"test": "bigfoot scampers with a torn umberella",
				"nested": map[string]interface{}{
					"non-map": 1,
				},
			},
			to: map[string]interface{}{
				"nested": "wut",
			},
			expected: map[string]interface{}{
				"test": "bigfoot scampers with a torn umberella",
				"nested": map[string]interface{}{
					"non-map": 1,
				},
			},
		},
	}

	for _, test := range tests {
		merge(test.to, test.from)
		require.EqualValues(t, test.expected, test.to)
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
