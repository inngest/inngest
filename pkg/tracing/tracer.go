package tracing

import (
	"context"

	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest/version"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type TracerProvider struct {
	q sqlc.Querier
}

func NewTracerProvider(q sqlc.Querier) *TracerProvider {
	return &TracerProvider{
		q: q,
	}
}

func (p *TracerProvider) NewTracer(ctx context.Context, md statev2.Metadata, qi queue.Item) (context.Context, *Tracer) {
	exp := &DBExporter{q: p.q}
	base := sdktrace.NewSimpleSpanProcessor(exp)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(newExecutionProcessor(md, qi, base)),
	)

	return contextFromTraceparent(ctx, *qi.TraceParentID), &Tracer{tp: tp}
}

type Tracer struct {
	tp *sdktrace.TracerProvider
}

func (t *Tracer) Tracer() trace.Tracer {
	return t.tp.Tracer("inngest", trace.WithInstrumentationVersion(version.Print()))
}
