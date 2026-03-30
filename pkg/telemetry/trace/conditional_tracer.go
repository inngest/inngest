package trace

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type TraceEnabledFn func(ctx context.Context, accountID uuid.UUID, envID uuid.UUID, fnID uuid.UUID) bool

type conditionalTracer struct {
	enabledFn TraceEnabledFn
	tracer    trace.Tracer
}

type ConditionalTracer interface {
	NewSpan(ctx context.Context, spanName string, accountID uuid.UUID, envID uuid.UUID, fnID uuid.UUID) (context.Context, trace.Span)
}

func NewConditionalTracer(tracer trace.Tracer, fn TraceEnabledFn) ConditionalTracer {
	return &conditionalTracer{
		tracer:    tracer,
		enabledFn: fn,
	}
}

func NoopConditionalTracer() ConditionalTracer {
	return NewConditionalTracer(noop.Tracer{}, func(ctx context.Context, accountID, envID, fnID uuid.UUID) bool {
		return false
	})
}

func (t *conditionalTracer) NewSpan(ctx context.Context, spanName string, accountID uuid.UUID, envID uuid.UUID, fnID uuid.UUID) (context.Context, trace.Span) {
	if t.enabledFn(ctx, accountID, envID, fnID) {
		ctx, span := t.tracer.Start(ctx, spanName)
		span.SetAttributes(attribute.String("account_id", accountID.String()))
		span.SetAttributes(attribute.String("env_id", envID.String()))
		span.SetAttributes(attribute.String("fn_id", fnID.String()))
		return ctx, span
	}

	return ctx, noop.Span{}
}

func AlwaysTrace(ctx context.Context, _, _, _ uuid.UUID) bool {
	return true
}
