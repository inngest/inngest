package driver

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// startQuackCatalogServer spawns a bare (no DuckLake attach of its own)
// quack-serving subprocess for TestDuckLakeQuackCatalogAttaches/
// TestOpenWithDuckLakeQuackCatalog to attach a DuckLake catalog against, as
// a remote metadata store — DuckLakeOptions.QuackCatalogAddr/
// QuackCatalogToken's "the remote instance is expected to already be
// listening" precondition. A fixed token (rather than the ordinary random
// per-spawn one) is required here specifically so the attaching side can
// know it in advance — see Options.QuackServeToken's doc comment.
func startQuackCatalogServer(t *testing.T, binPath, addr, token string) *sql.DB {
	t.Helper()
	db, err := Open(t.Context(), Options{
		BinaryPath:      binPath,
		DBFile:          ":memory:",
		QuackAddr:       &addr,
		QuackServeToken: token,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestDuckLakeQuackCatalogAttaches proves a DuckLakeOptions.QuackCatalogAddr
// actually attaches against a remote, quack-served DuckDB instance acting as
// the metadata catalog: create a table, insert a row, read it back through
// the same lake alias file/postgres/sqlite catalogs use. The metadata server
// here has only the quack extension loaded (no ducklake) — confirmed
// manually that this still works (falls back to an ordinary client-driven
// commit instead of QuackMetadataManager's server-side fast path), so this
// pins the same minimal, correctness-only case rather than requiring the
// heavier both-processes-need-ducklake setup.
func TestDuckLakeQuackCatalogAttaches(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	metadataAddr := freeLocalAddr(t)
	const metadataToken = "test-quack-catalog-token"
	startQuackCatalogServer(t, binPath, metadataAddr, metadataToken)

	dir := t.TempDir()
	table := "quack_catalog_" + ulid.Make().String()

	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", &DuckLakeOptions{
		QuackCatalogAddr:  metadataAddr,
		QuackCatalogToken: metadataToken,
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

// TestOpenWithDuckLakeQuackCatalog exercises the quack catalog mode through
// the public Open API, mirroring TestOpenWithDuckLakeSQLiteCatalog.
func TestOpenWithDuckLakeQuackCatalog(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	metadataAddr := freeLocalAddr(t)
	const metadataToken = "test-quack-catalog-token-2"
	startQuackCatalogServer(t, binPath, metadataAddr, metadataToken)

	dir := t.TempDir()
	table := "quack_open_" + ulid.Make().String()

	db, err := Open(t.Context(), Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &DuckLakeOptions{
			QuackCatalogAddr:  metadataAddr,
			QuackCatalogToken: metadataToken,
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

// TestDuckLakeQuackCatalogWithDuckLakeServerHandlesJSONColumn is
// TestDuckLakeQuackCatalogAttaches's counterpart with the metadata server
// also carrying the ducklake extension loaded (not attached — just
// available), exercising QuackMetadataManager's server-side commit fast
// path (ProbeServerCapabilities finds ducklake_commit server-side) rather
// than the minimal quack-only server. A small insert with a JSON column,
// left at the default inlining limit (no ducklake#1175-style bug here, same
// as SQLiteCatalogPath), must round-trip correctly either way.
func TestDuckLakeQuackCatalogWithDuckLakeServerHandlesJSONColumn(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	metadataAddr := freeLocalAddr(t)
	const metadataToken = "test-quack-catalog-token-3"
	metadataDB := startQuackCatalogServer(t, binPath, metadataAddr, metadataToken)
	// Loading ducklake here (without attaching anything) is what lets the
	// remote instance serve ducklake_commit for the server-side fast path —
	// confirmed manually that DuckLake's quack catalog mode works fine
	// without it too (TestDuckLakeQuackCatalogAttaches), this just exercises
	// the other branch.
	_, err := metadataDB.ExecContext(t.Context(), "INSTALL ducklake; LOAD ducklake;")
	require.NoError(t, err)

	dir := t.TempDir()
	table := "quack_json_" + ulid.Make().String()

	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", &DuckLakeOptions{
		QuackCatalogAddr:  metadataAddr,
		QuackCatalogToken: metadataToken,
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
