package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// selfBuiltDuckLakeBuild names the local files a self-built DuckDB the
// ducklake#1175/postgres-multi-statement-fix investigation produced:
//   - a duckdb CLI with ducklake statically linked (built from
//     github.com/duckdb/ducklake's own vendored duckdb submodule, which
//     already contains the postgres_metadata_manager.cpp SplitBatchStatements
//     fix from ducklake#1407 — see FINDINGS/notes.md)
//   - postgres_scanner and quack built as loadable .duckdb_extension files
//     from that exact same duckdb commit (quack from a separate
//     github.com/duckdb/duckdb-quack checkout, with its own duckdb submodule
//     pointed at the same commit so the two are ABI-compatible)
//
// These are never present in a fresh checkout or CI — they're multi-hundred-
// megabyte local build artifacts from an ad hoc C++ build, not something this
// repo vendors or downloads. The test is skipped unless all three paths are
// given via env vars and actually exist on disk.
type selfBuiltDuckLakeBuild struct {
	duckdbBinary           string
	postgresScannerExtPath string
	quackExtPath           string
	httpfsExtPath          string
}

func requireSelfBuiltDuckLakeBuild(t *testing.T) selfBuiltDuckLakeBuild {
	t.Helper()

	b := selfBuiltDuckLakeBuild{
		duckdbBinary:           os.Getenv("DUCKLAKE_SELFBUILT_DUCKDB"),
		postgresScannerExtPath: os.Getenv("DUCKLAKE_SELFBUILT_POSTGRES_SCANNER"),
		quackExtPath:           os.Getenv("DUCKLAKE_SELFBUILT_QUACK"),
		httpfsExtPath:          os.Getenv("DUCKLAKE_SELFBUILT_HTTPFS"),
	}
	if b.duckdbBinary == "" || b.postgresScannerExtPath == "" || b.quackExtPath == "" || b.httpfsExtPath == "" {
		t.Skip("skipping: set DUCKLAKE_SELFBUILT_DUCKDB, DUCKLAKE_SELFBUILT_POSTGRES_SCANNER, DUCKLAKE_SELFBUILT_QUACK, and DUCKLAKE_SELFBUILT_HTTPFS to run against a self-built DuckDB+DuckLake+postgres_scanner+quack+httpfs (see FINDINGS/notes.md)")
	}
	for _, p := range []string{b.duckdbBinary, b.postgresScannerExtPath, b.quackExtPath, b.httpfsExtPath} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("skipping: %v", err)
		}
	}
	return b
}

// TestSelfBuiltDuckLakePostgresCatalogEndToEnd is the full-stack regression
// test for the ducklake#1175-adjacent bug this session found and fixed
// upstream (not yet published to any resolvable extension-repository build):
// DuckLake's postgres catalog backend rejected its own batched metadata SQL
// with "cannot insert multiple commands into a prepared statement" on every
// CREATE TABLE, on every DuckDB nightly available at the time. A self-built
// duckdb (see selfBuiltDuckLakeBuild) confirmed the fix already on the
// ducklake main branch; this test proves it through this package's actual
// production driver — Options.AllowUnsignedExtensions/LocalExtensionPaths,
// not a hand-rolled CLI invocation — covering both the postgres catalog
// attach/CREATE TABLE and the quack transport together, exactly as
// production and cmd/duckdbbench would drive them.
func TestSelfBuiltDuckLakePostgresCatalogEndToEnd(t *testing.T) {
	build := requireSelfBuiltDuckLakeBuild(t)
	pgURI := requireLocalPostgres(t)

	dir := t.TempDir()
	addr := "127.0.0.1:0"
	table := "selfbuilt_" + ulid.Make().String()

	db, err := Open(t.Context(), Options{
		BinaryPath: build.duckdbBinary,
		DBFile:     ":memory:",
		DuckLake: &DuckLakeOptions{
			PostgresCatalogURI: pgURI,
			DataPath:           filepath.Join(dir, "data"),
			// Deliberately request inlining -- must still be forced to 0 for
			// a postgres catalog (see duckLakePostgresInliningOverridden),
			// same guarantee as the (upstream-blocked) ducklake_postgres_test.go
			// tests this mirrors.
			DataInliningRowLimit: 1000,
		},
		AllowUnsignedExtensions: true,
		LocalExtensionPaths: map[string]string{
			"postgres": build.postgresScannerExtPath,
			"quack":    build.quackExtPath,
			// httpfs is only loaded as a prerequisite so quack_serve gets a
			// real (not read-only) crypto engine on this from-scratch
			// self-build -- see Options.LocalExtensionPaths's doc comment.
			"httpfs": build.httpfsExtPath,
			// ducklake is statically linked into this self-built binary (no
			// separate .duckdb_extension file), so plain "LOAD ducklake;"
			// already resolves it -- the bare name also works as a LOAD
			// path (verified: `duckdb :memory: -c "LOAD 'ducklake';"`
			// succeeds identically to unquoted `LOAD ducklake;`). This
			// skips the ordinary "INSTALL ducklake;" this package would
			// otherwise also issue, which fails outright: INSTALL always
			// hits the network regardless of static linkage, and this
			// commit was never published to the extension repository.
			"ducklake": "ducklake",
		},
		QuackAddr: &addr,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS inngest.%s;", table))
		_ = db.Close()
	})

	_, err = db.ExecContext(t.Context(), fmt.Sprintf("CREATE TABLE inngest.%s (id INTEGER, data JSON);", table))
	require.NoError(t, err, "CREATE TABLE against a postgres catalog must succeed on the fixed build")

	_, err = db.ExecContext(t.Context(), fmt.Sprintf(`INSERT INTO inngest.%s VALUES (1, '{"a": 1}');`, table))
	require.NoError(t, err, "a JSON-column insert must not fail (ducklake#1175)")

	var count int
	require.NoError(t, db.QueryRowContext(t.Context(), fmt.Sprintf("SELECT count(*) AS c FROM inngest.%s;", table)).Scan(&count))
	require.Equal(t, 1, count)

	var fileCount int
	require.NoError(t, db.QueryRowContext(t.Context(),
		fmt.Sprintf("SELECT file_count FROM ducklake_table_info('inngest') WHERE table_name = '%s';", table)).
		Scan(&fileCount))
	require.Greater(t, fileCount, 0, "inlining must be forced off for a postgres catalog even on this build")
}
