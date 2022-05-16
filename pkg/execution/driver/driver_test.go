package driver

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetryable(t *testing.T) {
	var unmarshalledTrue, unmarshalledFalse map[string]interface{}

	err := json.Unmarshal([]byte(`{"body":"yea","status":501}`), &unmarshalledTrue)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(`{"body":"yea","status":403}`), &unmarshalledFalse)
	require.NoError(t, err)

	tests := []struct {
		name     string
		r        Response
		expected bool
	}{
		{
			name: "no output, with error",
			r: Response{
				Output: map[string]interface{}{},
				Err:    fmt.Errorf("some err"),
			},
			expected: true,
		},
		{
			name: "no output, no error",
			r: Response{
				Output: map[string]interface{}{},
			},
			expected: false,
		},
		{
			name: "no status, with error",
			r: Response{
				Output: map[string]interface{}{
					"hi": "my g",
				},
				Err: fmt.Errorf("some err"),
			},
			expected: true,
		},
		{
			name: "no status, no error",
			r: Response{
				Output: map[string]interface{}{
					"hi": "my g",
				},
			},
			expected: false,
		},
		{
			name: "success status with error",
			r: Response{
				Output: map[string]interface{}{
					"hi":     "my g",
					"status": 200,
				},
				Err: fmt.Errorf("some err"),
			},
			expected: false,
		},
		{
			name: "4xx status",
			r: Response{
				Output: map[string]interface{}{
					"hi":     "my g",
					"status": 401,
				},
				Err: fmt.Errorf("some err"),
			},
			expected: false,
		},
		{
			name: "5xx json status",
			r: Response{
				Output: unmarshalledFalse,
				Err:    fmt.Errorf("some err"),
			},
			expected: false,
		},
		{
			name: "5xx status",
			r: Response{
				Output: map[string]interface{}{
					"hi":     "my g",
					"status": 500,
				},
				Err: fmt.Errorf("some err"),
			},
			expected: true,
		},
		{
			name: "5xx json status",
			r: Response{
				Output: unmarshalledTrue,
				Err:    fmt.Errorf("some err"),
			},
			expected: true,
		},
		{
			name: "5xx json status, no error",
			r: Response{
				Output: unmarshalledTrue,
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.r.Retryable()
			require.Equal(t, test.expected, actual)
		})
	}
}
