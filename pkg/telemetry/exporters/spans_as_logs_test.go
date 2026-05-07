package exporters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	otraceapi "go.opentelemetry.io/otel/trace"
)

type recordingLogger struct {
	embedded.Logger
	mu      sync.Mutex
	records []log.Record
}

func (r *recordingLogger) Emit(_ context.Context, rec log.Record) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, rec)
}

func (r *recordingLogger) Enabled(context.Context, log.EnabledParameters) bool { return true }

func (r *recordingLogger) snapshot() []log.Record {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]log.Record(nil), r.records...)
}

func newSpan(scope string, attrs ...attribute.KeyValue) sdktrace.ReadOnlySpan {
	tid, _ := otraceapi.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	sid, _ := otraceapi.SpanIDFromHex("1112131415161718")
	stub := tracetest.SpanStub{
		Name: "function.run",
		SpanContext: otraceapi.NewSpanContext(otraceapi.SpanContextConfig{
			TraceID: tid,
			SpanID:  sid,
		}),
		StartTime:            time.Unix(1700000000, 0),
		EndTime:              time.Unix(1700000001, 500_000_000),
		Attributes:           attrs,
		InstrumentationScope: instrumentation.Scope{Name: scope},
		SpanKind:             otraceapi.SpanKindInternal,
	}
	return stub.Snapshot()
}

func newEvent(name string, attrs ...attribute.KeyValue) sdktrace.Event {
	return sdktrace.Event{
		Name:       name,
		Time:       time.Unix(1700000001, 0),
		Attributes: attrs,
	}
}

func TestSpansAsLogs_LifecycleFilter(t *testing.T) {
	cases := []struct {
		name     string
		scope    string
		attrs    []attribute.KeyValue
		wantType string
		wantEmit bool
	}{
		{
			name:     "run started",
			scope:    consts.OtelScopeFunction,
			attrs:    []attribute.KeyValue{attribute.String(consts.OtelSysLifecycleID, lifecycleFunctionStarted)},
			wantType: InngestLogTypeRunStarted,
			wantEmit: true,
		},
		{
			name:     "function finished",
			scope:    consts.OtelScopeFunction,
			attrs:    []attribute.KeyValue{attribute.String(consts.OtelSysLifecycleID, lifecycleFunctionFinished)},
			wantType: InngestLogTypeFunctionEnded,
			wantEmit: true,
		},
		{
			name:     "step finished",
			scope:    consts.OtelScopeExecution,
			attrs:    []attribute.KeyValue{attribute.String(consts.OtelSysLifecycleID, lifecycleStepFinished), attribute.String(consts.OtelSysStepID, "step-id"), attribute.String(consts.OtelSysStepOpcode, enums.OpcodeStepRun.String())},
			wantType: InngestLogTypeStepEnded,
			wantEmit: true,
		},
		{
			name:     "durable step resumed",
			scope:    consts.OtelScopeStep,
			attrs:    []attribute.KeyValue{attribute.String(consts.OtelSysLifecycleID, lifecycleWaitEventResumed)},
			wantType: InngestLogTypeStepEnded,
			wantEmit: true,
		},
		{
			name:     "trigger dropped",
			scope:    consts.OtelScopeTrigger,
			attrs:    []attribute.KeyValue{attribute.String(consts.OtelSysLifecycleID, lifecycleFunctionStarted)},
			wantEmit: false,
		},
		{
			name:     "step started dropped",
			scope:    consts.OtelScopeExecution,
			attrs:    []attribute.KeyValue{attribute.String(consts.OtelSysLifecycleID, "OnStepStarted"), attribute.String(consts.OtelSysStepOpcode, enums.OpcodeStepPlanned.String())},
			wantEmit: false,
		},
		{
			name:     "planned step dropped",
			scope:    consts.OtelScopeExecution,
			attrs:    []attribute.KeyValue{attribute.String(consts.OtelSysLifecycleID, lifecycleStepFinished), attribute.Bool(consts.OtelSysStepPlan, true), attribute.String(consts.OtelSysStepOpcode, enums.OpcodeStepPlanned.String())},
			wantEmit: false,
		},
		{
			name:     "step without id dropped",
			scope:    consts.OtelScopeExecution,
			attrs:    []attribute.KeyValue{attribute.String(consts.OtelSysLifecycleID, lifecycleStepFinished), attribute.String(consts.OtelSysStepOpcode, enums.OpcodeStepRun.String())},
			wantEmit: false,
		},
		{
			name:     "userland dropped",
			scope:    consts.OtelScopeUserland,
			attrs:    []attribute.KeyValue{attribute.String(consts.OtelSysLifecycleID, lifecycleStepFinished)},
			wantEmit: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lg := &recordingLogger{}
			p := NewSpansAsLogsProcessor(lg)
			p.OnEnd(newSpan(tc.scope, tc.attrs...))
			got := len(lg.snapshot())
			if (got == 1) != tc.wantEmit {
				t.Fatalf("emit=%v, want=%v", got == 1, tc.wantEmit)
			}
			if !tc.wantEmit {
				return
			}
			body := unmarshalBody(t, lg.snapshot()[0])
			if got := body[InngestLogTypeKey].(string); got != tc.wantType {
				t.Fatalf("log type = %q, want %q", got, tc.wantType)
			}
		})
	}
}

func TestSpansAsLogs_SeverityFromFunctionStatus(t *testing.T) {
	cases := []struct {
		name   string
		status enums.RunStatus
		want   log.Severity
	}{
		{"completed", enums.RunStatusCompleted, log.SeverityInfo},
		{"failed", enums.RunStatusFailed, log.SeverityError},
		{"overflowed", enums.RunStatusOverflowed, log.SeverityError},
		{"cancelled", enums.RunStatusCancelled, log.SeverityWarn},
		{"skipped", enums.RunStatusSkipped, log.SeverityDebug},
		{"running_defaults_info", enums.RunStatusRunning, log.SeverityInfo},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lg := &recordingLogger{}
			p := NewSpansAsLogsProcessor(lg)
			span := newSpan(consts.OtelScopeFunction,
				attribute.String(consts.OtelSysLifecycleID, lifecycleFunctionFinished),
				attribute.Int64(consts.OtelSysFunctionStatusCode, tc.status.ToCode()),
			)
			p.OnEnd(span)
			recs := lg.snapshot()
			if len(recs) != 1 {
				t.Fatalf("expected 1 record, got %d", len(recs))
			}
			if recs[0].Severity() != tc.want {
				t.Fatalf("severity: got %v, want %v", recs[0].Severity(), tc.want)
			}
		})
	}
}

func TestSpansAsLogs_SeverityFallbackToOtelStatus(t *testing.T) {
	lg := &recordingLogger{}
	p := NewSpansAsLogsProcessor(lg)

	tid, _ := otraceapi.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	sid, _ := otraceapi.SpanIDFromHex("1112131415161718")
	stub := tracetest.SpanStub{
		Name:                 "function.run",
		SpanContext:          otraceapi.NewSpanContext(otraceapi.SpanContextConfig{TraceID: tid, SpanID: sid}),
		StartTime:            time.Unix(1700000000, 0),
		EndTime:              time.Unix(1700000001, 0),
		Attributes:           []attribute.KeyValue{attribute.String(consts.OtelSysLifecycleID, lifecycleWaitEventResumed)},
		InstrumentationScope: instrumentation.Scope{Name: consts.OtelScopeStep},
		Status:               sdktrace.Status{Code: codes.Error, Description: "boom"},
	}
	p.OnEnd(stub.Snapshot())
	recs := lg.snapshot()
	if recs[0].Severity() != log.SeverityError {
		t.Fatalf("got %v, want SeverityError", recs[0].Severity())
	}
}

func TestSpansAsLogs_PayloadTruncation(t *testing.T) {
	lg := &recordingLogger{}
	p := NewSpansAsLogsProcessor(lg, WithLogsPayloadCapBytes(10))

	long := strings.Repeat("x", 200)
	tid, _ := otraceapi.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	sid, _ := otraceapi.SpanIDFromHex("1112131415161718")
	stub := tracetest.SpanStub{
		Name:        "step.run",
		SpanContext: otraceapi.NewSpanContext(otraceapi.SpanContextConfig{TraceID: tid, SpanID: sid}),
		StartTime:   time.Unix(1700000000, 0),
		EndTime:     time.Unix(1700000001, 0),
		Attributes: []attribute.KeyValue{
			attribute.String(consts.OtelSysLifecycleID, lifecycleStepFinished),
			attribute.String(consts.OtelSysStepID, "step-id"),
			attribute.String(consts.OtelSysStepOpcode, enums.OpcodeStepRun.String()),
			attribute.String(consts.OtelSysStepDisplayName, long),
		},
		Events: []sdktrace.Event{
			newEvent(long, attribute.Bool(consts.OtelSysStepInput, true)),
		},
		InstrumentationScope: instrumentation.Scope{Name: consts.OtelScopeExecution},
	}
	span := stub.Snapshot()

	p.OnEnd(span)
	recs := lg.snapshot()
	body := unmarshalBody(t, recs[0])

	got, ok := body[consts.OtelSysStepInput].(string)
	if !ok {
		t.Fatalf("payload attr missing or wrong type")
	}
	if !strings.HasPrefix(got, "xxxxxxxxxx") || !strings.Contains(got, "truncated") {
		t.Fatalf("payload not truncated: %q", got)
	}
	if got := body[consts.OtelSysStepDisplayName].(string); got != long {
		t.Fatalf("non-payload attr was modified")
	}
}

func TestSpansAsLogs_RunStartedIncludesEventDataOnlyOnRunStart(t *testing.T) {
	eventJSON := `{"name":"test/event","data":{"email":"a@example.com","plan":"pro"}}`

	for _, tc := range []struct {
		logType        string
		lifecycle      string
		wantEventData  bool
		wantRecordType string
	}{
		{logType: "run_started", lifecycle: lifecycleFunctionStarted, wantEventData: true, wantRecordType: InngestLogTypeRunStarted},
		{logType: "function_ended", lifecycle: lifecycleFunctionFinished, wantEventData: false, wantRecordType: InngestLogTypeFunctionEnded},
	} {
		t.Run(tc.logType, func(t *testing.T) {
			lg := &recordingLogger{}
			p := NewSpansAsLogsProcessor(lg)

			tid, _ := otraceapi.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
			sid, _ := otraceapi.SpanIDFromHex("1112131415161718")
			stub := tracetest.SpanStub{
				Name:        "function.run",
				SpanContext: otraceapi.NewSpanContext(otraceapi.SpanContextConfig{TraceID: tid, SpanID: sid}),
				StartTime:   time.Unix(1700000000, 0),
				EndTime:     time.Unix(1700000001, 0),
				Attributes: []attribute.KeyValue{
					attribute.String(consts.OtelSysLifecycleID, tc.lifecycle),
					attribute.String(consts.OtelAttrSDKRunID, "run-id"),
				},
				Events: []sdktrace.Event{
					newEvent(eventJSON,
						attribute.Bool(consts.OtelSysEventData, true),
						attribute.String(consts.OtelSysEventInternalID, "event-id"),
					),
				},
				InstrumentationScope: instrumentation.Scope{Name: consts.OtelScopeFunction},
			}

			p.OnEnd(stub.Snapshot())
			body := unmarshalBody(t, lg.snapshot()[0])
			if got := body[InngestLogTypeKey].(string); got != tc.wantRecordType {
				t.Fatalf("log type = %q, want %q", got, tc.wantRecordType)
			}
			_, hasEventData := body[consts.OtelSysEventData]
			if hasEventData != tc.wantEventData {
				t.Fatalf("has event data = %v, want %v. body: %#v", hasEventData, tc.wantEventData, body)
			}
			if tc.wantEventData && !strings.Contains(fmt.Sprintf("%v", body[consts.OtelSysEventData]), "a@example.com") {
				t.Fatalf("event data missing expected payload: %#v", body[consts.OtelSysEventData])
			}
		})
	}
}

func TestSpansAsLogs_AttributesUseLatestValueForFiltering(t *testing.T) {
	lg := &recordingLogger{}
	p := NewSpansAsLogsProcessor(lg)

	span := newSpan(consts.OtelScopeExecution,
		attribute.String(consts.OtelSysLifecycleID, lifecycleStepFinished),
		attribute.String(consts.OtelSysStepID, "step-id"),
		attribute.String(consts.OtelSysStepOpcode, enums.OpcodeStepPlanned.String()),
		attribute.String(consts.OtelSysStepOpcode, enums.OpcodeStepRun.String()),
	)
	p.OnEnd(span)
	recs := lg.snapshot()
	if len(recs) != 1 {
		t.Fatalf("expected latest opcode to emit step, got %d records", len(recs))
	}
}

func TestSpansAsLogs_BodyShape(t *testing.T) {
	lg := &recordingLogger{}
	p := NewSpansAsLogsProcessor(lg)

	span := newSpan(consts.OtelScopeFunction,
		attribute.String(consts.OtelSysLifecycleID, lifecycleFunctionFinished),
		attribute.String(consts.OtelSysFunctionSlug, "my-app/my-fn"),
		attribute.Int64(consts.OtelSysFunctionStatusCode, enums.RunStatusCompleted.ToCode()),
	)
	p.OnEnd(span)
	body := unmarshalBody(t, lg.snapshot()[0])

	for _, key := range []string{
		"span.name",
		"span.kind",
		"span.scope",
		"span.duration_ms",
		"span.trace_id",
		"span.span_id",
		InngestLogTypeKey,
		consts.OtelSysFunctionSlug,
		consts.OtelSysFunctionStatusCode,
	} {
		if _, ok := body[key]; !ok {
			t.Fatalf("body missing key %q. body: %#v", key, body)
		}
	}
	if got := body["span.scope"].(string); got != consts.OtelScopeFunction {
		t.Fatalf("span.scope = %q, want %q", got, consts.OtelScopeFunction)
	}
	if got := body["span.duration_ms"].(float64); got != 1500 {
		t.Fatalf("span.duration_ms = %v, want 1500", got)
	}
}

func unmarshalBody(t *testing.T, rec log.Record) map[string]any {
	t.Helper()
	s := rec.Body().AsString()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("body not valid JSON: %v\n%s", err, s)
	}
	return m
}
