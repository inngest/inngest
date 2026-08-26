package duckdbquery

import (
	"database/sql"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/inngest/inngest/pkg/db/duckdb"
)

// newTestDuckDB opens a real duckdb subprocess with DuckLake attached and
// the dual-write schema migrated, mirroring
// pkg/execution/dualwrite/helpers_test.go's newTestDuckDB.
func newTestDuckDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	dir := t.TempDir()
	db, err := duckdb.Open(t.Context(), duckdb.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &duckdb.DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.duckdb"),
			DataPath:    filepath.Join(dir, "data"),
		},
	})
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	if err := duckdb.Migrate(t.Context(), db); err != nil {
		t.Fatalf("migrating duckdb: %v", err)
	}
	return db, func() { _ = db.Close() }
}
