package exporters

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/sdk/trace"
)

type batchSpanProcessor struct {
}

func NewBatchSpanProcessor(exporter trace.SpanExporter) trace.SpanProcessor {
	return &batchSpanProcessor{}
}

// No op
func (b *batchSpanProcessor) OnStart(ctx context.Context, s trace.ReadWriteSpan) {}

func (b *batchSpanProcessor) OnEnd(s trace.ReadOnlySpan) {
	// TODO
}

func (b *batchSpanProcessor) Shutdown(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (b *batchSpanProcessor) ForceFlush(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}
