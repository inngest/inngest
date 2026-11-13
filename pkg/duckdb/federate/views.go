package federate

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/parser"
	"github.com/inngest/inngest/pkg/duckdb/schema"
)

// TerminalStatuses are schema.TerminalStatuses: the statuses a final run row
// carries. The buffer's finished-runs derivation must use this exact set.
var TerminalStatuses = schema.TerminalStatuses

// DeltaColumns is the projection every delta for t returns: all of its
// physical table's columns, in declared order.
//
// NOTE: runs/run_trace_spans' output is split into output and error at
// write time (cqrs.SplitSpanOutputEnvelope), and a delta carries both, so a
// DeltaStreamer over a buffer that stores the unsplit envelope must derive
// them the same way: output is the envelope's "data" value (the whole value
// for a step.waitForEvent-shaped or unrecognized output, NULL for an error
// envelope w/o data), error its "error" value (else NULL). A buffer can
// store the envelope w/ both halves (IE {"data": ..., "error": ...}) so it
// splits back to exactly what the lake stores.
func DeltaColumns(t Table) []Column {
	phys := Physical[t]
	cols := make([]Column, len(phys.Columns))
	for i, c := range phys.Columns {
		cols[i] = Column{Name: c.Name, Type: c.Type}
	}
	return cols
}

// SourceName is the CTE a federated physical table is read through: its
// lake rows ∪ its delta rows, uncollapsed. Callers pass it to a macro that
// takes a physical source (inngest.raw_runs, inngest.insights_runs, ...).
func SourceName(t Table) string { return GeneratedPrefix + "fed_" + Physical[t].Name }

// GeneratedPrefix prefixes every CTE name the executor generates, so they
// can't collide w/ a caller's own CTEs when merged into one WITH. It's
// insights' reserved prefix (insights.TableCTEPrefix), which user queries
// can't use.
const GeneratedPrefix = "__inngest_"

// LakeSource is the lake half of the union: a DuckLake catalog alias,
// optionally pinned to a snapshot so one query reads one lake version.
type LakeSource struct {
	Catalog  string
	Snapshot *int64
}

func (s LakeSource) table(name string) *parser.BaseTableRef {
	ref := tableRef(s.Catalog, name)
	if s.Snapshot != nil {
		ref.At = &parser.AtClause{Unit: "VERSION", Expr: numLit(*s.Snapshot)}
	}
	return ref
}

// sourceSpec is what one physical source CTE needs: tenant, run-time
// bound, the lake, and the relation the delta streams through.
type sourceSpec struct {
	AccountID  uuid.UUID
	EnvID      uuid.UUID
	AllTenants bool
	From, To   time.Time
	Lake       LakeSource
	// Watermark bounds the lake side, bucket_at <= W; the delta holds the
	// rest (bucket_at > W).
	Watermark time.Time
	// Delta is the delta's streamed CTE. Empty, the source is the lake
	// table alone and unbounded: the stand-in deltaColumnsUsed plans with,
	// whose scan then reads only what the query does.
	Delta string
	// DeltaColumns, when set, are the only physical columns Delta carries:
	// the ones the query reads (see deltaColumnsUsed). Every other column is
	// a typed NULL on the delta side, so the source keeps the table's shape.
	DeltaColumns map[string]bool
	// TopN, if set, keeps the lake side to its first rows (Query.TopN), its
	// predicates applied exactly.
	TopN *topNSpec
	// Collapses says the table's view collapses rows (see collapses), so a
	// TopN cuts the lake side on its own.
	Collapses bool
}

// collapses are the tables whose views collapse rows (raw_runs keeps each
// run's latest): DuckDB can't push a top-n through that window, so their
// lake side is cut to its own first rows. Other tables' lake sides are
// left whole: DuckDB applies the query's own top-n to the plain union, and
// cutting the lake side first only cost time (a benchmark's small-tenant
// events list: 23ms against 32ms).
var collapses = map[Table]bool{TableRuns: true}

// topNSpec is a table's TopN and the predicates its lake side applies.
type topNSpec struct {
	TopN
	Predicates []Predicate
}

// topNOf is t's top-n spec in q, if q pushes one.
func topNOf(q Query, t Table) *topNSpec {
	tn, ok := q.TopN[t]
	if !ok {
		return nil
	}
	return &topNSpec{TopN: tn, Predicates: q.Predicates[t]}
}

// sourceTimeColumn is the column bounding a physical table's run-time range.
var sourceTimeColumn = map[Table]string{TableRuns: "queued_at", TableSpans: "run_queued_at", TableEvents: "received_at", TableMetadata: "run_queued_at"}

// deltaBoundColumns are the columns sourceCTE's delta-side bounds read, so
// the delta must carry them whatever the query reads.
func deltaBoundColumns(t Table, allTenants bool) []string {
	cols := []string{sourceTimeColumn[t]}
	if !allTenants {
		cols = append(cols, "account_id", "env_id")
	}
	if t != TableRuns {
		cols = append(cols, "bucket_at")
	}
	return cols
}

// sourceCTE builds t's physical source CTE (lake ∪ delta, every physical
// column, in declared order). The lake side is cut at
// the watermark, as the delta is, so rows the lake holds past W (a newer
// export) aren't read twice. The delta side re-applies the tenant and time
// bounds (and, for spans and events, the watermark) rather than trusting
// the streamer to, so a streamer bug can't leak another tenant's rows or
// read a row twice. Runs skip the watermark there: a run still open can sit
// in the delta whatever its bucket. The delta's columns are
// cast to their lake types (the JSON stream to the lake's VARCHAR text,
// which is free, STRUCT lists), so the union is the physical table's own
// shape and any macro over the table reads it unchanged. JSON text stays
// VARCHAR here so filters and extractions read it w/o a JSON cast (a
// validating parse per row); the lake's macros cast it to JSON only
// where they return it.
func sourceCTE(t Table, spec sourceSpec) (*parser.CTE, error) {
	phys, ok := Physical[t]
	if !ok {
		return nil, fmt.Errorf("federate: no physical table for %q", t)
	}
	lake := make([]*parser.SelectItem, len(phys.Columns))
	delta := make([]*parser.SelectItem, len(phys.Columns))
	for i, c := range phys.Columns {
		lake[i] = item(colRef(c.Name), "")
		if spec.DeltaColumns != nil && !spec.DeltaColumns[c.Name] {
			delta[i] = item(cast(&parser.Literal{Kind: parser.LitNull}, c.PhysicalType()), c.Name)
			continue
		}
		delta[i] = deltaSelect(c)
	}
	lakeSide := &parser.SelectStatement{Columns: lake, From: from(spec.Lake.table(phys.Name))}
	if spec.Delta == "" {
		return cte(SourceName(t), lakeSide), nil
	}

	timeCol := colRef(sourceTimeColumn[t])
	bounds := []parser.Expr{binary(">=", timeCol, timeLit(spec.From)), binary("<", timeCol, timeLit(spec.To))}
	if !spec.AllTenants {
		bounds = append([]parser.Expr{
			binary("=", colRef("account_id"), uuidLit(spec.AccountID.String())),
			binary("=", colRef("env_id"), uuidLit(spec.EnvID.String())),
		}, bounds...)
	}
	lakeSide.Where = and(append(bounds, binary("<=", colRef("bucket_at"), timeLit(spec.Watermark)))...)
	deltaWhere := and(bounds...)
	if t != TableRuns {
		deltaWhere = and(deltaWhere, binary(">", colRef("bucket_at"), timeLit(spec.Watermark)))
	}
	if spec.TopN != nil && spec.Collapses {
		where, err := predicatesExpr(t, spec.TopN.Predicates)
		if err != nil {
			return nil, err
		}
		lakeSide.Where = and(lakeSide.Where, where)
		lakeSide.OrderBy = orderBy(spec.TopN.OrderBy)
		lakeSide.Limit = &parser.LimitClause{Limit: numLit(int64(spec.TopN.Limit))}
	}
	deltaSide := &parser.SelectStatement{Columns: delta, From: from(tableRef(spec.Delta)), Where: deltaWhere}
	return cte(SourceName(t), unionAll(lakeSide, deltaSide, false)), nil
}

func deltaSelect(c schema.Column) *parser.SelectItem {
	if c.PhysicalType() == c.Type.DuckDBType() {
		return item(colRef(c.Name), "")
	}
	return item(cast(colRef(c.Name), c.PhysicalType()), c.Name)
}

// rawView is the plain-SQL path's logical table for t: runs collapsed by
// inngest.raw_runs over their federated source (unless runsCollapsed, see
// Query.RunsCollapsed); spans, events and metadata are the source itself
// (each row is its own emission / event).
func rawView(t Table, lake LakeSource, runsCollapsed bool) (*parser.CTE, error) {
	switch t {
	case TableRuns:
		if runsCollapsed {
			return cte(string(t), selectStar(tableRef(SourceName(t)))), nil
		}
		return cte(string(t), selectStar(&parser.TableFunctionRef{
			Name: []string{lake.Catalog, "raw_runs"},
			Args: []parser.Expr{&parser.NamedArg{Name: "p_runs", Value: strLit(SourceName(t))}},
		})), nil
	case TableSpans, TableEvents, TableMetadata:
		return cte(string(t), selectStar(tableRef(SourceName(t)))), nil
	}
	return nil, fmt.Errorf("federate: no view for table %q", t)
}
