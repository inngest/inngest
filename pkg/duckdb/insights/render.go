package insights

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/federate"
	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// TableCTE is the body of one logical table's reserved CTE.
type TableCTE struct {
	// SQL is a complete SELECT producing the logical table's rows and
	// columns. Its '?' placeholders bind to Args.
	SQL  string
	Args []any
	// NotMaterialized renders the CTE AS NOT MATERIALIZED, so DuckDB
	// inlines it at every reference and pushes each reference's filters
	// and projections into it. Without it, DuckDB materializes a CTE that's
	// referenced more than once (one unfiltered scan, then filters applied
	// above it), which is right for an expensive body read several times
	// and wrong for a plain table scan.
	NotMaterialized bool
	// Prelude is CTE definitions SQL depends on that must sit in the
	// query's top-level WITH, ahead of every table CTE (e.g. a federated
	// executor's physical sources and stream-backed deltas). Each distinct
	// one is rendered once, its args ahead of every table CTE's.
	Prelude []federate.CTEDef
}

// TableSource supplies the CTE body for one logical table.
type TableSource func(table string) (TableCTE, error)

// MacroSource is the plain (non-federated) source: each table's CTE is an
// inlined alias for its tenant-scoped table macro, e.g.
// __inngest_runs AS NOT MATERIALIZED (SELECT * FROM inngest.insights_runs(?, ?)),
// which plans exactly like calling the macro at each reference.
func MacroSource(accountID, envID uuid.UUID) TableSource {
	return func(table string) (TableCTE, error) {
		tbl, ok := logicalTables[table]
		if !ok || tbl.view == "" {
			return TableCTE{}, fmt.Errorf("insights: no table macro for %q", table)
		}
		return TableCTE{
			SQL:             fmt.Sprintf("SELECT * FROM %s.%s(?, ?)", driver.DuckLakeAlias, tbl.view),
			Args:            []any{accountID.String(), envID.String()},
			NotMaterialized: true,
		}, nil
	}
}

// defaultSource is the result's non-federated source: MacroSource, or in raw
// mode each physical table's raw macro, unscoped.
func (r *TranspileResult) defaultSource() TableSource {
	if !r.cat.raw {
		return MacroSource(r.accountID, r.envID)
	}
	return func(table string) (TableCTE, error) {
		tbl, ok := r.cat.tables[table]
		if !ok {
			return TableCTE{}, fmt.Errorf("insights: no raw table %q", table)
		}
		return TableCTE{SQL: fmt.Sprintf("SELECT * FROM %s.%s()", driver.DuckLakeAlias, tbl.view), NotMaterialized: true}, nil
	}
}

// Render returns the query with a reserved CTE prepended for every logical
// table it reads, its body taken from src, merged into the query's own WITH
// clause when it has one (ahead of the user's CTEs, which may read them).
// Args are the CTE bodies' args, in table order, then the query's own.
func (r *TranspileResult) Render(src TableSource) (string, []any, error) {
	if len(r.tables) == 0 {
		return r.body, append([]any(nil), r.bodyArgs...), nil
	}
	var prelude []string
	var preludeArgs []any
	seen := map[string]bool{}
	defs := make([]string, len(r.tables))
	var args []any
	for i, t := range r.tables {
		c, err := src(t)
		if err != nil {
			return "", nil, err
		}
		for _, p := range c.Prelude {
			if !seen[p.SQL] {
				seen[p.SQL] = true
				prelude = append(prelude, p.SQL)
				preludeArgs = append(preludeArgs, p.Args...)
			}
		}
		as := "AS "
		if c.NotMaterialized {
			as = "AS NOT MATERIALIZED "
		}
		defs[i] = TableCTEName(t) + " " + as + "(" + c.SQL + ")"
		args = append(args, c.Args...)
	}
	args = append(append(preludeArgs, args...), r.bodyArgs...)

	// parser.Write always opens a WITH clause with exactly one of these.
	ctes := strings.Join(append(prelude, defs...), ", ")
	for _, kw := range []string{"WITH RECURSIVE ", "WITH "} {
		if rest, ok := strings.CutPrefix(r.body, kw); ok {
			return kw + ctes + ", " + rest, args, nil
		}
	}
	return "WITH " + ctes + " " + r.body, args, nil
}

type federatedSource struct {
	table federate.Table
	param string
}

// federatedSources maps each table that can be federated, per mode, to its
// physical table and the parameter its macro takes that table's source in.
var federatedSources = map[bool]map[string]federatedSource{
	false: {
		"runs":                 {federate.TableRuns, "p_runs"},
		"extended_trace_spans": {federate.TableSpans, "p_spans"},
		"events":               {federate.TableEvents, "p_events"},
	},
	true: {
		"runs":            {federate.TableRuns, "p_runs"},
		"run_trace_spans": {federate.TableSpans, "p_spans"},
		"events":          {federate.TableEvents, "p_events"},
	},
}

// Federated returns the physical tables the query needs federated, the
// pushdown translated onto them (see physicalPushdown), and a renderer for
// federate.Query.Render: each logical table's reserved CTE calls its own
// macro (its one projection, joins and collapse) with the executor's lake ∪
// delta source in place of the lake table. A logical table with no federated
// source is an error, never a silent read of the lake alone.
func (r *TranspileResult) Federated() ([]federate.Table, map[federate.Table][]federate.Predicate, func(map[federate.Table]federate.Source) (string, []any, error)) {
	var tables []federate.Table
	seen := map[federate.Table]bool{}
	for _, t := range r.tables {
		if fs, ok := federatedSources[r.cat.raw][t]; ok && !seen[fs.table] {
			seen[fs.table] = true
			tables = append(tables, fs.table)
		}
	}
	render := func(sources map[federate.Table]federate.Source) (string, []any, error) {
		return r.Render(func(table string) (TableCTE, error) {
			fs, ok := federatedSources[r.cat.raw][table]
			if !ok {
				return TableCTE{}, fmt.Errorf("insights: table %q can't be federated", table)
			}
			src, ok := sources[fs.table]
			if !ok {
				return TableCTE{}, fmt.Errorf("insights: no federated source for table %q", table)
			}
			view := r.cat.tables[table].view
			if r.cat.raw {
				return TableCTE{
					SQL:     fmt.Sprintf("SELECT * FROM %s.%s(%s := '%s')", driver.DuckLakeAlias, view, fs.param, src.Name),
					Prelude: src.Prelude,
				}, nil
			}
			return TableCTE{
				SQL:     fmt.Sprintf("SELECT * FROM %s.%s(?, ?, %s := '%s')", driver.DuckLakeAlias, view, fs.param, src.Name),
				Args:    []any{r.accountID.String(), r.envID.String()},
				Prelude: src.Prelude,
			}, nil
		})
	}
	return tables, physicalPushdown(r.Pushdown, r.cat), render
}

// physicalPushdown translates logical-table predicates onto their federated
// physical tables (see physicalPredicate), keeping only the pushable ones.
func physicalPushdown(logical Pushdown, cat catalog) map[federate.Table][]federate.Predicate {
	out := map[federate.Table][]federate.Predicate{}
	for lt, preds := range logical {
		for _, p := range preds {
			if t, phys, kept := physicalPredicate(lt, p, cat); kept == "" {
				out[t] = append(out[t], phys)
			}
		}
	}
	return out
}

// Why a logical predicate stays in DuckDB instead of being pushed.
const (
	keptNotFederated = "table not federated"
	keptComputed     = "computed column"
)

// physicalPredicate translates one logical-table predicate onto its
// federated physical table: the logical column becomes the physical column
// its macro passes through (knownColumn.physical). kept is "" when it's
// forwarded to the delta (whose streamer applies it if it can represent it),
// else why it isn't: the table isn't federated, or the column is computed by
// the macro.
//
// A boolean predicate is translated part by part, and kept whole if any
// part is.
func physicalPredicate(lt federate.Table, p federate.Predicate, cat catalog) (t federate.Table, phys federate.Predicate, kept string) {
	fs, ok := federatedSources[cat.raw][string(lt)]
	if !ok {
		return "", federate.Predicate{}, keptNotFederated
	}
	if p.IsBool() {
		args := make([]federate.Predicate, len(p.Args))
		for i, a := range p.Args {
			_, pa, k := physicalPredicate(lt, a, cat)
			if k != "" {
				return fs.table, federate.Predicate{}, k
			}
			args[i] = pa
		}
		p.Args = args
		return fs.table, p, ""
	}
	col := cat.tables[string(lt)].columns[p.Column].physical
	if col == "" {
		return fs.table, federate.Predicate{}, keptComputed
	}
	p.Column = col
	return fs.table, p, ""
}

// checkReservedNames rejects a user CTE named with TableCTEPrefix, which
// would collide with (or impersonate) a logical table's reserved CTE.
func checkReservedNames(stmt *parser.SelectStatement) error {
	v := &reservedNameCheck{}
	parser.Walk(v, stmt)
	return v.err
}

type reservedNameCheck struct{ err error }

func (v *reservedNameCheck) Visit(n parser.Node) parser.Visitor {
	if v.err != nil {
		return nil
	}
	if c, ok := n.(*parser.CTE); ok && strings.HasPrefix(c.Name, TableCTEPrefix) {
		v.err = &ValidationError{Pos: c.Pos(), End: c.End(),
			Message: fmt.Sprintf("CTE names may not start with %q", TableCTEPrefix)}
		return nil
	}
	return v
}
