package insights

import (
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// rewriteJSONDotComparison makes a dot access into a JSON column
// (data.user.id) compare as text wherever it's compared against a string,
// rewriting it to its ->> extraction (data ->> '$."user"."id"', VARCHAR).
// Mutates stmt in place.
//
// DuckDB's own dot access on a JSON value returns JSON, so data.user.id =
// 'u1' casts 'u1' to JSON, which fails ('u1' isn't valid JSON text), and
// data.user.id LIKE 'u%' silently matches nothing (the JSON text is
// "u1", quotes and all). Everywhere else the dot access is left alone:
// projected, grouped or ordered it stays JSON (an object stays an object),
// and compared against a number or boolean DuckDB already casts the JSON to
// that type itself (data.n > 5 works, data.n ->> ... > 5 wouldn't: VARCHAR
// vs INTEGER needs an explicit cast).
//
// What counts as "compared against a string" (see stringPeer):
//
//	x = | <> | != | < | <= | > | >= y, x IS [NOT] DISTINCT FROM y
//	x [NOT] IN (y, ...), x [NOT] IN (SELECT y ...), x op ANY|ALL (...)
//	x [NOT] BETWEEN y AND z
//	CASE x WHEN y ...
//
// where some other operand is string-typed (inferType), plus x [NOT]
// LIKE/ILIKE/GLOB/SIMILAR TO y and x IS [NOT] NULL unconditionally (a
// pattern is always text, and a JSON null is SQL NULL once extracted as
// text, so IS NULL matches a null value and a missing key alike).
func rewriteJSONDotComparison(stmt *parser.SelectStatement, ctes map[string]logicalTable, outerScope *tableScope) ([]Diagnostic, error) {
	return rewriteScopedExprs(stmt, ctes, outerScope, (*exprRewriter).rewriteJSONDotOperands)
}

// comparisonOps are the BinaryExpr operators rewriteJSONDotOperands treats
// as comparing their two operands.
var comparisonOps = map[string]bool{
	"=": true, "==": true, "<>": true, "!=": true,
	"<": true, "<=": true, ">": true, ">=": true,
}

// rewriteJSONDotOperands is rewriteJSONDotComparison's exprHook.
func (r *exprRewriter) rewriteJSONDotOperands(e parser.Expr) parser.Expr {
	switch x := e.(type) {
	case *parser.BinaryExpr:
		if comparisonOps[x.Op] {
			r.textIfStringPeer(&x.Left, &x.Right)
		}
	case *parser.DistinctFromExpr:
		r.textIfStringPeer(&x.Left, &x.Right)
	case *parser.QuantifiedExpr:
		r.textIfStringPeer(&x.Left, &x.Right)
	case *parser.BetweenExpr:
		r.textIfStringPeer(&x.X, &x.Low, &x.High)
	case *parser.InExpr:
		if x.Contains {
			// x IN list_expr is list containment, not a comparison.
			return e
		}
		operands := []*parser.Expr{&x.X}
		for i := range x.List {
			operands = append(operands, &x.List[i])
		}
		if x.Subquery != nil && r.selectType(x.Subquery) == ColumnTypeString {
			r.textAlways(operands...)
			return e
		}
		r.textIfStringPeer(operands...)
	case *parser.CaseExpr:
		if x.Operand == nil {
			return e
		}
		operands := []*parser.Expr{&x.Operand}
		for _, w := range x.Whens {
			operands = append(operands, &w.Cond)
		}
		r.textIfStringPeer(operands...)
	case *parser.LikeExpr:
		r.textAlways(&x.X, &x.Pattern)
	case *parser.NullTest:
		r.textAlways(&x.X)
	case *parser.IsExpr:
		if strings.EqualFold(x.Value, "NULL") {
			r.textAlways(&x.X)
		}
	}
	return e
}

// textIfStringPeer rewrites each JSON dot access among operands to its text
// extraction when any operand is string-typed.
func (r *exprRewriter) textIfStringPeer(operands ...*parser.Expr) {
	for _, op := range operands {
		if *op != nil && r.peerType(*op) == ColumnTypeString {
			r.textAlways(operands...)
			return
		}
	}
}

// textAlways rewrites each JSON dot access among operands to its text
// extraction.
func (r *exprRewriter) textAlways(operands ...*parser.Expr) {
	for _, op := range operands {
		if base, path, ok := jsonAccess(*op, r.scope, r.isBound); ok && len(path) > 0 {
			*op = jsonTextAccess(base, path)
		}
	}
}

// peerType is inferType, plus a scalar subquery's single column's type.
func (r *exprRewriter) peerType(e parser.Expr) ColumnType {
	if sub, ok := e.(*parser.SubqueryExpr); ok {
		return r.selectType(sub.Select)
	}
	return inferType(e, r.scope)
}

// selectType is the type of sel's only output column, or ColumnTypeUnknown
// when it has more than one or can't be derived.
func (r *exprRewriter) selectType(sel *parser.SelectStatement) ColumnType {
	tbl, err := deriveTable(sel, "", nil, r.ctes, r.scope, nil)
	if err != nil || len(tbl.columnOrder) != 1 {
		return ColumnTypeUnknown
	}
	col, _ := tbl.column(tbl.columnOrder[0])
	return col.colType
}

// jsonAccess reports whether e is a JSON value: a json column (data, or
// table-qualified e.data) or a dot access into one (data.user.id, or the
// parenthesized (data.user).id, a DotExpr), returning the column reference
// as base and the keys past it as path (user, id; nil for the column
// itself). Name resolution matches checkQualifiedIdent (validate.go): a
// leading table name wins over a same-named column. bound, if non-nil,
// reports a list-comprehension variable, which shadows any column.
func jsonAccess(e parser.Expr, scope *tableScope, bound func(string) bool) (base parser.Expr, path []string, ok bool) {
	switch x := e.(type) {
	case *parser.Ident:
		if bound != nil && bound(x.Parts[0]) {
			return nil, nil, false
		}
		if len(x.Parts) >= 2 {
			if tbl, isTable := scope.lookup(x.Parts[0]); isTable {
				col, found := tbl.column(x.Parts[1])
				if !found || !col.json {
					return nil, nil, false
				}
				return &parser.Ident{Parts: append([]string(nil), x.Parts[:2]...)}, x.Parts[2:], true
			}
		}
		col, found := scope.uniqueColumn(x.Parts[0])
		if !found || !col.json {
			return nil, nil, false
		}
		return &parser.Ident{Parts: []string{x.Parts[0]}}, x.Parts[1:], true
	case *parser.DotExpr:
		base, path, ok := jsonAccess(x.X, scope, bound)
		if !ok {
			return nil, nil, false
		}
		return base, append(append([]string(nil), path...), x.Field), true
	default:
		return nil, nil, false
	}
}

// jsonTextAccess builds base ->> '$."k1"."k2"', every key quoted so one
// holding a dot, space or other JSONPath syntax is still one key (as
// DuckDB's own dot access treats it), matching defer.go's
// (pkg/duckdb/query) ->>'$."key"' quoting.
func jsonTextAccess(base parser.Expr, path []string) parser.Expr {
	var sb strings.Builder
	sb.WriteString("$")
	for _, key := range path {
		sb.WriteString(`."`)
		sb.WriteString(strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(key))
		sb.WriteString(`"`)
	}
	return &parser.BinaryExpr{
		Op:    "->>",
		Left:  base,
		Right: &parser.Literal{Kind: parser.LitString, Text: sb.String()},
	}
}

// isJSONValue reports whether expr is a JSON value (see jsonAccess), for
// a derived table's json flag.
func isJSONValue(expr parser.Expr, scope *tableScope) bool {
	_, _, ok := jsonAccess(expr, scope, nil)
	return ok
}
