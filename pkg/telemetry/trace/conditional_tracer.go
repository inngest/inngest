package trace

import (
	"context"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type TraceEnabledFn func(ctx context.Context, accountID uuid.UUID, envID uuid.UUID) bool

type conditionalTracer struct {
	enabledFn TraceEnabledFn
	tracer    trace.Tracer
}

type ConditionalTracer interface {
	NewSpan(ctx context.Context, spanName string, accountID uuid.UUID, envID uuid.UUID) (context.Context, trace.Span)
}

func NewConditionalTracer(tracer trace.Tracer, fn TraceEnabledFn) ConditionalTracer {
	return &conditionalTracer{
		tracer:    tracer,
		enabledFn: fn,
	}
}

func (t *conditionalTracer) NewSpan(ctx context.Context, spanName string, accountID uuid.UUID, envID uuid.UUID) (context.Context, trace.Span) {
	if t.enabledFn(ctx, accountID, envID) {
		return t.tracer.Start(ctx, spanName)
	}

	return ctx, noop.Span{}
}

func AlwaysTrace(ctx context.Context, _, _ uuid.UUID) bool {
	return true
}
