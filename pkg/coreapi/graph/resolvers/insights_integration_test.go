package resolvers

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs/duckdbquery"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// newTestDuckDB duplicates pkg/cqrs/duckdbquery/testutil_test.go's
// unexported helper of the same name/shape verbatim (see
// docs/plans/012-duckdb-insights-query-layer-plan.md's Global Constraints)
// -- identical to pkg/duckdb/insights/views_test.go's copy (Task 9),
// redefined here because it's a different package.
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

// insertRunRow duplicates pkg/duckdb/insights/views_test.go's helper of
// the same name/shape (Task 9) -- redefined here because it's a different
// package.
func insertRunRow(t *testing.T, db *sql.DB, accountID, envID uuid.UUID, runID, status string, queuedAt time.Time) {
	t.Helper()
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO inngest.runs (account_id, env_id, run_id, queued_at, app_id, app_name, function_id, function_slug, status, attributes, inputs)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		accountID.String(), envID.String(), runID, queuedAt, uuid.New().String(), "test-app", uuid.New().String(), "test-fn", status, "{}", "{}",
	)
	require.NoError(t, err)
}

// insertStepSpan duplicates pkg/duckdb/insights/views_test.go's helper of
// the same name/shape (Task 9), extended with a parentSpanID so a caller
// can build a real parent/child tree for GetSpansByRunID's root-selection
// logic to assemble -- redefined here because it's a different package.
func insertStepSpan(t *testing.T, db *sql.DB, accountID, envID uuid.UUID, runID, parentSpanID, spanID, stepID string, attempt int, startTime time.Time) {
	t.Helper()
	appID, functionID := uuid.New(), uuid.New()
	// Both keys carry stepID: insights_step_attempts (pkg/db/duckdb/
	// migrations/000004_insights_views.sql) reads its step_id column from
	// "_inngest.step.userland.id", while cqrs.ApplyExtractedSpanAttributes
	// (via GetSpansByRunID, checked later in this file) populates
	// ExtractedValues.StepID from the separate, generic "_inngest.step.id" --
	// this test's own assertions check both, so the fixture must set both.
	attrs := fmt.Sprintf(`{"_inngest.step.id": %q, "_inngest.step.userland.id": %q, "_inngest.step.attempt": %d}`, stepID, stepID, attempt)
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO inngest.run_trace_spans
		   (account_id, env_id, run_id, run_queued_at, app_id, app_name, function_id, function_slug, name, start_time, end_time, trace_id, span_id, parent_span_id, attributes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'executor.step', ?, ?, ?, ?, ?, ?);`,
		accountID.String(), envID.String(), runID, startTime, appID.String(), "test-app", functionID.String(), "test-fn",
		startTime, startTime.Add(time.Second), uuid.New().String(), spanID, parentSpanID, attrs,
	)
	require.NoError(t, err)
}

// insertRootSpan inserts a genuinely rootless (no parent_span_id) span,
// mirroring the real "executor.run.started" root span 007/008's flat-span
// reader (pkg/cqrs/duckdbquery.GetSpansByRunID) always roots a run's tree
// at.
func insertRootSpan(t *testing.T, db *sql.DB, accountID, envID uuid.UUID, runID, spanID string, startTime time.Time) {
	t.Helper()
	appID, functionID := uuid.New(), uuid.New()
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO inngest.run_trace_spans
		   (account_id, env_id, run_id, run_queued_at, app_id, app_name, function_id, function_slug, name, start_time, end_time, trace_id, span_id, attributes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'executor.run.started', ?, ?, ?, ?, '{}');`,
		accountID.String(), envID.String(), runID, startTime, appID.String(), "test-app", functionID.String(), "test-fn",
		startTime, startTime.Add(time.Second), uuid.New().String(), spanID,
	)
	require.NoError(t, err)
}

func TestInsightsEnvIsolation(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	// consts.DevServerAccountID/EnvID is the only account/env this
	// resolver ever scopes to today (see insights.go's doc comment) --
	// one row goes there, the other under an unrelated env, so this test
	// actually exercises isolation rather than both rows landing outside
	// the scoped account/env.
	insertRunRow(t, db, consts.DevServerAccountID, consts.DevServerEnvID, "run-in-scope", "Completed", time.Now())
	insertRunRow(t, db, consts.DevServerAccountID, uuid.New(), "run-other-env", "Completed", time.Now())

	r := &Resolver{DuckDB: db}
	qr := r.Query().(*queryResolver)

	result, err := qr.Insights(context.Background(), "SELECT run_id FROM runs")
	require.NoError(t, err)

	var runIDs []string
	for _, row := range result.Rows {
		runIDs = append(runIDs, row[0].(string))
	}
	require.Contains(t, runIDs, "run-in-scope")
	require.NotContains(t, runIDs, "run-other-env")
}

// TestInsightsValidationRejection covers a validation-rejected query --
// unknown table/column, a disallowed function -- which resolves as a
// normal (empty) InsightsQueryResult carrying a fatal (ERROR-severity)
// diagnostic instead of a bare top-level GQL error, so the UI can point
// at exactly where the query is wrong. See Insights' own doc comment.
func TestInsightsValidationRejection(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	r := &Resolver{DuckDB: db}
	qr := r.Query().(*queryResolver)

	cases := []string{
		"SELECT * FROM nonexistent_table",
		"SELECT nonexistent_column FROM runs",
		"SELECT read_csv('/etc/passwd') FROM runs",
	}
	for _, sql := range cases {
		t.Run(sql, func(t *testing.T) {
			result, err := qr.Insights(context.Background(), sql)
			require.NoError(t, err)
			require.Empty(t, result.Columns)
			require.Empty(t, result.Rows)
			require.Len(t, result.Diagnostics, 1)
			require.Equal(t, models.InsightsDiagnosticSeverityError, result.Diagnostics[0].Severity)
			require.Equal(t, "validation-error", result.Diagnostics[0].Code)
			require.NotEmpty(t, result.Diagnostics[0].Message)
		})
	}
}

// TestInsightsParseErrorStillReturnsGQLError covers malformed SQL the
// parser itself rejects (not a *insights.ValidationError, so it doesn't
// get the fatal-diagnostic treatment above) -- this still has to fail the
// GQL request outright, since there's no query structure to attach a
// position-aware diagnostic to in the first place.
func TestInsightsParseErrorStillReturnsGQLError(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	r := &Resolver{DuckDB: db}
	qr := r.Query().(*queryResolver)

	_, err := qr.Insights(context.Background(), "SELECT run_id FROM runs; SELECT run_id FROM runs;")
	require.Error(t, err)
}

func TestInsightsStepsMatchesFlatSpanReader(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	runID := ulid.MustNew(ulid.Now(), nil).String()
	insertRootSpan(t, db, consts.DevServerAccountID, consts.DevServerEnvID, runID, "root-span", time.Now().Add(-2*time.Minute))
	insertStepSpan(t, db, consts.DevServerAccountID, consts.DevServerEnvID, runID, "root-span", "span-1", "step-1", 0, time.Now().Add(-time.Minute))
	insertStepSpan(t, db, consts.DevServerAccountID, consts.DevServerEnvID, runID, "root-span", "span-2", "step-1", 1, time.Now())

	r := &Resolver{DuckDB: db}
	qr := r.Query().(*queryResolver)
	result, err := qr.Insights(context.Background(),
		fmt.Sprintf("SELECT step_id, step_attempt FROM steps WHERE run_id = '%s'", runID))
	require.NoError(t, err)
	require.Len(t, result.Rows, 1, "insights_steps must collapse to the latest attempt")
	require.Equal(t, "step-1", result.Rows[0][0])

	// duckdbquery.Wrap's underlying cqrs.Manager is never consulted by
	// GetSpansByRunID (see pkg/cqrs/duckdbquery/spans.go — it only reads
	// m.db), so nil is safe here; this test only needs the DuckDB-backed
	// flat-span reader 007/008 already shipped.
	mgr := duckdbquery.Wrap(nil, db)
	parsedRunID, err := ulid.Parse(runID)
	require.NoError(t, err)
	root, err := mgr.GetSpansByRunID(context.Background(), parsedRunID)
	require.NoError(t, err)

	// root is the run's own root span (executor.run.started), with both
	// step attempts as children — GetSpansByRunID builds the full tree
	// without collapsing attempts, unlike insights_steps. Every span's
	// Attributes (*meta.ExtractedValues, pkg/tracing/meta/
	// extracted_values_gen.go:60's StepID *string, :.. StepAttempt *int)
	// is populated by cqrs.ApplyExtractedSpanAttributes, called from
	// scanSpan for every row (pkg/cqrs/duckdbquery/spans.go).
	var attempts []int
	for _, child := range root.Children {
		if child.Name != "executor.step" {
			continue
		}
		require.NotNil(t, child.Attributes)
		require.NotNil(t, child.Attributes.StepID)
		require.Equal(t, "step-1", *child.Attributes.StepID)
		require.NotNil(t, child.Attributes.StepAttempt)
		attempts = append(attempts, *child.Attributes.StepAttempt)
	}
	require.ElementsMatch(t, []int{0, 1}, attempts, "GetSpansByRunID must see both attempts insights_step_attempts reported")
}

// TestInsightsShowTablesAndDescribeShortCircuit proves SHOW TABLES/DESCRIBE
// resolve through Query.insights without ever touching DuckDB -- see
// pkg/duckdb/insights/describe.go. Still requires qr.DuckDB non-nil,
// same dual-write gate every other query goes through, even though this
// path never uses the connection itself.
func TestInsightsShowTablesAndDescribeShortCircuit(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	r := &Resolver{DuckDB: db}
	qr := r.Query().(*queryResolver)

	result, err := qr.Insights(context.Background(), "SHOW TABLES")
	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	require.Len(t, result.Columns, 2)
	require.Equal(t, "name", result.Columns[0].Name)
	require.Equal(t, "description", result.Columns[1].Name)
	require.Len(t, result.Rows, 6)
	require.ElementsMatch(t, result.Info.Tables, []string{
		"runs", "events", "metadata", "extended_trace_spans", "steps", "step_attempts",
	})

	result, err = qr.Insights(context.Background(), "DESCRIBE runs")
	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	require.Equal(t, []string{"runs"}, result.Info.Tables)
	require.Equal(t, "run_id", result.Rows[0][0])
	require.Equal(t, "STRING", result.Rows[0][1])
	require.NotEmpty(t, result.Rows[0][2])

	result, err = qr.Insights(context.Background(), "DESCRIBE nonexistent_table")
	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	require.Contains(t, result.Diagnostics[0].Message, "unknown table")
}

func TestInsightsErrorsCleanlyWhenDualWriteOff(t *testing.T) {
	r := &Resolver{DuckDB: nil}
	qr := r.Query().(*queryResolver)

	result, err := qr.Insights(context.Background(), "SELECT run_id FROM runs")
	require.Nil(t, result)
	require.Error(t, err)
}
