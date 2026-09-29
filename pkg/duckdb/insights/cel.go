package insights

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/google/cel-go/common/operators"
	"github.com/inngest/expr"
	"github.com/inngest/inngest/pkg/expressions"
)

// CELEventFilters converts exprs (a run.ExpressionHandler.EventExprList)
// into SQL filter expressions against runsInputsCELScope (event.*,
// targeting a runs.inputs array element), for the pre-collapse half of
// pkg/duckdb/query's GetTraceRuns filter. It pairs with CELOutputFilters:
// ANDing both halves' results selects exactly what each original
// expression means — see celExprsToSQL's doc comment for how a mixed
// expression is divided between them.
func CELEventFilters(ctx context.Context, exprs []string) ([]sq.Expression, error) {
	return celExprsToSQL(ctx, exprs, func(root *expr.Node) (sq.Expression, error) {
		var pushable []*expr.Node
		for _, c := range celConjuncts(root) {
			if !celHasOutputLeaf(c) {
				pushable = append(pushable, c)
			}
		}
		return celNodeToSQL(&expr.Node{Ands: pushable}, runsInputsCELScope, isEventPredicateIdent)
	})
}

// CELOutputFilters is CELEventFilters' post-collapse (QUALIFY)
// counterpart, converting exprs (a run.ExpressionHandler.OutputExprList)
// — see CELEventFilters' doc comment.
func CELOutputFilters(ctx context.Context, exprs []string) ([]sq.Expression, error) {
	return celExprsToSQL(ctx, exprs, func(root *expr.Node) (sq.Expression, error) {
		var rest []*expr.Node
		for _, c := range celConjuncts(root) {
			if celHasOutputLeaf(c) {
				rest = append(rest, c)
			}
		}
		return celPostCollapseToSQL(rest)
	})
}

// CELEventTableFilters converts exprs into SQL filter expressions against
// eventsTableCELScope (event.*, targeting inngest.events' own columns
// directly), for CEL search over the events table itself rather than a
// run's triggering events. There's no output.*/error.* namespace for a
// bare event search, so any such predicate is treated as unconstrained,
// same as an unrecognized field already is (see celNodeToSQL).
func CELEventTableFilters(ctx context.Context, exprs []string) ([]sq.Expression, error) {
	return celExprsToSQL(ctx, exprs, func(root *expr.Node) (sq.Expression, error) {
		return celNodeToSQL(root, eventsTableCELScope, isEventPredicateIdent)
	})
}

// celExprsToSQL parses each of exprs and converts its root node via
// convert, returning one filter per expression that constrains anything.
//
// This reimplements CEL-node-to-SQL conversion and AND/OR-folding rather
// than reusing pkg/run.ExpressionHandler.ToSQLFilters: a single CEL string
// can reference both event.* and output.*/error.* (e.g. "event.name == 'x'
// && output.ok == true"), and DuckDB's query needs to apply event.*
// predicates at a different point (a pre-collapse inputs-array match) than
// output.*/error.* predicates (a post-collapse QUALIFY clause) — two
// separate SQL fragments from the same mixed expression, which
// ToSQLFilters' shared union can't produce.
//
// The expression is only ever divided along its top-level conjuncts
// (celConjuncts): each conjunct with no output.*/error.* leaf goes to
// CELEventFilters, every other one to CELOutputFilters. Dividing anywhere
// else would change its meaning — "event.name == 'x' || output.ok == true"
// can't be applied as an event half ANDed with an output half. A conjunct
// mixing both namespaces is instead evaluated whole, post-collapse, with
// its event.* parts matched against the run's inputs array there
// (celPostCollapseToSQL). A run's inputs are the same on every one of its
// lifecycle rows, so an event.* predicate means the same thing on either
// side of the collapse.
func celExprsToSQL(ctx context.Context, exprs []string, convert func(root *expr.Node) (sq.Expression, error)) ([]sq.Expression, error) {
	filters := []sq.Expression{}
	parser := expressions.ParserSingleton()

	for _, celExpr := range exprs {
		tree, err := parser.Parse(ctx, expr.StringExpression(celExpr))
		if err != nil {
			return nil, fmt.Errorf("insights: parsing CEL expression %q: %w", celExpr, err)
		}
		// Callers reaching this via pkg/run.ExpressionHandler already reject
		// macros earlier, but this function re-parses independently with no
		// such gate of its own — check again so a macro expression fails
		// clearly here too.
		if tree.HasMacros {
			return nil, fmt.Errorf("insights: macros are not supported in CEL expression %q", celExpr)
		}

		filter, err := convert(&tree.Root)
		if err != nil {
			return nil, err
		}
		if filter != nil {
			filters = append(filters, filter)
		}
	}

	return filters, nil
}

// celNodeToSQL converts n into one SQL expression, following expr.Node's
// own semantics: n holds when its Predicate, every Ands child, and at
// least one Ors child all hold. Only leaf predicates include accepts are
// converted; any other leaf (or one with no registration in scope) is
// treated as unconstrained, so it can only ever widen the filter, never
// narrow it. Returns nil when n doesn't constrain anything.
func celNodeToSQL(n *expr.Node, scope *celFieldScope, include func(ident string) bool) (sq.Expression, error) {
	var conj []sq.Expression

	if n.HasPredicate() && include(n.Predicate.Ident) {
		res, err := celFieldConverter(n, scope)
		if err != nil {
			return nil, err
		}
		conj = append(conj, res...)
	}

	for _, c := range n.Ands {
		e, err := celNodeToSQL(c, scope, include)
		if err != nil {
			return nil, err
		}
		if e != nil {
			conj = append(conj, e)
		}
	}

	if len(n.Ors) > 0 {
		e, err := celOrToSQL(n.Ors, func(c *expr.Node) (sq.Expression, error) {
			return celNodeToSQL(c, scope, include)
		})
		if err != nil {
			return nil, err
		}
		if e != nil {
			conj = append(conj, e)
		}
	}

	return celAnd(conj), nil
}

// celOrToSQL converts each of nodes via convert and ORs the results. One
// unconstrained branch makes the whole group unconstrained — dropping just
// that branch would instead narrow the group to its other branches.
func celOrToSQL(nodes []*expr.Node, convert func(*expr.Node) (sq.Expression, error)) (sq.Expression, error) {
	disj := make([]sq.Expression, 0, len(nodes))
	unconstrained := false
	for _, c := range nodes {
		e, err := convert(c)
		if err != nil {
			return nil, err
		}
		if e == nil {
			unconstrained = true
			continue
		}
		disj = append(disj, e)
	}
	switch {
	case unconstrained || len(disj) == 0:
		return nil, nil
	case len(disj) == 1:
		return disj[0], nil
	default:
		return sq.Or(disj...), nil
	}
}

func celAnd(conj []sq.Expression) sq.Expression {
	switch len(conj) {
	case 0:
		return nil
	case 1:
		return conj[0]
	default:
		return sq.And(conj...)
	}
}

// celConjuncts flattens n into the nodes it ANDs together: its Predicate
// (as a leaf node), each Ands child's own conjuncts, and its Ors group (as
// a node holding only those Ors). Every returned node is therefore either
// a leaf or an Ors-only group.
func celConjuncts(n *expr.Node) []*expr.Node {
	var out []*expr.Node
	if n.HasPredicate() {
		out = append(out, &expr.Node{Predicate: n.Predicate})
	}
	for _, c := range n.Ands {
		out = append(out, celConjuncts(c)...)
	}
	if len(n.Ors) > 0 {
		out = append(out, &expr.Node{Ors: n.Ors})
	}
	return out
}

func celHasOutputLeaf(n *expr.Node) bool {
	if n.HasPredicate() && isOutputPredicateIdent(n.Predicate.Ident) {
		return true
	}
	for _, c := range n.Ands {
		if celHasOutputLeaf(c) {
			return true
		}
	}
	for _, c := range n.Ors {
		if celHasOutputLeaf(c) {
			return true
		}
	}
	return false
}

// celPostCollapseToSQL converts conjuncts (as celConjuncts returns them)
// for evaluation after a run's rows collapse, where output.*/error.* is a
// plain column but event.* has to be matched against the run's inputs
// array: the event-only conjuncts are grouped into one inputs-array match
// (so, as in CELEventFilters, they must all hold for the same triggering
// event), and every other conjunct is converted recursively.
func celPostCollapseToSQL(conjuncts []*expr.Node) (sq.Expression, error) {
	var (
		conj      []sq.Expression
		eventOnly []*expr.Node
	)
	for _, c := range conjuncts {
		if !celHasOutputLeaf(c) {
			eventOnly = append(eventOnly, c)
			continue
		}
		var (
			e   sq.Expression
			err error
		)
		if c.HasPredicate() {
			e, err = celNodeToSQL(c, runsInputsCELScope, isOutputPredicateIdent)
		} else {
			e, err = celOrToSQL(c.Ors, func(n *expr.Node) (sq.Expression, error) {
				return celPostCollapseToSQL(celConjuncts(n))
			})
		}
		if err != nil {
			return nil, err
		}
		if e != nil {
			conj = append(conj, e)
		}
	}

	if len(eventOnly) > 0 {
		e, err := celNodeToSQL(&expr.Node{Ands: eventOnly}, runsInputsCELScope, isEventPredicateIdent)
		if err != nil {
			return nil, err
		}
		if e != nil {
			conj = append(conj, sq.L(runsInputsMatchSQL, e))
		}
	}

	return celAnd(conj), nil
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
//
// expr's own ::JSON cast (harmless/a no-op when expr is already JSON, as
// with runsInputsCELScope's "x" lambda parameter — already the result of
// json_transform — but load-bearing for eventsTableCELScope's event_data,
// a VARIANT column) exists because -> and ->> only extract from JSON: given
// a VARIANT operand instead, DuckDB silently stringifies it first with its
// own single-quoted struct syntax (e.g. `{'foo': bar}`, not
// `{"foo":"bar"}`) rather than erroring, so the extraction then re-parses
// that as JSON and fails with a "Malformed JSON at byte 1" error — see
// pkg/execution/dualwrite/tracing.go's materializeRuns for the same
// underlying VARIANT/JSON distinction on the write side.
//
// The JSONPath is bound as a query parameter rather than spliced into the
// SQL text: a CEL string index (e.g. event.data.foo["a'b"]) carries an
// arbitrary user string into fieldPath, so it must never reach the SQL as
// a literal.
func handleJSONFilter(expr, fieldPath string, literal any, op string) ([]sq.Expression, error) {
	expr = fmt.Sprintf("%s::JSON", expr)
	jsonPath := celJSONPath(fieldPath)
	textExpr := fmt.Sprintf("(%s->>?)", expr)

	switch v := literal.(type) {
	case string:
		return handleStringOp(sq.L(textExpr, jsonPath), v, op)
	case int64, float64:
		return handleNumericOp(sq.L(fmt.Sprintf("CAST(%s AS DOUBLE)", textExpr), jsonPath), v, op)
	case bool:
		// DuckDB's ->> extracts a JSON boolean as the text "true"/"false".
		boolStr := "false"
		if v {
			boolStr = "true"
		}
		return handleStringOp(sq.L(textExpr, jsonPath), boolStr, op)
	case nil:
		return handleNullOp(fmt.Sprintf("(%s->?)", expr), jsonPath, op)
	default:
		return nil, fmt.Errorf("unsupported literal type: %T", literal)
	}
}

// celJSONPath converts fieldPath — a CEL field path as expr's parser
// renders it: dot-separated names, each optionally followed by [key]
// index accesses — into a DuckDB JSONPath. An integer index stays [n];
// any other name or key that isn't a plain identifier (e.g. a CEL string
// index like foo["a.b"]) becomes a double-quoted member, so JSONPath-
// special characters in it can't change the path's structure.
func celJSONPath(fieldPath string) string {
	var b strings.Builder
	b.WriteString("$")
	var name strings.Builder
	flush := func() {
		if name.Len() > 0 {
			writeJSONPathMember(&b, name.String())
			name.Reset()
		}
	}
	for i := 0; i < len(fieldPath); i++ {
		switch c := fieldPath[i]; c {
		case '.':
			flush()
		case '[':
			flush()
			key := fieldPath[i+1:]
			if end := strings.IndexByte(key, ']'); end >= 0 {
				key = key[:end]
			}
			i += len(key) + 1
			if isDecimalDigits(key) {
				b.WriteString("[" + key + "]")
			} else {
				writeJSONPathMember(&b, key)
			}
		default:
			name.WriteByte(c)
		}
	}
	flush()
	return b.String()
}

func writeJSONPathMember(b *strings.Builder, name string) {
	if jsonPathIdentRegex.MatchString(name) {
		b.WriteString("." + name)
		return
	}
	b.WriteString(`."`)
	b.WriteString(strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(name))
	b.WriteString(`"`)
}

var jsonPathIdentRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// handleNullOp creates SQL filters for JSON null comparison in DuckDB.
// json_type() returns SQL NULL both for a missing key and for an explicit
// JSON null (verified against v2.1.0), unlike SQLite/Postgres where it
// returns "null" only for an explicit JSON null — so it can't tell the two
// apart. Comparing the extracted JSON value directly against 'null'::JSON
// works instead, and keeps "missing key matches neither == null nor !=
// null" via SQL's three-valued logic (NULL = 'null'::JSON is NULL either
// way).
func handleNullOp(jsonExpr string, jsonPath string, op string) ([]sq.Expression, error) {
	switch op {
	case operators.Equals:
		return []sq.Expression{sq.L(fmt.Sprintf("%s = 'null'::JSON", jsonExpr), jsonPath)}, nil
	case operators.NotEquals:
		return []sq.Expression{sq.L(fmt.Sprintf("%s != 'null'::JSON", jsonExpr), jsonPath)}, nil
	}
	return nil, fmt.Errorf("unsupported null operator: %s (only == and != are supported for null)", op)
}

func handleStringOp(expr exp.LiteralExpression, value string, op string) ([]sq.Expression, error) {
	switch op {
	case operators.Equals:
		return []sq.Expression{expr.Eq(value)}, nil
	case operators.NotEquals:
		return []sq.Expression{expr.Neq(value)}, nil
	}
	return nil, fmt.Errorf("unsupported string operator: %s", op)
}

func handleNumericOp(expr exp.LiteralExpression, value any, op string) ([]sq.Expression, error) {
	switch op {
	case operators.Equals:
		return []sq.Expression{expr.Eq(value)}, nil
	case operators.NotEquals:
		return []sq.Expression{expr.Neq(value)}, nil
	case operators.Greater:
		return []sq.Expression{expr.Gt(value)}, nil
	case operators.GreaterEquals:
		return []sq.Expression{expr.Gte(value)}, nil
	case operators.Less:
		return []sq.Expression{expr.Lt(value)}, nil
	case operators.LessEquals:
		return []sq.Expression{expr.Lte(value)}, nil
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
