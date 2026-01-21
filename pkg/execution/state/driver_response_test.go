package state

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

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
				Err: strptr("some err"),
			},
			expected: false,
		},
		{
			name: "error marked as final is final",
			r: DriverResponse{
				Err:   strptr("some err"),
				final: true,
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

func TestDriverResponseFormatError(t *testing.T) {
	tests := []struct {
		name     string
		r        DriverResponse
		expected StandardError
	}{
		{
			name: "with no Output and Err",
			r:    DriverResponse{Output: nil, Err: strptr("something went wrong")},
			expected: StandardError{
				Error:   "something went wrong",
				Name:    "Error",
				Message: "something went wrong",
			},
		},
		{
			name: "with no Output and no Err",
			r:    DriverResponse{Output: nil},
			expected: StandardError{
				Error:   DefaultErrorMessage,
				Name:    "Error",
				Message: DefaultErrorMessage,
			},
		},
		{
			name: "with encoded response",
			r: DriverResponse{
				Output: map[string]any{
					"error": `{"name":"Error","message":"test"}`,
				},
			},
			expected: StandardError{
				Error:   DefaultErrorMessage,
				Name:    "Error",
				Message: "test",
			},
		},
		// encoded JS errors
		{
			name: "with body as error string",
			r: DriverResponse{Output: map[string]any{
				"body": "{\"name\":\"Error\",\"message\":\"lolk\",\"stack\":\"stack\",\"__serialized\":true}",
			}},
			expected: StandardError{
				Error:   DefaultErrorMessage,
				Name:    "Error",
				Message: "lolk",
				Stack:   "stack",
			},
		},
		{
			name: "with body as error json.RawMessage",
			r: DriverResponse{Output: map[string]any{
				"body": json.RawMessage("{\"name\":\"Error\",\"message\":\"lolk\",\"stack\":\"stack\",\"__serialized\":true}"),
			}},
			expected: StandardError{
				Error:   DefaultErrorMessage,
				Name:    "Error",
				Message: "lolk",
				Stack:   "stack",
			},
		},
		{
			name: "with body as error bytes",
			r: DriverResponse{Output: map[string]any{
				"body": []byte("{\"name\":\"Error\",\"message\":\"lolk\",\"stack\":\"stack\",\"__serialized\":true}"),
			}},
			expected: StandardError{
				Error:   DefaultErrorMessage,
				Name:    "Error",
				Message: "lolk",
				Stack:   "stack",
			},
		},

		{
			name: "non map Output",
			r:    DriverResponse{Output: "YOLO"},
			expected: StandardError{
				Error:   DefaultErrorMessage,
				Name:    "Error",
				Message: "YOLO",
			},
		},

		{
			name: "non map Output with error",
			r:    DriverResponse{Output: "YOLO", Err: strptr("502 broken")},
			expected: StandardError{
				Error:   "502 broken",
				Name:    "Error",
				Message: "YOLO",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.EqualValues(t, test.expected, test.r.StandardError(), test.name)
		})
	}
}

func TestGeneratorSleepDuration(t *testing.T) {
	// Check that this works with a timestamp
	at := time.Now().Truncate(time.Second).Add(time.Minute)
	g := GeneratorOpcode{
		Op:   enums.OpcodeSleep,
		Name: at.Format(time.RFC3339),
	}
	duration, err := g.SleepDuration()
	require.Nil(t, err)
	require.WithinDuration(t, time.Now().Truncate(time.Second).Add(duration), at, time.Second)

	// Check that this works with a timestamp
	g = GeneratorOpcode{
		Op:   enums.OpcodeSleep,
		Name: "2022-01-01T10:30:00.468Z",
	}
	duration, err = g.SleepDuration()
	require.Nil(t, err)
	require.EqualValues(t, 0, duration.Seconds())

	// Check that this works with a duration string
	g = GeneratorOpcode{
		Op:   enums.OpcodeSleep,
		Name: "60s",
	}
	duration, err = g.SleepDuration()
	require.Nil(t, err)
	require.WithinDuration(t, time.Now().Truncate(time.Second).Add(time.Minute), time.Now().Add(duration), time.Second)
}

func TestGetTraceFunctionOutput(t *testing.T) {
	tests := []struct {
		name     string
		r        DriverResponse
		expected string
		hasError bool
	}{
		{
			name: "valid JSON output wrapped in data",
			r: DriverResponse{
				Generator: []*GeneratorOpcode{
					{
						Op:   enums.OpcodeRunComplete,
						Data: json.RawMessage(`{"hello": "world"}`),
					},
				},
			},
			expected: `{"data":{"hello": "world"}}`,
		},
		{
			name: "HTML output wrapped in error with quoted string",
			r: DriverResponse{
				Generator: []*GeneratorOpcode{},
				Err:       strptr("failed"),
				Output:    "<html><body>502 Bad Gateway</body></html>",
			},
			expected: `{"error":"<html><body>502 Bad Gateway</body></html>"}`,
		},
		{
			name: "non-JSON string output wrapped in data with quotes",
			r: DriverResponse{
				Generator: []*GeneratorOpcode{},
				Output:    "plain text response",
			},
			expected: `{"data":"plain text response"}`,
		},
		{
			name: "valid JSON string output wrapped in data",
			r: DriverResponse{
				Generator: []*GeneratorOpcode{},
				Output:    `{"result": "success"}`,
			},
			expected: `{"data":{"result":"success"}}`,
		},
		{
			name: "byte slice HTML output wrapped in data with quotes",
			r: DriverResponse{
				Generator: []*GeneratorOpcode{},
				Output:    []byte("<html><body>Not Found</body></html>"),
			},
			expected: `{"data":"<html><body>Not Found</body></html>"}`,
		},
		{
			name: "complex object marshalled and wrapped in data",
			r: DriverResponse{
				Generator: []*GeneratorOpcode{},
				Output:    map[string]interface{}{"status": "ok", "count": 42},
			},
			expected: `{"data":{"count":42,"status":"ok"}}`,
		},
		{
			name: "error with non-JSON output",
			r: DriverResponse{
				Generator: []*GeneratorOpcode{},
				Err:       strptr("request failed"),
				Output:    "<html>Error page</html>",
			},
			expected: `{"error":"<html>Error page</html>"}`,
		},
		{
			name: "already wrapped error returned as-is",
			r: DriverResponse{
				Generator: []*GeneratorOpcode{},
				Err:       strptr("failed"),
				Output:    `{"error":{"message":"SDK error"}}`,
			},
			expected: `{"error":{"message":"SDK error"}}`,
		},
		{
			name: "empty string output should error",
			r: DriverResponse{
				Generator: []*GeneratorOpcode{},
				Output:    "",
			},
			hasError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := test.r.GetTraceFunctionOutput()

			if test.hasError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expected, actual)

			// Verify the output is valid JSON
			if actual != "" {
				var temp interface{}
				require.NoError(t, json.Unmarshal([]byte(actual), &temp), "output should be valid JSON")
			}
		})
	}
}

func strptr(s string) *string {
	return &s
}
