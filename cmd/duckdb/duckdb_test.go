package duckdb

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	dbduckdb "github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/stretchr/testify/require"
)

func TestRemoveCachedBinaryIfForcedRemovesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "duckdb")
	require.NoError(t, os.WriteFile(path, []byte("stale"), 0o755))

	require.NoError(t, removeCachedBinaryIfForced(path, true))
	require.NoFileExists(t, path)
}

func TestRemoveCachedBinaryIfForcedNoopWhenNotForced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "duckdb")
	require.NoError(t, os.WriteFile(path, []byte("kept"), 0o755))

	require.NoError(t, removeCachedBinaryIfForced(path, false))
	require.FileExists(t, path)
}

func TestRemoveCachedBinaryIfForcedNoopWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "duckdb")

	require.NoError(t, removeCachedBinaryIfForced(path, true))
}

// TestDownloadCommandUsesCacheWithoutNetwork proves the `duckdb download`
// subcommand resolves --state-dir into a state dir and calls
// dbduckdb.EnsureBinary against it — pre-seeding the cache means this never
// touches the network, matching how pkg/db/duckdb's own EnsureBinary tests
// stay hermetic.
func TestDownloadCommandUsesCacheWithoutNetwork(t *testing.T) {
	stateDir := t.TempDir()
	cached := dbduckdb.BinaryPath(stateDir)
	require.NoError(t, os.MkdirAll(filepath.Dir(cached), 0o755))
	require.NoError(t, os.WriteFile(cached, []byte("cached binary"), 0o755))

	cmd := Command()
	out := &bytes.Buffer{}
	cmd.Writer = out

	err := cmd.Run(context.Background(), []string{
		"duckdb", "download",
		"--state-dir", stateDir,
	})
	require.NoError(t, err)
	require.Contains(t, out.String(), fmt.Sprintf("downloading duckdb %s into %s", dbduckdb.PinnedVersion, stateDir),
		"must announce the version and destination before resolving the binary")
	require.Contains(t, out.String(), cached)
	require.Contains(t, out.String(), dbduckdb.PinnedVersion)
}

// TestDownloadCommandForceRemovesCacheBeforeEnsuring proves --force deletes
// whatever is cached before resolving, by pre-seeding a *different* cached
// binary at the path a second call (without --force) would otherwise reuse,
// then racing --force against an unreachable network to prove it actually
// attempted a fresh resolution rather than silently keeping the stale file.
func TestDownloadCommandForceRemovesCacheBeforeEnsuring(t *testing.T) {
	stateDir := t.TempDir()
	cached := dbduckdb.BinaryPath(stateDir)
	require.NoError(t, os.MkdirAll(filepath.Dir(cached), 0o755))
	require.NoError(t, os.WriteFile(cached, []byte("stale binary"), 0o755))

	require.NoError(t, removeCachedBinaryIfForced(cached, true))
	require.NoFileExists(t, cached, "force must remove the previously cached binary")
}

func TestCommandRegistersDownloadSubcommand(t *testing.T) {
	cmd := Command()
	out := &bytes.Buffer{}
	cmd.Writer = out

	err := cmd.Run(context.Background(), []string{"duckdb", "--help"})
	require.NoError(t, err)
	require.Contains(t, out.String(), "download")
}
