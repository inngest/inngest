package insights

import (
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// jsonPathAccess recognizes every AST shape this package treats as a
// literal JSON key/path extraction off a column: col ->> 'key' (BinaryExpr,
// base bare or table-qualified), col -> 'key' (a single-param LambdaExpr —
// the parser only folds a *bare* base into this shape, so a qualified base
// arrives as "->"'s BinaryExpr instead, which this function does not
// recognize), col.key (DotExpr), and json_extract(col, 'key') /
// json_extract_string(col, 'key') (FunctionExpr). ok is false for anything
// else (computed path, function over the extracted value, non-literal path
// arg) — no hint rather than a guess, matching buildColumnHints' contract.
func jsonPathAccess(expr parser.Expr) (col *parser.Ident, path string, ok bool) {
	switch e := expr.(type) {
	case *parser.BinaryExpr:
		if e.Op != "->>" {
			return nil, "", false
		}
		return identAndLiteral(e.Left, e.Right)
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
		baseIdent, ok := e.X.(*parser.Ident)
		if !ok || len(baseIdent.Parts) != 1 {
			return nil, "", false
		}
		return baseIdent, e.Field, true
	case *parser.FunctionExpr:
		name := strings.ToLower(strings.Join(e.Name, "."))
		if (name != "json_extract" && name != "json_extract_string") || len(e.Args) != 2 {
			return nil, "", false
		}
		return identAndLiteral(e.Args[0], e.Args[1])
	default:
		return nil, "", false
	}
}

// identAndLiteral resolves a two-operand JSON path access's operands
// (base ->> pathExpr, or json_extract_string(base, pathExpr)), only when
// base is a bare or table-qualified column Ident and pathExpr is a literal
// string.
func identAndLiteral(baseExpr, pathExpr parser.Expr) (col *parser.Ident, path string, ok bool) {
	baseIdent, isIdent := baseExpr.(*parser.Ident)
	if !isIdent || len(baseIdent.Parts) == 0 || len(baseIdent.Parts) > 2 {
		return nil, "", false
	}
	lit, isLit := pathExpr.(*parser.Literal)
	if !isLit || lit.Kind != parser.LitString {
		return nil, "", false
	}
	return baseIdent, lit.Text, true
}
