package driver

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/icebergcatalog"
	"github.com/stretchr/testify/require"
)

// testIceberg starts an embedded Iceberg catalog on a temp dir for the rest
// of the test. Other packages use icebergtest.Start, which can't be
// imported here (it imports this package).
func testIceberg(t testing.TB) *IcebergOptions {
	t.Helper()
	dir := t.TempDir()
	e, err := icebergcatalog.Open(context.Background(), icebergcatalog.Config{
		SQLitePath: filepath.Join(dir, "catalog.db"),
		Warehouse:  filepath.Join(dir, "warehouse"),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.Close() })
	return &IcebergOptions{URI: e.URL, Warehouse: "inngest", Token: e.Token, DataPath: e.Warehouse}
}

func TestIcebergBootstrapStmts(t *testing.T) {
	stmts, err := icebergBootstrapStmts(IcebergOptions{
		URI:       "http://127.0.0.1:1234/",
		Warehouse: "inngest",
		Token:     "tok",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, []string{
		"INSTALL httpfs;",
		"LOAD httpfs;",
		"INSTALL iceberg;",
		"LOAD iceberg;",
		"CREATE OR REPLACE SECRET inngest_iceberg (TYPE iceberg, TOKEN 'tok');",
		"ATTACH IF NOT EXISTS 'inngest' AS inngest (TYPE iceberg, URI 'http://127.0.0.1:1234', SECRET inngest_iceberg);",
		"CREATE SCHEMA IF NOT EXISTS inngest.main;",
		"DETACH inngest;",
		"ATTACH 'inngest' AS inngest (TYPE iceberg, URI 'http://127.0.0.1:1234', SECRET inngest_iceberg, DEFAULT_SCHEMA 'main');",
	}, stmts)

	stmts, err = icebergBootstrapStmts(IcebergOptions{URI: "http://x", Warehouse: "w"}, map[string]string{"iceberg": "/ext/iceberg.duckdb_extension"})
	require.NoError(t, err)
	require.Contains(t, stmts, "LOAD '/ext/iceberg.duckdb_extension';")
	require.NotContains(t, stmts, "INSTALL iceberg;")
	require.Contains(t, strings.Join(stmts, "\n"), "AUTHORIZATION_TYPE 'none'")

	_, err = icebergBootstrapStmts(IcebergOptions{Warehouse: "w"}, nil)
	require.Error(t, err)
	_, err = icebergBootstrapStmts(IcebergOptions{URI: "http://x"}, nil)
	require.Error(t, err)
}

// TestIcebergAttachReadWrite covers what dual-write and the federated reads
// need from the lake: format-version 3 tables w/ VARIANT, partitioning by
// account + time, deletes, and pinned (time travel) reads.
func TestIcebergAttachReadWrite(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:", Iceberg: testIceberg(t)})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := t.Context()

	for _, q := range []string{
		"CREATE TABLE inngest.t (account_id UUID NOT NULL, ts TIMESTAMPTZ NOT NULL, tags VARCHAR[], v VARIANT) PARTITIONED BY (account_id, month(ts)) WITH ('format-version' = '3');",
		"ALTER TABLE inngest.t SET SORTED BY (account_id, ts);",
		"INSERT INTO inngest.t VALUES ('00000000-0000-0000-0000-000000000001', '2026-01-01T00:00:00Z', ['a'], '{\"x\":1}'::JSON::VARIANT), ('00000000-0000-0000-0000-000000000002', '2026-02-01T00:00:00Z', [], 2::VARIANT);",
	} {
		_, err := db.ExecContext(ctx, q)
		require.NoError(t, err, q)
	}

	var snap int64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT max(snapshot_id) FROM iceberg_snapshots('inngest.main.t');").Scan(&snap))

	_, err = db.ExecContext(ctx, "DELETE FROM inngest.t WHERE account_id = '00000000-0000-0000-0000-000000000001';")
	require.NoError(t, err)

	count := func(q string) int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, q).Scan(&n))
		return n
	}
	require.Equal(t, 1, count("SELECT count(*) FROM inngest.t;"))
	require.Equal(t, 2, count(fmt.Sprintf("SELECT count(*) FROM inngest.t AT (VERSION => %d);", snap)))
	require.Equal(t, 1, count(fmt.Sprintf("SELECT count(*) FROM inngest.t AT (VERSION => %d) WHERE v.x::INT = 1;", snap)))
}

// TestIcebergReattachesAfterRestart proves a crash-triggered respawn comes
// back w/ the lake attached (and its default schema), not just healthy.
func TestIcebergReattachesAfterRestart(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	p, err := startProcessWithLake(t.Context(), binPath, ":memory:", testIceberg(t), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(context.Background()) })

	_, _, err = p.Exec(t.Context(), "CREATE TABLE inngest.r (id INTEGER); INSERT INTO inngest.r VALUES (1);")
	require.NoError(t, err)
	require.NoError(t, p.restart(t.Context()))
	_, rows, err := p.Exec(t.Context(), "SELECT count(*) AS n FROM inngest.r;")
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

// TestIcebergSurvivesProcessReopen proves the lake persists across
// processes against the same catalog, IE a devserver restart.
func TestIcebergSurvivesProcessReopen(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	ice := testIceberg(t)
	open := func() *sql.DB {
		db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:", Iceberg: ice})
		require.NoError(t, err)
		return db
	}

	db := open()
	_, err := db.ExecContext(t.Context(), "CREATE TABLE inngest.p (id INTEGER);")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), "INSERT INTO inngest.p VALUES (1), (2);")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	db = open()
	t.Cleanup(func() { _ = db.Close() })
	var n int
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) FROM inngest.p;").Scan(&n))
	require.Equal(t, 2, n)
}
