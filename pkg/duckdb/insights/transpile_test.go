package insights

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTranspileHappyPath(t *testing.T) {
	tr, err := Transpile("SELECT run_id, app_id FROM runs WHERE status = 'Completed'", testAccountID, testEnvID)
	require.NoError(t, err)

	require.Equal(t, []any{testAccountID.String(), testEnvID.String()}, tr.Args)
	require.Equal(t, "runs", tr.PrimaryTable)
	require.Equal(t, []string{"runs"}, tr.Tables)
	require.True(t, tr.Limited)
	require.Equal(t, []ColumnHint{HintRunID, HintAppID}, tr.ColumnHints)
	require.Contains(t, tr.SQL, "inngest.insights_runs")
	require.Contains(t, tr.SQL, "LIMIT 1000")

	require.Len(t, tr.Diagnostics, 1)
	require.Equal(t, "default-limit-applied", tr.Diagnostics[0].Code)
	require.Equal(t, DiagnosticInfo, tr.Diagnostics[0].Severity)
}

func TestTranspileNoLimitDiagnosticWhenLimitSpecified(t *testing.T) {
	tr, err := Transpile("SELECT run_id FROM runs LIMIT 5", testAccountID, testEnvID)
	require.NoError(t, err)
	require.Empty(t, tr.Diagnostics)
}

func TestTranspileArrayOfStructDiagnostic(t *testing.T) {
	tr, err := Transpile("SELECT sessions.key FROM runs LIMIT 5", testAccountID, testEnvID)
	require.NoError(t, err)
	require.Contains(t, tr.SQL, "sessions ->> '$[*].key'")
	require.Len(t, tr.Diagnostics, 1)
	require.Equal(t, "array-of-struct-access-rewritten", tr.Diagnostics[0].Code)
}

func TestTranspileRejectsInvalidQuery(t *testing.T) {
	_, err := Transpile("SELECT * FROM nonexistent_table", testAccountID, testEnvID)
	require.Error(t, err)
}

func TestTranspileRejectsMalformedSQL(t *testing.T) {
	_, err := Transpile("SELEKT * FROM runs", testAccountID, testEnvID)
	require.Error(t, err)
}

func TestTranspileRejectsMultipleStatements(t *testing.T) {
	_, err := Transpile("SELECT run_id FROM runs; SELECT run_id FROM runs;", testAccountID, testEnvID)
	require.Error(t, err)
}

func TestTranspileRespectsExistingLimit(t *testing.T) {
	tr, err := Transpile("SELECT run_id FROM runs LIMIT 5", testAccountID, testEnvID)
	require.NoError(t, err)
	require.False(t, tr.Limited)
}
