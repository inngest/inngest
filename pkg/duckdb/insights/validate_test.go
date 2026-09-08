package insights

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAccepts(t *testing.T) {
	queries := []string{
		"SELECT run_id, app_id FROM runs",
		"SELECT r.run_id FROM runs r",
		"SELECT * FROM runs WHERE status = 'Completed'",
		"SELECT COUNT(*) FROM runs GROUP BY app_id",
		"SELECT run_id FROM runs JOIN events ON runs.run_id = events.id",
		"SELECT run_id FROM runs UNION SELECT run_id FROM extended_trace_spans",
		"SELECT attributes ->> 'function.id' FROM extended_trace_spans",
		"SELECT attributes -> 'function.id' FROM extended_trace_spans",
		"SELECT json_extract_string(attributes, 'function.id') FROM extended_trace_spans",
		// DuckDB's dot operator also works as a JSON path extraction off a
		// JSON-typed column (ClickHouse-dialect ergonomics) -- data isn't a
		// table here, but it is a known JSON column on events.
		"SELECT data.function_id FROM events",
		// The same access chains to arbitrary depth, quoted path segments
		// included -- inngest is a known JSON column on runs.
		`SELECT * FROM runs WHERE inngest.score."latency-score".value > 0.5`,
		// A qualified reference (table.column) can go past the column too,
		// once that column is JSON-typed.
		"SELECT runs.inngest.score FROM runs",
		// UNNEST as a SELECT-list expression.
		"SELECT UNNEST(event_ids) FROM runs",
		// UNNEST as a FROM-clause table function -- the one table function
		// this package allows, since (unlike read_csv/read_parquet/etc.) it
		// can't reach the filesystem or catalog. Its argument (run_id) is
		// validated like any other expression.
		"SELECT unnest FROM runs, UNNEST(run_id)",
		"SELECT t.unnest FROM runs, UNNEST(run_id) AS t",
		// A sampling of the newly allowed functions.
		"SELECT YEAR(queued_at), MONTH(queued_at) FROM runs",
		"SELECT REGEXP_REPLACE(run_id, 'a', 'b') FROM runs",
		"SELECT STARTS_WITH(run_id, 'x') FROM runs",
		"SELECT JSON_KEYS(inngest) FROM runs",
		"SELECT IF(status = 'Completed', run_id, app_id) FROM runs",
		"SELECT IFNULL(app_id, function_id) FROM runs",
	}
	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			stmt := mustParse(t, q)
			_, _, err := validate(stmt)
			require.NoError(t, err)
		})
	}
}

func TestValidateAcceptsNamedWindowClause(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id, SUM(1) OVER w FROM runs WINDOW w AS (PARTITION BY app_id ORDER BY run_id)")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

// Regression test: a named WINDOW clause is a separate top-level field
// (SelectStatement.Windows) that parser.Walk never reaches unless
// collectExprs explicitly includes it — an inline OVER(...) is reachable
// through its FunctionExpr, but "WINDOW w AS (...)" referenced elsewhere
// as "OVER w" is not. Before this was fixed, this query passed validate
// with no error at all, letting an arbitrary column reference inside a
// named window's PARTITION BY/ORDER BY through completely unchecked.
func TestValidateRejectsUnknownColumnInNamedWindowClause(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id, SUM(1) OVER w FROM runs WINDOW w AS (PARTITION BY nonexistent_evil_column)")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateAcceptsDistinctOn(t *testing.T) {
	stmt := mustParse(t, "SELECT DISTINCT ON (app_id) run_id FROM runs")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

// Regression test: same class of bug as the named-WINDOW gap — DistinctClause.On
// is a separate top-level field collectExprs didn't walk.
func TestValidateRejectsUnknownColumnInDistinctOn(t *testing.T) {
	stmt := mustParse(t, "SELECT DISTINCT ON (nonexistent_evil_column) run_id FROM runs")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateAcceptsGroupingSets(t *testing.T) {
	stmt := mustParse(t, "SELECT app_id, function_id, COUNT(*) FROM runs GROUP BY GROUPING SETS ((app_id), (function_id))")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

// Regression test: groupByItemExprs previously returned nil for
// *parser.GroupingSets (an unhandled default case), so any column inside
// one of its sets bypassed validation entirely.
func TestValidateRejectsUnknownColumnInGroupingSets(t *testing.T) {
	stmt := mustParse(t, "SELECT app_id, COUNT(*) FROM runs GROUP BY GROUPING SETS ((app_id), (nonexistent_evil_column))")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateAcceptsStarExclude(t *testing.T) {
	stmt := mustParse(t, "SELECT * EXCLUDE (run_id) FROM runs")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

// Regression test: StarExpr.Children() returns nil (Qualifier/Exclude are
// plain string slices, not Expr/Ident nodes), so without an explicit
// checkStar case, neither field was ever checked against the known table/
// column set at all.
func TestValidateRejectsUnknownColumnInStarExclude(t *testing.T) {
	stmt := mustParse(t, "SELECT * EXCLUDE (nonexistent_evil_column) FROM runs")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsUnknownQualifierInStar(t *testing.T) {
	stmt := mustParse(t, "SELECT bogus_alias.* FROM runs")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown table")
}

func TestValidateRejectsUnknownTable(t *testing.T) {
	stmt := mustParse(t, "SELECT * FROM nonexistent_table")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown table")
}

func TestValidateRejectsNonUnnestTableFunctions(t *testing.T) {
	// UNNEST is the one FROM-clause table function this package allows
	// (addTableFunction, scope.go) -- everything else, filesystem/
	// catalog-touching or not, stays rejected.
	queries := []string{
		"SELECT * FROM read_csv('/etc/passwd')",
		"SELECT * FROM read_parquet('/etc/passwd')",
		"SELECT * FROM generate_series(1, 10)",
	}
	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			stmt := mustParse(t, q)
			_, _, err := validate(stmt)
			require.Error(t, err)
			require.Contains(t, err.Error(), "not allowed")
		})
	}
}

func TestValidateRejectsUnnestWithWrongArgCount(t *testing.T) {
	stmt := mustParse(t, "SELECT * FROM runs, UNNEST(run_id, app_id)")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exactly one argument")
}

func TestValidateRejectsUnnestArgumentToUnknownColumn(t *testing.T) {
	// UNNEST's own argument is validated like any other expression --
	// bogus_column doesn't exist, so this must fail rather than reach
	// remapTables/DuckDB unchecked.
	stmt := mustParse(t, "SELECT * FROM runs, UNNEST(bogus_column)")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsDotAccessOnNonJSONColumn(t *testing.T) {
	// run_id is a known column on runs, but it's a string, not JSON, so
	// run_id.foo must still fail as an unknown table rather than being
	// silently accepted as a JSON path extraction.
	stmt := mustParse(t, "SELECT run_id.foo FROM runs")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown table "run_id"`)
}

func TestValidateRejectsDeepDotAccessOnNonJSONQualifiedColumn(t *testing.T) {
	// runs.run_id is a real, known column, but it's a string, not JSON,
	// so a path past it must still be rejected.
	stmt := mustParse(t, "SELECT runs.run_id.foo FROM runs")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported identifier")
}

func TestValidateRejectsUnknownColumn(t *testing.T) {
	stmt := mustParse(t, "SELECT nonexistent_column FROM runs")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsDisallowedFunction(t *testing.T) {
	stmt := mustParse(t, "SELECT read_csv('/etc/passwd') FROM runs")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestValidateAcceptsSubquery(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs WHERE run_id IN (SELECT run_id FROM runs)")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateAcceptsCorrelatedSubquery(t *testing.T) {
	// An expression-position subquery (IN/scalar/EXISTS) may correlate to
	// its enclosing query's own scope with no special syntax, matching
	// standard SQL — runs.status resolves against the outer query's own
	// "runs" even though the subquery's own FROM is "events", which has
	// no "status" column of its own. See tableScope.outer's doc comment.
	stmt := mustParse(t, "SELECT run_id FROM runs WHERE run_id IN (SELECT id FROM events WHERE status = runs.status)")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateRejectsCorrelatedSubqueryToGenuinelyUnknownColumn(t *testing.T) {
	// A reference that isn't a column of the subquery's own scope, nor of
	// any enclosing scope, still fails as "unknown column" — the
	// correlation fallback finds nothing to resolve to, it doesn't ever
	// silently succeed.
	stmt := mustParse(t, "SELECT run_id FROM runs WHERE run_id IN (SELECT id FROM events WHERE bogus_evil_column = 1)")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateAcceptsMultiLevelCorrelatedSubquery(t *testing.T) {
	// Correlation chains transitively: a subquery nested two levels deep
	// can still reach the outermost query's own scope, not just its
	// immediate parent's.
	stmt := mustParse(t, "SELECT run_id FROM runs WHERE EXISTS (SELECT 1 FROM events WHERE EXISTS (SELECT 1 FROM extended_trace_spans WHERE extended_trace_spans.run_id = runs.run_id))")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateAcceptsLateralFromSubquery(t *testing.T) {
	// LATERAL is the FROM-clause mechanism for the same kind of
	// correlation an expression-position subquery gets for free: it may
	// reference columns from FROM items that appear strictly *before* it
	// in the same FROM clause.
	stmt := mustParse(t, "SELECT run_id FROM runs, LATERAL (SELECT id FROM events WHERE events.id = runs.run_id) AS sub")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateRejectsNonLateralFromSubqueryCorrelation(t *testing.T) {
	// The same query, minus LATERAL, must still be rejected — matching
	// real DuckDB's own rule that a derived table can't see a preceding
	// FROM item without it. "runs" is qualifying a column here, so an
	// unresolvable qualifier is reported as "unknown table", same as any
	// other bogus qualifier (TestValidateRejectsUnknownQualifierInStar).
	stmt := mustParse(t, "SELECT run_id FROM runs, (SELECT id FROM events WHERE events.id = runs.run_id) AS sub")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown table")
}

func TestValidateRejectsLateralFromSubqueryCorrelatingToLaterItem(t *testing.T) {
	// LATERAL only grants left-to-right visibility: a LATERAL subquery
	// can't see a FROM item that appears *after* it, even though that
	// item is otherwise in scope for the enclosing query as a whole.
	stmt := mustParse(t, "SELECT run_id FROM LATERAL (SELECT id FROM events WHERE events.id = runs.run_id) AS sub, runs")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown table")
}

func TestValidateAcceptsCTE(t *testing.T) {
	stmt := mustParse(t, "WITH x AS (SELECT run_id FROM runs) SELECT * FROM x")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateAcceptsSequentialCTEs(t *testing.T) {
	stmt := mustParse(t, "WITH a AS (SELECT run_id, app_id FROM runs), b AS (SELECT run_id FROM a) SELECT * FROM b")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateAcceptsFromSubquery(t *testing.T) {
	stmt := mustParse(t, "SELECT sub.run_id FROM (SELECT run_id FROM runs) AS sub")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateRejectsFromSubqueryWithoutAlias(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM (SELECT run_id FROM runs)")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "alias")
}

func TestValidateRejectsCTEUnknownColumn(t *testing.T) {
	stmt := mustParse(t, "WITH x AS (SELECT nonexistent_evil_column FROM runs) SELECT * FROM x")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsReferenceToOuterColumnFromCTEBody(t *testing.T) {
	// A CTE's own body only ever sees its own FROM scope plus earlier
	// CTEs -- never anything from whatever eventually consumes it.
	stmt := mustParse(t, "WITH x AS (SELECT run_id FROM runs WHERE run_id = outer_alias.run_id) SELECT * FROM x, runs AS outer_alias")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown")
}

func TestValidateRejectsSelfReferencingCTE(t *testing.T) {
	// Non-recursive CTEs (RECURSIVE is rejected outright) can't reference
	// themselves -- x isn't yet a known table while its own body is being
	// validated.
	stmt := mustParse(t, "WITH x AS (SELECT run_id FROM x) SELECT * FROM x")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown table")
}

func TestValidateRejectsRecursiveCTE(t *testing.T) {
	stmt := mustParse(t, "WITH RECURSIVE x AS (SELECT run_id FROM runs) SELECT * FROM x")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "recursive")
}

func TestValidateRejectsUnaliasedComputedColumnInCTE(t *testing.T) {
	stmt := mustParse(t, "WITH x AS (SELECT COUNT(*) FROM runs) SELECT * FROM x")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must have an alias")
}

func TestValidateAcceptsCTEReferencedTwice(t *testing.T) {
	stmt := mustParse(t, "WITH x AS (SELECT run_id, app_id FROM runs) SELECT a.run_id FROM x a JOIN x b ON a.app_id = b.app_id")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateRejectsLambdaOnUnknownColumn(t *testing.T) {
	stmt := mustParse(t, "SELECT bogus -> 'x' FROM extended_trace_spans")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsMultiParamLambda(t *testing.T) {
	stmt := mustParse(t, "SELECT list_transform([1,2], (x, y) -> x + y) FROM runs")
	_, _, err := validate(stmt)
	require.Error(t, err)
}

func TestValidateAcceptsListComprehension(t *testing.T) {
	// e is a loop variable bound by the comprehension itself, not a real
	// column of runs -- it must resolve without ever consulting scope.
	stmt := mustParse(t, "SELECT [UPPER(e) FOR e IN event_ids] FROM runs")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateAcceptsListComprehensionWithFilter(t *testing.T) {
	stmt := mustParse(t, "SELECT [e FOR e IN event_ids IF e IS NOT NULL] FROM runs")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateAcceptsNestedListComprehension(t *testing.T) {
	// The outer comprehension's Source is itself a comprehension binding
	// its own loop variable (y) -- that inner binding must be fully popped
	// again before the outer one (x) is pushed, or the two would bleed
	// into each other.
	stmt := mustParse(t, "SELECT [x FOR x IN [y FOR y IN event_ids]] FROM runs")
	_, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateRejectsListComprehensionUnknownColumnInSource(t *testing.T) {
	stmt := mustParse(t, "SELECT [e FOR e IN nonexistent_evil_column] FROM runs")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsListComprehensionUnknownColumnInBody(t *testing.T) {
	// nonexistent_evil_column is neither a real column nor the
	// comprehension's own bound variable (e) -- still rejected.
	stmt := mustParse(t, "SELECT [nonexistent_evil_column FOR e IN event_ids] FROM runs")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsListComprehensionVariableLeakingOutsideItsScope(t *testing.T) {
	// e is only bound inside the comprehension's own Expr/Filter -- once
	// validation moves past it, a bare "e" outside must still fail as
	// unknown, proving the binding doesn't leak into the rest of the query.
	stmt := mustParse(t, "SELECT [e FOR e IN event_ids], e FROM runs")
	_, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}
