package exporters

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"go.opentelemetry.io/otel/sdk/trace"
)

const (
	defaultBatchMaxSize = 10_000
	defaultBatchTimeout = 200 * time.Millisecond
)

type BatchSpanProcessorOpt func(b *batchSpanProcessor)

func WithBatchProcessorBufferSize(size int) BatchSpanProcessorOpt {
	return func(b *batchSpanProcessor) {
		if size > 0 {
			b.maxSize = size
		}
	}
}

func WithBatchProcessorInterval(timeout time.Duration) BatchSpanProcessorOpt {
	return func(b *batchSpanProcessor) {
		if timeout > 0 {
			b.timeout = timeout
		}
	}
}

type batchSpanProcessor struct {
	mt       sync.RWMutex
	exporter trace.SpanExporter
	maxSize  int
	timeout  time.Duration
	in       chan *trace.ReadOnlySpan
	buffer   map[string][]trace.ReadOnlySpan
	pointer  uuid.UUID
}

func NewBatchSpanProcessor(ctx context.Context, exporter trace.SpanExporter, opts ...BatchSpanProcessorOpt) trace.SpanProcessor {
	p := &batchSpanProcessor{
		mt:       sync.RWMutex{},
		exporter: exporter,
		maxSize:  defaultBatchMaxSize,
		timeout:  defaultBatchTimeout,
		buffer:   map[string][]trace.ReadOnlySpan{},
		pointer:  uuid.New(),
	}

	for _, apply := range opts {
		apply(p)
	}
	p.in = make(chan *trace.ReadOnlySpan, p.maxSize)

	// start process loop
	go p.run(ctx)

	return p
}

// No op
func (b *batchSpanProcessor) OnStart(ctx context.Context, s trace.ReadWriteSpan) {}

func (b *batchSpanProcessor) OnEnd(s trace.ReadOnlySpan) {
	// pass span into the channel
	b.in <- &s
	metrics.IncrBatchProcessorEnqueuedCounter(context.TODO(), metrics.CounterOpt{PkgName: pkgName})
}

func (b *batchSpanProcessor) Shutdown(ctx context.Context) error {
	if err := b.flush(ctx); err != nil {
		logger.StdlibLogger(ctx).Error("error flushing spans on shutdown", "error", err)
	}
	return b.exporter.Shutdown(ctx)
}

func (b *batchSpanProcessor) ForceFlush(ctx context.Context) error {
	return b.flush(ctx)
}

func (b *batchSpanProcessor) run(ctx context.Context) {
	for {
		select {
		case span := <-b.in:
			b.append(ctx, span)

		case <-ctx.Done():
			if err := b.flush(ctx); err != nil {
				logger.StdlibLogger(ctx).Error("error flushing spans on completion", "error", err)
			}
			return
		}
	}
}

// append add the span into the buffer the pointer is currently pointing to
func (b *batchSpanProcessor) append(ctx context.Context, span *trace.ReadOnlySpan) {
	b.mt.Lock()
	defer b.mt.Unlock()

	p := b.pointer
	buf, ok := b.buffer[p.String()]
	if !ok {
		buf = []trace.ReadOnlySpan{}
	}

	buf = append(buf, *span)
	b.buffer[p.String()] = buf

	switch len(buf) {
	case 1:
		// attempt to send the spans on timeout if this is a new batch
		go b.sendLater(ctx, p.String())

	case b.maxSize:
		// reset buffer
		newPointer := uuid.New()
		b.pointer = newPointer

		// start execution right away
		go func() {
			if err := b.send(ctx, p.String()); err != nil {
				logger.StdlibLogger(ctx).Error("error sending spans on full batch", "error", err)
			}
		}()
	}
}

// sendLater defers the sending after the timeout
func (b *batchSpanProcessor) sendLater(ctx context.Context, id string) {
	<-time.After(b.timeout)

	// update the pointer to something else so it doesn't attempt to update the same buffer
	b.mt.Lock()
	// only update if the pointer value is still the same
	if b.pointer.String() == id {
		b.pointer = uuid.New()
	}
	b.mt.Unlock()

	if err := b.send(ctx, id); err != nil {
		logger.StdlibLogger(ctx).Error("error sending spans after delay", "error", err)
	}
}

// send attempts to process the buffer of spans identified by id
func (b *batchSpanProcessor) send(ctx context.Context, id string) error {
	b.mt.Lock()
	spans, ok := b.buffer[id]
	b.mt.Unlock()

	if !ok {
		// likely already processed
		return nil
	}

	count := len(spans)
	metrics.IncrBatchProcessorAttemptCounter(ctx, int64(count), metrics.CounterOpt{PkgName: pkgName})

	err := b.exporter.ExportSpans(ctx, spans)
	if err != nil {
		logger.StdlibLogger(ctx).Error("error batch exporting spans", "error", err, "id", id)
	}

	// remove the buffer from the map so it doesn't build up memory
	b.mt.Lock()
	delete(b.buffer, id)
	b.mt.Unlock()

	return err
}

// flush attempts to send out all spans in the buffer
func (b *batchSpanProcessor) flush(ctx context.Context) error {
	var errs error

	for id := range b.buffer {
		if err := b.send(ctx, id); err != nil {
			errs = multierror.Append(err, errs)
		}
	}

	return errs
}
