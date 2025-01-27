package exporters

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/sdk/trace"
)

var (
	notImplementedErr = fmt.Errorf("not implemented")
)

type kafkaSpanExporter struct {
}

type kafkaSpansExporterOpts struct{}

func NewKafkaSpanExporter(ctx context.Context, opts ...kafkaSpansExporterOpts) (trace.SpanExporter, error) {
	exp := &kafkaSpanExporter{}

	return exp, nil
}

func (e *kafkaSpanExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	return notImplementedErr
}

func (e *kafkaSpanExporter) Shutdown(ctx context.Context) error {
	return notImplementedErr
}
