package run

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/google/cel-go/common/operators"
	"github.com/inngest/expr"
)

// EventFieldConverter creates filters for basic event field queries.
// This currently works in both SQLite and Postgres because the table structure is very similar despite slight differences in column types
// Supports: event.id, event.name, event.ts, event.v
//
// This applies to searching the events table, either by itself or when joined to runs for CEL search.
func EventFieldConverter(ctx context.Context, n *expr.Node) ([]sq.Expression, error) {
	if !n.HasPredicate() {
		return []sq.Expression{}, nil
	}

	literal := n.Predicate.Literal
	var f sq.Expression

	switch n.Predicate.Ident {
	case "event.id":
		id, ok := literal.(string)
		if !ok {
			return nil, fmt.Errorf("expects 'event.id' to be a string: %v", literal)
		}
		field := "event_id"
		switch n.Predicate.Operator {
		case operators.Equals:
			f = sq.C(field).Eq(id)
		case operators.NotEquals:
			f = sq.C(field).Neq(id)
		}

	case "event.name":
		name, ok := literal.(string)
		if !ok {
			return nil, fmt.Errorf("expects 'event.name' to be a string: %v", literal)
		}
		field := "event_name"
		switch n.Predicate.Operator {
		case operators.Equals:
			f = sq.C(field).Eq(name)
		case operators.NotEquals:
			f = sq.C(field).Neq(name)
		}

	case "event.ts":
		ts, ok := literal.(int64)
		if !ok {
			return nil, fmt.Errorf("expects 'event.ts' to be an integer: %v", literal)
		}
		field := "event_ts"
		switch n.Predicate.Operator {
		case operators.Greater:
			f = sq.C(field).Gt(ts)
		case operators.GreaterEquals:
			f = sq.C(field).Gte(ts)
		case operators.Equals:
			f = sq.C(field).Eq(ts)
		case operators.Less:
			f = sq.C(field).Lt(ts)
		case operators.LessEquals:
			f = sq.C(field).Lte(ts)
		case operators.NotEquals:
			f = sq.C(field).Neq(ts)
		}

	case "event.v":
		v, ok := literal.(string)
		if !ok {
			return nil, fmt.Errorf("expects 'event.v' to be a string: %v", literal)
		}
		field := "event_v"
		switch n.Predicate.Operator {
		case operators.Equals:
			f = sq.C(field).Eq(v)
		case operators.NotEquals:
			f = sq.C(field).Neq(v)
		}
	}

	if f != nil {
		return []sq.Expression{f}, nil
	}
	return []sq.Expression{}, nil
}

// dbDialect contains database-specific SQL generation settings for span+event queries.
// These are mainly for differences in null handing, casting and json behavior across SQL dialects
// TODO: if we require minimum of Postgres 17 for self hosted, we can possibly collapse more of these
// using SQL standard JSON such as json_value
type dbDialect struct {
	eventDataExpr   string // Expression for event.data.* queries
	outputDataExpr  string // Expression for output.* queries (points to spans.output.data subtree)
	outputErrorExpr string // Expression for error.* queries (points to spans.output.error subtree)

	handleJSONFilter func(expr, fieldPath string, literal interface{}, op string) ([]sq.Expression, error)
}

var sqliteDialect = dbDialect{
	eventDataExpr:    "NULLIF(events.event_data, '')",
	outputDataExpr:   "json_extract(spans.output, '$.data')",
	outputErrorExpr:  "json_extract(spans.output, '$.error')",
	handleJSONFilter: handleSQLiteJSONFilter,
}

var postgresDialect = dbDialect{
	eventDataExpr: "(NULLIF(events.event_data, '')::jsonb)",
	// although spans.output is a JSONB, we are storing stringified JSON in there, so we have to
	// extract that string then cast it back to jsonb to parse it as a JSON
	outputDataExpr:   "((spans.output#>>'{}')::jsonb->'data')",
	outputErrorExpr:  "((spans.output#>>'{}')::jsonb->'error')",
	handleJSONFilter: handlePostgresJSONFilter,
}

// SpanEventSQLiteConverter and SpanEventPostgresConverter convert CEL to SQL for runs CEL searches
// Supports: event.*, event.data.*, output.*, error.*

func SpanEventSQLiteConverter(ctx context.Context, n *expr.Node) ([]sq.Expression, error) {
	return spanEventConverter(ctx, n, sqliteDialect)
}

func SpanEventPostgresConverter(ctx context.Context, n *expr.Node) ([]sq.Expression, error) {
	return spanEventConverter(ctx, n, postgresDialect)
}

func spanEventConverter(ctx context.Context, n *expr.Node, dialect dbDialect) ([]sq.Expression, error) {
	if !n.HasPredicate() {
		return []sq.Expression{}, nil
	}

	literal := n.Predicate.Literal
	ident := n.Predicate.Ident
	op := n.Predicate.Operator

	// Handle output.* → searches spans.output.data
	if strings.HasPrefix(ident, "output.") {
		fieldPath := strings.TrimPrefix(ident, "output.")
		return dialect.handleJSONFilter(dialect.outputDataExpr, fieldPath, literal, op)
	}

	// Handle error.* → searches spans.output.error
	if strings.HasPrefix(ident, "error.") {
		fieldPath := strings.TrimPrefix(ident, "error.")
		return dialect.handleJSONFilter(dialect.outputErrorExpr, fieldPath, literal, op)
	}

	// Handle event.data.* (JSON extraction from events table)
	if strings.HasPrefix(ident, "event.data.") {
		fieldPath := strings.TrimPrefix(ident, "event.data.")
		return dialect.handleJSONFilter(dialect.eventDataExpr, fieldPath, literal, op)
	}

	// Fallback to base converter for other event.* fields, which have no db specific JSON handling
	return EventFieldConverter(ctx, n)
}

// handleSQLiteJSONFilter creates SQL filters for JSON field access in SQLite.
// Uses json_extract with dot-notation paths (e.g., $.field.subfield).
func handleSQLiteJSONFilter(expr, fieldPath string, literal interface{}, op string) ([]sq.Expression, error) {
	jsonPath := fmt.Sprintf("$.%s", fieldPath)
	jsonExpr := fmt.Sprintf("json_extract(%s, '%s')", expr, jsonPath)

	switch v := literal.(type) {
	case string:
		return handleStringOp(jsonExpr, v, op)
	case int64, float64:
		numExpr := fmt.Sprintf("CAST(%s AS NUMERIC)", jsonExpr)
		return handleNumericOp(numExpr, v, op)
	default:
		return nil, fmt.Errorf("unsupported literal type: %T", literal)
	}
}

// handlePostgresJSONFilter creates SQL filters for JSON field access in PostgreSQL.
// Uses #>> operator with array path notation (e.g., {field,subfield}).
func handlePostgresJSONFilter(expr, fieldPath string, literal interface{}, op string) ([]sq.Expression, error) {
	// Postgres: use #>> for path traversal to handle nested fields (e.g., "foo.bar" -> '{foo,bar}')
	pathParts := strings.Split(fieldPath, ".")
	pgPath := fmt.Sprintf("{%s}", strings.Join(pathParts, ","))
	jsonExpr := fmt.Sprintf("%s#>>'%s'", expr, pgPath)

	switch v := literal.(type) {
	case string:
		return handleStringOp(jsonExpr, v, op)
	case int64, float64:
		numExpr := fmt.Sprintf("(%s)::numeric", jsonExpr)
		return handleNumericOp(numExpr, v, op)
	case bool:
		// PostgreSQL JSONB stores booleans as 'true'/'false' strings when extracted with #>>
		boolStr := "false"
		if v {
			boolStr = "true"
		}
		return handleStringOp(jsonExpr, boolStr, op)
	default:
		return nil, fmt.Errorf("unsupported literal type: %T", literal)
	}
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

func handleNumericOp(expr string, value interface{}, op string) ([]sq.Expression, error) {
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
