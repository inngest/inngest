package tracing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing/meta"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	cleanAttrs = false
)

func NewSqlcTracerProvider(q sqlc.Querier) TracerProvider {
	return NewOtelTracerProvider(&dbExporter{q: q})
}

type dbExporter struct {
	q sqlc.Querier
}

func (e *dbExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		e.exportSpan(ctx, span)
	}

	return nil
}

func (e *dbExporter) exportSpan(ctx context.Context, span sdktrace.ReadOnlySpan) {
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	parentID := span.Parent().SpanID().String()
	isExtensionSpan := span.Name() == meta.SpanNameDynamicExtension

	var (
		err            error
		envID          string
		accountID      string
		appID          string
		dynamicSpanID  string
		functionID     string
		output         interface{}
		runID          string
		debugSessionID string
		debugRunID     string
	)

	defer func() {
		if err != nil {
			logger.StdlibLogger(ctx).Error(err.Error(),
				"span_id", spanID,
				"trace_id", traceID,
				"parent_id", parentID,
				"name", span.Name(),
				"start_time", span.StartTime(),
				"end_time", span.EndTime(),
				"app_id", appID,
				"function_id", functionID,
			)
		}

		meta.NotifySpanExported(spanID, err)
	}()

	attrs := make(map[string]any)
	for _, attr := range span.Attributes() {
		// If output, extract and store separately
		// This is always cleaned
		if string(attr.Key) == meta.Attrs.StepOutput.Key() {
			output = attr.Value.AsInterface()
			continue
		}

		if string(attr.Key) == meta.Attrs.AccountID.Key() {
			accountID = attr.Value.AsString()
			if cleanAttrs {
				continue
			}
		}

		if string(attr.Key) == meta.Attrs.EnvID.Key() {
			envID = attr.Value.AsString()
			if cleanAttrs {
				continue
			}
		}

		// Capture but omit the run ID attribute from the span attributes
		if string(attr.Key) == meta.Attrs.RunID.Key() {
			runID = attr.Value.AsString()
			if cleanAttrs {
				continue
			}
		}

		if string(attr.Key) == meta.Attrs.AppID.Key() {
			appID = attr.Value.AsString()
			if cleanAttrs {
				continue
			}
		}

		if string(attr.Key) == meta.Attrs.FunctionID.Key() {
			functionID = attr.Value.AsString()
			if cleanAttrs {
				continue
			}
		}

		// Capture but omit the dynamic span ID attribute from the span attributes
		if string(attr.Key) == meta.Attrs.DynamicSpanID.Key() {
			dynamicSpanID = attr.Value.AsString()
			if cleanAttrs {
				continue
			}
		}

		// Omit drop span attribute if we're an extension span
		if isExtensionSpan && string(attr.Key) == meta.Attrs.DropSpan.Key() {
			if cleanAttrs {
				continue
			}
		}

		if string(attr.Key) == meta.Attrs.DebugSessionID.Key() {
			debugSessionID = attr.Value.AsString()
			if cleanAttrs {
				continue
			}
		}

		if string(attr.Key) == meta.Attrs.DebugRunID.Key() {
			debugRunID = attr.Value.AsString()
			if cleanAttrs {
				continue
			}
		}

		attrs[string(attr.Key)] = attr.Value.AsInterface()
	}

	// If we don't have a run ID, we can't store this span
	if runID == "" {
		err = fmt.Errorf("span missing run ID")
		return
	}

	attrsByt, err := json.Marshal(attrs)
	if err != nil {
		err = fmt.Errorf("failed to marshal span attributes: %w", err)
		return
	}

	linksByt, err := json.Marshal(span.Links())
	if err != nil {
		err = fmt.Errorf("failed to marshal span links: %w", err)
		return
	}

	if err = e.q.InsertSpan(ctx, sqlc.InsertSpanParams{
		SpanID:       spanID,
		TraceID:      traceID,
		ParentSpanID: sql.NullString{String: parentID, Valid: parentID != ""},
		Name:         span.Name(),
		StartTime:    span.StartTime(),
		EndTime:      span.EndTime(),
		RunID:        runID,
		AppID:        appID,
		FunctionID:   functionID,
		Attributes:   string(attrsByt),
		Links:        string(linksByt),
		DynamicSpanID: sql.NullString{
			String: dynamicSpanID,
			Valid:  dynamicSpanID != "",
		},
		AccountID: accountID,
		EnvID:     envID,
		Output:    output,
		DebugSessionID: sql.NullString{
			String: debugSessionID,
			Valid:  debugSessionID != "",
		},
		DebugRunID: sql.NullString{
			String: debugRunID,
			Valid:  debugRunID != "",
		},
	}); err != nil {
		err = fmt.Errorf("failed to insert span into database: %w", err)
	}
}

func (e *dbExporter) Shutdown(context.Context) error { return nil }
