package exporters

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/plasne/go-batcher/v2"
	"go.opentelemetry.io/otel/sdk/trace"
)

const (
	defaultBatcherBufferSize = 10_000
	defaultFlushInternal     = 200 * time.Millisecond
)

type BatchSpanProcessorOpt func(b *batchSpanProcessor)

func WithBatchProcessorBufferSize(size uint32) BatchSpanProcessorOpt {
	return func(b *batchSpanProcessor) {
		b.bufferSize = size
	}
}

func WithBatchProcessorInterval(flush time.Duration) BatchSpanProcessorOpt {
	return func(b *batchSpanProcessor) {
		b.flushInternal = flush
	}
}

type batchSpanProcessor struct {
	bufferSize    uint32
	flushInternal time.Duration

	exporter trace.SpanExporter
	batcher  batcher.Batcher
	watcher  batcher.Watcher
}

func NewBatchSpanProcessor(ctx context.Context, exporter trace.SpanExporter, opts ...BatchSpanProcessorOpt) (trace.SpanProcessor, error) {
	processor := &batchSpanProcessor{
		bufferSize:    defaultBatcherBufferSize,
		flushInternal: defaultFlushInternal,
	}

	for _, apply := range opts {
		apply(processor)
	}

	processor.batcher = batcher.NewBatcherWithBuffer(processor.bufferSize).WithFlushInterval(processor.flushInternal)
	if err := processor.batcher.Start(ctx); err != nil {
		return nil, fmt.Errorf("error starting batch processor: %w", err)
	}
	processor.watcher = batcher.NewWatcher(processor.onReady)

	return processor, nil
}

// No op
func (b *batchSpanProcessor) OnStart(ctx context.Context, s trace.ReadWriteSpan) {}

func (b *batchSpanProcessor) OnEnd(s trace.ReadOnlySpan) {
	ctx := context.Background()

	op := batcher.NewOperation(b.watcher, 1, s, true)
	if err := b.batcher.Enqueue(op); err != nil {
		logger.StdlibLogger(ctx).Error("error enqueueing span for batch", "error", err)
		// TODO: add metric here
	}
}

func (b *batchSpanProcessor) Shutdown(ctx context.Context) error {
	if err := b.ForceFlush(ctx); err != nil {
		// TODO: add metric
		return err
	}

	return b.exporter.Shutdown(ctx)
}

func (b *batchSpanProcessor) ForceFlush(ctx context.Context) error {
	b.batcher.Flush()
	return nil
}

func (b *batchSpanProcessor) onReady(batch []batcher.Operation) {
	ctx := context.Background()
	// size := len(batch)

	// TODO: add metric here

	spans := []trace.ReadOnlySpan{}
	for _, op := range batch {
		span, ok := op.Payload().(trace.ReadOnlySpan)
		if !ok {
			logger.StdlibLogger(ctx).Warn("payload is not a span", "payload", op.Payload())
			// TODO: add metric here
		}
		spans = append(spans, span)
	}

	if err := b.exporter.ExportSpans(ctx, spans); err != nil {
		logger.StdlibLogger(ctx).Error("error batch exporting spans", "error", err)
		// TODO: add metric here
	}
}
