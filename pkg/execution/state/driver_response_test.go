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
		expected map[string]any
	}{
		{
			name: "with no Output and Err",
			r:    DriverResponse{Output: nil, Err: strptr("something went wrong")},
			expected: map[string]any{
				"error":   "something went wrong",
				"name":    "Error",
				"message": "something went wrong",
			},
		},
		{
			name: "with no Output and no Err",
			r:    DriverResponse{Output: nil},
			expected: map[string]any{
				"error":   DefaultErrorMessage,
				"name":    "Error",
				"message": DefaultErrorMessage,
			},
		},
		{
			name: "with encoded response",
			r: DriverResponse{
				Output: map[string]any{
					"error": `{"name":"Error","message":"test"}`,
				},
			},
			expected: map[string]any{
				"name":    "Error",
				"message": "test",
			},
		},
		{
			name: "with Output and no body",
			r: DriverResponse{Output: map[string]any{
				"data": "error response",
			}},
			expected: map[string]any{
				// Auto-fill required fields
				"name":    "Error",
				"message": DefaultErrorMessage,
				"data":    "error response",
			},
		},
		// encoded JS errors
		{
			name: "with body as error string",
			r: DriverResponse{Output: map[string]any{
				"body": "{\"name\":\"Error\",\"message\":\"lolk\",\"stack\":\"stack\",\"__serialized\":true}",
			}},
			expected: map[string]any{
				"name":         "Error",
				"message":      "lolk",
				"stack":        "stack",
				"__serialized": true,
			},
		},
		{
			name: "with body as error json.RawMessage",
			r: DriverResponse{Output: map[string]any{
				"body": json.RawMessage("{\"name\":\"Error\",\"message\":\"lolk\",\"stack\":\"stack\",\"__serialized\":true}"),
			}},
			expected: map[string]any{
				"name":         "Error",
				"message":      "lolk",
				"stack":        "stack",
				"__serialized": true,
			},
		},
		{
			name: "with body as error bytes",
			r: DriverResponse{Output: map[string]any{
				"body": []byte("{\"name\":\"Error\",\"message\":\"lolk\",\"stack\":\"stack\",\"__serialized\":true}"),
			}},
			expected: map[string]any{
				"name":         "Error",
				"message":      "lolk",
				"stack":        "stack",
				"__serialized": true,
			},
		},

		{
			name: "non map Output",
			r:    DriverResponse{Output: "YOLO"},
			expected: map[string]any{
				"name":    "Error",
				"message": "YOLO",
			},
		},

		{
			name: "non map Output with error",
			r:    DriverResponse{Output: "YOLO", Err: strptr("502 broken")},
			expected: map[string]any{
				"name":    "Error",
				"message": "YOLO",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// require.EqualValues(t, test.expected, test.r.UserError(), test.name)
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

func strptr(s string) *string {
	return &s
}
