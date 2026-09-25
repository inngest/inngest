package driver

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/duckdbtest"
	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/stretchr/testify/require"
)

func TestStmtSummaryOmitsLiterals(t *testing.T) {
	cases := map[string]string{
		"CALL quack_serve('quack:127.0.0.1:0', token = 'sekrit');":                                   "CALL quack_serve",
		"ATTACH IF NOT EXISTS 'ducklake:postgres:postgres://u:pw@h/db' AS inngest (DATA_PATH 'x/');": "ATTACH IF NOT EXISTS",
		"CREATE OR REPLACE SECRET ducklake_quack_catalog (TYPE quack, TOKEN 'sekrit', SCOPE 'x');":   "CREATE OR REPLACE SECRET ducklake_quack_catalog",
		"SET home_directory='/tmp/x';": "SET home_directory",
		"INSTALL quack;":               "INSTALL quack",
		"LOAD '/path/to/ext';":         "LOAD",
	}
	for stmt, want := range cases {
		require.Equal(t, want, stmtSummary(stmt), stmt)
	}
}

func TestRedactErrScrubsSecretsButKeepsClassification(t *testing.T) {
	r := &secretRedactor{}
	r.add("sekrit", "postgres://u:pw@h/db", "")

	raw := fmt.Errorf("%w: Parser Error: LINE 1: ... token = 'sekrit' ... ATTACH 'postgres://u:pw@h/db'", result.ErrStatementFailed)
	got := r.redactErr(raw)
	require.NotContains(t, got.Error(), "sekrit")
	require.NotContains(t, got.Error(), "pw@h")
	require.Contains(t, got.Error(), redactedPlaceholder)
	require.ErrorIs(t, got, result.ErrStatementFailed)
	require.Nil(t, errors.Unwrap(got), "unwrapping must not hand back the unredacted message")

	clean := errors.New("nothing secret here")
	require.Same(t, clean, r.redactErr(clean))

	var nilRedactor *secretRedactor
	require.Equal(t, "sekrit", nilRedactor.redact("sekrit"))
}

// TestQuackBootstrapErrorDoesNotLeakToken drives a real bootstrap failure
// whose DuckDB error echoes the failing statement, and checks the token
// never reaches the returned error.
func TestQuackBootstrapErrorDoesNotLeakToken(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	duckdbtest.RequireQuackExtension(t, binPath)

	// A non-loopback host makes quack_serve itself fail after the token is
	// already part of the statement.
	addr := "192.0.2.1:0"
	const token = "do-not-leak-this-token"
	_, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &addr, withQuackServeToken(token))
	require.Error(t, err)
	require.False(t, strings.Contains(err.Error(), token), "bootstrap error leaked the quack token: %v", err)
	require.Contains(t, err.Error(), "CALL quack_serve")
}
