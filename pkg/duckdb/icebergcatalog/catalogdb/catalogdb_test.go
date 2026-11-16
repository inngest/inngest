package catalogdb

import (
	"context"
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

func TestOpenDBNeedsOneBackend(t *testing.T) {
	for _, b := range []Backend{{}, {SQLitePath: "x", PostgresURI: "y"}} {
		_, _, err := b.OpenDB()
		require.ErrorContains(t, err, "exactly one")
	}
}
