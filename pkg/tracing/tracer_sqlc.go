package tracing

import (
	"context"
	"database/sql"
	"encoding/json"

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
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()
		parentID := span.Parent().SpanID().String()
		isExtensionSpan := span.Name() == meta.SpanNameDynamicExtension
		var envID string
		var accountID string
		var appID string
		var dynamicSpanID string
		var functionID string
		var output any
		var input any
		var runID string
		var debugSessionID string
		var debugRunID string
		var status string
		var eventIdsByt []byte

		attrs := make(map[string]any)
		for _, attr := range span.Attributes() {
			// If output, extract and store separately
			// This is always cleaned
			if string(attr.Key) == meta.Attrs.StepOutput.Key() {
				output = attr.Value.AsInterface()
				continue
			}

			// If input, extract and store separately
			// This is always cleaned
			if string(attr.Key) == meta.Attrs.EventsInput.Key() {
				input = attr.Value.AsInterface()
				continue
			}

			if string(attr.Key) == meta.Attrs.EventIDs.Key() {
				var err error
				// We store event IDs as a JSON array of strings
				if eventIdsByt, err = json.Marshal(attr.Value.AsStringSlice()); err != nil {
					logger.StdlibLogger(ctx).Error("failed to marshal event IDs",
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
				if cleanAttrs {
					continue
				}
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

			// If we've been given a trace ID, it should overwrite whatever
			// we've found in the span's own context; the caller knows best
			if string(attr.Key) == meta.Attrs.DynamicTraceID.Key() {
				traceID = attr.Value.AsString()
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

			if string(attr.Key) == meta.Attrs.DynamicStatus.Key() {
				status = attr.Value.AsString()
				if cleanAttrs {
					continue
				}
			}

			attrs[string(attr.Key)] = attr.Value.AsInterface()
		}

		// If we don't have a run ID, we can't store this span
		if runID == "" {
			logger.StdlibLogger(ctx).Error("span missing run ID",
				"span_id", spanID,
				"trace_id", traceID,
				"parent_id", parentID,
				"name", span.Name(),
				"start_time", span.StartTime(),
				"end_time", span.EndTime(),
				"app_id", appID,
				"function_id", functionID,
			)
			continue
		}

		attrsByt, err := json.Marshal(attrs)
		if err != nil {
			logger.StdlibLogger(ctx).Error("failed to marshal span attributes",
				"span_id", spanID,
				"trace_id", traceID,
				"parent_id", parentID,
				"name", span.Name(),
				"start_time", span.StartTime(),
				"end_time", span.EndTime(),
				"app_id", appID,
				"function_id", functionID,
				"error", err,
			)
			continue
		}

		linksByt, err := json.Marshal(span.Links())
		if err != nil {
			logger.StdlibLogger(ctx).Error("failed to marshal span links",
				"span_id", spanID,
				"trace_id", traceID,
				"parent_id", parentID,
				"name", span.Name(),
				"start_time", span.StartTime(),
				"end_time", span.EndTime(),
				"app_id", appID,
				"function_id", functionID,
				"error", err,
			)
			continue
		}

		err = e.q.InsertSpan(ctx, sqlc.InsertSpanParams{
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
			Input:     input,
			DebugSessionID: sql.NullString{
				String: debugSessionID,
				Valid:  debugSessionID != "",
			},
			DebugRunID: sql.NullString{
				String: debugRunID,
				Valid:  debugRunID != "",
			},
			Status: sql.NullString{
				String: status,
				Valid:  status != "",
			},
			EventIds: string(eventIdsByt),
		})
		if err != nil {
			logger.StdlibLogger(ctx).Error("failed to insert span into database",
				"span_id", spanID,
				"trace_id", traceID,
				"parent_id", parentID,
				"name", span.Name(),
				"start_time", span.StartTime(),
				"end_time", span.EndTime(),
				"app_id", appID,
				"function_id", functionID,
				"error", err,
			)
			continue
		}
	}
	return nil
}

func (e *dbExporter) Shutdown(context.Context) error { return nil }
