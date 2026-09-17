package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDuckLakeBootstrapAttachesOnStart covers the happy path of the opt-in
// bootstrap: with DuckLake enabled, a freshly started subprocess must already
// have the lake catalog attached, so a caller can create and query a
// DuckLake-backed table without issuing any INSTALL/LOAD/ATTACH itself. It
// also pins the os.MkdirAll behaviour — the data path directory is *not*
// pre-created here, because DuckLake requires it to exist before ATTACH runs.
func TestDuckLakeBootstrapAttachesOnStart(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	dir := t.TempDir()
	catalog := filepath.Join(dir, "catalog.ducklake")
	dataPath := filepath.Join(dir, "data")

	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", &DuckLakeOptions{
		CatalogPath: catalog,
		DataPath:    dataPath,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	info, err := os.Stat(dataPath)
	require.NoError(t, err, "bootstrap must create the DuckLake data directory")
	require.True(t, info.IsDir())

	_, _, err = p.exec(t.Context(), "CREATE TABLE inngest.dl_t (id INTEGER);")
	require.NoError(t, err)

	_, _, err = p.exec(t.Context(), "INSERT INTO inngest.dl_t VALUES (1);")
	require.NoError(t, err)

	_, rows, err := p.exec(t.Context(), "SELECT count(*) AS c FROM inngest.dl_t;")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, float64(1), rows[0]["c"])
}

// TestDuckLakeInlinesSmallInsertsUpToRowLimit pins the
// DATA_INLINING_ROW_LIMIT the bootstrap attaches with: small batched inserts
// (mirroring dual-write's flush pattern and cmd/duckdbseed's batched writes)
// must stay inlined in the DuckLake catalog rather than each becoming its
// own tiny Parquet file. Verified empirically against duckdb v1.5.5: without
// this option, five separate 200-row INSERTs produce five Parquet files;
// with it set to 1000, the same five inserts (1000 rows total) produce
// none — ducklake_table_info's file_count stays 0.
func TestDuckLakeInlinesSmallInsertsUpToRowLimit(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	dir := t.TempDir()
	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", &DuckLakeOptions{
		CatalogPath: filepath.Join(dir, "catalog.ducklake"),
		DataPath:    filepath.Join(dir, "data"),
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	_, _, err = p.exec(t.Context(), "CREATE TABLE inngest.inline_t (id INTEGER);")
	require.NoError(t, err)

	for range 5 {
		_, _, err = p.exec(t.Context(), "INSERT INTO inngest.inline_t SELECT range FROM range(200);")
		require.NoError(t, err)
	}

	_, rows, err := p.exec(t.Context(), "SELECT file_count FROM ducklake_table_info('inngest') WHERE table_name = 'inline_t';")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, float64(0), rows[0]["file_count"], "1000 rows at the row limit must stay inlined, not flushed to Parquet")
}

// TestDuckLakeInliningRowLimitIsConfigurable proves
// DuckLakeOptions.DataInliningRowLimit actually reaches the ATTACH
// statement, not just DefaultDataInliningRowLimit: a caller-supplied limit
// low enough to be exceeded by a single insert must produce a real Parquet
// file, unlike the default-limit case above where the same-shaped writes
// stay inlined.
func TestDuckLakeInliningRowLimitIsConfigurable(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	dir := t.TempDir()
	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", &DuckLakeOptions{
		CatalogPath:          filepath.Join(dir, "catalog.ducklake"),
		DataPath:             filepath.Join(dir, "data"),
		DataInliningRowLimit: 2,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	_, _, err = p.exec(t.Context(), "CREATE TABLE inngest.low_limit_t (id INTEGER);")
	require.NoError(t, err)

	_, _, err = p.exec(t.Context(), "INSERT INTO inngest.low_limit_t SELECT range FROM range(200);")
	require.NoError(t, err)

	_, rows, err := p.exec(t.Context(), "SELECT file_count FROM ducklake_table_info('inngest') WHERE table_name = 'low_limit_t';")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Greater(t, rows[0]["file_count"], float64(0), "200 rows must exceed a row limit of 2 and flush to Parquet")
}

// TestDuckLakeReattachesAfterCrash is the whole reason the bootstrap lives
// inside the process lifecycle rather than being a one-shot call from Open's
// caller. A DuckDB subprocess starts with a completely fresh, unattached
// session every time it spawns, so a crash-triggered restart that only
// health-checks would come back "healthy" with no lake catalog at all, and
// every subsequent lake.* statement would fail with a Catalog Error.
//
// This writes a row to a DuckLake table, SIGKILLs the real subprocess, then
// queries the lake table through p.exec — the same path conn.go uses, which
// detects the dead session, restarts, and retries. A successful count of 1
// proves both that the restart re-attached the catalog and that DuckLake's own
// on-disk durability survived the crash (the main database is :memory:, so the
// row can only have come from the DuckLake data files).
func TestDuckLakeReattachesAfterCrash(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	dir := t.TempDir()
	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", &DuckLakeOptions{
		CatalogPath: filepath.Join(dir, "catalog.ducklake"),
		DataPath:    filepath.Join(dir, "data"),
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	_, _, err = p.exec(t.Context(), "CREATE TABLE inngest.crash_t (id INTEGER);")
	require.NoError(t, err)
	_, _, err = p.exec(t.Context(), "INSERT INTO inngest.crash_t VALUES (42);")
	require.NoError(t, err)

	pidBefore := p.cmd.Process.Pid

	// Simulate a crash out from under the session, bypassing close(), exactly
	// as the existing restart tests do.
	require.NoError(t, p.cmd.Process.Kill())
	_, _ = p.cmd.Process.Wait()

	// Goes through the production path: dead session detected -> restart ->
	// health check -> DuckLake bootstrap -> retry the statement.
	_, rows, err := p.exec(t.Context(), "SELECT count(*) AS c FROM inngest.crash_t;")
	require.NoError(t, err, "the restart must re-attach the DuckLake catalog")
	require.Len(t, rows, 1)
	require.Equal(t, float64(1), rows[0]["c"],
		"the pre-crash row must still be readable from the re-attached lake")

	p.mu.Lock()
	disabled := p.disabled
	pidAfter := p.cmd.Process.Pid
	p.mu.Unlock()
	require.False(t, disabled, "a successful restart must not disable the process")
	require.NotEqual(t, pidBefore, pidAfter, "the subprocess must actually have been respawned")

	// The re-attached session must keep working for writes too, not just the
	// one retried read.
	_, _, err = p.exec(t.Context(), "INSERT INTO inngest.crash_t VALUES (43);")
	require.NoError(t, err)
	_, rows, err = p.exec(t.Context(), "SELECT count(*) AS c FROM inngest.crash_t;")
	require.NoError(t, err)
	require.Equal(t, float64(2), rows[0]["c"])
}

// TestOpenWithDuckLake exercises the Options wiring through the public API.
func TestOpenWithDuckLake(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	dir := t.TempDir()
	db, err := Open(t.Context(), Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.ducklake"),
			DataPath:    filepath.Join(dir, "data"),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE inngest.open_t (id INTEGER);")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), "INSERT INTO inngest.open_t VALUES (7);")
	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) AS c FROM inngest.open_t;").Scan(&count))
	require.Equal(t, 1, count)
}

// TestOpenWithoutDuckLakeHasNoLakeCatalog pins the opt-in contract: the zero
// value of Options must behave exactly as before, with no lake catalog
// attached and no DuckLake extension loaded.
func TestOpenWithoutDuckLakeHasNoLakeCatalog(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE inngest.nope (id INTEGER);")
	require.Error(t, err, "no DuckLake catalog should be attached when DuckLake is not configured")
}

// TestInstallOrLoadStmtsDefaultsToInstallByName pins the ordinary path: no
// local override means the usual network-resolved INSTALL/LOAD pair.
func TestInstallOrLoadStmtsDefaultsToInstallByName(t *testing.T) {
	stmts, err := installOrLoadStmts("quack", nil)
	require.NoError(t, err)
	require.Equal(t, []string{"INSTALL quack;", "LOAD quack;"}, stmts)
}

// TestInstallOrLoadStmtsUsesLocalPathWhenConfigured pins the override this
// session's self-built dev extensions need: a configured local path loads
// directly by file path, with no INSTALL (and so no name-based resolution
// against the extension repository, which never has a build for an
// arbitrary local dev commit).
func TestInstallOrLoadStmtsUsesLocalPathWhenConfigured(t *testing.T) {
	stmts, err := installOrLoadStmts("quack", map[string]string{"quack": "/path/to/quack.duckdb_extension"})
	require.NoError(t, err)
	require.Equal(t, []string{"LOAD '/path/to/quack.duckdb_extension';"}, stmts)
}

// TestInstallOrLoadStmtsIgnoresUnrelatedLocalPaths pins that the local-path
// override is keyed by extension name: a map with entries for other
// extensions must not affect this one.
func TestInstallOrLoadStmtsIgnoresUnrelatedLocalPaths(t *testing.T) {
	stmts, err := installOrLoadStmts("quack", map[string]string{"ducklake": "/path/to/ducklake.duckdb_extension"})
	require.NoError(t, err)
	require.Equal(t, []string{"INSTALL quack;", "LOAD quack;"}, stmts)
}

// TestDuckLakeBootstrapStmtsUsesLocalExtensionPaths pins that
// duckLakeBootstrapStmts routes both the ducklake and (postgres-catalog
// mode) postgres extension loads through installOrLoadStmts, so a local
// build of either substitutes cleanly for the name-based INSTALL.
func TestDuckLakeBootstrapStmtsUsesLocalExtensionPaths(t *testing.T) {
	stmts, err := duckLakeBootstrapStmts(DuckLakeOptions{
		PostgresCatalogURI: "postgres://user:pass@localhost:5432/catalog",
		DataPath:           filepath.Join(t.TempDir(), "lake-data"),
	}, map[string]string{
		"ducklake": "/local/ducklake.duckdb_extension",
		"postgres": "/local/postgres_scanner.duckdb_extension",
	})
	require.NoError(t, err)
	require.Contains(t, stmts, "LOAD '/local/ducklake.duckdb_extension';")
	require.Contains(t, stmts, "LOAD '/local/postgres_scanner.duckdb_extension';")
	require.NotContains(t, stmts, "INSTALL ducklake;")
	require.NotContains(t, stmts, "INSTALL postgres;")
}

// TestDuckLakeBootstrapStmtsPostgresForcesInliningOff pins the workaround for
// https://github.com/duckdb/ducklake/issues/1175: DuckLake's postgres catalog
// backend corrupts inlined rows that contain JSON columns, so a postgres
// catalog must always attach with DATA_INLINING_ROW_LIMIT 0, even when the
// caller explicitly configured a non-zero DataInliningRowLimit.
func TestDuckLakeBootstrapStmtsPostgresForcesInliningOff(t *testing.T) {
	stmts, err := duckLakeBootstrapStmts(DuckLakeOptions{
		PostgresCatalogURI:   "postgres://user:pass@localhost:5432/catalog",
		DataPath:             filepath.Join(t.TempDir(), "lake-data"),
		DataInliningRowLimit: 500,
	}, nil)
	require.NoError(t, err)

	attach := requireAttachStmt(t, stmts)
	require.Contains(t, attach, "DATA_INLINING_ROW_LIMIT 0")
	require.NotContains(t, attach, "DATA_INLINING_ROW_LIMIT 500")
}

// TestDuckLakeBootstrapStmtsPostgresLoadsExtension pins that a postgres
// catalog additionally installs/loads the postgres extension before the
// ducklake ATTACH, since DuckLake's postgres catalog backend depends on it
// (verified against the ducklake#1175 repro).
func TestDuckLakeBootstrapStmtsPostgresLoadsExtension(t *testing.T) {
	stmts, err := duckLakeBootstrapStmts(DuckLakeOptions{
		PostgresCatalogURI: "postgres://user:pass@localhost:5432/catalog",
		DataPath:           filepath.Join(t.TempDir(), "lake-data"),
	}, nil)
	require.NoError(t, err)
	require.Contains(t, stmts, "INSTALL postgres;")
	require.Contains(t, stmts, "LOAD postgres;")

	installIdx := indexOfStmt(t, stmts, "INSTALL postgres;")
	ducklakeIdx := indexOfStmt(t, stmts, "INSTALL ducklake;")
	require.Less(t, installIdx, ducklakeIdx, "postgres extension must install before ducklake attaches against it")
}

// TestDuckLakeBootstrapStmtsFileCatalogDoesNotLoadPostgres is a regression
// guard: the existing file-based catalog path must stay byte-for-byte
// unaffected by the postgres-catalog addition.
func TestDuckLakeBootstrapStmtsFileCatalogDoesNotLoadPostgres(t *testing.T) {
	dir := t.TempDir()
	stmts, err := duckLakeBootstrapStmts(DuckLakeOptions{
		CatalogPath: filepath.Join(dir, "catalog.ducklake"),
		DataPath:    filepath.Join(dir, "lake-data"),
	}, nil)
	require.NoError(t, err)
	for _, s := range stmts {
		require.NotContains(t, s, "postgres", "a file-based catalog must never touch the postgres extension")
	}
}

// TestDuckLakeBootstrapStmtsMetadataSchema pins that a non-empty
// MetadataSchema adds a METADATA_SCHEMA clause to the ATTACH statement --
// needed to isolate repeated runs against the same Postgres database (each
// gets its own catalog namespace instead of colliding in the default
// schema), the postgres-catalog analogue of a fresh tb.TempDir() per file
// catalog.
func TestDuckLakeBootstrapStmtsMetadataSchema(t *testing.T) {
	stmts, err := duckLakeBootstrapStmts(DuckLakeOptions{
		PostgresCatalogURI: "postgres://user:pass@localhost:5432/catalog",
		DataPath:           filepath.Join(t.TempDir(), "lake-data"),
		MetadataSchema:     "bench_abc123",
	}, nil)
	require.NoError(t, err)

	attach := requireAttachStmt(t, stmts)
	require.Contains(t, attach, "METADATA_SCHEMA 'bench_abc123'")
}

// TestDuckLakeBootstrapStmtsNoMetadataSchemaByDefault pins the unset case:
// no METADATA_SCHEMA clause at all, leaving DuckLake's own default schema
// in effect -- a regression guard for every existing caller that never sets
// this field.
func TestDuckLakeBootstrapStmtsNoMetadataSchemaByDefault(t *testing.T) {
	stmts, err := duckLakeBootstrapStmts(DuckLakeOptions{
		PostgresCatalogURI: "postgres://user:pass@localhost:5432/catalog",
		DataPath:           filepath.Join(t.TempDir(), "lake-data"),
	}, nil)
	require.NoError(t, err)

	attach := requireAttachStmt(t, stmts)
	require.NotContains(t, attach, "METADATA_SCHEMA")
}

// TestDuckLakeBootstrapStmtsPostgresAttachTarget pins the ATTACH target
// format for a postgres catalog: 'ducklake:postgres:<URI>', distinct from the
// file-based 'ducklake:<CatalogPath>' form.
func TestDuckLakeBootstrapStmtsPostgresAttachTarget(t *testing.T) {
	stmts, err := duckLakeBootstrapStmts(DuckLakeOptions{
		PostgresCatalogURI: "postgres://user:pass@localhost:5432/catalog",
		DataPath:           filepath.Join(t.TempDir(), "lake-data"),
	}, nil)
	require.NoError(t, err)

	attach := requireAttachStmt(t, stmts)
	require.Contains(t, attach, "'ducklake:postgres:postgres://user:pass@localhost:5432/catalog'")
}

// TestDuckLakePostgresInliningOverridden pins exactly when the postgres
// forced-inlining-off override is worth warning a caller about: only an
// explicit, positive DataInliningRowLimit is silently discarded. Zero ("use
// the default") and negative ("explicitly disable") both already agree with
// (or are subsumed by) the forced override, so neither should warn -- and a
// non-postgres catalog never overrides anything, regardless of the limit.
func TestDuckLakePostgresInliningOverridden(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "lake-data")

	cases := []struct {
		name     string
		opts     DuckLakeOptions
		expected bool
	}{
		{
			name: "postgres with explicit positive limit warns",
			opts: DuckLakeOptions{
				PostgresCatalogURI:   "postgres://user:pass@localhost:5432/catalog",
				DataPath:             dataPath,
				DataInliningRowLimit: 500,
			},
			expected: true,
		},
		{
			name: "postgres with default (zero) limit does not warn",
			opts: DuckLakeOptions{
				PostgresCatalogURI: "postgres://user:pass@localhost:5432/catalog",
				DataPath:           dataPath,
			},
			expected: false,
		},
		{
			name: "postgres with explicitly-disabled (negative) limit does not warn",
			opts: DuckLakeOptions{
				PostgresCatalogURI:   "postgres://user:pass@localhost:5432/catalog",
				DataPath:             dataPath,
				DataInliningRowLimit: -1,
			},
			expected: false,
		},
		{
			name: "file catalog with positive limit does not warn",
			opts: DuckLakeOptions{
				CatalogPath:          filepath.Join(t.TempDir(), "catalog.ducklake"),
				DataPath:             dataPath,
				DataInliningRowLimit: 500,
			},
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, duckLakePostgresInliningOverridden(tc.opts))
		})
	}
}

// TestDuckLakeBootstrapStmtsRejectsAmbiguousCatalog covers the misconfigured
// case where a caller sets both CatalogPath and PostgresCatalogURI: exactly
// one catalog source is required, so this must fail loudly rather than
// silently preferring one.
func TestDuckLakeBootstrapStmtsRejectsAmbiguousCatalog(t *testing.T) {
	dir := t.TempDir()
	_, err := duckLakeBootstrapStmts(DuckLakeOptions{
		CatalogPath:        filepath.Join(dir, "catalog.ducklake"),
		PostgresCatalogURI: "postgres://user:pass@localhost:5432/catalog",
		DataPath:           filepath.Join(dir, "lake-data"),
	}, nil)
	require.Error(t, err)
}

// TestDuckLakeBootstrapStmtsRejectsAmbiguousCatalogSQLite is
// TestDuckLakeBootstrapStmtsRejectsAmbiguousCatalog's SQLiteCatalogPath
// analogue: it must be just as mutually exclusive with the other two
// catalog sources as they already are with each other.
func TestDuckLakeBootstrapStmtsRejectsAmbiguousCatalogSQLite(t *testing.T) {
	dir := t.TempDir()
	_, err := duckLakeBootstrapStmts(DuckLakeOptions{
		CatalogPath:       filepath.Join(dir, "catalog.ducklake"),
		SQLiteCatalogPath: filepath.Join(dir, "catalog.sqlite"),
		DataPath:          filepath.Join(dir, "lake-data"),
	}, nil)
	require.Error(t, err)
}

// TestDuckLakeBootstrapStmtsSQLiteAttachTarget pins the ATTACH target format
// for a SQLite catalog: 'ducklake:sqlite:<SQLiteCatalogPath>', distinct from
// both the file-based 'ducklake:<CatalogPath>' and postgres
// 'ducklake:postgres:<URI>' forms.
func TestDuckLakeBootstrapStmtsSQLiteAttachTarget(t *testing.T) {
	dir := t.TempDir()
	stmts, err := duckLakeBootstrapStmts(DuckLakeOptions{
		SQLiteCatalogPath: filepath.Join(dir, "catalog.sqlite"),
		DataPath:          filepath.Join(dir, "lake-data"),
	}, nil)
	require.NoError(t, err)

	attach := requireAttachStmt(t, stmts)
	require.Contains(t, attach, "'ducklake:sqlite:"+filepath.Join(dir, "catalog.sqlite")+"'")
}

// TestDuckLakeBootstrapStmtsSQLiteLoadsExtension pins that a SQLite catalog
// additionally installs/loads the sqlite extension before the ducklake
// ATTACH, since DuckLake's SQLite catalog backend depends on it (mirrors
// TestDuckLakeBootstrapStmtsPostgresLoadsExtension for postgres).
func TestDuckLakeBootstrapStmtsSQLiteLoadsExtension(t *testing.T) {
	dir := t.TempDir()
	stmts, err := duckLakeBootstrapStmts(DuckLakeOptions{
		SQLiteCatalogPath: filepath.Join(dir, "catalog.sqlite"),
		DataPath:          filepath.Join(dir, "lake-data"),
	}, nil)
	require.NoError(t, err)
	require.Contains(t, stmts, "INSTALL sqlite;")
	require.Contains(t, stmts, "LOAD sqlite;")

	installIdx := indexOfStmt(t, stmts, "INSTALL sqlite;")
	ducklakeIdx := indexOfStmt(t, stmts, "INSTALL ducklake;")
	require.Less(t, installIdx, ducklakeIdx, "sqlite extension must install before ducklake attaches against it")
}

// TestDuckLakeBootstrapStmtsSQLiteDoesNotForceInliningOff guards against
// accidentally copying postgres's ducklake#1175 workaround onto SQLite: a
// SQLite catalog has no equivalent JSON-inlining bug (see
// SQLiteCatalogPath's doc comment), so a caller's explicit
// DataInliningRowLimit must be respected as-is.
func TestDuckLakeBootstrapStmtsSQLiteDoesNotForceInliningOff(t *testing.T) {
	dir := t.TempDir()
	stmts, err := duckLakeBootstrapStmts(DuckLakeOptions{
		SQLiteCatalogPath:    filepath.Join(dir, "catalog.sqlite"),
		DataPath:             filepath.Join(dir, "lake-data"),
		DataInliningRowLimit: 500,
	}, nil)
	require.NoError(t, err)

	attach := requireAttachStmt(t, stmts)
	require.Contains(t, attach, "DATA_INLINING_ROW_LIMIT 500")
}

// TestDuckLakeBootstrapStmtsQuackAttachTarget pins the ATTACH target format
// for a quack catalog: 'ducklake:quack:<QuackCatalogAddr>', distinct from
// the file/postgres/sqlite forms.
func TestDuckLakeBootstrapStmtsQuackAttachTarget(t *testing.T) {
	stmts, err := duckLakeBootstrapStmts(DuckLakeOptions{
		QuackCatalogAddr:  "127.0.0.1:19494",
		QuackCatalogToken: "tok",
		DataPath:          filepath.Join(t.TempDir(), "lake-data"),
	}, nil)
	require.NoError(t, err)

	attach := requireAttachStmt(t, stmts)
	require.Contains(t, attach, "'ducklake:quack:127.0.0.1:19494'")
}

// TestDuckLakeBootstrapStmtsQuackLoadsExtensionAndCreatesSecret pins that a
// quack catalog additionally installs/loads the quack extension and issues
// a CREATE OR REPLACE SECRET (scoped to QuackCatalogAddr, carrying
// QuackCatalogToken) before the ducklake ATTACH, since it needs to
// authenticate as a quack client of that listener before DuckLake can reach
// it (mirrors TestDuckLakeBootstrapStmtsPostgresLoadsExtension/
// TestDuckLakeBootstrapStmtsSQLiteLoadsExtension for the other two remote
// backends).
func TestDuckLakeBootstrapStmtsQuackLoadsExtensionAndCreatesSecret(t *testing.T) {
	stmts, err := duckLakeBootstrapStmts(DuckLakeOptions{
		QuackCatalogAddr:  "127.0.0.1:19494",
		QuackCatalogToken: "tok",
		DataPath:          filepath.Join(t.TempDir(), "lake-data"),
	}, nil)
	require.NoError(t, err)
	require.Contains(t, stmts, "INSTALL quack;")
	require.Contains(t, stmts, "LOAD quack;")

	installIdx := indexOfStmt(t, stmts, "INSTALL quack;")
	ducklakeIdx := indexOfStmt(t, stmts, "INSTALL ducklake;")
	require.Less(t, installIdx, ducklakeIdx, "quack extension must install before ducklake attaches against it")

	secretIdx := -1
	attachIdx := -1
	for i, s := range stmts {
		if strings.Contains(s, "CREATE OR REPLACE SECRET") {
			secretIdx = i
			require.Contains(t, s, "TOKEN 'tok'")
			require.Contains(t, s, "SCOPE 'quack:127.0.0.1:19494'")
		}
		if strings.HasPrefix(s, "ATTACH") {
			attachIdx = i
		}
	}
	require.GreaterOrEqual(t, secretIdx, 0, "expected a CREATE OR REPLACE SECRET statement")
	require.Less(t, secretIdx, attachIdx, "the secret must be created before the ATTACH that needs it")
}

// TestDuckLakeBootstrapStmtsQuackRequiresToken guards the validation that
// QuackCatalogAddr without QuackCatalogToken must fail clearly up front —
// the CREATE SECRET this driver issues would otherwise have nothing to
// authenticate with.
func TestDuckLakeBootstrapStmtsQuackRequiresToken(t *testing.T) {
	_, err := duckLakeBootstrapStmts(DuckLakeOptions{
		QuackCatalogAddr: "127.0.0.1:19494",
		DataPath:         filepath.Join(t.TempDir(), "lake-data"),
	}, nil)
	require.Error(t, err)
}

// TestDuckLakeBootstrapStmtsQuackDoesNotForceInliningOff mirrors
// TestDuckLakeBootstrapStmtsSQLiteDoesNotForceInliningOff: a remote quack
// catalog is an ordinary DuckDB instance, with no restricted type support to
// work around, so a caller's explicit DataInliningRowLimit must be
// respected as-is.
func TestDuckLakeBootstrapStmtsQuackDoesNotForceInliningOff(t *testing.T) {
	stmts, err := duckLakeBootstrapStmts(DuckLakeOptions{
		QuackCatalogAddr:     "127.0.0.1:19494",
		QuackCatalogToken:    "tok",
		DataPath:             filepath.Join(t.TempDir(), "lake-data"),
		DataInliningRowLimit: 500,
	}, nil)
	require.NoError(t, err)

	attach := requireAttachStmt(t, stmts)
	require.Contains(t, attach, "DATA_INLINING_ROW_LIMIT 500")
}

// TestDuckLakeBootstrapStmtsRejectsAmbiguousCatalogQuack is
// TestDuckLakeBootstrapStmtsRejectsAmbiguousCatalogSQLite's QuackCatalogAddr
// analogue: it must be just as mutually exclusive with the other three
// catalog sources as they already are with each other.
func TestDuckLakeBootstrapStmtsRejectsAmbiguousCatalogQuack(t *testing.T) {
	dir := t.TempDir()
	_, err := duckLakeBootstrapStmts(DuckLakeOptions{
		SQLiteCatalogPath: filepath.Join(dir, "catalog.sqlite"),
		QuackCatalogAddr:  "127.0.0.1:19494",
		QuackCatalogToken: "tok",
		DataPath:          filepath.Join(dir, "lake-data"),
	}, nil)
	require.Error(t, err)
}

// TestDuckLakeBootstrapStmtsRejectsNoCatalog covers the misconfigured case
// where neither catalog source is set.
func TestDuckLakeBootstrapStmtsRejectsNoCatalog(t *testing.T) {
	_, err := duckLakeBootstrapStmts(DuckLakeOptions{
		DataPath: filepath.Join(t.TempDir(), "lake-data"),
	}, nil)
	require.Error(t, err)
}

// requireAttachStmt returns the one ATTACH statement in stmts, failing the
// test if there isn't exactly one.
func requireAttachStmt(t *testing.T, stmts []string) string {
	t.Helper()
	var attach string
	found := 0
	for _, s := range stmts {
		if strings.HasPrefix(s, "ATTACH ") {
			attach = s
			found++
		}
	}
	require.Equal(t, 1, found, "expected exactly one ATTACH statement in %v", stmts)
	return attach
}

// indexOfStmt returns the index of the first exact match of want in stmts,
// failing the test if it isn't present.
func indexOfStmt(t *testing.T, stmts []string, want string) int {
	t.Helper()
	for i, s := range stmts {
		if s == want {
			return i
		}
	}
	require.Fail(t, "statement not found", "%q not found in %v", want, stmts)
	return -1
}

// TestDuckLakeOptionsValidation covers the misconfiguration errors: an enabled
// DuckLake with a missing path must fail loudly at startup rather than
// producing a subprocess that looks healthy but has no lake.
func TestDuckLakeOptionsValidation(t *testing.T) {
	binPath := RequireDuckDBBinary(t)

	dir := t.TempDir()

	t.Run("missing catalog path", func(t *testing.T) {
		p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", &DuckLakeOptions{
			DataPath: filepath.Join(dir, "data"),
		}, nil)
		require.Error(t, err)
		require.Nil(t, p)
	})

	t.Run("missing data path", func(t *testing.T) {
		p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", &DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.ducklake"),
		}, nil)
		require.Error(t, err)
		require.Nil(t, p)
	})

	t.Run("undirectoriable data path", func(t *testing.T) {
		// A regular file where the data directory should be: os.MkdirAll must
		// fail, and that failure must surface as a real error rather than
		// being swallowed into a lake-less "healthy" process.
		blocker := filepath.Join(dir, "blocker")
		require.NoError(t, os.WriteFile(blocker, []byte("not a dir"), 0o600))

		p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", &DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog2.ducklake"),
			DataPath:    filepath.Join(blocker, "data"),
		}, nil)
		require.Error(t, err)
		require.Nil(t, p)
	})
}
