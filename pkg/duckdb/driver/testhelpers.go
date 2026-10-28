package driver

import (
	"os/exec"
	"testing"
)

// RequireDuckDBBinary skips the calling test if no "duckdb" binary is on
// PATH, returning its path otherwise. Exported (and kept in a non-_test.go
// file) specifically so pkg/db/duckdb's and other packages' tests can call
// it too. It deliberately doesn't delegate to internal/duckdbtest, whose
// testify dependency would otherwise leak into this production package.
func RequireDuckDBBinary(t testing.TB) string {
	t.Helper()
	path, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping subprocess test")
	}
	return path
}
