package duckdb

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/stretchr/testify/require"
)

// testDuckLakeOptions returns driver.Options wired with a fresh DuckLake catalog
// under t.TempDir(), so "inngest" resolves as a catalog name (see
// driver.DuckLakeAlias) and the migration's "inngest.<table>" DDL succeeds — the
// same wiring setupDualWrite uses in production.
func testDuckLakeOptions(t *testing.T, binPath string) driver.Options {
	dir := t.TempDir()
	return driver.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &driver.DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.duckdb"),
			DataPath:    filepath.Join(dir, "data"),
		},
	}
}

func TestMigrateCreatesStagingTables(t *testing.T) {
	binPath := driver.RequireDuckDBBinary(t)

	db, err := driver.Open(t.Context(), testDuckLakeOptions(t, binPath))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, Migrate(t.Context(), db, ModeRawLake))

	for _, table := range []string{"runs", "run_trace_spans", "events"} {
		rows, err := db.QueryContext(t.Context(), "SELECT * FROM "+driver.DuckLakeAlias+"."+table+" LIMIT 0;")
		require.NoErrorf(t, err, "table %s should exist and be queryable", table)
		require.NoError(t, rows.Close())
	}

	// insights_* logical tables are parameterized table macros (see
	// migrations/000002_insights_views.sql), invoked as table
	// functions with (account_id, env_id) arguments rather than queried as
	// bare tables.
	for _, macro := range []string{
		"insights_runs", "insights_events", "insights_metadata",
		"insights_extended_trace_spans", "insights_step_attempts", "insights_steps",
	} {
		rows, err := db.QueryContext(t.Context(), "SELECT * FROM "+driver.DuckLakeAlias+"."+macro+"(?, ?) LIMIT 0;",
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
	binPath := driver.RequireDuckDBBinary(t)

	db, err := driver.Open(t.Context(), driver.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, Migrate(t.Context(), db, ModeMemory))

	for _, table := range []string{"runs", "run_trace_spans", "events", "run_metadata"} {
		rows, err := db.QueryContext(t.Context(), "SELECT * FROM "+driver.DuckLakeAlias+"."+table+" LIMIT 0;")
		require.NoErrorf(t, err, "table %s should exist and be queryable", table)
		require.NoError(t, rows.Close())
	}
}

// TestMigrateInMemoryIsIdempotent mirrors TestMigrateIsIdempotent for the
// persist=false path, guarding against the env-var override or schema
// creation breaking on a second run.
func TestMigrateInMemoryIsIdempotent(t *testing.T) {
	binPath := driver.RequireDuckDBBinary(t)

	db, err := driver.Open(t.Context(), driver.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, Migrate(t.Context(), db, ModeMemory))
	require.NoError(t, Migrate(t.Context(), db, ModeMemory))
}

func TestMigrateIsIdempotent(t *testing.T) {
	binPath := driver.RequireDuckDBBinary(t)

	db, err := driver.Open(t.Context(), testDuckLakeOptions(t, binPath))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, Migrate(t.Context(), db, ModeRawLake))
	require.NoError(t, Migrate(t.Context(), db, ModeRawLake)) // second call must be a no-op, not an error
}

// TestMigrateModeRawRunsCollapse checks raw_runs collapses a run's rows to
// its latest in every mode but ModeFlatLake, whose runs are already one row per
// run, and that insights_runs (which reads through raw_runs) follows.
func TestMigrateModeRawRunsCollapse(t *testing.T) {
	binPath := driver.RequireDuckDBBinary(t)

	cases := []struct {
		mode Mode
		want int
	}{
		{ModeMemory, 1},
		{ModeRawLake, 1},
		{ModeFlatLake, 2},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("mode %d", tc.mode), func(t *testing.T) {
			opts := testDuckLakeOptions(t, binPath)
			if tc.mode == ModeMemory {
				opts = driver.Options{BinaryPath: binPath, DBFile: ":memory:"}
			}
			db, err := driver.Open(t.Context(), opts)
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			require.NoError(t, Migrate(t.Context(), db, tc.mode))

			account, env := "00000000-0000-4000-a000-000000000000", "00000000-0000-4000-b000-000000000000"
			for _, status := range []string{"Queued", "Completed"} {
				_, err := db.ExecContext(t.Context(), `INSERT INTO inngest.runs
					(account_id, env_id, run_id, queued_at, scheduled_at, ended_at, app_id, app_name, function_id, function_slug, status, attributes, inputs)
					VALUES (?::UUID, ?::UUID, 'run-1', now(), now(), CASE WHEN ? = 'Completed' THEN now() END, uuid(), 'app', uuid(), 'fn', ?, '{}'::JSON::VARIANT, '[]'::JSON::VARIANT)`,
					account, env, status, status)
				require.NoError(t, err)
			}

			for _, q := range []string{
				"SELECT count(*) FROM inngest.raw_runs()",
				"SELECT count(*) FROM inngest.insights_runs('" + account + "', '" + env + "')",
			} {
				var n int
				require.NoError(t, db.QueryRowContext(t.Context(), q).Scan(&n))
				require.Equal(t, tc.want, n, q)
			}
		})
	}
}
