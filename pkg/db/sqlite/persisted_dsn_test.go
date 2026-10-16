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
	tests := []struct {
		name       string
		file       string
		wantPrefix string
	}{
		{"absolute", "/tmp/x/main.db", "file:///tmp/x/main.db?"},
		{"drive", "C:/Users/x/.inngest/main.db", "file:///C:/Users/x/.inngest/main.db?"},
		{"relative", ".inngest/main.db", "file:///.inngest/main.db?"},
		{"special chars", "/tmp/dir with spaces & special=chars/main.db", "file:///tmp/dir%20with%20spaces%20&%20special=chars/main.db?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := persistedDSN(tt.file)
			assert.True(t, strings.HasPrefix(dsn, tt.wantPrefix), "got %q", dsn)
		})
	}
}

// TestOpenPersistedDefaultDirectory exercises the path every default-flags
// server takes: Directory "" resolves against the process working directory
// instead of building a file://.inngest authority URI.
func TestOpenPersistedDefaultDirectory(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)

	ctx := context.Background()
	conn, err := Open(ctx, Options{Persist: true, ForTest: true})
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	persistedSettings(t, conn)

	_, err = os.Stat(filepath.Join(wd, consts.DefaultInngestConfigDir, consts.SQLiteDbFileName))
	require.NoError(t, err, "database file must land under ./%s", consts.DefaultInngestConfigDir)
}
