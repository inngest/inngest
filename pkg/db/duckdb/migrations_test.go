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

	require.NoError(t, Migrate(t.Context(), db, true))

	for _, table := range []string{"runs", "run_trace_spans", "events"} {
		rows, err := db.QueryContext(t.Context(), "SELECT * FROM "+DuckLakeAlias+"."+table+" LIMIT 0;")
		require.NoErrorf(t, err, "table %s should exist and be queryable", table)
		require.NoError(t, rows.Close())
	}

	// insights_* logical tables are parameterized table macros (Task 9 of
	// docs/plans/012-duckdb-insights-query-layer-plan.md), invoked as table
	// functions with (account_id, env_id) arguments rather than queried as
	// bare tables.
	for _, macro := range []string{
		"insights_runs", "insights_events", "insights_metadata",
		"insights_extended_trace_spans", "insights_step_attempts", "insights_steps",
	} {
		rows, err := db.QueryContext(t.Context(), "SELECT * FROM "+DuckLakeAlias+"."+macro+"(?, ?) LIMIT 0;",
			"00000000-0000-4000-a000-000000000000", "00000000-0000-4000-b000-000000000000")
		require.NoErrorf(t, err, "macro %s should exist and be queryable", macro)
		require.NoError(t, rows.Close())
	}
}

// TestMigrateInMemoryWithoutDuckLake proves persist=false's whole point: the
// same migration files run cleanly against a bare in-memory catalog with no
// DuckLake ATTACH at all, because the ENVSUB-wrapped SET SORTED/PARTITIONED
// BY statements (DuckLake-only syntax that would otherwise hard-fail with a
// Binder/Parser error) are overridden to a no-op, and the "inngest" schema
// Migrate creates directly stands in for the schema DuckLake's ATTACH would
// otherwise have provided.
func TestMigrateInMemoryWithoutDuckLake(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, Migrate(t.Context(), db, false))

	for _, table := range []string{"runs", "run_trace_spans", "events", "run_metadata"} {
		rows, err := db.QueryContext(t.Context(), "SELECT * FROM "+DuckLakeAlias+"."+table+" LIMIT 0;")
		require.NoErrorf(t, err, "table %s should exist and be queryable", table)
		require.NoError(t, rows.Close())
	}
}

// TestMigrateInMemoryIsIdempotent mirrors TestMigrateIsIdempotent for the
// persist=false path, guarding against the env-var override or schema
// creation breaking on a second run.
func TestMigrateInMemoryIsIdempotent(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, Migrate(t.Context(), db, false))
	require.NoError(t, Migrate(t.Context(), db, false))
}

func TestMigrateIsIdempotent(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), testDuckLakeOptions(t, binPath))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, Migrate(t.Context(), db, true))
	require.NoError(t, Migrate(t.Context(), db, true)) // second call must be a no-op, not an error
}
