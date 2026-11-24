package driver

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/stretchr/testify/require"
)

func TestStmtSummaryOmitsLiterals(t *testing.T) {
	cases := map[string]string{
		"CALL quack_serve('quack:127.0.0.1:0', token = 'sekrit');":                             "CALL quack_serve",
		"ATTACH IF NOT EXISTS 'inngest' AS inngest (TYPE iceberg, URI 'http://127.0.0.1:1/');": "ATTACH IF NOT EXISTS",
		"CREATE OR REPLACE SECRET inngest_iceberg (TYPE iceberg, TOKEN 'sekrit');":             "CREATE OR REPLACE SECRET inngest_iceberg",
		"SET home_directory='/tmp/x';":                                                         "SET home_directory",
		"INSTALL quack;":                                                                       "INSTALL quack",
		"LOAD '/path/to/ext';":                                                                 "LOAD",
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

// TestIcebergBootstrapErrorDoesNotLeakToken drives a real bootstrap failure
// whose DuckDB error echoes the failing statement, and checks the token
// never reaches the returned error.
func TestIcebergBootstrapErrorDoesNotLeakToken(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	const token = "do-not-leak-this-token"
	ice := &IcebergOptions{
		URI:       "http://127.0.0.1:1",
		Warehouse: "test_wh",
		Token:     token,
	}
	_, err := startProcessWithLake(t.Context(), binPath, ":memory:", ice, nil)
	require.Error(t, err)
	require.False(t, strings.Contains(err.Error(), token), "bootstrap error leaked the Iceberg token: %v", err)
}
