package tracing

import (
	"context"

	"github.com/inngest/inngest/pkg/tracing/meta"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type noopTracerProvider struct{}

// CreateDroppableSpan implements TracerProvider.
func (n *noopTracerProvider) CreateDroppableSpan(name string, opts *CreateSpanOptions) (*DroppableSpan, error) {
	_, span := sdktrace.NewTracerProvider().Tracer("inngest").Start(context.Background(), "noop")

	return &DroppableSpan{
		span: span,
		Ref:  &meta.SpanReference{},
	}, nil
}

// CreateSpan implements TracerProvider.
func (n *noopTracerProvider) CreateSpan(name string, opts *CreateSpanOptions) (*meta.SpanReference, error) {
	span, err := n.CreateDroppableSpan(name, opts)
	if err != nil {
		return nil, err
	}

	_ = span.Send()

	return span.Ref, nil
}

// UpdateSpan implements TracerProvider.
func (n *noopTracerProvider) UpdateSpan(opts *UpdateSpanOptions) error {
	return nil
}

func NewNoopTracerProvider() TracerProvider {
	return &noopTracerProvider{}
}
