package tracing

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/davecgh/go-spew/spew"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type DBExporter struct {
	q sqlc.Querier
}

func (e *DBExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()
		parentID := span.Parent().SpanID().String()
		runID := extractRunID(span.Attributes())

		attrs := make(map[string]interface{})
		for _, attr := range span.Attributes() {
			attrs[string(attr.Key)] = attr.Value.AsInterface()
		}
		data, err := json.Marshal(attrs)
		if err != nil {
			// TODO Log error
			spew.Dump("Failed to marshal span attributes", err)
			continue
		}

		err = e.q.InsertSpan(ctx, sqlc.InsertSpanParams{
			SpanID:       spanID,
			TraceID:      traceID,
			ParentSpanID: sql.NullString{String: parentID, Valid: parentID != ""},
			Name:         span.Name(),
			StartTime:    span.StartTime(),
			EndTime:      span.EndTime(),
			RunID:        sql.NullString{String: runID, Valid: runID != ""},
			Attributes:   string(data),
		})
		if err != nil {
			// TODO Log error
			spew.Dump("Failed to insert span", err)
			continue
		}
	}
	return nil
}

func (e *DBExporter) Shutdown(context.Context) error { return nil }
