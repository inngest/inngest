package exporters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const defaultPayloadCapBytes = 4096

const (
	InngestLogTypeKey = "inngest.log.type"

	InngestLogTypeRunStarted    = "run.started"
	InngestLogTypeStepEnded     = "step.ended"
	InngestLogTypeFunctionEnded = "function.ended"

	lifecycleFunctionStarted   = "OnFunctionStarted"
	lifecycleFunctionFinished  = "OnFunctionFinished"
	lifecycleFunctionCancelled = "OnFunctionCancelled"
	lifecycleFunctionSkipped   = "OnFunctionSkipped"
	lifecycleStepFinished      = "OnStepFinished"
	lifecycleStepGatewayDone   = "OnStepGatewayRequestFinished"
	lifecycleSleep             = "OnSleep"
	lifecycleInvokeResumed     = "OnInvokeFunctionResumed"
	lifecycleWaitEventResumed  = "OnWaitForEventResumed"
	lifecycleWaitSignalResumed = "OnWaitForSignalResumed"
)

var defaultPayloadAttrs = map[string]struct{}{
	consts.OtelSysStepInput:      {},
	consts.OtelSysStepOutput:     {},
	consts.OtelSysFunctionOutput: {},
	consts.OtelSysStepAIRequest:  {},
	consts.OtelSysStepAIResponse: {},
}

type SpansAsLogsOpt func(*spansAsLogsProcessor)

func WithLogsPayloadCapBytes(n int) SpansAsLogsOpt {
	return func(p *spansAsLogsProcessor) {
		if n > 0 {
			p.payloadCap = n
		}
	}
}

type spansAsLogsProcessor struct {
	logger     otellog.Logger
	payloadCap int
	payloads   map[string]struct{}
}

// NewSpansAsLogsProcessor returns a SpanProcessor that emits one OTLP LogRecord
// for the finalized run and step lifecycle spans that power run search. Other
// spans are silently dropped from the logs pipeline (the underlying trace
// pipeline is unaffected).
func NewSpansAsLogsProcessor(lg otellog.Logger, opts ...SpansAsLogsOpt) sdktrace.SpanProcessor {
	p := &spansAsLogsProcessor{
		logger:     lg,
		payloadCap: defaultPayloadCapBytes,
		payloads:   defaultPayloadAttrs,
	}
	for _, apply := range opts {
		apply(p)
	}
	return p
}

func (p *spansAsLogsProcessor) OnStart(_ context.Context, _ sdktrace.ReadWriteSpan) {}

func (p *spansAsLogsProcessor) OnEnd(span sdktrace.ReadOnlySpan) {
	logType, ok := p.logType(span)
	if !ok {
		return
	}
	rec := p.spanToRecord(span, logType)
	p.logger.Emit(context.Background(), rec)
}

func (p *spansAsLogsProcessor) Shutdown(_ context.Context) error   { return nil }
func (p *spansAsLogsProcessor) ForceFlush(_ context.Context) error { return nil }

func (p *spansAsLogsProcessor) spanToRecord(span sdktrace.ReadOnlySpan, logType string) otellog.Record {
	var rec otellog.Record

	end := span.EndTime()
	rec.SetTimestamp(end)
	rec.SetObservedTimestamp(time.Now())

	body := make(map[string]any, len(span.Attributes())+8)
	body[InngestLogTypeKey] = logType
	body["span.name"] = span.Name()
	body["span.kind"] = span.SpanKind().String()
	body["span.scope"] = span.InstrumentationScope().Name
	body["span.duration_ms"] = end.Sub(span.StartTime()).Milliseconds()
	attrs := []otellog.KeyValue{
		{Key: InngestLogTypeKey, Value: otellog.StringValue(logType)},
		{Key: "span.name", Value: otellog.StringValue(span.Name())},
		{Key: "span.kind", Value: otellog.StringValue(span.SpanKind().String())},
		{Key: "span.scope", Value: otellog.StringValue(span.InstrumentationScope().Name)},
		{Key: "span.duration_ms", Value: otellog.Int64Value(end.Sub(span.StartTime()).Milliseconds())},
	}
	if sc := span.SpanContext(); sc.IsValid() {
		body["span.trace_id"] = sc.TraceID().String()
		body["span.span_id"] = sc.SpanID().String()
		attrs = append(attrs,
			otellog.KeyValue{Key: "span.trace_id", Value: otellog.StringValue(sc.TraceID().String())},
			otellog.KeyValue{Key: "span.span_id", Value: otellog.StringValue(sc.SpanID().String())},
		)
	}
	if parent := span.Parent(); parent.IsValid() {
		body["span.parent_span_id"] = parent.SpanID().String()
		attrs = append(attrs, otellog.KeyValue{Key: "span.parent_span_id", Value: otellog.StringValue(parent.SpanID().String())})
	}

	for _, kv := range span.Attributes() {
		key := string(kv.Key)
		raw := attrAny(kv.Value)
		body[key] = raw
		attrs = append(attrs, otellog.KeyValue{Key: key, Value: toLogValue(raw)})
	}

	p.addSpanEvents(body, &attrs, span, logType)

	rec.AddAttributes(attrs...)

	rec.SetSeverity(deriveSeverity(span))
	rec.SetSeverityText(rec.Severity().String())

	if buf, err := json.Marshal(body); err == nil {
		rec.SetBody(otellog.StringValue(string(buf)))
	} else {
		rec.SetBody(otellog.StringValue(span.Name()))
	}
	return rec
}

func (p *spansAsLogsProcessor) logType(span sdktrace.ReadOnlySpan) (string, bool) {
	attrs := latestAttrs(span.Attributes())
	lifecycle := attrString(attrs, consts.OtelSysLifecycleID)

	switch span.InstrumentationScope().Name {
	case consts.OtelScopeFunction:
		switch lifecycle {
		case lifecycleFunctionStarted:
			return InngestLogTypeRunStarted, true
		case lifecycleFunctionFinished, lifecycleFunctionCancelled, lifecycleFunctionSkipped:
			return InngestLogTypeFunctionEnded, true
		}

	case consts.OtelScopeExecution:
		switch lifecycle {
		case lifecycleStepGatewayDone:
			return InngestLogTypeStepEnded, true
		case lifecycleStepFinished:
			if attrBool(attrs, consts.OtelSysStepPlan) {
				return "", false
			}
			if attrString(attrs, consts.OtelSysStepOpcode) == enums.OpcodeStepPlanned.String() {
				return "", false
			}
			if attrString(attrs, consts.OtelSysStepID) == "" {
				return "", false
			}
			return InngestLogTypeStepEnded, true
		}

	case consts.OtelScopeStep:
		switch lifecycle {
		case lifecycleSleep, lifecycleInvokeResumed, lifecycleWaitEventResumed, lifecycleWaitSignalResumed:
			return InngestLogTypeStepEnded, true
		}
	}

	return "", false
}

func (p *spansAsLogsProcessor) addSpanEvents(body map[string]any, attrs *[]otellog.KeyValue, span sdktrace.ReadOnlySpan, logType string) {
	var runEvents []any

	for _, evt := range span.Events() {
		evtAttrs := latestAttrs(evt.Attributes)
		if attrBool(evtAttrs, consts.OtelSysEventData) {
			if logType == InngestLogTypeRunStarted {
				runEvents = append(runEvents, jsonPayload(evt.Name))
			}
			continue
		}

		for key := range p.payloads {
			if !attrBool(evtAttrs, key) {
				continue
			}

			raw := truncateForPayload(evt.Name, p.payloadCap)
			body[key] = raw
			*attrs = append(*attrs, otellog.KeyValue{Key: key, Value: toLogValue(raw)})
			break
		}
	}

	if len(runEvents) > 0 {
		body[consts.OtelSysEventData] = runEvents
	}
}

func latestAttrs(attrs []attribute.KeyValue) map[string]attribute.Value {
	res := make(map[string]attribute.Value, len(attrs))
	for _, kv := range attrs {
		if kv.Valid() {
			res[string(kv.Key)] = kv.Value
		}
	}
	return res
}

func attrString(attrs map[string]attribute.Value, key string) string {
	v, ok := attrs[key]
	if !ok {
		return ""
	}
	return v.AsString()
}

func attrBool(attrs map[string]attribute.Value, key string) bool {
	v, ok := attrs[key]
	return ok && v.AsBool()
}

func jsonPayload(s string) any {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	return v
}

// deriveSeverity maps Inngest run/step status onto OTLP severity. The function
// status code is the primary signal (set on root function spans); for non-root
// spans, fall back to the step status string and finally the OTel span status.
func deriveSeverity(span sdktrace.ReadOnlySpan) otellog.Severity {
	for _, kv := range span.Attributes() {
		if string(kv.Key) == consts.OtelSysFunctionStatusCode && kv.Value.Type() == attribute.INT64 {
			switch enums.RunCodeToStatus(kv.Value.AsInt64()) {
			case enums.RunStatusCompleted:
				return otellog.SeverityInfo
			case enums.RunStatusFailed, enums.RunStatusOverflowed:
				return otellog.SeverityError
			case enums.RunStatusCancelled:
				return otellog.SeverityWarn
			case enums.RunStatusSkipped:
				return otellog.SeverityDebug
			}
		}
	}
	for _, kv := range span.Attributes() {
		if string(kv.Key) == consts.OtelSysStepStatus {
			switch kv.Value.AsString() {
			case "Completed":
				return otellog.SeverityInfo
			case "Failed", "Errored":
				return otellog.SeverityError
			case "Cancelled", "TimedOut":
				return otellog.SeverityWarn
			case "Skipped":
				return otellog.SeverityDebug
			}
		}
	}
	if span.Status().Code == codes.Error {
		return otellog.SeverityError
	}
	return otellog.SeverityInfo
}

// attrAny converts an attribute.Value to a JSON-friendly Go value.
func attrAny(v attribute.Value) any {
	switch v.Type() {
	case attribute.BOOL:
		return v.AsBool()
	case attribute.INT64:
		return v.AsInt64()
	case attribute.FLOAT64:
		return v.AsFloat64()
	case attribute.STRING:
		return v.AsString()
	case attribute.BOOLSLICE:
		return v.AsBoolSlice()
	case attribute.INT64SLICE:
		return v.AsInt64Slice()
	case attribute.FLOAT64SLICE:
		return v.AsFloat64Slice()
	case attribute.STRINGSLICE:
		return v.AsStringSlice()
	default:
		return v.Emit()
	}
}

// truncateForPayload caps a string-shaped attribute at n bytes, leaving a
// suffix that records the original size so consumers can detect truncation.
// Non-string payloads are passed through (they're already bounded).
func truncateForPayload(v any, n int) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	if len(s) <= n {
		return s
	}
	return fmt.Sprintf("%s…(truncated, original %d bytes)", s[:n], len(s))
}

// toLogValue mirrors the Go-typed value into an OTLP LogRecord attribute value.
func toLogValue(v any) otellog.Value {
	switch x := v.(type) {
	case string:
		return otellog.StringValue(x)
	case bool:
		return otellog.BoolValue(x)
	case int64:
		return otellog.Int64Value(x)
	case int:
		return otellog.IntValue(x)
	case float64:
		return otellog.Float64Value(x)
	case []byte:
		return otellog.BytesValue(x)
	default:
		return otellog.StringValue(fmt.Sprintf("%v", x))
	}
}
