package insights_test

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/icebergcatalog/icebergtest"

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
	addr := driver.EphemeralQuackAddr
	db, err := driver.Open(t.Context(), driver.Options{
		BinaryPath:             binPath,
		DBFile:                 ":memory:",
		Iceberg:                icebergtest.Start(t),
		QuackAddr:              &addr,
		RestrictExternalAccess: true,
		ResourceLimits:         driver.DefaultResourceLimits(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, duckdb.Migrate(t.Context(), db, duckdb.ModeRawLake))
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
// huge cell would otherwise be buffered whole. The oversized query skips
// Transpile, whose own size guard (sizelimit.go) would reject its repeat
// before Execute's limit ever came into play; the small one goes through
// Transpile and the default limits, end to end.
func TestExecuteRejectsOversizedResult(t *testing.T) {
	db := newTestQuackDuckDB(t)

	_, err := insights.Execute(context.Background(), db, &insights.TranspileResult{
		SQL: fmt.Sprintf("SELECT repeat('a', %d) AS big;", insights.MaxResultBytes()+1),
	})
	var eerr *insights.ExecutionError
	require.ErrorAs(t, err, &eerr)
	require.ErrorIs(t, err, driver.ErrResultTooLarge)
	require.ErrorContains(t, err, "MiB limit")

	// A result under the limit still runs.
	tr, err := insights.Transpile("SELECT repeat('a', 1000) AS small", uuid.New(), uuid.New())
	require.NoError(t, err)
	res, err := insights.Execute(context.Background(), db, tr)
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
}

// TestExecuteCallerCancellationIsNotATimeLimit: a caller that gives up
// first gets its own ctx error back, not Execute's "time limit" message
// (TestExecuteTimesOutAndInterrupts covers Execute's own deadline).
func TestExecuteCallerCancellationIsNotATimeLimit(t *testing.T) {
	db := newTestQuackDuckDB(t)

	tr, err := insights.Transpile("SELECT 1 AS one", uuid.New(), uuid.New())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = insights.Execute(ctx, db, tr)
	require.ErrorIs(t, err, context.Canceled)
	require.NotContains(t, err.Error(), "time limit")
}

// TestExecuteTooManyAbandonedQueries: once
// driver.Options.MaxAbandonedStatements timed-out queries are still running
// server-side, Execute over an OpenShared handle fails fast w/ a user-facing
// message instead of piling up more. The constant-folded levenshtein ignores
// its interrupt (see driver.TestQuackAbandonedStatementsAreCapped).
func TestExecuteTooManyAbandonedQueries(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	addr := driver.EphemeralQuackAddr
	connector, primary, err := driver.OpenConnector(t.Context(), driver.Options{
		BinaryPath:             binPath,
		DBFile:                 ":memory:",
		QuackAddr:              &addr,
		MaxAbandonedStatements: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = primary.Close() })
	db := connector.OpenShared(4)
	t.Cleanup(func() { _ = db.Close() })

	insights.SetExecuteTimeout(t, 200*time.Millisecond)
	insights.DisableRenderedSQLGuard(t)

	_, err = insights.Execute(context.Background(), db, &insights.TranspileResult{
		SQL: "SELECT levenshtein(repeat('a', 40000), repeat('b', 40000)) AS d;",
	})
	require.ErrorContains(t, err, "time limit")

	start := time.Now()
	_, err = insights.Execute(context.Background(), db, &insights.TranspileResult{SQL: "SELECT 1 AS one;"})
	var eerr *insights.ExecutionError
	require.ErrorAs(t, err, &eerr)
	require.ErrorIs(t, err, driver.ErrTooManyAbandoned)
	require.ErrorContains(t, err, "too many cancelled queries are still finishing; try again shortly")
	require.Less(t, time.Since(start), time.Second)

	// The primary handle (dual-write's) isn't gated.
	var one int
	require.NoError(t, primary.QueryRowContext(context.Background(), "SELECT 1;").Scan(&one))

	require.Eventually(t, func() bool {
		_, err := insights.Execute(context.Background(), db, &insights.TranspileResult{SQL: "SELECT 1 AS one;"})
		return err == nil
	}, 30*time.Second, 100*time.Millisecond)
}
