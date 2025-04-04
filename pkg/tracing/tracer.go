package tracing

import (
	"context"

	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type TracerProvider struct {
	q sqlc.Querier
}

func init() {
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
}

func NewTracerProvider(q sqlc.Querier) *TracerProvider {
	return &TracerProvider{
		q: q,
	}
}

func (tp *TracerProvider) NewTracer(ctx context.Context, md statev2.Metadata, qi queue.Item) (context.Context, *Tracer) {
	exp := &DBExporter{q: tp.q}
	base := sdktrace.NewSimpleSpanProcessor(exp)

	otelTP := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(newExecutionProcessor(md, qi, base)),
	)

	metadata := qi.Metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}

	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(metadata)), &Tracer{tp: otelTP}
}

func (tp *TracerProvider) GetLineage(ctx context.Context) map[string]string {
	metadata := make(map[string]string)
	tp.Inject(ctx, metadata)

	return metadata
}

func (tp *TracerProvider) Inject(ctx context.Context, metadata map[string]string) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(metadata))
}

func (tp *TracerProvider) Extract(ctx context.Context, metadata map[string]string) context.Context {
	if metadata == nil {
		metadata = make(map[string]string)
	}

	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(metadata))
}

type Tracer struct {
	tp *sdktrace.TracerProvider
}

func (t *Tracer) Tracer() trace.Tracer {
	return t.tp.Tracer("inngest", trace.WithInstrumentationVersion(version.Print()))
}
