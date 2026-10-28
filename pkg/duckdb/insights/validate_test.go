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
		// A sampling of the second, larger round of allowed functions --
		// see allowedFunctions' own doc comment (functions.go) for the full
		// categorized list and functionReturnType (typecheck.go) for their
		// return-type rules.
		"SELECT ROW_NUMBER() OVER (ORDER BY queued_at) FROM runs",
		"SELECT RANK() OVER (PARTITION BY app_id ORDER BY queued_at) FROM runs",
		"SELECT LAG(status) OVER (ORDER BY queued_at) FROM runs",
		"SELECT MEDIAN(queued_at), QUANTILE_CONT(queued_at, 0.95) FROM runs",
		"SELECT STDDEV(step_index), APPROX_QUANTILE(step_index, 0.5) FROM steps",
		"SELECT FIRST(status), LAST(status), ARG_MAX(run_id, queued_at) FROM runs GROUP BY app_id",
		"SELECT BOOL_AND(is_deferred) FROM runs GROUP BY app_id",
		"SELECT LEFT(run_id, 4), RIGHT(run_id, 4), MD5(run_id), HASH(run_id) FROM runs",
		"SELECT CEIL(step_index / 2.0), SQRT(step_index), MOD(step_index, 2) FROM steps",
		"SELECT ISODOW(queued_at), WEEK(queued_at), QUARTER(queued_at) FROM runs",
		"SELECT JSON_OBJECT('run', run_id), JSON_ARRAY(1, 2) FROM runs",
		"SELECT LIST_CONTAINS(event_ids, 'x'), ARRAY_LENGTH(event_ids) FROM runs",
		"SELECT TODAY(), CURRENT_DATE() FROM runs",
	}
	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			stmt := mustParse(t, q)
			_, _, _, err := validate(stmt)
			require.NoError(t, err)
		})
	}
}

func TestValidateAcceptsNamedWindowClause(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id, SUM(1) OVER w FROM runs WINDOW w AS (PARTITION BY app_id ORDER BY run_id)")
	_, _, _, err := validate(stmt)
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
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

// TestValidateErrorSpansOffendingIdentifier proves a *ValidationError's
// End is the offending node's own End(), not a zero-width copy of Pos --
// otherwise Diagnostic() can only underline a single character in the
// editor instead of the whole bad identifier.
func TestValidateErrorSpansOffendingIdentifier(t *testing.T) {
	stmt := mustParse(t, "SELECT nonexistent_evil_column FROM runs")
	_, _, _, err := validate(stmt)
	require.Error(t, err)

	verr, ok := err.(*ValidationError)
	require.True(t, ok)
	require.NotEqual(t, verr.Pos, verr.End, "End must span the identifier, not collapse to Pos")
	require.Equal(t, len("nonexistent_evil_column"), verr.End.Column-verr.Pos.Column)
}

func TestValidateAcceptsDistinctOn(t *testing.T) {
	stmt := mustParse(t, "SELECT DISTINCT ON (app_id) run_id FROM runs")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

// Regression test: same class of bug as the named-WINDOW gap — DistinctClause.On
// is a separate top-level field collectExprs didn't walk.
func TestValidateRejectsUnknownColumnInDistinctOn(t *testing.T) {
	stmt := mustParse(t, "SELECT DISTINCT ON (nonexistent_evil_column) run_id FROM runs")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateAcceptsGroupingSets(t *testing.T) {
	stmt := mustParse(t, "SELECT app_id, function_id, COUNT(*) FROM runs GROUP BY GROUPING SETS ((app_id), (function_id))")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

// Regression test: groupByItemExprs previously returned nil for
// *parser.GroupingSets (an unhandled default case), so any column inside
// one of its sets bypassed validation entirely.
func TestValidateRejectsUnknownColumnInGroupingSets(t *testing.T) {
	stmt := mustParse(t, "SELECT app_id, COUNT(*) FROM runs GROUP BY GROUPING SETS ((app_id), (nonexistent_evil_column))")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateAcceptsStarExclude(t *testing.T) {
	stmt := mustParse(t, "SELECT * EXCLUDE (run_id) FROM runs")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

// Regression test: StarExpr.Children() returns nil (Qualifier/Exclude are
// plain string slices, not Expr/Ident nodes), so without an explicit
// checkStar case, neither field was ever checked against the known table/
// column set at all.
func TestValidateRejectsUnknownColumnInStarExclude(t *testing.T) {
	stmt := mustParse(t, "SELECT * EXCLUDE (nonexistent_evil_column) FROM runs")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsUnknownQualifierInStar(t *testing.T) {
	stmt := mustParse(t, "SELECT bogus_alias.* FROM runs")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown table")
}

func TestValidateRejectsUnknownTable(t *testing.T) {
	stmt := mustParse(t, "SELECT * FROM nonexistent_table")
	_, _, _, err := validate(stmt)
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
			_, _, _, err := validate(stmt)
			require.Error(t, err)
			require.Contains(t, err.Error(), "not allowed")
		})
	}
}

func TestValidateRejectsUnnestWithWrongArgCount(t *testing.T) {
	stmt := mustParse(t, "SELECT * FROM runs, UNNEST(run_id, app_id)")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exactly one argument")
}

func TestValidateRejectsUnnestArgumentToUnknownColumn(t *testing.T) {
	// UNNEST's own argument is validated like any other expression --
	// bogus_column doesn't exist, so this must fail rather than reach
	// remapTables/DuckDB unchecked.
	stmt := mustParse(t, "SELECT * FROM runs, UNNEST(bogus_column)")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsDotAccessOnNonJSONColumn(t *testing.T) {
	// run_id is a known column on runs, but it's a string, not JSON, so
	// run_id.foo must still fail as an unknown table rather than being
	// silently accepted as a JSON path extraction.
	stmt := mustParse(t, "SELECT run_id.foo FROM runs")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown table "run_id"`)
}

func TestValidateRejectsDeepDotAccessOnNonJSONQualifiedColumn(t *testing.T) {
	// runs.run_id is a real, known column, but it's a string, not JSON,
	// so a path past it must still be rejected.
	stmt := mustParse(t, "SELECT runs.run_id.foo FROM runs")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported identifier")
}

func TestValidateRejectsUnknownColumn(t *testing.T) {
	stmt := mustParse(t, "SELECT nonexistent_column FROM runs")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsDisallowedFunction(t *testing.T) {
	stmt := mustParse(t, "SELECT read_csv('/etc/passwd') FROM runs")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestValidateAcceptsSubquery(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs WHERE run_id IN (SELECT run_id FROM runs)")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateAcceptsCorrelatedSubquery(t *testing.T) {
	// An expression-position subquery (IN/scalar/EXISTS) may correlate to
	// its enclosing query's own scope with no special syntax, matching
	// standard SQL — runs.status resolves against the outer query's own
	// "runs" even though the subquery's own FROM is "events", which has
	// no "status" column of its own. See tableScope.outer's doc comment.
	stmt := mustParse(t, "SELECT run_id FROM runs WHERE run_id IN (SELECT id FROM events WHERE status = runs.status)")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateRejectsCorrelatedSubqueryToGenuinelyUnknownColumn(t *testing.T) {
	// A reference that isn't a column of the subquery's own scope, nor of
	// any enclosing scope, still fails as "unknown column" — the
	// correlation fallback finds nothing to resolve to, it doesn't ever
	// silently succeed.
	stmt := mustParse(t, "SELECT run_id FROM runs WHERE run_id IN (SELECT id FROM events WHERE bogus_evil_column = 1)")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateScopesQuantifiedSubqueryLikeInSubquery(t *testing.T) {
	// A subquery under = ANY / ALL is an expression-position subquery just
	// like IN (SELECT ...): its tables and columns are validated, and it
	// may correlate to the enclosing scope.
	for _, sql := range []string{
		"SELECT run_id FROM runs WHERE run_id = ANY (SELECT run_id FROM runs)",
		"SELECT run_id FROM runs WHERE run_id <> ALL (SELECT id FROM events WHERE status = runs.status)",
	} {
		_, _, _, err := validate(mustParse(t, sql))
		require.NoError(t, err, sql)
	}

	_, _, _, err := validate(mustParse(t, "SELECT run_id FROM runs WHERE run_id = ANY (SELECT id FROM events WHERE bogus_evil_column = 1)"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")

	_, _, _, err = validate(mustParse(t, "SELECT run_id FROM runs WHERE run_id = ANY (SELECT * FROM nonexistent_table)"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown table")

	_, _, _, err = validate(mustParse(t, "SELECT run_id FROM runs WHERE run_id >= ALL (SELECT read_csv('/etc/passwd') FROM runs)"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestValidateAcceptsMultiLevelCorrelatedSubquery(t *testing.T) {
	// Correlation chains transitively: a subquery nested two levels deep
	// can still reach the outermost query's own scope, not just its
	// immediate parent's.
	stmt := mustParse(t, "SELECT run_id FROM runs WHERE EXISTS (SELECT 1 FROM events WHERE EXISTS (SELECT 1 FROM extended_trace_spans WHERE extended_trace_spans.run_id = runs.run_id))")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateAcceptsLateralFromSubquery(t *testing.T) {
	// LATERAL is the FROM-clause mechanism for the same kind of
	// correlation an expression-position subquery gets for free: it may
	// reference columns from FROM items that appear strictly *before* it
	// in the same FROM clause.
	stmt := mustParse(t, "SELECT run_id FROM runs, LATERAL (SELECT id FROM events WHERE events.id = runs.run_id) AS sub")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateRejectsNonLateralFromSubqueryCorrelation(t *testing.T) {
	// The same query, minus LATERAL, must still be rejected — matching
	// real DuckDB's own rule that a derived table can't see a preceding
	// FROM item without it. "runs" is qualifying a column here, so an
	// unresolvable qualifier is reported as "unknown table", same as any
	// other bogus qualifier (TestValidateRejectsUnknownQualifierInStar).
	stmt := mustParse(t, "SELECT run_id FROM runs, (SELECT id FROM events WHERE events.id = runs.run_id) AS sub")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown table")
}

func TestValidateRejectsLateralFromSubqueryCorrelatingToLaterItem(t *testing.T) {
	// LATERAL only grants left-to-right visibility: a LATERAL subquery
	// can't see a FROM item that appears *after* it, even though that
	// item is otherwise in scope for the enclosing query as a whole.
	stmt := mustParse(t, "SELECT run_id FROM LATERAL (SELECT id FROM events WHERE events.id = runs.run_id) AS sub, runs")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown table")
}

func TestValidateAcceptsCTE(t *testing.T) {
	stmt := mustParse(t, "WITH x AS (SELECT run_id FROM runs) SELECT * FROM x")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateAcceptsSequentialCTEs(t *testing.T) {
	stmt := mustParse(t, "WITH a AS (SELECT run_id, app_id FROM runs), b AS (SELECT run_id FROM a) SELECT * FROM b")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateAcceptsFromSubquery(t *testing.T) {
	stmt := mustParse(t, "SELECT sub.run_id FROM (SELECT run_id FROM runs) AS sub")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateRejectsFromSubqueryWithoutAlias(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM (SELECT run_id FROM runs)")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "alias")
}

func TestValidateRejectsCTEUnknownColumn(t *testing.T) {
	stmt := mustParse(t, "WITH x AS (SELECT nonexistent_evil_column FROM runs) SELECT * FROM x")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsReferenceToOuterColumnFromCTEBody(t *testing.T) {
	// A CTE's own body only ever sees its own FROM scope plus earlier
	// CTEs -- never anything from whatever eventually consumes it.
	stmt := mustParse(t, "WITH x AS (SELECT run_id FROM runs WHERE run_id = outer_alias.run_id) SELECT * FROM x, runs AS outer_alias")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown")
}

func TestValidateRejectsSelfReferencingCTE(t *testing.T) {
	// Non-recursive CTEs (RECURSIVE is rejected outright) can't reference
	// themselves -- x isn't yet a known table while its own body is being
	// validated.
	stmt := mustParse(t, "WITH x AS (SELECT run_id FROM x) SELECT * FROM x")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown table")
}

func TestValidateRejectsRecursiveCTE(t *testing.T) {
	stmt := mustParse(t, "WITH RECURSIVE x AS (SELECT run_id FROM runs) SELECT * FROM x")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "recursive")
}

func TestValidateRejectsUnaliasedComputedColumnInCTE(t *testing.T) {
	stmt := mustParse(t, "WITH x AS (SELECT COUNT(*) FROM runs) SELECT * FROM x")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must have an alias")
}

func TestValidateAcceptsCTEReferencedTwice(t *testing.T) {
	stmt := mustParse(t, "WITH x AS (SELECT run_id, app_id FROM runs) SELECT a.run_id FROM x a JOIN x b ON a.app_id = b.app_id")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateRejectsLambdaOnUnknownColumn(t *testing.T) {
	stmt := mustParse(t, "SELECT bogus -> 'x' FROM extended_trace_spans")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsMultiParamLambda(t *testing.T) {
	stmt := mustParse(t, "SELECT list_transform([1,2], (x, y) -> x + y) FROM runs")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
}

func TestValidateAcceptsListComprehension(t *testing.T) {
	// e is a loop variable bound by the comprehension itself, not a real
	// column of runs -- it must resolve without ever consulting scope.
	stmt := mustParse(t, "SELECT [UPPER(e) FOR e IN event_ids] FROM runs")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateAcceptsListComprehensionWithFilter(t *testing.T) {
	stmt := mustParse(t, "SELECT [e FOR e IN event_ids IF e IS NOT NULL] FROM runs")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateAcceptsNestedListComprehension(t *testing.T) {
	// The outer comprehension's Source is itself a comprehension binding
	// its own loop variable (y) -- that inner binding must be fully popped
	// again before the outer one (x) is pushed, or the two would bleed
	// into each other.
	stmt := mustParse(t, "SELECT [x FOR x IN [y FOR y IN event_ids]] FROM runs")
	_, _, _, err := validate(stmt)
	require.NoError(t, err)
}

func TestValidateRejectsListComprehensionUnknownColumnInSource(t *testing.T) {
	stmt := mustParse(t, "SELECT [e FOR e IN nonexistent_evil_column] FROM runs")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsListComprehensionUnknownColumnInBody(t *testing.T) {
	// nonexistent_evil_column is neither a real column nor the
	// comprehension's own bound variable (e) -- still rejected.
	stmt := mustParse(t, "SELECT [nonexistent_evil_column FOR e IN event_ids] FROM runs")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

func TestValidateRejectsListComprehensionVariableLeakingOutsideItsScope(t *testing.T) {
	// e is only bound inside the comprehension's own Expr/Filter -- once
	// validation moves past it, a bare "e" outside must still fail as
	// unknown, proving the binding doesn't leak into the rest of the query.
	stmt := mustParse(t, "SELECT [e FOR e IN event_ids], e FROM runs")
	_, _, _, err := validate(stmt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown column")
}

// A set operation's own ORDER BY/LIMIT/OFFSET hang off the set-op node,
// not either operand, and used to skip validation entirely.
func TestValidateChecksSetOpModifiers(t *testing.T) {
	rejected := []struct{ sql, want string }{
		{"SELECT run_id FROM runs UNION ALL SELECT run_id FROM runs LIMIT 5 OFFSET length(read_text('x'))", "not allowed"},
		{"SELECT run_id FROM runs UNION ALL SELECT run_id FROM runs LIMIT length(read_text('x'))", "not allowed"},
		{"SELECT run_id FROM runs UNION ALL SELECT run_id FROM runs ORDER BY read_text('x')", "not allowed"},
		{"SELECT run_id FROM runs UNION ALL SELECT run_id FROM runs ORDER BY nonexistent_column", "unknown column"},
		{"SELECT run_id FROM runs UNION ALL SELECT run_id FROM runs ORDER BY (SELECT 1 FROM nonexistent_table)", "unknown table"},
		{"SELECT run_id FROM runs WHERE run_id IN (SELECT run_id FROM runs UNION SELECT run_id FROM runs OFFSET length(read_text('x')))", "not allowed"},
	}
	for _, c := range rejected {
		t.Run(c.sql, func(t *testing.T) {
			_, _, _, err := validate(mustParse(t, c.sql))
			require.ErrorContains(t, err, c.want)
		})
	}

	accepted := []string{
		"SELECT run_id FROM runs UNION ALL SELECT run_id FROM runs ORDER BY run_id LIMIT 5 OFFSET 2",
		"SELECT r.run_id FROM runs r UNION ALL SELECT run_id FROM runs ORDER BY r.run_id",
		"SELECT count(*) AS n FROM runs UNION ALL SELECT count(*) FROM runs ORDER BY n",
		"SELECT now() AS t FROM runs UNION ALL SELECT now() FROM runs ORDER BY now()",
	}
	for _, sql := range accepted {
		t.Run(sql, func(t *testing.T) {
			_, _, _, err := validate(mustParse(t, sql))
			require.NoError(t, err)
		})
	}
}

// A user-written parameter would collide with the "?" placeholders
// remapTables binds the macro calls' (account_id, env_id) to.
func TestValidateRejectsParameters(t *testing.T) {
	queries := []string{
		"SELECT ? FROM runs",
		"SELECT run_id FROM runs WHERE run_id = $1",
		"SELECT run_id FROM runs WHERE run_id = $name",
		"SELECT run_id FROM runs WHERE run_id IN (SELECT ? FROM events)",
		"SELECT run_id FROM runs LIMIT ?",
	}
	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			_, _, _, err := validate(mustParse(t, q))
			require.Error(t, err)
			require.Contains(t, err.Error(), "query parameters are not supported")
		})
	}
}

// DuckDB identifiers are case-insensitive (quoted ones too), so table,
// column, alias, CTE, EXCLUDE and comprehension-variable names all resolve
// regardless of case.
func TestValidateIdentifiersCaseInsensitive(t *testing.T) {
	queries := []string{
		"SELECT Run_ID FROM RUNS",
		`SELECT "RUN_ID" FROM "Runs"`,
		"SELECT R.Run_Id FROM runs r",
		"SELECT RUNS.run_id FROM runs",
		"WITH X AS (SELECT run_id AS Foo FROM runs) SELECT x.FOO, foo FROM x",
		"SELECT * EXCLUDE (RUN_ID) FROM runs",
		"SELECT r.* FROM runs R",
		"SELECT run_id AS Foo FROM runs ORDER BY foo",
		"SELECT [S FOR s IN event_ids] AS x FROM runs",
		"SELECT Inngest.score FROM runs",
		"SELECT t.FOO FROM (SELECT run_id AS foo FROM runs) T",
	}
	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			_, _, _, err := validate(mustParse(t, q))
			require.NoError(t, err)
		})
	}
}

// The parser represents "x COLLATE nocase"'s collation as an Ident, which
// isn't a column reference.
func TestValidateAcceptsCollate(t *testing.T) {
	for _, q := range []string{
		"SELECT status COLLATE nocase FROM runs",
		"SELECT run_id FROM runs ORDER BY status COLLATE nocase",
		`SELECT run_id FROM runs WHERE status COLLATE "nocase.noaccent" = 'completed'`,
	} {
		_, _, _, err := validate(mustParse(t, q))
		require.NoError(t, err, q)
	}
	// The collated operand itself is still checked.
	_, _, _, err := validate(mustParse(t, "SELECT nope COLLATE nocase FROM runs"))
	require.ErrorContains(t, err, `unknown column "nope"`)
}

// Only a (possibly dotted) collation name may follow COLLATE -- anything
// else would otherwise skip validation entirely.
func TestValidateRejectsNonNameCollate(t *testing.T) {
	for _, q := range []string{
		"SELECT status COLLATE (SELECT name FROM events) FROM runs",
		"SELECT status COLLATE lower(status) FROM runs",
		"SELECT status COLLATE read_csv('/etc/passwd') FROM runs",
		"SELECT run_id FROM runs ORDER BY status COLLATE 'nocase'",
	} {
		_, _, _, err := validate(mustParse(t, q))
		require.ErrorContains(t, err, "unsupported COLLATE expression", q)
	}
}

// sessions.key on a COLLATE's right is a collation name, not an
// array-of-struct access to rewrite.
func TestRewriteArrayOfStructAccessSkipsCollation(t *testing.T) {
	tr, err := Transpile("SELECT status COLLATE sessions.key AS s FROM runs", testAccountID, testEnvID)
	require.NoError(t, err)
	require.Contains(t, tr.SQL, "status COLLATE sessions.key AS s")
}

func TestValidateStarExcludeQualified(t *testing.T) {
	for _, q := range []string{
		"SELECT * EXCLUDE (runs.run_id) FROM runs",
		`SELECT * EXCLUDE ("R"."run_id") FROM runs r`,
		"SELECT * EXCLUDE (e.run_id) FROM runs JOIN extended_trace_spans e ON runs.run_id = e.run_id",
		"SELECT r.* EXCLUDE (r.run_id) FROM runs r",
	} {
		_, _, _, err := validate(mustParse(t, q))
		require.NoError(t, err, q)
	}
	for _, q := range []string{
		// DuckDB: Column "events.id" in EXCLUDE list not found in runs.
		"SELECT runs.* EXCLUDE (events.id) FROM runs, events",
		"SELECT * EXCLUDE (runs.id) FROM runs, events",
		"SELECT * EXCLUDE (bogus.run_id) FROM runs",
		"SELECT * EXCLUDE (a.b.c) FROM runs",
	} {
		_, _, _, err := validate(mustParse(t, q))
		require.Error(t, err, q)
	}
}

// Each USING column must exist, unambiguously, on both sides of its join
// (DuckDB's own binder errors).
func TestValidateChecksUsingColumns(t *testing.T) {
	for q, msg := range map[string]string{
		"SELECT * FROM runs JOIN events USING (bogus)":                                                         `column "bogus" does not exist on left side of join`,
		"SELECT * FROM runs JOIN events USING (run_id)":                                                        `column "run_id" does not exist on right side of join`,
		"SELECT * FROM runs r JOIN extended_trace_spans e ON r.run_id = e.run_id JOIN metadata USING (run_id)": `ambiguous column "run_id" on left side of join`,
		"SELECT * FROM runs r JOIN extended_trace_spans e ON r.run_id = e.run_id NATURAL JOIN metadata":        `ambiguous column "run_id" on left side of join`,
	} {
		_, _, _, err := validate(mustParse(t, q))
		require.ErrorContains(t, err, msg, q)
	}
	for _, q := range []string{
		// A merged column isn't ambiguous.
		"SELECT run_id, span_id FROM metadata JOIN extended_trace_spans USING (run_id, span_id)",
		"SELECT run_id FROM metadata m NATURAL JOIN extended_trace_spans e",
		"SELECT run_id FROM runs JOIN extended_trace_spans USING (RUN_ID) JOIN metadata USING (run_id)",
	} {
		_, _, _, err := validate(mustParse(t, q))
		require.NoError(t, err, q)
	}
}
