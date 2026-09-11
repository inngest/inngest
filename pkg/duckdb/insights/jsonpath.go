package insights

import (
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// jsonPathAccess recognizes every AST shape this package treats as a
// literal JSON key/path extraction off some base value: base ->> 'key' or
// base -> 'key' (BinaryExpr -- base bare/table-qualified or a computed
// expression like a LIST()/ARRAY_AGG()/UNNEST() call; DuckDB's "->>"
// extracts VARCHAR text and "->" a JSON value, but this package only ever
// traces the *hint*, not the extracted value's own representation, so both
// operators are treated identically here), col -> 'key' (a single-param
// LambdaExpr — the parser only folds a *bare* column into this exact
// shape; a qualified or computed base instead arrives as "->"'s BinaryExpr
// above, per adaptLambdaArrowExpression, pkg/duckdb/parser/adapter_expr.go),
// base.key (DotExpr), and json_extract(base, 'key') /
// json_extract_string(base, 'key') (FunctionExpr). ok is false for
// anything else (a non-literal path arg, or a function over the
// *extracted* value, since only the outer shape is examined here) — no
// hint rather than a guess, matching buildColumnHints' contract.
//
// base is returned as a general parser.Expr, not restricted to an Ident,
// specifically so a computed expression's own pathHints
// (resolveColumnPathHints, columnhints.go -- today, a LIST()/ARRAY_AGG()/
// UNNEST() call) can be looked up the same way a bare/qualified column's
// are: e.g. list(inputs) -> '$[*][*].meta.sessions' projects inputs' own
// {wc, "meta", "sessions"} pathHints entry (runInputsHints, tables.go)
// through list(inputs)'s own extra leading wc (listReturnType,
// functions.go, for aggregating an already-array-typed column) and this
// access's own two-wildcard queryPath, landing back on HintSession.
// resolveColumnPathHints itself is what actually rejects a base it
// doesn't recognize (returning nil), so no restriction is needed here.
func jsonPathAccess(expr parser.Expr) (base parser.Expr, path string, ok bool) {
	switch e := expr.(type) {
	case *parser.BinaryExpr:
		if e.Op != "->>" && e.Op != "->" {
			return nil, "", false
		}
		return baseAndLiteralPath(e.Left, e.Right)
	case *parser.LambdaExpr:
		if len(e.Params) != 1 {
			return nil, "", false
		}
		lit, ok := e.Body.(*parser.Literal)
		if !ok || lit.Kind != parser.LitString {
			return nil, "", false
		}
		return &parser.Ident{Parts: []string{e.Params[0]}}, lit.Text, true
	case *parser.DotExpr:
		return e.X, e.Field, true
	case *parser.FunctionExpr:
		name := strings.ToLower(strings.Join(e.Name, "."))
		if (name != "json_extract" && name != "json_extract_string") || len(e.Args) != 2 {
			return nil, "", false
		}
		return baseAndLiteralPath(e.Args[0], e.Args[1])
	default:
		return nil, "", false
	}
}

// baseAndLiteralPath resolves a two-operand JSON path access's operands
// (base ->> pathExpr, or json_extract_string(base, pathExpr)) -- base is
// returned as-is (jsonPathAccess's own doc comment explains why this
// isn't restricted to an Ident), only when pathExpr is a literal string.
func baseAndLiteralPath(baseExpr, pathExpr parser.Expr) (base parser.Expr, path string, ok bool) {
	lit, isLit := pathExpr.(*parser.Literal)
	if !isLit || lit.Kind != parser.LitString {
		return nil, "", false
	}
	return baseExpr, lit.Text, true
}
