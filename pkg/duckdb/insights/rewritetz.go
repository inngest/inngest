package insights

import (
	"fmt"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// tzInstantFunctions is every functions.go allowedFunctions entry whose
// return type is unconditionally TIMESTAMP WITH TIME ZONE (an instant),
// never a plain (naive) TIMESTAMP -- see rewriteTZFunctions' own doc
// comment for why that matters against this schema's physical columns.
var tzInstantFunctions = map[string]bool{
	"now":          true,
	"to_timestamp": true,
}

// rewriteTZFunctions casts every now()/to_timestamp(...) call -- anywhere
// in stmt, CTEs and subqueries included -- to ::TIMESTAMP. Mutates stmt in
// place; mirrors rewriteArrayOfStructAccess's recursion shape (rewrite.go)
// minus scope tracking, which this rewrite has no use for.
//
// Every physical timestamp column this package's six logical tables read
// from (queued_at, started_at, ts, received_at, ...) is declared
// TIMESTAMP_MS (pkg/db/duckdb/migrations/000001_baseline.sql) -- a naive
// timestamp, not an instant. DuckDB never implicitly compares a naive
// TIMESTAMP/TIMESTAMP_MS against a TIMESTAMP WITH TIME ZONE, even though
// both are, physically, the same microseconds-since-epoch encoding
// (quackTimestampToTime's doc comment, pkg/db/duckdb/quack_protocol.go):
// unlike widening a TIMESTAMP_MS to TIMESTAMP_US, crossing the instant/
// naive boundary requires deciding which timezone the naive side means,
// and DuckDB refuses to guess that silently -- it'd not merely be a type
// widening, but a decision the caller could keep silently getting wrong
// if their session's TimeZone weren't what they assumed. So
// `WHERE queued_at > now() - INTERVAL 1 MINUTE` fails with "Cannot compare
// values of type TIMESTAMP_MS and type TIMESTAMP WITH TIME ZONE" even
// though the comparison is well-defined here.
//
// Casting at now()/to_timestamp()'s own call site, rather than at every
// comparison that might use one, lets any expression built on top
// (interval arithmetic, date_trunc, another function's argument, ...)
// inherit the naive TIMESTAMP type for free -- CAST(now() AS TIMESTAMP)
// - INTERVAL 1 MINUTE is itself a TIMESTAMP.
//
// This cast is lossless only because this project never loads DuckDB's
// icu extension (see pkg/db/duckdb/process.go's bootstrap statements),
// which pins the session TimeZone to UTC -- the same timezone
// encodeLiteral (pkg/db/duckdb/literal.go) normalizes every bound
// time.Time into before writing it as a TIMESTAMP literal. Without icu,
// DuckDB's TIMESTAMPTZ<->TIMESTAMP cast is a straight reinterpretation of
// the identical underlying microsecond value, not a wall-clock
// conversion, so both sides of a rewritten comparison agree on "UTC,
// untagged". If icu is ever loaded and TimeZone changed away from UTC,
// this cast would start silently mismatching wall-clock time against the
// UTC-normalized columns -- re-audit this file before making that change.
func rewriteTZFunctions(stmt *parser.SelectStatement) []Diagnostic {
	var diagnostics []Diagnostic

	if stmt.With != nil {
		for _, cte := range stmt.With.CTEs {
			diagnostics = append(diagnostics, rewriteTZFunctions(cte.Select)...)
		}
	}

	if stmt.SetOp != parser.SetOpNone {
		diagnostics = append(diagnostics, rewriteTZFunctions(stmt.SetLeft)...)
		diagnostics = append(diagnostics, rewriteTZFunctions(stmt.SetRight)...)
		return diagnostics
	}

	if stmt.From != nil {
		for _, ref := range stmt.From.Refs {
			diagnostics = append(diagnostics, rewriteTZFunctionsInRef(ref)...)
		}
	}

	r := &tzFunctionRewriter{}
	for _, item := range stmt.Columns {
		item.Expr = r.rewrite(item.Expr)
	}
	if stmt.Where != nil {
		stmt.Where = r.rewrite(stmt.Where)
	}
	if stmt.Having != nil {
		stmt.Having = r.rewrite(stmt.Having)
	}
	if stmt.Qualify != nil {
		stmt.Qualify = r.rewrite(stmt.Qualify)
	}
	if stmt.Distinct != nil {
		for i, e := range stmt.Distinct.On {
			stmt.Distinct.On[i] = r.rewrite(e)
		}
	}
	if stmt.GroupBy != nil {
		for _, item := range stmt.GroupBy.Items {
			r.rewriteGroupByItem(item)
		}
	}
	if stmt.OrderBy != nil {
		for _, item := range stmt.OrderBy.Items {
			item.X = r.rewrite(item.X)
		}
	}
	for _, w := range stmt.Windows {
		r.rewriteWindowSpec(w.Spec)
	}
	diagnostics = append(diagnostics, r.diagnostics...)
	return diagnostics
}

// rewriteTZFunctionsInRef mirrors rewriteArrayOfStructAccessInRef's
// FROM-tree recursion shape.
func rewriteTZFunctionsInRef(ref parser.TableRef) []Diagnostic {
	switch r := ref.(type) {
	case *parser.JoinRef:
		var diagnostics []Diagnostic
		diagnostics = append(diagnostics, rewriteTZFunctionsInRef(r.Left)...)
		diagnostics = append(diagnostics, rewriteTZFunctionsInRef(r.Right)...)
		if r.On != nil {
			rw := &tzFunctionRewriter{}
			r.On = rw.rewrite(r.On)
			diagnostics = append(diagnostics, rw.diagnostics...)
		}
		return diagnostics
	case *parser.TableSubqueryRef:
		return rewriteTZFunctions(r.Select)
	default:
		return nil
	}
}

// tzFunctionRewriter recursively rewrites one expression tree, wrapping
// every tzInstantFunctions call it finds in a ::TIMESTAMP cast and
// recursing into any nested SelectStatement via a fresh, independent
// rewriteTZFunctions call -- mirroring arrayOfStructRewriter's shape
// (rewrite.go) minus scope tracking.
type tzFunctionRewriter struct {
	diagnostics []Diagnostic
}

func (r *tzFunctionRewriter) rewrite(expr parser.Expr) parser.Expr {
	if expr == nil {
		return expr
	}

	switch e := expr.(type) {
	case *parser.BinaryExpr:
		e.Left, e.Right = r.rewrite(e.Left), r.rewrite(e.Right)
	case *parser.UnaryExpr:
		e.X = r.rewrite(e.X)
	case *parser.NullTest:
		e.X = r.rewrite(e.X)
	case *parser.IsExpr:
		e.X = r.rewrite(e.X)
	case *parser.DistinctFromExpr:
		e.Left, e.Right = r.rewrite(e.Left), r.rewrite(e.Right)
	case *parser.BetweenExpr:
		e.X, e.Low, e.High = r.rewrite(e.X), r.rewrite(e.Low), r.rewrite(e.High)
	case *parser.InExpr:
		e.X = r.rewrite(e.X)
		for i, item := range e.List {
			e.List[i] = r.rewrite(item)
		}
		if e.Subquery != nil {
			r.diagnostics = append(r.diagnostics, rewriteTZFunctions(e.Subquery)...)
		}
	case *parser.LikeExpr:
		e.X, e.Pattern, e.Escape = r.rewrite(e.X), r.rewrite(e.Pattern), r.rewrite(e.Escape)
	case *parser.CastExpr:
		e.X = r.rewrite(e.X)
	case *parser.CaseExpr:
		e.Operand = r.rewrite(e.Operand)
		for _, w := range e.Whens {
			w.Cond, w.Result = r.rewrite(w.Cond), r.rewrite(w.Result)
		}
		e.Else = r.rewrite(e.Else)
	case *parser.SubqueryExpr:
		r.diagnostics = append(r.diagnostics, rewriteTZFunctions(e.Select)...)
	case *parser.ListExpr:
		for i, el := range e.Elems {
			e.Elems[i] = r.rewrite(el)
		}
	case *parser.StructExpr:
		for _, f := range e.Fields {
			f.Value = r.rewrite(f.Value)
		}
	case *parser.MapExpr:
		for i := range e.Keys {
			e.Keys[i], e.Values[i] = r.rewrite(e.Keys[i]), r.rewrite(e.Values[i])
		}
	case *parser.IntervalExpr:
		e.Value = r.rewrite(e.Value)
	case *parser.FunctionExpr:
		e.Receiver = r.rewrite(e.Receiver)
		for i, a := range e.Args {
			e.Args[i] = r.rewrite(a)
		}
		e.Filter = r.rewrite(e.Filter)
		if e.Over != nil {
			r.rewriteWindowSpec(e.Over)
		}
		if tzInstantFunctions[strings.ToLower(strings.Join(e.Name, "."))] {
			return r.castToTimestamp(e)
		}
	case *parser.DotExpr:
		e.X = r.rewrite(e.X)
	case *parser.SliceExpr:
		e.X, e.Start, e.Stop, e.Step = r.rewrite(e.X), r.rewrite(e.Start), r.rewrite(e.Stop), r.rewrite(e.Step)
	case *parser.LambdaExpr:
		e.Body = r.rewrite(e.Body)
	}
	return expr
}

func (r *tzFunctionRewriter) rewriteWindowSpec(spec *parser.WindowSpec) {
	if spec == nil {
		return
	}
	for i, p := range spec.PartitionBy {
		spec.PartitionBy[i] = r.rewrite(p)
	}
	if spec.OrderBy != nil {
		for _, item := range spec.OrderBy.Items {
			item.X = r.rewrite(item.X)
		}
	}
}

func (r *tzFunctionRewriter) rewriteGroupByItem(item parser.GroupByItem) {
	switch g := item.(type) {
	case *parser.GroupByExprItem:
		g.X = r.rewrite(g.X)
	case *parser.GroupByCube:
		for i, e := range g.Items {
			g.Items[i] = r.rewrite(e)
		}
	case *parser.GroupByRollup:
		for i, e := range g.Items {
			g.Items[i] = r.rewrite(e)
		}
	case *parser.GroupingSets:
		for _, set := range g.Sets {
			r.rewriteGroupByItem(set)
		}
	}
}

// castToTimestamp wraps f (a tzInstantFunctions call) in a ::TIMESTAMP
// cast and records an info diagnostic at f's original position explaining
// the rewrite -- see rewriteTZFunctions' own doc comment.
func (r *tzFunctionRewriter) castToTimestamp(f *parser.FunctionExpr) parser.Expr {
	replacement := &parser.CastExpr{X: f, Type: "TIMESTAMP"}
	r.diagnostics = append(r.diagnostics, diagnosticAt(f, DiagnosticInfo, "tz-function-cast-to-timestamp",
		fmt.Sprintf("%s returns TIMESTAMP WITH TIME ZONE; cast to TIMESTAMP to compare against this schema's timestamp columns.",
			parser.String(f))))
	return replacement
}
