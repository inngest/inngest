package duckdb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrateCreatesStagingTables(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, Migrate(t.Context(), db))

	for _, table := range []string{"runs_staging", "run_spans_staging", "events_staging"} {
		rows, err := db.QueryContext(t.Context(), "SELECT * FROM "+table+" LIMIT 0;")
		require.NoErrorf(t, err, "table %s should exist and be queryable", table)
		require.NoError(t, rows.Close())
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, Migrate(t.Context(), db))
	require.NoError(t, Migrate(t.Context(), db)) // second call must be a no-op, not an error
}
