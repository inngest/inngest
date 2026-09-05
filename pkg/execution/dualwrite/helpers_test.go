package dualwrite

import (
	"database/sql"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/db/duckdb"
)

// newTestDuckDB opens a real duckdb subprocess with DuckLake attached under
// duckdb.DuckLakeAlias ("inngest") — required for Migrate's
// "inngest.runs"/"inngest.run_trace_spans"/"inngest.events" DDL to resolve
// at all: "inngest" only exists as a catalog name once DuckLake is attached
// under it (see pkg/db/duckdb/process.go's DuckLakeAlias doc comment).
// Without this, every migration statement fails with "Catalog Error: Schema
// with name inngest does not exist!", exactly like production would if
// setupDualWrite ever opened a session without DuckLake configured.
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

func timeAfter() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		<-time.After(time.Second)
		close(ch)
	}()
	return ch
}
