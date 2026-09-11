package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/stretchr/testify/require"
)

func TestRunSeedsAFreshTargetWhenSourceAndTargetAreTheSameEmptyDir(t *testing.T) {
	requireDuckDBOnPath(t)
	requireQuackExtension(t)
	dir := t.TempDir()

	summary, err := run(t.Context(), runConfig{
		SourceDir:   dir,
		TargetDir:   dir,
		RunCount:    4,
		Window:      time.Hour,
		SampleLimit: 200,
		Seed:        1,
		Now:         time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Equal(t, 4, summary.Runs)
	require.Positive(t, summary.Spans)
	require.Positive(t, summary.Events)

	db := openTestDB(t, dir)
	var runCount int
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) FROM "+duckdb.DuckLakeAlias+".runs;").Scan(&runCount))
	require.Equal(t, 4, runCount)
}

func TestRunSamplesFromSourceWithoutModifyingIt(t *testing.T) {
	requireDuckDBOnPath(t)
	requireQuackExtension(t)
	sourceDir := filepath.Join(t.TempDir(), "source")
	targetDir := filepath.Join(t.TempDir(), "target")

	// Seeded via its own connector, explicitly closed before run() opens
	// sourceDir itself — unlike the earlier cgo/embedded-driver version of
	// this tool, two real subprocesses can't both hold the same DuckLake
	// catalog open at once.
	seedConnector, seedDB, err := openDuckDB(t.Context(), sourceDir, 1)
	require.NoError(t, err)
	_, err = seedDB.ExecContext(t.Context(),
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, app_id, function_id, status, inputs)
		 VALUES ('11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 'run-1', now(), '33333333-3333-3333-3333-333333333333', '44444444-4444-4444-4444-444444444444', 'Completed', '{}');`,
	)
	require.NoError(t, err)
	require.NoError(t, seedDB.Close())
	require.NoError(t, seedConnector.Close())

	summary, err := run(t.Context(), runConfig{
		SourceDir:   sourceDir,
		TargetDir:   targetDir,
		RunCount:    2,
		Window:      time.Hour,
		SampleLimit: 200,
		Seed:        1,
		Now:         time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Equal(t, 2, summary.Runs)

	sourceDB := openTestDB(t, sourceDir)
	var sourceRunCount int
	require.NoError(t, sourceDB.QueryRowContext(t.Context(), "SELECT count(*) FROM "+duckdb.DuckLakeAlias+".runs;").Scan(&sourceRunCount))
	require.Equal(t, 1, sourceRunCount, "seeding the target must not add rows to the source")

	targetDB := openTestDB(t, targetDir)
	var targetRunCount int
	require.NoError(t, targetDB.QueryRowContext(t.Context(), "SELECT count(*) FROM "+duckdb.DuckLakeAlias+".runs;").Scan(&targetRunCount))
	require.Equal(t, 2, targetRunCount)
}

func TestProgressLoggerLogsEveryBatchWithoutThrottling(t *testing.T) {
	var lines []string
	logf := func(format string, args ...any) {
		lines = append(lines, fmt.Sprintf(format, args...))
	}

	onProgress := progressLogger(logf)
	// Mirrors the (batchIndex, totalBatches, done, total) sequence
	// InsertGeneratedRuns emits for 5 runs at batch size 2: two full
	// batches then one partial batch.
	onProgress(1, 3, 2, 5)
	onProgress(2, 3, 4, 5)
	onProgress(3, 3, 5, 5)

	require.Len(t, lines, 3, "every batch flush must produce its own log line, not a throttled subset")
	require.Contains(t, lines[0], "1/3")
	require.Contains(t, lines[0], "2/5")
	require.Contains(t, lines[1], "2/3")
	require.Contains(t, lines[1], "4/5")
	require.Contains(t, lines[2], "3/3")
	require.Contains(t, lines[2], "5/5")
}

func TestRunLogsSamplingGenerationAndInsertionProgress(t *testing.T) {
	requireDuckDBOnPath(t)
	requireQuackExtension(t)
	sourceDir := filepath.Join(t.TempDir(), "source")
	targetDir := filepath.Join(t.TempDir(), "target")

	var mu sync.Mutex
	var lines []string
	logf := func(format string, args ...any) {
		mu.Lock()
		defer mu.Unlock()
		lines = append(lines, fmt.Sprintf(format, args...))
	}

	_, err := run(t.Context(), runConfig{
		SourceDir:   sourceDir,
		TargetDir:   targetDir,
		RunCount:    3,
		Window:      time.Hour,
		SampleLimit: 200,
		Seed:        1,
		Now:         time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
		Log:         logf,
	})
	require.NoError(t, err)

	joined := strings.Join(lines, "\n")
	require.Contains(t, joined, "default", "should note it fell back to default templates for an empty source")
	require.Contains(t, joined, "generating and inserting 3 run")
	require.Contains(t, joined, "3/3", "should report insertion progress reaching the total")
}

// TestRunDryRunReportsCountsWithoutTouchingTarget proves dry run computes
// exactly what a real run would write (matching counts) while never opening
// or creating anything at TargetDir — pointed at a path where MkdirAll
// cannot possibly succeed, which would fail loudly if dry run tried to open
// it anyway.
func TestRunDryRunReportsCountsWithoutTouchingTarget(t *testing.T) {
	requireDuckDBOnPath(t)
	requireQuackExtension(t)
	sourceDir := t.TempDir()

	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("not a dir"), 0o600))
	unusableTargetDir := filepath.Join(blocker, "target")

	summary, err := run(t.Context(), runConfig{
		SourceDir:   sourceDir,
		TargetDir:   unusableTargetDir,
		RunCount:    4,
		Window:      time.Hour,
		SampleLimit: 200,
		Seed:        1,
		Now:         time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
		DryRun:      true,
	})
	require.NoError(t, err, "dry run must not attempt to open or create the target directory — "+
		"a real run would fail here since blocker is a file, not a directory, and MkdirAll(unusableTargetDir) cannot succeed")
	require.Equal(t, 4, summary.Runs)
	require.Positive(t, summary.Spans)
	require.Positive(t, summary.Events)
}

// TestRunWithParallelismWritesAllRowsExactlyOnce is a correctness check for
// the concurrent insert path: many Appender workers race to write batches
// into the same target, and every row must land exactly once.
func TestRunWithParallelismWritesAllRowsExactlyOnce(t *testing.T) {
	requireDuckDBOnPath(t)
	requireQuackExtension(t)
	dir := t.TempDir()

	summary, err := run(t.Context(), runConfig{
		SourceDir:   dir,
		TargetDir:   dir,
		RunCount:    40,
		Window:      time.Hour,
		SampleLimit: 200,
		Seed:        1,
		BatchSize:   5,
		Parallelism: 4,
		Now:         time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Equal(t, 40, summary.Runs)

	db := openTestDB(t, dir)
	var runCount int
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) FROM "+duckdb.DuckLakeAlias+".runs;").Scan(&runCount))
	require.Equal(t, 40, runCount)

	var distinctRunIDs int
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(DISTINCT run_id) FROM "+duckdb.DuckLakeAlias+".runs;").Scan(&distinctRunIDs))
	require.Equal(t, 40, distinctRunIDs, "no run should have been written more than once")
}

func openTestDB(t *testing.T, dir string) *sql.DB {
	t.Helper()
	connector, db, err := openDuckDB(t.Context(), dir, 1)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
		_ = connector.Close()
	})
	return db
}
