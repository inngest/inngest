package logs

import (
	"time"

	"go.opentelemetry.io/otel/trace"
	cpb "go.opentelemetry.io/proto/otlp/common/v1"
	lpb "go.opentelemetry.io/proto/otlp/logs/v1"

	api "go.opentelemetry.io/otel/log"
)

type ParsedRecord struct {
	api.Record

	TraceID string
	SpanID  string
}

// LogRecordFromProto converts an OTLP LogRecord back to api.Record format.
func LogRecordFromProto(lr *lpb.LogRecord) ParsedRecord {
	// Create an API record and set its properties
	apiRecord := api.Record{}

	apiRecord.SetTimestamp(time.Unix(0, int64(lr.TimeUnixNano)))
	apiRecord.SetObservedTimestamp(time.Unix(0, int64(lr.ObservedTimeUnixNano)))
	apiRecord.SetEventName(lr.EventName)
	apiRecord.SetSeverity(SeverityFromProto(lr.SeverityNumber))
	apiRecord.SetSeverityText(lr.SeverityText)

	if lr.Body != nil {
		apiRecord.SetBody(LogValueFromProto(lr.Body))
	}

	// Add attributes
	attrs := make([]api.KeyValue, len(lr.Attributes))
	for i, attr := range lr.Attributes {
		attrs[i] = api.KeyValue{
			Key:   attr.Key,
			Value: LogValueFromProto(attr.Value),
		}
	}
	apiRecord.AddAttributes(attrs...)

	// Set trace context if present
	if len(lr.TraceId) > 0 {
		var traceID trace.TraceID
		copy(traceID[:], lr.TraceId)
		// Note: TraceID/SpanID are typically set via trace context, not directly on the record
	}

	var traceIDStr, spanIDStr string
	{
		if lr.TraceId != nil {
			if traceID := trace.TraceID(lr.TraceId); traceID.IsValid() {
				traceIDStr = traceID.String()
			}
		}

		if lr.SpanId != nil {
			if spanID := trace.SpanID(lr.SpanId); spanID.IsValid() {
				spanIDStr = spanID.String()
			}
		}
	}

	return ParsedRecord{
		Record:  apiRecord,
		TraceID: traceIDStr,
		SpanID:  spanIDStr,
	}
}

// SeverityFromProto converts OTLP SeverityNumber to api.Severity.
func SeverityFromProto(s lpb.SeverityNumber) api.Severity {
	switch s {
	case lpb.SeverityNumber_SEVERITY_NUMBER_TRACE:
		return api.SeverityTrace
	case lpb.SeverityNumber_SEVERITY_NUMBER_TRACE2:
		return api.SeverityTrace2
	case lpb.SeverityNumber_SEVERITY_NUMBER_TRACE3:
		return api.SeverityTrace3
	case lpb.SeverityNumber_SEVERITY_NUMBER_TRACE4:
		return api.SeverityTrace4
	case lpb.SeverityNumber_SEVERITY_NUMBER_DEBUG:
		return api.SeverityDebug
	case lpb.SeverityNumber_SEVERITY_NUMBER_DEBUG2:
		return api.SeverityDebug2
	case lpb.SeverityNumber_SEVERITY_NUMBER_DEBUG3:
		return api.SeverityDebug3
	case lpb.SeverityNumber_SEVERITY_NUMBER_DEBUG4:
		return api.SeverityDebug4
	case lpb.SeverityNumber_SEVERITY_NUMBER_INFO:
		return api.SeverityInfo
	case lpb.SeverityNumber_SEVERITY_NUMBER_INFO2:
		return api.SeverityInfo2
	case lpb.SeverityNumber_SEVERITY_NUMBER_INFO3:
		return api.SeverityInfo3
	case lpb.SeverityNumber_SEVERITY_NUMBER_INFO4:
		return api.SeverityInfo4
	case lpb.SeverityNumber_SEVERITY_NUMBER_WARN:
		return api.SeverityWarn
	case lpb.SeverityNumber_SEVERITY_NUMBER_WARN2:
		return api.SeverityWarn2
	case lpb.SeverityNumber_SEVERITY_NUMBER_WARN3:
		return api.SeverityWarn3
	case lpb.SeverityNumber_SEVERITY_NUMBER_WARN4:
		return api.SeverityWarn4
	case lpb.SeverityNumber_SEVERITY_NUMBER_ERROR:
		return api.SeverityError
	case lpb.SeverityNumber_SEVERITY_NUMBER_ERROR2:
		return api.SeverityError2
	case lpb.SeverityNumber_SEVERITY_NUMBER_ERROR3:
		return api.SeverityError3
	case lpb.SeverityNumber_SEVERITY_NUMBER_ERROR4:
		return api.SeverityError4
	case lpb.SeverityNumber_SEVERITY_NUMBER_FATAL:
		return api.SeverityFatal
	case lpb.SeverityNumber_SEVERITY_NUMBER_FATAL2:
		return api.SeverityFatal2
	case lpb.SeverityNumber_SEVERITY_NUMBER_FATAL3:
		return api.SeverityFatal3
	case lpb.SeverityNumber_SEVERITY_NUMBER_FATAL4:
		return api.SeverityFatal4
	default:
		return api.SeverityInfo
	}
}

// LogValueFromProto converts OTLP AnyValue to api.Value.
func LogValueFromProto(av *cpb.AnyValue) api.Value {
	if av == nil {
		return api.Value{}
	}

	switch v := av.Value.(type) {
	case *cpb.AnyValue_BoolValue:
		return api.BoolValue(v.BoolValue)
	case *cpb.AnyValue_IntValue:
		return api.Int64Value(v.IntValue)
	case *cpb.AnyValue_DoubleValue:
		return api.Float64Value(v.DoubleValue)
	case *cpb.AnyValue_StringValue:
		return api.StringValue(v.StringValue)
	case *cpb.AnyValue_BytesValue:
		return api.BytesValue(v.BytesValue)
	case *cpb.AnyValue_ArrayValue:
		vals := make([]api.Value, len(v.ArrayValue.Values))
		for i, val := range v.ArrayValue.Values {
			vals[i] = LogValueFromProto(val)
		}
		return api.SliceValue(vals...)
	case *cpb.AnyValue_KvlistValue:
		kvs := make([]api.KeyValue, len(v.KvlistValue.Values))
		for i, kv := range v.KvlistValue.Values {
			kvs[i] = api.KeyValue{
				Key:   kv.Key,
				Value: LogValueFromProto(kv.Value),
			}
		}
		return api.MapValue(kvs...)
	default:
		return api.StringValue("INVALID")
	}
}
