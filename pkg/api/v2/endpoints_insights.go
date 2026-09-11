package apiv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// QueryInsights backs POST /v2/insights/query -- the REST counterpart of
// GQL's Query.insights (pkg/coreapi/graph/resolvers/insights.go), running
// the same pkg/duckdb/insights pipeline: TryShortCircuit (SHOW TABLES/
// DESCRIBE) short-circuits entirely, otherwise Transpile then Execute
// against s.duckDB. Like that resolver, a query that fails to validate or
// execute still resolves as a normal (200) response with a fatal
// diagnostic attached rather than a top-level error -- QueryInsightsData's
// own diagnostics field exists for exactly this, matching the resolver's
// precedent that this is a query problem a client can point back at, not
// an API failure.
func (s *Service) QueryInsights(ctx context.Context, req *apiv2.QueryInsightsRequest) (*apiv2.QueryInsightsResponse, error) {
	if s.duckDB == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Insights requires dual-write (--duckdb) to be enabled")
	}

	if sc, ok, err := insights.TryShortCircuit(req.Query); ok {
		if err != nil {
			if verr, ok := errors.AsType[*insights.ValidationError](err); ok {
				return queryInsightsDiagnosticResponse(verr.Diagnostic(), req.Query), nil
			}
			return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, err.Error())
		}
		return toQueryInsightsResponse(nil, sc.Result)
	}

	// consts.DevServerAccountID/EnvID, matching every other apiv2 endpoint
	// (endpoints.go's FetchAccount/FetchEnv) and the GQL resolver -- this
	// service has no per-request, auth-derived account/env ID today.
	tr, err := insights.Transpile(req.Query, consts.DevServerAccountID, consts.DevServerEnvID)
	if err != nil {
		if verr, ok := errors.AsType[*insights.ValidationError](err); ok {
			return queryInsightsDiagnosticResponse(verr.Diagnostic(), req.Query), nil
		}
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorValidationError, err.Error())
	}

	result, err := insights.Execute(ctx, s.duckDB, tr)
	if err != nil {
		if eerr, ok := errors.AsType[*insights.ExecutionError](err); ok {
			return queryInsightsDiagnosticResponse(eerr.Diagnostic(), req.Query), nil
		}
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, err.Error())
	}

	return toQueryInsightsResponse(tr.Diagnostics, result)
}

// ListInsightsTables backs GET /v2/insights/tables, describing the same six
// logical tables QueryInsights validates against. Unlike QueryInsights,
// this never touches s.duckDB -- insights.AllTableSchemas() is this
// package's own static registry, the same source of truth DESCRIBE/SHOW
// TABLES short-circuit from -- so this works even without --duckdb dual-
// write enabled.
func (s *Service) ListInsightsTables(ctx context.Context, req *apiv2.ListInsightsTablesRequest) (*apiv2.ListInsightsTablesResponse, error) {
	schemas := insights.AllTableSchemas()
	tables := make([]*apiv2.InsightsTable, len(schemas))
	for i, schema := range schemas {
		tables[i] = toInsightsTable(schema)
	}
	return &apiv2.ListInsightsTablesResponse{
		Data:     tables,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}, nil
}

// QueryInsightsPrompt would generate an Insights SQL query from a natural
// language prompt. No such capability exists anywhere in this codebase yet
// (no LLM integration backs pkg/duckdb/insights) -- unlike QueryInsights/
// ListInsightsTables, there's nothing here to wire up, so this stays a
// stub until that capability exists.
func (s *Service) QueryInsightsPrompt(ctx context.Context, req *apiv2.QueryInsightsPromptRequest) (*apiv2.QueryInsightsPromptResponse, error) {
	return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Insights not implemented in OSS")
}

// ListInsightsEventSchemas would list every ingested event type's inferred
// JSON schema. pkg/duckdb/insights has no event-schema-catalog concept --
// it exposes six fixed logical tables (including a passthrough "events"
// table), not per-event-type schema inference -- so this stays a stub
// until that capability exists.
func (s *Service) ListInsightsEventSchemas(ctx context.Context, req *apiv2.ListInsightsEventSchemasRequest) (*apiv2.ListInsightsEventSchemasResponse, error) {
	return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Insights not implemented in OSS")
}

func toQueryInsightsResponse(diagnostics []insights.Diagnostic, result *insights.Result) (*apiv2.QueryInsightsResponse, error) {
	columns := make([]*apiv2.InsightsOutputColumn, len(result.Columns))
	for i, c := range result.Columns {
		columns[i] = &apiv2.InsightsOutputColumn{Name: c.Name, Type: toInsightsOutputColumnType(c.Type)}
	}

	rows := make([]*apiv2.InsightsRow, len(result.Rows))
	for i, row := range result.Rows {
		r, err := toInsightsRow(row)
		if err != nil {
			return nil, err
		}
		rows[i] = r
	}

	diags := make([]*apiv2.InsightsDiagnostic, len(diagnostics))
	for i, d := range diagnostics {
		diags[i] = toInsightsDiagnostic(d, "")
	}

	return &apiv2.QueryInsightsResponse{
		Data: &apiv2.QueryInsightsData{
			Columns:     columns,
			Rows:        rows,
			Diagnostics: diags,
		},
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}, nil
}

// queryInsightsDiagnosticResponse mirrors the GQL resolver's
// emptyInsightsResultWithDiagnostic: a query that fails validation or
// execution still gets a normal response, with empty columns/rows and a
// single fatal diagnostic in its place. query is the original request text,
// used to slice out the diagnostic's context snippet.
func queryInsightsDiagnosticResponse(d insights.Diagnostic, query string) *apiv2.QueryInsightsResponse {
	return &apiv2.QueryInsightsResponse{
		Data: &apiv2.QueryInsightsData{
			Columns:     []*apiv2.InsightsOutputColumn{},
			Rows:        []*apiv2.InsightsRow{},
			Diagnostics: []*apiv2.InsightsDiagnostic{toInsightsDiagnostic(d, query)},
		},
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}
}

func toInsightsRow(row []any) (*apiv2.InsightsRow, error) {
	values := make([]*structpb.Value, len(row))
	for i, v := range row {
		pv, err := driverValueToProtoValue(v)
		if err != nil {
			return nil, fmt.Errorf("converting column %d: %w", i, err)
		}
		values[i] = pv
	}
	return &apiv2.InsightsRow{Values: values}, nil
}

// driverValueToProtoValue converts one decoded DuckDB cell (execute.go's
// Result.Rows -- "the driver's own native decoded Go value, raw, not
// pre-stringified") into a structpb.Value via a JSON round trip. The
// driver can return types structpb.NewValue doesn't accept directly (e.g.
// time.Time), but every one of them already has the JSON encoding this
// package's GQL twin relies on (gql_scalars.MarshalUnknown, which also
// just calls json.Marshal) -- round-tripping through JSON reuses that
// same encoding instead of hand-rolling a second conversion per driver
// type.
func driverValueToProtoValue(v any) (*structpb.Value, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshaling value: %w", err)
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("unmarshaling value: %w", err)
	}
	return structpb.NewValue(decoded)
}

func toInsightsOutputColumnType(t insights.ColumnType) apiv2.InsightsOutputColumnType {
	switch t {
	case insights.ColumnTypeString:
		return apiv2.InsightsOutputColumnType_STRING
	case insights.ColumnTypeNumber:
		return apiv2.InsightsOutputColumnType_NUMBER
	case insights.ColumnTypeBoolean:
		return apiv2.InsightsOutputColumnType_BOOLEAN
	case insights.ColumnTypeDatetime:
		return apiv2.InsightsOutputColumnType_DATETIME
	case insights.ColumnTypeJSON:
		return apiv2.InsightsOutputColumnType_COMPLEX
	default:
		return apiv2.InsightsOutputColumnType_VALUE_TYPE_UNSPECIFIED
	}
}

func toInsightsDiagnosticSeverity(s insights.DiagnosticSeverity) apiv2.InsightsDiagnosticSeverity {
	switch s {
	case insights.DiagnosticError:
		return apiv2.InsightsDiagnosticSeverity_ERROR
	case insights.DiagnosticWarning:
		return apiv2.InsightsDiagnosticSeverity_WARNING
	default:
		return apiv2.InsightsDiagnosticSeverity_INFO
	}
}

// toInsightsDiagnostic converts one insights.Diagnostic into its apiv2
// wire shape. query is the original request text the diagnostic's
// Start/End offsets index into, used to slice out its context snippet --
// "" (no snippet) for a diagnostic with no meaningful source text to slice
// (a short-circuited DESCRIBE/SHOW TABLES query never produces one).
func toInsightsDiagnostic(d insights.Diagnostic, query string) *apiv2.InsightsDiagnostic {
	start, end := d.Start.Offset, d.End.Offset
	var position *apiv2.InsightsDiagnosticPosition
	if query != "" && start >= 0 && end >= start && end <= len(query) {
		position = &apiv2.InsightsDiagnosticPosition{
			Start:   int32(start),
			End:     int32(end),
			Context: query[start:end],
		}
	}
	return &apiv2.InsightsDiagnostic{
		Severity: toInsightsDiagnosticSeverity(d.Severity),
		Code:     d.Code,
		Message:  d.Message,
		Position: position,
	}
}

func toInsightsTable(schema insights.TableSchema) *apiv2.InsightsTable {
	columns := make([]*apiv2.InsightsTableColumn, len(schema.Columns))
	for i, c := range schema.Columns {
		columns[i] = &apiv2.InsightsTableColumn{
			Name:        c.Name,
			Description: c.Description,
			Type:        c.Type.String(),
		}
	}
	return &apiv2.InsightsTable{
		Name:        schema.Name,
		Description: schema.Description,
		Columns:     columns,
	}
}
