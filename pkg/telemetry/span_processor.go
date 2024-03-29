package telemetry

import (
	"context"

	"github.com/inngest/inngest/pkg/consts"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

// NewSpanProcessor creates a new span processor that wraps the passed in
// exporter, used to share common logic between different trace providers.
//
// Internally it creates a single span processor that chains together multiple
// span processors, each of which can modify the span or decide to not forward
// it to the next processor.
func NewSpanProcessor(exp trace.SpanExporter) trace.SpanProcessor {
	// Specify all custom span processors here. The order in which they are
	// specified is the order in which they will be run.
	//
	// The last span processor in the chain should be the exporter.
	processors := []newSpanProcessorFunc{
		newIgnoredSpanProcessor,
		func(trace.SpanProcessor) trace.SpanProcessor {
			return trace.NewBatchSpanProcessor(exp)
		},
	}

	// Create a chain of span processors, starting with the exporter and
	// wrapping in with a series of custom processors.
	var next trace.SpanProcessor
	for i := len(processors) - 1; i >= 0; i-- {
		next = processors[i](next)
	}
	return next
}

// newSpanProcessorFunc is a function that creates a span processor which wraps
// the `next` span processor, used to create a chain of span processors.
type newSpanProcessorFunc func(next trace.SpanProcessor) trace.SpanProcessor

// chainableSpanProcessor is a span processor that can be chained with other
// span processors. It forwards calls to the next processor in the chain.
//
// Custom span processors should embed this type to get sensible defaults for
// each hook and should call the next processor in the chain if they declare any
// methods themselves.
//
// We use this pattern of chaining to ensure that each span processor can decide
// when to interrupt the flow of processing by not calling the `next“ processor,
// e.g. if a span is ignored. It also ensures that each processor continues to
// satisfy the `SpanProcessor` interface.
type chainableSpanProcessor struct {
	next trace.SpanProcessor
}

func newChainableSpanProcessor(next trace.SpanProcessor) *chainableSpanProcessor {
	return &chainableSpanProcessor{next: next}
}

func (p *chainableSpanProcessor) OnStart(parent context.Context, s trace.ReadWriteSpan) {
	if p.next != nil {
		p.next.OnStart(parent, s)
	}
}

func (p *chainableSpanProcessor) OnEnd(s trace.ReadOnlySpan) {
	if p.next != nil {
		p.next.OnEnd(s)
	}
}

func (p *chainableSpanProcessor) Shutdown(ctx context.Context) error {
	if p.next != nil {
		return p.next.Shutdown(ctx)
	}
	return nil
}

func (p *chainableSpanProcessor) ForceFlush(ctx context.Context) error {
	if p.next != nil {
		return p.next.ForceFlush(ctx)
	}
	return nil
}

// IgnoredSpanProcessor processes spans and does not send them to the next
// processor if they have the "ignored" attribute set to true.
//
// This can be useful for if a span's creation was required for propagation
// purposes, but turned out not to be used, e.g. if idempotency filtered out a
// queued item.
type ignoredSpanProcessor struct {
	chainableSpanProcessor
}

func newIgnoredSpanProcessor(next trace.SpanProcessor) trace.SpanProcessor {
	return &ignoredSpanProcessor{chainableSpanProcessor: *newChainableSpanProcessor(next)}
}

func (p *ignoredSpanProcessor) OnEnd(s trace.ReadOnlySpan) {
	// Check for the "ignored" attribute
	for _, kv := range s.Attributes() {
		if kv.Key == consts.OtelSysIgnored && kv.Value.Type() == attribute.BOOL && kv.Value.AsBool() {
			return // If "ignored: true", don't forward the span to the next processor
		}
	}

	if p.next != nil {
		p.next.OnEnd(s)
	}
}
