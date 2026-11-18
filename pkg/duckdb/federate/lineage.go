package federate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/parser"
	"github.com/inngest/inngest/pkg/duckdb/schema"
)

// errLineage is a query shape the column lineage doesn't model. It's never
// returned to a caller: the executor then streams every delta column.
var errLineage = errors.New("federate: column lineage")

// demand is the columns of a relation something reads: every one, or the
// named ones (lowercased).
type demand struct {
	all   bool
	names map[string]bool
}

func allColumns() demand { return demand{all: true} }

func (d *demand) add(name string) {
	if d.names == nil {
		d.names = map[string]bool{}
	}
	d.names[strings.ToLower(name)] = true
}

func (d *demand) merge(o demand) {
	d.all = d.all || o.all
	for n := range o.names {
		d.add(n)
	}
}

// sourceColumnsRead works out, from the query's AST alone, which columns
// it reads from each needed table's source CTE (SourceName): a top-down
// pass from "every column the query returns" through its CTEs,
// subqueries, joins and the views.sql macros (incl. query_table) to the
// source CTEs, which it doesn't descend into. A column the query only
// filters, joins, groups or sorts on counts, as do the macros' own reads
// (IE raw_runs' collapse).
//
// It errs (errLineage) on anything it doesn't model, and the caller then
// streams every column: an ambiguous name only ever adds columns (it's
// pushed to every relation it could be), never drops one.
func sourceColumnsRead(stmt *parser.SelectStatement, needed []Table) (map[Table]demand, error) {
	macros, err := duckdb.TableMacros()
	if err != nil {
		return nil, err
	}
	budget := lineageBudget
	l := &lineage{macros: macros, sources: map[string]Table{}, lake: map[string]Table{}, out: map[Table]demand{}, budget: &budget, names: map[*cteDef][]string{}}
	for _, t := range needed {
		l.sources[strings.ToLower(SourceName(t))] = t
	}
	for t, phys := range Physical {
		l.lake[strings.ToLower(phys.Name)] = t
	}
	l.analyze(stmt, &env{}, nil, allColumns())
	if l.err != nil {
		return nil, l.err
	}
	return l.out, nil
}

type lineage struct {
	macros  map[string]*duckdb.TableMacro
	sources map[string]Table
	// lake is the lake tables by physical name: a macro's default source
	// reads one directly. They're known but have no delta.
	lake map[string]Table
	out  map[Table]demand
	err  error
	// depth guards against a macro that (somehow) expands into itself.
	depth int
	// budget is how many more SELECTs it may analyze, shared w/ probes:
	// each reference to a CTE re-analyzes its body, so a query chaining
	// CTEs that each read the one before twice is exponential. Past it,
	// the lineage gives up (and every column streams).
	budget *int
	// names memoizes a CTE's output names.
	names map[*cteDef][]string
}

// lineageBudget bounds the SELECTs one lineage analyzes. The views.sql
// macros under a query take ~20.
const lineageBudget = 2000

// spend takes one SELECT off the budget, false once it's gone.
func (l *lineage) spend() bool {
	*l.budget--
	if *l.budget < 0 {
		l.fail("query too complex")
		return false
	}
	return true
}

func (l *lineage) fail(format string, args ...any) {
	if l.err == nil {
		l.err = fmt.Errorf("%w: %s", errLineage, fmt.Sprintf(format, args...))
	}
}

// env is what a name resolves against besides the FROM clause: the CTEs
// in scope and, inside a macro body, its parameters' bindings.
type env struct {
	parent *env
	ctes   map[string]*cteDef
	params map[string]parser.Expr
}

type cteDef struct {
	cte *parser.CTE
	env *env
}

func (e *env) cte(name string) (*cteDef, bool) {
	for ; e != nil; e = e.parent {
		if d, ok := e.ctes[strings.ToLower(name)]; ok {
			return d, true
		}
	}
	return nil, false
}

func (e *env) param(name string) (parser.Expr, bool) {
	for ; e != nil; e = e.parent {
		if p, ok := e.params[strings.ToLower(name)]; ok {
			return p, true
		}
	}
	return nil, false
}

// rel is one FROM item: its alias, its output columns (lowercased, in
// order; nil when unknown) and how to pass a demand on them down.
type rel struct {
	alias string
	cols  []string
	need  demand
	flush func(demand)
}

func (r *rel) has(col string) bool {
	if r.cols == nil {
		return true // unknown: it could be any column
	}
	for _, c := range r.cols {
		if c == strings.ToLower(col) {
			return true
		}
	}
	return false
}

// scope is one SELECT's FROM clause, its select list (for aliases the
// other clauses can reference) and its enclosing query's scope (for a
// correlated subquery).
type scope struct {
	rels  []*rel
	outer *scope
	env   *env
	items map[string]parser.Expr
	// bound are lambda and list comprehension variables, innermost last.
	bound []string
}

// analyze passes demand d on stmt's output columns down to whatever stmt
// reads.
func (l *lineage) analyze(stmt *parser.SelectStatement, e *env, outer *scope, d demand) {
	if l.err != nil || stmt == nil || !l.spend() {
		return
	}
	if stmt.With != nil {
		if stmt.With.Recursive {
			l.fail("recursive CTE")
			return
		}
		e = l.withCTEs(stmt.With, e)
	}
	if stmt.SetOp != parser.SetOpNone {
		// Positional (or by name) across the operands, and INTERSECT/
		// EXCEPT compare whole rows: every column of both sides.
		l.analyze(stmt.SetLeft, e, outer, allColumns())
		l.analyze(stmt.SetRight, e, outer, allColumns())
		return
	}
	if stmt.Values != nil {
		sc := &scope{outer: outer, env: e}
		for _, row := range stmt.Values {
			for _, x := range row {
				l.expr(x, sc)
			}
		}
		return
	}

	sc := l.fromScope(stmt.From, e, outer)
	if l.err != nil {
		return
	}
	sc.items = map[string]parser.Expr{}
	for _, it := range stmt.Columns {
		if it.Alias != "" {
			sc.items[strings.ToLower(it.Alias)] = it.Expr
		}
	}

	// Every select item matters when the select list itself decides the
	// rows (DISTINCT, GROUP BY ALL) or is referenced by position.
	whole := d.all || stmt.Distinct != nil ||
		stmt.GroupBy != nil && stmt.GroupBy.All ||
		stmt.OrderBy != nil && stmt.OrderBy.All ||
		positional(stmt)
	outs := l.outputs(stmt, sc)
	matched := map[string]bool{}
	var unnamed []output
	for _, o := range outs {
		if o.name == "" {
			unnamed = append(unnamed, o)
		}
		if whole || d.names[o.name] {
			l.output(o, sc)
			matched[o.name] = true
		}
	}
	if !whole {
		for n := range d.names {
			if matched[n] {
				continue
			}
			// A name none of the outputs we can name has: it's one DuckDB
			// generated (IE "count_star()"), so it's one of the unnamed
			// ones.
			if len(unnamed) == 0 {
				l.fail("no output column %q", n)
				return
			}
			for _, o := range unnamed {
				l.output(o, sc)
			}
			break
		}
	}

	l.expr(stmt.Where, sc)
	if stmt.GroupBy != nil {
		l.walk(stmt.GroupBy, sc)
	}
	l.expr(stmt.Having, sc)
	for _, w := range stmt.Windows {
		l.walk(w, sc)
	}
	l.expr(stmt.Qualify, sc)
	if stmt.OrderBy != nil {
		l.walk(stmt.OrderBy, sc)
	}
	if stmt.Distinct != nil {
		l.walk(stmt.Distinct, sc)
	}
	if stmt.Limit != nil {
		l.walk(stmt.Limit, sc)
	}
	// Last first: a LATERAL item reads the ones before it, adding to
	// their demand when it's flushed.
	for i := len(sc.rels) - 1; i >= 0; i-- {
		if l.err != nil {
			return
		}
		sc.rels[i].flush(sc.rels[i].need)
	}
}

func (l *lineage) withCTEs(w *parser.WithClause, e *env) *env {
	inner := &env{parent: e, ctes: map[string]*cteDef{}}
	for _, c := range w.CTEs {
		// A CTE sees the ones before it (and not itself: no RECURSIVE).
		before := &env{parent: inner.parent, ctes: map[string]*cteDef{}}
		for k, v := range inner.ctes {
			before.ctes[k] = v
		}
		inner.ctes[strings.ToLower(c.Name)] = &cteDef{cte: c, env: before}
	}
	return inner
}

// positional reports whether stmt's GROUP BY or ORDER BY refers to a
// select item by its position.
func positional(stmt *parser.SelectStatement) bool {
	isNum := func(x parser.Expr) bool {
		if _, ok := x.(*parser.PositionalExpr); ok {
			return true // #n
		}
		lit, ok := x.(*parser.Literal)
		return ok && lit.Kind == parser.LitNumber
	}
	if stmt.OrderBy != nil {
		for _, it := range stmt.OrderBy.Items {
			if isNum(it.X) {
				return true
			}
		}
	}
	if stmt.GroupBy != nil {
		for _, it := range stmt.GroupBy.Items {
			if g, ok := it.(*parser.GroupByExprItem); ok && isNum(g.X) {
				return true
			}
		}
	}
	return false
}

// output is one column a SELECT returns: a select item, or one column a
// star expands to (from rel, or wholly unknown when rel's columns are).
type output struct {
	name string
	expr parser.Expr
	// star is set for a star's column: col of rel, or every column of rel
	// when rel's columns are unknown (col "").
	star *rel
	col  string
}

func (l *lineage) output(o output, sc *scope) {
	switch {
	case o.expr != nil:
		l.expr(o.expr, sc)
	case o.star != nil && o.col == "":
		o.star.need.merge(allColumns())
	case o.star != nil:
		o.star.need.add(o.col)
	}
}

// outputs is stmt's output columns, a star expanded over sc's relations
// w/ its EXCLUDE/REPLACE/RENAME applied.
func (l *lineage) outputs(stmt *parser.SelectStatement, sc *scope) []output {
	var out []output
	for _, it := range stmt.Columns {
		star, ok := it.Expr.(*parser.StarExpr)
		if !ok {
			out = append(out, output{name: lineageOutputName(it), expr: it.Expr})
			continue
		}
		out = append(out, l.starOutputs(star, sc)...)
	}
	return out
}

func lineageOutputName(it *parser.SelectItem) string {
	if it.Alias != "" {
		return strings.ToLower(it.Alias)
	}
	switch x := it.Expr.(type) {
	case *parser.Ident:
		return strings.ToLower(x.Parts[len(x.Parts)-1])
	case *parser.DotExpr:
		return strings.ToLower(x.Field)
	}
	return ""
}

func (l *lineage) starOutputs(star *parser.StarExpr, sc *scope) []output {
	var out []output
	for _, r := range sc.rels {
		if len(star.Qualifier) > 0 && !strings.EqualFold(star.Qualifier[len(star.Qualifier)-1], r.alias) {
			continue
		}
		if r.cols == nil {
			out = append(out, output{star: r})
			continue
		}
		for _, c := range r.cols {
			if starExcludes(star, r.alias, c) {
				continue
			}
			o := output{name: c, star: r, col: c}
			for _, rep := range star.Replace {
				if strings.EqualFold(rep.Alias, c) {
					o.star, o.expr = nil, rep.Expr
				}
			}
			for _, ren := range star.Rename {
				if starNameMatches(ren.From, r.alias, c) {
					o.name = strings.ToLower(ren.To)
				}
			}
			out = append(out, o)
		}
	}
	return out
}

func starExcludes(star *parser.StarExpr, alias, col string) bool {
	for i, x := range star.Exclude {
		parts := []string{x}
		if star.ExcludeParts != nil {
			parts = star.ExcludeParts[i]
		}
		if starNameMatches(parts, alias, col) {
			return true
		}
	}
	return false
}

// starNameMatches reports whether a star modifier's (dotted) column name
// names col of the relation aliased alias.
func starNameMatches(parts []string, alias, col string) bool {
	if len(parts) == 0 || !strings.EqualFold(parts[len(parts)-1], col) {
		return false
	}
	return len(parts) == 1 || strings.EqualFold(parts[len(parts)-2], alias)
}

// fromScope builds the scope of a FROM clause: one rel per table it reads.
// A LATERAL subquery or table function sees the rels before it.
func (l *lineage) fromScope(from *parser.FromClause, e *env, outer *scope) *scope {
	sc := &scope{outer: outer, env: e}
	if from == nil {
		return sc
	}
	var conds []func()
	for _, ref := range from.Refs {
		l.addRef(ref, sc, &conds)
	}
	for _, c := range conds {
		c()
	}
	return sc
}

// addRef adds ref's rels to sc, queueing a join's condition (resolved once
// every rel is in scope) on conds.
func (l *lineage) addRef(ref parser.TableRef, sc *scope, conds *[]func()) {
	if l.err != nil {
		return
	}
	switch r := ref.(type) {
	case *parser.BaseTableRef:
		alias := r.Alias
		if alias == "" {
			alias = r.Name[len(r.Name)-1]
		}
		sc.rels = append(sc.rels, l.tableRel(r.Name, alias, sc.env))
	case *parser.TableSubqueryRef:
		var outer *scope
		if r.Lateral {
			outer = sc
		}
		sub, e := r.Select, sc.env
		sc.rels = append(sc.rels, &rel{
			alias: r.Alias,
			cols:  l.outputNames(sub, e, outer),
			flush: func(d demand) { l.analyze(sub, e, outer, d) },
		})
	case *parser.TableFunctionRef:
		sc.rels = append(sc.rels, l.functionRel(r, sc))
	case *parser.JoinRef:
		l.addRef(r.Left, sc, conds)
		left := len(sc.rels)
		l.addRef(r.Right, sc, conds)
		lefts, rights := sc.rels[:left], sc.rels[left:]
		on, using := r.On, r.Using
		natural := strings.Contains(strings.ToUpper(r.Type), "NATURAL")
		*conds = append(*conds, func() {
			l.expr(on, sc)
			for _, u := range using {
				for _, x := range append(append([]*rel{}, lefts...), rights...) {
					if x.has(u) {
						x.need.add(u)
					}
				}
			}
			if natural {
				// Every column both sides share; unknown columns could be
				// any of them.
				for _, x := range lefts {
					for _, y := range rights {
						if x.cols == nil || y.cols == nil {
							x.need.merge(allColumns())
							y.need.merge(allColumns())
							continue
						}
						for _, c := range x.cols {
							if y.has(c) {
								x.need.add(c)
								y.need.add(c)
							}
						}
					}
				}
			}
		})
	case *parser.ParensTableRef:
		if r.Alias != "" {
			l.fail("aliased parenthesized FROM")
			return
		}
		l.addRef(r.Ref, sc, conds)
	default:
		l.fail("FROM item %T", ref)
	}
}

// tableRel is the rel for a table read by name: a source CTE (a sink:
// what's read from it is the answer), another CTE, or a lake table.
func (l *lineage) tableRel(name []string, alias string, e *env) *rel {
	last := strings.ToLower(name[len(name)-1])
	if t, ok := l.sources[last]; ok && len(name) == 1 {
		return &rel{alias: alias, cols: physicalColumns(t), flush: func(d demand) {
			got := l.out[t]
			got.merge(d)
			l.out[t] = got
		}}
	}
	if len(name) == 1 {
		if def, ok := e.cte(last); ok {
			body, de := def.cte.Select, def.env
			cols, ok := l.names[def]
			if !ok {
				cols = l.outputNames(body, de, nil)
				l.names[def] = cols
			}
			if len(def.cte.ColumnNames) > 0 {
				cols = lowerAll(def.cte.ColumnNames)
			}
			renamed := len(def.cte.ColumnNames) > 0
			return &rel{alias: alias, cols: cols, flush: func(d demand) {
				if renamed {
					d = allColumns()
				}
				l.analyze(body, de, nil, d)
			}}
		}
	}
	if t, ok := l.lake[last]; ok {
		return &rel{alias: alias, cols: physicalColumns(t), flush: func(demand) {}}
	}
	l.fail("unknown table %q", strings.Join(name, "."))
	return &rel{alias: alias, flush: func(demand) {}}
}

// functionRel is the rel for a FROM-clause table function: a views.sql
// macro (expanded), query_table (the table it names), or UNNEST.
func (l *lineage) functionRel(r *parser.TableFunctionRef, sc *scope) *rel {
	name := strings.ToLower(r.Name[len(r.Name)-1])
	qualified := len(r.Name) == 1 || len(r.Name) == 2 && strings.EqualFold(r.Name[0], schema.MacroSchema)
	switch {
	case name == "query_table" && len(r.Name) == 1:
		if len(r.Args) != 1 {
			l.fail("query_table w/ %d args", len(r.Args))
			return &rel{flush: func(demand) {}}
		}
		table, ok := l.stringValue(r.Args[0], sc.env)
		if !ok {
			l.fail("query_table of a non-constant name")
			return &rel{flush: func(demand) {}}
		}
		parts := strings.Split(table, ".")
		alias := r.Alias
		if alias == "" {
			alias = parts[len(parts)-1]
		}
		return l.tableRel(parts, alias, sc.env)
	case name == "unnest" && len(r.Name) == 1:
		for _, a := range r.Args {
			l.expr(a, sc) // lateral: it reads the rels before it
		}
		alias := r.Alias
		if alias == "" {
			alias = "unnest"
		}
		return &rel{alias: alias, cols: []string{strings.ToLower(alias), "unnest"}, flush: func(demand) {}}
	case qualified && l.macros[name] != nil:
		return l.macroRel(l.macros[name], r, sc)
	}
	l.fail("table function %q", strings.Join(r.Name, "."))
	return &rel{flush: func(demand) {}}
}

// macroRel expands a views.sql macro call: its body, w/ its parameters
// bound to the call's arguments (or their defaults). Table names in the
// body (via query_table) resolve against the caller's CTEs, like DuckDB's
// own expansion.
func (l *lineage) macroRel(m *duckdb.TableMacro, call *parser.TableFunctionRef, sc *scope) *rel {
	if l.depth > 16 {
		l.fail("macro %s nests too deep", m.Name)
		return &rel{flush: func(demand) {}}
	}
	params := map[string]parser.Expr{}
	pos := 0
	for _, a := range call.Args {
		if na, ok := a.(*parser.NamedArg); ok {
			params[strings.ToLower(na.Name)] = l.bindArg(na.Value, sc)
			continue
		}
		if pos >= len(m.Params) {
			l.fail("macro %s: too many arguments", m.Name)
			return &rel{flush: func(demand) {}}
		}
		params[strings.ToLower(m.Params[pos])] = l.bindArg(a, sc)
		pos++
	}
	for p, def := range m.Defaults {
		if _, ok := params[strings.ToLower(p)]; !ok {
			params[strings.ToLower(p)] = &parser.Literal{Kind: parser.LitString, Text: def}
		}
	}
	// The caller's CTEs, minus any macro parameters it was itself bound
	// in: a macro body only sees its own.
	callerCTEs := sc.env
	for callerCTEs != nil && callerCTEs.params != nil {
		callerCTEs = callerCTEs.parent
	}
	e := &env{parent: callerCTEs, params: params}
	alias := call.Alias
	if alias == "" {
		alias = m.Name
	}
	return &rel{
		alias: alias,
		cols:  l.outputNames(m.Body, e, nil),
		flush: func(d demand) {
			l.depth++
			l.analyze(m.Body, e, nil, d)
			l.depth--
		},
	}
}

// bindArg is a macro argument as its parameter's binding: a parameter of
// an enclosing macro passed through resolves to its own binding. Column
// references in the argument are read at the call site.
func (l *lineage) bindArg(a parser.Expr, sc *scope) parser.Expr {
	if id, ok := a.(*parser.Ident); ok && len(id.Parts) == 1 {
		if p, ok := sc.env.param(id.Parts[0]); ok {
			return p
		}
	}
	l.expr(a, sc)
	return a
}

// stringValue is x's value when it's a string constant: a literal, or a
// macro parameter bound to one.
func (l *lineage) stringValue(x parser.Expr, e *env) (string, bool) {
	switch v := x.(type) {
	case *parser.Literal:
		return v.Text, v.Kind == parser.LitString
	case *parser.Ident:
		if len(v.Parts) == 1 {
			if p, ok := e.param(v.Parts[0]); ok {
				return l.stringValue(p, nil)
			}
		}
	}
	return "", false
}

// outputNames is stmt's output column names, lowercased, w/o reading
// anything (nil when not all of them are known).
func (l *lineage) outputNames(stmt *parser.SelectStatement, e *env, outer *scope) []string {
	if stmt == nil || l.err != nil || !l.spend() {
		return nil
	}
	for stmt.SetOp != parser.SetOpNone {
		stmt = stmt.SetLeft
	}
	if stmt.Values != nil {
		return nil
	}
	if stmt.With != nil && !stmt.With.Recursive {
		e = l.withCTEs(stmt.With, e)
	}
	// A throwaway lineage: building the scope mustn't record demand.
	probe := &lineage{macros: l.macros, sources: l.sources, lake: l.lake, out: map[Table]demand{}, depth: l.depth, budget: l.budget, names: l.names}
	sc := probe.fromScope(stmt.From, e, outer)
	if probe.err != nil {
		if errors.Is(probe.err, errLineage) && *l.budget < 0 {
			l.err = probe.err // out of budget: the whole lineage is
		}
		return nil
	}
	var names []string
	for _, o := range probe.outputs(stmt, sc) {
		if o.name == "" {
			return nil
		}
		names = append(names, o.name)
	}
	return names
}

// walk reads every expression under n.
func (l *lineage) walk(n parser.Node, sc *scope) {
	if n == nil {
		return
	}
	for _, c := range n.Children() {
		if x, ok := c.(parser.Expr); ok {
			l.expr(x, sc)
			continue
		}
		l.walk(c, sc)
	}
}

// expr reads every column x references, in sc.
func (l *lineage) expr(x parser.Expr, sc *scope) {
	if x == nil || l.err != nil {
		return
	}
	switch v := x.(type) {
	case *parser.Ident:
		l.ident(v.Parts, sc)
	case *parser.StarExpr:
		// Outside a select list: count(*) reads nothing, anything else
		// (IE struct_pack(*)) reads what the star would.
		for _, o := range l.starOutputs(v, sc) {
			l.output(o, sc)
		}
	case *parser.FunctionExpr:
		if len(v.Args) == 1 && len(v.Name) == 1 && strings.EqualFold(v.Name[0], "count") {
			if _, ok := v.Args[0].(*parser.StarExpr); ok {
				l.walk(&parser.FunctionExpr{Filter: v.Filter, Over: v.Over, OrderBy: v.OrderBy}, sc)
				return
			}
		}
		l.walk(v, sc)
	case *parser.SubqueryExpr:
		l.analyze(v.Select, sc.env, sc, allColumns())
	case *parser.PositionalExpr:
		// #n is the nth column of the FROM clause: every column of every
		// rel in scope, rather than counting through them.
		for _, r := range sc.rels {
			r.need.merge(allColumns())
		}
	case *parser.LambdaExpr:
		// `col -> 'key'` parses as a lambda too (the JSON arrow), so a
		// parameter that's also a column is read as well.
		for _, p := range v.Params {
			l.identIfColumn(p, sc)
		}
		inner := *sc
		inner.bound = append(append([]string{}, sc.bound...), v.Params...)
		l.expr(v.Body, &inner)
	case *parser.ListComprehensionExpr:
		l.expr(v.Source, sc)
		inner := *sc
		inner.bound = append(append([]string{}, sc.bound...), v.Vars...)
		l.expr(v.Expr, &inner)
		l.expr(v.Filter, &inner)
	default:
		l.walk(x, sc)
	}
}

// identIfColumn reads name if it's a column in scope, and is a no-op
// otherwise.
func (l *lineage) identIfColumn(name string, sc *scope) {
	for s := sc; s != nil; s = s.outer {
		for _, r := range s.rels {
			if r.cols != nil && r.has(name) {
				r.need.add(name)
			}
		}
	}
}

// ident reads the column a (possibly qualified, possibly struct-field)
// reference names. Every reading DuckDB could take is read: as
// rel.col (w/ any catalog/schema before rel), and as col.field...
func (l *lineage) ident(parts []string, sc *scope) {
	if len(parts) == 1 && sc.isBound(parts[0]) {
		return
	}
	for s := sc; s != nil; s = s.outer {
		found := false
		// Qualified: rel.col, schema.rel.col, catalog.schema.rel.col.
		for i := 0; i < len(parts)-1 && i < 3; i++ {
			for _, r := range s.rels {
				if strings.EqualFold(r.alias, parts[i]) && r.has(parts[i+1]) {
					r.need.add(parts[i+1])
					found = true
				}
			}
		}
		// Unqualified: col (any further parts are struct fields).
		for _, r := range s.rels {
			if r.has(parts[0]) {
				r.need.add(parts[0])
				found = true
			}
		}
		if found {
			return
		}
		// A select list alias (WHERE/GROUP BY/HAVING/QUALIFY/ORDER BY can
		// reference one).
		if x, ok := s.items[strings.ToLower(parts[0])]; ok && len(parts) == 1 {
			delete(s.items, strings.ToLower(parts[0])) // once, and no cycles
			l.expr(x, s)
			return
		}
	}
	if len(parts) == 1 {
		if _, ok := sc.env.param(parts[0]); ok {
			return
		}
	}
	l.fail("unresolved reference %q", strings.Join(parts, "."))
}

func (s *scope) isBound(name string) bool {
	for _, b := range s.bound {
		if strings.EqualFold(b, name) {
			return true
		}
	}
	return false
}

func physicalColumns(t Table) []string {
	var cols []string
	for _, c := range Physical[t].Columns {
		cols = append(cols, strings.ToLower(c.Name))
	}
	return cols
}

func lowerAll(names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = strings.ToLower(n)
	}
	return out
}
