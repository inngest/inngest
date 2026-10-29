package federate

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/schema"
)

// TerminalStatuses are schema.TerminalStatuses: the statuses a final run row
// carries. The buffer's finished-runs derivation must use this exact set.
var TerminalStatuses = schema.TerminalStatuses

// DeltaColumns is the projection every delta for t returns: all of its
// physical table's columns, in declared order.
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
func SourceName(t Table) string { return "__fed_" + Physical[t].Name }

// LakeSource is the lake half of the union: a DuckLake catalog alias,
// optionally pinned to a snapshot so one query reads one lake version.
type LakeSource struct {
	Catalog  string
	Snapshot *int64
}

func (s LakeSource) table(name string) string {
	ref := quoteIdent(s.Catalog) + "." + quoteIdent(name)
	if s.Snapshot != nil {
		ref += fmt.Sprintf(" AT (VERSION => %d)", *s.Snapshot)
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
var sourceTimeColumn = map[Table]string{TableRuns: "queued_at", TableSpans: "run_queued_at", TableEvents: "received_at"}

// sourceCTE renders t's physical source CTE (lake ∪ delta, every physical
// column, in declared order) and its bound args. The lake side is cut at
// the watermark, as the delta is, so rows the lake holds past W (a newer
// export) aren't read twice. The delta's columns are
// cast to their lake types (JSON text to VARIANT, STRUCT lists), so the
// union is the physical table's own shape and any macro over the table
// reads it unchanged.
func sourceCTE(t Table, spec sourceSpec) (CTEDef, error) {
	phys, ok := Physical[t]
	if !ok {
		return CTEDef{}, fmt.Errorf("federate: no physical table for %q", t)
	}
	lake := make([]string, len(phys.Columns))
	delta := make([]string, len(phys.Columns))
	for i, c := range phys.Columns {
		lake[i] = quoteIdent(c.Name)
		if spec.DeltaColumns != nil && !spec.DeltaColumns[c.Name] {
			delta[i] = fmt.Sprintf("CAST(NULL AS %s) AS %s", c.PhysicalType(), quoteIdent(c.Name))
			continue
		}
		delta[i] = deltaSelect(c)
	}
	timeCol := quoteIdent(sourceTimeColumn[t])
	filter := fmt.Sprintf("%s >= ? AND %s < ? AND bucket_at <= ?", timeCol, timeCol)
	args := []any{spec.From, spec.To, spec.Watermark}
	if !spec.AllTenants {
		filter = "account_id = ?::UUID AND env_id = ?::UUID AND " + filter
		args = append([]any{spec.AccountID.String(), spec.EnvID.String()}, args...)
	}
	if spec.Delta == "" {
		return CTEDef{SQL: fmt.Sprintf("%s AS (\n  SELECT %s\n  FROM %s\n)",
			quoteIdent(SourceName(t)), strings.Join(lake, ", "), spec.Lake.table(phys.Name))}, nil
	}
	lakeSide := fmt.Sprintf("SELECT %s\n  FROM %s WHERE %s", strings.Join(lake, ", "), spec.Lake.table(phys.Name), filter)
	if spec.TopN != nil && spec.Collapses {
		where, whereArgs, err := predicatesSQL(t, spec.TopN.Predicates)
		if err != nil {
			return CTEDef{}, err
		}
		lakeSide = fmt.Sprintf("(%s AND %s\n  ORDER BY %s LIMIT %d)", lakeSide, where, orderSQL(spec.TopN.OrderBy, quoteIdent), spec.TopN.Limit)
		args = append(args, whereArgs...)
	}
	sql := fmt.Sprintf(`%s AS (
  %s
  UNION ALL
  SELECT %s
  FROM %s
)`, quoteIdent(SourceName(t)), lakeSide, strings.Join(delta, ", "), quoteIdent(spec.Delta))
	return CTEDef{SQL: sql, Args: args}, nil
}

func deltaSelect(c schema.Column) string {
	if c.PhysicalType() == c.Type.DuckDBType() {
		return quoteIdent(c.Name)
	}
	return fmt.Sprintf("CAST(%s AS %s) AS %s", quoteIdent(c.Name), c.PhysicalType(), quoteIdent(c.Name))
}

// rawView is the plain-SQL path's logical table for t: runs collapsed by
// inngest.raw_runs over their federated source (unless runsCollapsed, see
// Query.RunsCollapsed); spans and events are the source itself (each row is
// its own emission / event).
func rawView(t Table, lake LakeSource, runsCollapsed bool) (string, error) {
	switch t {
	case TableRuns:
		if runsCollapsed {
			return fmt.Sprintf("%s AS (SELECT * FROM %s)", quoteIdent(string(t)), quoteIdent(SourceName(t))), nil
		}
		return fmt.Sprintf("%s AS (SELECT * FROM %s.raw_runs(p_runs := '%s'))",
			quoteIdent(string(t)), quoteIdent(lake.Catalog), SourceName(t)), nil
	case TableSpans, TableEvents:
		return fmt.Sprintf("%s AS (SELECT * FROM %s)", quoteIdent(string(t)), quoteIdent(SourceName(t))), nil
	}
	return "", fmt.Errorf("federate: no view for table %q", t)
}
