package insights_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

var testEnvIDForExecuteTest = uuid.MustParse("00000000-0000-4000-b000-000000000001")

// seedOneCompletedRun inserts one inngest.runs row with a known run_id/
// status, reusing insertRunRow (defined in views_test.go, same package).
func seedOneCompletedRun(t *testing.T, db *sql.DB) (accountID, envID uuid.UUID, runID string) {
	t.Helper()
	accountID, envID = uuid.New(), testEnvIDForExecuteTest
	runID = ulid.MustNew(ulid.Now(), nil).String()
	insertRunRow(t, db, accountID, envID, runID, "Completed", time.Now())
	return accountID, envID, runID
}

func TestExecuteReturnsTypedColumnsAndRows(t *testing.T) {
	db, cleanup := newTestDuckDB(t) // defined in views_test.go, Task 9
	defer cleanup()
	ctx := context.Background()
	accountID, envID, runID := seedOneCompletedRun(t, db)

	tr, err := insights.Transpile("SELECT run_id, status FROM runs WHERE run_id = '"+runID+"'", accountID, envID)
	require.NoError(t, err)

	result, err := insights.Execute(ctx, db, tr)
	require.NoError(t, err)

	require.Len(t, result.Columns, 2)
	require.Equal(t, "run_id", result.Columns[0].Name)
	require.Equal(t, insights.ColumnTypeString, result.Columns[0].Type)
	require.Equal(t, insights.HintRunID, result.Columns[0].Hint)
	require.Equal(t, insights.HintNone, result.Columns[1].Hint)

	require.Len(t, result.Rows, 1)
	require.Equal(t, runID, result.Rows[0][0])
	require.Equal(t, "Completed", result.Rows[0][1])
}

// TestExecuteWrapsFailureAsExecutionError exercises Execute directly against
// a hand-built TranspileResult (bypassing Transpile) whose SQL fails at the
// database rather than at validation -- the same shape a stale tables.go
// allowlist entry or an unforeseen runtime type conversion would produce.
// The GQL resolver relies on this wrapping to render the failure as a
// diagnostic instead of a bare top-level error, matching *ValidationError's
// own treatment.
func TestExecuteWrapsFailureAsExecutionError(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	tr := &insights.TranspileResult{SQL: "SELECT * FROM this_table_does_not_exist;"}
	_, err := insights.Execute(context.Background(), db, tr)
	require.Error(t, err)

	var eerr *insights.ExecutionError
	require.ErrorAs(t, err, &eerr)
	d := eerr.Diagnostic()
	require.Equal(t, insights.DiagnosticError, d.Severity)
	require.Equal(t, "execution-error", d.Code)
	require.NotEmpty(t, d.Message)
}

func TestExecuteEmptyResultStillReportsColumns(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	accountID := uuid.New()
	tr, err := insights.Transpile("SELECT run_id FROM runs WHERE run_id = 'nonexistent'", accountID, testEnvIDForExecuteTest)
	require.NoError(t, err)

	result, err := insights.Execute(context.Background(), db, tr)
	require.NoError(t, err)
	require.Empty(t, result.Rows)
	// The whole point of the DESCRIBE fallback: a zero-row result must
	// still report its column, not silently drop it.
	require.Len(t, result.Columns, 1)
	require.Equal(t, "run_id", result.Columns[0].Name)
	require.Equal(t, insights.HintRunID, result.Columns[0].Hint)
}
