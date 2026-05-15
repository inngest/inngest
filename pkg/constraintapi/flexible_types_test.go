package constraintapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlexibleIntArray_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []int
		wantErr  bool
	}{
		{
			name:     "empty array",
			input:    "[]",
			expected: []int{},
			wantErr:  false,
		},
		{
			name:     "empty object (from Lua cjson.empty_array)",
			input:    "{}",
			expected: []int{},
			wantErr:  false,
		},
		{
			name:     "single element array",
			input:    "[1]",
			expected: []int{1},
			wantErr:  false,
		},
		{
			name:     "multiple elements",
			input:    "[1, 2, 3]",
			expected: []int{1, 2, 3},
			wantErr:  false,
		},
		{
			name:     "array with zeros",
			input:    "[0, 1, 0]",
			expected: []int{0, 1, 0},
			wantErr:  false,
		},
		{
			name:     "array with negative numbers",
			input:    "[-1, 2, -3]",
			expected: []int{-1, 2, -3},
			wantErr:  false,
		},
		{
			name:     "large numbers",
			input:    "[999999, 1000000]",
			expected: []int{999999, 1000000},
			wantErr:  false,
		},
		{
			name:    "invalid JSON",
			input:   "[1, 2",
			wantErr: true,
		},
		{
			name:    "non-empty object",
			input:   `{"key": "value"}`,
			wantErr: true,
		},
		{
			name:    "object with integer values",
			input:   `{"a": 1, "b": 2}`,
			wantErr: true,
		},
		{
			name:    "string instead of array",
			input:   `"not an array"`,
			wantErr: true,
		},
		{
			name:     "null value",
			input:    "null",
			wantErr:  false,
			expected: nil,
		},
		{
			name:    "array with mixed types",
			input:   `[1, "string", true]`,
			wantErr: true,
		},
		{
			name:    "array with floats (should fail since we expect ints)",
			input:   "[1.5, 2.7]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var arr flexibleIntArray
			err := json.Unmarshal([]byte(tt.input), &arr)

			if tt.wantErr {
				require.Error(t, err, "Expected error for input: %s", tt.input)
				return
			}

			require.NoError(t, err, "Unexpected error for input: %s", tt.input)
			require.Equal(t, tt.expected, []int(arr), "Unexpected result for input: %s", tt.input)
		})
	}
}

func TestFlexibleIntArray_InStruct(t *testing.T) {
	type TestStruct struct {
		Numbers flexibleIntArray `json:"numbers"`
		Name    string           `json:"name"`
	}

	tests := []struct {
		name     string
		input    string
		expected TestStruct
		wantErr  bool
	}{
		{
			name:  "struct with empty array",
			input: `{"numbers": [], "name": "test"}`,
			expected: TestStruct{
				Numbers: flexibleIntArray{},
				Name:    "test",
			},
		},
		{
			name:  "struct with empty object",
			input: `{"numbers": {}, "name": "test"}`,
			expected: TestStruct{
				Numbers: flexibleIntArray{},
				Name:    "test",
			},
		},
		{
			name:  "struct with populated array",
			input: `{"numbers": [1, 2, 3], "name": "test"}`,
			expected: TestStruct{
				Numbers: flexibleIntArray{1, 2, 3},
				Name:    "test",
			},
		},
		{
			name:    "struct with invalid numbers field",
			input:   `{"numbers": "invalid", "name": "test"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result TestStruct
			err := json.Unmarshal([]byte(tt.input), &result)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFlexibleStringArray_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		wantErr  bool
	}{
		{
			name:     "empty array",
			input:    "[]",
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "empty object (from Lua cjson.empty_array)",
			input:    "{}",
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "single element array",
			input:    `["hello"]`,
			expected: []string{"hello"},
			wantErr:  false,
		},
		{
			name:     "multiple elements",
			input:    `["hello", "world", "test"]`,
			expected: []string{"hello", "world", "test"},
			wantErr:  false,
		},
		{
			name:     "empty strings",
			input:    `["", "test", ""]`,
			expected: []string{"", "test", ""},
			wantErr:  false,
		},
		{
			name:     "strings with special characters",
			input:    `["hello\nworld", "test\"quote", "unicode: 🎉"]`,
			expected: []string{"hello\nworld", "test\"quote", "unicode: 🎉"},
			wantErr:  false,
		},
		{
			name:     "strings with spaces and punctuation",
			input:    `["hello world!", "test, another", "final."]`,
			expected: []string{"hello world!", "test, another", "final."},
			wantErr:  false,
		},
		{
			name:    "invalid JSON",
			input:   `["hello", "world"`,
			wantErr: true,
		},
		{
			name:    "non-empty object",
			input:   `{"key": "value"}`,
			wantErr: true,
		},
		{
			name:    "object with string values",
			input:   `{"a": "hello", "b": "world"}`,
			wantErr: true,
		},
		{
			name:    "number instead of array",
			input:   "123",
			wantErr: true,
		},
		{
			name:     "null value",
			input:    "null",
			wantErr:  false,
			expected: nil,
		},
		{
			name:    "array with mixed types",
			input:   `["string", 123, true]`,
			wantErr: true,
		},
		{
			name:    "array with numbers only",
			input:   "[1, 2, 3]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var arr flexibleStringArray
			err := json.Unmarshal([]byte(tt.input), &arr)

			if tt.wantErr {
				require.Error(t, err, "Expected error for input: %s", tt.input)
				return
			}

			require.NoError(t, err, "Unexpected error for input: %s", tt.input)
			require.Equal(t, tt.expected, []string(arr), "Unexpected result for input: %s", tt.input)
		})
	}
}

func TestFlexibleStringArray_InStruct(t *testing.T) {
	type TestStruct struct {
		Messages flexibleStringArray `json:"messages"`
		Count    int                 `json:"count"`
	}

	tests := []struct {
		name     string
		input    string
		expected TestStruct
		wantErr  bool
	}{
		{
			name:  "struct with empty array",
			input: `{"messages": [], "count": 0}`,
			expected: TestStruct{
				Messages: flexibleStringArray{},
				Count:    0,
			},
		},
		{
			name:  "struct with empty object",
			input: `{"messages": {}, "count": 0}`,
			expected: TestStruct{
				Messages: flexibleStringArray{},
				Count:    0,
			},
		},
		{
			name:  "struct with populated array",
			input: `{"messages": ["debug", "info", "error"], "count": 3}`,
			expected: TestStruct{
				Messages: flexibleStringArray{"debug", "info", "error"},
				Count:    3,
			},
		},
		{
			name:    "struct with invalid messages field",
			input:   `{"messages": 123, "count": 0}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result TestStruct
			err := json.Unmarshal([]byte(tt.input), &result)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFlexibleTypes_RealWorldScenarios(t *testing.T) {
	// Test scenarios that mimic actual Lua script responses
	t.Run("acquire script response with empty arrays", func(t *testing.T) {
		type MockAcquireResponse struct {
			Status              int                 `json:"s"`
			LimitingConstraints flexibleIntArray    `json:"lc"`
			Debug               flexibleStringArray `json:"d"`
		}

		// Test with cjson.empty_array (empty objects)
		input := `{"s": 3, "lc": {}, "d": {}}`
		var resp MockAcquireResponse
		err := json.Unmarshal([]byte(input), &resp)

		require.NoError(t, err)
		require.Equal(t, 3, resp.Status)
		require.Equal(t, []int{}, []int(resp.LimitingConstraints))
		require.Equal(t, []string{}, []string(resp.Debug))
	})

	t.Run("check script response with populated arrays", func(t *testing.T) {
		type MockCheckResponse struct {
			Status              int                 `json:"s"`
			LimitingConstraints flexibleIntArray    `json:"lc"`
			Debug               flexibleStringArray `json:"d"`
		}

		input := `{"s": 1, "lc": [1, 3], "d": ["checking constraint 1", "evaluating throttle"]}`
		var resp MockCheckResponse
		err := json.Unmarshal([]byte(input), &resp)

		require.NoError(t, err)
		require.Equal(t, 1, resp.Status)
		require.Equal(t, []int{1, 3}, []int(resp.LimitingConstraints))
		require.Equal(t, []string{"checking constraint 1", "evaluating throttle"}, []string(resp.Debug))
	})

	t.Run("mixed empty and populated arrays", func(t *testing.T) {
		type MixedResponse struct {
			EmptyInts    flexibleIntArray    `json:"empty_ints"`
			PopulatedInt flexibleIntArray    `json:"populated_ints"`
			EmptyStrings flexibleStringArray `json:"empty_strings"`
			PopulatedStr flexibleStringArray `json:"populated_strings"`
		}

		input := `{
			"empty_ints": {},
			"populated_ints": [1, 2, 3],
			"empty_strings": {},
			"populated_strings": ["hello", "world"]
		}`

		var resp MixedResponse
		err := json.Unmarshal([]byte(input), &resp)

		require.NoError(t, err)
		require.Equal(t, []int{}, []int(resp.EmptyInts))
		require.Equal(t, []int{1, 2, 3}, []int(resp.PopulatedInt))
		require.Equal(t, []string{}, []string(resp.EmptyStrings))
		require.Equal(t, []string{"hello", "world"}, []string(resp.PopulatedStr))
	})
}

func TestFlexibleTypes_EdgeCases(t *testing.T) {
	t.Run("very large arrays", func(t *testing.T) {
		// Create a large array input
		large := make([]int, 1000)
		for i := range large {
			large[i] = i
		}

		input, err := json.Marshal(large)
		require.NoError(t, err)

		var arr flexibleIntArray
		err = json.Unmarshal(input, &arr)
		require.NoError(t, err)
		require.Equal(t, large, []int(arr))
	})

	t.Run("nested object that's not empty", func(t *testing.T) {
		var arr flexibleIntArray
		err := json.Unmarshal([]byte(`{"nested": {"key": "value"}}`), &arr)
		require.Error(t, err)
	})

	t.Run("object with null values", func(t *testing.T) {
		var arr flexibleStringArray
		err := json.Unmarshal([]byte(`{"key": null}`), &arr)
		require.Error(t, err)
	})

	t.Run("whitespace handling", func(t *testing.T) {
		tests := []string{
			"  []  ",
			"\n[]\n",
			"\t{}\t",
			"  {}  ",
		}

		for _, input := range tests {
			var intArr flexibleIntArray
			var strArr flexibleStringArray

			require.NoError(t, json.Unmarshal([]byte(input), &intArr))
			require.NoError(t, json.Unmarshal([]byte(input), &strArr))

			require.Equal(t, []int{}, []int(intArr))
			require.Equal(t, []string{}, []string(strArr))
		}
	})
}

func TestFlexibleTypes_TypeConversions(t *testing.T) {
	t.Run("int array conversion back to slice", func(t *testing.T) {
		arr := flexibleIntArray{1, 2, 3}
		slice := []int(arr)
		require.Equal(t, []int{1, 2, 3}, slice)
	})

	t.Run("string array conversion back to slice", func(t *testing.T) {
		arr := flexibleStringArray{"hello", "world"}
		slice := []string(arr)
		require.Equal(t, []string{"hello", "world"}, slice)
	})

	t.Run("length operations", func(t *testing.T) {
		intArr := flexibleIntArray{1, 2, 3}
		strArr := flexibleStringArray{"a", "b"}

		require.Equal(t, 3, len(intArr))
		require.Equal(t, 2, len(strArr))
	})

	t.Run("iteration", func(t *testing.T) {
		arr := flexibleIntArray{10, 20, 30}
		sum := 0
		for _, v := range arr {
			sum += v
		}
		require.Equal(t, 60, sum)
	})
}

