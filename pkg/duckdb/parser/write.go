// pkg/duckdb/parser/write.go
package parser

import (
	"fmt"
	"io"
	"strings"
)

// Write renders n back into DuckDB SQL text, writing it to w. It
// reconstructs syntax from the AST's semantic fields, not by slicing the
// original source via Pos()/End() (baseExpr's doc comment reserves those
// spans for diagnostics), so the output need not match the original
// spelling — whitespace, keyword casing, redundant parens, LIMIT/OFFSET vs
// FETCH, a PIVOT statement vs. its table-suffix form — but it is always a
// semantically equivalent DuckDB SELECT statement for any AST this
// package's own parser produced. It returns the first error w returned, if
// any; every other write is skipped once one occurs.
func Write(w io.Writer, n Node) error {
	sw := &sqlWriter{w: w}
	sw.node(n)
	return sw.err
}

// String renders n back into DuckDB SQL text and returns it directly.
func String(n Node) string {
	var sb strings.Builder
	_ = Write(&sb, n) // strings.Builder's Write never returns an error
	return sb.String()
}

type sqlWriter struct {
	w   io.Writer
	err error
}

func (sw *sqlWriter) str(s string) {
	if sw.err != nil || s == "" {
		return
	}
	_, sw.err = io.WriteString(sw.w, s)
}

// sep writes s before every element after the first — call with the loop
// index before writing element i.
func (sw *sqlWriter) sep(i int, s string) {
	if i > 0 {
		sw.str(s)
	}
}

// --- precedence: the 17-level chain from adaptExpression's "root to leaf"
// comment in adapter_expr.go, loosest (1) to tightest (17). Needed to
// re-insert parens around a subexpression whenever printing it flat would
// change how it reparses — grouping parens are transparent in this
// package's AST (see adaptSingleExpression's ParensExpression case), so
// this is the only mechanism that keeps Write's output semantically
// equivalent to the source it came from. ---

const (
	precLambdaArrow = iota + 1
	precOr
	precAnd
	precNot
	precIs
	precDistinctFrom
	precComparison
	precBetweenInLike
	precOther
	precBitwise
	precAdditive
	precMultiplicative
	precExponent
	precCollate
	precAtTimeZone
	precPrefix
	precAtom
)

func prec(e Expr) int {
	switch v := e.(type) {
	case *LambdaExpr:
		return precLambdaArrow
	case *BinaryExpr:
		return binaryPrec(v.Op)
	case *UnaryExpr:
		switch {
		case v.Postfix:
			return precAtom
		case v.Op == "NOT":
			return precNot
		default:
			return precPrefix
		}
	case *IsExpr, *NullTest:
		return precIs
	case *DistinctFromExpr:
		return precDistinctFrom
	case *BetweenExpr, *InExpr, *LikeExpr:
		return precBetweenInLike
	default:
		return precAtom
	}
}

// binaryPrec maps a BinaryExpr.Op string to its precedence level. AND/OR
// and the AT TIME ZONE operator are BinaryExpr too (foldBinaryLevel builds
// every one of these 9 levels the same way — see adapter_expr.go) even
// though they read like keywords, not symbols. Any operator text this
// switch doesn't recognize (a custom NamedOtherOperator, OPERATOR(...), an
// ANY/ALL-suffixed comparison, ...) falls through to precOther, which
// matches where OtherOperatorExpression sits in the grammar regardless of
// the specific operator spelling.
func binaryPrec(op string) int {
	switch op {
	case "->":
		return precLambdaArrow
	case "OR":
		return precOr
	case "AND":
		return precAnd
	case "=", "==", "!=", "<>", "<", ">", "<=", ">=":
		return precComparison
	case "&", "|", "<<", ">>":
		return precBitwise
	case "+", "-":
		return precAdditive
	case "*", "/", "//", "%":
		return precMultiplicative
	case "^", "**":
		return precExponent
	case "COLLATE":
		return precCollate
	case "AT TIME ZONE":
		return precAtTimeZone
	default:
		return precOther
	}
}

// exprAtLeast prints e, parenthesizing it if its precedence is lower than
// minPrec — i.e. if printing it flat at this position would reparse with
// different grouping than the tree actually has.
func (sw *sqlWriter) exprAtLeast(e Expr, minPrec int) {
	if prec(e) < minPrec {
		sw.str("(")
		sw.exprTop(e)
		sw.str(")")
		return
	}
	sw.exprTop(e)
}

func (sw *sqlWriter) exprListTop(es []Expr) {
	for i, e := range es {
		sw.sep(i, ", ")
		sw.exprTop(e)
	}
}

// --- identifiers ---

func isSimpleIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		letter := c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		digit := c >= '0' && c <= '9'
		if i == 0 {
			if !letter {
				return false
			}
			continue
		}
		if !letter && !digit {
			return false
		}
	}
	return true
}

// writeIdentPart quotes s with DuckDB's double-quote identifier syntax
// unless it's already a plain, unquotable-collision-free identifier. This
// package's adapter decodes a quoted identifier's escapes away (see
// primitives.go's scanQuotedIdentifier), so whether the source quoted a
// given name isn't preserved — this always re-quotes when the plain form
// wouldn't parse back to the same identifier, but doesn't reserved-word
// check (that needs the loaded keyword sets, not just the identifier
// text), so a name that collides with a reserved keyword round-trips only
// if it also fails isSimpleIdent for some other reason.
func writeIdentPart(sw *sqlWriter, s string) {
	if isSimpleIdent(s) {
		sw.str(s)
		return
	}
	sw.str(`"`)
	sw.str(strings.ReplaceAll(s, `"`, `""`))
	sw.str(`"`)
}

func writeDottedName(sw *sqlWriter, parts []string) {
	for i, p := range parts {
		sw.sep(i, ".")
		writeIdentPart(sw, p)
	}
}

// writeNameList prints a single bare identifier, or a parenthesized,
// comma-separated list when there's more than one — the shape UNPIVOT's
// header/name lists use (see UnpivotHeader in expression.gram).
func writeNameList(sw *sqlWriter, names []string) {
	if len(names) == 1 {
		writeIdentPart(sw, names[0])
		return
	}
	sw.str("(")
	for i, n := range names {
		sw.sep(i, ", ")
		writeIdentPart(sw, n)
	}
	sw.str(")")
}

func startsWithLetter(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// --- node dispatch ---

func (sw *sqlWriter) node(n Node) {
	// Concrete cases must come first: SelectStatement, FromClause, and the
	// other clause types below all embed baseExpr too (for Pos()/End()), so
	// they satisfy Expr just as much as an actual expression node does —
	// the broad Expr/TableRef/GroupByItem cases would otherwise shadow them.
	switch v := n.(type) {
	case *SelectStatement:
		sw.selectStatement(v)
	case *FromClause:
		sw.fromClause(v)
	case *WithClause:
		sw.withClause(v)
	case *CTE:
		sw.cte(v)
	case *WindowDef:
		sw.windowDef(v)
	case *OrderByClause:
		sw.orderByClause(v)
	case *OrderByItem:
		sw.exprTop(v.X)
		sw.orderBySuffix(v)
	case *GroupByClause:
		sw.groupByClause(v)
	case *DistinctClause:
		sw.distinctClause(v)
	case *LimitClause:
		sw.limitClause(v)
	case *SelectItem:
		sw.selectItem(v)
	case *PivotColumn:
		sw.pivotColumn(v)
	case *FrameClause:
		sw.frameClause(v)
	case *FrameBound:
		sw.frameBound(v)
	case Expr:
		sw.exprTop(v)
	case TableRef:
		sw.tableRef(v)
	case GroupByItem:
		sw.groupByItem(v)
	default:
		panic(fmt.Sprintf("duckdb/parser: Write: unhandled Node %T", n))
	}
}

// --- expressions ---

func (sw *sqlWriter) exprTop(e Expr) {
	switch v := e.(type) {
	case *Ident:
		writeDottedName(sw, v.Parts)
	case *Literal:
		sw.literal(v)
	case *Parameter:
		sw.parameter(v)
	case *BinaryExpr:
		p := binaryPrec(v.Op)
		sw.exprAtLeast(v.Left, p)
		sw.str(" ")
		sw.str(v.Op)
		sw.str(" ")
		sw.exprAtLeast(v.Right, p+1)
	case *UnaryExpr:
		sw.unaryExpr(v)
	case *NullTest:
		// NullTest and IsExpr both model "IS [NOT] NULL" — NullTest also
		// covers DuckDB's compact ISNULL/NOTNULL/NOT-NULL spellings, which
		// aren't distinguished from each other in the AST. Always emitting
		// the keyword-with-IS form here is a normalization, not a loss: all
		// spellings are semantically identical.
		sw.exprAtLeast(v.X, precIs)
		if v.Not {
			sw.str(" IS NOT NULL")
		} else {
			sw.str(" IS NULL")
		}
	case *IsExpr:
		sw.exprAtLeast(v.X, precIs)
		sw.str(" IS ")
		if v.Not {
			sw.str("NOT ")
		}
		sw.str(v.Value)
	case *DistinctFromExpr:
		sw.exprAtLeast(v.Left, precDistinctFrom)
		sw.str(" IS ")
		if v.Not {
			sw.str("NOT ")
		}
		sw.str("DISTINCT FROM ")
		sw.exprAtLeast(v.Right, precDistinctFrom+1)
	case *BetweenExpr:
		sw.exprAtLeast(v.X, precBetweenInLike)
		if v.Not {
			sw.str(" NOT BETWEEN ")
		} else {
			sw.str(" BETWEEN ")
		}
		sw.exprAtLeast(v.Low, precOther)
		sw.str(" AND ")
		sw.exprAtLeast(v.High, precOther)
	case *InExpr:
		sw.exprAtLeast(v.X, precBetweenInLike)
		if v.Not {
			sw.str(" NOT IN (")
		} else {
			sw.str(" IN (")
		}
		if v.Subquery != nil {
			sw.selectStatement(v.Subquery)
		} else {
			sw.exprListTop(v.List)
		}
		sw.str(")")
	case *LikeExpr:
		sw.exprAtLeast(v.X, precBetweenInLike)
		sw.str(" ")
		if v.Not {
			sw.str("NOT ")
		}
		sw.str(v.Op)
		sw.str(" ")
		sw.exprAtLeast(v.Pattern, precOther)
		if v.Escape != nil {
			sw.str(" ESCAPE ")
			sw.exprTop(v.Escape)
		}
	case *CastExpr:
		if v.TryCast {
			sw.str("TRY_CAST(")
			sw.exprTop(v.X)
			sw.str(" AS ")
			sw.str(v.Type)
			sw.str(")")
			return
		}
		sw.exprAtLeast(v.X, precAtom)
		sw.str("::")
		sw.str(v.Type)
	case *CaseWhen:
		sw.str("WHEN ")
		sw.exprTop(v.Cond)
		sw.str(" THEN ")
		sw.exprTop(v.Result)
	case *CaseExpr:
		sw.caseExpr(v)
	case *StarExpr:
		sw.starExpr(v)
	case *SubqueryExpr:
		if v.Exists {
			if v.Not {
				sw.str("NOT ")
			}
			sw.str("EXISTS ")
		}
		sw.str("(")
		sw.selectStatement(v.Select)
		sw.str(")")
	case *ListExpr:
		sw.listExpr(v)
	case *StructField:
		writeIdentPart(sw, v.Key)
		sw.str(": ")
		sw.exprTop(v.Value)
	case *StructExpr:
		sw.str("{")
		for i, f := range v.Fields {
			sw.sep(i, ", ")
			sw.exprTop(f)
		}
		sw.str("}")
	case *MapExpr:
		sw.str("MAP {")
		for i := range v.Keys {
			sw.sep(i, ", ")
			sw.exprTop(v.Keys[i])
			sw.str(": ")
			sw.exprTop(v.Values[i])
		}
		sw.str("}")
	case *IntervalExpr:
		sw.str("INTERVAL ")
		sw.exprTop(v.Value)
		if v.Unit != "" {
			sw.str(" ")
			sw.str(v.Unit)
		}
	case *TypeLiteral:
		sw.str(v.Type)
		sw.str(" '")
		sw.str(strings.ReplaceAll(v.Value, "'", "''"))
		sw.str("'")
	case *NamedArg:
		sw.str(v.Name)
		sw.str(" := ")
		sw.exprTop(v.Value)
	case *FunctionExpr:
		sw.functionExpr(v)
	case *WindowSpec:
		sw.windowSpec(v)
	case *DotExpr:
		sw.exprAtLeast(v.X, precAtom)
		sw.str(".")
		writeIdentPart(sw, v.Field)
	case *SliceExpr:
		sw.sliceExpr(v)
	case *LambdaExpr:
		sw.lambdaExpr(v)
	case *PositionalExpr:
		sw.str("#")
		sw.str(v.Index)
	case *DefaultExpr:
		sw.str("DEFAULT")
	case *ListComprehensionExpr:
		sw.listComprehensionExpr(v)
	case *GroupingExpr:
		sw.str("GROUPING(")
		sw.exprListTop(v.Args)
		sw.str(")")
	default:
		panic(fmt.Sprintf("duckdb/parser: Write: unhandled Expr %T", e))
	}
}

func (sw *sqlWriter) literal(l *Literal) {
	switch l.Kind {
	case LitString:
		sw.str("'")
		sw.str(strings.ReplaceAll(l.Text, "'", "''"))
		sw.str("'")
	case LitNumber:
		sw.str(l.Text)
	case LitNull:
		sw.str("NULL")
	case LitTrue:
		sw.str("TRUE")
	case LitFalse:
		sw.str("FALSE")
	}
}

func (sw *sqlWriter) parameter(p *Parameter) {
	switch p.Kind {
	case ParamAnonymous:
		sw.str("?")
	case ParamNumbered:
		sw.str("$")
		sw.str(p.Value)
	default: // ParamNamed
		sw.str("$")
		writeIdentPart(sw, p.Value)
	}
}

func (sw *sqlWriter) unaryExpr(u *UnaryExpr) {
	if u.Postfix {
		sw.exprAtLeast(u.X, precAtom)
		sw.str(u.Op)
		return
	}
	sw.str(u.Op)
	if startsWithLetter(u.Op) {
		sw.str(" ")
	}
	sw.exprAtLeast(u.X, prec(u))
}

func (sw *sqlWriter) caseExpr(c *CaseExpr) {
	sw.str("CASE")
	if c.Operand != nil {
		sw.str(" ")
		sw.exprTop(c.Operand)
	}
	for _, w := range c.Whens {
		sw.str(" ")
		sw.exprTop(w)
	}
	if c.Else != nil {
		sw.str(" ELSE ")
		sw.exprTop(c.Else)
	}
	sw.str(" END")
}

func (sw *sqlWriter) starExpr(s *StarExpr) {
	if len(s.Qualifier) > 0 {
		writeDottedName(sw, s.Qualifier)
		sw.str(".")
	}
	sw.str("*")
	if len(s.Exclude) > 0 {
		sw.str(" EXCLUDE (")
		for i, e := range s.Exclude {
			sw.sep(i, ", ")
			writeIdentPart(sw, e)
		}
		sw.str(")")
	}
}

// listExpr prints both ListExpr shapes: Paren=false is a bracketed array
// literal ("[a, b]"), Paren=true is a parenthesized tuple / grouping-parens
// literal ("(a, b)" or "()" — see adaptParenthesisExpression).
func (sw *sqlWriter) listExpr(l *ListExpr) {
	if l.Paren {
		sw.str("(")
		sw.exprListTop(l.Elems)
		sw.str(")")
		return
	}
	sw.str("[")
	sw.exprListTop(l.Elems)
	sw.str("]")
}

// sliceExpr can't always distinguish a bare index x[i] from an open-ended
// slice x[i:] — both adapt to Start=i, Stop=nil, HasStep=false, since
// SliceExpr has no separate "a colon was written" flag (see
// adaptSliceExpression). This always emits the bare-index form for that
// shape; HasStep alone is unambiguous (the grammar can't match a step
// colon without first matching a stop colon), so the ":" before a step is
// always printed correctly.
func (sw *sqlWriter) sliceExpr(s *SliceExpr) {
	sw.exprAtLeast(s.X, precAtom)
	sw.str("[")
	if s.Start != nil {
		sw.exprTop(s.Start)
	}
	if s.Stop != nil || s.HasStep {
		sw.str(":")
		if s.Stop != nil {
			sw.exprTop(s.Stop)
		}
	}
	if s.HasStep {
		sw.str(":")
		if s.Step != nil {
			sw.exprTop(s.Step)
		}
	}
	sw.str("]")
}

func (sw *sqlWriter) lambdaExpr(l *LambdaExpr) {
	if len(l.Params) == 1 {
		sw.str(l.Params[0])
	} else {
		sw.str("(")
		for i, p := range l.Params {
			sw.sep(i, ", ")
			sw.str(p)
		}
		sw.str(")")
	}
	sw.str(" -> ")
	sw.exprTop(l.Body)
}

func (sw *sqlWriter) listComprehensionExpr(l *ListComprehensionExpr) {
	sw.str("[")
	sw.exprTop(l.Expr)
	sw.str(" FOR ")
	for i, v := range l.Vars {
		sw.sep(i, ", ")
		sw.str(v)
	}
	sw.str(" IN ")
	sw.exprTop(l.Source)
	if l.Filter != nil {
		sw.str(" IF ")
		sw.exprTop(l.Filter)
	}
	sw.str("]")
}

func (sw *sqlWriter) functionExpr(f *FunctionExpr) {
	if f.Receiver != nil {
		sw.exprAtLeast(f.Receiver, precAtom)
		sw.str(".")
	}
	writeDottedName(sw, f.Name)
	sw.str("(")
	if f.Distinct {
		sw.str("DISTINCT ")
	}
	if f.All {
		sw.str("ALL ")
	}
	for i, a := range f.Args {
		sw.sep(i, ", ")
		sw.exprTop(a)
	}
	if f.OrderBy != nil {
		if len(f.Args) > 0 {
			sw.str(" ")
		}
		sw.orderByClause(f.OrderBy)
	}
	if f.IgnoreNulls {
		sw.str(" IGNORE NULLS")
	}
	if f.RespectNulls {
		sw.str(" RESPECT NULLS")
	}
	sw.str(")")
	if f.Within != nil {
		sw.str(" WITHIN GROUP (")
		sw.orderByClause(f.Within)
		sw.str(")")
	}
	if f.Filter != nil {
		sw.str(" FILTER (WHERE ")
		sw.exprTop(f.Filter)
		sw.str(")")
	}
	if f.Export {
		sw.str(" EXPORT_STATE")
	}
	if f.Over != nil {
		sw.windowSpec(f.Over)
	}
}

// windowSpec prints a FunctionExpr's OVER clause. A spec that's just a bare
// name (no partition/order/frame) always prints as the unparenthesized
// "OVER name" form — DuckDB accepts "OVER (name)" too, but the two aren't
// distinguished in the AST (see adaptOverClause's ParensIdentifier case).
func (sw *sqlWriter) windowSpec(w *WindowSpec) {
	sw.str(" OVER ")
	if w.Name != "" && w.PartitionBy == nil && w.OrderBy == nil && w.Frame == nil {
		writeIdentPart(sw, w.Name)
		return
	}
	sw.str("(")
	sw.windowSpecBody(w)
	sw.str(")")
}

func (sw *sqlWriter) windowSpecBody(w *WindowSpec) {
	wrote := false
	if w.Name != "" {
		writeIdentPart(sw, w.Name)
		wrote = true
	}
	if len(w.PartitionBy) > 0 {
		if wrote {
			sw.str(" ")
		}
		sw.str("PARTITION BY ")
		sw.exprListTop(w.PartitionBy)
		wrote = true
	}
	if w.OrderBy != nil {
		if wrote {
			sw.str(" ")
		}
		sw.orderByClause(w.OrderBy)
		wrote = true
	}
	if w.Frame != nil {
		if wrote {
			sw.str(" ")
		}
		sw.frameClause(w.Frame)
	}
}

func (sw *sqlWriter) frameClause(f *FrameClause) {
	sw.str(f.Unit)
	sw.str(" ")
	if f.EndBound != nil {
		sw.str("BETWEEN ")
		sw.frameBound(f.Start)
		sw.str(" AND ")
		sw.frameBound(f.EndBound)
	} else {
		sw.frameBound(f.Start)
	}
	if f.Exclude != "" {
		sw.str(" EXCLUDE ")
		sw.str(f.Exclude)
	}
}

func (sw *sqlWriter) frameBound(b *FrameBound) {
	switch b.Kind {
	case FrameUnboundedPreceding:
		sw.str("UNBOUNDED PRECEDING")
	case FrameUnboundedFollowing:
		sw.str("UNBOUNDED FOLLOWING")
	case FrameCurrentRow:
		sw.str("CURRENT ROW")
	case FramePreceding:
		sw.exprTop(b.Expr)
		sw.str(" PRECEDING")
	case FrameFollowing:
		sw.exprTop(b.Expr)
		sw.str(" FOLLOWING")
	}
}

// --- ORDER BY / GROUP BY / DISTINCT / LIMIT ---

func (sw *sqlWriter) orderByClause(o *OrderByClause) {
	sw.str("ORDER BY ")
	if o.All {
		sw.str("ALL")
		if len(o.Items) > 0 {
			sw.orderBySuffix(o.Items[0])
		}
		return
	}
	for i, it := range o.Items {
		sw.sep(i, ", ")
		sw.exprTop(it.X)
		sw.orderBySuffix(it)
	}
}

func (sw *sqlWriter) orderBySuffix(it *OrderByItem) {
	if it.Desc {
		sw.str(" DESC")
	}
	if it.HasNulls {
		if it.NullsFirst {
			sw.str(" NULLS FIRST")
		} else {
			sw.str(" NULLS LAST")
		}
	}
}

func (sw *sqlWriter) groupByClause(g *GroupByClause) {
	sw.str("GROUP BY ")
	if g.All {
		sw.str("ALL")
		return
	}
	for i, it := range g.Items {
		sw.sep(i, ", ")
		sw.groupByItem(it)
	}
}

func (sw *sqlWriter) groupByItem(g GroupByItem) {
	switch v := g.(type) {
	case *GroupByExprItem:
		sw.exprTop(v.X)
	case *GroupByEmpty:
		sw.str("()")
	case *GroupByCube:
		sw.str("CUBE (")
		sw.exprListTop(v.Items)
		sw.str(")")
	case *GroupByRollup:
		sw.str("ROLLUP (")
		sw.exprListTop(v.Items)
		sw.str(")")
	case *GroupingSets:
		sw.str("GROUPING SETS (")
		for i, s := range v.Sets {
			sw.sep(i, ", ")
			sw.groupByItem(s)
		}
		sw.str(")")
	default:
		panic(fmt.Sprintf("duckdb/parser: Write: unhandled GroupByItem %T", g))
	}
}

func (sw *sqlWriter) distinctClause(d *DistinctClause) {
	sw.str("DISTINCT")
	if len(d.On) > 0 {
		sw.str(" ON (")
		sw.exprListTop(d.On)
		sw.str(")")
	}
}

// limitClause always normalizes to LIMIT/OFFSET syntax, even when the
// source used FETCH FIRST ... ROWS ONLY / OFFSET ... ROWS (adaptLimitOffset
// folds every LimitOffset alternative into the same LimitClause shape,
// discarding which spelling was used) — semantically identical, and this
// is the simpler, more universally-recognized DuckDB form.
func (sw *sqlWriter) limitClause(l *LimitClause) {
	sw.str("LIMIT ")
	if l.All {
		sw.str("ALL")
	} else {
		sw.exprTop(l.Limit)
	}
	if l.Percent {
		sw.str(" PERCENT")
	}
	if l.Offset != nil {
		sw.str(" OFFSET ")
		sw.exprTop(l.Offset)
	}
}

// --- WITH / CTE ---

func (sw *sqlWriter) withClause(w *WithClause) {
	sw.str("WITH ")
	if w.Recursive {
		sw.str("RECURSIVE ")
	}
	for i, c := range w.CTEs {
		sw.sep(i, ", ")
		sw.cte(c)
	}
}

func (sw *sqlWriter) cte(c *CTE) {
	writeIdentPart(sw, c.Name)
	if len(c.ColumnNames) > 0 {
		sw.str(" (")
		for i, n := range c.ColumnNames {
			sw.sep(i, ", ")
			writeIdentPart(sw, n)
		}
		sw.str(")")
	}
	sw.str(" AS ")
	if c.Materialized != nil {
		if *c.Materialized {
			sw.str("MATERIALIZED ")
		} else {
			sw.str("NOT MATERIALIZED ")
		}
	}
	sw.str("(")
	sw.selectStatement(c.Select)
	sw.str(")")
}

func (sw *sqlWriter) windowDef(w *WindowDef) {
	writeIdentPart(sw, w.Name)
	sw.str(" AS (")
	sw.windowSpecBody(w.Spec)
	sw.str(")")
}

// --- SELECT / FROM ---

func (sw *sqlWriter) selectItem(item *SelectItem) {
	sw.exprTop(item.Expr)
	if item.Alias != "" {
		sw.str(" AS ")
		writeIdentPart(sw, item.Alias)
	}
}

func (sw *sqlWriter) fromClause(f *FromClause) {
	for i, r := range f.Refs {
		sw.sep(i, ", ")
		sw.tableRef(r)
	}
}

func (sw *sqlWriter) tableAlias(alias string) {
	if alias != "" {
		sw.str(" AS ")
		writeIdentPart(sw, alias)
	}
}

func (sw *sqlWriter) tableRef(t TableRef) {
	switch v := t.(type) {
	case *BaseTableRef:
		writeDottedName(sw, v.Name)
		sw.tableAlias(v.Alias)
	case *TableFunctionRef:
		if v.Lateral {
			sw.str("LATERAL ")
		}
		writeDottedName(sw, v.Name)
		sw.str("(")
		sw.exprListTop(v.Args)
		sw.str(")")
		if v.WithOrdinality {
			sw.str(" WITH ORDINALITY")
		}
		sw.tableAlias(v.Alias)
	case *TableSubqueryRef:
		if v.Lateral {
			sw.str("LATERAL ")
		}
		sw.str("(")
		sw.selectStatement(v.Select)
		sw.str(")")
		sw.tableAlias(v.Alias)
	case *ParensTableRef:
		sw.str("(")
		sw.tableRef(v.Ref)
		sw.str(")")
		sw.tableAlias(v.Alias)
	case *JoinRef:
		sw.tableRef(v.Left)
		sw.str(" ")
		sw.str(v.Type)
		sw.str(" JOIN ")
		sw.tableRef(v.Right)
		if v.On != nil {
			sw.str(" ON ")
			sw.exprTop(v.On)
		} else if len(v.Using) > 0 {
			sw.str(" USING (")
			for i, u := range v.Using {
				sw.sep(i, ", ")
				writeIdentPart(sw, u)
			}
			sw.str(")")
		}
	case *PivotRef:
		sw.pivotRef(v)
	case *UnpivotRef:
		sw.unpivotRef(v)
	default:
		panic(fmt.Sprintf("duckdb/parser: Write: unhandled TableRef %T", t))
	}
}

// pivotColumn omits "IN (...)" entirely when In is empty — the bare "ON
// col" form of the top-level PIVOT statement (PivotColumnExpression in
// adaptPivotColumnEntry) never populates In at all, asking DuckDB to
// auto-detect the pivot values; only the table-suffix form's "FOR col IN
// (...)" (PivotValueList) always has at least one entry, since its IN
// clause isn't optional in the grammar.
func (sw *sqlWriter) pivotColumn(p *PivotColumn) {
	sw.exprTop(p.Header)
	if len(p.In) == 0 {
		return
	}
	sw.str(" IN (")
	for i, v := range p.In {
		sw.sep(i, ", ")
		sw.str(v) // already-formatted SQL text (or a bare name) — see adaptPivotValueList
	}
	sw.str(")")
}

// pivotRef prints the table-suffix "source PIVOT (cols FOR col1 IN (...),
// ...)" form when every For entry has an explicit IN list, since that's
// the only shape TablePivotClause's grammar accepts (its IN is mandatory,
// unlike PivotOn's — see pivotColumn's comment). When some entry has no IN
// list, it can only have come from the top-level PIVOT statement (bare
// "ON col" form), which isn't embeddable as a table suffix at all — that
// shape is handled by isPivotOrUnpivotStatement/simpleSelectCore instead,
// printing "PIVOT source ON ..." with no FROM keyword and no alias slot,
// so pivotRef itself is never called in that case.
func (sw *sqlWriter) pivotRef(v *PivotRef) {
	sw.tableRef(v.Source)
	sw.str(" PIVOT (")
	for i, c := range v.Columns {
		sw.sep(i, ", ")
		sw.selectItem(c)
	}
	sw.str(" FOR ")
	for i, p := range v.For {
		sw.sep(i, ", ")
		sw.pivotColumn(p)
	}
	if len(v.GroupBy) > 0 {
		sw.str(" GROUP BY ")
		for i, g := range v.GroupBy {
			sw.sep(i, ", ")
			writeIdentPart(sw, g)
		}
	}
	sw.str(")")
	sw.tableAlias(v.Alias)
}

// pivotStatement prints the top-level "PIVOT source ON ..." form —
// PivotKeyword comes before the source TableRef here (unlike the
// table-suffix form's "source PIVOT (...)"), and there's no alias slot
// (adaptPivotStatement never sets one).
func (sw *sqlWriter) pivotStatement(v *PivotRef) {
	sw.str("PIVOT ")
	sw.tableRef(v.Source)
	sw.str(" ON ")
	for i, p := range v.For {
		sw.sep(i, ", ")
		sw.pivotColumn(p)
	}
	if len(v.Columns) > 0 {
		sw.str(" USING ")
		for i, c := range v.Columns {
			sw.sep(i, ", ")
			sw.selectItem(c)
		}
	}
	if len(v.GroupBy) > 0 {
		sw.str(" GROUP BY ")
		for i, g := range v.GroupBy {
			sw.sep(i, ", ")
			writeIdentPart(sw, g)
		}
	}
}

// unpivotStatement prints the top-level "UNPIVOT source ON columns" form —
// UnpivotKeyword before the source TableRef, no header/name/alias (none of
// those exist in this grammar shape; adaptUnpivotStatement never sets
// them).
func (sw *sqlWriter) unpivotStatement(v *UnpivotRef) {
	sw.str("UNPIVOT ")
	sw.tableRef(v.Source)
	sw.str(" ON ")
	for i, c := range v.Columns {
		sw.sep(i, ", ")
		sw.selectItem(c)
	}
}

// isPivotOrUnpivotStatement reports whether f is really a top-level PIVOT
// or UNPIVOT statement in FromClause disguise: adaptPivotStatement and
// adaptUnpivotStatement both return a bare *SelectStatement with only From
// set (see adapter_pivot.go), the same shape a genuine bare "FROM ..."
// query produces — but PivotStatement/UnpivotStatement's grammar puts the
// keyword *before* the source TableRef with no FROM at all, so this case
// can't be printed by prefixing "FROM " onto the ref the way a real bare
// FROM can. The signal that disambiguates: PivotRef's bare "ON col" (no IN
// list) and UnpivotRef's statement form (no Header) are unreachable from
// the table-suffix grammar, which requires them (see pivotRef/pivotColumn
// and unpivotRef's comments) — so their presence here is conclusive.
func isPivotOrUnpivotStatement(f *FromClause) (Node, bool) {
	if f == nil || len(f.Refs) != 1 {
		return nil, false
	}
	switch v := f.Refs[0].(type) {
	case *PivotRef:
		for _, p := range v.For {
			if len(p.In) == 0 {
				return v, true
			}
		}
	case *UnpivotRef:
		if len(v.Header) == 0 {
			return v, true
		}
	}
	return nil, false
}

// unpivotRef prints the table-suffix "source UNPIVOT (header FOR name IN
// (columns))" form — the only shape TableUnpivotClause's grammar accepts.
// The top-level "UNPIVOT source ON columns" statement form (empty Header —
// see isPivotOrUnpivotStatement) isn't embeddable as a table suffix at
// all, so unpivotStatement handles that shape instead; unpivotRef is never
// called for it. UnpivotRef also already flattens every "FOR name IN
// (...)" group's columns together and loses which columns belong to which
// Names entry when there's more than one group (see UnpivotRef's field
// comment in ast.go) — this writer matches that same single-group
// assumption rather than trying to recover grouping the AST no longer has.
func (sw *sqlWriter) unpivotRef(v *UnpivotRef) {
	sw.tableRef(v.Source)
	sw.str(" UNPIVOT")
	if v.IncludeNulls {
		sw.str(" INCLUDE NULLS")
	}
	if v.ExcludeNulls {
		sw.str(" EXCLUDE NULLS")
	}
	sw.str(" (")
	writeNameList(sw, v.Header)
	sw.str(" FOR ")
	writeNameList(sw, v.Names)
	sw.str(" IN (")
	for i, c := range v.Columns {
		sw.sep(i, ", ")
		sw.selectItem(c)
	}
	sw.str("))")
}

// --- SelectStatement ---

func (sw *sqlWriter) selectStatement(s *SelectStatement) {
	if s.With != nil {
		sw.withClause(s.With)
		sw.str(" ")
	}

	switch {
	case s.SetOp != SetOpNone:
		sw.setOperand(s.SetLeft, s.SetOp)
		sw.str(" ")
		sw.str(s.SetOp.String())
		if s.SetAll {
			sw.str(" ALL")
		}
		if s.SetByName {
			sw.str(" BY NAME")
		}
		sw.str(" ")
		sw.setOperand(s.SetRight, s.SetOp)
	case s.Values != nil:
		sw.str("VALUES ")
		for i, row := range s.Values {
			sw.sep(i, ", ")
			sw.str("(")
			sw.exprListTop(row)
			sw.str(")")
		}
	default:
		sw.simpleSelectCore(s)
	}

	if s.Where != nil {
		sw.str(" WHERE ")
		sw.exprTop(s.Where)
	}
	if s.GroupBy != nil {
		sw.str(" ")
		sw.groupByClause(s.GroupBy)
	}
	if s.Having != nil {
		sw.str(" HAVING ")
		sw.exprTop(s.Having)
	}
	for i, wd := range s.Windows {
		if i == 0 {
			sw.str(" WINDOW ")
		} else {
			sw.str(", ")
		}
		sw.windowDef(wd)
	}
	if s.Qualify != nil {
		sw.str(" QUALIFY ")
		sw.exprTop(s.Qualify)
	}
	if s.OrderBy != nil {
		sw.str(" ")
		sw.orderByClause(s.OrderBy)
	}
	if s.Limit != nil {
		sw.str(" ")
		sw.limitClause(s.Limit)
	}
}

func (sw *sqlWriter) simpleSelectCore(s *SelectStatement) {
	if s.Columns == nil && s.Distinct == nil {
		if ref, ok := isPivotOrUnpivotStatement(s.From); ok {
			switch v := ref.(type) {
			case *PivotRef:
				sw.pivotStatement(v)
			case *UnpivotRef:
				sw.unpivotStatement(v)
			}
			return
		}
		// Bare "FROM ..." — also what a PIVOT/UNPIVOT-as-statement with an
		// explicit IN list adapts to (adaptPivotStatement/
		// adaptUnpivotStatement only ever set From), which DuckDB's
		// "FROM-first" syntax expresses identically to "SELECT * FROM ...".
		if s.From != nil {
			sw.str("FROM ")
			sw.fromClause(s.From)
		}
		return
	}
	sw.str("SELECT")
	if s.Distinct != nil {
		sw.str(" ")
		sw.distinctClause(s.Distinct)
	}
	if len(s.Columns) > 0 {
		sw.str(" ")
		for i, c := range s.Columns {
			sw.sep(i, ", ")
			sw.selectItem(c)
		}
	}
	if s.From != nil {
		sw.str(" FROM ")
		sw.fromClause(s.From)
	}
}

// setOperand prints one side of a UNION/INTERSECT/EXCEPT combination,
// adding parens when needed to reproduce the original grouping: when child
// carries its own WITH/ORDER BY/LIMIT (those only apply to a single
// operand in the source if it was itself parenthesized — see
// adaptSelectAtom's SelectParens case), or when parentOp is the
// tighter-binding INTERSECT and child is the looser UNION/EXCEPT (see
// adaptSelectSetOpChain / adaptIntersectChain's two-tier precedence).
func (sw *sqlWriter) setOperand(child *SelectStatement, parentOp SetOp) {
	needsParens := child.With != nil || child.OrderBy != nil || child.Limit != nil ||
		(parentOp == SetOpIntersect && (child.SetOp == SetOpUnion || child.SetOp == SetOpExcept))
	if needsParens {
		sw.str("(")
		sw.selectStatement(child)
		sw.str(")")
		return
	}
	sw.selectStatement(child)
}
