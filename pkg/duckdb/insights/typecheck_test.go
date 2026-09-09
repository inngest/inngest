package insights

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func typeOfFirstColumn(t *testing.T, sql string) ColumnType {
	t.Helper()
	stmt := mustParse(t, sql)
	scope, err := resolveScope(stmt.From, nil)
	require.NoError(t, err)
	return inferType(stmt.Columns[0].Expr, scope)
}

func TestInferTypeLiterals(t *testing.T) {
	require.Equal(t, ColumnTypeString, typeOfFirstColumn(t, "SELECT 'hello' FROM runs"))
	require.Equal(t, ColumnTypeNumber, typeOfFirstColumn(t, "SELECT 42 FROM runs"))
	require.Equal(t, ColumnTypeBoolean, typeOfFirstColumn(t, "SELECT TRUE FROM runs"))
	require.Equal(t, ColumnTypeBoolean, typeOfFirstColumn(t, "SELECT FALSE FROM runs"))
	require.Equal(t, ColumnTypeUnknown, typeOfFirstColumn(t, "SELECT NULL FROM runs"))
}

func TestInferTypeBareColumn(t *testing.T) {
	require.Equal(t, ColumnTypeString, typeOfFirstColumn(t, "SELECT run_id FROM runs"))
	require.Equal(t, ColumnTypeDatetime, typeOfFirstColumn(t, "SELECT queued_at FROM runs"))
	require.Equal(t, ColumnTypeBoolean, typeOfFirstColumn(t, "SELECT is_deferred FROM runs"))
	require.Equal(t, ColumnTypeJSON, typeOfFirstColumn(t, "SELECT inputs FROM runs"))
	require.Equal(t, ColumnTypeNumber, typeOfFirstColumn(t, "SELECT step_index FROM metadata"))
}

func TestInferTypeQualifiedColumn(t *testing.T) {
	require.Equal(t, ColumnTypeString, typeOfFirstColumn(t, "SELECT runs.run_id FROM runs"))
}

func TestInferTypeUnknownColumnIsUnknown(t *testing.T) {
	// Ambiguous across a join -> unresolvable, same as hints.
	require.Equal(t, ColumnTypeUnknown, typeOfFirstColumn(t,
		"SELECT run_id FROM runs JOIN extended_trace_spans ON runs.run_id = extended_trace_spans.run_id"))
}

func TestInferTypeCast(t *testing.T) {
	require.Equal(t, ColumnTypeString, typeOfFirstColumn(t, "SELECT CAST(step_index AS VARCHAR) FROM metadata"))
	require.Equal(t, ColumnTypeNumber, typeOfFirstColumn(t, "SELECT TRY_CAST(run_id AS INTEGER) FROM runs"))
}

func TestInferTypeCase(t *testing.T) {
	require.Equal(t, ColumnTypeString, typeOfFirstColumn(t, "SELECT CASE WHEN status = 'Completed' THEN 'done' ELSE 'pending' END FROM runs"))
}

func TestInferTypeCaseMismatchedBranchesIsUnknown(t *testing.T) {
	require.Equal(t, ColumnTypeUnknown, typeOfFirstColumn(t, "SELECT CASE WHEN status = 'Completed' THEN 'done' ELSE 1 END FROM runs"))
}

func TestInferTypeComparisonAndLogical(t *testing.T) {
	require.Equal(t, ColumnTypeBoolean, typeOfFirstColumn(t, "SELECT status = 'Completed' FROM runs"))
	require.Equal(t, ColumnTypeBoolean, typeOfFirstColumn(t, "SELECT status = 'Completed' AND app_id IS NOT NULL FROM runs"))
	require.Equal(t, ColumnTypeBoolean, typeOfFirstColumn(t, "SELECT run_id IS NULL FROM runs"))
	require.Equal(t, ColumnTypeBoolean, typeOfFirstColumn(t, "SELECT step_index BETWEEN 1 AND 5 FROM metadata"))
	require.Equal(t, ColumnTypeBoolean, typeOfFirstColumn(t, "SELECT status LIKE 'Comp%' FROM runs"))
}

func TestInferTypeArithmetic(t *testing.T) {
	require.Equal(t, ColumnTypeNumber, typeOfFirstColumn(t, "SELECT step_index + 1 FROM metadata"))
	require.Equal(t, ColumnTypeDatetime, typeOfFirstColumn(t, "SELECT queued_at + INTERVAL 1 DAY FROM runs"))
}

func TestInferTypeStringConcat(t *testing.T) {
	require.Equal(t, ColumnTypeString, typeOfFirstColumn(t, "SELECT run_id || '-suffix' FROM runs"))
}

func TestInferTypeJSONOperators(t *testing.T) {
	require.Equal(t, ColumnTypeString, typeOfFirstColumn(t, "SELECT attributes ->> '_inngest.function.id' FROM extended_trace_spans"))
	require.Equal(t, ColumnTypeJSON, typeOfFirstColumn(t, "SELECT attributes -> '_inngest.function.id' FROM extended_trace_spans"))
}

func TestInferTypeFunctions(t *testing.T) {
	cases := map[string]ColumnType{
		"SELECT COUNT(*) FROM runs":                    ColumnTypeNumber,
		"SELECT UPPER(run_id) FROM runs":               ColumnTypeString,
		"SELECT NOW() FROM runs":                       ColumnTypeDatetime,
		"SELECT JSON_VALID(inputs) FROM runs":          ColumnTypeBoolean,
		"SELECT TO_JSON(status) FROM runs":             ColumnTypeJSON,
		"SELECT MAX(step_index) FROM metadata":         ColumnTypeNumber,
		"SELECT GREATEST(status, 'unknown') FROM runs": ColumnTypeString,
	}
	for sql, want := range cases {
		t.Run(sql, func(t *testing.T) {
			require.Equal(t, want, typeOfFirstColumn(t, sql))
		})
	}
}

// TestInferTypeListAggregateResolvesLiteralAggregateName proves
// list_aggregate/array_aggregate's return type is only statically knowable
// when the aggregate name (args[1]) is a literal string constant --
// resolved by recursing into functionReturnType with that name, so it
// reports the exact type the named aggregate itself would.
func TestInferTypeListAggregateResolvesLiteralAggregateName(t *testing.T) {
	cases := map[string]ColumnType{
		// SUM/AVG/COUNT-family aggregates always return Number.
		"SELECT LIST_AGGREGATE(event_ids, 'count') FROM runs":  ColumnTypeNumber,
		"SELECT ARRAY_AGGREGATE(event_ids, 'sum') FROM runs":   ColumnTypeNumber,
		"SELECT LIST_AGGREGATE(event_ids, 'stddev') FROM runs": ColumnTypeNumber,
		// string_agg always returns String.
		"SELECT LIST_AGGREGATE(event_ids, 'string_agg') FROM runs": ColumnTypeString,
		// bool_and/bool_or always return Boolean.
		"SELECT LIST_AGGREGATE(event_ids, 'bool_and') FROM runs": ColumnTypeBoolean,
		// min/max/first/last preserve the list's own element type --
		// event_ids is JSON-bucketed (a VARCHAR[]).
		"SELECT LIST_AGGREGATE(event_ids, 'min') FROM runs": ColumnTypeJSON,
		// An unrecognized aggregate name is honestly unknown, not a guess.
		"SELECT LIST_AGGREGATE(event_ids, 'not_a_real_aggregate') FROM runs": ColumnTypeUnknown,
		// A dynamic (non-literal) name can't be resolved statically either.
		"SELECT LIST_AGGREGATE(event_ids, LOWER('SUM')) FROM runs": ColumnTypeUnknown,
	}
	for sql, want := range cases {
		t.Run(sql, func(t *testing.T) {
			require.Equal(t, want, typeOfFirstColumn(t, sql))
		})
	}
}

func TestInferTypeUnaryNot(t *testing.T) {
	require.Equal(t, ColumnTypeBoolean, typeOfFirstColumn(t, "SELECT NOT is_deferred FROM runs"))
}

func TestInferTypeListComprehension(t *testing.T) {
	// A list comprehension always produces a LIST, same JSON bucket as any
	// other composite type (ListExpr/StructExpr/MapExpr).
	require.Equal(t, ColumnTypeJSON, typeOfFirstColumn(t, "SELECT [e FOR e IN event_ids] FROM runs"))
}
