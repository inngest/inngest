package insights

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTryShortCircuitShowTables(t *testing.T) {
	for _, sql := range []string{"SHOW TABLES", "show tables", "  SHOW TABLES;  "} {
		sc, ok, err := TryShortCircuit(sql)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, "name", sc.Result.Columns[0].Name)
		require.Equal(t, "description", sc.Result.Columns[1].Name)
		require.Len(t, sc.Result.Rows, len(logicalTables))
		require.ElementsMatch(t, sc.Tables, []string{
			"runs", "events", "metadata", "extended_trace_spans", "steps", "step_attempts",
		})
		for _, row := range sc.Result.Rows {
			require.NotEmptyf(t, row[1], "table %q must have a non-empty description", row[0])
		}
	}
}

func TestTryShortCircuitDescribeKnownTable(t *testing.T) {
	sc, ok, err := TryShortCircuit("DESCRIBE runs")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{"runs"}, sc.Tables)
	require.Equal(t, "column_name", sc.Result.Columns[0].Name)
	require.Equal(t, "column_type", sc.Result.Columns[1].Name)
	require.Equal(t, "description", sc.Result.Columns[2].Name)
	require.Len(t, sc.Result.Rows, len(logicalTables["runs"].columnOrder))
	require.Equal(t, "run_id", sc.Result.Rows[0][0])
	require.Equal(t, "STRING", sc.Result.Rows[0][1])
	require.NotEmpty(t, sc.Result.Rows[0][2])

	for _, row := range sc.Result.Rows {
		require.NotEmptyf(t, row[2], "column %q must have a non-empty description", row[0])
	}
}

// TestAllTablesAndColumnsHaveDescriptions guards every logical table (and
// every column on it) against a missing description -- a bare "" would
// otherwise silently pass validate/Transpile and only show up as an empty
// cell in DESCRIBE's output or the cmd/gen-insights-schema JSON dump.
func TestAllTablesAndColumnsHaveDescriptions(t *testing.T) {
	for name, tbl := range logicalTables {
		require.NotEmptyf(t, tbl.description, "table %q has no description", name)
		for _, col := range tbl.columnOrder {
			require.NotEmptyf(t, tbl.columns[col].description, "table %q column %q has no description", name, col)
		}
	}
}

func TestTryShortCircuitDescribeIsCaseInsensitive(t *testing.T) {
	sc, ok, err := TryShortCircuit("describe RUNS;")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{"runs"}, sc.Tables)
}

func TestTryShortCircuitDescribeUnknownTable(t *testing.T) {
	_, ok, err := TryShortCircuit("DESCRIBE nonexistent_evil_table")
	require.True(t, ok)
	require.Error(t, err)
	verr, isValidationErr := err.(*ValidationError)
	require.True(t, isValidationErr)
	require.Contains(t, verr.Message, "unknown table")
}

func TestTryShortCircuitFallsThroughForRealQueries(t *testing.T) {
	for _, sql := range []string{
		"SELECT * FROM runs",
		"DESCRIBE",
		"DESCRIBE ",
		"SHOW TABLE runs",
	} {
		_, ok, err := TryShortCircuit(sql)
		require.NoError(t, err)
		require.Falsef(t, ok, "expected %q not to be short-circuited", sql)
	}
}
