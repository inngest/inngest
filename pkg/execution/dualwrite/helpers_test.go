package dualwrite

import (
	"database/sql"
	"os/exec"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/db/duckdb"
)

// newTestDuckDB opens a real duckdb subprocess with no DuckLake attachment —
// a plain in-memory database, nothing written to disk. Migrate(persist:
// false) creates the "inngest" schema directly (rather than relying on a
// DuckLake ATTACH to alias it) and neuters the DuckLake-only SET
// SORTED/PARTITIONED BY DDL (see Migrate's doc comment), so every table
// these tests exercise is a native, non-DuckLake DuckDB table.
func newTestDuckDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	db, err := duckdb.Open(t.Context(), duckdb.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
	})
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	if err := duckdb.Migrate(t.Context(), db, false); err != nil {
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
