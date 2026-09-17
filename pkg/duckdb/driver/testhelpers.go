package driver

import (
	"os/exec"
	"testing"
)

// RequireDuckDBBinary skips the calling test if no "duckdb" binary is on
// PATH, returning its path otherwise. Exported (and kept in a non-_test.go
// file) specifically so pkg/db/duckdb's own tests can call it too: Go never
// lets one package import another's _test.go-only symbols, and pkg/db/duckdb
// already imports this package in production code (for driver.Open/Options),
// so this is the one direction that avoids both problems.
func RequireDuckDBBinary(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping subprocess test")
	}
	return path
}
