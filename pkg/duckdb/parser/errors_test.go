// pkg/duckdb/parser/errors_test.go
package parser

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseErrorCarriesPosition(t *testing.T) {
	_, err := ParseString("SELECT * FROM")
	require.Error(t, err)
	var perr *ParseError
	require.True(t, errors.As(err, &perr))
	require.Greater(t, perr.Pos.Line, 0)
	require.Greater(t, perr.Pos.Column, 0)
}

func TestParseErrorMessageIncludesPosition(t *testing.T) {
	_, err := ParseString("SELECT *\nFROM")
	require.Error(t, err)
	require.Contains(t, err.Error(), "2:")
}

func TestParseStringNeverPanics(t *testing.T) {
	inputs := []string{
		"", "   ", ";", "SELECT", "SELECT *", "SELECT * FROM", "SELECT * FROM t WHERE",
		"SELECT (", "SELECT 'unterminated", `SELECT "unterminated`, "SELECT * FROM t WHERE a = ",
		"SELECT * FROM t GROUP BY", "SELECT * FROM t JOIN", "WITH", "(((", ")))",
		"SELECT a FROM t) FROM (SELECT", "\x00\x01\x02", strings.Repeat("(", 500),
	}
	// Every fixture truncated at every 7th byte offset — cheap, broad coverage
	// of "parser is mid-construct when input ends" without a real fuzzer.
	files, err := filepath.Glob("testdata/queries/*.sql")
	require.NoError(t, err)
	for _, f := range files {
		src, err := os.ReadFile(f)
		require.NoError(t, err)
		for i := 1; i < len(src); i += 7 {
			inputs = append(inputs, string(src[:i]))
		}
	}
	for _, in := range inputs {
		in := in
		require.NotPanicsf(t, func() { _, _ = ParseString(in) }, "input: %q", in)
	}
}
