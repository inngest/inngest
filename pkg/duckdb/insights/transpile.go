package insights

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// TranspileResult is Transpile's output: DuckDB-ready SQL text, its
// positional args, and everything the GQL layer needs to build an
// InsightsQueryResult without touching the database itself.
//
// Every logical table reference points at a reserved CTE (TableCTEName).
// SQL/Args are the query rendered with MacroSource (each CTE an inlined
// alias for its table macro); Render renders it with any other source, e.g.
// a federated executor's lake-plus-buffer bodies.
type TranspileResult struct {
	SQL          string
	Args         []any
	PrimaryTable string
	Tables       []string
	Limited      bool
	// Start/End are the original query's full source span (captured at
	// parse time, before any pipeline stage rewrites stmt), for Execute to
	// anchor an *ExecutionError's Diagnostic to when DuckDB itself reports
	// no usable position -- see ExecutionError's own doc comment for why
	// this is the whole query, not something more precise.
	Start, End parser.Position
	// ColumnPathHints has one column's-worth of pathHints per output
	// column, left to right, for a non-UNION query; only a root
	// (empty-Path) entry, if any, for a UNION/INTERSECT/EXCEPT query's
	// reconciled operands. An index beyond this slice's length, or a nil
	// entry, means no hint at all. See buildColumnPathHints' own doc
	// comment for exactly what's covered (including the empty-Path entry
	// that used to be a separate ColumnHints result) and the non-root
	// entries' own path-segment conventions.
	ColumnPathHints [][]PathHint
	// Diagnostics is every non-fatal note a pipeline stage produced.
	// Ordered by pipeline stage, not by source position.
	Diagnostics []Diagnostic
	// Pushdown is the pre-filter predicates a federated executor may push
	// into each logical table's delta source. See extractPushdown.
	Pushdown Pushdown

	// body is the rewritten query without the reserved CTEs; tables are
	// the logical tables it reads (one reserved CTE each), in first-seen
	// order.
	body     string
	bodyArgs []any
	tables   []string
	// accountID/envID scope the table macros a renderer calls; cat is the
	// mode's catalog.
	accountID, envID uuid.UUID
	cat              catalog
}

// Raw reports whether the query was transpiled in raw mode (TranspileRaw).
func (r *TranspileResult) Raw() bool { return r.cat.raw }

// ReadTables returns the logical tables the query reads, one reserved CTE
// each, in the order Render defines them.
func (r *TranspileResult) ReadTables() []string { return append([]string(nil), r.tables...) }

// pipelineState threads one query's working state through Transpile's
// ordered stage list. Each stage reads/writes only the fields it owns;
// stmt itself is mutated in place by remapTables/addDefaultLimit, matching
// parser.SelectStatement's own mutable-AST style.
type pipelineState struct {
	stmt      *parser.SelectStatement
	accountID uuid.UUID
	envID     uuid.UUID

	cat         catalog
	scope       *tableScope
	ctes        map[string]logicalTable
	info        QueryInfo
	pushdown    Pushdown
	pathHints   [][]PathHint
	tables      []string
	limited     bool
	diagnostics []Diagnostic
}

// stage is one step of Transpile's pipeline. Adding a stage is one new
// function plus one line in pipeline below — never an edit to Transpile
// itself.
type stage func(*pipelineState) error

var pipeline = []stage{
	stageCheckReservedNames,
	stageValidate,
	stageExtractQueryInfo,
	stageExtractPushdown,
	stageBuildColumnHints,
	stageRewriteArrayOfStructAccess,
	stageCastTZFunctions,
	stageRemapTables,
	stageAddDefaultLimit,
}

// stageValidate also resolves and stores ps.scope and ps.ctes: validate has
// already computed both internally to do its own checking, so later stages
// reuse that result instead of recomputing it. scope is nil for a
// top-level UNION/INTERSECT/EXCEPT statement — stageBuildColumnHints
// already handles that case. ctes lets stageBuildColumnHints resolve a
// UNION operand's own scope independently.
func stageValidate(ps *pipelineState) error {
	scope, ctes, diags, err := validate(ps.stmt, ps.cat)
	ps.diagnostics = append(ps.diagnostics, diags...)
	if err != nil {
		return err
	}
	ps.scope = scope
	ps.ctes = ctes
	return nil
}

func stageExtractQueryInfo(ps *pipelineState) error {
	ps.info = extractQueryInfo(ps.stmt)
	return nil
}

// stageExtractPushdown must run before remapTables, which rewrites away the
// logical table names it reads.
func stageExtractPushdown(ps *pipelineState) error {
	ps.pushdown = extractPushdown(ps.stmt, ps.cat.tables)
	return nil
}

func stageBuildColumnHints(ps *pipelineState) error {
	ps.pathHints = buildColumnPathHints(ps.stmt, ps.scope, ps.ctes)
	return nil
}

// stageRewriteArrayOfStructAccess runs after stageBuildColumnHints (hints
// resolve against the query's original, unrewritten shape) and before
// stageRemapTables. This rewrite only ever touches expression-position
// column references, never a FROM-clause table name, so it never collides
// with remapping.
func stageRewriteArrayOfStructAccess(ps *pipelineState) error {
	diags, err := rewriteArrayOfStructAccess(ps.stmt, ps.ctes, nil)
	ps.diagnostics = append(ps.diagnostics, diags...)
	return err
}

// stageCastTZFunctions runs after stageRewriteArrayOfStructAccess (hints
// and array-of-struct rewriting are unaffected either way, but this keeps
// every AST-rewrite stage grouped together) and before stageRemapTables.
func stageCastTZFunctions(ps *pipelineState) error {
	ps.diagnostics = append(ps.diagnostics, rewriteTZFunctions(ps.stmt)...)
	return nil
}

func stageCheckReservedNames(ps *pipelineState) error {
	return checkReservedNames(ps.stmt)
}

func stageRemapTables(ps *pipelineState) error {
	ps.tables = remapTables(ps.stmt, ps.cat.tables)
	return nil
}

func stageAddDefaultLimit(ps *pipelineState) error {
	outcome, err := addDefaultLimit(ps.stmt)
	if err != nil {
		return err
	}
	ps.limited = outcome != limitUnchanged
	switch outcome {
	case limitDefaulted:
		ps.diagnostics = append(ps.diagnostics, diagnosticAt(ps.stmt, DiagnosticInfo, "default-limit-applied",
			fmt.Sprintf("No LIMIT was specified; results were capped at %d rows.", defaultInsightsLimit)))
	case limitCapped:
		ps.diagnostics = append(ps.diagnostics, diagnosticAt(ps.stmt.Limit, DiagnosticInfo, "limit-capped",
			fmt.Sprintf("The requested LIMIT exceeds the maximum of %d rows; results were capped.", defaultInsightsLimit)))
	}
	return nil
}

// Transpile parses sql, then runs every stage in pipeline, in order. Never
// touches the database. A panic in any stage (an AST shape a stage or
// parser.Write doesn't handle) is returned as an error, never propagated:
// sql is user-controlled, so it must not be able to crash the caller.
func Transpile(sql string, accountID, envID uuid.UUID) (*TranspileResult, error) {
	return transpile(sql, accountID, envID, productCatalog)
}

// TranspileRaw transpiles sql in raw mode, for internal debugging and usage
// only — never expose it to customers: the query reads the physical tables
// themselves (runs, collapsed to one row per run, and run_trace_spans), with
// every column, across every tenant. account_id/env_id are ordinary
// columns, and pushable like any other.
func TranspileRaw(sql string) (*TranspileResult, error) {
	return transpile(sql, uuid.Nil, uuid.Nil, rawCatalog)
}

func transpile(sql string, accountID, envID uuid.UUID, cat catalog) (res *TranspileResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			res, err = nil, fmt.Errorf("insights: transpiling query: %v", r)
		}
	}()

	stmt, err := parser.ParseString(sql)
	if err != nil {
		return nil, fmt.Errorf("insights: parsing query: %w", err)
	}

	ps := &pipelineState{stmt: stmt, accountID: accountID, envID: envID, cat: cat}
	for _, st := range pipeline {
		if err := st(ps); err != nil {
			return nil, err
		}
	}

	var out strings.Builder
	if err := parser.Write(&out, ps.stmt); err != nil {
		return nil, fmt.Errorf("insights: rendering query: %w", err)
	}

	res = &TranspileResult{
		body:            out.String(),
		tables:          ps.tables,
		accountID:       accountID,
		envID:           envID,
		cat:             cat,
		Start:           stmt.Pos(),
		End:             stmt.End(),
		PrimaryTable:    ps.info.PrimaryTable,
		Tables:          ps.info.Tables,
		Limited:         ps.limited,
		ColumnPathHints: ps.pathHints,
		Diagnostics:     ps.diagnostics,
		Pushdown:        ps.pushdown,
	}
	if res.SQL, res.Args, err = res.Render(res.defaultSource()); err != nil {
		return nil, fmt.Errorf("insights: rendering query: %w", err)
	}
	return res, nil
}
