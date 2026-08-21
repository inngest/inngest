package dualwrite

import (
	"database/sql"
	"os/exec"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/db/duckdb"
)

func newTestDuckDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	db, err := duckdb.Open(t.Context(), duckdb.Options{BinaryPath: binPath, DBFile: ":memory:"})
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
