package insights

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// TranspileResult is Transpile's output: DuckDB-ready SQL text, its
// positional args, and everything the GQL layer needs to build an
// InsightsQueryResult without touching the database itself.
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
	// ColumnHints has one entry per output column for a non-UNION query,
	// and is empty for a UNION/INTERSECT/EXCEPT query. An index beyond
	// this slice's length is treated as HintNone.
	ColumnHints []ColumnHint
	// Diagnostics is every non-fatal note a pipeline stage produced.
	// Ordered by pipeline stage, not by source position.
	Diagnostics []Diagnostic
}

// pipelineState threads one query's working state through Transpile's
// ordered stage list. Each stage reads/writes only the fields it owns;
// stmt itself is mutated in place by remapTables/addDefaultLimit, matching
// parser.SelectStatement's own mutable-AST style.
type pipelineState struct {
	stmt      *parser.SelectStatement
	accountID uuid.UUID
	envID     uuid.UUID

	scope       *tableScope
	ctes        map[string]logicalTable
	info        QueryInfo
	hints       []ColumnHint
	args        []any
	limited     bool
	diagnostics []Diagnostic
}

// stage is one step of Transpile's pipeline. Adding a stage is one new
// function plus one line in pipeline below — never an edit to Transpile
// itself.
type stage func(*pipelineState) error

var pipeline = []stage{
	stageValidate,
	stageExtractQueryInfo,
	stageBuildColumnHints,
	stageRewriteArrayOfStructAccess,
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
	scope, ctes, err := validate(ps.stmt)
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

func stageBuildColumnHints(ps *pipelineState) error {
	ps.hints = buildColumnHints(ps.stmt, ps.scope, ps.ctes)
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

func stageRemapTables(ps *pipelineState) error {
	ps.args = remapTables(ps.stmt, ps.accountID, ps.envID)
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
// touches the database.
func Transpile(sql string, accountID, envID uuid.UUID) (*TranspileResult, error) {
	stmt, err := parser.ParseString(sql)
	if err != nil {
		return nil, fmt.Errorf("insights: parsing query: %w", err)
	}

	ps := &pipelineState{stmt: stmt, accountID: accountID, envID: envID}
	for _, st := range pipeline {
		if err := st(ps); err != nil {
			return nil, err
		}
	}

	return &TranspileResult{
		SQL:          parser.String(ps.stmt),
		Args:         ps.args,
		Start:        stmt.Pos(),
		End:          stmt.End(),
		PrimaryTable: ps.info.PrimaryTable,
		Tables:       ps.info.Tables,
		Limited:      ps.limited,
		ColumnHints:  ps.hints,
		Diagnostics:  ps.diagnostics,
	}, nil
}
