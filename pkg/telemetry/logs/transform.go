package logs

import (
	cpb "go.opentelemetry.io/proto/otlp/common/v1"
	lpb "go.opentelemetry.io/proto/otlp/logs/v1"
	"time"

	api "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
)

//
// NOTE: This is sourced from the OTLP log exporter:
// - https://github.com/open-telemetry/opentelemetry-go/blob/exporters/otlp/otlplog/otlploggrpc/v0.13.0/exporters/otlp/otlplog/otlploggrpc/internal/transform/log.go
//
// Unfortunately, the code linked above is part of an internal module and thus cannot trivially be imported.
//

// LogRecord returns an OTLP LogRecord generated from record.
func LogRecord(record log.Record) *lpb.LogRecord {
	r := &lpb.LogRecord{
		TimeUnixNano:         timeUnixNano(record.Timestamp()),
		ObservedTimeUnixNano: timeUnixNano(record.ObservedTimestamp()),
		EventName:            record.EventName(),
		SeverityNumber:       SeverityNumber(record.Severity()),
		SeverityText:         record.SeverityText(),
		Body:                 LogAttrValue(record.Body()),
		Attributes:           make([]*cpb.KeyValue, 0, record.AttributesLen()),
		Flags:                uint32(record.TraceFlags()),
		// TODO: DroppedAttributesCount: /* ... */,
	}
	record.WalkAttributes(func(kv api.KeyValue) bool {
		r.Attributes = append(r.Attributes, LogAttr(kv))
		return true
	})
	if tID := record.TraceID(); tID.IsValid() {
		r.TraceId = tID[:]
	}
	if sID := record.SpanID(); sID.IsValid() {
		r.SpanId = sID[:]
	}
	return r
}

// timeUnixNano returns t as a Unix time, the number of nanoseconds elapsed
// since January 1, 1970 UTC as uint64. The result is undefined if the Unix
// time in nanoseconds cannot be represented by an int64 (a date before the
// year 1678 or after 2262). timeUnixNano on the zero Time returns 0. The
// result does not depend on the location associated with t.
func timeUnixNano(t time.Time) uint64 {
	nano := t.UnixNano()
	if nano < 0 {
		return 0
	}
	return uint64(nano) // nolint:gosec // Overflow checked.
}

// SeverityNumber transforms a [log.Severity] into an OTLP SeverityNumber.
func SeverityNumber(s api.Severity) lpb.SeverityNumber {
	switch s {
	case api.SeverityTrace:
		return lpb.SeverityNumber_SEVERITY_NUMBER_TRACE
	case api.SeverityTrace2:
		return lpb.SeverityNumber_SEVERITY_NUMBER_TRACE2
	case api.SeverityTrace3:
		return lpb.SeverityNumber_SEVERITY_NUMBER_TRACE3
	case api.SeverityTrace4:
		return lpb.SeverityNumber_SEVERITY_NUMBER_TRACE4
	case api.SeverityDebug:
		return lpb.SeverityNumber_SEVERITY_NUMBER_DEBUG
	case api.SeverityDebug2:
		return lpb.SeverityNumber_SEVERITY_NUMBER_DEBUG2
	case api.SeverityDebug3:
		return lpb.SeverityNumber_SEVERITY_NUMBER_DEBUG3
	case api.SeverityDebug4:
		return lpb.SeverityNumber_SEVERITY_NUMBER_DEBUG4
	case api.SeverityInfo:
		return lpb.SeverityNumber_SEVERITY_NUMBER_INFO
	case api.SeverityInfo2:
		return lpb.SeverityNumber_SEVERITY_NUMBER_INFO2
	case api.SeverityInfo3:
		return lpb.SeverityNumber_SEVERITY_NUMBER_INFO3
	case api.SeverityInfo4:
		return lpb.SeverityNumber_SEVERITY_NUMBER_INFO4
	case api.SeverityWarn:
		return lpb.SeverityNumber_SEVERITY_NUMBER_WARN
	case api.SeverityWarn2:
		return lpb.SeverityNumber_SEVERITY_NUMBER_WARN2
	case api.SeverityWarn3:
		return lpb.SeverityNumber_SEVERITY_NUMBER_WARN3
	case api.SeverityWarn4:
		return lpb.SeverityNumber_SEVERITY_NUMBER_WARN4
	case api.SeverityError:
		return lpb.SeverityNumber_SEVERITY_NUMBER_ERROR
	case api.SeverityError2:
		return lpb.SeverityNumber_SEVERITY_NUMBER_ERROR2
	case api.SeverityError3:
		return lpb.SeverityNumber_SEVERITY_NUMBER_ERROR3
	case api.SeverityError4:
		return lpb.SeverityNumber_SEVERITY_NUMBER_ERROR4
	case api.SeverityFatal:
		return lpb.SeverityNumber_SEVERITY_NUMBER_FATAL
	case api.SeverityFatal2:
		return lpb.SeverityNumber_SEVERITY_NUMBER_FATAL2
	case api.SeverityFatal3:
		return lpb.SeverityNumber_SEVERITY_NUMBER_FATAL3
	case api.SeverityFatal4:
		return lpb.SeverityNumber_SEVERITY_NUMBER_FATAL4
	}
	return lpb.SeverityNumber_SEVERITY_NUMBER_UNSPECIFIED
}

// LogAttrValues transforms a slice of [api.Value] into an OTLP []AnyValue.
func LogAttrValues(vals []api.Value) []*cpb.AnyValue {
	if len(vals) == 0 {
		return nil
	}

	out := make([]*cpb.AnyValue, 0, len(vals))
	for _, v := range vals {
		out = append(out, LogAttrValue(v))
	}
	return out
}

// LogAttrValue transforms an [api.Value] into an OTLP AnyValue.
func LogAttrValue(v api.Value) *cpb.AnyValue {
	av := new(cpb.AnyValue)
	switch v.Kind() {
	case api.KindBool:
		av.Value = &cpb.AnyValue_BoolValue{
			BoolValue: v.AsBool(),
		}
	case api.KindInt64:
		av.Value = &cpb.AnyValue_IntValue{
			IntValue: v.AsInt64(),
		}
	case api.KindFloat64:
		av.Value = &cpb.AnyValue_DoubleValue{
			DoubleValue: v.AsFloat64(),
		}
	case api.KindString:
		av.Value = &cpb.AnyValue_StringValue{
			StringValue: v.AsString(),
		}
	case api.KindBytes:
		av.Value = &cpb.AnyValue_BytesValue{
			BytesValue: v.AsBytes(),
		}
	case api.KindSlice:
		av.Value = &cpb.AnyValue_ArrayValue{
			ArrayValue: &cpb.ArrayValue{
				Values: LogAttrValues(v.AsSlice()),
			},
		}
	case api.KindMap:
		av.Value = &cpb.AnyValue_KvlistValue{
			KvlistValue: &cpb.KeyValueList{
				Values: LogAttrs(v.AsMap()),
			},
		}
	default:
		av.Value = &cpb.AnyValue_StringValue{
			StringValue: "INVALID",
		}
	}
	return av
}

// LogAttrs transforms a slice of [api.KeyValue] into OTLP key-values.
func LogAttrs(attrs []api.KeyValue) []*cpb.KeyValue {
	if len(attrs) == 0 {
		return nil
	}

	out := make([]*cpb.KeyValue, 0, len(attrs))
	for _, kv := range attrs {
		out = append(out, LogAttr(kv))
	}
	return out
}

// LogAttr transforms an [api.KeyValue] into an OTLP key-value.
func LogAttr(attr api.KeyValue) *cpb.KeyValue {
	return &cpb.KeyValue{
		Key:   attr.Key,
		Value: LogAttrValue(attr.Value),
	}
}
