package resolvers

import (
	"context"
	"errors"
	"fmt"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// Insights backs Query.insights — see
// docs/plans/010-duckdb-insights-query-layer.md. Requires --duckdb dual-
// write to have started successfully (qr.DuckDB non-nil); unlike Data
// (Runs/RunTrace/etc.), there's no non-DuckDB fallback to fall through to,
// so a nil DuckDB errors clearly rather than resolving empty — matching
// 007/008's precedent of a clear GQL error over a silent empty result.
func (qr *queryResolver) Insights(ctx context.Context, sql string) (*models.InsightsQueryResult, error) {
	if qr.DuckDB == nil {
		return nil, fmt.Errorf("insights requires dual-write (--duckdb) to be enabled")
	}

	// consts.DevServerAccountID/EnvID, matching every other resolver in
	// this package (app.go, events.go, runs_v2.go, ...) — this GQL layer
	// has no per-request, auth-derived account/env ID to draw from today.
	// See docs/plans/012-duckdb-insights-query-layer-plan.md's Global
	// Constraints on env scoping.
	tr, err := insights.Transpile(sql, consts.DevServerAccountID, consts.DevServerEnvID)
	if err != nil {
		// A rejected query (bad SQL, unknown table/column, disallowed
		// function, ...) still resolves as a normal (empty) result with a
		// fatal diagnostic attached, rather than a bare top-level GQL
		// error -- this is what lets the UI point at exactly where the
		// query is wrong (a Monaco marker) instead of just showing a
		// generic error banner. Transpile's own contract is unchanged: it
		// still rejects the query outright, same as ever.
		if verr, ok := errors.AsType[*insights.ValidationError](err); ok {
			return emptyInsightsResultWithDiagnostic(verr.Diagnostic()), nil
		}
		return nil, err
	}

	result, err := insights.Execute(ctx, qr.DuckDB, tr)
	if err != nil {
		// A query that parsed and validated can still fail once it
		// actually runs (a stale allowlist entry a macro doesn't project,
		// a runtime type conversion this package's static checks can't
		// see coming, ...). Route it through the same diagnostic-result
		// path as a *ValidationError rather than a bare top-level GQL
		// error, for the same reason: the UI can render it as a query
		// problem instead of a generic error banner.
		if eerr, ok := errors.AsType[*insights.ExecutionError](err); ok {
			return emptyInsightsResultWithDiagnostic(eerr.Diagnostic()), nil
		}
		return nil, err
	}

	return toInsightsQueryResult(tr, result), nil
}

func emptyInsightsResultWithDiagnostic(d insights.Diagnostic) *models.InsightsQueryResult {
	return &models.InsightsQueryResult{
		Columns:     []*models.InsightsQueryColumn{},
		Rows:        [][]any{},
		Info:        &models.InsightsQueryInfo{Tables: []string{}},
		Diagnostics: []*models.InsightsDiagnostic{toGQLDiagnostic(d)},
	}
}

func toInsightsQueryResult(tr *insights.TranspileResult, result *insights.Result) *models.InsightsQueryResult {
	columns := make([]*models.InsightsQueryColumn, len(result.Columns))
	for i, c := range result.Columns {
		columns[i] = &models.InsightsQueryColumn{
			Name: c.Name,
			Type: toGQLColumnType(c.Type),
			Hint: toGQLColumnHint(c.Hint),
		}
	}

	var primaryTable *string
	if tr.PrimaryTable != "" {
		primaryTable = &tr.PrimaryTable
	}

	diagnostics := make([]*models.InsightsDiagnostic, len(tr.Diagnostics))
	for i, d := range tr.Diagnostics {
		diagnostics[i] = toGQLDiagnostic(d)
	}

	return &models.InsightsQueryResult{
		Columns: columns,
		Rows:    result.Rows,
		Info: &models.InsightsQueryInfo{
			PrimaryTable: primaryTable,
			Tables:       tr.Tables,
			Limited:      tr.Limited,
		},
		Diagnostics: diagnostics,
	}
}

func toGQLDiagnostic(d insights.Diagnostic) *models.InsightsDiagnostic {
	return &models.InsightsDiagnostic{
		Start:    toGQLPosition(d.Start),
		End:      toGQLPosition(d.End),
		Severity: toGQLDiagnosticSeverity(d.Severity),
		Code:     d.Code,
		Message:  d.Message,
	}
}

func toGQLPosition(p parser.Position) *models.InsightsDiagnosticPosition {
	return &models.InsightsDiagnosticPosition{Line: p.Line, Column: p.Column}
}

func toGQLDiagnosticSeverity(s insights.DiagnosticSeverity) models.InsightsDiagnosticSeverity {
	switch s {
	case insights.DiagnosticWarning:
		return models.InsightsDiagnosticSeverityWarning
	case insights.DiagnosticError:
		return models.InsightsDiagnosticSeverityError
	default:
		return models.InsightsDiagnosticSeverityInfo
	}
}

func toGQLColumnType(t insights.ColumnType) models.InsightsColumnType {
	switch t {
	case insights.ColumnTypeString:
		return models.InsightsColumnTypeString
	case insights.ColumnTypeNumber:
		return models.InsightsColumnTypeNumber
	case insights.ColumnTypeBoolean:
		return models.InsightsColumnTypeBoolean
	case insights.ColumnTypeDatetime:
		return models.InsightsColumnTypeDatetime
	case insights.ColumnTypeJSON:
		return models.InsightsColumnTypeJSON
	default:
		return models.InsightsColumnTypeUnknown
	}
}

func toGQLColumnHint(h insights.ColumnHint) *models.InsightsColumnHint {
	var hint models.InsightsColumnHint
	switch h {
	case insights.HintAppID:
		hint = models.InsightsColumnHintAppID
	case insights.HintFunctionID:
		hint = models.InsightsColumnHintFunctionID
	case insights.HintRunID:
		hint = models.InsightsColumnHintRunID
	case insights.HintEventID:
		hint = models.InsightsColumnHintEventID
	default:
		return nil
	}
	return &hint
}
