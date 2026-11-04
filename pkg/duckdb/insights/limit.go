package insights

import (
	"strconv"
	"strings"

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
		return limitUnchanged, &ValidationError{Pos: stmt.Limit.Pos(), End: stmt.Limit.End(), Message: "PERCENT-based LIMIT is not supported"}
	}

	// "OFFSET n" with no LIMIT parses as a LimitClause whose Limit is nil
	// -- just as unbounded as no clause at all, so it gets the same
	// default, keeping the user's OFFSET.
	if stmt.Limit.Limit == nil && !stmt.Limit.All {
		stmt.Limit.Limit = &parser.Literal{Kind: parser.LitNumber, Text: strconv.Itoa(defaultInsightsLimit)}
		return limitDefaulted, nil
	}

	if stmt.Limit.All {
		stmt.Limit.All = false
		stmt.Limit.Limit = &parser.Literal{Kind: parser.LitNumber, Text: strconv.Itoa(defaultInsightsLimit)}
		return limitCapped, nil
	}

	lit, ok := stmt.Limit.Limit.(*parser.Literal)
	// A computed expression (a parameter, arithmetic, a subquery, ...) --
	// this package has no static way to bound its runtime value, so it's
	// rejected rather than reaching DuckDB uncapped. So is a non-integer
	// number literal ("1e9", "10.5"), which DuckDB accepts but a plain
	// integer parse can't bound.
	if !ok || lit.Kind != parser.LitNumber {
		return limitUnchanged, &ValidationError{Pos: stmt.Limit.Pos(), End: stmt.Limit.End(), Message: "LIMIT must be a literal, non-negative integer"}
	}
	digits, ok := limitDigits(lit.Text)
	if !ok {
		return limitUnchanged, &ValidationError{Pos: stmt.Limit.Pos(), End: stmt.Limit.End(), Message: "LIMIT must be a literal, non-negative integer"}
	}

	// A digit string too large for uint64 (ErrRange) is still an integer
	// above the ceiling, so it's capped like any other.
	n, err := strconv.ParseUint(digits, 10, 64)
	if err == nil && n <= defaultInsightsLimit {
		return limitUnchanged, nil
	}

	lit.Text = strconv.Itoa(defaultInsightsLimit)
	return limitCapped, nil
}

// limitDigits returns an integer literal's digits with any `_` separators
// (`1_000`) dropped, or false if s isn't a plain integer. DuckDB only lexes
// a `_` between two digits as a separator (`1__0` is `1 AS __0`, `1_` is
// `1 AS _`), so anywhere else it's rejected rather than guessed at.
func limitDigits(s string) (string, bool) {
	if !strings.Contains(s, "_") {
		return s, isDecimalDigits(s)
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '_' && (i == 0 || i == len(s)-1 || !isDigit(s[i-1]) || !isDigit(s[i+1])) {
			return "", false
		}
	}
	digits := strings.ReplaceAll(s, "_", "")
	return digits, isDecimalDigits(digits)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isDecimalDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
