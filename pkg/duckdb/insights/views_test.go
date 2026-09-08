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
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// newTestDuckDB duplicates pkg/cqrs/duckdbquery/testutil_test.go's
// unexported helper of the same name/shape verbatim -- that helper can't
// be imported from this package (package-private, _test.go) -- see
// docs/plans/012-duckdb-insights-query-layer-plan.md's Global Constraints.
// Returns a cleanup func rather than using t.Cleanup internally, matching
// the original exactly.
func newTestDuckDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	dir := t.TempDir()
	db, err := duckdb.Open(t.Context(), duckdb.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &duckdb.DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.duckdb"),
			DataPath:    filepath.Join(dir, "data"),
		},
	})
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	if err := duckdb.Migrate(t.Context(), db, true); err != nil {
		t.Fatalf("migrating duckdb: %v", err)
	}
	return db, func() { _ = db.Close() }
}

// insertRunRow inserts a minimal inngest.runs row satisfying every NOT
// NULL column. Only queued_at is set (not started_at/ended_at), so
// inngest.insights_runs' COALESCE(ended_at, started_at, queued_at)
// collapse ordering is driven entirely by queuedAt.
//
// Every query string in this file ends with ";" -- this driver's
// subprocess (pkg/db/duckdb, "-jsonlines" mode) hangs indefinitely waiting
// for more input if a statement isn't terminated, rather than erroring;
// confirmed against the real binary. Every existing caller in this
// codebase already follows this convention.
func insertRunRow(t *testing.T, db *sql.DB, accountID, envID uuid.UUID, runID, status string, queuedAt time.Time) {
	t.Helper()
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, app_id, app_name, function_id, function_slug, status, attributes, inputs)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		accountID.String(), envID.String(), runID, queuedAt, uuid.New().String(), "test-app", uuid.New().String(), "test-fn", status, "{}", "{}",
	)
	require.NoError(t, err)
}

// insertStepSpan inserts one inngest.run_trace_spans row shaped like a
// step attempt (name executor.step, attributes carrying the "_inngest."
// prefixed step.userland.id/step.attempt keys insights_step_attempts reads
// its own step_id/step_attempt columns from -- see the macro in
// pkg/db/duckdb/migrations/000004_insights_views.sql and
// pkg/tracing/meta's AttrKeyPrefix/StepUserlandID), satisfying every NOT
// NULL column on that table.
func insertStepSpan(t *testing.T, db *sql.DB, accountID, envID uuid.UUID, runID, spanID, stepID string, attempt int, startTime time.Time) {
	t.Helper()
	appID, functionID := uuid.New(), uuid.New()
	attrs := fmt.Sprintf(`{"_inngest.step.userland.id": %q, "_inngest.step.attempt": %d}`, stepID, attempt)
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO inngest.run_trace_spans
		   (account_id, env_id, run_id, run_queued_at, app_id, app_name, function_id, function_slug, name, start_time, end_time, trace_id, span_id, attributes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'executor.step', ?, ?, ?, ?, ?);`,
		accountID.String(), envID.String(), runID, startTime, appID.String(), "test-app", functionID.String(), "test-fn",
		startTime, startTime.Add(time.Second), uuid.New().String(), spanID, attrs,
	)
	require.NoError(t, err)
}

// insertMetadataRow inserts one inngest.run_metadata emission row.
func insertMetadataRow(t *testing.T, db *sql.DB, accountID, envID uuid.UUID, runID, spanID, kind string, isUser bool, values string, createdAt time.Time) {
	t.Helper()
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO inngest.run_metadata
		   (account_id, env_id, run_id, run_queued_at, span_id, scope, kind, is_user, values, created_at)
		 VALUES (?, ?, ?, ?, ?, 'run', ?, ?, ?, ?);`,
		accountID.String(), envID.String(), runID, createdAt, spanID, kind, isUser, values, createdAt,
	)
	require.NoError(t, err)
}

func TestInsightsRunsMacroCollapsesToLatestRow(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := context.Background()
	accountID, envID, runID := uuid.New(), uuid.New(), "01HXXX"

	insertRunRow(t, db, accountID, envID, runID, "queued", time.Now().Add(-time.Minute))
	insertRunRow(t, db, accountID, envID, runID, "Completed", time.Now())

	var status string
	var count int
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT COUNT(*), status FROM inngest.insights_runs(?, ?) WHERE run_id = ? GROUP BY status;", accountID.String(), envID.String(), runID,
	).Scan(&count, &status))
	require.Equal(t, 1, count)
	require.Equal(t, "Completed", status)
}

func TestInsightsRunsMacroExcludesAccountEnvColumns(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID, envID := uuid.New(), uuid.New()
	insertRunRow(t, db, accountID, envID, "run-1", "Completed", time.Now())

	// This driver's jsonlines transport derives column names from actual
	// returned row data, not query metadata -- a result with zero rows
	// (e.g. LIMIT 0) reports no columns at all (confirmed against the real
	// binary; see Execute's DESCRIBE-based workaround, Task 10). Query a
	// real row instead of LIMIT 0 here.
	rows, err := db.QueryContext(t.Context(), "SELECT * FROM inngest.insights_runs(?, ?);", accountID.String(), envID.String())
	require.NoError(t, err)
	defer rows.Close()
	cols, err := rows.Columns()
	require.NoError(t, err)
	require.NotContains(t, cols, "account_id")
	require.NotContains(t, cols, "env_id")
	require.Contains(t, cols, "metadata")
	require.Contains(t, cols, "inngest")
}

func TestInsightsRunsMacroIsolatesByEnv(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID := uuid.New()
	envA, envB := uuid.New(), uuid.New()
	insertRunRow(t, db, accountID, envA, "run-a", "Completed", time.Now())
	insertRunRow(t, db, accountID, envB, "run-b", "Completed", time.Now())

	var runIDs []string
	rows, err := db.QueryContext(t.Context(), "SELECT run_id FROM inngest.insights_runs(?, ?);", accountID.String(), envA.String())
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		runIDs = append(runIDs, id)
	}
	require.Equal(t, []string{"run-a"}, runIDs)
}

func TestInsightsRunsMacroMergesRunScopedMetadata(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID, envID := uuid.New(), uuid.New()
	runID := "run-with-metadata"
	insertRunRow(t, db, accountID, envID, runID, "Completed", time.Now())
	insertMetadataRow(t, db, accountID, envID, runID, "run-span", "k1", true, `{"x": 1}`, time.Now())
	insertMetadataRow(t, db, accountID, envID, runID, "run-span", "k2", true, `{"y": 2}`, time.Now().Add(time.Second))
	insertMetadataRow(t, db, accountID, envID, runID, "run-span", "sys", false, `{"z": 3}`, time.Now())

	// This driver decodes a JSON column into its native Go value
	// (map[string]any here), not a string -- matching Execute's design
	// (Task 10), which relies on exactly this for pkg/gql_scalars.Unknown.
	var metadata, internal any
	require.NoError(t, db.QueryRowContext(t.Context(),
		"SELECT metadata, inngest FROM inngest.insights_runs(?, ?) WHERE run_id = ?;", accountID.String(), envID.String(), runID,
	).Scan(&metadata, &internal))
	require.Equal(t, map[string]any{"k1": map[string]any{"x": float64(1)}, "k2": map[string]any{"y": float64(2)}}, metadata)
	require.Equal(t, map[string]any{"sys": map[string]any{"z": float64(3)}}, internal)
}

func TestInsightsStepAttemptsFiltersToStepSpans(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := context.Background()
	accountID, envID := uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), nil).String()
	insertStepSpan(t, db, accountID, envID, runID, "span-1", "step-1", 0, time.Now())

	var total int
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM inngest.insights_step_attempts(?, ?) WHERE run_id = ?;", accountID.String(), envID.String(), runID,
	).Scan(&total))
	require.Greater(t, total, 0)

	var stepID string
	var attempt int
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT step_id, step_attempt FROM inngest.insights_step_attempts(?, ?) WHERE run_id = ?;", accountID.String(), envID.String(), runID,
	).Scan(&stepID, &attempt))
	require.Equal(t, "step-1", stepID)
	require.Equal(t, 0, attempt)
}

func TestInsightsStepsCollapsesToLatestAttempt(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID, envID := uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), nil).String()
	insertStepSpan(t, db, accountID, envID, runID, "span-1", "step-1", 0, time.Now().Add(-time.Minute))
	insertStepSpan(t, db, accountID, envID, runID, "span-2", "step-1", 1, time.Now())

	var count int
	var attempt int
	require.NoError(t, db.QueryRowContext(t.Context(),
		"SELECT COUNT(*), MAX(step_attempt) FROM inngest.insights_steps(?, ?) WHERE run_id = ?;", accountID.String(), envID.String(), runID,
	).Scan(&count, &attempt))
	require.Equal(t, 1, count)
	require.Equal(t, 1, attempt)

	require.NoError(t, db.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM inngest.insights_step_attempts(?, ?) WHERE run_id = ?;", accountID.String(), envID.String(), runID,
	).Scan(&count))
	require.Equal(t, 2, count)
}

func TestInsightsExtendedTraceSpansFiltersToSDKSpans(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID, envID := uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), nil).String()

	appID, functionID := uuid.New(), uuid.New()
	insert := func(name string) {
		_, err := db.ExecContext(t.Context(),
			`INSERT INTO inngest.run_trace_spans
			   (account_id, env_id, run_id, run_queued_at, app_id, app_name, function_id, function_slug, name, start_time, end_time, trace_id, span_id, attributes)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '{}');`,
			accountID.String(), envID.String(), runID, time.Now(), appID.String(), "test-app", functionID.String(), "test-fn",
			name, time.Now(), time.Now(), uuid.New().String(), uuid.New().String(),
		)
		require.NoError(t, err)
	}
	insert("sdk.extended_trace")
	insert("executor.step")
	insert("executor.run.started")

	var count int
	require.NoError(t, db.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM inngest.insights_extended_trace_spans(?, ?) WHERE run_id = ?;", accountID.String(), envID.String(), runID,
	).Scan(&count))
	require.Equal(t, 1, count)
}
