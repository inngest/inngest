package duckdb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// copyExecutable copies src to dst with executable permissions, used to seed
// a fake "cached pinned binary" from the real duckdb binary already on PATH,
// so these tests exercise Connect's resolution logic without any network
// access.
func copyExecutable(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o755))
	require.NoError(t, os.WriteFile(dst, data, 0o755))
}

// TestConnectPrefersCachedPinnedBinaryOverPath proves that when BinaryPath is
// left empty and StateDir is set, Connect resolves the pinned/downloaded
// binary under StateDir rather than falling back to whatever "duckdb" PATH
// lookup would find — the "always prefer pinned/downloaded" behavior.
func TestConnectPrefersCachedPinnedBinaryOverPath(t *testing.T) {
	pathBin := requireDuckDBBinary(t)

	stateDir := t.TempDir()
	cached := BinaryPath(stateDir)
	copyExecutable(t, pathBin, cached)

	c := &Connector{opts: Options{DBFile: ":memory:", StateDir: stateDir}}
	_, err := c.Connect(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	require.Equal(t, cached, c.proc.binaryPath,
		"Connect must resolve the cached pinned binary under StateDir, not a PATH lookup")
}

// TestConnectIsolatesExtensionsUnderStateDir proves that when Options.StateDir
// is set, the spawned subprocess is bootstrapped with an isolated extension
// directory under it, rather than using whatever the ambient HOME already
// has.
func TestConnectIsolatesExtensionsUnderStateDir(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	stateDir := t.TempDir()

	c := &Connector{opts: Options{BinaryPath: binPath, DBFile: ":memory:", StateDir: stateDir}}
	_, err := c.Connect(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	require.DirExists(t, filepath.Join(stateDir, "duckdb", "extensions"))
}

// TestConnectWithoutStateDirDoesNotCreateOne is the regression guard: leaving
// Options.StateDir unset (every caller before this feature, and still
// cmd/duckdbseed today) must remain a complete no-op for extension-dir
// isolation.
func TestConnectWithoutStateDirDoesNotCreateOne(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	wd := t.TempDir()
	t.Chdir(wd)

	c := &Connector{opts: Options{BinaryPath: binPath, DBFile: ":memory:"}}
	_, err := c.Connect(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	require.NoDirExists(t, filepath.Join(wd, "duckdb"))
}
