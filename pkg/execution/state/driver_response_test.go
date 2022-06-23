package state

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDriverResponseRetryable(t *testing.T) {
	var unmarshalledTrue, unmarshalledFalse map[string]interface{}

	err := json.Unmarshal([]byte(`{"body":"yea","status":501}`), &unmarshalledTrue)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(`{"body":"yea","status":403}`), &unmarshalledFalse)
	require.NoError(t, err)

	tests := []struct {
		name     string
		r        DriverResponse
		expected bool
	}{
		{
			name: "no output, with error",
			r: DriverResponse{
				Output: map[string]interface{}{},
				Err:    fmt.Errorf("some err"),
			},
			expected: true,
		},
		{
			name: "no output, no error",
			r: DriverResponse{
				Output: map[string]interface{}{},
			},
			expected: false,
		},
		{
			name: "no status, with error",
			r: DriverResponse{
				Output: map[string]interface{}{
					"hi": "my g",
				},
				Err: fmt.Errorf("some err"),
			},
			expected: true,
		},
		{
			name: "no status, no error",
			r: DriverResponse{
				Output: map[string]interface{}{
					"hi": "my g",
				},
			},
			expected: false,
		},
		{
			name: "success status with error",
			r: DriverResponse{
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
			r: DriverResponse{
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
			r: DriverResponse{
				Output: unmarshalledFalse,
				Err:    fmt.Errorf("some err"),
			},
			expected: false,
		},
		{
			name: "5xx status",
			r: DriverResponse{
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
			r: DriverResponse{
				Output: unmarshalledTrue,
				Err:    fmt.Errorf("some err"),
			},
			expected: true,
		},
		{
			name: "5xx json status, no error",
			r: DriverResponse{
				Output: unmarshalledTrue,
			},
			expected: false,
		},
		{
			name: "5xx wth final",
			r: DriverResponse{
				Output: map[string]interface{}{
					"hi":     "dont retry me plz",
					"status": 500,
				},
				Err:   fmt.Errorf("some err"),
				final: true,
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

func TestDriverResponseFinal(t *testing.T) {

	tests := []struct {
		name     string
		r        DriverResponse
		expected bool
	}{
		{
			name: "output is not final",
			r: DriverResponse{
				Output: map[string]interface{}{
					"hi": "there",
				},
			},
			expected: false,
		},
		{
			name: "error is not final",
			r: DriverResponse{
				Err: fmt.Errorf("some err"),
			},
			expected: false,
		},
		{
			name: "error marked as final is final",
			r: DriverResponse{
				Err:   fmt.Errorf("some err"),
				final: true,
			},
			expected: true,
		},
		{
			name: "non-retryable error is final",
			r: DriverResponse{
				Err: fmt.Errorf("final err plz"),
				Output: map[string]interface{}{
					"status": 401,
				},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.r.Final()
			require.Equal(t, test.expected, actual)
		})
	}
}
