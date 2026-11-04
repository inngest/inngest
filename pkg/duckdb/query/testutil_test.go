package query

import (
	"database/sql"
	"os/exec"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/icebergcatalog/icebergtest"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/driver"
)

// newTestDuckDB opens a real duckdb subprocess with an Iceberg lake attached and
// the dual-write schema migrated, mirroring
// pkg/duckdb/tracing/helpers_test.go's newTestDuckDB.
func newTestDuckDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	db, err := driver.Open(t.Context(), driver.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		Iceberg:    icebergtest.Start(t),
	})
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	if err := duckdb.Migrate(t.Context(), db, duckdb.ModeRawLake); err != nil {
		t.Fatalf("migrating duckdb: %v", err)
	}
	return db, func() { _ = db.Close() }
}
