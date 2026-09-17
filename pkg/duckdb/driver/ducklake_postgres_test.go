package driver

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// defaultLocalPostgresTestURI is the DSN these tests connect to unless
// DUCKLAKE_POSTGRES_TEST_URI overrides it. It matches the postgres/postgres
// superuser and inngest_test database this repo's own e2e CI already spins up
// (see .github/workflows/e2e.yml), so the same local
// `docker run -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres postgres`
// setup works here without extra configuration.
const defaultLocalPostgresTestURI = "postgres://postgres:postgres@localhost:5432/inngest_test?sslmode=disable"

// requireLocalPostgres skips the test unless a real, reachable postgres
// server is available. These tests exercise DuckLake's own postgres catalog
// backend end-to-end — there is no meaningful way to fake that — so they
// must never fail an ordinary `go test ./...` run that doesn't have a local
// postgres running; they only run for a developer who has one up (or in a CI
// job that explicitly provides DUCKLAKE_POSTGRES_TEST_URI).
func requireLocalPostgres(t *testing.T) string {
	t.Helper()

	uri := defaultLocalPostgresTestURI
	if env, ok := os.LookupEnv("DUCKLAKE_POSTGRES_TEST_URI"); ok {
		uri = env
	}

	db, err := sql.Open("pgx", uri)
	if err != nil {
		t.Skipf("skipping: could not open postgres connection %q: %v", uri, err)
	}
	defer func() { _ = db.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("skipping: local postgres not reachable at %q: %v (set DUCKLAKE_POSTGRES_TEST_URI to point elsewhere)", uri, err)
	}

	return uri
}

// TestDuckLakePostgresCatalogAttaches proves a DuckLakeOptions.PostgresCatalogURI
// actually attaches against a live postgres catalog: create a table, insert a
// row, read it back through the same lake alias file-based catalogs use.
func TestDuckLakePostgresCatalogAttaches(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	pgURI := requireLocalPostgres(t)

	dir := t.TempDir()
	table := "pg_catalog_" + ulid.Make().String()

	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", &DuckLakeOptions{
		PostgresCatalogURI: pgURI,
		DataPath:           filepath.Join(dir, "data"),
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _, _ = p.exec(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS inngest.%s;", table))
		_ = p.close(context.Background())
	})

	_, _, err = p.exec(t.Context(), fmt.Sprintf("CREATE TABLE inngest.%s (id INTEGER);", table))
	require.NoError(t, err)

	_, _, err = p.exec(t.Context(), fmt.Sprintf("INSERT INTO inngest.%s VALUES (1);", table))
	require.NoError(t, err)

	_, rows, err := p.exec(t.Context(), fmt.Sprintf("SELECT count(*) AS c FROM inngest.%s;", table))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, float64(1), rows[0]["c"])
}

// TestDuckLakePostgresCatalogInliningStaysOffWithJSONColumn is the live
// regression test for https://github.com/duckdb/ducklake/issues/1175: on a
// postgres catalog, inserting a row with a JSON column used to corrupt the
// catalog once inlining kicked in. Because bootstrapDuckLakeLocked forces
// DATA_INLINING_ROW_LIMIT to 0 for a postgres catalog (see
// duckLakeBootstrapStmts), a single small insert with a JSON column must
// succeed and immediately flush to Parquet (file_count > 0) rather than
// attempt to inline.
func TestDuckLakePostgresCatalogInliningStaysOffWithJSONColumn(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	pgURI := requireLocalPostgres(t)

	dir := t.TempDir()
	table := "pg_json_" + ulid.Make().String()

	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", &DuckLakeOptions{
		PostgresCatalogURI: pgURI,
		DataPath:           filepath.Join(dir, "data"),
		// Deliberately request inlining -- duckLakeBootstrapStmts must
		// override this to 0 for a postgres catalog regardless.
		DataInliningRowLimit: 1000,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _, _ = p.exec(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS inngest.%s;", table))
		_ = p.close(context.Background())
	})

	_, _, err = p.exec(t.Context(), fmt.Sprintf("CREATE TABLE inngest.%s (id INTEGER, data JSON);", table))
	require.NoError(t, err)

	_, _, err = p.exec(t.Context(), fmt.Sprintf(`INSERT INTO inngest.%s VALUES (1, '{"a": 1}');`, table))
	require.NoError(t, err, "a JSON-column insert must not fail against a postgres catalog (ducklake#1175)")

	_, rows, err := p.exec(t.Context(), fmt.Sprintf("SELECT file_count FROM ducklake_table_info('inngest') WHERE table_name = '%s';", table))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Greater(t, rows[0]["file_count"], float64(0), "inlining must be forced off for a postgres catalog, so even one small insert should flush to Parquet")
}

// TestOpenWithDuckLakePostgresCatalog exercises the Postgres catalog mode
// through the public Open API, mirroring TestOpenWithDuckLake.
func TestOpenWithDuckLakePostgresCatalog(t *testing.T) {
	binPath := RequireDuckDBBinary(t)
	pgURI := requireLocalPostgres(t)

	dir := t.TempDir()
	table := "pg_open_" + ulid.Make().String()

	db, err := Open(t.Context(), Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &DuckLakeOptions{
			PostgresCatalogURI: pgURI,
			DataPath:           filepath.Join(dir, "data"),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS inngest.%s;", table))
		_ = db.Close()
	})

	_, err = db.ExecContext(t.Context(), fmt.Sprintf("CREATE TABLE inngest.%s (id INTEGER);", table))
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), fmt.Sprintf("INSERT INTO inngest.%s VALUES (7);", table))
	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRowContext(t.Context(), fmt.Sprintf("SELECT count(*) AS c FROM inngest.%s;", table)).Scan(&count))
	require.Equal(t, 1, count)
}
