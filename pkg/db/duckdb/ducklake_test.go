package duckdb

import (
	"os"
	"path/filepath"
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
	binPath := requireDuckDBBinary(t)

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
	binPath := requireDuckDBBinary(t)

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
	binPath := requireDuckDBBinary(t)

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
	binPath := requireDuckDBBinary(t)

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
	binPath := requireDuckDBBinary(t)

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
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE inngest.nope (id INTEGER);")
	require.Error(t, err, "no DuckLake catalog should be attached when DuckLake is not configured")
}

// TestDuckLakeOptionsValidation covers the misconfiguration errors: an enabled
// DuckLake with a missing path must fail loudly at startup rather than
// producing a subprocess that looks healthy but has no lake.
func TestDuckLakeOptionsValidation(t *testing.T) {
	binPath := requireDuckDBBinary(t)

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
