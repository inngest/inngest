package resolvers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/inngest/inngest/pkg/duckdb/parser"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
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

	start := time.Now()

	// SHOW TABLES / DESCRIBE <table> answer entirely from this package's
	// own static schema registry (tables.go) -- no SQL parsing, no
	// rewrite, no database round trip. Checked before Transpile since
	// neither form is valid SELECT syntax the parser understands at all.
	if sc, ok, err := insights.TryShortCircuit(sql); ok {
		if err != nil {
			if verr, ok := errors.AsType[*insights.ValidationError](err); ok {
				recordInsightsQueryMetrics(ctx, start, "meta_error", "", 0)
				return emptyInsightsResultWithDiagnostic(verr.Diagnostic()), nil
			}
			recordInsightsQueryMetrics(ctx, start, "meta_error", "", 0)
			return nil, err
		}
		recordInsightsQueryMetrics(ctx, start, "meta", "", len(sc.Result.Rows))
		return toShortCircuitResult(sc), nil
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
			recordInsightsQueryMetrics(ctx, start, "validation_error", "", 0)
			return emptyInsightsResultWithDiagnostic(verr.Diagnostic()), nil
		}
		recordInsightsQueryMetrics(ctx, start, "parse_error", "", 0)
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
			recordInsightsQueryMetrics(ctx, start, "execution_error", tr.PrimaryTable, 0)
			return emptyInsightsResultWithDiagnostic(eerr.Diagnostic()), nil
		}
		recordInsightsQueryMetrics(ctx, start, "execution_error", tr.PrimaryTable, 0)
		return nil, err
	}

	recordInsightsQueryMetrics(ctx, start, "ok", tr.PrimaryTable, len(result.Rows))
	return toInsightsQueryResult(tr, result), nil
}

// recordInsightsQueryMetrics is this resolver's only per-query
// observability today: a status-tagged count, an end-to-end duration (from
// just before Transpile to the final outcome), and -- for a successful
// query -- how many rows it returned. primaryTable is tagged rather than
// the full table list to keep cardinality bounded to the six logical
// tables, not their combinations; it's "" for a query that never reached
// stageExtractQueryInfo (a parse/early-validation failure).
func recordInsightsQueryMetrics(ctx context.Context, start time.Time, status, primaryTable string, rowCount int) {
	tags := map[string]any{}
	if primaryTable != "" {
		tags["primary_table"] = primaryTable
	}
	metrics.IncrInsightsQueryCounter(ctx, status, metrics.CounterOpt{PkgName: pkgName, Tags: tags})
	metrics.HistogramInsightsQueryDuration(ctx, time.Since(start), metrics.HistogramOpt{PkgName: pkgName, Tags: tags})
	if status == "ok" {
		metrics.HistogramInsightsQueryRowCount(ctx, int64(rowCount), metrics.HistogramOpt{PkgName: pkgName, Tags: tags})
	}
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
			Name:      c.Name,
			Type:      toGQLColumnType(c.Type),
			PathHints: toGQLPathHints(c.PathHints),
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

// toShortCircuitResult mirrors toInsightsQueryResult's shape for a
// TryShortCircuit result -- no PrimaryTable/ColumnHints/Diagnostics/Limited,
// since none of those pipeline concepts apply to a query that never went
// through Transpile.
func toShortCircuitResult(sc *insights.ShortCircuitResult) *models.InsightsQueryResult {
	columns := make([]*models.InsightsQueryColumn, len(sc.Result.Columns))
	for i, c := range sc.Result.Columns {
		columns[i] = &models.InsightsQueryColumn{Name: c.Name, Type: toGQLColumnType(c.Type)}
	}
	return &models.InsightsQueryResult{
		Columns: columns,
		Rows:    sc.Result.Rows,
		Info:    &models.InsightsQueryInfo{Tables: sc.Tables},
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

// toGQLPathHints converts a column's whole []insights.PathHint verbatim --
// see gql.schema.graphql's InsightsPathHint/InsightsPathSegment doc
// comments for how a consumer (a UI cell-detail renderer, ultimately)
// reads path/wildcard entries the same way pkg/duckdb/insights' own
// resolveSubPathHint/resolveArrayElementHint do. A HintNone entry (never
// actually produced by pkg/duckdb/insights, whose own hint() lookups
// treat "not found" as HintNone without materializing an entry for it)
// is skipped defensively rather than surfaced as a meaningless hint.
func toGQLPathHints(hints []insights.PathHint) []*models.InsightsPathHint {
	out := make([]*models.InsightsPathHint, 0, len(hints))
	for _, ph := range hints {
		hint := toGQLColumnHint(ph.Hint)
		if hint == nil {
			continue
		}
		out = append(out, &models.InsightsPathHint{
			Path: toGQLPathSegments(ph.Path),
			Hint: *hint,
		})
	}
	return out
}

func toGQLPathSegments(path []insights.PathSegment) []*models.InsightsPathSegment {
	segs := make([]*models.InsightsPathSegment, len(path))
	for i, s := range path {
		seg := &models.InsightsPathSegment{Wildcard: s.Wildcard}
		if !s.Wildcard {
			key := s.Key
			seg.Key = &key
		}
		segs[i] = seg
	}
	return segs
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
	case insights.HintSession:
		hint = models.InsightsColumnHintSession
	default:
		return nil
	}
	return &hint
}
