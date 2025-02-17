package connect

import (
	"context"
	"github.com/google/uuid"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type TraceEnabledFn func(ctx context.Context, accountID uuid.UUID, envID uuid.UUID) bool

type conditionalTracer struct {
	enabledFn TraceEnabledFn
}

type ConditionalTracer interface {
	NewSpan(ctx context.Context, spanName string, accountID uuid.UUID, envID uuid.UUID) (context.Context, trace.Span)
}

func NewConditionalTracer(fn TraceEnabledFn) ConditionalTracer {
	return &conditionalTracer{
		enabledFn: fn,
	}
}

func (t *conditionalTracer) NewSpan(ctx context.Context, spanName string, accountID uuid.UUID, envID uuid.UUID) (context.Context, trace.Span) {
	if t.enabledFn(ctx, accountID, envID) {
		return itrace.ConnectTracer().Start(ctx, spanName)
	}

	return ctx, noop.Span{}
}
