package insights

import (
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/parser"
	"github.com/stretchr/testify/require"
)

func mustParse(t *testing.T, sql string) *parser.SelectStatement {
	t.Helper()
	stmt, err := parser.ParseString(sql)
	require.NoError(t, err)
	return stmt
}

func TestResolveScopeBaseTable(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs")
	scope, err := resolveScope(stmt.From, nil, nil)
	require.NoError(t, err)

	tbl, ok := scope.lookup("runs")
	require.True(t, ok)
	require.Equal(t, "runs", tbl.name)
	require.Equal(t, 1, scope.columnCount("run_id"))
	require.Equal(t, 0, scope.columnCount("nonexistent"))
}

func TestResolveScopeColumnCountAmbiguous(t *testing.T) {
	// run_id exists on both runs and extended_trace_spans.
	stmt := mustParse(t, "SELECT run_id FROM runs JOIN extended_trace_spans ON runs.run_id = extended_trace_spans.run_id")
	scope, err := resolveScope(stmt.From, nil, nil)
	require.NoError(t, err)
	require.Equal(t, 2, scope.columnCount("run_id"))
}

func TestResolveScopeAlias(t *testing.T) {
	stmt := mustParse(t, "SELECT r.run_id FROM runs r")
	scope, err := resolveScope(stmt.From, nil, nil)
	require.NoError(t, err)

	_, ok := scope.lookup("runs")
	require.False(t, ok, "unaliased name should not resolve once aliased")
	tbl, ok := scope.lookup("r")
	require.True(t, ok)
	require.Equal(t, "runs", tbl.name)
}

func TestResolveScopeJoin(t *testing.T) {
	stmt := mustParse(t, "SELECT * FROM runs JOIN events ON runs.run_id = events.id")
	scope, err := resolveScope(stmt.From, nil, nil)
	require.NoError(t, err)

	_, ok := scope.lookup("runs")
	require.True(t, ok)
	_, ok = scope.lookup("events")
	require.True(t, ok)
}

func TestResolveScopeUnknownTable(t *testing.T) {
	stmt := mustParse(t, "SELECT * FROM nonexistent_table")
	_, err := resolveScope(stmt.From, nil, nil)
	require.Error(t, err)
	var verr *ValidationError
	require.ErrorAs(t, err, &verr)
}

func TestResolveScopeUniqueColumn(t *testing.T) {
	stmt := mustParse(t, "SELECT * FROM runs")
	scope, err := resolveScope(stmt.From, nil, nil)
	require.NoError(t, err)

	col, ok := scope.uniqueColumn("run_id")
	require.True(t, ok)
	require.Equal(t, HintRunID, col.hint())

	_, ok = scope.uniqueColumn("nonexistent")
	require.False(t, ok)
}

func TestLogicalTablesColumnOrderMatchesColumns(t *testing.T) {
	for name, tbl := range logicalTables {
		require.Lenf(t, tbl.columnOrder, len(tbl.columns), "table %s: columnOrder length must match columns map length", name)
		for _, col := range tbl.columnOrder {
			_, ok := tbl.columns[col]
			require.Truef(t, ok, "table %s: columnOrder entry %q missing from columns map", name, col)
		}
	}
}

// Matches DuckDB's derived-table column naming, checked with DESCRIBE.
func TestDedupeColumnNames(t *testing.T) {
	require.Equal(t, []string{"x", "x_1", "x_2", "x_1_1"}, dedupeColumnNames([]string{"x", "x", "x", "x_1"}))
	require.Equal(t, []string{"x", "x_1", "x_1_1"}, dedupeColumnNames([]string{"x", "x", "x_1"}))
	require.Equal(t, []string{"x", "X_1"}, dedupeColumnNames([]string{"x", "X"}))
}

// A duplicate output name is renamed rather than collapsed, so the derived
// table keeps every column (and "SELECT *" over it stays aligned); an
// explicit column list applies before the renaming (DuckDB: t(p, x) over
// (x, y, x) is (p, x, x_1)).
func TestInferOutputColumnsRenamesDuplicates(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id AS x, app_id AS y, function_id AS x FROM runs")
	scope, _, _, err := validate(stmt)
	require.NoError(t, err)

	order, cols, err := inferOutputColumns(stmt, scope, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"x", "y", "x_1"}, order)
	require.Equal(t, HintFunctionID, cols["x_1"].hint())

	order, cols, err = inferOutputColumns(stmt, scope, []string{"p", "x"})
	require.NoError(t, err)
	require.Equal(t, []string{"p", "x", "x_1"}, order)
	require.Equal(t, HintRunID, cols["p"].hint())
	require.Equal(t, HintAppID, cols["x"].hint())
	require.Equal(t, HintFunctionID, cols["x_1"].hint())
}

func TestValidateAcceptsRenamedDuplicateColumn(t *testing.T) {
	stmt := mustParse(t, "WITH x AS (SELECT r.run_id, e.run_id FROM runs r JOIN extended_trace_spans e ON r.run_id = e.run_id) SELECT run_id_1 FROM x")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}
