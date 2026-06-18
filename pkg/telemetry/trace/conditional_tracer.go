package trace

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type Scope interface {
	traceScope()
}

type SystemScope struct {
	QueueName      *string
	QueueShardName string
	ItemKind       string
}

func (SystemScope) traceScope() {}

type UserScope struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
	FnID      uuid.UUID
}

func (UserScope) traceScope() {}

type TraceEnabledFn func(ctx context.Context, scope Scope) bool

type conditionalTracer struct {
	enabledFn TraceEnabledFn
	tracer    trace.Tracer
}

type ConditionalTracer interface {
	NewSpan(ctx context.Context, spanName string, scope Scope) (context.Context, trace.Span)
	NewUserSpan(ctx context.Context, spanName string, accountID uuid.UUID, envID uuid.UUID, fnID uuid.UUID) (context.Context, trace.Span)
}

func NewConditionalTracer(tracer trace.Tracer, fn TraceEnabledFn) ConditionalTracer {
	return &conditionalTracer{
		tracer:    tracer,
		enabledFn: fn,
	}
}

func NoopConditionalTracer() ConditionalTracer {
	return NewConditionalTracer(noop.Tracer{}, func(ctx context.Context, scope Scope) bool {
		return false
	})
}

func (t *conditionalTracer) NewUserSpan(ctx context.Context, spanName string, accountID uuid.UUID, envID uuid.UUID, fnID uuid.UUID) (context.Context, trace.Span) {
	return t.NewSpan(ctx, spanName, UserScope{
		AccountID: accountID,
		EnvID:     envID,
		FnID:      fnID,
	})
}

func (t *conditionalTracer) NewSpan(ctx context.Context, spanName string, scope Scope) (context.Context, trace.Span) {
	if t.enabledFn(ctx, scope) {
		ctx, span := t.tracer.Start(ctx, spanName)
		setScopeAttributes(span, scope)
		return ctx, span
	}

	return ctx, noop.Span{}
}

func setScopeAttributes(span trace.Span, scope Scope) {
	switch s := scope.(type) {
	case UserScope:
		span.SetAttributes(attribute.String("account_id", s.AccountID.String()))
		span.SetAttributes(attribute.String("env_id", s.EnvID.String()))
		span.SetAttributes(attribute.String("fn_id", s.FnID.String()))
	case SystemScope:
		if s.QueueName != nil {
			span.SetAttributes(attribute.String("queue_name", *s.QueueName))
		}
		if s.QueueShardName != "" {
			span.SetAttributes(attribute.String("queue_shard", s.QueueShardName))
		}
		if s.ItemKind != "" {
			span.SetAttributes(attribute.String("item_kind", s.ItemKind))
		}
	}
}

func AlwaysTrace(ctx context.Context, scope Scope) bool {
	return true
}
