package insights

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/google/cel-go/common/operators"
	"github.com/inngest/expr"
	"github.com/inngest/inngest/pkg/expressions"
)

// CELEventFilters converts exprs (a run.ExpressionHandler.EventExprList)
// into SQL filter expressions against runsInputsCELScope (event.*,
// targeting a runs.inputs array element), keeping only the event.* leaf
// predicates of each expression — see celExprsToSQL's doc comment for why
// a mixed expression's output.*/error.* half is dropped here.
func CELEventFilters(ctx context.Context, exprs []string) ([]sq.Expression, error) {
	return celExprsToSQL(ctx, exprs, runsInputsCELScope, isEventPredicateIdent)
}

// CELOutputFilters is CELEventFilters' output.*/error.* counterpart — see
// its doc comment.
func CELOutputFilters(ctx context.Context, exprs []string) ([]sq.Expression, error) {
	return celExprsToSQL(ctx, exprs, runsInputsCELScope, isOutputPredicateIdent)
}

// CELEventTableFilters converts exprs into SQL filter expressions against
// eventsTableCELScope (event.*, targeting inngest.events' own columns
// directly), for CEL search over the events table itself rather than a
// run's triggering events. There's no output.*/error.* namespace for a
// bare event search, so any such predicate is silently dropped, same as an
// unrecognized field already is.
func CELEventTableFilters(ctx context.Context, exprs []string) ([]sq.Expression, error) {
	return celExprsToSQL(ctx, exprs, eventsTableCELScope, isEventPredicateIdent)
}

// celExprsToSQL parses each of exprs and converts it into SQL filter
// expressions via scope, keeping only the leaf predicates include accepts.
//
// This reimplements CEL-node-to-SQL conversion and AND/OR-folding rather
// than reusing pkg/run.ExpressionHandler.ToSQLFilters: a single CEL string
// can reference both event.* and output.*/error.* (e.g. "event.name == 'x'
// && output.ok == true"), and DuckDB's query needs to apply event.*
// predicates at a different point (a pre-collapse inputs-array match) than
// output.*/error.* predicates (a post-collapse QUALIFY clause) — two
// separate SQL fragments from the same mixed expression, which
// ToSQLFilters' shared union can't produce.
func celExprsToSQL(ctx context.Context, exprs []string, scope *celFieldScope, include func(ident string) bool) ([]sq.Expression, error) {
	filters := []sq.Expression{}
	parser := expressions.ParserSingleton()

	for _, exp := range exprs {
		tree, err := parser.Parse(ctx, expr.StringExpression(exp))
		if err != nil {
			return nil, fmt.Errorf("insights: parsing CEL expression %q: %w", exp, err)
		}
		// Callers reaching this via pkg/run.ExpressionHandler already reject
		// macros earlier, but this function re-parses independently with no
		// such gate of its own — check again so a macro expression fails
		// clearly here too.
		if tree.HasMacros {
			return nil, fmt.Errorf("insights: macros are not supported in CEL expression %q", exp)
		}

		nodeFilters, err := celNodesToSQL([]*expr.Node{&tree.Root}, scope, include)
		if err != nil {
			return nil, err
		}
		filters = append(filters, nodeFilters...)
	}

	return filters, nil
}

// celNodesToSQL walks nodes, converting each leaf predicate include accepts
// via celFieldConverter and folding And/Or children the same way the CEL
// tree itself combines them. A leaf include rejects is dropped from the
// result as if it didn't exist.
func celNodesToSQL(nodes []*expr.Node, scope *celFieldScope, include func(ident string) bool) ([]sq.Expression, error) {
	filters := []sq.Expression{}

	for _, n := range nodes {
		if n.HasPredicate() && include(n.Predicate.Ident) {
			res, err := celFieldConverter(n, scope)
			if err != nil {
				return nil, err
			}
			filters = append(filters, res...)
		}

		if n.Ands != nil {
			nested, err := celNodesToSQL(n.Ands, scope, include)
			if err != nil {
				return nil, err
			}
			switch len(nested) {
			case 0: // no op
			case 1:
				filters = append(filters, nested[0])
			default:
				filters = append(filters, sq.And(nested...))
			}
		}

		if n.Ors != nil {
			nested, err := celNodesToSQL(n.Ors, scope, include)
			if err != nil {
				return nil, err
			}
			switch len(nested) {
			case 0: // no op
			case 1:
				filters = append(filters, nested[0])
			default:
				filters = append(filters, sq.Or(nested...))
			}
		}
	}

	return filters, nil
}

func isEventPredicateIdent(ident string) bool {
	return strings.HasPrefix(ident, "event.")
}

func isOutputPredicateIdent(ident string) bool {
	return strings.HasPrefix(ident, "output.") || strings.HasPrefix(ident, "error.")
}

// celFieldConverter converts one CEL leaf predicate into DuckDB SQL by
// resolving it against scope and handing off to whatever handler it's
// registered to. An ident with no registration is silently dropped.
func celFieldConverter(n *expr.Node, scope *celFieldScope) ([]sq.Expression, error) {
	if !n.HasPredicate() {
		return []sq.Expression{}, nil
	}

	ident := n.Predicate.Ident
	handler, ok := scope.Get(strings.Split(ident, "."))
	if !ok {
		return []sq.Expression{}, nil
	}
	return handler(ident, n.Predicate.Literal, n.Predicate.Operator)
}

// handleJSONFilter creates SQL filters for JSON field access in DuckDB. ->
// and ->> both accept a JSONPath string directly (e.g. "$.a.b.c"),
// navigating every segment in one call — DuckDB (verified against v2.1.0)
// doesn't parse a chain of -> and ->> operators left-to-right the way most
// binary operators associate, so a chained path would otherwise need
// explicit parens around every hop. The whole expression is still wrapped
// in one outer paren pair: -> binds looser than =/!=/etc., so an
// unparenthesized comparison right after it (e.g. "expr->'$.a' = ?")
// associates as "expr->('$.a' = ?)" instead.
func handleJSONFilter(expr, fieldPath string, literal any, op string) ([]sq.Expression, error) {
	jsonPath := fmt.Sprintf("$.%s", fieldPath)
	jsonExpr := fmt.Sprintf("(%s->'%s')", expr, jsonPath)
	textExpr := fmt.Sprintf("(%s->>'%s')", expr, jsonPath)

	switch v := literal.(type) {
	case string:
		return handleStringOp(textExpr, v, op)
	case int64, float64:
		numExpr := fmt.Sprintf("CAST(%s AS DOUBLE)", textExpr)
		return handleNumericOp(numExpr, v, op)
	case bool:
		// DuckDB's ->> extracts a JSON boolean as the text "true"/"false".
		boolStr := "false"
		if v {
			boolStr = "true"
		}
		return handleStringOp(textExpr, boolStr, op)
	case nil:
		return handleNullOp(jsonExpr, op)
	default:
		return nil, fmt.Errorf("unsupported literal type: %T", literal)
	}
}

// handleNullOp creates SQL filters for JSON null comparison in DuckDB.
// json_type() returns SQL NULL both for a missing key and for an explicit
// JSON null (verified against v2.1.0), unlike SQLite/Postgres where it
// returns "null" only for an explicit JSON null — so it can't tell the two
// apart. Comparing the extracted JSON value directly against 'null'::JSON
// works instead, and keeps "missing key matches neither == null nor !=
// null" via SQL's three-valued logic (NULL = 'null'::JSON is NULL either
// way).
func handleNullOp(jsonExpr string, op string) ([]sq.Expression, error) {
	switch op {
	case operators.Equals:
		return []sq.Expression{sq.L(fmt.Sprintf("%s = 'null'::JSON", jsonExpr))}, nil
	case operators.NotEquals:
		return []sq.Expression{sq.L(fmt.Sprintf("%s != 'null'::JSON", jsonExpr))}, nil
	}
	return nil, fmt.Errorf("unsupported null operator: %s (only == and != are supported for null)", op)
}

func handleStringOp(expr string, value string, op string) ([]sq.Expression, error) {
	switch op {
	case operators.Equals:
		return []sq.Expression{sq.L(expr).Eq(value)}, nil
	case operators.NotEquals:
		return []sq.Expression{sq.L(expr).Neq(value)}, nil
	}
	return nil, fmt.Errorf("unsupported string operator: %s", op)
}

func handleNumericOp(expr string, value any, op string) ([]sq.Expression, error) {
	switch op {
	case operators.Equals:
		return []sq.Expression{sq.L(expr).Eq(value)}, nil
	case operators.NotEquals:
		return []sq.Expression{sq.L(expr).Neq(value)}, nil
	case operators.Greater:
		return []sq.Expression{sq.L(expr).Gt(value)}, nil
	case operators.GreaterEquals:
		return []sq.Expression{sq.L(expr).Gte(value)}, nil
	case operators.Less:
		return []sq.Expression{sq.L(expr).Lt(value)}, nil
	case operators.LessEquals:
		return []sq.Expression{sq.L(expr).Lte(value)}, nil
	}
	return nil, fmt.Errorf("unsupported numeric operator: %s", op)
}

// renderedWhereSQLPrefix is the fixed, deterministic prefix RenderWhereSQL's
// dialect-less "SELECT * WHERE ..." rendering always produces (no From/
// Select columns are ever set), used to strip the SELECT down to just the
// WHERE fragment.
const renderedWhereSQLPrefix = "SELECT * WHERE "

// RenderWhereSQL renders goqu filter expressions (as returned by
// CELEventFilters/CELOutputFilters) into a prepared-statement WHERE-clause
// fragment and its positional args, for callers like pkg/cqrs/duckdbquery
// that build queries as plain SQL strings/"?" args rather than through
// goqu end-to-end. Returns "", nil, nil for an empty filter set.
func RenderWhereSQL(filters []sq.Expression) (string, []any, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}

	sqlText, args, err := sq.Select().Prepared(true).Where(filters...).ToSQL()
	if err != nil {
		return "", nil, err
	}
	frag, ok := strings.CutPrefix(sqlText, renderedWhereSQLPrefix)
	if !ok {
		return "", nil, fmt.Errorf("insights: unexpected rendered WHERE SQL shape: %q", sqlText)
	}
	return frag, args, nil
}
