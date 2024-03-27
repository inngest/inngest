package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/sdk/trace"
)

func newInngestSpanProcessor(p ...trace.SpanProcessor) trace.SpanProcessor {
	return &inngestSpanProcessor{processors: p}
}

type inngestSpanProcessor struct {
	processors []trace.SpanProcessor
}

func (sp *inngestSpanProcessor) OnStart(parent context.Context, s trace.ReadWriteSpan) {
	for _, sp := range sp.processors {
		sp.OnStart(parent, s)
	}
}

func (sp *inngestSpanProcessor) OnEnd(s trace.ReadOnlySpan) {
	for _, sp := range sp.processors {
		sp.OnEnd(s)
	}
}

func (sp *inngestSpanProcessor) Shutdown(ctx context.Context) error {
	for _, sp := range sp.processors {
		if err := sp.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (sp *inngestSpanProcessor) ForceFlush(ctx context.Context) error {
	for _, sp := range sp.processors {
		if err := sp.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}
