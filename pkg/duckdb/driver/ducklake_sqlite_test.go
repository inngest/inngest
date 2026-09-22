package driver

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestDuckLakeSQLiteCatalogAttaches proves a DuckLakeOptions.SQLiteCatalogPath
// actually attaches against a SQLite catalog file: create a table, insert a
// row, read it back through the same lake alias file/postgres catalogs use.
// Unlike the postgres-catalog tests (requireLocalPostgres, an external
// server), this needs nothing beyond the standard pinned duckdb binary and
// its sqlite extension — the catalog "server" is just the file itself.
func TestDuckLakeSQLiteCatalogAttaches(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	dir := t.TempDir()
	table := "sqlite_catalog_" + ulid.Make().String()

	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", &DuckLakeOptions{
		SQLiteCatalogPath: filepath.Join(dir, "catalog.sqlite"),
		DataPath:          filepath.Join(dir, "data"),
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(context.Background()) })

	_, _, err = p.exec(t.Context(), fmt.Sprintf("CREATE TABLE inngest.%s (id INTEGER);", table))
	require.NoError(t, err)

	_, _, err = p.exec(t.Context(), fmt.Sprintf("INSERT INTO inngest.%s VALUES (1);", table))
	require.NoError(t, err)

	_, rows, err := p.exec(t.Context(), fmt.Sprintf("SELECT count(*) AS c FROM inngest.%s;", table))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, float64(1), rows[0].get("c"))
}

// TestDuckLakeSQLiteCatalogHandlesJSONColumn guards against a SQLite-catalog
// analogue of ducklake#1175 (the postgres-catalog JSON-inlining corruption
// bug): a small insert with a JSON column, left at the default inlining
// limit (unlike postgres, SQLiteCatalogPath doesn't force inlining off), must
// round-trip correctly.
func TestDuckLakeSQLiteCatalogHandlesJSONColumn(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	dir := t.TempDir()
	table := "sqlite_json_" + ulid.Make().String()

	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", &DuckLakeOptions{
		SQLiteCatalogPath: filepath.Join(dir, "catalog.sqlite"),
		DataPath:          filepath.Join(dir, "data"),
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(context.Background()) })

	_, _, err = p.exec(t.Context(), fmt.Sprintf("CREATE TABLE inngest.%s (id INTEGER, data JSON);", table))
	require.NoError(t, err)

	_, _, err = p.exec(t.Context(), fmt.Sprintf(`INSERT INTO inngest.%s VALUES (1, '{"a": 1}');`, table))
	require.NoError(t, err)

	_, rows, err := p.exec(t.Context(), fmt.Sprintf("SELECT data FROM inngest.%s WHERE id = 1;", table))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, map[string]any{"a": float64(1)}, rows[0].get("data"))
}

// TestOpenWithDuckLakeSQLiteCatalog exercises the SQLite catalog mode
// through the public Open API, mirroring TestOpenWithDuckLakePostgresCatalog.
func TestOpenWithDuckLakeSQLiteCatalog(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	dir := t.TempDir()
	table := "sqlite_open_" + ulid.Make().String()

	db, err := Open(t.Context(), Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &DuckLakeOptions{
			SQLiteCatalogPath: filepath.Join(dir, "catalog.sqlite"),
			DataPath:          filepath.Join(dir, "data"),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), fmt.Sprintf("CREATE TABLE inngest.%s (id INTEGER);", table))
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), fmt.Sprintf("INSERT INTO inngest.%s VALUES (7);", table))
	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRowContext(t.Context(), fmt.Sprintf("SELECT count(*) AS c FROM inngest.%s;", table)).Scan(&count))
	require.Equal(t, 1, count)
}
