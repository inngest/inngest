package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPersistedDSNShapes asserts the file: URI never carries an authority
// segment: SQLite rejects any authority except empty/localhost, so a
// relative path or a Windows drive path must not become file://<authority>.
func TestPersistedDSNShapes(t *testing.T) {
	dsn := persistedDSN("/tmp/x/main.db")
	assert.True(t, strings.HasPrefix(dsn, "file:///tmp/x/main.db?"),
		"absolute path must root as file:///, got %q", dsn)

	dsn = persistedDSN("C:/Users/x/.inngest/main.db")
	assert.True(t, strings.HasPrefix(dsn, "file:///C:/Users/x/.inngest/main.db?"),
		"drive path must not become an authority, got %q", dsn)

	dsn = persistedDSN(".inngest/main.db")
	assert.True(t, strings.HasPrefix(dsn, "file:///.inngest/main.db?"),
		"relative path must root, got %q", dsn)
	assert.NotContains(t, dsn, "file://.inngest/")

	// URI-sensitive characters in directory names must survive the round trip.
	dsn = persistedDSN("/tmp/dir with spaces & special=chars/main.db")
	assert.Contains(t, dsn, "file:///")
	assert.NotContains(t, dsn, "file://dir")
}

// TestOpenPersistedDefaultDirectory exercises the path every default-flags
// server takes: Directory "" resolves against the process working directory
// instead of building a file://.inngest authority URI.
func TestOpenPersistedDefaultDirectory(t *testing.T) {
	wd := t.TempDir()
	require.NoError(t, os.Chdir(wd))
	t.Cleanup(func() {
		// Restore best-effort; test binaries chdir per test process.
		_ = os.Chdir("/")
	})

	ctx := context.Background()
	conn, err := Open(ctx, Options{Persist: true, ForTest: true})
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	persistedSettings(t, conn)

	_, err = os.Stat(filepath.Join(wd, consts.DefaultInngestConfigDir, consts.SQLiteDbFileName))
	require.NoError(t, err, "database file must land under ./%s", consts.DefaultInngestConfigDir)
}
