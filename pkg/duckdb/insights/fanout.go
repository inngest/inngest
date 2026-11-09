package insights

import (
	"fmt"
	"slices"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// checkComprehensionFanOut bounds how much a list comprehension can multiply
// its row's data, which checkSizeComposition (only about size-limited calls)
// doesn't. DuckDB evaluates a comprehension's body/filter once per source
// element for a whole 2048-row vector at once, outside memory_limit, so
// anything the body copies from the row is multiplied by the source's
// length. Measured against the pinned duckdb w/ memory_limit=2GiB, threads=4
// over 20k rows of a ~10KB string s:
//
//   - [x for x in string_split(s, ”)] (linear) -> ~1.5GB RSS
//   - [s for x in string_split(s, ”)] -> >19GB (killed at 60s)
//   - [[y for y in string_split(s, ”)] for x in string_split(s, ”)] -> >17GB
//
// So, fail-closed and anywhere in the statement (CTEs and subqueries
// included):
//
//  1. A comprehension can't be nested in another one's body or filter (one in
//     its source is evaluated once per row, like any other source, and is
//     checked on its own).
//  2. A comprehension's body/filter can only reference that comprehension's
//     own variables and literals, combined w/ operators and (allowlisted)
//     function calls. A column, a positional (#1) or star reference, a
//     subquery, or any expression shape not listed in fanOutAllowed fails
//     closed. A qualified x.field off a variable counts only when x isn't
//     also a table/alias/CTE name in the statement, since DuckDB resolves
//     t.col against the FROM tables first. Names compare case-insensitively,
//     like DuckDB's own identifiers.
//
// The source can still be any (already validated) expression: evaluating it
// is one linear pass over the row. So [x.key for x in sessions] and
// [x for x in event_ids if x <> ”] are fine, while [x || run_id for x in
// ...] is rejected even though it's usually harmless.
func checkComprehensionFanOut(stmt *parser.SelectStatement) error {
	f := &fanOut{tables: statementTableNames(stmt)}
	parser.Walk(f, stmt)
	return f.err
}

// fanOut is checkComprehensionFanOut's parser.Visitor over the statement,
// stopping at each outermost comprehension to check it.
type fanOut struct {
	tables map[string]bool
	err    error
}

func (f *fanOut) Visit(n parser.Node) parser.Visitor {
	if f.err != nil {
		return nil
	}
	if l, ok := n.(*parser.ListComprehensionExpr); ok {
		f.err = f.checkComprehension(l)
		return nil
	}
	return f
}

func (f *fanOut) checkComprehension(l *parser.ListComprehensionExpr) error {
	// A comprehension in the source runs once per row, like any other source
	// expression, so it's only checked on its own.
	parser.Walk(f, l.Source)
	if f.err != nil {
		return f.err
	}
	for _, part := range []parser.Expr{l.Expr, l.Filter} {
		if part == nil {
			continue
		}
		var nested parser.Node
		parser.Walk(visitFunc(func(n parser.Node) bool {
			if _, ok := n.(*parser.ListComprehensionExpr); ok {
				nested = n
			}
			return nested == nil
		}), part)
		if nested != nil {
			return fanOutError(nested, "a list comprehension can't be nested inside another list comprehension")
		}
	}
	for _, part := range []parser.Expr{l.Expr, l.Filter} {
		if part == nil {
			continue
		}
		if err := f.checkBody(l.Vars, part); err != nil {
			return err
		}
	}
	return nil
}

// checkBody walks a comprehension's body/filter, allowing only fanOutAllowed
// shapes and references to vars (rule 2).
func (f *fanOut) checkBody(vars []string, e parser.Expr) error {
	var err error
	isVar := func(name string) bool {
		return slices.ContainsFunc(vars, func(v string) bool { return strings.EqualFold(v, name) })
	}
	parser.Walk(visitFunc(func(n parser.Node) bool {
		switch x := n.(type) {
		case *parser.Ident:
			if !isVar(x.Parts[0]) || (len(x.Parts) > 1 && f.tables[strings.ToLower(x.Parts[0])]) {
				err = fanOutError(x, fmt.Sprintf("a list comprehension can only reference its own variables, not %q (it would be copied once per element)", identName(x)))
			}
		case *parser.LambdaExpr:
			// Always JSON-arrow access (var -> 'key'), see checkLambda.
			if len(x.Params) != 1 || !isVar(x.Params[0]) {
				err = fanOutError(x, "a list comprehension can only reference its own variables")
			}
		default:
			if !fanOutAllowed(n) {
				err = fanOutError(n, "a list comprehension's expression can only use its own variables, literals, operators, and function calls")
			}
		}
		return err == nil
	}), e)
	return err
}

// fanOutAllowed lists the expression shapes a comprehension body/filter may
// use besides variable references, none of which read the row on their
// own. Everything else fails closed.
func fanOutAllowed(n parser.Node) bool {
	switch x := n.(type) {
	case *parser.Literal, *parser.FunctionExpr, *parser.BinaryExpr, *parser.UnaryExpr,
		*parser.NullTest, *parser.IsExpr, *parser.DistinctFromExpr, *parser.BetweenExpr,
		*parser.LikeExpr, *parser.CastExpr, *parser.CaseExpr, *parser.CaseWhen,
		*parser.ListExpr, *parser.StructExpr, *parser.StructField, *parser.MapExpr,
		*parser.IntervalExpr, *parser.TypeLiteral, *parser.NamedArg, *parser.DotExpr,
		*parser.SliceExpr, *parser.TypeName, *parser.TypeField:
		return true
	case *parser.InExpr:
		return x.Subquery == nil
	}
	return false
}

// statementTableNames returns every table name, alias, and CTE name in stmt,
// which a qualified reference resolves against before a comprehension
// variable.
func statementTableNames(stmt *parser.SelectStatement) map[string]bool {
	names := map[string]bool{}
	add := func(s string) {
		if s != "" {
			names[strings.ToLower(s)] = true
		}
	}
	parser.Walk(visitFunc(func(n parser.Node) bool {
		switch x := n.(type) {
		case *parser.BaseTableRef:
			if len(x.Name) > 0 {
				add(x.Name[len(x.Name)-1])
			}
			add(x.Alias)
		case *parser.TableFunctionRef:
			add(x.Alias)
		case *parser.TableSubqueryRef:
			add(x.Alias)
		case *parser.ParensTableRef:
			add(x.Alias)
		case *parser.CTE:
			add(x.Name)
		}
		return true
	}), stmt)
	return names
}

func identName(id *parser.Ident) string {
	name := id.Parts[0]
	for _, p := range id.Parts[1:] {
		name += "." + p
	}
	return name
}

func fanOutError(n parser.Node, msg string) error {
	return &ValidationError{Pos: n.Pos(), End: n.End(), Message: msg}
}
