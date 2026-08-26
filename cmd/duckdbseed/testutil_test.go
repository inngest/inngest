package main

import (
	"context"
	"database/sql"
	"os/exec"
	"testing"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/stretchr/testify/require"
)

func requireDuckDBOnPath(t *testing.T) {
	t.Helper()
	if _, err := lookDuckDBBinary(); err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
}

func requireQuackExtension(t *testing.T) {
	t.Helper()
	binPath, err := lookDuckDBBinary()
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	cmd := exec.Command(binPath, ":memory:", "-c", "INSTALL quack; LOAD quack;")
	if err := cmd.Run(); err != nil {
		t.Skip("quack extension not installable in this environment; skipping")
	}
}

// newTestDB opens a fresh DuckDB subprocess (DuckLake attached, schema
// migrated, quack enabled with parallelism connections) rooted at a temp
// directory, and returns both the connector (needed for additional
// concurrent connections — see InsertGeneratedRuns) and the *sql.DB (for
// ordinary querying). Both are closed automatically at test cleanup.
func newTestDB(t *testing.T, parallelism int) (*duckdb.Connector, *sql.DB) {
	t.Helper()
	requireDuckDBOnPath(t)
	requireQuackExtension(t)
	dir := t.TempDir()

	connector, db, err := openDuckDB(t.Context(), dir, parallelism)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
		_ = connector.Close()
	})
	return connector, db
}

// runIDsInDB returns every run_id in db's inngest.runs table, sorted, for
// equivalence assertions between two independently-populated databases.
func runIDsInDB(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "SELECT run_id FROM "+duckdb.DuckLakeAlias+".runs ORDER BY run_id;")
	require.NoError(t, err)
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	require.NoError(t, rows.Err())
	return ids
}
