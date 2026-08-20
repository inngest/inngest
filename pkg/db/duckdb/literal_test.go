package duckdb

import (
	"database/sql/driver"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEncodeLiteral(t *testing.T) {
	cases := []struct {
		name  string
		value driver.Value
		want  string
	}{
		{"nil", nil, "NULL"},
		{"bool true", true, "TRUE"},
		{"bool false", false, "FALSE"},
		{"int64", int64(42), "42"},
		{"float64", float64(3.5), "3.5"},
		{"plain string", "hello", "'hello'"},
		{"string with single quote", "O'Brien", "'O''Brien'"},
		{"string with backslash", `a\b`, `'a\b'`},
		{"bytes", []byte{0x01, 0x02}, "'\\x01\\x02'::BLOB"},
		{
			"time.Time",
			time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			"TIMESTAMP '2026-01-02 03:04:05'",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := encodeLiteral(tc.value)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestInterpolateReplacesPositionalPlaceholders(t *testing.T) {
	got, err := interpolate("INSERT INTO t VALUES (?, ?);", []driver.NamedValue{
		{Ordinal: 1, Value: int64(1)},
		{Ordinal: 2, Value: "a"},
	})
	require.NoError(t, err)
	require.Equal(t, "INSERT INTO t VALUES (1, 'a');", got)
}

func TestInterpolateRejectsMismatchedPlaceholderCount(t *testing.T) {
	_, err := interpolate("INSERT INTO t VALUES (?, ?);", []driver.NamedValue{
		{Ordinal: 1, Value: int64(1)},
	})
	require.Error(t, err)
}
