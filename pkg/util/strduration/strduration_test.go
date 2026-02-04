package strduration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStringDurationMarshalJSON(t *testing.T) {
	sd := Duration(24*time.Hour + 30*time.Minute)

	data, err := json.Marshal(sd)
	require.NoError(t, err)
	require.Equal(t, `"1d30m"`, string(data))
}

func TestStringDurationUnmarshalJSONString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Duration
	}{
		{
			name:     "days and minutes",
			input:    `"1d30m"`,
			expected: Duration(24*time.Hour + 30*time.Minute),
		},
		{
			name:     "hours only",
			input:    `"2h"`,
			expected: Duration(2 * time.Hour),
		},
		{
			name:     "minutes only",
			input:    `"45m"`,
			expected: Duration(45 * time.Minute),
		},
		{
			name:     "seconds only",
			input:    `"30s"`,
			expected: Duration(30 * time.Second),
		},
		{
			name:     "hours minutes seconds",
			input:    `"1h30m10s"`,
			expected: Duration(time.Hour + 30*time.Minute + 10*time.Second),
		},
		{
			name:     "days only",
			input:    `"7d"`,
			expected: Duration(7 * 24 * time.Hour),
		},
		{
			name:     "weeks",
			input:    `"1w"`,
			expected: Duration(7 * 24 * time.Hour),
		},
		{
			name:     "milliseconds",
			input:    `"500ms"`,
			expected: Duration(500 * time.Millisecond),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sd Duration
			err := json.Unmarshal([]byte(tt.input), &sd)
			require.NoError(t, err)
			require.Equal(t, tt.expected, sd)
		})
	}
}

func TestStringDurationUnmarshalJSONNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Duration
	}{
		{
			name:     "integer nanoseconds",
			input:    `1000000000`,
			expected: Duration(time.Second),
		},
		{
			name:     "float nanoseconds",
			input:    `1.5e9`,
			expected: Duration(1500 * time.Millisecond),
		},
		{
			name:     "negative nanoseconds",
			input:    `-5000000000`,
			expected: Duration(-5 * time.Second),
		},
		{
			name:     "zero",
			input:    `0`,
			expected: Duration(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sd Duration
			err := json.Unmarshal([]byte(tt.input), &sd)
			require.NoError(t, err)
			require.Equal(t, tt.expected, sd)
		})
	}
}

func TestStringDurationUnmarshalJSONErrors(t *testing.T) {
	var sd Duration

	err := json.Unmarshal([]byte(`"invalid"`), &sd)
	require.Error(t, err)

	sd = 123
	err = json.Unmarshal([]byte(`""`), &sd)
	require.NoError(t, err)
	require.Equal(t, Duration(0), sd)

	sd = 123
	err = json.Unmarshal([]byte(`null`), &sd)
	require.NoError(t, err)
	require.Equal(t, Duration(0), sd)
}
