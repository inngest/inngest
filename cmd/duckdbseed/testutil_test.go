package main

import (
	"context"
	"database/sql"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/stretchr/testify/require"
)

func requireDuckDBOnPath(t *testing.T) {
	t.Helper()
	binPath := driver.RequireDuckDBBinary(t)
	t.Setenv("DUCKDB_BINARY", binPath)
}

func requireQuackExtension(t *testing.T) {
	t.Helper()
	requireDuckDBOnPath(t)
}

// newTestDB opens a fresh DuckDB worker (Iceberg lake attached, schema
// migrated, with parallelism connections) rooted at a temp
// directory, and returns both the connector (needed for additional
// concurrent connections — see InsertGeneratedRuns) and the *sql.DB (for
// ordinary querying). Both are closed automatically at test cleanup.
func newTestDB(t *testing.T, parallelism int) (*driver.Connector, *sql.DB) {
	t.Helper()
	requireDuckDBOnPath(t)
	dir := t.TempDir()

	connector, db, closeDB, err := openDuckDB(t.Context(), dir, parallelism)
	require.NoError(t, err)
	t.Cleanup(closeDB)
	return connector, db
}

// runIDsInDB returns every run_id in db's inngest.runs table, sorted, for
// equivalence assertions between two independently-populated databases.
func runIDsInDB(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "SELECT run_id FROM "+driver.LakeAlias+".runs ORDER BY run_id;")
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
