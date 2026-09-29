package driver

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/duckdbtest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestSandboxStmts(t *testing.T) {
	stmts, err := sandboxStmts(nil)
	require.NoError(t, err)
	require.Equal(t, []string{"SET enable_external_access=false;"}, stmts)

	stmts, err = sandboxStmts(&DuckLakeOptions{DataPath: "/lake/data"})
	require.NoError(t, err)
	require.Equal(t, []string{
		"SET allowed_directories=['/lake/data/'];",
		"SET enable_external_access=false;",
	}, stmts)
}

// requireSandboxed asserts db can't reach the filesystem or environment
// outside what sandboxStmts allows, and can't undo that.
func requireSandboxed(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := t.Context()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o600))

	for _, q := range []string{
		"SELECT getenv('HOME') AS v;",
		fmt.Sprintf("SELECT content AS v FROM read_text('%s');", outside),
		fmt.Sprintf("SELECT * FROM read_csv('%s');", outside),
		fmt.Sprintf("SELECT file AS v FROM glob('%s');", filepath.Join(filepath.Dir(outside), "*")),
		fmt.Sprintf("ATTACH '%s' AS other;", filepath.Join(filepath.Dir(outside), "other.duckdb")),
		fmt.Sprintf("COPY (SELECT 1) TO '%s';", filepath.Join(filepath.Dir(outside), "out.csv")),
		"SET enable_external_access=true;",
		fmt.Sprintf("SET allowed_directories=['%s/'];", filepath.Dir(outside)),
	} {
		_, err := db.ExecContext(ctx, q)
		require.Error(t, err, q)
	}
}

// TestRestrictExternalAccessKeepsDuckLakeWorking proves the sandbox blocks
// file/env access while DuckLake reads, writes, deletes, and compaction —
// everything dual-write does — keep working, over both the primary and
// pooled quack connections. The Postgres catalog isn't covered (its first
// commit failed against postgres:16 with the pinned binary even without the
// sandbox, "cannot insert multiple commands into a prepared statement"), so
// setupDualWrite leaves the sandbox off in that mode until it can be verified.
func TestRestrictExternalAccessKeepsDuckLakeWorking(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	catalogs := map[string]func(t *testing.T, dir string) *DuckLakeOptions{
		"duckdb": func(t *testing.T, dir string) *DuckLakeOptions {
			return &DuckLakeOptions{CatalogPath: filepath.Join(dir, "catalog.duckdb")}
		},
		"sqlite": func(t *testing.T, dir string) *DuckLakeOptions {
			return &DuckLakeOptions{SQLiteCatalogPath: filepath.Join(dir, "catalog.sqlite")}
		},
		"quack": func(t *testing.T, dir string) *DuckLakeOptions {
			addr := duckdbtest.FreeLocalAddr(t)
			const token = "test-sandbox-quack-catalog-token"
			startQuackCatalogServer(t, binPath, addr, token)
			return &DuckLakeOptions{QuackCatalogAddr: addr, QuackCatalogToken: token}
		},
	}

	for name, catalog := range catalogs {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			lake := catalog(t, dir)
			lake.DataPath = filepath.Join(dir, "data")
			lake.DataInliningRowLimit = -1 // force real Parquet files under DataPath

			addr := EphemeralQuackAddr
			db, err := Open(t.Context(), Options{
				BinaryPath:             binPath,
				DBFile:                 filepath.Join(dir, "main.duckdb"),
				DuckLake:               lake,
				QuackAddr:              &addr,
				QuackConns:             4,
				RestrictExternalAccess: true,
			})
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })

			// Hold several connections at once so pooled ones are exercised too.
			ctx := t.Context()
			var conns []*sql.Conn
			for range 3 {
				c, err := db.Conn(ctx)
				require.NoError(t, err)
				conns = append(conns, c)
			}
			for _, c := range conns {
				require.NoError(t, c.Close())
			}

			table := "inngest.sandbox_" + ulid.Make().String()
			for _, q := range []string{
				fmt.Sprintf("CREATE TABLE %s (id INTEGER, s VARCHAR);", table),
				fmt.Sprintf("INSERT INTO %s SELECT i, 'a' FROM range(100) r(i);", table),
				fmt.Sprintf("INSERT INTO %s SELECT i, 'b' FROM range(100) r(i);", table),
				fmt.Sprintf("DELETE FROM %s WHERE id < 10;", table),
				fmt.Sprintf("CALL ducklake_merge_adjacent_files('%s');", DuckLakeAlias),
			} {
				_, err := db.ExecContext(ctx, q)
				require.NoError(t, err, q)
			}
			var n int
			require.NoError(t, db.QueryRowContext(ctx, fmt.Sprintf("SELECT count(*) FROM %s;", table)).Scan(&n))
			require.Equal(t, 180, n)

			requireSandboxed(t, db)
		})
	}
}

// TestRestrictExternalAccessSurvivesRestart proves a crash-triggered respawn
// is sandboxed again, not left wide open.
func TestRestrictExternalAccessSurvivesRestart(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, nil, withRestrictExternalAccess(true))
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(context.Background()) })

	_, _, err = p.Exec(t.Context(), "SELECT getenv('HOME') AS v;")
	require.ErrorContains(t, err, "getenv is disabled")

	require.NoError(t, p.restart(t.Context()))

	_, _, err = p.Exec(t.Context(), "SELECT getenv('HOME') AS v;")
	require.ErrorContains(t, err, "getenv is disabled")
}

// TestRestrictExternalAccessOffByDefault pins that the sandbox is opt-in.
func TestRestrictExternalAccessOffByDefault(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	p, err := startProcess(t.Context(), binPath, ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(context.Background()) })

	_, rows, err := p.Exec(t.Context(), "SELECT current_setting('enable_external_access') AS v;")
	require.NoError(t, err)
	require.Equal(t, true, rows[0].Get("v"))
}
