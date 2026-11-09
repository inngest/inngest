package insights

import (
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// rewriteJSONDotComparison makes a dot access into a JSON column
// (data.user.id) compare as text wherever it's compared against a string,
// wrapping it in its text extraction (json_extract_string(data.user.id,
// '$'), VARCHAR, see jsonText). Mutates stmt in place.
//
// DuckDB's own dot access on a JSON value returns JSON, so data.user.id =
// 'u1' casts 'u1' to JSON, which fails ('u1' isn't valid JSON text), and
// data.user.id LIKE 'u%' silently matches nothing (the JSON text is
// "u1", quotes and all). Everywhere else the dot access is left alone:
// projected, grouped or ordered it stays JSON (an object stays an object),
// and compared against a number or boolean DuckDB already casts the JSON to
// that type itself (data.n > 5 works, its text extraction > 5 wouldn't:
// VARCHAR vs INTEGER needs an explicit cast).
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
//
// Text contexts get the same extraction anywhere in the query: x || y (both
// operands), x directly an argument of a string function (stringFunctions),
// and CAST(x AS VARCHAR) (TEXT, STRING, ::VARCHAR, TRY_CAST), which the
// extraction replaces. A projected x on its own stays JSON: Execute decodes
// it to its native value already.
func rewriteJSONDotComparison(stmt *parser.SelectStatement, ctes map[string]logicalTable, outerScope *tableScope) ([]Diagnostic, error) {
	return rewriteScopedExprs(stmt, ctes, outerScope, (*exprRewriter).rewriteJSONDotOperands)
}

// comparisonOps are the BinaryExpr operators rewriteJSONDotOperands treats
// as comparing their two operands.
var comparisonOps = map[string]bool{
	"=": true, "==": true, "<>": true, "!=": true,
	"<": true, "<=": true, ">": true, ">=": true,
}

// stringFunctions take their arguments as text, so a JSON dot access passed
// directly to one is wrapped in its text extraction. length/len/strlen
// included: DuckDB already measures a JSON value (array included) as its
// text, so the only change is a string's quotes no longer counting.
var stringFunctions = map[string]bool{
	"starts_with": true, "prefix": true, "ends_with": true, "suffix": true, "contains": true,
	"lower": true, "lcase": true, "upper": true, "ucase": true,
	"length": true, "strlen": true, "len": true,
	"regexp_matches": true, "regexp_full_match": true, "regexp_extract": true, "regexp_replace": true, "regexp_split_to_array": true,
	"substring": true, "substr": true, "replace": true, "trim": true, "ltrim": true, "rtrim": true,
	"split_part": true, "left": true, "right": true, "strpos": true, "instr": true, "position": true,
	"concat": true, "concat_ws": true,
}

// rewriteJSONDotOperands is rewriteJSONDotComparison's exprHook.
func (r *exprRewriter) rewriteJSONDotOperands(e parser.Expr) parser.Expr {
	switch x := e.(type) {
	case *parser.BinaryExpr:
		switch {
		case comparisonOps[x.Op]:
			r.textIfStringPeer(&x.Left, &x.Right)
		case x.Op == "||":
			r.textAlways(&x.Left, &x.Right)
		}
	case *parser.CastExpr:
		// A cast to text already asks for the value as text, so its
		// extraction replaces the cast outright (TRY_CAST included: neither
		// can fail on JSON).
		if len(x.Type.Arrays) == 0 && x.Type.BaseTypeName() == "VARCHAR" && isJSONDot(x.X, r.scope, r.isBound) {
			return jsonText(x.X)
		}
	case *parser.FunctionExpr:
		if stringFunctions[strings.ToLower(strings.Join(x.Name, "."))] {
			r.textAlways(&x.Receiver)
			for i := range x.Args {
				r.textAlways(&x.Args[i])
			}
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

// textAlways wraps each JSON dot access among operands in its text
// extraction.
func (r *exprRewriter) textAlways(operands ...*parser.Expr) {
	for _, op := range operands {
		if isJSONDot(*op, r.scope, r.isBound) {
			*op = jsonText(*op)
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

// jsonAccess reports whether e is a JSON value (json): a json column (data,
// or table-qualified e.data), a computed one (isComputedJSON), or a dot
// access into either; and whether it's such a dot access (dot): data.user.id,
// or the parenthesized (data.user).id (a DotExpr). Name resolution matches
// checkQualifiedIdent (validate.go): a leading table name wins over a
// same-named column. bound, if non-nil, reports a list-comprehension
// variable, which shadows any column.
func jsonAccess(e parser.Expr, scope *tableScope, bound func(string) bool) (json, dot bool) {
	switch x := e.(type) {
	case *parser.Ident:
		if bound != nil && bound(x.Parts[0]) {
			return false, false
		}
		col, rest, found := resolveColumnPrefix(x.Parts, scope)
		if !found || !col.json {
			return false, false
		}
		return true, len(rest) > 0
	case *parser.DotExpr:
		json, _ := jsonAccess(x.X, scope, bound)
		return json, json
	default:
		return isComputedJSON(e), false
	}
}

// resolveColumnPrefix resolves the column parts names (table.col.rest... if
// parts[0] is a table in scope, else col.rest...), returning it and the
// parts past it.
func resolveColumnPrefix(parts []string, scope *tableScope) (col knownColumn, rest []string, ok bool) {
	if len(parts) >= 2 {
		if tbl, isTable := scope.lookup(parts[0]); isTable {
			col, ok = tbl.column(parts[1])
			return col, parts[2:], ok
		}
	}
	col, ok = scope.uniqueColumn(parts[0])
	return col, parts[1:], ok
}

// isJSONDot reports whether e is a dot access into a JSON value (see
// jsonAccess).
func isJSONDot(e parser.Expr, scope *tableScope, bound func(string) bool) bool {
	_, dot := jsonAccess(e, scope, bound)
	return dot
}

// jsonReturningFunctions return a single JSON value whatever their
// arguments.
var jsonReturningFunctions = map[string]bool{
	"to_json": true, "json_object": true, "json_array": true, "json_merge_patch": true,
}

// isComputedJSON reports whether e is an expression DuckDB types as JSON
// proper: a cast to JSON, a JSON-building function, or a single-path
// extraction (json_extract(x, '$.k'), x -> 'k'). A wildcard path
// ('$[*].k') or a list of paths extracts JSON[] instead, which isn't.
func isComputedJSON(e parser.Expr) bool {
	switch x := e.(type) {
	case *parser.CastExpr:
		return len(x.Type.Arrays) == 0 && x.Type.BaseTypeName() == "JSON"
	case *parser.FunctionExpr:
		name := strings.ToLower(strings.Join(x.Name, "."))
		if jsonReturningFunctions[name] {
			return true
		}
		if (name != "json_extract" && name != "json_extract_path") || len(x.Args) != 2 {
			return false
		}
		return isScalarJSONPath(x.Args[1])
	case *parser.BinaryExpr:
		return x.Op == "->" && isScalarJSONPath(x.Right)
	case *parser.LambdaExpr:
		// A bare col -> 'key' (see jsonPathAccess, jsonpath.go).
		return len(x.Params) == 1 && isScalarJSONPath(x.Body)
	default:
		return false
	}
}

// isScalarJSONPath reports whether e is a literal JSON path (or array
// index) without a wildcard.
func isScalarJSONPath(e parser.Expr) bool {
	lit, ok := e.(*parser.Literal)
	if !ok {
		return false
	}
	switch lit.Kind {
	case parser.LitNumber:
		return true
	case parser.LitString:
		return !strings.Contains(lit.Text, "*")
	default:
		return false
	}
}

// jsonText wraps e, a JSON value, in json_extract_string(e, '$'): its
// text, the same VARCHAR ->> extracts (a string unquoted, a number, bool,
// object or array as its JSON text, a JSON null or missing key as NULL).
// Wrapping the dot access as written, rather than re-encoding its keys as
// one ->> JSONPath, keeps any base (a parenthesized or computed one) and key
// (dots, quotes) exactly as DuckDB resolves it. json_extract(e, '$') would
// be a no-op, still JSON.
func jsonText(e parser.Expr) parser.Expr {
	return &parser.FunctionExpr{
		Name: []string{"json_extract_string"},
		Args: []parser.Expr{e, &parser.Literal{Kind: parser.LitString, Text: "$"}},
	}
}

// isJSONValue reports whether expr is a JSON value (see jsonAccess), for
// a derived table's json flag.
func isJSONValue(expr parser.Expr, scope *tableScope) bool {
	json, _ := jsonAccess(expr, scope, nil)
	return json
}
