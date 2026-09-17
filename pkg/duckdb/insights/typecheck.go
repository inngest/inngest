package insights

import (
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// inferType computes expr's ColumnType against scope, covering every AST
// shape validate's allowlist can actually produce. This is what lets
// deriveTable assign a real type to a CTE's or subquery's computed output
// columns instead of requiring an alias for anything that isn't a bare
// column reference.
//
// Unlike a general-purpose SQL type checker, this never needs to model
// implicit-cast promotion across arbitrary types, unknown functions, or
// DDL — the function allowlist is small and fixed, so every case below is
// exhaustive against it. Anything this function genuinely can't determine
// returns ColumnTypeUnknown rather than guessing.
func inferType(expr parser.Expr, scope *tableScope) ColumnType {
	switch e := expr.(type) {
	case *parser.Literal:
		switch e.Kind {
		case parser.LitString:
			return ColumnTypeString
		case parser.LitNumber:
			return ColumnTypeNumber
		case parser.LitTrue, parser.LitFalse:
			return ColumnTypeBoolean
		default: // LitNull: no type information in a bare NULL
			return ColumnTypeUnknown
		}
	case *parser.Ident:
		return identType(e, scope)
	case *parser.CastExpr:
		return DuckDBToColumnType(e.Type)
	case *parser.TypeLiteral:
		return DuckDBToColumnType(e.Type)
	case *parser.IntervalExpr:
		return ColumnTypeDatetime
	case *parser.CaseExpr:
		return caseExprType(e, scope)
	case *parser.FunctionExpr:
		// Diagnostics are discarded here: this call site only wants a type
		// for CTE/subquery column derivation, and every FunctionExpr this
		// package can reach is also independently walked (and its
		// diagnostics collected) by validate's own exprValidator.
		t, _, _ := functionReturnType(strings.ToLower(strings.Join(e.Name, ".")), e.Args, scope)
		return t
	case *parser.BinaryExpr:
		return binaryExprType(e, scope)
	case *parser.UnaryExpr:
		return unaryExprType(e, scope)
	case *parser.LambdaExpr:
		// A single-param LambdaExpr is how this parser represents a bare
		// "col -> 'key'" JSON access, which returns JSON in DuckDB (unlike
		// "->>", which returns VARCHAR).
		return ColumnTypeJSON
	case *parser.DotExpr:
		// A struct-field/JSON-path access — the exact underlying type
		// isn't tracked at this granularity (this package doesn't model
		// STRUCT field schemas), so this reports the same generic JSON
		// bucket DuckDBToColumnType gives any composite type.
		return ColumnTypeJSON
	case *parser.NullTest, *parser.IsExpr, *parser.BetweenExpr, *parser.LikeExpr,
		*parser.InExpr, *parser.DistinctFromExpr:
		return ColumnTypeBoolean
	case *parser.ListExpr, *parser.StructExpr, *parser.MapExpr:
		return ColumnTypeJSON
	case *parser.ListComprehensionExpr:
		// Always produces a LIST, same JSON bucket as any other composite
		// type above -- regardless of what its Expr/Filter reference.
		return ColumnTypeJSON
	default:
		// *parser.SubqueryExpr, *parser.Parameter, *parser.StarExpr (never
		// meaningful as a single type; deriveTable expands it before ever
		// calling inferType), *parser.PositionalExpr, *parser.DefaultExpr,
		// *parser.GroupingExpr, *parser.SliceExpr: none of these have a
		// type this package can determine.
		return ColumnTypeUnknown
	}
}

// identType resolves a (possibly table-qualified) column reference to its
// declared knownColumn.colType — mirrors resolveIdentHint's (Task 5)
// bare/qualified resolution exactly, but for type instead of hint.
func identType(id *parser.Ident, scope *tableScope) ColumnType {
	switch len(id.Parts) {
	case 1:
		col, ok := scope.uniqueColumn(id.Parts[0])
		if !ok {
			return ColumnTypeUnknown
		}
		return col.colType
	case 2:
		tbl, ok := scope.lookup(id.Parts[0])
		if !ok {
			return ColumnTypeUnknown
		}
		col, ok := tbl.columns[id.Parts[1]]
		if !ok {
			return ColumnTypeUnknown
		}
		return col.colType
	default:
		return ColumnTypeUnknown
	}
}

// caseExprType unifies a CASE expression's WHEN results and ELSE branch:
// if every branch present agrees on type, that's the CASE's type;
// otherwise (including "no branches produced a determinable type")
// ColumnTypeUnknown.
func caseExprType(c *parser.CaseExpr, scope *tableScope) ColumnType {
	var (
		result ColumnType
		set    bool
	)
	consider := func(t ColumnType) bool {
		if !set {
			result, set = t, true
			return true
		}
		return t == result
	}
	for _, w := range c.Whens {
		if !consider(inferType(w.Result, scope)) {
			return ColumnTypeUnknown
		}
	}
	if c.Else != nil {
		if !consider(inferType(c.Else, scope)) {
			return ColumnTypeUnknown
		}
	}
	if !set {
		return ColumnTypeUnknown
	}
	return result
}

// binaryExprType categorizes a BinaryExpr by its operator. Arithmetic
// operators report DATETIME when either operand is DATETIME-typed (date
// +/- interval, the common case this package's tables actually support —
// e.g. queued_at - started_at), NUMBER otherwise.
func binaryExprType(b *parser.BinaryExpr, scope *tableScope) ColumnType {
	switch strings.ToUpper(b.Op) {
	case "=", "==", "<>", "!=", "<", ">", "<=", ">=", "AND", "OR":
		return ColumnTypeBoolean
	case "||":
		return ColumnTypeString
	case "->>":
		return ColumnTypeString
	case "->":
		return ColumnTypeJSON
	case "+", "-", "*", "/", "%", "^", "//":
		if inferType(b.Left, scope) == ColumnTypeDatetime || inferType(b.Right, scope) == ColumnTypeDatetime {
			return ColumnTypeDatetime
		}
		return ColumnTypeNumber
	case "&", "|", "~", "<<", ">>":
		return ColumnTypeNumber
	default:
		return ColumnTypeUnknown
	}
}

// unaryExprType categorizes a UnaryExpr by its operator.
func unaryExprType(u *parser.UnaryExpr, scope *tableScope) ColumnType {
	switch strings.ToUpper(u.Op) {
	case "NOT":
		return ColumnTypeBoolean
	case "-", "+", "~":
		return inferType(u.X, scope)
	default:
		return ColumnTypeUnknown
	}
}

// functionReturnType looks up name's returnType rule in allowedFunctions
// (functions.go) and evaluates it against args/scope, also returning that
// call's result pathHints (nil for the overwhelming majority of
// functions -- see functionInfo.returnType's own doc comment for which
// ones report one) and any non-fatal Diagnostics that rule produced about
// this call (e.g. a wrong argument count). That field is the single
// source of truth for a function's return type and result pathHints,
// colocated with its description/docsURL -- there's no separate switch,
// or separate hint-only lookup, to keep in sync when adding a function
// anymore.
func functionReturnType(name string, args []parser.Expr, scope *tableScope) (ColumnType, []PathHint, []Diagnostic) {
	info, ok := allowedFunctions[name]
	if !ok {
		return ColumnTypeUnknown, nil, nil
	}
	return info.returnType(name, args, scope)
}
