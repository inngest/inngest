package trace

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type recordingSpanProcessor struct {
	ended int
}

func (r *recordingSpanProcessor) OnStart(context.Context, sdktrace.ReadWriteSpan) {}

func (r *recordingSpanProcessor) OnEnd(sdktrace.ReadOnlySpan) {
	r.ended++
}

func (r *recordingSpanProcessor) Shutdown(context.Context) error {
	return nil
}

func (r *recordingSpanProcessor) ForceFlush(context.Context) error {
	return nil
}

func TestTracerExportFansOutToTraceAndLogsProcessors(t *testing.T) {
	traceProcessor := &recordingSpanProcessor{}
	logsProcessor := &recordingSpanProcessor{}
	tr := &tracer{
		processor:     traceProcessor,
		logsProcessor: logsProcessor,
	}

	tid, _ := oteltrace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	sid, _ := oteltrace.SpanIDFromHex("1112131415161718")
	span := tracetest.SpanStub{
		Name:        "function.run",
		SpanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{TraceID: tid, SpanID: sid}),
		StartTime:   time.Unix(1700000000, 0),
		EndTime:     time.Unix(1700000001, 0),
		Attributes: []attribute.KeyValue{
			attribute.String("sys.lifecycle.id", "OnFunctionFinished"),
		},
		InstrumentationScope: instrumentation.Scope{Name: "function.app.env.inngest"},
	}.Snapshot()

	if err := tr.Export(span); err != nil {
		t.Fatalf("Export returned error: %v", err)
	}
	if traceProcessor.ended != 1 {
		t.Fatalf("trace processor ended %d spans, want 1", traceProcessor.ended)
	}
	if logsProcessor.ended != 1 {
		t.Fatalf("logs processor ended %d spans, want 1", logsProcessor.ended)
	}
}
