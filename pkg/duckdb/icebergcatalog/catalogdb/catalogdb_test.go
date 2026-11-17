package catalogdb

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/icebergcatalog"
	"github.com/stretchr/testify/require"
)

func TestOpenSQLite(t *testing.T) {
	dir := t.TempDir()
	c, err := Open(context.Background(), SQLite(filepath.Join(dir, "nested", "catalog.db")), icebergcatalog.Config{Warehouse: filepath.Join(dir, "warehouse")})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodGet, c.URL+"/v1/config", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, c.Close())
	require.ErrorContains(t, c.db.Ping(), "closed", "Close closes the db too")
}

// TestOpenSQLiteOddPath checks the catalog lands at its path even when
// the path has URI-significant characters in it.
func TestOpenSQLiteOddPath(t *testing.T) {
	// A subtest's TempDir is named after it, so the warehouse (which can't
	// have these characters) comes from the parent's.
	whDir := t.TempDir()
	for i, name := range []string{"hash#dir", "q?dir", "pct%41dir", "a space"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name, "catalog.db")
			c, err := Open(context.Background(), SQLite(path), icebergcatalog.Config{Warehouse: filepath.Join(whDir, fmt.Sprint(i), "warehouse")})
			require.NoError(t, err)
			require.NoError(t, c.Close())
			require.FileExists(t, path)
		})
	}

	t.Run("relative", func(t *testing.T) {
		t.Chdir(t.TempDir())
		path := filepath.Join("rel#dir", "catalog.db")
		c, err := Open(context.Background(), SQLite(path), icebergcatalog.Config{Warehouse: filepath.Join(whDir, "relative", "warehouse")})
		require.NoError(t, err)
		require.NoError(t, c.Close())
		require.FileExists(t, path)
	})
}

func TestOpenDBNeedsOneBackend(t *testing.T) {
	for _, b := range []Backend{{}, {SQLitePath: "x", PostgresURI: "y"}} {
		_, _, err := b.OpenDB()
		require.ErrorContains(t, err, "exactly one")
	}
}
