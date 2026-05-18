package tracing

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	dbpkg "github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	cleanAttrs = false
)

func NewSqlcTracerProvider(q dbpkg.Querier) TracerProvider {
	return NewOtelTracerProvider(&dbExporter{q: q}, 5*time.Second)
}

type dbExporter struct {
	q dbpkg.Querier
}

func (e *dbExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if len(spans) == 0 {
		return nil
	}

	params := make([]dbpkg.InsertSpanParams, 0, len(spans))
	for _, span := range spans {
		p, ok := e.parseSpan(ctx, span)
		if !ok {
			continue
		}
		params = append(params, p)
	}

	if len(params) == 0 {
		return nil
	}

	if err := e.q.InsertSpans(ctx, params); err != nil {
		logger.StdlibLogger(ctx).Error("failed to bulk insert spans into database",
			"count", len(params),
			"error", err,
		)
	}
	return nil
}

// spanFields holds the extracted metadata from a span's attributes.
type spanFields struct {
	traceID        string
	spanID         string
	parentID       string
	envID          string
	accountID      string
	appID          string
	dynamicSpanID  string
	functionID     string
	output         any
	input          any
	runID          string
	debugSessionID string
	debugRunID     string
	status         string
	eventIdsByt    []byte
	attrs          map[string]any
}

// extractSpanFields iterates over a span's attributes and extracts known
// metadata fields into a spanFields struct. Generic attributes are collected
// into the attrs map.
func extractSpanFields(ctx context.Context, span sdktrace.ReadOnlySpan) spanFields {
	sf := spanFields{
		traceID:  span.SpanContext().TraceID().String(),
		spanID:   span.SpanContext().SpanID().String(),
		parentID: span.Parent().SpanID().String(),
		attrs:    make(map[string]any),
	}
	isExtensionSpan := span.Name() == meta.SpanNameDynamicExtension
	for _, attr := range span.Attributes() {
		if store := assignSpanAttr(ctx, &sf, attr, span.Name(), isExtensionSpan); store {
			sf.attrs[string(attr.Key)] = attr.Value.AsInterface()
		}
	}
	return sf
}

// assignSpanAttr extracts a known attribute into the spanFields struct and
// returns whether the attribute should also be stored in the generic attrs map.
func assignSpanAttr(ctx context.Context, sf *spanFields, attr attribute.KeyValue, spanName string, isExtensionSpan bool) bool {
	key := string(attr.Key)
	switch key {
	case meta.Attrs.StepOutput.Key():
		sf.output = attr.Value.AsInterface()
		return false
	case meta.Attrs.EventsInput.Key(), meta.Attrs.StepInput.Key():
		sf.input = attr.Value.AsInterface()
		return false
	case meta.Attrs.EventIDs.Key():
		if byt, err := json.Marshal(attr.Value.AsStringSlice()); err != nil {
			logger.StdlibLogger(ctx).Error("failed to marshal event IDs",
				"span_id", sf.spanID, "trace_id", sf.traceID,
				"name", spanName, "error", err,
			)
		} else {
			sf.eventIdsByt = byt
		}
		return !cleanAttrs
	case meta.Attrs.AccountID.Key():
		sf.accountID = attr.Value.AsString()
		return !cleanAttrs
	case meta.Attrs.EnvID.Key():
		sf.envID = attr.Value.AsString()
		return !cleanAttrs
	case meta.Attrs.RunID.Key():
		sf.runID = attr.Value.AsString()
		return !cleanAttrs
	case meta.Attrs.AppID.Key():
		sf.appID = attr.Value.AsString()
		return !cleanAttrs
	case meta.Attrs.FunctionID.Key():
		sf.functionID = attr.Value.AsString()
		return !cleanAttrs
	case meta.Attrs.DynamicTraceID.Key():
		sf.traceID = attr.Value.AsString()
		return !cleanAttrs
	case meta.Attrs.DynamicSpanID.Key():
		sf.dynamicSpanID = attr.Value.AsString()
		return !cleanAttrs
	case meta.Attrs.DebugSessionID.Key():
		sf.debugSessionID = attr.Value.AsString()
		return !cleanAttrs
	case meta.Attrs.DebugRunID.Key():
		sf.debugRunID = attr.Value.AsString()
		return !cleanAttrs
	case meta.Attrs.DynamicStatus.Key():
		sf.status = attr.Value.AsString()
		return !cleanAttrs
	case meta.Attrs.UserlandSpanID.Key():
		return !cleanAttrs
	case meta.Attrs.DropSpan.Key():
		return !cleanAttrs || !isExtensionSpan
	default:
		return true
	}
}

// marshalSpanJSON marshals the attributes map and links slice, returning the
// serialised bytes. Returns an error on marshal failure.
func marshalSpanJSON(ctx context.Context, sf spanFields, span sdktrace.ReadOnlySpan) (attrsByt, linksByt []byte, err error) {
	attrsByt, err = json.Marshal(sf.attrs)
	if err != nil {
		logger.StdlibLogger(ctx).Error("failed to marshal span attributes",
			"span_id", sf.spanID, "trace_id", sf.traceID,
			"name", span.Name(), "error", err,
		)
		return nil, nil, err
	}

	linksByt, err = json.Marshal(span.Links())
	if err != nil {
		logger.StdlibLogger(ctx).Error("failed to marshal span links",
			"span_id", sf.spanID, "trace_id", sf.traceID,
			"name", span.Name(), "error", err,
		)
		return nil, nil, err
	}
	return attrsByt, linksByt, nil
}

// buildInsertSpanParams constructs the DB insert params from the extracted
// fields and serialised JSON payloads.
func buildInsertSpanParams(sf spanFields, span sdktrace.ReadOnlySpan, attrsByt, linksByt []byte) dbpkg.InsertSpanParams {
	return dbpkg.InsertSpanParams{
		SpanID:       sf.spanID,
		TraceID:      sf.traceID,
		ParentSpanID: sql.NullString{String: sf.parentID, Valid: sf.parentID != ""},
		Name:         span.Name(),
		StartTime:    span.StartTime().Round(0),
		EndTime:      span.EndTime().Round(0),
		RunID:        sf.runID,
		AppID:        sf.appID,
		FunctionID:   sf.functionID,
		Attributes:   attrsByt,
		Links:        linksByt,
		DynamicSpanID: sql.NullString{
			String: sf.dynamicSpanID,
			Valid:  sf.dynamicSpanID != "",
		},
		AccountID: sf.accountID,
		EnvID:     sf.envID,
		Output:    anyToBytes(sf.output),
		Input:     anyToBytes(sf.input),
		DebugSessionID: sql.NullString{
			String: sf.debugSessionID,
			Valid:  sf.debugSessionID != "",
		},
		DebugRunID: sql.NullString{
			String: sf.debugRunID,
			Valid:  sf.debugRunID != "",
		},
		Status: sql.NullString{
			String: sf.status,
			Valid:  sf.status != "",
		},
		EventIds: sf.eventIdsByt,
	}
}

func (e *dbExporter) parseSpan(ctx context.Context, span sdktrace.ReadOnlySpan) (dbpkg.InsertSpanParams, bool) {
	sf := extractSpanFields(ctx, span)

	if sf.runID == "" {
		logger.StdlibLogger(ctx).Error("span missing run ID",
			"span_id", sf.spanID, "trace_id", sf.traceID,
			"name", span.Name(),
		)
		return dbpkg.InsertSpanParams{}, false
	}

	attrsByt, linksByt, err := marshalSpanJSON(ctx, sf, span)
	if err != nil {
		return dbpkg.InsertSpanParams{}, false
	}

	return buildInsertSpanParams(sf, span, attrsByt, linksByt), true
}

func (e *dbExporter) Shutdown(context.Context) error { return nil }

// anyToBytes converts a value to []byte for storage in a JSON column.
// Strings and byte slices are used directly to avoid double-encoding;
// other types are JSON-marshaled.
func anyToBytes(v any) []byte {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string:
		if val == "" {
			return nil
		}
		return []byte(val)
	case []byte:
		if len(val) == 0 {
			return nil
		}
		return val
	default:
		byt, _ := json.Marshal(val)
		return byt
	}
}
