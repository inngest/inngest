package duckdb

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireDuckDBBinary(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping subprocess test")
	}
	return path
}

func TestStartProcessHealthCheck(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	p, err := startProcess(t.Context(), binPath, ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	require.NoError(t, p.healthCheck(t.Context()))
}

func TestStartProcessExecAndQuery(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	p, err := startProcess(t.Context(), binPath, ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	_, err = p.sess.exec(t.Context(), "CREATE TABLE t (id INTEGER, name VARCHAR);")
	require.NoError(t, err)

	_, err = p.sess.exec(t.Context(), "INSERT INTO t VALUES (1, 'a');")
	require.NoError(t, err)

	rows, err := p.sess.exec(t.Context(), "SELECT id, name FROM t;")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, float64(1), rows[0]["id"])
	require.Equal(t, "a", rows[0]["name"])
}

func TestProcessRestartAfterCrash(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	p, err := startProcess(t.Context(), binPath, ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	require.NoError(t, p.healthCheck(t.Context()))

	// Simulate a crash: kill the underlying subprocess out from under the
	// session, bypassing the normal close() path.
	require.NoError(t, p.cmd.Process.Kill())
	_, _ = p.cmd.Process.Wait()

	// The stale session should now fail health checks (broken pipe / EOF).
	require.Error(t, p.healthCheck(t.Context()))

	require.NoError(t, p.restart(t.Context()))
	require.NoError(t, p.healthCheck(t.Context()))
}

func TestOpenAndSetMaxOpenConns(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE t2 (id INTEGER);")
	require.NoError(t, err)
}
