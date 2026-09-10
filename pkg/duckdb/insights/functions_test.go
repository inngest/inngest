package insights

import (
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/parser"
	"github.com/stretchr/testify/require"
)

// dummyArgs returns n placeholder expressions -- withArity only ever looks
// at len(args), never their contents, so a literal's actual value doesn't
// matter for these tests.
func dummyArgs(n int) []parser.Expr {
	args := make([]parser.Expr, n)
	for i := range args {
		args[i] = &parser.Literal{Kind: parser.LitNumber, Text: "1"}
	}
	return args
}

// TestWithArityBounds tests the wrapper's own boundary arithmetic in
// isolation from any real function's returnType body or any DuckDB
// semantics -- next is never invoked when the count is out of range, and
// always invoked (with a real ColumnType/nil diagnostics reply) when in
// range.
func TestWithArityBounds(t *testing.T) {
	var calledWith []parser.Expr
	next := func(_ string, args []parser.Expr, _ *tableScope) (ColumnType, []Diagnostic) {
		calledWith = args
		return ColumnTypeNumber, nil
	}

	t.Run("fewer than min rejects without calling next", func(t *testing.T) {
		calledWith = nil
		wrapped := withArity(2, 4, next)
		ct, diags := wrapped("foo", dummyArgs(1), nil)
		require.Equal(t, ColumnTypeUnknown, ct)
		require.Len(t, diags, 1)
		require.Equal(t, DiagnosticError, diags[0].Severity)
		require.Equal(t, "wrong-argument-count", diags[0].Code)
		require.Contains(t, diags[0].Message, "FOO")
		require.Contains(t, diags[0].Message, "at least 2")
		require.Contains(t, diags[0].Message, "got 1")
		require.Nil(t, calledWith, "next must not be called when arity is violated")
	})

	t.Run("more than max rejects without calling next", func(t *testing.T) {
		calledWith = nil
		wrapped := withArity(1, 2, next)
		ct, diags := wrapped("bar", dummyArgs(3), nil)
		require.Equal(t, ColumnTypeUnknown, ct)
		require.Len(t, diags, 1)
		require.Equal(t, DiagnosticError, diags[0].Severity)
		require.Contains(t, diags[0].Message, "BAR")
		require.Contains(t, diags[0].Message, "at most 2")
		require.Contains(t, diags[0].Message, "got 3")
		require.Nil(t, calledWith)
	})

	t.Run("within bounds calls next", func(t *testing.T) {
		calledWith = nil
		wrapped := withArity(1, 3, next)
		ct, diags := wrapped("baz", dummyArgs(2), nil)
		require.Equal(t, ColumnTypeNumber, ct)
		require.Empty(t, diags)
		require.Len(t, calledWith, 2)
	})

	t.Run("at exactly min and exactly max both succeed", func(t *testing.T) {
		wrapped := withArity(2, 2, next)
		ct, diags := wrapped("exact", dummyArgs(2), nil)
		require.Equal(t, ColumnTypeNumber, ct)
		require.Empty(t, diags)
	})

	t.Run("negative max means unbounded -- no upper rejection ever", func(t *testing.T) {
		wrapped := withArity(1, -1, next)
		ct, diags := wrapped("unbounded", dummyArgs(50), nil)
		require.Equal(t, ColumnTypeNumber, ct)
		require.Empty(t, diags)
	})
}

// TestValidateRejectsWrongArgumentCount exercises withArity through the
// real pipeline (validate), for a sample spanning every distinct bound
// shape allowedFunctions actually uses: a plain min/max pair
// (fixedType-wrapped scalar), an unbounded max (varargs), a zero-arg
// window function, a bounded-but->1 window function whose real minimum
// required hand-verification (see withArity's own doc comment on why),
// and the two special exact-arity forms (IF, UNNEST).
func TestValidateRejectsWrongArgumentCount(t *testing.T) {
	cases := []struct {
		name       string
		sql        string
		wantErrSub string
	}{
		{"abs with 0 args", "SELECT abs() FROM runs", "at least 1"},
		{"abs with 2 args", "SELECT abs(1, 2) FROM runs", "at most 1"},
		{"concat_ws with 1 arg", "SELECT concat_ws(',') FROM runs", "at least 2"},
		{"count_if with 2 args", "SELECT count_if(is_deferred, is_deferred) FROM runs", "at most 1"},
		{"row_number with 1 arg", "SELECT row_number(1) OVER (ORDER BY queued_at) FROM runs", "at most 0"},
		{"lag with 0 args", "SELECT lag() OVER (ORDER BY queued_at) FROM runs", "at least 1"},
		{"lag with 4 args", "SELECT lag(status, 1, 'x', 'y') OVER (ORDER BY queued_at) FROM runs", "at most 3"},
		{"if with 2 args", "SELECT if(true, 1) FROM runs", "at least 3"},
		{"if with 4 args", "SELECT if(true, 1, 2, 3) FROM runs", "at most 3"},
		{"unnest with 2 args", "SELECT unnest(event_ids, event_ids) FROM runs", "at most 1"},
		{"list_aggregate with 1 arg", "SELECT list_aggregate(event_ids) FROM runs", "at least 2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParse(t, tc.sql)
			_, _, _, err := validate(stmt)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErrSub)

			verr, ok := err.(*ValidationError)
			require.True(t, ok, "must be a *ValidationError, not any other error type")
			require.NotEqual(t, verr.Pos, parser.Position{}, "must anchor to a real position, not the zero value")
		})
	}
}

// TestValidateAcceptsBoundaryArgumentCounts is TestValidateRejectsWrongArgumentCount's
// mirror: the exact min/max boundary itself, and a representative
// mid-range/unbounded call, must all still succeed.
func TestValidateAcceptsBoundaryArgumentCounts(t *testing.T) {
	queries := []string{
		"SELECT abs(step_index) FROM steps",                              // exactly min (1) and max (1)
		"SELECT concat_ws(',', run_id, app_id) FROM runs",                // exactly min (2)
		"SELECT concat_ws(',', run_id, app_id, function_id) FROM runs",   // above min, unbounded max
		"SELECT row_number() OVER (ORDER BY queued_at) FROM runs",        // exactly 0 args
		"SELECT lag(status) OVER (ORDER BY queued_at) FROM runs",         // exactly min (1)
		"SELECT lag(status, 1, 'x') OVER (ORDER BY queued_at) FROM runs", // exactly max (3)
		"SELECT if(status = 'Completed', run_id, app_id) FROM runs",      // exactly 3
		"SELECT unnest(event_ids) FROM runs",                             // exactly 1
		"SELECT list_aggregate(event_ids, 'count') FROM runs",            // exactly min (2)
		"SELECT list_aggregate(event_ids, 'string_agg', ',') FROM runs",  // above min, unbounded max
	}
	for _, sql := range queries {
		t.Run(sql, func(t *testing.T) {
			stmt := mustParse(t, sql)
			_, _, _, err := validate(stmt)
			require.NoError(t, err)
		})
	}
}

// TestValidateRejectsIfConditionTypeMismatch/TestValidateAcceptsIfConditionVariants
// cover ifReturnType's condition-type check: only a STRING *literal* whose
// text doesn't parse as a DuckDB boolean is rejected (confirmed
// empirically against a live duckdb -- see ifReturnType's own doc
// comment); a column/dynamic expression is never rejected regardless of
// its declared type, since its runtime value can't be checked statically.
func TestValidateRejectsIfConditionTypeMismatch(t *testing.T) {
	cases := []string{
		"SELECT if('a', 1, 2) FROM runs",
		"SELECT if('maybe', 1, 2) FROM runs",
		"SELECT if('2', 1, 2) FROM runs",
		"SELECT if('on', 1, 2) FROM runs",
		"SELECT if('', 1, 2) FROM runs",
	}
	for _, sql := range cases {
		t.Run(sql, func(t *testing.T) {
			stmt := mustParse(t, sql)
			_, _, _, err := validate(stmt)
			require.Error(t, err)
			require.Contains(t, err.Error(), "IF's argument 1 expects BOOLEAN, got STRING")
		})
	}
}

func TestValidateAcceptsIfConditionVariants(t *testing.T) {
	cases := []string{
		"SELECT if('true', 1, 2) FROM runs",
		"SELECT if('FALSE', 1, 2) FROM runs",
		"SELECT if('t', 1, 2) FROM runs",
		"SELECT if('yes', 1, 2) FROM runs",
		"SELECT if('y', 1, 2) FROM runs",
		"SELECT if('1', 1, 2) FROM runs",
		"SELECT if(1, 'a', 'b') FROM runs",       // NUMBER condition: DuckDB truthy/falsy, no error
		"SELECT if(NULL, 'a', 'b') FROM runs",    // NULL condition: takes else branch, no error
		"SELECT if(is_deferred, 1, 2) FROM runs", // dynamic (column) condition: never checked statically
	}
	for _, sql := range cases {
		t.Run(sql, func(t *testing.T) {
			stmt := mustParse(t, sql)
			_, _, _, err := validate(stmt)
			require.NoError(t, err)
		})
	}
}

// TestValidateRejectsListAggregateNameTypeMismatch/
// TestValidateAcceptsListAggregateNameVariants cover
// listAggregateReturnType's own name-argument check: a *literal* of the
// wrong kind is rejected outright (confirmed empirically -- no DuckDB
// overload of list_aggregate accepts a non-VARCHAR second argument), but a
// dynamic (non-literal) name or an unrecognized-but-string-shaped name
// stays accepted, since this package's own allowlist isn't necessarily
// DuckDB's complete aggregate list.
func TestValidateRejectsListAggregateNameTypeMismatch(t *testing.T) {
	cases := []struct {
		sql      string
		wantType string
	}{
		{"SELECT list_aggregate(event_ids, 123) FROM runs", "NUMBER"},
		{"SELECT list_aggregate(event_ids, true) FROM runs", "BOOLEAN"},
	}
	for _, tc := range cases {
		t.Run(tc.sql, func(t *testing.T) {
			stmt := mustParse(t, tc.sql)
			_, _, _, err := validate(stmt)
			require.Error(t, err)
			require.Contains(t, err.Error(), "LIST_AGGREGATE's argument 2 expects STRING, got "+tc.wantType)
		})
	}
}

func TestValidateAcceptsListAggregateNameVariants(t *testing.T) {
	cases := []string{
		"SELECT list_aggregate(event_ids, 'count') FROM runs",
		"SELECT list_aggregate(event_ids, 'not_a_real_aggregate') FROM runs", // unrecognized but string-shaped: stays accepted
		"SELECT list_aggregate(event_ids, LOWER('SUM')) FROM runs",           // dynamic name: can't be checked statically
	}
	for _, sql := range cases {
		t.Run(sql, func(t *testing.T) {
			stmt := mustParse(t, sql)
			_, _, _, err := validate(stmt)
			require.NoError(t, err)
		})
	}
}
