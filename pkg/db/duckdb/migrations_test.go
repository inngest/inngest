package duckdb

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// testDuckLakeOptions returns Options wired with a fresh DuckLake catalog
// under t.TempDir(), so "inngest" resolves as a catalog name (see
// DuckLakeAlias) and the migration's "inngest.<table>" DDL succeeds — the
// same wiring setupDualWrite uses in production.
func testDuckLakeOptions(t *testing.T, binPath string) Options {
	dir := t.TempDir()
	return Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.duckdb"),
			DataPath:    filepath.Join(dir, "data"),
		},
	}
}

func TestMigrateCreatesStagingTables(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), testDuckLakeOptions(t, binPath))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, Migrate(t.Context(), db))

	for _, table := range []string{"runs", "run_trace_spans", "events"} {
		rows, err := db.QueryContext(t.Context(), "SELECT * FROM "+DuckLakeAlias+"."+table+" LIMIT 0;")
		require.NoErrorf(t, err, "table %s should exist and be queryable", table)
		require.NoError(t, rows.Close())
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), testDuckLakeOptions(t, binPath))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, Migrate(t.Context(), db))
	require.NoError(t, Migrate(t.Context(), db)) // second call must be a no-op, not an error
}
