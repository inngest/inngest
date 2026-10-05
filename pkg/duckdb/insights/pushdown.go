package insights

import (
	"strconv"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/federate"
	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// Pushdown is the pre-filter predicates extracted per logical table, for a
// federated executor to push into each table's delta source (e.g. the
// ClickHouse buffer). Predicates are pre-filters only: the query still
// applies its full WHERE, so a source may ignore any it can't translate
// exactly. Column names are the logical table's own; a source must only
// push a column whose stored value equals what the logical table exposes.
type Pushdown map[federate.Table][]federate.Predicate

// extractPushdown returns the predicates that are safe to push for each
// logical table stmt reads.
//
// A table gets predicates only when stmt scans it exactly once (one
// BaseTableRef, anywhere: CTE bodies, set-op operands, FROM and
// expression-position subqueries all count). Its rows then enter the query
// only through that one FROM, so every conjunct of that SELECT's WHERE must
// hold for any row that matters, whatever encloses the SELECT (joins,
// EXISTS / NOT IN, CTEs, set ops): dropping the rows that fail it early
// can't change the result. A second scan would need the OR of both scans'
// filters (or none), so it pushes nothing for that table.
//
// Only top-level AND conjuncts of these shapes are pushed, all comparing a
// column of that table against non-NULL literals of the column's own type
// (string, integer, boolean):
//
//	col = | <> | != | < | <= | > | >= lit   (either side)
//	col IN (lit, ...)
//	col BETWEEN lit AND lit
//
// Each is null-rejecting (NULL never satisfies it), which is what keeps
// pushing it correct when the table sits on either side of an outer join:
// a row it drops early would have been dropped after the join too. OR, NOT,
// functions, casts, JSON paths, NULL tests, datetime columns (the query's
// time bound is extracted separately) and anything ambiguous are not
// pushed.
func extractPushdown(stmt *parser.SelectStatement, tables map[string]logicalTable) Pushdown {
	w := &pushdownWalk{tables: tables, counts: map[string]int{}, shadowed: map[string]bool{}}
	w.walk(stmt)

	out := Pushdown{}
	for _, c := range w.candidates {
		if w.counts[c.table] != 1 || w.shadowed[c.table] {
			continue
		}
		t := federate.Table(c.table)
		out[t] = append(out[t], c.pred)
	}
	return out
}

type pushdownCandidate struct {
	table string
	pred  federate.Predicate
}

type pushdownWalk struct {
	// tables is the transpile mode's catalog.
	tables map[string]logicalTable
	// counts is the number of scans of each logical table.
	counts map[string]int
	// shadowed marks logical table names some CTE redeclares. Rather than
	// tracking CTE scope precisely, such a table never gets predicates.
	shadowed   map[string]bool
	candidates []pushdownCandidate
}

// walk visits every node under n, counting logical table scans and
// collecting candidate predicates from every SELECT's WHERE.
func (w *pushdownWalk) walk(n parser.Node) {
	if n == nil {
		return
	}
	switch x := n.(type) {
	case *parser.CTE:
		if _, ok := w.tables[x.Name]; ok {
			w.shadowed[x.Name] = true
		}
	case *parser.BaseTableRef:
		if len(x.Name) == 1 {
			if _, ok := w.tables[x.Name[0]]; ok {
				w.counts[x.Name[0]]++
			}
		}
	case *parser.SelectStatement:
		w.collect(x)
	}
	for _, c := range n.Children() {
		w.walk(c)
	}
}

// pushdownEntry is one FROM item of a SELECT. table is "" when the item
// isn't a logical table (a CTE reference, subquery or table function),
// whose columns are unknown here.
type pushdownEntry struct {
	alias string
	table string
	cols  map[string]knownColumn
}

func (w *pushdownWalk) collect(s *parser.SelectStatement) {
	if s.Where == nil || s.From == nil {
		return
	}
	var entries []pushdownEntry
	for _, ref := range s.From.Refs {
		entries = appendPushdownEntries(w.tables, entries, ref)
	}
	for _, conj := range conjuncts(s.Where) {
		for _, c := range pushdownPredicates(conj, entries) {
			w.candidates = append(w.candidates, c)
		}
	}
}

func appendPushdownEntries(tables map[string]logicalTable, entries []pushdownEntry, ref parser.TableRef) []pushdownEntry {
	switch r := ref.(type) {
	case *parser.BaseTableRef:
		e := pushdownEntry{alias: r.Alias}
		if len(r.Name) == 1 {
			if e.alias == "" {
				e.alias = r.Name[0]
			}
			if tbl, ok := tables[r.Name[0]]; ok {
				e.table, e.cols = r.Name[0], tbl.columns
			}
		}
		return append(entries, e)
	case *parser.JoinRef:
		entries = appendPushdownEntries(tables, entries, r.Left)
		return appendPushdownEntries(tables, entries, r.Right)
	case *parser.TableSubqueryRef:
		return append(entries, pushdownEntry{alias: r.Alias})
	default:
		// Table functions, parenthesized refs, PIVOT, ...: an entry with
		// unknown columns, so unqualified names stay ambiguous.
		return append(entries, pushdownEntry{})
	}
}

// conjuncts flattens a tree of ANDs into its operands.
func conjuncts(e parser.Expr) []parser.Expr {
	if b, ok := e.(*parser.BinaryExpr); ok && strings.EqualFold(b.Op, "AND") {
		return append(conjuncts(b.Left), conjuncts(b.Right)...)
	}
	return []parser.Expr{e}
}

var pushdownOps = map[string]federate.Op{
	"=": federate.OpEq, "==": federate.OpEq,
	"<>": federate.OpNeq, "!=": federate.OpNeq,
	"<": federate.OpLt, "<=": federate.OpLte,
	">": federate.OpGt, ">=": federate.OpGte,
}

// flippedOps maps an operator to its mirror, for lit OP col.
var flippedOps = map[federate.Op]federate.Op{
	federate.OpEq: federate.OpEq, federate.OpNeq: federate.OpNeq,
	federate.OpLt: federate.OpGt, federate.OpLte: federate.OpGte,
	federate.OpGt: federate.OpLt, federate.OpGte: federate.OpLte,
}

// pushdownPredicates translates one conjunct, or returns nil when it isn't
// a pushable shape. A comparison, IN or BETWEEN (two bounds) yields flat
// predicates; an OR, NOT, LIKE or NOT IN yields one predicate tree
// (pushdownExpr).
func pushdownPredicates(e parser.Expr, entries []pushdownEntry) []pushdownCandidate {
	if c := leafPredicates(e, entries); c != nil {
		return c
	}
	if table, pred, ok := pushdownExpr(e, entries); ok {
		return []pushdownCandidate{{table: table, pred: pred}}
	}
	return nil
}

// pushdownExpr translates e into one predicate on one logical table: a
// comparison, IN, BETWEEN (an AND of its bounds) or LIKE, or an AND, OR or
// NOT of those, every part on the same table. It fails if any part doesn't
// translate: dropping an OR's branch or a NOT's operand would change which
// rows it keeps. LIKE translates without an ESCAPE clause, its pattern a
// string literal; ILIKE, GLOB and SIMILAR TO don't.
func pushdownExpr(e parser.Expr, entries []pushdownEntry) (string, federate.Predicate, bool) {
	combine := func(op federate.Op, parts ...parser.Expr) (string, federate.Predicate, bool) {
		var table string
		args := make([]federate.Predicate, 0, len(parts))
		for _, part := range parts {
			t, p, ok := pushdownExpr(part, entries)
			if !ok || (table != "" && t != table) {
				return "", federate.Predicate{}, false
			}
			table = t
			if p.Op == op && op != federate.OpNot {
				args = append(args, p.Args...) // flatten a AND (b AND c)
			} else {
				args = append(args, p)
			}
		}
		return table, federate.Predicate{Op: op, Args: args}, true
	}
	switch x := e.(type) {
	case *parser.BinaryExpr:
		switch {
		case strings.EqualFold(x.Op, "AND"):
			return combine(federate.OpAnd, x.Left, x.Right)
		case strings.EqualFold(x.Op, "OR"):
			return combine(federate.OpOr, x.Left, x.Right)
		}
	case *parser.UnaryExpr:
		if strings.EqualFold(x.Op, "NOT") {
			return combine(federate.OpNot, x.X)
		}
	case *parser.InExpr:
		if x.Not {
			in := *x
			in.Not = false
			return combine(federate.OpNot, &in)
		}
	case *parser.BetweenExpr:
		if x.Not {
			b := *x
			b.Not = false
			return combine(federate.OpNot, &b)
		}
	case *parser.LikeExpr:
		if !strings.EqualFold(x.Op, "LIKE") || x.Escape != nil {
			return "", federate.Predicate{}, false
		}
		table, col, path, kc, ok := resolvePushdownTarget(x.X, entries)
		if !ok || (path == "" && kc.colType != ColumnTypeString) {
			return "", federate.Predicate{}, false
		}
		lit, ok := x.Pattern.(*parser.Literal)
		if !ok || lit.Kind != parser.LitString {
			return "", federate.Predicate{}, false
		}
		like := federate.Predicate{Column: col, Path: path, Op: federate.OpLike, Value: lit.Text}
		if x.Not {
			like = federate.Predicate{Op: federate.OpNot, Args: []federate.Predicate{like}}
		}
		return table, like, true
	}
	c := leafPredicates(e, entries)
	switch len(c) {
	case 1:
		return c[0].table, c[0].pred, true
	case 2: // BETWEEN's bounds
		return c[0].table, federate.Predicate{Op: federate.OpAnd, Args: []federate.Predicate{c[0].pred, c[1].pred}}, true
	}
	return "", federate.Predicate{}, false
}

// leafPredicates translates a comparison, IN or BETWEEN, or returns nil.
// BETWEEN yields two predicates.
func leafPredicates(e parser.Expr, entries []pushdownEntry) []pushdownCandidate {
	switch x := e.(type) {
	case *parser.BinaryExpr:
		op, ok := pushdownOps[x.Op]
		if !ok {
			return nil
		}
		colExpr, litExpr := x.Left, x.Right
		if !isPushdownTarget(colExpr) {
			colExpr, litExpr, op = x.Right, x.Left, flippedOps[op]
		}
		table, col, path, kc, ok := resolvePushdownTarget(colExpr, entries)
		if !ok {
			return nil
		}
		if path != "" && op != federate.OpEq && op != federate.OpNeq {
			return nil
		}
		v, ok := targetValue(litExpr, kc, path)
		if !ok {
			return nil
		}
		return []pushdownCandidate{{table: table, pred: federate.Predicate{Column: col, Path: path, Op: op, Value: v}}}

	case *parser.InExpr:
		if x.Not || x.Contains || x.Subquery != nil || len(x.List) == 0 {
			return nil
		}
		table, col, path, kc, ok := resolvePushdownTarget(x.X, entries)
		if !ok {
			return nil
		}
		vals := make([]any, len(x.List))
		for i, item := range x.List {
			if vals[i], ok = targetValue(item, kc, path); !ok {
				return nil
			}
		}
		return []pushdownCandidate{{table: table, pred: federate.Predicate{Column: col, Path: path, Op: federate.OpIn, Value: vals}}}

	case *parser.BetweenExpr:
		if x.Not {
			return nil
		}
		table, col, kc, ok := resolvePushdownColumn(x.X, entries)
		if !ok {
			return nil
		}
		low, okLow := pushdownValue(x.Low, kc)
		high, okHigh := pushdownValue(x.High, kc)
		if !okLow || !okHigh {
			return nil
		}
		return []pushdownCandidate{
			{table: table, pred: federate.Predicate{Column: col, Op: federate.OpGte, Value: low}},
			{table: table, pred: federate.Predicate{Column: col, Op: federate.OpLte, Value: high}},
		}
	}
	return nil
}

// resolvePushdownColumn resolves a column reference to its logical table
// within this SELECT's own FROM. alias.col must name a local logical-table
// entry. A bare col must belong to exactly one local entry, with no
// unknown-column entry (CTE, subquery, table function) that might also have
// it. Anything else (including a correlated reference to an outer query)
// doesn't resolve.
func resolvePushdownColumn(e parser.Expr, entries []pushdownEntry) (table, col string, kc knownColumn, ok bool) {
	id, isIdent := e.(*parser.Ident)
	if !isIdent {
		return "", "", knownColumn{}, false
	}
	switch len(id.Parts) {
	case 1:
		col = id.Parts[0]
		for _, en := range entries {
			if en.table == "" {
				return "", "", knownColumn{}, false
			}
			if c, has := en.cols[col]; has {
				if table != "" {
					return "", "", knownColumn{}, false
				}
				table, kc = en.table, c
			}
		}
		return table, col, kc, table != ""
	case 2:
		alias, col := id.Parts[0], id.Parts[1]
		for _, en := range entries {
			if en.alias != alias {
				continue
			}
			if en.table == "" {
				return "", "", knownColumn{}, false
			}
			c, has := en.cols[col]
			if !has {
				return "", "", knownColumn{}, false
			}
			return en.table, col, c, true
		}
	}
	return "", "", knownColumn{}, false
}

// pushdownValue converts a literal to a predicate value, only when its type
// matches the column's exactly, so the source compares like DuckDB does
// (no implicit casts).
func pushdownValue(e parser.Expr, kc knownColumn) (any, bool) {
	neg := false
	if u, ok := e.(*parser.UnaryExpr); ok && u.Op == "-" {
		neg, e = true, u.X
	}
	lit, ok := e.(*parser.Literal)
	if !ok {
		return nil, false
	}
	switch kc.colType {
	case ColumnTypeString:
		if lit.Kind == parser.LitString && !neg {
			return lit.Text, true
		}
	case ColumnTypeBoolean:
		if !neg && (lit.Kind == parser.LitTrue || lit.Kind == parser.LitFalse) {
			return lit.Kind == parser.LitTrue, true
		}
	case ColumnTypeNumber:
		if lit.Kind != parser.LitNumber {
			return nil, false
		}
		n, err := strconv.ParseInt(lit.Text, 10, 64)
		if err != nil {
			// Decimals, exponents, hex, out-of-range: not exact enough.
			return nil, false
		}
		if neg {
			n = -n
		}
		return n, true
	}
	return nil, false
}

// isPushdownTarget reports whether e is a shape a predicate can filter on: a
// column reference, or a text extraction of one key of a JSON column.
func isPushdownTarget(e parser.Expr) bool {
	if _, ok := e.(*parser.Ident); ok {
		return true
	}
	_, _, ok := jsonKeyText(e)
	return ok
}

// resolvePushdownTarget resolves a column reference (path "") or a JSON key
// extraction (col->>'key', json_extract_string(col, 'key')) on a JSON
// column of this SELECT's own FROM. Only a plain top-level key is taken,
// not a $-JSONPath.
func resolvePushdownTarget(e parser.Expr, entries []pushdownEntry) (table, col, path string, kc knownColumn, ok bool) {
	if base, key, isPath := jsonKeyText(e); isPath {
		table, col, kc, ok = resolvePushdownColumn(base, entries)
		if !ok || kc.colType != ColumnTypeJSON {
			return "", "", "", knownColumn{}, false
		}
		return table, col, key, kc, true
	}
	table, col, kc, ok = resolvePushdownColumn(e, entries)
	return table, col, "", kc, ok
}

// jsonKeyText recognizes the text extractions of one plain JSON key.
func jsonKeyText(e parser.Expr) (base parser.Expr, key string, ok bool) {
	switch x := e.(type) {
	case *parser.BinaryExpr:
		if x.Op != "->>" {
			return nil, "", false
		}
	case *parser.FunctionExpr:
		if strings.ToLower(strings.Join(x.Name, ".")) != "json_extract_string" {
			return nil, "", false
		}
	default:
		return nil, "", false
	}
	base, key, ok = jsonPathAccess(e)
	if !ok || key == "" || strings.HasPrefix(key, "$") {
		return nil, "", false
	}
	return base, key, true
}

// targetValue is pushdownValue for a column, or a string literal for a JSON
// key (its value is compared as text).
func targetValue(e parser.Expr, kc knownColumn, path string) (any, bool) {
	if path == "" {
		return pushdownValue(e, kc)
	}
	lit, ok := e.(*parser.Literal)
	if !ok || lit.Kind != parser.LitString {
		return nil, false
	}
	return lit.Text, true
}
