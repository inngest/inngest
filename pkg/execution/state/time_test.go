package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimeout(t *testing.T) {
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	nowFn := func() time.Time { return now }

	rfcAt, err := time.Parse(time.RFC3339, "2023-10-12T07:20:50.52Z")
	assert.NoError(t, err)

	tests := []struct {
		name        string
		input       string
		want        time.Time
		wantErr     bool
		errContains string
	}{
		{
			name:  "valid duration",
			input: "3d",
			want:  now.Add(72 * time.Hour),
		},
		{
			name:  "valid RFC3339",
			input: "2023-10-12T07:20:50.52Z",
			want:  rfcAt,
		},
		{
			name:        "invalid duration",
			input:       "3q",
			wantErr:     true,
			errContains: "invalid duration",
		},
		{
			name:        "invalid RFC3339 missing Z",
			input:       "2023-10-12T07:20:50.52",
			wantErr:     true,
			errContains: "invalid RFC 3339 timestamp",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			timeoutAt, err := parseTimeout(tc.input, nowFn)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, timeoutAt)
		})
	}
}
