package insights

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/federate"
	"github.com/inngest/inngest/pkg/duckdb/parser"
	"github.com/inngest/inngest/pkg/duckdb/schema"
)

// TableCTE is the body of one logical table's reserved CTE.
type TableCTE struct {
	// Select produces the logical table's rows and columns. Its '?'
	// placeholders bind to Args.
	Select *parser.SelectStatement
	Args   []any
	// NotMaterialized renders the CTE AS NOT MATERIALIZED, so DuckDB
	// inlines it at every reference and pushes each reference's filters
	// and projections into it. Without it, DuckDB materializes a CTE that's
	// referenced more than once (one unfiltered scan, then filters applied
	// above it), which is right for an expensive body read several times
	// and wrong for a plain table scan.
	NotMaterialized bool
	// Prelude is CTE definitions Select depends on that must sit in the
	// query's top-level WITH, ahead of every table CTE (e.g. a federated
	// executor's physical sources and stream-backed deltas). Each one (by
	// name) is rendered once. They take no args.
	Prelude []*parser.CTE
}

// TableSource supplies the CTE body for one logical table.
type TableSource func(table string) (TableCTE, error)

// MacroSource is the plain (non-federated) source: each table's CTE is an
// inlined alias for its tenant-scoped table macro, e.g.
// __inngest_runs AS NOT MATERIALIZED (SELECT * FROM inngest_local.insights_runs(?, ?)),
// which plans exactly like calling the macro at each reference.
func MacroSource(accountID, envID uuid.UUID) TableSource {
	return func(table string) (TableCTE, error) {
		tbl, ok := logicalTables[table]
		if !ok || tbl.view == "" {
			return TableCTE{}, fmt.Errorf("insights: no table macro for %q", table)
		}
		return TableCTE{
			Select:          macroSelect(tbl.view, tenantParams()...),
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
		return TableCTE{Select: macroSelect(tbl.view), NotMaterialized: true}, nil
	}
}

// renderSQL writes stmt as SQL text.
func renderSQL(stmt *parser.SelectStatement) (string, error) {
	var b strings.Builder
	if err := parser.Write(&b, stmt); err != nil {
		return "", fmt.Errorf("insights: rendering query: %w", err)
	}
	return b.String(), nil
}

// macroSelect is SELECT * FROM the table macro view, called w/ args.
func macroSelect(view string, args ...parser.Expr) *parser.SelectStatement {
	return &parser.SelectStatement{
		Columns: []*parser.SelectItem{{Expr: &parser.StarExpr{}}},
		From: &parser.FromClause{Refs: []parser.TableRef{
			&parser.TableFunctionRef{Name: []string{schema.MacroSchema, view}, Args: args},
		}},
	}
}

// tenantParams are a tenant-scoped macro's account and env args, as '?'
// placeholders.
func tenantParams() []parser.Expr {
	return []parser.Expr{&parser.Parameter{Kind: parser.ParamAnonymous}, &parser.Parameter{Kind: parser.ParamAnonymous}}
}

// Render returns the query with a reserved CTE prepended for every logical
// table it reads, its body taken from src, merged into the query's own WITH
// clause when it has one (ahead of the user's CTEs, which may read them).
// Args are the CTE bodies' args, in table order: the only placeholders, as
// the preludes and the query itself take none.
func (r *TranspileResult) Render(src TableSource) (*parser.SelectStatement, []any, error) {
	if len(r.tables) == 0 {
		return r.stmt, nil, nil
	}
	var ctes []*parser.CTE
	var args []any
	seen := map[string]bool{}
	defs := make([]*parser.CTE, len(r.tables))
	for i, t := range r.tables {
		c, err := src(t)
		if err != nil {
			return nil, nil, err
		}
		for _, p := range c.Prelude {
			if !seen[p.Name] {
				seen[p.Name] = true
				ctes = append(ctes, p)
			}
		}
		def := &parser.CTE{Name: TableCTEName(t), Select: c.Select}
		if c.NotMaterialized {
			notMaterialized := false
			def.Materialized = &notMaterialized
		}
		defs[i] = def
		args = append(args, c.Args...)
	}

	out := *r.stmt
	w := &parser.WithClause{}
	if r.stmt.With != nil {
		*w = *r.stmt.With
	}
	w.CTEs = append(append(ctes, defs...), w.CTEs...)
	out.With = w
	return &out, args, nil
}

type federatedSource struct {
	table federate.Table
	param string
	// joined maps the logical table's columns to this (joined, not first)
	// source's physical columns for pushdown. Only columns that can't
	// change what the join contributes to a kept row belong here, IE the
	// run_id the metadata rollup is per.
	joined map[string]string
}

// metadataByRun is the metadata rollup joined on a logical table's run_id.
var metadataByRun = federatedSource{federate.TableMetadata, "p_metadata", map[string]string{"run_id": "run_id"}}

// federatedSources maps each table that can be federated, per mode, to the
// physical tables its macro reads and the parameter it takes each one's
// source in. The first is the table's own rows, the one its pushdown
// targets; the rest are joined on (e.g. the metadata rollup).
var federatedSources = map[bool]map[string][]federatedSource{
	false: {
		"runs":                 {{table: federate.TableRuns, param: "p_runs"}, metadataByRun},
		"extended_trace_spans": {{table: federate.TableSpans, param: "p_spans"}, metadataByRun},
		"steps":                {{table: federate.TableSpans, param: "p_spans"}, metadataByRun},
		"step_attempts":        {{table: federate.TableSpans, param: "p_spans"}, metadataByRun},
		"events":               {{table: federate.TableEvents, param: "p_events"}},
		"metadata":             {{table: federate.TableMetadata, param: "p_metadata"}},
	},
	true: {
		"runs":            {{table: federate.TableRuns, param: "p_runs"}},
		"run_trace_spans": {{table: federate.TableSpans, param: "p_spans"}},
		"events":          {{table: federate.TableEvents, param: "p_events"}},
		"run_metadata":    {{table: federate.TableMetadata, param: "p_metadata"}},
	},
}

// Federated returns the physical tables the query needs federated, the
// pushdown translated onto them (see physicalPushdown), and a renderer for
// federate.Query.Render: each logical table's reserved CTE calls its own
// macro (its one projection, joins and collapse) with the executor's lake ∪
// delta sources in place of the lake tables. A logical table with no
// federated source is an error, never a silent read of the lake alone. Run
// it through ExecuteFederated, which applies Execute's limits and
// rendered-SQL check.
func (r *TranspileResult) Federated() ([]federate.Table, map[federate.Table][]federate.Predicate, func(map[federate.Table]federate.Source) (*parser.SelectStatement, []any, error)) {
	var tables []federate.Table
	seen := map[federate.Table]bool{}
	for _, t := range r.tables {
		for _, fs := range federatedSources[r.cat.raw][t] {
			if !seen[fs.table] {
				seen[fs.table] = true
				tables = append(tables, fs.table)
			}
		}
	}
	render := func(sources map[federate.Table]federate.Source) (*parser.SelectStatement, []any, error) {
		return r.Render(func(table string) (TableCTE, error) {
			fss, ok := federatedSources[r.cat.raw][table]
			if !ok {
				return TableCTE{}, fmt.Errorf("insights: table %q can't be federated", table)
			}
			var params []parser.Expr
			var prelude []*parser.CTE
			for _, fs := range fss {
				src, ok := sources[fs.table]
				if !ok {
					return TableCTE{}, fmt.Errorf("insights: no federated %s source for table %q", fs.table, table)
				}
				params = append(params, &parser.NamedArg{Name: fs.param, Value: &parser.Literal{Kind: parser.LitString, Text: src.Name}})
				prelude = append(prelude, src.Prelude...)
			}
			view := r.cat.tables[table].view
			if r.cat.raw {
				return TableCTE{Select: macroSelect(view, params...), Prelude: prelude}, nil
			}
			return TableCTE{
				Select:  macroSelect(view, append(tenantParams(), params...)...),
				Args:    []any{r.accountID.String(), r.envID.String()},
				Prelude: prelude,
			}, nil
		})
	}
	return tables, r.physicalPushdown(), render
}

// physicalPushdown translates the logical-table predicates onto their
// federated physical tables (see physicalPredicates), keeping only the
// pushable ones.
func (r *TranspileResult) physicalPushdown() map[federate.Table][]federate.Predicate {
	out := map[federate.Table][]federate.Predicate{}
	for lt, preds := range r.Pushdown {
		for _, p := range preds {
			for _, pp := range r.physicalPredicates(lt, p) {
				if pp.kept == "" {
					out[pp.table] = append(out[pp.table], pp.pred)
				}
			}
		}
	}
	return out
}

// Why a logical predicate stays in DuckDB instead of being pushed.
const (
	keptNotFederated = "table not federated"
	keptComputed     = "computed column"
	keptShared       = "source shared with another table"
)

// physicalPredicate is one logical predicate translated onto one of its
// table's federated sources.
type physicalPredicate struct {
	table federate.Table
	pred  federate.Predicate
	// kept is "" when it's forwarded to the delta (whose streamer applies
	// it if it can represent it), else why it isn't.
	kept string
}

// physicalPredicates translates one logical-table predicate onto its
// federated sources. On the first (the table's own rows) the logical column
// becomes the physical column its macro passes through
// (knownColumn.physical); on a joined one it's federatedSource.joined, and
// a predicate on a column that doesn't map there is just left out. A
// source's predicate is kept when the table isn't federated, the column is
// computed by the macro, or another logical table in the query reads the
// same source (one delta serves them all, so it can't be filtered for just
// one, e.g. metadata and the rollup runs joins on).
//
// A boolean predicate is translated part by part, and kept whole if any
// part is.
func (r *TranspileResult) physicalPredicates(lt federate.Table, p federate.Predicate) []physicalPredicate {
	fss, ok := federatedSources[r.cat.raw][string(lt)]
	if !ok {
		return []physicalPredicate{{kept: keptNotFederated}}
	}
	tbl := r.cat.tables[string(lt)]
	var out []physicalPredicate
	for i, fs := range fss {
		column := func(c string) string { return tbl.columns[c].physical }
		if i > 0 {
			column = func(c string) string { return fs.joined[c] }
		}
		pp := physicalPredicate{table: fs.table}
		pp.pred, pp.kept = translatePredicate(p, column)
		if i > 0 && pp.kept != "" {
			continue
		}
		if pp.kept == "" && r.sharedSource(lt, fs.table) {
			pp.pred, pp.kept = federate.Predicate{}, keptShared
		}
		out = append(out, pp)
	}
	return out
}

// sharedSource is whether a logical table in the query other than lt reads
// physical table t.
func (r *TranspileResult) sharedSource(lt, t federate.Table) bool {
	for _, other := range r.tables {
		if other == string(lt) {
			continue
		}
		for _, fs := range federatedSources[r.cat.raw][other] {
			if fs.table == t {
				return true
			}
		}
	}
	return false
}

// translatePredicate renames p's columns to their physical ones (by
// column), or says why it can't.
func translatePredicate(p federate.Predicate, column func(string) string) (federate.Predicate, string) {
	if p.IsBool() {
		args := make([]federate.Predicate, len(p.Args))
		for i, a := range p.Args {
			pa, k := translatePredicate(a, column)
			if k != "" {
				return federate.Predicate{}, k
			}
			args[i] = pa
		}
		p.Args = args
		return p, ""
	}
	col := column(p.Column)
	if col == "" {
		return federate.Predicate{}, keptComputed
	}
	p.Column = col
	return p, ""
}

// checkReservedNames rejects a user CTE named with TableCTEPrefix, which
// would collide with (or impersonate) a logical table's reserved CTE, or a
// federated executor's generated ones (federate.GeneratedPrefix). The
// prefix matches case-insensitively, like DuckDB's own CTE names.
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
	if c, ok := n.(*parser.CTE); ok && strings.HasPrefix(strings.ToLower(c.Name), TableCTEPrefix) {
		v.err = &ValidationError{Pos: c.Pos(), End: c.End(),
			Message: fmt.Sprintf("CTE names may not start with %q", TableCTEPrefix)}
		return nil
	}
	return v
}
