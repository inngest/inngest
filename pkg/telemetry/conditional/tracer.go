package conditional

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// ScopedConditionalTracer provides a scoped wrapper for conditional tracing.
type ScopedConditionalTracer struct {
	tracer trace.Tracer
	scope  string
}

// NewScopedConditionalTracer creates a new ScopedConditionalTracer with the given tracer and scope.
func NewScopedConditionalTracer(tracer trace.Tracer, scope string) *ScopedConditionalTracer {
	return &ScopedConditionalTracer{
		tracer: tracer,
		scope:  scope,
	}
}

// Scope returns the scope of this tracer.
func (t *ScopedConditionalTracer) Scope() string {
	return t.scope
}

// Tracer returns the underlying tracer.
func (t *ScopedConditionalTracer) Tracer() trace.Tracer {
	return t.tracer
}

// Start creates a new span if tracing is enabled for this scope.
// If tracing is disabled, returns the original context and a noop span.
func (t *ScopedConditionalTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !IsTracingEnabled(ctx, t.scope) {
		return ctx, noop.Span{}
	}
	return t.tracer.Start(ctx, spanName, opts...)
}

// StartWithContext creates a new span if tracing is enabled, using the FeatureFlagContext
// from the provided context.
func (t *ScopedConditionalTracer) StartWithContext(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.Start(ctx, spanName, opts...)
}

// WithScope returns a new ScopedConditionalTracer with a different scope.
func (t *ScopedConditionalTracer) WithScope(scope string) *ScopedConditionalTracer {
	return &ScopedConditionalTracer{
		tracer: t.tracer,
		scope:  scope,
	}
}

// ConditionalStart creates a new span if tracing is enabled for the given scope.
// This is a package-level convenience function.
func ConditionalStart(ctx context.Context, tracer trace.Tracer, scope string, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !IsTracingEnabled(ctx, scope) {
		return ctx, noop.Span{}
	}
	return tracer.Start(ctx, spanName, opts...)
}

// ConditionalSpan is a wrapper that helps with conditional span creation and management.
// It can be used when you need to conditionally start spans at multiple points.
type ConditionalSpan struct {
	ctx     context.Context
	span    trace.Span
	enabled bool
}

// StartConditionalSpan starts a conditional span and returns a ConditionalSpan wrapper.
func StartConditionalSpan(ctx context.Context, tracer trace.Tracer, scope string, spanName string, opts ...trace.SpanStartOption) *ConditionalSpan {
	enabled := IsTracingEnabled(ctx, scope)
	if !enabled {
		return &ConditionalSpan{
			ctx:     ctx,
			span:    noop.Span{},
			enabled: false,
		}
	}

	newCtx, span := tracer.Start(ctx, spanName, opts...)
	return &ConditionalSpan{
		ctx:     newCtx,
		span:    span,
		enabled: true,
	}
}

// Context returns the context (with span if enabled).
func (cs *ConditionalSpan) Context() context.Context {
	return cs.ctx
}

// Span returns the span (noop if disabled).
func (cs *ConditionalSpan) Span() trace.Span {
	return cs.span
}

// Enabled returns whether the span is enabled.
func (cs *ConditionalSpan) Enabled() bool {
	return cs.enabled
}

// End ends the span if it was enabled.
func (cs *ConditionalSpan) End(opts ...trace.SpanEndOption) {
	if cs.enabled {
		cs.span.End(opts...)
	}
}

// SetAttributes sets attributes on the span if it was enabled.
func (cs *ConditionalSpan) SetAttributes(attrs ...attribute.KeyValue) {
	if cs.enabled {
		cs.span.SetAttributes(attrs...)
	}
}

// RecordError records an error on the span if it was enabled.
func (cs *ConditionalSpan) RecordError(err error, opts ...trace.EventOption) {
	if cs.enabled {
		cs.span.RecordError(err, opts...)
	}
}

// SetStatus sets the status on the span if it was enabled.
func (cs *ConditionalSpan) SetStatus(code codes.Code, description string) {
	if cs.enabled {
		cs.span.SetStatus(code, description)
	}
}

// SetName sets the name on the span if it was enabled.
func (cs *ConditionalSpan) SetName(name string) {
	if cs.enabled {
		cs.span.SetName(name)
	}
}

// AddEvent adds an event to the span if it was enabled.
func (cs *ConditionalSpan) AddEvent(name string, opts ...trace.EventOption) {
	if cs.enabled {
		cs.span.AddEvent(name, opts...)
	}
}
