package insights_test

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/stretchr/testify/require"
)

// newTestQuackDuckDB is newTestDuckDB over the quack transport with a single
// connection, the transport devserver's setupDualWrite always uses: a
// cancelled statement there is interrupted server-side rather than
// respawning the subprocess (jsonlines' only recourse). One connection makes
// a follow-up query queue behind anything still running on it.
func newTestQuackDuckDB(t *testing.T) *sql.DB {
	t.Helper()
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	dir := t.TempDir()
	addr := driver.EphemeralQuackAddr
	db, err := driver.Open(t.Context(), driver.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &driver.DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.duckdb"),
			DataPath:    filepath.Join(dir, "data"),
		},
		QuackAddr:              &addr,
		RestrictExternalAccess: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, duckdb.Migrate(t.Context(), db, true))
	return db
}

// TestExecuteTimesOutAndInterrupts proves Execute's deadline stops an
// expensive query in DuckDB itself: the follow-up query shares the only
// connection, so it could not finish promptly if the first were still
// running.
func TestExecuteTimesOutAndInterrupts(t *testing.T) {
	db := newTestQuackDuckDB(t)
	insights.SetExecuteTimeout(t, 500*time.Millisecond)
	// No logical-table query is expensive against an empty store, so run a
	// raw cross join; the guard would (rightly) reject its range().
	insights.DisableRenderedSQLGuard(t)

	start := time.Now()
	_, err := insights.Execute(context.Background(), db, &insights.TranspileResult{
		SQL: "SELECT count(*) FROM range(1000000000) a, range(1000000) b WHERE (a.range + b.range) % 7 = 3;",
	})
	var eerr *insights.ExecutionError
	require.ErrorAs(t, err, &eerr)
	require.ErrorContains(t, err, "time limit")
	require.Less(t, time.Since(start), 5*time.Second)

	start = time.Now()
	var one int
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT 1;").Scan(&one))
	require.Less(t, time.Since(start), 5*time.Second)
}

// TestExecuteRejectsOversizedResult: LIMIT bounds rows, not bytes, so one
// huge cell would otherwise be buffered whole. This goes through Transpile
// and the default limits, end to end.
func TestExecuteRejectsOversizedResult(t *testing.T) {
	db := newTestQuackDuckDB(t)

	tr, err := insights.Transpile(fmt.Sprintf("SELECT repeat('a', %d) AS big", insights.MaxResultBytes()+1), uuid.New(), uuid.New())
	require.NoError(t, err)

	_, err = insights.Execute(context.Background(), db, tr)
	var eerr *insights.ExecutionError
	require.ErrorAs(t, err, &eerr)
	require.ErrorIs(t, err, driver.ErrResultTooLarge)
	require.ErrorContains(t, err, "MiB limit")

	// A result under the limit still runs.
	tr, err = insights.Transpile("SELECT repeat('a', 1000) AS small", uuid.New(), uuid.New())
	require.NoError(t, err)
	res, err := insights.Execute(context.Background(), db, tr)
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
}

// TestExecuteCapsConcurrency: with every slot taken, a query waits only
// until its own deadline, then fails without touching the database.
func TestExecuteCapsConcurrency(t *testing.T) {
	db := newTestQuackDuckDB(t)
	insights.SetExecuteTimeout(t, 200*time.Millisecond)
	insights.OccupyExecuteSlots(t)

	tr, err := insights.Transpile("SELECT 1 AS one", uuid.New(), uuid.New())
	require.NoError(t, err)

	_, err = insights.Execute(context.Background(), db, tr)
	var eerr *insights.ExecutionError
	require.ErrorAs(t, err, &eerr)
	require.ErrorContains(t, err, "too many Insights queries")
}
