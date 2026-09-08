package insights

import (
	"strconv"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

const defaultInsightsLimit = 1000

// limitOutcome reports what, if anything, addDefaultLimit changed about
// stmt's LIMIT clause, so stageAddDefaultLimit (transpile.go) can report an
// accurate diagnostic message -- "no LIMIT given" and "LIMIT too high" are
// different situations for a user reading it.
type limitOutcome int

const (
	limitUnchanged limitOutcome = iota
	limitDefaulted
	limitCapped
)

// addDefaultLimit enforces defaultInsightsLimit as both stmt's default and
// its hard ceiling -- this is the one place standing between a query and
// unbounded row materialization in Execute, so a user-supplied LIMIT is
// clamped down just as much as a missing one is filled in. UNION/INTERSECT/
// EXCEPT carry their LIMIT on the top-level combined SelectStatement, never
// per-operand, so only stmt itself (never SetLeft/SetRight) needs checking.
// A CTE or subquery's own LIMIT is left untouched -- it never directly
// bounds the client-visible result set, only the query's own top-level
// LIMIT does.
func addDefaultLimit(stmt *parser.SelectStatement) (limitOutcome, error) {
	if stmt.Limit == nil {
		stmt.Limit = &parser.LimitClause{
			Limit: &parser.Literal{Kind: parser.LitNumber, Text: strconv.Itoa(defaultInsightsLimit)},
		}
		return limitDefaulted, nil
	}

	// "LIMIT n PERCENT" bounds a fraction of the result set, not an
	// absolute row count -- there's no static row cap to compare n
	// against, so this shape is rejected outright rather than silently
	// let through uncapped (matching this package's own precedent: see
	// pkg/duckdb/insights/CLAUDE.md's "don't assume the database will
	// catch it anyway" gotcha).
	if stmt.Limit.Percent {
		return limitUnchanged, &ValidationError{Pos: stmt.Limit.Pos(), Message: "PERCENT-based LIMIT is not supported"}
	}

	if stmt.Limit.All {
		stmt.Limit.All = false
		stmt.Limit.Limit = &parser.Literal{Kind: parser.LitNumber, Text: strconv.Itoa(defaultInsightsLimit)}
		return limitCapped, nil
	}

	lit, ok := stmt.Limit.Limit.(*parser.Literal)
	if !ok || lit.Kind != parser.LitNumber {
		// A computed expression (a parameter, arithmetic, a subquery, ...)
		// -- this package has no static way to bound its runtime value, so
		// it's rejected rather than reaching DuckDB uncapped.
		return limitUnchanged, &ValidationError{Pos: stmt.Limit.Pos(), Message: "LIMIT must be a literal, non-negative integer"}
	}

	n, err := strconv.Atoi(lit.Text)
	if err != nil || n <= defaultInsightsLimit {
		return limitUnchanged, nil
	}

	lit.Text = strconv.Itoa(defaultInsightsLimit)
	return limitCapped, nil
}
